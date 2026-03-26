package ui

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether the given file is connected to a terminal.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// ShouldInteract returns true if interactive UI should be used.
// It returns false when noInteractive is true, or when isTTY is false.
func ShouldInteract(isTTY, noInteractive bool) bool {
	if noInteractive {
		return false
	}
	return isTTY
}
