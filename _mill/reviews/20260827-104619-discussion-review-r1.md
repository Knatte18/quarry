# Review: Thin quarry/ facade over internal/quarryengine

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-08-27
```

## Findings

### [NIT:consistency] "15 exported identifiers" contradicts the document's own enumerated list
**Section:** Problem (¶3, "15 exported identifiers") and Decisions → Facade surface
**Issue:** Both cite 15 as the engine's exported-identifier count, but the Facade surface decision's own enumeration (8 type aliases + 6 error type aliases + 7 sentinel vars + 8 delegating funcs) lists 29 distinct identifiers. Independent grep of `quarry/*.go` (non-test) confirms exactly 29 top-level exported declarations, matching the enumerated list, not the "15" headline.
**Suggested fix:** Correct "15" to "29" in both places, or state what narrower subset "15" was meant to describe. The enumerated list itself is accurate and complete, so this does not affect what gets built — only the motivating/summary prose is wrong.

### [NIT:consistency] scoutengine: prefix count off by one
**Section:** Problem (¶3), Scope (bullet 9), Decisions → `scoutengine: ` message prefixes
**Issue:** States 60 occurrences across 9 files with `errors.go` carrying 13. Direct grep of the current `quarry/*.go` non-test files finds 59 total occurrences, with `errors.go` carrying 12 — the other 8 per-file counts match exactly.
**Suggested fix:** Correct to 59 (errors.go: 12). Non-blocking: the rename itself should be executed as a grep-driven mechanical replace, and the Testing section's own acceptance check (`grep -rn 'scoutengine' --include='*.go' .` must return only the one historical comment) is count-agnostic and self-corrects regardless of which number is quoted in prose.

## Verdict

APPROVE
Exceptionally well-grounded discussion; only two minor, non-blocking numeric inconsistencies found after spot-verifying dependency counts, LOC, file layout, and doc references against the actual source tree.
