# Scorecard — Task 03: reintroduced missed-caller bug (MCP arm)

Rerun of the 2026-08-28 task with quarry exposed as native MCP tools instead
of CLI-via-Bash. B and C reused unchanged. Same execution/verification
method as task 01's scorecard — see its header note.

## Correctness (A-mcp vs A-cli vs B)

All three arms — A-mcp, the 2026-08-28 CLI arm, and even the no-quarry
baseline B — independently found the exact same bug:
`internal/loomcli/run.go:295` still calls the deleted free function
`attachArgv`, a hard compile break. A-mcp additionally ran `go build
./internal/loomcli/...` itself as independent corroboration; the compiler's
own error (`run.go:295:46: undefined: attachArgv`) matched quarry's
`textDocument_definition`/`workspace_symbol` findings exactly, at the same
line.

This task is a known weak differentiator for tool choice — task 04's own
design notes call it out explicitly: "its bug was a plain undefined-symbol
compile break, which `go build` alone catches with certainty regardless of
tool, leaving quarry's LSP precision nothing to differentiate on." All three
arms getting it right, including the completely tool-blind B, confirms that
diagnosis again here. Correctness is not the axis to read this task on.

## Efficiency

| | tool_uses | duration | grep/CLI fallback |
|---|---|---|---|
| A-mcp (2026-08-29) | 4 | 77.9s | 0 |
| A-cli (2026-08-28) | 21 | 128.9s | 2 |
| B, no quarry (2026-08-28) | 8 | 52.1s | -- |
| C, fasit (2026-08-28) | 36 | 270.9s | -- |

A-mcp is a clear efficiency win over the CLI arm on every axis (4 vs 21
tool_uses, 78s vs 129s, 0 vs 2 grep fallbacks) — the CLI arm needed two grep
escapes despite being told to prefer quarry; A-mcp needed none. But B (no
quarry at all, plain Read/Grep/`go build`) is still faster and used fewer
tool calls than A-mcp (8 calls/52s vs A-mcp's 4 calls/78s — fewer calls for
A-mcp, but more wall-clock, plausibly LSP/daemon warm-up cost on a freshly
built worktree that a pure Read/grep baseline never pays).

## Warm-daemon rerun (2026-08-29, same day)

The cold-vs-baseline comparison above is an artifact of this benchmark's own
design: every task builds a brand-new disposable `git worktree`, so the
daemon's state (keyed by an absolute-path hash of `targetDir`, see
`mcp-target-dir-ergonomics`) has never seen it before and pays gopls's full
workspace-index cost on the first call. That is not how quarry is meant to
run in production — Lyx starts it at boot and keeps it running as a daemon
continuously; a real agent, hours into a session, always meets an
already-warm quarry.

Rerun with one throwaway warm-up call (`toc_dir` against the same worktree)
issued *before* starting the clock, then timing only the actual task:

| | tool_uses | duration | grep/CLI fallback |
|---|---|---|---|
| A-mcp, warm daemon | 2 (+1 Bash `go build` check) | **13.8s** | 0 |
| A-mcp, cold daemon | 4 | 77.9s | 0 |
| A-cli, cold (2026-08-28) | 21 | 128.9s | 2 |
| B, no quarry (2026-08-28) | 8 | 52.1s | -- |

Warm MCP is ~3.8x faster than the no-tool baseline B, not slower. Same
finding (`attachArgv` undefined, `go build` confirms at the identical line).

Caveat: this rerun was executed directly by the orchestrating session
(already familiar with the codebase from the cold run), not a fresh blind
agent — so part of the gain could be "already knew what to query" rather
than pure daemon warmth. Tool-call count dropped too (4 -> 2), which is the
more daemon-attributable signal, since no exploratory calls were added or
removed, only the same two calls run against a pre-warmed index.

## Correction: the warm rerun above was contaminated, not representative

The 13.8s warm number above was produced by the orchestrating session
itself, which — by the time it ran that rerun — had already read this
task's own `Ground truth` section (file+line of the real bug) earlier in
the same conversation, while building the benchmark runbook. That run
was not exploration, it was verification of an already-known answer with
the minimum tool calls needed to cite evidence. It measures "zero
orientation cost", not daemon warmth.

A genuinely blind fresh agent was then run against the same pre-warmed
worktree (`bench/loomyard-eval/results/2026-08-29-warm/`,
`.scratch/bench-mcp-runbook-warm.md`), with no memory of this
conversation. Result: **89.4s — slower than the cold-daemon run (77.9s),
not faster.** Same correct verdict, same bug found, but a more roundabout
path: 10 tool calls including 3 `textDocument_definition`/3
`workspace_symbol` calls plus one grep the agent's own self-report did not
disclose (see that run's `usage.json` — a separate, real self-report
discrepancy caught by transcript verification, not just an undercount).

## Verdict

Cold: MCP fixed the CLI arm's grep-fallback problem but didn't beat the
no-tool baseline on raw speed. The "warm MCP decisively wins" claim
originally written here does not hold up — it was an artifact of the
orchestrating session already knowing the answer, not of daemon warmth.
The one clean warm-vs-cold comparison available (task 01, blind agent both
times) showed a real but modest ~25% improvement, not the 3-4x this task's
contaminated rerun implied. Whether warm daemon reliably helps on tasks
shaped like this one is genuinely unresolved with n=1 blind data —
exactly the gap `mcp-capability-bench` (backlog) exists to close with
multiple repetitions per configuration.
