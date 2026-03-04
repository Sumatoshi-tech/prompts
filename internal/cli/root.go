// Package cli implements the promptkit command-line interface.
package cli

import (
	"github.com/spf13/cobra"
)

var rootFlags struct {
	verbose    bool
	configPath string
}

var rootCmd = &cobra.Command{
	Use:   "promptkit",
	Short: "Spec-driven development CLI for AI coding agents",
	Long: `promptkit scaffolds and manages AI agent instructions for Go projects.

It generates AGENTS.md, instruction files, Makefile, linter config,
and other development files from templates, customized for your project.

Workflow:
  1. promptkit init       - scaffold a new project
  2. Edit .promptkit.yaml - customize settings
  3. promptkit update     - re-render templates after config changes
  4. promptkit template   - manage template overrides`,
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(
		&rootFlags.configPath, "config", "",
		"path to config directory (default: search from current directory upward)",
	)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
