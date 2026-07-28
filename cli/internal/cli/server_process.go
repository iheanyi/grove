package cli

import (
	"os"
	"syscall"
	"time"
)

func signalServerPID(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return syscall.ESRCH
	}

	// Daemonized servers are launched in a process group. Signal the whole
	// group so the server and every process in its logging pipeline exit.
	if err := syscall.Kill(-pid, sig); err == nil {
		return nil
	} else if err != syscall.ESRCH {
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(sig)
}

func isServerPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	if err := syscall.Kill(-pid, 0); err == nil {
		return true
	} else if err != syscall.ESRCH {
		return true
	}

	return isProcessRunning(pid)
}

func waitForServerPIDExit(pid int, timeout time.Duration) bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		if !isServerPIDRunning(pid) {
			return true
		}

		select {
		case <-ticker.C:
		case <-timer.C:
			return !isServerPIDRunning(pid)
		}
	}
}
