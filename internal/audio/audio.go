package audio

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/kartoza/kartoza-screencaster/internal/models"
	"github.com/kartoza/kartoza-screencaster/internal/notify"
)

// Note: Recorder is defined in platform-specific files:
// - audio_linux.go: uses pw-record (PipeWire)
// - audio_darwin.go: uses ffmpeg with avfoundation
// - audio_windows.go: uses ffmpeg with dshow

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
	_ = notify.ProcessingStep("Analyzing audio levels...")

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
	_ = notify.ProcessingStep("Normalizing audio...")

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
			_ = notify.Warning("Audio Normalization Warning", "Using original audio")
			return nil
		}

		if err := p.Normalize(inputFile, outputFile, stats); err != nil {
			_ = notify.Warning("Audio Normalization Warning", "Using original audio")
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

