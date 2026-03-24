package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Sumatoshi-tech/prompts/internal/config"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	cfg := config.Default()

	if cfg.GoVersion == "" {
		t.Error("default GoVersion should not be empty")
	}

	if cfg.Quality.CoverageMin != 85 {
		t.Errorf("default CoverageMin = %d, want 85", cfg.Quality.CoverageMin)
	}

	if cfg.Quality.ComplexityMax != 15 {
		t.Errorf("default ComplexityMax = %d, want 15", cfg.Quality.ComplexityMax)
	}

	if cfg.Ecosystem != "golang" {
		t.Errorf("default Ecosystem = %q, want %q", cfg.Ecosystem, "golang")
	}
}

func TestValidate_Valid(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.ProjectName = "testproject"
	cfg.ModulePath = "github.com/user/testproject"
	cfg.Binaries = []config.Binary{
		{Name: "testproject", CmdPath: "./cmd/testproject"},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidate_MissingFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(*config.Config)
		wantErr string
	}{
		{
			name:    "missing project name",
			modify:  func(c *config.Config) { c.ProjectName = "" },
			wantErr: "project_name is required",
		},
		{
			name:    "missing module path",
			modify:  func(c *config.Config) { c.ModulePath = "" },
			wantErr: "module_path is required",
		},
		{
			name:    "missing go version",
			modify:  func(c *config.Config) { c.GoVersion = "" },
			wantErr: "go_version is required for golang ecosystem",
		},
		{
			name:    "no binaries",
			modify:  func(c *config.Config) { c.Binaries = nil },
			wantErr: "at least one binary is required",
		},
		{
			name: "binary missing name",
			modify: func(c *config.Config) {
				c.Binaries = []config.Binary{{Name: "", CmdPath: "./cmd/x"}}
			},
			wantErr: "binaries[0].name is required",
		},
		{
			name: "binary missing cmd_path",
			modify: func(c *config.Config) {
				c.Binaries = []config.Binary{{Name: "x", CmdPath: ""}}
			},
			wantErr: "binaries[0].cmd_path is required",
		},
		{
			name:    "invalid coverage",
			modify:  func(c *config.Config) { c.Quality.CoverageMin = 0 },
			wantErr: "quality.coverage_min must be 1-100",
		},
		{
			name:    "invalid complexity",
			modify:  func(c *config.Config) { c.Quality.ComplexityMax = 0 },
			wantErr: "quality.complexity_max must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	original := validConfig()
	original.Description = "A test project"
	original.Expertise = "testing"
	original.Features.Docker = true
	original.Features.CGO = false

	if err := config.Save(original, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file was created.
	path := filepath.Join(dir, config.FileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.ProjectName != original.ProjectName {
		t.Errorf("ProjectName = %q, want %q", loaded.ProjectName, original.ProjectName)
	}

	if loaded.ModulePath != original.ModulePath {
		t.Errorf("ModulePath = %q, want %q", loaded.ModulePath, original.ModulePath)
	}

	if loaded.Description != original.Description {
		t.Errorf("Description = %q, want %q", loaded.Description, original.Description)
	}

	if loaded.Features.Docker != original.Features.Docker {
		t.Errorf("Features.Docker = %v, want %v", loaded.Features.Docker, original.Features.Docker)
	}
}

func TestLoad_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)

	if err := os.WriteFile(path, []byte(":::invalid:::"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestValidate_InvalidAgentName(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Agents = []string{"claude", "invalid-agent"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid agent name")
	}

	got := err.Error()
	if !contains(got, `unknown agent "invalid-agent"`) {
		t.Errorf("error = %q, want to contain unknown agent message", got)
	}

	if !contains(got, "valid:") {
		t.Errorf("error = %q, want to contain valid agents list", got)
	}
}

func TestValidate_AllValidAgents(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Agents = config.ValidAgentNames()

	if err := cfg.Validate(); err != nil {
		t.Errorf("all valid agents should pass validation: %v", err)
	}
}

func TestValidate_CoverageCritical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		critical int
		wantErr  string
	}{
		{"zero", 0, "quality.coverage_critical must be 1-100"},
		{"over 100", 101, "quality.coverage_critical must be 1-100"},
		{"valid", 90, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			cfg.Quality.CoverageCritical = tt.critical

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				return
			}

			if err == nil {
				t.Fatal("expected validation error")
			}

			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidate_CoverageCriticalLessThanMin(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Quality.CoverageMin = 90
	cfg.Quality.CoverageCritical = 80

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when critical < min")
	}

	if !contains(err.Error(), "coverage_critical (80) must be >= quality.coverage_min (90)") {
		t.Errorf("error = %q, want constraint message", err.Error())
	}
}

func TestValidate_LineLength(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Quality.LineLength = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for zero line length")
	}

	if !contains(err.Error(), "quality.line_length must be positive") {
		t.Errorf("error = %q, want line_length message", err.Error())
	}
}

func TestValidate_AggregatesMultipleErrors(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.ProjectName = ""
	cfg.ModulePath = ""
	cfg.Quality.CoverageMin = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	got := err.Error()
	if !contains(got, "project_name is required") {
		t.Error("missing project_name error")
	}

	if !contains(got, "module_path is required") {
		t.Error("missing module_path error")
	}

	if !contains(got, "quality.coverage_min must be 1-100") {
		t.Error("missing coverage_min error")
	}
}

func TestValidate_ErrorSuggestions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		modify     func(*config.Config)
		wantSuffix string
	}{
		{
			name:       "project_name suggestion",
			modify:     func(c *config.Config) { c.ProjectName = "" },
			wantSuffix: "set project_name in .promptkit.yaml",
		},
		{
			name:       "module_path suggestion",
			modify:     func(c *config.Config) { c.ModulePath = "" },
			wantSuffix: "module_path is required for golang ecosystem",
		},
		{
			name:       "coverage_min default",
			modify:     func(c *config.Config) { c.Quality.CoverageMin = 150 },
			wantSuffix: "default: 85",
		},
		{
			name:       "line_length default",
			modify:     func(c *config.Config) { c.Quality.LineLength = 0 },
			wantSuffix: "default: 140",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}

			if !contains(err.Error(), tt.wantSuffix) {
				t.Errorf("error = %q, want to contain suggestion %q", err.Error(), tt.wantSuffix)
			}
		})
	}
}

func TestValidAgentNames(t *testing.T) {
	t.Parallel()

	names := config.ValidAgentNames()
	if len(names) != 6 {
		t.Errorf("expected 6 valid agents, got %d", len(names))
	}

	// Should be sorted.
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("ValidAgentNames not sorted: %v", names)
			break
		}
	}
}

func TestMarshalCommented_ContainsComments(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	data := config.MarshalCommented(cfg)
	s := string(data)

	expectedComments := []string{
		"# promptkit configuration",
		"# Project name used in AGENTS.md",
		"# Module/crate path",
		"# Code quality thresholds",
		"# Feature flags",
		"# Target AI agents",
	}

	for _, comment := range expectedComments {
		if !contains(s, comment) {
			t.Errorf("MarshalCommented output missing comment: %q", comment)
		}
	}
}

func TestMarshalCommented_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	original := validConfig()
	original.Description = "Round trip test"
	original.GeneratedFiles = []string{"AGENTS.md", "Makefile"}

	if err := config.Save(original, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.ProjectName != original.ProjectName {
		t.Errorf("ProjectName = %q, want %q", loaded.ProjectName, original.ProjectName)
	}

	if loaded.Description != original.Description {
		t.Errorf("Description = %q, want %q", loaded.Description, original.Description)
	}

	if len(loaded.GeneratedFiles) != len(original.GeneratedFiles) {
		t.Errorf("GeneratedFiles len = %d, want %d", len(loaded.GeneratedFiles), len(original.GeneratedFiles))
	}
}

func TestLoad_MergeConflictMarkers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)

	content := `project_name: myapp
<<<<<<< HEAD
quality:
  coverage_min: 85
=======
quality:
  coverage_min: 90
>>>>>>> feature-branch
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for merge conflict markers")
	}

	if !contains(err.Error(), "merge conflict") {
		t.Errorf("error = %q, want to contain 'merge conflict'", err.Error())
	}
}

func TestDefault_AnalysisCmd(t *testing.T) {
	t.Parallel()

	cfg := config.Default()

	if cfg.AnalysisCmd != "go vet ./..." {
		t.Errorf("default AnalysisCmd = %q, want %q", cfg.AnalysisCmd, "go vet ./...")
	}
}

func TestMarshalCommented_MergeConflictTip(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	data := config.MarshalCommented(cfg)
	s := string(data)

	if !contains(s, "merge conflicts") {
		t.Error("MarshalCommented should contain merge conflict tip")
	}
}

func TestDefault_Version(t *testing.T) {
	t.Parallel()

	cfg := config.Default()

	if cfg.Version != config.CurrentVersion {
		t.Errorf("default Version = %d, want %d", cfg.Version, config.CurrentVersion)
	}
}

func TestFindConfig_CurrentDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)

	if err := os.WriteFile(path, []byte("project_name: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, err := config.FindConfig(dir)
	if err != nil {
		t.Fatalf("FindConfig() error: %v", err)
	}

	if found != dir {
		t.Errorf("FindConfig() = %q, want %q", found, dir)
	}
}

func TestFindConfig_ParentDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "deep")

	if err := os.MkdirAll(child, 0o750); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(parent, config.FileName)
	if err := os.WriteFile(path, []byte("project_name: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, err := config.FindConfig(child)
	if err != nil {
		t.Fatalf("FindConfig() error: %v", err)
	}

	if found != parent {
		t.Errorf("FindConfig() = %q, want %q", found, parent)
	}
}

func TestFindConfig_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := config.FindConfig(dir)
	if err == nil {
		t.Fatal("expected error when config not found")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestMarshalCommented_Version(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	data := config.MarshalCommented(cfg)
	s := string(data)

	if !contains(s, "version: 2") {
		t.Error("MarshalCommented should contain version: 2")
	}

	if !contains(s, "do not edit") {
		t.Error("MarshalCommented should contain 'do not edit' note for version")
	}
}

func TestFieldFiles_AllKeysHaveDescriptions(t *testing.T) {
	t.Parallel()

	for key := range config.AllFieldFiles() {
		_, desc, ok := config.ExplainField(key)
		if !ok {
			t.Errorf("ExplainField(%q) returned not found", key)
		}

		if desc == "" {
			t.Errorf("ExplainField(%q) has empty description", key)
		}
	}
}

func TestReverseFieldMap(t *testing.T) {
	t.Parallel()

	rev := config.ReverseFieldMap()

	// AGENTS.md should have multiple keys.
	keys, ok := rev["AGENTS.md"]
	if !ok {
		t.Fatal("ReverseFieldMap missing AGENTS.md")
	}

	if len(keys) < 3 {
		t.Errorf("AGENTS.md should have at least 3 config keys, got %d", len(keys))
	}

	// Keys should be sorted.
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Errorf("ReverseFieldMap keys not sorted for AGENTS.md: %v", keys)
			break
		}
	}
}

func TestMigrate_V0ToV1(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Version:     0,
		AnalysisCmd: "",
	}

	changes := config.Migrate(cfg)
	if len(changes) == 0 {
		t.Fatal("expected migration changes for v0 config")
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("after migration, Version = %d, want %d", cfg.Version, config.CurrentVersion)
	}

	if cfg.AnalysisCmd != "go vet ./..." {
		t.Errorf("after migration, AnalysisCmd = %q, want %q", cfg.AnalysisCmd, "go vet ./...")
	}
}

func TestMigrate_AlreadyCurrent(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	changes := config.Migrate(cfg)

	if len(changes) != 0 {
		t.Errorf("expected no changes for current version, got %v", changes)
	}
}

func TestValidate_EmptyAgentsList(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Agents = []string{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty agents list")
	}

	if !contains(err.Error(), "agents list is empty") {
		t.Errorf("error = %q, want to contain 'agents list is empty'", err.Error())
	}

	if !contains(err.Error(), "at least one agent is required") {
		t.Errorf("error = %q, want to contain suggestion", err.Error())
	}
}

func TestValidate_DidYouMean(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Agents = []string{"cluade"} // typo for "claude".

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for misspelled agent")
	}

	if !contains(err.Error(), `did you mean "claude"`) {
		t.Errorf("error = %q, want did-you-mean suggestion for claude", err.Error())
	}
}

func TestValidate_DidYouMean_NoSuggestion(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Agents = []string{"zzzzzzz"} // too distant from any agent.

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	// Should NOT contain "did you mean" for a completely unrelated string.
	if contains(err.Error(), "did you mean") {
		t.Errorf("error = %q, should not suggest for very distant input", err.Error())
	}
}

func TestLoad_UnknownFieldError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)

	content := `project_name: myapp
module_path: github.com/test/myapp
agnets:
  - claude
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for unknown YAML field 'agnets'")
	}

	if !contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want to contain 'parsing config'", err.Error())
	}
}

func TestValidEcosystemNames(t *testing.T) {
	t.Parallel()

	names := config.ValidEcosystemNames()
	if len(names) != 3 {
		t.Errorf("expected 3 valid ecosystems, got %d", len(names))
	}

	// Should be sorted.
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("ValidEcosystemNames not sorted: %v", names)
			break
		}
	}

	// Should contain the expected values.
	expected := []string{"golang", "rust", "zig"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("ValidEcosystemNames()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestValidate_ValidEcosystems(t *testing.T) {
	t.Parallel()

	configs := map[string]*config.Config{
		"golang": validConfig(),
		"rust":   validRustConfig(),
		"zig":    validZigConfig(),
	}

	for _, eco := range config.ValidEcosystemNames() {
		t.Run(eco, func(t *testing.T) {
			t.Parallel()

			cfg := configs[eco]
			if err := cfg.Validate(); err != nil {
				t.Errorf("ecosystem %q should be valid: %v", eco, err)
			}
		})
	}
}

func TestValidate_InvalidEcosystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ecosystem string
		wantErr   string
		wantHint  string
		noHint    bool
	}{
		{
			name:      "unknown ecosystem python",
			ecosystem: "python",
			wantErr:   `unknown ecosystem "python"`,
			noHint:    true,
		},
		{
			name:      "typo rustt suggests rust",
			ecosystem: "rustt",
			wantErr:   `unknown ecosystem "rustt"`,
			wantHint:  `did you mean "rust"`,
		},
		{
			name:      "typo goland suggests golang",
			ecosystem: "goland",
			wantErr:   `unknown ecosystem "goland"`,
			wantHint:  `did you mean "golang"`,
		},
		{
			name:      "typo zi suggests zig",
			ecosystem: "zi",
			wantErr:   `unknown ecosystem "zi"`,
			wantHint:  `did you mean "zig"`,
		},
		{
			name:      "empty ecosystem",
			ecosystem: "",
			wantErr:   `unknown ecosystem ""`,
			noHint:    true,
		},
		{
			name:      "case sensitive GOLANG",
			ecosystem: "GOLANG",
			wantErr:   `unknown ecosystem "GOLANG"`,
			wantHint:  `did you mean "golang"`,
		},
		{
			name:      "distant string no suggestion",
			ecosystem: "zzzzzzz",
			wantErr:   `unknown ecosystem "zzzzzzz"`,
			noHint:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			cfg.Ecosystem = tt.ecosystem

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}

			got := err.Error()
			if !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", got, tt.wantErr)
			}

			if !contains(got, "valid: golang, rust, zig") {
				t.Errorf("error = %q, want to contain valid ecosystems list", got)
			}

			if tt.wantHint != "" {
				if !contains(got, tt.wantHint) {
					t.Errorf("error = %q, want to contain hint %q", got, tt.wantHint)
				}
			}

			if tt.noHint {
				if contains(got, "did you mean") {
					t.Errorf("error = %q, should not contain did-you-mean for distant input", got)
				}
			}
		})
	}
}

func TestEcosystemDescriptions_AllEcosystemsCovered(t *testing.T) {
	t.Parallel()

	for _, name := range config.ValidEcosystemNames() {
		desc, ok := config.EcosystemDescriptions()[name]
		if !ok {
			t.Errorf("missing description for ecosystem %q", name)
			continue
		}

		if desc == "" {
			t.Errorf("empty description for ecosystem %q", name)
		}
	}
}

func TestEcosystemConstants(t *testing.T) {
	t.Parallel()

	if config.EcosystemGolang != "golang" {
		t.Errorf("EcosystemGolang = %q, want %q", config.EcosystemGolang, "golang")
	}

	if config.EcosystemRust != "rust" {
		t.Errorf("EcosystemRust = %q, want %q", config.EcosystemRust, "rust")
	}

	if config.EcosystemZig != "zig" {
		t.Errorf("EcosystemZig = %q, want %q", config.EcosystemZig, "zig")
	}
}

func TestValidate_RustEcosystem(t *testing.T) {
	t.Parallel()

	t.Run("valid rust config", func(t *testing.T) {
		t.Parallel()

		cfg := validRustConfig()
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})

	t.Run("missing rust_edition", func(t *testing.T) {
		t.Parallel()

		cfg := validRustConfig()
		cfg.RustEdition = ""

		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}

		if !contains(err.Error(), "rust_edition is required") {
			t.Errorf("error = %q, want to contain rust_edition message", err.Error())
		}
	})

	t.Run("go_version not required for rust", func(t *testing.T) {
		t.Parallel()

		cfg := validRustConfig()
		cfg.GoVersion = ""

		if err := cfg.Validate(); err != nil {
			t.Errorf("go_version should not be required for rust: %v", err)
		}
	})

	t.Run("invalid unsafe_policy", func(t *testing.T) {
		t.Parallel()

		cfg := validRustConfig()
		cfg.UnsafePolicy = "yolo"

		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}

		if !contains(err.Error(), `unknown value "yolo"`) {
			t.Errorf("error = %q, want to contain unknown value message", err.Error())
		}

		if !contains(err.Error(), "valid: allow, deny, forbid, warn") {
			t.Errorf("error = %q, want to contain valid policies list", err.Error())
		}
	})

	t.Run("valid unsafe policies", func(t *testing.T) {
		t.Parallel()

		for _, policy := range []string{"allow", "deny", "forbid", "warn"} {
			cfg := validRustConfig()
			cfg.UnsafePolicy = policy

			if err := cfg.Validate(); err != nil {
				t.Errorf("policy %q should be valid: %v", policy, err)
			}
		}
	})

	t.Run("empty unsafe_policy allowed", func(t *testing.T) {
		t.Parallel()

		cfg := validRustConfig()
		cfg.UnsafePolicy = ""

		if err := cfg.Validate(); err != nil {
			t.Errorf("empty unsafe_policy should be valid: %v", err)
		}
	})
}

func TestValidate_ZigEcosystem(t *testing.T) {
	t.Parallel()

	t.Run("valid zig config", func(t *testing.T) {
		t.Parallel()

		cfg := validZigConfig()
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})

	t.Run("go_version not required for zig", func(t *testing.T) {
		t.Parallel()

		cfg := validZigConfig()
		cfg.GoVersion = ""

		if err := cfg.Validate(); err != nil {
			t.Errorf("go_version should not be required for zig: %v", err)
		}
	})

	t.Run("link_libc true", func(t *testing.T) {
		t.Parallel()

		cfg := validZigConfig()
		cfg.LinkLibc = true

		if err := cfg.Validate(); err != nil {
			t.Errorf("link_libc true should be valid: %v", err)
		}
	})
}

func TestWorkflowConstants(t *testing.T) {
	t.Parallel()

	if config.WorkflowFRD != "frd" {
		t.Errorf("WorkflowFRD = %q, want %q", config.WorkflowFRD, "frd")
	}

	if config.WorkflowJourney != "journey" {
		t.Errorf("WorkflowJourney = %q, want %q", config.WorkflowJourney, "journey")
	}
}

func TestValidWorkflowNames(t *testing.T) {
	t.Parallel()

	names := config.ValidWorkflowNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(names))
	}

	if names[0] != "frd" || names[1] != "journey" {
		t.Errorf("expected [frd, journey], got %v", names)
	}
}

func TestValidate_InvalidWorkflow(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Workflow = "waterfall"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid workflow")
	}

	if !contains(err.Error(), `unknown workflow "waterfall"`) {
		t.Errorf("error = %q, want to contain unknown workflow message", err.Error())
	}

	if !contains(err.Error(), "valid: frd, journey") {
		t.Errorf("error = %q, want to contain valid workflow list", err.Error())
	}
}

func TestValidate_ValidWorkflows(t *testing.T) {
	t.Parallel()

	for _, wf := range []string{"frd", "journey"} {
		cfg := validConfig()
		cfg.Workflow = wf

		if err := cfg.Validate(); err != nil {
			t.Errorf("workflow %q should be valid: %v", wf, err)
		}
	}
}

func TestValidate_DefaultWorkflowIsFRD(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	if cfg.Workflow != config.WorkflowFRD {
		t.Errorf("Default().Workflow = %q, want %q", cfg.Workflow, config.WorkflowFRD)
	}
}

func TestMarshalCommented_Workflow(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Workflow = config.WorkflowJourney
	data := config.MarshalCommented(cfg)
	s := string(data)

	if !contains(s, "workflow: journey") {
		t.Error("MarshalCommented should contain workflow: journey")
	}

	if !contains(s, "Development workflow") {
		t.Error("MarshalCommented should contain workflow comment")
	}
}

func TestAgentDescriptions_AllAgentsCovered(t *testing.T) {
	t.Parallel()

	for _, name := range config.ValidAgentNames() {
		desc, ok := config.AgentDescriptions[name]
		if !ok {
			t.Errorf("missing description for agent %q", name)
			continue
		}

		if desc == "" {
			t.Errorf("empty description for agent %q", name)
		}
	}
}

func TestMarshalCommented_GolangConfig(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.GoVersion = "1.22"
	data := config.MarshalCommented(cfg)
	s := string(data)

	// Verify golang-specific commented field appears.
	if !contains(s, "go_version") {
		t.Error("MarshalCommented for golang should contain go_version")
	}

	if !contains(s, `"1.22"`) {
		t.Error("MarshalCommented for golang should quote go_version value")
	}

	// Verify the output can be decoded back as valid YAML.
	dir := t.TempDir()
	if err := config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() round-trip error: %v", err)
	}

	if loaded.GoVersion != "1.22" {
		t.Errorf("round-trip GoVersion = %q, want %q", loaded.GoVersion, "1.22")
	}
}

func TestMarshalCommented_RustConfig(t *testing.T) {
	t.Parallel()

	cfg := validRustConfig()
	data := config.MarshalCommented(cfg)
	s := string(data)

	// Verify rust-specific commented fields appear.
	if !contains(s, "rust_edition") {
		t.Error("MarshalCommented for rust should contain rust_edition")
	}

	if !contains(s, `"2021"`) {
		t.Error("MarshalCommented for rust should quote rust_edition value")
	}

	if !contains(s, "unsafe_policy") {
		t.Error("MarshalCommented for rust should contain unsafe_policy")
	}

	if !contains(s, "unsafe_policy: deny") {
		t.Error("MarshalCommented for rust should contain unsafe_policy: deny (unquoted)")
	}

	// Verify round-trip.
	dir := t.TempDir()
	if err := config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() round-trip error: %v", err)
	}

	if loaded.RustEdition != "2021" {
		t.Errorf("round-trip RustEdition = %q, want %q", loaded.RustEdition, "2021")
	}

	if loaded.UnsafePolicy != "deny" {
		t.Errorf("round-trip UnsafePolicy = %q, want %q", loaded.UnsafePolicy, "deny")
	}
}

func TestMarshalCommented_ZigConfig(t *testing.T) {
	t.Parallel()

	cfg := validZigConfig()
	cfg.LinkLibc = true
	data := config.MarshalCommented(cfg)
	s := string(data)

	// Verify zig-specific commented fields appear.
	if !contains(s, "zig_version") {
		t.Error("MarshalCommented for zig should contain zig_version")
	}

	if !contains(s, `"0.13"`) {
		t.Error("MarshalCommented for zig should quote zig_version value")
	}

	if !contains(s, "link_libc: true") {
		t.Error("MarshalCommented for zig should contain link_libc: true")
	}

	// Verify round-trip.
	dir := t.TempDir()
	if err := config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() round-trip error: %v", err)
	}

	if loaded.ZigVersion != "0.13" {
		t.Errorf("round-trip ZigVersion = %q, want %q", loaded.ZigVersion, "0.13")
	}

	if !loaded.LinkLibc {
		t.Error("round-trip LinkLibc = false, want true")
	}
}

func TestMarshalCommented_CGOLibs(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Features.CGO = true
	cfg.Features.CGOLibs = []config.CGOLib{
		{
			Name:      "sqlite3",
			PkgConfig: "sqlite3",
			Include:   "/usr/include",
			LibDir:    "/usr/lib",
		},
	}

	data := config.MarshalCommented(cfg)
	s := string(data)

	if !contains(s, "cgo_libs:") {
		t.Error("MarshalCommented should contain cgo_libs section")
	}

	if !contains(s, "name: sqlite3") {
		t.Error("MarshalCommented should contain CGO lib name")
	}

	if !contains(s, "pkg_config: sqlite3") {
		t.Error("MarshalCommented should contain CGO lib pkg_config")
	}

	if !contains(s, "include: /usr/include") {
		t.Error("MarshalCommented should contain CGO lib include")
	}

	if !contains(s, "lib_dir: /usr/lib") {
		t.Error("MarshalCommented should contain CGO lib lib_dir")
	}
}

func TestMarshalCommented_Checksums(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Checksums = map[string]string{
		"AGENTS.md": "abc123",
		"Makefile":  "def456",
	}

	data := config.MarshalCommented(cfg)
	s := string(data)

	if !contains(s, "checksums:") {
		t.Error("MarshalCommented should contain checksums section")
	}

	if !contains(s, "AGENTS.md: abc123") {
		t.Error("MarshalCommented should contain AGENTS.md checksum")
	}

	if !contains(s, "Makefile: def456") {
		t.Error("MarshalCommented should contain Makefile checksum")
	}

	// Verify checksums are sorted by key.
	agentsIdx := 0
	makefileIdx := 0

	for i, line := range splitLines(s) {
		if contains(line, "AGENTS.md: abc123") {
			agentsIdx = i
		}

		if contains(line, "Makefile: def456") {
			makefileIdx = i
		}
	}

	if agentsIdx >= makefileIdx {
		t.Error("checksums should be sorted: AGENTS.md before Makefile")
	}
}

func TestQuoteVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string returns empty quotes",
			input: "",
			want:  `""`,
		},
		{
			name:  "version string is quoted",
			input: "1.21",
			want:  `"1.21"`,
		},
		{
			name:  "rust edition is quoted",
			input: "2021",
			want:  `"2021"`,
		},
		{
			name:  "zig version is quoted",
			input: "0.13",
			want:  `"0.13"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test quoteVersion indirectly via MarshalCommented.
			// quoteVersion is unexported, so we verify its behavior through
			// the commented YAML output for ecosystem fields that use Quote: true.
			cfg := validConfig()
			cfg.GoVersion = tt.input

			data := config.MarshalCommented(cfg)
			s := string(data)

			if tt.input == "" {
				// When GoVersion is empty, quoteVersion returns "" which renders as: go_version: "".
				if !contains(s, `go_version: ""`) {
					t.Errorf("MarshalCommented should contain go_version: \"\" for empty version, got:\n%s", s)
				}
			} else {
				expected := "go_version: " + tt.want
				if !contains(s, expected) {
					t.Errorf("MarshalCommented should contain %q, got:\n%s", expected, s)
				}
			}
		})
	}
}

func TestClosestWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "exact match frd",
			input: "frd",
			want:  "frd",
		},
		{
			name:  "exact match journey",
			input: "journey",
			want:  "journey",
		},
		{
			name:  "typo fRd",
			input: "fRd",
			want:  "frd",
		},
		{
			name:  "typo journy",
			input: "journy",
			want:  "journey",
		},
		{
			name:  "typo journe",
			input: "journe",
			want:  "journey",
		},
		{
			name:  "completely wrong returns empty",
			input: "zzzzzzzzz",
			want:  "",
		},
		{
			name:  "empty string",
			input: "",
			want:  "frd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// closestWorkflow is unexported, so we test it indirectly through Validate.
			cfg := validConfig()
			cfg.Workflow = tt.input

			err := cfg.Validate()
			if tt.want == tt.input {
				// Exact match should not produce a workflow error.
				if err != nil && contains(err.Error(), "unknown workflow") {
					t.Errorf("valid workflow %q should not produce workflow error: %v", tt.input, err)
				}

				return
			}

			// Invalid workflow should produce an error.
			if err == nil {
				t.Fatalf("expected validation error for workflow %q", tt.input)
			}

			got := err.Error()
			if !contains(got, "unknown workflow") {
				t.Errorf("error = %q, want to contain 'unknown workflow'", got)
			}

			if tt.want != "" {
				wantHint := `did you mean "` + tt.want + `"`
				if !contains(got, wantHint) {
					t.Errorf("error = %q, want to contain %q", got, wantHint)
				}
			} else if contains(got, "did you mean") {
				t.Errorf("error = %q, should not contain did-you-mean for distant input", got)
			}
		})
	}
}

func TestValidate_WorkflowDidYouMean(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Workflow = "journy" // close typo for "journey".

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for misspelled workflow")
	}

	if !contains(err.Error(), `did you mean "journey"`) {
		t.Errorf("error = %q, want did-you-mean suggestion for journey", err.Error())
	}
}

func TestValidate_WorkflowNoSuggestion(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Workflow = "zzzzzzzzz" // too distant from any workflow.

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if contains(err.Error(), "did you mean") {
		t.Errorf("error = %q, should not suggest for very distant input", err.Error())
	}
}

func splitLines(s string) []string {
	var lines []string

	start := 0

	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

func validConfig() *config.Config {
	cfg := config.Default()
	cfg.ProjectName = "testproject"
	cfg.ModulePath = "github.com/user/testproject"
	cfg.Binaries = []config.Binary{
		{Name: "testproject", CmdPath: "./cmd/testproject"},
	}

	return cfg
}

func validRustConfig() *config.Config {
	cfg := config.Default()
	cfg.ProjectName = "rustproject"
	cfg.ModulePath = "github.com/user/rustproject"
	cfg.Ecosystem = config.EcosystemRust
	cfg.RustEdition = "2021"
	cfg.UnsafePolicy = "deny"
	cfg.AnalysisCmd = "cargo clippy -- -D warnings"
	cfg.Binaries = []config.Binary{
		{Name: "rustproject", CmdPath: "src/main.rs"},
	}

	return cfg
}

func validZigConfig() *config.Config {
	cfg := config.Default()
	cfg.ProjectName = "zigproject"
	cfg.ModulePath = "github.com/user/zigproject"
	cfg.Ecosystem = config.EcosystemZig
	cfg.ZigVersion = "0.13"
	cfg.AnalysisCmd = "zig build test"
	cfg.Binaries = []config.Binary{
		{Name: "zigproject", CmdPath: "src/main.zig"},
	}

	return cfg
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
