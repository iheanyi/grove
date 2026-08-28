package cli

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

type signalServerPIDFunc func(int, syscall.Signal) error
type waitForServerPIDExitFunc func(int, time.Duration) bool

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

func terminateServerPID(
	pid int,
	gracefulTimeout time.Duration,
	signal signalServerPIDFunc,
	wait waitForServerPIDExitFunc,
) error {
	if err := signal(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return fmt.Errorf("send SIGTERM to server process group: %w", err)
	}
	if wait(pid, gracefulTimeout) {
		return nil
	}

	if err := signal(pid, syscall.SIGKILL); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return fmt.Errorf("send SIGKILL to server process group: %w", err)
	}
	if !wait(pid, 5*time.Second) {
		return fmt.Errorf("server process group %d is still running after SIGKILL", pid)
	}

	return nil
}
