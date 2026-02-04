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
	"github.com/charmbracelet/x/ansi"
	"github.com/fsnotify/fsnotify"
	"github.com/iheanyi/grove/internal/loghighlight"
	"github.com/iheanyi/grove/internal/registry"
)

// logEntry represents a single log line with metadata
type logEntry struct {
	serverName string
	line       string
}

// MultiLogViewerModel represents the multi-server log viewer
type MultiLogViewerModel struct {
	viewport   viewport.Model
	servers    []*registry.Server
	entries    []logEntry
	autoScroll bool
	ready      bool
	err        error
	width      int
	height     int
}

// multiLogLinesMsg is sent when log lines are loaded/updated
type multiLogLinesMsg struct {
	entries []logEntry
}

// multiLogFileChangedMsg is sent when any log file changes
type multiLogFileChangedMsg struct{}

// NewMultiLogViewer creates a new multi-server log viewer
func NewMultiLogViewer(servers []*registry.Server) *MultiLogViewerModel {
	return &MultiLogViewerModel{
		servers:    servers,
		entries:    []logEntry{},
		autoScroll: true,
	}
}

// Init initializes the multi-log viewer
func (m *MultiLogViewerModel) Init() tea.Cmd {
	return m.loadAllLogs()
}

// loadAllLogs loads logs from all servers
func (m *MultiLogViewerModel) loadAllLogs() tea.Cmd {
	return func() tea.Msg {
		var entries []logEntry

		for _, server := range m.servers {
			if server.LogFile == "" {
				continue
			}

			file, err := os.Open(server.LogFile)
			if err != nil {
				continue
			}

			// Use bufio.Reader instead of Scanner to handle long lines
			reader := bufio.NewReader(file)
			var lines []string
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						if len(line) > 0 {
							lines = append(lines, strings.TrimSuffix(line, "\n"))
						}
						break
					}
					break // Skip this file on other errors
				}
				lines = append(lines, strings.TrimSuffix(line, "\n"))
			}
			file.Close()

			// Take last 100 lines
			start := 0
			if len(lines) > 100 {
				start = len(lines) - 100
			}

			for _, line := range lines[start:] {
				entries = append(entries, logEntry{
					serverName: server.Name,
					line:       line,
				})
			}
		}

		return multiLogLinesMsg{entries: entries}
	}
}

// watchAllLogFiles watches all log files for changes
func (m *MultiLogViewerModel) watchAllLogFiles() tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			return multiLogFileChangedMsg{}
		}
		defer watcher.Close()

		// Watch all log files
		for _, server := range m.servers {
			if server.LogFile != "" {
				watcher.Add(server.LogFile) //nolint:errcheck
			}
		}

		// Wait for any write event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if event.Has(fsnotify.Write) {
					time.Sleep(50 * time.Millisecond)
					return multiLogFileChangedMsg{}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				time.Sleep(500 * time.Millisecond)
				return multiLogFileChangedMsg{}
			}
		}
	}
}

// Update handles messages
func (m *MultiLogViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-5)
			m.viewport.YPosition = 0
			m.ready = true
			m.updateViewport()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 5
			m.updateViewport()
		}
		return m, nil

	case multiLogLinesMsg:
		m.entries = msg.entries
		m.updateViewport()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		cmds = append(cmds, m.watchAllLogFiles())
		return m, tea.Batch(cmds...)

	case multiLogFileChangedMsg:
		cmds = append(cmds, m.loadAllLogs())
		return m, tea.Batch(cmds...)

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
func (m *MultiLogViewerModel) updateViewport() {
	var b strings.Builder

	// Color palette for different servers
	colors := []lipgloss.Color{
		lipgloss.Color("39"),  // Blue
		lipgloss.Color("208"), // Orange
		lipgloss.Color("135"), // Purple
		lipgloss.Color("42"),  // Cyan
		lipgloss.Color("205"), // Pink
		lipgloss.Color("220"), // Yellow
	}

	// Map server names to colors
	serverColors := make(map[string]lipgloss.Color)
	for i, server := range m.servers {
		serverColors[server.Name] = colors[i%len(colors)]
	}

	// Calculate max server name length for alignment
	maxNameLen := 0
	for _, server := range m.servers {
		if len(server.Name) > maxNameLen {
			maxNameLen = len(server.Name)
		}
	}
	// Cap at reasonable width
	if maxNameLen > 15 {
		maxNameLen = 15
	}

	for _, entry := range m.entries {
		// Server name prefix with color
		color := serverColors[entry.serverName]
		nameStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

		// Truncate and pad server name
		name := entry.serverName
		if len(name) > maxNameLen {
			name = ansi.Truncate(name, maxNameLen, "…")
		}
		name = fmt.Sprintf("%-*s", maxNameLen, name)

		prefix := nameStyle.Render(name) + " │ "

		// Format the log line
		line := m.formatLogLine(entry.line)

		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}

	m.viewport.SetContent(b.String())
}

// formatLogLine formats a log line with syntax highlighting
func (m *MultiLogViewerModel) formatLogLine(line string) string {
	// Use the loghighlight package for rich syntax highlighting
	return loghighlight.Highlight(line)
}

// View renders the multi-log viewer
func (m *MultiLogViewerModel) View() string {
	if !m.ready {
		return "\n  Loading logs..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to go back", m.err)
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)
	b.WriteString(headerStyle.Render("  All Server Logs"))

	// Status info
	autoScrollIndicator := lipgloss.NewStyle().Foreground(mutedColor).Render("off")
	if m.autoScroll {
		autoScrollIndicator = lipgloss.NewStyle().Foreground(secondaryColor).Render("on")
	}

	scrollPercent := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPercent = int(m.viewport.ScrollPercent() * 100)
	}

	statusParts := []string{
		fmt.Sprintf("%d servers", len(m.servers)),
		fmt.Sprintf("%d lines", len(m.entries)),
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

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(mutedColor)
	help := helpStyle.Render("  [a]auto-scroll  [↑↓/jk]scroll  [shift+↑↓/bf]page  [g/G]top/bottom  [q/esc]back")
	b.WriteString(help)

	return b.String()
}
