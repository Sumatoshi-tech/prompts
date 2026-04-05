package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/Sumatoshi-tech/prompts/internal/config"
)

// parseAgents tests (existing).

func TestParseAgents_Deduplication(t *testing.T) {
	t.Parallel()

	got := parseAgents("claude,claude,cursor,claude")
	want := []string{"claude", "cursor"}

	if len(got) != len(want) {
		t.Fatalf("parseAgents() returned %d agents, want %d: %v", len(got), len(want), got)
	}

	for i, agent := range got {
		if agent != want[i] {
			t.Errorf("parseAgents()[%d] = %q, want %q", i, agent, want[i])
		}
	}
}

func TestParseAgents_TrimsAndLowercases(t *testing.T) {
	t.Parallel()

	got := parseAgents(" Claude , CURSOR , gemini ")
	want := []string{"claude", "cursor", "gemini"}

	if len(got) != len(want) {
		t.Fatalf("parseAgents() returned %d agents, want %d: %v", len(got), len(want), got)
	}

	for i, agent := range got {
		if agent != want[i] {
			t.Errorf("parseAgents()[%d] = %q, want %q", i, agent, want[i])
		}
	}
}

func TestParseAgents_SkipsEmpty(t *testing.T) {
	t.Parallel()

	got := parseAgents("claude,,cursor,")
	want := []string{"claude", "cursor"}

	if len(got) != len(want) {
		t.Fatalf("parseAgents() returned %d agents, want %d: %v", len(got), len(want), got)
	}

	for i, agent := range got {
		if agent != want[i] {
			t.Errorf("parseAgents()[%d] = %q, want %q", i, agent, want[i])
		}
	}
}

func TestParseAgents_Single(t *testing.T) {
	t.Parallel()

	got := parseAgents("claude")
	if len(got) != 1 || got[0] != "claude" {
		t.Errorf("parseAgents(\"claude\") = %v, want [\"claude\"]", got)
	}
}

// Helpers for test input.

// newTestInput creates a pipe and writes the given lines into it.
// Returns an [io.Reader] that can be passed to [gatherConfig] or wrapped with [bufio.NewReader].
// No global state is modified; safe for parallel tests.
func newTestInput(t *testing.T, lines ...string) io.Reader {
	t.Helper()

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	t.Cleanup(func() { pipeReader.Close() })

	go func() {
		defer pipeWriter.Close()

		for _, line := range lines {
			fmt.Fprintln(pipeWriter, line)
		}
	}()

	return pipeReader
}

// assertStrEqual is a test helper that compares two strings.
func assertStrEqual(t *testing.T, field, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

// assertBoolEqual is a test helper that compares two bools.
func assertBoolEqual(t *testing.T, field string, got, want bool) {
	t.Helper()

	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

// assertStrSliceEqual is a test helper that compares two string slices.
//
//nolint:unparam // field is always "Agents" for now.
func assertStrSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: got %d elements %v, want %d elements %v", field, len(got), got, len(want), want)
	}

	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", field, i, got[i], want[i])
		}
	}
}

// ask tests.

func TestAsk_ReturnsUserInput(t *testing.T) {
	t.Parallel()

	in := newTestInput(t, "hello world")
	reader := bufio.NewReader(in)

	got, err := ask(reader, io.Discard, "Enter value", "default")
	if err != nil {
		t.Fatalf("ask() error: %v", err)
	}

	assertStrEqual(t, "ask()", got, "hello world")
}

func TestAsk_ReturnsDefaultOnEmpty(t *testing.T) {
	t.Parallel()

	in := newTestInput(t, "")
	reader := bufio.NewReader(in)

	got, err := ask(reader, io.Discard, "Enter value", "mydefault")
	if err != nil {
		t.Fatalf("ask() error: %v", err)
	}

	assertStrEqual(t, "ask()", got, "mydefault")
}

func TestAsk_ReturnsEmptyWhenNoDefault(t *testing.T) {
	t.Parallel()

	in := newTestInput(t, "")
	reader := bufio.NewReader(in)

	got, err := ask(reader, io.Discard, "Enter value", "")
	if err != nil {
		t.Fatalf("ask() error: %v", err)
	}

	assertStrEqual(t, "ask()", got, "")
}

func TestAsk_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	in := newTestInput(t, "  trimmed  ")
	reader := bufio.NewReader(in)

	got, err := ask(reader, io.Discard, "Enter value", "")
	if err != nil {
		t.Fatalf("ask() error: %v", err)
	}

	assertStrEqual(t, "ask()", got, "trimmed")
}

func TestAsk_ErrorOnClosedPipe(t *testing.T) {
	t.Parallel()

	// Provide no input at all (pipe closes immediately).
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	pipeWriter.Close()

	t.Cleanup(func() { pipeReader.Close() })

	reader := bufio.NewReader(pipeReader)

	_, err = ask(reader, io.Discard, "Enter value", "default")
	if err == nil {
		t.Error("ask() should return error when stdin is closed")
	}
}

// askBool tests.

func TestAskBool_DefaultTrue(t *testing.T) {
	t.Parallel()

	in := newTestInput(t, "")
	reader := bufio.NewReader(in)

	got, err := askBool(reader, io.Discard, "Enable?", true)
	if err != nil {
		t.Fatalf("askBool() error: %v", err)
	}

	assertBoolEqual(t, "askBool(default=true)", got, true)
}

func TestAskBool_DefaultFalse(t *testing.T) {
	t.Parallel()

	in := newTestInput(t, "")
	reader := bufio.NewReader(in)

	got, err := askBool(reader, io.Discard, "Enable?", false)
	if err != nil {
		t.Fatalf("askBool() error: %v", err)
	}

	assertBoolEqual(t, "askBool(default=false)", got, false)
}

func TestAskBool_YesInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			in := newTestInput(t, tc.input)
			reader := bufio.NewReader(in)

			got, err := askBool(reader, io.Discard, "Enable?", false)
			if err != nil {
				t.Fatalf("askBool(%q) error: %v", tc.input, err)
			}

			assertBoolEqual(t, fmt.Sprintf("askBool(%q)", tc.input), got, tc.want)
		})
	}
}

func TestAskBool_NoInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"n", false},
		{"N", false},
		{"no", false},
		{"NO", false},
		{"nope", false},
		{"something", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			in := newTestInput(t, tc.input)
			reader := bufio.NewReader(in)

			got, err := askBool(reader, io.Discard, "Enable?", true)
			if err != nil {
				t.Fatalf("askBool(%q) error: %v", tc.input, err)
			}

			assertBoolEqual(t, fmt.Sprintf("askBool(%q)", tc.input), got, tc.want)
		})
	}
}

func TestAskBool_ErrorOnClosedPipe(t *testing.T) {
	t.Parallel()

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	pipeWriter.Close()

	t.Cleanup(func() { pipeReader.Close() })

	reader := bufio.NewReader(pipeReader)

	_, err = askBool(reader, io.Discard, "Enable?", false)
	if err == nil {
		t.Error("askBool() should return error when stdin is closed")
	}
}

// RunInitPrompts tests.

func TestRunInitPrompts_Golang(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"golang",                      // 1. Ecosystem.
		"mygoproject",                 // 2. Project name.
		"github.com/user/mygoproject", // 3. Module path.
		"A Go project",                // 4. Description.
		"distributed systems",         // 5. Expertise.
		"1.23",                        // 6. Go version.
		"n",                           // 7. CGO.
		"frd",                         // 8. Workflow.
		"mygoproject",                 // 9. Binary name.
		"y",                           // 10. Docker.
		"claude",                      // 11. Agents.
		"n",                           // 12. Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "defaultname", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "golang")
	assertStrEqual(t, "ProjectName", result.ProjectName, "mygoproject")
	assertStrEqual(t, "ModulePath", result.ModulePath, "github.com/user/mygoproject")
	assertStrEqual(t, "Description", result.Description, "A Go project")
	assertStrEqual(t, "Expertise", result.Expertise, "distributed systems")
	assertStrEqual(t, "GoVersion", result.GoVersion, "1.23")
	assertBoolEqual(t, "Features.CGO", result.Features.CGO, false)
	assertStrEqual(t, "Workflow", result.Workflow, "frd")
	assertStrEqual(t, "AnalysisCmd", result.AnalysisCmd, "go vet ./...")
	assertBoolEqual(t, "Features.Docker", result.Features.Docker, true)
	assertStrSliceEqual(t, "Agents", result.Agents, []string{"claude"})

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries: got %d, want 1", len(result.Binaries))
	}

	assertStrEqual(t, "Binaries[0].Name", result.Binaries[0].Name, "mygoproject")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "./cmd/mygoproject")
}

func TestRunInitPrompts_GolangWithCGO(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"golang",
		"cgoproject",
		"github.com/user/cgoproject",
		"CGO enabled project",
		"C interop",
		"1.22",
		"y", // CGO enabled.
		"frd",
		"cgoproject",
		"y",
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "cgoproject", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "golang")
	assertBoolEqual(t, "Features.CGO", result.Features.CGO, true)
	assertStrEqual(t, "GoVersion", result.GoVersion, "1.22")
}

func TestRunInitPrompts_Rust(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"rust",
		"mycrustproject",
		"github.com/user/mycrustproject",
		"A Rust project",
		"systems programming",
		"2024",
		"forbid",
		"journey",
		"mycrustproject",
		"n",
		"claude,cursor",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "defaultname", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "rust")
	assertStrEqual(t, "ProjectName", result.ProjectName, "mycrustproject")
	assertStrEqual(t, "ModulePath", result.ModulePath, "github.com/user/mycrustproject")
	assertStrEqual(t, "Description", result.Description, "A Rust project")
	assertStrEqual(t, "Expertise", result.Expertise, "systems programming")
	assertStrEqual(t, "RustEdition", result.RustEdition, "2024")
	assertStrEqual(t, "UnsafePolicy", result.UnsafePolicy, "forbid")
	assertStrEqual(t, "Workflow", result.Workflow, "journey")
	assertStrEqual(t, "AnalysisCmd", result.AnalysisCmd, "cargo clippy -- -D warnings")
	assertBoolEqual(t, "Features.Docker", result.Features.Docker, false)
	assertStrSliceEqual(t, "Agents", result.Agents, []string{"claude", "cursor"})

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries: got %d, want 1", len(result.Binaries))
	}

	assertStrEqual(t, "Binaries[0].Name", result.Binaries[0].Name, "mycrustproject")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "src/main.rs")
}

func TestRunInitPrompts_Zig(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"zig",
		"myzigproject",
		"github.com/user/myzigproject",
		"A Zig project",
		"low-level systems",
		"0.14",
		"y",
		"frd",
		"myzigproject",
		"y",
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "defaultname", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "zig")
	assertStrEqual(t, "ProjectName", result.ProjectName, "myzigproject")
	assertStrEqual(t, "ModulePath", result.ModulePath, "github.com/user/myzigproject")
	assertStrEqual(t, "Description", result.Description, "A Zig project")
	assertStrEqual(t, "Expertise", result.Expertise, "low-level systems")
	assertStrEqual(t, "ZigVersion", result.ZigVersion, "0.14")
	assertBoolEqual(t, "LinkLibc", result.LinkLibc, true)
	assertStrEqual(t, "Workflow", result.Workflow, "frd")
	assertStrEqual(t, "AnalysisCmd", result.AnalysisCmd, "zig build test")
	assertBoolEqual(t, "Features.Docker", result.Features.Docker, true)
	assertStrSliceEqual(t, "Agents", result.Agents, []string{"claude"})

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries: got %d, want 1", len(result.Binaries))
	}

	assertStrEqual(t, "Binaries[0].Name", result.Binaries[0].Name, "myzigproject")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "src/main.zig")
}

func TestRunInitPrompts_ZigWithoutLibc(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"zig",
		"zignolibc",
		"github.com/user/zignolibc",
		"No libc",
		"compilers",
		"0.13",
		"n", // no libc.
		"frd",
		"zignolibc",
		"y",
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "zignolibc", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "zig")
	assertBoolEqual(t, "LinkLibc", result.LinkLibc, false)
	assertStrEqual(t, "ZigVersion", result.ZigVersion, "0.13")
}

func TestRunInitPrompts_DefaultValues(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"",                            // 1. Default ecosystem (golang).
		"",                            // 2. Default project name (testdefault).
		"github.com/user/testdefault", // 3. Module path (required).
		"",                            // 4. Default description.
		"",                            // 5. Default expertise.
		"",                            // 6. Default go version (1.22).
		"",                            // 7. Default CGO (no).
		"",                            // 8. Default workflow (frd).
		"",                            // 9. Default binary name (testdefault).
		"",                            // 10. Default docker (yes).
		"",                            // 11. Default agents (claude).
		"n",                           // 12. Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "testdefault", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "golang")
	assertStrEqual(t, "ProjectName", result.ProjectName, "testdefault")
	assertStrEqual(t, "ModulePath", result.ModulePath, "github.com/user/testdefault")
	assertStrEqual(t, "Description", result.Description, "")
	assertStrEqual(t, "Expertise", result.Expertise, "")
	assertStrEqual(t, "GoVersion", result.GoVersion, "1.22")
	assertBoolEqual(t, "Features.CGO", result.Features.CGO, false)
	assertStrEqual(t, "Workflow", result.Workflow, "frd")
	assertBoolEqual(t, "Features.Docker", result.Features.Docker, true)
	assertStrSliceEqual(t, "Agents", result.Agents, []string{"claude"})

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries: got %d, want 1", len(result.Binaries))
	}

	assertStrEqual(t, "Binaries[0].Name", result.Binaries[0].Name, "testdefault")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "./cmd/testdefault")
}

func TestRunInitPrompts_RustDefaults(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"rust",
		"rustdefault",
		"github.com/user/rustdefault",
		"",  // Default description.
		"",  // Default expertise.
		"",  // Default rust edition (2021).
		"",  // Default unsafe policy (deny).
		"",  // Default workflow (frd).
		"",  // Default binary name (rustdefault).
		"",  // Default docker (yes).
		"",  // Default agents (claude).
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "rustdefault", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Ecosystem", result.Ecosystem, "rust")
	assertStrEqual(t, "RustEdition", result.RustEdition, "2021")
	assertStrEqual(t, "UnsafePolicy", result.UnsafePolicy, "deny")
	assertStrEqual(t, "ProjectName", result.ProjectName, "rustdefault")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "src/main.rs")
}

func TestRunInitPrompts_JourneyWorkflow(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"golang",
		"journeyproject",
		"github.com/user/journeyproject",
		"",
		"",
		"",
		"",
		"journey", // Select journey workflow.
		"journeyproject",
		"y",
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "journeyproject", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Workflow", result.Workflow, "journey")
}

func TestRunInitPrompts_MultipleAgents(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"golang",
		"multiagent",
		"github.com/user/multiagent",
		"",
		"",
		"",
		"",
		"frd",
		"multiagent",
		"y",
		"claude,cursor,copilot",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "multiagent", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrSliceEqual(t, "Agents", result.Agents, []string{"claude", "cursor", "copilot"})
}

func TestRunInitPrompts_DockerDisabled(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"golang",
		"nodocker",
		"github.com/user/nodocker",
		"",
		"",
		"",
		"",
		"frd",
		"nodocker",
		"n", // Disable docker.
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "nodocker", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertBoolEqual(t, "Features.Docker", result.Features.Docker, false)
}

func TestRunInitPrompts_CustomBinaryName(t *testing.T) {
	t.Parallel()

	in := newTestInput(t,
		"golang",
		"myproject",
		"github.com/user/myproject",
		"",
		"",
		"",
		"",
		"frd",
		"custombin", // Different binary name from project name.
		"y",
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "myproject", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	if len(result.Binaries) != 1 {
		t.Fatalf("Binaries: got %d, want 1", len(result.Binaries))
	}

	assertStrEqual(t, "Binaries[0].Name", result.Binaries[0].Name, "custombin")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "./cmd/custombin")
}

func TestRunInitPrompts_ErrorOnEarlyEOF(t *testing.T) {
	t.Parallel()

	// Provide only one input, then EOF.
	in := newTestInput(t, "golang")

	cfg := config.Default()

	_, err := gatherConfig(cfg, "testproject", in, io.Discard, nil)
	if err == nil {
		t.Error("gatherConfig() should error when stdin closes before all prompts are answered")
	}
}

func TestRunInitPrompts_RustCustomBinaryPath(t *testing.T) {
	t.Parallel()

	// Verify that rust ecosystem uses src/main.rs as cmd path regardless of binary name.
	in := newTestInput(t,
		"rust",
		"rustbin",
		"github.com/user/rustbin",
		"",
		"",
		"2021",
		"deny",
		"frd",
		"customrust", // Custom binary name.
		"y",
		"claude",
		"n", // Generate skills.
	)

	cfg := config.Default()

	result, err := gatherConfig(cfg, "rustbin", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "Binaries[0].Name", result.Binaries[0].Name, "customrust")
	assertStrEqual(t, "Binaries[0].CmdPath", result.Binaries[0].CmdPath, "src/main.rs")
}

func TestRunInitPrompts_AnalysisCmdGolang(t *testing.T) {
	t.Parallel()

	testAnalysisCmdForEcosystem(t, "golang", []string{"1.22", "n"}, "go vet ./...")
}

func TestRunInitPrompts_AnalysisCmdRust(t *testing.T) {
	t.Parallel()

	testAnalysisCmdForEcosystem(t, "rust", []string{"2021", "deny"}, "cargo clippy -- -D warnings")
}

func TestRunInitPrompts_AnalysisCmdZig(t *testing.T) {
	t.Parallel()

	testAnalysisCmdForEcosystem(t, "zig", []string{"0.13", "n"}, "zig build test")
}

func testAnalysisCmdForEcosystem(t *testing.T, ecosystem string, ecoInputs []string, wantAnalCmd string) {
	t.Helper()

	lines := make([]string, 0, 5+len(ecoInputs)+4)
	lines = append(lines,
		ecosystem,
		ecosystem+"proj",
		"github.com/user/"+ecosystem+"proj",
		"",
		"",
	)
	lines = append(lines, ecoInputs...)
	lines = append(lines,
		"frd",
		ecosystem+"proj",
		"y",
		"claude",
		"n", // Generate skills.
	)

	in := newTestInput(t, lines...)
	cfg := config.Default()

	result, err := gatherConfig(cfg, ecosystem+"proj", in, io.Discard, nil)
	if err != nil {
		t.Fatalf("gatherConfig() error: %v", err)
	}

	assertStrEqual(t, "AnalysisCmd", result.AnalysisCmd, wantAnalCmd)
}
