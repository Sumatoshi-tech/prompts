package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Sumatoshi-tech/promptkit/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configExplainCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage promptkit configuration",
}

var configExplainCmd = &cobra.Command{
	Use:   "explain [key]",
	Short: "Show which output files are affected by config fields",
	Long: `Displays the mapping between config keys and the output files they affect.
Without arguments, shows all mappings. With a key argument, shows just that key.

Examples:
  promptkit config explain
  promptkit config explain quality.coverage_min
  promptkit config explain features.docker`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigExplain,
}

func runConfigExplain(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		key := args[0]
		files, desc, ok := config.ExplainField(key)
		if !ok {
			// List valid keys in the error.
			keys := make([]string, 0, len(config.FieldFiles))
			for k := range config.FieldFiles {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			return fmt.Errorf("unknown config key %q\nValid keys: %s", key, strings.Join(keys, ", "))
		}

		fmt.Printf("%s\n", key)
		if desc != "" {
			fmt.Printf("  %s\n", desc)
		}

		if len(files) == 0 {
			fmt.Println("  Affects: agent-specific file placement")
		} else {
			fmt.Printf("  Affects: %s\n", strings.Join(files, ", "))
		}

		return nil
	}

	// Show all mappings.
	keys := make([]string, 0, len(config.FieldFiles))
	for k := range config.FieldFiles {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	fmt.Println("Config field -> output file mapping:")
	fmt.Println()

	for _, key := range keys {
		files, desc, _ := config.ExplainField(key)
		fmt.Printf("  %s\n", key)
		if desc != "" {
			fmt.Printf("    %s\n", desc)
		}

		if len(files) == 0 {
			fmt.Println("    Affects: agent-specific file placement")
		} else {
			fmt.Printf("    Affects: %s\n", strings.Join(files, ", "))
		}
	}

	return nil
}
