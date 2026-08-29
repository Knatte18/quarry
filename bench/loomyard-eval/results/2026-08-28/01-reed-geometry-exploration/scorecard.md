# Scorecard — Task 01: reed terminal-geometry exploration

Dispatched directly by the top-level orchestrating session (three parallel
top-level `Agent` calls — **not** a subagent dispatching further subagents;
an earlier attempt at that pattern produced fabricated results and was
reverted, see commit history). Ran against Loomyard pinned to
`975578cda8d6f3a81580bd4e73725e060211b766`, worktree at
`/tmp/loomyard-eval-01` (removed after this run).

## Correctness (A/B vs C)

**`relevant_files`** (C listed 20 files total — notably more exhaustive than
A/B: it includes test files, render helpers, config templates, and one
out-of-scope-but-related caller in `internal/loomcli/run.go`):

| | recall | precision |
|---|---|---|
| A | 8/20 = 40% | 8/11 = 73% |
| B | 12/20 = 60% | 12/16 = 75% |

**`key_symbols`** (C named 20 symbols):

| | recall | precision |
|---|---|---|
| A | 11/20 = 55% | 11/12 = 92% |
| B | 10/20 = 50% | 10/11 = 91% |

Roughly a wash — A ahead on symbol recall, B ahead on file recall, both
within noise for a single sample. Both A and B independently arrived at the
same correct core mechanism C confirmed: geometry is never persisted;
`AttachArgv` builds a layout from the *told* client size (never a live tmux
query, since the live window is still pre-attach-sized at that point) and
chains it after `attach-session` so it lands after tmux has already resized
the window — avoiding tmux's silent proportional rescale of a mismatched
layout string. Both also correctly identified the pin-then-readback pattern
(`pinGeometryOptionsLocked` + `readWindowSizeLatestLocked`/`readStatusRowsLocked`)
as the actual reconciliation point. Neither A nor B fabricated anything
contradicted by C.

**Shared false positive:** both A and B listed `internal/reedengine/geometry.go`
as relevant. C explicitly flagged this as a naming trap — that file is
identity/path plumbing (`SocketKey`, `SessionName`, ...), not terminal
geometry — and correctly excluded it. Neither A nor B verified this deeply
enough to catch it; this was anticipated in task 01's own "Notes for scorer"
section as a plausible trap. Not a quarry-vs-no-quarry differentiator since
both arms made the same mistake.

**What only C caught:** the `errAttachChainSuppressed` sentinel name, the
`parseWindowSize`/`reservedRowsFromStatus`/`windowSizeAllowsChain` low-level
helpers, the reserved-row floor (`reserved > rows-1`), the second production
caller of `AttachArgv` in `internal/loomcli/run.go`, the still-open gap
("resize while attached" has no reed-side reaction at all, deferred to a
future watchdog daemon per `manifest/roadmap.md`), and live verification via
`git log 8b14f32b..HEAD` plus running the actual unit test suite. This is
consistent with C's role (high-effort, no budget pressure, cross-checked)
rather than a finding about A or B being wrong.

## Efficiency (A vs B only — comparable since neither was wrong)

| | tokens | tool_uses | duration |
|---|---|---|---|
| A (quarry) | 82,441 | 21 | 146.7s |
| B (baseline) | 82,611 | 22 | 133.7s |
| C (fasit, for reference only) | 113,851 | 47 | 359.8s |

A and B are within noise of each other on every axis. A was very slightly
cheaper (170 fewer tokens, 1 fewer tool call) but took *longer* wall-clock
than B. No measurable efficiency edge for quarry on this task.

## A notable methodological finding: tool distrust

Inspected A's actual tool-call transcript (not just its final report). Of
A's 12 `Bash` invocations, only ~7 were `quarry-bench` calls (`toc dir`/`toc
file`) — the other 4 were plain `grep -rn`/`grep -n`/`grep -rln` commands,
run despite quarry being available and the prompt telling A to prefer it.
A never used `refs`, `definition`, or `impact` at all — only `toc`, plus a
grep fallback for exactly the kind of cross-reference question those verbs
exist to answer. This is likely why A's resource usage didn't come in lower
than B's: A effectively paid for *both* toolsets rather than substituting
one for the other. Worth tracking as an explicit metric
(`grep_fallback_count` or similar) in future runs rather than assuming an
agent given a tool will rely on it exclusively.

## Revision history (v1 -> v5, A only)

The findings above were captured from A's v1 run (original preamble, `toc`
lacked `--target-dir`, no explicit "prefer quarry" instruction). Debugging
A's tool usage across four reruns turned up two real quarry bugs, fixed and
regression-tested in the same session (see `internal/cli/toc.go`,
`internal/cli/cli.go`, `internal/cli/impact.go`, and the new
`refs_targetdir_lsp_test.go`):

1. `toc dir`/`toc file` rejected `--target-dir` outright (`unknown flag`),
   despite every other verb accepting it — an agent pattern-matching the
   flag across verbs paid for a failed call before self-correcting.
2. `refs`/`definition`/`assert-no-callers`/`impact` resolved a relative
   `file:line:col` position (and `--in-file` paths) against the process's
   own cwd instead of `--target-dir` — silently wrong the moment cwd and
   `--target-dir` diverged, which they always do for an agent invoked from
   the quarry repo but targeting a different codebase.

| run | change | tokens | tool_uses | duration_ms | grep calls |
|---|---|---|---|---|---|
| v1 | baseline | 82,441 | 21 | 146,674 | 4 |
| v2 | + strict "prefer quarry" instruction (toc bug still present) | 98,107 | 25 | 238,468 | 0 |
| v3 | + `toc --target-dir` fix | 85,374 | 34 | 222,915 | 0 |
| v4 | + position-arg resolution fix | 94,789 | 29 | 188,315 | 0 |
| v5 | + prompt tip: reuse a position already known from `toc` instead of a bare-name search | 86,509 | 26 | 193,128 | 0 |

Tool-call count trended down (34 -> 26) as each real bug was fixed; token
count is noisy but not clearly improving; duration plateaued around 190s,
still well above B's 133.7s.

**Direct latency measurement, to rule out a stale-daemon explanation:**
timed `quarry refs`/`impact` calls against this exact worktree, both against
a brand-new `--state-dir` (forcing a fresh gopls process) and against the
long-running default daemon (confirmed alive via `ps`, started by A's first
v1 call and still warm hours later, correctly keyed by `--target-dir`).
Every call — cold or warm — returned in 0.5-0.7s; `toc` (tree-sitter, no LSP
at all) in 5-7ms. 26 tool calls at ~0.7s worst case is ~18s of actual quarry
execution time against a 193s total. **quarry's own execution time cannot
explain the gap** — the remaining ~175s is the subagent's own reasoning and
prose-generation time between tool calls (this task's answer is a multi-
paragraph free-text explanation, not a bounded structured verdict), which
neither the CLI nor the prompt can reduce further. `duration_ms` is
therefore not a fair quarry-effectiveness signal for this specific task
shape; tokens/tool_uses and correctness remain meaningful, duration is not.

## Verdict

**No measurable quarry advantage on this task, on this single run.** Both A
and B reached essentially the same correct understanding, at essentially the
same cost, and made the same mistake (the `geometry.go` naming trap). The
one real signal specific to quarry's presence is behavioral, not
performance-based: given access to quarry, A still hedged with grep for
roughly a third of its tool calls rather than reaching for `refs`/`impact` —
suggesting the win (if any) may require either a stronger prompt nudging
toward the caller-lookup verbs specifically, or tasks where grep is
structurally worse at the job (e.g. interface-dispatched calls), not just
"has quarry available." Task 01 (a single flat "explain this mechanism"
prompt) may not stress that gap. Re-run recommended before drawing a firm
conclusion, and tasks 03/04 (code review, `impact`-focused, where grep is
much weaker at finding interface-dispatched callers) are a more promising
place to look for a real quarry advantage than further exploration tasks
like this one.
