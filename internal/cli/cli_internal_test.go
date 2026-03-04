package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	promptkit "github.com/Sumatoshi-tech/promptkit"
	"github.com/Sumatoshi-tech/promptkit/internal/config"
	"github.com/Sumatoshi-tech/promptkit/internal/scaffold"
)

// internalSetupProject creates a scaffolded project in a temp dir and returns the dir path.
func internalSetupProject(t *testing.T, modify func(*config.Config)) string {
	t.Helper()

	dir := t.TempDir()
	cfg := internalTestConfig()

	if modify != nil {
		modify(cfg)
	}

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	cfg.GeneratedFiles = scaffold.FileManifest(rendered)
	cfg.Checksums = scaffold.ComputeChecksums(rendered)

	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err = scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	return dir
}

func internalTestConfig() *config.Config {
	return &config.Config{
		Version:     config.CurrentVersion,
		ProjectName: "testproject",
		ModulePath:  "github.com/user/testproject",
		GoVersion:   "1.22",
		Description: "A test project",
		Expertise:   "testing",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "testproject", CmdPath: "./cmd/testproject"},
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
		Agents:       []string{config.AgentClaude},
		Ecosystem:    "golang",
		Workflow:     "frd",
		AnalysisCmd:  "go vet ./...",
		TemplateOver: ".promptkit/templates",
	}
}

// captureStdout captures stdout for the duration of fn and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}

	os.Stdout = writer //nolint:reassign // test helper must capture stdout

	fn()

	writer.Close()

	os.Stdout = old //nolint:reassign // restoring original stdout

	var buf bytes.Buffer

	_, _ = io.Copy(&buf, reader)

	return buf.String()
}

// withCwd changes to dir for the test, restoring the original dir on cleanup.
func withCwd(t *testing.T, dir string) {
	t.Helper()

	t.Chdir(dir)
}

// saveRootFlags saves and restores rootFlags after the test.
func saveRootFlags(t *testing.T) {
	t.Helper()

	oldVerbose := rootFlags.verbose
	oldConfig := rootFlags.configPath

	t.Cleanup(func() {
		rootFlags.verbose = oldVerbose
		rootFlags.configPath = oldConfig
	})
}

// Priority 1: Pure functions.

func TestSha256hex_EmptyString(t *testing.T) {
	t.Parallel()

	got := sha256hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	if got != want {
		t.Errorf("sha256hex(empty) = %q, want %q", got, want)
	}
}

func TestSha256hex_KnownValue(t *testing.T) {
	t.Parallel()

	got := sha256hex([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	if got != want {
		t.Errorf("sha256hex(hello) = %q, want %q", got, want)
	}
}

func TestSha256hex_Binary(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0xFF, 0x01, 0xFE}
	got := sha256hex(data)

	if len(got) != 64 {
		t.Errorf("sha256hex(binary) has length %d, want 64", len(got))
	}
}

func TestResolveConfigDir_WithFlag(t *testing.T) { //nolint:paralleltest // modifies global rootFlags
	saveRootFlags(t)

	rootFlags.configPath = "/some/custom/path"

	dir, err := resolveConfigDir()
	if err != nil {
		t.Fatalf("resolveConfigDir() error: %v", err)
	}

	if dir != "/some/custom/path" {
		t.Errorf("resolveConfigDir() = %q, want %q", dir, "/some/custom/path")
	}
}

func TestResolveConfigDir_SearchesCWD(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and cwd
	saveRootFlags(t)

	rootFlags.configPath = ""

	dir := internalSetupProject(t, nil)
	withCwd(t, dir)

	result, err := resolveConfigDir()
	if err != nil {
		t.Fatalf("resolveConfigDir() error: %v", err)
	}

	if result != dir {
		t.Errorf("resolveConfigDir() = %q, want %q", result, dir)
	}
}

func TestResolveConfigDir_NotFound(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and cwd
	saveRootFlags(t)

	rootFlags.configPath = ""

	dir := t.TempDir()
	withCwd(t, dir)

	_, err := resolveConfigDir()
	if err == nil {
		t.Fatal("expected error when config not found")
	}

	if !strings.Contains(err.Error(), config.FileName) {
		t.Errorf("error = %q, want to contain %q", err.Error(), config.FileName)
	}
}

// Priority 2: Doctor command.

func TestRunDoctor_AllGreen(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	output := captureStdout(t, func() {
		err := runDoctor(nil, nil)
		if err != nil {
			t.Errorf("runDoctor() error: %v", err)
		}
	})

	if !strings.Contains(output, "promptkit doctor:") {
		t.Errorf("output should contain header, got: %s", output)
	}

	if !strings.Contains(output, "[ok] Config loads and validates") {
		t.Errorf("output should report config ok, got: %s", output)
	}

	if !strings.Contains(output, "[ok]") {
		t.Errorf("output should contain [ok], got: %s", output)
	}
}

func TestRunDoctor_MissingFiles(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatalf("removing AGENTS.md: %v", err)
	}

	output := captureStdout(t, func() {
		err := runDoctor(nil, nil)
		if err == nil {
			t.Error("expected error from runDoctor with missing files")
		} else if !strings.Contains(err.Error(), "doctor found errors") {
			t.Errorf("error = %q, want 'doctor found errors'", err.Error())
		}
	})

	if !strings.Contains(output, "[err] Missing: AGENTS.md") {
		t.Errorf("output should report missing AGENTS.md, got: %s", output)
	}
}

func TestRunDoctor_InvalidConfig(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	dir := t.TempDir()
	rootFlags.configPath = dir

	configPath := filepath.Join(dir, config.FileName)
	if err := os.WriteFile(configPath, []byte(":::invalid:::"), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	output := captureStdout(t, func() {
		err := runDoctor(nil, nil)
		if err == nil {
			t.Error("expected error from runDoctor with invalid config")
		}
	})

	if !strings.Contains(output, "[err] Config") {
		t.Errorf("output should report config error, got: %s", output)
	}
}

func TestRunDoctor_ModifiedFiles(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	// Manually modify a generated file to trigger the [warn] path.
	agentsPath := filepath.Join(dir, "AGENTS.md")

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}

	modified := slices.Concat(data, []byte("\n# manual edit\n"))
	if err = os.WriteFile(agentsPath, modified, 0o600); err != nil {
		t.Fatalf("writing modified AGENTS.md: %v", err)
	}

	output := captureStdout(t, func() {
		err = runDoctor(nil, nil)
		// Modified files are warnings, not errors.
		if err != nil {
			t.Errorf("runDoctor() unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "[warn]") {
		t.Errorf("output should contain [warn] for modified files, got: %s", output)
	}
}

// Priority 2: Clean command.

func TestRunClean_NoStaleFiles(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, cleanFlags and os.Stdout
	saveRootFlags(t)

	oldYes := cleanFlags.yes
	oldDryRun := cleanFlags.dryRun

	t.Cleanup(func() {
		cleanFlags.yes = oldYes
		cleanFlags.dryRun = oldDryRun
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir
	cleanFlags.yes = true
	cleanFlags.dryRun = false

	output := captureStdout(t, func() {
		err := runClean(nil, nil)
		if err != nil {
			t.Errorf("runClean() error: %v", err)
		}
	})

	if !strings.Contains(output, "No stale files") {
		t.Errorf("output should say 'No stale files', got: %s", output)
	}
}

func TestRunClean_DryRun(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, cleanFlags and os.Stdout
	saveRootFlags(t)

	oldYes := cleanFlags.yes
	oldDryRun := cleanFlags.dryRun

	t.Cleanup(func() {
		cleanFlags.yes = oldYes
		cleanFlags.dryRun = oldDryRun
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	// Add a fake stale file to the manifest.
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	staleFile := "stale-file.txt"
	if err = os.WriteFile(filepath.Join(dir, staleFile), []byte("stale"), 0o600); err != nil {
		t.Fatalf("writing stale file: %v", err)
	}

	cfg.GeneratedFiles = append(cfg.GeneratedFiles, staleFile)
	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	cleanFlags.yes = false
	cleanFlags.dryRun = true

	output := captureStdout(t, func() {
		err = runClean(nil, nil)
		if err != nil {
			t.Errorf("runClean() error: %v", err)
		}
	})

	if !strings.Contains(output, "stale-file.txt") {
		t.Errorf("output should mention stale file, got: %s", output)
	}

	if !strings.Contains(output, "Dry run") {
		t.Errorf("output should mention dry run, got: %s", output)
	}

	// File should still exist.
	if _, statErr := os.Stat(filepath.Join(dir, staleFile)); statErr != nil {
		t.Error("stale file should still exist after dry run")
	}
}

func TestRunClean_RemovesStale(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, cleanFlags and os.Stdout
	saveRootFlags(t)

	oldYes := cleanFlags.yes
	oldDryRun := cleanFlags.dryRun

	t.Cleanup(func() {
		cleanFlags.yes = oldYes
		cleanFlags.dryRun = oldDryRun
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	staleFile := "stale-output.txt"
	if err := os.WriteFile(filepath.Join(dir, staleFile), []byte("stale"), 0o600); err != nil {
		t.Fatalf("writing stale file: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg.GeneratedFiles = append(cfg.GeneratedFiles, staleFile)
	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	cleanFlags.yes = true
	cleanFlags.dryRun = false

	output := captureStdout(t, func() {
		err = runClean(nil, nil)
		if err != nil {
			t.Errorf("runClean() error: %v", err)
		}
	})

	if !strings.Contains(output, "stale-output.txt") {
		t.Errorf("output should mention stale file, got: %s", output)
	}

	if !strings.Contains(output, "Removed") {
		t.Errorf("output should mention removal, got: %s", output)
	}

	if _, statErr := os.Stat(filepath.Join(dir, staleFile)); !os.IsNotExist(statErr) {
		t.Error("stale file should be removed")
	}
}

// Priority 2: Diff command.

func TestRunDiff_NoChanges(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, diffFlags and os.Stdout
	saveRootFlags(t)

	oldUpstream := diffFlags.upstream

	t.Cleanup(func() {
		diffFlags.upstream = oldUpstream
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir
	diffFlags.upstream = ""
	rootFlags.verbose = false

	output := captureStdout(t, func() {
		err := runDiff(nil, nil)
		if err != nil {
			t.Errorf("runDiff() error: %v", err)
		}
	})

	if !strings.Contains(output, "All files are up to date") {
		t.Errorf("output should say up to date, got: %s", output)
	}
}

func TestRunDiff_WithChanges(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, diffFlags and os.Stdout
	saveRootFlags(t)

	oldUpstream := diffFlags.upstream

	t.Cleanup(func() {
		diffFlags.upstream = oldUpstream
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir
	diffFlags.upstream = ""
	rootFlags.verbose = false

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg.Quality.CoverageMin = 70
	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	output := captureStdout(t, func() {
		err = runDiff(nil, nil)
		if err == nil {
			t.Error("expected error from runDiff when files differ")
		}
	})

	if !strings.Contains(output, "file(s) with changes") {
		t.Errorf("output should report changes, got: %s", output)
	}
}

func TestRunDiff_Upstream(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, diffFlags and os.Stdout
	saveRootFlags(t)

	oldUpstream := diffFlags.upstream

	t.Cleanup(func() {
		diffFlags.upstream = oldUpstream
	})

	localDir := internalSetupProject(t, nil)
	upstreamDir := internalSetupProject(t, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 70
	})

	rootFlags.configPath = localDir
	diffFlags.upstream = upstreamDir

	output := captureStdout(t, func() {
		err := runDiff(nil, nil)
		if err == nil {
			t.Error("expected error from runDiff when configs differ")
		}
	})

	if !strings.Contains(output, "file(s) with differences") {
		t.Errorf("output should report differences, got: %s", output)
	}
}

func TestRunDiff_UpstreamIdentical(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, diffFlags and os.Stdout
	saveRootFlags(t)

	oldUpstream := diffFlags.upstream

	t.Cleanup(func() {
		diffFlags.upstream = oldUpstream
	})

	dir1 := internalSetupProject(t, nil)
	dir2 := internalSetupProject(t, nil)

	rootFlags.configPath = dir1
	diffFlags.upstream = dir2

	output := captureStdout(t, func() {
		err := runDiff(nil, nil)
		if err != nil {
			t.Errorf("runDiff() error: %v", err)
		}
	})

	if !strings.Contains(output, "identical output") {
		t.Errorf("output should report identical, got: %s", output)
	}
}

// Priority 2: Config explain.

func TestRunConfigExplain_AllFields(t *testing.T) { //nolint:paralleltest // modifies os.Stdout via captureStdout
	output := captureStdout(t, func() {
		err := runConfigExplain(nil, []string{})
		if err != nil {
			t.Errorf("runConfigExplain() error: %v", err)
		}
	})

	if !strings.Contains(output, "Config field -> output file mapping:") {
		t.Errorf("output should contain header, got: %s", output)
	}

	if !strings.Contains(output, "project_name") {
		t.Errorf("output should contain project_name, got: %s", output)
	}

	if !strings.Contains(output, "quality.coverage_min") {
		t.Errorf("output should contain quality.coverage_min, got: %s", output)
	}

	if !strings.Contains(output, "agents") {
		t.Errorf("output should contain agents, got: %s", output)
	}
}

func TestRunConfigExplain_SingleField(t *testing.T) { //nolint:paralleltest // modifies os.Stdout via captureStdout
	output := captureStdout(t, func() {
		err := runConfigExplain(nil, []string{"quality.coverage_min"})
		if err != nil {
			t.Errorf("runConfigExplain() error: %v", err)
		}
	})

	if !strings.Contains(output, "quality.coverage_min") {
		t.Errorf("output should contain field name, got: %s", output)
	}

	if !strings.Contains(output, "Affects:") {
		t.Errorf("output should contain Affects:, got: %s", output)
	}

	if !strings.Contains(output, "AGENTS.md") {
		t.Errorf("output should mention AGENTS.md, got: %s", output)
	}
}

func TestRunConfigExplain_UnknownKey(t *testing.T) {
	t.Parallel()

	err := runConfigExplain(nil, []string{"nonexistent_key"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}

	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error = %q, want to contain 'unknown config key'", err.Error())
	}

	if !strings.Contains(err.Error(), "Valid keys:") {
		t.Errorf("error = %q, want to contain 'Valid keys:'", err.Error())
	}
}

func TestRunConfigExplain_AgentsField(t *testing.T) { //nolint:paralleltest // modifies os.Stdout via captureStdout
	output := captureStdout(t, func() {
		err := runConfigExplain(nil, []string{"agents"})
		if err != nil {
			t.Errorf("runConfigExplain() error: %v", err)
		}
	})

	if !strings.Contains(output, "agents") {
		t.Errorf("output should contain 'agents', got: %s", output)
	}

	if !strings.Contains(output, "agent-specific file placement") {
		t.Errorf("output should mention agent-specific placement, got: %s", output)
	}
}

// Priority 3: Template commands.

func TestRunTemplateList(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	output := captureStdout(t, func() {
		err := runTemplateList(nil, nil)
		if err != nil {
			t.Errorf("runTemplateList() error: %v", err)
		}
	})

	if !strings.Contains(output, "Available templates") {
		t.Errorf("output should contain header, got: %s", output)
	}

	if !strings.Contains(output, "AGENTS.md") {
		t.Errorf("output should list AGENTS.md, got: %s", output)
	}

	if !strings.Contains(output, "Makefile") {
		t.Errorf("output should list Makefile, got: %s", output)
	}
}

func TestRunTemplateList_FallbackEcosystem(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	// Use a temp dir with no config -- should fall back to golang.
	dir := t.TempDir()
	rootFlags.configPath = dir

	output := captureStdout(t, func() {
		err := runTemplateList(nil, nil)
		if err != nil {
			t.Errorf("runTemplateList() error: %v", err)
		}
	})

	if !strings.Contains(output, "Available templates") {
		t.Errorf("output should contain header, got: %s", output)
	}
}

func TestRunTemplateVars(t *testing.T) { //nolint:paralleltest // modifies os.Stdout via captureStdout
	output := captureStdout(t, func() {
		err := runTemplateVars(nil, nil)
		if err != nil {
			t.Errorf("runTemplateVars() error: %v", err)
		}
	})

	if !strings.Contains(output, "Available template variables") {
		t.Errorf("output should contain header, got: %s", output)
	}

	if !strings.Contains(output, ".ProjectName") {
		t.Errorf("output should contain .ProjectName, got: %s", output)
	}

	if !strings.Contains(output, ".ModulePath") {
		t.Errorf("output should contain .ModulePath, got: %s", output)
	}

	if !strings.Contains(output, ".Quality") {
		t.Errorf("output should contain .Quality, got: %s", output)
	}
}

func TestRunTemplateRender(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and os.Stdout
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	output := captureStdout(t, func() {
		err := runTemplateRender(nil, []string{"AGENTS.md"})
		if err != nil {
			t.Errorf("runTemplateRender() error: %v", err)
		}
	})

	if !strings.Contains(output, "testproject") {
		t.Errorf("rendered AGENTS.md should contain project name, got: %s", output)
	}
}

func TestRunTemplateRender_NotFound(t *testing.T) { //nolint:paralleltest // modifies global rootFlags
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	err := runTemplateRender(nil, []string{"nonexistent-template"})
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestPrintStructFields(t *testing.T) { //nolint:paralleltest // modifies os.Stdout via captureStdout
	output := captureStdout(t, func() {
		printStructFields(reflect.TypeFor[config.Config](), ".", "  ")
	})

	if !strings.Contains(output, ".ProjectName") {
		t.Errorf("output should contain .ProjectName, got: %s", output)
	}

	if !strings.Contains(output, ".ModulePath") {
		t.Errorf("output should contain .ModulePath, got: %s", output)
	}

	if !strings.Contains(output, ".Quality") {
		t.Errorf("output should contain .Quality, got: %s", output)
	}

	if !strings.Contains(output, ".Features") {
		t.Errorf("output should contain .Features, got: %s", output)
	}

	if !strings.Contains(output, ".Agents") {
		t.Errorf("output should contain .Agents, got: %s", output)
	}

	// Should contain nested fields.
	if !strings.Contains(output, ".Quality.CoverageMin") {
		t.Errorf("output should contain .Quality.CoverageMin, got: %s", output)
	}

	// Nested feature fields.
	if !strings.Contains(output, ".Features.CGO") {
		t.Errorf("output should contain .Features.CGO, got: %s", output)
	}

	if !strings.Contains(output, ".Features.Docker") {
		t.Errorf("output should contain .Features.Docker, got: %s", output)
	}

	// Should show type names.
	if !strings.Contains(output, "string") {
		t.Errorf("output should show string type, got: %s", output)
	}
}

// Priority 4: buildConfigFromFlags.

func TestBuildConfigFromFlags_Golang(t *testing.T) { //nolint:paralleltest // modifies global initFlags
	oldFlags := initFlags

	t.Cleanup(func() {
		initFlags = oldFlags
	})

	initFlags.nonInteractive = true
	initFlags.projectName = "mygoapp"
	initFlags.modulePath = "github.com/test/mygoapp"
	initFlags.description = "A Go app"
	initFlags.expertise = "distributed systems"
	initFlags.binary = "mygoapp"
	initFlags.cgo = false
	initFlags.docker = true
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"

	cfg := config.Default()

	result, err := buildConfigFromFlags(cfg, "/tmp/test")
	if err != nil {
		t.Fatalf("buildConfigFromFlags() error: %v", err)
	}

	if result.ProjectName != "mygoapp" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "mygoapp")
	}

	if result.ModulePath != "github.com/test/mygoapp" {
		t.Errorf("ModulePath = %q, want %q", result.ModulePath, "github.com/test/mygoapp")
	}

	if result.Description != "A Go app" {
		t.Errorf("Description = %q, want %q", result.Description, "A Go app")
	}

	if result.Expertise != "distributed systems" {
		t.Errorf("Expertise = %q, want %q", result.Expertise, "distributed systems")
	}

	if result.Ecosystem != "golang" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "golang")
	}

	if result.Workflow != "frd" {
		t.Errorf("Workflow = %q, want %q", result.Workflow, "frd")
	}

	if result.Features.CGO {
		t.Error("Features.CGO should be false")
	}

	if !result.Features.Docker {
		t.Error("Features.Docker should be true")
	}

	if len(result.Agents) != 1 || result.Agents[0] != "claude" {
		t.Errorf("Agents = %v, want [claude]", result.Agents)
	}

	if result.AnalysisCmd != "go vet ./..." {
		t.Errorf("AnalysisCmd = %q, want 'go vet ./...'", result.AnalysisCmd)
	}

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries len = %d, want 1", len(result.Binaries))
	}

	if result.Binaries[0].Name != "mygoapp" {
		t.Errorf("Binaries[0].Name = %q, want %q", result.Binaries[0].Name, "mygoapp")
	}

	if result.Binaries[0].CmdPath != "./cmd/mygoapp" {
		t.Errorf("Binaries[0].CmdPath = %q, want %q", result.Binaries[0].CmdPath, "./cmd/mygoapp")
	}
}

func TestBuildConfigFromFlags_Rust(t *testing.T) { //nolint:paralleltest // modifies global initFlags
	oldFlags := initFlags

	t.Cleanup(func() {
		initFlags = oldFlags
	})

	initFlags.nonInteractive = true
	initFlags.projectName = "myrustapp"
	initFlags.modulePath = "github.com/test/myrustapp"
	initFlags.description = "A Rust app"
	initFlags.expertise = "systems programming"
	initFlags.binary = "myrustapp"
	initFlags.cgo = false
	initFlags.docker = true
	initFlags.agents = []string{"claude", "cursor"}
	initFlags.ecosystem = "rust"
	initFlags.workflow = "frd"

	cfg := config.Default()

	result, err := buildConfigFromFlags(cfg, "/tmp/test")
	if err != nil {
		t.Fatalf("buildConfigFromFlags() error: %v", err)
	}

	if result.ProjectName != "myrustapp" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "myrustapp")
	}

	if result.Ecosystem != "rust" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "rust")
	}

	if result.AnalysisCmd != "cargo clippy -- -D warnings" {
		t.Errorf("AnalysisCmd = %q, want 'cargo clippy -- -D warnings'", result.AnalysisCmd)
	}

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries len = %d, want 1", len(result.Binaries))
	}

	if result.Binaries[0].CmdPath != "src/main.rs" {
		t.Errorf("Binaries[0].CmdPath = %q, want %q", result.Binaries[0].CmdPath, "src/main.rs")
	}

	if len(result.Agents) != 2 {
		t.Errorf("Agents len = %d, want 2", len(result.Agents))
	}
}

func TestBuildConfigFromFlags_DefaultProjectName(t *testing.T) { //nolint:paralleltest // modifies global initFlags
	oldFlags := initFlags

	t.Cleanup(func() {
		initFlags = oldFlags
	})

	initFlags.nonInteractive = true
	initFlags.projectName = "" // empty: should use dir basename.
	initFlags.modulePath = "github.com/test/myapp"
	initFlags.binary = ""
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"
	initFlags.description = ""
	initFlags.expertise = ""
	initFlags.cgo = false
	initFlags.docker = true

	cfg := config.Default()

	result, err := buildConfigFromFlags(cfg, "/tmp/myapp")
	if err != nil {
		t.Fatalf("buildConfigFromFlags() error: %v", err)
	}

	if result.ProjectName != "myapp" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "myapp")
	}

	if result.Binaries[0].Name != "myapp" {
		t.Errorf("Binaries[0].Name = %q, want %q", result.Binaries[0].Name, "myapp")
	}
}

func TestBuildConfigFromFlags_InvalidConfig(t *testing.T) { //nolint:paralleltest // modifies global initFlags
	oldFlags := initFlags

	t.Cleanup(func() {
		initFlags = oldFlags
	})

	initFlags.nonInteractive = true
	initFlags.projectName = ""
	initFlags.modulePath = "" // required field empty.
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"
	initFlags.binary = ""
	initFlags.description = ""
	initFlags.expertise = ""
	initFlags.cgo = false
	initFlags.docker = true

	cfg := config.Default()

	_, err := buildConfigFromFlags(cfg, "/tmp/test")
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	if !strings.Contains(err.Error(), "invalid config") {
		t.Errorf("error = %q, want to contain 'invalid config'", err.Error())
	}
}

// printFilesByAgent.

func TestPrintFilesByAgent(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude, config.AgentCursor}
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	output := captureStdout(t, func() {
		printFilesByAgent(rendered, cfg.Agents, cfg.Workflow)
	})

	if !strings.Contains(output, "Wrote") {
		t.Errorf("output should contain 'Wrote', got: %s", output)
	}

	if !strings.Contains(output, "file(s)") {
		t.Errorf("output should contain 'file(s)', got: %s", output)
	}

	if !strings.Contains(output, "Shared:") {
		t.Errorf("output should contain 'Shared:', got: %s", output)
	}
}

func TestPrintFilesByAgent_SingleAgent(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	output := captureStdout(t, func() {
		printFilesByAgent(rendered, cfg.Agents, cfg.Workflow)
	})

	if !strings.Contains(output, "Wrote") {
		t.Errorf("output should contain 'Wrote', got: %s", output)
	}

	// Single agent should not have "Agents:" summary line.
	if strings.Contains(output, "\nAgents:") {
		t.Errorf("output should not have Agents: summary for single agent, got: %s", output)
	}
}

// runVerify.

func TestRunVerify_CustomCommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	dir := t.TempDir()

	err := runVerify(&buf, dir, "true")
	if err != nil {
		t.Fatalf("runVerify() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Running verification: true") {
		t.Errorf("output should show command, got: %s", output)
	}

	if !strings.Contains(output, "Verification passed") {
		t.Errorf("output should say passed, got: %s", output)
	}
}

func TestRunVerify_FailingCommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	dir := t.TempDir()

	err := runVerify(&buf, dir, "false")
	if err == nil {
		t.Fatal("expected error for failing command")
	}

	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("error = %q, want to contain 'verification failed'", err.Error())
	}
}

func TestRunVerify_DefaultCommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	dir := t.TempDir()

	// Empty analysisCmd defaults to "make lint".
	err := runVerify(&buf, dir, "")
	if err == nil {
		t.Fatal("expected error since make lint won't exist")
	}

	output := buf.String()

	if !strings.Contains(output, "Running verification: make lint") {
		t.Errorf("output should show default command, got: %s", output)
	}
}

func TestRunVerify_MultiWordCommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	dir := t.TempDir()

	// Use "echo hello" which should succeed.
	err := runVerify(&buf, dir, "echo hello")
	if err != nil {
		t.Fatalf("runVerify() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Verification passed") {
		t.Errorf("output should say passed, got: %s", output)
	}
}

// Template extract.

func TestRunTemplateExtract(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, extractFlags and cwd
	saveRootFlags(t)

	oldForce := extractFlags.force

	t.Cleanup(func() {
		extractFlags.force = oldForce
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir
	extractFlags.force = false

	withCwd(t, dir)

	output := captureStdout(t, func() {
		err := runTemplateExtract(nil, []string{"AGENTS.md"})
		if err != nil {
			t.Errorf("runTemplateExtract() error: %v", err)
		}
	})

	if !strings.Contains(output, "Extracted template") {
		t.Errorf("output should confirm extraction, got: %s", output)
	}

	extractedPath := filepath.Join(dir, ".promptkit", "templates", "AGENTS.md.tmpl")
	if _, err := os.Stat(extractedPath); err != nil {
		t.Errorf("extracted file should exist at %s: %v", extractedPath, err)
	}
}

func TestRunTemplateExtract_AlreadyExists(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, extractFlags and cwd
	saveRootFlags(t)

	oldForce := extractFlags.force

	t.Cleanup(func() {
		extractFlags.force = oldForce
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir
	extractFlags.force = false

	withCwd(t, dir)

	// Extract once.
	captureStdout(t, func() {
		if err := runTemplateExtract(nil, []string{"AGENTS.md"}); err != nil {
			t.Fatalf("first extract error: %v", err)
		}
	})

	// Second extract should fail.
	err := runTemplateExtract(nil, []string{"AGENTS.md"})
	if err == nil {
		t.Fatal("expected error for existing override")
	}

	if !strings.Contains(err.Error(), "override already exists") {
		t.Errorf("error = %q, want 'override already exists'", err.Error())
	}
}

func TestRunTemplateExtract_Force(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, extractFlags and cwd
	saveRootFlags(t)

	oldForce := extractFlags.force

	t.Cleanup(func() {
		extractFlags.force = oldForce
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	withCwd(t, dir)

	// Extract once.
	extractFlags.force = false

	captureStdout(t, func() {
		if err := runTemplateExtract(nil, []string{"AGENTS.md"}); err != nil {
			t.Fatalf("first extract error: %v", err)
		}
	})

	// Second extract with --force should succeed.
	extractFlags.force = true

	output := captureStdout(t, func() {
		err := runTemplateExtract(nil, []string{"AGENTS.md"})
		if err != nil {
			t.Errorf("force extract error: %v", err)
		}
	})

	if !strings.Contains(output, "Extracted template") {
		t.Errorf("output should confirm extraction, got: %s", output)
	}
}

func TestRunTemplateExtract_NotFound(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, extractFlags and cwd
	saveRootFlags(t)

	oldForce := extractFlags.force

	t.Cleanup(func() {
		extractFlags.force = oldForce
	})

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir
	extractFlags.force = false

	withCwd(t, dir)

	err := runTemplateExtract(nil, []string{"nonexistent-template-name"})
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}

	if !strings.Contains(err.Error(), "not found in embedded templates") {
		t.Errorf("error = %q, want 'not found in embedded templates'", err.Error())
	}
}

// hasDiff.

func TestHasDiff(t *testing.T) {
	t.Parallel()

	diffs := []scaffold.FileDiff{
		{Path: "AGENTS.md"},
		{Path: "Makefile"},
	}

	if !hasDiff(diffs, "AGENTS.md") {
		t.Error("hasDiff should return true for AGENTS.md")
	}

	if !hasDiff(diffs, "Makefile") {
		t.Error("hasDiff should return true for Makefile")
	}

	if hasDiff(diffs, "nonexistent.txt") {
		t.Error("hasDiff should return false for nonexistent.txt")
	}
}

// RunUpdate verify flag.

func TestRunUpdate_VerifySuccess(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg.Description = "Updated for verify test"
	cfg.AnalysisCmd = "true"

	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer

	opts := UpdateOptions{
		Dir:    dir,
		Yes:    true,
		Verify: true,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	}

	if err = RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Running verification") {
		t.Errorf("output should show verification, got: %s", output)
	}

	if !strings.Contains(output, "Verification passed") {
		t.Errorf("output should show verification passed, got: %s", output)
	}
}

func TestRunUpdate_VerifyFailure(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg.Description = "Updated for verify fail test"
	cfg.AnalysisCmd = "false"

	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer

	opts := UpdateOptions{
		Dir:    dir,
		Yes:    true,
		Verify: true,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	}

	err = RunUpdate(opts)
	if err == nil {
		t.Fatal("expected error from RunUpdate with failing verify")
	}

	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("error = %q, want 'verification failed'", err.Error())
	}
}

// RunUpdate quiet mode.

func TestRunUpdate_QuietMode(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	var buf bytes.Buffer

	opts := UpdateOptions{
		Dir:    dir,
		Yes:    true,
		Quiet:  true,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	}

	if err := RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()

	if strings.Contains(output, "[1/") {
		t.Errorf("quiet mode should suppress progress, got: %s", output)
	}

	if strings.Contains(output, "Loading config") {
		t.Errorf("quiet mode should suppress step messages, got: %s", output)
	}
}

// runTemplateAdd.

func TestRunTemplateAdd(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, cwd and os.Stdout
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	withCwd(t, dir)

	// Create a source file to add as a template.
	srcFile := filepath.Join(dir, "my-template-source.txt")
	if err := os.WriteFile(srcFile, []byte("{{.ProjectName}} custom template"), 0o600); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	output := captureStdout(t, func() {
		err := runTemplateAdd(nil, []string{"custom-file", srcFile})
		if err != nil {
			t.Errorf("runTemplateAdd() error: %v", err)
		}
	})

	if !strings.Contains(output, "Added template override") {
		t.Errorf("output should confirm add, got: %s", output)
	}

	// Verify the override file was created.
	overridePath := filepath.Join(dir, ".promptkit", "templates", "custom-file.tmpl")

	data, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("reading override file: %v", err)
	}

	if string(data) != "{{.ProjectName}} custom template" {
		t.Errorf("override content = %q, want source content", string(data))
	}
}

func TestRunTemplateAdd_SourceNotFound(t *testing.T) { //nolint:paralleltest // modifies global rootFlags and cwd
	saveRootFlags(t)

	dir := internalSetupProject(t, nil)
	rootFlags.configPath = dir

	withCwd(t, dir)

	err := runTemplateAdd(nil, []string{"custom-file", "/nonexistent/path/source.txt"})
	if err == nil {
		t.Fatal("expected error for nonexistent source file")
	}

	if !strings.Contains(err.Error(), "reading source file") {
		t.Errorf("error = %q, want 'reading source file'", err.Error())
	}
}

// runInit non-interactive.

func TestRunInit_NonInteractive(t *testing.T) { //nolint:paralleltest // modifies global initFlags, rootFlags and os.Stdout
	oldInitFlags := initFlags

	saveRootFlags(t)

	t.Cleanup(func() {
		initFlags = oldInitFlags
	})

	dir := t.TempDir()
	targetDir := filepath.Join(dir, "myproject")

	initFlags.force = false
	initFlags.dryRun = false
	initFlags.nonInteractive = true
	initFlags.projectName = "myproject"
	initFlags.modulePath = "github.com/test/myproject"
	initFlags.description = "Test init project"
	initFlags.expertise = "testing"
	initFlags.binary = "myproject"
	initFlags.cgo = false
	initFlags.docker = true
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"
	rootFlags.verbose = false

	output := captureStdout(t, func() {
		err := runInit(nil, []string{targetDir})
		if err != nil {
			t.Errorf("runInit() error: %v", err)
		}
	})

	if !strings.Contains(output, "Project initialized") {
		t.Errorf("output should say 'Project initialized', got: %s", output)
	}

	// Config file should exist.
	if _, err := os.Stat(filepath.Join(targetDir, config.FileName)); err != nil {
		t.Errorf("config file should exist: %v", err)
	}

	// AGENTS.md should exist.
	if _, err := os.Stat(filepath.Join(targetDir, "AGENTS.md")); err != nil {
		t.Errorf("AGENTS.md should exist: %v", err)
	}
}

func TestRunInit_DryRun(t *testing.T) { //nolint:paralleltest // modifies global initFlags, rootFlags and os.Stdout
	oldInitFlags := initFlags

	saveRootFlags(t)

	t.Cleanup(func() {
		initFlags = oldInitFlags
	})

	dir := t.TempDir()
	targetDir := filepath.Join(dir, "dryrunproject")

	initFlags.force = false
	initFlags.dryRun = true
	initFlags.nonInteractive = true
	initFlags.projectName = "dryrunproject"
	initFlags.modulePath = "github.com/test/dryrunproject"
	initFlags.description = "Dry run test"
	initFlags.expertise = "testing"
	initFlags.binary = "dryrunproject"
	initFlags.cgo = false
	initFlags.docker = true
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"
	rootFlags.verbose = false

	output := captureStdout(t, func() {
		err := runInit(nil, []string{targetDir})
		if err != nil {
			t.Errorf("runInit() error: %v", err)
		}
	})

	if !strings.Contains(output, "Dry run") {
		t.Errorf("output should say 'Dry run', got: %s", output)
	}

	// No files should be written (except possibly the directory itself).
	if _, err := os.Stat(filepath.Join(targetDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("AGENTS.md should not exist in dry-run mode")
	}
}

func TestRunInit_AlreadyExists(t *testing.T) { //nolint:paralleltest // modifies global initFlags, rootFlags and os.Stdout
	oldInitFlags := initFlags

	saveRootFlags(t)

	t.Cleanup(func() {
		initFlags = oldInitFlags
	})

	dir := internalSetupProject(t, nil)

	initFlags.force = false
	initFlags.nonInteractive = true
	initFlags.projectName = "testproject"
	initFlags.modulePath = "github.com/test/testproject"
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"

	captureStdout(t, func() {
		err := runInit(nil, []string{dir})
		if err == nil {
			t.Fatal("expected error when config already exists")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error = %q, want 'already exists'", err.Error())
		}
	})
}

func TestRunInit_VerboseMode(t *testing.T) { //nolint:paralleltest // modifies global initFlags, rootFlags and os.Stdout
	oldInitFlags := initFlags

	saveRootFlags(t)

	t.Cleanup(func() {
		initFlags = oldInitFlags
	})

	dir := t.TempDir()
	targetDir := filepath.Join(dir, "verboseproject")

	initFlags.force = false
	initFlags.dryRun = false
	initFlags.nonInteractive = true
	initFlags.projectName = "verboseproject"
	initFlags.modulePath = "github.com/test/verboseproject"
	initFlags.description = "Verbose init test"
	initFlags.expertise = "testing"
	initFlags.binary = "verboseproject"
	initFlags.cgo = false
	initFlags.docker = true
	initFlags.agents = []string{"claude"}
	initFlags.ecosystem = "golang"
	initFlags.workflow = "frd"
	rootFlags.verbose = true

	output := captureStdout(t, func() {
		err := runInit(nil, []string{targetDir})
		if err != nil {
			t.Errorf("runInit() error: %v", err)
		}
	})

	if !strings.Contains(output, "Rendered") {
		t.Errorf("verbose output should mention rendered files, got: %s", output)
	}

	if !strings.Contains(output, "Project initialized") {
		t.Errorf("output should say 'Project initialized', got: %s", output)
	}
}

// runDiffUpstream with new-only files.

func TestRunDiff_UpstreamNewFilesOnly(t *testing.T) { //nolint:paralleltest // modifies global rootFlags, diffFlags and os.Stdout
	saveRootFlags(t)

	oldUpstream := diffFlags.upstream

	t.Cleanup(func() {
		diffFlags.upstream = oldUpstream
	})

	// Local has claude+cursor, upstream has only claude.
	localDir := internalSetupProject(t, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude, config.AgentCursor}
	})
	upstreamDir := internalSetupProject(t, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude}
	})

	rootFlags.configPath = localDir
	diffFlags.upstream = upstreamDir

	output := captureStdout(t, func() {
		err := runDiff(nil, nil)
		// Diff with differences returns error.
		if err == nil {
			t.Error("expected error from runDiff when configs differ")
		}
	})

	// Should report files that exist only in local or only in upstream.
	if !strings.Contains(output, "file(s) with differences") {
		t.Errorf("output should report differences, got: %s", output)
	}
}

// RunUpdate with checksums/manual edit warning.

func TestRunUpdate_ChecksumWarning(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	// Manually modify a file on disk so the checksum differs.
	agentsPath := filepath.Join(dir, "AGENTS.md")

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}

	modified := slices.Concat(data, []byte("\n# manual edit\n"))
	if err = os.WriteFile(agentsPath, modified, 0o600); err != nil {
		t.Fatalf("writing modified AGENTS.md: %v", err)
	}

	// Also change config so there are diffs to show.
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg.Description = "Changed description"
	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer

	opts := UpdateOptions{
		Dir:     dir,
		Yes:     true,
		Verbose: true,
		DryRun:  true,
		Stdout:  &buf,
		Stdin:   strings.NewReader(""),
	}

	_ = RunUpdate(opts)

	output := buf.String()

	if !strings.Contains(output, "Warning:") && !strings.Contains(output, "manually modified") {
		t.Errorf("verbose output should warn about manual modification, got: %s", output)
	}
}

// Execute.

func TestExecute(t *testing.T) { //nolint:paralleltest // modifies os.Args and os.Stdout
	// Exercise Execute by running the root command with --help.
	// This just ensures it doesn't panic; the cobra framework handles --help.
	oldArgs := os.Args

	t.Cleanup(func() {
		os.Args = oldArgs //nolint:reassign // restoring original os.Args
	})

	os.Args = []string{"promptkit", "--help"} //nolint:reassign // test must set os.Args for cobra

	output := captureStdout(t, func() {
		_ = Execute()
	})

	if !strings.Contains(output, "promptkit") {
		t.Errorf("help output should contain 'promptkit', got: %s", output)
	}
}

// RunUpdate with explain + verify step counting.

func TestRunUpdate_ExplainWithVerify(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	var buf bytes.Buffer

	opts := UpdateOptions{
		Dir:     dir,
		Yes:     true,
		Explain: true,
		Verify:  true,
		Stdout:  &buf,
		Stdin:   strings.NewReader(""),
	}

	// Will fail at verify because there's no analysis command that works,
	// but the explain output should include step 10 (verify).
	// The project has no diffs, so verify won't be reached. Let's just
	// check the explain output.
	_ = RunUpdate(opts)

	output := buf.String()

	if !strings.Contains(output, "10. Run verification command") {
		t.Errorf("explain output with verify should include step 10, got: %s", output)
	}
}

// RunUpdate with stale files removal.

func TestRunUpdate_RemovesStaleFiles(t *testing.T) {
	t.Parallel()

	dir := internalSetupProject(t, nil)

	// Add a stale file to the manifest.
	staleFile := "stale-from-update.txt"
	if err := os.WriteFile(filepath.Join(dir, staleFile), []byte("old content"), 0o600); err != nil {
		t.Fatalf("writing stale file: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg.GeneratedFiles = append(cfg.GeneratedFiles, staleFile)
	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer

	opts := UpdateOptions{
		Dir:    dir,
		Yes:    true,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	}

	if err = RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Stale") || !strings.Contains(output, staleFile) {
		t.Errorf("output should report stale file, got: %s", output)
	}

	if !strings.Contains(output, "Removed") {
		t.Errorf("output should confirm removal, got: %s", output)
	}

	// The stale file should be removed.
	if _, statErr := os.Stat(filepath.Join(dir, staleFile)); !os.IsNotExist(statErr) {
		t.Error("stale file should have been removed")
	}
}
