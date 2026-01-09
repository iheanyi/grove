package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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
