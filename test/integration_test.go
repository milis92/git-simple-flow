//go:build integration

// test/integration_test.go
package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles the git-sf binary into a temp directory and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "git-sf")
	// Resolve the module root (one level up from the test directory)
	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("failed to resolve module root: %s", err)
	}
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return binary
}

// setupRepo creates a temporary git repo with an initial commit on the main branch.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", c, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write README: %s", err)
	}
	for _, c := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", c, err, out)
		}
	}

	return dir
}

// setupRepoWithRemote creates a temporary git repo backed by a local bare clone as "origin".
// This allows git pull/push to work without a network connection.
func setupRepoWithRemote(t *testing.T) string {
	t.Helper()

	// First create a bare repo to serve as origin
	bareDir := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", bareDir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %s\n%s", err, out)
	}

	// Clone the bare repo to get a working copy
	workDir := t.TempDir()
	if out, err := exec.Command("git", "clone", bareDir, workDir).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %s\n%s", err, out)
	}

	// Configure identity in the working copy
	for _, c := range [][]string{
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", c, err, out)
		}
	}

	// Create an initial commit and push it so origin/main exists
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write README: %s", err)
	}
	for _, c := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
		{"git", "push", "-u", "origin", "main"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", c, err, out)
		}
	}

	return workDir
}

func TestFeatureStartDryRun(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	cmd := exec.Command(binary, "feature", "start", "test-feature", "--dry-run")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "feature/test-feature") {
		t.Errorf("expected branch name in output, got: %s", output)
	}

	// Verify no branch was actually created
	branchCmd := exec.Command("git", "branch")
	branchCmd.Dir = dir
	branchOut, _ := branchCmd.Output()
	if strings.Contains(string(branchOut), "feature/test-feature") {
		t.Error("dry-run should not create branch")
	}
}

func TestFeatureStartActual(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepoWithRemote(t)

	cmd := exec.Command(binary, "feature", "start", "my-feature")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}

	// Verify branch was created and checked out
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = dir
	branchOut, _ := branchCmd.Output()
	branch := strings.TrimSpace(string(branchOut))
	if branch != "feature/my-feature" {
		t.Errorf("expected branch feature/my-feature, got %q", branch)
	}
}

func TestReleaseFromNonMain(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	// Create and switch to a feature branch
	if out, err := exec.Command("git", "-C", dir, "checkout", "-b", "feature/test").CombinedOutput(); err != nil {
		t.Fatalf("git checkout failed: %s\n%s", err, out)
	}

	cmd := exec.Command(binary, "release")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when releasing from non-main branch")
	}
	if !strings.Contains(string(out), "must be on") {
		t.Errorf("expected 'must be on main' error, got: %s", out)
	}
}

func TestFeatureStartDirtyTree(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	// Dirty the working tree with an untracked file
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to write dirty file: %s", err)
	}

	cmd := exec.Command(binary, "feature", "start", "test")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error with dirty tree")
	}
	if !strings.Contains(string(out), "not clean") {
		t.Errorf("expected 'not clean' error, got: %s", out)
	}
}

func TestHotfixStartFromTag(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	// Create a version tag
	if out, err := exec.Command("git", "-C", dir, "tag", "v1.0.0").CombinedOutput(); err != nil {
		t.Fatalf("git tag failed: %s\n%s", err, out)
	}

	cmd := exec.Command(binary, "hotfix", "start", "fix-crash")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}

	// Verify the hotfix branch was created and checked out
	branchCmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, _ := branchCmd.Output()
	branch := strings.TrimSpace(string(branchOut))
	if branch != "hotfix/fix-crash" {
		t.Errorf("expected branch hotfix/fix-crash, got %q", branch)
	}
}

func TestHotfixStartNoTags(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	cmd := exec.Command(binary, "hotfix", "start", "fix")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when no tags exist")
	}
	if !strings.Contains(string(out), "no tags found") {
		t.Errorf("expected 'no tags found' error, got: %s", out)
	}
}

func TestConfigInit(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	cmd := exec.Command(binary, "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), ".sfconfig.yml") {
		t.Errorf("expected config file path in output, got: %s", out)
	}

	// Verify file was created
	configPath := filepath.Join(dir, ".sfconfig.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected .sfconfig.yml to be created")
	}
}

func TestConfigShow(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	cmd := exec.Command(binary, "config")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "main_branch") {
		t.Errorf("expected main_branch in config output, got: %s", output)
	}
	if !strings.Contains(output, "squash") {
		t.Errorf("expected default merge_strategy 'squash' in output, got: %s", output)
	}
}

func TestCompletionBash(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "completion", "bash")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}
	if len(out) == 0 {
		t.Error("expected non-empty bash completion script")
	}
}

func TestStatusOnMain(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepo(t)

	cmd := exec.Command(binary, "status")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "main") {
		t.Errorf("expected 'main' in status output, got: %s", out)
	}
}

// runGitCmd runs a git command in dir, returns trimmed stdout, and fatals on error.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestReleaseFirstRelease(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepoWithRemote(t)

	// Run release without --dry-run so that branch detection works correctly.
	// Provide "n\n" on stdin so the confirmation prompt aborts without tagging.
	// The command prints the proposed version before asking for confirmation.
	cmd := exec.Command(binary, "release")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("n\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "v0.1.0") {
		t.Errorf("expected v0.1.0 for first release, got: %s", out)
	}
}

func TestReleasePreview(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepoWithRemote(t)

	// Create a stable tag first
	runGitCmd(t, dir, "tag", "v0.1.0")
	runGitCmd(t, dir, "push", "origin", "v0.1.0")

	// Run release preview
	cmd := exec.Command(binary, "release", "preview", "--scope", "patch", "--yes")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release preview failed: %s\n%s", err, out)
	}

	// Verify tag was created
	tagOut := runGitCmd(t, dir, "tag", "-l", "v0.1.1-beta.1")
	if tagOut != "v0.1.1-beta.1" {
		t.Errorf("expected tag v0.1.1-beta.1, got %q", tagOut)
	}

	// Verify tag was pushed to remote
	remoteOut := runGitCmd(t, dir, "ls-remote", "--tags", "origin", "v0.1.1-beta.1")
	if !strings.Contains(remoteOut, "v0.1.1-beta.1") {
		t.Errorf("tag not found on remote, got %q", remoteOut)
	}
}

func TestReleasePreviewIncrementsCounter(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepoWithRemote(t)

	runGitCmd(t, dir, "tag", "v0.1.0")
	runGitCmd(t, dir, "push", "origin", "v0.1.0")

	// First preview
	cmd := exec.Command(binary, "release", "preview", "--scope", "patch", "--yes")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("first release preview failed: %s\n%s", err, out)
	}

	// Second preview
	cmd = exec.Command(binary, "release", "preview", "--scope", "patch", "--yes")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("second release preview failed: %s\n%s", err, out)
	}

	// Verify counter incremented
	tagOut := runGitCmd(t, dir, "tag", "-l", "v0.1.1-beta.2")
	if tagOut != "v0.1.1-beta.2" {
		t.Errorf("expected tag v0.1.1-beta.2, got %q", tagOut)
	}
}

func TestReleasePreviewThenStable(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepoWithRemote(t)

	runGitCmd(t, dir, "tag", "v0.1.0")
	runGitCmd(t, dir, "push", "origin", "v0.1.0")

	// Create a preview tag
	cmd := exec.Command(binary, "release", "preview", "--scope", "patch", "--yes")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("release preview failed: %s\n%s", err, out)
	}

	// Now create a stable release — should bump from v0.1.0, not from the preview tag
	cmd = exec.Command(binary, "release", "patch", "--yes")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stable release failed: %s\n%s", err, out)
	}

	// Verify stable tag was created correctly
	tagOut := runGitCmd(t, dir, "tag", "-l", "v0.1.1")
	if tagOut != "v0.1.1" {
		t.Errorf("expected stable tag v0.1.1, got %q", tagOut)
	}
}

func TestReleasePreviewFreshRepo(t *testing.T) {
	binary := buildBinary(t)
	dir := setupRepoWithRemote(t)

	// No existing tags — first preview should be v0.1.0-beta.1
	cmd := exec.Command(binary, "release", "preview", "--yes")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release preview failed: %s\n%s", err, out)
	}

	tagOut := runGitCmd(t, dir, "tag", "-l", "v0.1.0-beta.1")
	if tagOut != "v0.1.0-beta.1" {
		t.Errorf("expected tag v0.1.0-beta.1, got %q", tagOut)
	}
}
