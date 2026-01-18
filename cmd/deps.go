package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/deps"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Check for required dependencies",
	Long:  `Check if all required external programs are installed and available.`,
	Run: func(cmd *cobra.Command, args []string) {
		required, optional := deps.CheckAll()

		// Colors
		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#E95420"))
		gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#9A9EA0"))
		cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BCD4"))
		bold := lipgloss.NewStyle().Bold(true)

		fmt.Println()

		// Show detected display server
		displayServer := deps.DetectDisplayServer()
		fmt.Printf("%s %s\n\n", bold.Render("Display Server:"), cyan.Render(deps.GetDisplayServerName()))

		// Show which screen recording method will be used
		switch displayServer {
		case deps.DisplayServerWayland:
			fmt.Printf("%s wl-screenrec (Wayland native)\n\n", gray.Render("Screen recording:"))
		case deps.DisplayServerX11:
			fmt.Printf("%s ffmpeg x11grab (X11)\n\n", gray.Render("Screen recording:"))
		default:
			fmt.Printf("%s Unknown display server\n\n", gray.Render("Screen recording:"))
		}

		fmt.Println(bold.Render("Required Dependencies:"))
		fmt.Println()

		allRequiredOk := true
		for _, r := range required {
			var status string
			if r.Available {
				status = green.Render("✓")
			} else {
				status = red.Render("✗")
				allRequiredOk = false
			}
			fmt.Printf("  %s %s\n", status, bold.Render(r.Dependency.Name))
			fmt.Printf("    %s\n", gray.Render(r.Dependency.Description))
			if r.Available {
				fmt.Printf("    Path: %s\n", r.Path)
			}
			fmt.Println()
		}

		fmt.Println(bold.Render("Optional Dependencies:"))
		fmt.Println()

		for _, r := range optional {
			var status string
			if r.Available {
				status = green.Render("✓")
			} else {
				status = gray.Render("○")
			}
			fmt.Printf("  %s %s\n", status, bold.Render(r.Dependency.Name))
			fmt.Printf("    %s\n", gray.Render(r.Dependency.Description))
			if r.Available {
				fmt.Printf("    Path: %s\n", r.Path)
			}
			fmt.Println()
		}

		if allRequiredOk {
			fmt.Println(green.Render("All required dependencies are installed!"))
		} else {
			fmt.Println(red.Render("Some required dependencies are missing."))
			fmt.Println("Please install them before using the application.")
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(depsCmd)
}
