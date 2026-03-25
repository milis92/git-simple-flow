// Package status displays current branch information, PR status, CI checks,
// and release info.
package status

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/config"
	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
)

// Service orchestrates git, GitHub CLI, UI, and config to display
// the current repository status.
type Service struct {
	Git    *git.Git
	GH     *gh.GH
	UI     *ui.UI
	Config config.Config
}

// Show displays a status summary for the current branch. It includes:
// branch name and type (feature/hotfix/main), PR info with CI check counts
// (if on a feature or hotfix branch), ahead/behind commit counts relative
// to main, the latest release tag, and possible next version numbers.
func (s *Service) Show() error {
	if err := git.CheckGitInstalled(); err != nil {
		return err
	}

	branch, err := s.Git.CurrentBranch()
	if err != nil {
		return err
	}

	// Determine branch type
	branchType := "other"
	switch {
	case branch == s.Config.MainBranch:
		branchType = s.Config.MainBranch
	case strings.HasPrefix(branch, s.Config.FeaturePrefix):
		branchType = "feature"
	case strings.HasPrefix(branch, s.Config.HotfixPrefix):
		branchType = "hotfix"
	}

	s.UI.Blank()
	fmt.Fprintf(s.UI.Out, "  Branch:     %s\n", branch)
	if branchType != "other" && branchType != s.Config.MainBranch {
		fmt.Fprintf(s.UI.Out, "  Type:       %s\n", branchType)
	}

	// PR info (if on feature/hotfix branch)
	if branchType == "feature" || branchType == "hotfix" {
		if gh.CheckGHInstalled() == nil {
			if pr, err := s.GH.GetCurrentPR(); err == nil {
				draft := ""
				if pr.Draft {
					draft = " (draft)"
				}
				fmt.Fprintf(s.UI.Out, "  PR:         #%d%s — %s\n", pr.Number, draft, pr.URL)

				// Show checks
				if checks, err := s.GH.GetPRChecks(); err == nil && len(checks) > 0 {
					passing, failing, pending := 0, 0, 0
					for _, c := range checks {
						switch {
						case c.Conclusion == "success":
							passing++
						case c.Conclusion == "failure":
							failing++
						default:
							pending++
						}
					}
					total := len(checks)
					status := fmt.Sprintf("%d/%d passing", passing, total)
					if failing > 0 {
						status += fmt.Sprintf(", %d failing", failing)
					}
					if pending > 0 {
						status += fmt.Sprintf(", %d pending", pending)
					}
					fmt.Fprintf(s.UI.Out, "  Checks:     %s\n", status)
				}
			}
		}

		// Behind main
		ahead, behind, err := s.Git.CommitsAheadBehind(branch, s.Config.MainBranch)
		if err == nil {
			if behind > 0 {
				fmt.Fprintf(s.UI.Out, "  Behind:     %d commits behind %s\n", behind, s.Config.MainBranch)
			}
			if ahead > 0 {
				fmt.Fprintf(s.UI.Out, "  Ahead:      %d commits ahead of %s\n", ahead, s.Config.MainBranch)
			}
		}
	}

	// Tag/release info
	s.UI.Blank()
	tag, err := s.Git.LatestTag(s.Config.TagPrefix)
	if err != nil {
		fmt.Fprintf(s.UI.Out, "  Latest tag:    (none)\n")
	} else {
		fmt.Fprintf(s.UI.Out, "  Latest tag:    %s\n", tag)

		if branch == s.Config.MainBranch {
			ahead, _, err := s.Git.CommitsAheadBehind(s.Config.MainBranch, tag)
			if err == nil && ahead > 0 {
				fmt.Fprintf(s.UI.Out, "  Ahead:         %d commits since %s\n", ahead, tag)
			}
		}

		// Show next versions
		current, parseErr := version.Parse(strings.TrimPrefix(tag, s.Config.TagPrefix))
		if parseErr == nil {
			major, _ := current.Bump("major")
			minor, _ := current.Bump("minor")
			patch, _ := current.Bump("patch")
			fmt.Fprintf(s.UI.Out, "  Next release:  %s (minor) / %s (patch) / %s (major)\n",
				minor.FormatWithPrefix(s.Config.TagPrefix),
				patch.FormatWithPrefix(s.Config.TagPrefix),
				major.FormatWithPrefix(s.Config.TagPrefix))
		}
	}

	s.UI.Blank()
	return nil
}
