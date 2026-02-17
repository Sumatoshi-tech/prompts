package config

import "sort"

// FieldFiles maps config YAML keys to the output files they affect.
// Ecosystem-specific fields are registered via EcosystemModule.FieldEntries.
var FieldFiles = map[string][]string{
	"project_name":              {"AGENTS.md", "Makefile"},
	"module_path":               {"Makefile"},
	"description":               {"AGENTS.md", "instructions/instr-implement.md"},
	"expertise":                 {"AGENTS.md"},
	"identity_years":            {"AGENTS.md", "instructions/instr-implement.md", "instructions/instr-roadmaper.md"},
	"analysis_command":          {"AGENTS.md", "instructions/instr-implement.md"},
	"binaries":                  {"Makefile"},
	"quality.coverage_min":      {"AGENTS.md"},
	"quality.coverage_critical": {"AGENTS.md", "instructions/instr-implement.md"},
	"quality.complexity_max":    {"AGENTS.md"},
	"quality.line_length":       {".golangci.yml"},
	"features.cgo":              {"AGENTS.md", "Makefile", "instructions/instr-perf.md", "scripts/deadcode-filter.sh"},
	"features.docker":           {"AGENTS.md", "Makefile"},
	"features.benchmarks":       {"AGENTS.md", "Makefile"},
	"features.cgo_libs":         {"Makefile", "scripts/deadcode-filter.sh"},
	"agents":                    {}, // affects agent-specific placement, not template content
	"workflow":                  {"instructions/instr-implement.md", "instructions/instr-roadmaper.md"},
}

// AllFieldFiles returns FieldFiles merged with ecosystem module field entries.
func AllFieldFiles() map[string][]string {
	result := make(map[string][]string, len(FieldFiles))
	for k, v := range FieldFiles {
		result[k] = v
	}

	for _, mod := range AllEcosystems() {
		for _, fe := range mod.FieldEntries {
			result[fe.Key] = fe.Files
		}
	}

	return result
}

// ReverseFieldMap returns a map from output file to the config keys that affect it.
func ReverseFieldMap() map[string][]string {
	rev := make(map[string][]string)

	for key, files := range AllFieldFiles() {
		for _, f := range files {
			rev[f] = append(rev[f], key)
		}
	}

	for f := range rev {
		sort.Strings(rev[f])
	}

	return rev
}

// base descriptions for common (non-ecosystem) config fields.
var baseDescriptions = map[string]string{
	"project_name":              "Project name used in AGENTS.md identity and Makefile",
	"module_path":               "Module path for build targets",
	"description":               "Project description used in AGENTS.md",
	"expertise":                 "Domain expertise for agent personality",
	"identity_years":            "Years of experience in agent identity statement",
	"analysis_command":          "Analysis command run with --verify",
	"binaries":                  "Build targets generating Makefile entries",
	"quality.coverage_min":      "Minimum test coverage threshold",
	"quality.coverage_critical": "Critical test coverage threshold for new code",
	"quality.complexity_max":    "Maximum cyclomatic complexity allowed",
	"quality.line_length":       "Maximum line length for linter",
	"features.cgo":              "CGO support flag affecting build and instructions",
	"features.docker":           "Docker support flag affecting Makefile targets",
	"features.benchmarks":       "Benchmarks flag affecting Makefile targets",
	"features.cgo_libs":         "CGO library dependencies",
	"agents":                    "Target AI agents for file placement",
	"workflow":                  "Development workflow (frd or journey)",
}

// ExplainField returns the output files affected by a config key,
// a human-readable description, and whether the key was found.
func ExplainField(key string) (files []string, desc string, ok bool) {
	allFiles := AllFieldFiles()

	files, ok = allFiles[key]
	if !ok {
		return nil, "", false
	}

	// Check base descriptions first.
	if d, found := baseDescriptions[key]; found {
		return files, d, true
	}

	// Check ecosystem module field entries.
	for _, mod := range AllEcosystems() {
		for _, fe := range mod.FieldEntries {
			if fe.Key == key {
				return files, fe.Description, true
			}
		}
	}

	return files, "", true
}
