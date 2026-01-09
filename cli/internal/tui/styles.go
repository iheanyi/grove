package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/styles"
)

var (
	// Colors - use shared styles package
	primaryColor   = styles.Primary
	secondaryColor = styles.Secondary
	warningColor   = styles.Warning
	errorColor     = styles.Error
	mutedColor     = styles.Muted

	// Status colors
	runningColor = styles.Secondary
	stoppedColor = styles.Muted
	crashedColor = styles.Error

	// Health colors
	healthyColor   = styles.Secondary
	unhealthyColor = styles.Error
	unknownColor   = styles.Muted

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(runningColor)

	statusStoppedStyle = lipgloss.NewStyle().
				Foreground(stoppedColor)

	statusCrashedStyle = lipgloss.NewStyle().
				Foreground(crashedColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

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

	// Action panel style
	actionPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1).
				MarginTop(1)
)
