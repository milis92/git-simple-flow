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

func TestReleaseNonInteractiveRequiresYes(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var buf bytes.Buffer

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &buf
	u.Interactive = false

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
	if err == nil {
		t.Fatal("Release() error = nil, want confirmation-required error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("Release() error = %q, want guidance to rerun with --yes", err.Error())
	}
	if strings.Contains(runGit(t, repoDir, "tag", "-l", "v0.1.1"), "v0.1.1") {
		t.Fatal("release should not create a tag when confirmation is unavailable")
	}
	if !strings.Contains(buf.String(), "skipped (non-interactive; rerun with --yes)") {
		t.Fatalf("Release() output = %q, want non-interactive skip message", buf.String())
	}
}

// initReleaseRepo creates a temp git repo with a bare remote, an initial commit
// on main, and an existing v0.1.0 tag, suitable for testing Release().
func initReleaseRepo(t *testing.T) string {
	t.Helper()

	// Create a bare "remote" repo.
	bareDir := t.TempDir()
	runGit(t, bareDir, "init", "--bare")

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
