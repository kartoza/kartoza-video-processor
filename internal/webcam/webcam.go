package webcam

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Note: Webcam struct and recording methods are defined in platform-specific files:
// - webcam_linux.go: uses ffmpeg with v4l2
// - webcam_darwin.go: uses ffmpeg with avfoundation
// - webcam_windows.go: uses ffmpeg with dshow

// Options for webcam recording
type Options struct {
	Device     string
	FPS        int
	Resolution string
	OutputFile string
}

// DefaultOptions returns default webcam recording options
func DefaultOptions() Options {
	return Options{
		Device:     "",
		FPS:        60,
		Resolution: "1920x1080",
	}
}

// GetVideoInfo returns video dimensions from a file
func GetVideoInfo(filepath string) (width, height int, err error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
		filepath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get video info: %w", err)
	}

	// Parse "width,height" format
	var w, h int
	_, err = fmt.Sscanf(string(output), "%d,%d", &w, &h)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse video dimensions: %w", err)
	}

	return w, h, nil
}

// VideoMetadata contains comprehensive video file information
type VideoMetadata struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	FPS         float64 `json:"fps"`
	AspectRatio string  `json:"aspect_ratio"`
	Duration    float64 `json:"duration_seconds"`
	Codec       string  `json:"codec"`
	Bitrate     int64   `json:"bitrate,omitempty"`
}

// GetFullVideoInfo returns comprehensive video metadata from a file
func GetFullVideoInfo(filepath string) (*VideoMetadata, error) {
	// Get width, height, fps, duration, codec using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate,codec_name,bit_rate:format=duration",
		"-of", "json",
		filepath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// Parse JSON output
	var probeResult struct {
		Streams []struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
			CodecName  string `json:"codec_name"`
			BitRate    string `json:"bit_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if len(probeResult.Streams) == 0 {
		return nil, fmt.Errorf("no video streams found")
	}

	stream := probeResult.Streams[0]
	meta := &VideoMetadata{
		Width:  stream.Width,
		Height: stream.Height,
		Codec:  stream.CodecName,
	}

	// Parse frame rate (format: "num/den")
	if stream.RFrameRate != "" {
		var num, den int
		if _, err := fmt.Sscanf(stream.RFrameRate, "%d/%d", &num, &den); err == nil && den > 0 {
			meta.FPS = float64(num) / float64(den)
		}
	}

	// Parse duration
	if probeResult.Format.Duration != "" {
		fmt.Sscanf(probeResult.Format.Duration, "%f", &meta.Duration)
	}

	// Parse bitrate
	if stream.BitRate != "" {
		fmt.Sscanf(stream.BitRate, "%d", &meta.Bitrate)
	}

	// Calculate aspect ratio
	meta.AspectRatio = calculateAspectRatio(stream.Width, stream.Height)

	return meta, nil
}

// calculateAspectRatio returns a human-readable aspect ratio string
func calculateAspectRatio(width, height int) string {
	if width == 0 || height == 0 {
		return "unknown"
	}

	// Find GCD
	gcd := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}

	g := gcd(width, height)
	w := width / g
	h := height / g

	// Common aspect ratios
	ratio := float64(width) / float64(height)
	switch {
	case ratio > 1.7 && ratio < 1.8: // ~16:9
		return "16:9"
	case ratio > 1.3 && ratio < 1.4: // ~4:3
		return "4:3"
	case ratio > 0.55 && ratio < 0.57: // ~9:16
		return "9:16"
	case ratio > 0.99 && ratio < 1.01: // ~1:1
		return "1:1"
	default:
		return fmt.Sprintf("%d:%d", w, h)
	}
}
