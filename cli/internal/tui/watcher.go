package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/iheanyi/grove/internal/config"
)

// RegistryChangedMsg is sent when the registry file changes
type RegistryChangedMsg struct{}

// registryWatcher is a persistent watcher shared across WatchRegistry calls.
// This avoids the overhead of creating and destroying an fsnotify watcher
// for every single file change event.
var registryWatcher *fsnotify.Watcher

// WatchRegistry returns a command that watches the registry file for changes.
// It reuses a persistent fsnotify watcher to avoid create/destroy overhead per event.
func WatchRegistry() tea.Cmd {
	return func() tea.Msg {
		// Initialize the persistent watcher on first call
		if registryWatcher == nil {
			w, err := fsnotify.NewWatcher()
			if err != nil {
				return nil
			}
			registryWatcher = w

			registryPath := config.RegistryPath()
			if err := registryWatcher.Add(registryPath); err != nil {
				// If registry doesn't exist yet, watch the config dir
				configDir := config.ConfigDir()
				if err := registryWatcher.Add(configDir); err != nil {
					registryWatcher.Close()
					registryWatcher = nil
					return nil
				}
			}
		}

		// Wait for a file change event on the persistent watcher
		for {
			select {
			case event, ok := <-registryWatcher.Events:
				if !ok {
					// Watcher closed, recreate on next call
					registryWatcher = nil
					return nil
				}
				// Only care about write/create events
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					// Small debounce to let writes complete
					time.Sleep(50 * time.Millisecond)
					return RegistryChangedMsg{}
				}
			case _, ok := <-registryWatcher.Errors:
				if !ok {
					registryWatcher = nil
					return nil
				}
				// Ignore errors, keep watching
			}
		}
	}
}
