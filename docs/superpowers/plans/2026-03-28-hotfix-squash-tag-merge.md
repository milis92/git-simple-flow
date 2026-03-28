# Hotfix Squash-Tag-Merge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change `git sf hotfix finish --release` to squash the hotfix branch, tag the squashed commit, force push, then merge via `gh pr merge --merge` — so the release tag never includes unreleased feature commits.

**Architecture:** Four new git primitives (`MergeBase`, `ResetSoft`, `CommitWithMessage`, `ForcePush`) are added to `internal/git/git.go`. A new `MergePRWithMessage` method is added to `internal/gh/gh.go`. The hotfix service (`internal/hotfix/hotfix.go`) orchestrates these in the release path, bypassing the shared `workflow.FinishWorkflow` and using its own step sequence. Non-release hotfix finish and all feature workflows remain unchanged.

**Tech Stack:** Go, spf13/cobra, `git` CLI, `gh` CLI, lipgloss/bubbletea (existing UI layer)

---

### Task 1: Add git primitives — `MergeBase`

**Files:**
- Modify: `internal/git/git.go:193` (append after `CommitsAheadBehind`)
- Test: `internal/git/git_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/git/git_test.go`, add:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/git/ -run TestMergeBase -v`
Expected: FAIL — `g.MergeBase undefined`

- [ ] **Step 3: Write minimal implementation**

In `internal/git/git.go`, append after the `CommitsAheadBehind` method:

```go
// MergeBase returns the best common ancestor commit between two refs.
func (g *Git) MergeBase(a, b string) (string, error) {
	return g.run("merge-base", a, b)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/git/ -run TestMergeBase -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat(git): add MergeBase primitive"
```

---

### Task 2: Add git primitives — `ResetSoft` and `CommitWithMessage`

**Files:**
- Modify: `internal/git/git.go` (append after `MergeBase`)
- Test: `internal/git/git_test.go`

- [ ] **Step 1: Write the failing tests**

In `internal/git/git_test.go`, add:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/git/ -run TestResetSoftAndCommitWithMessage -v`
Expected: FAIL — `g.ResetSoft undefined`

- [ ] **Step 3: Write minimal implementation**

In `internal/git/git.go`, append after `MergeBase`:

```go
// ResetSoft moves HEAD to the given ref while keeping all changes staged.
func (g *Git) ResetSoft(ref string) error {
	_, err := g.run("reset", "--soft", ref)
	return err
}

// CommitWithMessage creates a commit with the given message from staged changes.
func (g *Git) CommitWithMessage(msg string) error {
	_, err := g.run("commit", "-m", msg)
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/git/ -run TestResetSoftAndCommitWithMessage -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat(git): add ResetSoft and CommitWithMessage primitives"
```

---

### Task 3: Add git primitive — `ForcePush`

**Files:**
- Modify: `internal/git/git.go` (append after `CommitWithMessage`)
- Test: `internal/git/git_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/git/git_test.go`, add:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/git/ -run TestForcePush -v`
Expected: FAIL — `g.ForcePush undefined`

- [ ] **Step 3: Write minimal implementation**

In `internal/git/git.go`, append after `CommitWithMessage`:

```go
// ForcePush force-pushes the given branch to origin, overwriting remote history.
func (g *Git) ForcePush(branch string) error {
	_, err := g.run("push", "--force", "origin", branch)
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/git/ -run TestForcePush -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat(git): add ForcePush primitive"
```

---

### Task 4: Add `MergePRWithMessage` to GH

**Files:**
- Modify: `internal/gh/gh.go:86` (after `MergePR`)
- Test: `internal/gh/gh_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/gh/gh_test.go`, add:

```go
func TestMergePRWithMessagePassesFlags(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  case "$*" in
    *--merge*) ;;
    *) echo "missing --merge flag in: $*" >&2; exit 1 ;;
  esac
  case "$*" in
    *--subject*) ;;
    *) echo "missing --subject flag in: $*" >&2; exit 1 ;;
  esac
  case "$*" in
    *--body*) ;;
    *) echo "missing --body flag in: $*" >&2; exit 1 ;;
  esac
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	if err := client.MergePRWithMessage("merge", "Merge hotfix v1.2.4", ""); err != nil {
		t.Fatalf("MergePRWithMessage() error = %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/gh/ -run TestMergePRWithMessagePassesFlags -v`
Expected: FAIL — `client.MergePRWithMessage undefined`

- [ ] **Step 3: Write minimal implementation**

In `internal/gh/gh.go`, add after the `MergePR` method:

```go
// MergePRWithMessage merges the current branch's PR using the given strategy
// and sets the merge commit subject and body.
func (g *GH) MergePRWithMessage(strategy, subject, body string) error {
	args := []string{"pr", "merge", "--" + strategy, "--subject", subject, "--body", body}
	_, err := g.runner.Run("gh", args...)
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/gh/ -run TestMergePRWithMessagePassesFlags -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gh/gh.go internal/gh/gh_test.go
git commit -m "feat(gh): add MergePRWithMessage for custom merge commit messages"
```

---

### Task 5: Modify `finishClassic` for squash-tag-merge release path

**Files:**
- Modify: `internal/hotfix/hotfix.go:306-405`

This task replaces the release logic in `finishClassic`. When `--release` is active, instead of merging-then-tagging-main, the flow becomes: squash → force push → tag → push tag → merge PR (forced `--merge` strategy) → cleanup.

- [ ] **Step 1: Write the failing test**

In `internal/hotfix/hotfix_test.go`, add a test that verifies the squash-tag-merge sequence. The test uses a fake `gh` that records which commands are called and in what order:

```go
func TestFinishClassicReleaseSquashesTagsMerges(t *testing.T) {
	repoDir := initHotfixRepoWithRemoteAndTag(t)
	orderLog := filepath.Join(t.TempDir(), "order.log")
	installReleaseGH(t, orderLog)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.Interactive = false
	u.AutoConfirm = true

	svc := &Service{
		Git:            git.New(r, repoDir),
		GH:             gh.New(r),
		UI:             u,
		Config:         config.Defaults(),
		RunTitlePrompt: ui.RunTitlePrompt,
		RunProgress:    ui.RunProgress,
	}

	// Add two commits on the hotfix branch
	if err := os.WriteFile(filepath.Join(repoDir, "fix1.txt"), []byte("fix1"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "wip: first attempt")
	if err := os.WriteFile(filepath.Join(repoDir, "fix2.txt"), []byte("fix2"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "wip: second attempt")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	err := svc.Finish(FinishOpts{Release: true})
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}

	// Verify squash: hotfix branch should have exactly 1 commit beyond the tag
	// (We're on main now after finish, so check the tag)
	commitCount := runGit(t, repoDir, "rev-list", "--count", "v1.0.0..v1.0.1")
	if commitCount != "1" {
		t.Errorf("commits between tags = %q, want 1 (squashed)", commitCount)
	}

	// Verify tag v1.0.1 exists
	tags := runGit(t, repoDir, "tag", "-l", "v1.0.1")
	if tags != "v1.0.1" {
		t.Errorf("expected tag v1.0.1, got %q", tags)
	}

	// Verify the squashed commit message starts with "hotfix:"
	msg := runGit(t, repoDir, "log", "-1", "--format=%s", "v1.0.1")
	if !strings.HasPrefix(msg, "hotfix:") {
		t.Errorf("squashed commit message = %q, want prefix 'hotfix:'", msg)
	}

	// Verify gh pr merge was called with --merge strategy
	orderBytes, err := os.ReadFile(orderLog)
	if err != nil {
		t.Fatalf("ReadFile(order.log) error = %v", err)
	}
	order := string(orderBytes)
	if !strings.Contains(order, "merge --merge") {
		t.Errorf("expected --merge strategy in gh commands, got: %s", order)
	}

	// Verify output mentions the released tag
	if !strings.Contains(out.String(), "v1.0.1") {
		t.Errorf("output = %q, want mention of v1.0.1", out.String())
	}
}
```

Also add these helper functions:

```go
func initHotfixRepoWithRemoteAndTag(t *testing.T) string {
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
	runGit(t, repoDir, "tag", "v1.0.0")
	runGit(t, repoDir, "push", "origin", "main")
	runGit(t, repoDir, "push", "origin", "v1.0.0")
	runGit(t, repoDir, "checkout", "-b", "hotfix/test")
	runGit(t, repoDir, "push", "-u", "origin", "hotfix/test")

	return repoDir
}

func installReleaseGH(t *testing.T, orderLog string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
log() { echo "$*" >> "` + orderLog + `"; }
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo '{"number":1,"title":"Fix crash","state":"OPEN","url":"https://example.com/pr/1","isDraft":false}'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  echo '[{"name":"ci","state":"SUCCESS","bucket":"pass"}]'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  log "merge $3"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hotfix/ -run TestFinishClassicReleaseSquashesTagsMerges -v`
Expected: FAIL — the current code merges then tags main, so `v1.0.1` won't be a single commit after squash, and `--merge` strategy won't be forced.

- [ ] **Step 3: Rewrite `finishClassic` release path**

In `internal/hotfix/hotfix.go`, replace the `finishClassic` method. The non-release path stays the same. The release path becomes the squash-tag-merge flow:

```go
// finishClassic runs the hotfix finish workflow with print-style output.
func (s *Service) finishClassic(branch string, opts FinishOpts, qGH *gh.GH) error {
	pr, err := qGH.GetCurrentPR()
	if err != nil {
		return workflow.CurrentPRError(err, "git sf hotfix publish")
	}

	if !opts.Force {
		checks, err := qGH.GetPRChecks()
		if err != nil {
			return fmt.Errorf("could not fetch PR checks: %w", err)
		}
		for _, c := range checks {
			switch {
			case gh.CheckIsPending(c):
				s.UI.Warning(c.Name + " — " + c.State)
			case gh.CheckAllowsMerge(c):
				s.UI.Success(c.Name + " — " + c.State)
			default:
				s.UI.Error(c.Name + " — " + c.State)
			}
		}
		failing, pending := gh.ClassifyChecks(checks)
		if len(failing) > 0 {
			return fmt.Errorf("PR checks failed: %s (use --force to override)", strings.Join(failing, ", "))
		}
		if len(pending) > 0 {
			return fmt.Errorf("PR checks still running: %s (use --force to override)", strings.Join(pending, ", "))
		}
	}

	confirmed, err := s.UI.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
	if err != nil {
		return err
	}
	if !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	doRelease := opts.Release || s.Config.HotfixAutoRelease

	if doRelease {
		// Squash-Tag-Merge flow
		qGit := s.Git.ForQuery()

		// Squash
		base, err := qGit.MergeBase(s.Config.MainBranch, "HEAD")
		if err != nil {
			return fmt.Errorf("could not find merge base: %w", err)
		}
		if err := s.Git.ResetSoft(base); err != nil {
			return fmt.Errorf("could not squash commits: %w", err)
		}
		squashMsg := "hotfix: " + pr.Title
		if err := s.Git.CommitWithMessage(squashMsg); err != nil {
			return fmt.Errorf("could not create squashed commit: %w", err)
		}
		s.UI.Success("Squashed commits")

		// Force push
		if err := s.Git.ForcePush(branch); err != nil {
			return fmt.Errorf("could not force push: %w", err)
		}
		s.UI.Success("Force pushed " + branch)

		// Compute version
		tag, err := qGit.LatestTag(s.Config.TagPrefix)
		if err != nil {
			return err
		}
		current, err := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
		if err != nil {
			return err
		}
		next, err := current.Bump("patch")
		if err != nil {
			return fmt.Errorf("could not bump version: %w", err)
		}
		newTag := next.FormatWithPrefix(s.Config.TagPrefix)

		// Tag
		if err := s.Git.Tag(newTag); err != nil {
			return err
		}
		s.UI.Success("Tagged " + newTag)

		// Push tag
		if err := s.Git.PushTag(newTag); err != nil {
			return err
		}
		s.UI.Success("Pushed tag to origin")

		// Merge PR with --merge strategy and custom subject
		mergeSubject := fmt.Sprintf("Merge hotfix %s", newTag)
		if err := s.GH.MergePRWithMessage("merge", mergeSubject, ""); err != nil {
			return err
		}
		s.UI.Success("PR merged (merge)")

		// Cleanup
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
			s.UI.Warning(fmt.Sprintf("Could not delete remote branch %s: %s", branch, err))
		} else {
			s.UI.Success("Deleted branch " + branch + " (remote)")
		}

		s.UI.Result("Hotfix released " + newTag)
		return nil
	}

	// Non-release path — unchanged
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
		s.UI.Warning(fmt.Sprintf("Could not delete remote branch %s: %s", branch, err))
	} else {
		s.UI.Success("Deleted branch " + branch + " (remote)")
	}

	s.UI.Result("Done.")
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hotfix/ -run TestFinishClassicReleaseSquashesTagsMerges -v`
Expected: PASS

- [ ] **Step 5: Run all existing hotfix tests to verify no regressions**

Run: `go test ./internal/hotfix/ -v`
Expected: All tests pass (existing non-release tests should be unaffected).

- [ ] **Step 6: Commit**

```bash
git add internal/hotfix/hotfix.go internal/hotfix/hotfix_test.go
git commit -m "feat(hotfix): squash-tag-merge release path in finishClassic"
```

---

### Task 6: Modify `finishInteractive` for squash-tag-merge release path

**Files:**
- Modify: `internal/hotfix/hotfix.go:211-303`

This task replaces the interactive release path with the same squash-tag-merge flow, using the interactive step definitions and progress callbacks.

- [ ] **Step 1: Rewrite `finishInteractive` release path**

Replace the `finishInteractive` method. When `doRelease` is true, it uses its own complete step sequence instead of composing with `workflow.FinishWorkflow`:

```go
// finishInteractive runs the hotfix finish workflow with a Bubble Tea progress view.
// It prompts for confirmation before launching the progress view.
func (s *Service) finishInteractive(branch string, opts FinishOpts, qGH *gh.GH) error {
	pr, err := qGH.GetCurrentPR()
	if err != nil {
		return workflow.CurrentPRError(err, "git sf hotfix publish")
	}
	s.UI.Info(fmt.Sprintf("Found PR #%d — %q", pr.Number, pr.Title))

	ok, err := s.UI.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
	if err != nil {
		return err
	}
	if !ok {
		s.UI.Muted("Aborted.")
		return nil
	}

	doRelease := opts.Release || s.Config.HotfixAutoRelease

	if doRelease {
		defs := []ui.StepDef{
			{Label: "Check CI"},
			{Label: "Squash commits"},
			{Label: "Force push"},
			{Label: "Create patch tag"},
			{Label: "Push tag"},
			{Label: "Merge PR"},
			{Label: "Switch to " + s.Config.MainBranch},
			{Label: "Pull latest"},
			{Label: "Delete local branch"},
			{Label: "Delete remote branch"},
		}
		if opts.Force {
			defs[0].Label = "Check CI (skipped)"
		}

		var releasedTag string
		err = s.RunProgress("git sf hotfix finish", branch, defs, func(ctx context.Context, cb ui.StepCallbacks) error {
			ctxGit := s.Git.WithContext(ctx)
			ctxGH := s.GH.WithContext(ctx)

			// Check CI
			cb.Start()
			if !opts.Force {
				checks, err := ctxGH.ForQuery().GetPRChecks()
				if err != nil {
					cb.Fail(fmt.Sprintf("could not fetch PR checks: %s", err))
					return fmt.Errorf("could not fetch PR checks: %w", err)
				}
				failing, pending := gh.ClassifyChecks(checks)
				if len(failing) > 0 {
					errMsg := fmt.Sprintf("PR checks failed: %s (use --force to override)", strings.Join(failing, ", "))
					cb.Fail(errMsg)
					return fmt.Errorf("%s", errMsg)
				}
				if len(pending) > 0 {
					errMsg := fmt.Sprintf("PR checks still running: %s (use --force to override)", strings.Join(pending, ", "))
					cb.Fail(errMsg)
					return fmt.Errorf("%s", errMsg)
				}
			}
			cb.Done()

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Squash commits
			cb.Start()
			base, err := ctxGit.ForQuery().MergeBase(s.Config.MainBranch, "HEAD")
			if err != nil {
				cb.Fail(err.Error())
				return fmt.Errorf("could not find merge base: %w", err)
			}
			if err := ctxGit.ResetSoft(base); err != nil {
				cb.Fail(err.Error())
				return fmt.Errorf("could not squash commits: %w", err)
			}
			squashMsg := "hotfix: " + pr.Title
			if err := ctxGit.CommitWithMessage(squashMsg); err != nil {
				cb.Fail(err.Error())
				return fmt.Errorf("could not create squashed commit: %w", err)
			}
			cb.Done()

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Force push
			if err := cb.Run(func() error { return ctxGit.ForcePush(branch) }); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Create patch tag
			cb.Start()
			tag, err := ctxGit.ForQuery().LatestTag(s.Config.TagPrefix)
			if err != nil {
				cb.Fail(err.Error())
				return err
			}
			current, err := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
			if err != nil {
				cb.Fail(err.Error())
				return err
			}
			next, err := current.Bump("patch")
			if err != nil {
				cb.Fail(err.Error())
				return err
			}
			newTag := next.FormatWithPrefix(s.Config.TagPrefix)
			if err := ctxGit.Tag(newTag); err != nil {
				cb.Fail(err.Error())
				return err
			}
			cb.Done()

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Push tag
			if err := cb.Run(func() error { return ctxGit.PushTag(newTag) }); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Merge PR with --merge strategy
			mergeSubject := fmt.Sprintf("Merge hotfix %s", newTag)
			if err := cb.Run(func() error { return ctxGH.MergePRWithMessage("merge", mergeSubject, "") }); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Switch to main
			if err := cb.Run(func() error { return ctxGit.Checkout(s.Config.MainBranch) }); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Pull latest
			if err := cb.Run(func() error { return ctxGit.Pull() }); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Delete local branch
			if err := cb.Run(func() error { return ctxGit.DeleteLocalBranch(branch) }); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Delete remote branch (soft fail)
			if err := cb.RunSoftFail(func() error { return ctxGit.DeleteRemoteBranch(branch) }); err != nil {
				return err
			}

			releasedTag = newTag
			return nil
		})
		if err != nil {
			return err
		}
		s.UI.Result("Hotfix released " + releasedTag)
		return nil
	}

	// Non-release path — unchanged, uses shared FinishWorkflow
	defs := workflow.FinishStepDefs(s.Config.MainBranch)
	if opts.Force {
		defs[0].Label = "Check CI (skipped)"
	}

	commonFinish := workflow.FinishWorkflow(s.Git, s.GH, branch, s.Config.MainBranch, s.Config.MergeStrategy, opts.Force)
	err = s.RunProgress("git sf hotfix finish", branch, defs, commonFinish)
	if err != nil {
		return err
	}
	s.UI.Result("Hotfix complete!")
	return nil
}
```

- [ ] **Step 2: Run all hotfix tests**

Run: `go test ./internal/hotfix/ -v`
Expected: All tests pass.

- [ ] **Step 3: Run the full unit test suite**

Run: `go test ./internal/... -v`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/hotfix/hotfix.go
git commit -m "feat(hotfix): squash-tag-merge release path in finishInteractive"
```

---

### Task 7: Verify non-release hotfix finish is unchanged

**Files:**
- Test: `internal/hotfix/hotfix_test.go`

- [ ] **Step 1: Write a test for non-release finish**

In `internal/hotfix/hotfix_test.go`, add:

```go
func TestFinishClassicNonReleaseUsesConfiguredStrategy(t *testing.T) {
	repoDir := initHotfixRepoWithRemoteAndTag(t)
	orderLog := filepath.Join(t.TempDir(), "order.log")
	installReleaseGH(t, orderLog)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.Interactive = false
	u.AutoConfirm = true

	cfg := config.Defaults()
	cfg.MergeStrategy = "squash"

	svc := &Service{
		Git:            git.New(r, repoDir),
		GH:             gh.New(r),
		UI:             u,
		Config:         cfg,
		RunTitlePrompt: ui.RunTitlePrompt,
		RunProgress:    ui.RunProgress,
	}

	// Add a commit on the hotfix branch and push
	if err := os.WriteFile(filepath.Join(repoDir, "fix.txt"), []byte("fix"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "the fix")
	runGit(t, repoDir, "push", "origin", "hotfix/test")

	err := svc.Finish(FinishOpts{Release: false})
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}

	// Verify gh pr merge was called with --squash (not --merge)
	orderBytes, err := os.ReadFile(orderLog)
	if err != nil {
		t.Fatalf("ReadFile(order.log) error = %v", err)
	}
	order := string(orderBytes)
	if !strings.Contains(order, "merge --squash") {
		t.Errorf("expected --squash strategy in gh commands, got: %s", order)
	}

	// Verify no new tags were created
	tags := runGit(t, repoDir, "tag", "-l", "v1.0.1")
	if tags != "" {
		t.Errorf("expected no v1.0.1 tag, got %q", tags)
	}

	// Verify output says "Done." not "Hotfix released"
	if strings.Contains(out.String(), "released") {
		t.Errorf("output should not mention release: %q", out.String())
	}
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/hotfix/ -run TestFinishClassicNonReleaseUsesConfiguredStrategy -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/hotfix/hotfix_test.go
git commit -m "test(hotfix): verify non-release finish uses configured merge strategy"
```

---

### Task 8: Update documentation — `docs/simple-flow.md`

**Files:**
- Modify: `docs/simple-flow.md`

- [ ] **Step 1: Update the hotfix workflow section (step 4)**

Replace lines 117-122 (`4. **Finish with a release.**` section) with:

```markdown
4. **Finish with a release.** Squashes the branch to a single commit, tags it with the next patch version, force-pushes,
   then merges the PR into `main` via a merge commit. The tag lives on the hotfix branch — it contains only released code
   plus the fix. The `--release` flag (or [`hotfix_auto_release`](../README.md#configuration) in config) handles this automatically.

   ```bash
   git sf hotfix finish --release
   ```

   The resulting git graph:

   ```
   (v1.2.3) A --- B --- C --- M (main)     ← merge commit
                   \           /
                    `-- D (v1.2.4)'         ← squashed hotfix commit
   ```
```

- [ ] **Step 2: Update the "Key point" callout (line 130-131)**

Replace:

```markdown
> [!IMPORTANT]
> **Key point:** The hotfix branches from the *tag*, not from `main`. This guarantees the hotfix contains only released
> code plus the fix — no unreleased feature work leaks in.
```

With:

```markdown
> [!IMPORTANT]
> **Key point:** The hotfix branches from the *tag*, not from `main`, and the release tag is placed on the hotfix branch
> before merging. This guarantees the tag contains only released code plus the fix — no unreleased feature work leaks in.
> `git log --first-parent main` shows a clean linear history of merge commits.
```

- [ ] **Step 3: Update the "Hotfix fast path" section (lines 224-227)**

Replace:

```markdown
### Hotfix fast path

When a hotfix merges with `--release`, the patch tag is created immediately. This means the hotfix goes from dev →
beta → production in a single `git sf hotfix finish --release`, and your CI handles each stage automatically.
```

With:

```markdown
### Hotfix fast path

When a hotfix finishes with `--release`, the branch is squashed to a single commit, tagged with the next patch version,
and force-pushed before merging into `main`. The tag push triggers the production release immediately — before the merge
to `main` even happens. This means the hotfix goes from dev → production in seconds, and the subsequent merge to `main`
triggers the beta/RC channel. Your CI handles each stage automatically.
```

- [ ] **Step 4: Update the Core Principles hotfix bullet (line 27-28)**

Replace:

```markdown
- **Hotfixes branch from tags** — Branch from the release tag, fix, merge back to `main`, tag a patch release. The fix
  contains only released code plus the change.
```

With:

```markdown
- **Hotfixes branch from tags** — Branch from the release tag, fix, squash, tag the branch, and merge back to `main`.
  The release tag lives on the hotfix branch — it contains only released code plus the fix.
```

- [ ] **Step 5: Update line 18 overview text**

Replace:

```markdown
All work branches from `main`. Releases are tags on `main`. Hotfixes branch from the latest release tag, merge back to `main`,
and get their own patch tag. There are no long-lived branches other than `main`.
```

With:

```markdown
All work branches from `main`. Releases are tags on `main`. Hotfixes branch from the latest release tag — the patch tag
is placed on the hotfix branch before merging back to `main`, so it never includes unreleased work. There are no
long-lived branches other than `main`.
```

- [ ] **Step 6: Commit**

```bash
git add docs/simple-flow.md
git commit -m "docs: update simple-flow.md for squash-tag-merge hotfix model"
```

---

### Task 9: Update documentation — `README.md`

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the `--release` flag description (lines 137-138)**

Replace:

```markdown
`--release` on `finish` (or `hotfix_auto_release` in config) automatically bumps the patch version, creates a new tag,
and pushes it — so `v1.2.3` becomes `v1.2.4` without a separate `release` command.
```

With:

```markdown
`--release` on `finish` (or `hotfix_auto_release` in config) squashes the hotfix branch to a single commit, tags it with
the next patch version, force-pushes, then merges the PR into `main` via a merge commit. The tag lives on the hotfix
branch — so `v1.2.3` becomes `v1.2.4` containing only released code plus the fix. No separate `release` command needed.
```

- [ ] **Step 2: Update the feature finish description (lines 113-114)**

Replace:

```markdown
`finish` verifies all CI checks pass before merging (override with `--force`), prompts for confirmation, merges using
your configured strategy (`squash`/`merge`/`rebase`), then deletes the local and remote branches.
```

With:

```markdown
`finish` verifies all CI checks pass before merging (override with `--force`), prompts for confirmation, merges using
your configured strategy (`squash`/`merge`/`rebase`), then deletes the local and remote branches. For hotfix
`finish --release`, the merge strategy is always `merge` (to preserve the tag as an ancestor of `main`).
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README for squash-tag-merge hotfix model"
```

---

### Task 10: Update skill files

**Files:**
- Modify: `git-sf-skill/skills/git-sf-workflow/SKILL.md`
- Modify: `.claude-plugin/plugin/skills/git-sf-workflow/SKILL.md`

- [ ] **Step 1: Update the skill workflow note (line 41)**

In both files, replace:

```markdown
- Hotfix: `git sf hotfix finish --release` auto-tags patch. `hotfix_auto_release: true` config does the same.
```

With:

```markdown
- Hotfix: `git sf hotfix finish --release` squashes the branch, tags the squashed commit with the next patch version, force-pushes, then merges via merge commit. The tag lives on the hotfix branch (not on main HEAD). `hotfix_auto_release: true` config does the same.
```

- [ ] **Step 2: Verify both files are identical**

Run: `diff git-sf-skill/skills/git-sf-workflow/SKILL.md .claude-plugin/plugin/skills/git-sf-workflow/SKILL.md`
Expected: No differences.

- [ ] **Step 3: Commit**

```bash
git add git-sf-skill/skills/git-sf-workflow/SKILL.md .claude-plugin/plugin/skills/git-sf-workflow/SKILL.md
git commit -m "docs: update skill files for squash-tag-merge hotfix model"
```

---

### Task 11: Update test scenarios

**Files:**
- Modify: `.claude-plugin/plugin/tests/scenarios.md`

- [ ] **Step 1: Update Scenario B expected behavior**

In Scenario B's GREEN results section (lines 82-85), replace:

```markdown
### Scenario B — PASS
- Correctly identified as hotfix (production issue)
- Used `git sf hotfix start fix-payment-null-pointer`
- Used `git sf hotfix publish` and `git sf hotfix finish --release`
- Used raw git for commits (correct)
```

With:

```markdown
### Scenario B — PASS
- Correctly identified as hotfix (production issue)
- Used `git sf hotfix start fix-payment-null-pointer`
- Used `git sf hotfix publish` and `git sf hotfix finish --release`
- Used raw git for commits (correct)
- Note: `--release` squash-tag-merges the branch (squash → force push → tag → push tag → merge commit)
```

- [ ] **Step 2: Commit**

```bash
git add .claude-plugin/plugin/tests/scenarios.md
git commit -m "docs: update test scenarios for squash-tag-merge hotfix model"
```

---

### Task 12: Run full test suite and lint

**Files:** None (verification only)

- [ ] **Step 1: Run all unit tests**

Run: `go test ./internal/... -v`
Expected: All tests pass.

- [ ] **Step 2: Run integration tests**

Run: `go test -tags integration ./test/... -v -count=1`
Expected: All tests pass (integration tests don't cover hotfix --release against real remote).

- [ ] **Step 3: Run lint**

Run: `make lint`
Expected: No lint errors.

- [ ] **Step 4: Build the binary**

Run: `go build -o git-sf .`
Expected: Clean build with no errors.

- [ ] **Step 5: Commit any fixes from lint/test**

Only if needed. Otherwise skip.
