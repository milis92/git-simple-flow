// cmd/release.go
package cmd

import (
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/release"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release [major|minor|patch]",
	Short: "Tag and push a release from main",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &release.Service{
			Git:    git.New(r, "."),
			UI:     ui.New(),
			Config: cfg,
		}

		scope := cfg.DefaultReleaseBump
		if len(args) > 0 {
			scope = args[0]
		}

		return svc.Release(scope)
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
