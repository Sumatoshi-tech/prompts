# Commands Reference

## `promptkit init [project-dir]`

Scaffold a new project. Interactive by default; use `--non-interactive` for scripted usage.

```bash
promptkit init my-project
promptkit init my-project --non-interactive --name "myapp" --module "github.com/user/myapp"
promptkit init my-project --dry-run    # Preview without writing
promptkit init --force                 # Re-initialize existing project
```

### General Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--name` | Project name | Directory name |
| `--module` | Module path (e.g. github.com/user/project) | -- |
| `--ecosystem` | Template ecosystem (`golang`, `rust`, `zig`) | `golang` |
| `--workflow` | Development workflow (`frd`, `journey`) | `frd` |
| `--description` | Project description | -- |
| `--expertise` | Agent domain expertise | -- |
| `--binary` | Binary name | Project name |
| `--docker` | Enable Docker support | `true` |
| `--ai` | Target AI agents (comma-separated) | `claude` |
| `--non-interactive` | Use flags instead of prompts | `false` |
| `--dry-run` | Preview files without writing | `false` |
| `--force, -f` | Overwrite existing project | `false` |

### Go Ecosystem Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--go-version` | Go version | `1.22` |
| `--cgo` | Enable CGO support | `false` |

### Rust Ecosystem Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--rust-edition` | Rust edition (e.g. `2021`, `2024`) | `2021` |
| `--unsafe-policy` | Unsafe code policy (`forbid`, `deny`, `warn`, `allow`) | `deny` |

### Zig Ecosystem Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--zig-version` | Zig version (e.g. `0.13`) | `0.13` |
| `--link-libc` | Link libc for C interop | `false` |

## `promptkit update`

Re-render all templates after editing `.promptkit.yaml`. Shows a unified diff of every changed file before writing.

```bash
vim .promptkit.yaml          # Change quality thresholds, add an agent, etc.
promptkit update             # Review diff, approve
promptkit update --yes       # Auto-approve (CI-friendly)
promptkit update --dry-run   # Preview only, exit 1 if drift detected
promptkit update -i          # Per-file approval (y/n/a/q)
promptkit update --verify    # Run analysis command after applying
promptkit update --explain   # Print pipeline steps before running
```

| Flag | Description |
|------|-------------|
| `--yes, -y` | Auto-approve all changes |
| `--dry-run` | Preview without writing; exit 1 if changes pending |
| `--interactive, -i` | Per-file approval (y/n/a=all/q=quit) |
| `--verify` | Run analysis command after applying |
| `--explain` | Show pipeline description before running |

### Update Pipeline

1. Load and validate `.promptkit.yaml`
2. Render embedded templates using config values
3. Apply local overrides from `.promptkit/templates/`
4. Generate agent-specific adapter files
5. Compute diffs against files on disk
6. Show unified diffs with config-key annotations
7. Prompt for approval
8. Back up existing files, write atomically
9. Remove stale files, update manifest and checksums

## `promptkit diff`

Preview what `update` would change. Alias for `update --dry-run`.

```bash
promptkit diff                         # Compare rendered vs. on-disk
promptkit diff --upstream /path/to/ref # Compare local vs. reference config
```

The `--upstream` flag renders both configs through the full pipeline and shows file-level differences -- useful for comparing team configurations or detecting drift.

## `promptkit status`

Check if generated files are up to date. Exits 0 if current, 1 if drift detected.

```bash
promptkit status   # Use in CI to catch config drift
```

## `promptkit doctor`

Validate generated files for correctness.

```bash
promptkit doctor
```

```
promptkit doctor:
  [ok] Config loads and validates
  [ok] 24/24 generated files exist
  [ok] claude: 8/8 agent files present
  [ok] cursor: 5/5 agent files present
  [warn] AGENTS.md: modified since last generation
  [ok] No stale files
```

Checks:
- Config parses and validates
- All generated files exist on disk
- Agent-specific files are present for each configured agent
- Checksums match (warns on manually modified files)
- No stale files from previous configurations
- Override staleness (warns if upstream template changed)

## `promptkit clean`

Remove stale generated files -- files that were previously generated but are no longer produced by the current config.

```bash
promptkit clean              # List stale files, prompt for confirmation
promptkit clean --dry-run    # List without removing
promptkit clean --yes        # Remove without prompting
```

## `promptkit template`

Manage template overrides.

```bash
promptkit template list               # List all available templates
promptkit template render AGENTS.md   # Render a single template to stdout
promptkit template extract AGENTS.md  # Extract embedded template for customization
promptkit template add Makefile ./my-makefile.tmpl   # Add override from file
promptkit template vars               # List available template variables with types
```

### `template list`

Lists all embedded templates and their output file names.

### `template render <name>`

Renders a single template using current config and prints to stdout. Useful for debugging templates and overrides.

### `template extract <name>`

Copies the embedded template source to `.promptkit/templates/<name>.tmpl` for local customization. The extracted file becomes a local override.

```bash
promptkit template extract AGENTS.md           # Extract for editing
promptkit template extract AGENTS.md --force   # Overwrite existing override
```

Records the upstream template checksum so `update` can warn if the embedded template changes in a future promptkit version.

### `template add <name> <source-file>`

Installs a file as a local template override.

### `template vars`

Lists all variables available in templates with types and descriptions. See [Template Variables](templates.md#template-variables) for the full reference.

## `promptkit config explain [key]`

Show which output files are affected by each config field.

```bash
promptkit config explain                      # Show all mappings
promptkit config explain quality.coverage_min # Show specific field
```

```
  quality.coverage_min
    Minimum test coverage percentage
    Affects: AGENTS.md
```
