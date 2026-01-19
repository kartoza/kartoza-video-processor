package audio

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/kartoza/kartoza-video-processor/internal/deps"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/notify"
)

// Recorder handles audio recording via PipeWire
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

// Start begins audio recording
func (r *Recorder) Start() error {
	currentOS := deps.DetectOS()
	
	switch currentOS {
	case deps.OSWindows:
		return r.startWindows()
	case deps.OSDarwin:
		return r.startMacOS()
	case deps.OSLinux:
		return r.startLinux()
	default:
		// Unknown OS - try Linux as fallback
		return r.startLinux()
	}
}

// startLinux begins audio recording on Linux using PipeWire
func (r *Recorder) startLinux() error {
	r.cmd = exec.Command("pw-record", "--target", r.device, r.outputFile)
	r.cmd.Stdout = nil
	r.cmd.Stderr = nil

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start audio recording: %w", err)
	}

	r.pid = r.cmd.Process.Pid
	return nil
}

// startWindows begins audio recording on Windows using ffmpeg with dshow
func (r *Recorder) startWindows() error {
	// Use ffmpeg with dshow to record audio on Windows
	// ffmpeg -f dshow -i audio="Microphone" output.wav
	audioDevice := "audio=" + r.device
	if r.device == "@DEFAULT_SOURCE@" || r.device == "" {
		// Use default audio device
		audioDevice = "audio=@device_cm_{33D9A762-90C8-11D0-BD43-00A0C911CE86}\\wave_{00000000-0000-0000-0000-000000000000}"
	}
	
	r.cmd = exec.Command("ffmpeg",
		"-f", "dshow",
		"-i", audioDevice,
		"-y", // Overwrite output
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

// startMacOS begins audio recording on macOS using ffmpeg with avfoundation
func (r *Recorder) startMacOS() error {
	// Use ffmpeg with avfoundation to record audio on macOS
	// ffmpeg -f avfoundation -i ":0" output.wav
	// ":0" means no video, audio device 0 (default microphone)
	audioInput := ":0"
	if r.device != "@DEFAULT_SOURCE@" && r.device != "" {
		// Try to use specified device
		audioInput = ":" + r.device
	}
	
	r.cmd = exec.Command("ffmpeg",
		"-f", "avfoundation",
		"-i", audioInput,
		"-y", // Overwrite output
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

// Processor handles audio post-processing
type Processor struct {
	options models.AudioProcessingOptions
}

// NewProcessor creates a new audio processor
func NewProcessor(opts models.AudioProcessingOptions) *Processor {
	return &Processor{options: opts}
}

// AnalyzeLoudness performs first-pass loudnorm analysis
func (p *Processor) AnalyzeLoudness(inputFile string) (*models.LoudnormStats, error) {
	notify.ProcessingStep("Analyzing audio levels...")

	filter := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f:print_format=json",
		p.options.TargetLoudness,
		p.options.TruePeak,
		p.options.LoudnessRange,
	)

	cmd := exec.Command("ffmpeg",
		"-i", inputFile,
		"-af", filter,
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("loudness analysis failed: %w", err)
	}

	// Extract JSON from ffmpeg output
	stats, err := parseLoudnormOutput(string(output))
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Normalize performs two-pass loudness normalization
func (p *Processor) Normalize(inputFile, outputFile string, stats *models.LoudnormStats) error {
	notify.ProcessingStep("Normalizing audio...")

	filter := fmt.Sprintf(
		"loudnorm=I=%.1f:TP=%.1f:LRA=%.1f:measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s:linear=true:print_format=summary",
		p.options.TargetLoudness,
		p.options.TruePeak,
		p.options.LoudnessRange,
		stats.InputI,
		stats.InputTP,
		stats.InputLRA,
		stats.InputThresh,
	)

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputFile,
		"-af", filter,
		"-c:a", "pcm_s16le",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("normalization failed: %w, output: %s", err, output)
	}

	return nil
}

// Process performs full audio processing pipeline
func (p *Processor) Process(inputFile, outputFile string) error {
	if p.options.NormalizeEnabled {
		stats, err := p.AnalyzeLoudness(inputFile)
		if err != nil {
			notify.Warning("Audio Normalization Warning", "Using original audio")
			return nil
		}

		if err := p.Normalize(inputFile, outputFile, stats); err != nil {
			notify.Warning("Audio Normalization Warning", "Using original audio")
			return nil
		}
	}

	return nil
}

// parseLoudnormOutput extracts loudnorm stats from ffmpeg output
func parseLoudnormOutput(output string) (*models.LoudnormStats, error) {
	// Find JSON block in output
	re := regexp.MustCompile(`\{[^}]+\}`)
	matches := re.FindAllString(output, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no loudnorm stats found in output")
	}

	// Use the last JSON block (should be the loudnorm stats)
	jsonStr := matches[len(matches)-1]

	var stats models.LoudnormStats
	if err := json.Unmarshal([]byte(jsonStr), &stats); err != nil {
		// Try to parse individual values using regex
		stats = extractLoudnormValues(output)
	}

	return &stats, nil
}

// extractLoudnormValues extracts values from ffmpeg output using regex
func extractLoudnormValues(output string) models.LoudnormStats {
	stats := models.LoudnormStats{}

	patterns := map[string]*string{
		`"input_i"\s*:\s*"([^"]+)"`:       &stats.InputI,
		`"input_tp"\s*:\s*"([^"]+)"`:      &stats.InputTP,
		`"input_lra"\s*:\s*"([^"]+)"`:     &stats.InputLRA,
		`"input_thresh"\s*:\s*"([^"]+)"`:  &stats.InputThresh,
		`"output_i"\s*:\s*"([^"]+)"`:      &stats.OutputI,
		`"output_tp"\s*:\s*"([^"]+)"`:     &stats.OutputTP,
		`"output_lra"\s*:\s*"([^"]+)"`:    &stats.OutputLRA,
		`"output_thresh"\s*:\s*"([^"]+)"`: &stats.OutputThresh,
		`"target_offset"\s*:\s*"([^"]+)"`: &stats.TargetOffset,
	}

	for pattern, target := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			*target = matches[1]
		}
	}

	return stats
}

