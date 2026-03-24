// Package hotfix implements the hotfix branch lifecycle. Unlike features,
// hotfixes branch from the latest release tag (not main) and can optionally
// trigger a patch release on finish.
package hotfix

import (
	"fmt"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/milis92/git-simple-flow/internal/version"
)

// Service orchestrates git, GitHub CLI, UI, and config to execute
// the hotfix branch workflow.
type Service struct {
	Git    *git.Git
	GH     *gh.GH
	UI     *ui.UI
	Config config.Config
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

// Publish pushes the current hotfix branch and creates a PR to main.
// It warns if the working tree is dirty but does not block.
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

// Finish merges the current hotfix PR and cleans up. After merging and branch
// deletion, if Release is set or hotfix_auto_release is configured, it
// automatically creates and pushes a patch version tag.
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
				switch {
				case c.Conclusion == "failure":
					s.UI.Error(c.Name + " — failed")
					failing = true
				case c.Status != "completed":
					s.UI.Warning(c.Name + " — " + c.Status)
				default:
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

// Discard abandons the current hotfix branch. It confirms with the user,
// closes the PR (posting reason as a comment if provided), switches to main,
// and deletes the local and remote branches.
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
