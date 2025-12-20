package cli

import (
	"os"

	"github.com/iheanyi/wt/internal/registry"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for wt.

To load completions:

Bash:
  $ wt completion bash > /etc/bash_completion.d/wt
  or
  $ wt completion bash > /usr/local/etc/bash_completion.d/wt

Zsh:
  $ wt completion zsh > "${fpath[1]}/_wt"
  or
  $ wt completion zsh > /usr/local/share/zsh/site-functions/_wt

  You may need to start a new shell for this setup to take effect.

Fish:
  $ wt completion fish > ~/.config/fish/completions/wt.fish
  or
  $ wt completion fish > /usr/local/share/fish/vendor_completions.d/wt.fish`,
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
	Long: `Generate bash completion script for wt.

To load completions:
  $ wt completion bash > /etc/bash_completion.d/wt
  or
  $ wt completion bash > /usr/local/etc/bash_completion.d/wt`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(os.Stdout)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long: `Generate zsh completion script for wt.

To load completions:
  $ wt completion zsh > "${fpath[1]}/_wt"
  or
  $ wt completion zsh > /usr/local/share/zsh/site-functions/_wt

You may need to start a new shell for this setup to take effect.`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Long: `Generate fish completion script for wt.

To load completions:
  $ wt completion fish > ~/.config/fish/completions/wt.fish
  or
  $ wt completion fish > /usr/local/share/fish/vendor_completions.d/wt.fish`,
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
	// For 'wt stop <name>' - complete with running server names
	stopCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRunningServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'wt logs <name>' - complete with all server names
	logsCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getAllServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'wt restart <name>' - complete with server names
	restartCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getAllServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'wt url <name>' - complete with running server names
	urlCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRunningServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'wt open <name>' - complete with running server names
	openCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRunningServerNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// For 'wt switch <name>' - complete with worktree names
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
