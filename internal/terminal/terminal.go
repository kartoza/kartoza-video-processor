package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/kartoza/kartoza-screencaster/internal/config"
)

const (
	// AsciinemaFPSFile stores the FPS for rendering
	AsciinemaFPSFile = "/tmp/kartoza-asciinema-fps"
	// AsciinemaColsFile stores terminal columns
	AsciinemaColsFile = "/tmp/kartoza-asciinema-cols"
	// AsciinemaRowsFile stores terminal rows
	AsciinemaRowsFile = "/tmp/kartoza-asciinema-rows"
)

// IsTerminalOnly checks if we're in a terminal-only environment (no graphical display)
func IsTerminalOnly() bool {
	display := os.Getenv("DISPLAY")
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	sessionType := os.Getenv("XDG_SESSION_TYPE")

	// If no display variables are set, we're in a terminal-only environment
	if display == "" && waylandDisplay == "" {
		return true
	}

	// If session type is explicitly "tty", we're in a terminal
	if sessionType == "tty" {
		return true
	}

	return false
}

// IsAsciinemaAvailable checks if asciinema is installed
func IsAsciinemaAvailable() bool {
	_, err := exec.LookPath("asciinema")
	return err == nil
}

// IsAggAvailable checks if agg (asciinema gif generator) is installed
func IsAggAvailable() bool {
	_, err := exec.LookPath("agg")
	return err == nil
}

// RecorderOptions contains options for terminal recording
type RecorderOptions struct {
	OutputDir   string
	Title       string
	IdleTimeMax float64 // Max idle time in seconds (0 = no limit)
	Cols        int     // Terminal columns (0 = auto)
	Rows        int     // Terminal rows (0 = auto)
}

// Recorder handles asciinema-based terminal recording
type Recorder struct {
	cmd        *exec.Cmd
	castFile   string
	outputDir  string
	pid        int
	isRecording bool
}

// New creates a new terminal recorder
func New() *Recorder {
	return &Recorder{}
}

// Start begins recording the terminal session with asciinema
func (r *Recorder) Start(opts RecorderOptions) error {
	if !IsAsciinemaAvailable() {
		return fmt.Errorf("asciinema is not installed")
	}

	r.outputDir = opts.OutputDir
	r.castFile = filepath.Join(opts.OutputDir, "terminal.cast")

	// Build asciinema command
	args := []string{"rec", "--overwrite"}

	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	if opts.IdleTimeMax > 0 {
		args = append(args, "--idle-time-limit", fmt.Sprintf("%.1f", opts.IdleTimeMax))
	}

	if opts.Cols > 0 {
		args = append(args, "--cols", strconv.Itoa(opts.Cols))
		os.WriteFile(AsciinemaColsFile, []byte(strconv.Itoa(opts.Cols)), 0644)
	}

	if opts.Rows > 0 {
		args = append(args, "--rows", strconv.Itoa(opts.Rows))
		os.WriteFile(AsciinemaRowsFile, []byte(strconv.Itoa(opts.Rows)), 0644)
	}

	args = append(args, r.castFile)

	r.cmd = exec.Command("asciinema", args...)
	r.cmd.Stdin = os.Stdin
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	// Set platform-specific process attributes
	setSysProcAttr(r.cmd)

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start asciinema: %w", err)
	}

	r.pid = r.cmd.Process.Pid
	r.isRecording = true

	// Save PID
	os.WriteFile(config.VideoPIDFile, []byte(strconv.Itoa(r.pid)), 0644)

	return nil
}

// Stop stops the terminal recording
func (r *Recorder) Stop() error {
	if !r.isRecording || r.cmd == nil || r.cmd.Process == nil {
		return nil
	}

	// Stop process using platform-specific method
	stopProcess(r.cmd)

	// Wait for process to finish
	r.cmd.Wait()
	r.isRecording = false

	// Clean up PID file
	os.Remove(config.VideoPIDFile)

	return nil
}

// GetCastFile returns the path to the recorded .cast file
func (r *Recorder) GetCastFile() string {
	return r.castFile
}

// ConvertToGif converts an asciinema cast file to GIF using agg
func ConvertToGif(castFile, gifFile string, opts *GifOptions) error {
	if !IsAggAvailable() {
		return fmt.Errorf("agg is not installed")
	}

	args := []string{}

	if opts != nil {
		if opts.FontSize > 0 {
			args = append(args, "--font-size", strconv.Itoa(opts.FontSize))
		}
		if opts.FontFamily != "" {
			args = append(args, "--font-family", opts.FontFamily)
		}
		if opts.Theme != "" {
			args = append(args, "--theme", opts.Theme)
		}
		if opts.FPSCap > 0 {
			args = append(args, "--fps-cap", strconv.Itoa(opts.FPSCap))
		}
		if opts.IdleTimeLimit > 0 {
			args = append(args, "--idle-time-limit", fmt.Sprintf("%.1f", opts.IdleTimeLimit))
		}
		if opts.Cols > 0 {
			args = append(args, "--cols", strconv.Itoa(opts.Cols))
		}
		if opts.Rows > 0 {
			args = append(args, "--rows", strconv.Itoa(opts.Rows))
		}
	}

	args = append(args, castFile, gifFile)

	cmd := exec.Command("agg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("agg failed: %w: %s", err, string(output))
	}

	return nil
}

// ConvertGifToMp4 converts a GIF to MP4 using ffmpeg
func ConvertGifToMp4(gifFile, mp4File string) error {
	// Use ffmpeg to convert GIF to MP4
	// -movflags +faststart makes it web-friendly
	// -pix_fmt yuv420p ensures compatibility
	// -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" ensures even dimensions
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", gifFile,
		"-movflags", "+faststart",
		"-pix_fmt", "yuv420p",
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		mp4File,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", err, string(output))
	}

	return nil
}

// ConvertCastToMp4 converts a cast file directly to MP4
func ConvertCastToMp4(castFile, mp4File string, opts *GifOptions) error {
	// First convert to GIF
	gifFile := mp4File[:len(mp4File)-4] + ".gif"

	if err := ConvertToGif(castFile, gifFile, opts); err != nil {
		return err
	}

	// Then convert GIF to MP4
	if err := ConvertGifToMp4(gifFile, mp4File); err != nil {
		return err
	}

	// Optionally remove the intermediate GIF
	// os.Remove(gifFile)

	return nil
}

// GifOptions contains options for GIF generation
type GifOptions struct {
	FontSize      int
	FontFamily    string
	Theme         string
	FPSCap        int
	IdleTimeLimit float64
	Cols          int
	Rows          int
}

// DefaultGifOptions returns sensible defaults for GIF generation
func DefaultGifOptions() *GifOptions {
	return &GifOptions{
		FontSize:      16,
		FontFamily:    "JetBrains Mono,Fira Code,monospace",
		FPSCap:        30,
		IdleTimeLimit: 2.0,
	}
}
