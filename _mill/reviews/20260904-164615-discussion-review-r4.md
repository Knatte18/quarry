MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1)

```yaml
duration_s: 288.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (opus-5, high reasoning effort)
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Resume contradicts the clean-tree provenance carve-out
**Demoted-from:** BLOCKING
**Section:** `matrix-shortfall-disposition` vs `m2-lands-before-the-matrix`
**Issue:** `m2-lands-before-the-matrix` states the only tolerated `quarry_dirty` entry is the `_mill/briefs/<batch>*.md` carve-out, but `.gitignore` ignores only `results/*/raw/`, so invocation 1 leaves `results/<date>-breadth/provenance.json` (and `summary.json`/`table.txt`) untracked, and `CollectInvocation`'s `git status --porcelain` at invocation 2's startup records them in `quarry_dirty_files` — which the blessed resume path makes routine, not exceptional.
**Fix:** state the disposition for a resumed invocation's dirty tree — either widen the tolerated carve-out to the in-progress results root, or require the partial root committed between invocations — so the plan writer is not choosing between two stated rules.

### [BLOCKING:design] Resume grants one attempt, and the connect abort never re-arms
**Section:** `matrix-shortfall-disposition` remedy 1 and 2
**Issue:** `run.go` compares `InvalidateRep`'s return — the cumulative `.invalid-<n>` suffix — against `MaxAttempts`, so a repetition re-entered with three existing invalid directories gets exactly one attempt before being recorded incomplete again; and because `connectFailures` is loop-local while `attempts` is cumulative, `connectFailures == attempts` is unreachable on a re-entered root, so an unfixed server fault no longer aborts the run and instead burns one real call per remaining repetition across all six cells.
**Fix:** state both consequences in the remedy — that "fix the cause and re-invoke" buys one attempt per invocation, not a fresh three, and that the whole-run abort protecting cost discipline exists only on a fresh root.

### [BLOCKING:design] Blinding-failed reps are a fourth, unenumerated shortfall
**Section:** `matrix-shortfall-disposition`; Testing (pre-matrix gates)
**Issue:** `CheckRenderedControlPrompt` fails a control rep before dispatch when the rendered prompt carries the bare token `quarry`, the server name, or any `quarry_tools` entry (`toc`) — and this task authors exactly the two new prompts that feed `c0-none` and `d0-none`; such a rep is written void, never produces an `invalid_reason.txt`, is `blindingFailed` (so `RepIsComplete` is false and it is re-attempted deterministically forever), and does not abort the run, leaving the rung cell to spend five real calls against a control that can never complete. None of the three enumerated shortfalls covers it, and the pre-matrix offline gate list (`LoadLadder`, `LoadTaskFile`, fasit parse) omits the one pure check that would catch it.
**Fix:** add the blinding-failed case to the shortfall disposition and add a pre-matrix offline assertion that `CheckRenderedControlPrompt` returns nil for `c0-none`'s and `d0-none`'s rendered prompts.

## Verdict

REQUEST_CHANGES
Three resume- and blinding-path gaps leave the matrix's failure semantics and provenance rules underspecified.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
