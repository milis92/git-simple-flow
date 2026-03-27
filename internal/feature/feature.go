// Package feature implements the feature branch lifecycle: start, publish,
// finish, and discard. A feature branches from main, gets a PR, and merges
// back to main when done.
package feature

import (
	"context"
	"fmt"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/milis92/git-simple-flow/internal/workflow"
)

// Service orchestrates git, GitHub CLI, UI, and config to execute
// the feature branch workflow.
type Service struct {
	Git            *git.Git
	GH             *gh.GH
	UI             *ui.UI
	Config         config.Config
	RunTitlePrompt func(string, bool) (ui.InputPromptResult, error)
	RunProgress    func(string, string, []ui.StepDef, func(context.Context, ui.StepCallbacks) error) error
}

// StartOpts configures feature branch creation.
type StartOpts struct {
	// DraftPR, when true, creates a draft PR immediately after branching.
	DraftPR bool
	// Title overrides the auto-generated PR title (derived from branch name).
	Title string
}

// PublishOpts configures PR creation for an existing feature branch.
type PublishOpts struct {
	// Title overrides the auto-generated PR title.
	Title string
	// Body is the PR description.
	Body string
}

// FinishOpts configures feature branch completion.
type FinishOpts struct {
	// Force skips CI check verification before merging.
	Force bool
}

// Start creates a new feature branch from main. It checks out main, pulls
// the latest changes, and creates the branch. If DraftPR is set (or configured
// globally), it also pushes the branch and creates a draft PR.
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
		title, _, err := workflow.ResolvePRInput(s.UI, s.RunTitlePrompt, branchName, s.Config.FeaturePrefix, opts.Title, "", false)
		if err != nil {
			return err
		}
		if err := s.Git.Push(branchName); err != nil {
			return err
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

// Publish pushes the current feature branch and creates a ready-for-review PR.
// It warns if the working tree is dirty but does not block.
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

	title, body, err := workflow.ResolvePRInput(s.UI, s.RunTitlePrompt, branch, s.Config.FeaturePrefix, opts.Title, opts.Body, true)
	if err != nil {
		return err
	}

	if err := s.Git.Push(branch); err != nil {
		return err
	}
	s.UI.Success("Pushed branch " + branch)

	pr, err := s.GH.CreatePR(s.Config.MainBranch, title, body, false)
	if err != nil {
		return err
	}
	s.UI.Success("Created PR: " + pr.URL)

	s.UI.Result("PR is open. When ready to merge: git sf feature finish")
	return nil
}

// Finish merges the current feature branch's PR and cleans up. It runs
// preflight checks, detects the current branch, then routes to
// finishInteractive (Bubble Tea progress) or finishClassic (print-style)
// based on whether the UI is in interactive mode.
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

	if err := qGit.CheckIsRepo(); err != nil {
		return err
	}
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

// finishInteractive runs the feature finish workflow using the Bubble Tea
// progress view. It prompts for confirmation before launching the progress view.
func (s *Service) finishInteractive(branch string, opts FinishOpts, qGH *gh.GH) error {
	pr, err := qGH.GetCurrentPR()
	if err != nil {
		return workflow.CurrentPRError(err, "git sf feature publish")
	}
	s.UI.Info(fmt.Sprintf("Found PR #%d — %q", pr.Number, pr.Title))

	ok, err := s.UI.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
	if err != nil {
		return err
	}
	if !ok {
		s.UI.Info("Merge cancelled")
		return nil
	}

	defs := workflow.FinishStepDefs(s.Config.MainBranch)
	if opts.Force {
		defs[0].Label = "Check CI (skipped)"
	}

	wf := workflow.FinishWorkflow(s.Git, s.GH, branch, s.Config.MainBranch, s.Config.MergeStrategy, opts.Force)
	if err := s.RunProgress("git sf feature finish", branch, defs, wf); err != nil {
		return err
	}

	s.UI.Result("Feature complete!")
	return nil
}

// finishClassic runs the existing print-style feature finish workflow with a
// confirmation prompt. This is used when the UI is not in interactive mode.
func (s *Service) finishClassic(branch string, opts FinishOpts, qGH *gh.GH) error {
	pr, err := qGH.GetCurrentPR()
	if err != nil {
		return workflow.CurrentPRError(err, "git sf feature publish")
	}
	s.UI.Info(fmt.Sprintf("Found PR #%d — %q", pr.Number, pr.Title))

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
		s.UI.Warning(fmt.Sprintf("Could not delete remote branch %s: %s", branch, err))
	} else {
		s.UI.Success("Deleted remote branch " + branch)
	}

	s.UI.Result("Feature complete!")
	return nil
}

// Discard abandons the current feature branch. It runs preflight checks,
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

	if !strings.HasPrefix(branch, s.Config.FeaturePrefix) {
		return fmt.Errorf("not on a feature branch (current branch: %s)", branch)
	}

	if s.UI.Interactive {
		return s.discardInteractive(branch, reason)
	}
	return s.discardClassic(branch, reason)
}

// discardInteractive runs the feature discard workflow using the Bubble Tea
// progress view. It prompts for confirmation before launching the progress view.
func (s *Service) discardInteractive(branch string, reason string) error {
	ok, err := s.UI.Confirm(fmt.Sprintf("Discard feature branch %q and close its PR?", branch))
	if err != nil {
		return err
	}
	if !ok {
		s.UI.Info("Discard cancelled")
		return nil
	}

	defs := workflow.DiscardStepDefs(s.Config.MainBranch)
	wf := workflow.DiscardWorkflow(s.Git, s.GH, branch, s.Config.MainBranch, reason)
	if err := s.RunProgress("git sf feature discard", branch, defs, wf); err != nil {
		return err
	}

	s.UI.Result("Feature discarded.")
	return nil
}

// discardClassic runs the existing print-style feature discard workflow with a
// confirmation prompt. This is used when the UI is not in interactive mode.
func (s *Service) discardClassic(branch string, reason string) error {
	ok, err := s.UI.Confirm(fmt.Sprintf("Discard feature branch %q and close its PR?", branch))
	if err != nil {
		return err
	}
	if !ok {
		s.UI.Info("Discard cancelled")
		return nil
	}

	if ghErr := gh.CheckGHInstalled(); ghErr == nil {
		if err := s.GH.CheckAuthenticated(); err != nil {
			s.UI.Warning("gh not authenticated — skipping PR close")
		} else if err := s.GH.ClosePR(branch, reason); err != nil {
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
		s.UI.Warning(fmt.Sprintf("Could not delete remote branch %s: %s", branch, err))
	} else {
		s.UI.Success("Deleted remote branch " + branch)
	}

	s.UI.Result("Feature discarded.")
	return nil
}
