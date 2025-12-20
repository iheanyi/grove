package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/registry"
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
	rootCmd.AddCommand(selectCmd)
}

// selectItem represents a server in the selection list
type selectItem struct {
	server *registry.Server
}

func (i selectItem) Title() string {
	status := "○"
	if i.server.IsRunning() {
		status = "●"
	} else if i.server.Status == registry.StatusCrashed {
		status = "✗"
	}
	return fmt.Sprintf("%s %s", status, i.server.Name)
}

func (i selectItem) Description() string {
	url := cfg.ServerURL(i.server.Name, i.server.Port)
	return url
}

func (i selectItem) FilterValue() string {
	return i.server.Name
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
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if item, ok := m.list.SelectedItem().(selectItem); ok {
				m.selected = item.server.Name
			}
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"))):
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

	reg.Cleanup()
	servers := reg.List()

	if len(servers) == 0 {
		return fmt.Errorf("no servers registered")
	}

	// Create list items
	items := make([]list.Item, len(servers))
	for i, s := range servers {
		items[i] = selectItem{server: s}
	}

	// Configure list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a server"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
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
