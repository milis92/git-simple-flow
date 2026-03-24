// cmd/hotfix.go
package cmd

import (
	"fmt"
	"strings"

	runner "github.com/nickssmallpdf/git-sf/internal/exec"
	"github.com/nickssmallpdf/git-sf/internal/gh"
	gitpkg "github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
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
		name := args[0]
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := gitpkg.New(r, ".")
		h := gh.New(r)

		if err := gitpkg.CheckGitInstalled(); err != nil {
			return err
		}
		if err := g.CheckIsRepo(); err != nil {
			return err
		}
		if err := g.CheckCleanTree(); err != nil {
			return err
		}

		tag, err := g.LatestTag(cfg.TagPrefix)
		if err != nil {
			return fmt.Errorf("no tags found. Create an initial release first with 'git sf release'")
		}

		if err := g.Checkout(tag); err != nil {
			return err
		}
		u.Success("Checked out " + tag)

		branchName := cfg.HotfixPrefix + name
		if err := g.CreateBranch(branchName); err != nil {
			return err
		}
		u.Success("Created branch " + branchName)

		draftPR, _ := cmd.Flags().GetBool("draft-pr")
		if draftPR || cfg.DraftPROnStart {
			if err := gh.CheckGHInstalled(); err != nil {
				return err
			}
			if err := h.CheckAuthenticated(); err != nil {
				return err
			}
			if err := g.Push(branchName); err != nil {
				return err
			}
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				title = gh.HumanizeBranchName(branchName, cfg.HotfixPrefix)
			}
			pr, err := h.CreatePR(cfg.MainBranch, title, "", true)
			if err != nil {
				return err
			}
			u.Success("Created draft PR: " + pr.URL)
		}

		u.Result("Ready to work. When done: git sf hotfix publish")
		return nil
	},
}

var hotfixPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Push branch and create a PR to main",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := gitpkg.New(r, ".")
		h := gh.New(r)

		if err := gitpkg.CheckGitInstalled(); err != nil {
			return err
		}
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := h.CheckAuthenticated(); err != nil {
			return err
		}

		clean, _ := g.IsClean()
		if !clean {
			u.Warning("You have uncommitted changes that won't be included in the PR.")
		}

		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}

		if err := g.Push(branch); err != nil {
			return err
		}
		u.Success("Pushed " + branch)

		title, _ := cmd.Flags().GetString("title")
		if title == "" {
			title = gh.HumanizeBranchName(branch, cfg.HotfixPrefix)
		}
		body, _ := cmd.Flags().GetString("body")

		pr, err := h.CreatePR(cfg.MainBranch, title, body, false)
		if err != nil {
			return err
		}
		u.Success("Created PR: " + pr.URL)

		u.Result("PR is up. When ready: git sf hotfix finish")
		return nil
	},
}

var hotfixFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Merge PR, switch to main, delete branch, optionally tag release",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := gitpkg.New(r, ".")
		h := gh.New(r)

		if err := gitpkg.CheckGitInstalled(); err != nil {
			return err
		}
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := h.CheckAuthenticated(); err != nil {
			return err
		}
		if err := g.CheckCleanTree(); err != nil {
			return err
		}

		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}

		pr, err := h.GetCurrentPR()
		if err != nil {
			return fmt.Errorf("no PR found for this branch. Run 'git sf hotfix publish' first")
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			checks, err := h.GetPRChecks()
			if err == nil {
				failing := false
				for _, c := range checks {
					if c.Conclusion == "failure" {
						u.Error(c.Name + " — failed")
						failing = true
					} else if c.Status != "completed" {
						u.Warning(c.Name + " — " + c.Status)
					} else {
						u.Success(c.Name + " — passed")
					}
				}
				if failing {
					return fmt.Errorf("PR has failing checks. Fix them or use --force to merge anyway")
				}
			}
		}

		confirmed, err := u.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
		if err != nil || !confirmed {
			u.Muted("Aborted.")
			return nil
		}

		u.Info(fmt.Sprintf("Merging PR #%d — %q", pr.Number, pr.Title))

		if err := h.MergePR(cfg.MergeStrategy); err != nil {
			return err
		}
		u.Success(fmt.Sprintf("PR merged (%s)", cfg.MergeStrategy))

		if err := g.Checkout(cfg.MainBranch); err != nil {
			return err
		}
		u.Success("Switched to " + cfg.MainBranch)

		if err := g.Pull(); err != nil {
			return err
		}
		u.Success("Pulled latest changes")

		if err := g.DeleteLocalBranch(branch); err != nil {
			return err
		}
		u.Success("Deleted branch " + branch + " (local)")

		if err := g.DeleteRemoteBranch(branch); err != nil {
			u.Warning("Remote branch already deleted")
		} else {
			u.Success("Deleted branch " + branch + " (remote)")
		}

		// Auto-release if --release flag or config
		release, _ := cmd.Flags().GetBool("release")
		if release || cfg.HotfixAutoRelease {
			tag, err := g.LatestTag(cfg.TagPrefix)
			if err != nil {
				return err
			}
			current, err := version.Parse(strings.TrimPrefix(tag, cfg.TagPrefix))
			if err != nil {
				return err
			}
			next, _ := current.Bump("patch")
			newTag := next.FormatWithPrefix(cfg.TagPrefix)

			if err := g.Tag(newTag); err != nil {
				return err
			}
			u.Success("Tagged " + newTag)

			if err := g.PushTag(newTag); err != nil {
				return err
			}
			u.Success("Pushed tag to origin")

			u.Result("Hotfix released " + newTag)
			return nil
		}

		u.Result("Done.")
		return nil
	},
}

var hotfixDiscardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Close PR, delete branch, switch to main",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := gitpkg.New(r, ".")
		h := gh.New(r)

		if err := gitpkg.CheckGitInstalled(); err != nil {
			return err
		}
		if err := g.CheckCleanTree(); err != nil {
			return err
		}

		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}

		if !strings.HasPrefix(branch, cfg.HotfixPrefix) {
			return fmt.Errorf("not on a hotfix branch (current: %s)", branch)
		}

		confirmed, err := u.Confirm("Discard branch " + branch + "?")
		if err != nil || !confirmed {
			u.Muted("Aborted.")
			return nil
		}

		if err := gh.CheckGHInstalled(); err == nil {
			if err := h.CheckAuthenticated(); err == nil {
				reason, _ := cmd.Flags().GetString("reason")
				if err := h.ClosePR(reason); err != nil {
					u.Warning("No PR to close or already closed")
				} else {
					u.Success("Closed PR")
				}
			}
		}

		if err := g.Checkout(cfg.MainBranch); err != nil {
			return err
		}
		u.Success("Switched to " + cfg.MainBranch)

		if err := g.DeleteLocalBranch(branch); err != nil {
			return err
		}
		u.Success("Deleted branch " + branch + " (local)")

		if err := g.DeleteRemoteBranch(branch); err != nil {
			u.Warning("Remote branch already deleted")
		} else {
			u.Success("Deleted branch " + branch + " (remote)")
		}

		u.Result("Discarded.")
		return nil
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
