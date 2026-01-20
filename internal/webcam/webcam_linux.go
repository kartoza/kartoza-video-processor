//go:build linux

package webcam

import (
	"fmt"
	"os"
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

// Start begins webcam recording using v4l2
func (w *Webcam) Start() error {
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

	// Send SIGINT for graceful shutdown
	if err := w.cmd.Process.Signal(syscall.SIGINT); err != nil {
		// If SIGINT fails, try SIGTERM
		if err := w.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return w.cmd.Process.Kill()
		}
	}

	// Wait for process to finish
	_ = w.cmd.Wait()
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

// ListDevices returns a list of available webcam devices on Linux
func ListDevices() ([]string, error) {
	var devices []string
	for i := 0; i < 10; i++ {
		dev := fmt.Sprintf("video%d", i)
		path := "/dev/" + dev
		if info, err := os.Stat(path); err == nil {
			if info.Mode()&os.ModeCharDevice != 0 {
				devices = append(devices, dev)
			}
		}
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no webcam devices found")
	}
	return devices, nil
}
