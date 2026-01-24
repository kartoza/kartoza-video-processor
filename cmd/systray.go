package cmd

import (
	"github.com/kartoza/kartoza-video-processor/internal/systray"
	"github.com/spf13/cobra"
)

var systrayCmd = &cobra.Command{
	Use:   "systray",
	Short: "Run as a system tray application",
	Long: `Run the Kartoza Video Processor as a system tray application.

The system tray applet provides quick access to recording controls:
  - Left-click: Toggle recording (start if idle, stop if recording)
  - Right-click: Show menu with pause/resume, open TUI, quit options

When you stop a recording via the systray, the TUI will automatically
open so you can provide a title and description for your recording.

This mode is ideal for quick recordings where you want to start recording
immediately without filling in metadata first.`,
	Run: func(cmd *cobra.Command, args []string) {
		systray.RunWithHandler()
	},
}

func init() {
	rootCmd.AddCommand(systrayCmd)
}
