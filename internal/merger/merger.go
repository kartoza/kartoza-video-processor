package merger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/audio"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/notify"
	"github.com/kartoza/kartoza-video-processor/internal/webcam"
)

// ProcessingStep represents a step in the processing pipeline
type ProcessingStep int

const (
	StepAnalyzingAudio ProcessingStep = iota
	StepNormalizing
	StepMerging
	StepCreatingVertical
)

// ProgressCallback is called when a processing step starts or completes
type ProgressCallback func(step ProcessingStep, completed bool, skipped bool, err error)

// PercentCallback is called to report progress percentage during a step
type PercentCallback func(step ProcessingStep, percent float64)

// Merger handles merging of video, audio, and webcam recordings
type Merger struct {
	audioOpts  models.AudioProcessingOptions
	onProgress ProgressCallback
	onPercent  PercentCallback
}

// New creates a new Merger
func New(audioOpts models.AudioProcessingOptions) *Merger {
	return &Merger{audioOpts: audioOpts}
}

// SetProgressCallback sets the callback for progress updates
func (m *Merger) SetProgressCallback(cb ProgressCallback) {
	m.onProgress = cb
}

// SetPercentCallback sets the callback for percentage progress updates
func (m *Merger) SetPercentCallback(cb PercentCallback) {
	m.onPercent = cb
}

// reportProgress reports progress if callback is set
func (m *Merger) reportProgress(step ProcessingStep, completed bool, skipped bool, err error) {
	if m.onProgress != nil {
		m.onProgress(step, completed, skipped, err)
	}
}

// reportPercent reports percentage progress if callback is set
func (m *Merger) reportPercent(step ProcessingStep, percent float64) {
	if m.onPercent != nil {
		m.onPercent(step, percent)
	}
}

// runFFmpegWithProgress runs an FFmpeg command and reports progress
// durationUs is the expected duration in microseconds for calculating percentage
func (m *Merger) runFFmpegWithProgress(step ProcessingStep, durationUs int64, args ...string) error {
	// Add progress pipe and stats period to args for frequent updates
	// -stats_period 0.5 outputs progress every 0.5 seconds
	progressArgs := append([]string{"-progress", "pipe:1", "-stats_period", "0.5", "-nostats"}, args...)

	cmd := exec.Command("ffmpeg", progressArgs...)

	// Capture stdout for progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr for errors
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Report initial progress
	m.reportPercent(step, 0)

	// Parse progress output
	// FFmpeg outputs progress in key=value format, one per line
	// We're looking for out_time_us (microseconds)
	// Note: out_time_us can be "N/A" at the start, which we skip
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse out_time_us (microseconds)
		if strings.HasPrefix(line, "out_time_us=") {
			timeStr := strings.TrimPrefix(line, "out_time_us=")
			// Skip N/A values
			if timeStr == "N/A" {
				continue
			}
			if timeUs, err := strconv.ParseInt(timeStr, 10, 64); err == nil && durationUs > 0 && timeUs >= 0 {
				percent := float64(timeUs) / float64(durationUs) * 100
				if percent > 100 {
					percent = 100
				}
				m.reportPercent(step, percent)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w, stderr: %s", err, stderrBuf.String())
	}

	return nil
}

// getVideoDurationUs returns the duration of a video in microseconds
func getVideoDurationUs(filepath string) int64 {
	meta, err := webcam.GetFullVideoInfo(filepath)
	if err != nil {
		return 0
	}
	return int64(meta.Duration * 1000000) // Convert seconds to microseconds
}

// MergeOptions contains options for merging recordings
type MergeOptions struct {
	VideoFile      string
	AudioFile      string
	WebcamFile     string
	CreateVertical bool
	AddLogos       bool   // Whether to add logo overlays
	ProductLogo1   string // Path to product logo 1 (top-left)
	ProductLogo2   string // Path to product logo 2 (top-right)
	CompanyLogo    string // Path to company logo (lower third)
	VideoTitle     string // Title for lower third overlay
	TitleColor     string // Color for title text (e.g., "white", "black", "yellow")
	OutputDir      string // Directory for output files
}

// MergeResult contains the paths to merged files and processing info
type MergeResult struct {
	MergedFile       string
	VerticalFile     string
	NormalizeApplied bool
}

// Merge merges video and audio recordings
func (m *Merger) Merge(opts MergeOptions) (*MergeResult, error) {
	result := &MergeResult{}

	// Process audio first
	normalizedAudio := strings.TrimSuffix(opts.AudioFile, ".wav") + "-normalized.wav"
	processor := audio.NewProcessor(m.audioOpts)
	currentAudio := opts.AudioFile

	// Step 1: Analyze audio levels
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

	// Step 2: Normalize audio
	m.reportProgress(StepNormalizing, false, false, nil)
	if m.audioOpts.NormalizeEnabled && stats != nil {
		if err := processor.Normalize(currentAudio, normalizedAudio, stats); err != nil {
			m.reportProgress(StepNormalizing, true, true, err)
			notify.Warning("Audio Normalization Warning", "Using original audio")
			normalizedAudio = currentAudio
		} else {
			result.NormalizeApplied = true
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

		if err := m.createVerticalVideo(opts.VideoFile, opts.WebcamFile, normalizedAudio, verticalFile, &opts); err != nil {
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
	// Get video duration for progress calculation
	durationUs := getVideoDurationUs(videoFile)

	// CRF 18 = high quality, preset medium = good speed/quality balance
	// 30fps for smaller files and faster encoding
	// AAC at 320k for highest audio quality
	args := []string{
		"-y",
		"-i", videoFile,
		"-i", audioFile,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-r", "30",
		"-c:a", "aac",
		"-b:a", "320k",
		"-shortest",
		outputFile,
	}

	return m.runFFmpegWithProgress(StepMerging, durationUs, args...)
}

// YouTube Shorts recommended dimensions
const (
	YouTubeShortsWidth  = 1080
	YouTubeShortsHeight = 1920
)

// createVerticalVideo creates a vertical video with webcam at the bottom
// Output is always 1080x1920 (9:16) for YouTube Shorts compatibility
func (m *Merger) createVerticalVideo(videoFile, webcamFile, audioFile, outputFile string, opts *MergeOptions) error {
	notify.ProcessingStep("Creating vertical video (1080x1920) with webcam...")

	// Get screen video dimensions
	screenWidth, screenHeight, err := webcam.GetVideoInfo(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get screen dimensions: %w", err)
	}

	// Get webcam dimensions
	webcamWidth, webcamHeight, err := webcam.GetVideoInfo(webcamFile)
	if err != nil {
		return fmt.Errorf("failed to get webcam dimensions: %w", err)
	}

	// Calculate layout for 1080x1920 output
	// Screen takes top portion, webcam takes bottom portion
	// Scale screen to fit 1080 width while maintaining aspect ratio
	scaledScreenWidth := YouTubeShortsWidth
	scaledScreenHeight := screenHeight * YouTubeShortsWidth / screenWidth

	// Calculate remaining height for webcam
	remainingHeight := YouTubeShortsHeight - scaledScreenHeight

	// Scale webcam to fit remaining space while maintaining aspect ratio
	// First try fitting to width
	scaledWebcamWidth := YouTubeShortsWidth
	scaledWebcamHeight := webcamHeight * YouTubeShortsWidth / webcamWidth

	// If webcam is too tall, scale to fit height instead
	if scaledWebcamHeight > remainingHeight {
		scaledWebcamHeight = remainingHeight
		scaledWebcamWidth = webcamWidth * remainingHeight / webcamHeight
	}

	// Calculate padding needed to center webcam horizontally
	webcamPadX := (YouTubeShortsWidth - scaledWebcamWidth) / 2

	// Build inputs list
	inputs := []string{"-y", "-i", videoFile, "-i", webcamFile, "-i", audioFile}

	// Copy logos to output directory if needed
	var logo1Path, logo2Path, companyLogoPath string
	if opts != nil && opts.AddLogos && opts.OutputDir != "" {
		if opts.ProductLogo1 != "" {
			logo1Path = m.copyLogoToOutputDir(opts.ProductLogo1, opts.OutputDir, "product_logo_1")
			if logo1Path != "" {
				inputs = appendLogoInput(inputs, logo1Path)
			}
		}
		if opts.ProductLogo2 != "" {
			logo2Path = m.copyLogoToOutputDir(opts.ProductLogo2, opts.OutputDir, "product_logo_2")
			if logo2Path != "" {
				inputs = appendLogoInput(inputs, logo2Path)
			}
		}
		if opts.CompanyLogo != "" {
			companyLogoPath = m.copyLogoToOutputDir(opts.CompanyLogo, opts.OutputDir, "company_logo")
			if companyLogoPath != "" {
				inputs = appendLogoInput(inputs, companyLogoPath)
			}
		}
	}

	// Build filter complex for 1080x1920 output
	// 1. Scale screen to fit width (1080)
	// 2. Scale webcam to fit remaining height
	// 3. Create black canvas of 1080x1920
	// 4. Overlay screen at top
	// 5. Overlay webcam at bottom (centered)
	filterComplex := fmt.Sprintf(
		// Scale screen video to 1080 width
		"[0:v]scale=%d:%d:flags=lanczos[screen];"+
			// Scale webcam to fit
			"[1:v]scale=%d:%d:flags=lanczos[webcam];"+
			// Create black background canvas
			"color=black:size=%dx%d:duration=99999[bg];"+
			// Overlay screen at top center
			"[bg][screen]overlay=(W-w)/2:0[with_screen];"+
			// Overlay webcam at bottom center
			"[with_screen][webcam]overlay=%d:%d[stacked]",
		scaledScreenWidth, scaledScreenHeight,
		scaledWebcamWidth, scaledWebcamHeight,
		YouTubeShortsWidth, YouTubeShortsHeight,
		webcamPadX, scaledScreenHeight,
	)

	currentOutput := "[stacked]"
	logoInputIndex := 3 // Start after video, webcam, audio

	// Determine title color (default to white if not specified)
	titleColor := "white"
	if opts != nil && opts.TitleColor != "" {
		titleColor = opts.TitleColor
	}

	// Add logo overlays if logos are provided
	// Using shortest=1 ensures animated GIFs stop when the base video ends
	// For GIFs, we add a white background using split and overlay to handle transparency
	if logo1Path != "" {
		// Product logo 1 in top-left of webcam area
		logoY := scaledScreenHeight + 10 // Position in webcam area
		if isGif(logo1Path) {
			// For GIFs: create white background, then overlay the GIF on it
			filterComplex += fmt.Sprintf(
				";[%d:v]scale=iw/4:-1[logo1_raw];"+
					"[logo1_raw]split[logo1_a][logo1_b];"+
					"[logo1_a]drawbox=c=white:t=fill[logo1_bg];"+
					"[logo1_bg][logo1_b]overlay=0:0:format=auto[logo1];"+
					"%s[logo1]overlay=10:%d:format=auto:shortest=1[out1]",
				logoInputIndex, currentOutput, logoY,
			)
		} else {
			filterComplex += fmt.Sprintf(
				";[%d:v]scale=iw/4:-1[logo1];%s[logo1]overlay=10:%d:format=auto:shortest=1[out1]",
				logoInputIndex, currentOutput, logoY,
			)
		}
		currentOutput = "[out1]"
		logoInputIndex++
	}

	if logo2Path != "" {
		// Product logo 2 in top-right of webcam area
		logoY := scaledScreenHeight + 10
		if isGif(logo2Path) {
			// For GIFs: create white background, then overlay the GIF on it
			filterComplex += fmt.Sprintf(
				";[%d:v]scale=iw/4:-1[logo2_raw];"+
					"[logo2_raw]split[logo2_a][logo2_b];"+
					"[logo2_a]drawbox=c=white:t=fill[logo2_bg];"+
					"[logo2_bg][logo2_b]overlay=0:0:format=auto[logo2];"+
					"%s[logo2]overlay=W-w-10:%d:format=auto:shortest=1[out2]",
				logoInputIndex, currentOutput, logoY,
			)
		} else {
			filterComplex += fmt.Sprintf(
				";[%d:v]scale=iw/4:-1[logo2];%s[logo2]overlay=W-w-10:%d:format=auto:shortest=1[out2]",
				logoInputIndex, currentOutput, logoY,
			)
		}
		currentOutput = "[out2]"
		logoInputIndex++
	}

	if companyLogoPath != "" && opts != nil && opts.VideoTitle != "" {
		// Company logo as lower third with title overlay
		// Title text is horizontally centered using (w-text_w)/2
		lowerThirdY := YouTubeShortsHeight - 100 // Position near bottom
		if isGif(companyLogoPath) {
			// For GIFs: create white background, then overlay the GIF on it
			filterComplex += fmt.Sprintf(
				";[%d:v]scale=200:-1[complogo_raw];"+
					"[complogo_raw]split[complogo_a][complogo_b];"+
					"[complogo_a]drawbox=c=white:t=fill[complogo_bg];"+
					"[complogo_bg][complogo_b]overlay=0:0:format=auto[complogo];"+
					"%s[complogo]overlay=10:%d:format=auto:shortest=1[out3];"+
					"[out3]drawtext=text='%s':fontcolor=%s:fontsize=36:x=(w-text_w)/2:y=%d[outv]",
				logoInputIndex, currentOutput, lowerThirdY, escapeFFmpegText(opts.VideoTitle), titleColor, lowerThirdY+30,
			)
		} else {
			filterComplex += fmt.Sprintf(
				";[%d:v]scale=200:-1[complogo];%s[complogo]overlay=10:%d:format=auto:shortest=1[out3];"+
					"[out3]drawtext=text='%s':fontcolor=%s:fontsize=36:x=(w-text_w)/2:y=%d[outv]",
				logoInputIndex, currentOutput, lowerThirdY, escapeFFmpegText(opts.VideoTitle), titleColor, lowerThirdY+30,
			)
		}
	} else {
		filterComplex += fmt.Sprintf(";%s[outv]", currentOutput)
	}

	// Get video duration for progress calculation
	durationUs := getVideoDurationUs(videoFile)

	args := append(inputs,
		"-filter_complex", filterComplex,
		"-map", "[outv]",
		"-map", "2:a",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-r", "30", // 30fps for smaller files and faster encoding
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "320k",
		"-shortest",
		outputFile,
	)

	return m.runFFmpegWithProgress(StepCreatingVertical, durationUs, args...)
}

// copyLogoToOutputDir copies a logo file to the output directory
func (m *Merger) copyLogoToOutputDir(srcPath, outputDir, baseName string) string {
	if srcPath == "" {
		return ""
	}

	// Check if source file exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return ""
	}

	// Get the extension
	ext := filepath.Ext(srcPath)
	destPath := filepath.Join(outputDir, baseName+ext)

	// Copy the file
	src, err := os.Open(srcPath)
	if err != nil {
		return ""
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return ""
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return ""
	}

	return destPath
}

// escapeFFmpegText escapes special characters for FFmpeg drawtext filter
func escapeFFmpegText(text string) string {
	// Escape special characters for FFmpeg
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "'", "'\\''")
	text = strings.ReplaceAll(text, ":", "\\:")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")
	return text
}

// isGif checks if the file is a GIF based on extension
func isGif(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".gif"
}

// appendLogoInput adds a logo input to the FFmpeg args
// For GIFs, it adds -ignore_loop 0 to make them loop forever
func appendLogoInput(inputs []string, logoPath string) []string {
	if isGif(logoPath) {
		// For animated GIFs: -ignore_loop 0 makes the GIF loop forever
		return append(inputs, "-ignore_loop", "0", "-i", logoPath)
	}
	return append(inputs, "-i", logoPath)
}
