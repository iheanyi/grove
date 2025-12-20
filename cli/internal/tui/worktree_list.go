package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/wt/internal/discovery"
	"github.com/iheanyi/wt/internal/registry"
)

// WorktreeItem represents a worktree in the list
type WorktreeItem struct {
	worktree *discovery.Worktree
	server   *registry.Server
}

func (i WorktreeItem) Title() string {
	name := i.worktree.Name

	// Add server status if exists
	if i.server != nil {
		statusIcon := "○"
		statusStyle := statusStoppedStyle

		if i.server.IsRunning() {
			statusIcon = "●"
			statusStyle = statusRunningStyle
		} else if i.server.Status == registry.StatusCrashed {
			statusIcon = "✗"
			statusStyle = statusCrashedStyle
		}

		// Add health indicator
		healthIndicator := ""
		if i.server.IsRunning() {
			switch i.server.Health {
			case registry.HealthHealthy:
				healthIndicator = " " + healthyStyle.Render("✓")
			case registry.HealthUnhealthy:
				healthIndicator = " " + unhealthyStyle.Render("✗")
			case registry.HealthUnknown:
				healthIndicator = " " + unknownStyle.Render("?")
			}
		}

		return statusStyle.Render(statusIcon) + " " + name + healthIndicator
	}

	return statusStoppedStyle.Render("○") + " " + name
}

func (i WorktreeItem) Description() string {
	var parts []string

	// Add branch name
	if i.worktree.Branch != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(mutedColor).Render("branch: "+i.worktree.Branch))
	}

	// Add path
	parts = append(parts, lipgloss.NewStyle().Foreground(mutedColor).Render(i.worktree.Path))

	// Add server info if exists
	if i.server != nil {
		if i.server.IsRunning() {
			serverInfo := fmt.Sprintf("%s :%d", i.server.URL, i.server.Port)
			parts = append(parts, lipgloss.NewStyle().Foreground(secondaryColor).Render(serverInfo))

			// Add uptime
			uptime := i.server.UptimeString()
			if uptime != "-" {
				parts = append(parts, lipgloss.NewStyle().Foreground(mutedColor).Render("↑ "+uptime))
			}

			// Add last health check
			if !i.server.LastHealthCheck.IsZero() {
				lastCheck := FormatLastHealthCheck(i.server.LastHealthCheck)
				parts = append(parts, lipgloss.NewStyle().Foreground(mutedColor).Render("checked "+lastCheck))
			}
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(mutedColor).Render(fmt.Sprintf("port: %d (stopped)", i.server.Port)))
		}
	} else {
		parts = append(parts, lipgloss.NewStyle().Foreground(mutedColor).Render("no server"))
	}

	return strings.Join(parts, "  |  ")
}

func (i WorktreeItem) FilterValue() string {
	return i.worktree.Name
}

// WorktreeListModel represents the worktree list view
type WorktreeListModel struct {
	list         list.Model
	reg          *registry.Registry
	worktrees    []*discovery.Worktree
	width        int
	height       int
	showHelp     bool
	searchMode   bool
	searchBar    *SearchBar
	notification *Notification
}

// NewWorktreeList creates a new worktree list model
func NewWorktreeList(reg *registry.Registry, worktrees []*discovery.Worktree) *WorktreeListModel {
	items := makeWorktreeItems(reg, worktrees)

	// Create list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedStyle
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Git Worktrees"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.SetShowHelp(false)

	return &WorktreeListModel{
		list:      l,
		reg:       reg,
		worktrees: worktrees,
		searchBar: NewSearchBar(50),
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
		// Handle search mode
		if m.searchMode {
			return m.handleSearchInput(msg)
		}

		// Don't process keys if filtering
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "/":
			m.searchMode = true
			m.searchBar.Activate()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// handleSearchInput handles keyboard input in search mode
func (m WorktreeListModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchBar.Deactivate()
		m.list.ResetFilter()
		return m, nil

	case "enter":
		m.searchMode = false
		m.searchBar.Active = false
		// The list handles filtering internally when user types
		return m, nil

	case "backspace":
		m.searchBar.DeleteChar()
		return m, nil

	case "left":
		m.searchBar.MoveCursorLeft()
		return m, nil

	case "right":
		m.searchBar.MoveCursorRight()
		return m, nil

	default:
		// Insert character
		if len(msg.String()) == 1 {
			m.searchBar.InsertChar(rune(msg.String()[0]))
		}
		return m, nil
	}
}

// View renders the worktree list
func (m WorktreeListModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Search bar (if active)
	if m.searchMode {
		b.WriteString("\n")
		b.WriteString(m.searchBar.View())
		b.WriteString("\n\n")
	}

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
