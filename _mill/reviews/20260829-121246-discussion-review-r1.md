# Review: Per-capability quarry-mcp benchmark suite

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [NIT:consistency] Reused "Agent A preamble" is CLI-shaped, not MCP-shaped
**Demoted-from:** BLOCKING
**Section:** "Reusable assets from #006" (Technical context)
**Issue:** The discussion says README.md's "Agent A preamble" is reused verbatim and merely
"regenerated to describe only the tools that rung actually has." The actual preamble text
(`bench/loomyard-eval/README.md:84-166`, verified by reading it) is the *CLI* preamble from
#006's original bundle: `Binary: /tmp/quarry-bench`, and CLI verb syntax
(`quarry toc dir <path>`, `quarry refs <symbol> --target-dir <TARGET_DIR>`, etc.), not
`mcp__quarry__<tool>` MCP tool-call names. README.md contains zero mentions of "mcp" anywhere.
Trimming this template's verb list per rung (as instructed) still leaves the binary path and
CLI shell-command syntax intact, directly contradicting this task's own Scope line: "No
`/tmp/quarry-bench` CLI binary is built for any run." The real MCP-flavored preamble used for
#006's actual MCP arm lived only in an ad hoc, gitignored scratch runbook
(`.scratch/bench-mcp-runbook.md`, referenced by `results/2026-08-29/*/usage.json`'s own
`_note`/warm-experiment notes) that was never committed and no longer exists in this worktree.
**Suggested fix:** State explicitly that the A preamble must be *rewritten* as an MCP
tool-call preamble (naming `mcp__quarry__<tool>` for each allowed tool, no binary path, no
shell verb syntax) per rung, not merely trimmed from the existing CLI template.

### [BLOCKING:design] Cold-cell daemon state and run dispatch ordering are unspecified
**Section:** "Daemon warmth held constant, with one explicit comparison cell"
**Issue:** The decision says the main matrix runs against "a pre-warmed quarry daemon" and
separately that "one dedicated cell runs config A5 (bundle, task 01) N = 3 more times against
a cold daemon" — but never says how the harness actually forces cold state for that cell.
quarry's daemon is keyed per absolute target-dir path (`workspaceKey`, `internal/cli/paths.go`)
and only self-expires after `daemonIdleTimeout = 10 * time.Minute`
(`internal/quarryengine/daemon/ensureserver.go:143`). Task 01's worktree is shared read-only
across all 18 main-matrix task-01 runs (A0-A5 × N=3) *and* the 3 cold-cell runs. If the cold
cell runs anywhere near those 18 runs in time (which nothing in the discussion prevents), the
same-keyed daemon will almost certainly still be warm from a preceding run — silently
reproducing the exact orchestrator-adjacent contamination the two retracted #006 scorecards
(task 03/04) explicitly asked this task to resolve. The discussion also never states whether
the 45 runs dispatch sequentially or concurrently; that choice independently determines both
whether "cold" is achievable at all and whether `duration_ms` — the metric the reporting
discipline's disjoint-range rule leans on most heavily — stays comparable across configs.
**Suggested fix:** Add an explicit decision: e.g. the cold cell runs first, before any other
run touches task 01's worktree, or uses a second disposable worktree copy so its daemon key
never overlaps the main matrix's; and state the dispatch mode (sequential is consistent with
the existing "must be resumable, skip already-completed runs" constraint).

## Verdict

REQUEST_CHANGES
Two concrete, source-grounded gaps: the reused A-preamble template is CLI-shaped and would tell agents to invoke a binary the scope forbids building, and the dedicated cold-daemon cell has no stated mechanism for actually being cold given the daemon's real idle-timeout and worktree-sharing design.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
