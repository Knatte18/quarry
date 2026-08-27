MILL_REVIEW_BEGIN
# Review: Thin quarry/ facade over internal/quarryengine

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnetmax
reviewer_self_id: Claude Sonnet 5 (claude-sonnet-5)
reviewed_file: _mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] Layering guard's test-file walk is self-contradictory
**Section:** Decisions → *A new layering guard pins the DAG* / *The toolchain test seam is exported...* → "Consequence for the layering guard"
**Issue:** `layering_test.go` is specified as "table-driven off the same walk the seam test uses," and the seam-test Decision defines that walk as visiting only non-test `.go` files — yet the daemontest consequence separately requires the layering guard to "not treat a `_test.go`-only importer of `daemontest` as a layering violation," a scenario that can only arise if the walk *does* visit `_test.go` files. The two statements describe incompatible walk scopes for the same test.
**Fix:** State explicitly whether `layering_test.go`'s walk includes `_test.go` files; if it does, clarify whether the daemontest carve-out is the only test-file exemption or whether every package's test-file imports are layering-checked against the production table (e.g. would `registry`'s own test file importing `daemon` now be caught).

## Verdict

REQUEST_CHANGES
One BLOCKING consistency gap: the layering guard's test-file scanning scope is unreconciled between two Decisions.
MILL_REVIEW_END
