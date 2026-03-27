package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigEditDryRunDoesNotWriteConfig(t *testing.T) {
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script utility not available")
	}

	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)
	configPath := filepath.Join(repoDir, ".sfconfig.yml")

	initial := "main_branch: main\n"
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Answer the interactive wizard with values that would visibly change the file
	// if config edit incorrectly ignores --dry-run.
	inputs := strings.Join([]string{
		"2",
		"feat/",
		"fix/",
		"rel-",
		"3",
		"2",
		"y",
		"y",
		"",
	}, "\n")

	cmd := exec.Command(scriptPath, "-qec", binary+" --dry-run config edit", "/dev/null")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")
	cmd.Stdin = strings.NewReader(inputs)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config edit command failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != initial {
		t.Fatalf("config file changed during --dry-run\noutput:\n%s\nwant:\n%s\ngot:\n%s", out, initial, got)
	}
}

func TestConfigEditYesSkipsWizard(t *testing.T) {
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script utility not available")
	}

	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)
	configPath := filepath.Join(repoDir, ".sfconfig.yml")
	if err := os.WriteFile(configPath, []byte("main_branch: main\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptPath, "-qec", binary+" --yes config edit", "/dev/null")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("config edit --yes hung waiting for wizard input\noutput:\n%s", out)
	}
	if err != nil {
		t.Fatalf("config edit --yes failed: %v\n%s", err, out)
	}
}

func TestConfigEditYesAllowsNonTTY(t *testing.T) {
	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)
	configPath := filepath.Join(repoDir, ".sfconfig.yml")

	initial := "main_branch: main\n"
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := exec.Command(binary, "--yes", "config", "edit")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config edit --yes should not require a TTY when prompts are skipped: %v\n%s", err, out)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != initial {
		t.Fatalf("config edit --yes changed config unexpectedly\noutput:\n%s\nwant:\n%s\ngot:\n%s", out, initial, got)
	}
}

func buildReviewBinary(t *testing.T) string {
	t.Helper()

	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Abs(..) error = %v", err)
	}

	binary := filepath.Join(t.TempDir(), "git-sf")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "GOCACHE=/tmp/gocache", "GOPATH=/tmp/gopath")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	return binary
}

func initConfigEditRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runConfigGit(t, repoDir, "init", "-b", "main")
	runConfigGit(t, repoDir, "config", "user.email", "test@example.com")
	runConfigGit(t, repoDir, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runConfigGit(t, repoDir, "add", "README.md")
	runConfigGit(t, repoDir, "commit", "-m", "init")

	return repoDir
}
