# Scorecard — Task 01: reed terminal-geometry exploration (MCP arm)

Rerun of the 2026-08-28 task with quarry exposed as native MCP tools instead
of CLI-via-Bash for the "A" arm. B (no quarry) and C (fasit) are reused
unchanged from 2026-08-28 — only A changed. Executed by the top-level
orchestrating session directly (not a fresh dispatched agent — see the
sequential-session caveat in usage.json), against a runbook file
(`.scratch/bench-mcp-runbook.md`) prepared in advance; verified against the
session's own transcript, not the agent's self-report, which undercounted
`toc_file` and `Read` calls (see usage.json's `_note`).

## Correctness (A-mcp vs C, the sharp check)

A-mcp's key_symbols set overlaps almost entirely with C's core mechanism
list (attachCmd, AttachArgv, liveBoxLocked, parseWindowSize,
pinGeometryOptionsLocked, readWindowSizeLatestLocked, readStatusRowsLocked,
planLayout, applyLayoutLocked, reconcileApplyPersistLocked) and independently
arrives at the same central finding C calls out explicitly: no
SIGWINCH/resize handler exists anywhere in reed; live resize is delegated
entirely to tmux's own `window-size=latest` option; re-attach uses an
explicitly "told" box rather than a live query, precisely because the live
window still reflects the pre-attach size at build time. This matches C's
own named "told-box-wins-live-query-is-the-fallback" decision, independently
re-derived in A-mcp's own words.

Two gaps, both minor and both *shared with the 2026-08-28 CLI arm* (not
MCP-specific):
- `internal/reedengine/geometry.go` is listed in `relevant_files` despite C's
  own open_questions flagging it as an out-of-scope decoy (a `Geometry` type
  for paths/identity, not terminal size). The CLI arm made the identical
  inclusion on 2026-08-28.
- The session-boot path (`lifecycle.go`'s `ensureServerAndSessionLocked`,
  which pins geometry options on fresh boot) is absent from A-mcp's answer.
  The 2026-08-28 CLI arm *did* mention this symbol — a small completeness
  edge to the CLI run on this one point.

Confidence: high, matches the CLI arm's correctness ceiling on this task.

## Efficiency

| | tool_uses | duration | grep/CLI fallback |
|---|---|---|---|
| A-mcp (2026-08-29) | 9 | 113.9s | 0 |
| A-cli (2026-08-28) | 26 | 193.1s | 0 |
| B, no quarry (2026-08-28) | 22 | 133.7s | -- |
| C, fasit (2026-08-28) | 47 | 359.8s | -- |

**This is the first task in the whole benchmark where quarry-in-any-form
beats the no-tool baseline.** Every prior scorecard (see 2026-08-28's three)
found quarry-via-CLI statistically indistinguishable from or worse than
plain Read/Grep. Here, A-mcp used fewer tool calls and less wall-clock time
than *both* the CLI arm (9 vs 26 calls, 114s vs 193s) *and* the no-quarry
baseline (114s vs 134s) — at equal correctness.

Caveat on causation: part of the tool-call gap is structural, not purely
"MCP is faster" — `toc_file`'s array-shaped `targets` parameter let A-mcp
survey 8 files in one call where the CLI arm's per-invocation flag shape
likely encouraged more, smaller calls. That's a real ergonomics difference
MCP's schema enables (this is exactly the friction quarry-mcp-wrapper was
built to reduce), not an artifact to discount away — but it means "9 vs 26"
overstates how much is "the same work done more cheaply" vs. "batching
behavior quarry-mcp's schema makes more natural."

## Verdict

quarry via MCP measurably helped here, for the first time across the whole
benchmark set — fewer turns, less wall-clock, matching correctness, zero
grep fallback, beating even the no-tool baseline. One data point, one task,
run sequentially rather than cold-start (see usage.json) — worth confirming
this isn't a warm-up artifact before treating it as the benchmark's
headline finding, but it's the strongest result quarry has produced in
either exposure form so far.
