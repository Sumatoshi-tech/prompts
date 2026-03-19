package scaffold_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	promptkit "github.com/Sumatoshi-tech/prompts"
	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/scaffold"
)

func TestRender_SimpleTemplate(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}\n\n{{.Description}}\n"),
		},
	}

	cfg := testConfig()

	result, err := scaffold.Render(cfg, tmplFS)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	content, ok := result["README.md"]
	if !ok {
		t.Fatal("expected README.md in rendered output")
	}

	got := string(content)
	if !strings.Contains(got, "testproject") {
		t.Errorf("rendered content missing project name: %s", got)
	}

	if !strings.Contains(got, "A test project") {
		t.Errorf("rendered content missing description: %s", got)
	}
}

func TestRender_NonTemplateFile(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/static.txt": &fstest.MapFile{
			Data: []byte("no templates here"),
		},
	}

	cfg := testConfig()

	result, err := scaffold.Render(cfg, tmplFS)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	content, ok := result["static.txt"]
	if !ok {
		t.Fatal("expected static.txt in rendered output")
	}

	if string(content) != "no templates here" {
		t.Errorf("static file content changed: %s", string(content))
	}
}

func TestRender_NestedPaths(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/.agents/instructions/instr-frd.md.tmpl": &fstest.MapFile{
			Data: []byte("# FRD Template"),
		},
	}

	cfg := testConfig()

	result, err := scaffold.Render(cfg, tmplFS)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if _, ok := result[".agents/instructions/instr-frd.md"]; !ok {
		t.Error("expected .agents/instructions/instr-frd.md in output")
	}
}

func TestRender_ConditionalCGO(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/test.txt.tmpl": &fstest.MapFile{
			Data: []byte("base{{if .Features.CGO}}\ncgo enabled{{end}}\n"),
		},
	}

	t.Run("cgo disabled", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		cfg.Features.CGO = false

		result, err := scaffold.Render(cfg, tmplFS)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}

		got := string(result["test.txt"])
		if strings.Contains(got, "cgo enabled") {
			t.Error("CGO section should not appear when CGO is disabled")
		}
	})

	t.Run("cgo enabled", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		cfg.Features.CGO = true

		result, err := scaffold.Render(cfg, tmplFS)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}

		got := string(result["test.txt"])
		if !strings.Contains(got, "cgo enabled") {
			t.Error("CGO section should appear when CGO is enabled")
		}
	})
}

func TestRender_BinaryIteration(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/bins.txt.tmpl": &fstest.MapFile{
			Data: []byte("{{range .Binaries}}{{.Name}} {{end}}"),
		},
	}

	cfg := testConfig()
	cfg.Binaries = []config.Binary{
		{Name: "alpha", CmdPath: "./cmd/alpha"},
		{Name: "beta", CmdPath: "./cmd/beta"},
	}

	result, err := scaffold.Render(cfg, tmplFS)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	got := string(result["bins.txt"])
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("expected both binary names, got: %s", got)
	}
}

func TestApply_ModeCreate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Pre-create a file that should NOT be overwritten.
	existingPath := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	rendered := map[string][]byte{
		"existing.txt": []byte("overwritten"),
		"new.txt":      []byte("new content"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeCreate); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// existing.txt should remain unchanged.
	got, _ := os.ReadFile(existingPath)
	if string(got) != "original" {
		t.Errorf("existing file was overwritten: %s", string(got))
	}

	// new.txt should be created.
	newPath := filepath.Join(dir, "new.txt")

	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("new file not created: %v", err)
	}

	if string(got) != "new content" {
		t.Errorf("new file content = %q, want %q", string(got), "new content")
	}
}

func TestApply_ModeForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	existingPath := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	rendered := map[string][]byte{
		"existing.txt": []byte("overwritten"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, _ := os.ReadFile(existingPath)
	if string(got) != "overwritten" {
		t.Errorf("existing file should be overwritten, got: %s", string(got))
	}
}

func TestApply_CreatesSubdirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	rendered := map[string][]byte{
		"a/b/c/deep.txt": []byte("deep content"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeCreate); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("deep file not created: %v", err)
	}

	if string(got) != "deep content" {
		t.Errorf("content = %q, want %q", string(got), "deep content")
	}
}

func TestApply_ShellScriptPermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	rendered := map[string][]byte{
		"scripts/run.sh": []byte("#!/bin/bash\necho hello"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeCreate); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "scripts", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm&0o111 == 0 {
		t.Errorf("shell script should be executable, got permissions: %o", perm)
	}
}

func TestDiff_NewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	rendered := map[string][]byte{
		"new.txt": []byte("new content"),
	}

	diffs, err := scaffold.Diff(rendered, dir)
	if err != nil {
		t.Fatalf("Diff() error: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	if !diffs[0].IsNew {
		t.Error("expected diff to be marked as new")
	}
}

func TestDiff_ModifiedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	rendered := map[string][]byte{
		"file.txt": []byte("new"),
	}

	diffs, err := scaffold.Diff(rendered, dir)
	if err != nil {
		t.Fatalf("Diff() error: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	if diffs[0].IsNew {
		t.Error("diff should not be marked as new")
	}

	if string(diffs[0].Existing) != "old" {
		t.Errorf("existing = %q, want %q", string(diffs[0].Existing), "old")
	}
}

func TestDiff_IdenticalFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := []byte("same content")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	rendered := map[string][]byte{
		"file.txt": content,
	}

	diffs, err := scaffold.Diff(rendered, dir)
	if err != nil {
		t.Fatalf("Diff() error: %v", err)
	}

	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical file, got %d", len(diffs))
	}
}

func TestRenderWithOverrides(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/base.txt.tmpl": &fstest.MapFile{
			Data: []byte("base: {{.ProjectName}}"),
		},
	}

	overrideDir := t.TempDir()
	overridePath := filepath.Join(overrideDir, "base.txt.tmpl")

	if err := os.WriteFile(overridePath, []byte("override: {{.ProjectName}}"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()

	result, err := scaffold.RenderWithOverrides(cfg, tmplFS, overrideDir)
	if err != nil {
		t.Fatalf("RenderWithOverrides() error: %v", err)
	}

	got := string(result["base.txt"])
	if !strings.Contains(got, "override:") {
		t.Errorf("override should take precedence, got: %s", got)
	}
}

func TestUnifiedDiff_NoDifference(t *testing.T) {
	t.Parallel()

	old := []byte("line1\nline2\nline3\n")
	result := scaffold.UnifiedDiff(old, old, "test.txt")

	if result != "" {
		t.Errorf("expected empty diff for identical content, got:\n%s", result)
	}
}

func TestUnifiedDiff_AddedLines(t *testing.T) {
	t.Parallel()

	old := []byte("line1\nline3\n")
	updated := []byte("line1\nline2\nline3\n")

	result := scaffold.UnifiedDiff(old, updated, "test.txt")
	if result == "" {
		t.Fatal("expected non-empty diff")
	}

	if !strings.Contains(result, "--- a/test.txt") {
		t.Error("missing old file header")
	}

	if !strings.Contains(result, "+++ b/test.txt") {
		t.Error("missing new file header")
	}

	if !strings.Contains(result, "+line2") {
		t.Error("missing added line marker")
	}

	if !strings.Contains(result, "@@") {
		t.Error("missing hunk header")
	}
}

func TestUnifiedDiff_RemovedLines(t *testing.T) {
	t.Parallel()

	old := []byte("line1\nline2\nline3\n")
	updated := []byte("line1\nline3\n")

	result := scaffold.UnifiedDiff(old, updated, "test.txt")
	if result == "" {
		t.Fatal("expected non-empty diff")
	}

	if !strings.Contains(result, "-line2") {
		t.Error("missing removed line marker")
	}
}

func TestUnifiedDiff_ModifiedLines(t *testing.T) {
	t.Parallel()

	old := []byte("hello world\n")
	updated := []byte("hello universe\n")

	result := scaffold.UnifiedDiff(old, updated, "greeting.txt")
	if result == "" {
		t.Fatal("expected non-empty diff")
	}

	if !strings.Contains(result, "-hello world") {
		t.Error("missing old line")
	}

	if !strings.Contains(result, "+hello universe") {
		t.Error("missing new line")
	}
}

func TestUnifiedDiff_EmptyOld(t *testing.T) {
	t.Parallel()

	updated := []byte("line1\nline2\n")

	result := scaffold.UnifiedDiff(nil, updated, "new.txt")
	if result == "" {
		t.Fatal("expected non-empty diff for new file")
	}

	if !strings.Contains(result, "+line1") {
		t.Error("missing added line")
	}
}

func TestDetectStale_NoStale(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"a.txt": []byte("a"),
		"b.txt": []byte("b"),
	}
	manifest := []string{"a.txt", "b.txt"}

	stale := scaffold.DetectStale(rendered, manifest)
	if len(stale) != 0 {
		t.Errorf("expected no stale files, got %v", stale)
	}
}

func TestDetectStale_HasStale(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"a.txt": []byte("a"),
	}
	manifest := []string{"a.txt", "b.txt", "c.txt"}

	stale := scaffold.DetectStale(rendered, manifest)
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale files, got %d: %v", len(stale), stale)
	}

	// Should be sorted.
	if stale[0] != "b.txt" || stale[1] != "c.txt" {
		t.Errorf("stale = %v, want [b.txt, c.txt]", stale)
	}
}

func TestDetectStale_EmptyManifest(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"a.txt": []byte("a"),
	}

	stale := scaffold.DetectStale(rendered, nil)
	if len(stale) != 0 {
		t.Errorf("expected no stale files with empty manifest, got %v", stale)
	}
}

func TestRemoveFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create files to remove.
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := scaffold.RemoveFiles(dir, []string{"a.txt", "nonexistent.txt"}); err != nil {
		t.Fatalf("RemoveFiles() error: %v", err)
	}

	// a.txt should be gone.
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Error("a.txt should have been removed")
	}

	// b.txt should remain.
	if _, err := os.Stat(filepath.Join(dir, "b.txt")); err != nil {
		t.Error("b.txt should not have been removed")
	}
}

func TestFileManifest(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"c.txt": []byte("c"),
		"a.txt": []byte("a"),
		"b.txt": []byte("b"),
	}

	manifest := scaffold.FileManifest(rendered)
	if len(manifest) != 3 {
		t.Fatalf("expected 3 files, got %d", len(manifest))
	}

	// Should be sorted.
	if manifest[0] != "a.txt" || manifest[1] != "b.txt" || manifest[2] != "c.txt" {
		t.Errorf("manifest not sorted: %v", manifest)
	}
}

func TestComputeChecksums(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"a.txt": []byte("hello"),
		"b.txt": []byte("world"),
	}

	checksums := scaffold.ComputeChecksums(rendered)
	if len(checksums) != 2 {
		t.Fatalf("expected 2 checksums, got %d", len(checksums))
	}

	// Same content should produce same checksum.
	rendered2 := map[string][]byte{
		"a.txt": []byte("hello"),
	}

	checksums2 := scaffold.ComputeChecksums(rendered2)
	if checksums["a.txt"] != checksums2["a.txt"] {
		t.Error("same content should produce same checksum")
	}

	// Different content should produce different checksum.
	if checksums["a.txt"] == checksums["b.txt"] {
		t.Error("different content should produce different checksums")
	}

	// Checksums should be hex strings of expected length (SHA-256 = 64 hex chars).
	for path, sum := range checksums {
		if len(sum) != 64 {
			t.Errorf("checksum for %s has length %d, want 64", path, len(sum))
		}
	}
}

func TestRenderWithOverrides_ErrorIdentifiesFile(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/base.txt.tmpl": &fstest.MapFile{
			Data: []byte("base: {{.ProjectName}}"),
		},
	}

	overrideDir := t.TempDir()
	overridePath := filepath.Join(overrideDir, "broken.txt.tmpl")

	// Write a template with invalid syntax.
	if err := os.WriteFile(overridePath, []byte("{{.Invalid.{{Bad}}"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()

	_, err := scaffold.RenderWithOverrides(cfg, tmplFS, overrideDir)
	if err == nil {
		t.Fatal("expected error for broken override template")
	}

	// Error should include the full path.
	if !strings.Contains(err.Error(), overrideDir) {
		t.Errorf("error = %q, want to contain override dir %q", err.Error(), overrideDir)
	}
}

func TestRenderSingle_Embedded(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}"),
		},
	}

	cfg := testConfig()

	result, err := scaffold.RenderSingle(cfg, tmplFS, "", "README.md")
	if err != nil {
		t.Fatalf("RenderSingle() error: %v", err)
	}

	if !strings.Contains(string(result), "testproject") {
		t.Errorf("rendered content missing project name: %s", string(result))
	}
}

func TestRenderSingle_Override(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# Embedded {{.ProjectName}}"),
		},
	}

	overrideDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(overrideDir, "README.md.tmpl"),
		[]byte("# Override {{.ProjectName}}"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()

	result, err := scaffold.RenderSingle(cfg, tmplFS, overrideDir, "README.md")
	if err != nil {
		t.Fatalf("RenderSingle() error: %v", err)
	}

	if !strings.Contains(string(result), "Override") {
		t.Errorf("expected override content, got: %s", string(result))
	}
}

func TestRenderSingle_NotFound(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{}

	cfg := testConfig()

	_, err := scaffold.RenderSingle(cfg, tmplFS, "", "nonexistent.md")
	if err == nil {
		t.Fatal("expected error for missing template")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestBackupFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create files to back up.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("content-a"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("content-b"), 0o600); err != nil {
		t.Fatal(err)
	}

	backupDir, err := scaffold.BackupFiles(dir, []string{"a.txt", "b.txt"})
	if err != nil {
		t.Fatalf("BackupFiles() error: %v", err)
	}

	if backupDir == "" {
		t.Fatal("expected non-empty backup dir")
	}

	// Verify backup contents.
	data, err := os.ReadFile(filepath.Join(backupDir, "a.txt"))
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}

	if string(data) != "content-a" {
		t.Errorf("backup content = %q, want %q", string(data), "content-a")
	}
}

func TestBackupFiles_NewFilesSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// No files exist on disk — should skip gracefully.
	backupDir, err := scaffold.BackupFiles(dir, []string{"new.txt"})
	if err != nil {
		t.Fatalf("BackupFiles() error: %v", err)
	}

	if backupDir != "" {
		t.Errorf("expected empty backup dir for nonexistent files, got %q", backupDir)
	}
}

func TestRestoreBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")

	// Create a backup.
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(backupDir, "file.txt"), []byte("backed up"), 0o600); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(dir, "target")
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := scaffold.RestoreBackup(backupDir, targetDir); err != nil {
		t.Fatalf("RestoreBackup() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "file.txt"))
	if err != nil {
		t.Fatalf("reading restored file: %v", err)
	}

	if string(data) != "backed up" {
		t.Errorf("restored content = %q, want %q", string(data), "backed up")
	}
}

func TestApply_Atomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	rendered := map[string][]byte{
		"test.txt": []byte("atomic content"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// Verify file exists with correct content.
	data, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if string(data) != "atomic content" {
		t.Errorf("content = %q, want %q", string(data), "atomic content")
	}

	// Verify no temp files left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".promptkit-") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestProvenance_MarkdownFiles(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"AGENTS.md": []byte("# Agent\nContent"),
	}

	result := scaffold.AddProvenance(rendered)

	s := string(result["AGENTS.md"])
	if !strings.Contains(s, "Generated by promptkit") {
		t.Error("AGENTS.md: missing provenance comment")
	}

	if !strings.HasPrefix(s, "<!--") {
		t.Errorf("AGENTS.md: markdown file should use HTML comment, got: %s", s[:min(40, len(s))])
	}
}

func TestProvenance_SkipsFrontmatter(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"rules/r.mdc":                  []byte("---\nalwaysApply: true\n---\nContent"),
		".agents/skills/impl/SKILL.md": []byte("---\nname: impl\n---\nBody"),
	}

	result := scaffold.AddProvenance(rendered)

	for path, content := range result {
		s := string(content)
		if strings.Contains(s, "Generated by promptkit") {
			t.Errorf("%s: files with YAML frontmatter should NOT have provenance", path)
		}

		if !strings.HasPrefix(s, "---\n") {
			t.Errorf("%s: frontmatter should be preserved, got: %s", path, s[:min(20, len(s))])
		}
	}
}

func TestProvenance_YAMLFiles(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		".golangci.yml": []byte("version: 2"),
		"config.yaml":   []byte("key: value"),
	}

	result := scaffold.AddProvenance(rendered)

	for path, content := range result {
		s := string(content)
		if !strings.HasPrefix(s, "# Generated by promptkit") {
			t.Errorf("%s: YAML file should use # comment, got: %s", path, s[:min(40, len(s))])
		}
	}
}

func TestProvenance_ShellAndMakefile(t *testing.T) {
	t.Parallel()

	rendered := map[string][]byte{
		"scripts/run.sh": []byte("#!/bin/bash\necho hello"),
		"Makefile":       []byte("all: build"),
		"config.toml":    []byte("[section]\nkey = val"),
	}

	result := scaffold.AddProvenance(rendered)

	for path, content := range result {
		s := string(content)
		if !strings.HasPrefix(s, "# Generated by promptkit") {
			t.Errorf("%s: should use # comment, got: %s", path, s[:min(40, len(s))])
		}
	}
}

func TestProvenance_PreservesOriginalContent(t *testing.T) {
	t.Parallel()

	original := []byte("# My Content\nHello world")
	rendered := map[string][]byte{
		"test.md": original,
	}

	result := scaffold.AddProvenance(rendered)
	s := string(result["test.md"])

	if !strings.Contains(s, "# My Content\nHello world") {
		t.Error("provenance should preserve original content")
	}
}

func TestDiffRendered_Identical(t *testing.T) {
	t.Parallel()

	local := map[string][]byte{"a.txt": []byte("same")}
	upstream := map[string][]byte{"a.txt": []byte("same")}

	diffs := scaffold.DiffRendered(local, upstream)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical maps, got %d", len(diffs))
	}
}

func TestDiffRendered_Modified(t *testing.T) {
	t.Parallel()

	local := map[string][]byte{"a.txt": []byte("new")}
	upstream := map[string][]byte{"a.txt": []byte("old")}

	diffs := scaffold.DiffRendered(local, upstream)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	if diffs[0].IsNew {
		t.Error("modified file should not be marked as new")
	}

	if string(diffs[0].Existing) != "old" {
		t.Errorf("existing = %q, want %q", string(diffs[0].Existing), "old")
	}

	if string(diffs[0].Rendered) != "new" {
		t.Errorf("rendered = %q, want %q", string(diffs[0].Rendered), "new")
	}
}

func TestDiffRendered_OnlyInLocal(t *testing.T) {
	t.Parallel()

	local := map[string][]byte{"a.txt": []byte("new"), "b.txt": []byte("local only")}
	upstream := map[string][]byte{"a.txt": []byte("new")}

	diffs := scaffold.DiffRendered(local, upstream)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	if !diffs[0].IsNew {
		t.Error("file only in local should be marked as new")
	}

	if diffs[0].Path != "b.txt" {
		t.Errorf("path = %q, want %q", diffs[0].Path, "b.txt")
	}
}

func TestDiffRendered_OnlyInUpstream(t *testing.T) {
	t.Parallel()

	local := map[string][]byte{"a.txt": []byte("shared")}
	upstream := map[string][]byte{"a.txt": []byte("shared"), "c.txt": []byte("upstream only")}

	diffs := scaffold.DiffRendered(local, upstream)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	if diffs[0].Path != "c.txt" {
		t.Errorf("path = %q, want %q", diffs[0].Path, "c.txt")
	}

	// File only in upstream should have Existing but no Rendered.
	if len(diffs[0].Rendered) != 0 {
		t.Error("file only in upstream should have empty Rendered")
	}

	if string(diffs[0].Existing) != "upstream only" {
		t.Errorf("existing = %q, want %q", string(diffs[0].Existing), "upstream only")
	}
}

func TestDiffRendered_Sorted(t *testing.T) {
	t.Parallel()

	local := map[string][]byte{
		"c.txt": []byte("c"),
		"a.txt": []byte("a"),
		"b.txt": []byte("b"),
	}
	upstream := map[string][]byte{}

	diffs := scaffold.DiffRendered(local, upstream)
	if len(diffs) != 3 {
		t.Fatalf("expected 3 diffs, got %d", len(diffs))
	}

	if diffs[0].Path != "a.txt" || diffs[1].Path != "b.txt" || diffs[2].Path != "c.txt" {
		t.Errorf("diffs not sorted: %s, %s, %s", diffs[0].Path, diffs[1].Path, diffs[2].Path)
	}
}

func TestCheckOverrideStaleness_NoChange(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/test.md.tmpl": &fstest.MapFile{
			Data: []byte("# Original content"),
		},
	}

	dir := t.TempDir()
	overrideDir := filepath.Join(dir, "templates")
	os.MkdirAll(overrideDir, 0o750)

	// Save checksum matching the current embedded template.
	scaffold.SaveOverrideChecksum(overrideDir, "test.md.tmpl", []byte("# Original content"))

	stale := scaffold.CheckOverrideStaleness(tmplFS, overrideDir, "golang")
	if len(stale) != 0 {
		t.Errorf("expected no stale overrides, got %v", stale)
	}
}

func TestCheckOverrideStaleness_DetectsChange(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/test.md.tmpl": &fstest.MapFile{
			Data: []byte("# Updated content"),
		},
	}

	dir := t.TempDir()
	overrideDir := filepath.Join(dir, "templates")
	os.MkdirAll(overrideDir, 0o750)

	// Save checksum for old version of the embedded template.
	scaffold.SaveOverrideChecksum(overrideDir, "test.md.tmpl", []byte("# Old content"))

	stale := scaffold.CheckOverrideStaleness(tmplFS, overrideDir, "golang")
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale override, got %d", len(stale))
	}

	if stale[0] != "test.md.tmpl" {
		t.Errorf("stale = %q, want %q", stale[0], "test.md.tmpl")
	}
}

func TestCheckOverrideStaleness_EmptyDir(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{}

	stale := scaffold.CheckOverrideStaleness(tmplFS, "", "golang")
	if len(stale) != 0 {
		t.Errorf("expected no stale overrides for empty dir, got %v", stale)
	}
}

func TestTemplateDirForEcosystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ecosystem string
		want      string
	}{
		{"golang", "templates/golang"},
		{"rust", "templates/rust"},
		{"zig", "templates/zig"},
	}

	for _, tt := range tests {
		t.Run(tt.ecosystem, func(t *testing.T) {
			t.Parallel()

			got := scaffold.TemplateDirForEcosystem(tt.ecosystem)
			if got != tt.want {
				t.Errorf("TemplateDirForEcosystem(%q) = %q, want %q", tt.ecosystem, got, tt.want)
			}
		})
	}
}

func TestRender_RustEcosystem(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/rust/Cargo.toml.tmpl": &fstest.MapFile{
			Data: []byte("[package]\nname = \"{{.ProjectName}}\"\n"),
		},
	}

	cfg := testConfig()
	cfg.Ecosystem = "rust"

	result, err := scaffold.Render(cfg, tmplFS)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	content, ok := result["Cargo.toml"]
	if !ok {
		t.Fatal("expected Cargo.toml in rendered output")
	}

	if !strings.Contains(string(content), "testproject") {
		t.Errorf("rendered content missing project name: %s", string(content))
	}
}

func TestRender_ZigEcosystem(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/zig/build.zig.tmpl": &fstest.MapFile{
			Data: []byte("// {{.ProjectName}} build\n"),
		},
	}

	cfg := testConfig()
	cfg.Ecosystem = "zig"

	result, err := scaffold.Render(cfg, tmplFS)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	content, ok := result["build.zig"]
	if !ok {
		t.Fatal("expected build.zig in rendered output")
	}

	if !strings.Contains(string(content), "testproject") {
		t.Errorf("rendered content missing project name: %s", string(content))
	}
}

func TestRenderSingle_RustEcosystem(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/rust/Cargo.toml.tmpl": &fstest.MapFile{
			Data: []byte("[package]\nname = \"{{.ProjectName}}\"\n"),
		},
	}

	cfg := testConfig()
	cfg.Ecosystem = "rust"

	result, err := scaffold.RenderSingle(cfg, tmplFS, "", "Cargo.toml")
	if err != nil {
		t.Fatalf("RenderSingle() error: %v", err)
	}

	if !strings.Contains(string(result), "testproject") {
		t.Errorf("rendered content missing project name: %s", string(result))
	}
}

func testConfig() *config.Config {
	cfg := config.Default()
	cfg.ProjectName = "testproject"
	cfg.ModulePath = "github.com/user/testproject"
	cfg.Description = "A test project"
	cfg.Expertise = "testing"
	cfg.Binaries = []config.Binary{
		{Name: "testproject", CmdPath: "./cmd/testproject"},
	}

	return cfg
}

// RenderFull tests.

func TestRenderFull_BasicGolang(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Agents = []string{config.AgentClaude}
	cfg.Workflow = config.WorkflowFRD

	result, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	// Agent Skills SKILL.md files should be present (FRD/journey stay under .agents/instructions/).
	expectedSkills := []string{
		".agents/skills/implement/SKILL.md",
		".agents/skills/roadmap/SKILL.md",
		".agents/skills/perf/SKILL.md",
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

	// Claude legacy commands should be present.
	for _, name := range []string{"implement", "roadmap", "perf"} {
		path := ".claude/commands/" + name + ".md"
		if _, ok := result[path]; !ok {
			t.Errorf("missing Claude command: %s", path)
		}
	}

	// Skill-backed instructions removed; workflow template remains on disk.
	removedPaths := []string{
		".agents/instructions/instr-implement.md",
		".agents/instructions/instr-roadmaper.md",
		".agents/instructions/instr-perf.md",
	}

	for _, path := range removedPaths {
		if _, ok := result[path]; ok {
			t.Errorf("skill instruction file should be removed: %s", path)
		}
	}

	if _, ok := result[".agents/instructions/instr-frd.md"]; !ok {
		t.Error("expected .agents/instructions/instr-frd.md to remain for frd workflow")
	}

	if _, ok := result[".agents/instructions/instr-journey.md"]; ok {
		t.Error("instr-journey.md should not be shipped for frd workflow")
	}

	// Non-instruction base files should still be present.
	for _, path := range []string{"AGENTS.md", ".golangci.yml", "Makefile"} {
		if _, ok := result[path]; !ok {
			t.Errorf("missing expected base file: %s", path)
		}
	}

	// All files should have provenance or frontmatter.
	for path, content := range result {
		s := string(content)
		hasProvenance := strings.Contains(s, "Generated by promptkit")

		hasFrontmatter := strings.HasPrefix(s, "---\n")
		if !hasProvenance && !hasFrontmatter {
			t.Errorf("file %s has neither provenance comment nor frontmatter", path)
		}
	}
}

func TestRenderFull_JourneyWorkflow(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Agents = []string{config.AgentClaude}
	cfg.Workflow = config.WorkflowJourney

	result, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	// Journey template on disk; no separate journey skill.
	if _, ok := result[".agents/instructions/instr-journey.md"]; !ok {
		t.Error("expected .agents/instructions/instr-journey.md for journey workflow")
	}

	if _, ok := result[".agents/instructions/instr-frd.md"]; ok {
		t.Error("instr-frd.md should not be shipped for journey workflow")
	}

	if _, ok := result[".agents/skills/journey/SKILL.md"]; ok {
		t.Error("journey should not be an Agent Skill")
	}

	if _, ok := result[".agents/skills/frd/SKILL.md"]; ok {
		t.Error("frd should not be an Agent Skill")
	}

	// Base skills should still be present.
	for _, name := range []string{"implement", "roadmap", "perf"} {
		path := ".agents/skills/" + name + "/SKILL.md"
		if _, ok := result[path]; !ok {
			t.Errorf("missing base skill: %s", path)
		}
	}
}

func TestRenderFull_NoAgents(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}"),
		},
	}

	cfg := testConfig()
	cfg.Agents = nil

	result, err := scaffold.RenderFull(cfg, tmplFS)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	// With no agents, rendered output should just contain the template files
	// plus provenance, and no agent-specific files.
	if _, ok := result["README.md"]; !ok {
		t.Error("expected README.md in output")
	}

	// No agent-specific directories should be present.
	for path := range result {
		if strings.HasPrefix(path, ".agents/") ||
			strings.HasPrefix(path, ".claude/") ||
			strings.HasPrefix(path, ".cursor/") {
			t.Errorf("unexpected agent file when agents is empty: %s", path)
		}
	}
}

func TestRenderFull_MultipleAgents(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Agents = []string{config.AgentClaude, config.AgentCursor, config.AgentGemini}
	cfg.Workflow = config.WorkflowFRD

	result, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		t.Fatalf("RenderFull() error: %v", err)
	}

	// Claude-specific.
	if _, ok := result[".claude/commands/implement.md"]; !ok {
		t.Error("missing Claude command file")
	}

	// Cursor-specific.
	if _, ok := result[".cursor/rules/agents.mdc"]; !ok {
		t.Error("missing Cursor rules file")
	}

	// Gemini-specific.
	if _, ok := result["GEMINI.md"]; !ok {
		t.Error("missing GEMINI.md")
	}

	if _, ok := result[".gemini/commands/implement.toml"]; !ok {
		t.Error("missing Gemini command file")
	}
}

// RenderFullWithOverrides tests.

func TestRenderFullWithOverrides_Basic(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# Embedded {{.ProjectName}}"),
		},
		"templates/golang/.agents/instructions/instr-implement.md.tmpl": &fstest.MapFile{
			Data: []byte("implement instruction for {{.ProjectName}}"),
		},
	}

	overrideDir := t.TempDir()

	// Create an override for README.md.
	if err := os.WriteFile(
		filepath.Join(overrideDir, "README.md.tmpl"),
		[]byte("# Overridden {{.ProjectName}}"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()
	cfg.Agents = []string{config.AgentClaude}
	cfg.Workflow = config.WorkflowFRD

	result, err := scaffold.RenderFullWithOverrides(cfg, tmplFS, overrideDir)
	if err != nil {
		t.Fatalf("RenderFullWithOverrides() error: %v", err)
	}

	// Override content should take precedence.
	readme, ok := result["README.md"]
	if !ok {
		t.Fatal("expected README.md in output")
	}

	if !strings.Contains(string(readme), "Overridden") {
		t.Errorf("expected override content, got: %s", string(readme))
	}

	// Agent adapter files should still be generated.
	if _, skillOK := result[".agents/skills/implement/SKILL.md"]; !skillOK {
		t.Error("missing skill file after override")
	}

	// Provenance should still be applied.
	for path, content := range result {
		s := string(content)
		hasProvenance := strings.Contains(s, "Generated by promptkit")

		hasFrontmatter := strings.HasPrefix(s, "---\n")
		if !hasProvenance && !hasFrontmatter {
			t.Errorf("file %s has neither provenance nor frontmatter after override", path)
		}
	}
}

func TestRenderFullWithOverrides_EmptyOverrideDir(t *testing.T) {
	t.Parallel()

	tmplFS := fstest.MapFS{
		"templates/golang/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}"),
		},
	}

	cfg := testConfig()
	cfg.Agents = nil

	// Empty override dir — should behave like RenderFull.
	result, err := scaffold.RenderFullWithOverrides(cfg, tmplFS, "")
	if err != nil {
		t.Fatalf("RenderFullWithOverrides() error: %v", err)
	}

	if _, ok := result["README.md"]; !ok {
		t.Error("expected README.md in output with empty override dir")
	}
}

// writeFileAtomic tests (exercised through Apply).

func TestWriteFileAtomic_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := []byte("atomic write content")

	rendered := map[string][]byte{
		"output.txt": content,
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "output.txt"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("content = %q, want %q", string(got), string(content))
	}

	// Verify no temp files remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".promptkit.tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestWriteFileAtomic_SetsPermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// .sh files get 0o755 permissions; non-.sh files get 0o644.
	rendered := map[string][]byte{
		"run.sh":    []byte("#!/bin/bash\necho hello"),
		"config.md": []byte("# Config"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// Shell script should be executable (0o755).
	shInfo, err := os.Stat(filepath.Join(dir, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}

	shPerm := shInfo.Mode().Perm()
	if shPerm != 0o755 {
		t.Errorf("run.sh permission = %o, want 0755", shPerm)
	}

	// Non-shell file should be 0o644.
	mdInfo, err := os.Stat(filepath.Join(dir, "config.md"))
	if err != nil {
		t.Fatal(err)
	}

	mdPerm := mdInfo.Mode().Perm()
	if mdPerm != 0o644 {
		t.Errorf("config.md permission = %o, want 0644", mdPerm)
	}
}

func TestWriteFileAtomic_OverwritesExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")

	// Write initial content.
	if err := os.WriteFile(filePath, []byte("original content"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Overwrite via Apply (ModeForce triggers writeFileAtomic on existing file).
	rendered := map[string][]byte{
		"file.txt": []byte("updated content"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if string(got) != "updated content" {
		t.Errorf("content = %q, want %q", string(got), "updated content")
	}
}

func TestWriteFileAtomic_LargeContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a large content buffer (1 MB).
	large := make([]byte, 1<<20)
	for i := range large {
		large[i] = byte('A' + (i % 26))
	}

	rendered := map[string][]byte{
		"large.txt": large,
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "large.txt"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if len(got) != len(large) {
		t.Errorf("content length = %d, want %d", len(got), len(large))
	}

	// Verify no temp files remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".promptkit.tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestWriteFileAtomic_NestedDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	rendered := map[string][]byte{
		"a/b/c/nested.txt": []byte("deeply nested"),
	}

	if err := scaffold.Apply(rendered, dir, scaffold.ModeForce); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "nested.txt"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if string(got) != "deeply nested" {
		t.Errorf("content = %q, want %q", string(got), "deeply nested")
	}

	// Verify no temp files in nested directories.
	entries, _ := os.ReadDir(filepath.Join(dir, "a", "b", "c"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".promptkit.tmp") {
			t.Errorf("temp file left behind in nested dir: %s", e.Name())
		}
	}
}
