package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner represents a loading spinner
type Spinner struct {
	frames []string
	index  int
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		index:  0,
	}
}

// Tick advances the spinner
func (s *Spinner) Tick() {
	s.index = (s.index + 1) % len(s.frames)
}

// View renders the spinner
func (s *Spinner) View() string {
	return spinnerStyle.Render(s.frames[s.index])
}

// SpinnerTickMsg is sent to advance the spinner
type SpinnerTickMsg time.Time

// SpinnerTickCmd returns a command that sends spinner tick messages
func SpinnerTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

// Notification represents a temporary notification message
type Notification struct {
	Message   string
	Type      NotificationType
	Timestamp time.Time
	Duration  time.Duration
}

// NotificationType represents the type of notification
type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationSuccess
	NotificationWarning
	NotificationError
)

// NewNotification creates a new notification
func NewNotification(message string, notifType NotificationType) *Notification {
	return &Notification{
		Message:   message,
		Type:      notifType,
		Timestamp: time.Now(),
		Duration:  3 * time.Second,
	}
}

// IsVisible returns true if the notification should be shown
func (n *Notification) IsVisible() bool {
	return time.Since(n.Timestamp) < n.Duration
}

// View renders the notification
func (n *Notification) View() string {
	if !n.IsVisible() {
		return ""
	}

	var style lipgloss.Style
	var icon string

	switch n.Type {
	case NotificationSuccess:
		style = notificationStyle
		icon = "✓"
	case NotificationWarning:
		style = warningNotificationStyle
		icon = "⚠"
	case NotificationError:
		style = errorNotificationStyle
		icon = "✗"
	default:
		style = lipgloss.NewStyle().Foreground(mutedColor)
		icon = "ℹ"
	}

	return style.Render(fmt.Sprintf("  %s %s", icon, n.Message))
}

// NotificationMsg is sent to display a notification
type NotificationMsg struct {
	Message string
	Type    NotificationType
}

// ProgressBar represents a progress bar
type ProgressBar struct {
	Width    int
	Progress float64 // 0.0 to 1.0
	Label    string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int, label string) *ProgressBar {
	return &ProgressBar{
		Width:    width,
		Progress: 0,
		Label:    label,
	}
}

// SetProgress sets the progress (0.0 to 1.0)
func (p *ProgressBar) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	p.Progress = progress
}

// View renders the progress bar
func (p *ProgressBar) View() string {
	filled := int(float64(p.Width) * p.Progress)
	empty := p.Width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percentage := int(p.Progress * 100)

	return fmt.Sprintf("%s [%s] %d%%",
		p.Label,
		progressBarStyle.Render(bar),
		percentage,
	)
}

// ProgressMsg is sent to update progress
type ProgressMsg struct {
	Progress float64
}

// ActionPanel represents the quick actions panel
type ActionPanel struct {
	Actions []Action
	Visible bool
}

// Action represents a single action in the panel
type Action struct {
	Key         string
	Description string
	Enabled     bool
}

// NewActionPanel creates a new action panel
func NewActionPanel() *ActionPanel {
	return &ActionPanel{
		Actions: []Action{
			{Key: "s", Description: "start server", Enabled: true},
			{Key: "x", Description: "stop server", Enabled: true},
			{Key: "r", Description: "restart server", Enabled: true},
			{Key: "c", Description: "copy URL", Enabled: true},
			{Key: "b", Description: "open in browser", Enabled: true},
			{Key: "l", Description: "view logs", Enabled: true},
		},
		Visible: true,
	}
}

// View renders the action panel
func (a *ActionPanel) View() string {
	if !a.Visible {
		return ""
	}

	var items []string
	for _, action := range a.Actions {
		if action.Enabled {
			item := fmt.Sprintf("[%s] %s", action.Key, action.Description)
			items = append(items, lipgloss.NewStyle().Foreground(mutedColor).Render(item))
		}
	}

	content := strings.Join(items, "  ")
	return actionPanelStyle.Render(content)
}

// UpdateActionAvailability updates which actions are enabled
func (a *ActionPanel) UpdateActionAvailability(serverRunning bool) {
	for i := range a.Actions {
		switch a.Actions[i].Key {
		case "s": // start
			a.Actions[i].Enabled = !serverRunning
		case "x": // stop
			a.Actions[i].Enabled = serverRunning
		case "r": // restart
			a.Actions[i].Enabled = serverRunning
		case "b": // browser
			a.Actions[i].Enabled = serverRunning
		case "l": // logs
			a.Actions[i].Enabled = true // Always available if log file exists
		case "c": // copy URL
			a.Actions[i].Enabled = serverRunning
		}
	}
}

// SearchBar represents a search/filter input
type SearchBar struct {
	Value    string
	Active   bool
	Cursor   int
	MaxWidth int
}

// NewSearchBar creates a new search bar
func NewSearchBar(maxWidth int) *SearchBar {
	return &SearchBar{
		Value:    "",
		Active:   false,
		Cursor:   0,
		MaxWidth: maxWidth,
	}
}

// Activate activates the search bar
func (s *SearchBar) Activate() {
	s.Active = true
}

// Deactivate deactivates the search bar
func (s *SearchBar) Deactivate() {
	s.Active = false
	s.Value = ""
	s.Cursor = 0
}

// InsertChar inserts a character at the cursor position
func (s *SearchBar) InsertChar(ch rune) {
	if s.Cursor >= len(s.Value) {
		s.Value += string(ch)
	} else {
		s.Value = s.Value[:s.Cursor] + string(ch) + s.Value[s.Cursor:]
	}
	s.Cursor++
}

// DeleteChar deletes the character before the cursor
func (s *SearchBar) DeleteChar() {
	if s.Cursor > 0 {
		s.Value = s.Value[:s.Cursor-1] + s.Value[s.Cursor:]
		s.Cursor--
	}
}

// MoveCursorLeft moves the cursor left
func (s *SearchBar) MoveCursorLeft() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

// MoveCursorRight moves the cursor right
func (s *SearchBar) MoveCursorRight() {
	if s.Cursor < len(s.Value) {
		s.Cursor++
	}
}

// View renders the search bar
func (s *SearchBar) View() string {
	if !s.Active {
		return ""
	}

	prompt := "Search: "
	cursor := "█"

	var displayValue string
	if s.Cursor < len(s.Value) {
		displayValue = s.Value[:s.Cursor] + cursor + s.Value[s.Cursor:]
	} else {
		displayValue = s.Value + cursor
	}

	return searchBarStyle.Render(prompt) + displayValue
}
