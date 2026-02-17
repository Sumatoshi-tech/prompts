# Configuration Reference

All project-specific values live in `.promptkit.yaml`.

## Example

```yaml
# promptkit configuration -- edit this file, then run: promptkit update

# Config schema version (do not edit)
version: 2

# Project identity
project_name: myapp
module_path: github.com/user/myapp
description: A CLI tool for awesome things
expertise: distributed systems and API design
identity_years: 15

# Build targets
binaries:
  - name: myapp
    cmd_path: ./cmd/myapp

# Code quality thresholds -- enforced in AGENTS.md, linter config, and Makefile
quality:
  coverage_min: 85         # Minimum test coverage (1-100)
  coverage_critical: 90    # Critical path coverage (1-100, >= coverage_min)
  complexity_max: 15       # Max cyclomatic complexity
  line_length: 140         # Max line length

# Feature flags -- control conditional sections in Makefile and templates
features:
  cgo: false
  docker: true
  benchmarks: true

# Target AI agents (valid: claude, codex, copilot, cursor, gemini, windsurf)
agents:
  - claude
  - cursor

# Template ecosystem (golang, rust, zig)
ecosystem: golang

# Development workflow (frd, journey)
workflow: frd

# Ecosystem-specific fields (shown for Go; Rust and Zig have their own)
go_version: "1.23"

# Analysis command for --verify (ecosystem-specific default)
analysis_command: go vet ./...

# Directory for local template overrides
template_overrides: .promptkit/templates
```

## Field Reference

### Core Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `version` | int | `2` | Schema version (auto-migrated) |
| `project_name` | string | -- | **Required.** Project name used in AGENTS.md and Makefile |
| `module_path` | string | -- | **Required.** Module path (e.g. github.com/org/project) |
| `description` | string | -- | Project description for AGENTS.md |
| `expertise` | string | -- | Domain expertise for agent personality |
| `identity_years` | int | `15` | Years of experience in agent identity statement |
| `binaries` | list | -- | **Required (min 1).** Each entry: `name` + `cmd_path` |
| `agents` | list | `[claude]` | **Required (min 1).** Target AI agents |
| `ecosystem` | string | `golang` | Template ecosystem (`golang`, `rust`, `zig`) |
| `workflow` | string | `frd` | Development workflow (`frd`, `journey`) |
| `analysis_command` | string | varies | Command run by `--verify` (ecosystem-specific default) |
| `template_overrides` | string | `.promptkit/templates` | Override directory path |

### Quality

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `quality.coverage_min` | int | `85` | Minimum test coverage percentage (1-100) |
| `quality.coverage_critical` | int | `90` | Critical path coverage (1-100, >= coverage_min) |
| `quality.complexity_max` | int | `15` | Max cyclomatic complexity per function |
| `quality.line_length` | int | `140` | Max line length |

### Features

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `features.cgo` | bool | `false` | Enable CGO support (Go ecosystem) |
| `features.docker` | bool | `true` | Enable Docker targets in Makefile |
| `features.benchmarks` | bool | `true` | Enable benchmark targets in Makefile |
| `features.cgo_libs` | list | `[]` | CGO library dependencies (Go ecosystem) |

### Go Ecosystem Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `go_version` | string | `1.22` | Go version for linter config and build targets |

### Rust Ecosystem Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rust_edition` | string | `2021` | Rust edition for code generation |
| `unsafe_policy` | string | `deny` | Unsafe code policy (`forbid`, `deny`, `warn`, `allow`) |

### Zig Ecosystem Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `zig_version` | string | `0.13` | Zig version for build configuration |
| `link_libc` | bool | `false` | Link libc for C interop |

## Validation

promptkit validates config on every load:

- Required fields must be non-empty
- Quality ranges enforced (1-100 for coverage, positive for complexity/line length)
- `coverage_critical >= coverage_min`
- Ecosystem and workflow must be valid (typos get did-you-mean suggestions)
- All agent names must be valid (typos get did-you-mean suggestions)
- Ecosystem-specific fields validated by their modules
- Unknown YAML keys are rejected (catches misspellings like `agnets`)
- Merge conflict markers detected with actionable error message

## Migration

When loading a config from an older schema version, promptkit automatically applies forward migrations (filling in missing fields with sensible defaults) and saves the updated config.

### Migration History

| Version | Changes |
|---------|---------|
| 0 -> 1 | Added `analysis_command`, `template_overrides` |
| 1 -> 2 | Added `workflow` (defaults to `frd`) |

## Field-to-File Mapping

Use [`promptkit config explain`](commands.md#promptkit-config-explain-key) to see which output files are affected by each config field:

```bash
promptkit config explain                      # Show all mappings
promptkit config explain quality.coverage_min # Show specific field
```
