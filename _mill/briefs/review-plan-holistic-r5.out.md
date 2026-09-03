MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Anthropic), Opus-class model; exact build reported to me as claude-opus-5
reviewed_file: plan/
date: 2026-09-03
```

## Findings

### [BLOCKING:design] worktree_dirtied observation lands after run.json
**Location:** batch 6 / card 26 (with card 25) **Issue:** the write order ends "`run.json` last; record the dirtied observation from the worktree's porcelain status and restore the worktree", but `RunState.observations` is the only carrier the plan gives that observation (card 29's table prints only gate-1, hash-drift and fingerprint findings; card 28's `CellRecord` carries only gate 1), so as sequenced it is either discarded or forces a second write of the state file that the six-filenames decision says is written last. **Fix:** state that `WorktreeStatus`/`CheckWorktreeDirtied` runs before `run.json` is written and that its finding is appended to `observations`, with the restore following the state write.

### [BLOCKING:scope] card 36 cannot locate the transcript it asserts on
**Location:** batch 8 / card 36 **Issue:** the live test "asserts three things about the resulting transcript", which requires `RepDir` and `TranscriptFile` from `runstate.go`, but `runstate.go` is not in the card's `Context:` (only run.go, stream.go, worktree.go, config.go, ladder-toc.yaml). **Fix:** add `bench/loomyard-eval/ladder/internal/ladder/runstate.go` to card 36's `Context:`.

### [NIT:consistency] migrated ladder file keeps a pointer to a deleted file
**Location:** batch 1 / cards 5 and 6 **Issue:** card 5 rewrites only "the file's long header comment" and says to keep `source_repo` unchanged, but its own inline comment (ladder-toc.yaml lines 86–87) reads "at prepare-session time -- see ladder.yaml's own source_repo comment", and card 6 deletes `ladder.yaml`; the phrase "prepare-session" is also retired V1 vocabulary. **Fix:** have card 5 name that inline comment explicitly among the lines to rewrite.

### [NIT:consistency] card 1 claims go.sum will be updated; it will not
**Location:** batch 1 / card 1 **Issue:** `go.sum` already carries both `gopkg.in/yaml.v3 v3.0.1` lines (h1 and /go.mod), so `go get gopkg.in/yaml.v3` changes `go.mod` only and the declared `Edits: go.sum` produces no diff — the requirement's "`go.mod` and `go.sum` are both updated by the toolchain" is false against this tree. **Fix:** reword to say `go.sum` may already be satisfied and an unchanged `go.sum` is the expected outcome.

### [NIT:scope] report subcommand's summary.json write is never stated
**Location:** batch 7 / card 30 **Issue:** the run subcommand is told to "summarise, write the summary, print and write the table", but the report subcommand only "re-derives the summary and the table", leaving unstated whether it writes `summary.json` at the results root — the discussion's command-surface requires it, and card 35 compares "the produced summary" to a golden. **Fix:** state that the report subcommand calls `WriteSummary` and `WriteTable` at the results root as `run` does.

### [NIT:consistency] fixture root called "complete" but omits the redacted answer
**Location:** batch 8 / card 35 **Issue:** the card says "commit a complete raw tree" and justifies only the deliberate omission of the metrics file, yet its `Creates:` list also has no `answer.redacted.json` for repetition 1, which the six-filenames decision requires of a scored rep. **Fix:** state the redacted-answer omission and its reason alongside the metrics-file one, or add the file.

## Verdict

REQUEST_CHANGES
Two blocking gaps: the dirtied-observation write ordering, and card 36's missing runstate context.
MILL_REVIEW_END
