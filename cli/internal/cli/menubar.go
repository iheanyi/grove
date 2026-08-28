package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var menubarCmd = &cobra.Command{
	Use:   "menubar",
	Short: "Manage the Grove menubar app",
	Long: `Manage the Grove menubar application.

The menubar app provides a native macOS interface for managing Grove servers.

Examples:
  grove menubar start   # Start the menubar app in background
  grove menubar stop    # Stop the menubar app
  grove menubar status  # Check if menubar app is running`,
}

var menubarStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Grove menubar app",
	Long: `Start the Grove menubar app in the background.

Looks for Grove.app in these locations (in order):
  1. /Applications/Grove.app (Homebrew cask install)
  2. ~/Applications/Grove.app
  3. Development build location`,
	RunE: runMenubarStart,
}

var menubarStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Grove menubar app",
	RunE:  runMenubarStop,
}

var menubarStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the menubar app is running",
	RunE:  runMenubarStatus,
}

func init() {
	menubarCmd.AddCommand(menubarStartCmd)
	menubarCmd.AddCommand(menubarStopCmd)
	menubarCmd.AddCommand(menubarStatusCmd)
}

func runMenubarStart(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("menubar app is only available on macOS")
	}

	app := findGroveApp()
	if app == nil {
		return fmt.Errorf("Grove.app not found\n\nInstall with: brew install --cask iheanyi/tap/grove-menubar")
	}

	// Check if already running
	if isMenubarRunning() {
		fmt.Println("Grove menubar is already running")
		return nil
	}

	var startCmd *exec.Cmd
	if app.isBundle {
		// Use 'open -g' to launch .app bundle in background (non-blocking)
		startCmd = exec.Command("open", "-g", app.path)
	} else {
		// Launch binary directly in background
		startCmd = exec.Command(app.path)
		startCmd.Stdout = nil
		startCmd.Stderr = nil
		startCmd.Stdin = nil
		// Detach from parent process
		startCmd.SysProcAttr = nil
	}

	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("failed to start menubar app: %w", err)
	}

	// For direct binary execution, don't wait
	if !app.isBundle {
		// Release the process so it continues running after we exit
		if startCmd.Process != nil {
			if err := startCmd.Process.Release(); err != nil {
				// Non-fatal: process started successfully, just couldn't detach cleanly
				fmt.Printf("Warning: could not detach process: %v\n", err)
			}
		}
	}

	fmt.Printf("Started Grove menubar (%s)\n", app.path)
	return nil
}

func runMenubarStop(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("menubar app is only available on macOS")
	}

	if !isMenubarRunning() {
		fmt.Println("Grove menubar is not running")
		return nil
	}

	// Try to stop both possible process names
	// pkill returns exit code 1 if no processes matched, which is fine
	devErr := exec.Command("pkill", "-x", "GroveMenubar").Run()
	prodErr := exec.Command("pkill", "-x", "Grove").Run()

	// If both failed, something unexpected happened
	if devErr != nil && prodErr != nil {
		return fmt.Errorf("failed to stop menubar (was it already stopped?)")
	}

	fmt.Println("Stopped Grove menubar")
	return nil
}

func runMenubarStatus(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("menubar app is only available on macOS")
	}

	if isMenubarRunning() {
		fmt.Println("Grove menubar is running")
	} else {
		fmt.Println("Grove menubar is not running")
		app := findGroveApp()
		if app != nil {
			fmt.Printf("Found at: %s\n", app.path)
			fmt.Println("Start with: grove menubar start")
		} else {
			fmt.Println("Not installed. Install with: brew install --cask iheanyi/tap/grove-menubar")
		}
	}
	return nil
}

// groveAppLocation holds info about where Grove was found
type groveAppLocation struct {
	path     string
	isBundle bool // true for .app bundles, false for raw binary
}

// findGroveApp looks for Grove.app or GroveMenubar binary in standard locations
func findGroveApp() *groveAppLocation {
	home, _ := os.UserHomeDir()

	// First check for .app bundles (production installs)
	appBundles := []string{
		"/Applications/Grove.app",
		filepath.Join(home, "Applications", "Grove.app"),
	}

	for _, loc := range appBundles {
		if _, err := os.Stat(loc); err == nil {
			return &groveAppLocation{path: loc, isBundle: true}
		}
	}

	// Then check for development binaries
	devBinaries := []string{}

	// Try to find the grove repo root from current directory
	if repoRoot := findGroveRepoRoot(); repoRoot != "" {
		devBinaries = append(devBinaries,
			filepath.Join(repoRoot, "menubar/GroveMenubar/.build/arm64-apple-macosx/debug/GroveMenubar"),
			filepath.Join(repoRoot, "menubar/GroveMenubar/.build/debug/GroveMenubar"),
			filepath.Join(repoRoot, "menubar/GroveMenubar/.build/arm64-apple-macosx/release/GroveMenubar"),
			filepath.Join(repoRoot, "menubar/GroveMenubar/.build/release/GroveMenubar"),
		)
	}

	for _, loc := range devBinaries {
		if _, err := os.Stat(loc); err == nil {
			return &groveAppLocation{path: loc, isBundle: false}
		}
	}

	return nil
}

// findGroveRepoRoot walks up from cwd looking for the grove repo root
// It identifies the repo by checking for the menubar/GroveMenubar directory structure
func findGroveRepoRoot() string {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Also check from executable location (in case running installed CLI from within repo)
	exeDir := ""
	if exePath, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exePath)
	}

	// Walk up from both locations
	for _, startDir := range []string{dir, exeDir} {
		if startDir == "" {
			continue
		}

		current := startDir
		for {
			// Check if this looks like the grove repo root
			if isGroveRepoRoot(current) {
				return current
			}

			// Go up one directory
			parent := filepath.Dir(current)
			if parent == current {
				// Reached filesystem root
				break
			}
			current = parent
		}
	}

	return ""
}

// isGroveRepoRoot checks if a directory is the grove repo root
func isGroveRepoRoot(dir string) bool {
	// Check for the menubar directory structure
	menubarPath := filepath.Join(dir, "menubar", "GroveMenubar", "Package.swift")
	if _, err := os.Stat(menubarPath); err == nil {
		return true
	}

	// Also check for cli/go.mod with the right module name
	goModPath := filepath.Join(dir, "cli", "go.mod")
	if data, err := os.ReadFile(goModPath); err == nil {
		if strings.Contains(string(data), "github.com/iheanyi/grove") {
			return true
		}
	}

	return false
}

// isMenubarRunning checks if the menubar app is currently running
func isMenubarRunning() bool {
	// Check for both release name (Grove) and dev name (GroveMenubar)
	cmd := exec.Command("pgrep", "-x", "Grove")
	if err := cmd.Run(); err == nil {
		return true
	}

	cmd = exec.Command("pgrep", "-x", "GroveMenubar")
	return cmd.Run() == nil
}
