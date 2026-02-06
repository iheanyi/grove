package tui

import (
	"os"
	"syscall"

	"github.com/iheanyi/grove/internal/config"
)

// Run starts the TUI
func Run(cfg *config.Config) error {
	// Use enhanced version by default
	return RunEnhanced(cfg)
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
