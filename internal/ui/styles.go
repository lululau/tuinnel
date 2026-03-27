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
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorCyan)

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
)

// Ensure color.Color is used to prevent import being removed.
var _ color.Color = ColorGreen
