package status

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestShowClassifiesChecksUsingSharedMergeSemantics(t *testing.T) {
	repoDir := initStatusRepo(t, "feature/test")
	installStatusGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo '{"number":123,"title":"Feature PR","state":"OPEN","url":"https://example.com/pr/123","isDraft":false}'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  case "$*" in
    *--required*) ;;
    *) echo "missing --required flag in: $*" >&2; exit 1 ;;
  esac
  echo '[{"name":"build","state":"SUCCESS","bucket":"pass"},{"name":"docs","state":"NEUTRAL","bucket":"skipping"},{"name":"e2e","state":"TIMED_OUT","bucket":"fail"},{"name":"lint","state":"PENDING","bucket":"pending"}]'
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	if err := svc.Show(); err != nil {
		t.Fatalf("Show() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Checks:     2/4 passing, 1 failing, 1 pending") {
		t.Fatalf("Show() output = %q, want shared check classification summary", output)
	}
}

func TestShowDistinguishesNoPRFromFetchErrors(t *testing.T) {
	repoDir := initStatusRepo(t, "feature/test")
	installStatusGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo "GraphQL API unavailable" >&2
  exit 1
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
	}

	if err := svc.Show(); err != nil {
		t.Fatalf("Show() error = %v", err)
	}

	output := out.String()
	if strings.Contains(output, "No PR for this branch") {
		t.Fatalf("Show() output = %q, should not claim there is no PR on unrelated gh errors", output)
	}
	if !strings.Contains(output, "Could not fetch PR info:") {
		t.Fatalf("Show() output = %q, want fetch error message", output)
	}
}

func TestShowDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initStatusRepo(t, "feature/test")
	installStatusGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo '{"number":123,"title":"Feature PR","state":"OPEN","url":"https://example.com/pr/123","isDraft":false}'
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
echo "unexpected gh command: $*" >&2
exit 1
`)

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

	if err := svc.Show(); err != nil {
		t.Fatalf("Show() error = %v, want dry-run to succeed using real repo state", err)
	}

	output := out.String()
	if !strings.Contains(output, "feature/test") {
		t.Fatalf("Show() output = %q, want branch name resolved from real repo", output)
	}
}

func initStatusRepo(t *testing.T, branch string) string {
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
	runGit(t, repoDir, "checkout", "-b", branch)

	return repoDir
}

func installStatusGH(t *testing.T, script string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
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
