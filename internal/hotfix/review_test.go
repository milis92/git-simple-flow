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

func TestFinishReleaseDoesNotOverwriteRemoteOnlyHotfixChanges(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	bareDir := runGit(t, repoDir, "remote", "get-url", "origin")
	installFinishReleaseMergeFailureGH(t)

	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("local change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "local hotfix")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	parentDir := t.TempDir()
	collaboratorDir := filepath.Join(parentDir, "collaborator")
	runGit(t, parentDir, "clone", bareDir, "collaborator")
	runGit(t, collaboratorDir, "config", "user.name", "Collaborator")
	runGit(t, collaboratorDir, "config", "user.email", "collab@example.com")
	runGit(t, collaboratorDir, "checkout", "hotfix/test")
	f, err := os.OpenFile(filepath.Join(collaboratorDir, "fix.txt"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("collab change\n"); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	runGit(t, collaboratorDir, "add", ".")
	runGit(t, collaboratorDir, "commit", "-m", "collab hotfix")
	runGit(t, collaboratorDir, "push", "origin", "hotfix/test")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err = svc.Finish(FinishOpts{Release: true})
	if err == nil {
		t.Fatal("Finish() error = nil, want error due to remote divergence")
	}

	remoteFix := runGit(t, bareDir, "show", "hotfix/test:fix.txt")
	if !strings.Contains(remoteFix, "collab change") {
		t.Fatalf("remote hotfix branch lost collaborator change after release attempt: %q", remoteFix)
	}
}

func TestFinishReleaseDoesNotTagUnreleasedMainChanges(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	installFinishReleaseGH(t)

	runGit(t, repoDir, "checkout", "main")
	if err := os.WriteFile(filepath.Join(repoDir, "unreleased.txt"), []byte("unreleased\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "unreleased feature")
	runGit(t, repoDir, "push", "origin", "main")

	runGit(t, repoDir, "checkout", "hotfix/test")
	runGit(t, repoDir, "merge", "--no-edit", "main")
	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("critical fix\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "hotfix change")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err := svc.Finish(FinishOpts{Release: true})
	if err == nil {
		t.Fatal("Finish() error = nil, want error due to unreleased main commits in hotfix branch")
	}

	// Verify no tag was created
	tags := runGit(t, repoDir, "tag", "-l", "v0.1.1")
	if tags == "v0.1.1" {
		tree := runGit(t, repoDir, "ls-tree", "-r", "--name-only", "v0.1.1")
		if strings.Contains(tree, "unreleased.txt") {
			t.Fatalf("release tag v0.1.1 includes unreleased main content: %q", tree)
		}
	}
}

func TestFinishReleaseRejectsOriginMainContaminationWhenLocalMainIsStale(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	bareDir := runGit(t, repoDir, "remote", "get-url", "origin")
	installFinishReleaseGH(t)

	// Push an unreleased commit to origin/main from a second clone,
	// so the local main stays stale.
	parentDir := t.TempDir()
	pusherDir := filepath.Join(parentDir, "pusher")
	runGit(t, parentDir, "clone", bareDir, "pusher")
	runGit(t, pusherDir, "config", "user.name", "Pusher")
	runGit(t, pusherDir, "config", "user.email", "pusher@example.com")
	if err := os.WriteFile(filepath.Join(pusherDir, "unreleased.txt"), []byte("unreleased\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "unreleased feature")
	runGit(t, pusherDir, "push", "origin", "main")

	// On the hotfix branch, merge origin/main (fetched), which includes
	// the unreleased commit that local main doesn't know about.
	runGit(t, repoDir, "fetch", "origin")
	runGit(t, repoDir, "merge", "--no-edit", "origin/main")
	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "hotfix change")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err := svc.Finish(FinishOpts{Release: true})
	if err == nil {
		t.Fatal("Finish() error = nil, want error due to unreleased origin/main commits in hotfix branch")
	}
	if !strings.Contains(err.Error(), "unreleased main commits") {
		t.Fatalf("Finish() error = %q, want 'unreleased main commits' message", err.Error())
	}
}

func TestFinishReleaseAcceptsAnnotatedLatestTag(t *testing.T) {
	// Set up repo with an annotated tag instead of lightweight
	bareDir := t.TempDir()
	runGit(t, bareDir, "init", "--bare", "-b", "main")

	parentDir := t.TempDir()
	repoDir := filepath.Join(parentDir, "work")
	runGit(t, parentDir, "clone", bareDir, "work")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	runGit(t, repoDir, "push", "origin", "main")
	// Create annotated tag (not lightweight)
	runGit(t, repoDir, "tag", "-a", "v0.1.0", "-m", "Release v0.1.0")
	runGit(t, repoDir, "push", "origin", "v0.1.0")
	runGit(t, repoDir, "checkout", "-b", "hotfix/test")

	// Add a fix commit
	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "the fix")
	runGit(t, repoDir, "push", "-u", "origin", "hotfix/test")

	installFinishReleaseGH(t)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err := svc.Finish(FinishOpts{Release: true})
	if err != nil {
		t.Fatalf("Finish() error = %v, want success with annotated tag", err)
	}
	if !strings.Contains(out.String(), "v0.1.1") {
		t.Errorf("output = %q, want mention of v0.1.1", out.String())
	}
}

func TestFinishReleaseDoesNotPushTagWhenMergeFails(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	installFinishReleaseMergeFailureGH(t)

	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "the fix")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	bareDir := runGit(t, repoDir, "remote", "get-url", "origin")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	err := svc.Finish(FinishOpts{Release: true})
	if err == nil {
		t.Fatal("Finish() error = nil, want merge failure")
	}

	// Verify no tag was pushed to remote
	remoteTags, _ := runner.NewRunner(false, false).Run("git", "-C", bareDir, "tag", "-l", "v0.1.1")
	if strings.Contains(remoteTags, "v0.1.1") {
		t.Fatalf("tag v0.1.1 was pushed to remote despite merge failure")
	}
}

func TestFinishReleaseRetryKeepsPostSquashHeadStable(t *testing.T) {
	repoDir := initHotfixReleaseRepo(t)
	installFinishReleasePostSquashChecksPendingGH(t)

	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "first fix")
	if err := os.WriteFile(filepath.Join(repoDir, "fix-2.txt"), []byte("fix again\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "second fix")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	t.Setenv("GIT_AUTHOR_DATE", "2026-03-29T10:00:00Z")
	t.Setenv("GIT_COMMITTER_DATE", "2026-03-29T10:00:00Z")
	err := svc.Finish(FinishOpts{Release: true})
	if err == nil {
		t.Fatal("first Finish() error = nil, want merge blocked on post-squash checks")
	}
	firstRetryHead := runGit(t, repoDir, "rev-parse", "HEAD")

	t.Setenv("GIT_AUTHOR_DATE", "2026-03-29T10:00:10Z")
	t.Setenv("GIT_COMMITTER_DATE", "2026-03-29T10:00:10Z")
	err = svc.Finish(FinishOpts{Release: true})
	if err == nil {
		t.Fatal("second Finish() error = nil, want merge to remain blocked")
	}
	secondRetryHead := runGit(t, repoDir, "rev-parse", "HEAD")

	if firstRetryHead != secondRetryHead {
		t.Fatalf("retry rewrote post-squash head from %s to %s; rerunning finish invalidates the checks it is waiting on", firstRetryHead, secondRetryHead)
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

func installFinishReleaseMergeFailureGH(t *testing.T) {
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
  echo "merge blocked" >&2
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

func installFinishReleasePostSquashChecksPendingGH(t *testing.T) {
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
  echo "required status check \"ci\" is expected on the head SHA before merge" >&2
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
