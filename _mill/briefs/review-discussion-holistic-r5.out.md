MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (per runtime metadata); best-effort self-assessment is an Anthropic Claude Opus-tier model, version not independently verifiable from inside the session
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:consistency] Q&A contradicts the worktree-dirtiness disposition
**Section:** Q&A log ("The shared task worktree is described as read-only") vs Technical context ("The shared worktree is writable…") and the later Q&A ("A run that edits the target and runs `go build`…")
**Issue:** The first Q&A entry states "dirty tree invalidates that run" and that "invalidating rather than only restoring matters"; the Technical-context worktree paragraph, the Testing gates ("Worktree dirtiness is **recorded, not gated**"), and the later Q&A all state the opposite — unconditional restore, `worktree_dirtied` recorded as an observation only.
**Fix:** Delete or rewrite the superseded Q&A entry so one disposition is stated; the bias argument for not gating is only in the surviving text.

### [BLOCKING:design] Config ids are not unique across ladders or the cold cell
**Section:** Ladder shape tables; Layout under `bench/loomyard-eval/ladder/`; Runs dispatch sequentially
**Issue:** The `config id` column reuses `none` (A0 and B0) and `bundle` (A5 and B7), and the cold cell is "config A5 (`bundle`, task 01)" with no id of its own — yet `raw/<config-id>/<n>/`, the `run.json` resume key, `summarize.py`'s per-config grouping, and the "each config has exactly 3 complete runs" gate all key on config id. Three pairs of runs collide on the same directory, and the disjoint-range rule's "its own ladder's `none` control" is unresolvable from the id alone.
**Fix:** State ladder-qualified unique ids in the tables (and an explicit id for the cold cell), and say that `ladder.yaml`'s `id` is that unique key.

### [BLOCKING:design] Rewritten A preamble drops the parallel-tool-calls directive
**Section:** Technical context, "The **Agent A preamble must be rewritten, not reused**"
**Issue:** The carry-over list is exclusive ("carries over only … prefer quarry over grep, do not re-verify …, pass a known `file:line:character` position"), so the generated A preamble loses the `USE PARALLEL TOOL CALLS` / `<use_parallel_tool_calls>` block that `bench/loomyard-eval/README.md`'s committed A **and** B preambles both carry. The `none` control keeps it (B preamble used unchanged), so every quarry rung would be batching-unsteered against a batching-steered control — a second, unrecorded confound landing directly on `num_turns` and `duration_ms`, the metrics the "disjoint from `none`" benefit rule depends on.
**Fix:** State whether the parallel-tool-calls block is carried into the generated A preamble (and assert it in the preamble-generation tests), or list it alongside the anti-grep steering in the recorded-confound section.

### [BLOCKING:consistency] Scorer input contract stated two ways
**Section:** "Correctness scoring against the existing fasit" vs "Layout under `bench/loomyard-eval/ladder/`"
**Issue:** The decision says the scorer's input is "exactly three things: the run's `answer.json`, the task's fasit `c.json`, and the task's `<TASK TEXT>`", with rationale that Ladder A's `summary` judgement is impossible without the task text; the layout paragraph says `score_run.py` "is handed only the run's `answer.json` and the task's fasit `c.json`". A plan writer following the layout text builds a scorer that cannot apply Ladder A's rule.
**Fix:** Make the layout paragraph restate the three-input contract, or drop its enumeration and reference the decision.

### [NIT:design] "Cannot read quarry's source" is a cwd claim, not an enforcement
**Section:** Hard enforcement via generated per-run settings deny-list
**Issue:** "Each run's process working directory is its own task worktree … so no run of any config can read quarry's source, docs, or `.mcp.json`" — but Bash is in the allow-set and cwd does not bound Bash reads; the actual guard is the post-hoc gate "a `none` run's transcript contains no `mcp__quarry__*` tool and no occurrence of 'quarry'".
**Fix:** Reword the premise as detection-based (the transcript gate) rather than structural, so the blinding constraint's coverage is stated accurately.

### [NIT:scope] Cold-cell daemons outlive their runs; matrix order unspecified
**Section:** Daemon warmth held constant / Runs dispatch sequentially
**Issue:** A supervised cold run leaves a gopls daemon alive for `daemonIdleTimeout = 10 * time.Minute` (`internal/quarryengine/daemon/ensureserver.go:143`) against a worktree the harness then deletes; sequential dispatch is required but the run order is only called "a defined order", so up to three leaked daemons could be resident during later timed runs.
**Fix:** State where the cold cell sits in the defined order (or that leaked daemons are drained) so `duration_ms` is not silently loaded.

## Verdict

REQUEST_CHANGES
Four blocking issues: a superseded Q&A entry, colliding config ids, a dropped preamble directive, and a split scorer contract.
MILL_REVIEW_END
