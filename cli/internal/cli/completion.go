package cli

import (
	"os"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for grove.

To load completions:

Bash:
  $ grove completion bash > /etc/bash_completion.d/grove
  or
  $ grove completion bash > /usr/local/etc/bash_completion.d/grove

Zsh:
  $ grove completion zsh > "${fpath[1]}/_grove"
  or
  $ grove completion zsh > /usr/local/share/zsh/site-functions/_grove

  You may need to start a new shell for this setup to take effect.

Fish:
  $ grove completion fish > ~/.config/fish/completions/grove.fish
  or
  $ grove completion fish > /usr/local/share/fish/vendor_completions.d/grove.fish`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  runCompletion,
}

func init() {
	// Add subcommands for each shell
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)

	// Setup dynamic completions for commands that take server names
	setupDynamicCompletions()
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long: `Generate bash completion script for grove.

To load completions:
  $ grove completion bash > /etc/bash_completion.d/grove
  or
  $ grove completion bash > /usr/local/etc/bash_completion.d/grove`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(os.Stdout)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long: `Generate zsh completion script for grove.

To load completions:
  $ grove completion zsh > "${fpath[1]}/_grove"
  or
  $ grove completion zsh > /usr/local/share/zsh/site-functions/_grove

You may need to start a new shell for this setup to take effect.`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Long: `Generate fish completion script for grove.

To load completions:
  $ grove completion fish > ~/.config/fish/completions/grove.fish
  or
  $ grove completion fish > /usr/local/share/fish/vendor_completions.d/grove.fish`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	}
	return nil
}

// setupDynamicCompletions adds dynamic completion functions for commands
func setupDynamicCompletions() {
	// For 'grove stop <name>' - complete with running server names
	stopCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRunningServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'grove logs <name>' - complete with all server names
	logsCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getAllServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'grove restart <name>' - complete with server names
	restartCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getAllServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'grove url <name>' - complete with running server names
	urlCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRunningServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'grove open <name>' - complete with running server names
	openCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRunningServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'grove switch <name>' - complete with worktree names
	switchCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getWorktreeNames(), cobra.ShellCompDirectiveNoFileComp
	}
}

// getRunningServerNames returns a list of running server names for completion
func getRunningServerNames() []string {
	reg, err := registry.Load()
	if err != nil {
		return nil
	}

	running := reg.ListRunning()
	names := make([]string, 0, len(running))
	for _, server := range running {
		names = append(names, server.Name)
	}
	return names
}

// getAllServerNames returns a list of all server names for completion
func getAllServerNames() []string {
	reg, err := registry.Load()
	if err != nil {
		return nil
	}

	servers := reg.List()
	names := make([]string, 0, len(servers))
	for _, server := range servers {
		names = append(names, server.Name)
	}
	return names
}

// getWorktreeNames returns a list of worktree names for completion
func getWorktreeNames() []string {
	reg, err := registry.Load()
	if err != nil {
		return nil
	}

	worktrees := reg.ListWorktrees()
	names := make([]string, 0, len(worktrees))
	for _, wt := range worktrees {
		names = append(names, wt.Name)
	}
	return names
}
