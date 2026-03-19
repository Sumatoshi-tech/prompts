package adapters_test

import (
	"strings"
	"testing"

	"github.com/Sumatoshi-tech/prompts/internal/adapters"
	"github.com/Sumatoshi-tech/prompts/internal/config"
)

func TestPlaceForAgents_Claude(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentClaude})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Should produce Agent Skills standard paths.
	assertContainsPath(t, paths, ".agents/skills/implement/SKILL.md")
	assertContainsPath(t, paths, ".agents/skills/roadmap/SKILL.md")
	assertContainsPath(t, paths, ".agents/skills/perf/SKILL.md")

	// FRD/journey are templates under .agents/instructions/, not separate skills.
	if paths[".agents/skills/frd/SKILL.md"] || paths[".agents/skills/journey/SKILL.md"] {
		t.Error("frd/journey should not be Agent Skills")
	}

	// Should also produce Claude legacy commands.
	assertContainsPath(t, paths, ".claude/commands/implement.md")
	assertContainsPath(t, paths, ".claude/commands/roadmap.md")
	assertContainsPath(t, paths, ".claude/commands/perf.md")

	if paths[".claude/commands/frd.md"] || paths[".claude/commands/journey.md"] {
		t.Error("frd/journey should not have Claude slash-command files")
	}
}

func TestPlaceForAgents_SkillMDHasFrontmatter(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentClaude})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Path, "SKILL.md") {
			continue
		}

		content := string(f.Content)
		if !strings.HasPrefix(content, "---\n") {
			t.Errorf("SKILL.md at %s missing YAML frontmatter", f.Path)
		}

		if !strings.Contains(content, "name:") {
			t.Errorf("SKILL.md at %s missing name field", f.Path)
		}

		if !strings.Contains(content, "description:") {
			t.Errorf("SKILL.md at %s missing description field", f.Path)
		}
	}
}

func TestPlaceForAgents_Codex(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentCodex})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Codex uses Agent Skills standard.
	assertContainsPath(t, paths, ".agents/skills/implement/SKILL.md")
	assertContainsPath(t, paths, ".agents/skills/roadmap/SKILL.md")
}

func TestPlaceForAgents_Copilot(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentCopilot})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Copilot uses Agent Skills + copilot-instructions.md.
	assertContainsPath(t, paths, ".agents/skills/implement/SKILL.md")
	assertContainsPath(t, paths, ".github/copilot-instructions.md")
}

func TestPlaceForAgents_Cursor(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentCursor})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Cursor uses Agent Skills + .cursor/rules/*.mdc.
	assertContainsPath(t, paths, ".agents/skills/implement/SKILL.md")
	assertContainsPath(t, paths, ".cursor/rules/agents.mdc")

	// Verify MDC frontmatter.
	for _, f := range files {
		if f.Path == ".cursor/rules/agents.mdc" {
			content := string(f.Content)
			if !strings.Contains(content, "alwaysApply: true") {
				t.Error("MDC file missing alwaysApply frontmatter")
			}

			if !strings.Contains(content, "globs:") {
				t.Error("MDC file missing globs frontmatter")
			}
		}
	}
}

func TestPlaceForAgents_Gemini(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentGemini})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Gemini uses TOML commands + GEMINI.md.
	assertContainsPath(t, paths, "GEMINI.md")
	assertContainsPath(t, paths, ".gemini/commands/implement.toml")
	assertContainsPath(t, paths, ".gemini/commands/roadmap.toml")

	// Verify TOML format.
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".toml") {
			continue
		}

		content := string(f.Content)
		if !strings.Contains(content, "description =") {
			t.Errorf("TOML file %s missing description field", f.Path)
		}

		if !strings.Contains(content, "prompt = \"\"\"") {
			t.Errorf("TOML file %s missing prompt field", f.Path)
		}
	}
}

func TestPlaceForAgents_Windsurf(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentWindsurf})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Windsurf uses .windsurfrules + .windsurf/workflows/*.md.
	assertContainsPath(t, paths, ".windsurfrules")
	assertContainsPath(t, paths, ".windsurf/workflows/implement.md")
	assertContainsPath(t, paths, ".windsurf/workflows/roadmap.md")
}

func TestPlaceForAgents_MultipleAgents(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	agents := []string{config.AgentClaude, config.AgentGemini, config.AgentCursor}

	files, err := adapters.PlaceForAgents(rendered, agents)
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	// Agent Skills (shared across claude, cursor).
	assertContainsPath(t, paths, ".agents/skills/implement/SKILL.md")

	// Claude-specific.
	assertContainsPath(t, paths, ".claude/commands/implement.md")

	// Gemini-specific.
	assertContainsPath(t, paths, ".gemini/commands/implement.toml")
	assertContainsPath(t, paths, "GEMINI.md")

	// Cursor-specific.
	assertContainsPath(t, paths, ".cursor/rules/agents.mdc")
}

func TestPlaceForAgents_UnknownAgent(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	_, err := adapters.PlaceForAgents(rendered, []string{"unknown"})
	if err == nil {
		t.Fatal("expected error for unknown agent, got nil")
	}
}

func TestRemoveInstructionPaths(t *testing.T) {
	t.Parallel()

	paths := adapters.RemoveInstructionPaths()

	if len(paths) == 0 {
		t.Fatal("expected instruction paths to remove")
	}

	for _, p := range paths {
		if !strings.HasPrefix(p, adapters.AgentInstructionsDir+"/") {
			t.Errorf("unexpected removal path: %s", p)
		}
	}
}

func testRendered() map[string][]byte {
	return map[string][]byte{
		"AGENTS.md":                       []byte("# Agent Personality\nTest content"),
		".agents/instructions/instr-implement.md": []byte("# Implementation instructions"),
		".agents/instructions/instr-roadmaper.md": []byte("# Roadmap instructions"),
		".agents/instructions/instr-frd.md":       []byte("# FRD template"),
		".agents/instructions/instr-perf.md":      []byte("# Performance instructions"),
		".golangci.yml":                   []byte("version: 2"),
		"Makefile":                        []byte("all: build"),
	}
}

func filePaths(files []adapters.PlacedFile) map[string]bool {
	result := make(map[string]bool, len(files))
	for _, f := range files {
		result[f.Path] = true
	}

	return result
}

func assertContainsPath(t *testing.T, paths map[string]bool, expected string) {
	t.Helper()

	if !paths[expected] {
		t.Errorf("missing expected path: %s", expected)
	}
}

func TestFileOwnership_SingleAgent(t *testing.T) {
	t.Parallel()

	rendered := testRendered()

	ownership, err := adapters.FileOwnership(rendered, []string{config.AgentClaude})
	if err != nil {
		t.Fatalf("FileOwnership() error: %v", err)
	}

	// Claude-specific file should be owned by claude.
	fa, ok := ownership[".claude/commands/implement.md"]
	if !ok {
		t.Fatal("missing .claude/commands/implement.md in ownership")
	}

	if len(fa.Agents) != 1 || fa.Agents[0] != config.AgentClaude {
		t.Errorf("expected [claude], got %v", fa.Agents)
	}

	if fa.IsShared {
		t.Error("expected IsShared=false for single-agent file")
	}
}

func TestFileOwnership_MultipleAgents(t *testing.T) {
	t.Parallel()

	rendered := testRendered()
	agents := []string{config.AgentClaude, config.AgentCursor}

	ownership, err := adapters.FileOwnership(rendered, agents)
	if err != nil {
		t.Fatalf("FileOwnership() error: %v", err)
	}

	// Agent Skills files are shared across claude and cursor.
	fa, ok := ownership[".agents/skills/implement/SKILL.md"]
	if !ok {
		t.Fatal("missing .agents/skills/implement/SKILL.md in ownership")
	}

	if len(fa.Agents) < 2 {
		t.Errorf("expected multiple agents, got %v", fa.Agents)
	}
}

func TestFileOwnership_SharedVsSpecific(t *testing.T) {
	t.Parallel()

	rendered := testRendered()
	agents := []string{config.AgentClaude, config.AgentGemini}

	ownership, err := adapters.FileOwnership(rendered, agents)
	if err != nil {
		t.Fatalf("FileOwnership() error: %v", err)
	}

	// AGENTS.md is a base file — no agent ownership.
	if fa, ok := ownership["AGENTS.md"]; ok {
		if len(fa.Agents) > 0 {
			t.Errorf("AGENTS.md should be a base file, got agents: %v", fa.Agents)
		}
	}

	// GEMINI.md is Gemini-specific.
	if fa, ok := ownership["GEMINI.md"]; ok {
		if fa.IsShared {
			t.Error("GEMINI.md should not be shared")
		}

		if len(fa.Agents) != 1 || fa.Agents[0] != config.AgentGemini {
			t.Errorf("GEMINI.md expected [gemini], got %v", fa.Agents)
		}
	} else {
		t.Error("missing GEMINI.md in ownership")
	}
}

func TestPlaceForAgents_JourneyRendered_SameSkillsAsFRD(t *testing.T) {
	t.Parallel()

	// Workflow templates are not skills; journey vs frd only affects which
	// Workflow template under .agents/instructions/ remains on disk — not a skill.
	rendered := testRenderedJourney()

	files, err := adapters.PlaceForAgents(rendered, []string{config.AgentClaude})
	if err != nil {
		t.Fatalf("PlaceForAgents() error: %v", err)
	}

	paths := filePaths(files)

	assertContainsPath(t, paths, ".agents/skills/implement/SKILL.md")
	assertContainsPath(t, paths, ".agents/skills/roadmap/SKILL.md")
	assertContainsPath(t, paths, ".agents/skills/perf/SKILL.md")

	if paths[".agents/skills/frd/SKILL.md"] || paths[".agents/skills/journey/SKILL.md"] {
		t.Error("frd/journey should not be Agent Skills")
	}
}

func TestRemoveInstructionPaths_SkillsOnlyNotWorkflowTemplates(t *testing.T) {
	t.Parallel()

	paths := adapters.RemoveInstructionPaths()
	pathSet := make(map[string]bool)

	for _, p := range paths {
		pathSet[p] = true
	}

	// Workflow outlines stay on disk for /implement — do not delete them here.
	if pathSet[".agents/instructions/instr-frd.md"] {
		t.Error("RemoveInstructionPaths must not include instr-frd.md")
	}

	if pathSet[".agents/instructions/instr-journey.md"] {
		t.Error("RemoveInstructionPaths must not include instr-journey.md")
	}

	if !pathSet[".agents/instructions/instr-implement.md"] {
		t.Error("RemoveInstructionPaths should include instr-implement.md")
	}
}

func testRenderedJourney() map[string][]byte {
	return map[string][]byte{
		"AGENTS.md":                       []byte("# Agent Personality\nTest content"),
		".agents/instructions/instr-implement.md": []byte("# Implementation instructions"),
		".agents/instructions/instr-roadmaper.md": []byte("# Roadmap instructions"),
		".agents/instructions/instr-journey.md":   []byte("# Journey template"),
		".agents/instructions/instr-perf.md":      []byte("# Performance instructions"),
		".golangci.yml":                   []byte("version: 2"),
		"Makefile":                        []byte("all: build"),
	}
}

// TestSkillContent_SemanticEquivalence verifies that the same skill content
// is semantically preserved across different agent formats (SKILL.md, .md,
// .toml, .mdc, workflows). The raw instruction body should appear in every
// agent's output regardless of wrapper format.
func TestSkillContent_SemanticEquivalence(t *testing.T) {
	t.Parallel()

	rendered := testRendered()
	allAgents := []string{
		config.AgentClaude,
		config.AgentCodex,
		config.AgentCopilot,
		config.AgentCursor,
		config.AgentGemini,
		config.AgentWindsurf,
	}

	// The raw body of the "implement" skill.
	rawBody := string(rendered[".agents/instructions/instr-implement.md"])
	if rawBody == "" {
		t.Fatal("missing implement instruction in rendered")
	}

	// Strip whitespace for comparison.
	normalize := func(s string) string {
		return strings.Join(strings.Fields(s), " ")
	}
	expected := normalize(rawBody)

	for _, agent := range allAgents {
		files, err := adapters.PlaceForAgents(rendered, []string{agent})
		if err != nil {
			t.Fatalf("PlaceForAgents(%s) error: %v", agent, err)
		}

		// Find the "implement" skill output for this agent.
		found := false

		for _, f := range files {
			content := normalize(string(f.Content))
			if strings.Contains(content, expected) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("agent %s: implement skill body not found in any output file", agent)
		}
	}
}
