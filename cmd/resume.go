package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-video-processor/internal/recorder"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a paused recording",
	Long: `Resume a previously paused recording session.

This creates a new part file (e.g., screen_part001.mp4) and continues
recording. All parts will be concatenated when the recording is stopped.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()

		if !rec.IsPaused() {
			return fmt.Errorf("no paused recording to resume")
		}

		fmt.Println("Resuming recording...")
		if err := rec.Resume(); err != nil {
			return err
		}

		status := rec.GetStatus()
		fmt.Printf("Recording resumed at part %d.\n", status.CurrentPart)
		fmt.Println("Use 'kartoza-video-processor pause' to pause or 'stop' to finish.")

		return nil
	},
}
