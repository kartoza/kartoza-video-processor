package audio

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

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

// Processor handles audio post-processing
type Processor struct {
	options models.AudioProcessingOptions
}

// NewProcessor creates a new audio processor
func NewProcessor(opts models.AudioProcessingOptions) *Processor {
	return &Processor{options: opts}
}

// Denoise removes background noise from audio
func (p *Processor) Denoise(inputFile, outputFile string) error {
	notify.ProcessingStep("Removing background noise...")

	// Build filter chain
	// - highpass: Remove low-frequency rumble
	// - afftdn: FFT-based denoiser for constant background noise
	filter := fmt.Sprintf("highpass=f=%d,afftdn=nf=%d:tn=%d",
		p.options.HighpassFreq,
		p.options.NoiseFloor,
		boolToInt(p.options.TrackNoise),
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
		return fmt.Errorf("noise reduction failed: %w, output: %s", err, output)
	}

	return nil
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
	currentFile := inputFile
	tempDenoised := strings.TrimSuffix(outputFile, ".wav") + "-denoised.wav"

	// Step 1: Denoise if enabled
	if p.options.DenoiseEnabled {
		if err := p.Denoise(currentFile, tempDenoised); err != nil {
			notify.Warning("Noise Reduction Warning", "Skipping noise reduction")
		} else {
			currentFile = tempDenoised
		}
	}

	// Step 2: Normalize if enabled
	if p.options.NormalizeEnabled {
		stats, err := p.AnalyzeLoudness(currentFile)
		if err != nil {
			notify.Warning("Audio Normalization Warning", "Using original audio")
			return nil
		}

		if err := p.Normalize(currentFile, outputFile, stats); err != nil {
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
