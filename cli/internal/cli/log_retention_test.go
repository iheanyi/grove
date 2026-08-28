package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
)

func TestConfiguredRetentionPreservesRunningServerLog(t *testing.T) {
	logDir := t.TempDir()
	activeLog := filepath.Join(logDir, "running-server.log")
	if err := os.WriteFile(activeLog, []byte("still streaming"), 0644); err != nil {
		t.Fatalf("write active log: %v", err)
	}
	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	old := now.Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(activeLog, old, old); err != nil {
		t.Fatalf("age active log: %v", err)
	}

	cfg := config.Default()
	cfg.LogDir = logDir
	reg := registry.New()
	reg.SetWorkspaceWithoutSave(&registry.Workspace{
		Name: "running-server",
		Server: &registry.ServerState{
			Status:  registry.StatusRunning,
			LogFile: activeLog,
		},
	})

	if err := cleanupConfiguredLogsUsingRegistry(cfg, reg, now); err != nil {
		t.Fatalf("cleanupConfiguredLogsUsingRegistry() error = %v", err)
	}
	if _, err := os.Stat(activeLog); err != nil {
		t.Fatalf("running server log was removed: %v", err)
	}
}

func TestConfiguredRetentionPreservesStoppingServerLog(t *testing.T) {
	logDir := t.TempDir()
	activeLog := filepath.Join(logDir, "stopping-server.log")
	if err := os.WriteFile(activeLog, []byte("draining"), 0644); err != nil {
		t.Fatalf("write active log: %v", err)
	}
	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	old := now.Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(activeLog, old, old); err != nil {
		t.Fatalf("age active log: %v", err)
	}

	cfg := config.Default()
	cfg.LogDir = logDir
	reg := registry.New()
	reg.SetWorkspaceWithoutSave(&registry.Workspace{
		Name: "stopping-server",
		Server: &registry.ServerState{
			Status:  registry.StatusStopping,
			LogFile: activeLog,
		},
	})

	if err := cleanupConfiguredLogsUsingRegistry(cfg, reg, now); err != nil {
		t.Fatalf("cleanupConfiguredLogsUsingRegistry() error = %v", err)
	}
	if _, err := os.Stat(activeLog); err != nil {
		t.Fatalf("stopping server log was removed: %v", err)
	}
}
