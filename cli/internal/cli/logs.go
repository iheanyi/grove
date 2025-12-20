package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/iheanyi/wt/internal/registry"
	"github.com/iheanyi/wt/internal/worktree"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Stream logs for a server",
	Long: `Stream logs for the current worktree's server or a named server.

Examples:
  wt logs              # Stream logs for current worktree
  wt logs feature-auth # Stream logs for named server
  wt logs -n 50        # Show last 50 lines
  wt logs -f           # Follow logs (stream new lines)`,
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().IntP("lines", "n", 20, "Number of lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow logs (stream new lines)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	lines, _ := cmd.Flags().GetInt("lines")
	follow, _ := cmd.Flags().GetBool("follow")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Determine which server
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

	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'", name)
	}

	if server.LogFile == "" {
		return fmt.Errorf("no log file configured for '%s'", name)
	}

	// Check if log file exists
	if _, err := os.Stat(server.LogFile); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", server.LogFile)
	}

	if follow {
		return tailFollow(server.LogFile)
	}

	return tailLines(server.LogFile, lines)
}

// tailLines shows the last n lines of a file
func tailLines(path string, n int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read all lines (simple implementation)
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Get last n lines
	start := 0
	if len(allLines) > n {
		start = len(allLines) - n
	}

	for _, line := range allLines[start:] {
		fmt.Println(line)
	}

	return nil
}

// tailFollow follows the log file and prints new lines
func tailFollow(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to end
	file.Seek(0, io.SeekEnd)

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait for more data
				continue
			}
			return err
		}
		fmt.Print(line)
	}
}
