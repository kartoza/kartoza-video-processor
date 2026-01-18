package merger

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/audio"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/notify"
	"github.com/kartoza/kartoza-video-processor/internal/webcam"
)

// ProcessingStep represents a step in the processing pipeline
type ProcessingStep int

const (
	StepDenoising ProcessingStep = iota
	StepAnalyzingAudio
	StepNormalizing
	StepMerging
	StepCreatingVertical
)

// ProgressCallback is called when a processing step starts or completes
type ProgressCallback func(step ProcessingStep, completed bool, skipped bool, err error)

// Merger handles merging of video, audio, and webcam recordings
type Merger struct {
	audioOpts models.AudioProcessingOptions
	onProgress ProgressCallback
}

// New creates a new Merger
func New(audioOpts models.AudioProcessingOptions) *Merger {
	return &Merger{audioOpts: audioOpts}
}

// SetProgressCallback sets the callback for progress updates
func (m *Merger) SetProgressCallback(cb ProgressCallback) {
	m.onProgress = cb
}

// reportProgress reports progress if callback is set
func (m *Merger) reportProgress(step ProcessingStep, completed bool, skipped bool, err error) {
	if m.onProgress != nil {
		m.onProgress(step, completed, skipped, err)
	}
}

// MergeOptions contains options for merging recordings
type MergeOptions struct {
	VideoFile      string
	AudioFile      string
	WebcamFile     string
	CreateVertical bool
}

// MergeResult contains the paths to merged files
type MergeResult struct {
	MergedFile   string
	VerticalFile string
}

// Merge merges video and audio recordings
func (m *Merger) Merge(opts MergeOptions) (*MergeResult, error) {
	result := &MergeResult{}

	// Process audio first
	normalizedAudio := strings.TrimSuffix(opts.AudioFile, ".wav") + "-normalized.wav"
	processor := audio.NewProcessor(m.audioOpts)

	// Step 1: Denoise
	m.reportProgress(StepDenoising, false, false, nil)
	denoisedAudio := strings.TrimSuffix(opts.AudioFile, ".wav") + "-denoised.wav"
	currentAudio := opts.AudioFile

	if m.audioOpts.DenoiseEnabled {
		if err := processor.Denoise(opts.AudioFile, denoisedAudio); err != nil {
			m.reportProgress(StepDenoising, true, true, err)
			notify.Warning("Noise Reduction Warning", "Skipping noise reduction")
		} else {
			currentAudio = denoisedAudio
			m.reportProgress(StepDenoising, true, false, nil)
		}
	} else {
		m.reportProgress(StepDenoising, true, true, nil)
	}

	// Step 2: Analyze audio levels
	m.reportProgress(StepAnalyzingAudio, false, false, nil)
	var stats *models.LoudnormStats
	if m.audioOpts.NormalizeEnabled {
		var err error
		stats, err = processor.AnalyzeLoudness(currentAudio)
		if err != nil {
			m.reportProgress(StepAnalyzingAudio, true, true, err)
			notify.Warning("Audio Analysis Warning", "Skipping normalization")
		} else {
			m.reportProgress(StepAnalyzingAudio, true, false, nil)
		}
	} else {
		m.reportProgress(StepAnalyzingAudio, true, true, nil)
	}

	// Step 3: Normalize audio
	m.reportProgress(StepNormalizing, false, false, nil)
	if m.audioOpts.NormalizeEnabled && stats != nil {
		if err := processor.Normalize(currentAudio, normalizedAudio, stats); err != nil {
			m.reportProgress(StepNormalizing, true, true, err)
			notify.Warning("Audio Normalization Warning", "Using original audio")
			normalizedAudio = currentAudio
		} else {
			m.reportProgress(StepNormalizing, true, false, nil)
		}
	} else {
		normalizedAudio = currentAudio
		m.reportProgress(StepNormalizing, true, true, nil)
	}

	// Step 4: Merge video and audio
	m.reportProgress(StepMerging, false, false, nil)
	outputFile := strings.TrimSuffix(opts.VideoFile, ".mp4") + "-merged.mp4"
	notify.ProcessingStep("Merging video and audio...")

	if err := m.mergeVideoAudio(opts.VideoFile, normalizedAudio, outputFile); err != nil {
		m.reportProgress(StepMerging, true, false, err)
		return nil, fmt.Errorf("failed to merge video and audio: %w", err)
	}
	m.reportProgress(StepMerging, true, false, nil)

	result.MergedFile = outputFile
	notify.RecordingComplete(filepath.Base(outputFile))

	// Step 5: Create vertical video with webcam if available
	m.reportProgress(StepCreatingVertical, false, false, nil)
	if opts.CreateVertical && opts.WebcamFile != "" {
		verticalFile := strings.TrimSuffix(opts.VideoFile, ".mp4") + "-vertical.mp4"

		if err := m.createVerticalVideo(opts.VideoFile, opts.WebcamFile, normalizedAudio, verticalFile); err != nil {
			m.reportProgress(StepCreatingVertical, true, true, err)
			notify.Warning("Vertical Video Warning", "Failed to create vertical video")
		} else {
			result.VerticalFile = verticalFile
			m.reportProgress(StepCreatingVertical, true, false, nil)
			notify.VerticalComplete(filepath.Base(verticalFile))
		}
	} else {
		m.reportProgress(StepCreatingVertical, true, true, nil)
	}

	return result, nil
}

// mergeVideoAudio merges video and audio using ffmpeg
func (m *Merger) mergeVideoAudio(videoFile, audioFile, outputFile string) error {
	// CRF 0 = completely lossless, preset veryslow = best quality/compression
	// AAC at 320k for highest audio quality
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", videoFile,
		"-i", audioFile,
		"-c:v", "libx264",
		"-preset", "veryslow",
		"-crf", "0",
		"-c:a", "aac",
		"-b:a", "320k",
		"-shortest",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg merge failed: %w, output: %s", err, output)
	}

	return nil
}

// createVerticalVideo creates a vertical video with webcam at the bottom
func (m *Merger) createVerticalVideo(videoFile, webcamFile, audioFile, outputFile string) error {
	notify.ProcessingStep("Creating vertical video with webcam...")

	// Get screen video dimensions
	screenWidth, screenHeight, err := webcam.GetVideoInfo(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get screen dimensions: %w", err)
	}

	// Get webcam dimensions
	webcamWidth, webcamHeightOrig, err := webcam.GetVideoInfo(webcamFile)
	if err != nil {
		return fmt.Errorf("failed to get webcam dimensions: %w", err)
	}

	// Calculate webcam height to match screen width (maintain aspect ratio)
	webcamHeight := screenWidth * webcamHeightOrig / webcamWidth
	if webcamWidth <= 0 {
		webcamHeight = screenWidth * 3 / 4 // Fallback to 4:3
	}

	// Build filter complex for vertical stacking
	filterComplex := fmt.Sprintf(
		"[0:v]scale=%d:%d:flags=lanczos[screen];"+
			"[1:v]scale=%d:%d:flags=lanczos[webcam];"+
			"[screen][webcam]vstack=inputs=2[outv]",
		screenWidth, screenHeight,
		screenWidth, webcamHeight,
	)

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", videoFile,
		"-i", webcamFile,
		"-i", audioFile,
		"-filter_complex", filterComplex,
		"-map", "[outv]",
		"-map", "2:a",
		"-c:v", "libx264",
		"-preset", "veryslow",
		"-crf", "0",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "320k",
		"-shortest",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg vertical failed: %w, output: %s", err, output)
	}

	return nil
}
