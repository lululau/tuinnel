package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

var (
	ColorGreen   = lipgloss.Color("#04B575")
	ColorRed     = lipgloss.Color("#FF5F57")
	ColorYellow  = lipgloss.Color("#F1C40F")
	ColorGray    = lipgloss.Color("#6C7086")
	ColorCyan    = lipgloss.Color("#00D7FF")
	ColorDim     = lipgloss.Color("#45475A")
	ColorText    = lipgloss.Color("#CDD6F4")
	ColorSubtext = lipgloss.Color("#A6ADC8")
	ColorBg      = lipgloss.Color("#1E1E2E")
	ColorSurface = lipgloss.Color("#313244")

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan)

	StyleRunning = lipgloss.NewStyle().
			Foreground(ColorGreen)

	StyleStopped = lipgloss.NewStyle().
			Foreground(ColorGray)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorRed)

	StyleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			Padding(0, 1).
			Background(ColorSurface)

	StyleTabSep = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorSubtext).
				Padding(0, 1)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorGray)

	StyleFocused = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleInput = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSurface)

	StyleHelpOverlay = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorCyan).
				Padding(1, 3)

	StyleHelpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			MarginBottom(1)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true).
			Width(14)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleHelpSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSubtext).
			MarginTop(1).
			MarginBottom(0)

	StyleHelpClose = lipgloss.NewStyle().
			Foreground(ColorGray).
			MarginTop(1)
)

// Ensure color.Color is used to prevent import being removed.
var _ color.Color = ColorGreen
