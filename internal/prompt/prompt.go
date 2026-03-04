package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Sumatoshi-tech/promptkit/internal/config"
)

// RunInitPrompts interactively gathers project configuration from the user.
func RunInitPrompts(cfg *config.Config, defaultName string) (*config.Config, error) {
	var err error

	// Ecosystem selection.
	fmt.Println("Available ecosystems:")

	for _, name := range config.ValidEcosystemNames() {
		descs := config.EcosystemDescriptions()
		fmt.Printf("  %-10s %s\n", name, descs[name])
	}

	fmt.Println()

	ecoStr, err := ask("Ecosystem", cfg.Ecosystem)
	if err != nil {
		return nil, err
	}

	cfg.Ecosystem = ecoStr

	mod := config.GetEcosystem(cfg.Ecosystem)

	cfg.ProjectName, err = ask("Project name", defaultName)
	if err != nil {
		return nil, err
	}

	cfg.ModulePath, err = ask("Module path (e.g. github.com/user/project)", "")
	if err != nil {
		return nil, err
	}

	cfg.Description, err = ask("Project description", "")
	if err != nil {
		return nil, err
	}

	cfg.Expertise, err = ask("Agent domain expertise (e.g. distributed systems)", "")
	if err != nil {
		return nil, err
	}

	// Ecosystem-specific prompts (e.g. Go version, Rust edition, Zig version).
	if mod != nil && mod.RunPrompts != nil {
		if err := mod.RunPrompts(cfg, ask, askBool); err != nil {
			return nil, err
		}
	}

	// Workflow selection.
	fmt.Println("\nAvailable workflows:")

	for _, name := range config.ValidWorkflowNames() {
		fmt.Printf("  %-10s %s\n", name, config.WorkflowDescriptions[name])
	}

	fmt.Println()

	wfStr, err := ask("Workflow", cfg.Workflow)
	if err != nil {
		return nil, err
	}

	cfg.Workflow = wfStr

	binaryName, err := ask("Binary name", cfg.ProjectName)
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

	docker, err := askBool("Enable Docker support?", true)
	if err != nil {
		return nil, err
	}

	cfg.Features.Docker = docker

	fmt.Println("\nAvailable AI agents:")

	for _, name := range config.ValidAgentNames() {
		fmt.Printf("  %-10s %s\n", name, config.AgentDescriptions[name])
	}

	fmt.Println()

	agentsStr, err := ask("Target AI agents (comma-separated)", "claude")
	if err != nil {
		return nil, err
	}

	cfg.Agents = parseAgents(agentsStr)

	if err := cfg.Validate(); err != nil {
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

func ask(prompt, defaultVal string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
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

func askBool(prompt string, defaultVal bool) (bool, error) {
	defStr := "y/N"
	if defaultVal {
		defStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defStr)

	reader := bufio.NewReader(os.Stdin)

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
