package scaffold_test

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"strings"
	"testing"

	promptkit "github.com/Sumatoshi-tech/promptkit"
	"github.com/Sumatoshi-tech/promptkit/internal/config"
	"github.com/Sumatoshi-tech/promptkit/internal/scaffold"
)

// benchConfig returns a representative config used across all benchmarks.
func benchConfig() *config.Config {
	return &config.Config{
		ProjectName: "benchproject",
		ModulePath:  "github.com/user/benchproject",
		GoVersion:   "1.22",
		Description: "A benchmark test project for performance profiling",
		Expertise:   "distributed systems",
		IdentityYrs: 15,
		Binaries: []config.Binary{
			{Name: "benchproject", CmdPath: "./cmd/benchproject"},
			{Name: "benchctl", CmdPath: "./cmd/benchctl"},
		},
		Quality: config.Quality{
			CoverageMin:      85,
			CoverageCritical: 90,
			ComplexityMax:    15,
			LineLength:       140,
		},
		Features: config.Features{
			CGO:        true,
			Docker:     true,
			Benchmarks: true,
			CGOLibs: []config.CGOLib{
				{Name: "mylib", PkgConfig: "mylib", Include: "third_party/mylib/include", LibDir: "third_party/mylib/lib"},
			},
		},
		Agents:    []string{"claude", "codex", "copilot", "cursor", "gemini", "windsurf"},
		Ecosystem: "golang",
	}
}

// BenchmarkRender measures raw template rendering (no agent adapters).
func BenchmarkRender(b *testing.B) {
	cfg := benchConfig()

	// Warm-up (discard).
	_, err := scaffold.Render(cfg, promptkit.Templates)
	if err != nil {
		b.Fatalf("warm-up Render: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		result, renderErr := scaffold.Render(cfg, promptkit.Templates)
		if renderErr != nil {
			b.Fatalf("Render: %v", renderErr)
		}

		if len(result) == 0 {
			b.Fatal("Render returned empty result")
		}
	}
}

// BenchmarkRenderFull measures template rendering + agent adapter placement + provenance.
func BenchmarkRenderFull(b *testing.B) {
	cfg := benchConfig()

	// Warm-up.
	_, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		b.Fatalf("warm-up RenderFull: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		result, renderErr := scaffold.RenderFull(cfg, promptkit.Templates)
		if renderErr != nil {
			b.Fatalf("RenderFull: %v", renderErr)
		}

		if len(result) == 0 {
			b.Fatal("RenderFull returned empty result")
		}
	}
}

// BenchmarkComputeChecksums measures SHA-256 checksum computation over rendered files.
func BenchmarkComputeChecksums(b *testing.B) {
	cfg := benchConfig()

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		b.Fatalf("setup RenderFull: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		checksums := scaffold.ComputeChecksums(rendered)
		if len(checksums) == 0 {
			b.Fatal("ComputeChecksums returned empty")
		}
	}
}

// BenchmarkUnifiedDiff measures the LCS-based diff algorithm.
func BenchmarkUnifiedDiff(b *testing.B) {
	cfg := benchConfig()

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		b.Fatalf("setup RenderFull: %v", err)
	}

	// Create a modified version of AGENTS.md to diff against.
	var agentsContent []byte

	for path, content := range rendered {
		if path == "AGENTS.md" {
			agentsContent = content

			break
		}
	}

	if agentsContent == nil {
		b.Fatal("AGENTS.md not found in rendered output")
	}

	// Simulate a realistic diff: change ~10% of lines.
	lines := strings.Split(string(agentsContent), "\n")
	modified := make([]string, len(lines))
	copy(modified, lines)

	for i := 0; i < len(modified); i += 10 {
		modified[i] += " [modified]"
	}

	modifiedContent := []byte(strings.Join(modified, "\n"))

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		diff := scaffold.UnifiedDiff(agentsContent, modifiedContent, "AGENTS.md")
		if diff == "" {
			b.Fatal("UnifiedDiff returned empty for modified content")
		}
	}
}

// BenchmarkUnifiedDiff_LargeFile measures diff performance on a large synthetic file.
func BenchmarkUnifiedDiff_LargeFile(b *testing.B) {
	// Create a 1000-line file to stress the LCS algorithm.
	oldLines := make([]string, 0, 1000)

	for i := range 1000 {
		oldLines = append(oldLines, fmt.Sprintf("line %d: some content here that is representative", i))
	}

	// Modify ~10% of lines, insert a few, delete a few.
	newLines := make([]string, 0, 1050)

	for i := range 1000 {
		if i%100 == 50 {
			newLines = append(newLines, fmt.Sprintf("inserted line near %d", i))
		}

		if i%100 == 75 {
			continue
		}

		if i%10 == 0 {
			newLines = append(newLines, fmt.Sprintf("line %d: MODIFIED content here that is representative", i))
		} else {
			newLines = append(newLines, oldLines[i])
		}
	}

	oldContent := []byte(strings.Join(oldLines, "\n") + "\n")
	newContent := []byte(strings.Join(newLines, "\n") + "\n")

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		diff := scaffold.UnifiedDiff(oldContent, newContent, "large.txt")
		if diff == "" {
			b.Fatal("UnifiedDiff returned empty for modified content")
		}
	}
}

// BenchmarkAddProvenance measures the provenance header prepending.
func BenchmarkAddProvenance(b *testing.B) {
	cfg := benchConfig()

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}

	// Copy so we can re-add provenance.
	stripped := make(map[string][]byte, len(rendered))
	maps.Copy(stripped, rendered)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		result := scaffold.AddProvenance(stripped)
		if len(result) == 0 {
			b.Fatal("AddProvenance returned empty")
		}
	}
}

// BenchmarkSHA256 measures raw SHA-256 cost to understand baseline.
func BenchmarkSHA256(b *testing.B) {
	cfg := benchConfig()

	rendered, err := scaffold.RenderFull(cfg, promptkit.Templates)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}

	// Collect all content into a single slice for raw hashing.
	allContent := make([][]byte, 0, len(rendered))

	for _, content := range rendered {
		allContent = append(allContent, content)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		for _, content := range allContent {
			_ = sha256.Sum256(content)
		}
	}
}
