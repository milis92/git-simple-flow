package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	DryRun  bool
	Verbose bool
	Output  io.Writer
}

func NewRunner(dryRun, verbose bool) *Runner {
	return &Runner{
		DryRun:  dryRun,
		Verbose: verbose,
		Output:  os.Stderr,
	}
}

func (r *Runner) Run(name string, args ...string) (string, error) {
	cmdStr := name + " " + strings.Join(args, " ")

	if r.DryRun {
		fmt.Fprintf(r.Output, "  [dry-run] %s\n", cmdStr)
		return "", nil
	}

	if r.Verbose {
		fmt.Fprintf(r.Output, "  $ %s\n", cmdStr)
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
