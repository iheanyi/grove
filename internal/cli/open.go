package cli

import (
	"fmt"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/iheanyi/grove/pkg/browser"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [name]",
	Short: "Open a server in the browser",
	Long: `Open the current worktree's server or a named server in the default browser.

Examples:
  grove open              # Open current worktree's server
  grove open feature-auth # Open named server`,
	RunE: runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Determine which server
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use current worktree
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect worktree: %w", err)
		}
		name = wt.Name
	}

	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'\nUse 'grove start' to start a server first", name)
	}

	if !server.IsRunning() {
		return fmt.Errorf("server '%s' is not running\nUse 'grove start' to start it", name)
	}

	fmt.Printf("Opening %s...\n", server.URL)
	return browser.Open(server.URL)
}
