package cli

import (
	"fmt"
	"os"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "grove",
	Short: "Worktree Server Manager - Manage dev servers across git worktrees",
	Long: `grove is a CLI tool that automatically manages dev servers across git worktrees
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/grove/config.yaml)")

	// Define command groups
	rootCmd.AddGroup(
		&cobra.Group{ID: "server", Title: "Server Management:"},
		&cobra.Group{ID: "worktree", Title: "Worktree Management:"},
		&cobra.Group{ID: "monitoring", Title: "Logs & Monitoring:"},
		&cobra.Group{ID: "config", Title: "Configuration:"},
		&cobra.Group{ID: "proxy", Title: "Proxy:"},
		&cobra.Group{ID: "maintenance", Title: "Maintenance:"},
	)

	// Server Management
	startCmd.GroupID = "server"
	stopCmd.GroupID = "server"
	restartCmd.GroupID = "server"
	lsCmd.GroupID = "server"
	statusCmd.GroupID = "server"
	urlCmd.GroupID = "server"
	openCmd.GroupID = "server"
	attachCmd.GroupID = "server"
	detachCmd.GroupID = "server"
	tagCmd.GroupID = "server"

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(urlCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(detachCmd)
	rootCmd.AddCommand(tagCmd)

	// Worktree Management
	newCmd.GroupID = "worktree"
	switchCmd.GroupID = "worktree"
	cloneCmd.GroupID = "worktree"
	cdCmd.GroupID = "worktree"
	deleteCmd.GroupID = "worktree"
	infoCmd.GroupID = "worktree"
	pruneCmd.GroupID = "worktree"

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(cdCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(pruneCmd)

	// Logs & Monitoring
	logsCmd.GroupID = "monitoring"

	rootCmd.AddCommand(logsCmd)

	// Configuration
	initCmd.GroupID = "config"
	setupCmd.GroupID = "config"

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setupCmd)

	// Proxy
	proxyCmd.GroupID = "proxy"

	rootCmd.AddCommand(proxyCmd)

	// Maintenance
	doctorCmd.GroupID = "maintenance"
	cleanupCmd.GroupID = "maintenance"
	uiCmd.GroupID = "maintenance"
	versionCmd.GroupID = "maintenance"
	completionCmd.GroupID = "maintenance"
	menubarCmd.GroupID = "maintenance"

	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(menubarCmd)
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
