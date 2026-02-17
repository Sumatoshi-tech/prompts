package cli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	promptkit "github.com/Sumatoshi-tech/promptkit"
	"github.com/Sumatoshi-tech/promptkit/internal/adapters"
	"github.com/Sumatoshi-tech/promptkit/internal/config"
	"github.com/Sumatoshi-tech/promptkit/internal/scaffold"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check generated files for correctness",
	Long: `Validates that all generated files exist and are consistent with
the current config. Reports missing files, stale files, and files
that have been manually modified since the last generation.

Exit code 0 if all checks pass (warnings are ok), 1 if errors found.`,
	RunE: runDoctor,
}

func runDoctor(_ *cobra.Command, _ []string) error {
	dir, err := resolveConfigDir()
	if err != nil {
		return err
	}

	fmt.Println("promptkit doctor:")

	// Check 1: Config loads and validates.
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Printf("  [err] Config: %v\n", err)
		return fmt.Errorf("doctor found errors")
	}

	fmt.Println("  [ok] Config loads and validates")

	hasErrors := false

	// Check 2: All generated files exist on disk.
	missing := 0

	for _, path := range cfg.GeneratedFiles {
		fullPath := filepath.Join(dir, path)
		if _, err := os.Stat(fullPath); err != nil {
			fmt.Printf("  [err] Missing: %s\n", path)
			missing++
		}
	}

	if missing > 0 {
		hasErrors = true
	} else {
		fmt.Printf("  [ok] %d/%d generated files exist\n",
			len(cfg.GeneratedFiles), len(cfg.GeneratedFiles))
	}

	// Check 3: Agent-specific files exist.
	rendered, err := scaffold.RenderFullWithOverrides(cfg, promptkit.Templates, cfg.TemplateOver)
	if err != nil {
		fmt.Printf("  [err] Render failed: %v\n", err)
		return fmt.Errorf("doctor found errors")
	}

	ownership, _ := adapters.FileOwnership(rendered, cfg.Agents, cfg.Workflow)

	for _, agent := range cfg.Agents {
		total := 0
		present := 0

		for _, fa := range ownership {
			for _, a := range fa.Agents {
				if a == agent {
					total++

					fullPath := filepath.Join(dir, fa.Path)
					if _, err := os.Stat(fullPath); err == nil {
						present++
					}
				}
			}
		}

		if present == total {
			fmt.Printf("  [ok] %s: %d/%d agent files present\n", agent, present, total)
		} else {
			fmt.Printf("  [err] %s: %d/%d agent files present\n", agent, present, total)
			hasErrors = true
		}
	}

	// Check 4: Modified files (warnings).
	if cfg.Checksums != nil {
		modified := 0

		paths := make([]string, 0, len(cfg.Checksums))
		for p := range cfg.Checksums {
			paths = append(paths, p)
		}

		sort.Strings(paths)

		for _, path := range paths {
			storedSum := cfg.Checksums[path]
			fullPath := filepath.Join(dir, path)

			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			diskSum := fmt.Sprintf("%x", sha256.Sum256(data))
			if diskSum != storedSum {
				fmt.Printf("  [warn] %s: modified since last generation\n", path)
				modified++
			}
		}

		if modified == 0 {
			fmt.Println("  [ok] No manually modified files")
		}
	}

	// Check 5: Stale files.
	stale := scaffold.DetectStale(rendered, cfg.GeneratedFiles)
	if len(stale) > 0 {
		for _, s := range stale {
			fmt.Printf("  [warn] Stale: %s\n", s)
		}
	} else {
		fmt.Println("  [ok] No stale files")
	}

	// Check 6: Override staleness.
	staleOverrides := scaffold.CheckOverrideStaleness(promptkit.Templates, cfg.TemplateOver, cfg.Ecosystem)
	if len(staleOverrides) > 0 {
		for _, s := range staleOverrides {
			fmt.Printf("  [warn] Override may be stale: %s\n", s)
		}
	}

	if hasErrors {
		return fmt.Errorf("doctor found errors")
	}

	return nil
}
