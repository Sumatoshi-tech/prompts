# Workflows

promptkit supports two spec-driven development methodologies. Both follow the same overall flow but differ in how feature requirements are documented. See [Templates](templates.md) for skill details and [Configuration](configuration.md) for the `workflow` field.

## FRD Workflow (default)

```
Research --> Spec --> /roadmap --> /frd --> /implement (iterative) --> /perf
                        |                      |
                        v                      v
                   specs/{name}/         specs/frds/FRD-{id}.md
                   roadmap.md            (created by /implement)
```

1. **Research** the problem domain
2. **Write a specification** in `specs/`
3. **`/roadmap`** -- Decompose the spec into a progressive roadmap with testable steps
4. **`/implement`** -- For each roadmap item: write FRD, write tests first, implement, lint, iterate
5. **`/perf`** -- Profile, classify bottleneck, optimize with evidence

Each step produces traceable artifacts. FRDs link back to roadmap items. Implementation files link back to FRDs.

### FRD Contents

- MoSCoW prioritization
- 10+ stressor scenarios
- Residue-first design
- Acceptance criteria
- Test matrix
- Risk mitigation and rollback strategy
- Traceability links

## Journey Workflow

```
Research --> Spec --> /roadmap --> /journey --> /implement (iterative) --> /perf
                        |                          |
                        v                          v
                   specs/{name}/            specs/journeys/JOURNEY-{id}.md
                   roadmap.md               (created by /implement)
```

Same flow, but each roadmap item produces a journey document instead of an FRD.

### Journey Contents

- Journey statement (when / I want / so I can)
- Customer Journey Map with phases (User Intent, Actions, Pain/Risk, Success Signal)
- Friction and Opportunity analysis
- North Star Summary
- UX Implementation and Assessment checklist
- Test cases
- Traceability links

## Choosing a Workflow

| | FRD | Journey |
|---|-----|---------|
| **Focus** | System requirements and engineering resilience | User experience and journey mapping |
| **Best for** | Backend services, libraries, infrastructure | User-facing features, UX-driven development |
| **Artifact** | `specs/frds/FRD-{id}.md` | `specs/journeys/JOURNEY-{id}.md` |
| **Key sections** | Stressor scenarios, rollback strategy | CJM phases, friction analysis |

## Small Changes

For trivial fixes (< 15 lines, no new API, no architecture impact), `/implement` offers a fast path in both workflows: make the change, run tests, run lint -- skip the full FRD/journey and micro-TDD ceremony.

## Switching Workflows

Edit `workflow:` in `.promptkit.yaml` and run `promptkit update`. The `/frd` skill will be replaced by `/journey` (or vice versa) and the `/implement` skill will update its artifact references.
