package cmd

import (
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/status"
	"github.com/milis92/git-simple-flow/internal/ui"
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
