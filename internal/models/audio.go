package models

// LoudnormStats contains the measured audio levels from ffmpeg loudnorm analysis
type LoudnormStats struct {
	InputI       string `json:"input_i"`
	InputTP      string `json:"input_tp"`
	InputLRA     string `json:"input_lra"`
	InputThresh  string `json:"input_thresh"`
	OutputI      string `json:"output_i"`
	OutputTP     string `json:"output_tp"`
	OutputLRA    string `json:"output_lra"`
	OutputThresh string `json:"output_thresh"`
	TargetOffset string `json:"target_offset"`
}

// AudioProcessingOptions contains options for audio post-processing
type AudioProcessingOptions struct {
	// NormalizeEnabled enables EBU R128 loudness normalization
	NormalizeEnabled bool
	// TargetLoudness is the target integrated loudness in LUFS
	TargetLoudness float64
	// TruePeak is the maximum true peak level in dB
	TruePeak float64
	// LoudnessRange is the target loudness range
	LoudnessRange float64
}

// DefaultAudioProcessingOptions returns sensible defaults for audio processing
func DefaultAudioProcessingOptions() AudioProcessingOptions {
	return AudioProcessingOptions{
		NormalizeEnabled: true,
		TargetLoudness:   -14.0, // Louder than broadcast, good for screen recordings
		TruePeak:         -1.5,  // Prevents clipping
		LoudnessRange:    11.0,  // Preserves dynamic range
	}
}
