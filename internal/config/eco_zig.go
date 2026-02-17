package config

import (
	"github.com/spf13/cobra"
)

const defaultZigVersion = "0.13"

var zigFlags struct {
	zigVersion string
	linkLibc   bool
}

func init() {
	RegisterEcosystem(&EcosystemModule{
		Name:               EcosystemZig,
		Description:        "Zig — build.zig, zig test, zig fmt",
		DefaultAnalysisCmd: "zig build test",

		DefaultCmdPath: func(_ string) string {
			return "src/main.zig"
		},

		ApplyDefaults: func(cfg *Config) {
			cfg.GoVersion = ""

			if cfg.ZigVersion == "" {
				cfg.ZigVersion = defaultZigVersion
			}

			cfg.AnalysisCmd = "zig build test"
		},

		Validate: func(_ *Config) []error {
			return nil
		},

		RegisterFlags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&zigFlags.zigVersion, "zig-version", "", "Zig version (e.g. 0.13)")
			cmd.Flags().BoolVar(&zigFlags.linkLibc, "link-libc", false, "Zig: link libc for C interop")
		},

		ApplyFlags: func(cfg *Config) {
			cfg.GoVersion = ""

			if zigFlags.zigVersion != "" {
				cfg.ZigVersion = zigFlags.zigVersion
			} else if cfg.ZigVersion == "" {
				cfg.ZigVersion = defaultZigVersion
			}

			cfg.LinkLibc = zigFlags.linkLibc
		},

		RunPrompts: func(cfg *Config, askFn AskFunc, askBoolFn AskBoolFunc) error {
			ver, err := askFn("Zig version (e.g. 0.13, 0.14)", defaultZigVersion)
			if err != nil {
				return err
			}

			cfg.ZigVersion = ver

			linkLibc, err := askBoolFn("Link libc for C interop?", false)
			if err != nil {
				return err
			}

			cfg.LinkLibc = linkLibc

			return nil
		},

		CommentedFields: func(cfg *Config) []CommentedField {
			return []CommentedField{
				{
					Comment: "# Zig version (e.g. \"0.13\")",
					Key:     "zig_version",
					Value:   cfg.ZigVersion,
					Quote:   true,
				},
				{
					Comment: "# Link libc for C interop",
					Key:     "link_libc",
					Value:   cfg.LinkLibc,
				},
			}
		},

		FieldEntries: []FieldEntry{
			{
				Key:         "zig_version",
				Files:       []string{"AGENTS.md", "instructions/instr-implement.md", "instructions/instr-roadmaper.md"},
				Description: "Zig version for build configuration",
			},
			{
				Key:         "link_libc",
				Files:       []string{"AGENTS.md", "Makefile", "instructions/instr-perf.md"},
				Description: "Link libc for C interop",
			},
		},
	})
}
