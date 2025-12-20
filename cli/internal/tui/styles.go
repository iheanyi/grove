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
)
