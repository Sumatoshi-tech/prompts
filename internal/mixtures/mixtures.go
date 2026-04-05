// Package mixtures handles loading, resolving, and rendering mixture templates.
// A mixture is a reusable content fragment injected into targeted skills.
package mixtures

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/Sumatoshi-tech/prompts/internal/config"
)

const (
	sharedMixturesDir = "templates/_shared/mixtures"
	metadataFile      = "mixture.yaml"
)

// Def holds metadata for a single mixture.
type Def struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	UseCase     string   `yaml:"use_case"`
	Targets     []string `yaml:"targets"`
}

// mixturesDir returns the ecosystem-specific mixtures directory.
func mixturesDir(ecosystem string) string {
	return "templates/" + ecosystem + "/mixtures"
}

// LoadAll discovers all mixture definitions from the shared mixtures directory.
// Returns a map keyed by mixture name.
func LoadAll(tmplFS fs.FS) (map[string]Def, error) {
	result := make(map[string]Def)

	entries, err := fs.ReadDir(tmplFS, sharedMixturesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, nil
		}

		return nil, fmt.Errorf("reading mixtures directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := sharedMixturesDir + "/" + entry.Name() + "/" + metadataFile

		data, readErr := fs.ReadFile(tmplFS, metaPath)
		if readErr != nil {
			continue // Directory without mixture.yaml is skipped.
		}

		var def Def
		if parseErr := yaml.Unmarshal(data, &def); parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", metaPath, parseErr)
		}

		if def.Name == "" {
			def.Name = entry.Name()
		}

		result[def.Name] = def
	}

	return result, nil
}

// Names returns a sorted list of all available mixture names from the embedded FS.
func Names(tmplFS fs.FS) []string {
	defs, err := LoadAll(tmplFS)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// RenderForSkill renders a single mixture's content for a single skill.
// Resolution order: ecosystem override first, then shared fallback.
// Returns nil if the mixture has no template for this skill.
func RenderForSkill(tmplFS fs.FS, ecosystem, mixtureName, skillName string, cfg *config.Config) ([]byte, error) {
	tmplData, err := loadMixtureTemplate(tmplFS, ecosystem, mixtureName, skillName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("loading mixture template %s/%s: %w", mixtureName, skillName, err)
	}

	tmplName := mixtureName + "/" + skillName

	tmpl, err := template.New(tmplName).Funcs(templateFuncMap()).Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("parsing mixture template %s: %w", tmplName, err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("executing mixture template %s: %w", tmplName, err)
	}

	return buf.Bytes(), nil
}

// AppendToSkill renders all active mixtures targeting the given skill
// and appends them to the skill body. Mixtures are appended in sorted
// order by name for deterministic output.
func AppendToSkill(
	body []byte, tmplFS fs.FS, ecosystem string,
	activeMixtures []string, skillName string, cfg *config.Config,
) ([]byte, error) {
	if len(activeMixtures) == 0 {
		return body, nil
	}

	allDefs, err := LoadAll(tmplFS)
	if err != nil {
		return nil, fmt.Errorf("loading mixture definitions: %w", err)
	}

	// Sort for deterministic output.
	sorted := make([]string, len(activeMixtures))
	copy(sorted, activeMixtures)
	sort.Strings(sorted)

	var appended []byte

	for _, name := range sorted {
		def, ok := allDefs[name]
		if !ok {
			continue // Unknown mixture — skip (validation catches this elsewhere).
		}

		if !targetsSkill(def.Targets, skillName) {
			continue
		}

		rendered, renderErr := RenderForSkill(tmplFS, ecosystem, name, skillName, cfg)
		if renderErr != nil {
			return nil, fmt.Errorf("rendering mixture %q for skill %q: %w", name, skillName, renderErr)
		}

		if len(rendered) == 0 {
			continue
		}

		section := fmt.Sprintf("\n\n---\n\n## Mixture: %s\n\n", def.Description)
		appended = append(appended, []byte(section)...)
		appended = append(appended, rendered...)
	}

	if len(appended) == 0 {
		return body, nil
	}

	result := make([]byte, len(body)+len(appended))
	copy(result, body)
	copy(result[len(body):], appended)

	return result, nil
}

func targetsSkill(targets []string, skillName string) bool {
	return slices.Contains(targets, skillName)
}

// loadMixtureTemplate reads a mixture template for a specific skill.
// Checks ecosystem-specific directory first, falls back to shared.
func loadMixtureTemplate(tmplFS fs.FS, ecosystem, mixtureName, skillName string) ([]byte, error) {
	fileName := skillName + ".md.tmpl"

	// Try ecosystem override first.
	ecoPath := mixturesDir(ecosystem) + "/" + mixtureName + "/" + fileName

	data, err := fs.ReadFile(tmplFS, ecoPath)
	if err == nil {
		return data, nil
	}

	// Fall back to shared.
	sharedPath := sharedMixturesDir + "/" + mixtureName + "/" + fileName

	data, err = fs.ReadFile(tmplFS, sharedPath)
	if err != nil {
		return nil, fmt.Errorf("reading mixture template %s: %w", sharedPath, err)
	}

	return data, nil
}

func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"join":  strings.Join,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}
}
