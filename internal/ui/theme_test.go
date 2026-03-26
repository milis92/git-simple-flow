package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestThemeColorsAreDefined(t *testing.T) {
	theme := DefaultTheme()

	// All status colors must be non-empty styles
	styles := map[string]lipgloss.Style{
		"Success": theme.Success,
		"Error":   theme.Error,
		"Warning": theme.Warning,
		"Info":    theme.Info,
		"Muted":   theme.Muted,
		"DryRun":  theme.DryRun,
	}
	for name, style := range styles {
		rendered := style.Render("test")
		if rendered == "" {
			t.Errorf("theme.%s.Render produced empty string", name)
		}
	}
}

func TestThemeIconsAreDefined(t *testing.T) {
	theme := DefaultTheme()

	icons := map[string]string{
		"Done":    theme.IconDone,
		"Fail":    theme.IconFail,
		"Pending": theme.IconPending,
		"Confirm": theme.IconConfirm,
	}
	for name, icon := range icons {
		if icon == "" {
			t.Errorf("theme.Icon%s is empty", name)
		}
	}
}

func TestHuhThemeIsNotNil(t *testing.T) {
	theme := DefaultTheme()
	huhTheme := theme.HuhTheme()
	if huhTheme == nil {
		t.Error("HuhTheme() returned nil")
	}
}
