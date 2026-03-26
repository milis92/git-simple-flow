package workflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestFinishWorkflowBlocksTimedOutChecksBeforeMerge(t *testing.T) {
	repoDir := initWorkflowRepo(t)
	mergeMarker := filepath.Join(t.TempDir(), "merged")
	installWorkflowGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  echo '[{"name":"integration","status":"completed","conclusion":"timed_out"}]'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  touch "$MERGE_MARKER"
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`, mergeMarker)

	r := runner.NewRunner(false, false)
	wf := FinishWorkflow(git.New(r, repoDir), gh.New(r), "feature/test", "main", "squash", false)

	err := wf(context.Background(), ui.StepCallbacks{
		Start: func() {},
		Done:  func() {},
		Fail:  func(string) {},
	})
	if err == nil {
		t.Fatal("FinishWorkflow() error = nil, want timed_out checks to block merge")
	}
	want := "PR checks failed: integration (use --force to override)"
	if err.Error() != want {
		t.Fatalf("FinishWorkflow() error = %q, want %q", err.Error(), want)
	}
	if _, statErr := os.Stat(mergeMarker); !os.IsNotExist(statErr) {
		t.Fatalf("merge marker should not exist when checks fail, stat err = %v", statErr)
	}
}

func TestDiscardWorkflowDoesNotMarkSkippedPRCloseAsFailure(t *testing.T) {
	repoDir := initWorkflowRepoWithRemoteBranch(t, "feature/test")
	installWorkflowGH(t, `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  echo "not logged in" >&2
  exit 1
fi
echo "unexpected gh command: $*" >&2
exit 1
`, "")

	r := runner.NewRunner(false, false)
	failMsgs := make([]string, 0, 1)
	wf := DiscardWorkflow(git.New(r, repoDir), gh.New(r), "feature/test", "main", "")

	err := wf(context.Background(), ui.StepCallbacks{
		Start: func() {},
		Done:  func() {},
		Fail: func(msg string) {
			failMsgs = append(failMsgs, msg)
		},
	})
	if err != nil {
		t.Fatalf("DiscardWorkflow() error = %v, want nil", err)
	}
	if len(failMsgs) != 0 {
		t.Fatalf("DiscardWorkflow() should not mark skipped PR close as failed, got %v", failMsgs)
	}
}

func installWorkflowGH(t *testing.T, script, mergeMarker string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	if mergeMarker != "" {
		t.Setenv("MERGE_MARKER", mergeMarker)
	}
}

func initWorkflowRepo(t *testing.T) string {
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

	return repoDir
}

func initWorkflowRepoWithRemoteBranch(t *testing.T, branch string) string {
	t.Helper()

	bareDir := t.TempDir()
	runGit(t, bareDir, "init", "--bare")

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
	runGit(t, repoDir, "checkout", "-b", branch)
	runGit(t, repoDir, "push", "-u", "origin", branch)

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
