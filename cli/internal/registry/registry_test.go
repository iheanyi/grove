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
	if r.Workspaces == nil {
		t.Error("Workspaces map should be initialized")
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
		path:       filepath.Join(tmpDir, "nonexistent", "registry.json"),
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	err := r.load()
	if err == nil {
		t.Error("load() should fail for invalid JSON")
	}
}

func TestLoad_ValidJSON_LegacyFormat(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create valid registry JSON in legacy format (no workspaces)
	data := map[string]interface{}{
		"servers": map[string]interface{}{
			"test-server": map[string]interface{}{
				"name":   "test-server",
				"port":   3000,
				"status": "running",
				"pid":    12345,
			},
		},
		"worktrees": map[string]interface{}{},
		"proxy":     map[string]interface{}{"http_port": 80, "https_port": 443},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(registryPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	err = r.load()
	if err != nil {
		t.Errorf("load() failed: %v", err)
	}

	// Should have migrated to workspaces
	if len(r.Workspaces) != 1 {
		t.Errorf("Expected 1 workspace after migration, got %d", len(r.Workspaces))
	}

	ws, ok := r.Workspaces["test-server"]
	if !ok {
		t.Error("Expected test-server workspace to exist")
	} else {
		if ws.Server == nil {
			t.Error("Expected workspace to have server state")
		} else if ws.Server.Port != 3000 {
			t.Errorf("Expected port 3000, got %d", ws.Server.Port)
		}
	}

	// Backward-compatible Get should still work
	server, ok := r.Get("test-server")
	if !ok {
		t.Error("Get() should find test-server")
	} else if server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", server.Port)
	}
}

func TestLoad_ValidJSON_NewFormat(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create valid registry JSON in new format (with workspaces)
	data := map[string]interface{}{
		"workspaces": map[string]interface{}{
			"test-server": map[string]interface{}{
				"name":   "test-server",
				"path":   "/test/path",
				"branch": "main",
				"server": map[string]interface{}{
					"port":   3000,
					"status": "running",
					"pid":    12345,
				},
			},
		},
		"proxy": map[string]interface{}{"http_port": 80, "https_port": 443},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(registryPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	err = r.load()
	if err != nil {
		t.Errorf("load() failed: %v", err)
	}

	// Should load workspaces directly
	if len(r.Workspaces) != 1 {
		t.Errorf("Expected 1 workspace, got %d", len(r.Workspaces))
	}

	ws, ok := r.Workspaces["test-server"]
	if !ok {
		t.Error("Expected test-server workspace to exist")
	} else {
		if ws.Path != "/test/path" {
			t.Errorf("Expected path /test/path, got %s", ws.Path)
		}
		if ws.Server == nil {
			t.Error("Expected workspace to have server state")
		} else if ws.Server.Port != 3000 {
			t.Errorf("Expected port 3000, got %d", ws.Server.Port)
		}
	}
}

func TestSave_Success(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{HTTPPort: 80},
	}

	// Add a workspace with server state
	r.Workspaces["test"] = &Workspace{
		Name: "test",
		Path: "/test/path",
		Server: &ServerState{
			Port:   3000,
			Status: StatusRunning,
		},
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

	var loaded map[string]interface{}
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("Failed to unmarshal saved data: %v", err)
	}

	// Check workspaces exist
	workspaces, ok := loaded["workspaces"].(map[string]interface{})
	if !ok {
		t.Error("Expected workspaces to be saved")
	} else if len(workspaces) != 1 {
		t.Errorf("Expected 1 workspace in saved data, got %d", len(workspaces))
	}

	// Check legacy servers are synced
	servers, ok := loaded["servers"].(map[string]interface{})
	if !ok {
		t.Error("Expected servers to be synced for backward compatibility")
	} else if len(servers) != 1 {
		t.Errorf("Expected 1 server in saved data (legacy), got %d", len(servers))
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
		_ = os.Chmod(readOnlyDir, 0755)
	}()

	registryPath := filepath.Join(readOnlyDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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

	// Verify server was added via Get (backward compatible)
	got, ok := r.Get("new-server")
	if !ok {
		t.Error("Server was not added")
	} else if got.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", got.Port)
	}

	// Verify workspace was created
	ws, ok := r.Workspaces["new-server"]
	if !ok {
		t.Error("Workspace should have been created")
	} else if ws.Server == nil || ws.Server.Port != 3000 {
		t.Error("Workspace should have server state with port 3000")
	}
}

func TestSet_UpdatesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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

	// Verify workspace is also gone
	_, ok = r.Workspaces["to-remove"]
	if ok {
		t.Error("Workspace should have been removed")
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	// Add multiple workspaces
	r.Workspaces["server1"] = &Workspace{Name: "server1", Server: &ServerState{Port: 3000}}
	r.Workspaces["server2"] = &Workspace{Name: "server2", Server: &ServerState{Port: 3001}}
	r.Workspaces["server3"] = &Workspace{Name: "server3", Server: &ServerState{Port: 3002}}

	servers := r.List()
	if len(servers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(servers))
	}
}

func TestListRunning(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	// Add workspaces with different server statuses
	r.Workspaces["running1"] = &Workspace{
		Name:   "running1",
		Server: &ServerState{Status: StatusRunning, PID: os.Getpid()},
	}
	r.Workspaces["running2"] = &Workspace{
		Name:   "running2",
		Server: &ServerState{Status: StatusRunning, PID: os.Getpid()},
	}
	r.Workspaces["stopped"] = &Workspace{
		Name:   "stopped",
		Server: &ServerState{Status: StatusStopped},
	}

	running := r.ListRunning()
	if len(running) != 2 {
		t.Errorf("Expected 2 running servers, got %d", len(running))
	}
}

func TestGetUsedPorts(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	// Add running workspaces with ports
	r.Workspaces["server1"] = &Workspace{
		Name:   "server1",
		Server: &ServerState{Port: 3000, Status: StatusRunning, PID: os.Getpid()},
	}
	r.Workspaces["server2"] = &Workspace{
		Name:   "server2",
		Server: &ServerState{Port: 3001, Status: StatusRunning, PID: os.Getpid()},
	}
	r.Workspaces["stopped"] = &Workspace{
		Name:   "stopped",
		Server: &ServerState{Port: 3002, Status: StatusStopped},
	}

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
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      nil,
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
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	// Add workspace with non-existent PID
	r.Workspaces["dead-server"] = &Workspace{
		Name: "dead-server",
		Server: &ServerState{
			Port:   3000,
			Status: StatusRunning,
			PID:    999999999, // Very high PID that almost certainly doesn't exist
		},
	}

	// Add workspace with valid PID (current process)
	r.Workspaces["alive-server"] = &Workspace{
		Name: "alive-server",
		Server: &ServerState{
			Port:   3001,
			Status: StatusRunning,
			PID:    os.Getpid(),
		},
	}

	result, err := r.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}

	if len(result.Stopped) != 1 || result.Stopped[0] != "dead-server" {
		t.Errorf("Expected [dead-server] to be stopped, got %v", result.Stopped)
	}

	// Verify dead server status changed
	deadWs := r.Workspaces["dead-server"]
	if deadWs.Server.Status != StatusStopped {
		t.Errorf("Expected dead server status to be %s, got %s", StatusStopped, deadWs.Server.Status)
	}
	if deadWs.Server.PID != 0 {
		t.Errorf("Expected dead server PID to be 0, got %d", deadWs.Server.PID)
	}

	// Verify alive server unchanged
	aliveWs := r.Workspaces["alive-server"]
	if aliveWs.Server.Status != StatusRunning {
		t.Errorf("Expected alive server status to remain %s, got %s", StatusRunning, aliveWs.Server.Status)
	}
}

func TestCleanup_NoChangesWhenNoDeadProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	// Add only alive workspace
	r.Workspaces["alive"] = &Workspace{
		Name: "alive",
		Server: &ServerState{
			Status: StatusRunning,
			PID:    os.Getpid(),
		},
	}

	result, err := r.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}

	if len(result.Stopped) != 0 || len(result.RemovedServers) != 0 || len(result.RemovedWorktrees) != 0 {
		t.Errorf("Expected no changes, got stopped=%v, removedServers=%v, removedWorktrees=%v",
			result.Stopped, result.RemovedServers, result.RemovedWorktrees)
	}
}

func TestWorktreeOperations(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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

	// Verify workspace was created
	ws, ok := r.Workspaces["feature-branch"]
	if !ok {
		t.Error("Workspace should have been created from worktree")
	} else if ws.Path != "/path/to/worktree" {
		t.Errorf("Expected workspace path /path/to/worktree, got %s", ws.Path)
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

	// Verify workspace was also removed
	_, ok = r.Workspaces["feature-branch"]
	if ok {
		t.Error("Workspace should have been removed")
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

func TestWorkspaceIsRunning(t *testing.T) {
	tests := []struct {
		name     string
		ws       *Workspace
		expected bool
	}{
		{
			"nil server is not running",
			&Workspace{Name: "test", Server: nil},
			false,
		},
		{
			"running server is running",
			&Workspace{Name: "test", Server: &ServerState{Status: StatusRunning}},
			true,
		},
		{
			"starting server is running",
			&Workspace{Name: "test", Server: &ServerState{Status: StatusStarting}},
			true,
		},
		{
			"stopped server is not running",
			&Workspace{Name: "test", Server: &ServerState{Status: StatusStopped}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ws.IsRunning(); got != tt.expected {
				t.Errorf("Workspace.IsRunning() = %v, want %v", got, tt.expected)
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
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	// Test concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = r.Set(&Server{
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

func TestWorkspaceConversion(t *testing.T) {
	// Test WorkspaceFromServer
	server := &Server{
		Name:      "test-server",
		Port:      3000,
		PID:       12345,
		Status:    StatusRunning,
		URL:       "http://test.localhost",
		Path:      "/test/path",
		Branch:    "main",
		Command:   []string{"npm", "start"},
		LogFile:   "/var/log/test.log",
		StartedAt: time.Now(),
		Tags:      []string{"frontend"},
	}

	ws := WorkspaceFromServer(server)
	if ws == nil {
		t.Fatal("WorkspaceFromServer returned nil")
	}
	if ws.Name != "test-server" {
		t.Errorf("Expected name test-server, got %s", ws.Name)
	}
	if ws.Server == nil {
		t.Fatal("Expected server state")
	}
	if ws.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", ws.Server.Port)
	}

	// Test ToServer round-trip
	backToServer := ws.ToServer()
	if backToServer.Name != server.Name {
		t.Errorf("Expected name %s, got %s", server.Name, backToServer.Name)
	}
	if backToServer.Port != server.Port {
		t.Errorf("Expected port %d, got %d", server.Port, backToServer.Port)
	}

	// Test WorkspaceFromWorktree
	wt := &discovery.Worktree{
		Name:         "feature-branch",
		Path:         "/worktree/path",
		Branch:       "feature",
		MainRepo:     "/main/repo",
		GitDirty:     true,
		HasClaude:    true,
		HasVSCode:    false,
		DiscoveredAt: time.Now(),
	}

	wsFromWt := WorkspaceFromWorktree(wt)
	if wsFromWt == nil {
		t.Fatal("WorkspaceFromWorktree returned nil")
	}
	if wsFromWt.Name != "feature-branch" {
		t.Errorf("Expected name feature-branch, got %s", wsFromWt.Name)
	}
	if wsFromWt.GitDirty != true {
		t.Error("Expected GitDirty to be true")
	}
	if wsFromWt.HasClaude != true {
		t.Error("Expected HasClaude to be true")
	}
	if wsFromWt.Server != nil {
		t.Error("Expected no server state for worktree-only workspace")
	}
}

func TestMigration(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create a registry with both servers and worktrees (legacy format)
	legacyData := map[string]interface{}{
		"servers": map[string]interface{}{
			"my-server": map[string]interface{}{
				"name":   "my-server",
				"port":   3000,
				"status": "running",
				"path":   "/server/path",
			},
		},
		"worktrees": map[string]interface{}{
			"my-server": map[string]interface{}{
				"name":      "my-server",
				"path":      "/server/path",
				"branch":    "main",
				"main_repo": "/main/repo",
				"git_dirty": true,
			},
			"worktree-only": map[string]interface{}{
				"name":   "worktree-only",
				"path":   "/worktree/path",
				"branch": "feature",
			},
		},
		"proxy": map[string]interface{}{},
	}

	jsonData, err := json.MarshalIndent(legacyData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(registryPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load and verify migration
	r := &Registry{
		path:       registryPath,
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
	}

	if err := r.load(); err != nil {
		t.Fatalf("load() failed: %v", err)
	}

	// Should have 2 workspaces
	if len(r.Workspaces) != 2 {
		t.Errorf("Expected 2 workspaces after migration, got %d", len(r.Workspaces))
	}

	// Check merged workspace (server + worktree)
	mergedWs, ok := r.Workspaces["my-server"]
	if !ok {
		t.Fatal("Expected my-server workspace")
	}
	if mergedWs.Server == nil {
		t.Error("Expected server state from server data")
	} else if mergedWs.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", mergedWs.Server.Port)
	}
	if mergedWs.Branch != "main" {
		t.Errorf("Expected branch main from worktree data, got %s", mergedWs.Branch)
	}
	if !mergedWs.GitDirty {
		t.Error("Expected GitDirty from worktree data")
	}
	if mergedWs.MainRepo != "/main/repo" {
		t.Errorf("Expected MainRepo /main/repo, got %s", mergedWs.MainRepo)
	}

	// Check worktree-only workspace
	wtOnlyWs, ok := r.Workspaces["worktree-only"]
	if !ok {
		t.Fatal("Expected worktree-only workspace")
	}
	if wtOnlyWs.Server != nil {
		t.Error("Expected no server state for worktree-only workspace")
	}
	if wtOnlyWs.Branch != "feature" {
		t.Errorf("Expected branch feature, got %s", wtOnlyWs.Branch)
	}
}
