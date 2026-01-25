package cmd

import (
	"github.com/kartoza/kartoza-screencaster/internal/tui"
)

func runTUIApp(noSplash bool) error {
	return tui.Run(noSplash)
}
