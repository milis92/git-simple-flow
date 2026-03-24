// cmd/release.go
package cmd

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release [major|minor|patch]",
	Short: "Tag and push a release from main",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(dryRun, verbose)
		g := git.New(r, ".")

		if err := git.CheckGitInstalled(); err != nil {
			return err
		}
		if err := g.CheckIsRepo(); err != nil {
			return err
		}
		if err := g.CheckOnBranch(cfg.MainBranch); err != nil {
			return err
		}

		// Fetch and verify sync
		if err := g.Fetch(); err != nil {
			return err
		}

		inSync, err := g.IsInSyncWithRemote(cfg.MainBranch)
		if err != nil {
			return err
		}
		if !inSync {
			return fmt.Errorf("local %s is not in sync with origin/%s — pull or push first", cfg.MainBranch, cfg.MainBranch)
		}

		// Determine scope
		scope := cfg.DefaultReleaseBump
		if len(args) > 0 {
			scope = args[0]
		}

		// Get latest tag and compute next version
		tag, err := g.LatestTag(cfg.TagPrefix)
		var next version.Version
		var currentDisplay string

		if err != nil {
			// No tags — first release
			next = version.Version{Major: 0, Minor: 1, Patch: 0}
			currentDisplay = "(no tags)"
			u.Info("No existing tags found. Starting at " + next.FormatWithPrefix(cfg.TagPrefix))
		} else {
			current, parseErr := version.Parse(strings.TrimPrefix(tag, cfg.TagPrefix))
			if parseErr != nil {
				return parseErr
			}
			currentDisplay = tag
			next, err = current.Bump(scope)
			if err != nil {
				return err
			}
		}

		newTag := next.FormatWithPrefix(cfg.TagPrefix)

		u.Blank()
		u.Muted("Current: " + currentDisplay)
		u.Muted(fmt.Sprintf("Next:    %s (%s)", newTag, scope))
		u.Blank()

		confirmed, err := u.Confirm("Confirm release?")
		if err != nil || !confirmed {
			u.Muted("Aborted.")
			return nil
		}

		u.Blank()

		if err := g.Tag(newTag); err != nil {
			return err
		}
		u.Success("Tagged " + newTag)

		if err := g.PushTag(newTag); err != nil {
			return err
		}
		u.Success("Pushed tag to origin")

		u.Result("Released " + newTag)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
