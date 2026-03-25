package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/spf13/cobra"
)

// initCmd creates a .sfconfig.yml file with default settings in the repo root.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create .sfconfig.yml with default settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		path := filepath.Join(repoRoot(), ".sfconfig.yml")
		force, _ := cmd.Flags().GetBool("force")

		var err error
		if force {
			err = config.ForceWriteDefaults(path)
		} else {
			err = config.WriteDefaults(path)
		}

		if err != nil {
			return err
		}

		u.Success("Created " + path)
		u.Muted("Edit this file to customize your workflow.")
		return nil
	},
}

// configCmd displays the effective configuration with source attribution.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		cfg := loadConfig()

		// Load individual layers to show source
		homeDir, _ := os.UserHomeDir()
		globalPath := filepath.Join(homeDir, ".config", "git-sf", "config.yml")
		global, _ := config.LoadFromFile(globalPath)
		repo, _ := config.LoadFromFile(".sfconfig.yml")

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
		fmt.Fprintf(u.Out, "  %-25s %-15v %s\n", "draft_pr_on_start", cfg.DraftPROnStart, "(default)")
		fmt.Fprintf(u.Out, "  %-25s %-15v %s\n", "hotfix_auto_release", cfg.HotfixAutoRelease, "(default)")
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
	fmt.Fprintf(u.Out, "  %-25s %-15s %s\n", name, value, source)
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config file")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
}
