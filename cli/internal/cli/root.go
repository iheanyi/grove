package cli

import (
	"fmt"
	"os"

	"github.com/iheanyi/wt/internal/config"
	"github.com/iheanyi/wt/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Worktree Server Manager - Manage dev servers across git worktrees",
	Long: `wt is a CLI tool that automatically manages dev servers across git worktrees
with clean localhost URLs like https://feature-branch.localhost.

When run without arguments, it launches an interactive TUI dashboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior: launch TUI
		return runTUI()
	},
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/wt/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(urlCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)

	// Worktree management commands
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(infoCmd)
}

func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = config.Default()
	}
}

func runTUI() error {
	return tui.Run(cfg)
}
