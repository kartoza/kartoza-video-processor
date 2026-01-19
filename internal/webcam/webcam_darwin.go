//go:build darwin

package webcam

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
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

// New creates a new Webcam recorder
func New(opts Options) *Webcam {
	return &Webcam{
		device:     opts.Device,
		fps:        opts.FPS,
		resolution: opts.Resolution,
		outputFile: opts.OutputFile,
	}
}

// DetectDevice finds the first available webcam device on macOS
func DetectDevice() (string, error) {
	// On macOS with avfoundation, "0" typically refers to the FaceTime camera
	// Users can run: ffmpeg -f avfoundation -list_devices true -i ""
	// to find available devices
	return "0", nil
}

// Start begins webcam recording using ffmpeg with avfoundation
func (w *Webcam) Start() error {
	device := w.device
	if device == "" {
		device = "0" // Default to first video device
	}

	// Build ffmpeg command for real-time webcam capture on macOS
	// avfoundation input format: "video_device_index:audio_device_index"
	// Using just "0" for video only (no audio)
	args := []string{
		"-f", "avfoundation",
		"-framerate", strconv.Itoa(w.fps),
		"-video_size", w.resolution,
		"-i", device + ":", // video:audio format, empty audio means no audio
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-bf", "0",
		"-g", strconv.Itoa(w.fps * 2), // Keyframe every 2 seconds
		"-threads", "0",
		"-y",
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

	// Send SIGINT for graceful shutdown (ffmpeg handles this)
	if err := w.cmd.Process.Signal(syscall.SIGINT); err != nil {
		return w.cmd.Process.Kill()
	}

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

// ListDevices returns a list of available webcam devices on macOS
func ListDevices() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	output, _ := cmd.CombinedOutput() // ffmpeg returns error even on success for listing

	// Return raw output - full parsing would need regex for device names
	return []string{string(output)}, nil
}
