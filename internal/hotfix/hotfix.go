// Package hotfix implements the hotfix branch lifecycle. Unlike features,
// hotfixes branch from the latest release tag (not main) and can optionally
// trigger a patch release on finish.
package hotfix

import (
	"context"
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

	tag, err := qGit.LatestTag(s.Config.TagPrefix)
	if err != nil {
		return fmt.Errorf("no tags found. Create an initial release first with 'git sf release'")
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

	defs := workflow.FinishStepDefs(s.Config.MainBranch)
	if opts.Force {
		defs[0].Label = "Check CI (skipped)"
	}

	doRelease := opts.Release || s.Config.HotfixAutoRelease
	if doRelease {
		defs = append(defs,
			ui.StepDef{Label: "Create patch tag"},
			ui.StepDef{Label: "Push tag"},
		)
	}

	var releasedTag string
	commonFinish := workflow.FinishWorkflow(s.Git, s.GH, branch, s.Config.MainBranch, s.Config.MergeStrategy, opts.Force)
	err = s.RunProgress("git sf hotfix finish", branch, defs, func(ctx context.Context, cb ui.StepCallbacks) error {
		if err := commonFinish(ctx, cb); err != nil {
			return err
		}

		if !doRelease {
			return nil
		}

		ctxGit := s.Git.WithContext(ctx)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Create patch tag — LatestTag is a read-only query that must
		// execute even in dry-run mode to resolve the next version.
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

		releasedTag = newTag
		return nil
	})
	if err != nil {
		return err
	}
	if releasedTag != "" {
		s.UI.Result("Hotfix released " + releasedTag)
	} else {
		s.UI.Result("Hotfix complete!")
	}
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

	s.UI.Info(fmt.Sprintf("Merging PR #%d — %q", pr.Number, pr.Title))

	if err := s.GH.MergePR(s.Config.MergeStrategy); err != nil {
		return err
	}
	if err := s.GH.VerifyPRMerged(); err != nil {
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

	// Auto-release if --release flag or config
	if opts.Release || s.Config.HotfixAutoRelease {
		tag, err := s.Git.ForQuery().LatestTag(s.Config.TagPrefix)
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
