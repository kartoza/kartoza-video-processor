package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	debugMode bool
	dataDir   string
	noSplash  bool
)

// SetVersion sets the application version (called from main)
func SetVersion(v string) {
	version = v
}

var rootCmd = &cobra.Command{
	Use:   "kartoza-video-processor",
	Short: "Screen recording and video processing tool for Wayland",
	Long: `Kartoza Video Processor is a screen recording tool for Wayland compositors.

It supports:
  - Multi-monitor screen recording with automatic cursor detection
  - Separate audio recording with noise reduction
  - Webcam recording and vertical video creation
  - Audio normalization using EBU R128 loudness standards
  - Hardware and software video encoding

The tool integrates with Hyprland and other wlroots-based compositors.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action: start TUI or toggle recording
		if err := runTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "Data directory (default: ~/.config/kartoza-video-processor)")
	rootCmd.PersistentFlags().BoolVar(&noSplash, "nosplash", false, "Skip splash screens on startup and exit")

	// Add subcommands
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(monitorsCmd)
}

func runTUI() error {
	// Import and run the TUI application
	return runTUIApp(noSplash)
}
