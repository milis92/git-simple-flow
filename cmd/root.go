// Package cmd implements the CLI commands for git-sf.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/ui"
	"github.com/spf13/cobra"
)

var (
	dryRun        bool
	verbose       bool
	noInteractive bool
	autoConfirm   bool
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
		fmt.Fprintf(os.Stderr, "warning: could not determine repository root: %s (using current directory)\n", err)
		return "."
	}
	return strings.TrimSpace(string(out))
}

// loadConfig loads the 3-layer config: built-in defaults, then global
// (~/.config/git-sf/config.yml), then repo (.sfconfig.yml).
func loadConfig() config.Config {
	base := config.Defaults()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not determine home directory: %s\n", err)
		return base
	}
	globalPath := filepath.Join(homeDir, ".config", "git-sf", "config.yml")
	global, err := config.LoadFromFile(globalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load global config %s: %s\n", globalPath, err)
	}
	repoPath := filepath.Join(repoRoot(), ".sfconfig.yml")
	repo, err := config.LoadFromFile(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load repo config %s: %s\n", repoPath, err)
	}
	cfg := config.Merge(base, global, repo)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid config: %s (using defaults)\n", err)
		return config.Defaults()
	}
	return cfg
}

// newUI creates a UI instance with interactive mode set based on TTY and flags.
func newUI() *ui.UI {
	u := ui.New()
	isTTY := ui.IsTerminal(os.Stdin) && ui.IsTerminal(os.Stdout)
	u.Interactive = ui.ShouldInteract(isTTY, noInteractive)
	u.AutoConfirm = autoConfirm
	return u
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print commands without executing them")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print commands as they execute")
	rootCmd.PersistentFlags().BoolVar(&noInteractive, "no-interactive", false, "disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&autoConfirm, "yes", false, "auto-confirm all prompts")
}
