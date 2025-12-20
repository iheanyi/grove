package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Yellow
	errorColor     = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	bgColor        = lipgloss.Color("#1F2937") // Dark gray

	// Status colors
	runningColor = lipgloss.Color("#10B981") // Green
	stoppedColor = lipgloss.Color("#6B7280") // Gray
	crashedColor = lipgloss.Color("#EF4444") // Red

	// Health colors
	healthyColor   = lipgloss.Color("#10B981") // Green
	unhealthyColor = lipgloss.Color("#EF4444") // Red
	unknownColor   = lipgloss.Color("#6B7280") // Gray

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(runningColor)

	statusStoppedStyle = lipgloss.NewStyle().
				Foreground(stoppedColor)

	statusCrashedStyle = lipgloss.NewStyle().
				Foreground(crashedColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	logStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	logHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor).
			MarginBottom(1)

	// Health styles
	healthyStyle = lipgloss.NewStyle().
			Foreground(healthyColor)

	unhealthyStyle = lipgloss.NewStyle().
			Foreground(unhealthyColor)

	unknownStyle = lipgloss.NewStyle().
			Foreground(unknownColor)

	// Notification styles
	notificationStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	errorNotificationStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	warningNotificationStyle = lipgloss.NewStyle().
					Foreground(warningColor).
					Bold(true)

	// Spinner style
	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// Progress bar style
	progressBarStyle = lipgloss.NewStyle().
				Foreground(primaryColor)

	// Action panel style
	actionPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1).
				MarginTop(1)

	// Search bar style
	searchBarStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
)
