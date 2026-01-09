// Package styles provides shared styling constants and utilities for Grove CLI and TUI.
package styles

import "github.com/charmbracelet/lipgloss"

// Colors - semantic color palette for consistent theming
var (
	// Primary colors
	Primary   = lipgloss.Color("#7C3AED") // Purple - brand color
	Secondary = lipgloss.Color("#10B981") // Green - success/running
	Warning   = lipgloss.Color("#F59E0B") // Yellow/Amber - warnings
	Error     = lipgloss.Color("#EF4444") // Red - errors/stopped
	Muted     = lipgloss.Color("#6B7280") // Gray - secondary text

	// Accent colors
	Accent      = lipgloss.Color("#A78BFA") // Light purple - selection highlight
	Info        = lipgloss.Color("#3B82F6") // Blue - informational
	Cyan        = lipgloss.Color("#06B6D4") // Cyan - numbers/special
	Purple      = lipgloss.Color("#8B5CF6") // Purple - paths
	PurpleLight = lipgloss.Color("#A855F7") // Light purple - duration
	Yellow      = lipgloss.Color("#EAB308") // Yellow/Gold - PATCH method, controllers

	// Neutral colors
	White   = lipgloss.Color("#FFFFFF")
	Dim     = lipgloss.Color("240") // ANSI 240 - borders, dim text
	Header  = lipgloss.Color("252") // ANSI 252 - table headers
	Link    = lipgloss.Color("12")  // ANSI 12 - blue links
	Success = lipgloss.Color("10")  // ANSI 10 - green success
	Number  = lipgloss.Color("11")  // ANSI 11 - yellow numbers
	Name    = lipgloss.Color("14")  // ANSI 14 - cyan names
)

// Column widths for table formatting
const (
	// Standard column widths
	ColWidthName     = 35
	ColWidthBranch   = 25
	ColWidthStatus   = 10
	ColWidthPort     = 6
	ColWidthPath     = 40
	ColWidthWorktree = 30
	ColWidthType     = 10
	ColWidthTask     = 25
	ColWidthWorkDir  = 50

	// Truncation widths for text display
	TruncateDefault = 60
	TruncateTitle   = 50
	TruncateShort   = 25
)

// Table separator widths
const (
	SeparatorFull   = 120
	SeparatorMedium = 80
	SeparatorShort  = 70
	SeparatorSmall  = 40
)

// Truncation tail
const TruncateTail = "..."

// Common styles
var (
	// Header styles
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(Header).PaddingRight(2)
	LinkHeader  = lipgloss.NewStyle().Bold(true).Foreground(Link)

	// Text styles
	NameStyle    = lipgloss.NewStyle().Bold(true).Foreground(Name)
	URLStyle     = lipgloss.NewStyle().Foreground(Success)
	StatsStyle   = lipgloss.NewStyle().Foreground(Number)
	DimStyle     = lipgloss.NewStyle().Foreground(Dim)
	AccentStyle  = lipgloss.NewStyle().Foreground(Accent)
	MutedStyle   = lipgloss.NewStyle().Foreground(Muted)
	PrimaryStyle = lipgloss.NewStyle().Foreground(Primary)

	// Status styles
	RunningStyle = lipgloss.NewStyle().Foreground(Secondary)
	StoppedStyle = lipgloss.NewStyle().Foreground(Muted)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Error).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(Secondary).Bold(true)

	// Selection styles
	SelectedTitle = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	SelectedDesc  = lipgloss.NewStyle().Foreground(Muted)

	// Table styles
	CellStyle   = lipgloss.NewStyle().PaddingRight(2)
	BorderStyle = lipgloss.NewStyle().Foreground(Dim)
)
