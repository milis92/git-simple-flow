package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/spf13/cobra"
)

// initCmd creates a .sfconfig.yml file in the repo root. In interactive mode
// it runs a wizard to customize settings; otherwise it writes defaults.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create .sfconfig.yml with default settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := newUI()
		path := filepath.Join(repoRoot(), ".sfconfig.yml")
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", path)
			}
		}

		if u.Interactive {
			defaults := ui.InitFormResultFromDefaults(config.Defaults())

			// Detect existing branches for the selector
			r := runner.NewRunner(false, false)
			g := git.New(r, ".")
			branches, _ := g.ListBranches()
			if len(branches) == 0 {
				branches = []string{"main", "develop", "master"}
			}

			result, err := ui.RunInitForm(defaults, branches)
			if err != nil {
				return err
			}

			partial := result.ToPartialConfig()
			if err := config.WritePartialConfig(path, partial); err != nil {
				return err
			}
		} else {
			if force {
				if err := config.ForceWriteDefaults(path); err != nil {
					return err
				}
			} else {
				if err := config.WriteDefaults(path); err != nil {
					return err
				}
			}
		}

		u.Success("Created " + path)
		u.Muted("Edit this file to customize your workflow, or run: git sf config edit")
		return nil
	},
}

// configEditCmd interactively edits .sfconfig.yml in the repo root.
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Interactively edit .sfconfig.yml",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := newUI()
		path := filepath.Join(repoRoot(), ".sfconfig.yml")

		if !u.Interactive {
			return fmt.Errorf("config edit requires an interactive terminal (remove --no-interactive or edit .sfconfig.yml directly)")
		}

		cfg := loadConfig()
		defaults := ui.InitFormResult{
			MainBranch:    cfg.MainBranch,
			FeaturePrefix: cfg.FeaturePrefix,
			HotfixPrefix:  cfg.HotfixPrefix,
			TagPrefix:     cfg.TagPrefix,
			DraftPR:       cfg.DraftPROnStart,
		}

		r := runner.NewRunner(false, false)
		g := git.New(r, ".")
		branches, _ := g.ListBranches()
		if len(branches) == 0 {
			branches = []string{"main", "develop", "master"}
		}

		result, err := ui.RunInitForm(defaults, branches)
		if err != nil {
			return err
		}

		partial := result.ToPartialConfig()
		if err := config.WritePartialConfig(path, partial); err != nil {
			return err
		}

		u.Success("Updated " + path)
		return nil
	},
}

// configCmd displays the effective configuration with source attribution.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := newUI()
		cfg := loadConfig()

		// Load individual layers to show source
		var global *config.PartialConfig
		homeDir, err := os.UserHomeDir()
		if err != nil {
			u.Muted(fmt.Sprintf("Could not determine home directory: %s", err))
		} else {
			globalPath := filepath.Join(homeDir, ".config", "git-sf", "config.yml")
			var globalErr error
			global, globalErr = config.LoadFromFile(globalPath)
			if globalErr != nil {
				u.Muted(fmt.Sprintf("Could not load global config: %s", globalErr))
			}
		}
		repo, repoErr := config.LoadFromFile(".sfconfig.yml")
		if repoErr != nil {
			u.Muted(fmt.Sprintf("Could not load repo config: %s", repoErr))
		}

		u.Blank()
		printConfigField(u, "main_branch", cfg.MainBranch, "main", global, repo,
			func(c *config.PartialConfig) string { return c.MainBranch })
		printConfigField(u, "tag_prefix", cfg.TagPrefix, "v", global, repo,
			func(c *config.PartialConfig) string { return c.TagPrefix })
		printConfigField(u, "feature_prefix", cfg.FeaturePrefix, "feature/", global, repo,
			func(c *config.PartialConfig) string { return c.FeaturePrefix })
		printConfigField(u, "hotfix_prefix", cfg.HotfixPrefix, "hotfix/", global, repo,
			func(c *config.PartialConfig) string { return c.HotfixPrefix })
		printConfigField(u, "merge_strategy", cfg.MergeStrategy, "squash", global, repo,
			func(c *config.PartialConfig) string { return c.MergeStrategy })
		printConfigField(u, "default_release_bump", cfg.DefaultReleaseBump, "minor", global, repo,
			func(c *config.PartialConfig) string { return c.DefaultReleaseBump })
		_, _ = fmt.Fprintf(u.Out, "  %-25s %-15v %s\n", "draft_pr_on_start", cfg.DraftPROnStart, "(default)")
		_, _ = fmt.Fprintf(u.Out, "  %-25s %-15v %s\n", "hotfix_auto_release", cfg.HotfixAutoRelease, "(default)")
		u.Blank()

		return nil
	},
}

// printConfigField displays a single config field with its value and source
// (default, global, or repo), determined by checking each config layer.
func printConfigField(u *ui.UI, name, value, defaultVal string,
	global, repo *config.PartialConfig, getter func(*config.PartialConfig) string) {
	source := "(default)"
	if repo != nil && getter(repo) != "" {
		source = "(repo: .sfconfig.yml)"
	} else if global != nil && getter(global) != "" {
		source = "(global: ~/.config/git-sf/config.yml)"
	}
	_, _ = fmt.Fprintf(u.Out, "  %-25s %-15s %s\n", name, value, source)
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config file")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configEditCmd)
}
