package release

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

// TestReleaseAnnotatedTagWhenMessageProvided verifies that Release() creates an
// annotated tag (via TagAnnotated) when a non-empty message is supplied.
func TestReleaseAnnotatedTagWhenMessageProvided(t *testing.T) {
	repoDir := initReleaseRepo(t)

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           config.Defaults(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	err := svc.Release("patch", "Release message")
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// The new tag should be v0.1.1 (patch bump from v0.1.0).
	tagOutput := runGit(t, repoDir, "tag", "-l", "v0.1.1")
	if tagOutput != "v0.1.1" {
		t.Fatalf("expected tag v0.1.1, got %q", tagOutput)
	}

	// Verify it is annotated: "git cat-file -t v0.1.1" should return "tag" (not "commit").
	objType := runGit(t, repoDir, "cat-file", "-t", "v0.1.1")
	if objType != "tag" {
		t.Fatalf("expected annotated tag (object type 'tag'), got %q", objType)
	}

	// Verify the tag message.
	tagMsg := runGit(t, repoDir, "tag", "-l", "--format=%(contents:subject)", "v0.1.1")
	if tagMsg != "Release message" {
		t.Fatalf("expected tag message %q, got %q", "Release message", tagMsg)
	}
}

// TestReleasePromptsForMessageWhenInteractive verifies that Release() invokes
// the message prompt when message is empty and ShouldPrompt() returns true,
// and that the returned message is used for an annotated tag.
func TestReleasePromptsForMessageWhenInteractive(t *testing.T) {
	repoDir := initReleaseRepo(t)

	promptCalled := false

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	// Provide "y" for the confirmation prompt.
	u.In = strings.NewReader("y\n")

	svc := &Service{
		Git:    git.New(r, repoDir),
		UI:     u,
		Config: config.Defaults(),
		RunMessagePrompt: func(tagName string) (string, error) {
			promptCalled = true
			if tagName != "v0.1.1" {
				t.Fatalf("prompt tagName = %q, want %q", tagName, "v0.1.1")
			}
			return "Message from prompt", nil
		},
	}

	err := svc.Release("patch", "")
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if !promptCalled {
		t.Fatal("expected RunMessagePrompt to be called, but it was not")
	}

	// The tag should be annotated because the prompt returned a non-empty message.
	objType := runGit(t, repoDir, "cat-file", "-t", "v0.1.1")
	if objType != "tag" {
		t.Fatalf("expected annotated tag (object type 'tag'), got %q", objType)
	}

	tagMsg := runGit(t, repoDir, "tag", "-l", "--format=%(contents:subject)", "v0.1.1")
	if tagMsg != "Message from prompt" {
		t.Fatalf("expected tag message %q, got %q", "Message from prompt", tagMsg)
	}
}

// TestReleaseLightweightTagWhenNonInteractive verifies that Release() creates a
// lightweight tag (via Tag, not TagAnnotated) when message is empty and the UI
// is non-interactive.
func TestReleaseLightweightTagWhenNonInteractive(t *testing.T) {
	repoDir := initReleaseRepo(t)

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = false
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		UI:     u,
		Config: config.Defaults(),
		RunMessagePrompt: func(tagName string) (string, error) {
			t.Fatal("RunMessagePrompt should not be called in non-interactive mode")
			return "", nil
		},
	}

	err := svc.Release("patch", "")
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// The new tag should be v0.1.1.
	tagOutput := runGit(t, repoDir, "tag", "-l", "v0.1.1")
	if tagOutput != "v0.1.1" {
		t.Fatalf("expected tag v0.1.1, got %q", tagOutput)
	}

	// Verify it is lightweight: "git cat-file -t v0.1.1" should return "commit" (not "tag").
	objType := runGit(t, repoDir, "cat-file", "-t", "v0.1.1")
	if objType != "commit" {
		t.Fatalf("expected lightweight tag (object type 'commit'), got %q", objType)
	}
}

func TestReleaseNonInteractiveDeclineAbortsWithoutTag(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var buf bytes.Buffer

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &buf
	u.Interactive = false
	u.In = strings.NewReader("n\n")

	svc := &Service{
		Git:    git.New(r, repoDir),
		UI:     u,
		Config: config.Defaults(),
		RunMessagePrompt: func(tagName string) (string, error) {
			t.Fatal("RunMessagePrompt should not be called in non-interactive mode")
			return "", nil
		},
	}

	err := svc.Release("patch", "")
	if err != nil {
		t.Fatalf("Release() error = %v, want nil when confirmation is declined", err)
	}
	if strings.Contains(runGit(t, repoDir, "tag", "-l", "v0.1.1"), "v0.1.1") {
		t.Fatal("release should not create a tag when confirmation is declined")
	}
	if !strings.Contains(buf.String(), "Confirm release? [y/N]") {
		t.Fatalf("Release() output = %q, want confirmation prompt", buf.String())
	}
	if !strings.Contains(buf.String(), "Aborted.") {
		t.Fatalf("Release() output = %q, want abort message", buf.String())
	}
}

// TestReleaseIgnoresOffMainHotfixTag verifies that Release() bumps from the
// latest tag reachable from main, not a globally-latest off-main hotfix tag.
func TestReleaseIgnoresOffMainHotfixTag(t *testing.T) {
	repoDir := initReleaseRepo(t)

	// Create a hotfix tag v0.2.0 on a side branch (not reachable from main).
	runGit(t, repoDir, "checkout", "-b", "hotfix/side")
	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "hotfix")
	runGit(t, repoDir, "tag", "v0.2.0")
	runGit(t, repoDir, "push", "origin", "v0.2.0")

	// Switch back to main and run release (patch bump).
	runGit(t, repoDir, "checkout", "main")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           config.Defaults(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.Release("patch", ""); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// Should bump from v0.1.0 (on main) → v0.1.1, NOT from v0.2.0 → v0.2.1.
	if strings.Contains(out.String(), "v0.2.1") {
		t.Fatalf("Release() bumped from off-main tag v0.2.0, output = %q", out.String())
	}
	if !strings.Contains(out.String(), "v0.1.1") {
		t.Fatalf("Release() output = %q, want v0.1.1 (bumped from v0.1.0 on main)", out.String())
	}
}

// initReleaseRepo creates a temp git repo with a bare remote, an initial commit
// on main, and an existing v0.1.0 tag, suitable for testing Release().
func initReleaseRepo(t *testing.T) string {
	t.Helper()

	// Create a bare "remote" repo.
	bareDir := t.TempDir()
	runGit(t, bareDir, "init", "--bare", "-b", "main")

	// Create a working repo that clones from the bare remote.
	parentDir := t.TempDir()
	repoDir := filepath.Join(parentDir, "work")
	runGit(t, parentDir, "clone", bareDir, "work")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	// Create an initial commit.
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")

	// Push to origin so fetch/sync checks pass.
	runGit(t, repoDir, "push", "origin", "main")

	// Create an existing tag and push it.
	runGit(t, repoDir, "tag", "v0.1.0")
	runGit(t, repoDir, "push", "origin", "v0.1.0")

	return repoDir
}

func TestPreviewReleaseHappyPath(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewRelease("patch", ""); err != nil {
		t.Fatalf("PreviewRelease() error = %v", err)
	}

	tagOutput := runGit(t, repoDir, "tag", "-l", "v0.1.1-beta.1")
	if tagOutput != "v0.1.1-beta.1" {
		t.Fatalf("expected tag v0.1.1-beta.1, got %q", tagOutput)
	}
}

func TestPreviewReleaseFirstTagNoStable(t *testing.T) {
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

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewRelease("patch", ""); err != nil {
		t.Fatalf("PreviewRelease() error = %v", err)
	}

	tagOutput := runGit(t, repoDir, "tag", "-l", "v0.1.0-beta.1")
	if tagOutput != "v0.1.0-beta.1" {
		t.Fatalf("expected tag v0.1.0-beta.1, got %q", tagOutput)
	}
}

func TestPreviewReleaseIncrementsCounter(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewRelease("patch", ""); err != nil {
		t.Fatalf("first PreviewRelease() error = %v", err)
	}
	if err := svc.PreviewRelease("patch", ""); err != nil {
		t.Fatalf("second PreviewRelease() error = %v", err)
	}

	tagOutput := runGit(t, repoDir, "tag", "-l", "v0.1.1-beta.2")
	if tagOutput != "v0.1.1-beta.2" {
		t.Fatalf("expected tag v0.1.1-beta.2, got %q", tagOutput)
	}
}

func TestPreviewReleaseAnnotatedTag(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewRelease("patch", "Preview message"); err != nil {
		t.Fatalf("PreviewRelease() error = %v", err)
	}

	objType := runGit(t, repoDir, "cat-file", "-t", "v0.1.1-beta.1")
	if objType != "tag" {
		t.Fatalf("expected annotated tag, got object type %q", objType)
	}
}

func TestPreviewReleaseLightweightTag(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewRelease("patch", ""); err != nil {
		t.Fatalf("PreviewRelease() error = %v", err)
	}

	objType := runGit(t, repoDir, "cat-file", "-t", "v0.1.1-beta.1")
	if objType != "commit" {
		t.Fatalf("expected lightweight tag, got object type %q", objType)
	}
}

func TestPreviewReleaseNotOnMainErrors(t *testing.T) {
	repoDir := initReleaseRepo(t)
	r := runner.NewRunner(false, false)
	g := git.New(r, repoDir)
	if err := g.CreateBranch("feature/test"); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              g,
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	err := svc.PreviewRelease("patch", "")
	if err == nil {
		t.Fatal("expected error when not on main")
	}
	if !strings.Contains(err.Error(), "must be on main") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStableReleaseIgnoresPreviewTags(t *testing.T) {
	repoDir := initReleaseRepo(t)

	r := runner.NewRunner(false, false)

	runGit(t, repoDir, "tag", "v0.1.1-beta.1")
	runGit(t, repoDir, "push", "origin", "v0.1.1-beta.1")

	var out bytes.Buffer
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           config.Defaults(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.Release("patch", ""); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if !strings.Contains(out.String(), "v0.1.1") {
		t.Fatalf("Release() output = %q, want v0.1.1", out.String())
	}
}

// TestRecoveryPushesLocalOnlyTagOnRerun verifies that rerunning
// PreviewReleaseCore after a failed push recovers the existing local tag.
func TestRecoveryPushesLocalOnlyTagOnRerun(t *testing.T) {
	repoDir := initReleaseRepo(t)

	// Create a local preview tag but do NOT push it (simulates failed push).
	runGit(t, repoDir, "tag", "v0.1.1-beta.1")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewReleaseCore("patch", ""); err != nil {
		t.Fatalf("PreviewReleaseCore() error = %v", err)
	}

	if !strings.Contains(out.String(), "Found unpushed local tag v0.1.1-beta.1") {
		t.Fatalf("expected recovery message, got %q", out.String())
	}

	// Verify the tag was pushed to origin.
	remoteRefs := runGit(t, repoDir, "ls-remote", "--tags", "origin", "v0.1.1-beta.1")
	if !strings.Contains(remoteRefs, "v0.1.1-beta.1") {
		t.Fatalf("expected v0.1.1-beta.1 on remote, got %q", remoteRefs)
	}
}

// TestRecoverySkipsStaleCounterBelowRemote verifies that a local-only tag
// with a counter <= the highest published remote counter is not recovered.
func TestRecoverySkipsStaleCounterBelowRemote(t *testing.T) {
	repoDir := initReleaseRepo(t)

	// Push beta.3 to remote (the highest published counter).
	runGit(t, repoDir, "tag", "v0.1.1-beta.3")
	runGit(t, repoDir, "push", "origin", "v0.1.1-beta.3")

	// Leave beta.2 as local-only (stale leftover).
	runGit(t, repoDir, "tag", "v0.1.1-beta.2")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewReleaseCore("patch", ""); err != nil {
		t.Fatalf("PreviewReleaseCore() error = %v", err)
	}

	// Should NOT recover beta.2 — it should create beta.4 (next after beta.3).
	if strings.Contains(out.String(), "Found unpushed local tag") {
		t.Fatalf("should not recover stale tag, got %q", out.String())
	}
	tagOutput := runGit(t, repoDir, "tag", "-l", "v0.1.1-beta.4")
	if tagOutput != "v0.1.1-beta.4" {
		t.Fatalf("expected tag v0.1.1-beta.4, got %q", tagOutput)
	}
}

// TestRecoveryPicksHighestCounter verifies that when multiple local-only
// tags exist on HEAD, recovery selects the one with the highest counter.
func TestRecoveryPicksHighestCounter(t *testing.T) {
	repoDir := initReleaseRepo(t)

	// Create two local-only preview tags (neither pushed).
	runGit(t, repoDir, "tag", "v0.1.1-beta.1")
	runGit(t, repoDir, "tag", "v0.1.1-beta.2")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewReleaseCore("patch", ""); err != nil {
		t.Fatalf("PreviewReleaseCore() error = %v", err)
	}

	if !strings.Contains(out.String(), "Found unpushed local tag v0.1.1-beta.2") {
		t.Fatalf("expected recovery of beta.2 (highest), got %q", out.String())
	}
}

// TestRecoveryReAnnotatesWhenMessageProvided verifies that recovery with a
// non-empty message produces an annotated tag instead of the original lightweight one.
func TestRecoveryReAnnotatesWhenMessageProvided(t *testing.T) {
	repoDir := initReleaseRepo(t)

	// Create a lightweight local-only preview tag.
	runGit(t, repoDir, "tag", "v0.1.1-beta.1")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewReleaseCore("patch", "Recovery message"); err != nil {
		t.Fatalf("PreviewReleaseCore() error = %v", err)
	}

	// Should be annotated now.
	objType := runGit(t, repoDir, "cat-file", "-t", "v0.1.1-beta.1")
	if objType != "tag" {
		t.Fatalf("expected annotated tag after recovery, got object type %q", objType)
	}

	tagMsg := runGit(t, repoDir, "tag", "-l", "--format=%(contents:subject)", "v0.1.1-beta.1")
	if tagMsg != "Recovery message" {
		t.Fatalf("expected tag message %q, got %q", "Recovery message", tagMsg)
	}
}

// TestRecoveryIgnoresOffBranchRemoteTag verifies that an off-branch remote
// preview tag with a higher counter does not suppress recovery of a valid
// local-only tag on HEAD.
func TestRecoveryIgnoresOffBranchRemoteTag(t *testing.T) {
	repoDir := initReleaseRepo(t)

	// Create beta.5 on a side branch and push it to remote.
	runGit(t, repoDir, "checkout", "-b", "side")
	if err := os.WriteFile(filepath.Join(repoDir, "side.txt"), []byte("side\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "side commit")
	runGit(t, repoDir, "tag", "v0.1.1-beta.5")
	runGit(t, repoDir, "push", "origin", "v0.1.1-beta.5")

	// Switch back to main and create a local-only beta.4.
	runGit(t, repoDir, "checkout", "main")
	runGit(t, repoDir, "tag", "v0.1.1-beta.4")

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           previewConfig(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.PreviewReleaseCore("patch", ""); err != nil {
		t.Fatalf("PreviewReleaseCore() error = %v", err)
	}

	// beta.4 should be recovered — the off-branch beta.5 is not reachable
	// from main and should not suppress recovery.
	if !strings.Contains(out.String(), "Found unpushed local tag v0.1.1-beta.4") {
		t.Fatalf("expected recovery of beta.4, got %q", out.String())
	}
}

func previewConfig() config.Config {
	cfg := config.Defaults()
	cfg.PrereleaseSuffix = "beta"
	cfg.DefaultPrereleaseBump = "patch"
	return cfg
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
