package cli

import (
	"fmt"

	"github.com/iheanyi/wt/internal/registry"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale entries from the registry",
	Long: `Remove stale entries from the registry.

This command removes entries for servers whose processes are no longer running.
This can happen if a server crashes or is killed externally.`,
	RunE: runCleanup,
}

func runCleanup(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	removed, err := reg.Cleanup()
	if err != nil {
		return fmt.Errorf("failed to cleanup registry: %w", err)
	}

	if len(removed) == 0 {
		fmt.Println("No stale entries found")
		return nil
	}

	fmt.Printf("Cleaned up %d stale entries:\n", len(removed))
	for _, name := range removed {
		fmt.Printf("  - %s\n", name)
	}

	return nil
}
