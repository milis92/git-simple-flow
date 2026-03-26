package cmd

import (
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/hotfix"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/spf13/cobra"
)

// hotfixCmd is the parent command for hotfix branch operations.
var hotfixCmd = &cobra.Command{
	Use:   "hotfix",
	Short: "Manage hotfix branches",
}

// hotfixStartCmd creates a new hotfix branch from the latest release tag.
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
			UI:     newUI(),
			Config: cfg,
		}
		draftPR, _ := cmd.Flags().GetBool("draft-pr")
		title, _ := cmd.Flags().GetString("title")
		return svc.Start(args[0], hotfix.StartOpts{DraftPR: draftPR, Title: title})
	},
}

// hotfixPublishCmd pushes the current hotfix branch and creates a PR.
var hotfixPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Push branch and create a PR to main",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     newUI(),
			Config: cfg,
		}
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		return svc.Publish(hotfix.PublishOpts{Title: title, Body: body})
	},
}

// hotfixFinishCmd merges the hotfix PR, cleans up, and optionally tags a release.
var hotfixFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Merge PR, switch to main, delete branch, optionally tag release",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     newUI(),
			Config: cfg,
		}
		force, _ := cmd.Flags().GetBool("force")
		release, _ := cmd.Flags().GetBool("release")
		return svc.Finish(hotfix.FinishOpts{Force: force, Release: release})
	},
}

// hotfixDiscardCmd abandons the current hotfix branch and closes its PR.
var hotfixDiscardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Close PR, delete branch, switch to main",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &hotfix.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     newUI(),
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
