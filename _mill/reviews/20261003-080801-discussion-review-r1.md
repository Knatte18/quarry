# Review: Resolve self-target path: per-call dirPackage memo (GH #34)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-10-03
```

## Findings

### [NIT:consistency] Stale fileTargetAnswer doc comment contradicts the vote rationale
**Section:** Decisions / Two-level memo: files by path, votes by directory **Issue:** the rationale ("an explicitly named gitignored file target joins the vote set") matches the code — `fileTargetAnswer` keeps the target through `!isTarget && ig.match` and passes it into `dirPackage`'s entries, so it votes — but `fileTargetAnswer`'s own doc comment (toc.go, "A gitignored file still does not vote in the package tie-break") says the opposite. **Fix:** since Scope already updates walk.go's two-pass header comment, add aligning this stale comment to the same comment-update bullet so the contradiction does not survive the refactor.

### [NIT:consistency] unitMemo.parses keeps its name but loses its meaning
**Section:** Testing / Existing suites unchanged **Issue:** once `symbolsOf` reads per-file records, the existing unit-level `parses` counter counts record lookups rather than parses, so `TestResolve_ParsesEachUnitOnce` still passes but its name and the counter's "calls made" doc drift from what is counted. **Fix:** have the plan restate the counter's doc (and, if wanted, the test's comment) to "unit extractions requested", keeping the per-file counter as the only parse count.

## Verdict

APPROVE
Acceptance criterion fully carried; mechanism, constraints and tests verified against the tree; two comment-level NITs.
