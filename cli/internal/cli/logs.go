package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
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
		return tailFollow(server.LogFile, name)
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

// tailFollow follows the log file and prints new lines using file watching
func tailFollow(path string, serverName string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Print header so user knows what's happening
	fmt.Printf("\n  Streaming logs for \033[1m%s\033[0m\n", serverName)
	fmt.Printf("  Press \033[1mCtrl+C\033[0m to exit\n")
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Println()

	// Seek to end to only show new content
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// Fall back to polling if fsnotify fails
		return tailFollowPoll(file, offset)
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		// Fall back to polling
		return tailFollowPoll(file, offset)
	}

	reader := bufio.NewReader(file)

	// Print any lines that appeared since we opened the file
	readAndPrintLines(reader)

	// Watch for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) {
				readAndPrintLines(reader)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// readAndPrintLines reads and prints all available lines from the reader
func readAndPrintLines(reader *bufio.Reader) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No more data available right now
				// Print partial line if any
				if len(line) > 0 {
					printLine(line)
				}
				return
			}
			// Other error, just return
			return
		}
		// Remove trailing newline
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		printLine(line)
	}
}

// tailFollowPoll is a fallback that uses polling instead of file watching
func tailFollowPoll(file *os.File, offset int64) error {
	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait before checking again - don't spin!
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		// Remove trailing newline
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		printLine(line)
	}
}
