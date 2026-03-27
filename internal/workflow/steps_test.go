package workflow

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
  case "$*" in
    *--required*) ;;
    *) echo "missing --required flag in: $*" >&2; exit 1 ;;
  esac
  echo '[{"name":"integration","state":"TIMED_OUT","bucket":"fail"}]'
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

func TestDiscardWorkflowPassesBranchSelectorToClose(t *testing.T) {
	repoDir := initWorkflowRepoWithRemoteBranch(t, "feature/test")
	installWorkflowGH(t, `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "close" ]; then
  if [ "$3" != "feature/test" ]; then
    echo "expected branch selector 'feature/test', got '$3'" >&2
    exit 1
  fi
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`, "")

	r := runner.NewRunner(false, false)
	var skipped []string
	wf := DiscardWorkflow(git.New(r, repoDir), gh.New(r), "feature/test", "main", "")

	err := wf(context.Background(), ui.StepCallbacks{
		Start: func() {},
		Done:  func() {},
		Fail:  func(string) {},
		Skip: func(reason string) {
			skipped = append(skipped, reason)
		},
	})
	if err != nil {
		t.Fatalf("DiscardWorkflow() error = %v, want nil", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("DiscardWorkflow() skipped steps %v, want PR close to succeed (not soft-fail)", skipped)
	}
}

func TestFinishWorkflowPropagatesCancellationDuringRemoteDelete(t *testing.T) {
	repoDir := initWorkflowRepoWithRemoteBranch(t, "feature/test")
	deleteStarted := filepath.Join(t.TempDir(), "delete-started")
	deleteDone := filepath.Join(t.TempDir(), "delete-done")
	installWorkflowGH(t, `#!/bin/sh
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
`, "")
	installWorkflowGit(t, deleteStarted, deleteDone, "feature/test")

	r := runner.NewRunner(false, false)
	wf := FinishWorkflow(git.New(r, repoDir), gh.New(r), "feature/test", "main", "squash", false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- wf(ctx, ui.StepCallbacks{
			Start: func() {},
			Done:  func() {},
			Fail:  func(string) {},
			Skip:  func(string) {},
		})
	}()

	waitForWorkflowFile(t, deleteStarted, 2*time.Second)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("FinishWorkflow() error = %v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("FinishWorkflow() did not stop after cancellation")
	}

	if _, statErr := os.Stat(deleteDone); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("remote delete should not complete after cancellation, stat err = %v", statErr)
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

func installWorkflowGit(t *testing.T, deleteStarted, deleteDone, branch string) {
	t.Helper()

	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("LookPath(git) error = %v", err)
	}

	binDir := t.TempDir()
	gitPath := filepath.Join(binDir, "git")
	script := `#!/bin/sh
if [ "$1" = "-C" ] && [ "$3" = "push" ] && [ "$4" = "origin" ] && [ "$5" = "--delete" ] && [ "$6" = "$WORKFLOW_DELETE_BRANCH" ]; then
  exec env GO_WANT_WORKFLOW_HELPER_PROCESS=1 WORKFLOW_DELETE_STARTED="$WORKFLOW_DELETE_STARTED" WORKFLOW_DELETE_DONE="$WORKFLOW_DELETE_DONE" "` + os.Args[0] + `" -test.run=TestWorkflowHelperProcess -- remote-delete-helper
fi
exec "` + realGit + `" "$@"
`
	if err := os.WriteFile(gitPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	t.Setenv("WORKFLOW_DELETE_STARTED", deleteStarted)
	t.Setenv("WORKFLOW_DELETE_DONE", deleteDone)
	t.Setenv("WORKFLOW_DELETE_BRANCH", branch)
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

func waitForWorkflowFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWorkflowHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_WORKFLOW_HELPER_PROCESS") != "1" {
		return
	}

	args := helperArgs(os.Args)
	if len(args) != 1 || args[0] != "remote-delete-helper" {
		os.Exit(2)
	}

	if err := os.WriteFile(os.Getenv("WORKFLOW_DELETE_STARTED"), []byte("started"), 0644); err != nil {
		os.Exit(1)
	}

	time.Sleep(5 * time.Second)

	if err := os.WriteFile(os.Getenv("WORKFLOW_DELETE_DONE"), []byte("done"), 0644); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

func helperArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}

	return nil
}
