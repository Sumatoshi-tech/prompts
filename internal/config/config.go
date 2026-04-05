// Package config handles loading, validating, and saving promptkit configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Configuration constants.
const (
	FileName           = ".promptkit.yaml"
	defaultCoverageMin = 85
	defaultCoverageCrt = 90
	defaultComplexity  = 15
	defaultLineLength  = 140
	defaultYears       = 15
)

// Config holds the project configuration for template rendering.
type Config struct {
	Version        int               `yaml:"version,omitempty"`
	ProjectName    string            `yaml:"project_name"`
	ModulePath     string            `yaml:"module_path"`
	GoVersion      string            `yaml:"go_version,omitempty"`
	Description    string            `yaml:"description"`
	Expertise      string            `yaml:"expertise"`
	IdentityYrs    int               `yaml:"identity_years"`
	Binaries       []Binary          `yaml:"binaries"`
	Quality        Quality           `yaml:"quality"`
	Features       Features          `yaml:"features"`
	Agents         []string          `yaml:"agents"`
	Ecosystem      string            `yaml:"ecosystem"`
	Workflow       string            `yaml:"workflow"`
	Mixtures       []string          `yaml:"mixtures,omitempty"`
	RustEdition    string            `yaml:"rust_edition,omitempty"`
	UnsafePolicy   string            `yaml:"unsafe_policy,omitempty"`
	ZigVersion     string            `yaml:"zig_version,omitempty"`
	LinkLibc       bool              `yaml:"link_libc,omitempty"`
	AnalysisCmd    string            `yaml:"analysis_command"`
	TemplateOver   string            `yaml:"template_overrides"`
	GeneratedFiles []string          `yaml:"generated_files,omitempty"`
	Checksums      map[string]string `yaml:"checksums,omitempty"`
}

// Supported ecosystem identifiers.
const (
	EcosystemGolang = "golang"
	EcosystemRust   = "rust"
	EcosystemZig    = "zig"
)

// Supported workflow identifiers.
const (
	WorkflowFRD     = "frd"
	WorkflowJourney = "journey"
)

// ValidWorkflows is the set of all supported workflow identifiers.
var ValidWorkflows = map[string]bool{
	WorkflowFRD:     true,
	WorkflowJourney: true,
}

// WorkflowDescriptions provides human-readable descriptions for each workflow.
var WorkflowDescriptions = map[string]string{
	WorkflowFRD:     "FRD-based — spec -> /roadmap -> /implement (uses .agents/instructions/instr-frd.md per item) -> /perf",
	WorkflowJourney: "Journey-based — spec -> /roadmap -> /implement (uses .agents/instructions/instr-journey.md per item) -> /perf",
}

// ValidWorkflowNames returns a sorted list of valid workflow names.
func ValidWorkflowNames() []string {
	names := make([]string, 0, len(ValidWorkflows))
	for name := range ValidWorkflows {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// Supported agent identifiers.
const (
	AgentClaude   = "claude"
	AgentCodex    = "codex"
	AgentCopilot  = "copilot"
	AgentCursor   = "cursor"
	AgentGemini   = "gemini"
	AgentWindsurf = "windsurf"
)

// ValidAgents is the set of all supported agent identifiers.
var ValidAgents = map[string]bool{
	AgentClaude:   true,
	AgentCodex:    true,
	AgentCopilot:  true,
	AgentCursor:   true,
	AgentGemini:   true,
	AgentWindsurf: true,
}

// AgentDescriptions provides human-readable descriptions for each agent.
var AgentDescriptions = map[string]string{
	AgentClaude:   "Anthropic Claude Code — reads AGENTS.md, .claude/commands/, .agents/skills/",
	AgentCodex:    "OpenAI Codex CLI — reads AGENTS.md, .agents/skills/",
	AgentCopilot:  "GitHub Copilot — reads .github/copilot-instructions.md, .agents/skills/",
	AgentCursor:   "Cursor IDE — reads .cursor/rules/*.mdc, .agents/skills/",
	AgentGemini:   "Google Gemini CLI — reads GEMINI.md, .gemini/commands/*.toml",
	AgentWindsurf: "Windsurf IDE — reads .windsurfrules, .windsurf/workflows/*.md",
}

// ValidAgentNames returns a sorted list of valid agent names.
func ValidAgentNames() []string {
	names := make([]string, 0, len(ValidAgents))
	for name := range ValidAgents {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// Binary represents a single build target.
type Binary struct {
	Name    string `yaml:"name"`
	CmdPath string `yaml:"cmd_path"`
}

// Quality holds code quality thresholds.
type Quality struct {
	CoverageMin      int `yaml:"coverage_min"`
	CoverageCritical int `yaml:"coverage_critical"`
	ComplexityMax    int `yaml:"complexity_max"`
	LineLength       int `yaml:"line_length"`
}

// Features holds feature flags for conditional template sections.
type Features struct {
	CGO        bool     `yaml:"cgo"`
	Docker     bool     `yaml:"docker"`
	Benchmarks bool     `yaml:"benchmarks"`
	CGOLibs    []CGOLib `yaml:"cgo_libs,omitempty"`
}

// CGOLib describes a CGO dependency.
type CGOLib struct {
	Name      string `yaml:"name"`
	PkgConfig string `yaml:"pkg_config"`
	Include   string `yaml:"include"`
	LibDir    string `yaml:"lib_dir"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	cfg := &Config{
		Version:      CurrentVersion,
		IdentityYrs:  defaultYears,
		Ecosystem:    EcosystemGolang,
		Workflow:     WorkflowFRD,
		TemplateOver: ".promptkit/templates",
		Agents:       []string{AgentClaude},
		Quality: Quality{
			CoverageMin:      defaultCoverageMin,
			CoverageCritical: defaultCoverageCrt,
			ComplexityMax:    defaultComplexity,
			LineLength:       defaultLineLength,
		},
		Features: Features{
			Docker:     true,
			Benchmarks: true,
		},
	}

	if mod := GetEcosystem(cfg.Ecosystem); mod != nil && mod.ApplyDefaults != nil {
		mod.ApplyDefaults(cfg)
	}

	return cfg
}

// fieldValue returns the string value of a config field by its yaml key name.
func (c *Config) fieldValue(yamlKey string) string {
	switch yamlKey {
	case "project_name":
		return c.ProjectName
	case "module_path":
		return c.ModulePath
	case "go_version":
		return c.GoVersion
	case "rust_edition":
		return c.RustEdition
	case "zig_version":
		return c.ZigVersion
	case "description":
		return c.Description
	case "expertise":
		return c.Expertise
	case "ecosystem":
		return c.Ecosystem
	case "workflow":
		return c.Workflow
	case "unsafe_policy":
		return c.UnsafePolicy
	case "analysis_command":
		return c.AnalysisCmd
	default:
		return ""
	}
}

// Validate checks required fields and returns an error if invalid.
func (c *Config) Validate() error {
	var errs []error

	if c.ProjectName == "" {
		errs = append(errs, errors.New("project_name is required (set project_name in .promptkit.yaml)"))
	}

	if mod := GetEcosystem(c.Ecosystem); mod != nil {
		for _, field := range mod.RequiredFields {
			if c.fieldValue(field) == "" {
				errs = append(errs, fmt.Errorf("%s is required for %s ecosystem", field, c.Ecosystem))
			}
		}

		if mod.Validate != nil {
			errs = append(errs, mod.Validate(c)...)
		}
	}

	if len(c.Binaries) == 0 {
		errs = append(errs, errors.New("at least one binary is required"))
	}

	for i, b := range c.Binaries {
		if b.Name == "" {
			errs = append(errs, fmt.Errorf("binaries[%d].name is required", i))
		}

		if b.CmdPath == "" {
			errs = append(errs, fmt.Errorf("binaries[%d].cmd_path is required", i))
		}
	}

	if c.Quality.CoverageMin < 1 || c.Quality.CoverageMin > 100 {
		errs = append(errs, fmt.Errorf(
			"quality.coverage_min must be 1-100, got %d (default: %d)",
			c.Quality.CoverageMin, defaultCoverageMin))
	}

	if c.Quality.CoverageCritical < 1 || c.Quality.CoverageCritical > 100 {
		errs = append(errs, fmt.Errorf(
			"quality.coverage_critical must be 1-100, got %d (default: %d)",
			c.Quality.CoverageCritical, defaultCoverageCrt))
	}

	if c.Quality.CoverageMin > 0 && c.Quality.CoverageCritical > 0 &&
		c.Quality.CoverageCritical < c.Quality.CoverageMin {
		errs = append(errs, fmt.Errorf(
			"quality.coverage_critical (%d) must be >= quality.coverage_min (%d)",
			c.Quality.CoverageCritical, c.Quality.CoverageMin))
	}

	if c.Quality.ComplexityMax <= 0 {
		errs = append(errs, fmt.Errorf(
			"quality.complexity_max must be positive, got %d (default: %d)",
			c.Quality.ComplexityMax, defaultComplexity))
	}

	if c.Quality.LineLength <= 0 {
		errs = append(errs, fmt.Errorf(
			"quality.line_length must be positive, got %d (default: %d)",
			c.Quality.LineLength, defaultLineLength))
	}

	if len(c.Agents) == 0 {
		errs = append(errs, errors.New(
			"agents list is empty; at least one agent is required (e.g. agents: [claude])"))
	}

	for _, agent := range c.Agents {
		if !ValidAgents[agent] {
			suggestion := closestAgent(agent)

			hint := ""
			if suggestion != "" {
				hint = fmt.Sprintf("; did you mean %q?", suggestion)
			}

			errs = append(errs, fmt.Errorf(
				"agents: unknown agent %q (valid: %s)%s",
				agent, strings.Join(ValidAgentNames(), ", "), hint))
		}
	}

	if !ValidEcosystem(c.Ecosystem) {
		suggestion := ClosestEcosystem(c.Ecosystem)

		hint := ""
		if suggestion != "" {
			hint = fmt.Sprintf("; did you mean %q?", suggestion)
		}

		errs = append(errs, fmt.Errorf(
			"ecosystem: unknown ecosystem %q (valid: %s)%s",
			c.Ecosystem, strings.Join(ValidEcosystemNames(), ", "), hint))
	}

	if !ValidWorkflows[c.Workflow] {
		suggestion := closestWorkflow(c.Workflow)

		hint := ""
		if suggestion != "" {
			hint = fmt.Sprintf("; did you mean %q?", suggestion)
		}

		errs = append(errs, fmt.Errorf(
			"workflow: unknown workflow %q (valid: %s)%s",
			c.Workflow, strings.Join(ValidWorkflowNames(), ", "), hint))
	}

	return errors.Join(errs...)
}

// closestAgent returns the valid agent name closest to input by edit distance.
// Returns empty string if no name is within distance 3.
func closestAgent(input string) string {
	best := ""
	bestDist := 4 // threshold: only suggest if distance <= 3

	for _, name := range ValidAgentNames() {
		d := editDistance(input, name)
		if d < bestDist {
			bestDist = d
			best = name
		}
	}

	return best
}

// closestWorkflow returns the valid workflow name closest to input by edit distance.
// Returns empty string if no name is within distance 3.
func closestWorkflow(input string) string {
	best := ""
	bestDist := 4

	for _, name := range ValidWorkflowNames() {
		d := editDistance(input, name)
		if d < bestDist {
			bestDist = d
			best = name
		}
	}

	return best
}

// editDistance computes the Levenshtein distance between two strings.
func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}

	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i

		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost

			curr[j] = min(ins, del, sub)
		}

		prev = curr
	}

	return prev[lb]
}

// FindConfig walks from dir upward looking for .promptkit.yaml.
// Returns the directory containing the config, or an error if not found.
func FindConfig(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		path := filepath.Join(abs, FileName)
		if _, err = os.Stat(path); err == nil {
			return abs, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			// Reached filesystem root.
			return "", fmt.Errorf("%s not found (searched from %s to /)", FileName, dir)
		}

		abs = parent
	}
}

// Load reads a config file from the given directory.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, FileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if bytes.Contains(data, []byte("<<<<<<<")) {
		return nil, errors.New(
			"parsing config: file contains unresolved merge conflict markers — resolve conflicts, then run 'promptkit update'",
		)
	}

	cfg := Default()

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err = decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Run forward migrations.
	changes := Migrate(cfg)
	if len(changes) > 0 {
		if err = Save(cfg, dir); err != nil {
			return nil, fmt.Errorf("saving migrated config: %w", err)
		}
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Save writes the config to the given directory with descriptive comments.
func Save(cfg *Config, dir string) error {
	data := MarshalCommented(cfg)
	path := filepath.Join(dir, FileName)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// MarshalCommented produces YAML with inline comments explaining each field.
func MarshalCommented(cfg *Config) []byte {
	var sb strings.Builder

	sb.WriteString("# promptkit configuration — edit this file, then run: promptkit update\n")
	sb.WriteString("# Tip: After resolving merge conflicts, run 'promptkit update --dry-run' to validate.\n\n")

	sb.WriteString("# Config schema version (do not edit)\n")
	fmt.Fprintf(&sb, "version: %d\n\n", cfg.Version)

	sb.WriteString("# Project name used in AGENTS.md identity and Makefile\n")
	writeYAMLField(&sb, "project_name", cfg.ProjectName)

	sb.WriteString("# Module/crate path (e.g. github.com/org/project)\n")
	writeYAMLField(&sb, "module_path", cfg.ModulePath)

	sb.WriteString("# Project description used in AGENTS.md\n")
	writeYAMLField(&sb, "description", cfg.Description)

	sb.WriteString("# Domain expertise for agent personality (e.g. \"distributed systems\")\n")
	writeYAMLField(&sb, "expertise", cfg.Expertise)

	sb.WriteString("# Years of experience in agent identity statement\n")
	writeYAMLField(&sb, "identity_years", cfg.IdentityYrs)

	sb.WriteString("\n# Build targets — each entry generates a binary in the Makefile\n")
	sb.WriteString("binaries:\n")

	for _, b := range cfg.Binaries {
		fmt.Fprintf(&sb, "  - name: %s\n", b.Name)
		fmt.Fprintf(&sb, "    cmd_path: %s\n", b.CmdPath)
	}

	sb.WriteString("\n# Code quality thresholds — enforced in AGENTS.md, .golangci.yml, and Makefile\n")
	sb.WriteString("quality:\n")
	fmt.Fprintf(&sb, "  coverage_min: %d\n", cfg.Quality.CoverageMin)
	fmt.Fprintf(&sb, "  coverage_critical: %d\n", cfg.Quality.CoverageCritical)
	fmt.Fprintf(&sb, "  complexity_max: %d\n", cfg.Quality.ComplexityMax)
	fmt.Fprintf(&sb, "  line_length: %d\n", cfg.Quality.LineLength)

	sb.WriteString("\n# Feature flags — control conditional sections in Makefile and templates\n")
	sb.WriteString("features:\n")
	fmt.Fprintf(&sb, "  cgo: %t\n", cfg.Features.CGO)
	fmt.Fprintf(&sb, "  docker: %t\n", cfg.Features.Docker)
	fmt.Fprintf(&sb, "  benchmarks: %t\n", cfg.Features.Benchmarks)

	if len(cfg.Features.CGOLibs) > 0 {
		sb.WriteString("  cgo_libs:\n")

		for _, lib := range cfg.Features.CGOLibs {
			fmt.Fprintf(&sb, "    - name: %s\n", lib.Name)
			fmt.Fprintf(&sb, "      pkg_config: %s\n", lib.PkgConfig)
			fmt.Fprintf(&sb, "      include: %s\n", lib.Include)
			fmt.Fprintf(&sb, "      lib_dir: %s\n", lib.LibDir)
		}
	}

	sb.WriteString("\n# Target AI agents (valid: claude, codex, copilot, cursor, gemini, windsurf)\n")
	sb.WriteString("agents:\n")

	for _, a := range cfg.Agents {
		fmt.Fprintf(&sb, "  - %s\n", a)
	}

	fmt.Fprintf(&sb, "\n# Template ecosystem (valid: %s)\n", strings.Join(ValidEcosystemNames(), ", "))
	writeYAMLField(&sb, "ecosystem", cfg.Ecosystem)

	fmt.Fprintf(&sb, "\n# Development workflow (valid: %s)\n", strings.Join(ValidWorkflowNames(), ", "))
	writeYAMLField(&sb, "workflow", cfg.Workflow)

	if len(cfg.Mixtures) > 0 {
		sb.WriteString("\n# Instruction mixtures — cross-cutting concerns injected into targeted skills\n")
		sb.WriteString("# Run 'promptkit mixture list' to see available mixtures\n")
		sb.WriteString("mixtures:\n")

		for _, m := range cfg.Mixtures {
			fmt.Fprintf(&sb, "  - %s\n", m)
		}
	}

	if mod := GetEcosystem(cfg.Ecosystem); mod != nil && mod.CommentedFields != nil {
		for _, f := range mod.CommentedFields(cfg) {
			sb.WriteString(f.Comment + "\n")

			if f.Quote {
				writeYAMLField(&sb, f.Key, quoteVersion(fmt.Sprint(f.Value)))
			} else {
				writeYAMLField(&sb, f.Key, f.Value)
			}
		}
	}

	sb.WriteString("# Analysis command for --verify\n")
	writeYAMLField(&sb, "analysis_command", cfg.AnalysisCmd)

	sb.WriteString("# Directory for local template overrides\n")
	writeYAMLField(&sb, "template_overrides", cfg.TemplateOver)

	if len(cfg.GeneratedFiles) > 0 {
		sb.WriteString("\n# Files generated by promptkit (used for stale file detection)\n")
		sb.WriteString("generated_files:\n")

		sorted := make([]string, len(cfg.GeneratedFiles))
		copy(sorted, cfg.GeneratedFiles)
		sort.Strings(sorted)

		for _, f := range sorted {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
	}

	if len(cfg.Checksums) > 0 {
		sb.WriteString("\n# File checksums (used to detect manual edits)\n")
		sb.WriteString("checksums:\n")

		keys := make([]string, 0, len(cfg.Checksums))
		for k := range cfg.Checksums {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(&sb, "  %s: %s\n", k, cfg.Checksums[k])
		}
	}

	return []byte(sb.String())
}

func writeYAMLField(sb *strings.Builder, key string, value any) {
	fmt.Fprintf(sb, "%s: %v\n", key, value)
}

func quoteVersion(v string) string {
	if v == "" {
		return `""`
	}

	return fmt.Sprintf("%q", v)
}
