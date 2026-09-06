MILL_REVIEW_BEGIN
# Review: M4 matrix run: execute the descoped kick-start batch (cards 29-32) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (as reported by the runtime environment; no independent means to confirm)
reviewed_file: plan/
date: 2026-09-06
```

## Findings

### [BLOCKING:design] Taint-case rep deletion rests on a false premise
**Location:** batch 1 / card 30, "A `memory_path_scan` fatal finding is present" bullet
**Issue:** The card orders `raw/<cell>/<rep>/` deleted "so the resume genuinely re-runs it rather than skipping a repetition that is present but not counted complete" — but `RepIsComplete` in `internal/ladder/runstate.go` returns `State == "complete" && !BlindingFailed`, and `Run`'s loop guard `if RepIsComplete(dir) { continue }` in `internal/ladder/run.go` therefore already re-attempts a blinding-failed repetition; the taint path writes it via `writeCompleteState(..., blindingFailed=true, ...)` precisely so, per `RepIsComplete`'s own doc comment, "its transcript and reason survive on disk" and "the next invocation re-attempts it".
**Fix:** Drop the deletion and its stated rationale; state that the resume re-runs the tainted repetition in place, and remove the dependent deletion-reporting requirement from card 31's coverage section.

### [BLOCKING:design] LADDER_EXIT=1 branch omits the hard-error case
**Location:** batch 1 / card 30, "Completion predicate" and "Classifying a `LADDER_EXIT=1` run"
**Issue:** The card asserts exit 1 means "the run completed its own control flow" and that the two `incomplete`/`blindingFailed` causes are "both reachable causes"; `cmd/ladder/main.go` also exits 1 on `if err != nil` from `ladder.Run`, which `run.go` returns for a held/stale `AcquireRunLock`, the `reps_effective` mismatch guard, `verifyCardsAndPack` hash mismatch, the resume-time `ScanMemoryPaths` finding, and any mid-matrix `runCellRepetition` error — a truncated or never-started matrix that the card's no-`memory_path_scan` branch routes to "the matrix ran to completion ... treat the run as complete".
**Fix:** Add a third `LADDER_EXIT=1` outcome for a returned error — discriminated by `summary.json`/`table.txt` being absent, since `summarizeAndReport` in `main.go` is reached only when `Run` returns a nil error — and state its disposition (fix the fault, then the resume preconditions) rather than reading numbers.

### [NIT:consistency] Separating-result branch leaves the renumbering unstated
**Location:** batch 1 / card 32, edits 2 and 3
**Issue:** Edit 2 prescribes an exact renumbering (points 2,3,4 → 1,2,3), and edit 3's `e1 separates` branch then inserts M4b "after the Loomyard-adoption point" without saying what the resulting numbering is, so the two edits' outputs are only jointly determined by inference.
**Fix:** State the post-insertion numbering explicitly in edit 3's separating branch.

### [NIT:consistency] New results root's spelling in the roadmap is unspecified
**Location:** batch 1 / card 32, edit 1
**Issue:** `docs/roadmap.md`'s standing-rule paragraph spells its three roots as `results/2026-09-04-toc` style (relative to `bench/loomyard-eval/ladder/`), while the plan spells the new root `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart`; the card says only "naming the new results root".
**Fix:** Say which spelling the roadmap sentence uses, matching the existing paragraph's convention.

## Verdict

REQUEST_CHANGES
Two card-30 mechanism claims contradict run.go/runstate.go; the rest verifies clean.
MILL_REVIEW_END
