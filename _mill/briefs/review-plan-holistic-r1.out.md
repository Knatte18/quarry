MILL_REVIEW_BEGIN
# Review: Ladder, toc rerun (T7) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (self-reported ID "claude-opus-5"), high reasoning effort
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] Server-hash reading is taken before the binary exists
**Location:** batch 2 / card 3 step (5), card 4 ("Before every invocation, including the first")
**Issue:** `run.go:167-172` builds `<worktree-root>/bin/<server-name>` *inside* the invocation via `BuildServer`, and `live_test.go:98` passes `""` as `serverBinary` so `TestLive` never builds it — so before invocation 1 the file is absent or a stale leftover, and the last invocation's binary is never hashed out-of-band at all.
**Fix:** Take the reading immediately *after* each invocation (or after the build line in its log), not before it, and state in card 3 that a pre-existing `bin/quarry` is stale rather than a baseline.

### [BLOCKING:design] Resumed invocations record `quarry_dirty: true` by construction
**Location:** Shared Decisions/clean-tree-and-no-edits-mid-matrix; batch 2 / card 4; batch 3 / card 6
**Issue:** `CollectInvocation` derives the flag from plain `git status --porcelain` (provenance.go:223-227, 255), which lists untracked files; card 4 commits `summary.json`/`provenance.json`/`table.txt` only after the run terminates, so invocations 2 and 3 see them untracked under the tracked results root and record dirty — the exact condition the decision says aborts the run.
**Fix:** State the disposition explicitly (accept the carve-out and require card 6 to report `quarry_dirty`/`quarry_dirty_files` per invocation entry, or commit the machine artifacts between invocations).

### [BLOCKING:design] `probe.md` must assert a property its only source never states
**Location:** batch 1 / card 1, item (c)
**Issue:** Card 1 requires probe.md to record "the server connected and was listed in the session's `system` record", but the sole cited source — the round-1 review's `[NIT:consistency]` **Issue:** paragraph, which the same card requires transcribed verbatim — says only "connect + `mcp__quarry__toc` returned the §4 envelope"; the `system` record is never mentioned. The batch's own rule is that the card "does not generate fresh evidence and must not claim to".
**Fix:** Reduce item (c) to what the quoted paragraph actually states, or require the report to mark the `system`-record property as not covered by the operator's report.

### [BLOCKING:decision] `ABANDONED.md` has no producing card
**Location:** Shared Decisions/results-root-name and harness-fixes-restart-the-matrix; batch 3 / card 6
**Issue:** Both decisions require an abandoned root to "gain an `ABANDONED.md` naming the fix, the date and the successor root", and card 6 only *checks* that it exists — no card creates it, and no batch lists it in `Creates:`.
**Fix:** Name the card that writes `ABANDONED.md` on the restart path (batch 2 card 4 is the only card that knows the fix and the successor root).

### [BLOCKING:decision] The done gate exists only in prose
**Location:** `00-overview.md` frontmatter (`verify: null`); Shared Decisions/the-done-gate-runs-offline; batches 1 and 3 (`verify: null`)
**Issue:** The decision fixes the done gate as `go test ./... && golangci-lint run` with `LADDER_LIVE_TEST` unset, and three Batch Tests sections assert it "runs after this batch", but nothing machine-readable in the plan carries it — the only encoded verify is batch 2's package-scoped `go test ./bench/loomyard-eval/ladder/...`.
**Fix:** Put the decided command in the overview's `verify:` field so the gate the decision names is the gate that runs.

### [NIT:consistency] Metric-key decision misses two table renames
**Location:** Shared Decisions/metric-key-spellings; batch 3 / card 6
**Issue:** `report.go:24-31` also renames `quarry_tool_uses` → `prefixed_tool_uses` and `grep_fallback_total` → `grep_fallback` in `tableColumnNames`, but the decision presents only `cache_read`/`cache_creation` as table-only spellings and lists the other two as if shared — card 6's gate-1 tool-use reporting is exactly where that bites.
**Fix:** Add both renames to the spelling table in the decision.

### [NIT:scope] `ResolveLoomyardRepo` prefers the process env var, not the file
**Location:** batch 2 / card 3 step (3)
**Issue:** The card asserts "the ladder env file under .scratch/ resolves the Loomyard checkout"; `worktree.go:112-118` reads `LADDER_LOOMYARD_REPO` from the environment first and only falls back to `.scratch/ladder.env`, so an inherited stale value silently wins and is never checked.
**Fix:** Have step (3) confirm `LADDER_LOOMYARD_REPO` is unset in the invoking environment, or that it matches the file.

### [NIT:scope] Card 5 has no path when no `a2-toc-dir` repetition is complete
**Location:** batch 3 / card 5; card 6 ("the result of card 5's hand verification")
**Issue:** Card 5 falls back from repetition 1 to "the lowest numbered one that is [complete]", but says nothing about the case where the cell finished at n=0 — an outcome the plan's own termination rule explicitly permits — while card 6 unconditionally requires card 5's result.
**Fix:** State the n=0 disposition: skip the hand verification and have the conclusion record that it could not be performed.

### [NIT:consistency] Card 7 names §11 indirectly
**Location:** batch 3 / card 7
**Issue:** The card says "In §10's successor section, 'Open decisions'", while the discussion, Shared Decisions/raw-tree-stays-untracked and card 6 all call it plan §11 — `docs/rewrite-plan.md:520` is literally `## 11. Open decisions`.
**Fix:** Spell it §11, matching every other reference in the plan.

### [NIT:consistency] HANDOFF §2 left contradicting card 8's §4 rewrite
**Location:** batch 3 / card 8 ("Change no other section")
**Issue:** `HANDOFF.md:38-39` states "Only T6 and T7 remain on it [the critical path]"; card 8's §4 edit asserts "the critical path is finished", leaving the document self-contradictory, and card 8 forbids touching §2.
**Fix:** Add the one-line §2 correction to card 8's enumerated edits, or say explicitly why §2 is left alone.

## Verdict

REQUEST_CHANGES
Five blocking gaps: mis-sequenced hash reading, dirty-flag on resume, over-claimed probe, unowned ABANDONED.md, unencoded done gate.
MILL_REVIEW_END
