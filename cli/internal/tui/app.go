package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
	"github.com/iheanyi/grove/pkg/browser"
)

// KeyMap defines the keybindings for the TUI
type KeyMap struct {
	Quit       key.Binding
	Help       key.Binding
	Toggle     key.Binding
	Open       key.Binding
	Logs       key.Binding
	Refresh    key.Binding
	Up         key.Binding
	Down       key.Binding
	StartProxy key.Binding
}

var keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "start/stop"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
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
}

// ServerItem represents a server in the list
type ServerItem struct {
	server *registry.Server
}

// Title returns plain text with status icon prefix
func (i ServerItem) Title() string {
	statusIcon := "○"
	if i.server.IsRunning() {
		statusIcon = "●"
	} else if i.server.Status == registry.StatusCrashed {
		statusIcon = "✗"
	}
	return statusIcon + " " + i.server.Name
}

// Description returns plain text
func (i ServerItem) Description() string {
	return fmt.Sprintf("%s  :%d", i.server.URL, i.server.Port)
}

func (i ServerItem) FilterValue() string {
	return i.server.Name
}

// StatusIcon returns the status icon for display
func (i ServerItem) StatusIcon() string {
	if i.server.IsRunning() {
		return "●"
	} else if i.server.Status == registry.StatusCrashed {
		return "✗"
	}
	return "○"
}

// StatusStyle returns the lipgloss style for the status
func (i ServerItem) StatusStyle() lipgloss.Style {
	if i.server.IsRunning() {
		return statusRunningStyle
	} else if i.server.Status == registry.StatusCrashed {
		return statusCrashedStyle
	}
	return statusStoppedStyle
}

// Model is the main TUI model
type Model struct {
	list       list.Model
	reg        *registry.Registry
	cfg        *config.Config
	width      int
	height     int
	showHelp   bool
	statusMsg  string
	statusTime time.Time
}

// statusMsg is used to display temporary status messages
type statusMsgCmd string

// New creates a new TUI model
func New(cfg *config.Config) (*Model, error) {
	reg, err := registry.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Create list items from servers
	items := makeItems(reg)

	// Create default delegate - Title() includes status icon as plain text
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Accent)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(styles.Muted)

	l := list.New(items, delegate, 0, 0)
	l.Title = "grove - Worktree Server Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	return &Model{
		list: l,
		reg:  reg,
		cfg:  cfg,
	}, nil
}

func makeItems(reg *registry.Registry) []list.Item {
	servers := reg.List()
	items := make([]list.Item, len(servers))
	for i, s := range servers {
		items[i] = ServerItem{server: s}
	}
	return items
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		WatchRegistry(), // Watch for registry file changes instead of polling
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-8)
		return m, nil

	case RegistryChangedMsg:
		// Registry file changed - refresh if not filtering
		if reg, err := registry.Load(); err == nil {
			m.reg = reg
			// Cleanup stale entries (non-critical, ignore errors)
			m.reg.Cleanup() //nolint:errcheck // Best effort cleanup during refresh
			if m.list.FilterState() == list.Unfiltered {
				m.list.SetItems(makeItems(m.reg))
			}
		}
		// Continue watching for more changes
		return m, WatchRegistry()

	case statusMsgCmd:
		m.statusMsg = string(msg)
		m.statusTime = time.Now()
		return m, nil

	case tea.KeyMsg:
		// When actively filtering (typing in filter input), let the list handle most keys
		// But when filter is just "applied" (showing results), allow action keys
		if m.list.FilterState() == list.Filtering {
			// User is typing in the filter - let list handle all keys
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Handle our custom keys (works in both Unfiltered and FilterApplied states)
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, keys.Toggle):
			return m, m.toggleServer()

		case key.Matches(msg, keys.Open):
			return m, m.openServer()

		case key.Matches(msg, keys.Logs):
			return m, m.viewLogs()

		case key.Matches(msg, keys.Refresh):
			if reg, err := registry.Load(); err == nil {
				m.reg = reg
				m.reg.Cleanup() //nolint:errcheck // Best effort cleanup during refresh
				// Only update items if not filtering
				if m.list.FilterState() == list.Unfiltered {
					m.list.SetItems(makeItems(m.reg))
				}
			}
			return m, nil

		case key.Matches(msg, keys.StartProxy):
			return m, m.toggleProxy()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Main list
	b.WriteString(m.list.View())
	b.WriteString("\n")

	// Proxy status
	proxy := m.reg.GetProxy()
	if proxy.IsRunning() && isProcessRunning(proxy.PID) {
		b.WriteString(statusRunningStyle.Render(fmt.Sprintf("  Proxy: running on :%d/:%d", proxy.HTTPPort, proxy.HTTPSPort)))
	} else {
		b.WriteString(statusStoppedStyle.Render("  Proxy: not running (p to start)"))
	}
	b.WriteString("\n")

	// Status message (if recent)
	if m.statusMsg != "" && time.Since(m.statusTime) < 3*time.Second {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(secondaryColor).Render("  " + m.statusMsg))
	}

	// Help
	if m.showHelp {
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp())
	} else {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  [enter] start/stop  [o] open  [l] logs  [/] search  [?] help  [q] quit"))
	}

	return b.String()
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Keyboard Shortcuts\n"))
	b.WriteString("  ─────────────────────────────────────\n")
	b.WriteString("  enter, space  Start/stop selected server\n")
	b.WriteString("  o             Open server in browser\n")
	b.WriteString("  l             View server logs\n")
	b.WriteString("  p             Start/stop proxy\n")
	b.WriteString("  r             Refresh server list\n")
	b.WriteString("  /             Filter servers\n")
	b.WriteString("  ?             Toggle this help\n")
	b.WriteString("  q, ctrl+c     Quit\n")
	return b.String()
}

func (m *Model) toggleServer() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(ServerItem)
	server := item.server

	return func() tea.Msg {
		if server.IsRunning() {
			// Stop server
			if process, err := os.FindProcess(server.PID); err == nil {
				process.Signal(syscall.SIGTERM) //nolint:errcheck // Best effort signal
			}
			server.Status = registry.StatusStopped
			server.PID = 0
			server.StoppedAt = time.Now()
			if err := m.reg.Set(server); err != nil {
				return statusMsgCmd(fmt.Sprintf("Error updating registry: %v", err))
			}
			return statusMsgCmd(fmt.Sprintf("Stopped %s", server.Name))
		}
		// Can't start from TUI without knowing the command
		return statusMsgCmd(fmt.Sprintf("Use 'grove start' in terminal to start %s", server.Name))
	}
}

func (m *Model) openServer() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(ServerItem)
	server := item.server

	if !server.IsRunning() {
		return func() tea.Msg {
			return statusMsgCmd("Server not running")
		}
	}

	return func() tea.Msg {
		if err := browser.Open(server.URL); err != nil {
			return statusMsgCmd(fmt.Sprintf("Failed to open browser: %v", err))
		}
		return statusMsgCmd(fmt.Sprintf("Opened %s", server.URL))
	}
}

func (m *Model) viewLogs() tea.Cmd {
	if m.list.SelectedItem() == nil {
		return nil
	}

	item := m.list.SelectedItem().(ServerItem)
	server := item.server

	if server.LogFile == "" {
		return func() tea.Msg {
			return statusMsgCmd("No log file")
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

func (m *Model) toggleProxy() tea.Cmd {
	proxy := m.reg.GetProxy()

	return func() tea.Msg {
		if proxy.IsRunning() && isProcessRunning(proxy.PID) {
			// Stop proxy
			if process, err := os.FindProcess(proxy.PID); err == nil {
				process.Signal(syscall.SIGTERM) //nolint:errcheck // Best effort signal
			}
			proxy.PID = 0
			if err := m.reg.UpdateProxy(proxy); err != nil {
				return statusMsgCmd(fmt.Sprintf("Error updating registry: %v", err))
			}
			return statusMsgCmd("Proxy stopped")
		}
		return statusMsgCmd("Use 'grove proxy start' in terminal to start proxy")
	}
}

func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Run starts the TUI
func Run(cfg *config.Config) error {
	// Use enhanced version by default
	return RunEnhanced(cfg)
}

// RunClassic starts the classic TUI (kept for backwards compatibility)
func RunClassic(cfg *config.Config) error {
	m, err := New(cfg)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
