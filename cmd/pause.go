package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-screencaster/internal/recorder"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause the current recording",
	Long: `Pause the current recording session.

The recording can be resumed later with 'kartoza-screencaster resume'.
Each pause/resume cycle creates a new part file (e.g., screen_part001.mp4).
All parts will be concatenated when the recording is stopped.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()

		if rec.IsPaused() {
			return fmt.Errorf("recording is already paused")
		}

		if !rec.IsRecording() {
			return fmt.Errorf("no recording in progress")
		}

		fmt.Println("Pausing recording...")
		if err := rec.Pause(); err != nil {
			return err
		}

		status := rec.GetStatus()
		fmt.Printf("Recording paused at part %d.\n", status.CurrentPart-1)
		fmt.Println("Use 'kartoza-screencaster resume' to continue recording.")

		return nil
	},
}
