package recorder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
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
	Monitor        string
	NoAudio        bool
	NoWebcam       bool
	HWAccel        bool
	OutputDir      string
	WebcamDevice   string
	WebcamFPS      int
	AudioDevice    string
	Metadata       *models.RecordingMetadata
	RecordingInfo  *models.RecordingInfo
	CreateVertical bool
	LogoSelection  config.LogoSelection
}

// recorderInstance holds a single recorder's state
type recorderInstance struct {
	name    string
	cmd     *exec.Cmd
	pid     int
	file    string
	err     error
	started bool
}

// Recorder manages screen recording sessions
type Recorder struct {
	config *config.Config
	mu     sync.Mutex

	// Active recorder instances
	video  *recorderInstance
	audio  *recorderInstance
	webcam *recorderInstance

	// Recording metadata
	recordingInfo  *models.RecordingInfo
	createVertical bool
	logoSelection  config.LogoSelection

	// Synchronization
	startBarrier chan struct{}
	stopSignal   chan struct{}
	wg           sync.WaitGroup
}

// New creates a new Recorder
func New() *Recorder {
	cfg, _ := config.Load()
	return &Recorder{config: cfg}
}

// IsRecording checks if any recording is currently in progress
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return checkPID(config.VideoPIDFile) ||
		checkPID(config.AudioPIDFile) ||
		checkPID(config.WebcamPIDFile)
}

// GetStatus returns the current recording status
func (r *Recorder) GetStatus() models.RecordingStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	status := models.RecordingStatus{
		IsRecording: checkPID(config.VideoPIDFile) ||
			checkPID(config.AudioPIDFile) ||
			checkPID(config.WebcamPIDFile),
	}

	if status.IsRecording {
		status.VideoPID = readPID(config.VideoPIDFile)
		status.AudioPID = readPID(config.AudioPIDFile)
		status.WebcamPID = readPID(config.WebcamPIDFile)
		status.VideoFile = readPath(config.VideoPathFile)
		status.AudioFile = readPath(config.AudioPathFile)
		status.WebcamFile = readPath(config.WebcamPathFile)

		// Get monitor name - from instance if available, else extract from filename
		if r.video != nil {
			status.Monitor = r.video.name
		} else if status.VideoFile != "" {
			// Extract monitor name from filename: screenrecording-<monitor>-<timestamp>.mp4
			status.Monitor = extractMonitorFromPath(status.VideoFile)
		}

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
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.IsRecordingLocked() {
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

	// Store recording info and settings
	r.recordingInfo = opts.RecordingInfo
	r.createVertical = opts.CreateVertical
	r.logoSelection = opts.LogoSelection

	// Generate simplified filenames (no timestamp needed since they're in unique folder)
	videoFile := filepath.Join(outputDir, "screen.mp4")
	audioFile := filepath.Join(outputDir, "audio.wav")
	webcamFile := filepath.Join(outputDir, "webcam.mp4")

	// Initialize recorder instances
	r.video = &recorderInstance{name: monitorName, file: videoFile}
	if !opts.NoAudio {
		r.audio = &recorderInstance{name: "audio", file: audioFile}
	}
	if !opts.NoWebcam {
		r.webcam = &recorderInstance{name: "webcam", file: webcamFile}
	}

	// Update recording info with file paths
	if r.recordingInfo != nil {
		r.recordingInfo.Files.VideoFile = videoFile
		if r.audio != nil {
			r.recordingInfo.Files.AudioFile = audioFile
		}
		if r.webcam != nil {
			r.recordingInfo.Files.WebcamFile = webcamFile
		}
		// Save updated file paths
		r.recordingInfo.Save()
	}

	// Create synchronization primitives
	r.startBarrier = make(chan struct{})
	r.stopSignal = make(chan struct{})

	// Count how many recorders we're starting
	numRecorders := 1 // video is always started
	if r.audio != nil {
		numRecorders++
	}
	if r.webcam != nil {
		numRecorders++
	}

	// Channel to collect readiness and started confirmation
	ready := make(chan string, numRecorders)
	started := make(chan string, numRecorders)
	errors := make(chan error, numRecorders)

	// Start video recorder in goroutine
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.startVideoRecorder(opts.HWAccel, ready, started, errors)
	}()

	// Start audio recorder in goroutine
	if r.audio != nil {
		audioDevice := opts.AudioDevice
		if audioDevice == "" {
			audioDevice = "@DEFAULT_SOURCE@"
		}
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.startAudioRecorder(audioDevice, ready, started, errors)
		}()
	}

	// Start webcam recorder in goroutine
	if r.webcam != nil {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.startWebcamRecorder(opts, ready, started, errors)
		}()
	}

	// Wait for all recorders to be ready (with timeout)
	readyCount := 0
	timeout := time.After(5 * time.Second)

	for readyCount < numRecorders {
		select {
		case name := <-ready:
			readyCount++
			_ = name // Could log which recorder is ready
		case err := <-errors:
			// Non-fatal errors for audio/webcam
			notify.Warning("Recorder Warning", err.Error())
			numRecorders-- // Reduce expected count
		case <-timeout:
			// Proceed with what we have
			break
		}
	}

	// All ready - signal them to start simultaneously
	close(r.startBarrier)

	// Wait for recorders to actually start (with timeout)
	startedCount := 0
	expectedStarted := numRecorders
	startTimeout := time.After(5 * time.Second)

waitStarted:
	for startedCount < expectedStarted {
		select {
		case name := <-started:
			startedCount++
			_ = name
		case err := <-errors:
			// Error during start - reduce expected count
			notify.Warning("Recorder Error", err.Error())
			expectedStarted--
		case <-startTimeout:
			break waitStarted
		}
	}

	// Record start time
	startTime := time.Now()
	os.WriteFile(config.StatusFile, []byte(startTime.Format(time.RFC3339)), 0644)

	// Store file paths and PIDs
	if r.video != nil && r.video.started {
		writePID(config.VideoPIDFile, r.video.pid)
		os.WriteFile(config.VideoPathFile, []byte(r.video.file), 0644)
	}
	if r.audio != nil && r.audio.started {
		writePID(config.AudioPIDFile, r.audio.pid)
		os.WriteFile(config.AudioPathFile, []byte(r.audio.file), 0644)
	}
	if r.webcam != nil && r.webcam.started {
		writePID(config.WebcamPIDFile, r.webcam.pid)
		os.WriteFile(config.WebcamPathFile, []byte(r.webcam.file), 0644)
	}

	notify.RecordingStarted(monitorName)
	return nil
}

// startVideoRecorder starts the video recorder and waits for the start signal
func (r *Recorder) startVideoRecorder(hwAccel bool, ready, started chan<- string, errors chan<- error) {
	args := []string{}

	// Software encoding by default (more compatible)
	if !hwAccel {
		args = append(args, "--no-hw")
	}

	args = append(args,
		"--output="+r.video.name,
		"--filename="+r.video.file,
		"--encode-pixfmt", "yuv420p",
	)

	r.video.cmd = exec.Command("wl-screenrec", args...)
	r.video.cmd.Stdout = nil
	r.video.cmd.Stderr = nil

	// Signal we're ready
	ready <- "video"

	// Wait for synchronized start signal
	<-r.startBarrier

	if err := r.video.cmd.Start(); err != nil {
		r.video.err = fmt.Errorf("failed to start wl-screenrec: %w", err)
		errors <- r.video.err
		return
	}

	r.video.pid = r.video.cmd.Process.Pid
	r.video.started = true

	// Signal that we've started
	started <- "video"

	// Wait for stop signal or process exit
	done := make(chan error, 1)
	go func() {
		done <- r.video.cmd.Wait()
	}()

	select {
	case <-r.stopSignal:
		// Stop requested
	case err := <-done:
		if err != nil {
			r.video.err = err
		}
	}
}

// startAudioRecorder starts the audio recorder and waits for the start signal
func (r *Recorder) startAudioRecorder(device string, ready, started chan<- string, errors chan<- error) {
	audioRecorder := audio.NewRecorder(device, r.audio.file)

	// Signal we're ready
	ready <- "audio"

	// Wait for synchronized start signal
	<-r.startBarrier

	if err := audioRecorder.Start(); err != nil {
		r.audio.err = fmt.Errorf("failed to start audio recording: %w", err)
		errors <- r.audio.err
		return
	}

	r.audio.pid = audioRecorder.PID()
	r.audio.started = true

	// Signal that we've started
	started <- "audio"

	// Wait for stop signal
	<-r.stopSignal
}

// startWebcamRecorder starts the webcam recorder and waits for the start signal
func (r *Recorder) startWebcamRecorder(opts Options, ready, started chan<- string, errors chan<- error) {
	webcamOpts := webcam.Options{
		Device:     opts.WebcamDevice,
		FPS:        opts.WebcamFPS,
		Resolution: "1920x1080",
		OutputFile: r.webcam.file,
	}

	if webcamOpts.FPS == 0 {
		webcamOpts.FPS = 60
	}

	cam := webcam.New(webcamOpts)

	// Signal we're ready
	ready <- "webcam"

	// Wait for synchronized start signal
	<-r.startBarrier

	if err := cam.Start(); err != nil {
		r.webcam.err = fmt.Errorf("no webcam detected: %w", err)
		errors <- r.webcam.err
		return
	}

	r.webcam.pid = cam.PID()
	r.webcam.started = true

	// Signal that we've started
	started <- "webcam"

	// Wait for stop signal
	<-r.stopSignal
}

// Stop stops the current recording
func (r *Recorder) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.IsRecordingLocked() {
		return fmt.Errorf("no recording in progress")
	}

	// Signal all recorders to stop simultaneously
	if r.stopSignal != nil {
		close(r.stopSignal)
	}

	// Stop all processes simultaneously using goroutines
	var stopWg sync.WaitGroup

	// Stop video
	if pid := readPID(config.VideoPIDFile); pid > 0 {
		stopWg.Add(1)
		go func(p int) {
			defer stopWg.Done()
			stopProcess(p)
			os.Remove(config.VideoPIDFile)
		}(pid)
	}

	// Stop audio
	if pid := readPID(config.AudioPIDFile); pid > 0 {
		stopWg.Add(1)
		go func(p int) {
			defer stopWg.Done()
			stopProcess(p)
			os.Remove(config.AudioPIDFile)
		}(pid)
	}

	// Stop webcam
	if pid := readPID(config.WebcamPIDFile); pid > 0 {
		stopWg.Add(1)
		go func(p int) {
			defer stopWg.Done()
			stopProcess(p)
			os.Remove(config.WebcamPIDFile)
		}(pid)
	}

	// Wait for all stop operations to complete
	stopWg.Wait()

	// Wait for recorder goroutines to finish
	r.wg.Wait()

	notify.RecordingStopped()

	// Wait for files to be fully written
	time.Sleep(2 * time.Second)

	// Update recording info with end time and file sizes
	if r.recordingInfo != nil {
		r.recordingInfo.SetEndTime(time.Now())
		r.recordingInfo.UpdateFileSizes()
		r.recordingInfo.Save()
	}

	// Merge recordings in background
	go r.processRecordings()

	// Clear instances
	r.video = nil
	r.audio = nil
	r.webcam = nil

	return nil
}

// IsRecordingLocked checks recording status without locking (internal use)
func (r *Recorder) IsRecordingLocked() bool {
	return checkPID(config.VideoPIDFile) ||
		checkPID(config.AudioPIDFile) ||
		checkPID(config.WebcamPIDFile)
}

// ProgressUpdate represents a progress update from the processing pipeline
type ProgressUpdate struct {
	Step      int     // Step index (0-based, add 1 for TUI which has "stopping recorders" as step 0)
	Completed bool
	Skipped   bool
	Error     error
	Percent   float64 // Progress percentage (0-100), -1 means not a percent update
}

// ProcessWithProgress processes recordings and sends progress updates to the channel
func (r *Recorder) ProcessWithProgress(progressChan chan<- ProgressUpdate) {
	defer close(progressChan)

	// Get file paths from recording info or fallback to path files
	var videoFile, audioFile, webcamFile string
	if r.recordingInfo != nil {
		videoFile = r.recordingInfo.Files.VideoFile
		audioFile = r.recordingInfo.Files.AudioFile
		webcamFile = r.recordingInfo.Files.WebcamFile
	} else {
		videoFile = readPath(config.VideoPathFile)
		audioFile = readPath(config.AudioPathFile)
		webcamFile = readPath(config.WebcamPathFile)
	}

	if videoFile == "" && audioFile == "" {
		return
	}

	m := merger.New(r.config.AudioProcessing)

	// Set up progress callback
	m.SetProgressCallback(func(step merger.ProcessingStep, completed bool, skipped bool, err error) {
		// Map merger steps to TUI steps (add 1 because TUI step 0 is "stopping recorders")
		tuiStep := int(step) + 1
		progressChan <- ProgressUpdate{
			Step:      tuiStep,
			Completed: completed,
			Skipped:   skipped,
			Error:     err,
			Percent:   -1, // Not a percent update
		}
	})

	// Set up percent callback for progress bars
	m.SetPercentCallback(func(step merger.ProcessingStep, percent float64) {
		tuiStep := int(step) + 1
		progressChan <- ProgressUpdate{
			Step:    tuiStep,
			Percent: percent,
		}
	})

	// Build merge options
	mergeOpts := merger.MergeOptions{
		VideoFile:      videoFile,
		AudioFile:      audioFile,
		WebcamFile:     webcamFile,
		CreateVertical: r.createVertical && webcamFile != "",
	}

	// Add logo options from the recording's logo selection
	mergeOpts.ProductLogo1 = r.logoSelection.LeftLogo
	mergeOpts.ProductLogo2 = r.logoSelection.RightLogo
	mergeOpts.CompanyLogo = r.logoSelection.BottomLogo
	mergeOpts.TitleColor = r.logoSelection.TitleColor
	mergeOpts.GifLoopMode = r.logoSelection.GifLoopMode
	// Check if any logos are configured
	mergeOpts.AddLogos = mergeOpts.ProductLogo1 != "" || mergeOpts.ProductLogo2 != "" || mergeOpts.CompanyLogo != ""

	// Get video title and output directory from recording info
	if r.recordingInfo != nil {
		mergeOpts.VideoTitle = r.recordingInfo.Metadata.Title
		mergeOpts.OutputDir = r.recordingInfo.Files.FolderPath
	}

	mergeResult, err := m.Merge(mergeOpts)

	if err != nil {
		notify.Error("Recording Error", "Failed to merge recordings")
		if r.recordingInfo != nil {
			r.recordingInfo.Processing.Errors = append(r.recordingInfo.Processing.Errors, err.Error())
		}
	}

	// Update recording info with merged file paths and processing info
	if r.recordingInfo != nil {
		if mergeResult.MergedFile != "" {
			r.recordingInfo.Files.MergedFile = mergeResult.MergedFile
		}
		if mergeResult.VerticalFile != "" {
			r.recordingInfo.Files.VerticalFile = mergeResult.VerticalFile
		}
		r.recordingInfo.Processing.NormalizeApplied = mergeResult.NormalizeApplied
		r.recordingInfo.Processing.VerticalCreated = mergeResult.VerticalFile != ""
		r.recordingInfo.UpdateFileSizes()

		// Update video metadata (resolution, fps, aspect ratio)
		r.recordingInfo.UpdateVideoMetadata(func(filepath string) (*models.VideoFileMetadata, error) {
			meta, err := webcam.GetFullVideoInfo(filepath)
			if err != nil {
				return nil, err
			}
			return &models.VideoFileMetadata{
				Width:       meta.Width,
				Height:      meta.Height,
				FPS:         meta.FPS,
				AspectRatio: meta.AspectRatio,
				Duration:    meta.Duration,
				Codec:       meta.Codec,
			}, nil
		})

		r.recordingInfo.Save()
	}

	// Clean up path files
	os.Remove(config.VideoPathFile)
	os.Remove(config.AudioPathFile)
	os.Remove(config.WebcamPathFile)
	os.Remove(config.StatusFile)
}

// processRecordings merges the recorded files (legacy method without progress)
func (r *Recorder) processRecordings() {
	progressChan := make(chan ProgressUpdate, 10)
	go func() {
		// Drain the channel
		for range progressChan {
		}
	}()
	r.ProcessWithProgress(progressChan)
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

// extractMonitorFromPath extracts monitor name from filename like:
// screenrecording-HDMI-A-1-20260118-103045.mp4 -> HDMI-A-1
func extractMonitorFromPath(filePath string) string {
	base := filepath.Base(filePath)
	// Remove extension
	name := base[:len(base)-len(filepath.Ext(base))]

	// Expected format: screenrecording-<monitor>-<timestamp>
	const prefix = "screenrecording-"
	if len(name) <= len(prefix) {
		return ""
	}
	name = name[len(prefix):]

	// Find the timestamp part (format: YYYYMMDD-HHMMSS)
	// Look for pattern: -NNNNNNNN-NNNNNN at the end
	// The timestamp is 15 chars: YYYYMMDD-HHMMSS
	if len(name) > 16 && name[len(name)-7] == '-' {
		// Find where the date starts (should be preceded by a -)
		for i := len(name) - 16; i >= 0; i-- {
			if name[i] == '-' && i > 0 {
				return name[:i]
			}
		}
	}

	return name
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
