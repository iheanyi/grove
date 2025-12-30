package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iheanyi/grove/internal/discovery"
)

func TestNewRegistry(t *testing.T) {
	r := New()

	if r.Servers == nil {
		t.Error("Servers map should be initialized")
	}
	if r.Worktrees == nil {
		t.Error("Worktrees map should be initialized")
	}
	if r.Proxy == nil {
		t.Error("Proxy should be initialized")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	// Create temp dir with non-existent registry path
	tmpDir := t.TempDir()
	oldPath := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldPath)

	r := &Registry{
		path:      filepath.Join(tmpDir, "nonexistent", "registry.json"),
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	err := r.load()
	if err != nil {
		t.Errorf("load() should succeed for non-existent file, got error: %v", err)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Write invalid JSON
	if err := os.WriteFile(registryPath, []byte("not valid json {{{"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	err := r.load()
	if err == nil {
		t.Error("load() should fail for invalid JSON")
	}
}

func TestLoad_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create valid registry JSON
	data := &Registry{
		Servers: map[string]*Server{
			"test-server": {
				Name:   "test-server",
				Port:   3000,
				Status: StatusRunning,
				PID:    12345,
			},
		},
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{HTTPPort: 80, HTTPSPort: 443},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(registryPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	err = r.load()
	if err != nil {
		t.Errorf("load() failed: %v", err)
	}

	if len(r.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(r.Servers))
	}

	server, ok := r.Servers["test-server"]
	if !ok {
		t.Error("Expected test-server to be loaded")
	} else {
		if server.Port != 3000 {
			t.Errorf("Expected port 3000, got %d", server.Port)
		}
	}
}

func TestSave_Success(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{HTTPPort: 80},
	}

	r.Servers["test"] = &Server{
		Name: "test",
		Port: 3000,
	}

	err := r.Save()
	if err != nil {
		t.Errorf("Save() failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}

	var loaded Registry
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("Failed to unmarshal saved data: %v", err)
	}

	if len(loaded.Servers) != 1 {
		t.Errorf("Expected 1 server in saved data, got %d", len(loaded.Servers))
	}
}

func TestSave_ReadOnlyDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")

	// Create directory and make it read-only
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.Chmod(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}
	defer func() {
		_ = os.Chmod(readOnlyDir, 0755) //nolint:errcheck // Best effort cleanup in test
	}()

	registryPath := filepath.Join(readOnlyDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	err := r.Save()
	if err == nil {
		t.Error("Save() should fail for read-only directory")
	}
}

func TestSet_Success(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	server := &Server{
		Name:   "new-server",
		Port:   3000,
		Status: StatusRunning,
	}

	err := r.Set(server)
	if err != nil {
		t.Errorf("Set() failed: %v", err)
	}

	// Verify server was added
	got, ok := r.Get("new-server")
	if !ok {
		t.Error("Server was not added")
	} else if got.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", got.Port)
	}
}

func TestSet_UpdatesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add initial server
	server := &Server{
		Name:   "server",
		Port:   3000,
		Status: StatusRunning,
	}
	if err := r.Set(server); err != nil {
		t.Fatalf("Failed to set initial server: %v", err)
	}

	// Update server
	server.Port = 4000
	server.Status = StatusStopped
	err := r.Set(server)
	if err != nil {
		t.Errorf("Set() failed on update: %v", err)
	}

	got, _ := r.Get("server")
	if got.Port != 4000 {
		t.Errorf("Expected port 4000 after update, got %d", got.Port)
	}
	if got.Status != StatusStopped {
		t.Errorf("Expected status %s, got %s", StatusStopped, got.Status)
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add a server
	if err := r.Set(&Server{Name: "to-remove", Port: 3000}); err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	// Remove it
	err := r.Remove("to-remove")
	if err != nil {
		t.Errorf("Remove() failed: %v", err)
	}

	// Verify it's gone
	_, ok := r.Get("to-remove")
	if ok {
		t.Error("Server should have been removed")
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add multiple servers
	r.Servers["server1"] = &Server{Name: "server1", Port: 3000}
	r.Servers["server2"] = &Server{Name: "server2", Port: 3001}
	r.Servers["server3"] = &Server{Name: "server3", Port: 3002}

	servers := r.List()
	if len(servers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(servers))
	}
}

func TestListRunning(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add servers with different statuses
	r.Servers["running1"] = &Server{Name: "running1", Status: StatusRunning, PID: os.Getpid()}
	r.Servers["running2"] = &Server{Name: "running2", Status: StatusRunning, PID: os.Getpid()}
	r.Servers["stopped"] = &Server{Name: "stopped", Status: StatusStopped}

	running := r.ListRunning()
	if len(running) != 2 {
		t.Errorf("Expected 2 running servers, got %d", len(running))
	}
}

func TestGetUsedPorts(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add running servers with ports
	r.Servers["server1"] = &Server{Name: "server1", Port: 3000, Status: StatusRunning, PID: os.Getpid()}
	r.Servers["server2"] = &Server{Name: "server2", Port: 3001, Status: StatusRunning, PID: os.Getpid()}
	r.Servers["stopped"] = &Server{Name: "stopped", Port: 3002, Status: StatusStopped}

	ports := r.GetUsedPorts()

	if !ports[3000] {
		t.Error("Port 3000 should be marked as used")
	}
	if !ports[3001] {
		t.Error("Port 3001 should be marked as used")
	}
	if ports[3002] {
		t.Error("Port 3002 should not be marked as used (server stopped)")
	}
}

func TestUpdateProxy(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	proxy := &ProxyInfo{
		PID:       12345,
		HTTPPort:  80,
		HTTPSPort: 443,
		StartedAt: time.Now(),
	}

	err := r.UpdateProxy(proxy)
	if err != nil {
		t.Errorf("UpdateProxy() failed: %v", err)
	}

	got := r.GetProxy()
	if got.PID != 12345 {
		t.Errorf("Expected PID 12345, got %d", got.PID)
	}
	if got.HTTPPort != 80 {
		t.Errorf("Expected HTTPPort 80, got %d", got.HTTPPort)
	}
}

func TestGetProxy_NilProxy(t *testing.T) {
	r := &Registry{
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     nil,
	}

	got := r.GetProxy()
	if got == nil {
		t.Error("GetProxy() should return non-nil ProxyInfo even when Proxy is nil")
	}
}

func TestCleanup_RemovesDeadProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add server with non-existent PID
	r.Servers["dead-server"] = &Server{
		Name:   "dead-server",
		Port:   3000,
		Status: StatusRunning,
		PID:    999999999, // Very high PID that almost certainly doesn't exist
	}

	// Add server with valid PID (current process)
	r.Servers["alive-server"] = &Server{
		Name:   "alive-server",
		Port:   3001,
		Status: StatusRunning,
		PID:    os.Getpid(),
	}

	removed, err := r.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}

	if len(removed) != 1 || removed[0] != "dead-server" {
		t.Errorf("Expected [dead-server] to be removed, got %v", removed)
	}

	// Verify dead server status changed
	deadServer := r.Servers["dead-server"]
	if deadServer.Status != StatusStopped {
		t.Errorf("Expected dead server status to be %s, got %s", StatusStopped, deadServer.Status)
	}
	if deadServer.PID != 0 {
		t.Errorf("Expected dead server PID to be 0, got %d", deadServer.PID)
	}

	// Verify alive server unchanged
	aliveServer := r.Servers["alive-server"]
	if aliveServer.Status != StatusRunning {
		t.Errorf("Expected alive server status to remain %s, got %s", StatusRunning, aliveServer.Status)
	}
}

func TestCleanup_NoChangesWhenNoDeadProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Add only alive server
	r.Servers["alive"] = &Server{
		Name:   "alive",
		Status: StatusRunning,
		PID:    os.Getpid(),
	}

	removed, err := r.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}

	if len(removed) != 0 {
		t.Errorf("Expected no servers removed, got %v", removed)
	}
}

func TestWorktreeOperations(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Test SetWorktree
	wt := &discovery.Worktree{
		Name: "feature-branch",
		Path: "/path/to/worktree",
	}

	err := r.SetWorktree(wt)
	if err != nil {
		t.Errorf("SetWorktree() failed: %v", err)
	}

	// Test GetWorktree
	got, ok := r.GetWorktree("feature-branch")
	if !ok {
		t.Error("GetWorktree() should find the worktree")
	}
	if got.Path != "/path/to/worktree" {
		t.Errorf("Expected path /path/to/worktree, got %s", got.Path)
	}

	// Test ListWorktrees
	worktrees := r.ListWorktrees()
	if len(worktrees) != 1 {
		t.Errorf("Expected 1 worktree, got %d", len(worktrees))
	}

	// Test RemoveWorktree
	err = r.RemoveWorktree("feature-branch")
	if err != nil {
		t.Errorf("RemoveWorktree() failed: %v", err)
	}

	_, ok = r.GetWorktree("feature-branch")
	if ok {
		t.Error("Worktree should have been removed")
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	if !isProcessRunning(os.Getpid()) {
		t.Error("Current process should be detected as running")
	}

	// Test with invalid PID
	if isProcessRunning(-1) {
		t.Error("PID -1 should not be detected as running")
	}

	// Test with very high PID (almost certainly doesn't exist)
	if isProcessRunning(999999999) {
		t.Error("PID 999999999 should not be detected as running")
	}
}

func TestServerStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   ServerStatus
		expected bool
	}{
		{"StatusRunning is running", StatusRunning, true},
		{"StatusStarting is running", StatusStarting, true},
		{"StatusStopped is not running", StatusStopped, false},
		{"StatusStopping is not running", StatusStopping, false},
		{"StatusCrashed is not running", StatusCrashed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				Name:   "test",
				Status: tt.status,
			}
			if got := server.IsRunning(); got != tt.expected {
				t.Errorf("Server.IsRunning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProxyIsRunning(t *testing.T) {
	proxy := &ProxyInfo{
		PID: os.Getpid(),
	}

	if !proxy.IsRunning() {
		t.Error("Proxy with valid PID should be running")
	}

	proxy.PID = 0
	if proxy.IsRunning() {
		t.Error("Proxy with PID 0 should not be running")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:      registryPath,
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
	}

	// Test concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = r.Set(&Server{ //nolint:errcheck // Best effort in concurrent test
				Name: "server",
				Port: 3000 + i,
			})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			r.Get("server")
			r.List()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}
