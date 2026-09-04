MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1)

```yaml
duration_s: 266.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), reasoning effort high
reviewed_file: /home/knatte/Code/quarry/wts/ladder-breadth/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:design] `attempt` invariant is false on a re-entered root
**Section:** `### m2-attempt-numbering`, and Testing bullet "Every e2e case above"
**Issue:** The decision asserts the loop-local counter and `InvalidateRep`'s suffix are "kept in agreement by assertion", but no specified test re-enters a results root that already holds `.invalid-*` directories — the exact case the rationale names as the one where they diverge; on a resumed invocation the counter restarts at 1 while `InvalidateRep` (runstate.go:141, scans for the next unused suffix) returns 4, so `attempt: k == suffix k` is simply untrue there and the safeguard is claimed rather than specified.
**Fix:** Decide what `attempt` means on a re-entered repetition (loop-local index vs. directory suffix), state it, and scope the equality assertion to the fresh-root case only or add the re-entry case that proves the chosen semantics.

### [NIT:scope] Fourth cause `result_error` is unprovable with the fixture
**Demoted-from:** BLOCKING
**Section:** `### m2-cause-taxonomy-and-exit-code` / `### m2-proven-by-test` / Testing
**Issue:** The enumeration fixes four causes, but only three get tests; `testdata/fakeclaude` cannot produce `result_error` at all — every stream variant calls `writeResult(..., isError=false)`, and the only stream with no result record (`partial_fail`) exits 1, so `invokeErr != nil` classifies it `runner_error` first (run.go:648-654 prefers `runErr` over `parseErr`).
**Fix:** State the disposition for `result_error` — either a new `FAKE_CLAUDE_STREAM` variant (an `is_error: true` result, or a truncated stream at exit 0) plus its e2e case, or an explicit decision to ship that branch untested and why.

### [NIT:decision] Roadmap propagation covers only the parked-T8 section
**Section:** Scope "Doc propagation" / `### conclusion-is-the-t8-input-and-propagates`
**Issue:** `docs/roadmap.md:16-19` will still carry the M1 and M2 rows in "Next wave: measure" — including the OSL-1033 rerun clause this task declares Out — with no stated disposition after the conclusion lands.
**Fix:** Say whether the M1/M2 rows are struck, marked done, or left, and how the OSL-1033 clause is recorded as deferred.

### [NIT:decision] Scope of "every reference to `ServerConnectFailureFile` is removed"
**Section:** Testing, "*Compile-time:*"
**Issue:** The constant is also named in prose in the committed `results/2026-09-04-toc/conclusion.md` (lines 145, 221), a frozen record the no-re-derivation convention argues against editing; the requirement reads as if it covers those too.
**Fix:** Scope the removal to Go source and tests, and state that prior conclusions stay untouched.

### [NIT:design] Ladder d subject choice precedes the evidence for its own constraints
**Section:** `### ladder-d-cold-start-orientation` / `### fasit-authored-by-a-reference-agent-card`
**Issue:** The implementer must pick a subject satisfying "(a) the answer spans at least two packages" and "(c) answerable at the pin", but only the later reference-agent card does the exhaustive read that establishes either.
**Fix:** State that (a)/(c) are provisional at pick time and confirmed by the fasit card, with `degenerate-fasit-is-a-pre-matrix-swap` as the sole remedy.

## Verdict

REQUEST_CHANGES
Two M2 gaps: attempt-number invariant untrue on resume, and `result_error` unprovable.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
MILL_REVIEW_END
