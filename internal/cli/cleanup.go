package cli

import (
	"fmt"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale entries from the registry",
	Long: `Remove stale entries from the registry.

This command:
- Removes entries for worktrees whose paths no longer exist (deleted directories)
- Marks servers as stopped if their processes are no longer running

Use this to clean up after deleting worktrees or when servers crash.`,
	RunE: runCleanup,
}

func runCleanup(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	result, err := reg.Cleanup()
	if err != nil {
		return fmt.Errorf("failed to cleanup registry: %w", err)
	}

	totalRemoved := len(result.RemovedServers) + len(result.RemovedWorktrees)
	if len(result.Stopped) == 0 && totalRemoved == 0 {
		fmt.Println("No stale entries found")
		return nil
	}

	if len(result.RemovedWorktrees) > 0 {
		fmt.Printf("Removed %d worktrees (path no longer exists):\n", len(result.RemovedWorktrees))
		for _, name := range result.RemovedWorktrees {
			fmt.Printf("  - %s\n", name)
		}
	}

	if len(result.RemovedServers) > 0 {
		fmt.Printf("Removed %d servers (path no longer exists):\n", len(result.RemovedServers))
		for _, name := range result.RemovedServers {
			fmt.Printf("  - %s\n", name)
		}
	}

	if len(result.Stopped) > 0 {
		fmt.Printf("Marked %d servers as stopped (process not running):\n", len(result.Stopped))
		for _, name := range result.Stopped {
			fmt.Printf("  - %s\n", name)
		}
	}

	return nil
}
