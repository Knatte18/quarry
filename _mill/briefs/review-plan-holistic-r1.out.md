MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5
reviewed_file: plan/
date: 2026-08-29
```

## Findings

### [BLOCKING:design] `gate_cold_after`'s premise is false for toc-only runs
**Location:** batch 3 card 9; batch 6 card 21; batch 8 card 26
**Issue:** `internal/mcpserver/tools_toc.go` implements `toc_file`/`toc_dir` as tree-sitter-backed and never reaches `resolveCall`→`EnsureServer`, so no `daemon.json` is written; the cold cell is `a5-bundle-cold` on task 01 (`internal/reedengine`/`internal/reedcli` exploration), a run that can plausibly use only the toc tools, and the plan states absence of `daemon.json` "means the native fallback was taken".
**Fix:** distinguish "no LSP-backed tool was invoked" from "native fallback" — gate on the transcript containing at least one daemon-backed quarry call before asserting `gate_cold_after`, and give that third outcome its own disposition.

### [BLOCKING:scope] `warm_daemon` and the probe need an MCP client nobody plans
**Location:** batch 6 card 17 (`warm_daemon`), card 18
**Issue:** `warm_daemon(server_path, target_dir, env)` must "start the server against the target dir, issue one cheap tool call, and shut it down" — that is an MCP stdio initialize/`tools/call` handshake in Python, but the Python-style decision restricts the suite to stdlib + PyYAML and no card names a transport, framing, or handshake; separately "one cheap tool call" is unnamed, and a `toc_*` call would warm no daemon at all.
**Fix:** name the concrete warm-up mechanism and pin the warm-up tool to a daemon-backed one (`workspace_symbol`), or state the dependency the harness may take.

### [BLOCKING:consistency] Resume contradicts `build_worktree`'s existence check
**Location:** batch 6 cards 16 and 22; batch 8 card 25
**Issue:** `build_worktree` "raises `HarnessError` when the worktree already exists at that path", and card 22's CLI order runs `ensure_task_worktrees` before `run_matrix` on every invocation; nothing removes the main-matrix worktrees, so the documented "re-invoke to resume" path halts on the first re-invocation. The CLI also re-runs the paid probe and rewrites the already-committed `probe.json` on every resume.
**Fix:** state `ensure_task_worktrees`' idempotent behaviour for an existing worktree at the declared pin, and make the probe skip when a valid `probe.json` is already present.

### [BLOCKING:scope] `task_text_for` cannot get a schema from task 01
**Location:** batch 6 card 19
**Issue:** the card requires extracting "the task file's `<TASK TEXT>` block and the task's output schema … from the committed task file rather than from a copy", but `tasks/01-reed-geometry-exploration.md` has only `## Setup`, `## Scope`, `` ## `<TASK TEXT>` ``, `## Notes` — no schema section; the exploration schema lives only in `bench/loomyard-eval/README.md`'s "Output schemas". Card 19's `Context:` also omits both task files whose markup its parser must handle.
**Fix:** state the per-task schema source explicitly (task file for `impact`, benchmark README for `exploration`) and add both task files to card 19's `Context:`.

### [BLOCKING:design] No field links the cold cell to its warm counterpart
**Location:** batch 5 card 14 (`compare_warm_cold`)
**Issue:** the builder takes only `(ladder, cells)` and must resolve `a5-bundle` beside `a5-bundle-cold`, but card 2's `configs:` schema carries only `id`/`ladder`/`task`/`allowed`/`cold` — the only available route is parsing the id suffix, which the plan explicitly forbids for `control_for` ("never by parsing its id string").
**Fix:** add a declared field on the cold entry naming its warm counterpart and resolve through it, as `control_for` resolves the control.

### [BLOCKING:scope] Batch 5 re-derives `is_complete` without depending on batch 3
**Location:** batch 5 card 13; overview `batches:` entry 5
**Issue:** `load_runs` must skip a run "whose `run.json` does not record `state: \"complete\"`" — that predicate is `is_complete` in `gates.py` (batch 3 card 10), but batch 5's `depends-on` is `[1]` and card 13's `Context:` omits `gates.py`, so the implementer either cold-starts or silently forks a second definition of run completeness.
**Fix:** add batch 3 to batch 5's `depends-on` and `gates.py` to card 13's `Context:`, and state that `load_runs` calls `is_complete`.

### [BLOCKING:consistency] `summary.json` promises fields its loader never reads
**Location:** batch 5 cards 13 and 15
**Issue:** card 15 requires `_meta.denied_tool_attempts_reported` "so the summary states whether that metric carries information", yet `denied_tool_attempts` is absent from card 13's `METRICS`, so the boolean qualifies nothing; likewise `worktree_dirtied_count` and `target_origin_quarry_mention_count` are "aggregated from the runs' recorded observations", which card 10 writes into `run.json`, while `load_runs` reads only `usage.json` and `score.json`.
**Fix:** add `denied_tool_attempts` to `METRICS` (or drop the `_meta` boolean), and state that `load_runs` also reads `run.json`'s observations.

### [BLOCKING:design] Two client flags are asserted without the planning check the others got
**Location:** batch 6 card 19 (`--permission-mode dontAsk`); batch 4 card 12 (`--effort`)
**Issue:** `--setting-sources ""` and `--strict-mcp-config` carry an explicit "confirmed accepted by the installed client (2.1.236) while planning" note, and the `permissions.allow` decision was established empirically; `--permission-mode dontAsk` and the scorer's `--effort` carry no such confirmation, and every one of the 46 dispatches depends on them being accepted.
**Fix:** record the same acceptance check for both flags in the overview, or name the verified spelling the harness uses.

### [BLOCKING:scope] "Wait for the previous run's daemon to exit" names no mechanism
**Location:** batch 6 card 21
**Issue:** no function, predicate, or bound is named for this wait; `daemon.json` is never removed on exit (`ensureSupervised` cleans only `daemon.sock`), so presence is not a liveness signal, and `daemonIdleTimeout` in `internal/quarryengine/daemon/ensureserver.go` is 10 minutes — an unbounded or naive wait adds ~30 minutes to batch 8 or never terminates.
**Fix:** name the liveness predicate (the recorded PID) and the bound, in a helper with a stable identifier.

### [NIT:consistency] `wall_clock_ms` is extracted but never summarised
**Location:** batch 2 card 7; batch 5 card 13
**Issue:** `extract_usage` records `wall_clock_ms` and card 20's sequential-dispatch rationale rests on wall-clock comparability, but `METRICS` carries `duration_ms` only.
**Fix:** include `wall_clock_ms` in `METRICS` or state why only `duration_ms` is reported.

### [NIT:design] `load_ladder` does not reject two controls in one ladder
**Location:** batch 1 card 3
**Issue:** the raise list covers "a ladder with no config whose `allowed` is empty" but not more than one, leaving `control_for` with an undefined result.
**Fix:** add the "at most one empty-`allowed` config per ladder" raise and its test.

## Verdict

REQUEST_CHANGES
Cold-cell premise, warm-up mechanism, resume path, and schema sourcing need resolution.
MILL_REVIEW_END
