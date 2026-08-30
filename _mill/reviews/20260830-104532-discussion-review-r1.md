MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go

```yaml
duration_s: 188.7
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5
reviewed_file: /home/knatte/Code/quarry/wts/port-ladder-bench-to-go/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [NIT:consistency] Repo-tracked skill unreachable from scratch-dir cwd
**Demoted-from:** BLOCKING
**Section:** §Decisions "The orchestration loop lives in a repo-tracked skill" vs "The session's working directory is a neutral scratch dir" / §Session launch
**Issue:** `/ladder-run` is specified as project-local at quarry's `.claude/skills/ladder-run/SKILL.md`, but every session launches from `/tmp/ladder-session-*`, and `prepare-session`'s enumerated writes (`.mcp.json`, `.claude/settings.json`, `.claude/agents/*.md`) contain no skills entry — so the project-local skill is not in the session's project scope. The repo has no `.claude/` at all today (verified by glob).
**Fix:** State how the skill reaches the session — `prepare-session` copies it into the scratch dir's `.claude/skills/`, or it is installed under `~/.claude/skills/` — and note it under the same empirical-discovery risk already flagged for agent definitions.

### [NIT:consistency] `none` arm's `.mcp.json` contradicts a committed invariant
**Demoted-from:** BLOCKING
**Section:** §Session launch and agent-definition discovery
**Issue:** `prepare-session` is described as unconditionally writing `.mcp.json` declaring the `quarry` server, but `README.md:89-94` and `run_ladder.py:224-241` (`write_run_inputs`) make it a load-bearing enforcement rule that an empty `allowed` set gets *no* MCP config, "because a declared server named quarry exposing an `mcp__quarry__*` namespace would itself be the structural leak the blinding forbids" — and §Decisions still claims "the `none` arm never sees the `mcp__quarry__*` namespace at all".
**Fix:** Decide explicitly whether `prepare-session` omits `.mcp.json` for `allowed: []` configs, and if it does not, restate the blinding claim to match.

### [NIT:consistency] Session count 18 conflicts with the 15-config / 45-run arithmetic
**Demoted-from:** BLOCKING
**Section:** §Decisions "One session per config"
**Issue:** `ladder.yaml` holds 14 non-cold configs plus `a5-bundle-cold`; `plan_runs` (`run_ladder.py:794-808`) defines the 45 pairs as 14×3 main *plus* the cold config's 3. "One per each of the 15 configs, plus one per each of the 3 cold-cell repetitions" therefore counts `a5-bundle-cold` four times: the true count is 17. The same sentence's "dispatches 3 run agents plus 3 scorer agents" is also false for a cold session (1 run, 1 score).
**Fix:** Fix the count to 14 warm sessions + 3 cold sessions = 17, and state the per-session dispatch counts separately for warm and cold.

### [BLOCKING:design] Single-flight gate implies a step order never stated
**Section:** §Decisions "Single-flight dispatch" vs "One session per config"
**Issue:** `ingest` is to fail when the predecessor rep has no `run.json`, but `run.json` is written last, by `record-score` (`gates.py:505-517`, `541-548`). That makes "3 run agents, then 3 scorer agents" impossible — rep 2's `ingest` would fail while rep 1 is still unscored. The per-session step order is never written down.
**Fix:** Pin the session loop as fully serialized per rep (dispatch → `ingest` → `redact` → score → `record-score` → next rep), or weaken the single-flight predicate to something the intended order satisfies.

### [NIT:scope] Retry/`MAX_ATTEMPTS` bookkeeping has no owner
**Demoted-from:** BLOCKING
**Section:** §Subcommand surface / §Testing
**Issue:** `MAX_ATTEMPTS = 3`, `invalidate`, and the retry-then-halt policy (`run_ladder.py:831-878`) are listed as ported and `invalidate` is a named TDD case, and the correlation description embeds `attempt <k>` — but none of the ten subcommands invalidates a failed run dir, tracks the attempt index, or enforces the 3-attempt ceiling. `ingest` only prints an outcome.
**Fix:** Assign attempt bookkeeping — either an eleventh subcommand (e.g. `invalidate`) plus a stated attempt-index source, or an explicit decision that the skill owns it and the Go side only exposes the primitive.

### [BLOCKING:design] `run_env` scrubbing has no application point under Agent dispatch
**Section:** §Technical context (module-by-module: `run_env`) / §Session launch
**Issue:** `run_env` (`run_ladder.py:372-388`) exists so `QUARRY_STATE_DIR`/`QUARRY_BUILD_TAGS` never reach `quarry-mcp`, and `resolve_state_dir` hard-errors if they are present (`gates.py:297-300`) — the cold cell's whole argument rests on it. Under the new topology the server is spawned by Claude Code from the operator's shell, and the specified `.mcp.json` carries only `command` and `args`, no `env`.
**Fix:** State where the scrub now happens (an `env` block in the generated `.mcp.json`, a `prepare-session` precondition check, or both) and which env the Go gates resolve against.

### [BLOCKING:design] `denied_tool_attempts` rests on an unobserved denial shape
**Section:** §Decisions "Metrics: what survives, changes, and dies"
**Issue:** The new source is "`tool_result` blocks marked `is_error` whose text matches a permission-denial shape" — replacing the structured `permission_denials` array (`extract_usage.py:189`). No such shape is quoted, and unlike the transcript format nothing in §Technical context claims a real denial record was observed; since neither probe runs in this task, the pattern would ship unvalidated against reality.
**Fix:** Quote the observed denial text/shape, or state that the field is provisional and how the follow-up matrix task validates it before the numbers are trusted.

### [NIT:scope] Metric enumeration omits two existing `usage.json` fields
**Section:** §Decisions "Metrics: what survives, changes, and dies"
**Issue:** The survives/changes/dies partition reads as exhaustive but never disposes of `model` or `transcript`, both written today (`extract_usage.py:193-195`); `transcript_source` is listed as "Added" without saying `transcript` is the field it replaces.
**Fix:** Add both to the partition.

## Verdict

REQUEST_CHANGES
Skill discovery, `none`-arm MCP config, session count, and retry ownership are unresolved.
_Note: 4 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 3._
MILL_REVIEW_END
