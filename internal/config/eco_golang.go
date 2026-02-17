package config

import (
	"errors"

	"github.com/spf13/cobra"
)

const defaultGoVersion = "1.22"

var golangFlags struct {
	goVersion string
}

func init() {
	RegisterEcosystem(&EcosystemModule{
		Name:               EcosystemGolang,
		Description:        "Go — modules, go vet, golangci-lint, go test",
		DefaultAnalysisCmd: "go vet ./...",

		DefaultCmdPath: func(binaryName string) string {
			return "./cmd/" + binaryName
		},

		ApplyDefaults: func(cfg *Config) {
			if cfg.GoVersion == "" {
				cfg.GoVersion = defaultGoVersion
			}

			cfg.AnalysisCmd = "go vet ./..."
		},

		Validate: func(cfg *Config) []error {
			if cfg.GoVersion == "" {
				return []error{errors.New("go_version is required for golang ecosystem (e.g. \"1.22\")")}
			}

			return nil
		},

		RegisterFlags: func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&golangFlags.goVersion, "go-version", "", "Go version")
		},

		ApplyFlags: func(cfg *Config) {
			if golangFlags.goVersion != "" {
				cfg.GoVersion = golangFlags.goVersion
			}
		},

		RunPrompts: func(cfg *Config, askFn AskFunc, askBoolFn AskBoolFunc) error {
			goVer, err := askFn("Go version", defaultGoVersion)
			if err != nil {
				return err
			}

			cfg.GoVersion = goVer

			cgo, err := askBoolFn("Enable CGO support?", false)
			if err != nil {
				return err
			}

			cfg.Features.CGO = cgo

			return nil
		},

		CommentedFields: func(cfg *Config) []CommentedField {
			return []CommentedField{
				{
					Comment: "# Go version for linter config and build targets",
					Key:     "go_version",
					Value:   cfg.GoVersion,
					Quote:   true,
				},
			}
		},

		FieldEntries: []FieldEntry{
			{
				Key:         "go_version",
				Files:       []string{"AGENTS.md", "instructions/instr-implement.md", "instructions/instr-roadmaper.md"},
				Description: "Go version for linter config and build targets",
			},
		},
	})
}
