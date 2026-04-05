package mixtures

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Sumatoshi-tech/prompts/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		ProjectName: "testproject",
		Ecosystem:   "golang",
		Workflow:    "frd",
	}
}

func TestLoadAll_EmptyFS(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{}

	defs, err := LoadAll(fs)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	if len(defs) != 0 {
		t.Errorf("LoadAll() returned %d defs, want 0", len(defs))
	}
}

func TestLoadAll_SingleMixture(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: security\ndescription: Security-first\nuse_case: Secure apps\ntargets:\n  - implement\n  - bug\n"),
		},
		"templates/_shared/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Security content for implement"),
		},
	}

	defs, err := LoadAll(fs)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("LoadAll() returned %d defs, want 1", len(defs))
	}

	def, ok := defs["security"]
	if !ok {
		t.Fatal("missing 'security' mixture")
	}

	if def.Description != "Security-first" {
		t.Errorf("Description = %q, want %q", def.Description, "Security-first")
	}

	if def.UseCase != "Secure apps" {
		t.Errorf("UseCase = %q, want %q", def.UseCase, "Secure apps")
	}

	if len(def.Targets) != 2 || def.Targets[0] != "implement" || def.Targets[1] != "bug" {
		t.Errorf("Targets = %v, want [implement, bug]", def.Targets)
	}
}

func TestLoadAll_MultipleMixtures(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: security\ndescription: Security\ntargets:\n  - implement\n"),
		},
		"templates/_shared/mixtures/observability/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: observability\ndescription: Observability\ntargets:\n  - implement\n  - perf\n"),
		},
	}

	defs, err := LoadAll(fs)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	if len(defs) != 2 {
		t.Fatalf("LoadAll() returned %d defs, want 2", len(defs))
	}

	if _, ok := defs["security"]; !ok {
		t.Error("missing 'security'")
	}

	if _, ok := defs["observability"]; !ok {
		t.Error("missing 'observability'")
	}
}

func TestLoadAll_SkipsDirWithoutYAML(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/incomplete/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("content without metadata"),
		},
	}

	defs, err := LoadAll(fs)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	if len(defs) != 0 {
		t.Errorf("LoadAll() returned %d defs, want 0 (no mixture.yaml)", len(defs))
	}
}

func TestNames_Sorted(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: security\ntargets: []\n"),
		},
		"templates/_shared/mixtures/alpha/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: alpha\ntargets: []\n"),
		},
	}

	names := Names(fs)
	if len(names) != 2 {
		t.Fatalf("Names() returned %d, want 2", len(names))
	}

	if names[0] != "alpha" || names[1] != "security" {
		t.Errorf("Names() = %v, want [alpha, security]", names)
	}
}

func TestRenderForSkill_SharedTemplate(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Security rules for {{.ProjectName}}"),
		},
	}

	cfg := testConfig()

	result, err := RenderForSkill(fs, "golang", "security", "implement", cfg)
	if err != nil {
		t.Fatalf("RenderForSkill() error: %v", err)
	}

	want := "Security rules for testproject"
	if string(result) != want {
		t.Errorf("RenderForSkill() = %q, want %q", string(result), want)
	}
}

func TestRenderForSkill_EcosystemOverride(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Shared security"),
		},
		"templates/golang/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Go-specific security for {{.ProjectName}}"),
		},
	}

	cfg := testConfig()

	result, err := RenderForSkill(fs, "golang", "security", "implement", cfg)
	if err != nil {
		t.Fatalf("RenderForSkill() error: %v", err)
	}

	want := "Go-specific security for testproject"
	if string(result) != want {
		t.Errorf("RenderForSkill() = %q, want %q", string(result), want)
	}
}

func TestRenderForSkill_NoTemplate(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{}

	cfg := testConfig()

	result, err := RenderForSkill(fs, "golang", "security", "implement", cfg)
	if err != nil {
		t.Fatalf("RenderForSkill() error: %v", err)
	}

	if result != nil {
		t.Errorf("RenderForSkill() = %v, want nil for missing template", result)
	}
}

func TestAppendToSkill_NoMixtures(t *testing.T) {
	t.Parallel()

	body := []byte("Original skill content")

	result, err := AppendToSkill(body, fstest.MapFS{}, "golang", nil, "implement", testConfig())
	if err != nil {
		t.Fatalf("AppendToSkill() error: %v", err)
	}

	if !bytes.Equal(result, body) {
		t.Errorf("AppendToSkill() modified body with no active mixtures")
	}
}

func TestAppendToSkill_InjectsSingleMixture(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: security\ndescription: Security-first\ntargets:\n  - implement\n"),
		},
		"templates/_shared/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Validate all inputs."),
		},
	}

	body := []byte("Original skill content")

	result, err := AppendToSkill(body, fs, "golang", []string{"security"}, "implement", testConfig())
	if err != nil {
		t.Fatalf("AppendToSkill() error: %v", err)
	}

	got := string(result)
	if !strings.HasPrefix(got, "Original skill content") {
		t.Error("original content should be preserved at start")
	}

	if !strings.Contains(got, "## Mixture: Security-first") {
		t.Error("mixture header missing")
	}

	if !strings.Contains(got, "Validate all inputs.") {
		t.Error("mixture content missing")
	}
}

func TestAppendToSkill_SkipsUntargetedSkill(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: security\ndescription: Security\ntargets:\n  - implement\n"),
		},
		"templates/_shared/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Security content"),
		},
	}

	body := []byte("Original perf content")

	// "perf" is not in the targets list.
	result, err := AppendToSkill(body, fs, "golang", []string{"security"}, "perf", testConfig())
	if err != nil {
		t.Fatalf("AppendToSkill() error: %v", err)
	}

	if !bytes.Equal(result, body) {
		t.Error("body should not be modified for untargeted skill")
	}
}

func TestAppendToSkill_MultipleMixturesSorted(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/_shared/mixtures/security/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: security\ndescription: Security\ntargets:\n  - implement\n"),
		},
		"templates/_shared/mixtures/security/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Security injection"),
		},
		"templates/_shared/mixtures/alpha/mixture.yaml": &fstest.MapFile{
			Data: []byte("name: alpha\ndescription: Alpha\ntargets:\n  - implement\n"),
		},
		"templates/_shared/mixtures/alpha/implement.md.tmpl": &fstest.MapFile{
			Data: []byte("Alpha injection"),
		},
	}

	body := []byte("Base")

	// Pass in reverse order — output should still be alphabetical.
	result, err := AppendToSkill(body, fs, "golang", []string{"security", "alpha"}, "implement", testConfig())
	if err != nil {
		t.Fatalf("AppendToSkill() error: %v", err)
	}

	got := string(result)

	alphaIdx := strings.Index(got, "Alpha injection")
	securityIdx := strings.Index(got, "Security injection")

	if alphaIdx == -1 || securityIdx == -1 {
		t.Fatalf("missing injections in: %s", got)
	}

	if alphaIdx > securityIdx {
		t.Error("alpha should appear before security (alphabetical order)")
	}
}
