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

// Merger handles merging of video, audio, and webcam recordings
type Merger struct {
	audioOpts models.AudioProcessingOptions
}

// New creates a new Merger
func New(audioOpts models.AudioProcessingOptions) *Merger {
	return &Merger{audioOpts: audioOpts}
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

	if err := processor.Process(opts.AudioFile, normalizedAudio); err != nil {
		// Use original audio if processing fails
		normalizedAudio = opts.AudioFile
	}

	// Merge video and audio
	outputFile := strings.TrimSuffix(opts.VideoFile, ".mp4") + "-merged.mp4"
	notify.ProcessingStep("Merging video and audio...")

	if err := m.mergeVideoAudio(opts.VideoFile, normalizedAudio, outputFile); err != nil {
		return nil, fmt.Errorf("failed to merge video and audio: %w", err)
	}

	result.MergedFile = outputFile
	notify.RecordingComplete(filepath.Base(outputFile))

	// Create vertical video with webcam if available
	if opts.CreateVertical && opts.WebcamFile != "" {
		verticalFile := strings.TrimSuffix(opts.VideoFile, ".mp4") + "-vertical.mp4"

		if err := m.createVerticalVideo(opts.VideoFile, opts.WebcamFile, normalizedAudio, verticalFile); err != nil {
			notify.Warning("Vertical Video Warning", "Failed to create vertical video")
		} else {
			result.VerticalFile = verticalFile
			notify.VerticalComplete(filepath.Base(verticalFile))
		}
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
