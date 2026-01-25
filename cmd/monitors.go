package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/kartoza/kartoza-screencaster/internal/monitor"
	"github.com/spf13/cobra"
)

var monitorsJsonOutput bool

var monitorsCmd = &cobra.Command{
	Use:   "monitors",
	Short: "List available monitors",
	Long:  `List all available monitors with their resolution and position information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		monitors, err := monitor.ListMonitors()
		if err != nil {
			return fmt.Errorf("failed to list monitors: %w", err)
		}

		if monitorsJsonOutput {
			data, err := json.MarshalIndent(monitors, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		cursor, err := monitor.GetMouseMonitor()
		if err != nil {
			cursor = ""
		}

		for _, m := range monitors {
			cursorMark := ""
			if m.Name == cursor {
				cursorMark = " (cursor)"
			}
			fmt.Printf("%s: %dx%d at (%d,%d)%s\n",
				m.Name, m.Width, m.Height, m.X, m.Y, cursorMark)
		}

		return nil
	},
}

func init() {
	monitorsCmd.Flags().BoolVar(&monitorsJsonOutput, "json", false, "Output monitors as JSON")
}
