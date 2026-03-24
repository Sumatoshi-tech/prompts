package promptkit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/scaffold"
)

func TestRenderAllTemplates(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "myapp",
		ModulePath:  "github.com/user/myapp",
		GoVersion:   "1.22",
		Description: "A sample application",
		Expertise:   "distributed systems",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "myapp", CmdPath: "./cmd/myapp"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			CGO:        false,
			Docker:     true,
			Benchmarks: true,
		},
		Ecosystem: "golang",
		Workflow:  "frd",
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	expectedFiles := []string{
		"AGENTS.md",
		".golangci.yml",
		"Makefile",
		"scripts/deadcode-filter.sh",
		"instructions/instr-implement.md",
		"instructions/instr-roadmaper.md",
		"instructions/instr-frd.md",
		"instructions/instr-perf.md",
		"instructions/instr-generalize.md",
	}

	for _, name := range expectedFiles {
		content, ok := result[name]
		if !ok {
			t.Errorf("missing expected file: %s", name)
			continue
		}

		if len(content) == 0 {
			t.Errorf("file %s is empty", name)
		}

		// No unresolved template markers should remain.
		if strings.Contains(string(content), "{{") {
			t.Errorf("file %s contains unresolved template markers", name)
		}
	}
}

func TestRenderAllTemplates_ProjectNameSubstitution(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "coolproject",
		ModulePath:  "github.com/cool/coolproject",
		GoVersion:   "1.23",
		Description: "Cool stuff",
		Expertise:   "cloud infrastructure",
		IdentityYrs: 20,
		Binaries: []config.Binary{
			{Name: "coolproject", CmdPath: "./cmd/coolproject"},
		},
		Quality: config.Quality{
			CoverageMin:      90,
			CoverageCritical: 95,
			ComplexityMax:    10,
			LineLength:       120,
		},
		Features: config.Features{
			Docker:     true,
			Benchmarks: true,
		},
		Ecosystem: "golang",
		Workflow:  "frd",
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// AGENTS.md should contain the project name and expertise.
	agents := string(result["AGENTS.md"])
	if !strings.Contains(agents, "coolproject") {
		t.Error("AGENTS.md missing project name")
	}

	if !strings.Contains(agents, "cloud infrastructure") {
		t.Error("AGENTS.md missing expertise")
	}

	if !strings.Contains(agents, "20+") {
		t.Error("AGENTS.md missing identity years")
	}

	// Quality thresholds should be rendered.
	if !strings.Contains(agents, ">=90%") {
		t.Error("AGENTS.md missing coverage threshold")
	}

	// .golangci.yml should have correct module path.
	golangci := string(result[".golangci.yml"])
	if !strings.Contains(golangci, "github.com/cool/coolproject") {
		t.Error(".golangci.yml missing module path")
	}

	if !strings.Contains(golangci, "1.23") {
		t.Error(".golangci.yml missing Go version")
	}

	// Makefile should reference the binary.
	makefile := string(result["Makefile"])
	if !strings.Contains(makefile, "coolproject") {
		t.Error("Makefile missing binary name")
	}
}

func TestRenderAllTemplates_CGOEnabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "cgoproject",
		ModulePath:  "github.com/user/cgoproject",
		GoVersion:   "1.22",
		Description: "A CGO project",
		Expertise:   "systems programming",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "cgoproject", CmdPath: "./cmd/cgoproject"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			CGO:        true,
			Docker:     true,
			Benchmarks: true,
			CGOLibs: []config.CGOLib{
				{
					Name:      "mylib",
					PkgConfig: "mylib",
					Include:   "third_party/mylib/include",
					LibDir:    "third_party/mylib/lib",
				},
			},
		},
		Ecosystem: "golang",
		Workflow:  "frd",
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Makefile should have CGO_ENABLED=1.
	makefile := string(result["Makefile"])
	if !strings.Contains(makefile, "CGO_ENABLED=1") {
		t.Error("Makefile should contain CGO_ENABLED=1 when CGO is enabled")
	}

	// Perf instructions should contain CGO sections.
	perf := string(result["instructions/instr-perf.md"])
	if !strings.Contains(perf, "cgo") && !strings.Contains(perf, "CGO") {
		t.Error("perf instructions should contain CGO guidance when enabled")
	}

	// AGENTS.md should have CGO troubleshooting.
	agents := string(result["AGENTS.md"])
	if !strings.Contains(agents, "CGO") {
		t.Error("AGENTS.md should contain CGO troubleshooting when enabled")
	}
}

func TestRenderAllTemplates_NoCGO(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "puregoproject",
		ModulePath:  "github.com/user/puregoproject",
		GoVersion:   "1.22",
		Description: "A pure Go project",
		Expertise:   "web development",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "puregoproject", CmdPath: "./cmd/puregoproject"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			CGO:        false,
			Docker:     false,
			Benchmarks: false,
		},
		Ecosystem: "golang",
		Workflow:  "frd",
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Makefile should have CGO_ENABLED=0.
	makefile := string(result["Makefile"])
	if !strings.Contains(makefile, "CGO_ENABLED=0") {
		t.Error("Makefile should contain CGO_ENABLED=0 when CGO is disabled")
	}

	// Makefile should not have docker targets.
	if strings.Contains(makefile, "docker-build") {
		t.Error("Makefile should not contain docker targets when Docker is disabled")
	}

	// Makefile should not have bench targets.
	if strings.Contains(makefile, "make bench") {
		t.Error("Makefile should not contain bench targets when benchmarks are disabled")
	}
}

func TestRenderFull_Claude(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "skillsapp",
		ModulePath:  "github.com/user/skillsapp",
		GoVersion:   "1.22",
		Description: "A skills test app",
		Expertise:   "testing",
		IdentityYrs: 15,
		Binaries:    []config.Binary{{Name: "skillsapp", CmdPath: "./cmd/skillsapp"}},
		Quality:     config.Quality{CoverageMin: 85, CoverageCritical: 90, ComplexityMax: 15, LineLength: 140},
		Features:    config.Features{Docker: true, Benchmarks: true},
		Agents:      []string{config.AgentClaude},
		Ecosystem:   "golang",
		Workflow:    "frd",
	}

	result, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	// Should have Agent Skills SKILL.md files.
	expectedSkills := []string{
		".agents/skills/implement/SKILL.md",
		".agents/skills/roadmap/SKILL.md",
		".agents/skills/frd/SKILL.md",
		".agents/skills/perf/SKILL.md",
		".agents/skills/generalize/SKILL.md",
	}

	for _, path := range expectedSkills {
		content, ok := result[path]
		if !ok {
			t.Errorf("missing skill file: %s", path)
			continue
		}

		if !strings.HasPrefix(string(content), "---\n") {
			t.Errorf("skill %s missing YAML frontmatter", path)
		}
	}

	// Should have Claude legacy commands.
	for _, name := range []string{"implement", "roadmap", "frd", "perf", "generalize"} {
		path := ".claude/commands/" + name + ".md"
		if _, ok := result[path]; !ok {
			t.Errorf("missing Claude command: %s", path)
		}
	}

	// Raw instructions should NOT be in the output.
	for _, path := range []string{
		"instructions/instr-implement.md",
		"instructions/instr-roadmaper.md",
		"instructions/instr-frd.md",
		"instructions/instr-perf.md",
		"instructions/instr-generalize.md",
	} {
		if _, ok := result[path]; ok {
			t.Errorf("raw instruction file should be removed: %s", path)
		}
	}

	// Non-instruction files should still be present.
	for _, path := range []string{"AGENTS.md", ".golangci.yml", "Makefile", "scripts/deadcode-filter.sh"} {
		if _, ok := result[path]; !ok {
			t.Errorf("missing expected file: %s", path)
		}
	}
}

func TestRenderFull_AllAgents(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "multiagent",
		ModulePath:  "github.com/user/multiagent",
		GoVersion:   "1.22",
		Description: "Multi-agent project",
		Expertise:   "platform engineering",
		IdentityYrs: 15,
		Binaries:    []config.Binary{{Name: "multiagent", CmdPath: "./cmd/multiagent"}},
		Quality:     config.Quality{CoverageMin: 85, CoverageCritical: 90, ComplexityMax: 15, LineLength: 140},
		Features:    config.Features{Docker: true, Benchmarks: true},
		Agents: []string{
			config.AgentClaude, config.AgentCodex, config.AgentCopilot,
			config.AgentCursor, config.AgentGemini, config.AgentWindsurf,
		},
		Ecosystem: "golang",
		Workflow:  "frd",
	}

	result, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	agentSpecificFiles := []string{
		// Agent Skills standard (shared).
		".agents/skills/implement/SKILL.md",
		// Claude.
		".claude/commands/implement.md",
		// Copilot.
		".github/copilot-instructions.md",
		// Cursor.
		".cursor/rules/agents.mdc",
		// Gemini.
		"GEMINI.md",
		".gemini/commands/implement.toml",
		// Windsurf.
		".windsurfrules",
		".windsurf/workflows/implement.md",
	}

	for _, path := range agentSpecificFiles {
		if _, ok := result[path]; !ok {
			t.Errorf("missing agent-specific file: %s", path)
		}
	}

	// No unresolved template markers in any file.
	for name, content := range result {
		if strings.Contains(string(content), "{{") {
			t.Errorf("file %s contains unresolved template markers", name)
		}
	}
}

func TestRenderAllTemplates_NoCodefangReferences(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "cleanproject",
		ModulePath:  "github.com/user/cleanproject",
		GoVersion:   "1.22",
		Description: "A clean project",
		Expertise:   "backend development",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "cleanproject", CmdPath: "./cmd/cleanproject"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			Docker:     true,
			Benchmarks: true,
		},
		Ecosystem: "golang",
		Workflow:  "frd",
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	codefangTerms := []string{"codefang", "Codefang", "Sumatoshi", "uast", "libgit2"}

	for name, content := range result {
		text := string(content)
		for _, term := range codefangTerms {
			if strings.Contains(text, term) {
				t.Errorf("file %s contains project-specific term %q — should be generalized", name, term)
			}
		}
	}
}

func TestRenderAllTemplates_RustEcosystem(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "rustapp",
		ModulePath:  "github.com/user/rustapp",
		Description: "A Rust application",
		Expertise:   "systems programming",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "rustapp", CmdPath: "src/main.rs"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			Docker:     true,
			Benchmarks: true,
		},
		Ecosystem:    "rust",
		Workflow:     "frd",
		RustEdition:  "2021",
		UnsafePolicy: "deny",
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	expectedFiles := []string{
		"AGENTS.md",
		"clippy.toml",
		"rustfmt.toml",
		"Makefile",
		"instructions/instr-implement.md",
		"instructions/instr-roadmaper.md",
		"instructions/instr-frd.md",
		"instructions/instr-perf.md",
		"instructions/instr-generalize.md",
	}

	for _, name := range expectedFiles {
		content, ok := result[name]
		if !ok {
			t.Errorf("missing expected file: %s", name)
			continue
		}

		if len(content) == 0 {
			t.Errorf("file %s is empty", name)
		}

		// No unresolved template markers should remain.
		if strings.Contains(string(content), "{{") {
			t.Errorf("file %s contains unresolved template markers", name)
		}
	}

	// AGENTS.md should reference Rust, not Go.
	agents := string(result["AGENTS.md"])
	if !strings.Contains(agents, "Rust") {
		t.Error("AGENTS.md should reference Rust")
	}

	if !strings.Contains(agents, "rustapp") {
		t.Error("AGENTS.md should contain project name")
	}

	if !strings.Contains(agents, "2021") {
		t.Error("AGENTS.md should contain Rust edition")
	}

	// Makefile should reference cargo, not go.
	makefile := string(result["Makefile"])
	if !strings.Contains(makefile, "cargo") {
		t.Error("Makefile should reference cargo")
	}

	if strings.Contains(makefile, "go build") {
		t.Error("Rust Makefile should not contain 'go build'")
	}

	// No Go-specific files should appear.
	for name := range result {
		if name == ".golangci.yml" {
			t.Error("Rust templates should not produce .golangci.yml")
		}
	}
}

func TestRenderAllTemplates_ZigEcosystem(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "zigapp",
		ModulePath:  "github.com/user/zigapp",
		Description: "A Zig application",
		Expertise:   "systems programming",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "zigapp", CmdPath: "src/main.zig"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			Docker:     true,
			Benchmarks: true,
		},
		Ecosystem:  "zig",
		Workflow:   "frd",
		ZigVersion: "0.13",
		LinkLibc:   false,
	}

	result, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	expectedFiles := []string{
		"AGENTS.md",
		"Makefile",
		"instructions/instr-implement.md",
		"instructions/instr-roadmaper.md",
		"instructions/instr-frd.md",
		"instructions/instr-perf.md",
		"instructions/instr-generalize.md",
	}

	for _, name := range expectedFiles {
		content, ok := result[name]
		if !ok {
			t.Errorf("missing expected file: %s", name)
			continue
		}

		if len(content) == 0 {
			t.Errorf("file %s is empty", name)
		}

		if strings.Contains(string(content), "{{") {
			t.Errorf("file %s contains unresolved template markers", name)
		}
	}

	// AGENTS.md should reference Zig, not Go or Rust.
	agents := string(result["AGENTS.md"])
	if !strings.Contains(agents, "Zig") {
		t.Error("AGENTS.md should reference Zig")
	}

	if !strings.Contains(agents, "zigapp") {
		t.Error("AGENTS.md should contain project name")
	}

	// Makefile should reference zig, not go or cargo.
	makefile := string(result["Makefile"])
	if !strings.Contains(makefile, "zig build") {
		t.Error("Makefile should reference 'zig build'")
	}

	if strings.Contains(makefile, "go build") {
		t.Error("Zig Makefile should not contain 'go build'")
	}

	if strings.Contains(makefile, "cargo") {
		t.Error("Zig Makefile should not contain 'cargo'")
	}

	// No Go-specific or Rust-specific files should appear.
	for name := range result {
		if name == ".golangci.yml" {
			t.Error("Zig templates should not produce .golangci.yml")
		}

		if name == "clippy.toml" || name == "rustfmt.toml" {
			t.Errorf("Zig templates should not produce %s", name)
		}
	}
}

// TestInit_NoPositionalArgUsesCWD verifies that when no project-dir argument
// is provided, init writes files to the current working directory (TC-23).
func TestInit_NoPositionalArgUsesCWD(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := &config.Config{
		ProjectName: "cwdproject",
		ModulePath:  "github.com/user/cwdproject",
		GoVersion:   "1.22",
		Description: "CWD test project",
		Expertise:   "testing",
		IdentityYrs: 15,
		Binaries:    []config.Binary{{Name: "cwdproject", CmdPath: "./cmd/cwdproject"}},
		Quality:     config.Quality{CoverageMin: 85, CoverageCritical: 90, ComplexityMax: 15, LineLength: 140},
		Features:    config.Features{Docker: true, Benchmarks: true},
		Agents:      []string{config.AgentClaude},
		Ecosystem:   "golang",
		Workflow:    "frd",
	}

	// Render and apply to tmpDir (simulating init writing to CWD).
	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	cfg.GeneratedFiles = scaffold.FileManifest(rendered)

	if err = config.Save(cfg, tmpDir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err = scaffold.Apply(rendered, tmpDir, scaffold.ModeCreate); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// Verify .promptkit.yaml exists in the target directory.
	configPath := filepath.Join(tmpDir, config.FileName)
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf(".promptkit.yaml not found in CWD target: %s", tmpDir)
	}

	// Verify key generated files exist.
	for _, path := range []string{"AGENTS.md", ".golangci.yml", "Makefile", "scripts/deadcode-filter.sh"} {
		fullPath := filepath.Join(tmpDir, path)
		if _, err = os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %s not found in CWD target", path)
		}
	}
}

func TestRenderFull_JourneyWorkflow_Golang(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName: "journeyapp",
		ModulePath:  "github.com/user/journeyapp",
		GoVersion:   "1.22",
		Description: "Journey workflow test",
		Expertise:   "testing",
		IdentityYrs: 15,
		Binaries:    []config.Binary{{Name: "journeyapp", CmdPath: "./cmd/journeyapp"}},
		Quality:     config.Quality{CoverageMin: 85, CoverageCritical: 90, ComplexityMax: 15, LineLength: 140},
		Features:    config.Features{Docker: true, Benchmarks: true},
		Agents:      []string{config.AgentClaude},
		Ecosystem:   "golang",
		Workflow:    "journey",
	}

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	// Journey skill should be present, FRD should not.
	hasJourney := false
	hasFRD := false

	for path := range rendered {
		if strings.Contains(path, "journey") {
			hasJourney = true
		}

		if strings.Contains(path, "/frd/") || strings.HasSuffix(path, "/frd.md") || strings.HasSuffix(path, "/frd.toml") {
			hasFRD = true
		}
	}

	if !hasJourney {
		t.Error("journey workflow should produce journey skill files")
	}

	if hasFRD {
		t.Error("journey workflow should not produce frd skill files")
	}

	// Implement skill should reference journey, not FRD.
	for path, content := range rendered {
		if strings.Contains(path, "implement") {
			s := string(content)
			if !strings.Contains(s, "specs/journeys/JOURNEY-{id}.md") &&
				strings.Contains(s, "specs/frds/FRD-{id}.md") {
				t.Errorf("implement skill at %s should reference journey, not FRD", path)
			}
		}
	}
}

func TestRenderFull_JourneyWorkflow_Rust(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ProjectName:  "rustjourney",
		ModulePath:   "github.com/user/rustjourney",
		Description:  "Rust journey test",
		Expertise:    "testing",
		IdentityYrs:  15,
		Binaries:     []config.Binary{{Name: "rustjourney", CmdPath: "src/main.rs"}},
		Quality:      config.Quality{CoverageMin: 85, CoverageCritical: 90, ComplexityMax: 15, LineLength: 140},
		Features:     config.Features{Docker: true, Benchmarks: true},
		Ecosystem:    "rust",
		Workflow:     "journey",
		RustEdition:  "2024",
		UnsafePolicy: "deny",
	}

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	hasJourney := false
	hasFRD := false

	for path := range rendered {
		if strings.Contains(path, "journey") {
			hasJourney = true
		}

		if strings.Contains(path, "/frd/") || strings.HasSuffix(path, "/frd.md") || strings.HasSuffix(path, "/frd.toml") {
			hasFRD = true
		}
	}

	if !hasJourney {
		t.Error("rust journey workflow should produce journey skill files")
	}

	if hasFRD {
		t.Error("rust journey workflow should not produce frd skill files")
	}
}
