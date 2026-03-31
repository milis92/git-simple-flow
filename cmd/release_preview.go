package cmd

import (
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/release"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/spf13/cobra"
)

// releasePreviewCmd tags and pushes a preview release from main.
var releasePreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Tag and push a preview release from main",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &release.Service{
			Git:              git.New(r, "."),
			UI:               newUI(),
			Config:           cfg,
			RunMessagePrompt: ui.RunMessagePrompt,
		}

		scope, _ := cmd.Flags().GetString("scope")
		message, _ := cmd.Flags().GetString("message")
		return svc.PreviewRelease(scope, message)
	},
}

func init() {
	releasePreviewCmd.Flags().String("scope", "", "bump scope: major, minor, or patch (default from config)")
	releasePreviewCmd.Flags().StringP("message", "m", "", "tag message (prompted if interactive and not provided)")
	releaseCmd.AddCommand(releasePreviewCmd)
}
