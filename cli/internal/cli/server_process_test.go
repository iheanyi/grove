package cli

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestSignalServerPIDStopsEntireProcessGroup(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "trap 'exit 0' TERM; sleep 30 & wait")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start process group: %v", err)
	}

	pid := cmd.Process.Pid
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	t.Cleanup(func() {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		select {
		case <-waitDone:
		case <-time.After(time.Second):
		}
	})

	if err := signalServerPID(pid, syscall.SIGTERM); err != nil {
		t.Fatalf("signal process group: %v", err)
	}

	if !waitForServerPIDExit(pid, 2*time.Second) {
		t.Fatalf("process group %d still running after SIGTERM", pid)
	}
}

func TestTerminateServerPIDReturnsErrorWhenForceKillFails(t *testing.T) {
	var signals []syscall.Signal
	signal := func(_ int, sig syscall.Signal) error {
		signals = append(signals, sig)
		if sig == syscall.SIGKILL {
			return syscall.EPERM
		}
		return nil
	}
	wait := func(_ int, _ time.Duration) bool {
		return false
	}

	err := terminateServerPID(42, time.Second, signal, wait)
	if !errors.Is(err, syscall.EPERM) {
		t.Fatalf("terminateServerPID error = %v, want EPERM", err)
	}
	if len(signals) != 2 || signals[0] != syscall.SIGTERM || signals[1] != syscall.SIGKILL {
		t.Fatalf("signals = %v, want [SIGTERM SIGKILL]", signals)
	}
}
