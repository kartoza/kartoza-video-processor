package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-video-processor/internal/recorder"
	"github.com/spf13/cobra"
)

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle screen recording on/off",
	Long:  `Toggle screen recording. If recording is active, stop it. If not recording, start a new recording.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()

		if rec.IsRecording() {
			fmt.Println("Stopping recording...")
			return rec.Stop()
		}

		fmt.Println("Starting recording...")
		return rec.Start()
	},
}
