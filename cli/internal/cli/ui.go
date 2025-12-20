package cli

import (
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the interactive TUI dashboard",
	Long: `Launch the interactive TUI dashboard.

The TUI provides a real-time view of all servers with:
- Live status updates
- Log streaming
- Quick actions (start/stop/open)
- Fuzzy search

This is the same as running 'wt' without arguments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}
