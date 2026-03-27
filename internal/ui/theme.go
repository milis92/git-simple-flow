package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme holds all shared visual styles for the CLI.
type Theme struct {
	// Status colors
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style
	DryRun  lipgloss.Style

	// Icons
	IconDone    string
	IconFail    string
	IconPending string
	IconConfirm string
	IconWarning string
	IconInfo    string

	// Panel styling
	Border lipgloss.Style
}

// DefaultTheme returns the standard git-sf visual theme.
func DefaultTheme() Theme {
	return Theme{
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		DryRun:  lipgloss.NewStyle().Foreground(lipgloss.Color("5")),

		IconDone:    "✓",
		IconFail:    "✗",
		IconPending: "○",
		IconConfirm: "?",
		IconWarning: "!",
		IconInfo:    "●",

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2),
	}
}

// HuhTheme returns a Huh form theme derived from the shared theme.
func (t Theme) HuhTheme() *huh.Theme {
	theme := huh.ThemeCharm()
	theme.Focused.Base = t.Border
	theme.Focused.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	theme.Focused.SelectedOption = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	return theme
}
