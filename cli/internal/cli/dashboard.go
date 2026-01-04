package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iheanyi/grove/internal/dashboard"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the web dashboard for managing workspaces and agents",
	Long: `Launch a web-based dashboard that provides a visual interface for managing
Grove workspaces, servers, and AI agents.

The dashboard shows:
  - All registered workspaces and their status
  - Running servers with health information
  - Active AI agents (Claude Code, etc.)
  - Real-time updates via WebSocket

Examples:
  grove dashboard              # Start on default port 3099
  grove dashboard --port 8080  # Start on custom port
  grove dashboard --no-browser # Don't open browser automatically
  grove dashboard --dev        # Dev mode: proxy to Vite dev server`,
	RunE: runDashboard,
}

func init() {
	dashboardCmd.Flags().Int("port", 3099, "Port to run the dashboard server on")
	dashboardCmd.Flags().Bool("no-browser", false, "Don't open browser automatically")
	dashboardCmd.Flags().Bool("dev", false, "Development mode: proxy to Vite dev server")
	dashboardCmd.Flags().String("dev-url", "http://localhost:5173", "Vite dev server URL (used with --dev)")
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	devMode, _ := cmd.Flags().GetBool("dev")
	devURL, _ := cmd.Flags().GetString("dev-url")

	cfg := dashboard.Config{
		Port:    port,
		DevMode: devMode,
		DevURL:  devURL,
	}

	server, err := dashboard.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create dashboard server: %w", err)
	}

	// Open browser unless disabled
	if !noBrowser {
		go func() {
			// Small delay to let server start
			url := server.URL()
			if err := dashboard.OpenBrowser(url); err != nil {
				log.Printf("Failed to open browser: %v", err)
				log.Printf("Please open %s manually", url)
			}
		}()
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down dashboard server...")
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
		os.Exit(0)
	}()

	// Print startup message
	if devMode {
		fmt.Printf("Dashboard starting in development mode\n")
		fmt.Printf("  Proxying to: %s\n", devURL)
	}
	fmt.Printf("Dashboard available at: %s\n", server.URL())
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Start the server (blocks)
	return server.Start()
}
