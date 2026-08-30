package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/iheanyi/grove/internal/loghighlight"
)

// LogViewerKeyMap defines keybindings for the log viewer
type LogViewerKeyMap struct {
	Quit       key.Binding
	AutoScroll key.Binding
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Top        key.Binding
	Bottom     key.Binding
}

var logViewerKeys = LogViewerKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "back"),
	),
	AutoScroll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "auto-scroll"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "b", "shift+up"),
		key.WithHelp("pgup/b/shift+↑", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "f", " ", "shift+down"),
		key.WithHelp("pgdn/f/shift+↓", "page down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g/home", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G/end", "bottom"),
	),
}

// maxLogLines is the maximum number of lines to keep in memory
const maxLogLines = 2000

// LogViewerModel represents the log viewer
type LogViewerModel struct {
	viewport     viewport.Model
	serverName   string
	logFile      string
	lines        []string
	lineCount    int
	autoScroll   bool
	ready        bool
	err          error
	lastFileSize int64 // Track file size for incremental reads
}

// logLinesMsg is sent when log lines are loaded/updated
type logLinesMsg struct {
	lines    []string
	initial  bool  // true if this is the initial load
	fileSize int64 // current file size for tracking
}

// logErrorMsg is sent when an error occurs
type logErrorMsg struct {
	err error
}

// logFileChangedMsg is sent when the log file changes
type logFileChangedMsg struct{}

// NewLogViewer creates a new log viewer model
func NewLogViewer(serverName, logFile string) *LogViewerModel {
	return &LogViewerModel{
		serverName: serverName,
		logFile:    logFile,
		lines:      []string{},
		autoScroll: true,
	}
}

// Init initializes the log viewer
func (m *LogViewerModel) Init() tea.Cmd {
	return m.loadLogs(true)
}

// loadLogs loads log content from the file
func (m *LogViewerModel) loadLogs(initial bool) tea.Cmd {
	lastSize := m.lastFileSize
	return func() tea.Msg {
		file, err := os.Open(m.logFile)
		if err != nil {
			return logErrorMsg{err: err}
		}
		defer file.Close()

		// Get current file size
		stat, err := file.Stat()
		if err != nil {
			return logErrorMsg{err: err}
		}
		currentSize := stat.Size()

		var lines []string

		if initial || lastSize == 0 {
			// Initial load: read last maxLogLines using tail-like approach
			lines = tailFile(file, maxLogLines)
		} else if currentSize > lastSize {
			// Incremental: seek to last position and read only new content
			_, err = file.Seek(lastSize, io.SeekStart)
			if err != nil {
				return logErrorMsg{err: err}
			}
			lines = readLines(file)
		} else if currentSize < lastSize {
			// File was truncated/rotated, re-read from start
			lines = tailFile(file, maxLogLines)
		}
		// If currentSize == lastSize, no new content

		return logLinesMsg{lines: lines, initial: initial, fileSize: currentSize}
	}
}

// tailFile reads the last n lines from a file efficiently
func tailFile(file *os.File, n int) []string {
	// Get file size
	stat, err := file.Stat()
	if err != nil || stat.Size() == 0 {
		return nil
	}

	// For small files, just read everything
	if stat.Size() < 64*1024 { // Less than 64KB
		_, _ = file.Seek(0, io.SeekStart)
		lines := readLines(file)
		if len(lines) > n {
			return lines[len(lines)-n:]
		}
		return lines
	}

	// For larger files, read from end in chunks
	const chunkSize = 32 * 1024 // 32KB chunks
	var lines []string
	fileSize := stat.Size()
	offset := fileSize

	for offset > 0 && len(lines) < n {
		// Calculate chunk start position
		chunkStart := offset - chunkSize
		if chunkStart < 0 {
			chunkStart = 0
		}

		// Read chunk
		_, _ = file.Seek(chunkStart, io.SeekStart)
		chunk := make([]byte, offset-chunkStart)
		bytesRead, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			break
		}
		chunk = chunk[:bytesRead]

		// Split into lines and prepend to result
		chunkLines := strings.Split(string(chunk), "\n")

		// If this isn't the last chunk, the first line might be partial
		if chunkStart > 0 && len(chunkLines) > 0 {
			// Prepend remaining lines (skip potentially partial first line)
			lines = append(chunkLines[1:], lines...)
		} else {
			lines = append(chunkLines, lines...)
		}

		offset = chunkStart
	}

	// Remove empty strings and trim to n lines
	var result []string
	for _, line := range lines {
		if line != "" || len(result) > 0 { // Keep empty lines after first non-empty
			result = append(result, line)
		}
	}

	if len(result) > n {
		return result[len(result)-n:]
	}
	return result
}

// readLines reads all lines from the current file position
func readLines(file *os.File) []string {
	var lines []string
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					lines = append(lines, strings.TrimSuffix(line, "\n"))
				}
				break
			}
			break
		}
		lines = append(lines, strings.TrimSuffix(line, "\n"))
	}
	return lines
}

// watchLogFile watches for changes and returns a command when the file changes
func (m *LogViewerModel) watchLogFile() tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			// Fall back to polling if fsnotify fails
			time.Sleep(500 * time.Millisecond)
			return logFileChangedMsg{}
		}
		defer watcher.Close()

		if err := watcher.Add(m.logFile); err != nil {
			// Fall back to polling
			time.Sleep(500 * time.Millisecond)
			return logFileChangedMsg{}
		}

		// Wait for a write event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if event.Has(fsnotify.Write) {
					// Small debounce for rapid writes
					time.Sleep(50 * time.Millisecond)
					return logFileChangedMsg{}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				// On error, fall back to polling
				time.Sleep(500 * time.Millisecond)
				return logFileChangedMsg{}
			}
		}
	}
}

// Update handles messages
func (m *LogViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.YPosition = 0
			m.ready = true
			m.updateViewport()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
			m.updateViewport()
		}
		return m, nil

	case logLinesMsg:
		// Update file size tracking
		m.lastFileSize = msg.fileSize

		if msg.initial {
			// Initial load: replace all lines
			m.lines = msg.lines
		} else if len(msg.lines) > 0 {
			// Incremental: append new lines
			m.lines = append(m.lines, msg.lines...)

			// Trim to maxLogLines to prevent unbounded memory growth
			if len(m.lines) > maxLogLines {
				// Remove oldest lines
				m.lines = m.lines[len(m.lines)-maxLogLines:]
			}
		}

		m.lineCount = len(m.lines)
		m.updateViewport()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		// Start watching for changes after initial load
		cmds = append(cmds, m.watchLogFile())
		return m, tea.Batch(cmds...)

	case logFileChangedMsg:
		// File changed, reload logs and continue watching
		cmds = append(cmds, m.loadLogs(false))
		return m, tea.Batch(cmds...)

	case logErrorMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, logViewerKeys.Quit):
			return m, tea.Quit

		case key.Matches(msg, logViewerKeys.AutoScroll):
			m.autoScroll = !m.autoScroll
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil

		case key.Matches(msg, logViewerKeys.Top):
			m.autoScroll = false
			m.viewport.GotoTop()
			return m, nil

		case key.Matches(msg, logViewerKeys.Bottom):
			m.viewport.GotoBottom()
			return m, nil

		case key.Matches(msg, logViewerKeys.PageUp):
			m.autoScroll = false
			m.viewport.PageUp()
			return m, nil

		case key.Matches(msg, logViewerKeys.PageDown):
			m.viewport.PageDown()
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// updateViewport updates the viewport content
func (m *LogViewerModel) updateViewport() {
	var b strings.Builder
	for _, line := range m.lines {
		b.WriteString(m.formatLogLine(line))
		b.WriteString("\n")
	}

	m.viewport.SetContent(b.String())
}

// formatLogLine formats a log line with syntax highlighting
func (m *LogViewerModel) formatLogLine(line string) string {
	// Use the loghighlight package for rich syntax highlighting
	return loghighlight.Highlight(line)
}

// View renders the log viewer
func (m *LogViewerModel) View() string {
	if !m.ready {
		return "\n  Loading logs..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to go back", m.err)
	}

	var b strings.Builder

	// Header with server name
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)
	b.WriteString(headerStyle.Render(fmt.Sprintf("  Logs: %s", m.serverName)))

	// Status info on same line (right side would be better but keeping it simple)
	autoScrollIndicator := lipgloss.NewStyle().Foreground(mutedColor).Render("off")
	if m.autoScroll {
		autoScrollIndicator = lipgloss.NewStyle().Foreground(secondaryColor).Render("on")
	}

	// Calculate scroll percentage
	scrollPercent := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPercent = int(m.viewport.ScrollPercent() * 100)
	}

	statusParts := []string{
		fmt.Sprintf("%d lines", m.lineCount),
		fmt.Sprintf("%d%%", scrollPercent),
		fmt.Sprintf("auto-scroll: %s", autoScrollIndicator),
	}
	status := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("  " + strings.Join(statusParts, "  │  "))
	b.WriteString(status)
	b.WriteString("\n")

	// Separator
	separator := lipgloss.NewStyle().Foreground(mutedColor).Render(strings.Repeat("─", m.viewport.Width))
	b.WriteString(separator)
	b.WriteString("\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Separator
	b.WriteString(separator)
	b.WriteString("\n")

	// Help - compact format
	helpStyle := lipgloss.NewStyle().Foreground(mutedColor)
	help := helpStyle.Render("  [a]auto-scroll  [↑↓/jk]scroll  [pgup/b]page up  [pgdn/f/space]page down  [g/G]top/bottom  [q/esc]back")
	b.WriteString(help)

	return b.String()
}

// RunLogViewer starts the log viewer
func RunLogViewer(serverName, logFile string) error {
	m := NewLogViewer(serverName, logFile)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
