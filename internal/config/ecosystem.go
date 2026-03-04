package config

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// AskFunc is the signature for interactive string prompts.
type AskFunc func(prompt, defaultVal string) (string, error)

// AskBoolFunc is the signature for interactive boolean prompts.
type AskBoolFunc func(prompt string, defaultVal bool) (bool, error)

// CommentedField represents a single field to include in MarshalCommented output.
type CommentedField struct {
	Comment string // e.g. "# Rust edition (e.g. \"2021\", \"2024\")".
	Key     string // YAML key, e.g. "rust_edition".
	Value   any    // value to render.
	Quote   bool   // whether to quote the value (for version strings).
}

// FieldEntry describes a config field for the explain/fieldmap system.
type FieldEntry struct {
	Key         string   // YAML key, e.g. "rust_edition".
	Files       []string // output files this field affects.
	Description string   // human-readable description.
}

// EcosystemModule encapsulates all ecosystem-specific logic.
type EcosystemModule struct {
	// Name is the ecosystem identifier (e.g. "golang", "rust", "zig").
	Name string

	// Description is a human-readable description for selection prompts.
	Description string

	// DefaultAnalysisCmd is the default analysis command for this ecosystem.
	DefaultAnalysisCmd string

	// DefaultCmdPath returns the default cmd_path for a binary name.
	DefaultCmdPath func(binaryName string) string

	// ApplyDefaults sets ecosystem-specific default values on a Config.
	ApplyDefaults func(cfg *Config)

	// Validate checks ecosystem-specific fields and returns errors.
	Validate func(cfg *Config) []error

	// RegisterFlags registers ecosystem-specific CLI flags on a cobra command.
	RegisterFlags func(cmd *cobra.Command)

	// ApplyFlags reads ecosystem-specific CLI flags and applies them to the config.
	ApplyFlags func(cfg *Config)

	// RunPrompts runs interactive prompts for ecosystem-specific fields.
	RunPrompts func(cfg *Config, askFn AskFunc, askBoolFn AskBoolFunc) error

	// CommentedFields returns ecosystem-specific fields for commented YAML output.
	CommentedFields func(cfg *Config) []CommentedField

	// FieldEntries is static metadata mapping config keys to affected files.
	FieldEntries []FieldEntry
}

// ecoRegistry holds all registered ecosystem modules.
var ecoRegistry = map[string]*EcosystemModule{}

// RegisterEcosystem adds an ecosystem module to the registry.
func RegisterEcosystem(m *EcosystemModule) {
	ecoRegistry[m.Name] = m
}

// GetEcosystem returns the module for the given ecosystem name, or nil if not found.
func GetEcosystem(name string) *EcosystemModule {
	return ecoRegistry[name]
}

// ValidEcosystemNames returns a sorted list of registered ecosystem names.
func ValidEcosystemNames() []string {
	names := make([]string, 0, len(ecoRegistry))
	for name := range ecoRegistry {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// ValidEcosystem returns true if the ecosystem name is registered.
func ValidEcosystem(name string) bool {
	_, ok := ecoRegistry[name]
	return ok
}

// EcosystemDescriptions returns a map of ecosystem name to human-readable description.
func EcosystemDescriptions() map[string]string {
	descs := make(map[string]string, len(ecoRegistry))
	for name, m := range ecoRegistry {
		descs[name] = m.Description
	}

	return descs
}

// AllEcosystems returns all registered modules sorted by name.
func AllEcosystems() []*EcosystemModule {
	names := ValidEcosystemNames()
	mods := make([]*EcosystemModule, 0, len(names))

	for _, name := range names {
		mods = append(mods, ecoRegistry[name])
	}

	return mods
}

// ClosestEcosystem returns the registered ecosystem name closest to input by edit distance.
// Uses case-insensitive comparison. Returns empty string if no match within distance 3.
func ClosestEcosystem(input string) string {
	if input == "" {
		return ""
	}

	low := strings.ToLower(input)
	best := ""
	bestDist := 4

	for _, name := range ValidEcosystemNames() {
		d := editDistance(low, name)
		if d < bestDist {
			bestDist = d
			best = name
		}
	}

	return best
}
