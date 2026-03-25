// Package runner provides a command execution abstraction with dry-run and verbose support.
package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Runner executes shell commands. It supports dry-run mode (prints commands
// without executing) and verbose mode (prints commands before executing).
type Runner struct {
	// DryRun, when true, prints the command to Output instead of executing it.
	DryRun bool
	// Verbose, when true, prints the command to Output before executing it.
	Verbose bool
	// Output is the writer for diagnostic output (dry-run and verbose messages).
	Output io.Writer
}

// NewRunner creates a Runner with the given dry-run and verbose settings.
// Diagnostic output defaults to stderr.
func NewRunner(dryRun, verbose bool) *Runner {
	return &Runner{
		DryRun:  dryRun,
		Verbose: verbose,
		Output:  os.Stderr,
	}
}

// Run executes the named command with the given arguments. It captures stdout
// and returns it as a trimmed string. On failure, the returned error includes
// the command string and stderr output.
func (r *Runner) Run(name string, args ...string) (string, error) {
	cmdStr := name + " " + strings.Join(args, " ")

	if r.DryRun {
		_, _ = fmt.Fprintf(r.Output, "  [dry-run] %s\n", cmdStr)
		return "", nil
	}

	if r.Verbose {
		_, _ = fmt.Fprintf(r.Output, "  $ %s\n", cmdStr)
	}

	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", cmdStr, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}
