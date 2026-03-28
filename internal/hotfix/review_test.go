package hotfix

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

func TestFinishInteractiveReleasePrintsReleasedTag(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	installFinishReleaseGH(t)

	// Add a commit on the hotfix branch so the squash flow has something to commit.
	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "wip: hotfix attempt")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunProgress: func(_ string, _ string, _ []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			return workflow(context.Background(), ui.StepCallbacks{
				Start: func() {},
				Done:  func() {},
				Fail:  func(string) {},
				Skip:  func(string) {},
			})
		},
	}

	if err := svc.Finish(FinishOpts{Release: true}); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}

	if got := runGit(t, repoDir, "tag", "-l", "v0.1.1"); got != "v0.1.1" {
		t.Fatalf("expected release tag v0.1.1 to be created, got %q", got)
	}

	if !strings.Contains(out.String(), "Hotfix released v0.1.1") {
		t.Fatalf("Finish() output = %q, want final result to include released tag", out.String())
	}
}

func TestFinishInteractiveDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initHotfixRepo(t)
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
	if progressTitle != "git sf hotfix finish" {
		t.Fatalf("progress title = %q, want %q", progressTitle, "git sf hotfix finish")
	}
	if progressBranch != "hotfix/test" {
		t.Fatalf("progress branch = %q, want %q", progressBranch, "hotfix/test")
	}
}

func TestFinishInteractiveReleaseDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	installFinishReleaseGH(t)

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
		RunProgress: func(_ string, _ string, _ []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			return workflow(context.Background(), ui.StepCallbacks{
				Start: func() {},
				Done:  func() {},
				Fail:  func(string) {},
				Skip:  func(string) {},
			})
		},
	}

	if err := svc.Finish(FinishOpts{Release: true}); err != nil {
		t.Fatalf("Finish() error = %v, want dry-run release preview to succeed", err)
	}
}

func TestFinishClassicReleaseDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	installFinishReleaseGH(t)

	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	if err := svc.Finish(FinishOpts{Release: true}); err != nil {
		t.Fatalf("Finish() error = %v, want non-interactive dry-run release preview to succeed", err)
	}
}

func TestDiscardInteractiveDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initHotfixRepo(t)

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
	if progressTitle != "git sf hotfix discard" {
		t.Fatalf("progress title = %q, want %q", progressTitle, "git sf hotfix discard")
	}
	if progressBranch != "hotfix/test" {
		t.Fatalf("progress branch = %q, want %q", progressBranch, "hotfix/test")
	}
}

func TestDiscardClassicDryRunChecksRealGHAuth(t *testing.T) {
	repoDir := initHotfixRepo(t)
	installDiscardAuthFailureGH(t)

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

func TestStartDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initHotfixRepo(t)
	// Switch back to main so Start() can branch from a tag.
	runGit(t, repoDir, "checkout", "main")
	runGit(t, repoDir, "tag", "v0.1.0")

	var out bytes.Buffer
	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &out

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	if err := svc.Start("dry-test", StartOpts{}); err != nil {
		t.Fatalf("Start() error = %v, want dry-run preview to succeed", err)
	}
}

func TestPublishDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initHotfixRepo(t)
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

	if !strings.Contains(out.String(), "hotfix/test") {
		t.Fatalf("Publish() output = %q, want branch name resolved from real repo", out.String())
	}
}

func TestStartDraftPRPromptCancellationDoesNotCreateBranch(t *testing.T) {
	repoDir := initHotfixStartRepo(t)
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

	err := svc.Start("urgent-fix", StartOpts{DraftPR: true})
	if !errors.Is(err, promptErr) {
		t.Fatalf("Start() error = %v, want %v", err, promptErr)
	}

	branches := strings.Fields(runGit(t, repoDir, "branch", "--format=%(refname:short)"))
	for _, branch := range branches {
		if branch == "hotfix/urgent-fix" {
			t.Fatalf("Start() created %q before the PR prompt completed", branch)
		}
	}

	if current := runGit(t, repoDir, "rev-parse", "--abbrev-ref", "HEAD"); current != "main" {
		t.Fatalf("HEAD = %q, want %q after prompt cancellation", current, "main")
	}
}

func initHotfixStartRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	runGit(t, repoDir, "tag", "v0.1.0")

	return repoDir
}

func installDiscardAuthFailureGH(t *testing.T) {
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

func initHotfixReleaseRepo(t *testing.T) string {
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
	runGit(t, repoDir, "push", "origin", "main")
	runGit(t, repoDir, "tag", "v0.1.0")
	runGit(t, repoDir, "push", "origin", "v0.1.0")
	runGit(t, repoDir, "checkout", "-b", "hotfix/test")
	runGit(t, repoDir, "push", "-u", "origin", "hotfix/test")

	return repoDir
}

func installFinishReleaseGH(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo '{"number":123,"title":"Hotfix PR","state":"OPEN","url":"https://example.com/pr/123","isDraft":false}'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  case "$*" in
    *--required*) ;;
    *) echo "missing --required flag in: $*" >&2; exit 1 ;;
  esac
  echo '[{"name":"ci","state":"SUCCESS","bucket":"pass"}]'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}
