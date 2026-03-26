package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	r := NewRunner(false, false)
	out, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Errorf("output = %q, want %q", out, "hello")
	}
}

func TestRunFailure(t *testing.T) {
	r := NewRunner(false, false)
	_, err := r.Run("false")
	if err == nil {
		t.Error("expected error from `false` command")
	}
}

func TestRunDryRun(t *testing.T) {
	var buf bytes.Buffer
	r := NewRunner(true, false)
	r.Output = &buf
	out, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
	if out != "" {
		t.Errorf("dry-run output should be empty, got %q", out)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("dry-run should log, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "echo hello") {
		t.Errorf("dry-run should show command, got %q", buf.String())
	}
}

func TestRunVerbose(t *testing.T) {
	var buf bytes.Buffer
	r := NewRunner(false, true)
	r.Output = &buf
	_, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "echo hello") {
		t.Errorf("verbose should log command, got %q", buf.String())
	}
}

func TestRunContextCancelsProcess(t *testing.T) {
	marker := t.TempDir() + "/done"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	time.AfterFunc(100*time.Millisecond, cancel)

	r := NewRunner(false, false)
	_, err := r.RunContext(
		ctx,
		"env",
		"GO_WANT_RUNNER_HELPER_PROCESS=1",
		os.Args[0],
		"-test.run=TestRunnerHelperProcess",
		"--",
		"sleep-write",
		marker,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunContext() error = %v, want context.Canceled", err)
	}

	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("marker file should not exist after cancellation, stat err = %v", statErr)
	}
}

func TestRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_RUNNER_HELPER_PROCESS") != "1" {
		return
	}

	args := helperArgs(os.Args)
	if len(args) != 2 || args[0] != "sleep-write" {
		os.Exit(2)
	}

	time.Sleep(2 * time.Second)
	if err := os.WriteFile(args[1], []byte("done"), 0644); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

func helperArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}

	return nil
}
