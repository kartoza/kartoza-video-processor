package models

import "time"

// RecordingState represents the current state of a recording session
type RecordingState string

const (
	StateIdle       RecordingState = "idle"
	StateRecording  RecordingState = "recording"
	StateProcessing RecordingState = "processing"
	StateError      RecordingState = "error"
)

// RecordingSession represents an active or completed recording session
type RecordingSession struct {
	ID           string           `json:"id"`
	State        RecordingState   `json:"state"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      time.Time        `json:"end_time,omitempty"`
	Duration     time.Duration    `json:"duration,omitempty"`
	Monitor      string           `json:"monitor"`
	VideoFile    string           `json:"video_file"`
	AudioFile    string           `json:"audio_file"`
	WebcamFile   string           `json:"webcam_file,omitempty"`
	MergedFile   string           `json:"merged_file,omitempty"`
	VerticalFile string           `json:"vertical_file,omitempty"`
	Options      RecordingOptions `json:"options"`
	Error        string           `json:"error,omitempty"`
}

// RecordingOptions contains configuration for a recording session
type RecordingOptions struct {
	Monitor          string `json:"monitor"`
	OutputDir        string `json:"output_dir"`
	NoAudio          bool   `json:"no_audio"`
	NoWebcam         bool   `json:"no_webcam"`
	NoScreen         bool   `json:"no_screen"`
	HWAccel          bool   `json:"hw_accel"`
	WebcamDevice     string `json:"webcam_device"`
	WebcamFPS        int    `json:"webcam_fps"`
	WebcamResolution string `json:"webcam_resolution"`
	AudioDevice      string `json:"audio_device"`
	DenoiseAudio     bool   `json:"denoise_audio"`
	NormalizeAudio   bool   `json:"normalize_audio"`
	CreateVertical   bool   `json:"create_vertical"`
}

// DefaultRecordingOptions returns the default recording options
func DefaultRecordingOptions() RecordingOptions {
	return RecordingOptions{
		OutputDir:        "",
		NoAudio:          false,
		NoWebcam:         false,
		HWAccel:          false,
		WebcamDevice:     "",
		WebcamFPS:        60,
		WebcamResolution: "1920x1080",
		AudioDevice:      "@DEFAULT_SOURCE@",
		DenoiseAudio:     true,
		NormalizeAudio:   true,
		CreateVertical:   true,
	}
}

// RecordingStatus is used for CLI/API status responses
type RecordingStatus struct {
	IsRecording bool      `json:"is_recording"`
	IsPaused    bool      `json:"is_paused"`
	CurrentPart int       `json:"current_part,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	Monitor     string    `json:"monitor,omitempty"`
	VideoFile   string    `json:"video_file,omitempty"`
	AudioFile   string    `json:"audio_file,omitempty"`
	WebcamFile  string    `json:"webcam_file,omitempty"`
	VideoPID    int       `json:"video_pid,omitempty"`
	AudioPID    int       `json:"audio_pid,omitempty"`
	WebcamPID   int       `json:"webcam_pid,omitempty"`
}
