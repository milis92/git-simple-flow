package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	dryRunStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
)

type UI struct {
	Out io.Writer
}

func New() *UI {
	return &UI{Out: os.Stdout}
}

func (u *UI) Success(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", successStyle.Render("✓"), msg)
}

func (u *UI) Error(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", errorStyle.Render("✗"), msg)
}

func (u *UI) Warning(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", warnStyle.Render("!"), msg)
}

func (u *UI) Info(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", infoStyle.Render("●"), msg)
}

func (u *UI) Muted(msg string) {
	fmt.Fprintf(u.Out, "  %s\n", mutedStyle.Render(msg))
}

func (u *UI) DryRun(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", dryRunStyle.Render("[dry-run]"), msg)
}

func (u *UI) Blank() {
	fmt.Fprintln(u.Out)
}

func (u *UI) Header(msg string) {
	fmt.Fprintf(u.Out, "\n  %s\n\n", msg)
}

func (u *UI) Result(msg string) {
	fmt.Fprintf(u.Out, "\n  %s\n\n", msg)
}

func (u *UI) Confirm(msg string) (bool, error) {
	fmt.Fprintf(u.Out, "  %s [y/N] ", msg)
	var response string
	_, err := fmt.Fscanln(os.Stdin, &response)
	if err != nil {
		return false, nil
	}
	return response == "y" || response == "Y" || response == "yes", nil
}
