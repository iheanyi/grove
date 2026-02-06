package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
	"github.com/spf13/cobra"
)

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Interactive server selector (fzf-style)",
	Long: `Interactive fuzzy-finder style server selector.

Use arrow keys or j/k to navigate, type to filter, enter to select.
The selected server's name is printed to stdout.

Useful with shell functions:
  # Start the selected server
  grove start $(grove select)

  # Open the selected server in browser
  grove open $(grove select)

  # Change to selected server's directory
  cd "$(grove cd $(grove select))"`,
	RunE: runSelect,
}

func init() {
	selectCmd.GroupID = "server"
	rootCmd.AddCommand(selectCmd)
}

// selectItem represents a server in the selection list
type selectItem struct {
	server *registry.Server
}

// Title returns plain text with status icon prefix
func (i selectItem) Title() string {
	statusIcon := "○"
	if i.server.IsRunning() {
		statusIcon = "●"
	} else if i.server.Status == registry.StatusCrashed {
		statusIcon = "✗"
	}
	return statusIcon + " " + i.server.Name
}

func (i selectItem) Description() string {
	url := cfg.ServerURL(i.server.Name, i.server.Port)
	return url
}

func (i selectItem) FilterValue() string {
	return i.server.Name
}

// StatusIcon returns the status icon for display
func (i selectItem) StatusIcon() string {
	if i.server.IsRunning() {
		return "●"
	} else if i.server.Status == registry.StatusCrashed {
		return "✗"
	}
	return "○"
}

// IsRunning returns whether the server is running
func (i selectItem) IsRunning() bool {
	return i.server.IsRunning()
}

// IsCrashed returns whether the server crashed
func (i selectItem) IsCrashed() bool {
	return i.server.Status == registry.StatusCrashed
}

// selectKeys defines key bindings for the selector
var selectKeys = struct {
	Enter key.Binding
	Quit  key.Binding
}{
	Enter: key.NewBinding(key.WithKeys("enter")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}

// selectModel is the bubbletea model for the selector
type selectModel struct {
	list     list.Model
	selected string
	quitting bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When filtering, let the list handle all keys
		if m.list.FilterState() != list.Unfiltered {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Only handle our keys when NOT filtering
		switch {
		case key.Matches(msg, selectKeys.Enter):
			if item, ok := m.list.SelectedItem().(selectItem); ok {
				m.selected = item.server.Name
			}
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, selectKeys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func runSelect(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Cleanup stale entries (non-critical)
	if _, err := reg.Cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
	}
	servers := reg.List()

	if len(servers) == 0 {
		return fmt.Errorf("no servers registered")
	}

	// Create list items
	items := make([]list.Item, len(servers))
	for i, s := range servers {
		items[i] = selectItem{server: s}
	}

	// Configure list with default delegate - Title() includes status icon as plain text
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(styles.Accent).
		Bold(true)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(styles.Muted)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a server"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(styles.Accent).
		Bold(true).
		Padding(0, 1)
	l.SetShowHelp(true)

	// Run the program
	m := selectModel{list: l}
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Print the selected server name to stdout
	if fm, ok := finalModel.(selectModel); ok && fm.selected != "" {
		fmt.Fprintln(os.Stdout, fm.selected)
	}

	return nil
}
