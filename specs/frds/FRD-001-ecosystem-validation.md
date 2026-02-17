# FRD-001: Ecosystem Validation in Config

**Roadmap Link:** specs/multi-ecosystem-support/roadmap.md — Step 1
**Date:** 2026-02-17

## Problem

The `ecosystem` field in `.promptkit.yaml` is stored and read but never validated. Any value (including typos like "goland" or unsupported languages like "python") passes silently. This must be validated before multi-ecosystem template rendering can work.

## Context

The config package already validates agents using a `ValidAgents` map, `ValidAgentNames()` sorted list, and `closestAgent()` did-you-mean using `editDistance()`. The ecosystem validation should follow the same proven pattern.

## Goal

Validate the `ecosystem` field during `config.Validate()`, rejecting unknown values with clear error messages and did-you-mean suggestions.

## In Scope

- `ValidEcosystems` map with `golang`, `rust`, `zig`
- `ValidEcosystemNames()` returning sorted list
- `EcosystemDescriptions` map with human-readable descriptions
- Ecosystem constants (`EcosystemGolang`, `EcosystemRust`, `EcosystemZig`)
- `closestEcosystem()` using existing `editDistance()` for typo suggestions
- Validation in `Validate()` rejecting unknown ecosystems
- Update `MarshalCommented()` ecosystem comment to list all valid values

## Out of Scope

- Ecosystem-specific config fields (Step 3)
- Template directory selection (Step 2)
- CLI flags (Step 6)
- Rust/Zig templates (Steps 4-5)

## Functional Requirements

### Must

- M1: `Validate()` returns error for unknown ecosystem values
- M2: Error message includes the invalid value and lists all valid ecosystems
- M3: Did-you-mean suggestion when ecosystem is within edit distance 3
- M4: Empty ecosystem field is treated as invalid (Default() sets it to "golang")
- M5: `golang`, `rust`, `zig` are the only valid ecosystems
- M6: `ValidEcosystems` map and `ValidEcosystemNames()` are exported
- M7: Backward compatible — existing configs with `ecosystem: golang` pass unchanged

### Should

- S1: `EcosystemDescriptions` map provides human-readable descriptions for each ecosystem
- S2: Error format matches agent validation: `ecosystem: unknown ecosystem "X" (valid: golang, rust, zig); did you mean "Y"?`

### Could

- C1: Ecosystem constants could be used throughout codebase instead of string literals

### Won't

- W1: No ecosystem-conditional validation of other fields in this step (deferred to Step 3)

## Stressors

1. Existing configs without `ecosystem` field — Default() sets "golang", so they pass
2. Typo in ecosystem name — did-you-mean handles this
3. Case sensitivity — "Golang" vs "golang" — validation is case-sensitive, matches agent pattern
4. Empty string ecosystem — validation catches it
5. Very long garbage string — editDistance returns > 3, no suggestion shown
6. Multiple validation errors — ecosystem error joins with other errors via errors.Join
7. YAML decode with `KnownFields(true)` already rejects misspelled `ecosystem` key
8. Config round-trip: Save then Load must preserve ecosystem value
9. Migration: v0 configs get ecosystem "golang" from Default() merge
10. MarshalCommented must reflect updated valid values comment

## Residue-First Design

- **Modularity:** Ecosystem validation follows the same pattern as agent validation — no new patterns introduced
- **Simplicity:** Reuses editDistance(), follows existing map+sorted-names pattern
- **Defensiveness:** Empty ecosystem caught, unknown values caught, close matches suggested
- **Observability:** Error messages include the invalid value and valid options
- **Reversibility:** Pure validation addition — no data migration, no config schema change

## Acceptance Criteria

1. `cfg.Ecosystem = "rust"` passes validation
2. `cfg.Ecosystem = "python"` fails with `unknown ecosystem "python" (valid: golang, rust, zig)`
3. `cfg.Ecosystem = "rustt"` fails with did-you-mean "rust"
4. `cfg.Ecosystem = ""` fails validation
5. `cfg.Ecosystem = "golang"` passes (backward compatible)
6. Existing test suite passes without changes
7. `make lint` and `make test` clean

## Test Matrix

| Input | Expected Error | Did-You-Mean |
|-------|---------------|-------------|
| "golang" | none | n/a |
| "rust" | none | n/a |
| "zig" | none | n/a |
| "python" | unknown ecosystem | no (distance > 3) |
| "rustt" | unknown ecosystem | "rust" |
| "goland" | unknown ecosystem | "golang" |
| "zi" | unknown ecosystem | "zig" |
| "" | unknown ecosystem | no |
| "GOLANG" | unknown ecosystem | "golang" |
| "zzzzzzz" | unknown ecosystem | no |

## Risks and Mitigation

| Risk | Mitigation |
|------|-----------|
| Breaking existing configs | Default() already sets "golang"; validation only adds new check |
| KnownFields rejects new YAML keys | No new YAML keys in this step |

## Rollback

Revert the additions to `config.go` and `config_test.go`. No data migration involved.

## Implementation Checklist

- [ ] Add ecosystem constants
- [ ] Add ValidEcosystems map
- [ ] Add ValidEcosystemNames()
- [ ] Add EcosystemDescriptions map
- [ ] Add closestEcosystem()
- [ ] Add ecosystem validation to Validate()
- [ ] Update MarshalCommented() ecosystem comment
- [ ] Write tests for all matrix entries
- [ ] Run make lint, make test
