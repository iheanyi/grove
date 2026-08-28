package cli

import (
	"fmt"
	"testing"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

func TestSyncPorts_UpdatesServerPortAndURL(t *testing.T) {
	reg := newTestRegistry(map[string]*registry.Workspace{
		"app": {
			Name: "app",
			Path: "/tmp/app",
			Server: &registry.ServerState{
				Port:   3000,
				PID:    111,
				Status: registry.StatusRunning,
				URL:    "http://localhost:3000",
			},
		},
	})

	originalCfg := cfg
	cfg = config.Default()
	t.Cleanup(func() { cfg = originalCfg })

	restore := stubSyncPortsDeps(t, reg, map[int]int{111: 4010})
	defer restore()

	cmd := newSyncPortsTestCmd(false)
	if err := runSyncPorts(cmd, []string{"app"}); err != nil {
		t.Fatalf("runSyncPorts returned error: %v", err)
	}

	if got := reg.Workspaces["app"].Server.Port; got != 4010 {
		t.Fatalf("expected updated port 4010, got %d", got)
	}
	if got := reg.Workspaces["app"].Server.URL; got != "http://localhost:4010" {
		t.Fatalf("expected updated URL http://localhost:4010, got %q", got)
	}
}

func TestSyncPorts_DoesNotUpdateWhenAlreadyInSync(t *testing.T) {
	reg := newTestRegistry(map[string]*registry.Workspace{
		"app": {
			Name: "app",
			Path: "/tmp/app",
			Server: &registry.ServerState{
				Port:   3000,
				PID:    111,
				Status: registry.StatusRunning,
				URL:    "http://localhost:3000",
			},
		},
	})

	originalCfg := cfg
	cfg = config.Default()
	t.Cleanup(func() { cfg = originalCfg })

	var setCalls int
	restore := stubSyncPortsDeps(t, reg, map[int]int{111: 3000})
	defer restore()
	originalSetter := setServerInRegistry
	setServerInRegistry = func(reg *registry.Registry, server *registry.Server) error {
		setCalls++
		return originalSetter(reg, server)
	}
	defer func() { setServerInRegistry = originalSetter }()

	cmd := newSyncPortsTestCmd(false)
	if err := runSyncPorts(cmd, []string{"app"}); err != nil {
		t.Fatalf("runSyncPorts returned error: %v", err)
	}

	if setCalls != 0 {
		t.Fatalf("expected no registry update calls, got %d", setCalls)
	}
	if got := reg.Workspaces["app"].Server.Port; got != 3000 {
		t.Fatalf("expected unchanged port 3000, got %d", got)
	}
}

func TestSyncPorts_SkipsUpdateOnRunningPortConflict(t *testing.T) {
	reg := newTestRegistry(map[string]*registry.Workspace{
		"app": {
			Name: "app",
			Path: "/tmp/app",
			Server: &registry.ServerState{
				Port:   3000,
				PID:    111,
				Status: registry.StatusRunning,
				URL:    "http://localhost:3000",
			},
		},
		"api": {
			Name: "api",
			Path: "/tmp/api",
			Server: &registry.ServerState{
				Port:   4000,
				PID:    222,
				Status: registry.StatusRunning,
				URL:    "http://localhost:4000",
			},
		},
	})

	originalCfg := cfg
	cfg = config.Default()
	t.Cleanup(func() { cfg = originalCfg })

	var setCalls int
	restore := stubSyncPortsDeps(t, reg, map[int]int{111: 4000, 222: 4000})
	defer restore()
	originalSetter := setServerInRegistry
	setServerInRegistry = func(reg *registry.Registry, server *registry.Server) error {
		setCalls++
		return originalSetter(reg, server)
	}
	defer func() { setServerInRegistry = originalSetter }()

	cmd := newSyncPortsTestCmd(false)
	if err := runSyncPorts(cmd, []string{"app"}); err != nil {
		t.Fatalf("runSyncPorts returned error: %v", err)
	}

	if setCalls != 0 {
		t.Fatalf("expected no registry update calls due to conflict, got %d", setCalls)
	}
	if got := reg.Workspaces["app"].Server.Port; got != 3000 {
		t.Fatalf("expected conflicting server to keep port 3000, got %d", got)
	}
}

func TestSyncPorts_SkipsRunningServerWithMissingPID(t *testing.T) {
	reg := newTestRegistry(map[string]*registry.Workspace{
		"app": {
			Name: "app",
			Path: "/tmp/app",
			Server: &registry.ServerState{
				Port:   3000,
				PID:    0,
				Status: registry.StatusRunning,
				URL:    "http://localhost:3000",
			},
		},
	})

	originalCfg := cfg
	cfg = config.Default()
	t.Cleanup(func() { cfg = originalCfg })

	var setCalls int
	restore := stubSyncPortsDeps(t, reg, map[int]int{})
	defer restore()
	originalSetter := setServerInRegistry
	setServerInRegistry = func(reg *registry.Registry, server *registry.Server) error {
		setCalls++
		return originalSetter(reg, server)
	}
	defer func() { setServerInRegistry = originalSetter }()

	cmd := newSyncPortsTestCmd(true)
	if err := runSyncPorts(cmd, nil); err != nil {
		t.Fatalf("runSyncPorts returned error: %v", err)
	}

	if setCalls != 0 {
		t.Fatalf("expected no registry update calls, got %d", setCalls)
	}
	if got := reg.Workspaces["app"].Server.Port; got != 3000 {
		t.Fatalf("expected unchanged port 3000, got %d", got)
	}
}

func TestSyncPorts_ReturnsErrorForUnknownServerName(t *testing.T) {
	reg := newTestRegistry(map[string]*registry.Workspace{
		"app": {
			Name: "app",
			Path: "/tmp/app",
			Server: &registry.ServerState{
				Port:   3000,
				PID:    111,
				Status: registry.StatusRunning,
				URL:    "http://localhost:3000",
			},
		},
	})

	originalCfg := cfg
	cfg = config.Default()
	t.Cleanup(func() { cfg = originalCfg })

	restore := stubSyncPortsDeps(t, reg, map[int]int{111: 3000})
	defer restore()

	cmd := newSyncPortsTestCmd(false)
	err := runSyncPorts(cmd, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for unknown server name")
	}
}

func newSyncPortsTestCmd(syncAll bool) *cobra.Command {
	cmd := &cobra.Command{Use: "sync-ports"}
	cmd.Flags().Bool("all", false, "Sync all running servers")
	if syncAll {
		_ = cmd.Flags().Set("all", "true")
	}
	return cmd
}

func newTestRegistry(workspaces map[string]*registry.Workspace) *registry.Registry {
	return &registry.Registry{
		Workspaces: workspaces,
		Servers:    make(map[string]*registry.Server),
		Worktrees:  nil,
		Proxy:      &registry.ProxyInfo{},
	}
}

func stubSyncPortsDeps(t *testing.T, reg *registry.Registry, pidToPort map[int]int) func() {
	t.Helper()

	originalLoad := loadRegistry
	originalGetPort := getListeningPortByPID
	originalSet := setServerInRegistry

	loadRegistry = func() (*registry.Registry, error) {
		return reg, nil
	}
	getListeningPortByPID = func(pid int) int {
		return pidToPort[pid]
	}
	setServerInRegistry = func(reg *registry.Registry, server *registry.Server) error {
		ws, ok := reg.Workspaces[server.Name]
		if !ok {
			return fmt.Errorf("workspace not found: %s", server.Name)
		}
		if ws.Server == nil {
			ws.Server = &registry.ServerState{}
		}
		ws.Server.Port = server.Port
		ws.Server.URL = server.URL
		ws.Server.PID = server.PID
		ws.Server.Status = server.Status
		return nil
	}

	return func() {
		loadRegistry = originalLoad
		getListeningPortByPID = originalGetPort
		setServerInRegistry = originalSet
	}
}
