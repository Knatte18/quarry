# Review: Batch-answer contract: per-target coverage + fail-closed Status helpers

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-09
```

## Findings

### [NIT:consistency] "resolve is also called directly elsewhere" overstates its callers
**Section:** Technical context, "The producer sites, verbatim as they stand."
**Issue:** The stated rationale for placing the verifier in the unexported `resolve` — "`resolve` is also called directly elsewhere in the package, so verifying there covers every path" — has exactly one non-test caller, the exported `Resolve` (resolve.go:401); the only other direct caller is a white-box test (resolve_test.go:651).
**Suggested fix:** Keep the placement (it is right — it also covers the test's direct path and any future in-package caller), but ground the rationale as "its one production caller plus white-box tests" so a plan writer does not hunt for other production call sites.

### [NIT:consistency] "existing NameReasons facade coverage" does not exist
**Section:** Testing, "`quarry` — the facade."
**Issue:** "Assert `quarry.Statuses` is the engine's own slice, in the same shape the existing `NameReasons` facade coverage uses" references facade-level test coverage that does not exist — `NameReasons` is tested only engine-side (`internal/engine/name_test.go:278`); no `quarry`-package test touches it.
**Suggested fix:** Write the `quarry.Statuses` identity assertion fresh (the intended check is fully specified already); optionally add the missing `quarry.NameReasons` twin in the same test since the pattern claim then becomes true.

## Verdict

APPROVE
All decisions carry rationale and rejected alternatives; every load-bearing code claim verified against the tree; two wording-level inaccuracies only.
