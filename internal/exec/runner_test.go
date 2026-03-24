package exec

import (
	"bytes"
	"strings"
	"testing"
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
