MILL_REVIEW_BEGIN
# Review: Ladder, toc rerun (T7) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus-class model, high reasoning effort)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] No recovery when the terminal invocation returns an error
**Location:** batch 2 card 4; batch 3 card 6
**Issue:** `runCommand` skips `summarizeAndReport` entirely whenever `ladder.Run` returns an error (main.go:88–96) — reachable via the resumed-root `ScanMemoryPaths` return (run.go:152), a `BuildServer`/`AcquireRunLock`/reps-mismatch failure, or any `runCellRepetition` error — so `summary.json` and `table.txt` are then absent or describe an earlier invocation, yet card 4's `Creates:` must commit them and every number card 6 prints is quoted from them.
**Fix:** Give card 4 an explicit disposition for `ladder report --results <root>` (it needs only the results root plus the already-written `provenance.json`) as the step that re-derives both artifacts before the commit whenever the last invocation exited on an error.

### [NIT:decision] `.ladder.lock` has no disposition between invocations
**Location:** batch 2, cards 3 and 4
**Issue:** Card 3 checks for a stale `.ladder.lock` once, pre-matrix; card 4's before-each-re-invocation checklist covers `git status` and `.invalid-*` counts but not the lock, and `AcquireRunLock` (worktree.go:293) refuses on an existing file — a killed background invocation makes the next one fail instantly, and the three-invocation ceiling would count that measurement-free failure.
**Fix:** Add the lock check to card 4's re-invocation checklist and state that a lock-refused invocation measured nothing and does not consume an arm of the ceiling.

### [NIT:consistency] Card 8 leaves HANDOFF §1's merged-wave inventory stale
**Location:** batch 3 card 8, steps (0) and (2)
**Issue:** The card makes §2/§4 assert the critical path is finished but forbids touching §1, which still reads "Waves 0–3 are merged on `main`" with the list stopping at T5a (no T6, no T7) and "Uncommitted on `main` right now: this file only" — the same document-disagrees-with-itself condition step (0) is justified by.
**Fix:** Either extend the card's edit list to §1's merged-wave sentence, or state in the card why §1 is deliberately left alone.

### [NIT:decision] The clean-tree obligation names no owning card
**Location:** overview `### Decision: clean-tree-and-no-edits-mid-matrix`; batch 1 card 2; batch 2 card 3 step (2)
**Issue:** The decision requires every `_mill/` artifact committed before the first invocation, but both cards that enforce it are zero-diff with `Commit: none`, so an uncommitted `_mill/` report can only abort the batch and no card can resolve it.
**Fix:** Name where that commit comes from (the pipeline's own per-card commit, or an explicit card step) so the check has a remedy rather than only a stop.

## Verdict

REQUEST_CHANGES
One reachable path leaves the artifacts every write-up card quotes unwritten and unrecoverable.
MILL_REVIEW_END
