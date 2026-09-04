MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1)

```yaml
duration_s: 326.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus-class model; harness reports claude-opus-5, which I cannot independently verify
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:design] `detail` field content and its quotation are undecided
**Section:** `m2-one-reason-file-for-every-cause` + `matrix-shortfall-disposition` 1 + Constraints
**Issue:** The file is specified as "one `key: value` per line" with a "free-form `detail`", but the only natural source for `detail` on the `runner_error` path is the wrapped error, and `ExecRunner.Run` (`worktree.go:81`) formats it as `run %s %s: %w` over the full arg vector — which `run.go:617-638` shows contains `-p <entire rendered prompt>`, the absolute `--mcp-config` path and the claude binary path; that is multi-line, kilobytes long, breaks the one-pair-per-line contract, and the discussion simultaneously requires shortfall-1 causes to be "quoted" in the committed `conclusion.md` against the constraint that no tracked file carries a machine path.
**Fix:** State what `detail` holds (e.g. a single-line, truncated, path-stripped message rather than `err.Error()`) and state that the conclusion quotes only the `cause` enumeration and counts, never `detail` verbatim.

### [NIT:consistency] "Equality unreachable on a re-entered root" is over-general
**Section:** `matrix-shortfall-disposition` 2
**Issue:** `connectFailures` is per-repetition (`run.go:394`) and `attempts` is `InvalidateRep`'s cumulative suffix, so the abort is unreachable only for a repetition that *already carries* `.invalid-*` directories; a repetition never attempted in a previous invocation (e.g. reps 4–5 of a resumed root) still reaches `3 == 3` and aborts the run normally.
**Fix:** Scope the claim to previously-invalidated repetitions; the out-of-band server check and the two-resume cap stand either way.

### [NIT:consistency] `m2-proven-by-test` names three cases; Testing requires five
**Section:** `m2-proven-by-test` vs Testing
**Issue:** The decision block lists only `partial_fail`, `no_fence` and the retargeted `GrantedCellServerNeverConnects`, omitting the `result_error` case that `m2-cause-taxonomy-and-exit-code` makes mandatory (it is the sole reason the fifth fixture variant is added) and the re-entry case `m2-attempt-numbering` requires; a plan writer treating the decision block as the test inventory ships two of the required cases missing.
**Fix:** Amend `m2-proven-by-test` to name all five cases, or state explicitly that Testing is the authoritative inventory.

### [NIT:consistency] Task-05 rejection cites `rewrite-plan.md` §2 point 2 inaccurately
**Section:** `ladder-c-reuses-task-02` (Rejected)
**Issue:** §2 point 2 says *LSP-shaped* tools (definition, references, symbol) sat flat; it says nothing about "impact-shaped tools", and the `impact` tool is unbuilt (parked in T8), so it cannot have measured flat — and as written the rationale would argue equally against ladder b (task 04), which is impact-schema and is in scope.
**Fix:** Reject task 05 on the ground that actually holds (one impact-schema negative control is enough, and 05 has no drafted multi-package shape advantage over 02), or cite a source that supports the flatness claim.

## Verdict

REQUEST_CHANGES
One blocking gap: the reason file's `detail` content and its quotation into the tracked conclusion.
MILL_REVIEW_END
