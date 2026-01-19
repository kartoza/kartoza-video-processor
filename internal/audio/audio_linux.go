//go:build linux

package audio

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Recorder handles audio recording via PipeWire (Linux)
type Recorder struct {
	device     string
	outputFile string
	cmd        *exec.Cmd
	pid        int
}

// NewRecorder creates a new audio recorder
func NewRecorder(device, outputFile string) *Recorder {
	if device == "" {
		device = "@DEFAULT_SOURCE@"
	}
	return &Recorder{
		device:     device,
		outputFile: outputFile,
	}
}

// Start begins audio recording using pw-record
func (r *Recorder) Start() error {
	r.cmd = exec.Command("pw-record", "--target", r.device, r.outputFile)
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

	// Send SIGINT for graceful shutdown
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

// ListAudioDevices returns a list of available audio input devices on Linux
func ListAudioDevices() ([]string, error) {
	// Use pw-record --list-targets to list available PipeWire sources
	cmd := exec.Command("pw-record", "--list-targets")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []string{"@DEFAULT_SOURCE@"}, nil
	}
	return []string{string(output)}, nil
}
