package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/version"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	r := runner.NewRunner(false, false)
	cmds := [][]string{
		{"git", "-C", dir, "init", "-b", "main"},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		if _, err := r.Run(c[0], c[1:]...); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Create(dir + "/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("init"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := r.Run("git", "-C", dir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "commit", "-m", "init"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCurrentBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(runner.NewRunner(false, false), dir)
	branch, err := g.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" && branch != "master" {
		t.Errorf("CurrentBranch() = %q, want main or master", branch)
	}
}

func TestCreateAndCheckoutBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(runner.NewRunner(false, false), dir)
	err := g.CreateBranch("feature/test")
	if err != nil {
		t.Fatal(err)
	}
	branch, _ := g.CurrentBranch()
	if branch != "feature/test" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "feature/test")
	}
}

func TestIsClean(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(runner.NewRunner(false, false), dir)
	clean, err := g.IsClean()
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Error("expected clean working tree")
	}
	if err := os.WriteFile(dir+"/dirty.txt", []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}
	clean, _ = g.IsClean()
	if clean {
		t.Error("expected dirty working tree")
	}
}

func TestLatestTag(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)
	_, err := g.LatestTag("v")
	if err == nil {
		t.Error("expected error when no tags exist")
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/f2.txt", []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "commit", "-m", "second"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.1.0"); err != nil {
		t.Fatal(err)
	}
	tag, err := g.LatestTag("v")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.1.0" {
		t.Errorf("LatestTag() = %q, want %q", tag, "v1.1.0")
	}
}

func TestLatestTagOnBranch(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	// Tag v1.0.0 on the initial commit (main).
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.0"); err != nil {
		t.Fatal(err)
	}

	// Create a side branch with its own tag v1.0.1.
	if err := g.CreateBranch("hotfix/side"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/fix.txt", []byte("fix"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "commit", "-m", "hotfix"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1"); err != nil {
		t.Fatal(err)
	}

	// Switch back to main. Global LatestTag sees v1.0.1, but
	// LatestTagOnBranch scoped to main should see only v1.0.0.
	if err := g.Checkout("main"); err != nil {
		t.Fatal(err)
	}

	globalTag, err := g.LatestTag("v")
	if err != nil {
		t.Fatal(err)
	}
	if globalTag != "v1.0.1" {
		t.Errorf("LatestTag() = %q, want %q", globalTag, "v1.0.1")
	}

	scopedTag, err := g.LatestTagOnBranch("v", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if scopedTag != "v1.0.0" {
		t.Errorf("LatestTagOnBranch(HEAD) = %q, want %q (should not see off-branch v1.0.1)", scopedTag, "v1.0.0")
	}
}

func TestMergeBase(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	// Record the initial commit SHA
	initSHA, err := r.Run("git", "-C", dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	// Create a branch and add a commit
	if err := g.CreateBranch("hotfix/test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/fix.txt", []byte("fix"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "commit", "-m", "hotfix commit"); err != nil {
		t.Fatal(err)
	}

	base, err := g.MergeBase("main", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if base != initSHA {
		t.Errorf("MergeBase() = %q, want %q", base, initSHA)
	}
}

func TestForcePush(t *testing.T) {
	// Set up a bare remote + cloned working copy
	bareDir := t.TempDir()
	r := runner.NewRunner(false, false)
	if _, err := r.Run("git", "init", "--bare", "-b", "main", bareDir); err != nil {
		t.Fatal(err)
	}

	parentDir := t.TempDir()
	workDir := filepath.Join(parentDir, "work")
	if _, err := r.Run("git", "clone", bareDir, workDir); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}

	// Initial commit + push
	if err := os.WriteFile(filepath.Join(workDir, "f.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "commit", "-m", "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "push", "-u", "origin", "main"); err != nil {
		t.Fatal(err)
	}

	// Create a branch, push it, then rewrite history
	g := New(r, workDir)
	if err := g.CreateBranch("hotfix/test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "fix.txt"), []byte("fix"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "commit", "-m", "original"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", workDir, "push", "-u", "origin", "hotfix/test"); err != nil {
		t.Fatal(err)
	}

	// Amend the commit (rewrite history)
	if _, err := r.Run("git", "-C", workDir, "commit", "--amend", "-m", "amended"); err != nil {
		t.Fatal(err)
	}

	// Force push should succeed
	if err := g.ForcePush("hotfix/test"); err != nil {
		t.Fatalf("ForcePush() error = %v", err)
	}

	// Verify remote has the amended message
	remoteMsg, err := r.Run("git", "-C", bareDir, "log", "-1", "--format=%s", "hotfix/test")
	if err != nil {
		t.Fatal(err)
	}
	if remoteMsg != "amended" {
		t.Errorf("remote commit message = %q, want %q", remoteMsg, "amended")
	}
}

func TestLatestTagSkipsPrerelease(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.2"); err != nil {
		t.Fatal(err)
	}

	tag, err := g.LatestTag("v")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.0.0" {
		t.Errorf("LatestTag() = %q, want %q (should skip prerelease tags)", tag, "v1.0.0")
	}
}

func TestLatestTagOnBranchSkipsPrerelease(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.1"); err != nil {
		t.Fatal(err)
	}

	tag, err := g.LatestTagOnBranch("v", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.0.0" {
		t.Errorf("LatestTagOnBranch() = %q, want %q (should skip prerelease)", tag, "v1.0.0")
	}
}

func TestResetSoftAndCommitWithMessage(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	// Record the initial commit SHA
	initSHA, err := r.Run("git", "-C", dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	// Create two more commits
	for i, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(dir+"/"+name, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Run("git", "-C", dir, "add", "."); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Run("git", "-C", dir, "commit", "-m", fmt.Sprintf("commit %d", i+1)); err != nil {
			t.Fatal(err)
		}
	}

	// Soft reset to the initial commit — changes should be staged
	if err := g.ResetSoft(initSHA); err != nil {
		t.Fatal(err)
	}

	// HEAD should now be the initial commit
	headSHA, err := r.Run("git", "-C", dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if headSHA != initSHA {
		t.Errorf("after ResetSoft, HEAD = %q, want %q", headSHA, initSHA)
	}

	// Commit the staged changes as a squashed commit
	if err := g.CommitWithMessage("squashed commit"); err != nil {
		t.Fatal(err)
	}

	// Verify there are exactly 2 commits (init + squashed)
	logOut, err := r.Run("git", "-C", dir, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if logOut != "2" {
		t.Errorf("commit count = %q, want %q", logOut, "2")
	}

	// Verify the squashed commit message
	msg, err := r.Run("git", "-C", dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	if msg != "squashed commit" {
		t.Errorf("commit message = %q, want %q", msg, "squashed commit")
	}
}

func TestLatestPreviewTag(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	target := version.Version{Major: 1, Minor: 0, Patch: 1}

	// No preview tags yet
	tag, err := g.LatestPreviewTag("v", "beta", "HEAD", target)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "" {
		t.Errorf("LatestPreviewTag() = %q, want empty when no tags exist", tag)
	}

	// Add preview tags
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.2"); err != nil {
		t.Fatal(err)
	}

	tag, err = g.LatestPreviewTag("v", "beta", "HEAD", target)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.0.1-beta.3" {
		t.Errorf("LatestPreviewTag() = %q, want %q", tag, "v1.0.1-beta.3")
	}
}

func TestLatestPreviewTagIgnoresStableAndWrongSuffix(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	target := version.Version{Major: 1, Minor: 0, Patch: 1}

	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-rc.5"); err != nil {
		t.Fatal(err)
	}

	tag, err := g.LatestPreviewTag("v", "beta", "HEAD", target)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "" {
		t.Errorf("LatestPreviewTag() = %q, want empty (no beta tags)", tag)
	}
}

func TestLatestPreviewTagIgnoresDifferentTarget(t *testing.T) {
	dir := setupTestRepo(t)
	r := runner.NewRunner(false, false)
	g := New(r, dir)

	if _, err := r.Run("git", "-C", dir, "tag", "v1.0.1-beta.1"); err != nil {
		t.Fatal(err)
	}

	target := version.Version{Major: 1, Minor: 0, Patch: 2}
	tag, err := g.LatestPreviewTag("v", "beta", "HEAD", target)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "" {
		t.Errorf("LatestPreviewTag() = %q, want empty (wrong target)", tag)
	}
}
