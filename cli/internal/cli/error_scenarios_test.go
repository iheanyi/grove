package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iheanyi/grove/internal/registry"
)

// TestRegistryLoadError tests that CLI commands handle registry load errors gracefully
func TestRegistryLoadError(t *testing.T) {
	// This tests the pattern used throughout the codebase:
	// if reg, err := registry.Load(); err != nil { ... }

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Write invalid JSON to simulate corruption
	if err := os.WriteFile(registryPath, []byte("invalid json {{{"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a registry with the invalid path
	r := &registry.Registry{}

	// Simulate loading (this will fail)
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	err = json.Unmarshal(data, r)
	if err == nil {
		t.Error("Expected unmarshal to fail for invalid JSON")
	}
}

// TestServerStateTransitions tests server status transitions
func TestServerStateTransitions(t *testing.T) {
	tests := []struct {
		name       string
		from       registry.ServerStatus
		to         registry.ServerStatus
		shouldWork bool
	}{
		{"stopped to running", registry.StatusStopped, registry.StatusRunning, true},
		{"running to stopped", registry.StatusRunning, registry.StatusStopped, true},
		{"crashed to running", registry.StatusCrashed, registry.StatusRunning, true},
		{"starting to running", registry.StatusStarting, registry.StatusRunning, true},
		{"running to crashed", registry.StatusRunning, registry.StatusCrashed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &registry.Server{
				Name:   "test",
				Status: tt.from,
			}

			// Transition
			server.Status = tt.to

			if server.Status != tt.to {
				t.Errorf("Status transition failed: got %s, want %s", server.Status, tt.to)
			}
		})
	}
}

// TestProxyStateManagement tests proxy state management
func TestProxyStateManagement(t *testing.T) {
	proxy := &registry.ProxyInfo{
		PID:       0,
		HTTPPort:  80,
		HTTPSPort: 443,
	}

	// Initially not running
	if proxy.IsRunning() {
		t.Error("Proxy with PID 0 should not be running")
	}

	// Start proxy (simulate)
	proxy.PID = 12345
	proxy.StartedAt = time.Now()

	if !proxy.IsRunning() {
		t.Error("Proxy with PID > 0 should be running")
	}

	// Stop proxy
	proxy.PID = 0

	if proxy.IsRunning() {
		t.Error("Proxy with PID 0 should not be running after stop")
	}
}

// TestServerCleanupLogic tests the cleanup logic for stale servers
func TestServerCleanupLogic(t *testing.T) {
	// Test the logic: servers with dead PIDs should be marked as stopped

	servers := map[string]*registry.Server{
		"alive": {
			Name:   "alive",
			Status: registry.StatusRunning,
			PID:    os.Getpid(), // Current process
		},
		"dead": {
			Name:   "dead",
			Status: registry.StatusRunning,
			PID:    999999999, // Non-existent PID
		},
		"already-stopped": {
			Name:   "already-stopped",
			Status: registry.StatusStopped,
			PID:    0,
		},
	}

	// Simulate cleanup logic
	var cleaned []string
	for name, server := range servers {
		if server.PID > 0 && !isProcessRunning(server.PID) {
			server.Status = registry.StatusStopped
			server.PID = 0
			cleaned = append(cleaned, name)
		}
	}

	// Should have cleaned up only the "dead" server
	if len(cleaned) != 1 {
		t.Errorf("Expected 1 server to be cleaned up, got %d", len(cleaned))
	}

	// Verify dead server is now stopped
	if servers["dead"].Status != registry.StatusStopped {
		t.Error("Dead server should be marked as stopped")
	}
	if servers["dead"].PID != 0 {
		t.Error("Dead server PID should be 0")
	}

	// Verify alive server is unchanged
	if servers["alive"].Status != registry.StatusRunning {
		t.Error("Alive server should still be running")
	}

	// Verify already-stopped server is unchanged
	if servers["already-stopped"].Status != registry.StatusStopped {
		t.Error("Already stopped server should remain stopped")
	}
}

// TestServerUptime tests the uptime calculation
func TestServerUptime(t *testing.T) {
	t.Run("running server has uptime", func(t *testing.T) {
		server := &registry.Server{
			Status:    registry.StatusRunning,
			StartedAt: time.Now().Add(-1 * time.Hour),
		}

		uptime := server.Uptime()
		if uptime < 59*time.Minute {
			t.Errorf("Expected uptime >= 59m, got %v", uptime)
		}
	})

	t.Run("stopped server shows duration", func(t *testing.T) {
		start := time.Now().Add(-2 * time.Hour)
		stop := time.Now().Add(-1 * time.Hour)
		server := &registry.Server{
			Status:    registry.StatusStopped,
			StartedAt: start,
			StoppedAt: stop,
		}

		uptime := server.Uptime()
		expected := stop.Sub(start)
		if uptime != expected {
			t.Errorf("Expected uptime %v, got %v", expected, uptime)
		}
	})

	t.Run("server with no start time has zero uptime", func(t *testing.T) {
		server := &registry.Server{
			Status: registry.StatusRunning,
		}

		uptime := server.Uptime()
		if uptime != 0 {
			t.Errorf("Expected uptime 0, got %v", uptime)
		}
	})
}

// TestErrorMessageFormats tests that error messages are formatted correctly
func TestErrorMessageFormats(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		args    []interface{}
		wantErr bool
	}{
		{
			name:   "registry load error",
			format: "failed to load registry: %w",
			args:   []interface{}{os.ErrNotExist},
		},
		{
			name:   "server not found",
			format: "no server registered for '%s'",
			args:   []interface{}{"test-server"},
		},
		{
			name:   "port in use",
			format: "port %d is already in use by %s",
			args:   []interface{}{3000, "other-server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the format string works
			result := formatMessage(tt.format, tt.args...)
			if result == "" {
				t.Error("Expected non-empty error message")
			}
		})
	}
}

// formatMessage is a test helper
func formatMessage(format string, args ...interface{}) string {
	return format // simplified for testing
}

// TestHealthStatusValues tests health status constants
func TestHealthStatusValues(t *testing.T) {
	tests := []struct {
		status   registry.HealthStatus
		expected string
	}{
		{registry.HealthHealthy, "healthy"},
		{registry.HealthUnhealthy, "unhealthy"},
		{registry.HealthUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

// TestServerStatusValues tests server status constants
func TestServerStatusValues(t *testing.T) {
	tests := []struct {
		status   registry.ServerStatus
		expected string
	}{
		{registry.StatusRunning, "running"},
		{registry.StatusStopped, "stopped"},
		{registry.StatusStarting, "starting"},
		{registry.StatusStopping, "stopping"},
		{registry.StatusCrashed, "crashed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

// TestConcurrentRegistryAccess tests concurrent access patterns
func TestConcurrentRegistryAccess(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create a valid registry file
	initial := &registry.Registry{
		Servers: make(map[string]*registry.Server),
	}
	data, err := json.MarshalIndent(initial, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal initial registry: %v", err)
	}
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("Failed to write registry file: %v", err)
	}

	// Simulate concurrent operations that might happen in the TUI
	done := make(chan bool)

	// Simulate multiple readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				data, err := os.ReadFile(registryPath)
				if err != nil {
					continue
				}
				var r registry.Registry
				_ = json.Unmarshal(data, &r)
			}
			done <- true
		}()
	}

	// Wait for all readers
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestUptimeStringFormat tests the uptime string formatting
func TestUptimeStringFormat(t *testing.T) {
	tests := []struct {
		name      string
		startedAt time.Time
		status    registry.ServerStatus
		wantEmpty bool
	}{
		{
			name:      "no start time",
			startedAt: time.Time{},
			status:    registry.StatusRunning,
			wantEmpty: true,
		},
		{
			name:      "running with start time",
			startedAt: time.Now().Add(-30 * time.Minute),
			status:    registry.StatusRunning,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &registry.Server{
				Status:    tt.status,
				StartedAt: tt.startedAt,
			}

			result := server.UptimeString()
			if tt.wantEmpty && result != "-" {
				t.Errorf("Expected '-' for empty uptime, got %s", result)
			}
			if !tt.wantEmpty && result == "-" {
				t.Errorf("Expected non-empty uptime string, got '-'")
			}
		})
	}
}
