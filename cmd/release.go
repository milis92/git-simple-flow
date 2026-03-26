package cmd

import (
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/release"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/spf13/cobra"
)

// releaseCmd tags and pushes a new semver release from main.
var releaseCmd = &cobra.Command{
	Use:   "release [major|minor|patch]",
	Short: "Tag and push a release from main",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		svc := &release.Service{
			Git:    git.New(r, "."),
			UI:     newUI(),
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
