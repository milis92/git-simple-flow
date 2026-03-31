// Package hotfix implements the hotfix branch lifecycle. Unlike features,
// hotfixes branch from the latest release tag (not main) and can optionally
// trigger a patch release on finish.
package hotfix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/milis92/git-simple-flow/internal/version"
	"github.com/milis92/git-simple-flow/internal/workflow"
)

// Service orchestrates git, GitHub CLI, UI, and config to execute
// the hotfix branch workflow.
type Service struct {
	Git            *git.Git
	GH             *gh.GH
	UI             *ui.UI
	Config         config.Config
	RunTitlePrompt func(string, bool) (ui.InputPromptResult, error)
	RunProgress    func(string, string, []ui.StepDef, func(context.Context, ui.StepCallbacks) error) error
}

// StartOpts configures hotfix branch creation.
type StartOpts struct {
	// DraftPR, when true, creates a draft PR immediately after branching.
	DraftPR bool
	// Title overrides the auto-generated PR title.
	Title string
}

// PublishOpts configures PR creation for an existing hotfix branch.
type PublishOpts struct {
	// Title overrides the auto-generated PR title.
	Title string
	// Body is the PR description.
	Body string
}

// FinishOpts configures hotfix branch completion.
type FinishOpts struct {
	// Force skips CI check verification before merging.
	Force bool
	// Release triggers a patch version tag after merging.
	Release bool
}

// Start creates a new hotfix branch from the latest release tag. It checks out
// the tag, creates the branch, and optionally creates a draft PR.
func (s *Service) Start(name string, opts StartOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	// Use query-mode runner for read-only preflight checks so they execute
	// even during --dry-run.
	qGit := s.Git.ForQuery()

	if err := qGit.CheckIsRepo(); err != nil {
		return err
	}
	if err := qGit.CheckCleanTree(); err != nil {
		return err
	}

	// Best-effort fetch so we see the latest tags from the remote.
	// Ignored when no remote is configured (e.g. local-only repos).
	hasRemote := qGit.Fetch() == nil

	// Prefer origin/main for fresh tag data; fall back to local main
	// when no remote is configured.
	tag, err := qGit.LatestTagOnBranch(s.Config.TagPrefix, "origin/"+s.Config.MainBranch)
	if err != nil {
		tag, err = qGit.LatestTagOnBranch(s.Config.TagPrefix, s.Config.MainBranch)
	}
	if err != nil {
		return fmt.Errorf("no tags found. Create an initial release first with 'git sf release'")
	}

	// Guard against local-only tags that were never pushed. A hotfix must
	// branch from a published release, not an unpushed local tag.
	if hasRemote {
		published, checkErr := qGit.TagExistsOnRemote(tag)
		if checkErr != nil {
			return fmt.Errorf("could not verify tag %s on remote: %w", tag, checkErr)
		}
		if !published {
			return fmt.Errorf("tag %s exists locally but not on origin — push it first or use a published release", tag)
		}
	}

	branchName := s.Config.HotfixPrefix + name

	// When a draft PR is requested, resolve the interactive prompt before
	// mutating any repo state so a user cancellation leaves no partial branch.
	var draftTitle string
	if opts.DraftPR || s.Config.DraftPROnStart {
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := s.GH.ForQuery().CheckAuthenticated(); err != nil {
			return err
		}
		t, _, err := workflow.ResolvePRInput(s.UI, s.RunTitlePrompt, branchName, s.Config.HotfixPrefix, opts.Title, "", false)
		if err != nil {
			return err
		}
		draftTitle = t
	}

	if err := s.Git.Checkout(tag); err != nil {
		return err
	}
	s.UI.Success("Checked out " + tag)

	if err := s.Git.CreateBranch(branchName); err != nil {
		return err
	}
	s.UI.Success("Created branch " + branchName)

	if draftTitle != "" {
		if err := s.Git.Push(branchName); err != nil {
			return err
		}
		pr, err := s.GH.CreatePR(s.Config.MainBranch, draftTitle, "", true)
		if err != nil {
			return err
		}
		s.UI.Success("Created draft PR: " + pr.URL)
	}

	s.UI.Result("Ready to work. When done: git sf hotfix publish")
	return nil
}

// Publish pushes the current hotfix branch and creates a PR to main.
// It warns if the working tree is dirty but does not block.
func (s *Service) Publish(opts PublishOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := gh.CheckGHInstalled(); err != nil {
		return err
	}

	// Use query-mode runners for read-only preflight checks so they execute
	// even during --dry-run.
	qGit := s.Git.ForQuery()
	qGH := s.GH.ForQuery()

	if err := qGH.CheckAuthenticated(); err != nil {
		return err
	}

	clean, err := qGit.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		s.UI.Warning("You have uncommitted changes that won't be included in the PR.")
	}

	branch, err := qGit.CurrentBranch()
	if err != nil {
		return err
	}

	title, body, err := workflow.ResolvePRInput(s.UI, s.RunTitlePrompt, branch, s.Config.HotfixPrefix, opts.Title, opts.Body, true)
	if err != nil {
		return err
	}

	if err := s.Git.Push(branch); err != nil {
		return err
	}
	s.UI.Success("Pushed " + branch)

	pr, err := s.GH.CreatePR(s.Config.MainBranch, title, body, false)
	if err != nil {
		return err
	}
	s.UI.Success("Created PR: " + pr.URL)

	s.UI.Result("PR is up. When ready: git sf hotfix finish")
	return nil
}

// Finish merges the current hotfix PR and cleans up. It runs preflight checks,
// detects the current branch, then routes to finishInteractive (Bubble Tea
// progress) or finishClassic (print-style) based on whether the UI is in
// interactive mode. If Release is set or hotfix_auto_release is configured,
// a patch version tag is created after merging.
func (s *Service) Finish(opts FinishOpts) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}
	if err := gh.CheckGHInstalled(); err != nil {
		return err
	}

	// Use query-mode runners for read-only preflight checks so they execute
	// even during --dry-run.
	qGit := s.Git.ForQuery()
	qGH := s.GH.ForQuery()

	if err := qGH.CheckAuthenticated(); err != nil {
		return err
	}
	if err := qGit.CheckCleanTree(); err != nil {
		return err
	}

	branch, err := qGit.CurrentBranch()
	if err != nil {
		return err
	}

	if !strings.HasPrefix(branch, s.Config.HotfixPrefix) {
		return fmt.Errorf("not on a hotfix branch (current: %s)", branch)
	}

	if s.UI.Interactive {
		return s.finishInteractive(branch, opts, qGH)
	}
	return s.finishClassic(branch, opts, qGH)
}

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
		pf, err := s.releasePreflight(branch, s.Git.ForQuery(), qGH)
		if err != nil {
			return err
		}
		if pf == nil {
			return nil // retry path handled it
		}

		defs := []ui.StepDef{
			{Label: "Check CI"},
			{Label: "Squash commits"},
			{Label: "Force push"},
			{Label: "Merge PR"},
			{Label: "Verify merge"},
			{Label: "Create patch tag"},
			{Label: "Push tag"},
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

			// Squash + force push (skip if already squashed from a previous attempt).
			cb.Start()
			didSquash, err := s.squashForRelease(ctxGit, pf.tagSHA, pr.Title)
			if err != nil {
				cb.Fail(err.Error())
				return err
			}
			cb.Done()

			if didSquash {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if err := cb.Run(func() error { return ctxGit.ForcePush(branch) }); err != nil {
					return err
				}
			} else {
				cb.Start()
				cb.Done()
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			squashedSHA, err := ctxGit.ForQuery().RevParse("HEAD")
			if err != nil {
				return fmt.Errorf("could not resolve HEAD after squash: %w", err)
			}

			newTag, err := s.computeReleaseTag(pf.latestTag)
			if err != nil {
				return err
			}

			// Merge PR with --merge strategy, pinned to the squashed SHA.
			// --match-head-commit rejects the merge if the branch moved after force-push.
			mergeSubject := fmt.Sprintf("Merge hotfix %s", newTag)
			if err := cb.Run(func() error {
				return ctxGH.MergePRWithMessage("merge", mergeSubject, "", squashedSHA)
			}); err != nil {
				return err
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Verify PR was actually merged before creating the tag.
			// Tag must not be published until the merge is confirmed.
			cb.Start()
			if err := ctxGH.VerifyPRMerged(); err != nil {
				if !errors.Is(err, gh.ErrPRNotMerged) {
					cb.Fail(fmt.Sprintf("post-merge verification failed: %s", err))
					return fmt.Errorf("post-merge verification failed: %w", err)
				}
				// PR is queued — skip tag + cleanup steps
				cb.SkipStep("PR queued — waiting")
				for range 5 {
					cb.Start()
					cb.SkipStep("PR queued — skipped")
				}
				s.UI.Warning("PR is queued or pending — run 'git sf hotfix finish --release' again after merge completes")
				return nil
			}
			cb.Done()

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Create patch tag — PR is confirmed merged.
			cb.Start()
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

			releasedTag = newTag

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

			return nil
		})
		if err != nil {
			return err
		}
		if releasedTag != "" {
			s.UI.Result("Hotfix released " + releasedTag)
		}
		return nil
	}

	// Non-release path — uses shared FinishWorkflow with merge verification
	defs := workflow.FinishStepDefs(s.Config.MainBranch)
	if opts.Force {
		defs[0].Label = "Check CI (skipped)"
	}

	var merged bool
	commonFinish := workflow.FinishWorkflow(s.Git, s.GH, branch, s.Config.MainBranch, s.Config.MergeStrategy, opts.Force, &merged, len(defs))
	err = s.RunProgress("git sf hotfix finish", branch, defs, commonFinish)
	if err != nil {
		return err
	}
	if !merged {
		s.UI.Warning("PR is queued or pending — re-run 'git sf hotfix finish' after merge completes")
		return nil
	}
	s.UI.Result("Hotfix complete!")
	return nil
}

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
		pf, err := s.releasePreflight(branch, s.Git.ForQuery(), s.GH.ForQuery())
		if err != nil {
			return err
		}
		if pf == nil {
			return nil // retry path handled it
		}

		// Squash + force push.
		didSquash, err := s.squashForRelease(s.Git, pf.tagSHA, pr.Title)
		if err != nil {
			return err
		}
		if didSquash {
			s.UI.Success("Squashed commits")
			if err := s.Git.ForcePush(branch); err != nil {
				return fmt.Errorf("could not force push: %w", err)
			}
			s.UI.Success("Force pushed " + branch)
		} else {
			s.UI.Muted("Already squashed, skipping")
		}

		squashedSHA, err := s.Git.ForQuery().RevParse("HEAD")
		if err != nil {
			return fmt.Errorf("could not resolve HEAD after squash: %w", err)
		}

		newTag, err := s.computeReleaseTag(pf.latestTag)
		if err != nil {
			return err
		}

		// Merge PR with --merge strategy, pinned to the squashed SHA.
		mergeSubject := fmt.Sprintf("Merge hotfix %s", newTag)
		if err := s.GH.MergePRWithMessage("merge", mergeSubject, "", squashedSHA); err != nil {
			return err
		}
		s.UI.Success("PR merge requested")

		// Verify PR was actually merged before creating the tag.
		// Tag must not be published until the merge is confirmed.
		if err := s.GH.VerifyPRMerged(); err != nil {
			if !errors.Is(err, gh.ErrPRNotMerged) {
				return fmt.Errorf("post-merge verification failed: %w", err)
			}
			s.UI.Warning("PR is queued or pending — run 'git sf hotfix finish --release' again after merge completes")
			return nil
		}
		s.UI.Success("PR merged (merge)")

		// Create and push tag — PR is confirmed merged.
		if err := s.Git.Tag(newTag); err != nil {
			return err
		}
		s.UI.Success("Tagged " + newTag)

		if err := s.Git.PushTag(newTag); err != nil {
			return err
		}
		s.UI.Success("Pushed tag to origin")

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

	// Non-release path
	s.UI.Info(fmt.Sprintf("Merging PR #%d — %q", pr.Number, pr.Title))

	if err := s.GH.MergePR(s.Config.MergeStrategy); err != nil {
		return err
	}

	if err := s.GH.VerifyPRMerged(); err != nil {
		if !errors.Is(err, gh.ErrPRNotMerged) {
			return fmt.Errorf("post-merge verification failed: %w", err)
		}
		s.UI.Warning("PR is queued or pending — re-run 'git sf hotfix finish' after merge completes")
		return nil
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

// releasePreflightResult holds the computed state from release preflight checks.
type releasePreflightResult struct {
	latestTag string
	tagSHA    string
}

// releasePreflight runs the shared preflight checks for the hotfix release flow:
// fetch, sync check, retry detection (returns nil result if retry handled),
// tag lookup scoped to origin/main, and merge-base contamination guard.
func (s *Service) releasePreflight(branch string, qGit *git.Git, qGH *gh.GH) (*releasePreflightResult, error) {
	// Fetch through the normal runner so --dry-run logs the command without
	// executing it. Using the query runner here would mutate remote tracking
	// refs even during a preview.
	if err := s.Git.Fetch(); err != nil {
		return nil, fmt.Errorf("could not fetch: %w", err)
	}

	// Retry detection runs before the sync check because the hotfix branch
	// may have been auto-deleted by GitHub after merge, making the sync
	// check fail on a missing remote ref.
	if verifyErr := qGH.VerifyPRMerged(); verifyErr == nil {
		if err := s.retryTagAndCleanup(branch, qGit, qGH); err != nil {
			return nil, err
		}
		return nil, nil // retry handled
	}

	inSync, err := qGit.IsInSyncWithRemote(branch)
	if err != nil {
		return nil, fmt.Errorf("could not check remote sync: %w", err)
	}
	if !inSync {
		return nil, fmt.Errorf("hotfix branch %s has diverged from remote; pull or reconcile before releasing", branch)
	}

	// Use the latest tag reachable from origin/main as the base version.
	// This prevents off-main hotfix tags from poisoning the preflight.
	latestTag, err := qGit.LatestTagOnBranch(s.Config.TagPrefix, "origin/"+s.Config.MainBranch)
	if err != nil {
		return nil, err
	}
	tagSHA, err := qGit.RevParse(latestTag + "^{commit}")
	if err != nil {
		return nil, fmt.Errorf("could not resolve tag %s: %w", latestTag, err)
	}

	// Ensure hotfix branch has not been contaminated with unreleased main commits.
	// First check: merge-base must equal the base tag (catches merges/rebases).
	mergeBase, err := qGit.MergeBase("origin/"+s.Config.MainBranch, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("could not find merge base: %w", err)
	}
	if mergeBase != tagSHA {
		return nil, fmt.Errorf("hotfix branch contains unreleased main commits (was rebased or merged with %s); cannot auto-release", s.Config.MainBranch)
	}

	// Second check: no cherry-picked commits from unreleased main (catches
	// cherry-picks which don't change the merge-base).
	hasCherries, err := qGit.HasCherryPickedCommits("origin/"+s.Config.MainBranch, "HEAD", tagSHA)
	if err != nil {
		return nil, fmt.Errorf("could not check for cherry-picked commits: %w", err)
	}
	if hasCherries {
		return nil, fmt.Errorf("hotfix branch contains cherry-picked commits from %s; cannot auto-release", s.Config.MainBranch)
	}

	return &releasePreflightResult{latestTag: latestTag, tagSHA: tagSHA}, nil
}

// squashForRelease checks whether the hotfix branch has multiple commits above
// the base tag and, if so, squashes them into a single commit. Returns true if
// a squash was performed (caller must force-push). Does not force-push itself
// because the interactive path needs that as a separate progress step.
func (s *Service) squashForRelease(g *git.Git, tagSHA, prTitle string) (didSquash bool, err error) {
	commitCount, err := g.ForQuery().CommitCount(tagSHA, "HEAD")
	if err != nil {
		return false, fmt.Errorf("could not count commits: %w", err)
	}
	if commitCount <= 1 {
		return false, nil
	}
	if err := g.ResetSoft(tagSHA); err != nil {
		return false, fmt.Errorf("could not squash commits: %w", err)
	}
	if err := g.CommitWithMessage("hotfix: " + prTitle); err != nil {
		return false, fmt.Errorf("could not create squashed commit: %w", err)
	}
	return true, nil
}

// computeReleaseTag parses the latest tag as semver, bumps the patch version,
// and returns the formatted new tag string.
func (s *Service) computeReleaseTag(latestTag string) (string, error) {
	current, err := version.Parse(strings.TrimPrefix(latestTag, s.Config.TagPrefix))
	if err != nil {
		return "", err
	}
	next, err := current.Bump("patch")
	if err != nil {
		return "", err
	}
	return next.FormatWithPrefix(s.Config.TagPrefix), nil
}

// cleanupAfterMerge performs post-merge cleanup when a previous run already
// completed the squash-tag-merge steps but was interrupted before cleanup.
// This is the retry path for queued merges that have since completed.
func (s *Service) cleanupAfterMerge(branch, tag string) error {
	s.UI.Info("Tag " + tag + " already exists and PR is merged — resuming cleanup")

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

	s.UI.Result("Hotfix released " + tag)
	return nil
}

// retryTagAndCleanup is the retry path for a previous run whose merge was
// queued and has since completed. It anchors everything to the PR's head ref
// OID — the immutable SHA that GitHub recorded at merge time — so it is
// independent of local HEAD (which may have moved) and local tags (which may
// exist without having been pushed).
func (s *Service) retryTagAndCleanup(branch string, qGit *git.Git, qGH *gh.GH) error {
	// Get the commit that was actually merged — not local HEAD, which may
	// have moved if someone pushed more commits after the merge.
	prHeadSHA, err := qGH.GetPRHeadSHA()
	if err != nil {
		return fmt.Errorf("could not determine merged commit: %w", err)
	}

	// Derive the base tag from the parent of the merged commit.
	// The squashed hotfix sits directly on top of the base tag, so
	// prHeadSHA^ is the base tag commit regardless of main's current state.
	baseTag, err := qGit.LatestTagOnBranch(s.Config.TagPrefix, prHeadSHA+"^")
	if err != nil {
		return fmt.Errorf("could not determine hotfix base tag: %w", err)
	}
	newTag, err := s.computeReleaseTag(baseTag)
	if err != nil {
		return err
	}

	// Always create and push the tag at the merged commit.
	// A local tag alone is not proof the remote has it (a previous retry
	// may have died between Tag and PushTag), so we always push.
	if _, tagErr := qGit.RevParse(newTag + "^{commit}"); tagErr != nil {
		if err := s.Git.TagAt(newTag, prHeadSHA); err != nil {
			return err
		}
	}
	if err := s.Git.PushTag(newTag); err != nil {
		return err
	}
	s.UI.Success("Tagged and pushed " + newTag)

	return s.cleanupAfterMerge(branch, newTag)
}

// Discard abandons the current hotfix branch. It runs preflight checks,
// detects the current branch, then routes to discardInteractive (Bubble Tea
// progress) or discardClassic (print-style) based on whether the UI is in
// interactive mode.
func (s *Service) Discard(reason string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	// Use query-mode runner for read-only preflight checks so they execute
	// even during --dry-run.
	qGit := s.Git.ForQuery()

	if err := qGit.CheckIsRepo(); err != nil {
		return err
	}
	if err := qGit.CheckCleanTree(); err != nil {
		return err
	}

	branch, err := qGit.CurrentBranch()
	if err != nil {
		return err
	}

	if !strings.HasPrefix(branch, s.Config.HotfixPrefix) {
		return fmt.Errorf("not on a hotfix branch (current: %s)", branch)
	}

	if s.UI.Interactive {
		return s.discardInteractive(branch, reason)
	}
	return s.discardClassic(branch, reason)
}

// discardInteractive runs the hotfix discard workflow using the Bubble Tea
// progress view. It prompts for confirmation before launching the progress view.
func (s *Service) discardInteractive(branch string, reason string) error {
	confirmed, err := s.UI.Confirm("Discard branch " + branch + "?")
	if err != nil {
		return err
	}
	if !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	defs := workflow.DiscardStepDefs(s.Config.MainBranch)
	wf := workflow.DiscardWorkflow(s.Git, s.GH, branch, s.Config.MainBranch, reason)
	if err := s.RunProgress("git sf hotfix discard", branch, defs, wf); err != nil {
		return err
	}

	s.UI.Result("Discarded.")
	return nil
}

// discardClassic runs the existing print-style hotfix discard workflow with a
// confirmation prompt. This is used when the UI is not in interactive mode.
func (s *Service) discardClassic(branch string, reason string) error {
	confirmed, err := s.UI.Confirm("Discard branch " + branch + "?")
	if err != nil {
		return err
	}
	if !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	if err := gh.CheckGHInstalled(); err == nil {
		if err := s.GH.ForQuery().CheckAuthenticated(); err != nil {
			s.UI.Warning("gh not authenticated — skipping PR close")
		} else if err := s.GH.ClosePR(branch, reason); err != nil {
			s.UI.Warning(fmt.Sprintf("Could not close PR: %s", err))
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
	s.UI.Success("Deleted branch " + branch + " (local)")

	if err := s.Git.DeleteRemoteBranch(branch); err != nil {
		s.UI.Warning(fmt.Sprintf("Could not delete remote branch %s: %s", branch, err))
	} else {
		s.UI.Success("Deleted branch " + branch + " (remote)")
	}

	s.UI.Result("Discarded.")
	return nil
}
