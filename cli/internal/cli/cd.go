package cli

import (
	"fmt"

	"github.com/iheanyi/wt/internal/registry"
	"github.com/iheanyi/wt/internal/worktree"
	"github.com/spf13/cobra"
)

var cdCmd = &cobra.Command{
	Use:   "cd [name]",
	Short: "Print the path to a worktree's directory",
	Long: `Print the path to a worktree's directory for shell integration.

This command is designed to work with a shell function. Add this to your
shell configuration:

  # Bash/Zsh
  wtcd() { cd "$(wt cd "$@")" }

  # Fish
  function wtcd; cd (wt cd $argv); end

Then use 'wtcd <name>' to change to a worktree's directory.

Examples:
  wt cd                # Print path for current worktree
  wt cd feature-auth   # Print path for named worktree
  wtcd feature-auth    # Change to worktree directory (with shell function)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCd,
}

func init() {
	rootCmd.AddCommand(cdCmd)
}

func runCd(cmd *cobra.Command, args []string) error {
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

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'", name)
	}

	if server.Path == "" {
		return fmt.Errorf("no path stored for '%s'", name)
	}

	// Just print the path - the shell function handles the cd
	fmt.Println(server.Path)
	return nil
}
