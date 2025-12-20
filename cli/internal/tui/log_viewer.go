package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// LogViewerKeyMap defines keybindings for the log viewer
type LogViewerKeyMap struct {
	Quit       key.Binding
	AutoScroll key.Binding
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
}

var logViewerKeys = LogViewerKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "quit"),
	),
	AutoScroll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "toggle auto-scroll"),
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
		key.WithKeys("pgup", "b"),
		key.WithHelp("pgup/b", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "f", " "),
		key.WithHelp("pgdown/f/space", "page down"),
	),
}

// LogViewerModel represents the log viewer
type LogViewerModel struct {
	viewport   viewport.Model
	serverName string
	logFile    string
	lines      []string
	lineCount  int
	autoScroll bool
	watcher    *fsnotify.Watcher
	mu         sync.RWMutex
	ready      bool
	err        error
}

// logLineMsg is sent when a new log line is detected
type logLineMsg struct {
	lines []string
}

// logErrorMsg is sent when an error occurs
type logErrorMsg struct {
	err error
}

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
	return tea.Batch(
		m.loadInitialLogs(),
		m.watchLogs(),
	)
}

// loadInitialLogs loads the existing log content
func (m *LogViewerModel) loadInitialLogs() tea.Cmd {
	return func() tea.Msg {
		file, err := os.Open(m.logFile)
		if err != nil {
			return logErrorMsg{err: err}
		}
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return logErrorMsg{err: err}
		}

		return logLineMsg{lines: lines}
	}
}

// watchLogs starts watching the log file for changes
func (m *LogViewerModel) watchLogs() tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return logErrorMsg{err: err}
		}

		err = watcher.Add(m.logFile)
		if err != nil {
			watcher.Close()
			return logErrorMsg{err: err}
		}

		// Start watching in a goroutine
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						// Read new lines
						if newLines := m.readNewLines(); len(newLines) > 0 {
							// Send message to update UI
							// Note: This is a simplified approach
							// In a real implementation, you'd use a proper channel
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					_ = err // Handle error if needed
				}
			}
		}()

		return nil
	}
}

// readNewLines reads new lines from the log file
func (m *LogViewerModel) readNewLines() []string {
	m.mu.RLock()
	currentLineCount := len(m.lines)
	m.mu.RUnlock()

	file, err := os.Open(m.logFile)
	if err != nil {
		return nil
	}
	defer file.Close()

	var newLines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		if lineNum >= currentLineCount {
			newLines = append(newLines, scanner.Text())
		}
		lineNum++
	}

	return newLines
}

// Update handles messages
func (m *LogViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = false
			m.ready = true
			m.updateViewport()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
			m.updateViewport()
		}

	case logLineMsg:
		m.mu.Lock()
		if len(msg.lines) > 0 {
			if len(m.lines) == 0 {
				// Initial load
				m.lines = msg.lines
			} else {
				// Append new lines
				m.lines = append(m.lines, msg.lines...)
			}
			m.lineCount = len(m.lines)
		}
		m.mu.Unlock()
		m.updateViewport()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case logErrorMsg:
		m.err = msg.err

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, logViewerKeys.Quit):
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, logViewerKeys.AutoScroll):
			m.autoScroll = !m.autoScroll
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateViewport updates the viewport content
func (m *LogViewerModel) updateViewport() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder
	for _, line := range m.lines {
		b.WriteString(m.formatLogLine(line))
		b.WriteString("\n")
	}

	m.viewport.SetContent(b.String())
}

// formatLogLine formats a log line with syntax highlighting
func (m *LogViewerModel) formatLogLine(line string) string {
	// Simple syntax highlighting
	if strings.Contains(strings.ToLower(line), "error") {
		return lipgloss.NewStyle().Foreground(errorColor).Render(line)
	}
	if strings.Contains(strings.ToLower(line), "warn") {
		return lipgloss.NewStyle().Foreground(warningColor).Render(line)
	}
	if strings.Contains(strings.ToLower(line), "info") {
		return lipgloss.NewStyle().Foreground(secondaryColor).Render(line)
	}
	return lipgloss.NewStyle().Foreground(mutedColor).Render(line)
}

// View renders the log viewer
func (m *LogViewerModel) View() string {
	if !m.ready {
		return "\n  Loading logs..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit", m.err)
	}

	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(fmt.Sprintf("Logs: %s", m.serverName))
	b.WriteString(header)
	b.WriteString("\n")

	// Line count and auto-scroll status
	autoScrollStatus := "off"
	if m.autoScroll {
		autoScrollStatus = "on"
	}
	status := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(fmt.Sprintf("Lines: %d  |  Auto-scroll: %s", m.lineCount, autoScrollStatus))
	b.WriteString(status)
	b.WriteString("\n\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	// Help
	help := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("  [a] toggle auto-scroll  [↑/↓] scroll  [pgup/pgdn] page  [q/esc] quit")
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
