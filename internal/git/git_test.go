package git

import (
	"os"
	"testing"

	"github.com/nickssmallpdf/git-sf/internal/exec"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	r := exec.NewRunner(false, false)
	cmds := [][]string{
		{"git", "-C", dir, "init"},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		if _, err := r.Run(c[0], c[1:]...); err != nil {
			t.Fatal(err)
		}
	}
	f, _ := os.Create(dir + "/README.md")
	f.WriteString("init")
	f.Close()
	r.Run("git", "-C", dir, "add", ".")
	r.Run("git", "-C", dir, "commit", "-m", "init")
	return dir
}

func TestCurrentBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(exec.NewRunner(false, false), dir)
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
	g := New(exec.NewRunner(false, false), dir)
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
	g := New(exec.NewRunner(false, false), dir)
	clean, err := g.IsClean()
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Error("expected clean working tree")
	}
	os.WriteFile(dir+"/dirty.txt", []byte("dirty"), 0644)
	clean, _ = g.IsClean()
	if clean {
		t.Error("expected dirty working tree")
	}
}

func TestLatestTag(t *testing.T) {
	dir := setupTestRepo(t)
	r := exec.NewRunner(false, false)
	g := New(r, dir)
	_, err := g.LatestTag("v")
	if err == nil {
		t.Error("expected error when no tags exist")
	}
	r.Run("git", "-C", dir, "tag", "v1.0.0")
	os.WriteFile(dir+"/f2.txt", []byte("x"), 0644)
	r.Run("git", "-C", dir, "add", ".")
	r.Run("git", "-C", dir, "commit", "-m", "second")
	r.Run("git", "-C", dir, "tag", "v1.1.0")
	tag, err := g.LatestTag("v")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.1.0" {
		t.Errorf("LatestTag() = %q, want %q", tag, "v1.1.0")
	}
}
