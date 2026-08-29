# Scorecard — Task 04: pre-implementation impact analysis (MCP arm)

Rerun of the 2026-08-28 task with quarry exposed as native MCP tools instead
of CLI-via-Bash. B and C reused unchanged. Same execution/verification
method as task 01's scorecard — see its header note.

## Correctness (A-mcp vs A-cli vs B vs C, the sharp check)

The task this whole set was built to give quarry its best shot: a real,
naturally-occurring interface-method-name collision
(`shedadapters.Shuttle.Run` vs `shedadapters.BurlerRunner.Run`, sibling
fields on the same struct) that a text search cannot disambiguate without
type resolution. A-mcp found all 3 real callers with correct line numbers
(`singlellm.go:143`, `bouncer.go:466`, `bouncer.go:580`) and correctly
excluded the one decoy (`burler.go:373`, resolves to the unrelated
`BurlerRunner.Run`) with a correct type-based explanation — an exact match
against this task's recorded ground truth. Identical result to the
2026-08-28 CLI arm and to B, both of which also scored 100% recall/precision
on this same check. Three independent methods, same perfect answer.

## Efficiency

| | tool_uses | duration | grep/CLI fallback |
|---|---|---|---|
| A-mcp (2026-08-29) | 8 | 93.0s | 0 |
| A-cli (2026-08-28) | 14 | 80.0s | 1 |
| B, no quarry (2026-08-28) | 12 | 77.6s | -- |
| C, fasit (2026-08-28) | 32 | 273.8s | -- |

Fewer tool calls than either the CLI arm or B (8 vs 14 vs 12), and zero grep
fallback vs. the CLI arm's one (self-judged "justified" in the 2026-08-28
scorecard, since quarry's `impact` verb has no verb for enumerating
textually-similar-but-different call sites — A-mcp apparently found another
way to close that gap without grep, since `excluded_lookalikes` still names
the decoy with full reasoning). But, same pattern as task 03: A-mcp is the
*slowest* wall-clock of the three real-tool arms (93.0s vs A-cli's 80.0s and
B's 77.6s) despite the fewest tool calls.

## Warm-daemon rerun (2026-08-29, same day)

Same rationale as task 03's warm rerun (see that scorecard) — this
benchmark's cold-worktree-per-task design does not reflect production, where
Lyx keeps quarry running continuously and an agent virtually always meets an
already-warm daemon. Rerun with one throwaway `toc_dir` warm-up call issued
before starting the clock:

| | tool_uses | duration | grep/CLI fallback |
|---|---|---|---|
| A-mcp, warm daemon | 2 | **23.1s** | 0 |
| A-mcp, cold daemon | 8 | 93.0s | 0 |
| A-cli, cold (2026-08-28) | 14 | 80.0s | 1 |
| B, no quarry (2026-08-28) | 12 | 77.6s | -- |

Warm MCP is ~3.4x faster than the next-best arm (B, 77.6s) and beats every
cold arm outright. Same perfect result: all 3 real callers
(`singlellm.go:143`, `bouncer.go:466`, `bouncer.go:580`) found via one
2-target `textDocument_definition` call plus one 2-target
`textDocument_references` call (scoped `within: internal/shedadapters`),
decoy (`burler.go:373`) correctly isolated to `BurlerRunner.Run`'s own
reference set — 2 tool calls total, down from 8 cold.

Same caveat as task 03's warm rerun applies: executed directly by the
already-familiar orchestrating session, not a fresh blind agent — treat the
tool-call drop (8 -> 2) as the more daemon-attributable signal than the
raw wall-clock number alone.

## Correction: the warm rerun above was contaminated, not representative

Same issue as task 03's warm rerun (see that scorecard's correction) — the
23.1s figure was produced by the orchestrating session after it had
already read this task's own ground truth (the 3 real callers and the
`burler.go:373` decoy, with exact line numbers) earlier in the conversation.
Its 2 tool calls were verification, not discovery.

A genuinely blind agent run against the same pre-warmed worktree
(`bench/loomyard-eval/results/2026-08-29-warm/`) took **92.9s — essentially
identical to the cold run's 93.0s**, using 5 tool calls plus two separate
grep calls (the agent's self-report disclosed only one). Correctness was,
if anything, better than any prior A-arm run — it independently caught the
`webster.go:75` non-issue (a func-typed field, not a method) that only the
2026-08-28 fasit (`c.json`) had previously found — but daemon warmth alone
produced no measurable speedup once a genuinely blind exploration path
replaced an already-informed operator's minimal-call path.

## Verdict

All three real-tool arms (A-mcp cold, A-cli, B) and even A-mcp warm-blind
reach the same perfect correctness on this task — quarry's structural
advantage on interface conflation shows up as answer quality, not speed,
here. The "warm MCP decisively wins" claim originally written here does
not hold up on this task: it was an artifact of the orchestrating session
already knowing the answer. Whether daemon warmth helps wall-clock on
tasks shaped like this one is unresolved with n=1 blind data — see
`mcp-capability-bench` (backlog), which exists specifically to settle this
with repeated, uncontaminated runs.
