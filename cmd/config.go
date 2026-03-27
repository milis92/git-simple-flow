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

		if dryRun {
			u.Success("(dry-run) Would create " + path)
			return nil
		}

		if shouldUseInitWizard(u) {
			defaults := initWizardDefaults(path, force)

			// Detect existing branches for the selector
			r := runner.NewRunner(dryRun, verbose)
			g := git.New(r, ".")
			branches := detectBranches(g, u)

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

func shouldUseInitWizard(u *ui.UI) bool {
	return u.ShouldPrompt()
}

// initWizardDefaults returns the form defaults for the init wizard. When force
// is true and an existing config file can be loaded, its values are merged on
// top of the built-in defaults so the wizard seeds from the current config.
func initWizardDefaults(path string, force bool) ui.InitFormResult {
	base := config.Defaults()
	if force {
		if existing, _, err := loadPartialConfig(path); err == nil {
			base = config.Merge(base, existing)
		}
	}
	return ui.InitFormResultFromDefaults(base)
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

		globalPath, pathErr := globalConfigPath()
		if pathErr != nil {
			u.Muted(fmt.Sprintf("Could not determine home directory: %s", pathErr))
		}

		var global *config.PartialConfig
		if globalPath != "" {
			var warnings []error
			var loadErr error
			global, warnings, loadErr = loadPartialConfig(globalPath)
			if loadErr != nil {
				u.Muted(fmt.Sprintf("Could not load global config %s: %s", globalPath, loadErr))
			}
			for _, warning := range warnings {
				u.Muted(fmt.Sprintf("Ignoring invalid global config %s: %s", globalPath, warning))
			}
		}

		repo, repoWarnings, repoErr := loadPartialConfig(path)
		if repoErr != nil {
			return fmt.Errorf("could not load repo config: %w", repoErr)
		}
		for _, warning := range repoWarnings {
			u.Muted(fmt.Sprintf("Ignoring invalid repo config %s: %s", path, warning))
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

		r := runner.NewRunner(dryRun, verbose)
		g := git.New(r, ".")
		branches := detectBranches(g, u)

		result, err := ui.RunInitForm(defaults, branches)
		if err != nil {
			return err
		}

		partial := buildRepoConfigForEdit(inherited, repo, result)
		if err := config.UpdatePartialConfigFile(path, partial); err != nil {
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
		globalPath, pathErr := globalConfigPath()
		if pathErr != nil {
			u.Muted(fmt.Sprintf("Could not determine home directory: %s", pathErr))
		}

		var global *config.PartialConfig
		if globalPath != "" {
			var warnings []error
			var loadErr error
			global, warnings, loadErr = loadPartialConfig(globalPath)
			if loadErr != nil {
				u.Muted(fmt.Sprintf("Could not load global config %s: %s", globalPath, loadErr))
			}
			for _, warning := range warnings {
				u.Muted(fmt.Sprintf("Ignoring invalid global config %s: %s", globalPath, warning))
			}
		}

		repo, repoWarnings, repoErr := loadPartialConfig(repoPath)
		if repoErr != nil {
			u.Muted(fmt.Sprintf("Could not load repo config %s: %s", repoPath, repoErr))
		}
		for _, warning := range repoWarnings {
			u.Muted(fmt.Sprintf("Ignoring invalid repo config %s: %s", repoPath, warning))
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

// detectBranches returns the list of branches in the repository. If git branch
// fails (e.g. permission error), it logs a muted warning and falls back to
// common defaults. A brand-new repo with zero branches also gets defaults.
func detectBranches(g *git.Git, u *ui.UI) []string {
	branches, err := g.ListBranches()
	if err != nil {
		u.Warning(fmt.Sprintf("Could not detect branches: %s (using defaults)", err))
		branches = []string{"main", "develop", "master"}
	} else if len(branches) == 0 {
		branches = []string{"main", "develop", "master"}
	}
	return branches
}

// buildRepoConfigForEdit preserves untouched repo-only fields and stores edited
// fields as minimal overrides against the inherited defaults+global config.
// Existing repo overrides that currently match inherited values are preserved so
// config edit does not silently unpin them.
func buildRepoConfigForEdit(inherited config.Config, existing *config.PartialConfig, result ui.InitFormResult) config.PartialConfig {
	var updated config.PartialConfig
	if existing != nil {
		updated = *existing
	}

	updated.MainBranch = repoStringOverride(result.MainBranch, inherited.MainBranch, updated.MainBranch)
	updated.FeaturePrefix = repoStringOverride(result.FeaturePrefix, inherited.FeaturePrefix, updated.FeaturePrefix)
	updated.HotfixPrefix = repoStringOverride(result.HotfixPrefix, inherited.HotfixPrefix, updated.HotfixPrefix)
	updated.TagPrefix = repoStringOverride(result.TagPrefix, inherited.TagPrefix, updated.TagPrefix)
	updated.MergeStrategy = repoStringOverride(result.MergeStrategy, inherited.MergeStrategy, updated.MergeStrategy)
	updated.DefaultReleaseBump = repoStringOverride(result.DefaultReleaseBump, inherited.DefaultReleaseBump, updated.DefaultReleaseBump)
	updated.DraftPROnStart = repoBoolOverride(result.DraftPR, inherited.DraftPROnStart, updated.DraftPROnStart)
	updated.HotfixAutoRelease = repoBoolOverride(result.HotfixAutoRelease, inherited.HotfixAutoRelease, updated.HotfixAutoRelease)

	return updated
}

func repoStringOverride(value, inherited, existing string) string {
	if value != inherited {
		return value
	}
	if existing != "" && existing == value {
		return value
	}
	return ""
}

func repoBoolOverride(value, inherited bool, existing *bool) *bool {
	if value != inherited {
		updated := value
		return &updated
	}
	if existing != nil && *existing == value {
		preserved := value
		return &preserved
	}
	return nil
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
