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
		if server.LogFile != "" && protectsLogFromRetention(server.Status) {
			activeLogFiles = append(activeLogFiles, server.LogFile)
		}
	}

	_, err = logretention.Cleanup(cfg.LogDir, retention, activeLogFiles, now)
	return err
}

// protectsLogFromRetention reports whether a server in the given status may
// still be writing to its log file. Unlike IsRunning, this includes
// StatusStopping: `grove stop` marks a server as stopping before waiting for
// graceful termination, and deleting the log in that window would leave the
// process appending to an unlinked file (or lose the log entirely if the
// stop fails and the server keeps running).
func protectsLogFromRetention(status registry.ServerStatus) bool {
	switch status {
	case registry.StatusRunning, registry.StatusStarting, registry.StatusStopping:
		return true
	default:
		return false
	}
}
