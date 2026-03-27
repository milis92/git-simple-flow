package feature

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestDiscardClassicDryRunChecksRealGHAuth(t *testing.T) {
	repoDir := initFeatureRepo(t)
	installFeatureDiscardAuthFailureGH(t)

	var out bytes.Buffer
	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	if err := svc.Discard(""); err != nil {
		t.Fatalf("Discard() error = %v, want dry-run preview to succeed", err)
	}

	if strings.Contains(out.String(), "Closed PR") {
		t.Fatalf("Discard() output = %q, should not claim the PR was closed when gh auth fails", out.String())
	}
	if !strings.Contains(out.String(), "gh not authenticated") {
		t.Fatalf("Discard() output = %q, want auth failure to be surfaced during dry-run", out.String())
	}
}

func TestFinishInteractiveDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initFeatureRepo(t)
	installChecksGH(t, `[{"name":"ci","state":"SUCCESS","bucket":"pass"}]`)

	var progressTitle, progressBranch string

	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:            git.New(r, repoDir),
		GH:             gh.New(r),
		UI:             u,
		Config:         config.Defaults(),
		RunTitlePrompt: ui.RunTitlePrompt,
		RunProgress: func(title, subtitle string, defs []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			progressTitle = title
			progressBranch = subtitle
			return workflow(context.Background(), ui.StepCallbacks{
				Start: func() {},
				Done:  func() {},
				Fail:  func(string) {},
				Skip:  func(string) {},
			})
		},
	}

	if err := svc.Finish(FinishOpts{}); err != nil {
		t.Fatalf("Finish() error = %v, want dry-run preview to succeed", err)
	}
	if progressTitle != "git sf feature finish" {
		t.Fatalf("progress title = %q, want %q", progressTitle, "git sf feature finish")
	}
	if progressBranch != "feature/test" {
		t.Fatalf("progress branch = %q, want %q", progressBranch, "feature/test")
	}
}

func TestDiscardInteractiveDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initFeatureRepo(t)

	var progressTitle, progressBranch string

	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunProgress: func(title, subtitle string, defs []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			progressTitle = title
			progressBranch = subtitle
			return nil
		},
	}

	if err := svc.Discard(""); err != nil {
		t.Fatalf("Discard() error = %v, want dry-run preview to succeed", err)
	}
	if progressTitle != "git sf feature discard" {
		t.Fatalf("progress title = %q, want %q", progressTitle, "git sf feature discard")
	}
	if progressBranch != "feature/test" {
		t.Fatalf("progress branch = %q, want %q", progressBranch, "feature/test")
	}
}

func TestPublishDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initFeatureRepo(t)
	installFakeGH(t)

	var out bytes.Buffer
	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	if err := svc.Publish(PublishOpts{Title: "test"}); err != nil {
		t.Fatalf("Publish() error = %v, want dry-run preview to succeed", err)
	}

	if !strings.Contains(out.String(), "feature/test") {
		t.Fatalf("Publish() output = %q, want branch name resolved from real repo", out.String())
	}
}

func TestStartDraftPRPromptCancellationDoesNotCreateBranch(t *testing.T) {
	repoDir := initFeatureStartRepo(t)
	installFakeGH(t)

	promptErr := errors.New("prompt cancelled")

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunTitlePrompt: func(string, bool) (ui.InputPromptResult, error) {
			return ui.InputPromptResult{}, promptErr
		},
		RunProgress: ui.RunProgress,
	}

	err := svc.Start("new-api", StartOpts{DraftPR: true})
	if !errors.Is(err, promptErr) {
		t.Fatalf("Start() error = %v, want %v", err, promptErr)
	}

	branches := strings.Fields(runGit(t, repoDir, "branch", "--format=%(refname:short)"))
	for _, branch := range branches {
		if branch == "feature/new-api" {
			t.Fatalf("Start() created %q before the PR prompt completed", branch)
		}
	}

	if current := runGit(t, repoDir, "rev-parse", "--abbrev-ref", "HEAD"); current != "main" {
		t.Fatalf("HEAD = %q, want %q after prompt cancellation", current, "main")
	}
}

func initFeatureStartRepo(t *testing.T) string {
	t.Helper()

	bareDir := t.TempDir()
	runGit(t, bareDir, "init", "--bare", "-b", "main")

	parentDir := t.TempDir()
	repoDir := filepath.Join(parentDir, "work")
	runGit(t, parentDir, "clone", bareDir, "work")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	runGit(t, repoDir, "push", "-u", "origin", "main")

	return repoDir
}

func installFeatureDiscardAuthFailureGH(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  echo "not logged in" >&2
  exit 1
fi
echo "unexpected gh command: $*" >&2
exit 1
`
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}
