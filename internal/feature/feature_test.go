package feature

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestPublishPromptsBeforePush(t *testing.T) {
	repoDir := initFeatureRepo(t)
	installFakeGH(t)

	promptErr := errors.New("prompt cancelled")
	oldPrompt := runTitlePrompt
	runTitlePrompt = func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
		return ui.InputPromptResult{}, promptErr
	}
	t.Cleanup(func() {
		runTitlePrompt = oldPrompt
	})

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err := svc.Publish(PublishOpts{})
	if !errors.Is(err, promptErr) {
		t.Fatalf("Publish() error = %v, want %v", err, promptErr)
	}
}

func TestPublishSkipsOptionalPromptWhenAutoConfirm(t *testing.T) {
	repoDir := initFeatureRepo(t)
	installFakeGH(t)

	promptErr := errors.New("prompt should be skipped")
	oldPrompt := runTitlePrompt
	runTitlePrompt = func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
		return ui.InputPromptResult{}, promptErr
	}
	t.Cleanup(func() {
		runTitlePrompt = oldPrompt
	})

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err := svc.Publish(PublishOpts{})
	if err == nil {
		t.Fatal("Publish() error = nil, want push error without a remote")
	}
	if errors.Is(err, promptErr) {
		t.Fatalf("Publish() error = %v, prompt should have been skipped", err)
	}
	if !strings.Contains(err.Error(), "push -u origin") {
		t.Fatalf("Publish() error = %v, want git push failure", err)
	}
}

func initFeatureRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	runGit(t, repoDir, "checkout", "-b", "feature/test")

	return repoDir
}

func installFakeGH(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n  exit 0\nfi\necho \"unexpected gh command: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}

	return strings.TrimSpace(string(out))
}
