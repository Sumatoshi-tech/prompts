// Package prompt implements interactive prompts for configuration setup.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Sumatoshi-tech/prompts/internal/config"
)

// RunInitPrompts interactively gathers project configuration from the user.
func RunInitPrompts(cfg *config.Config, defaultName string) (*config.Config, error) {
	return gatherConfig(cfg, defaultName, os.Stdin, os.Stdout)
}

func gatherConfig(cfg *config.Config, defaultName string, in io.Reader, out io.Writer) (*config.Config, error) {
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

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func parseAgents(input string) []string {
	parts := strings.Split(input, ",")
	agents := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.ToLower(part))
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true

			agents = append(agents, trimmed)
		}
	}

	return agents
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
