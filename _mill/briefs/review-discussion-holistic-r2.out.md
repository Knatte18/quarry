MILL_REVIEW_BEGIN
# Review: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] Ladder file as specified still has three controls
**Section:** D2 + D6 + D11
**Issue:** D2's default is "`*c.Control` when set, `len(c.Allowed) == 0` otherwise", but D6 and D11 name `control: true` only on `e0-names` and leave `e1-pack`/`e2-files` with `control:` unset and `allowed: []` — so both default back to control, `validate` (config.go:237-241) rejects the file with "expected exactly one control, found 3", and `summarize` would flag all three `Control` and build zero comparisons.
**Fix:** State explicitly that `e1-pack` and `e2-files` carry `control: false` in the ladder file, in D11's config list.

### [BLOCKING:design] `ladder pack`'s provenance write collides with the merge identity checks
**Section:** D3 step 5 / D4
**Issue:** D4 only says the block is carried forward; it never says whether `ladder pack` appends an `Invocation`. If it does not, `run` reads a `Provenance` with empty `LadderFile`/`ServerName`/`LoomyardRepoSHA256` and `RepsEffective: 0`, and dies at run.go:100 ("was written with reps_effective 0, this invocation requests 10") or in `MergeProvenance` on `ladder_file differs: "" vs ...` — before rep 1, every time.
**Fix:** Decide and state whether `ladder pack` runs `CollectInvocation`+`MergeProvenance` (and with which `reps_effective`/`selected_cells`), or writes only `kickstart_pack` with `MergeProvenance` explicitly exempting a pack-only record from the four identity checks.

### [BLOCKING:design] The `pack_targets` validation rule is not implementable in `validate`
**Section:** D11 (`pack_targets`)
**Issue:** "`validate` requires it to be non-empty when any config declares a `card` containing the pack sentinel" requires reading card files, but `(*Ladder).validate()` takes no repository root and touches no filesystem — a repository-relative `card:` path cannot even be resolved there, and making it filesystem-dependent changes every `config_test.go` fixture's contract.
**Fix:** Restate the rule at a layer that has the root (e.g. non-empty whenever any config declares a `card` at all, with the sentinel check moved into the pack/run path).

### [BLOCKING:design] Gate accounting omits max-turns and unscored reps
**Section:** D8 (gate-failure accounting)
**Issue:** The accounting splits reps into gate-pass and gate-fail, but the harness produces two further states that never yield a `summary_matches` value at all — a max-turns rep (`writeCompleteState(..., scored=false)`, excluded from recall/precision by `summarizeCell`) and an otherwise-unscored rep — so the per-rep gate column is undefined for them and the ">2 of 10 fail" trigger has no stated reading.
**Fix:** State how a max-turns or unscored rep is recorded in the gate column and whether it counts toward the 2-of-10 threshold.

### [NIT:consistency] `main.go`'s header comment forbids a third subcommand
**Section:** D3 / Technical context
**Issue:** `cmd/ladder/main.go`'s header states "there is nothing left for a third subcommand to do"; D3 adds one and the discussion does not note the comment as superseded.
**Fix:** Note that the file header and the two usage strings are updated alongside the new subcommand.

### [NIT:design] `ladder pack` and the run lock left unstated
**Section:** D3 step 1
**Issue:** `run` wraps its worktree work in `AcquireRunLock` (worktree.go:293); "prepares/restores the pinned worktree ... exactly as `run` does" does not say whether `ladder pack` takes the same lock, so a pack and a run can touch one worktree concurrently.
**Fix:** State whether `ladder pack` acquires the run lock.

## Verdict

REQUEST_CHANGES
Four blocking gaps: control defaulting, pack provenance merge, validate scope, gate accounting.
MILL_REVIEW_END
