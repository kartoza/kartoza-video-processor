//go:build !windows

package terminal

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets Unix-specific process attributes
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// stopProcess sends signals to gracefully stop the process on Unix
func stopProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Send SIGTERM to stop process gracefully
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGINT (Ctrl+C)
		cmd.Process.Signal(syscall.SIGINT)
	}

	return nil
}
