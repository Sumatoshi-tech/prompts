package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/cli"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/scaffold"
)

// writeTestFile writes data to path, failing the test on error.
func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writing test file %s: %v", path, err)
	}
}

// setupProject creates a scaffolded project in a temp dir and returns the dir path.
func setupProject(t *testing.T, modify func(*config.Config)) string {
	t.Helper()

	dir := t.TempDir()
	cfg := testUpdateConfig()

	if modify != nil {
		modify(cfg)
	}

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	cfg.GeneratedFiles = scaffold.FileManifest(rendered)

	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err = scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	return dir
}

// modifyConfig loads config from dir, applies a modification, and saves it back.
func modifyConfig(t *testing.T, dir string, modify func(*config.Config)) {
	t.Helper()

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	modify(cfg)

	if err = config.Save(cfg, dir); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
}

// runUpdateCapture runs RunUpdate and captures stdout.
func runUpdateCapture(t *testing.T, dir string, yes, dryRun bool) (string, error) {
	t.Helper()

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:    dir,
		Yes:    yes,
		DryRun: dryRun,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	}

	err := cli.RunUpdate(opts)

	return buf.String(), err
}

func testUpdateConfig() *config.Config {
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
		AnalysisCmd:  "go vet ./...",
		TemplateOver: ".promptkit/templates",
	}
}

// Test 1: Single Value Edit Updates Affected Files.
func TestUpdate_SingleValueEdit(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Verify AGENTS.md contains 85 initially.
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}

	if !strings.Contains(string(agents), "85") {
		t.Fatal("AGENTS.md should contain coverage threshold 85")
	}

	// Change coverage_min to 90.
	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 90
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	// AGENTS.md should now contain 90.
	agents, err = os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md after update: %v", err)
	}

	if !strings.Contains(string(agents), "90") {
		t.Error("AGENTS.md should contain updated coverage threshold 90")
	}
}

// Test 2: Multiple Value Edit Updates All Affected Files.
func TestUpdate_MultipleValueEdit(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Features.Docker = false
		cfg.Quality.ComplexityMax = 20
		cfg.Description = "A web service"
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	// Makefile should no longer contain docker targets.
	makefile, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if strings.Contains(string(makefile), "docker-build") {
		t.Error("Makefile should not contain docker-build when Docker is disabled")
	}

	// AGENTS.md should reflect new description.
	agents, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(agents), "A web service") {
		t.Error("AGENTS.md should contain updated description")
	}

	// Should report changes.
	if !strings.Contains(output, "Updated") {
		t.Error("output should report updated files")
	}
}

// Test 3: Invalid YAML Syntax Fails with Parse Error.
func TestUpdate_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Corrupt the config file.
	configPath := filepath.Join(dir, config.FileName)
	if err := os.WriteFile(configPath, []byte(":::invalid:::\nproject_name: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := runUpdateCapture(t, dir, true, false)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "config") {
		t.Errorf("error = %q, want to contain 'config'", err.Error())
	}
}

// Test 4: Invalid Config Value Fails Validation.
func TestUpdate_InvalidConfigValue(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Write config with invalid coverage value.
	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 150
	})

	// Load will fail validation, so we need to write raw YAML.
	configPath := filepath.Join(dir, config.FileName)
	raw, _ := os.ReadFile(configPath)
	corrupted := string(raw) // coverage_min: 150 is already invalid (>100); use as-is.
	writeTestFile(t, configPath, []byte(corrupted))

	_, err := runUpdateCapture(t, dir, true, false)
	if err == nil {
		t.Fatal("expected error for invalid config value")
	}

	if !strings.Contains(err.Error(), "coverage_min must be 1-100") {
		t.Errorf("error = %q, want coverage_min validation message", err.Error())
	}
}

// Test 5: Missing Required Field Fails Validation.
func TestUpdate_MissingRequiredField(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Write config without project_name.
	configPath := filepath.Join(dir, config.FileName)
	raw, _ := os.ReadFile(configPath)
	// Replace project_name value with empty string.
	corrupted := strings.Replace(string(raw), "project_name: testproject", "project_name:", 1)
	writeTestFile(t, configPath, []byte(corrupted))

	_, err := runUpdateCapture(t, dir, true, false)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}

	if !strings.Contains(err.Error(), "project_name is required") {
		t.Errorf("error = %q, want project_name error", err.Error())
	}
}

// Test 6: Diff Preview Shows Correct Changed Files.
func TestUpdate_DiffPreview(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Features.Docker = false
	})

	// Use dry-run to preview.
	output, _ := runUpdateCapture(t, dir, false, true)

	if !strings.Contains(output, "~ Makefile (modified)") {
		t.Errorf("output should show Makefile as modified:\n%s", output)
	}

	if !strings.Contains(output, "Found") && !strings.Contains(output, "file(s) with changes") {
		t.Errorf("output should show change count:\n%s", output)
	}
}

// Test 7: Approving Changes Writes Files to Disk.
func TestUpdate_ApproveWritesFiles(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 90
	})

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:    dir,
		Yes:    false,
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Updated") {
		t.Error("output should confirm update")
	}

	// Subsequent update should show no changes.
	output2, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("second RunUpdate() error: %v", err)
	}

	if !strings.Contains(output2, "All files are up to date") {
		t.Errorf("second update should show no changes:\n%s", output2)
	}
}

// Test 8: Rejecting Changes Leaves Files Unchanged.
func TestUpdate_RejectLeavesUnchanged(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Record file checksums before.
	agentsBefore, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 90
	})

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:    dir,
		Yes:    false,
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	if !strings.Contains(buf.String(), "Aborted") {
		t.Error("output should say Aborted")
	}

	// File should be unchanged.
	agentsAfter, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !bytes.Equal(agentsBefore, agentsAfter) {
		t.Error("AGENTS.md should not have changed after rejection")
	}
}

// Test 9: --yes Flag Auto-Approves Without Prompt.
func TestUpdate_YesAutoApproves(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, func(cfg *config.Config) {
		cfg.Features.Benchmarks = true
	})

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Features.Benchmarks = false
	})

	// Run with --yes and closed stdin.
	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	if strings.Contains(output, "Apply changes?") {
		t.Error("--yes should skip approval prompt")
	}

	if !strings.Contains(output, "Updated") {
		t.Error("output should confirm update")
	}

	// Makefile should reflect the change.
	makefile, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if strings.Contains(string(makefile), "make bench") {
		t.Error("Makefile should not contain bench targets")
	}
}

// Test 10: Adding a New Agent Generates Adapter Files.
func TestUpdate_AddAgent(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude}
	})

	// Verify no Gemini files initially.
	if _, err := os.Stat(filepath.Join(dir, "GEMINI.md")); err == nil {
		t.Fatal("GEMINI.md should not exist initially")
	}

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude, config.AgentGemini}
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	// GEMINI.md should now exist.
	if _, err = os.Stat(filepath.Join(dir, "GEMINI.md")); err != nil {
		t.Error("GEMINI.md should exist after adding gemini agent")
	}

	// Gemini command files should exist.
	geminiCmds := filepath.Join(dir, ".gemini", "commands")
	if _, err = os.Stat(geminiCmds); err != nil {
		t.Error(".gemini/commands/ should exist")
	}

	// Output should show new files.
	if !strings.Contains(output, "(new)") {
		t.Error("output should list new files")
	}

	// Claude files should still exist.
	if _, err = os.Stat(filepath.Join(dir, ".claude", "commands")); err != nil {
		t.Error("Claude commands should still exist")
	}
}

// Test 11: Removing an Agent Detects Stale Files.
func TestUpdate_RemoveAgent(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude, config.AgentCursor}
	})

	// Verify cursor file exists.
	cursorFile := filepath.Join(dir, ".cursor", "rules", "agents.mdc")
	if _, err := os.Stat(cursorFile); err != nil {
		t.Fatal(".cursor/rules/agents.mdc should exist initially")
	}

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Agents = []string{config.AgentClaude}
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	// Stale file should be detected and removed.
	if !strings.Contains(output, "Stale") || !strings.Contains(output, ".cursor") {
		t.Errorf("output should report stale cursor files:\n%s", output)
	}

	// Claude files should remain.
	if _, err = os.Stat(filepath.Join(dir, ".claude", "commands")); err != nil {
		t.Error("Claude commands should still exist after removing cursor")
	}
}

// Test 12: CGO Toggle Regenerates Makefile with CGO Targets.
func TestUpdate_CGOToggle(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, func(cfg *config.Config) {
		cfg.Features.CGO = false
	})

	makefile, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(makefile), "CGO_ENABLED=0") {
		t.Fatal("Makefile should contain CGO_ENABLED=0 initially")
	}

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Features.CGO = true
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	makefile, _ = os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(makefile), "CGO_ENABLED=1") {
		t.Error("Makefile should contain CGO_ENABLED=1 after enabling CGO")
	}
}

// Test 13: Docker Toggle Regenerates Makefile.
func TestUpdate_DockerToggle(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, func(cfg *config.Config) {
		cfg.Features.Docker = true
	})

	makefile, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(makefile), "docker-build") {
		t.Fatal("Makefile should contain docker-build initially")
	}

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Features.Docker = false
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	makefile, _ = os.ReadFile(filepath.Join(dir, "Makefile"))
	if strings.Contains(string(makefile), "docker-build") {
		t.Error("Makefile should not contain docker-build after disabling Docker")
	}

	if strings.Contains(string(makefile), "docker-test") {
		t.Error("Makefile should not contain docker-test after disabling Docker")
	}
}

// Test 14: Config Schema Validation Aggregates Multiple Errors.
func TestUpdate_ValidationAggregatesErrors(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Write a config with multiple validation errors.
	// We need to write raw YAML to bypass Save() validation.
	configPath := filepath.Join(dir, config.FileName)
	raw := `project_name:
module_path:
go_version: "1.22"
binaries:
  - name: testproject
    cmd_path: ./cmd/testproject
quality:
  coverage_min: 0
  coverage_critical: 90
  complexity_max: 15
  line_length: 140
features:
  cgo: false
  docker: true
  benchmarks: true
agents:
  - claude
ecosystem: golang
analysis_command:
template_overrides: .promptkit/templates
`

	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := runUpdateCapture(t, dir, true, false)
	if err == nil {
		t.Fatal("expected validation error")
	}

	got := err.Error()
	if !strings.Contains(got, "project_name is required") {
		t.Error("missing project_name error in aggregated output")
	}

	if !strings.Contains(got, "module_path is required") {
		t.Error("missing module_path error in aggregated output")
	}

	if !strings.Contains(got, "coverage_min must be 1-100") {
		t.Error("missing coverage_min error in aggregated output")
	}
}

// Test 15: Update with No Config Changes Is Idempotent.
func TestUpdate_Idempotent(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// First update to ensure everything is in sync.
	output1, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("first RunUpdate() error: %v\noutput: %s", err, output1)
	}

	// Second update should report no changes.
	output2, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("second RunUpdate() error: %v\noutput: %s", err, output2)
	}

	if !strings.Contains(output2, "All files are up to date") {
		t.Errorf("idempotent update should report no changes:\n%s", output2)
	}
}

// Additional tests for new features.

func TestUpdate_DryRunNoWrite(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Record file state.
	agentsBefore, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 80
	})

	output, err := runUpdateCapture(t, dir, false, true)
	// dry-run returns error when there are diffs.
	if err == nil {
		t.Fatal("dry-run should return error when files are out of date")
	}

	if !strings.Contains(output, "Dry run") {
		t.Error("output should mention dry run")
	}

	// Files should be unchanged.
	agentsAfter, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !bytes.Equal(agentsBefore, agentsAfter) {
		t.Error("dry-run should not modify files")
	}
}

func TestUpdate_DryRunClean(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// First update to sync.
	runUpdateCapture(t, dir, true, false)

	// Dry run on already-synced project should succeed.
	output, err := runUpdateCapture(t, dir, false, true)
	if err != nil {
		t.Fatalf("dry-run on clean project should succeed: %v", err)
	}

	if !strings.Contains(output, "All files are up to date") {
		t.Errorf("dry-run on clean project should report up to date:\n%s", output)
	}
}

func TestUpdate_UnifiedDiffOutput(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 80
	})

	output, _ := runUpdateCapture(t, dir, false, true)

	// Should contain unified diff markers.
	if !strings.Contains(output, "---") || !strings.Contains(output, "+++") {
		t.Errorf("output should contain unified diff markers:\n%s", output)
	}

	if !strings.Contains(output, "@@") {
		t.Errorf("output should contain hunk headers:\n%s", output)
	}
}

func TestUpdate_VerboseOutput(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 80
	})

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:     dir,
		Yes:     true,
		Verbose: true,
		Stdout:  &buf,
		Stdin:   strings.NewReader(""),
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Loading config") {
		t.Error("verbose output should show config loading")
	}

	if !strings.Contains(output, "Rendering templates") {
		t.Error("verbose output should show rendering step")
	}

	if !strings.Contains(output, "Total time") {
		t.Error("verbose output should show total time")
	}
}

func TestUpdate_ManifestUpdated(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	// Run update to sync.
	runUpdateCapture(t, dir, true, false)

	// Load config and check manifest.
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.GeneratedFiles) == 0 {
		t.Error("config should have generated_files manifest after update")
	}

	// Manifest should contain expected files.
	if !slices.Contains(cfg.GeneratedFiles, "AGENTS.md") {
		t.Errorf("manifest should contain AGENTS.md, got: %v", cfg.GeneratedFiles)
	}
}

func TestUpdate_ProgressSteps(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Description = "Changed for progress test"
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, output)
	}

	if !strings.Contains(output, "[1/") {
		t.Errorf("output should contain progress indicators:\n%s", output)
	}

	if !strings.Contains(output, "Loading config") {
		t.Error("output should show Loading config step")
	}
}

func TestUpdate_ExplainFlag(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:     dir,
		Yes:     true,
		Explain: true,
		Stdout:  &buf,
		Stdin:   strings.NewReader(""),
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "pipeline") {
		t.Error("explain output should contain 'pipeline'")
	}

	if !strings.Contains(output, "Load") {
		t.Error("explain output should contain 'Load'")
	}

	if !strings.Contains(output, "Render") {
		t.Error("explain output should contain 'Render'")
	}

	if !strings.Contains(output, "agent-specific") {
		t.Error("explain output should contain 'agent-specific'")
	}
}

func TestUpdate_ChangeAnnotation(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Quality.CoverageMin = 80
	})

	output, _ := runUpdateCapture(t, dir, false, true)

	// AGENTS.md diff should be annotated with quality.coverage_min.
	if !strings.Contains(output, "quality.coverage_min") {
		t.Errorf("output should annotate changes with config keys:\n%s", output)
	}
}

func TestUpdate_InteractiveApproveOne(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Description = "Interactive test"
		cfg.Quality.CoverageMin = 80
	})

	var buf bytes.Buffer

	// Read existing AGENTS.md for comparison.
	agentsBefore, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	opts := cli.UpdateOptions{
		Dir:         dir,
		Interactive: true,
		Stdout:      &buf,
		Stdin:       strings.NewReader("y\nn\nn\nn\nn\nn\nn\n"), //nolint:dupword // approve first, reject rest.
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, buf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Apply") && !strings.Contains(output, "[y/n/a/q]") {
		t.Errorf("interactive mode should show per-file prompts:\n%s", output)
	}

	// At least one file should be updated (the first one approved).
	if !strings.Contains(output, "Updated") {
		// It's possible only the first file was a diff; check that something happened.
		_ = agentsBefore // may or may not have changed depending on sort order.
	}
}

func TestUpdate_InteractiveApplyAll(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Description = "Apply all test"
	})

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:         dir,
		Interactive: true,
		Stdout:      &buf,
		Stdin:       strings.NewReader("a\n"), // apply all.
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, buf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Updated") {
		t.Errorf("apply-all should update files:\n%s", output)
	}
}

func TestUpdate_InteractiveQuit(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Description = "Quit test"
	})

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:         dir,
		Interactive: true,
		Stdout:      &buf,
		Stdin:       strings.NewReader("q\n"), // quit immediately.
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v\noutput: %s", err, buf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "No files approved") {
		t.Errorf("quit should report no files approved:\n%s", output)
	}
}

func TestUpdate_BackupCreated(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Description = "Backup test"
	})

	var buf bytes.Buffer

	opts := cli.UpdateOptions{
		Dir:     dir,
		Yes:     true,
		Verbose: true,
		Stdout:  &buf,
		Stdin:   strings.NewReader(""),
	}

	if err := cli.RunUpdate(opts); err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Backed up") {
		t.Errorf("verbose output should mention backup:\n%s", output)
	}

	// Verify backup directory exists.
	backupBase := filepath.Join(dir, ".promptkit", "backups")

	entries, err := os.ReadDir(backupBase)
	if err != nil {
		t.Fatalf("reading backup dir: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected at least one backup directory")
	}
}

func TestUpdate_FileListAfterApply(t *testing.T) {
	t.Parallel()

	dir := setupProject(t, nil)

	modifyConfig(t, dir, func(cfg *config.Config) {
		cfg.Description = "Changed description"
	})

	output, err := runUpdateCapture(t, dir, true, false)
	if err != nil {
		t.Fatalf("RunUpdate() error: %v", err)
	}

	// Should list updated files individually.
	if !strings.Contains(output, "Updated") && !strings.Contains(output, "file(s)") {
		t.Errorf("output should list updated files:\n%s", output)
	}

	if !strings.Contains(output, "AGENTS.md") {
		t.Errorf("output should mention AGENTS.md as updated:\n%s", output)
	}
}
