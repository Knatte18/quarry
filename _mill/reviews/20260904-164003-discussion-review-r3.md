MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1)

```yaml
duration_s: 215.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, high reasoning effort)
reviewed_file: /home/knatte/Code/quarry/wts/ladder-breadth/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Technical context still asserts attempt==suffix
**Demoted-from:** BLOCKING
**Section:** § Technical context, the `run.go` attempt-loop bullet ("see the `m2-attempt-numbering` decision for how the reason file's `attempt` field is made to agree with the suffix `InvalidateRep` produces").
**Issue:** That sentence is the round-1 wording the `m2-attempt-numbering` decision explicitly replaces — the decision says the two are **not** asserted equal in general and rejects any write-after-rename scheme; a plan writer reading only the technical context would implement the rejected alternative.
**Fix:** Restate the bullet as "how `attempt` is defined as the loop-local index, and why it diverges from the suffix on a re-entered root".

### [BLOCKING:design] No disposition for a matrix that misses its own done-when
**Section:** § Testing ("The matrix itself…"), § Scope ("one invocation, six cells × 5 reps"), `one-ladder-file-one-results-root`.
**Issue:** The done-when demands `5/5` complete per cell and `unscored_count: 0`, but two real paths in `run.go` fall short with no stated remedy: (a) a cell exhausting `MaxAttempts` is recorded incomplete, and `connectFailures == attempts` aborts the *whole* invocation mid-matrix — recovery requires a second invocation, which "one invocation" appears to forbid; (b) a scorer failure writes the repetition via `writeCompleteState(..., blindingFailed=false, scored=false, "scorer_failed")`, so `RepIsComplete` returns true and a resumed invocation skips it forever — `unscored_count: 0` is then unreachable in that root without deleting repetition directories by hand.
**Fix:** Decide and state whether resuming into the same results root is permitted (and still counts as "one root"), and what happens to a permanently unscored repetition — accept and report it, or delete-and-rerun that rep — before the plan is written.

### [NIT:consistency] Header-amendment list omits two more stale lines
**Section:** `one-ladder-file-one-results-root` ("the 'two task groups per file' line and the 'deliberately NOT here' entry … must be rewritten").
**Issue:** `ladder-toc.yaml`'s header carries at least two further statements this task falsifies and the decision does not name: line 10 "Design: four surviving cells" (it becomes eight) and the "Task 02 (three packages) has no fasit either" clause (this task authors it).
**Fix:** Instruct a whole-header pass rather than enumerating individual lines, so no false line survives.

### [NIT:decision] Orphaned roadmap sentence has no stated disposition
**Section:** `roadmap-row-disposition`.
**Issue:** The decision covers both table rows and the section heading, but not `docs/roadmap.md:19` ("M2 is small and independent; it can run as `mill-quick` or fold into M1's pre-matrix work"), which is left orphaned once the table and heading go.
**Fix:** Name that sentence as removed with the section.

## Verdict

REQUEST_CHANGES
One superseded technical-context claim and one unaddressed partial-matrix failure mode.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
MILL_REVIEW_END
