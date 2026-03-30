// Package release creates semver releases by tagging main and pushing to origin.
package release

import (
	"fmt"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/milis92/git-simple-flow/internal/version"
)

// Service orchestrates git, UI, and config to execute the release workflow.
type Service struct {
	Git              *git.Git
	UI               *ui.UI
	Config           config.Config
	RunMessagePrompt func(string) (string, error)
}

// Release creates a new semver tag on main and pushes it to origin. It verifies
// the repo is on main and in sync with the remote, finds the latest tag, computes
// the next version based on scope ("major", "minor", or "patch"), shows a
// confirmation prompt with current and next versions, then creates and pushes the tag.
// If message is provided (via flag or interactive prompt), an annotated tag is
// created; otherwise a lightweight tag is used.
// If no tags exist yet, it starts at v0.1.0.
func (s *Service) Release(scope, message string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	// Use query-mode runner for read-only preflight checks so they execute
	// even during --dry-run.
	qGit := s.Git.ForQuery()

	if err := qGit.CheckIsRepo(); err != nil {
		return err
	}
	if err := qGit.CheckOnBranch(s.Config.MainBranch); err != nil {
		return err
	}

	// Fetch and verify sync
	if err := s.Git.Fetch(); err != nil {
		return err
	}

	inSync, err := qGit.IsInSyncWithRemote(s.Config.MainBranch)
	if err != nil {
		return err
	}
	if !inSync {
		return fmt.Errorf("local %s is not in sync with origin/%s — pull or push first", s.Config.MainBranch, s.Config.MainBranch)
	}

	// Get latest tag reachable from HEAD (on main) and compute next version.
	// Using LatestTagOnBranch prevents off-main hotfix tags from being picked up.
	tag, err := qGit.LatestTagOnBranch(s.Config.TagPrefix, "HEAD")
	var next version.Version
	var currentDisplay string

	if err != nil {
		// No tags — first release
		next = version.Version{Major: 0, Minor: 1, Patch: 0}
		currentDisplay = "(no tags)"
		s.UI.Info("No existing tags found. Starting at " + next.FormatWithPrefix(s.Config.TagPrefix))
	} else {
		current, parseErr := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
		if parseErr != nil {
			return parseErr
		}
		currentDisplay = tag
		next, err = current.Bump(scope)
		if err != nil {
			return err
		}
	}

	newTag := next.FormatWithPrefix(s.Config.TagPrefix)

	s.UI.Blank()
	s.UI.Muted("Current: " + currentDisplay)
	s.UI.Muted(fmt.Sprintf("Next:    %s (%s)", newTag, scope))
	s.UI.Blank()

	confirmed, err := s.UI.Confirm("Confirm release?")
	if err != nil {
		return err
	}
	if !confirmed {
		s.UI.Muted("Aborted.")
		return nil
	}

	s.UI.Blank()

	if message == "" && s.UI.ShouldPrompt() {
		var promptErr error
		message, promptErr = s.RunMessagePrompt(newTag)
		if promptErr != nil {
			return promptErr
		}
	}

	if message != "" {
		if err := s.Git.TagAnnotated(newTag, message); err != nil {
			return err
		}
	} else {
		if err := s.Git.Tag(newTag); err != nil {
			return err
		}
	}
	s.UI.Success("Tagged " + newTag)

	if err := s.Git.PushTag(newTag); err != nil {
		return err
	}
	s.UI.Success("Pushed tag to origin")

	s.UI.Result("Released " + newTag)
	return nil
}
