// Package scaffold renders templates and manages generated file lifecycle.
package scaffold

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/Sumatoshi-tech/prompts/internal/adapters"
	"github.com/Sumatoshi-tech/prompts/internal/config"
)

const tmplSuffix = ".tmpl"

// TemplateDirForEcosystem returns the template directory path for the given ecosystem.
func TemplateDirForEcosystem(ecosystem string) string {
	return "templates/" + ecosystem
}

// ApplyMode controls how files are written to disk.
type ApplyMode int

const (
	// ModeCreate writes only files that do not exist.
	ModeCreate ApplyMode = iota
	// ModeForce overwrites all files.
	ModeForce
)

// FileDiff represents a difference between rendered and existing content.
type FileDiff struct {
	Path     string
	Existing []byte
	Rendered []byte
	IsNew    bool
}

// Render processes all templates from the given filesystem using the provided config.
// It returns a map of relative output paths to rendered content.
func Render(cfg *config.Config, tmplFS fs.FS) (map[string][]byte, error) {
	result := make(map[string][]byte)
	tmplDir := TemplateDirForEcosystem(cfg.Ecosystem)

	err := fs.WalkDir(tmplFS, tmplDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(tmplFS, path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// Strip the template directory prefix to get the output path.
		outPath := strings.TrimPrefix(path, tmplDir+"/")

		// If it's a .tmpl file, render it; otherwise copy as-is.
		if trimmed, ok := strings.CutSuffix(outPath, tmplSuffix); ok {
			outPath = trimmed

			var rendered []byte

			rendered, err = renderTemplate(path, string(data), cfg)
			if err != nil {
				return fmt.Errorf("rendering %s: %w", path, err)
			}

			result[outPath] = rendered
		} else {
			result[outPath] = data
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking templates: %w", err)
	}

	return result, nil
}

// RenderWithOverrides renders templates, preferring local overrides over embedded ones.
func RenderWithOverrides(cfg *config.Config, tmplFS fs.FS, overrideDir string) (map[string][]byte, error) {
	result, err := Render(cfg, tmplFS)
	if err != nil {
		return nil, err
	}

	// If an override directory exists, render those templates on top.
	if overrideDir == "" {
		return result, nil
	}

	info, err := os.Stat(overrideDir)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return result, nil
	}

	if err != nil {
		return nil, fmt.Errorf("checking override dir: %w", err)
	}

	overrides, err := loadOverrides(overrideDir)
	if err != nil {
		return nil, err
	}

	for path, data := range overrides {
		outPath := path
		if trimmed, ok := strings.CutSuffix(outPath, tmplSuffix); ok {
			outPath = trimmed

			fullPath := filepath.Join(overrideDir, path)

			var rendered []byte

			rendered, err = renderTemplate(path, string(data), cfg)
			if err != nil {
				return nil, fmt.Errorf("rendering override %s (from %s): %w", outPath, fullPath, err)
			}

			result[outPath] = rendered
		} else {
			result[outPath] = data
		}
	}

	return result, nil
}

// loadOverrides reads all files from dir into a map keyed by relative path.
// The directory is walked once and all files are loaded into memory.
func loadOverrides(dir string) (map[string][]byte, error) {
	overrides := make(map[string][]byte)
	overrideFS := os.DirFS(dir)

	err := fs.WalkDir(overrideFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, err := fs.ReadFile(overrideFS, path)
		if err != nil {
			return fmt.Errorf("reading override %s (from %s): %w", path, filepath.Join(dir, path), err)
		}

		overrides[path] = data

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking overrides in %s: %w", dir, err)
	}

	return overrides, nil
}

// Apply writes rendered files to the target directory.
// Files are written atomically using write-to-temp-then-rename.
func Apply(rendered map[string][]byte, targetDir string, mode ApplyMode) error {
	for relPath, content := range rendered {
		outPath := filepath.Clean(filepath.Join(targetDir, relPath))

		if mode == ModeCreate {
			if _, err := os.Stat(outPath); err == nil {
				continue
			}
		}

		dir := filepath.Dir(outPath)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		perm := filePermission(relPath)

		if err := writeFileAtomic(outPath, content, perm, relPath); err != nil {
			return err
		}
	}

	return nil
}

// writeFileAtomic writes content to outPath atomically via a temp file.
// The temp file is created in the same directory to ensure rename works.
func writeFileAtomic(outPath string, content []byte, perm os.FileMode, label string) error {
	outPath = filepath.Clean(outPath)
	tmpPath := outPath + ".promptkit.tmp"

	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating temp file for %s: %w", label, err)
	}

	if _, err = tmpFile.Write(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)

		return fmt.Errorf("writing temp file for %s: %w", label, err)
	}

	if err = tmpFile.Close(); err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("closing temp file for %s: %w", label, err)
	}

	if err = os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("setting permissions for %s: %w", label, err)
	}

	if err = os.Rename(tmpPath, outPath); err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("renaming temp file for %s: %w", label, err)
	}

	return nil
}

// BackupFiles copies existing files to .promptkit/backups/<timestamp>/.
// Only files that already exist on disk are backed up.
// Returns the backup directory path.
func BackupFiles(targetDir string, paths []string) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	backupDir := filepath.Join(targetDir, ".promptkit", "backups", timestamp)

	backed := 0

	for _, relPath := range paths {
		srcPath := filepath.Join(targetDir, relPath)

		data, err := os.ReadFile(srcPath)
		if err != nil {
			// File doesn't exist yet — skip.
			continue
		}

		destPath := filepath.Join(backupDir, relPath)

		if err = os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return "", fmt.Errorf("creating backup directory: %w", err)
		}

		if err = os.WriteFile(destPath, data, 0o600); err != nil {
			return "", fmt.Errorf("backing up %s: %w", relPath, err)
		}

		backed++
	}

	if backed == 0 {
		return "", nil
	}

	return backupDir, nil
}

// RestoreBackup restores files from a backup directory to the target directory.
func RestoreBackup(backupDir, targetDir string) error {
	return filepath.WalkDir(backupDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(backupDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading backup file %s: %w", relPath, err)
		}

		destPath := filepath.Join(targetDir, relPath)

		if err = os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return fmt.Errorf("creating directory for restore: %w", err)
		}

		if err = os.WriteFile(destPath, data, 0o600); err != nil {
			return fmt.Errorf("restoring %s: %w", relPath, err)
		}

		return nil
	})
}

// Diff computes differences between rendered content and existing files.
func Diff(rendered map[string][]byte, targetDir string) ([]FileDiff, error) {
	var diffs []FileDiff

	for relPath, content := range rendered {
		outPath := filepath.Join(targetDir, relPath)

		existing, err := os.ReadFile(outPath)
		if err != nil {
			diffs = append(diffs, FileDiff{
				Path:     relPath,
				Rendered: content,
				IsNew:    true,
			})

			continue
		}

		if !bytes.Equal(existing, content) {
			diffs = append(diffs, FileDiff{
				Path:     relPath,
				Existing: existing,
				Rendered: content,
			})
		}
	}

	return diffs, nil
}

// RenderFull renders templates and applies agent-specific placements.
// It returns the final map of files to write, with raw instructions replaced
// by agent-specific skill files.
func RenderFull(cfg *config.Config, tmplFS fs.FS) (map[string][]byte, error) {
	rendered, err := Render(cfg, tmplFS)
	if err != nil {
		return nil, err
	}

	result, err := applyAgentAdapters(rendered, cfg.Agents, cfg.Workflow)
	if err != nil {
		return nil, err
	}

	return AddProvenance(result), nil
}

// RenderFullWithOverrides is like RenderFull but supports local template overrides.
func RenderFullWithOverrides(cfg *config.Config, tmplFS fs.FS, overrideDir string) (map[string][]byte, error) {
	rendered, err := RenderWithOverrides(cfg, tmplFS, overrideDir)
	if err != nil {
		return nil, err
	}

	result, err := applyAgentAdapters(rendered, cfg.Agents, cfg.Workflow)
	if err != nil {
		return nil, err
	}

	return AddProvenance(result), nil
}

// AddProvenance prepends a provenance comment to every generated file.
// Files with YAML frontmatter (starting with "---") are skipped to avoid
// breaking frontmatter parsing in tools that consume SKILL.md and .mdc files.
func AddProvenance(rendered map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(rendered))

	for path, content := range rendered {
		if bytes.HasPrefix(content, []byte("---\n")) {
			result[path] = content

			continue
		}

		comment := provenanceComment(path)
		buf := make([]byte, len(comment)+len(content))
		copy(buf, comment)
		copy(buf[len(comment):], content)
		result[path] = buf
	}

	return result
}

const provenanceHashComment = "# Generated by promptkit -- do not edit. Regenerate with: promptkit update\n\n"

// provenanceComment returns the appropriate provenance header for a file type.
func provenanceComment(path string) string {
	switch {
	case strings.HasSuffix(path, ".md"), strings.HasSuffix(path, ".mdc"):
		return "<!-- Generated by promptkit -- do not edit. Regenerate with: promptkit update -->\n\n"
	default:
		return provenanceHashComment
	}
}

// RenderSingle renders a single template by output name (e.g. "AGENTS.md").
// It checks the override directory first, then embedded templates.
func RenderSingle(cfg *config.Config, tmplFS fs.FS, overrideDir, name string) ([]byte, error) {
	// Check override directory first.
	if overrideDir != "" {
		overridePath := filepath.Join(overrideDir, name+tmplSuffix)
		if data, err := os.ReadFile(overridePath); err == nil {
			var rendered []byte

			rendered, err = renderTemplate(overridePath, string(data), cfg)
			if err != nil {
				return nil, fmt.Errorf("rendering override %s (from %s): %w", name, overridePath, err)
			}

			return rendered, nil
		}
	}

	tmplDir := TemplateDirForEcosystem(cfg.Ecosystem)

	// Check embedded templates (.tmpl files).
	tmplPath := tmplDir + "/" + name + tmplSuffix

	data, err := fs.ReadFile(tmplFS, tmplPath)
	if err != nil {
		// Try without .tmpl suffix (static files).
		staticPath := tmplDir + "/" + name

		data, err = fs.ReadFile(tmplFS, staticPath)
		if err != nil {
			return nil, fmt.Errorf("template %q not found", name)
		}

		return data, nil
	}

	rendered, err := renderTemplate(tmplPath, string(data), cfg)
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", name, err)
	}

	return rendered, nil
}

func applyAgentAdapters(rendered map[string][]byte, agents []string, workflow string) (map[string][]byte, error) {
	pruneInactiveWorkflowTemplates(rendered, workflow)

	if len(agents) == 0 {
		return rendered, nil
	}

	placed, err := adapters.PlaceForAgents(rendered, agents)
	if err != nil {
		return nil, fmt.Errorf("placing agent files: %w", err)
	}

	// Remove raw instruction files — they are now placed as skills.
	for _, path := range adapters.RemoveInstructionPaths() {
		delete(rendered, path)
	}

	// Add agent-placed files.
	for _, pf := range placed {
		rendered[pf.Path] = pf.Content
	}

	return rendered, nil
}

// pruneInactiveWorkflowTemplates drops the workflow template that does not
// apply so only one of instr-frd.md / instr-journey.md is shipped.
func pruneInactiveWorkflowTemplates(rendered map[string][]byte, workflow string) {
	switch workflow {
	case config.WorkflowJourney:
		delete(rendered, adapters.InstrFRDPath)
	default:
		delete(rendered, adapters.InstrJourneyPath)
	}
}

func newTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"join":  strings.Join,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": cases.Title(language.English).String,
	}
}

func renderTemplate(name, text string, cfg *config.Config) ([]byte, error) {
	funcMap := newTemplateFuncMap()

	tmpl, err := template.New(name).Funcs(funcMap).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer

	execErr := tmpl.Execute(&buf, cfg)
	if execErr != nil {
		return nil, fmt.Errorf("executing template: %w", execErr)
	}

	return buf.Bytes(), nil
}

func filePermission(path string) os.FileMode {
	if strings.HasSuffix(path, ".sh") {
		return 0o755
	}

	return 0o644
}

// UnifiedDiff computes a unified diff between old and new content for a given path.
// Returns an empty string if there are no differences.
func UnifiedDiff(oldContent, newContent []byte, path string) string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	hunks := computeHunks(oldLines, newLines)
	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)

	for _, h := range hunks {
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
			h.oldStart+1, h.oldCount, h.newStart+1, h.newCount)

		for _, line := range h.lines {
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string
}

func computeHunks(oldLines, newLines []string) []hunk {
	const contextLines = 3

	// Compute edit operations using a simple LCS-based diff.
	ops := diffLines(oldLines, newLines)
	if len(ops) == 0 {
		return nil
	}

	// Group operations into hunks with context.
	var hunks []hunk

	var current *hunk

	oldIdx := 0
	newIdx := 0

	for i, op := range ops {
		switch op.kind {
		case opEqual:
			if current != nil {
				// Add trailing context up to contextLines.
				current.lines = append(current.lines, " "+op.text)
				current.oldCount++
				current.newCount++

				// Check if we should close this hunk.
				nextChange := -1

				for j := i + 1; j < len(ops); j++ {
					if ops[j].kind != opEqual {
						nextChange = j
						break
					}
				}

				if nextChange == -1 || nextChange-i > contextLines*2 {
					hunks = append(hunks, *current)
					current = nil
				}
			}

			oldIdx++
			newIdx++

		case opDelete:
			if current == nil {
				current = startHunk(ops, i, oldIdx, newIdx, oldLines, contextLines)
			}

			current.lines = append(current.lines, "-"+op.text)
			current.oldCount++
			oldIdx++

		case opInsert:
			if current == nil {
				current = startHunk(ops, i, oldIdx, newIdx, oldLines, contextLines)
			}

			current.lines = append(current.lines, "+"+op.text)
			current.newCount++
			newIdx++
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	return hunks
}

func startHunk(_ []editOp, _, oldIdx, newIdx int, oldLines []string, contextLines int) *hunk {
	h := &hunk{}

	// Add leading context.
	ctxStart := max(oldIdx-contextLines, 0)

	h.oldStart = ctxStart
	h.newStart = newIdx - (oldIdx - ctxStart)
	h.oldCount = oldIdx - ctxStart
	h.newCount = oldIdx - ctxStart

	for j := ctxStart; j < oldIdx; j++ {
		h.lines = append(h.lines, " "+oldLines[j])
	}

	return h
}

type opKind int

const (
	opEqual opKind = iota
	opDelete
	opInsert
)

type editOp struct {
	kind opKind
	text string
}

// diffLines computes edit operations to transform old into new using an
// LCS-based approach. Uses a flat allocation for the LCS table to minimize
// GC pressure.
func diffLines(oldLines, newLines []string) []editOp {
	oldLen := len(oldLines)
	newLen := len(newLines)
	stride := newLen + 1

	flat := buildLCSTable(oldLines, newLines, oldLen, newLen, stride)
	ops := backtrackLCS(flat, oldLines, newLines, oldLen, newLen, stride)

	if !hasChanges(ops) {
		return nil
	}

	return ops
}

// buildLCSTable computes the LCS dynamic programming table as a flat slice.
func buildLCSTable(oldLines, newLines []string, oldLen, newLen, stride int) []int {
	flat := make([]int, (oldLen+1)*stride)

	for i := oldLen - 1; i >= 0; i-- {
		row := flat[i*stride : (i+1)*stride]
		nextRow := flat[(i+1)*stride : (i+1)*stride+stride]

		for j := newLen - 1; j >= 0; j-- {
			switch {
			case oldLines[i] == newLines[j]:
				row[j] = nextRow[j+1] + 1
			case nextRow[j] >= row[j+1]:
				row[j] = nextRow[j]
			default:
				row[j] = row[j+1]
			}
		}
	}

	return flat
}

// backtrackLCS walks the LCS table to produce edit operations.
func backtrackLCS(flat []int, oldLines, newLines []string, oldLen, newLen, stride int) []editOp {
	ops := make([]editOp, 0, oldLen+newLen)

	i, j := 0, 0

	for i < oldLen && j < newLen {
		switch {
		case oldLines[i] == newLines[j]:
			ops = append(ops, editOp{opEqual, oldLines[i]})
			i++
			j++
		case flat[(i+1)*stride+j] >= flat[i*stride+j+1]:
			ops = append(ops, editOp{opDelete, oldLines[i]})
			i++
		default:
			ops = append(ops, editOp{opInsert, newLines[j]})
			j++
		}
	}

	for i < oldLen {
		ops = append(ops, editOp{opDelete, oldLines[i]})
		i++
	}

	for j < newLen {
		ops = append(ops, editOp{opInsert, newLines[j]})
		j++
	}

	return ops
}

func hasChanges(ops []editOp) bool {
	for _, op := range ops {
		if op.kind != opEqual {
			return true
		}
	}

	return false
}

func splitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	s := string(data)
	lines := strings.Split(s, "\n")

	// Remove trailing empty string from final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// DetectStale returns file paths that were in the previous manifest but are not
// in the current rendered output. These are files that promptkit previously
// generated but would no longer generate with the current config.
func DetectStale(rendered map[string][]byte, previousManifest []string) []string {
	var stale []string

	for _, path := range previousManifest {
		if _, exists := rendered[path]; !exists {
			stale = append(stale, path)
		}
	}

	sort.Strings(stale)

	return stale
}

// RemoveFiles deletes the given files relative to targetDir.
// It silently skips files that do not exist.
func RemoveFiles(targetDir string, paths []string) error {
	for _, relPath := range paths {
		fullPath := filepath.Join(targetDir, relPath)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale file %s: %w", relPath, err)
		}
	}

	return nil
}

// FileManifest returns a sorted list of all file paths in the rendered output.
func FileManifest(rendered map[string][]byte) []string {
	paths := make([]string, 0, len(rendered))
	for path := range rendered {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	return paths
}

// ComputeChecksums returns a map of file path to SHA-256 hex digest.
func ComputeChecksums(rendered map[string][]byte) map[string]string {
	checksums := make(map[string]string, len(rendered))

	for path, content := range rendered {
		sum := sha256.Sum256(content)
		checksums[path] = hex.EncodeToString(sum[:])
	}

	return checksums
}

// DiffRendered compares two rendered file maps and returns diffs.
// Files only in local are marked IsNew. Files only in upstream are returned
// as diffs with empty Rendered (representing removed files). Files in both
// with different content are returned as normal diffs.
func DiffRendered(local, upstream map[string][]byte) []FileDiff {
	var diffs []FileDiff

	for path, localContent := range local {
		upstreamContent, ok := upstream[path]
		if !ok {
			diffs = append(diffs, FileDiff{
				Path:     path,
				Rendered: localContent,
				IsNew:    true,
			})

			continue
		}

		if !bytes.Equal(localContent, upstreamContent) {
			diffs = append(diffs, FileDiff{
				Path:     path,
				Existing: upstreamContent,
				Rendered: localContent,
			})
		}
	}

	for path, upstreamContent := range upstream {
		if _, ok := local[path]; !ok {
			diffs = append(diffs, FileDiff{
				Path:     path,
				Existing: upstreamContent,
			})
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})

	return diffs
}

const overrideChecksumsFile = "override-checksums.json"

// SaveOverrideChecksum records the SHA-256 of an embedded template at the time
// an override is created, enabling later staleness detection.
func SaveOverrideChecksum(overrideDir, name string, embeddedContent []byte) {
	checksumPath := filepath.Join(filepath.Dir(overrideDir), overrideChecksumsFile)

	checksums := loadOverrideChecksums(checksumPath)
	sum := sha256.Sum256(embeddedContent)
	checksums[name] = hex.EncodeToString(sum[:])

	data, err := json.MarshalIndent(checksums, "", "  ")
	if err != nil {
		return
	}

	_ = os.MkdirAll(filepath.Dir(checksumPath), 0o750)
	_ = os.WriteFile(checksumPath, data, 0o600)
}

// CheckOverrideStaleness returns override file names whose upstream embedded
// template has changed since the override was created.
func CheckOverrideStaleness(tmplFS fs.FS, overrideDir, ecosystem string) []string {
	if overrideDir == "" {
		return nil
	}

	checksumPath := filepath.Join(filepath.Dir(overrideDir), overrideChecksumsFile)
	checksums := loadOverrideChecksums(checksumPath)

	if len(checksums) == 0 {
		return nil
	}

	tmplDir := TemplateDirForEcosystem(ecosystem)

	var stale []string

	for name, savedSum := range checksums {
		// Compute current embedded template checksum.
		tmplName := strings.TrimSuffix(name, ".tmpl")
		tmplPath := tmplDir + "/" + tmplName + ".tmpl"

		data, err := fs.ReadFile(tmplFS, tmplPath)
		if err != nil {
			// Try static file.
			data, err = fs.ReadFile(tmplFS, tmplDir+"/"+tmplName)
			if err != nil {
				continue
			}
		}

		sum := sha256.Sum256(data)

		currentSum := hex.EncodeToString(sum[:])
		if currentSum != savedSum {
			stale = append(stale, name)
		}
	}

	sort.Strings(stale)

	return stale
}

func loadOverrideChecksums(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]string)
	}

	var checksums map[string]string
	if err = json.Unmarshal(data, &checksums); err != nil {
		return make(map[string]string)
	}

	return checksums
}
