// cmd/status.go
package cmd

import (
	"fmt"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/gh"
	"github.com/nickssmallpdf/git-sf/internal/git"
	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/ui"
	"github.com/nickssmallpdf/git-sf/internal/version"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current branch, PR, and release info",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()
		r := runner.NewRunner(false, verbose) // status is read-only, no dry-run needed
		g := git.New(r, ".")
		h := gh.New(r)

		if err := git.CheckGitInstalled(); err != nil {
			return err
		}

		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}

		// Determine branch type
		branchType := "other"
		if branch == cfg.MainBranch {
			branchType = cfg.MainBranch
		} else if strings.HasPrefix(branch, cfg.FeaturePrefix) {
			branchType = "feature"
		} else if strings.HasPrefix(branch, cfg.HotfixPrefix) {
			branchType = "hotfix"
		}

		u.Blank()
		fmt.Fprintf(u.Out, "  Branch:     %s\n", branch)
		if branchType != "other" && branchType != cfg.MainBranch {
			fmt.Fprintf(u.Out, "  Type:       %s\n", branchType)
		}

		// PR info (if on feature/hotfix branch)
		if branchType == "feature" || branchType == "hotfix" {
			if gh.CheckGHInstalled() == nil {
				if pr, err := h.GetCurrentPR(); err == nil {
					draft := ""
					if pr.Draft {
						draft = " (draft)"
					}
					fmt.Fprintf(u.Out, "  PR:         #%d%s — %s\n", pr.Number, draft, pr.URL)

					// Show checks
					if checks, err := h.GetPRChecks(); err == nil && len(checks) > 0 {
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
						fmt.Fprintf(u.Out, "  Checks:     %s\n", status)
					}
				}
			}

			// Behind main
			ahead, behind, err := g.CommitsAheadBehind(branch, cfg.MainBranch)
			if err == nil {
				if behind > 0 {
					fmt.Fprintf(u.Out, "  Behind:     %d commits behind %s\n", behind, cfg.MainBranch)
				}
				if ahead > 0 {
					fmt.Fprintf(u.Out, "  Ahead:      %d commits ahead of %s\n", ahead, cfg.MainBranch)
				}
			}
		}

		// Tag/release info
		u.Blank()
		tag, err := g.LatestTag(cfg.TagPrefix)
		if err != nil {
			fmt.Fprintf(u.Out, "  Latest tag:    (none)\n")
		} else {
			fmt.Fprintf(u.Out, "  Latest tag:    %s\n", tag)

			if branch == cfg.MainBranch {
				ahead, _, err := g.CommitsAheadBehind(cfg.MainBranch, tag)
				if err == nil && ahead > 0 {
					fmt.Fprintf(u.Out, "  Ahead:         %d commits since %s\n", ahead, tag)
				}
			}

			// Show next versions
			current, parseErr := version.Parse(strings.TrimPrefix(tag, cfg.TagPrefix))
			if parseErr == nil {
				major, _ := current.Bump("major")
				minor, _ := current.Bump("minor")
				patch, _ := current.Bump("patch")
				fmt.Fprintf(u.Out, "  Next release:  %s (minor) / %s (patch) / %s (major)\n",
					minor.FormatWithPrefix(cfg.TagPrefix),
					patch.FormatWithPrefix(cfg.TagPrefix),
					major.FormatWithPrefix(cfg.TagPrefix))
			}
		}

		u.Blank()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
