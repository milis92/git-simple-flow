# Go Project Structure Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor git-sf to align with Go best practices — extract business logic from cmd/ into per-domain internal packages, rename internal/exec to internal/runner, and add integration build tags.

**Architecture:** Business logic moves from inline Cobra handlers into Service structs in `internal/feature`, `internal/hotfix`, `internal/release`, `internal/status`. Each Service holds its dependencies (Git, GH, UI, Config) and exposes methods for each subcommand. The `cmd/` files become thin wrappers: parse flags → construct Service → call method.

**Tech Stack:** Go 1.26, Cobra, Viper, lipgloss

**Spec:** `docs/superpowers/specs/2026-03-24-go-structure-refactor-design.md`

---

## File Map

### New files
| File | Purpose |
|------|---------|
| `internal/runner/runner.go` | Moved from `internal/exec/runner.go`, package renamed to `runner` |
| `internal/runner/runner_test.go` | Moved from `internal/exec/runner_test.go`, package renamed |
| `internal/feature/feature.go` | Feature branch workflow Service (start/publish/finish/discard) |
| `internal/hotfix/hotfix.go` | Hotfix branch workflow Service (start/publish/finish/discard) |
| `internal/release/release.go` | Release tagging Service |
| `internal/status/status.go` | Status display Service |

### Deleted files/dirs
| Path | Reason |
|------|--------|
| `internal/exec/` | Renamed to `internal/runner/` |

### Modified files
| File | Change |
|------|--------|
| `internal/git/git.go` | Import path `internal/exec` → `internal/runner`, rename constructor param |
| `internal/git/git_test.go` | Import path `internal/exec` → `internal/runner` |
| `internal/gh/gh.go` | Import path, remove alias |
| `cmd/feature.go` | Rewrite to thin wrapper delegating to `internal/feature` |
| `cmd/hotfix.go` | Rewrite to thin wrapper delegating to `internal/hotfix` |
| `cmd/release.go` | Rewrite to thin wrapper delegating to `internal/release` |
| `cmd/status.go` | Rewrite to thin wrapper delegating to `internal/status` |
| `test/integration_test.go` | Add `//go:build integration` tag |
| `Makefile` | Add `-tags integration` to integration/all targets |
| `.github/workflows/ci.yml` | Add `-tags integration` to integration test step |

---

## Task 1: Rename `internal/exec` → `internal/runner` and clean up import aliases

**Files:**
- Create: `internal/runner/runner.go`, `internal/runner/runner_test.go`
- Delete: `internal/exec/`
- Modify: `internal/git/git.go`, `internal/git/git_test.go`, `internal/gh/gh.go`, `cmd/feature.go`, `cmd/hotfix.go`, `cmd/release.go`, `cmd/status.go`

- [ ] **Step 1: Create `internal/runner/` and move files**

```bash
mkdir -p internal/runner
```

- [ ] **Step 2: Create `internal/runner/runner.go`**

Copy from `internal/exec/runner.go`, change `package exec` → `package runner`:

```go
package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	DryRun  bool
	Verbose bool
	Output  io.Writer
}

func NewRunner(dryRun, verbose bool) *Runner {
	return &Runner{
		DryRun:  dryRun,
		Verbose: verbose,
		Output:  os.Stderr,
	}
}

func (r *Runner) Run(name string, args ...string) (string, error) {
	cmdStr := name + " " + strings.Join(args, " ")

	if r.DryRun {
		fmt.Fprintf(r.Output, "  [dry-run] %s\n", cmdStr)
		return "", nil
	}

	if r.Verbose {
		fmt.Fprintf(r.Output, "  $ %s\n", cmdStr)
	}

	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", cmdStr, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}
```

- [ ] **Step 3: Create `internal/runner/runner_test.go`**

Copy from `internal/exec/runner_test.go`, change `package exec` → `package runner`:

```go
package runner

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	r := NewRunner(false, false)
	out, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Errorf("output = %q, want %q", out, "hello")
	}
}

func TestRunFailure(t *testing.T) {
	r := NewRunner(false, false)
	_, err := r.Run("false")
	if err == nil {
		t.Error("expected error from `false` command")
	}
}

func TestRunDryRun(t *testing.T) {
	var buf bytes.Buffer
	r := NewRunner(true, false)
	r.Output = &buf
	out, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
	if out != "" {
		t.Errorf("dry-run output should be empty, got %q", out)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("dry-run should log, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "echo hello") {
		t.Errorf("dry-run should show command, got %q", buf.String())
	}
}

func TestRunVerbose(t *testing.T) {
	var buf bytes.Buffer
	r := NewRunner(false, true)
	r.Output = &buf
	_, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "echo hello") {
		t.Errorf("verbose should log command, got %q", buf.String())
	}
}
```

- [ ] **Step 4: Delete `internal/exec/`**

```bash
rm -rf internal/exec
```

- [ ] **Step 5: Update `internal/git/git.go` imports and constructor**

Change import `internal/exec` → `internal/runner`. Update `Git` struct field type and `New` constructor parameter name to avoid shadowing:

```go
// import block: replace
//   "github.com/nickssmallpdf/git-sf/internal/exec"
// with
//   "github.com/nickssmallpdf/git-sf/internal/runner"

// struct: replace
//   runner *exec.Runner
// with
//   runner *runner.Runner

// constructor: replace
//   func New(runner *exec.Runner, dir string) *Git {
//       return &Git{runner: runner, dir: dir}
//   }
// with
//   func New(r *runner.Runner, dir string) *Git {
//       return &Git{runner: r, dir: dir}
//   }
```

- [ ] **Step 6: Update `internal/git/git_test.go` imports**

Replace all `exec.NewRunner` with `runner.NewRunner`, update import path:

```go
// import block: replace
//   "github.com/nickssmallpdf/git-sf/internal/exec"
// with
//   "github.com/nickssmallpdf/git-sf/internal/runner"

// body: replace all occurrences of
//   exec.NewRunner
// with
//   runner.NewRunner
```

- [ ] **Step 7: Update `internal/gh/gh.go` imports**

Remove alias, update import path:

```go
// import block: replace
//   runner "github.com/nickssmallpdf/git-sf/internal/exec"
// with
//   "github.com/nickssmallpdf/git-sf/internal/runner"
```

No body changes needed — existing code already uses `runner.Runner` via the alias.

- [ ] **Step 8: Update `cmd/feature.go` imports**

Replace aliased imports with clean imports, update body references:

```go
// import block: replace
//   runner "github.com/nickssmallpdf/git-sf/internal/exec"
//   gitpkg "github.com/nickssmallpdf/git-sf/internal/git"
// with
//   "github.com/nickssmallpdf/git-sf/internal/git"
//   "github.com/nickssmallpdf/git-sf/internal/runner"

// body: replace all occurrences of
//   gitpkg.
// with
//   git.
```

- [ ] **Step 9: Update `cmd/hotfix.go` imports**

Same pattern as step 8: remove aliases, update `gitpkg.` → `git.`

- [ ] **Step 10: Update `cmd/release.go` imports**

Same pattern: remove aliases, update `gitpkg.` → `git.`

- [ ] **Step 11: Update `cmd/status.go` imports**

Same pattern: remove aliases, update `gitpkg.` → `git.`

- [ ] **Step 12: Verify compilation and tests**

```bash
go build ./...
go test ./internal/... -v
go test ./test/... -v -count=1
```

Expected: all pass.

- [ ] **Step 13: Commit**

```bash
git add internal/runner/ internal/git/git.go internal/git/git_test.go internal/gh/gh.go cmd/feature.go cmd/hotfix.go cmd/release.go cmd/status.go
git rm -r internal/exec/
git commit -m "refactor: rename internal/exec to internal/runner and clean up import aliases"
```

---

## Task 2: Add `//go:build integration` tag and update Makefile/CI

**Files:**
- Modify: `test/integration_test.go`, `Makefile`, `.github/workflows/ci.yml`

- [ ] **Step 1: Add build tag to `test/integration_test.go`**

Add `//go:build integration` as the very first line, followed by a blank line before the package declaration:

```go
//go:build integration

// test/integration_test.go
package test
// ... rest unchanged
```

- [ ] **Step 2: Update `Makefile`**

Change `test-integration` and `test-all` targets to include `-tags integration`:

```makefile
test-integration:
	go test -tags integration ./test/... -v -count=1

test-all:
	go test -tags integration ./... -v -count=1
```

The `test` target (`go test ./internal/... -v`) stays unchanged.

- [ ] **Step 3: Update `.github/workflows/ci.yml`**

Change the integration test step (line 23) from:
```yaml
        run: go test ./test/... -v -count=1
```
to:
```yaml
        run: go test -tags integration ./test/... -v -count=1
```

- [ ] **Step 4: Verify**

```bash
# Unit tests still work without tag
go test ./internal/... -v

# Integration tests require tag now
go test -tags integration ./test/... -v -count=1

# ./... without tag should skip integration tests (no test files in test/ match)
go test ./... -v 2>&1 | grep -c "integration_test.go" && echo "FAIL: integration tests should be skipped" || echo "OK: integration tests skipped"
```

- [ ] **Step 5: Commit**

```bash
git add test/integration_test.go Makefile .github/workflows/ci.yml
git commit -m "ci: add //go:build integration tag to integration tests"
```

---

## Task 3: Extract `internal/feature` package

**Files:**
- Create: `internal/feature/feature.go`
- Modify: `cmd/feature.go`

- [ ] **Step 1: Create `internal/feature/feature.go`**

Extract all business logic from `cmd/feature.go` handlers into Service methods:

```go
package feature

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/config"
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/ui"
)

type Service struct {
	Git    *git.Git
	GH     *gh.GH
	UI     *ui.UI
	Config config.Config
}

type StartOpts struct {
	DraftPR bool
	Title   string
}

type PublishOpts struct {
	Title string
	Body  string
}

type FinishOpts struct {
	Force bool
}

func (s *Service) Start(name string, opts StartOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckIsRepo(); err != nil {
		return err
	}
	if err := s.Git.CheckCleanTree(); err != nil {
		return err
	}

	branchName := s.Config.FeaturePrefix + name

	if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
		return err
	}
	s.UI.Success("Switched to " + s.Config.MainBranch)

	if err := s.Git.Pull(); err != nil {
		return err
	}
	s.UI.Success("Pulled latest changes")

	if err := s.Git.CreateBranch(branchName); err != nil {
		return err
	}
	s.UI.Success("Created branch " + branchName)

	if opts.DraftPR || s.Config.DraftPROnStart {
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := s.GH.CheckAuthenticated(); err != nil {
			return err
		}
		if err := s.Git.Push(branchName); err != nil {
			return err
		}
		title := opts.Title
		if title == "" {
			title = gh.HumanizeBranchName(branchName, s.Config.FeaturePrefix)
		}
		pr, err := s.GH.CreatePR(s.Config.MainBranch, title, "", true)
		if err != nil {
			return err
		}
		s.UI.Success("Created draft PR: " + pr.URL)
	}

	s.UI.Result("Ready to work. When done: git sf feature publish")
	return nil
}

func (s *Service) Publish(opts PublishOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := gh.CheckGHInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckIsRepo(); err != nil {
		return err
	}
	if err := s.GH.CheckAuthenticated(); err != nil {
		return err
	}

	clean, err := s.Git.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		s.UI.Warning("You have uncommitted changes — consider committing or stashing them first")
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	if err := s.Git.Push(branch); err != nil {
		return err
	}
	s.UI.Success("Pushed branch " + branch)

	title := opts.Title
	if title == "" {
		title = gh.HumanizeBranchName(branch, s.Config.FeaturePrefix)
	}

	pr, err := s.GH.CreatePR(s.Config.MainBranch, title, opts.Body, false)
	if err != nil {
		return err
	}
	s.UI.Success("Created PR: " + pr.URL)

	s.UI.Result("PR is open. When ready to merge: git sf feature finish")
	return nil
}

func (s *Service) Finish(opts FinishOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := gh.CheckGHInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckIsRepo(); err != nil {
		return err
	}
	if err := s.GH.CheckAuthenticated(); err != nil {
		return err
	}
	if err := s.Git.CheckCleanTree(); err != nil {
		return err
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	pr, err := s.GH.GetCurrentPR()
	if err != nil {
		return err
	}
	s.UI.Info(fmt.Sprintf("Found PR #%d — %q", pr.Number, pr.Title))

	if !opts.Force {
		checks, err := s.GH.GetPRChecks()
		if err != nil {
			return err
		}
		var failing []string
		for _, c := range checks {
			if c.Conclusion == "failure" || c.Conclusion == "cancelled" {
				failing = append(failing, c.Name)
			}
		}
		if len(failing) > 0 {
			return fmt.Errorf("PR checks failed: %s (use --force to override)", strings.Join(failing, ", "))
		}
		s.UI.Success("PR checks passed")
	}

	ok, err := s.UI.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
	if err != nil {
		return err
	}
	if !ok {
		s.UI.Info("Merge cancelled")
		return nil
	}

	if err := s.GH.MergePR(s.Config.MergeStrategy); err != nil {
		return err
	}
	s.UI.Success("Merged PR #" + fmt.Sprint(pr.Number))

	if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
		return err
	}
	if err := s.Git.Pull(); err != nil {
		return err
	}
	s.UI.Success("Switched to " + s.Config.MainBranch + " and pulled latest changes")

	if err := s.Git.DeleteLocalBranch(branch); err != nil {
		return err
	}
	s.UI.Success("Deleted local branch " + branch)

	if err := s.Git.DeleteRemoteBranch(branch); err != nil {
		s.UI.Warning("Remote branch already deleted or could not be removed: " + branch)
	} else {
		s.UI.Success("Deleted remote branch " + branch)
	}

	s.UI.Result("Feature complete!")
	return nil
}

func (s *Service) Discard(reason string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckIsRepo(); err != nil {
		return err
	}
	if err := s.Git.CheckCleanTree(); err != nil {
		return err
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	if !strings.HasPrefix(branch, s.Config.FeaturePrefix) {
		return fmt.Errorf("not on a feature branch (current branch: %s)", branch)
	}

	ok, err := s.UI.Confirm(fmt.Sprintf("Discard feature branch %q and close its PR?", branch))
	if err != nil {
		return err
	}
	if !ok {
		s.UI.Info("Discard cancelled")
		return nil
	}

	if ghErr := gh.CheckGHInstalled(); ghErr == nil {
		if err := s.GH.ClosePR(reason); err != nil {
			s.UI.Warning("Could not close PR (may not exist): " + err.Error())
		} else {
			s.UI.Success("Closed PR")
		}
	} else {
		s.UI.Warning("gh CLI not available — skipping PR close")
	}

	if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
		return err
	}
	s.UI.Success("Switched to " + s.Config.MainBranch)

	if err := s.Git.DeleteLocalBranch(branch); err != nil {
		return err
	}
	s.UI.Success("Deleted local branch " + branch)

	if err := s.Git.DeleteRemoteBranch(branch); err != nil {
		s.UI.Warning("Remote branch already deleted or could not be removed: " + branch)
	} else {
		s.UI.Success("Deleted remote branch " + branch)
	}

	s.UI.Result("Feature discarded.")
	return nil
}
```

- [ ] **Step 2: Rewrite `cmd/feature.go` as thin wrapper**

```go
// cmd/feature.go
package cmd

import (
	"github.com/nickssmallpdf/git-sf/internal/feature"
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/spf13/cobra"
)

var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage feature branches",
}

var featureStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Create a new feature branch from main",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		draftPR, _ := cmd.Flags().GetBool("draft-pr")
		title, _ := cmd.Flags().GetString("title")
		return svc.Start(args[0], feature.StartOpts{DraftPR: draftPR, Title: title})
	},
}

var featurePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Push the current feature branch and open a PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		return svc.Publish(feature.PublishOpts{Title: title, Body: body})
	},
}

var featureFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Merge the current feature branch PR and clean up",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		force, _ := cmd.Flags().GetBool("force")
		return svc.Finish(feature.FinishOpts{Force: force})
	},
}

var featureDiscardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Abandon the current feature branch and close its PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		reason, _ := cmd.Flags().GetString("reason")
		return svc.Discard(reason)
	},
}

func init() {
	featureStartCmd.Flags().Bool("draft-pr", false, "create a draft PR immediately")
	featureStartCmd.Flags().String("title", "", "PR title (defaults to humanized branch name)")
	featurePublishCmd.Flags().String("title", "", "PR title (defaults to humanized branch name)")
	featurePublishCmd.Flags().String("body", "", "PR body/description")
	featureFinishCmd.Flags().Bool("force", false, "skip PR checks validation")
	featureDiscardCmd.Flags().String("reason", "", "comment to leave on the closed PR")

	featureCmd.AddCommand(featureStartCmd)
	featureCmd.AddCommand(featurePublishCmd)
	featureCmd.AddCommand(featureFinishCmd)
	featureCmd.AddCommand(featureDiscardCmd)
	rootCmd.AddCommand(featureCmd)
}
```

- [ ] **Step 3: Verify**

```bash
go build ./...
go test ./internal/... -v
go test -tags integration ./test/... -v -count=1
```

Expected: all pass. The integration tests for feature (TestFeatureStartDryRun, TestFeatureStartActual, TestFeatureStartDirtyTree) validate the behavior is preserved.

- [ ] **Step 4: Commit**

```bash
git add internal/feature/feature.go cmd/feature.go
git commit -m "refactor: extract feature business logic into internal/feature package"
```

---

## Task 4: Extract `internal/hotfix` package

**Files:**
- Create: `internal/hotfix/hotfix.go`
- Modify: `cmd/hotfix.go`

- [ ] **Step 1: Create `internal/hotfix/hotfix.go`**

```go
package hotfix

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/config"
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
)

type Service struct {
	Git    *git.Git
	GH     *gh.GH
	UI     *ui.UI
	Config config.Config
}

type StartOpts struct {
	DraftPR bool
	Title   string
}

type PublishOpts struct {
	Title string
	Body  string
}

type FinishOpts struct {
	Force   bool
	Release bool
}

func (s *Service) Start(name string, opts StartOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckIsRepo(); err != nil {
		return err
	}
	if err := s.Git.CheckCleanTree(); err != nil {
		return err
	}

	tag, err := s.Git.LatestTag(s.Config.TagPrefix)
	if err != nil {
		return fmt.Errorf("no tags found. Create an initial release first with 'git sf release'")
	}

	if err := s.Git.Checkout(tag); err != nil {
		return err
	}
	s.UI.Success("Checked out " + tag)

	branchName := s.Config.HotfixPrefix + name
	if err := s.Git.CreateBranch(branchName); err != nil {
		return err
	}
	s.UI.Success("Created branch " + branchName)

	if opts.DraftPR || s.Config.DraftPROnStart {
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := s.GH.CheckAuthenticated(); err != nil {
			return err
		}
		if err := s.Git.Push(branchName); err != nil {
			return err
		}
		title := opts.Title
		if title == "" {
			title = gh.HumanizeBranchName(branchName, s.Config.HotfixPrefix)
		}
		pr, err := s.GH.CreatePR(s.Config.MainBranch, title, "", true)
		if err != nil {
			return err
		}
		s.UI.Success("Created draft PR: " + pr.URL)
	}

	s.UI.Result("Ready to work. When done: git sf hotfix publish")
	return nil
}

func (s *Service) Publish(opts PublishOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := gh.CheckGHInstalled(); err != nil {
		return err
	}
	if err := s.GH.CheckAuthenticated(); err != nil {
		return err
	}

	clean, _ := s.Git.IsClean()
	if !clean {
		s.UI.Warning("You have uncommitted changes that won't be included in the PR.")
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	if err := s.Git.Push(branch); err != nil {
		return err
	}
	s.UI.Success("Pushed " + branch)

	title := opts.Title
	if title == "" {
		title = gh.HumanizeBranchName(branch, s.Config.HotfixPrefix)
	}

	pr, err := s.GH.CreatePR(s.Config.MainBranch, title, opts.Body, false)
	if err != nil {
		return err
	}
	s.UI.Success("Created PR: " + pr.URL)

	s.UI.Result("PR is up. When ready: git sf hotfix finish")
	return nil
}

func (s *Service) Finish(opts FinishOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := gh.CheckGHInstalled(); err != nil {
		return err
	}
	if err := s.GH.CheckAuthenticated(); err != nil {
		return err
	}
	if err := s.Git.CheckCleanTree(); err != nil {
		return err
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	pr, err := s.GH.GetCurrentPR()
	if err != nil {
		return fmt.Errorf("no PR found for this branch. Run 'git sf hotfix publish' first")
	}

	if !opts.Force {
		checks, err := s.GH.GetPRChecks()
		if err == nil {
			failing := false
			for _, c := range checks {
				if c.Conclusion == "failure" {
					s.UI.Error(c.Name + " — failed")
					failing = true
				} else if c.Status != "completed" {
					s.UI.Warning(c.Name + " — " + c.Status)
				} else {
					s.UI.Success(c.Name + " — passed")
				}
			}
			if failing {
				return fmt.Errorf("PR has failing checks. Fix them or use --force to merge anyway")
			}
		}
	}

	confirmed, err := s.UI.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
	if err != nil || !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	s.UI.Info(fmt.Sprintf("Merging PR #%d — %q", pr.Number, pr.Title))

	if err := s.GH.MergePR(s.Config.MergeStrategy); err != nil {
		return err
	}
	s.UI.Success(fmt.Sprintf("PR merged (%s)", s.Config.MergeStrategy))

	if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
		return err
	}
	s.UI.Success("Switched to " + s.Config.MainBranch)

	if err := s.Git.Pull(); err != nil {
		return err
	}
	s.UI.Success("Pulled latest changes")

	if err := s.Git.DeleteLocalBranch(branch); err != nil {
		return err
	}
	s.UI.Success("Deleted branch " + branch + " (local)")

	if err := s.Git.DeleteRemoteBranch(branch); err != nil {
		s.UI.Warning("Remote branch already deleted")
	} else {
		s.UI.Success("Deleted branch " + branch + " (remote)")
	}

	// Auto-release if --release flag or config
	if opts.Release || s.Config.HotfixAutoRelease {
		tag, err := s.Git.LatestTag(s.Config.TagPrefix)
		if err != nil {
			return err
		}
		current, err := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
		if err != nil {
			return err
		}
		next, _ := current.Bump("patch")
		newTag := next.FormatWithPrefix(s.Config.TagPrefix)

		if err := s.Git.Tag(newTag); err != nil {
			return err
		}
		s.UI.Success("Tagged " + newTag)

		if err := s.Git.PushTag(newTag); err != nil {
			return err
		}
		s.UI.Success("Pushed tag to origin")

		s.UI.Result("Hotfix released " + newTag)
		return nil
	}

	s.UI.Result("Done.")
	return nil
}

func (s *Service) Discard(reason string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckCleanTree(); err != nil {
		return err
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	if !strings.HasPrefix(branch, s.Config.HotfixPrefix) {
		return fmt.Errorf("not on a hotfix branch (current: %s)", branch)
	}

	confirmed, err := s.UI.Confirm("Discard branch " + branch + "?")
	if err != nil || !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	if err := gh.CheckGHInstalled(); err == nil {
		if err := s.GH.CheckAuthenticated(); err == nil {
			if err := s.GH.ClosePR(reason); err != nil {
				s.UI.Warning("No PR to close or already closed")
			} else {
				s.UI.Success("Closed PR")
			}
		}
	}

	if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
		return err
	}
	s.UI.Success("Switched to " + s.Config.MainBranch)

	if err := s.Git.DeleteLocalBranch(branch); err != nil {
		return err
	}
	s.UI.Success("Deleted branch " + branch + " (local)")

	if err := s.Git.DeleteRemoteBranch(branch); err != nil {
		s.UI.Warning("Remote branch already deleted")
	} else {
		s.UI.Success("Deleted branch " + branch + " (remote)")
	}

	s.UI.Result("Discarded.")
	return nil
}
```

- [ ] **Step 2: Rewrite `cmd/hotfix.go` as thin wrapper**

```go
// cmd/hotfix.go
package cmd

import (
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/hotfix"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/spf13/cobra"
)

var hotfixCmd = &cobra.Command{
	Use:   "hotfix",
	Short: "Manage hotfix branches",
}

var hotfixStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Create a new hotfix branch from the latest tag",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		draftPR, _ := cmd.Flags().GetBool("draft-pr")
		title, _ := cmd.Flags().GetString("title")
		return svc.Start(args[0], hotfix.StartOpts{DraftPR: draftPR, Title: title})
	},
}

var hotfixPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Push branch and create a PR to main",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		return svc.Publish(hotfix.PublishOpts{Title: title, Body: body})
	},
}

var hotfixFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Merge PR, switch to main, delete branch, optionally tag release",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		force, _ := cmd.Flags().GetBool("force")
		release, _ := cmd.Flags().GetBool("release")
		return svc.Finish(hotfix.FinishOpts{Force: force, Release: release})
	},
}

var hotfixDiscardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Close PR, delete branch, switch to main",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		reason, _ := cmd.Flags().GetString("reason")
		return svc.Discard(reason)
	},
}

func init() {
	hotfixStartCmd.Flags().Bool("draft-pr", false, "create a draft PR immediately")
	hotfixStartCmd.Flags().String("title", "", "PR title (defaults to humanized branch name)")
	hotfixPublishCmd.Flags().String("title", "", "PR title (defaults to humanized branch name)")
	hotfixPublishCmd.Flags().String("body", "", "PR body/description")
	hotfixFinishCmd.Flags().Bool("force", false, "skip PR checks validation")
	hotfixFinishCmd.Flags().Bool("release", false, "auto-tag a patch release after merge")
	hotfixDiscardCmd.Flags().String("reason", "", "comment to leave on the closed PR")

	hotfixCmd.AddCommand(hotfixStartCmd)
	hotfixCmd.AddCommand(hotfixPublishCmd)
	hotfixCmd.AddCommand(hotfixFinishCmd)
	hotfixCmd.AddCommand(hotfixDiscardCmd)
	rootCmd.AddCommand(hotfixCmd)
}
```

- [ ] **Step 3: Verify**

```bash
go build ./...
go test ./internal/... -v
go test -tags integration ./test/... -v -count=1
```

Expected: all pass. Integration tests TestHotfixStartFromTag and TestHotfixStartNoTags validate behavior.

- [ ] **Step 4: Commit**

```bash
git add internal/hotfix/hotfix.go cmd/hotfix.go
git commit -m "refactor: extract hotfix business logic into internal/hotfix package"
```

---

## Task 5: Extract `internal/release` package

**Files:**
- Create: `internal/release/release.go`
- Modify: `cmd/release.go`

- [ ] **Step 1: Create `internal/release/release.go`**

```go
package release

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/config"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
)

type Service struct {
	Git    *git.Git
	UI     *ui.UI
	Config config.Config
}

func (s *Service) Release(scope string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := s.Git.CheckIsRepo(); err != nil {
		return err
	}
	if err := s.Git.CheckOnBranch(s.Config.MainBranch); err != nil {
		return err
	}

	if err := s.Git.Fetch(); err != nil {
		return err
	}

	inSync, err := s.Git.IsInSyncWithRemote(s.Config.MainBranch)
	if err != nil {
		return err
	}
	if !inSync {
		return fmt.Errorf("local %s is not in sync with origin/%s — pull or push first", s.Config.MainBranch, s.Config.MainBranch)
	}

	tag, err := s.Git.LatestTag(s.Config.TagPrefix)
	var next version.Version
	var currentDisplay string

	if err != nil {
		// No tags — first release
		next = version.Version{Major: 0, Minor: 1, Patch: 0}
		currentDisplay = "(no tags)"
		s.UI.Info("No existing tags found. Starting at " + next.FormatWithPrefix(s.Config.TagPrefix))
	} else {
		current, parseErr := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
		if parseErr != nil {
			return parseErr
		}
		currentDisplay = tag
		next, err = current.Bump(scope)
		if err != nil {
			return err
		}
	}

	newTag := next.FormatWithPrefix(s.Config.TagPrefix)

	s.UI.Blank()
	s.UI.Muted("Current: " + currentDisplay)
	s.UI.Muted(fmt.Sprintf("Next:    %s (%s)", newTag, scope))
	s.UI.Blank()

	confirmed, err := s.UI.Confirm("Confirm release?")
	if err != nil || !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	s.UI.Blank()

	if err := s.Git.Tag(newTag); err != nil {
		return err
	}
	s.UI.Success("Tagged " + newTag)

	if err := s.Git.PushTag(newTag); err != nil {
		return err
	}
	s.UI.Success("Pushed tag to origin")

	s.UI.Result("Released " + newTag)
	return nil
}
```

- [ ] **Step 2: Rewrite `cmd/release.go` as thin wrapper**

```go
// cmd/release.go
package cmd

import (
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/release"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release [major|minor|patch]",
	Short: "Tag and push a release from main",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &release.Service{
			Git:    git.New(r, "."),
			UI:     ui.New(),
			Config: cfg,
		}
		scope := cfg.DefaultReleaseBump
		if len(args) > 0 {
			scope = args[0]
		}
		return svc.Release(scope)
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
```

- [ ] **Step 3: Verify**

```bash
go build ./...
go test ./internal/... -v
go test -tags integration ./test/... -v -count=1
```

Expected: all pass. Integration tests TestReleaseFromNonMain and TestReleaseFirstRelease validate behavior.

- [ ] **Step 4: Commit**

```bash
git add internal/release/release.go cmd/release.go
git commit -m "refactor: extract release business logic into internal/release package"
```

---

## Task 6: Extract `internal/status` package

**Files:**
- Create: `internal/status/status.go`
- Modify: `cmd/status.go`

- [ ] **Step 1: Create `internal/status/status.go`**

```go
package status

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/config"
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
)

type Service struct {
	Git    *git.Git
	GH     *gh.GH
	UI     *ui.UI
	Config config.Config
}

func (s *Service) Show() error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	// Determine branch type
	branchType := "other"
	if branch == s.Config.MainBranch {
		branchType = s.Config.MainBranch
	} else if strings.HasPrefix(branch, s.Config.FeaturePrefix) {
		branchType = "feature"
	} else if strings.HasPrefix(branch, s.Config.HotfixPrefix) {
		branchType = "hotfix"
	}

	s.UI.Blank()
	fmt.Fprintf(s.UI.Out, "  Branch:     %s\n", branch)
	if branchType != "other" && branchType != s.Config.MainBranch {
		fmt.Fprintf(s.UI.Out, "  Type:       %s\n", branchType)
	}

	// PR info (if on feature/hotfix branch)
	if branchType == "feature" || branchType == "hotfix" {
		if gh.CheckGHInstalled() == nil {
			if pr, err := s.GH.GetCurrentPR(); err == nil {
				draft := ""
				if pr.Draft {
					draft = " (draft)"
				}
				fmt.Fprintf(s.UI.Out, "  PR:         #%d%s — %s\n", pr.Number, draft, pr.URL)

				// Show checks
				if checks, err := s.GH.GetPRChecks(); err == nil && len(checks) > 0 {
					passing, failing, pending := 0, 0, 0
					for _, c := range checks {
						switch {
						case c.Conclusion == "success":
							passing++
						case c.Conclusion == "failure":
							failing++
						default:
							pending++
						}
					}
					total := len(checks)
					status := fmt.Sprintf("%d/%d passing", passing, total)
					if failing > 0 {
						status += fmt.Sprintf(", %d failing", failing)
					}
					if pending > 0 {
						status += fmt.Sprintf(", %d pending", pending)
					}
					fmt.Fprintf(s.UI.Out, "  Checks:     %s\n", status)
				}
			}
		}

		// Behind main
		ahead, behind, err := s.Git.CommitsAheadBehind(branch, s.Config.MainBranch)
		if err == nil {
			if behind > 0 {
				fmt.Fprintf(s.UI.Out, "  Behind:     %d commits behind %s\n", behind, s.Config.MainBranch)
			}
			if ahead > 0 {
				fmt.Fprintf(s.UI.Out, "  Ahead:      %d commits ahead of %s\n", ahead, s.Config.MainBranch)
			}
		}
	}

	// Tag/release info
	s.UI.Blank()
	tag, err := s.Git.LatestTag(s.Config.TagPrefix)
	if err != nil {
		fmt.Fprintf(s.UI.Out, "  Latest tag:    (none)\n")
	} else {
		fmt.Fprintf(s.UI.Out, "  Latest tag:    %s\n", tag)

		if branch == s.Config.MainBranch {
			ahead, _, err := s.Git.CommitsAheadBehind(s.Config.MainBranch, tag)
			if err == nil && ahead > 0 {
				fmt.Fprintf(s.UI.Out, "  Ahead:         %d commits since %s\n", ahead, tag)
			}
		}

		// Show next versions
		current, parseErr := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
		if parseErr == nil {
			major, _ := current.Bump("major")
			minor, _ := current.Bump("minor")
			patch, _ := current.Bump("patch")
			fmt.Fprintf(s.UI.Out, "  Next release:  %s (minor) / %s (patch) / %s (major)\n",
				minor.FormatWithPrefix(s.Config.TagPrefix),
				patch.FormatWithPrefix(s.Config.TagPrefix),
				major.FormatWithPrefix(s.Config.TagPrefix))
		}
	}

	s.UI.Blank()
	return nil
}
```

- [ ] **Step 2: Rewrite `cmd/status.go` as thin wrapper**

Note: status passes `false` for dryRun (read-only command).

```go
// cmd/status.go
package cmd

import (
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/status"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current branch, PR, and release info",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(false, verbose) // status is read-only, no dry-run needed
		svc := &status.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		return svc.Show()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
```

- [ ] **Step 3: Verify**

```bash
go build ./...
go test ./internal/... -v
go test -tags integration ./test/... -v -count=1
```

Expected: all pass. Integration test TestStatusOnMain validates behavior.

- [ ] **Step 4: Commit**

```bash
git add internal/status/status.go cmd/status.go
git commit -m "refactor: extract status business logic into internal/status package"
```

---

## Final Verification

After all tasks are complete:

```bash
# Full build
go build ./...

# All tests
go test -tags integration ./... -v -count=1

# Lint
golangci-lint run

# Verify internal/exec is gone
test ! -d internal/exec && echo "OK: internal/exec removed" || echo "FAIL: internal/exec still exists"
```
