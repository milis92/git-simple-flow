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

// PreviewRelease creates a prerelease tag on main with a confirmation prompt
// and optional message prompt. It delegates to PreviewReleaseCore for the
// actual tagging. This is the interactive entry point for `git sf release preview`.
func (s *Service) PreviewRelease(scope, message string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	qGit := s.Git.ForQuery()

	if err := qGit.CheckIsRepo(); err != nil {
		return err
	}
	if err := qGit.CheckOnBranch(s.Config.MainBranch); err != nil {
		return err
	}

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

	// Resolve scope
	if scope == "" {
		scope = s.Config.DefaultPrereleaseBump
	}

	suffix := s.Config.PrereleaseSuffix
	prefix := s.Config.TagPrefix

	// Find latest stable tag and compute target
	tag, err := qGit.LatestTagOnBranch(prefix, "HEAD")
	var target version.Version
	var currentDisplay string

	if err != nil {
		// No tags — first release
		target = version.Version{Major: 0, Minor: 1, Patch: 0}
		currentDisplay = "(no tags)"
		s.UI.Info("No existing tags found. Starting at " + target.FormatWithPrefix(prefix))
	} else {
		current, parseErr := version.Parse(strings.TrimPrefix(tag, prefix))
		if parseErr != nil {
			return parseErr
		}
		currentDisplay = tag
		target, err = current.Bump(scope)
		if err != nil {
			return err
		}
	}

	// Check if a previous run left an unpushed local tag that
	// PreviewReleaseCore will recover instead of creating a new one.
	recoverable, recoverErr := findRecoverablePreviewTag(qGit, prefix, suffix, target)
	if recoverErr != nil {
		return recoverErr
	}

	var newTag string
	if recoverable != "" {
		newTag = recoverable
	} else {
		// Compute the real counter so the confirmation prompt shows the
		// actual tag name that PreviewReleaseCore will create.
		counter := 1
		latestPreview, err := qGit.LatestPreviewTag(prefix, suffix, "HEAD", target)
		if err == nil && latestPreview != "" {
			if v, parseErr := version.Parse(strings.TrimPrefix(latestPreview, prefix)); parseErr == nil {
				counter = v.PreBuild + 1
			}
		}

		previewTag := version.Version{
			Major:      target.Major,
			Minor:      target.Minor,
			Patch:      target.Patch,
			Prerelease: suffix,
			PreBuild:   counter,
		}
		newTag = previewTag.FormatWithPrefix(prefix)
	}

	s.UI.Blank()
	s.UI.Muted("Current: " + currentDisplay)
	s.UI.Muted(fmt.Sprintf("Next:    %s (%s preview)", newTag, scope))
	s.UI.Blank()

	confirmed, err := s.UI.Confirm("Confirm preview release?")
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

	return s.PreviewReleaseCore(scope, message)
}

// PreviewReleaseCore creates a prerelease tag on main without any interactive
// prompts. It is the public method called by both PreviewRelease and feature
// finish. It handles retry recovery for previously failed tag pushes.
func (s *Service) PreviewReleaseCore(scope, message string) error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	qGit := s.Git.ForQuery()

	if err := qGit.CheckIsRepo(); err != nil {
		return err
	}
	if err := qGit.CheckOnBranch(s.Config.MainBranch); err != nil {
		return err
	}

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

	// Resolve scope
	if scope == "" {
		scope = s.Config.DefaultPrereleaseBump
	}

	suffix := s.Config.PrereleaseSuffix
	prefix := s.Config.TagPrefix

	// Find latest stable tag and compute target
	tag, err := qGit.LatestTagOnBranch(prefix, "HEAD")
	var target version.Version

	if err != nil {
		// No tags — first release
		target = version.Version{Major: 0, Minor: 1, Patch: 0}
		s.UI.Info("No existing tags found. Starting at " + target.FormatWithPrefix(prefix))
	} else {
		current, parseErr := version.Parse(strings.TrimPrefix(tag, prefix))
		if parseErr != nil {
			return parseErr
		}
		target, err = current.Bump(scope)
		if err != nil {
			return err
		}
	}

	// Retry recovery: check if a local preview tag exists but wasn't pushed
	recovered, err := s.recoverLocalPreviewTag(qGit, prefix, suffix, target)
	if err != nil {
		return err
	}
	if recovered {
		return nil
	}

	// Find latest preview tag for this target to determine next counter
	latestPreview, err := qGit.LatestPreviewTag(prefix, suffix, "HEAD", target)
	if err != nil {
		return err
	}

	counter := 1
	if latestPreview != "" {
		v, parseErr := version.Parse(strings.TrimPrefix(latestPreview, prefix))
		if parseErr != nil {
			return parseErr
		}
		counter = v.PreBuild + 1
	}

	next := version.Version{
		Major:      target.Major,
		Minor:      target.Minor,
		Patch:      target.Patch,
		Prerelease: suffix,
		PreBuild:   counter,
	}
	newTag := next.FormatWithPrefix(prefix)

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

	s.UI.Result("Preview released " + newTag)
	return nil
}

// findRecoverablePreviewTag returns the highest-counter local-only preview tag
// on HEAD for the given target version, or "" if no recovery is needed.
func findRecoverablePreviewTag(qGit *git.Git, prefix, suffix string, target version.Version) (string, error) {
	pattern := fmt.Sprintf("%s%s-%s.*", prefix, target.String(), suffix)
	localTags, err := qGit.ListTags(pattern)
	if err != nil {
		return "", err
	}

	headSHA, err := qGit.RevParse("HEAD")
	if err != nil {
		return "", err
	}

	var best string
	bestCounter := 0

	for _, tag := range localTags {
		onRemote, err := qGit.TagExistsOnRemote(tag)
		if err != nil {
			return "", err
		}
		if onRemote {
			continue
		}

		tagSHA, err := qGit.RevParse(tag + "^{commit}")
		if err != nil {
			return "", err
		}
		if tagSHA != headSHA {
			continue
		}

		v, parseErr := version.Parse(strings.TrimPrefix(tag, prefix))
		if parseErr != nil {
			continue
		}
		if v.PreBuild > bestCounter {
			best = tag
			bestCounter = v.PreBuild
		}
	}

	return best, nil
}

// recoverLocalPreviewTag checks whether a local preview tag exists for the
// given target version that was never pushed to origin (e.g. from a previous
// failed push). If found, it pushes the highest-counter tag and returns true.
// Returns false if no recovery was needed.
func (s *Service) recoverLocalPreviewTag(qGit *git.Git, prefix, suffix string, target version.Version) (bool, error) {
	tag, err := findRecoverablePreviewTag(qGit, prefix, suffix, target)
	if err != nil {
		return false, err
	}
	if tag == "" {
		return false, nil
	}

	s.UI.Info(fmt.Sprintf("Found unpushed local tag %s — pushing now", tag))
	if err := s.Git.PushTag(tag); err != nil {
		return false, err
	}
	s.UI.Success("Pushed recovered tag to origin")
	s.UI.Result("Preview released " + tag)
	return true, nil
}
