//go:build windows

package webcam

import (
	"fmt"
	"os/exec"
	"strconv"
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

// DetectDevice finds the first available webcam device on Windows
func DetectDevice() (string, error) {
	// On Windows with dshow, device names are like "Integrated Camera"
	// Users can run: ffmpeg -f dshow -list_devices true -i dummy
	// to find available devices
	// Default to generic webcam name
	return "video=Integrated Camera", nil
}

// Start begins webcam recording using ffmpeg with dshow
func (w *Webcam) Start() error {
	device := w.device
	if device == "" {
		device = "video=Integrated Camera" // Default webcam name
	}

	// Build ffmpeg command for real-time webcam capture on Windows
	// dshow input format: "video=Device Name"
	args := []string{
		"-f", "dshow",
		"-framerate", strconv.Itoa(w.fps),
		"-video_size", w.resolution,
		"-i", device,
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

	// On Windows, we need to kill the process directly
	// SIGINT doesn't work the same way
	return w.cmd.Process.Kill()
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

	// On Windows, check if process has exited
	// Process.Signal(0) doesn't work reliably on Windows
	return w.cmd.ProcessState == nil
}

// ListDevices returns a list of available webcam devices on Windows
func ListDevices() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-f", "dshow", "-list_devices", "true", "-i", "dummy")
	output, _ := cmd.CombinedOutput() // ffmpeg returns error even on success for listing

	// Return raw output - full parsing would need regex for device names
	return []string{string(output)}, nil
}
