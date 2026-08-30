package cli

import (
	"testing"
	"time"

	"github.com/iheanyi/grove/internal/registry"
)

func TestPrepareServerForRestartClearsRunningState(t *testing.T) {
	stoppedAt := time.Date(2026, time.July, 28, 18, 0, 0, 0, time.UTC)
	server := &registry.Server{
		Name:   "feature-ui",
		PID:    4242,
		Status: registry.StatusRunning,
	}

	prepareServerForRestart(server, stoppedAt)

	if server.IsRunning() {
		t.Fatal("server remains running after confirmed termination")
	}
	if server.PID != 0 {
		t.Fatalf("server PID = %d, want 0", server.PID)
	}
	if !server.StoppedAt.Equal(stoppedAt) {
		t.Fatalf("server stopped_at = %v, want %v", server.StoppedAt, stoppedAt)
	}
}
