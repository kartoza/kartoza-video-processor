package recorder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartoza/kartoza-video-processor/internal/config"
)

func TestNew(t *testing.T) {
	rec := New()

	if rec == nil {
		t.Fatal("New() returned nil")
	}

	if rec.config == nil {
		t.Error("expected config to be set")
	}
}

func TestRecorder_IsRecording_NoPIDFiles(t *testing.T) {
	// Clean up any existing PID files
	os.Remove(config.VideoPIDFile)
	os.Remove(config.AudioPIDFile)
	os.Remove(config.WebcamPIDFile)

	rec := New()

	if rec.IsRecording() {
		t.Error("expected IsRecording() to return false when no PID files exist")
	}
}

func TestRecorder_GetStatus_NotRecording(t *testing.T) {
	// Clean up any existing PID files
	os.Remove(config.VideoPIDFile)
	os.Remove(config.AudioPIDFile)
	os.Remove(config.WebcamPIDFile)
	os.Remove(config.StatusFile)

	rec := New()
	status := rec.GetStatus()

	if status.IsRecording {
		t.Error("expected IsRecording to be false")
	}

	if status.VideoPID != 0 {
		t.Errorf("expected VideoPID to be 0, got %d", status.VideoPID)
	}

	if status.AudioPID != 0 {
		t.Errorf("expected AudioPID to be 0, got %d", status.AudioPID)
	}

	if status.WebcamPID != 0 {
		t.Errorf("expected WebcamPID to be 0, got %d", status.WebcamPID)
	}
}

func TestOptions_DefaultValues(t *testing.T) {
	opts := Options{}

	if opts.Monitor != "" {
		t.Error("expected Monitor to be empty by default")
	}

	if opts.NoAudio {
		t.Error("expected NoAudio to be false by default")
	}

	if opts.NoWebcam {
		t.Error("expected NoWebcam to be false by default")
	}

	if opts.HWAccel {
		t.Error("expected HWAccel to be false by default")
	}

	if opts.OutputDir != "" {
		t.Error("expected OutputDir to be empty by default")
	}
}

func TestRecorderInstance_DefaultValues(t *testing.T) {
	ri := &recorderInstance{}

	if ri.name != "" {
		t.Error("expected name to be empty")
	}

	if ri.pid != 0 {
		t.Error("expected pid to be 0")
	}

	if ri.file != "" {
		t.Error("expected file to be empty")
	}

	if ri.started {
		t.Error("expected started to be false")
	}

	if ri.err != nil {
		t.Error("expected err to be nil")
	}
}

func TestExtractMonitorFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "/home/user/videos/screenrecording-HDMI-A-1-20260118-103045.mp4",
			expected: "HDMI-A-1",
		},
		{
			path:     "/videos/screenrecording-DP-1-20260118-103045.mp4",
			expected: "DP-1",
		},
		{
			path:     "/videos/screenrecording-eDP-1-20260118-103045.mp4",
			expected: "eDP-1",
		},
		{
			path:     "/videos/screenrecording-HDMI-A-2-20260118-103045.wav",
			expected: "HDMI-A-2",
		},
		{
			path:     "screenrecording-.mp4",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractMonitorFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("extractMonitorFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestCheckPID_NonExistent(t *testing.T) {
	// Use a non-existent PID file
	result := checkPID("/tmp/non-existent-pid-file-12345")

	if result {
		t.Error("expected checkPID to return false for non-existent file")
	}
}

func TestReadPID_NonExistent(t *testing.T) {
	pid := readPID("/tmp/non-existent-pid-file-12345")

	if pid != 0 {
		t.Errorf("expected readPID to return 0 for non-existent file, got %d", pid)
	}
}

func TestReadPID_InvalidContent(t *testing.T) {
	// Create a temp file with invalid content
	tmpFile := filepath.Join(os.TempDir(), "test-invalid-pid")
	err := os.WriteFile(tmpFile, []byte("not-a-number"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	pid := readPID(tmpFile)

	if pid != 0 {
		t.Errorf("expected readPID to return 0 for invalid content, got %d", pid)
	}
}

func TestReadPID_ValidContent(t *testing.T) {
	// Create a temp file with valid PID
	tmpFile := filepath.Join(os.TempDir(), "test-valid-pid")
	err := os.WriteFile(tmpFile, []byte("12345"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	pid := readPID(tmpFile)

	if pid != 12345 {
		t.Errorf("expected readPID to return 12345, got %d", pid)
	}
}

func TestWritePID(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-write-pid")
	defer os.Remove(tmpFile)

	writePID(tmpFile, 54321)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read PID file: %v", err)
	}

	if string(data) != "54321" {
		t.Errorf("expected PID file to contain '54321', got %q", string(data))
	}
}

func TestReadPath_NonExistent(t *testing.T) {
	path := readPath("/tmp/non-existent-path-file-12345")

	if path != "" {
		t.Errorf("expected readPath to return empty string, got %q", path)
	}
}

func TestReadPath_ValidContent(t *testing.T) {
	// Create a temp file with a path
	tmpFile := filepath.Join(os.TempDir(), "test-path-file")
	expected := "/home/user/videos/recording.mp4"
	err := os.WriteFile(tmpFile, []byte(expected), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	path := readPath(tmpFile)

	if path != expected {
		t.Errorf("expected readPath to return %q, got %q", expected, path)
	}
}

func TestRecorder_Stop_NoRecording(t *testing.T) {
	// Clean up any existing PID files
	os.Remove(config.VideoPIDFile)
	os.Remove(config.AudioPIDFile)
	os.Remove(config.WebcamPIDFile)

	rec := New()
	err := rec.Stop()

	if err == nil {
		t.Error("expected error when stopping without recording")
	}

	expectedErr := "no recording in progress"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestCheckPID_InvalidPID(t *testing.T) {
	// Create a temp file with PID 0
	tmpFile := filepath.Join(os.TempDir(), "test-zero-pid")
	err := os.WriteFile(tmpFile, []byte("0"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	result := checkPID(tmpFile)

	if result {
		t.Error("expected checkPID to return false for PID 0")
	}
}

func TestCheckPID_DeadProcess(t *testing.T) {
	// Create a temp file with a PID that doesn't exist
	// Using a very high PID that's unlikely to exist
	tmpFile := filepath.Join(os.TempDir(), "test-dead-pid")
	err := os.WriteFile(tmpFile, []byte("999999999"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	result := checkPID(tmpFile)

	// This should return false since the process doesn't exist
	// (Signal(0) will fail)
	if result {
		t.Error("expected checkPID to return false for non-existent process")
	}
}

func TestRecorder_IsRecordingLocked(t *testing.T) {
	// Clean up any existing PID files
	os.Remove(config.VideoPIDFile)
	os.Remove(config.AudioPIDFile)
	os.Remove(config.WebcamPIDFile)

	rec := New()

	if rec.IsRecordingLocked() {
		t.Error("expected IsRecordingLocked() to return false when no PID files exist")
	}
}

func TestRecorder_MultipleNew(t *testing.T) {
	// Ensure New() can be called multiple times
	rec1 := New()
	rec2 := New()

	if rec1 == nil || rec2 == nil {
		t.Fatal("New() should not return nil")
	}

	// They should be different instances
	if rec1 == rec2 {
		t.Error("expected different Recorder instances")
	}
}

func TestOptionsWithValues(t *testing.T) {
	opts := Options{
		Monitor:      "HDMI-A-1",
		NoAudio:      true,
		NoWebcam:     true,
		HWAccel:      true,
		OutputDir:    "/custom/output",
		WebcamDevice: "/dev/video0",
		WebcamFPS:    30,
		AudioDevice:  "alsa_input.pci-0000_00_1f.3.analog-stereo",
	}

	if opts.Monitor != "HDMI-A-1" {
		t.Errorf("expected Monitor to be 'HDMI-A-1', got %q", opts.Monitor)
	}

	if !opts.NoAudio {
		t.Error("expected NoAudio to be true")
	}

	if !opts.NoWebcam {
		t.Error("expected NoWebcam to be true")
	}

	if !opts.HWAccel {
		t.Error("expected HWAccel to be true")
	}

	if opts.OutputDir != "/custom/output" {
		t.Errorf("expected OutputDir to be '/custom/output', got %q", opts.OutputDir)
	}

	if opts.WebcamDevice != "/dev/video0" {
		t.Errorf("expected WebcamDevice to be '/dev/video0', got %q", opts.WebcamDevice)
	}

	if opts.WebcamFPS != 30 {
		t.Errorf("expected WebcamFPS to be 30, got %d", opts.WebcamFPS)
	}

	if opts.AudioDevice != "alsa_input.pci-0000_00_1f.3.analog-stereo" {
		t.Errorf("expected AudioDevice to be set correctly, got %q", opts.AudioDevice)
	}
}

func TestStopProcess_NonExistentPID(t *testing.T) {
	// Try to stop a process that doesn't exist
	// Using a very high PID that's unlikely to exist
	err := stopProcess(999999999)

	// Should not panic, might return an error
	_ = err
}
