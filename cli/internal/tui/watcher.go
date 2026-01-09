package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/iheanyi/grove/internal/config"
)

// RegistryChangedMsg is sent when the registry file changes
type RegistryChangedMsg struct{}

// WatchRegistry returns a command that watches the registry file for changes.
// It debounces rapid changes to avoid flooding with messages.
func WatchRegistry() tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			// Fall back to no watching if we can't create watcher
			return nil
		}

		registryPath := config.RegistryPath()
		if err := watcher.Add(registryPath); err != nil {
			// If registry doesn't exist yet, watch the config dir
			configDir := config.ConfigDir()
			if err := watcher.Add(configDir); err != nil {
				watcher.Close()
				return nil
			}
		}

		// Wait for a file change event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				// Only care about write/create events
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					watcher.Close()
					// Small debounce to let writes complete
					time.Sleep(50 * time.Millisecond)
					return RegistryChangedMsg{}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				// Ignore errors, just keep watching
			}
		}
	}
}
