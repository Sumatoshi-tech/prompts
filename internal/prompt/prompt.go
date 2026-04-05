// Package prompt implements interactive prompts for configuration setup.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/Sumatoshi-tech/prompts/internal/config"
	"github.com/Sumatoshi-tech/prompts/internal/mixtures"
)

const separatorWidth = 60

// RunInitPrompts interactively gathers project configuration from the user.
func RunInitPrompts(cfg *config.Config, defaultName string, tmplFS fs.FS) (*config.Config, error) {
	return gatherConfig(cfg, defaultName, os.Stdin, os.Stdout, tmplFS)
}

func gatherConfig(cfg *config.Config, defaultName string, in io.Reader, out io.Writer, tmplFS fs.FS) (*config.Config, error) {
	reader := bufio.NewReader(in)

	askFn := func(prompt, defaultVal string) (string, error) {
		return ask(reader, out, prompt, defaultVal)
	}

	askRequiredFn := func(prompt, defaultVal string) (string, error) {
		for {
			val, err := ask(reader, out, prompt, defaultVal)
			if err != nil {
				return "", err
			}

			if val != "" {
				return val, nil
			}

			fmt.Fprintf(out, "  %s is required, please provide a value.\n", prompt)
		}
	}

	askBoolFn := func(prompt string, defaultVal bool) (bool, error) {
		return askBool(reader, out, prompt, defaultVal)
	}

	var err error

	// Ecosystem selection.
	fmt.Fprintln(out, "Available ecosystems:")

	for _, name := range config.ValidEcosystemNames() {
		descs := config.EcosystemDescriptions()

		fmt.Fprintf(out, "  %-10s %s\n", name, descs[name])
	}

	fmt.Fprintln(out)

	ecoStr, err := askFn("Ecosystem", cfg.Ecosystem)
	if err != nil {
		return nil, err
	}

	cfg.Ecosystem = ecoStr

	mod := config.GetEcosystem(cfg.Ecosystem)

	cfg.ProjectName, err = askRequiredFn("Project name", defaultName)
	if err != nil {
		return nil, err
	}

	askField := askFn
	if mod != nil && mod.Requires("module_path") {
		askField = askRequiredFn
	}

	cfg.ModulePath, err = askField("Module path (e.g. github.com/user/project)", "")
	if err != nil {
		return nil, err
	}

	cfg.Description, err = askFn("Project description", "")
	if err != nil {
		return nil, err
	}

	cfg.Expertise, err = askFn("Agent domain expertise (e.g. distributed systems)", "")
	if err != nil {
		return nil, err
	}

	// Ecosystem-specific prompts (e.g. Go version, Rust edition, Zig version).
	if mod != nil && mod.RunPrompts != nil {
		if runErr := mod.RunPrompts(cfg, askFn, askBoolFn); runErr != nil {
			return nil, runErr
		}
	}

	// Workflow selection.
	fmt.Fprintln(out, "\nAvailable workflows:")

	for _, name := range config.ValidWorkflowNames() {
		fmt.Fprintf(out, "  %-10s %s\n", name, config.WorkflowDescriptions[name])
	}

	fmt.Fprintln(out)

	wfStr, err := askFn("Workflow", cfg.Workflow)
	if err != nil {
		return nil, err
	}

	cfg.Workflow = wfStr

	binaryName, err := askFn("Binary name", cfg.ProjectName)
	if err != nil {
		return nil, err
	}

	cmdPath := "./cmd/" + binaryName
	if mod != nil {
		cmdPath = mod.DefaultCmdPath(binaryName)
	}

	cfg.Binaries = []config.Binary{
		{Name: binaryName, CmdPath: cmdPath},
	}

	if mod != nil {
		cfg.AnalysisCmd = mod.DefaultAnalysisCmd
	}

	docker, err := askBoolFn("Enable Docker support?", true)
	if err != nil {
		return nil, err
	}

	cfg.Features.Docker = docker

	fmt.Fprintln(out, "\nAvailable AI agents:")

	for _, name := range config.ValidAgentNames() {
		fmt.Fprintf(out, "  %-10s %s\n", name, config.AgentDescriptions[name])
	}

	fmt.Fprintln(out)

	agentsStr, err := askFn("Target AI agents (comma-separated)", "claude")
	if err != nil {
		return nil, err
	}

	cfg.Agents = parseAgents(agentsStr)

	// Mixture selection (optional).
	if tmplFS != nil {
		selected, mixErr := promptMixtures(tmplFS, out, askFn)
		if mixErr != nil {
			return nil, mixErr
		}

		cfg.Mixtures = selected
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Offer to prepare project-related skills after init.
	genSkills, err := askBoolFn("Generate project-related skills after init?", false)
	if err != nil {
		return nil, err
	}

	if genSkills {
		printSkillGenerationPrompt(out, cfg)
	}

	return cfg, nil
}

// printSkillGenerationPrompt outputs a ready-to-use prompt that the user can
// feed to their selected AI agent to generate project-specific skills.
func printSkillGenerationPrompt(out io.Writer, cfg *config.Config) {
	fmt.Fprintln(out, "\n"+strings.Repeat("─", separatorWidth))
	fmt.Fprintln(out, "Copy the prompt below and run it with your AI agent after init.")
	fmt.Fprintln(out, "It will generate project-specific skills and document them.")
	fmt.Fprintln(out, strings.Repeat("─", separatorWidth))

	fmt.Fprintf(out, `
You are bootstrapping project-specific skills for "%s".

Project context:
- Name: %s
- Ecosystem: %s
- Workflow: %s
- Description: %s
- Expertise domain: %s
- Target agents: %s

Your task:
1. Read AGENTS.md and all existing skills in .agents/skills/ to understand the current skill set.
2. Read docs/ and specs/ to understand the project domain.
3. Based on the project's expertise domain ("%s") and ecosystem ("%s"),
   identify 2-5 project-specific skills that would accelerate development. Consider:
   - Domain-specific workflows (e.g., /migrate for database projects, /api for REST services, /protocol for network code)
   - Testing patterns unique to this domain (e.g., /fuzz for parsers, /bench for performance-critical code)
   - Integration workflows (e.g., /deploy, /release, /changelog)
   - Code generation patterns relevant to the ecosystem
4. For each skill:
   a. Create .agents/skills/<name>/SKILL.md with frontmatter (name, description) and detailed instructions
   b. Create the corresponding agent command file:
`,
		cfg.ProjectName,
		cfg.ProjectName,
		cfg.Ecosystem,
		cfg.Workflow,
		cfg.Description,
		cfg.Expertise,
		strings.Join(cfg.Agents, ", "),
		cfg.Expertise,
		cfg.Ecosystem,
	)

	for _, agent := range cfg.Agents {
		switch agent {
		case "claude":
			fmt.Fprintf(out, "      - Claude: .claude/commands/<name>.md\n")
		case "gemini":
			fmt.Fprintf(out, "      - Gemini: .gemini/commands/<name>.toml\n")
		case "windsurf":
			fmt.Fprintf(out, "      - Windsurf: .windsurf/workflows/<name>.md\n")
		case "cursor":
			fmt.Fprintf(out, "      - Cursor: .cursor/rules/<name>.mdc\n")
		}
	}

	fmt.Fprintf(out, `5. Update AGENTS.md to list the new skills in a "Project Skills" section.
6. Output a summary of what skills were created and why.

Rules:
- Each skill must follow the same structure as existing skills (frontmatter + instructions).
- Skills must be specific to this project's domain, not generic.
- Do not duplicate functionality already covered by /implement, /roadmap, /perf, /bug, /frd.
`)

	fmt.Fprintln(out, strings.Repeat("─", separatorWidth))
}

func parseMixtures(input string) []string {
	return parseCommaSeparated(input)
}

func parseAgents(input string) []string {
	return parseCommaSeparated(input)
}

func parseCommaSeparated(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.ToLower(part))
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true

			result = append(result, trimmed)
		}
	}

	return result
}

func promptMixtures(tmplFS fs.FS, out io.Writer, askFn func(string, string) (string, error)) ([]string, error) {
	allMixtures, err := mixtures.LoadAll(tmplFS)
	if err != nil {
		return nil, fmt.Errorf("loading mixtures: %w", err)
	}

	if len(allMixtures) == 0 {
		return nil, nil
	}

	fmt.Fprintln(out, "\nAvailable mixtures (cross-cutting concerns injected into skills):")

	names := make([]string, 0, len(allMixtures))
	for name := range allMixtures {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		def := allMixtures[name]
		fmt.Fprintf(out, "  %-15s %s\n", name, def.Description)

		if def.UseCase != "" {
			fmt.Fprintf(out, "  %-15s Use-case: %s\n", "", def.UseCase)
		}
	}

	fmt.Fprintln(out)

	mixturesStr, err := askFn("Mixtures (comma-separated, or empty to skip)", "")
	if err != nil {
		return nil, err
	}

	if mixturesStr == "" {
		return nil, nil
	}

	return parseMixtures(mixturesStr), nil
}

func ask(reader *bufio.Reader, out io.Writer, prompt, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(out, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(out, "%s: ", prompt)
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}

	return line, nil
}

func askBool(reader *bufio.Reader, out io.Writer, prompt string, defaultVal bool) (bool, error) {
	defStr := "y/N"
	if defaultVal {
		defStr = "Y/n"
	}

	fmt.Fprintf(out, "%s [%s]: ", prompt, defStr)

	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}

	line = strings.TrimSpace(strings.ToLower(line))

	if line == "" {
		return defaultVal, nil
	}

	return line == "y" || line == "yes", nil
}
