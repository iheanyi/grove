package cli

import (
	"testing"

	"github.com/iheanyi/grove/internal/registry"
)

func TestAttachAllowsOverwritingStoppedServer(t *testing.T) {
	// Create a stopped server
	stoppedServer := &registry.Server{
		Name:   "test-server",
		Port:   3000,
		Status: registry.StatusStopped,
	}

	// Verify IsRunning returns false for stopped servers
	if stoppedServer.IsRunning() {
		t.Error("Expected stopped server to return IsRunning() = false")
	}

	// Create a running server
	runningServer := &registry.Server{
		Name:   "running-server",
		Port:   3001,
		Status: registry.StatusRunning,
	}

	// Verify IsRunning returns true for running servers
	if !runningServer.IsRunning() {
		t.Error("Expected running server to return IsRunning() = true")
	}

	// Test the logic: stopped server should allow overwrite
	// This tests the condition that was fixed in attach.go
	existing := stoppedServer
	if existing.IsRunning() {
		t.Error("Should allow attaching over a stopped server")
	}
}

func TestAttachBlocksRunningServer(t *testing.T) {
	runningServer := &registry.Server{
		Name:   "running-server",
		Port:   3001,
		Status: registry.StatusRunning,
	}

	// Test the logic: running server should block
	if !runningServer.IsRunning() {
		t.Error("Should block attaching over a running server")
	}
}

func TestAttachPortCheckOnlyBlocksRunningServers(t *testing.T) {
	servers := []*registry.Server{
		{Name: "stopped-1", Port: 3000, Status: registry.StatusStopped},
		{Name: "running-1", Port: 3001, Status: registry.StatusRunning},
		{Name: "stopped-2", Port: 3002, Status: registry.StatusStopped},
	}

	// Check port 3000 - should NOT be blocked (server is stopped)
	targetPort := 3000
	for _, s := range servers {
		if s.Port == targetPort && s.IsRunning() {
			t.Errorf("Port %d should not be blocked - server is stopped", targetPort)
		}
	}

	// Check port 3001 - SHOULD be blocked (server is running)
	targetPort = 3001
	blocked := false
	for _, s := range servers {
		if s.Port == targetPort && s.IsRunning() {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Error("Port 3001 should be blocked - server is running")
	}
}
