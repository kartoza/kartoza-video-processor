//go:build windows

package terminal

import (
	"os"
	"os/exec"
)

// setSysProcAttr sets Windows-specific process attributes (no-op on Windows)
func setSysProcAttr(cmd *exec.Cmd) {
	// No equivalent on Windows - process groups work differently
}

// stopProcess kills the process on Windows
func stopProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// On Windows, just kill the process directly
	return cmd.Process.Signal(os.Kill)
}
