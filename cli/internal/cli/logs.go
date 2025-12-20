package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/iheanyi/grove/internal/loghighlight"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Stream logs for a server",
	Long: `Stream logs for the current worktree's server or a named server.

Logs are syntax-highlighted with colors for:
  - Log levels (ERROR, WARN, INFO, DEBUG)
  - HTTP methods (GET, POST, PUT, DELETE)
  - Status codes (2xx green, 4xx orange, 5xx red)
  - Timestamps, durations, Rails patterns

Examples:
  grove logs              # Stream logs for current worktree
  grove logs feature-auth # Stream logs for named server
  grove logs -n 50        # Show last 50 lines
  grove logs -f           # Follow logs (stream new lines)
  grove logs --no-color   # Disable syntax highlighting`,
	RunE: runLogs,
}

var logsNoColor bool

func init() {
	logsCmd.Flags().IntP("lines", "n", 20, "Number of lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow logs (stream new lines)")
	logsCmd.Flags().BoolVar(&logsNoColor, "no-color", false, "Disable syntax highlighting")
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

// printLine prints a log line with optional highlighting
func printLine(line string) {
	if logsNoColor {
		fmt.Println(line)
	} else {
		fmt.Println(loghighlight.Highlight(line))
	}
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
		printLine(line)
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
		// Remove trailing newline since printLine adds one
		line = line[:len(line)-1]
		printLine(line)
	}
}
