package webcam

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/kartoza/kartoza-video-processor/internal/deps"
)

// Webcam represents a webcam recording session
type Webcam struct {
	device     string
	fps        int
	resolution string
	outputFile string
	cmd        *exec.Cmd
	pid        int
}

// Options for webcam recording
type Options struct {
	Device     string
	FPS        int
	Resolution string
	OutputFile string
}

// DefaultOptions returns default webcam recording options
func DefaultOptions() Options {
	return Options{
		Device:     "",
		FPS:        60,
		Resolution: "1920x1080",
	}
}

// New creates a new Webcam recorder
func New(opts Options) *Webcam {
	return &Webcam{
		device:     opts.Device,
		fps:        opts.FPS,
		resolution: opts.Resolution,
		outputFile: opts.OutputFile,
	}
}

// DetectDevice finds the first available webcam device
func DetectDevice() (string, error) {
	devices := []string{"video0", "video1", "video2", "video3"}

	for _, dev := range devices {
		path := "/dev/" + dev
		if _, err := os.Stat(path); err == nil {
			// Check if it's a character device (webcam)
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			// Check for character device
			if info.Mode()&os.ModeCharDevice != 0 {
				return dev, nil
			}
		}
	}

	return "", fmt.Errorf("no webcam device found")
}

// Start begins webcam recording
func (w *Webcam) Start() error {
	currentOS := deps.DetectOS()

	switch currentOS {
	case deps.OSWindows:
		return w.startWindows()
	case deps.OSDarwin:
		return w.startMacOS()
	case deps.OSLinux:
		return w.startLinux()
	default:
		// Unknown OS - try Linux as fallback
		return w.startLinux()
	}
}

// startLinux begins webcam recording on Linux using v4l2
func (w *Webcam) startLinux() error {
	device := w.device
	if device == "" {
		var err error
		device, err = DetectDevice()
		if err != nil {
			return err
		}
	}

	// Build ffmpeg command for real-time webcam capture
	// - input_format=mjpeg: Use hardware MJPEG for lower CPU usage
	// - preset=ultrafast: Minimal encoding latency for real-time
	// - tune=zerolatency: Optimize for zero-latency encoding
	// - crf=18: Near-lossless quality
	args := []string{
		"-f", "v4l2",
		"-input_format", "mjpeg",
		"-framerate", strconv.Itoa(w.fps),
		"-video_size", w.resolution,
		"-i", "/dev/" + device,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-bf", "0",
		"-g", strconv.Itoa(w.fps * 2), // Keyframe every 2 seconds
		"-threads", "0",
		"-x264opts", "no-scenecut",
		w.outputFile,
	}

	w.cmd = exec.Command("ffmpeg", args...)
	w.cmd.Stdout = nil
	w.cmd.Stderr = nil

	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start webcam recording: %w", err)
	}

	w.pid = w.cmd.Process.Pid
	return nil
}

// startWindows begins webcam recording on Windows using ffmpeg with dshow
func (w *Webcam) startWindows() error {
	// Use ffmpeg with dshow to record webcam on Windows
	// Uses empty device name to let ffmpeg auto-detect default webcam
	videoDevice := w.device
	if w.device == "" {
		// Let ffmpeg use default video device
		videoDevice = ""
	}

	args := []string{
		"-f", "dshow",
		"-video_size", w.resolution,
		"-framerate", strconv.Itoa(w.fps),
	}
	if videoDevice != "" {
		args = append(args, "-i", "video="+videoDevice)
	} else {
		// Empty video= lets ffmpeg pick the default device
		args = append(args, "-i", "video=")
	}
	args = append(args,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-y", // Overwrite output
		w.outputFile,
	)

	w.cmd = exec.Command("ffmpeg", args...)
	w.cmd.Stdout = nil
	w.cmd.Stderr = nil

	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start webcam recording: %w", err)
	}

	w.pid = w.cmd.Process.Pid
	return nil
}

// startMacOS begins webcam recording on macOS using ffmpeg with avfoundation
func (w *Webcam) startMacOS() error {
	// Use ffmpeg with avfoundation to record webcam on macOS
	// ffmpeg -f avfoundation -framerate 60 -video_size 1920x1080 -i "0:none" output.mp4
	// "0:none" means video device 0 (default webcam), no audio
	videoInput := "0:none"
	if w.device != "" {
		videoInput = w.device + ":none"
	}

	args := []string{
		"-f", "avfoundation",
		"-framerate", strconv.Itoa(w.fps),
		"-video_size", w.resolution,
		"-i", videoInput,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-y", // Overwrite output
		w.outputFile,
	}

	w.cmd = exec.Command("ffmpeg", args...)
	w.cmd.Stdout = nil
	w.cmd.Stderr = nil

	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start webcam recording: %w", err)
	}

	w.pid = w.cmd.Process.Pid
	return nil
}

// Stop stops the webcam recording
func (w *Webcam) Stop() error {
	if w.cmd == nil || w.cmd.Process == nil {
		return nil
	}

	// Send SIGINT for graceful shutdown
	if err := w.cmd.Process.Signal(syscall.SIGINT); err != nil {
		// If SIGINT fails, try SIGTERM
		if err := w.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return w.cmd.Process.Kill()
		}
	}

	// Wait for process to finish
	w.cmd.Wait()
	return nil
}

// PID returns the process ID of the ffmpeg process
func (w *Webcam) PID() int {
	return w.pid
}

// IsRecording returns true if recording is in progress
func (w *Webcam) IsRecording() bool {
	if w.cmd == nil || w.cmd.Process == nil {
		return false
	}

	// Check if process is still running
	err := w.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// GetVideoInfo returns video dimensions from a file
func GetVideoInfo(filepath string) (width, height int, err error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
		filepath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get video info: %w", err)
	}

	// Parse "width,height" format
	var w, h int
	_, err = fmt.Sscanf(string(output), "%d,%d", &w, &h)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse video dimensions: %w", err)
	}

	return w, h, nil
}

// VideoMetadata contains comprehensive video file information
type VideoMetadata struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	FPS         float64 `json:"fps"`
	AspectRatio string  `json:"aspect_ratio"`
	Duration    float64 `json:"duration_seconds"`
	Codec       string  `json:"codec"`
	Bitrate     int64   `json:"bitrate,omitempty"`
}

// GetFullVideoInfo returns comprehensive video metadata from a file
func GetFullVideoInfo(filepath string) (*VideoMetadata, error) {
	// Get width, height, fps, duration, codec using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate,codec_name,bit_rate:format=duration",
		"-of", "json",
		filepath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// Parse JSON output
	var probeResult struct {
		Streams []struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
			CodecName  string `json:"codec_name"`
			BitRate    string `json:"bit_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if len(probeResult.Streams) == 0 {
		return nil, fmt.Errorf("no video streams found")
	}

	stream := probeResult.Streams[0]
	meta := &VideoMetadata{
		Width:  stream.Width,
		Height: stream.Height,
		Codec:  stream.CodecName,
	}

	// Parse frame rate (format: "num/den")
	if stream.RFrameRate != "" {
		var num, den int
		if _, err := fmt.Sscanf(stream.RFrameRate, "%d/%d", &num, &den); err == nil && den > 0 {
			meta.FPS = float64(num) / float64(den)
		}
	}

	// Parse duration
	if probeResult.Format.Duration != "" {
		fmt.Sscanf(probeResult.Format.Duration, "%f", &meta.Duration)
	}

	// Parse bitrate
	if stream.BitRate != "" {
		fmt.Sscanf(stream.BitRate, "%d", &meta.Bitrate)
	}

	// Calculate aspect ratio
	meta.AspectRatio = calculateAspectRatio(stream.Width, stream.Height)

	return meta, nil
}

// calculateAspectRatio returns a human-readable aspect ratio string
func calculateAspectRatio(width, height int) string {
	if width == 0 || height == 0 {
		return "unknown"
	}

	// Find GCD
	gcd := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}

	g := gcd(width, height)
	w := width / g
	h := height / g

	// Common aspect ratios
	ratio := float64(width) / float64(height)
	switch {
	case ratio > 1.7 && ratio < 1.8: // ~16:9
		return "16:9"
	case ratio > 1.3 && ratio < 1.4: // ~4:3
		return "4:3"
	case ratio > 0.55 && ratio < 0.57: // ~9:16
		return "9:16"
	case ratio > 0.99 && ratio < 1.01: // ~1:1
		return "1:1"
	default:
		return fmt.Sprintf("%d:%d", w, h)
	}
}
