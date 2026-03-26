// Package cmd implements the CLI commands for git-sf.
package cmd

import (
	"fmt"
	"io"
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

	var global *config.PartialConfig
	globalPath, err := globalConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not determine home directory: %s\n", err)
	} else {
		var warnings []error
		var loadErr error
		global, warnings, loadErr = loadPartialConfig(globalPath)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load global config %s: %s\n", globalPath, loadErr)
		}
		printConfigWarnings(os.Stderr, "global", globalPath, warnings)
	}

	repoPath := filepath.Join(repoRoot(), ".sfconfig.yml")
	repo, warnings, err := loadPartialConfig(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load repo config %s: %s\n", repoPath, err)
	}
	printConfigWarnings(os.Stderr, "repo", repoPath, warnings)

	cfg := config.Merge(base, global, repo)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid resolved config: %s (using defaults)\n", err)
		return config.Defaults()
	}
	return cfg
}

func globalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "git-sf", "config.yml"), nil
}

func loadPartialConfig(path string) (*config.PartialConfig, []error, error) {
	partial, err := config.LoadFromFile(path)
	if err != nil {
		return nil, nil, err
	}
	sanitized, warnings := config.SanitizePartial(partial)
	return sanitized, warnings, nil
}

func printConfigWarnings(out io.Writer, scope, path string, warnings []error) {
	for _, warning := range warnings {
		fmt.Fprintf(out, "warning: invalid %s config %s: %s (ignoring field)\n", scope, path, warning)
	}
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
