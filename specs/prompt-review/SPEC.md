# SPEC: Prompt Quality Review

## 1. Summary

A systematic review of all promptkit template prompts (AGENTS.md, 7 instruction skills, 3 mixtures) against Anthropic's official Claude prompt engineering guidelines (April 2026) and industry research (The Prompt Report survey of 58 techniques, Augment's 11 agent techniques, context engineering best practices). The review identifies 11 concrete gaps, ranks them by impact, and proposes fixes.

## 2. Background & Research

### Market Context

Promptkit generates system prompts for AI coding agents across multiple ecosystems and agent platforms. Its prompts are long-form, structured instructions that define agent personas and multi-step workflows. The closest comparable systems are:

1. **Claude Code's built-in system prompt** (~11K tokens) — uses XML tags extensively for section separation (`<default_to_action>`, `<investigate_before_answering>`, `<use_parallel_tool_calls>`), minimal aggressive language, context-aware guidance.
2. **Cline (VS Code extension)** — ~11K character system prompt with structured tool-use format, plan/act mode separation, step-by-step confirmation process.
3. **Cursor Rules / `.cursorrules`** — community-driven prompt patterns that emphasize conciseness, ecosystem-specific idioms, and few-shot examples.

Key takeaway: Production agent prompts are trending toward XML-structured, context-motivated, example-rich designs with less aggressive language.

### Technical Context — Claude's Official Recommendations (April 2026)

Source: [Prompting best practices - Claude API Docs](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)

**Critical findings for promptkit:**

| Recommendation | Status in promptkit | Impact |
|---|---|---|
| Use XML tags to structure prompts | Not used at all | HIGH |
| Provide context/motivation for rules | Mostly missing | HIGH |
| Include 3-5 few-shot examples | Zero examples in any skill | HIGH |
| Tell what to do, not what not to do | Many negative formulations | MEDIUM |
| Dial back aggressive language for 4.6 | Heavy ALL CAPS, !!!, MUST | HIGH |
| Ask Claude to self-check | Only in implement (Reflect step) | MEDIUM |
| Prefer general instructions over prescriptive steps | Micro-TDD is extremely prescriptive | MEDIUM |
| Be consistent across prompt components | Inconsistent patterns across skills | MEDIUM |

**Key quote from Anthropic:** "Claude Opus 4.5 and Claude Opus 4.6 are more responsive to the system prompt than previous models. If your prompts were designed to reduce undertriggering on tools or skills, these models may now overtrigger. The fix is to dial back any aggressive language. Where you might have said 'CRITICAL: You MUST use this tool when...', you can use more normal prompting like 'Use this tool when...'."

### Technical Context — Industry Research

**The Prompt Report (arXiv:2406.06608)** — Systematic survey of 58 LLM prompting techniques:
- Few-shot chain-of-thought consistently delivers superior results for reasoning tasks.
- Structured prompts with clear section separation outperform flat text.
- LLM reasoning performance starts degrading around 3,000 tokens (Levy, Jacoby & Goldberg, 2024).

**Augment's 11 Agent Techniques:**
- Technique #1: Focus on context first — prioritize high-quality, relevant information.
- Technique #3: Be consistent across prompt components — all elements must align.
- Technique #5: Be thorough — but avoid redundancy (thoroughness ≠ repetition).
- Technique #9: Be aware of prompt caching — structure to minimize cache invalidation.
- Technique #10: Models pay more attention to information position — place critical rules where attention is highest (beginning and end).

**Context Engineering (from Claude):**
- "Context rot" degrades performance: poisoning (incorrect info), distraction (irrelevant info), confusion (similar but distinct info mixed), clash (contradictory info).
- Skills should be under 500 lines; split at that threshold.
- Progressive disclosure: load content on-demand, not upfront.

### Deep Dives

**Claude Code's own system prompt** demonstrates the patterns Anthropic recommends internally:
- Extensive use of XML-tagged sections (`<default_to_action>`, `<do_not_act_before_instructions>`, `<investigate_before_answering>`)
- Context/motivation for every rule ("Your response will be read aloud by a text-to-speech engine, so never use ellipses since the text-to-speech engine will not know how to pronounce them")
- Balanced tone — authoritative without being aggressive

**System Prompt Design Patterns (Tetrate):**
- The "Specialist Pattern" works best for coding agents: focused expertise with explicit domain boundaries
- Anti-pattern: "Overly complex catch-all prompts attempting to handle all scenarios"
- Testing recommendation: 50-100+ test cases, change one element at a time

## 3. Proposal

### Approach

Apply 11 evidence-based improvements to all promptkit templates, organized by impact. Changes are structural (how prompts are organized) and linguistic (how instructions are phrased), not content changes (what the prompts teach the agent to do).

### Key Findings

#### Finding 1: No XML Tags — HIGH IMPACT

**Current state:** All prompts use only markdown headings for structure.

**Evidence:** Claude's docs state XML tags "help Claude parse complex prompts unambiguously, especially when your prompt mixes instructions, context, examples, and variable inputs." Claude Code's own system prompt uses XML tags extensively.

**Recommendation:** Wrap major sections in semantic XML tags:

```markdown
<role>
You are a pragmatic, test-obsessed Golang agent...
</role>

<instructions>
## Working Loop
1. Read the technical document...
</instructions>

<rules>
- Work in single-step increments (one test, one code change, one reflection)
- Use `make lint` to verify before moving on
</rules>

<examples>
<example title="One TDD loop iteration">
## Plan
Add validation for empty input to NewConfig()
...
</example>
</examples>

<output_format>
## Plan
<reflect what written in FRD>
...
</output_format>
```

#### Finding 2: Aggressive Language Causing Overtriggering — HIGH IMPACT

**Current state:** Multiple prompts use ALL CAPS, exclamation marks, and threatening language:

| Location | Current | Problem |
|---|---|---|
| `instr-implement.md.tmpl:1` | `[ ALL GIT OPERATIONS PROHIBITED . NEVER USE GIT AT ANY COST, DONT call git ]` | ALL CAPS shouting, grammatically broken |
| `instr-bug.md.tmpl:1` | Same | Same |
| `instr-roadmaper.md.tmpl:1` | Same | Same |
| `AGENTS.md.tmpl:99` | `!!!IMPORTANT!!! Destructive git operations are prohibited` | Triple exclamation marks |
| `instr-implement.md.tmpl:44` | `No warnings or dead code should present!!` | Double exclamation, grammatically incorrect |
| `instr-implement.md.tmpl:63` | `Follow this instructions and do every step described here. Do not skip` | Grammatical error, aggressive tone |

**Evidence:** Anthropic explicitly warns that Claude 4.6 is "more responsive to the system prompt" and aggressive language causes overtriggering. The recommendation is to use "normal prompting."

**Recommendation:** Replace with calm, clear language:

| Current | Proposed |
|---|---|
| `[ ALL GIT OPERATIONS PROHIBITED . NEVER USE GIT AT ANY COST, DONT call git ]` | `<constraints>Do not run git commands. All version control is handled by the user.</constraints>` |
| `!!!IMPORTANT!!!` | Remove entirely; the rule stands on its own |
| `No warnings or dead code should present!!` | `Resolve all warnings and remove dead code before proceeding.` |
| `Follow this instructions and do every step described here. Do not skip` | `Complete every step in this workflow.` |

#### Finding 3: Missing Few-Shot Examples — HIGH IMPACT

**Current state:** Zero worked examples in any skill. Output formats are specified but never demonstrated.

**Evidence:** Claude docs: "Examples are one of the most reliable ways to steer Claude's output format, tone, and structure. Include 3–5 examples for best results." The Prompt Report: "few-shot chain-of-thought consistently delivering superior results."

**Recommendation:** Add 1-2 concrete examples to each skill. Priority skills:

1. **implement** — Show one complete TDD loop iteration with actual code diffs (not placeholders)
2. **bug** — Show a complete Phase 1 output with real bug summary
3. **researcher** — Show a Phase 1 output and a Phase 6 summary example
4. **roadmap** — Show one roadmap item with proper DoD/DoR

Wrap examples in `<example>` tags per Claude's recommendation.

#### Finding 4: Missing Context/Motivation for Rules — HIGH IMPACT

**Current state:** Rules are stated as bare commands without explaining why.

| Rule | Missing motivation |
|---|---|
| "String/numeric literals without constants are prohibited" | Why? (Magic numbers obscure intent and break when the same value needs changing in multiple places) |
| "Never introduce two behaviors in one loop" | Why? (Multiple behaviors in one loop make it impossible to isolate which change caused a failure) |
| "No snapshots or golden files unless..." | Why? (Snapshot tests pass silently when behavior drifts; precise assertions catch regressions) |
| "TODOs are prohibited. Implement, or stop." | Why? (TODOs become permanent technical debt; incomplete features ship as bugs) |
| "Keep steps under 15 modified lines" | Why? (Smaller diffs are easier to review, revert, and reason about) |

**Evidence:** Claude docs: "Providing context or motivation behind your instructions helps Claude better understand your goals and deliver more targeted responses. Claude is smart enough to generalize from the explanation."

**Recommendation:** Add a brief "because" clause to each rule. Example:

```markdown
- Work in steps under 15 modified lines, because smaller diffs are easier to review and revert if something breaks.
- Use named constants instead of string/numeric literals, because magic values obscure intent and break when the same value needs changing in multiple places.
```

#### Finding 5: Content Duplication / Context Rot — MEDIUM-HIGH IMPACT

**Current state:** The micro-TDD loop contract appears nearly verbatim in:
1. `AGENTS.md.tmpl` lines 53-99 (47 lines)
2. `instr-implement.md.tmpl` lines 101-216 (116 lines)

Quality gates, working loop steps, and commands also overlap significantly between AGENTS.md and implement skill.

**Evidence:** Context engineering research identifies "confusion" (similar but distinct information mixed together) as a form of context rot that degrades performance. The Prompt Report finds reasoning degrades around 3,000 tokens. AGENTS.md alone is ~400 lines (~3,000+ tokens).

**Recommendation:**
- AGENTS.md should define persona, values, architecture preferences, code patterns, and quality gates.
- Instruction skills should define workflows and operational steps.
- Do not duplicate the TDD loop in both. AGENTS.md can reference it: "For implementation workflow, follow the /implement skill."
- Estimated token savings: ~500-800 tokens from AGENTS.md.

#### Finding 6: Negative Formulations — MEDIUM IMPACT

**Current state:** Many rules are phrased as "don't" or "never" instead of "do."

| Current (negative) | Proposed (positive) |
|---|---|
| "Never batch changes" | "Work in single-step increments" |
| "No snapshots or golden files unless..." | "Use precise assertions first; add snapshots only after pinning an invariant" |
| "TODOs are prohibited" | "Implement features completely before moving on" |
| "No god objects" | "Keep packages focused on a single responsibility" |
| "No vendor lock" | "Prefer vendor-neutral abstractions at integration boundaries" |
| "Never introduce two behaviors in one loop" | "Add exactly one behavior per TDD loop iteration" |

**Evidence:** Claude docs: "Tell Claude what to do instead of what not to do."

#### Finding 7: Inconsistent Patterns Across Skills — MEDIUM IMPACT

**Current state:**

| Element | Variations found |
|---|---|
| Git prohibition | 3 different formulations across skills |
| "Respect AGENTS.md" | Present in implement, bug, roadmap; absent from researcher, generalize, perf |
| Identity block | Different Go template block names, different fallback text |
| Constraint positioning | Top of file (implement, bug) vs embedded in rules (researcher) |

**Evidence:** Augment technique #3: "Ensure all prompt elements align consistently to avoid confusing the model."

**Recommendation:** Create a shared constraint block that all skills inherit:

```
{{block "shared_constraints" .}}
<constraints>
- Do not run git commands. Version control is handled by the user.
- Follow the persona and contracts defined in AGENTS.md.
- Run `make lint` before considering any step complete.
</constraints>
{{end}}
```

#### Finding 8: Missing Self-Verification Steps — MEDIUM IMPACT

**Current state:** Only the implement skill has a "Reflect" step. Other skills lack explicit self-checks.

**Evidence:** Claude docs: "Ask Claude to self-check. Append something like 'Before you finish, verify your answer against [test criteria].' This catches errors reliably."

**Recommendation:** Add verification checkpoints to each skill:

| Skill | Proposed self-check |
|---|---|
| researcher | "Before writing the spec, verify: Does your research cover at least 3 comparable products? Have you identified at least 3 key decisions with alternatives? Are anti-goals explicitly stated?" |
| bug | "Before proposing a fix, verify: Does the failing test reproduce the exact reported symptom? Is the root cause proven, not guessed?" |
| roadmap | "Before finalizing, verify: Can each item be tested independently? Does every item deliver value on its own? Are there circular dependencies?" |
| generalize | "Before writing SPEC.md, verify: Are all findings backed by exact file paths and line numbers? Are there false positives in LIST.md?" |
| perf | "Before proposing optimizations, verify: Is the bottleneck class supported by profiling evidence? Have you measured before/after?" |

#### Finding 9: Information Positioning — MEDIUM IMPACT

**Current state:** Critical constraints (git prohibition) are at the very top of some skills, which is good. But critical rules in the implement skill are scattered through the middle of the document.

**Evidence:** Augment technique #10: "The model prioritizes user messages, then beginning-of-input, then middle content." Claude docs: "Queries at the end can improve response quality by up to 30%."

**Recommendation:** Structure each skill in this order:
1. **Top:** Role/identity (high attention zone)
2. **Early middle:** Context and instructions
3. **Late middle:** Examples and output format
4. **Bottom:** Rules and constraints (high attention zone — end of prompt)

This matches the "primacy-recency" effect documented in the research.

#### Finding 10: Overly Prescriptive TDD Loop — LOW-MEDIUM IMPACT

**Current state:** The micro-TDD loop specifies 8 rigid steps with exact sub-steps, format requirements, and line count limits.

**Evidence:** Claude docs: "Prefer general instructions over prescriptive steps. A prompt like 'think thoroughly' often produces better reasoning than a hand-written step-by-step plan. Claude's reasoning frequently exceeds what a human would prescribe."

**Recommendation:** Keep the TDD loop structure (it's a deliberate methodology, not just "reasoning"), but:
- Make format outputs (diff blocks, rationale sections) suggested rather than mandatory
- Allow Claude to combine Reflect + Refactor when the step is trivial
- Add: "For trivial iterations, you may condense the output format while preserving the Plan → Test → Code → Verify sequence."

This preserves the methodology while giving Claude flexibility in presentation.

#### Finding 11: No Context Window Management Guidance — LOW IMPACT

**Current state:** No skill mentions what to do when context gets long during extended implementation sessions.

**Evidence:** Claude docs recommend: "save your current progress and state to memory before the context window refreshes" and "use git for state tracking."

**Recommendation:** Add to AGENTS.md:

```markdown
<context_management>
For long implementation sessions, save progress periodically:
- Commit working code at natural checkpoints.
- Update the roadmap to reflect completed items.
- If resuming in a fresh context window, start by reading AGENTS.md, the roadmap, and recent git log.
</context_management>
```

### ML (Minimum Loveable)

The minimum set of changes that would meaningfully improve prompt quality:

**IN:**
- Finding 1: Add XML tags to all skills (structural improvement)
- Finding 2: Fix aggressive language in all skills (prevents overtriggering)
- Finding 4: Add motivation to the top 10 most important rules
- Finding 5: Deduplicate TDD loop between AGENTS.md and implement

**OUT (for now):**
- Finding 3: Few-shot examples (highest effort, requires crafting realistic examples)
- Findings 6-11: Lower impact, can be done incrementally

### Anti-Goals

- **Do not rewrite prompt content.** The workflows (TDD, bug-fix, research) are sound methodology. Only improve how they are communicated to the model.
- **Do not add new skills or features.** This is a quality pass, not a scope expansion.
- **Do not optimize for a single model.** While findings cite Claude 4.6 specifics, the improvements (structure, examples, motivation) are universally beneficial.

## 4. Technical Design

### Architecture

Changes affect only template files in `templates/`. No Go code changes required.

**Files affected:**
- `templates/_shared/instructions/*.md.tmpl` (all 7 instruction templates)
- `templates/golang/AGENTS.md.tmpl`
- `templates/rust/AGENTS.md.tmpl`
- `templates/zig/AGENTS.md.tmpl`
- `templates/golang/instructions/*.md.tmpl` (ecosystem overrides)

### Non-Functional Requirements

- **Performance:** Deduplication should reduce AGENTS.md by ~500-800 tokens, improving inference speed.
- **Reliability:** XML tags improve parsing reliability, reducing misinterpretation of mixed content.
- **Observability:** No change.

### Testing Strategy

- **Manual:** Generate prompts for all 3 ecosystems, verify XML tags render correctly, no Go template errors.
- **Integration:** Run existing `integration_test.go` to ensure scaffold rendering still works.
- **Qualitative:** Use the improved prompts in 3-5 real implementation sessions and compare agent behavior (adherence to workflow, output quality, overtriggering).

### Migration & Compatibility

- Non-breaking. Templates render differently but the workflow content is preserved.
- Users who `promptkit update` will get improved prompts on next regeneration.
- Old generated files are not affected until regenerated.

### Dependencies

None. All changes are to markdown templates.

## 5. User Journey

### Persona

Developer using promptkit to scaffold AI coding workflows. Uses Claude Code or similar agent daily. Wants the agent to follow the prescribed workflows reliably without over-triggering or ignoring instructions.

### CJM Phases

| Phase | Action | Pain Point | Success Signal |
|---|---|---|---|
| Generate | Run `promptkit init` or `promptkit update` | — | Templates render without errors |
| Use implement skill | Invoke `/implement` for a feature | Agent ignores some steps, or over-triggers on git prohibition | Agent follows TDD loop naturally, doesn't over-explain git avoidance |
| Use researcher skill | Invoke `/researcher` for a new feature | Agent produces generic spec without real research | Agent produces evidence-based spec with clear examples |
| Debug quality | Review agent output quality | Hard to diagnose why agent deviated from instructions | Clear XML sections make it easy to pinpoint which instruction section the agent is following |

### Friction Map

| Friction | Phase | Opportunity |
|---|---|---|
| Agent wastes tokens re-reading duplicated TDD loop | Use implement | Deduplicate between AGENTS.md and implement skill |
| Agent overtriggers on aggressive constraints | All skills | Normalize language to calm, clear instructions |
| Agent's output format doesn't match spec | Use implement/bug | Add worked examples so agent has a reference |
| Hard to understand why agent deviated | Debug | XML tags make sections parseable and inspectable |

## 6. Risks & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| XML tags confuse non-Claude agents | Medium | Low | XML is universally understood; Cursor, Copilot, Gemini all handle it. Test on each target agent. |
| Deduplication removes context the agent needs | High | Medium | Reference the skill explicitly from AGENTS.md rather than just removing content. |
| Softening language reduces compliance | Medium | Low | Research shows calm language with motivation is MORE effective than aggressive language for Claude 4.6. Test in sessions. |
| Examples bias agent toward specific patterns | Low | Low | Use diverse examples covering different scenarios. Mark as `<example>` so agent distinguishes from instructions. |

## 7. Open Questions

1. Should XML tags be added to ecosystem-specific AGENTS.md templates or only to instruction skills? (Recommendation: both, for consistency.)
2. What is the right number of few-shot examples per skill? (Start with 1-2, measure before adding more.)
3. Should the shared constraint block be a Go template block or physically duplicated? (Recommendation: Go template `{{block}}` for single-source-of-truth.)
4. How do we measure "overtriggering" objectively? (Suggestion: count how many times the agent mentions git prohibition unprompted in a session.)

## 8. Implementation Roadmap

### Phase 1: Structural (Findings 1, 2, 5, 7)
1. Add XML tags to all shared instruction templates
2. Fix aggressive language across all templates
3. Deduplicate TDD loop between AGENTS.md and implement skill
4. Create shared constraint block
5. Regenerate and verify with integration tests

### Phase 2: Motivational (Findings 4, 6, 8, 9)
1. Add context/motivation to top rules in each skill
2. Convert negative formulations to positive ones
3. Add self-verification checkpoints to each skill
4. Reorganize information positioning (role → instructions → examples → rules)

### Phase 3: Exemplary (Finding 3)
1. Craft 1-2 worked examples for implement skill
2. Craft examples for bug, researcher, roadmap skills
3. Test with real implementation sessions
4. Iterate based on observed agent behavior

### Phase 4: Refinement (Findings 10, 11)
1. Soften TDD loop prescriptiveness
2. Add context window management guidance
3. Qualitative review after 2-4 weeks of usage

---

## Sources

- [Prompting best practices - Claude API Docs](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)
- [Use XML tags to structure your prompts - Claude API Docs](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/use-xml-tags)
- [Giving Claude a role with a system prompt - Claude API Docs](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/system-prompts)
- [The Prompt Report: A Systematic Survey (arXiv:2406.06608)](https://arxiv.org/abs/2406.06608)
- [Context Engineering from Claude (01.me)](https://01.me/en/2025/12/context-engineering-from-claude/)
- [11 Prompting Techniques for Better AI Agents (Augment Code)](https://www.augmentcode.com/blog/how-to-build-your-agent-11-prompting-techniques-for-better-ai-agents)
- [System Prompts: Design Patterns and Best Practices (Tetrate)](https://tetrate.io/learn/ai/system-prompts-guide)
