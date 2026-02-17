package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if generated files are up to date",
	Long: `Compares rendered templates against files on disk and reports
whether any files are out of date. Exits with code 0 if all files
are current, or code 1 if there is drift.

This is useful in CI to detect config drift without applying changes.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir, err := resolveConfigDir()
		if err != nil {
			return err
		}

		var buf bytes.Buffer

		opts := UpdateOptions{
			Dir:    dir,
			DryRun: true,
			Quiet:  true,
			Yes:    true,
			Stdout: &buf,
			Stdin:  os.Stdin,
		}

		err = RunUpdate(opts)
		if err != nil {
			// Drift detected — RunUpdate returns error for dry-run with changes.
			fmt.Println("promptkit status: files are out of date.")
			fmt.Println("Run 'promptkit update' to apply changes.")
			// Return the error so cobra exits with code 1.
			return err
		}

		fmt.Println("promptkit status: all files are up to date.")

		return nil
	},
}
