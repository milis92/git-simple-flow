package ui

import (
	"os"
	"testing"
)

func TestIsInteractiveReturnsFalseForPipe(t *testing.T) {
	// os.Stdin in tests is not a terminal
	result := IsTerminal(os.Stdin)
	if result {
		t.Error("IsTerminal should return false for piped stdin in tests")
	}
}

func TestShouldInteractRespectsForceFlag(t *testing.T) {
	// Even if stdin were a TTY, noInteractive=true should override
	result := ShouldInteract(false, true)
	if result {
		t.Error("ShouldInteract should return false when noInteractive is true")
	}
}

func TestShouldInteractReturnsFalseWhenNotTTY(t *testing.T) {
	// In tests, stdin is not a TTY, so isTTY=false
	result := ShouldInteract(false, false)
	if result {
		t.Error("ShouldInteract should return false when isTTY is false")
	}
}

func TestShouldInteractReturnsTrueWhenTTYAndNotDisabled(t *testing.T) {
	result := ShouldInteract(true, false)
	if !result {
		t.Error("ShouldInteract should return true when isTTY is true and noInteractive is false")
	}
}
