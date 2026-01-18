package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kartoza/kartoza-video-processor/internal/models"
)

const (
	// DefaultConfigDir is the default configuration directory
	DefaultConfigDir = ".config/kartoza-video-processor"
	// DefaultVideosDir is the default output directory for recordings
	DefaultVideosDir = "Videos/Screencasts"
	// ConfigFileName is the name of the configuration file
	ConfigFileName = "config.json"
)

// Paths for PID and state files
const (
	VideoPIDFile   = "/tmp/kartoza-video.pid"
	AudioPIDFile   = "/tmp/kartoza-audio.pid"
	WebcamPIDFile  = "/tmp/kartoza-webcam.pid"
	StatusFile     = "/tmp/kartoza-video.status"
	VideoPathFile  = "/tmp/kartoza-video.path"
	AudioPathFile  = "/tmp/kartoza-audio.path"
	WebcamPathFile = "/tmp/kartoza-webcam.path"
	StopSignalFile = "/tmp/kartoza-video.stop"
)

// LogoSelection holds the selected logos for a recording
type LogoSelection struct {
	LeftLogo   string `json:"left_logo,omitempty"`   // Top-left logo
	RightLogo  string `json:"right_logo,omitempty"`  // Top-right logo
	BottomLogo string `json:"bottom_logo,omitempty"` // Lower third logo
}

// Config holds the application configuration
type Config struct {
	OutputDir        string                        `json:"output_dir"`
	DefaultOptions   models.RecordingOptions       `json:"default_options"`
	AudioProcessing  models.AudioProcessingOptions `json:"audio_processing"`
	Topics           []models.Topic                `json:"topics,omitempty"`
	DefaultPresenter string                        `json:"default_presenter,omitempty"`
	RecordingCounter int                           `json:"recording_counter"`

	// Logo settings
	LogoDirectory  string        `json:"logo_directory,omitempty"`   // Directory to browse for logos
	LastUsedLogos  LogoSelection `json:"last_used_logos,omitempty"`  // Last used logo selection
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		OutputDir:       GetDefaultVideosDir(),
		DefaultOptions:  models.DefaultRecordingOptions(),
		AudioProcessing: models.DefaultAudioProcessingOptions(),
	}
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfigDir
	}
	return filepath.Join(home, DefaultConfigDir)
}

// GetDefaultVideosDir returns the default videos directory path
func GetDefaultVideosDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultVideosDir
	}
	return filepath.Join(home, DefaultVideosDir)
}

// EnsureDirectories creates the necessary directories
func EnsureDirectories() error {
	dirs := []string{
		GetConfigDir(),
		GetDefaultVideosDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	configPath := filepath.Join(GetConfigDir(), ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			cfg := DefaultConfig()
			return &cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func Save(cfg *Config) error {
	if err := EnsureDirectories(); err != nil {
		return err
	}

	configPath := filepath.Join(GetConfigDir(), ConfigFileName)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetNextRecordingNumber returns the next recording number and increments the counter
func GetNextRecordingNumber() (int, error) {
	cfg, err := Load()
	if err != nil {
		return 1, err
	}

	cfg.RecordingCounter++
	if err := Save(cfg); err != nil {
		return cfg.RecordingCounter, err
	}

	return cfg.RecordingCounter, nil
}

// GetCurrentRecordingNumber returns the current recording counter without incrementing
func GetCurrentRecordingNumber() int {
	cfg, err := Load()
	if err != nil {
		return 1
	}
	return cfg.RecordingCounter + 1 // Return what the next number will be
}
