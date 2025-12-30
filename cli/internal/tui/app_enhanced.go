package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/pkg/browser"
)

// EnhancedKeyMap defines the enhanced keybindings
type EnhancedKeyMap struct {
	Quit          key.Binding
	Help          key.Binding
	Start         key.Binding
	Stop          key.Binding
	Restart       key.Binding
	Open          key.Binding
	CopyURL       key.Binding
	Logs          key.Binding
	Refresh       key.Binding
	Up            key.Binding
	Down          key.Binding
	StartProxy    key.Binding
	Search        key.Binding
	ToggleActions key.Binding
}

var enhancedKeys = EnhancedKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	Stop: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "stop"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart"),
	),
	Open: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "browser"),
	),
	CopyURL: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy URL"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("F5"),
		key.WithHelp("F5", "refresh"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	StartProxy: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "proxy"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	ToggleActions: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "toggle actions"),
	),
}

// EnhancedServerItem represents a server in the list with health info
type EnhancedServerItem struct {
	server *registry.Server
}

func (i EnhancedServerItem) Title() string {
	// Status icon
	statusIcon := "○"
	statusStyle := statusStoppedStyle
	if i.server.IsRunning() {
		statusIcon = "●"
		statusStyle = statusRunningStyle
	} else if i.server.Status == registry.StatusCrashed {
		statusIcon = "✗"
		statusStyle = statusCrashedStyle
	}

	// Health indicator
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

	return statusStyle.Render(statusIcon) + " " + i.server.Name + healthIndicator
}

func (i EnhancedServerItem) Description() string {
	parts := []string{
		fmt.Sprintf("%s  :%d", i.server.URL, i.server.Port),
	}

	// Add uptime if running
	if i.server.IsRunning() {
		uptime := i.server.UptimeString()
		if uptime != "-" {
			parts = append(parts, "↑ "+uptime)
		}
	}

	// Add last health check time if available
	if i.server.IsRunning() && !i.server.LastHealthCheck.IsZero() {
		lastCheck := FormatLastHealthCheck(i.server.LastHealthCheck)
		parts = append(parts, "checked "+lastCheck)
	}

	return strings.Join(parts, "  |  ")
}

func (i EnhancedServerItem) FilterValue() string {
	return i.server.Name
}

// EnhancedModel is the enhanced TUI model
type EnhancedModel struct {
	list         list.Model
	reg          *registry.Registry
	cfg          *config.Config
	width        int
	height       int
	showHelp     bool
	notification *Notification
	spinner      *Spinner
	actionPanel  *ActionPanel
	searchBar    *SearchBar
	searchMode   bool
	serverHealth map[string]registry.HealthStatus
	starting     map[string]bool // Track servers currently starting
}

// NewEnhanced creates a new enhanced TUI model
func NewEnhanced(cfg *config.Config) (*EnhancedModel, error) {
	reg, err := registry.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Create list items from servers
	items := makeEnhancedItems(reg)

	// Create list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedStyle
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "grove - Worktree Server Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.SetShowHelp(false)

	return &EnhancedModel{
		list:         l,
		reg:          reg,
		cfg:          cfg,
		spinner:      NewSpinner(),
		actionPanel:  NewActionPanel(),
		searchBar:    NewSearchBar(50),
		serverHealth: make(map[string]registry.HealthStatus),
		starting:     make(map[string]bool),
	}, nil
}

func makeEnhancedItems(reg *registry.Registry) []list.Item {
	servers := reg.List()
	items := make([]list.Item, len(servers))
	for i, s := range servers {
		items[i] = EnhancedServerItem{server: s}
	}
	return items
}

// Init initializes the enhanced model
func (m EnhancedModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		SpinnerTickCmd(),
		HealthCheckTicker(10*time.Second),
	)
}

// Update handles messages
func (m EnhancedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-12) // More space for action panel
		return m, nil

	case tickMsg:
		// Refresh registry
		if reg, err := registry.Load(); err == nil {
			m.reg = reg
			m.reg.Cleanup() //nolint:errcheck // Best effort cleanup during refresh
			m.list.SetItems(makeEnhancedItems(m.reg))
		}
		return m, tickCmd()

	case SpinnerTickMsg:
		m.spinner.Tick()
		return m, SpinnerTickCmd()

	case healthCheckTickMsg:
		// Trigger health checks for running servers
		for _, server := range m.reg.ListRunning() {
			cmds = append(cmds, HealthCheckCmd(server))
		}
		return m, tea.Batch(append(cmds, HealthCheckTicker(10*time.Second))...)

	case HealthCheckMsg:
		// Update server health
		if server, ok := m.reg.Get(msg.ServerName); ok {
			server.Health = msg.Health
			server.LastHealthCheck = msg.CheckTime
			m.reg.Set(server) //nolint:errcheck // Best effort health update
			m.serverHealth[msg.ServerName] = msg.Health
			m.list.SetItems(makeEnhancedItems(m.reg))
		}
		return m, nil

	case NotificationMsg:
		m.notification = NewNotification(msg.Message, msg.Type)
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

		switch {
		case key.Matches(msg, enhancedKeys.Quit):
			return m, tea.Quit

		case key.Matches(msg, enhancedKeys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, enhancedKeys.Start):
			return m, m.startServer()

		case key.Matches(msg, enhancedKeys.Stop):
			return m, m.stopServer()

		case key.Matches(msg, enhancedKeys.Restart):
			return m, m.restartServer()

		case key.Matches(msg, enhancedKeys.Open):
			return m, m.openServer()

		case key.Matches(msg, enhancedKeys.CopyURL):
			return m, m.copyURL()

		case key.Matches(msg, enhancedKeys.Logs):
			return m, m.viewLogs()

		case key.Matches(msg, enhancedKeys.Refresh):
			if reg, err := registry.Load(); err == nil {
				m.reg = reg
				m.reg.Cleanup() //nolint:errcheck // Best effort cleanup during refresh
				m.list.SetItems(makeEnhancedItems(m.reg))
			}
			return m, nil

		case key.Matches(msg, enhancedKeys.StartProxy):
			return m, m.toggleProxy()

		case key.Matches(msg, enhancedKeys.Search):
			m.searchMode = true
			m.searchBar.Activate()
			return m, nil

		case key.Matches(msg, enhancedKeys.ToggleActions):
			m.actionPanel.Visible = !m.actionPanel.Visible
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// handleSearchInput handles keyboard input in search mode
func (m EnhancedModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// View renders the enhanced TUI
func (m EnhancedModel) View() string {
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

	// Show spinner if any server is starting
	if len(m.starting) > 0 {
		b.WriteString("\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		var startingServers []string
		for name := range m.starting {
			startingServers = append(startingServers, name)
		}
		b.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render(
			fmt.Sprintf("Starting: %s", strings.Join(startingServers, ", "))))
	}

	// Proxy status
	proxy := m.reg.GetProxy()
	if proxy.IsRunning() && isProcessRunning(proxy.PID) {
		b.WriteString(statusRunningStyle.Render(fmt.Sprintf("  Proxy: running on :%d/:%d", proxy.HTTPPort, proxy.HTTPSPort)))
	} else {
		b.WriteString(statusStoppedStyle.Render("  Proxy: not running (p to start)"))
	}
	b.WriteString("\n")

	// Notification (if visible)
	if m.notification != nil && m.notification.IsVisible() {
		b.WriteString("\n")
		b.WriteString(m.notification.View())
	}

	// Action panel
	if m.actionPanel.Visible {
		b.WriteString("\n")
		// Update action availability based on selected server
		if item := m.list.SelectedItem(); item != nil {
			eItem := item.(EnhancedServerItem)
			m.actionPanel.UpdateActionAvailability(eItem.server.IsRunning())
		}
		b.WriteString(m.actionPanel.View())
	}

	// Help
	if m.showHelp {
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp())
	} else {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Quick: [s]start [x]stop [r]restart [b]browser [c]copy [l]logs [/]search [?]help [q]quit"))
	}

	return b.String()
}

func (m EnhancedModel) renderHelp() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Keyboard Shortcuts\n"))
	b.WriteString("  ─────────────────────────────────────\n")
	b.WriteString("  s             Start selected server\n")
	b.WriteString("  x             Stop selected server\n")
	b.WriteString("  r             Restart selected server\n")
	b.WriteString("  b             Open server in browser\n")
	b.WriteString("  c             Copy URL to clipboard\n")
	b.WriteString("  l             View server logs\n")
	b.WriteString("  p             Start/stop proxy\n")
	b.WriteString("  F5            Refresh server list\n")
	b.WriteString("  /             Search/filter servers\n")
	b.WriteString("  a             Toggle action panel\n")
	b.WriteString("  ?             Toggle this help\n")
	b.WriteString("  q, ctrl+c     Quit\n")
	return b.String()
}

func (m *EnhancedModel) startServer() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(EnhancedServerItem)
	server := item.server

	if server.IsRunning() {
		return func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("%s is already running", server.Name),
				Type:    NotificationWarning,
			}
		}
	}

	// Mark as starting
	m.starting[server.Name] = true

	return func() tea.Msg {
		// In a real implementation, you would start the server here
		// For now, we just show a message
		delete(m.starting, server.Name)
		return NotificationMsg{
			Message: fmt.Sprintf("Use 'grove start %s' in terminal to start server", server.Name),
			Type:    NotificationInfo,
		}
	}
}

func (m *EnhancedModel) stopServer() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(EnhancedServerItem)
	server := item.server

	if !server.IsRunning() {
		return func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("%s is not running", server.Name),
				Type:    NotificationWarning,
			}
		}
	}

	return func() tea.Msg {
		// Stop server
		if process, err := os.FindProcess(server.PID); err == nil {
			process.Signal(syscall.SIGTERM) //nolint:errcheck // Best effort signal
		}
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		if err := m.reg.Set(server); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to update registry: %v", err),
				Type:    NotificationError,
			}
		}
		return NotificationMsg{
			Message: fmt.Sprintf("Stopped %s", server.Name),
			Type:    NotificationSuccess,
		}
	}
}

func (m *EnhancedModel) restartServer() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(EnhancedServerItem)
	server := item.server

	if !server.IsRunning() {
		return func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("%s is not running", server.Name),
				Type:    NotificationWarning,
			}
		}
	}

	return func() tea.Msg {
		// Stop server first
		if process, err := os.FindProcess(server.PID); err == nil {
			process.Signal(syscall.SIGTERM) //nolint:errcheck // Best effort signal
		}
		return NotificationMsg{
			Message: fmt.Sprintf("Restart %s with 'grove start %s'", server.Name, server.Name),
			Type:    NotificationInfo,
		}
	}
}

func (m *EnhancedModel) openServer() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(EnhancedServerItem)
	server := item.server

	if !server.IsRunning() {
		return func() tea.Msg {
			return NotificationMsg{
				Message: "Server not running",
				Type:    NotificationWarning,
			}
		}
	}

	return func() tea.Msg {
		if err := browser.Open(server.URL); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to open browser: %v", err),
				Type:    NotificationError,
			}
		}
		return NotificationMsg{
			Message: fmt.Sprintf("Opened %s", server.URL),
			Type:    NotificationSuccess,
		}
	}
}

func (m *EnhancedModel) copyURL() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(EnhancedServerItem)
	server := item.server

	return func() tea.Msg {
		if err := clipboard.WriteAll(server.URL); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to copy URL: %v", err),
				Type:    NotificationError,
			}
		}
		return NotificationMsg{
			Message: fmt.Sprintf("Copied %s to clipboard", server.URL),
			Type:    NotificationSuccess,
		}
	}
}

func (m *EnhancedModel) viewLogs() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(EnhancedServerItem)
	server := item.server

	if server.LogFile == "" {
		return func() tea.Msg {
			return NotificationMsg{
				Message: "No log file",
				Type:    NotificationWarning,
			}
		}
	}

	// Use grove logs command which has syntax highlighting
	grovePath, _ := exec.LookPath("grove")
	if grovePath == "" {
		// Fall back to less if grove not found
		return tea.ExecProcess(exec.Command("less", "+F", server.LogFile), func(err error) tea.Msg {
			return nil
		})
	}

	return tea.ExecProcess(exec.Command(grovePath, "logs", "-f", server.Name), func(err error) tea.Msg {
		return nil
	})
}

func (m *EnhancedModel) toggleProxy() tea.Cmd {
	proxy := m.reg.GetProxy()

	return func() tea.Msg {
		if proxy.IsRunning() && isProcessRunning(proxy.PID) {
			// Stop proxy
			if process, err := os.FindProcess(proxy.PID); err == nil {
				process.Signal(syscall.SIGTERM) //nolint:errcheck // Best effort signal
			}
			proxy.PID = 0
			if err := m.reg.UpdateProxy(proxy); err != nil {
				return NotificationMsg{
					Message: fmt.Sprintf("Failed to update registry: %v", err),
					Type:    NotificationError,
				}
			}
			return NotificationMsg{
				Message: "Proxy stopped",
				Type:    NotificationSuccess,
			}
		}
		return NotificationMsg{
			Message: "Use 'grove proxy start' in terminal to start proxy",
			Type:    NotificationInfo,
		}
	}
}

// RunEnhanced starts the enhanced TUI
func RunEnhanced(cfg *config.Config) error {
	m, err := NewEnhanced(cfg)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
