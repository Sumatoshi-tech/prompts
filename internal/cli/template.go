package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/scaffold"
)

var extractFlags struct {
	force bool
}

func init() {
	templateExtractCmd.Flags().BoolVarP(&extractFlags.force, "force", "f", false, "overwrite existing override")

	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateAddCmd)
	templateCmd.AddCommand(templateRenderCmd)
	templateCmd.AddCommand(templateExtractCmd)
	templateCmd.AddCommand(templateVarsCmd)
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage project templates",
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	RunE:  runTemplateList,
}

var templateAddCmd = &cobra.Command{
	Use:   "add <name> <source-file>",
	Short: "Add a local template override",
	Long: `Copies a file into .promptkit/templates/ as a local override.
Local overrides take precedence over embedded templates during rendering.`,
	Args: cobra.ExactArgs(2),
	RunE: runTemplateAdd,
}

func runTemplateList(_ *cobra.Command, _ []string) error {
	ecosystem := config.EcosystemGolang

	dir, err := resolveConfigDir()
	if err == nil {
		if cfg, loadErr := config.Load(dir); loadErr == nil {
			ecosystem = cfg.Ecosystem
		}
	}

	tmplDir := scaffold.TemplateDirForEcosystem(ecosystem)

	fmt.Printf("Available templates (%s):\n", ecosystem)
	fmt.Println()

	err = fs.WalkDir(promptkit.Templates, tmplDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}

		rel := strings.TrimPrefix(path, tmplDir+"/")
		outName := strings.TrimSuffix(rel, ".tmpl")
		fmt.Printf("  %s\n", outName)

		return nil
	})
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}

	return nil
}

var templateRenderCmd = &cobra.Command{
	Use:   "render <name>",
	Short: "Render a single template to stdout",
	Long: `Renders a single template using the current .promptkit.yaml config
and prints the result to stdout. Useful for debugging templates and overrides.

Example: promptkit template render AGENTS.md
         promptkit template render Makefile`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateRender,
}

func runTemplateRender(_ *cobra.Command, args []string) error {
	name := args[0]

	dir, err := resolveConfigDir()
	if err != nil {
		return err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	rendered, err := scaffold.RenderSingle(cfg, promptkit.Templates, cfg.TemplateOver, name)
	if err != nil {
		return err
	}

	fmt.Print(string(rendered))

	return nil
}

func runTemplateAdd(_ *cobra.Command, args []string) error {
	name := args[0]
	srcFile := args[1]

	ecosystem := config.EcosystemGolang

	dir, err := resolveConfigDir()
	if err == nil {
		if cfg, loadErr := config.Load(dir); loadErr == nil {
			ecosystem = cfg.Ecosystem
		}
	}

	overrideDir := ".promptkit/templates"
	destPath := filepath.Join(overrideDir, name+".tmpl")

	if err = os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		return fmt.Errorf("creating override directory: %w", err)
	}

	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("reading source file: %w", err)
	}

	if err = os.WriteFile(destPath, data, 0o600); err != nil {
		return fmt.Errorf("writing override: %w", err)
	}

	// Save upstream checksum for staleness detection.
	// Read the embedded template to record its current hash.
	tmplDir := scaffold.TemplateDirForEcosystem(ecosystem)
	tmplPath := tmplDir + "/" + name + ".tmpl"

	if embeddedData, readErr := fs.ReadFile(promptkit.Templates, tmplPath); readErr == nil {
		scaffold.SaveOverrideChecksum(overrideDir, name+".tmpl", embeddedData)
	}

	fmt.Printf("Added template override: %s -> %s\n", name, destPath)

	return nil
}

var templateExtractCmd = &cobra.Command{
	Use:   "extract <name>",
	Short: "Extract an embedded template to .promptkit/templates/ for customization",
	Long: `Copies the embedded template source into .promptkit/templates/ so you
can customize it. The extracted file becomes a local override that takes
precedence over the embedded version during rendering.

Example: promptkit template extract AGENTS.md
         promptkit template extract Makefile`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateExtract,
}

func runTemplateExtract(_ *cobra.Command, args []string) error {
	name := args[0]

	ecosystem := config.EcosystemGolang

	dir, err := resolveConfigDir()
	if err == nil {
		if cfg, loadErr := config.Load(dir); loadErr == nil {
			ecosystem = cfg.Ecosystem
		}
	}

	tmplDir := scaffold.TemplateDirForEcosystem(ecosystem)

	// Try .tmpl first, then static.
	tmplPath := tmplDir + "/" + name + ".tmpl"

	data, err := fs.ReadFile(promptkit.Templates, tmplPath)
	if err != nil {
		staticPath := tmplDir + "/" + name

		data, err = fs.ReadFile(promptkit.Templates, staticPath)
		if err != nil {
			return fmt.Errorf("template %q not found in embedded templates", name)
		}
	}

	overrideDir := ".promptkit/templates"
	destPath := filepath.Join(overrideDir, name+".tmpl")

	if !extractFlags.force {
		if _, err = os.Stat(destPath); err == nil {
			return fmt.Errorf("override already exists: %s (use --force to overwrite)", destPath)
		}
	}

	if err = os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		return fmt.Errorf("creating override directory: %w", err)
	}

	if err = os.WriteFile(destPath, data, 0o600); err != nil {
		return fmt.Errorf("writing extracted template: %w", err)
	}

	// Save upstream checksum for staleness detection.
	scaffold.SaveOverrideChecksum(overrideDir, name+".tmpl", data)

	fmt.Printf("Extracted template: %s -> %s\n", name, destPath)
	fmt.Println("Edit the file, then run 'promptkit update' to apply changes.")

	return nil
}

var templateVarsCmd = &cobra.Command{
	Use:   "vars",
	Short: "List available template variables with types",
	Long: `Shows all variables available in templates, accessed as .FieldName.
Useful when creating or editing template overrides.`,
	RunE: runTemplateVars,
}

func runTemplateVars(_ *cobra.Command, _ []string) error {
	fmt.Println("Available template variables (accessed as .FieldName):")
	fmt.Println()
	printStructFields(reflect.TypeFor[config.Config](), ".", "  ")

	return nil
}

func printStructFields(t reflect.Type, prefix, indent string) {
	for i := range t.NumField() {
		field := t.Field(i)

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Strip ",omitempty" and similar options.
		tagName := strings.SplitN(yamlTag, ",", 2)[0]

		accessor := prefix + field.Name
		typeName := field.Type.String()

		// Get description from ExplainField if available.
		_, desc, _ := config.ExplainField(tagName)

		if field.Type.Kind() == reflect.Struct {
			if desc != "" {
				fmt.Printf("%s%-25s %-10s %s\n", indent, accessor, "struct", desc)
			} else {
				fmt.Printf("%s%-25s %s\n", indent, accessor, "struct")
			}

			printStructFields(field.Type, accessor+".", indent+"  ")
		} else {
			if desc != "" {
				fmt.Printf("%s%-25s %-10s %s\n", indent, accessor, typeName, desc)
			} else {
				fmt.Printf("%s%-25s %s\n", indent, accessor, typeName)
			}
		}
	}
}
