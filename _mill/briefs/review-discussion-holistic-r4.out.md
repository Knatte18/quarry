MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/port-ladder-bench-to-go/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:design] Retry path drops per-attempt warm and restore
**Section:** § "The per-repetition session loop, and who owns retries"
**Issue:** The loop is specified per repetition (`warm` → dispatch → … → `restore-worktree`), but `run_matrix` in `scripts/run_ladder.py:854-873` calls `warm_daemon` and `restore_worktree` **inside** the attempt loop, unconditionally after every attempt, and `restore_worktree`'s docstring calls that placement load-bearing; the discussion never says where a retry re-enters.
**Fix:** State the retry sub-loop's step order explicitly — whether `warm` and `restore-worktree` run per attempt or per repetition — since a retry on an unrestored worktree misattributes `observe_worktree_dirtied` to the wrong attempt.

### [BLOCKING:design] Cold retry inside a live session has no owner
**Section:** § "The cold cell" / § retry ownership
**Issue:** `prepare-session` owns drain/build/clear/`gate_cold_before` but runs before launch; the text claims "the drain and the clear are re-run per attempt" without naming a subcommand a live session can call, and omits `build_worktree`/`remove_worktree`, which `run_cold_cell` (`run_ladder.py:958-972`) does run per attempt. Worse, the session's `quarry-mcp` is alive across attempts, so its daemon survives, while `clear_state_dir` deletes `daemon.json` — and `gate_cold_before` keys on a pid read from that file (`gates.py:333-366`), so attempt 2 would be reported cold against a warm daemon.
**Fix:** Decide whether a failed cold attempt relaunches the whole session, or name the subcommand that redoes remove/build/clear and how it detects a daemon whose state file was just deleted.

### [BLOCKING:design] Residual-limit list omits the session transcript
**Section:** § "Tool exposure…", the `skill_listing` residual-limit paragraph
**Issue:** The enumeration names only `~/.claude/skills/` as Bash-reachable, but the harness also writes `~/.claude/projects/-tmp-ladder-session-<id>/`, whose parent transcript carries `redact`'s scorer prompt — which embeds the unstripped fasit (`score_run.py:214-244`) — plus every `ladderbench` invocation naming `mcp__quarry__*` and the repo-root results path, in the same session that later dispatches reps 2 and 3 of a blinded config.
**Fix:** Either state this channel alongside the skills one and its detection story, or say plainly why the fasit reaching a later blinded rep is out of scope.

### [BLOCKING:decision] Run-dir settings/mcp copies left undecided
**Section:** § Technical context → "Results tree layout"
**Issue:** "plus the session's generated `settings.json` / `mcp.json` copies **if the port keeps recording them**" is an open option; `write_run_inputs` (`run_ladder.py:224-241`) writes both into the run dir today, so this is a parity item with no disposition.
**Fix:** State keep or drop, and if kept, name the subcommand that copies them into `<run_dir>/`.

### [NIT:scope] Single-flight enforced only within one config
**Section:** § "Single-flight dispatch, mechanically enforced"
**Issue:** `ingest`'s predicate compares rep `n` to rep `n-1` **of the same config**; nothing detects two concurrently launched sessions, which is the cross-config contention `run_matrix`'s docstring says makes wall-clock incomparable.
**Fix:** Say whether cross-session serialization stays operator discipline, and mark it as such rather than under "mechanically enforced".

### [NIT:consistency] `require_pins` vs the in-scope smoke launch
**Section:** § "`max_turns` becomes a post-hoc gate" / § "Session launch" smoke launch
**Issue:** With `run_model`, `max_turns`, and `run_effort` all shipping unset and `require_pins` rejecting each, `prepare-session` cannot run against the committed `ladder.yaml`; the discussion never says which subcommands call `require_pins`.
**Fix:** Name the subcommands that enforce `require_pins` and how the smoke launch satisfies it.

## Verdict

REQUEST_CHANGES
Retry paths, cold-attempt ownership, and one leak channel need decisions before planning.
MILL_REVIEW_END
