package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
)

// WorktreeItem represents a worktree in the list
type WorktreeItem struct {
	worktree *discovery.Worktree
	server   *registry.Server
}

// Title returns plain text with status icon prefix
func (i WorktreeItem) Title() string {
	statusIcon := "○"
	if i.server != nil {
		if i.server.IsRunning() {
			statusIcon = "●"
		} else if i.server.Status == registry.StatusCrashed {
			statusIcon = "✗"
		}
	}
	return statusIcon + " " + i.worktree.Name
}

// Description returns plain text
func (i WorktreeItem) Description() string {
	var parts []string

	// Add branch name
	if i.worktree.Branch != "" {
		parts = append(parts, "branch: "+i.worktree.Branch)
	}

	// Add path
	parts = append(parts, i.worktree.Path)

	// Add server info if exists
	if i.server != nil {
		if i.server.IsRunning() {
			serverInfo := fmt.Sprintf("%s :%d", i.server.URL, i.server.Port)
			parts = append(parts, serverInfo)

			// Add uptime
			uptime := i.server.UptimeString()
			if uptime != "-" {
				parts = append(parts, "↑ "+uptime)
			}

			// Add last health check
			if !i.server.LastHealthCheck.IsZero() {
				lastCheck := FormatLastHealthCheck(i.server.LastHealthCheck)
				parts = append(parts, "checked "+lastCheck)
			}
		} else {
			parts = append(parts, fmt.Sprintf("port: %d (stopped)", i.server.Port))
		}
	} else {
		parts = append(parts, "no server")
	}

	return strings.Join(parts, "  |  ")
}

func (i WorktreeItem) FilterValue() string {
	return i.worktree.Name
}

// StatusIcon returns the status icon for display
func (i WorktreeItem) StatusIcon() string {
	if i.server == nil {
		return "○"
	}
	if i.server.IsRunning() {
		return "●"
	} else if i.server.Status == registry.StatusCrashed {
		return "✗"
	}
	return "○"
}

// StatusStyle returns the lipgloss style for the status
func (i WorktreeItem) StatusStyle() lipgloss.Style {
	if i.server == nil {
		return statusStoppedStyle
	}
	if i.server.IsRunning() {
		return statusRunningStyle
	} else if i.server.Status == registry.StatusCrashed {
		return statusCrashedStyle
	}
	return statusStoppedStyle
}

// HealthIndicator returns the health indicator string
func (i WorktreeItem) HealthIndicator() string {
	if i.server == nil || !i.server.IsRunning() {
		return ""
	}
	switch i.server.Health {
	case registry.HealthHealthy:
		return " ✓"
	case registry.HealthUnhealthy:
		return " ✗"
	case registry.HealthUnknown:
		return " ?"
	}
	return ""
}

// HealthStyle returns the style for the health indicator
func (i WorktreeItem) HealthStyle() lipgloss.Style {
	if i.server == nil {
		return unknownStyle
	}
	switch i.server.Health {
	case registry.HealthHealthy:
		return healthyStyle
	case registry.HealthUnhealthy:
		return unhealthyStyle
	default:
		return unknownStyle
	}
}

// WorktreeListModel represents the worktree list view
type WorktreeListModel struct {
	list         list.Model
	reg          *registry.Registry
	worktrees    []*discovery.Worktree
	width        int
	height       int
	showHelp     bool
	notification *Notification
}

// NewWorktreeList creates a new worktree list model
func NewWorktreeList(reg *registry.Registry, worktrees []*discovery.Worktree) *WorktreeListModel {
	items := makeWorktreeItems(reg, worktrees)

	// Create default delegate - Title() includes status icon as plain text
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Accent)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(styles.Muted)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Git Worktrees"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	return &WorktreeListModel{
		list:      l,
		reg:       reg,
		worktrees: worktrees,
	}
}

func makeWorktreeItems(reg *registry.Registry, worktrees []*discovery.Worktree) []list.Item {
	items := make([]list.Item, len(worktrees))
	for i, wt := range worktrees {
		// Find associated server if exists
		var server *registry.Server
		if s, ok := reg.Get(wt.Name); ok {
			server = s
		}

		items[i] = WorktreeItem{
			worktree: wt,
			server:   server,
		}
	}
	return items
}

// Init initializes the worktree list
func (m WorktreeListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m WorktreeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-8)
		return m, nil

	case tea.KeyMsg:
		// When filtering (or filter applied), let the list handle all keys
		if m.list.FilterState() != list.Unfiltered {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Only handle our custom keys when NOT filtering
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the worktree list
func (m WorktreeListModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Main list
	b.WriteString(m.list.View())
	b.WriteString("\n")

	// Notification (if visible)
	if m.notification != nil && m.notification.IsVisible() {
		b.WriteString("\n")
		b.WriteString(m.notification.View())
	}

	// Help
	if m.showHelp {
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp())
	} else {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  [/] search  [?] help  [q] quit"))
	}

	return b.String()
}

func (m WorktreeListModel) renderHelp() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Keyboard Shortcuts\n"))
	b.WriteString("  ─────────────────────────────────────\n")
	b.WriteString("  /             Search/filter worktrees\n")
	b.WriteString("  ↑/↓, j/k      Navigate list\n")
	b.WriteString("  ?             Toggle this help\n")
	b.WriteString("  q, esc        Quit\n")
	return b.String()
}
