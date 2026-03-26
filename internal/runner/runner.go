// Package runner provides a command execution abstraction with dry-run and verbose support.
package runner

import (
	"bytes"
	"context"
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
	// Context is used by Run when set, allowing callers to cancel subprocesses.
	Context context.Context
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

// WithContext returns a shallow copy of the runner that uses ctx for command execution.
func (r *Runner) WithContext(ctx context.Context) *Runner {
	cloned := *r
	cloned.Context = ctx
	return &cloned
}

// Run executes the named command with the given arguments. It captures stdout
// and returns it as a trimmed string. On failure, the returned error includes
// the command string and stderr output.
func (r *Runner) Run(name string, args ...string) (string, error) {
	return r.RunContext(r.Context, name, args...)
}

// RunContext executes the named command with the given context. If the context
// is canceled, the subprocess is terminated and ctx.Err() is returned.
func (r *Runner) RunContext(ctx context.Context, name string, args ...string) (string, error) {
	cmdStr := name + " " + strings.Join(args, " ")

	if r.DryRun {
		_, _ = fmt.Fprintf(r.Output, "  [dry-run] %s\n", cmdStr)
		return "", nil
	}

	if r.Verbose {
		_, _ = fmt.Fprintf(r.Output, "  $ %s\n", cmdStr)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return "", fmt.Errorf("%s: %s", cmdStr, stderrText)
		}

		return "", fmt.Errorf("%s: %w", cmdStr, err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
