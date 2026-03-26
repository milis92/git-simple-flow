// Package ui provides styled terminal output, interactive forms, and progress
// views for the git-sf CLI using the Charm ecosystem (lipgloss, Huh, Bubble Tea).
package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrConfirmationRequired is returned when a confirmation prompt cannot be
// shown because interactive prompting is disabled.
var ErrConfirmationRequired = errors.New("confirmation required in non-interactive mode")

// UI handles styled terminal output. All output methods write to Out.
type UI struct {
	// Out is the writer for all UI output. Defaults to os.Stdout.
	Out io.Writer
	// In is the reader for user input. Defaults to os.Stdin.
	In io.Reader
	// Interactive indicates whether interactive prompts (Huh forms) should be used.
	Interactive bool
	// AutoConfirm, when true, skips confirmation prompts and returns true.
	AutoConfirm bool
	// theme holds the shared visual styles used by all output methods.
	theme Theme
}

// New creates a UI that writes to stdout and reads from stdin.
func New() *UI {
	return &UI{Out: os.Stdout, In: os.Stdin, theme: DefaultTheme()}
}

// ShouldPrompt reports whether optional interactive prompts should be shown.
// It returns true only when Interactive is true and AutoConfirm is false.
func (u *UI) ShouldPrompt() bool {
	return u.Interactive && !u.AutoConfirm
}

// Success prints a message with a green checkmark prefix.
func (u *UI) Success(msg string) {
	_, _ = fmt.Fprintf(u.Out, "  %s %s\n", u.theme.Success.Render(u.theme.IconDone), msg)
}

// Error prints a message with a red cross prefix.
func (u *UI) Error(msg string) {
	_, _ = fmt.Fprintf(u.Out, "  %s %s\n", u.theme.Error.Render(u.theme.IconFail), msg)
}

// Warning prints a message with a yellow exclamation prefix.
func (u *UI) Warning(msg string) {
	_, _ = fmt.Fprintf(u.Out, "  %s %s\n", u.theme.Warning.Render(u.theme.IconWarning), msg)
}

// Info prints a message with a blue bullet prefix.
func (u *UI) Info(msg string) {
	_, _ = fmt.Fprintf(u.Out, "  %s %s\n", u.theme.Info.Render(u.theme.IconInfo), msg)
}

// Muted prints a message in dimmed/grey text.
func (u *UI) Muted(msg string) {
	_, _ = fmt.Fprintf(u.Out, "  %s\n", u.theme.Muted.Render(msg))
}

// DryRun prints a message with a purple "[dry-run]" prefix.
func (u *UI) DryRun(msg string) {
	_, _ = fmt.Fprintf(u.Out, "  %s %s\n", u.theme.DryRun.Render("[dry-run]"), msg)
}

// Blank prints an empty line.
func (u *UI) Blank() {
	_, _ = fmt.Fprintln(u.Out)
}

// Header prints a message with surrounding blank lines.
func (u *UI) Header(msg string) {
	_, _ = fmt.Fprintf(u.Out, "\n  %s\n\n", msg)
}

// Result prints a message with surrounding blank lines, used for the final output of a workflow step.
func (u *UI) Result(msg string) {
	_, _ = fmt.Fprintf(u.Out, "\n  %s\n\n", msg)
}

// Confirm prints a y/N prompt and reads a single line from stdin.
// Returns true if the user answers "y", "Y", or "yes".
// If AutoConfirm is set, returns true immediately without waiting for input.
// If interactive prompting is disabled, it returns an error instructing the
// caller to rerun with --yes.
func (u *UI) Confirm(msg string) (bool, error) {
	if u.AutoConfirm {
		_, _ = fmt.Fprintf(u.Out, "  %s [y/N] y (auto-confirmed)\n", msg)
		return true, nil
	}
	if !u.Interactive {
		_, _ = fmt.Fprintf(u.Out, "  %s [y/N] skipped (non-interactive; rerun with --yes)\n", msg)
		return false, fmt.Errorf("%w; rerun with --yes to proceed", ErrConfirmationRequired)
	}
	_, _ = fmt.Fprintf(u.Out, "  %s [y/N] ", msg)
	var response string
	_, err := fmt.Fscanln(u.In, &response)
	if err != nil {
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "unexpected newline") {
			return false, nil
		}
		return false, fmt.Errorf("could not read user input: %w", err)
	}
	return response == "y" || response == "Y" || response == "yes", nil
}
