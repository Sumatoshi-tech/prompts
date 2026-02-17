# Multi-Ecosystem Support Roadmap

Source: `journeys/multi-ecosystem-support/journey.md`

## Overview

Make promptkit ecosystem-aware so that `ecosystem` field in `.promptkit.yaml` selects the template directory (`templates/golang/`, `templates/rust/`, `templates/zig/`), and all downstream rendering, validation, defaults, and CLI prompts adapt accordingly.

---

## Step 1: Ecosystem Validation in Config

**Description:** Add ecosystem validation to `config.Validate()` so that unknown ecosystem values are rejected with a clear error listing valid options (`golang`, `rust`, `zig`). Add `ValidEcosystems` map, `ValidEcosystemNames()`, `closestEcosystem()` with did-you-mean suggestions. Update `Default()` to keep `golang` default. Add ecosystem-aware `AnalysisCmd` defaults in `Migrate()`.

**Definition of Ready:**
- Config struct already has `Ecosystem` field
- Existing validation pattern (agents) provides the template for ecosystem validation
- editDistance() function exists for did-you-mean

**Definition of Done:**
- [ ] `config.Validate()` rejects unknown ecosystem values with error: `unknown ecosystem "X" (valid: golang, rust, zig)`
- [ ] Did-you-mean suggestions work for close misspellings (e.g., "rustt" -> "rust")
- [ ] `ValidEcosystems` map and `ValidEcosystemNames()` exported
- [ ] `EcosystemDescriptions` map provides human-readable descriptions
- [ ] `Default()` returns `ecosystem: "golang"` (backward compatible)
- [ ] All existing tests pass unchanged
- [ ] New tests cover: valid ecosystems, invalid ecosystem, did-you-mean, empty ecosystem
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-001-ecosystem-validation.md
**Implementation:** internal/config/config.go, internal/config/config_test.go

---

## Step 2: Ecosystem-Aware Template Directory Selection

**Description:** Replace the hardcoded `templateDir = "templates/golang"` constant in `scaffold.go` with a function that selects the template directory based on `cfg.Ecosystem`. All rendering functions (`Render`, `RenderWithOverrides`, `RenderFull`, etc.) must use the ecosystem-specific directory. `RenderSingle` and `CheckOverrideStaleness` must also be updated.

**Definition of Ready:**
- Step 1 complete (ecosystem field validated)
- Scaffold rendering pipeline understood
- Only `templates/golang/` exists currently -- Rust/Zig templates not yet needed for this step (tests use Go ecosystem)

**Definition of Done:**
- [ ] `templateDir` is no longer a constant; replaced by `templateDirForEcosystem(ecosystem string)` function
- [ ] All `Render*` functions accept or derive the ecosystem from config
- [ ] `RenderSingle` and `CheckOverrideStaleness` use ecosystem-specific template paths
- [ ] Rendering with `ecosystem: "golang"` produces identical output to current behavior
- [ ] Rendering with `ecosystem: "rust"` looks for `templates/rust/` directory
- [ ] Missing template directory produces a clear error, not empty output
- [ ] All existing tests pass (backward compatible)
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-002-template-dir-selection.md
**Implementation:** internal/scaffold/scaffold.go, internal/scaffold/scaffold_test.go

---

## Step 3: Ecosystem-Specific Config Fields (Rust & Zig)

**Description:** Add Rust-specific and Zig-specific fields to the Config struct: `RustEdition`, `UnsafePolicy` for Rust; `ZigVersion`, `LinkLibc` for Zig. Update `Validate()` with ecosystem-conditional validation (e.g., `go_version` only required when `ecosystem: golang`). Update `Default()` with ecosystem-aware defaults. Update `MarshalCommented()` to emit ecosystem-appropriate comments and fields. Update `Migrate()` for ecosystem-aware analysis command defaults.

**Definition of Ready:**
- Step 1 complete (ecosystem validation)
- Config struct pattern understood

**Definition of Done:**
- [ ] `RustEdition` (string), `UnsafePolicy` (string) fields added to Config
- [ ] `ZigVersion` (string), `LinkLibc` (bool) fields added to Config
- [ ] `Validate()` checks `go_version` only when `ecosystem: golang`
- [ ] `Validate()` checks `rust_edition` when `ecosystem: rust`
- [ ] `UnsafePolicy` validated against valid values: `forbid`, `deny`, `warn`, `allow`
- [ ] `Default()` sets ecosystem-appropriate defaults for analysis_command
- [ ] `MarshalCommented()` emits ecosystem-appropriate field comments
- [ ] Existing Go config round-trips identically (backward compatible)
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-003-ecosystem-config-fields.md
**Implementation:** internal/config/config.go, internal/config/config_test.go, internal/config/migrate.go

---

## Step 4: Rust Template Set

**Description:** Create `templates/rust/` with ecosystem-appropriate templates: `AGENTS.md.tmpl`, `Makefile.tmpl`, `clippy.toml.tmpl`, `rustfmt.toml.tmpl`, and all four instruction skill templates (`instr-implement.md.tmpl`, `instr-roadmaper.md.tmpl`, `instr-frd.md.tmpl`, `instr-perf.md.tmpl`). Templates reference Rust tooling exclusively. No Go references.

**Definition of Ready:**
- Steps 1-3 complete
- Go templates in `templates/golang/` serve as structural reference

**Definition of Done:**
- [ ] `templates/rust/` directory exists with all template files
- [ ] `AGENTS.md.tmpl` references cargo, clippy, rustfmt, Rust edition, ownership/borrowing
- [ ] `Makefile.tmpl` has targets: build, test, clippy, fmt, clean, audit, plus optional Docker
- [ ] `clippy.toml.tmpl` reflects quality thresholds and unsafe policy
- [ ] `rustfmt.toml.tmpl` reflects line length configuration
- [ ] Instruction skills reference `cargo test`, `cargo clippy`, `cargo tarpaulin`
- [ ] No Go-specific references in any Rust template
- [ ] Templates render without errors against a valid Rust config
- [ ] No unresolved template markers (`{{` or `}}`) in rendered output
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-004-rust-templates.md
**Implementation:** templates/rust/*.tmpl, templates.go (embed already covers all:templates)

---

## Step 5: Zig Template Set

**Description:** Create `templates/zig/` with ecosystem-appropriate templates: `AGENTS.md.tmpl`, `Makefile.tmpl`, and all four instruction skill templates. Templates reference Zig tooling exclusively. No Go or Rust references.

**Definition of Ready:**
- Steps 1-3 complete
- Rust templates (Step 4) serve as structural reference for non-Go ecosystem

**Definition of Done:**
- [ ] `templates/zig/` directory exists with all template files
- [ ] `AGENTS.md.tmpl` references zig build, zig test, zig fmt, comptime, allocators, error unions
- [ ] `Makefile.tmpl` has targets: build, test, fmt, clean, plus optional Docker
- [ ] Instruction skills reference `zig build test`, `zig fmt`, allocator profiling
- [ ] No Go-specific or Rust-specific references in any Zig template
- [ ] Templates render without errors against a valid Zig config
- [ ] No unresolved template markers in rendered output
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-005-zig-templates.md
**Implementation:** templates/zig/*.tmpl

---

## Step 6: CLI Ecosystem Flag and Interactive Prompts

**Description:** Add `--ecosystem` flag to `promptkit init`. Update interactive prompts to ask ecosystem first, then adapt subsequent prompts based on selection. Rust prompts: crate name, edition, unsafe policy. Zig prompts: module name, version, link-libc. Update non-interactive mode to accept `--rust-edition`, `--unsafe-policy`, `--zig-version`, `--link-libc` flags.

**Definition of Ready:**
- Steps 1-5 complete (config validated, templates exist)
- CLI init.go and prompt.go patterns understood

**Definition of Done:**
- [ ] `--ecosystem` flag available on `promptkit init`
- [ ] Interactive mode prompts for ecosystem early in the sequence
- [ ] Subsequent prompts adapt to selected ecosystem
- [ ] Non-interactive mode accepts ecosystem-specific flags
- [ ] `--ecosystem golang` behavior identical to current (no `--ecosystem` flag)
- [ ] `--ecosystem rust` with valid flags produces a Rust scaffold
- [ ] `--ecosystem zig` with valid flags produces a Zig scaffold
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-006-cli-ecosystem.md
**Implementation:** internal/cli/init.go, internal/prompt/prompt.go

---

## Step 7: Integration Tests

**Description:** End-to-end tests that scaffold complete projects for each ecosystem and verify: correct files generated, no cross-contamination, config round-trip, agent adapters work, stale detection on ecosystem switch.

**Definition of Ready:**
- Steps 1-6 complete
- All unit tests pass

**Definition of Done:**
- [ ] E2E test: init with --ecosystem rust produces Rust scaffold
- [ ] E2E test: init with --ecosystem zig produces Zig scaffold
- [ ] E2E test: init with --ecosystem golang identical to current
- [ ] E2E test: no cross-ecosystem file contamination
- [ ] E2E test: config round-trip (init then update = zero diffs) for all ecosystems
- [ ] E2E test: stale file detection on ecosystem switch
- [ ] E2E test: agent adapters produce ecosystem-flavored content
- [ ] Coverage >= 85% overall, >= 90% on config and scaffold packages
- [ ] `make lint` clean, `make test` pass

**FRD:** specs/frds/FRD-007-integration-tests.md
**Implementation:** integration_test.go (extend existing)
