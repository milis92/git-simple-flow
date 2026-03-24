// cmd/feature.go
package cmd

import (
	"fmt"
	"strings"

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
		name := args[0]
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := git.New(r, ".")
		h := gh.New(r)

		if err := git.CheckGitInstalled(); err != nil {
			return err
		}
		if err := g.CheckIsRepo(); err != nil {
			return err
		}
		if err := g.CheckCleanTree(); err != nil {
			return err
		}

		branchName := cfg.FeaturePrefix + name

		if err := g.Checkout(cfg.MainBranch); err != nil {
			return err
		}
		u.Success("Switched to " + cfg.MainBranch)

		if err := g.Pull(); err != nil {
			return err
		}
		u.Success("Pulled latest changes")

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
				title = gh.HumanizeBranchName(branchName, cfg.FeaturePrefix)
			}
			pr, err := h.CreatePR(cfg.MainBranch, title, "", true)
			if err != nil {
				return err
			}
			u.Success("Created draft PR: " + pr.URL)
		}

		u.Result("Ready to work. When done: git sf feature publish")
		return nil
	},
}

var featurePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Push the current feature branch and open a PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := git.New(r, ".")
		h := gh.New(r)

		if err := git.CheckGitInstalled(); err != nil {
			return err
		}
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := g.CheckIsRepo(); err != nil {
			return err
		}
		if err := h.CheckAuthenticated(); err != nil {
			return err
		}

		clean, err := g.IsClean()
		if err != nil {
			return err
		}
		if !clean {
			u.Warning("You have uncommitted changes — consider committing or stashing them first")
		}

		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}

		if err := g.Push(branch); err != nil {
			return err
		}
		u.Success("Pushed branch " + branch)

		title, _ := cmd.Flags().GetString("title")
		if title == "" {
			title = gh.HumanizeBranchName(branch, cfg.FeaturePrefix)
		}
		body, _ := cmd.Flags().GetString("body")

		pr, err := h.CreatePR(cfg.MainBranch, title, body, false)
		if err != nil {
			return err
		}
		u.Success("Created PR: " + pr.URL)

		u.Result("PR is open. When ready to merge: git sf feature finish")
		return nil
	},
}

var featureFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Merge the current feature branch PR and clean up",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := git.New(r, ".")
		h := gh.New(r)

		if err := git.CheckGitInstalled(); err != nil {
			return err
		}
		if err := gh.CheckGHInstalled(); err != nil {
			return err
		}
		if err := g.CheckIsRepo(); err != nil {
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
			return err
		}
		u.Info(fmt.Sprintf("Found PR #%d — %q", pr.Number, pr.Title))

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			checks, err := h.GetPRChecks()
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
			u.Success("PR checks passed")
		}

		ok, err := u.Confirm(fmt.Sprintf("Merge PR #%d — %q?", pr.Number, pr.Title))
		if err != nil {
			return err
		}
		if !ok {
			u.Info("Merge cancelled")
			return nil
		}

		if err := h.MergePR(cfg.MergeStrategy); err != nil {
			return err
		}
		u.Success("Merged PR #" + fmt.Sprint(pr.Number))

		if err := g.Checkout(cfg.MainBranch); err != nil {
			return err
		}
		if err := g.Pull(); err != nil {
			return err
		}
		u.Success("Switched to " + cfg.MainBranch + " and pulled latest changes")

		if err := g.DeleteLocalBranch(branch); err != nil {
			return err
		}
		u.Success("Deleted local branch " + branch)

		if err := g.DeleteRemoteBranch(branch); err != nil {
			u.Warning("Remote branch already deleted or could not be removed: " + branch)
		} else {
			u.Success("Deleted remote branch " + branch)
		}

		u.Result("Feature complete!")
		return nil
	},
}

var featureDiscardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Abandon the current feature branch and close its PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := git.New(r, ".")
		h := gh.New(r)

		if err := git.CheckGitInstalled(); err != nil {
			return err
		}
		if err := g.CheckIsRepo(); err != nil {
			return err
		}
		if err := g.CheckCleanTree(); err != nil {
			return err
		}

		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}

		if !strings.HasPrefix(branch, cfg.FeaturePrefix) {
			return fmt.Errorf("not on a feature branch (current branch: %s)", branch)
		}

		ok, err := u.Confirm(fmt.Sprintf("Discard feature branch %q and close its PR?", branch))
		if err != nil {
			return err
		}
		if !ok {
			u.Info("Discard cancelled")
			return nil
		}

		reason, _ := cmd.Flags().GetString("reason")
		if ghErr := gh.CheckGHInstalled(); ghErr == nil {
			if err := h.ClosePR(reason); err != nil {
				u.Warning("Could not close PR (may not exist): " + err.Error())
			} else {
				u.Success("Closed PR")
			}
		} else {
			u.Warning("gh CLI not available — skipping PR close")
		}

		if err := g.Checkout(cfg.MainBranch); err != nil {
			return err
		}
		u.Success("Switched to " + cfg.MainBranch)

		if err := g.DeleteLocalBranch(branch); err != nil {
			return err
		}
		u.Success("Deleted local branch " + branch)

		if err := g.DeleteRemoteBranch(branch); err != nil {
			u.Warning("Remote branch already deleted or could not be removed: " + branch)
		} else {
			u.Success("Deleted remote branch " + branch)
		}

		u.Result("Feature discarded.")
		return nil
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
