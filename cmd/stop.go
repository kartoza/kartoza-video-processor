package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-video-processor/internal/recorder"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop screen recording",
	Long:  `Stop the current screen recording session and process the captured files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()

		if !rec.IsRecording() {
			return fmt.Errorf("no recording in progress")
		}

		fmt.Println("Stopping recording...")
		return rec.Stop()
	},
}
