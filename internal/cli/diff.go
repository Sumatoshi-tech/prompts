package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	promptkit "github.com/Sumatoshi-tech/promptkit"
	"github.com/Sumatoshi-tech/promptkit/internal/config"
	"github.com/Sumatoshi-tech/promptkit/internal/scaffold"
	"github.com/spf13/cobra"
)

var diffFlags struct {
	upstream string
}

func init() {
	diffCmd.Flags().StringVar(&diffFlags.upstream, "upstream", "", "compare against a reference config directory")

	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Preview what update would change",
	Long: `Shows unified diffs for all files that would change without writing anything.

Without flags, this is equivalent to 'promptkit update --dry-run'.

With --upstream <path>, compares local rendered output against a reference
config's rendered output. Useful for comparing team configurations.

Exit code 0 if all files are up to date, 1 if changes are pending.`,
	RunE: runDiff,
}

func runDiff(_ *cobra.Command, _ []string) error {
	if diffFlags.upstream != "" {
		return runDiffUpstream()
	}

	dir, err := resolveConfigDir()
	if err != nil {
		return err
	}

	return RunUpdate(UpdateOptions{
		Dir:     dir,
		DryRun:  true,
		Verbose: rootFlags.verbose,
		Stdout:  os.Stdout,
		Stdin:   os.Stdin,
	})
}

func runDiffUpstream() error {
	// Load local config.
	localDir, err := resolveConfigDir()
	if err != nil {
		return fmt.Errorf("local config: %w", err)
	}

	localCfg, err := config.Load(localDir)
	if err != nil {
		return fmt.Errorf("loading local config: %w", err)
	}

	// Load upstream config.
	upstreamCfg, err := config.Load(diffFlags.upstream)
	if err != nil {
		return fmt.Errorf("loading upstream config from %s: %w", diffFlags.upstream, err)
	}

	// Render both.
	localRendered, err := scaffold.RenderFullWithOverrides(localCfg, promptkit.Templates, localCfg.TemplateOver)
	if err != nil {
		return fmt.Errorf("rendering local templates: %w", err)
	}

	upstreamRendered, err := scaffold.RenderFullWithOverrides(upstreamCfg, promptkit.Templates, upstreamCfg.TemplateOver)
	if err != nil {
		return fmt.Errorf("rendering upstream templates: %w", err)
	}

	// Compare.
	diffs := scaffold.DiffRendered(localRendered, upstreamRendered)

	if len(diffs) == 0 {
		fmt.Println("Local and upstream configs produce identical output.")
		return nil
	}

	// Sort for deterministic output.
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})

	fmt.Printf("Found %d file(s) with differences (local vs upstream):\n", len(diffs))

	for _, d := range diffs {
		switch {
		case d.IsNew:
			fmt.Printf("  + %s (only in local)\n", d.Path)
		case len(d.Rendered) == 0:
			fmt.Printf("  - %s (only in upstream)\n", d.Path)
		default:
			fmt.Printf("  ~ %s (modified)\n", d.Path)
		}
	}

	fmt.Println()

	for _, d := range diffs {
		switch {
		case d.IsNew:
			lines := strings.SplitN(string(d.Rendered), "\n", 11)
			preview := lines
			if len(preview) > 10 {
				preview = preview[:10]
			}

			fmt.Printf("--- upstream (not present)\n+++ local/%s\n", d.Path)

			for _, line := range preview {
				fmt.Printf("+%s\n", line)
			}

			if len(lines) > 10 {
				fmt.Printf("+... (%d more lines)\n", len(lines)-10)
			}

			fmt.Println()

		case len(d.Rendered) == 0:
			lines := strings.SplitN(string(d.Existing), "\n", 11)
			preview := lines
			if len(preview) > 10 {
				preview = preview[:10]
			}

			fmt.Printf("--- upstream/%s\n+++ local (not present)\n", d.Path)

			for _, line := range preview {
				fmt.Printf("-%s\n", line)
			}

			if len(lines) > 10 {
				fmt.Printf("-... (%d more lines)\n", len(lines)-10)
			}

			fmt.Println()

		default:
			diff := scaffold.UnifiedDiff(d.Existing, d.Rendered, d.Path)
			if diff != "" {
				fmt.Print(diff)
				fmt.Println()
			}
		}
	}

	return fmt.Errorf("configs differ (use 'promptkit config explain' to understand differences)")
}
