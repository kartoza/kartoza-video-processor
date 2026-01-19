package youtube

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ThumbnailOptions configures thumbnail extraction
type ThumbnailOptions struct {
	Timestamp time.Duration // Specific timestamp to extract from
	Width     int           // Output width (0 = original)
	Height    int           // Output height (0 = original)
	Quality   int           // JPEG quality 1-100 (default 85)
}

// DefaultThumbnailOptions returns sensible defaults
func DefaultThumbnailOptions() ThumbnailOptions {
	return ThumbnailOptions{
		Timestamp: 60 * time.Second,
		Quality:   85,
	}
}

// GetVideoDuration returns the duration of a video file using ffprobe
func GetVideoDuration(videoPath string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return time.Duration(duration * float64(time.Second)), nil
}

// ExtractThumbnail extracts a single frame from the video at the specified timestamp
func ExtractThumbnail(videoPath string, opts ThumbnailOptions, outputPath string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build ffmpeg command
	args := []string{
		"-y", // Overwrite output
		"-ss", formatDuration(opts.Timestamp), // Seek to timestamp
		"-i", videoPath,
		"-vframes", "1", // Extract single frame
		"-q:v", strconv.Itoa(max(1, min(31, 32-opts.Quality/3))), // Quality (ffmpeg uses 2-31, lower is better)
	}

	// Add scaling if requested
	if opts.Width > 0 || opts.Height > 0 {
		width := opts.Width
		height := opts.Height
		if width == 0 {
			width = -1 // Maintain aspect ratio
		}
		if height == 0 {
			height = -1
		}
		args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", width, height))
	}

	args = append(args, outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("thumbnail was not created")
	}

	return nil
}

// ExtractThumbnailAuto extracts a thumbnail at 60s or last frame if video is shorter
func ExtractThumbnailAuto(videoPath, outputPath string) error {
	duration, err := GetVideoDuration(videoPath)
	if err != nil {
		// If we can't get duration, try at 0s
		duration = 0
	}

	opts := DefaultThumbnailOptions()

	// Use 60s mark, or 75% into video if shorter, or 0 if very short
	if duration < opts.Timestamp {
		if duration > 4*time.Second {
			opts.Timestamp = duration * 3 / 4 // 75% mark
		} else if duration > time.Second {
			opts.Timestamp = duration / 2 // Middle
		} else {
			opts.Timestamp = 0 // Start
		}
	}

	return ExtractThumbnail(videoPath, opts, outputPath)
}

// ExtractThumbnailForYouTube extracts an optimized thumbnail for YouTube
// YouTube recommends 1280x720 (16:9 aspect ratio)
func ExtractThumbnailForYouTube(videoPath, outputPath string) error {
	duration, err := GetVideoDuration(videoPath)
	if err != nil {
		duration = 0
	}

	opts := ThumbnailOptions{
		Timestamp: 60 * time.Second,
		Width:     1280,
		Height:    720,
		Quality:   90,
	}

	// Adjust timestamp for short videos
	if duration < opts.Timestamp {
		if duration > 4*time.Second {
			opts.Timestamp = duration * 3 / 4
		} else if duration > time.Second {
			opts.Timestamp = duration / 2
		} else {
			opts.Timestamp = 0
		}
	}

	return ExtractThumbnail(videoPath, opts, outputPath)
}

// GetThumbnailPath returns the standard thumbnail path for a video
func GetThumbnailPath(videoPath string) string {
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	return filepath.Join(dir, base+"_thumbnail.jpg")
}

// formatDuration formats a duration for ffmpeg (HH:MM:SS.mmm)
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := d.Seconds() - float64(hours*3600) - float64(minutes*60)
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, seconds)
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ExtractMultipleThumbnails extracts thumbnails at multiple timestamps for preview selection
func ExtractMultipleThumbnails(videoPath, outputDir string, count int) ([]string, error) {
	duration, err := GetVideoDuration(videoPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	var paths []string
	interval := duration / time.Duration(count+1)

	for i := 1; i <= count; i++ {
		timestamp := interval * time.Duration(i)
		outputPath := filepath.Join(outputDir, fmt.Sprintf("thumb_%02d.jpg", i))

		opts := ThumbnailOptions{
			Timestamp: timestamp,
			Width:     320, // Small previews
			Height:    180,
			Quality:   75,
		}

		if err := ExtractThumbnail(videoPath, opts, outputPath); err != nil {
			continue // Skip failed extractions
		}

		paths = append(paths, outputPath)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("failed to extract any thumbnails")
	}

	return paths, nil
}
