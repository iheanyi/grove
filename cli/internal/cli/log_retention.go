package cli

import (
	"time"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/logretention"
	"github.com/iheanyi/grove/internal/registry"
)

func cleanupConfiguredLogs(cfg *config.Config, now time.Time) error {
	reg, err := registry.Load()
	if err != nil {
		return err
	}
	return cleanupConfiguredLogsUsingRegistry(cfg, reg, now)
}

func cleanupConfiguredLogsUsingRegistry(cfg *config.Config, reg *registry.Registry, now time.Time) error {
	retention, err := cfg.LogRetentionDuration()
	if err != nil {
		return err
	}

	servers := reg.List()
	activeLogFiles := make([]string, 0, len(servers))
	for _, server := range servers {
		if protectsLogFromRetention(server.Status) && server.LogFile != "" {
			activeLogFiles = append(activeLogFiles, server.LogFile)
		}
	}

	_, err = logretention.Cleanup(cfg.LogDir, retention, activeLogFiles, now)
	return err
}

func protectsLogFromRetention(status registry.ServerStatus) bool {
	return status == registry.StatusRunning ||
		status == registry.StatusStarting ||
		status == registry.StatusStopping
}
