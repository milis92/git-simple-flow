// cmd/root.go
package cmd

import (
	"os"

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

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print commands without executing them")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print commands as they execute")
}
