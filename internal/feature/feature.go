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
