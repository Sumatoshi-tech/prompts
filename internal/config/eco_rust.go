package config

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultRustEdition  = "2021"
	defaultUnsafePolicy = "deny"
)

var validUnsafePolicies = map[string]bool{
	"forbid": true,
	"deny":   true,
	"warn":   true,
	"allow":  true,
}

func validUnsafePolicyNames() []string {
	names := make([]string, 0, len(validUnsafePolicies))
	for n := range validUnsafePolicies {
		names = append(names, n)
	}

	sort.Strings(names)

	return names
}

var rustFlags struct {
	edition      string
	unsafePolicy string
}

func init() {
	RegisterEcosystem(&EcosystemModule{
		Name:               EcosystemRust,
		Description:        "Rust — Cargo, clippy, rustfmt, cargo test",
		DefaultAnalysisCmd: "cargo clippy -- -D warnings",

		DefaultCmdPath: func(_ string) string {
			return "src/main.rs"
		},

		ApplyDefaults: func(cfg *Config) {
			cfg.GoVersion = ""

			if cfg.RustEdition == "" {
				cfg.RustEdition = defaultRustEdition
			}

			if cfg.UnsafePolicy == "" {
				cfg.UnsafePolicy = defaultUnsafePolicy
			}

			cfg.AnalysisCmd = "cargo clippy -- -D warnings"
		},

		Validate: func(cfg *Config) []error {
			var errs []error

			if cfg.RustEdition == "" {
				errs = append(errs, errors.New("rust_edition is required for rust ecosystem (e.g. \"2021\")"))
			}

			if cfg.UnsafePolicy != "" && !validUnsafePolicies[cfg.UnsafePolicy] {
				errs = append(errs, fmt.Errorf(
					"unsafe_policy: unknown value %q (valid: %s)",
					cfg.UnsafePolicy, strings.Join(validUnsafePolicyNames(), ", ")))
			}

			return errs
		},

		RegisterFlags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&rustFlags.edition, "rust-edition", "", "Rust edition (e.g. 2021)")
			cmd.Flags().StringVar(&rustFlags.unsafePolicy, "unsafe-policy", "", "Rust unsafe policy (forbid, deny, warn, allow)")
		},

		ApplyFlags: func(cfg *Config) {
			cfg.GoVersion = ""

			if rustFlags.edition != "" {
				cfg.RustEdition = rustFlags.edition
			} else if cfg.RustEdition == "" {
				cfg.RustEdition = defaultRustEdition
			}

			if rustFlags.unsafePolicy != "" {
				cfg.UnsafePolicy = rustFlags.unsafePolicy
			} else if cfg.UnsafePolicy == "" {
				cfg.UnsafePolicy = defaultUnsafePolicy
			}
		},

		RunPrompts: func(cfg *Config, askFn AskFunc, _ AskBoolFunc) error {
			edition, err := askFn("Rust edition (e.g. 2021, 2024)", defaultRustEdition)
			if err != nil {
				return err
			}

			cfg.RustEdition = edition

			policy, err := askFn(
				fmt.Sprintf("Unsafe policy (%s)", strings.Join(validUnsafePolicyNames(), ", ")),
				defaultUnsafePolicy,
			)
			if err != nil {
				return err
			}

			cfg.UnsafePolicy = policy

			return nil
		},

		CommentedFields: func(cfg *Config) []CommentedField {
			return []CommentedField{
				{
					Comment: "# Rust edition (e.g. \"2021\", \"2024\")",
					Key:     "rust_edition",
					Value:   cfg.RustEdition,
					Quote:   true,
				},
				{
					Comment: fmt.Sprintf("# Unsafe policy (valid: %s)", strings.Join(validUnsafePolicyNames(), ", ")),
					Key:     "unsafe_policy",
					Value:   cfg.UnsafePolicy,
				},
			}
		},

		FieldEntries: []FieldEntry{
			{
				Key:         "rust_edition",
				Files:       []string{"AGENTS.md", "rustfmt.toml", ".agents/instructions/instr-implement.md", ".agents/instructions/instr-roadmaper.md"},
				Description: "Rust edition for code generation",
			},
			{
				Key:         "unsafe_policy",
				Files:       []string{"AGENTS.md", "clippy.toml"},
				Description: "Rust unsafe code policy",
			},
		},
	})
}
