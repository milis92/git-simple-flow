// Package cmd implements the CLI commands for git-sf.
package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/spf13/cobra"
)

var (
	dryRun  bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "git-sf",
	Short: "Simple Git Flow — all the structure, none of the ceremony",
	Long:  "A Git branching model that sits between Git Flow and GitHub Flow. Feature branches with semver versioning built in.",
}

// Execute runs the root command. It exits with code 1 on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// repoRoot returns the top-level directory of the current git repository,
// or "." if it cannot be determined.
func repoRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "."
	}
	return strings.TrimSpace(string(out))
}

// loadConfig loads the 3-layer config: built-in defaults, then global
// (~/.config/git-sf/config.yml), then repo (.sfconfig.yml).
func loadConfig() config.Config {
	base := config.Defaults()
	homeDir, _ := os.UserHomeDir()
	globalPath := filepath.Join(homeDir, ".config", "git-sf", "config.yml")
	global, _ := config.LoadFromFile(globalPath)
	repoPath := filepath.Join(repoRoot(), ".sfconfig.yml")
	repo, _ := config.LoadFromFile(repoPath)
	return config.Merge(base, global, repo)
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print commands without executing them")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print commands as they execute")
}
