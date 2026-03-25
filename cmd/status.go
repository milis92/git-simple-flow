package cmd

import (
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/status"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/spf13/cobra"
)

// statusCmd displays current branch, PR, and release information.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current branch, PR, and release info",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(false, verbose) // status is read-only, no dry-run needed
		svc := &status.Service{
			Git:    git.New(r, "."),
			GH:     gh.New(r),
			UI:     ui.New(),
			Config: cfg,
		}
		return svc.Show()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
