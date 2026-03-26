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

const (
	defaultConfigSource = "(default)"
	globalConfigSource  = "(global: ~/.config/git-sf/config.yml)"
	repoConfigSource    = "(repo: .sfconfig.yml)"
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

		global, globalErr := loadGlobalConfig()
		if globalErr != nil {
			u.Muted(fmt.Sprintf("Could not load global config: %s", globalErr))
		}
		repo, repoErr := config.LoadFromFile(path)
		if repoErr != nil {
			return fmt.Errorf("could not load repo config: %w", repoErr)
		}

		inherited := config.Merge(config.Defaults(), global)
		cfg := config.Merge(inherited, repo)
		defaults := ui.InitFormResult{
			MainBranch:         cfg.MainBranch,
			FeaturePrefix:      cfg.FeaturePrefix,
			HotfixPrefix:       cfg.HotfixPrefix,
			TagPrefix:          cfg.TagPrefix,
			MergeStrategy:      cfg.MergeStrategy,
			DefaultReleaseBump: cfg.DefaultReleaseBump,
			DraftPR:            cfg.DraftPROnStart,
			HotfixAutoRelease:  cfg.HotfixAutoRelease,
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

		partial := buildRepoConfigForEdit(inherited, repo, result)
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
		repoPath := filepath.Join(repoRoot(), ".sfconfig.yml")

		// Load individual layers to show source
		global, globalErr := loadGlobalConfig()
		if globalErr != nil {
			u.Muted(fmt.Sprintf("Could not load global config: %s", globalErr))
		}
		repo, repoErr := config.LoadFromFile(repoPath)
		if repoErr != nil {
			u.Muted(fmt.Sprintf("Could not load repo config: %s", repoErr))
		}
		cfg := config.Merge(config.Defaults(), global, repo)

		u.Blank()
		printConfigField(u, "main_branch", cfg.MainBranch, global, repo,
			func(c *config.PartialConfig) string { return c.MainBranch })
		printConfigField(u, "tag_prefix", cfg.TagPrefix, global, repo,
			func(c *config.PartialConfig) string { return c.TagPrefix })
		printConfigField(u, "feature_prefix", cfg.FeaturePrefix, global, repo,
			func(c *config.PartialConfig) string { return c.FeaturePrefix })
		printConfigField(u, "hotfix_prefix", cfg.HotfixPrefix, global, repo,
			func(c *config.PartialConfig) string { return c.HotfixPrefix })
		printConfigField(u, "merge_strategy", cfg.MergeStrategy, global, repo,
			func(c *config.PartialConfig) string { return c.MergeStrategy })
		printConfigField(u, "default_release_bump", cfg.DefaultReleaseBump, global, repo,
			func(c *config.PartialConfig) string { return c.DefaultReleaseBump })
		printConfigBoolField(u, "draft_pr_on_start", cfg.DraftPROnStart, global, repo,
			func(c *config.PartialConfig) *bool { return c.DraftPROnStart })
		printConfigBoolField(u, "hotfix_auto_release", cfg.HotfixAutoRelease, global, repo,
			func(c *config.PartialConfig) *bool { return c.HotfixAutoRelease })
		u.Blank()

		return nil
	},
}

func loadGlobalConfig() (*config.PartialConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	globalPath := filepath.Join(homeDir, ".config", "git-sf", "config.yml")
	return config.LoadFromFile(globalPath)
}

// buildRepoConfigForEdit preserves untouched repo-only fields and stores edited
// fields as minimal overrides against the inherited defaults+global config.
func buildRepoConfigForEdit(inherited config.Config, existing *config.PartialConfig, result ui.InitFormResult) config.PartialConfig {
	var updated config.PartialConfig
	if existing != nil {
		updated = *existing
	}

	updated.MainBranch = repoStringOverride(result.MainBranch, inherited.MainBranch)
	updated.FeaturePrefix = repoStringOverride(result.FeaturePrefix, inherited.FeaturePrefix)
	updated.HotfixPrefix = repoStringOverride(result.HotfixPrefix, inherited.HotfixPrefix)
	updated.TagPrefix = repoStringOverride(result.TagPrefix, inherited.TagPrefix)
	updated.MergeStrategy = repoStringOverride(result.MergeStrategy, inherited.MergeStrategy)
	updated.DefaultReleaseBump = repoStringOverride(result.DefaultReleaseBump, inherited.DefaultReleaseBump)
	updated.DraftPROnStart = repoBoolOverride(result.DraftPR, inherited.DraftPROnStart)
	updated.HotfixAutoRelease = repoBoolOverride(result.HotfixAutoRelease, inherited.HotfixAutoRelease)

	return updated
}

func repoStringOverride(value, inherited string) string {
	if value == inherited {
		return ""
	}
	return value
}

func repoBoolOverride(value, inherited bool) *bool {
	if value == inherited {
		return nil
	}
	updated := value
	return &updated
}

// printConfigField displays a single string config field with source
// attribution based on repo, global, and default layers.
func printConfigField(u *ui.UI, name, value string,
	global, repo *config.PartialConfig, getter func(*config.PartialConfig) string) {
	source := configFieldSource(global, repo, getter)
	_, _ = fmt.Fprintf(u.Out, "  %-25s %-15s %s\n", name, value, source)
}

// printConfigBoolField displays a single boolean config field with source
// attribution based on repo, global, and default layers.
func printConfigBoolField(u *ui.UI, name string, value bool,
	global, repo *config.PartialConfig, getter func(*config.PartialConfig) *bool) {
	source := configBoolSource(global, repo, getter)
	_, _ = fmt.Fprintf(u.Out, "  %-25s %-15v %s\n", name, value, source)
}

func configFieldSource(global, repo *config.PartialConfig, getter func(*config.PartialConfig) string) string {
	if repo != nil && getter(repo) != "" {
		return repoConfigSource
	}
	if global != nil && getter(global) != "" {
		return globalConfigSource
	}
	return defaultConfigSource
}

func configBoolSource(global, repo *config.PartialConfig, getter func(*config.PartialConfig) *bool) string {
	if repo != nil && getter(repo) != nil {
		return repoConfigSource
	}
	if global != nil && getter(global) != nil {
		return globalConfigSource
	}
	return defaultConfigSource
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config file")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configEditCmd)
}
