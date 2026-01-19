//go:build windows

package audio

import (
	"fmt"
	"os/exec"
)

// Recorder handles audio recording via ffmpeg dshow (Windows)
type Recorder struct {
	device     string
	outputFile string
	cmd        *exec.Cmd
	pid        int
}

// NewRecorder creates a new audio recorder
func NewRecorder(device, outputFile string) *Recorder {
	if device == "" {
		// Default audio input device on Windows
		// Users can run: ffmpeg -f dshow -list_devices true -i dummy
		// to find available audio devices
		device = "audio=Microphone"
	}
	return &Recorder{
		device:     device,
		outputFile: outputFile,
	}
}

// Start begins audio recording using ffmpeg with dshow
func (r *Recorder) Start() error {
	// ffmpeg -f dshow -i audio="Microphone" -c:a pcm_s16le output.wav
	r.cmd = exec.Command("ffmpeg",
		"-f", "dshow",
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

	// On Windows, we need to kill the process directly
	// SIGINT doesn't work the same way
	return r.cmd.Process.Kill()
}

// PID returns the process ID
func (r *Recorder) PID() int {
	return r.pid
}

// ListAudioDevices returns a list of available audio input devices on Windows
func ListAudioDevices() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-f", "dshow", "-list_devices", "true", "-i", "dummy")
	output, _ := cmd.CombinedOutput() // ffmpeg returns error even on success for listing

	// Parse output to extract audio devices
	// This is a simplified version - full parsing would need regex
	return []string{string(output)}, nil
}
