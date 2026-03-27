package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCommandWritesWarningsToStderr(t *testing.T) {
	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)

	configPath := filepath.Join(repoDir, ".sfconfig.yml")
	if err := os.WriteFile(configPath, []byte("main_branch: \"   \"\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := exec.Command(binary, "config")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("config command failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	if strings.Contains(stdout.String(), "Ignoring invalid repo config") {
		t.Fatalf("stdout should stay machine-parseable, got warning output:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Ignoring invalid repo config") {
		t.Fatalf("stderr = %q, want invalid repo config warning", stderr.String())
	}
}
