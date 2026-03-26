package cmd

import (
	"github.com/milis92/git-simple-flow/internal/feature"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/spf13/cobra"
)

// featureCmd is the parent command for feature branch operations.
var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage feature branches",
}

// featureStartCmd creates a new feature branch from main.
var featureStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Create a new feature branch from main",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     newUI(),
			Config: cfg,
		}
		draftPR, _ := cmd.Flags().GetBool("draft-pr")
		title, _ := cmd.Flags().GetString("title")
		return svc.Start(args[0], feature.StartOpts{DraftPR: draftPR, Title: title})
	},
}

// featurePublishCmd pushes the current feature branch and opens a PR.
var featurePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Push the current feature branch and open a PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     newUI(),
			Config: cfg,
		}
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		return svc.Publish(feature.PublishOpts{Title: title, Body: body})
	},
}

// featureFinishCmd merges the current feature PR and cleans up.
var featureFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Merge the current feature branch PR and clean up",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     newUI(),
			Config: cfg,
		}
		force, _ := cmd.Flags().GetBool("force")
		return svc.Finish(feature.FinishOpts{Force: force})
	},
}

// featureDiscardCmd abandons the current feature branch and closes its PR.
var featureDiscardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Abandon the current feature branch and close its PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &feature.Service{
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
