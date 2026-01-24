package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/notify"
	"github.com/kartoza/kartoza-video-processor/internal/terminal"
	"github.com/spf13/cobra"
)

var terminalCmd = &cobra.Command{
	Use:   "terminal",
	Short: "Record terminal session using asciinema",
	Long: `Record your terminal session using asciinema and convert it to video.

This command is ideal for:
- Terminal-only environments (no graphical display)
- Recording CLI tutorials and demonstrations
- Creating terminal-based content

The recording captures your terminal session as an asciinema cast file,
then converts it to GIF and MP4 formats when you stop recording.

Press Ctrl+D or type 'exit' to stop the recording.

Example:
  kartoza-video-processor terminal
  kartoza-video-processor terminal --title "My CLI Tutorial"`,
	Run: runTerminalRecording,
}

var (
	terminalTitle       string
	terminalIdleLimit   float64
	terminalFontSize    int
	terminalConvertOnly string
)

func init() {
	rootCmd.AddCommand(terminalCmd)

	terminalCmd.Flags().StringVarP(&terminalTitle, "title", "t", "", "Title for the recording")
	terminalCmd.Flags().Float64Var(&terminalIdleLimit, "idle-limit", 5.0, "Maximum idle time in seconds")
	terminalCmd.Flags().IntVar(&terminalFontSize, "font-size", 16, "Font size for video rendering")
	terminalCmd.Flags().StringVar(&terminalConvertOnly, "convert", "", "Convert existing .cast file to video (skip recording)")
}

func runTerminalRecording(cmd *cobra.Command, args []string) {
	// Check if we're just converting an existing cast file
	if terminalConvertOnly != "" {
		convertCastFile(terminalConvertOnly)
		return
	}

	// Check dependencies
	if !terminal.IsAsciinemaAvailable() {
		fmt.Fprintln(os.Stderr, "Error: asciinema is not installed")
		fmt.Fprintln(os.Stderr, "Install it with: nix-env -iA nixpkgs.asciinema")
		os.Exit(1)
	}

	if !terminal.IsAggAvailable() {
		fmt.Fprintln(os.Stderr, "Warning: agg is not installed - video conversion will be limited")
		fmt.Fprintln(os.Stderr, "Install it with: nix-env -iA nixpkgs.agg")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load config: %v\n", err)
		cfg = &config.Config{}
	}

	// Create output directory
	baseDir := cfg.OutputDir
	if baseDir == "" {
		baseDir = config.GetDefaultVideosDir()
	}

	timestamp := time.Now().Format("20060102-150405")
	folderName := fmt.Sprintf("terminal-%s", timestamp)
	if terminalTitle != "" {
		// Sanitize title for folder name
		safeTitle := sanitizeFolderName(terminalTitle)
		folderName = fmt.Sprintf("terminal-%s-%s", timestamp, safeTitle)
	}
	outputDir := filepath.Join(baseDir, folderName)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create output directory: %v\n", err)
		os.Exit(1)
	}

	// Create recording info
	metadata := models.RecordingMetadata{
		Title:       terminalTitle,
		Description: "Terminal recording",
		FolderName:  folderName,
	}
	if metadata.Title == "" {
		metadata.Title = "Terminal Recording"
	}

	recordingInfo := models.NewRecordingInfo(metadata, "", "")
	recordingInfo.Files.FolderPath = outputDir
	recordingInfo.Settings.ScreenEnabled = false
	recordingInfo.Settings.AudioEnabled = false
	recordingInfo.Settings.WebcamEnabled = false
	recordingInfo.SetStatus(models.StatusRecording)

	if err := recordingInfo.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not save recording info: %v\n", err)
	}

	// Create recorder
	recorder := terminal.New()

	// Set up signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nStopping recording...")
		recorder.Stop()
	}()

	// Print instructions
	fmt.Println("╭─────────────────────────────────────────────────────────╮")
	fmt.Println("│           Terminal Recording Started                     │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Printf("│  Output: %-47s │\n", outputDir)
	fmt.Println("│                                                          │")
	fmt.Println("│  Press Ctrl+D or type 'exit' to stop recording          │")
	fmt.Println("╰─────────────────────────────────────────────────────────╯")
	fmt.Println()

	// Start recording
	opts := terminal.RecorderOptions{
		OutputDir:   outputDir,
		Title:       terminalTitle,
		IdleTimeMax: terminalIdleLimit,
	}

	if err := recorder.Start(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to start recording: %v\n", err)
		os.Exit(1)
	}

	// Wait for asciinema to finish (it runs interactively)
	// The process will exit when user types 'exit' or presses Ctrl+D

	fmt.Println()
	fmt.Println("Recording stopped.")

	// Get the cast file path
	castFile := recorder.GetCastFile()
	if castFile == "" {
		castFile = filepath.Join(outputDir, "terminal.cast")
	}

	// Check if cast file exists
	if _, err := os.Stat(castFile); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: No recording file created")
		os.Exit(1)
	}

	// Update recording info
	recordingInfo.Files.VideoFile = "terminal.cast"
	recordingInfo.SetStatus(models.StatusProcessing)
	recordingInfo.Save()

	// Convert to video
	fmt.Println("Converting to video...")
	convertCastFileInDir(castFile, outputDir, cfg)

	// Update final status
	recordingInfo.SetStatus(models.StatusCompleted)
	recordingInfo.Save()

	notify.Info("Terminal Recording Complete", fmt.Sprintf("Saved to %s", outputDir))
}

func convertCastFile(castFile string) {
	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	outputDir := filepath.Dir(castFile)
	convertCastFileInDir(castFile, outputDir, cfg)
}

func convertCastFileInDir(castFile, outputDir string, cfg *config.Config) {
	// Get terminal settings
	termSettings := cfg.TerminalRecording
	if termSettings.FontSize == 0 {
		termSettings = config.DefaultTerminalRecordingSettings()
	}

	gifOpts := &terminal.GifOptions{
		FontSize:      termSettings.FontSize,
		FontFamily:    termSettings.FontFamily,
		Theme:         termSettings.Theme,
		FPSCap:        termSettings.FPSCap,
		IdleTimeLimit: termSettings.IdleTimeLimit,
	}

	// Override with command line options if provided
	if terminalFontSize > 0 {
		gifOpts.FontSize = terminalFontSize
	}

	gifFile := filepath.Join(outputDir, "terminal.gif")
	mp4File := filepath.Join(outputDir, "terminal.mp4")

	// Convert to GIF
	fmt.Print("  Creating GIF... ")
	if err := terminal.ConvertToGif(castFile, gifFile, gifOpts); err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		fmt.Println("done")
	}

	// Convert to MP4
	fmt.Print("  Creating MP4... ")
	if err := terminal.ConvertGifToMp4(gifFile, mp4File); err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		fmt.Println("done")
	}

	fmt.Println()
	fmt.Println("Output files:")
	fmt.Printf("  Cast: %s\n", castFile)
	if _, err := os.Stat(gifFile); err == nil {
		fmt.Printf("  GIF:  %s\n", gifFile)
	}
	if _, err := os.Stat(mp4File); err == nil {
		fmt.Printf("  MP4:  %s\n", mp4File)
	}
}

func sanitizeFolderName(name string) string {
	// Replace problematic characters
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		} else if r == ' ' {
			result += "-"
		}
	}
	// Limit length
	if len(result) > 30 {
		result = result[:30]
	}
	return result
}
