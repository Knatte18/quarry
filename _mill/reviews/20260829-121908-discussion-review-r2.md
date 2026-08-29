MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite

```yaml
duration_s: 273.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic)
reviewed_file: /home/knatte/Code/quarry/wts/mcp-capability-bench/_mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:design] No model pin or permission mode for the 45 runs
**Section:** § Hard enforcement via generated per-run settings deny-list
**Issue:** The decision fixes `--mcp-config` and `--settings` but never names the model each `claude -p` run uses, nor the permission mode/allow-list that lets a headless run call Read/Grep/Bash at all; `duration_ms` and every token class — the suite's headline metrics — are model-dependent, so an unpinned model makes cells incomparable and the matrix unreproducible.
**Fix:** State the exact `--model` value (and whether it is held identical across all 45 runs) and the permission mode / allow configuration under which the non-quarry toolset is usable.

### [BLOCKING:design] Ladder B has no correctness scoring definition
**Section:** § Correctness scoring against the existing fasit
**Issue:** The decision defers to `bench/loomyard-eval/README.md`'s "Scoring" section, but that section defines recall/precision only for **Exploration** and **Code review**; task 04 uses the third schema (`callers_to_update` / `excluded_lookalikes`), which no bullet there covers, and #006's own task-04 scorecard scored it ad hoc ("recall/precision on the scored core question"). 24 of 45 runs have no stated metric.
**Fix:** Write the impact-analysis recall/precision rule explicitly, including how `excluded_lookalikes` and a `burler.go:373`-style decoy appearing in `callers_to_update` are scored.

### [NIT:consistency] grep_fallback_count does not match #006's definition
**Demoted-from:** BLOCKING
**Section:** § Metrics — full benchmark accounting
**Issue:** The metric is defined as "Grep tool calls plus Bash invocations containing `grep`/`rg`, matching #006's definition so the numbers stay comparable", but #006's definition (README "Dispatch protocol" step 4) counts only Bash `"command"` fields matching `grep`/`rg` — Grep *tool* calls are outside it, so the broadened definition contradicts the stated comparability claim.
**Fix:** Either adopt #006's Bash-only definition, or keep the broader one and drop the comparability claim, reporting the two components as separate fields.

### [NIT:consistency] `none` arms are launched with the quarry MCP config
**Demoted-from:** BLOCKING
**Section:** § Hard enforcement … vs § Constraints (structural blinding)
**Issue:** The enforcement decision says *every* run is launched with `--mcp-config` pointing at the quarry server and `none` simply denies all seven, while the constraint requires `none` arms never to encounter the word "quarry" in prompt or reachable filesystem — the server name and `mcp__quarry__*` namespace are exactly that. The same unverified premise cuts the other way for `denied_tool_attempts`: if `permissions.deny` filters denied tools out of the advertised list, that metric is identically zero in every restricted rung.
**Fix:** Decide whether `none` runs omit `--mcp-config` entirely, state the run's cwd (the quarry repo must not be reachable), and state the verified `permissions.deny` visibility semantics that `denied_tool_attempts` depends on.

### [BLOCKING:design] Cold-cell keying holds only on the supervised strategy
**Section:** § Daemon warmth held constant, with one explicit comparison cell
**Issue:** "quarry keys its daemon per absolute target-dir path (`workspaceKey`)" is true only for `ensureSupervised` (socket derived from `stateDir`); `EnsureServer` falls back to `ensureNative`, whose own doc comment states the shared `-remote=auto` daemon address "is not a function of the state directory at all" (`internal/quarryengine/daemon/ensureserver.go:151-165`), and that path writes no `daemon.json`, so the harness's "no daemon state exists for that key" check passes vacuously. `ResolveStateDir` (`internal/cli/paths.go:118-134`) also lets `--state-dir`/`$QUARRY_STATE_DIR` bypass `workspaceKey` entirely.
**Fix:** State that the cold cell asserts a supervised connection (and how it detects the native fallback), and that `QUARRY_STATE_DIR`/`--state-dir` are pinned or cleared for every run.

### [NIT:design] Resumability cannot distinguish complete from invalidated
**Section:** § Runs dispatch sequentially / § Protocol validation gates
**Issue:** Re-invocation "skips already-completed runs", while a failed gate "invalidates the affected cell, which is re-run" — with no stated marker separating the two states, an invalidated run is indistinguishable from a completed one on disk.
**Fix:** Name the on-disk completion/invalidation record the harness writes and how invalidation clears it.

### [NIT:design] Per-call `targetDir` overrides the launch pin
**Section:** § Technical context (server launch)
**Issue:** Every tool's call-wide input carries an optional `targetDir` that overrides the launch default (e.g. `assertInput.TargetDir`, `internal/mcpserver/tools_assert.go:34`), so `--target-dir` is an instructed rather than structural pin for both the worktree constraint and the daemon key.
**Fix:** Note the override in the technical context and state whether the generated preamble forbids setting `targetDir`.

### [NIT:scope] Task 04 retained on the same evidence that excluded 03
**Section:** § Reuse existing tasks 01 and 04
**Issue:** Task 03 is excluded as non-discriminative per its own postmortem, but task 04's committed scorecard records both arms at 100% recall/precision and "effectively identical cost" ("quarry did not measurably help even here"), so Ladder B's correctness axis is likely saturated across all 8 rungs; the asymmetry is never reconciled.
**Fix:** State explicitly that Ladder B is expected to discriminate on efficiency only, and what a fully saturated correctness axis means for its conclusion.

## Verdict

REQUEST_CHANGES
Five blocking gaps: run model, Ladder B scoring, metric definition, blinding, cold-cell premise.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 3._
MILL_REVIEW_END
