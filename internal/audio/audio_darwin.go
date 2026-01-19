//go:build darwin

package audio

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Recorder handles audio recording via ffmpeg avfoundation (macOS)
type Recorder struct {
	device     string
	outputFile string
	cmd        *exec.Cmd
	pid        int
}

// NewRecorder creates a new audio recorder
func NewRecorder(device, outputFile string) *Recorder {
	if device == "" {
		// Default audio input device on macOS
		// Use ":0" for default audio input (no video, first audio device)
		device = ":0"
	}
	return &Recorder{
		device:     device,
		outputFile: outputFile,
	}
}

// Start begins audio recording using ffmpeg with avfoundation
func (r *Recorder) Start() error {
	// ffmpeg -f avfoundation -i ":0" -c:a pcm_s16le output.wav
	// The ":0" means no video input, audio device 0
	r.cmd = exec.Command("ffmpeg",
		"-f", "avfoundation",
		"-i", r.device,
		"-c:a", "pcm_s16le",
		"-y", // Overwrite output file
		r.outputFile,
	)
	r.cmd.Stdout = nil
	r.cmd.Stderr = nil

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start audio recording: %w", err)
	}

	r.pid = r.cmd.Process.Pid
	return nil
}

// Stop stops audio recording
func (r *Recorder) Stop() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}

	// Send SIGINT for graceful shutdown (ffmpeg handles this)
	if err := r.cmd.Process.Signal(syscall.SIGINT); err != nil {
		return r.cmd.Process.Kill()
	}

	r.cmd.Wait()
	return nil
}

// PID returns the process ID
func (r *Recorder) PID() int {
	return r.pid
}

// ListAudioDevices returns a list of available audio input devices on macOS
func ListAudioDevices() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	output, _ := cmd.CombinedOutput() // ffmpeg returns error even on success for listing

	// Parse output to extract audio devices
	// This is a simplified version - full parsing would need regex
	return []string{string(output)}, nil
}
