package cmd

import (
	"github.com/kartoza/kartoza-video-processor/internal/tui"
)

func runTUIApp(noSplash bool) error {
	return tui.Run(noSplash)
}
