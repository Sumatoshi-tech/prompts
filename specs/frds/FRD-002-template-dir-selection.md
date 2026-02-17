# FRD-002: Ecosystem-Aware Template Directory Selection

## Summary

Replace the hardcoded `templateDir = "templates/golang"` constant with a function that selects `templates/<ecosystem>/` based on `cfg.Ecosystem`. All rendering functions must use the ecosystem-specific directory.

## Requirements

### Must Have
- `TemplateDirForEcosystem(ecosystem)` returns `"templates/<ecosystem>"`
- `Render()` uses `cfg.Ecosystem` to determine template directory
- `RenderSingle()` uses `cfg.Ecosystem` for template lookup
- `CheckOverrideStaleness()` uses ecosystem-specific template paths
- Rendering with `ecosystem: "golang"` produces identical output to current behavior
- Missing template directory produces a clear error

### Should Have
- `TemplateDirForEcosystem` is exported for use by CLI template commands

### Won't Have
- Actual Rust/Zig template files (Step 4-5)

## Test Matrix

| Scenario | Expected |
|---|---|
| Render with golang ecosystem | Identical to current output |
| Render with rust ecosystem | Looks in `templates/rust/` |
| TemplateDirForEcosystem("golang") | Returns `"templates/golang"` |
| TemplateDirForEcosystem("rust") | Returns `"templates/rust"` |
| TemplateDirForEcosystem("zig") | Returns `"templates/zig"` |
| RenderSingle with golang | Finds template in `templates/golang/` |
| CheckOverrideStaleness with golang | Checks `templates/golang/` |

## Acceptance Criteria

- All existing tests pass unchanged
- `make test` and `make lint` pass
