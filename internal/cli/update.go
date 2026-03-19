package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/adapters"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/scaffold"
)

var updateFlags struct {
	yes         bool
	dryRun      bool
	verify      bool
	explain     bool
	interactive bool
}

func init() {
	updateCmd.Flags().BoolVarP(&updateFlags.yes, "yes", "y", false, "auto-approve all changes")
	updateCmd.Flags().BoolVar(&updateFlags.dryRun, "dry-run", false, "preview changes without writing files")
	updateCmd.Flags().BoolVar(&updateFlags.verify, "verify", false, "run analysis command after applying changes")
	updateCmd.Flags().BoolVar(&updateFlags.explain, "explain", false, "explain the update pipeline before running")
	updateCmd.Flags().BoolVarP(&updateFlags.interactive, "interactive", "i", false, "approve each file individually")
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Re-render templates from current config",
	Long: `Re-reads .promptkit.yaml and re-renders all templates.
Shows a unified diff for each changed file before writing.

Pipeline: load config -> render templates -> apply overrides -> generate
agent files -> compute diffs -> show changes -> approve -> apply atomically.

Use --dry-run to preview changes without writing.
Use --verify to run the analysis command after applying.
Use --explain to see a detailed pipeline description before running.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		dir, err := resolveConfigDir()
		if err != nil {
			return err
		}

		opts := UpdateOptions{
			Dir:         dir,
			Yes:         updateFlags.yes,
			DryRun:      updateFlags.dryRun,
			Verify:      updateFlags.verify,
			Explain:     updateFlags.explain,
			Interactive: updateFlags.interactive,
			Verbose:     rootFlags.verbose,
			Stdout:      os.Stdout,
			Stdin:       os.Stdin,
		}

		return RunUpdate(opts)
	},
}

// UpdateOptions configures the update behavior.
type UpdateOptions struct {
	Dir         string
	Yes         bool
	DryRun      bool
	Verify      bool
	Explain     bool
	Interactive bool
	Verbose     bool
	Quiet       bool
	Stdout      io.Writer
	Stdin       io.Reader
}

// RunUpdate executes the update workflow with the given options.
func RunUpdate(opts UpdateOptions) error {
	w := opts.Stdout
	start := time.Now()

	// Prevent concurrent execution.
	if err := scaffold.AcquireLock(opts.Dir); err != nil {
		return err
	}
	defer scaffold.ReleaseLock(opts.Dir)

	// Explain the pipeline if requested.
	if opts.Explain {
		fmt.Fprintln(w, "promptkit update pipeline:")
		fmt.Fprintln(w, "  1. Load and validate .promptkit.yaml")
		fmt.Fprintln(w, "  2. Render embedded templates using config values")
		fmt.Fprintln(w, "  3. Apply local overrides from .promptkit/templates/ (if any)")
		fmt.Fprintln(w, "  4. Generate agent-specific files for configured agents")
		fmt.Fprintln(w, "  5. Compare rendered output against existing files on disk")
		fmt.Fprintln(w, "  6. Show unified diffs for changed files")
		fmt.Fprintln(w, "  7. Prompt for approval (or auto-approve with --yes)")
		fmt.Fprintln(w, "  8. Back up existing files, write approved files atomically")
		fmt.Fprintln(w, "  9. Remove stale files, update manifest")

		if opts.Verify {
			fmt.Fprintln(w, " 10. Run verification command")
		}

		fmt.Fprintln(w)
	}

	// Determine step count for progress indicators.
	totalSteps := 4 // load, render, diff, apply.
	if opts.DryRun {
		totalSteps = 3 // load, render, diff (no apply).
	}

	if opts.Verify {
		totalSteps++ // adds verify step.
	}

	step := 0

	progress := func(msg string) {
		if opts.Quiet {
			return
		}

		step++
		fmt.Fprintf(w, "[%d/%d] %s\n", step, totalSteps, msg)
	}

	// Load config.
	progress("Loading config...")

	cfg, err := config.Load(opts.Dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	overrideDir := cfg.TemplateOver

	// Check for stale overrides.
	if !opts.Quiet {
		staleOverrides := scaffold.CheckOverrideStaleness(promptkit.Templates, overrideDir, cfg.Ecosystem)
		if len(staleOverrides) > 0 {
			fmt.Fprintln(w, "Warning: These template overrides may be stale (upstream template changed):")

			for _, s := range staleOverrides {
				name := strings.TrimSuffix(s, ".tmpl")
				fmt.Fprintf(w, "  %s/%s\n", overrideDir, s)
				fmt.Fprintf(w, "  Run 'promptkit template extract %s --force' to update.\n", name)
			}

			fmt.Fprintln(w)
		}
	}

	// Render templates.
	progress("Rendering templates...")

	renderStart := time.Now()

	rendered, err := scaffold.RenderFullWithOverrides(cfg, promptkit.Templates, overrideDir)
	if err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	// Print verbose details within this step.
	if !opts.Quiet && opts.Verbose {
		fmt.Fprintf(w, "  Rendered %d file(s) in %s\n", len(rendered), time.Since(renderStart))

		if overrideDir != "" {
			fmt.Fprintf(w, "  Applying template overrides from %s\n", overrideDir)
		}

		if len(cfg.Agents) > 0 {
			fmt.Fprintf(w, "  Generating adapters for agents: %s\n", strings.Join(cfg.Agents, ", "))
		}

		// Log per-file agent ownership.
		var ownership map[string]adapters.FileAgent
		if ownership, err = adapters.FileOwnership(rendered, cfg.Agents); err == nil {
			for _, agent := range cfg.Agents {
				var files []string

				for path, fa := range ownership {
					if !fa.IsShared {
						for _, a := range fa.Agents {
							if a == agent {
								files = append(files, path)
							}
						}
					}
				}

				if len(files) > 0 {
					sort.Strings(files)
					fmt.Fprintf(w, "  %s: %d file(s)\n", agent, len(files))
				}
			}
		}
	}

	// Compute diffs.
	progress("Computing diffs...")

	diffs, err := scaffold.Diff(rendered, opts.Dir)
	if err != nil {
		return fmt.Errorf("computing diffs: %w", err)
	}

	// Detect stale files.
	stale := scaffold.DetectStale(rendered, cfg.GeneratedFiles)

	if len(diffs) == 0 && len(stale) == 0 {
		fmt.Fprintln(w, "All files are up to date.")
		return nil
	}

	// Warn about manually modified files in verbose mode.
	if opts.Verbose && cfg.Checksums != nil {
		for _, d := range diffs {
			if d.IsNew {
				continue
			}

			storedSum, ok := cfg.Checksums[d.Path]
			if !ok {
				continue
			}

			diskSum := sha256hex(d.Existing)
			renderedSum := sha256hex(d.Rendered)

			if diskSum != storedSum && diskSum != renderedSum {
				fmt.Fprintf(w, "  Warning: %s was manually modified since last generation\n", d.Path)
			}
		}
	}

	// Sort diffs for deterministic output.
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})

	// Display summary with change and agent annotations.
	reverseMap := config.ReverseFieldMap()
	// Best-effort; display continues without annotations.
	ownership, _ := adapters.FileOwnership(rendered, cfg.Agents)

	if len(diffs) > 0 {
		fmt.Fprintf(w, "Found %d file(s) with changes:\n", len(diffs))

		for _, d := range diffs {
			// Build annotation: field keys + agent ownership.
			var parts []string

			if keys, ok := reverseMap[d.Path]; ok && len(keys) > 0 {
				parts = append(parts, strings.Join(keys, ", "))
			}

			if fa, ok := ownership[d.Path]; ok && len(fa.Agents) > 0 {
				if fa.IsShared {
					parts = append(parts, "shared")
				} else {
					parts = append(parts, fa.Agents[0])
				}
			}

			annotation := ""
			if len(parts) > 0 {
				annotation = " [" + strings.Join(parts, " | ") + "]"
			}

			if d.IsNew {
				fmt.Fprintf(w, "  + %s (new)%s\n", d.Path, annotation)
			} else {
				fmt.Fprintf(w, "  ~ %s (modified)%s\n", d.Path, annotation)
			}
		}
	}

	if len(stale) > 0 {
		fmt.Fprintf(w, "\nStale files (no longer generated):\n")

		for _, s := range stale {
			fmt.Fprintf(w, "  - %s\n", s)
		}
	}

	// Display unified diffs.
	fmt.Fprintln(w)

	for _, d := range diffs {
		if d.IsNew {
			lines := strings.SplitN(string(d.Rendered), "\n", 11)
			preview := lines
			truncated := false

			if len(preview) > 10 {
				preview = preview[:10]
				truncated = true
			}

			fmt.Fprintf(w, "--- /dev/null\n+++ b/%s\n", d.Path)

			for _, line := range preview {
				fmt.Fprintf(w, "+%s\n", line)
			}

			if truncated {
				fmt.Fprintf(w, "+... (%d more lines)\n", len(lines)-10)
			}

			fmt.Fprintln(w)
		} else {
			diff := scaffold.UnifiedDiff(d.Existing, d.Rendered, d.Path)
			if diff != "" {
				fmt.Fprint(w, diff)
				fmt.Fprintln(w)
			}
		}
	}

	// Dry run: show summary and exit.
	if opts.DryRun {
		fmt.Fprintln(w, "Dry run — no files were written.")

		if opts.Verbose {
			fmt.Fprintf(w, "Total time: %s\n", time.Since(start))
		}

		// Exit with code 1 if there are changes (useful for CI drift detection).
		if len(diffs) > 0 || len(stale) > 0 {
			return errors.New("files are out of date (use 'promptkit update' to apply)")
		}

		return nil
	}

	// Determine which files to apply based on approval mode.
	approvedDiffs := diffs
	approvedStale := stale

	if opts.Interactive && !opts.Yes {
		// Per-file approval mode.
		approvedDiffs = nil
		applyAll := false

		for _, d := range diffs {
			if applyAll {
				approvedDiffs = append(approvedDiffs, d)
				continue
			}

			fmt.Fprintf(w, "Apply %s? [y/n/a/q] ", d.Path)

			var answer string

			_, _ = fmt.Fscanln(opts.Stdin, &answer)

			switch strings.ToLower(answer) {
			case "y":
				approvedDiffs = append(approvedDiffs, d)
			case "a":
				applyAll = true

				approvedDiffs = append(approvedDiffs, d)
			case "q":
				goto doneApproval
			default:
			}
		}

		// Per-file approval for stale file removal.
		if len(stale) > 0 && !applyAll {
			approvedStale = nil

			for _, s := range stale {
				if applyAll {
					approvedStale = append(approvedStale, s)
					continue
				}

				fmt.Fprintf(w, "Remove stale %s? [y/n/a/q] ", s)

				var answer string

				_, _ = fmt.Fscanln(opts.Stdin, &answer)

				switch strings.ToLower(answer) {
				case "y":
					approvedStale = append(approvedStale, s)
				case "a":
					applyAll = true

					approvedStale = append(approvedStale, s)
				case "q":
					goto doneApproval
				default:
				}
			}
		}

	doneApproval:

		if len(approvedDiffs) == 0 && len(approvedStale) == 0 {
			fmt.Fprintln(w, "No files approved. Aborted.")
			return nil
		}
	} else if !opts.Yes {
		// Single approval for all changes.
		fmt.Fprint(w, "Apply changes? [y/N] ")

		var answer string

		_, _ = fmt.Fscanln(opts.Stdin, &answer)

		if answer != "y" && answer != "Y" {
			fmt.Fprintln(w, "Aborted.")
			return nil
		}
	}

	// Build the set of files to apply.
	toApply := make(map[string][]byte)
	approvedPaths := make(map[string]bool)

	for _, d := range approvedDiffs {
		approvedPaths[d.Path] = true
	}

	for path, content := range rendered {
		if approvedPaths[path] {
			// File has changes and was approved.
			toApply[path] = content
		} else if !hasDiff(diffs, path) {
			// File is unchanged — include it (no approval needed).
			toApply[path] = content
		}
	}

	// Apply changes.
	progress("Applying changes...")

	// Backup existing files before overwriting.
	allPaths := make([]string, 0, len(approvedDiffs))
	for _, d := range approvedDiffs {
		allPaths = append(allPaths, d.Path)
	}

	backupDir, err := scaffold.BackupFiles(opts.Dir, allPaths)
	if err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	if backupDir != "" && opts.Verbose {
		fmt.Fprintf(w, "  Backed up files to %s\n", backupDir)
	}

	if err = scaffold.Apply(toApply, opts.Dir, scaffold.ModeForce); err != nil {
		// Attempt restore on failure.
		if backupDir != "" {
			if restoreErr := scaffold.RestoreBackup(backupDir, opts.Dir); restoreErr != nil {
				fmt.Fprintf(w, "Warning: backup restore also failed: %v\n", restoreErr)
			} else {
				fmt.Fprintln(w, "Restored files from backup after apply failure.")
			}
		}

		return fmt.Errorf("applying changes: %w", err)
	}

	// Remove approved stale files.
	if len(approvedStale) > 0 {
		if err = scaffold.RemoveFiles(opts.Dir, approvedStale); err != nil {
			return fmt.Errorf("removing stale files: %w", err)
		}
	}

	// Update manifest in config.
	cfg.GeneratedFiles = scaffold.FileManifest(rendered)
	cfg.Checksums = scaffold.ComputeChecksums(rendered)

	if err = config.Save(cfg, opts.Dir); err != nil {
		return fmt.Errorf("saving config manifest: %w", err)
	}

	// Print updated file list.
	if len(approvedDiffs) > 0 {
		fmt.Fprintf(w, "Updated %d file(s):\n", len(approvedDiffs))

		for _, d := range approvedDiffs {
			fmt.Fprintf(w, "  %s\n", d.Path)
		}
	}

	if len(approvedStale) > 0 {
		fmt.Fprintf(w, "Removed %d stale file(s):\n", len(approvedStale))

		for _, s := range approvedStale {
			fmt.Fprintf(w, "  %s\n", s)
		}
	}

	// Verify if requested.
	if opts.Verify {
		progress("Running verification...")

		if err = runVerify(w, opts.Dir, cfg.AnalysisCmd); err != nil {
			if backupDir != "" {
				fmt.Fprintf(w, "To restore: copy files from %s\n", backupDir)
			}

			return err
		}
	}

	if opts.Verbose {
		fmt.Fprintf(w, "\nTotal time: %s\n", time.Since(start))
	}

	return nil
}

// resolveConfigDir determines the directory containing .promptkit.yaml.
// Uses --config flag if set, otherwise searches from cwd upward.
func resolveConfigDir() (string, error) {
	if rootFlags.configPath != "" {
		return rootFlags.configPath, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	found, err := config.FindConfig(dir)
	if err != nil {
		return "", err
	}

	return found, nil
}

func hasDiff(diffs []scaffold.FileDiff, path string) bool {
	for _, d := range diffs {
		if d.Path == path {
			return true
		}
	}

	return false
}

func runVerify(w io.Writer, dir, analysisCmd string) error {
	cmd := "make"
	args := []string{"lint"}

	if analysisCmd != "" {
		parts := strings.Fields(analysisCmd)
		cmd = parts[0]
		args = parts[1:]
	}

	fmt.Fprintf(w, "\nRunning verification: %s %s\n", cmd, strings.Join(args, " "))

	c := exec.CommandContext(context.Background(), cmd, args...)
	c.Dir = dir
	c.Stdout = w
	c.Stderr = w

	if err := c.Run(); err != nil {
		fmt.Fprintf(w, "\nVerification failed. To rollback: git checkout -- .\n")
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Fprintln(w, "Verification passed.")

	return nil
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
