// Package feature implements the feature branch lifecycle: start, publish,
// finish, and discard. A feature branches from main, gets a PR, and merges
// back to main when done.
package feature

import (
	"fmt"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/ui"
)

// Service orchestrates git, GitHub CLI, UI, and config to execute
// the feature branch workflow.
type Service struct {
	Git    *git.Git
	GH     *gh.GH
	UI     *ui.UI
	Config config.Config
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
		if err := s.Git.Push(branchName); err != nil {
			return err
		}
		title := opts.Title
		if title == "" && s.UI.Interactive {
			defaultTitle := gh.HumanizeBranchName(branchName, s.Config.FeaturePrefix)
			result, promptErr := ui.RunTitlePrompt(defaultTitle, false)
			if promptErr != nil {
				return promptErr
			}
			title = result.Title
		}
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

	if err := s.Git.Push(branch); err != nil {
		return err
	}
	s.UI.Success("Pushed branch " + branch)

	title := opts.Title
	body := opts.Body
	if title == "" && s.UI.Interactive {
		defaultTitle := gh.HumanizeBranchName(branch, s.Config.FeaturePrefix)
		result, promptErr := ui.RunTitlePrompt(defaultTitle, true)
		if promptErr != nil {
			return promptErr
		}
		title = result.Title
		if body == "" {
			body = result.Body
		}
	}
	if title == "" {
		title = gh.HumanizeBranchName(branch, s.Config.FeaturePrefix)
	}

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

	if s.UI.Interactive {
		return s.finishInteractive(branch, opts)
	}
	return s.finishClassic(branch, opts)
}

// finishInteractive runs the feature finish workflow using the Bubble Tea
// progress view. It skips the confirmation prompt — the user explicitly ran
// the finish command.
func (s *Service) finishInteractive(branch string, opts FinishOpts) error {
	defs := []ui.StepDef{
		{Label: "Find PR"},
		{Label: "Check CI"},
		{Label: "Merge PR"},
		{Label: "Switch to " + s.Config.MainBranch},
		{Label: "Pull latest"},
		{Label: "Delete local branch"},
		{Label: "Delete remote branch"},
	}

	if opts.Force {
		defs[1].Label = "Check CI (skipped)"
	}

	err := ui.RunProgress("git sf feature finish", branch, defs, func(cb ui.StepCallbacks) error {
		// Step 0: Find PR
		cb.Start()
		if _, err := s.GH.GetCurrentPR(); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 1: Check CI
		cb.Start()
		if !opts.Force {
			checks, err := s.GH.GetPRChecks()
			if err != nil {
				cb.Fail(err.Error())
				return err
			}
			var failing []string
			for _, c := range checks {
				if c.Conclusion == "failure" || c.Conclusion == "cancelled" {
					failing = append(failing, c.Name)
				}
			}
			if len(failing) > 0 {
				errMsg := fmt.Sprintf("PR checks failed: %s (use --force to override)", strings.Join(failing, ", "))
				cb.Fail(errMsg)
				return fmt.Errorf("%s", errMsg)
			}
		}
		cb.Done()

		// Step 2: Merge PR
		cb.Start()
		if err := s.GH.MergePR(s.Config.MergeStrategy); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 3: Switch to main
		cb.Start()
		if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 4: Pull latest
		cb.Start()
		if err := s.Git.Pull(); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 5: Delete local branch
		cb.Start()
		if err := s.Git.DeleteLocalBranch(branch); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 6: Delete remote branch (soft fail)
		cb.Start()
		if err := s.Git.DeleteRemoteBranch(branch); err != nil {
			cb.Fail("already deleted or could not be removed")
		} else {
			cb.Done()
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.UI.Result("Feature complete!")
	return nil
}

// finishClassic runs the existing print-style feature finish workflow with a
// confirmation prompt. This is used when the UI is not in interactive mode.
func (s *Service) finishClassic(branch string, opts FinishOpts) error {
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

// Discard abandons the current feature branch. It runs preflight checks,
// detects the current branch, then routes to discardInteractive (Bubble Tea
// progress) or discardClassic (print-style) based on whether the UI is in
// interactive mode.
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

	if s.UI.Interactive {
		return s.discardInteractive(branch, reason)
	}
	return s.discardClassic(branch, reason)
}

// discardInteractive runs the feature discard workflow using the Bubble Tea
// progress view. It skips the confirmation prompt — the user explicitly ran
// the discard command.
func (s *Service) discardInteractive(branch string, reason string) error {
	defs := []ui.StepDef{
		{Label: "Close PR"},
		{Label: "Switch to " + s.Config.MainBranch},
		{Label: "Delete local branch"},
		{Label: "Delete remote branch"},
	}

	err := ui.RunProgress("git sf feature discard", branch, defs, func(cb ui.StepCallbacks) error {
		// Step 0: Close PR (soft fail — PR may not exist)
		cb.Start()
		if ghErr := gh.CheckGHInstalled(); ghErr != nil {
			cb.Fail("gh CLI not available — skipped")
		} else if authErr := s.GH.CheckAuthenticated(); authErr != nil {
			cb.Fail("not authenticated — skipped")
		} else if err := s.GH.ClosePR(reason); err != nil {
			cb.Fail("no PR to close or already closed")
		} else {
			cb.Done()
		}

		// Step 1: Switch to main
		cb.Start()
		if err := s.Git.Checkout(s.Config.MainBranch); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 2: Delete local branch
		cb.Start()
		if err := s.Git.DeleteLocalBranch(branch); err != nil {
			cb.Fail(err.Error())
			return err
		}
		cb.Done()

		// Step 3: Delete remote branch (soft fail)
		cb.Start()
		if err := s.Git.DeleteRemoteBranch(branch); err != nil {
			cb.Fail("already deleted or could not be removed")
		} else {
			cb.Done()
		}

		return nil
	})
	if err != nil {
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
