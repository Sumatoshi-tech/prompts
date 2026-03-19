package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/adapters"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/prompt"
	"github.com/Sumatoshi-tech/prompts/internal/scaffold"
)

var initFlags struct {
	force          bool
	dryRun         bool
	nonInteractive bool
	projectName    string
	modulePath     string
	description    string
	expertise      string
	binary         string
	cgo            bool
	docker         bool
	agents         []string
	ecosystem      string
	workflow       string
}

func init() {
	initCmd.Flags().BoolVarP(&initFlags.force, "force", "f", false, "overwrite existing files")
	initCmd.Flags().BoolVar(&initFlags.dryRun, "dry-run", false, "preview files without writing")
	initCmd.Flags().BoolVar(&initFlags.nonInteractive, "non-interactive", false, "use flags instead of prompts")
	initCmd.Flags().StringVar(&initFlags.projectName, "name", "", "project name")
	initCmd.Flags().StringVar(&initFlags.modulePath, "module", "", "module path (e.g. github.com/user/project)")
	initCmd.Flags().StringVar(&initFlags.description, "description", "", "project description")
	initCmd.Flags().StringVar(&initFlags.expertise, "expertise", "", "agent domain expertise")
	initCmd.Flags().StringVar(&initFlags.binary, "binary", "", "binary name")
	initCmd.Flags().BoolVar(&initFlags.cgo, "cgo", false, "enable CGO support")
	initCmd.Flags().BoolVar(&initFlags.docker, "docker", true, "enable Docker support")
	initCmd.Flags().StringSliceVar(
		&initFlags.agents, "ai", []string{"claude"},
		"target AI agents (claude,codex,copilot,cursor,gemini,windsurf)",
	)
	initCmd.Flags().StringVar(&initFlags.ecosystem, "ecosystem", "golang", "template ecosystem (golang, rust, zig)")
	initCmd.Flags().StringVar(&initFlags.workflow, "workflow", "frd", "development workflow (frd, journey)")

	// Register ecosystem-specific flags from modules.
	for _, mod := range config.AllEcosystems() {
		mod.RegisterFlags(initCmd)
	}

	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [project-dir]",
	Short: "Initialize a new project with AI agent instructions",
	Long: `Scaffolds a new Go project with AGENTS.md, .agents/instructions/,
Makefile, golangci-lint config, and helper scripts.

If project-dir is omitted, the current directory is used.

Example (non-interactive):
  promptkit init my-project --non-interactive --name "myapp" \
    --module "github.com/user/myapp" --expertise "distributed systems" \
    --ai claude,cursor`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func runInit(_ *cobra.Command, args []string) error {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if err = os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Check if config already exists.
	configPath := filepath.Join(targetDir, config.FileName)
	configExists := false

	if _, statErr := os.Stat(configPath); statErr == nil {
		configExists = true

		if !initFlags.force {
			return fmt.Errorf("%s already exists (use --force to overwrite)", config.FileName)
		}
	}

	// Prevent concurrent execution.
	if lockErr := scaffold.AcquireLock(targetDir); lockErr != nil {
		return lockErr
	}
	defer scaffold.ReleaseLock(targetDir)

	// Progress helper.
	totalSteps := 3
	step := 0
	progress := func(msg string) {
		step++
		fmt.Printf("[%d/%d] %s\n", step, totalSteps, msg)
	}

	progress("Building config...")

	cfg, err := buildConfig(targetDir)
	if err != nil {
		return err
	}

	// Track generated files in manifest before saving config.
	progress("Rendering templates...")

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	cfg.GeneratedFiles = scaffold.FileManifest(rendered)

	if rootFlags.verbose {
		fmt.Printf("  Rendered %d file(s) for agents: %s\n", len(rendered), strings.Join(cfg.Agents, ", "))
	}

	// Dry run: show what would be generated without writing.
	if initFlags.dryRun {
		printFilesByAgent(rendered, cfg.Agents, cfg.Workflow)
		fmt.Println("\nDry run — no files were written.")

		return nil
	}

	if err = config.Save(cfg, targetDir); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Wrote %s\n", config.FileName)

	// When --force is used and config already existed, delegate to RunUpdate
	// to get diffs, stale detection, backup, and overrides.
	if initFlags.force && configExists {
		return RunUpdate(UpdateOptions{
			Dir:     targetDir,
			Yes:     true,
			DryRun:  initFlags.dryRun,
			Verbose: rootFlags.verbose,
			Stdout:  os.Stdout,
			Stdin:   os.Stdin,
		})
	}

	progress("Writing files...")

	// Backup existing files before applying (for rollback on failure).
	existingPaths := make([]string, 0)

	for path := range rendered {
		fullPath := filepath.Join(targetDir, path)
		if _, err = os.Stat(fullPath); err == nil {
			existingPaths = append(existingPaths, path)
		}
	}

	var backupDir string
	if len(existingPaths) > 0 {
		backupDir, err = scaffold.BackupFiles(targetDir, existingPaths)
		if err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
	}

	if err = scaffold.Apply(rendered, targetDir, scaffold.ModeCreate); err != nil {
		if backupDir != "" {
			if restoreErr := scaffold.RestoreBackup(backupDir, targetDir); restoreErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: backup restore also failed: %v\n", restoreErr)
			} else {
				fmt.Fprintln(os.Stderr, "Restored files from backup after apply failure.")
			}
		}

		return fmt.Errorf("applying templates: %w", err)
	}

	// Store checksums in config.
	cfg.Checksums = scaffold.ComputeChecksums(rendered)
	if err = config.Save(cfg, targetDir); err != nil {
		return fmt.Errorf("saving config checksums: %w", err)
	}

	printFilesByAgent(rendered, cfg.Agents, cfg.Workflow)

	fmt.Printf("\nProject initialized in %s\n", targetDir)

	return nil
}

func buildConfig(targetDir string) (*config.Config, error) {
	cfg := config.Default()

	if initFlags.nonInteractive {
		return buildConfigFromFlags(cfg, targetDir)
	}

	return prompt.RunInitPrompts(cfg, filepath.Base(targetDir))
}

func buildConfigFromFlags(cfg *config.Config, targetDir string) (*config.Config, error) {
	cfg.Ecosystem = initFlags.ecosystem
	cfg.Workflow = initFlags.workflow

	if initFlags.projectName != "" {
		cfg.ProjectName = initFlags.projectName
	} else {
		cfg.ProjectName = filepath.Base(targetDir)
	}

	if initFlags.modulePath != "" {
		cfg.ModulePath = initFlags.modulePath
	}

	if initFlags.description != "" {
		cfg.Description = initFlags.description
	}

	if initFlags.expertise != "" {
		cfg.Expertise = initFlags.expertise
	}

	binaryName := cfg.ProjectName
	if initFlags.binary != "" {
		binaryName = initFlags.binary
	}

	// Delegate ecosystem-specific defaults and flag application to the module.
	mod := config.GetEcosystem(cfg.Ecosystem)
	if mod != nil {
		mod.ApplyFlags(cfg)

		cfg.Binaries = []config.Binary{
			{Name: binaryName, CmdPath: mod.DefaultCmdPath(binaryName)},
		}

		cfg.AnalysisCmd = mod.DefaultAnalysisCmd
	}

	cfg.Features.CGO = initFlags.cgo
	cfg.Features.Docker = initFlags.docker

	if len(initFlags.agents) > 0 {
		cfg.Agents = initFlags.agents
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// printFilesByAgent prints generated files grouped by agent ownership.
func printFilesByAgent(rendered map[string][]byte, agents []string, workflow string) {
	ownership, err := adapters.FileOwnership(rendered, agents)
	if err != nil {
		// Fallback to flat list on error.
		paths := make([]string, 0, len(rendered))
		for p := range rendered {
			paths = append(paths, p)
		}

		sort.Strings(paths)

		fmt.Printf("Wrote %d file(s):\n", len(rendered))

		for _, p := range paths {
			fmt.Printf("  %s\n", p)
		}

		return
	}

	// Categorize files.
	var basePaths []string

	agentFiles := make(map[string][]string)

	for path, fa := range ownership {
		switch {
		case len(fa.Agents) == 0:
			basePaths = append(basePaths, path)
		case fa.IsShared:
			basePaths = append(basePaths, path)
		default:
			agent := fa.Agents[0]
			agentFiles[agent] = append(agentFiles[agent], path)
		}
	}

	sort.Strings(basePaths)

	fmt.Printf("Wrote %d file(s):\n", len(rendered))

	if len(basePaths) > 0 {
		fmt.Println("  Shared:")

		for _, p := range basePaths {
			fmt.Printf("    %s\n", p)
		}
	}

	// Print per-agent files in sorted agent order.
	sortedAgents := make([]string, 0, len(agentFiles))
	for a := range agentFiles {
		sortedAgents = append(sortedAgents, a)
	}

	sort.Strings(sortedAgents)

	for _, agent := range sortedAgents {
		files := agentFiles[agent]
		sort.Strings(files)

		fmt.Printf("  %s (%d files):\n", agent, len(files))

		for _, f := range files {
			fmt.Printf("    %s\n", f)
		}
	}

	// Summary line.
	if len(agents) > 1 {
		fmt.Printf("\nAgents: %s\n", strings.Join(agents, ", "))
	}
}
