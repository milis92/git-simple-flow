// Package ui provides styled terminal output using lipgloss.
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

// UI handles styled terminal output. All output methods write to Out.
type UI struct {
	// Out is the writer for all UI output. Defaults to os.Stdout.
	Out io.Writer
}

// New creates a UI that writes to stdout.
func New() *UI {
	return &UI{Out: os.Stdout}
}

// Success prints a message with a green checkmark prefix.
func (u *UI) Success(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", successStyle.Render("✓"), msg)
}

// Error prints a message with a red cross prefix.
func (u *UI) Error(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", errorStyle.Render("✗"), msg)
}

// Warning prints a message with a yellow exclamation prefix.
func (u *UI) Warning(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", warnStyle.Render("!"), msg)
}

// Info prints a message with a blue bullet prefix.
func (u *UI) Info(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", infoStyle.Render("●"), msg)
}

// Muted prints a message in dimmed/grey text.
func (u *UI) Muted(msg string) {
	fmt.Fprintf(u.Out, "  %s\n", mutedStyle.Render(msg))
}

// DryRun prints a message with a purple "[dry-run]" prefix.
func (u *UI) DryRun(msg string) {
	fmt.Fprintf(u.Out, "  %s %s\n", dryRunStyle.Render("[dry-run]"), msg)
}

// Blank prints an empty line.
func (u *UI) Blank() {
	fmt.Fprintln(u.Out)
}

// Header prints a message with surrounding blank lines.
func (u *UI) Header(msg string) {
	fmt.Fprintf(u.Out, "\n  %s\n\n", msg)
}

// Result prints a message with surrounding blank lines, used for the final output of a workflow step.
func (u *UI) Result(msg string) {
	fmt.Fprintf(u.Out, "\n  %s\n\n", msg)
}

// Confirm prints a y/N prompt and reads a single line from stdin.
// Returns true if the user answers "y", "Y", or "yes".
func (u *UI) Confirm(msg string) (bool, error) {
	fmt.Fprintf(u.Out, "  %s [y/N] ", msg)
	var response string
	_, err := fmt.Fscanln(os.Stdin, &response)
	if err != nil {
		return false, nil
	}
	return response == "y" || response == "Y" || response == "yes", nil
}
