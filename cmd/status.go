package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kartoza/kartoza-video-processor/internal/recorder"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show recording status",
	Long:  `Display the current recording status including duration, file paths, and active monitors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()
		status := rec.GetStatus()

		if jsonOutput {
			data, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		if status.IsRecording {
			duration := time.Since(status.StartTime).Round(time.Second)
			fmt.Printf("Recording: ACTIVE\n")
			fmt.Printf("Duration:  %s\n", duration)
			fmt.Printf("Part:      %d\n", status.CurrentPart)
			fmt.Printf("Monitor:   %s\n", status.Monitor)
			fmt.Printf("Video:     %s\n", status.VideoFile)
			fmt.Printf("Audio:     %s\n", status.AudioFile)
			if status.WebcamFile != "" {
				fmt.Printf("Webcam:    %s\n", status.WebcamFile)
			}
		} else if status.IsPaused {
			duration := time.Since(status.StartTime).Round(time.Second)
			fmt.Printf("Recording: PAUSED\n")
			fmt.Printf("Duration:  %s (before pause)\n", duration)
			fmt.Printf("Parts:     %d recorded\n", status.CurrentPart)
			fmt.Println("\nUse 'kartoza-video-processor resume' to continue recording.")
			fmt.Println("Use 'kartoza-video-processor stop' to finish and process.")
		} else {
			fmt.Println("Recording: INACTIVE")
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output status as JSON")
}
