package cmd

import (
	"github.com/kartoza/kartoza-screencaster/internal/tui"
)

func runTUIApp(noSplash bool, presetsMode bool) error {
	// Set the version in the global app state for header display
	tui.GlobalAppState.Version = version
	return tui.Run(noSplash, presetsMode, editRecordingMode)
}
