MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping)

```yaml
duration_s: 209.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus-class model (self-reported claude-opus-5, training cutoff 2026-05)
reviewed_file: /home/knatte/Code/quarry/wts/gopls-query-precision/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Empty declaration set silently fails the gate open
**Section:** `fail-closed-verification` / `declaration-match-is-positional`
**Issue:** Fail-closed is specified only per-reference; nothing covers the declaration-side `textDocument/definition` returning empty or erroring. `lookup` (`internal/quarryengine/query/refs.go:214`) returns `toSortedReferences(locations)` with **nil error** on an empty location list, so an empty declaration set is a silent, non-error outcome — and with an empty declaration set no reference can "intersect" it, so verification drops *every* reference and `assert-no-callers` exits 0 with an empty `callers` list. Today's code (`internal/cli/cli.go:637`) does the opposite: empty `declRefs` makes every reference a violation.
**Fix:** State the disposition for an empty or errored declaration set — e.g. verification is skipped entirely (all references kept) rather than applied against an empty set — and add the corresponding hermetic scenario to the verification-filter test list.

### [BLOCKING:design] Verification timeout vs `lookup`'s `timedOut` teardown flag
**Section:** `sequential-verification` / "The `lookup` pipeline"
**Issue:** The decision says a verification phase that hits its deadline "yields kept (unverified) references ... and is never an error", while Technical Context requires preserving `lookup`'s invariant that a later phase can still set `timedOut`. The two are stated independently and never reconciled: if a verification deadline is swallowed without setting `timedOut`, `teardownConnection` (`refs.go:151`) takes the `Close()` branch on a stalled native/legacy server — the exact re-block hazard `acquireConnection` documents at `refs.go:117-121`.
**Fix:** State explicitly whether a verification-phase deadline sets `timedOut` (teardown hard-kills) while still returning a successful result.

### [NIT:consistency] Q&A log carries the superseded concurrency answer
**Demoted-from:** BLOCKING
**Section:** `## Q&A log`, round-trip-budget entry
**Issue:** That entry answers "Bounded concurrency sharing one phase deadline", which `sequential-verification` and the later r1 Q&A entry both reject as unsafe against the single-flight `lsp.Client` (verified: `lspclient.go:341` unsynchronized `nextID`, `:263` no write lock, `:357` one shared `incoming` channel). A plan writer reading the Q&A log as the decision record gets the rejected answer.
**Fix:** Replace that Q&A answer with the sequential one, or mark it explicitly superseded by the r1 entry.

### [NIT:consistency] "40% of true call sites" is not what the cited source says
**Section:** `## Problem`, first defect
**Issue:** `docs/scout-multilang.md:115` reports 42 of 68 tagged sites for `hubgeometry.Resolve` (62%) and 17 of 38 for `Layout.WeftWorktree` (45%) — 59 of 106 combined (56%), not 40%. The only "40%" in `docs/` is `scout-vs-grep.md:77`'s unrelated "~40% fewer tokens".
**Fix:** Restate the figure as measured (59 of 106 across the two symbols) or cite it per-symbol.

### [NIT:decision] New verification entry point is never named
**Section:** `one-connection-verification-entry-point`
**Issue:** Every other new identifier is named exactly (`ErrBuildTagsUnsupported`, `InitializationOptions`, `BuildTags`), but the entry point that must be added to `query`, re-exported from `quarry/facade.go`, and reflected in that file's "exactly the 29 identifiers ... no more, no less" doc comment (`quarry/facade.go:8-9`) has no name.
**Fix:** Name it, and state the resulting facade identifier count so the doc-comment update is mechanical.

## Verdict

REQUEST_CHANGES
Verification can fail the safety gate open on an empty declaration set; two further gaps.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
