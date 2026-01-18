package recorder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/kartoza/kartoza-video-processor/internal/audio"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/merger"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/monitor"
	"github.com/kartoza/kartoza-video-processor/internal/notify"
	"github.com/kartoza/kartoza-video-processor/internal/webcam"
)

// Options for starting a recording
type Options struct {
	Monitor      string
	NoAudio      bool
	NoWebcam     bool
	HWAccel      bool
	OutputDir    string
	WebcamDevice string
	WebcamFPS    int
	AudioDevice  string
}

// Recorder manages screen recording sessions
type Recorder struct {
	config *config.Config
}

// New creates a new Recorder
func New() *Recorder {
	cfg, _ := config.Load()
	return &Recorder{config: cfg}
}

// IsRecording checks if any recording is currently in progress
func (r *Recorder) IsRecording() bool {
	return checkPID(config.VideoPIDFile) ||
		checkPID(config.AudioPIDFile) ||
		checkPID(config.WebcamPIDFile)
}

// GetStatus returns the current recording status
func (r *Recorder) GetStatus() models.RecordingStatus {
	status := models.RecordingStatus{
		IsRecording: r.IsRecording(),
	}

	if status.IsRecording {
		status.VideoPID = readPID(config.VideoPIDFile)
		status.AudioPID = readPID(config.AudioPIDFile)
		status.WebcamPID = readPID(config.WebcamPIDFile)
		status.VideoFile = readPath(config.VideoPathFile)
		status.AudioFile = readPath(config.AudioPathFile)
		status.WebcamFile = readPath(config.WebcamPathFile)

		// Read start time from status file
		if data, err := os.ReadFile(config.StatusFile); err == nil {
			if t, err := time.Parse(time.RFC3339, string(data)); err == nil {
				status.StartTime = t
			}
		}
	}

	return status
}

// Start starts a recording with default options
func (r *Recorder) Start() error {
	return r.StartWithOptions(Options{})
}

// StartWithOptions starts a recording with the given options
func (r *Recorder) StartWithOptions(opts Options) error {
	if r.IsRecording() {
		return fmt.Errorf("recording already in progress")
	}

	// Ensure output directory exists
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = config.GetDefaultVideosDir()
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get monitor to record
	monitorName := opts.Monitor
	if monitorName == "" {
		var err error
		monitorName, err = monitor.GetMouseMonitor()
		if err != nil {
			// Fallback to first monitor
			monitors, err := monitor.ListMonitors()
			if err != nil || len(monitors) == 0 {
				return fmt.Errorf("no monitors found")
			}
			monitorName = monitors[0].Name
		}
	}

	// Generate filenames
	timestamp := time.Now().Format("20060102-150405")
	videoFile := filepath.Join(outputDir, fmt.Sprintf("screenrecording-%s-%s.mp4", monitorName, timestamp))
	audioFile := filepath.Join(outputDir, fmt.Sprintf("screenrecording-%s-%s.wav", monitorName, timestamp))
	webcamFile := filepath.Join(outputDir, fmt.Sprintf("screenrecording-webcam-%s-%s.mp4", monitorName, timestamp))

	// Start video recording
	if err := r.startVideoRecording(videoFile, monitorName, opts.HWAccel); err != nil {
		return err
	}

	// Store file paths
	os.WriteFile(config.VideoPathFile, []byte(videoFile), 0644)
	os.WriteFile(config.StatusFile, []byte(time.Now().Format(time.RFC3339)), 0644)

	// Start audio recording
	if !opts.NoAudio {
		audioDevice := opts.AudioDevice
		if audioDevice == "" {
			audioDevice = "@DEFAULT_SOURCE@"
		}

		audioRecorder := audio.NewRecorder(audioDevice, audioFile)
		if err := audioRecorder.Start(); err != nil {
			notify.Warning("Audio Recording", "Failed to start audio recording")
		} else {
			writePID(config.AudioPIDFile, audioRecorder.PID())
			os.WriteFile(config.AudioPathFile, []byte(audioFile), 0644)
		}
	}

	// Start webcam recording
	if !opts.NoWebcam {
		webcamOpts := webcam.Options{
			Device:     opts.WebcamDevice,
			FPS:        opts.WebcamFPS,
			Resolution: "1920x1080",
			OutputFile: webcamFile,
		}

		if webcamOpts.FPS == 0 {
			webcamOpts.FPS = 60
		}

		cam := webcam.New(webcamOpts)
		if err := cam.Start(); err != nil {
			notify.Info("Webcam Recording", "No webcam detected")
		} else {
			writePID(config.WebcamPIDFile, cam.PID())
			os.WriteFile(config.WebcamPathFile, []byte(webcamFile), 0644)
		}
	}

	notify.RecordingStarted(monitorName)
	return nil
}

// Stop stops the current recording
func (r *Recorder) Stop() error {
	if !r.IsRecording() {
		return fmt.Errorf("no recording in progress")
	}

	// Stop video recording
	if pid := readPID(config.VideoPIDFile); pid > 0 {
		stopProcess(pid)
		os.Remove(config.VideoPIDFile)
	}

	// Stop audio recording
	if pid := readPID(config.AudioPIDFile); pid > 0 {
		stopProcess(pid)
		os.Remove(config.AudioPIDFile)
	}

	// Stop webcam recording
	if pid := readPID(config.WebcamPIDFile); pid > 0 {
		stopProcess(pid)
		os.Remove(config.WebcamPIDFile)
	}

	notify.RecordingStopped()

	// Wait for files to be fully written
	time.Sleep(2 * time.Second)

	// Merge recordings in background
	go r.processRecordings()

	return nil
}

// startVideoRecording starts the screen recording process
func (r *Recorder) startVideoRecording(outputFile, monitorName string, hwAccel bool) error {
	args := []string{}

	// Software encoding by default (more compatible)
	if !hwAccel {
		args = append(args, "--no-hw")
	}

	args = append(args,
		"--output="+monitorName,
		"--filename="+outputFile,
		"--encode-pixfmt", "yuv420p",
	)

	cmd := exec.Command("wl-screenrec", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start wl-screenrec: %w", err)
	}

	// Wait a moment to check if it started successfully
	time.Sleep(time.Second)

	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return fmt.Errorf("wl-screenrec failed to start")
	}

	writePID(config.VideoPIDFile, cmd.Process.Pid)
	return nil
}

// processRecordings merges the recorded files
func (r *Recorder) processRecordings() {
	videoFile := readPath(config.VideoPathFile)
	audioFile := readPath(config.AudioPathFile)
	webcamFile := readPath(config.WebcamPathFile)

	if videoFile == "" || audioFile == "" {
		return
	}

	m := merger.New(r.config.AudioProcessing)
	_, err := m.Merge(merger.MergeOptions{
		VideoFile:      videoFile,
		AudioFile:      audioFile,
		WebcamFile:     webcamFile,
		CreateVertical: webcamFile != "",
	})

	if err != nil {
		notify.Error("Recording Error", "Failed to merge recordings")
	}

	// Clean up path files
	os.Remove(config.VideoPathFile)
	os.Remove(config.AudioPathFile)
	os.Remove(config.WebcamPathFile)
	os.Remove(config.StatusFile)
}

// Helper functions

func checkPID(pidFile string) bool {
	pid := readPID(pidFile)
	if pid <= 0 {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func readPID(pidFile string) int {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}

	return pid
}

func writePID(pidFile string, pid int) {
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func readPath(pathFile string) string {
	data, err := os.ReadFile(pathFile)
	if err != nil {
		return ""
	}
	return string(data)
}

func stopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Send SIGINT for graceful shutdown
	if err := process.Signal(syscall.SIGINT); err != nil {
		// If SIGINT fails, try SIGTERM
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return process.Kill()
		}
	}

	// Wait for process to finish (max 5 seconds)
	for i := 0; i < 10; i++ {
		if err := process.Signal(syscall.Signal(0)); err != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
