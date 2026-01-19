package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-video-processor/internal/recorder"
	"github.com/spf13/cobra"
)

var noProcess bool

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop screen recording",
	Long: `Stop the current screen recording session and process the captured files.

By default, this command waits for post-processing to complete (merging audio,
creating vertical video, etc.). Use --no-process to skip post-processing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()

		if !rec.IsRecording() {
			return fmt.Errorf("no recording in progress")
		}

		fmt.Println("Stopping recording...")
		if err := rec.StopAndProcess(!noProcess); err != nil {
			return err
		}

		if noProcess {
			fmt.Println("Recording stopped. Post-processing skipped.")
		} else {
			fmt.Println("Recording stopped and processed.")
		}

		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVar(&noProcess, "no-process", false, "Skip post-processing (merging, vertical video, etc.)")
}
