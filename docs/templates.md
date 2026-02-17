# Templates

Each ecosystem (Go, Rust, Zig) has its own set of templates that produce ecosystem-appropriate configs, Makefiles, and instructions. Templates use Go `text/template` syntax and are embedded in the promptkit binary. See [Commands](commands.md#promptkit-template) for `template` subcommands and [Workflows](workflows.md) for how skills fit into development methodologies.

## Generated Files

### AGENTS.md

The core agent personality document. Defines:
- Identity, expertise, and non-negotiables
- 14-step working loop (read docs, write FRD/journey, TDD, lint, iterate)
- Micro-TDD development flow with loop contract
- E2E testing philosophy
- Architecture preferences and code patterns
- Residuality-based development (5-step resilience framework)
- Quality gates with configurable thresholds
- Checklist before commit

### Instruction Skills

| Skill | File | Workflow | Purpose |
|-------|------|----------|---------|
| `/implement` | `instr-implement.md` | both | Iterative TDD implementation following a roadmap. 16-step workflow with micro-TDD loop contract (Plan, RED, GREEN, Reflect, Refactor, Verify). Small-change fast path for trivial fixes. Cross-skill traceability. |
| `/roadmap` | `instr-roadmaper.md` | both | Decompose a specification into a progressive, testable roadmap with DoD/DoR per step. Detects existing implementations. Update mode for re-syncing with codebase changes. |
| `/frd` | `instr-frd.md` | frd | Feature Requirements Document template. MoSCoW format, stressor scenarios, acceptance criteria, test matrix. |
| `/journey` | `instr-journey.md` | journey | Journey-based feature requirements with CJM. Phases, friction analysis, UX assessment, test cases. |
| `/perf` | `instr-perf.md` | both | Performance diagnosis across 5 phases. Bottleneck classification (Class A-E). Platform-aware: Linux (perf) and macOS (Instruments/DTrace). |

### Ecosystem-Specific Files

| Ecosystem | Linter Config | Build System | Analysis Command |
|-----------|---------------|--------------|------------------|
| Go | `.golangci.yml` (golangci-lint v2) | Makefile | `go vet ./...` |
| Rust | Clippy/rustfmt configs | Makefile (cargo) | `cargo clippy -- -D warnings` |
| Zig | -- | Makefile (zig build) | `zig build test` |

## Template Overrides

Customize any generated file by creating a local override:

```bash
# 1. See what's available
promptkit template list

# 2. Extract the template you want to customize
promptkit template extract AGENTS.md

# 3. Edit the extracted template
vim .promptkit/templates/AGENTS.md.tmpl

# 4. Re-render with your customizations
promptkit update
```

Overrides use Go `text/template` syntax with full access to all config fields. Available functions: `join`, `upper`, `lower`, `title`.

```
# In your template override:
Project: {{.ProjectName}}
Module:  {{.ModulePath}}
Ecosystem: {{.Ecosystem}}
{{if eq .Workflow "journey"}}Using journey workflow{{end}}
{{range .Binaries}}Binary: {{.Name}} at {{.CmdPath}}
{{end}}
```

Overrides survive promptkit version upgrades. If an embedded template changes upstream, `promptkit update` warns you so you can review and re-extract if needed.

## Template Variables

Run `promptkit template vars` for the full list. Key variables:

```
Available template variables (accessed as .FieldName):
  .ProjectName              string     Project name used in AGENTS.md
  .ModulePath               string     Module path
  .Ecosystem                string     Template ecosystem (golang, rust, zig)
  .Workflow                 string     Development workflow (frd, journey)
  .GoVersion                string     Go version (Go ecosystem)
  .RustEdition              string     Rust edition (Rust ecosystem)
  .UnsafePolicy             string     Unsafe policy (Rust ecosystem)
  .ZigVersion               string     Zig version (Zig ecosystem)
  .Quality                  struct     Code quality thresholds
    .Quality.CoverageMin      int
    .Quality.CoverageCritical int
  .Features                 struct     Feature flags
    .Features.CGO             bool
    .Features.Docker          bool
  .Agents                   []string   Target AI agents
  ...
```
