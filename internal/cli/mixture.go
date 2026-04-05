package cli

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/mixtures"
)

func init() {
	rootCmd.AddCommand(mixtureCmd)
	mixtureCmd.AddCommand(mixtureListCmd)
	mixtureCmd.AddCommand(mixtureAddCmd)
	mixtureCmd.AddCommand(mixtureRemoveCmd)
}

var mixtureCmd = &cobra.Command{
	Use:   "mixture",
	Short: "Manage instruction mixtures",
	Long: `Mixtures are cross-cutting concerns injected into targeted skills.
For example, a "security" mixture adds threat modeling and secure coding
guidance to /implement, /bug, and /researcher skills.`,
}

var mixtureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available mixtures",
	RunE:  runMixtureList,
}

var mixtureAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a mixture to the project",
	Long: `Adds the named mixture to .promptkit.yaml and regenerates files.
Run 'promptkit mixture list' to see available mixtures.`,
	Args: cobra.ExactArgs(1),
	RunE: runMixtureAdd,
}

var mixtureRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a mixture from the project",
	Args:  cobra.ExactArgs(1),
	RunE:  runMixtureRemove,
}

func runMixtureList(_ *cobra.Command, _ []string) error {
	allDefs, err := mixtures.LoadAll(promptkit.Templates)
	if err != nil {
		return fmt.Errorf("loading mixtures: %w", err)
	}

	if len(allDefs) == 0 {
		fmt.Println("No mixtures available.")
		return nil
	}

	// Check which mixtures are active in the current project.
	var activeMixtures map[string]bool

	dir, findErr := resolveConfigDir()
	if findErr == nil {
		if cfg, loadErr := config.Load(dir); loadErr == nil {
			activeMixtures = make(map[string]bool, len(cfg.Mixtures))
			for _, m := range cfg.Mixtures {
				activeMixtures[m] = true
			}
		}
	}

	names := make([]string, 0, len(allDefs))
	for name := range allDefs {
		names = append(names, name)
	}

	sort.Strings(names)

	fmt.Println("Available mixtures:")
	fmt.Println()

	for _, name := range names {
		def := allDefs[name]

		marker := "  "
		if activeMixtures[name] {
			marker = "* "
		}

		fmt.Printf("%s%-15s %s\n", marker, name, def.Description)

		if def.UseCase != "" {
			fmt.Printf("  %-15s Use-case: %s\n", "", def.UseCase)
		}

		fmt.Printf("  %-15s Targets: %s\n", "", strings.Join(def.Targets, ", "))
		fmt.Println()
	}

	if len(activeMixtures) > 0 {
		fmt.Println("* = active in current project")
	}

	return nil
}

func runMixtureAdd(_ *cobra.Command, args []string) error {
	name := args[0]

	// Validate mixture exists.
	allDefs, err := mixtures.LoadAll(promptkit.Templates)
	if err != nil {
		return fmt.Errorf("loading mixtures: %w", err)
	}

	if _, ok := allDefs[name]; !ok {
		available := make([]string, 0, len(allDefs))
		for n := range allDefs {
			available = append(available, n)
		}

		sort.Strings(available)

		return fmt.Errorf("unknown mixture %q (available: %s)", name, strings.Join(available, ", "))
	}

	dir, err := resolveConfigDir()
	if err != nil {
		return err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check if already present.
	if slices.Contains(cfg.Mixtures, name) {
		fmt.Printf("Mixture %q is already active.\n", name)
		return nil
	}

	cfg.Mixtures = append(cfg.Mixtures, name)
	sort.Strings(cfg.Mixtures)

	if err = config.Save(cfg, dir); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Added mixture %q. Run 'promptkit update' to regenerate files.\n", name)

	return nil
}

func runMixtureRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	dir, err := resolveConfigDir()
	if err != nil {
		return err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	found := false
	filtered := make([]string, 0, len(cfg.Mixtures))

	for _, m := range cfg.Mixtures {
		if m == name {
			found = true
			continue
		}

		filtered = append(filtered, m)
	}

	if !found {
		fmt.Printf("Mixture %q is not active.\n", name)
		return nil
	}

	cfg.Mixtures = filtered

	if err = config.Save(cfg, dir); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Removed mixture %q. Run 'promptkit update' to regenerate files.\n", name)

	return nil
}
