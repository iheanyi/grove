package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <worktree-name>",
	Short: "Open a new terminal tab/window in the specified worktree",
	Long: `Open a new terminal tab/window in the specified worktree.

On macOS, this uses osascript to open a new Terminal tab/window.
Optionally starts the dev server if not already running.

Examples:
  grove switch myrepo-feature-auth         # Switch to worktree
  grove switch myrepo-feature-auth --start # Switch and start dev server`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitch,
}

func init() {
	switchCmd.Flags().Bool("start", false, "Start the dev server if not already running")
}

func runSwitch(cmd *cobra.Command, args []string) error {
	worktreeName := args[0]
	startServer, _ := cmd.Flags().GetBool("start")

	// Detect current worktree to find the main repo
	currentWt, err := worktree.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect current repository: %w", err)
	}

	// Determine the main repository path
	mainRepoPath := currentWt.Path
	if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
		mainRepoPath = currentWt.MainWorktreePath
	}

	// Find the target worktree
	worktreePath, err := findWorktree(mainRepoPath, worktreeName)
	if err != nil {
		return err
	}

	// Verify the worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree directory does not exist: %s", worktreePath)
	}

	fmt.Printf("Switching to worktree: %s\n", worktreeName)
	fmt.Printf("Path: %s\n", worktreePath)

	// Open terminal based on platform
	if err := openTerminal(worktreePath); err != nil {
		return fmt.Errorf("failed to open terminal: %w", err)
	}

	// Optionally start the dev server
	if startServer {
		fmt.Println("\nStarting dev server...")

		// Load registry to check if already running
		reg, err := registry.Load()
		if err != nil {
			return fmt.Errorf("failed to load registry: %w", err)
		}

		// Detect the target worktree info
		targetWt, err := worktree.DetectAt(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to detect target worktree: %w", err)
		}

		// Check if already running
		if existing, ok := reg.Get(targetWt.Name); ok && existing.IsRunning() {
			fmt.Printf("Server is already running at: %s\n", existing.URL)
		} else {
			fmt.Println("Note: Use 'grove start' in the new terminal to start the dev server")
			fmt.Println("(Auto-start from switch command would require backgrounding)")
		}
	}

	fmt.Println("\nTerminal opened successfully!")

	return nil
}

// findWorktree finds the path to a worktree given its name
func findWorktree(mainRepoPath, worktreeName string) (string, error) {
	// List all worktrees
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the worktree list
	lines := strings.Split(string(output), "\n")
	var currentPath string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") && currentPath != "" {
			// Check if this worktree path matches the name
			baseName := filepath.Base(currentPath)
			if baseName == worktreeName {
				return currentPath, nil
			}
		}
	}

	// If not found by exact match, try parent directory + name
	parentDir := filepath.Dir(mainRepoPath)
	candidatePath := filepath.Join(parentDir, worktreeName)

	// Verify this is actually a worktree
	cmd = exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err = cmd.Output()
	if err == nil {
		if strings.Contains(string(output), candidatePath) {
			return candidatePath, nil
		}
	}

	return "", fmt.Errorf("worktree '%s' not found\nUse 'git worktree list' to see available worktrees", worktreeName)
}

// openTerminal opens a new terminal window/tab on the current platform
func openTerminal(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return openMacOSTerminal(path)
	case "linux":
		return openLinuxTerminal(path)
	default:
		return fmt.Errorf("unsupported platform: %s\nPlease open the terminal manually at: %s", runtime.GOOS, path)
	}
}

// openMacOSTerminal opens a new Terminal tab on macOS
func openMacOSTerminal(path string) error {
	// AppleScript to open a new Terminal tab and cd to the directory
	script := fmt.Sprintf(`
		tell application "Terminal"
			activate
			tell application "System Events" to keystroke "t" using command down
			delay 0.5
			do script "cd %s; clear" in front window
		end tell
	`, shellEscape(path))

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		// Fallback: try opening a new window if tab fails
		script = fmt.Sprintf(`
			tell application "Terminal"
				activate
				do script "cd %s; clear"
			end tell
		`, shellEscape(path))

		cmd = exec.Command("osascript", "-e", script)
		return cmd.Run()
	}

	return nil
}

// openLinuxTerminal opens a new terminal on Linux
func openLinuxTerminal(path string) error {
	// Try common terminal emulators
	terminals := []struct {
		name string
		args []string
	}{
		{"gnome-terminal", []string{"--working-directory=" + path}},
		{"konsole", []string{"--workdir", path}},
		{"xfce4-terminal", []string{"--working-directory=" + path}},
		{"xterm", []string{"-e", fmt.Sprintf("cd %s && $SHELL", shellEscape(path))}},
	}

	for _, term := range terminals {
		// Check if terminal exists
		if _, err := exec.LookPath(term.name); err == nil {
			cmd := exec.Command(term.name, term.args...)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no supported terminal emulator found\nPlease open the terminal manually at: %s", path)
}

// shellEscape escapes a string for safe use in shell commands
func shellEscape(s string) string {
	// Simple escape: wrap in single quotes and escape any single quotes
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}
