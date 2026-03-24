// cmd/root.go
package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/config"
	"github.com/spf13/cobra"
)

var (
	dryRun  bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "git-sf",
	Short: "Simple Flow — trunk-based Git workflow CLI",
	Long:  "A lightweight, opinionated Git workflow CLI for trunk-based development.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func repoRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "."
	}
	return strings.TrimSpace(string(out))
}

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
