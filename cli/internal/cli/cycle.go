package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/pkg/browser"
	"github.com/spf13/cobra"
)

var cycleCmd = &cobra.Command{
	Use:   "cycle",
	Short: "Cycle through running servers in your browser",
	Long: `Cycle through all running servers, opening each one in your default browser.

Each invocation opens the next server's URL, wrapping back to the first when reaching the end.
This is useful for quickly switching between different development environments.

Examples:
  grove cycle              # Open next running server in browser
  grove cycle --reset      # Reset to first server
  grove cycle --list       # Show all servers in cycle order`,
	RunE: runCycle,
}

func init() {
	cycleCmd.Flags().Bool("reset", false, "Reset cycle index to first server")
	cycleCmd.Flags().Bool("list", false, "List all servers in cycle order without opening")
	rootCmd.AddCommand(cycleCmd)
}

// getCycleIndexPath returns the path to the cycle index file
func getCycleIndexPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to home directory
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "grove", "cycle-index")
}

// readCycleIndex reads the current cycle index from the cache file
func readCycleIndex() int {
	path := getCycleIndexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	index, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}

	return index
}

// writeCycleIndex writes the cycle index to the cache file
func writeCycleIndex(index int) error {
	path := getCycleIndexPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return os.WriteFile(path, []byte(strconv.Itoa(index)), 0644)
}

func runCycle(cmd *cobra.Command, args []string) error {
	resetFlag, _ := cmd.Flags().GetBool("reset")
	listFlag, _ := cmd.Flags().GetBool("list")

	// Load registry and get running servers
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get running workspaces
	running := reg.ListRunningWorkspaces()
	if len(running) == 0 {
		fmt.Println("No running servers to cycle through.")
		fmt.Println("\nStart a server with: grove start")
		return nil
	}

	// Sort by name for consistent ordering
	sort.Slice(running, func(i, j int) bool {
		return running[i].Name < running[j].Name
	})

	// Handle reset flag
	if resetFlag {
		if err := writeCycleIndex(0); err != nil {
			return fmt.Errorf("failed to reset cycle index: %w", err)
		}
		fmt.Println("Cycle index reset to first server.")
		return nil
	}

	// Handle list flag
	if listFlag {
		fmt.Printf("Running servers (%d):\n\n", len(running))
		currentIndex := readCycleIndex() % len(running)
		for i, ws := range running {
			marker := "  "
			if i == currentIndex {
				marker = "→ "
			}
			fmt.Printf("%s%d. %s - %s\n", marker, i+1, ws.Name, ws.GetURL())
		}
		return nil
	}

	// Get current index
	currentIndex := readCycleIndex()

	// Ensure index is within bounds (handles case where servers were stopped)
	if currentIndex >= len(running) {
		currentIndex = 0
	}

	// Get the current server
	ws := running[currentIndex]
	url := ws.GetURL()

	if url == "" {
		return fmt.Errorf("server %s has no URL", ws.Name)
	}

	// Open in browser
	if err := browser.Open(url); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	fmt.Printf("Opened %s (%s)\n", ws.Name, url)

	// Increment index and wrap around
	nextIndex := (currentIndex + 1) % len(running)
	if err := writeCycleIndex(nextIndex); err != nil {
		return fmt.Errorf("failed to save cycle index: %w", err)
	}

	// Show position
	if len(running) > 1 {
		fmt.Printf("[%d/%d servers]\n", currentIndex+1, len(running))
	}

	return nil
}
