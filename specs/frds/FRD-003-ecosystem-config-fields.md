# FRD-003: Ecosystem-Specific Config Fields

## Summary

Add Rust-specific (`RustEdition`, `UnsafePolicy`) and Zig-specific (`ZigVersion`, `LinkLibc`) fields to Config. Make validation ecosystem-conditional: `go_version` only required for golang, `rust_edition` for rust. Update `Default()`, `MarshalCommented()`, and `Migrate()`.

## Requirements

### Must Have
- `RustEdition` (string) and `UnsafePolicy` (string) fields in Config
- `ZigVersion` (string) and `LinkLibc` (bool) fields in Config
- `go_version` validated only when `ecosystem: golang`
- `rust_edition` validated when `ecosystem: rust`
- `UnsafePolicy` validated against: `forbid`, `deny`, `warn`, `allow`
- `Default()` sets ecosystem-appropriate `AnalysisCmd`
- `MarshalCommented()` emits ecosystem-appropriate fields

### Won't Have
- Ecosystem-specific binary/quality changes (those remain universal)

## Test Matrix

| Scenario | Expected |
|---|---|
| Rust config with valid edition | Passes validation |
| Rust config missing go_version | Passes (not required) |
| Golang config missing go_version | Fails validation |
| Invalid unsafe_policy | Fails with valid list |
| Zig config with link_libc | Passes validation |
| MarshalCommented for rust | Contains rust-specific fields |
| MarshalCommented for golang | Contains go-specific fields |

## Acceptance Criteria
- Existing Go config round-trips identically
- `make test` and `make lint` pass
