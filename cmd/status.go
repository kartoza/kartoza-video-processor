package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kartoza/kartoza-screencaster/internal/models"
	"github.com/kartoza/kartoza-screencaster/internal/recorder"
	"github.com/spf13/cobra"
)

var jsonOutput bool
var waybarOutput bool

// WaybarStatus represents the status information for waybar
type WaybarStatus struct {
	Text    string `json:"text"`
	Alt     string `json:"alt"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

// Nerd Font icons for waybar display
const (
	nfVideocam = "󰕧" // nf-md-videocam - for recording
	nfPause    = "󰏤" // nf-md-pause - for paused state
	nfStop     = "󰓛" // nf-md-stop - for stopped/idle state
	nfMonitor  = "󰍹" // nf-md-monitor - for monitor
	nfTimer    = "󰔛" // nf-md-timer - for duration
	nfFolder   = "󰉋" // nf-md-folder - for files
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show recording status",
	Long: `Display the current recording status including duration, file paths, and active monitors.

For waybar integration, use the --waybar flag to get JSON output suitable for waybar custom modules.

Example waybar configuration:
{
    "custom/recorder": {
        "exec": "kartoza-screencaster status --waybar",
        "return-type": "json",
        "interval": 2,
        "on-click": "kartoza-screencaster toggle"
    }
}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()
		status := rec.GetStatus()

		if waybarOutput {
			waybar := createWaybarStatus(status)
			data, err := json.Marshal(waybar)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

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
			fmt.Println("\nUse 'kartoza-screencaster resume' to continue recording.")
			fmt.Println("Use 'kartoza-screencaster stop' to finish and process.")
		} else {
			fmt.Println("Recording: INACTIVE")
		}

		return nil
	},
}

func createWaybarStatus(status models.RecordingStatus) WaybarStatus {
	if status.IsRecording {
		duration := time.Since(status.StartTime)
		elapsed := formatDuration(duration)

		// Build tooltip
		var tooltipLines []string
		tooltipLines = append(tooltipLines, nfVideocam+" Recording in progress")
		tooltipLines = append(tooltipLines, "")

		if status.Monitor != "" {
			tooltipLines = append(tooltipLines, fmt.Sprintf("%s Monitor: %s", nfMonitor, status.Monitor))
		}

		tooltipLines = append(tooltipLines, fmt.Sprintf("%s Duration: %s", nfTimer, elapsed))

		if status.CurrentPart > 1 {
			tooltipLines = append(tooltipLines, fmt.Sprintf("Part: %d", status.CurrentPart))
		}

		tooltipLines = append(tooltipLines, "")
		tooltipLines = append(tooltipLines, "Click to stop recording")

		return WaybarStatus{
			Text:    fmt.Sprintf("%s %s", nfVideocam, elapsed),
			Alt:     "recording",
			Tooltip: strings.Join(tooltipLines, "\n"),
			Class:   "recording",
		}
	}

	if status.IsPaused {
		// Build tooltip for paused state
		var tooltipLines []string
		tooltipLines = append(tooltipLines, nfPause+" Recording paused")
		tooltipLines = append(tooltipLines, "")

		if status.CurrentPart > 0 {
			tooltipLines = append(tooltipLines, fmt.Sprintf("Parts recorded: %d", status.CurrentPart))
		}

		tooltipLines = append(tooltipLines, "")
		tooltipLines = append(tooltipLines, "Click to resume or stop")

		return WaybarStatus{
			Text:    fmt.Sprintf("%s paused", nfPause),
			Alt:     "paused",
			Tooltip: strings.Join(tooltipLines, "\n"),
			Class:   "paused",
		}
	}

	// Idle state
	return WaybarStatus{
		Text:    nfVideocam,
		Alt:     "idle",
		Tooltip: "Click to start recording (Ctrl+6)",
		Class:   "idle",
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func init() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output status as JSON")
	statusCmd.Flags().BoolVar(&waybarOutput, "waybar", false, "Output status in waybar JSON format")
}
