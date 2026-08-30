# Discussion: Port the capability-ladder bench harness to Go

```yaml
task: Port the capability-ladder bench harness to Go
slug: port-ladder-bench-to-go
status: discussing
parent: main
```

## Problem

Task #008 (`mcp-capability-bench`) built the per-capability benchmark harness under `bench/loomyard-eval/ladder/` in Python: six modules under `scripts/`, six pytest files under `tests/`, ~5.5k lines total.
That was justified at the time by local precedent (`bench/loomyard-eval/scripts/gen_compact_toc.py`) and by TDD suitability, but the repo's actual policy is Go-only.
The concrete reason the policy exists: on the user's Windows 11 work machine, PowerShell 5 startup (~1s) plus Python interpreter load (~0.8s) adds ~1.8s of pure overhead to every invocation — every `pytest` verify run, every script call — which is tolerable on Linux with `uv` but not on the machine the user actually works from day to day.
#008 was allowed to finish in Python as a grandfathered exception because 5 of its 9 batches were already approved and committed when this was raised mid-flight.

#008 shipped through batch 7 (the harness plus its pytest suite) but never ran the matrix.
The execute-matrix and conclusion batches were descoped once the user realised the approved plan had `run_ladder.py` dispatching all 45 runs itself as headless `claude -p` subprocesses, with no way to watch or kill an individual run gone rogue — only a wall-clock guess.

This task therefore covers two things at once.
First, port the harness's mechanical pieces to Go as CLI tools: worktree setup, prompt/settings/deny-list generation, usage extraction, gating, blinded scoring, and disjoint-range summarization.
Second, replace the `claude -p` subprocess dispatch entirely: the per-run and per-scoring-call dispatch must go through the Agent Tool, one call per run, from a live Claude Code session the user is watching, so a rogue run can be killed on sight.
Because the Agent Tool exists only inside a live session, the orchestration loop itself cannot live in a standalone binary — the Go pieces do the mechanical work,
and the session does the dispatching.

## Scope

**In:**

- A single new Go binary `bench/loomyard-eval/ladder/cmd/ladderbench` (cobra, ten subcommands) plus its logic package `bench/loomyard-eval/ladder/internal/ladder/`.
- Go ports of every unit currently in `bench/loomyard-eval/ladder/scripts/`: ladder-config loading and validation, deny-list and settings derivation, preamble generation, fenced-JSON extraction, transcript parsing and usage extraction, every validation gate, answer redaction and scorer-prompt assembly, run-state and resume bookkeeping, and summary/median/disjoint-range building.
- Generation of per-config Claude Code **agent definitions** (`tools:` allowlist frontmatter) and per-session `.mcp.json` / `settings.json`, replacing the per-run `claude -p` flag assembly.
- Replacement of the transcript source: `claude -p --output-format stream-json` transcripts become Claude Code **subagent transcripts** (`agent-<id>.jsonl` + `agent-<id>.meta.json`).
- A repo-tracked orchestration skill at `.claude/skills/ladder-run/SKILL.md`, invoked as `/ladder-run`, that drives one config's session end to end.
- Go tests: per-unit table-driven tests plus one synthetic end-to-end test over a hand-written subagent transcript; fixtures moved from `ladder/tests/fixtures/` into `internal/ladder/testdata/`.
- Rewrite of `bench/loomyard-eval/ladder/README.md` in place, and refresh of `ladder.yaml`'s header comments.
- Two new `ladder.yaml` fields: `run_effort` and `session_dir_template`.
- Deletion of `bench/loomyard-eval/ladder/scripts/` and `bench/loomyard-eval/ladder/tests/` in the final batch, and removal of the `bench/loomyard-eval/ladder/` clause from `CLAUDE.md`'s grandfather note.

**Out:**

- **Running the 45-run matrix.** This task delivers the harness only; matrix execution and `conclusion.md` are a separate follow-up task. No paid run of any kind — not even a preflight probe — is made by this task.
- The sibling suite at `bench/loomyard-eval/` outside `ladder/`: its `tasks/`, `results/`, `README.md`, and `scripts/gen_compact_toc.py` are untouched. `gen_compact_toc.py` keeps its grandfathered exception in `CLAUDE.md`.
- The `quarry` and `quarry-mcp` binaries themselves, `internal/`, `cmd/`, and `quarry/`. This task builds `quarry-mcp` as a dependency; it does not modify it.
- Any committed results under `bench/loomyard-eval/ladder/results/` or `bench/loomyard-eval/results/`.
- Windows execution of the matrix. The Go code is written portably, but the matrix targets a pinned Linux checkout and is not expected to run on Windows.
- Millhouse. This harness is quarry-specific; nothing about it belongs in the millhouse repo.

## Decisions

### Code location and binary shape

- Decision: one binary at `bench/loomyard-eval/ladder/cmd/ladderbench` with cobra subcommands, and the logic in `bench/loomyard-eval/ladder/internal/ladder/`.
- Rationale: benchmark tooling stays out of the product's `cmd/` namespace, which currently holds only `quarry` and `quarry-mcp`. Go's `internal/` visibility rule scoped under `bench/loomyard-eval/ladder/` keeps the package unimportable from the product tree. One build, one path in the runbook.
- Rejected: `cmd/ladderbench` at repo root (puts bench tooling in the product's binary namespace); five small binaries mirroring the Python modules 1:1 (five builds, five runbook paths, no testability gain the table-driven unit tests do not already give).

### The orchestration loop lives in a repo-tracked skill

- Decision: `.claude/skills/ladder-run/SKILL.md`, invoked as `/ladder-run`, is the session-side driver. It is versioned in this repo alongside the harness.
- Rationale: the protocol `run_ladder.py --stage all` used to enforce has to live somewhere that enforces step order; README prose does not. A repo-tracked skill is versioned with the binary whose subcommands it calls, so the two cannot drift independently.
- Rejected: a runbook section in the README that the operator pastes into a session (protocol becomes unenforced prose); a millhouse skill (separate repo, and this harness is quarry-specific).

### One session per config

- Decision: 18 short Claude Code sessions total — one per each of the 15 configs, plus one per each of the 3 cold-cell repetitions. Each is launched by `ladderbench prepare-session --config-id <id>` and dispatches 3 run agents plus 3 scorer agents.
- Rationale: `settings.json` is one file per session, so a session hosting several configs can carry only one deny-list. The two-layer enforcement model below (agent-def allowlist as primary, session deny-list as backup) is only genuinely per-rung when the session is per-config. 18 manual launches is the honest cost of that model. Supervision also stays granular, which is the point of the whole dispatch swap.
- Rejected: per-ladder sessions with the deny-list reduced to `Task` only (drops the backup layer, and the second preflight probe would then be testing a layer that does not exist); per-ladder sessions with a per-rung deny-list written but unenforceable (a check that cannot fire reads as coverage that is not there).

### The session's working directory is a neutral scratch dir

- Decision: each session runs from a disposable scratch directory derived from `ladder.yaml`'s `session_dir_template` (e.g. `/tmp/ladder-session-a5-bundle`). The `ladderbench` binary and the results root are addressed by absolute path. The generated `.mcp.json`, `settings.json`, and agent definition live in that scratch dir.
- Rationale: it is the only cwd choice that keeps both `gate_blinding`'s repo-root check and `gate_worktree_neutralised` meaningful. Running from the quarry repo would make `repo_root` the subagents' own cwd, gutting the blinding check for the `none` arms. Running from the pinned task worktree would require writing `.claude/` into it, which `gate_worktree_neutralised` forbids and which `git clean -fdx` would delete after every run anyway.
- Rejected: cwd = the quarry repo; cwd = the pinned task worktree.

### Tool exposure is enforced by a generated agent definition, with the deny-list as backup

- Decision: each config gets a generated agent definition whose `tools:` frontmatter is an **allowlist** — `Read`, `Grep`, `Glob`, `Bash`, plus that config's `allowed` quarry tools under their `mcp__quarry__*` client-side names. The session's `settings.json` additionally carries the derived `permissions.deny` list (every canonical tool not in `allowed`, plus `Task`) as a second layer.
- Rationale: the Agent Tool has no per-call `--settings` equivalent, so per-rung restriction has to be a static subagent-type definition. An allowlist is structurally stronger than a deny-list: the `none` arm never sees the `mcp__quarry__*` namespace at all, rather than seeing it and being denied. The deny-list backup exists so a definition that fails to load does not silently promote a rung to the full bundle.
- Note: `Task` denial is now structural too — an allowlist that omits `Task` means a run agent cannot spawn its own subagents, which is what the old uniform `Task` deny existed to guarantee.
- Rejected: agent definitions alone (nothing catches a definition that failed to load); session-level deny only with one generic agent type (settings are not per-subagent).

### One pinned model id per role in `ladder.yaml`

- Decision: `ladder.yaml` stores exactly one pinned model id per role (`run_model: claude-sonnet-5`, `scorer.model: claude-opus-5`). Go maps that id to the agent-definition alias (`sonnet` / `opus`). `gate_model_pinned` continues to verify that the dispatched agent actually reported the pinned id, now read from the subagent transcript's assistant records rather than from a `system/init` event.
- Rationale: a single source of truth, and the id→alias mapping is verified empirically by the first run rather than sitting unchecked. Storing both representations gives two fields nothing can check agree; storing only the alias leaves the gate with nothing concrete to compare against.
- Rejected: storing id and alias side by side; storing only the alias.

### Metrics: what survives, changes, and dies under Agent dispatch

- Decision: `usage.json` is rebuilt around the subagent transcript. Surviving unchanged in definition: the four token classes summed independently across assistant records, `tool_uses`, `tool_uses_breakdown`, `quarry_tool_uses`, `bash_grep_count`, `grep_tool_count`, `grep_fallback_total`. Changed: `duration_ms` becomes last-minus-first record timestamp; `num_turns` becomes the count of `assistant` records; `denied_tool_attempts` becomes the count of `tool_result` blocks marked `is_error` whose text matches a permission-denial shape; `advertised_tools` is **renamed to `granted_tools`** and sourced from the run's generated agent definition rather than from the transcript. Dropped entirely: `cost_usd`, `wall_clock_ms`, `result_usage`, `result_subtype`, `result_is_error`, `session_id`. Added: `effort`, `agent_id`, `transcript_source`.
- Rationale: the subagent transcript has no `result` envelope and no `system/init` event, so `total_cost_usd`, `num_turns`, `permission_denials`, `subtype`, and the advertised-tool list have no source. A cost figure synthesised from a drifting price table is worse than no number — the field's whole point was that it came from the client. `wall_clock_ms` and `duration_ms` would now be the same measurement under two names, since there is no harness process wrapping the dispatch. The `advertised_tools` → `granted_tools` rename makes the provenance change explicit instead of silently swapping what a field means; the "extracted, never self-reported" rule still holds, because `granted_tools` comes from a file the harness itself generated, not from the agent.
- Consequence: `summarize`'s `METRICS` tuple loses `wall_clock_ms` and `cost_usd`; everything else in it is unchanged.
- Rejected: computing `cost_usd` from a pinned price table; keeping `cost_usd` as a permanent `null`; keeping `wall_clock_ms` as an orchestrator-measured value (a model session's timing is coarse and unreproducible).

### `model` is gated; `effort` is only recorded

- Decision: `gate_model_pinned` stays fatal. `effort` is recorded into `usage.json` from the assistant records but is not gated.
- Rationale: the model reaches the dispatch through an id→alias mapping this harness owns and could get wrong, so it needs empirical verification. Effort passes straight from the generated agent definition's frontmatter with no mapping in between; a mismatch there would be a Claude Code bug, not an operator or harness error.

### `max_turns` becomes a post-hoc gate

- Decision: `ladder.yaml`'s `max_turns` still applies, but as a fatal gate on the run's assistant-record count evaluated at `ingest`. Exceeding it produces the same `"truncated"` outcome, is never retried, and halts the matrix on first occurrence — identical semantics to today.
- Rationale: the Agent Tool has no `--max-turns` equivalent, so nothing bounds a run mid-flight. Live operator supervision is a real-time backstop, not a replacement for a recorded gate: a run that quietly ran long has to come out flagged, not merely "not killed".
- Rejected: dropping the ceiling and relying solely on the watching operator; instructing the agent in its preamble to stop after N turns (unenforced self-policing, and it changes the committed preamble text).

### Transcript correlation by unique description

- Decision: the orchestrator passes a unique `description` on every Agent call, of the form `ladder <config-id> rep <n> attempt <k>`. `ladderbench ingest` scans the session's `subagents/*.meta.json` for that exact description string and takes the sibling `.jsonl`. Zero matches or more than one match is a hard error, never a fallback.
- Rationale: deterministic, and the attempt index keeps retries from colliding. It does not bet on an agent-id field's format staying stable across dispatch paths, and a loud error beats a newest-mtime guess that silently picks the wrong file under any concurrent dispatch.
- Rejected: reading the agent id out of the completion notification; newest-mtime selection.

### Transcript custody

- Decision: `ingest` copies the located `agent-<id>.jsonl` into `<run_dir>/transcript.jsonl` and its `agent-<id>.meta.json` into `<run_dir>/transcript.meta.json`, and records the original absolute source path in `usage.json`'s `transcript_source` field.
- Rationale: the results tree stays self-contained and survives session-transcript pruning. `bench/loomyard-eval/ladder/results/**/raw/` is already gitignored, so the copies cost nothing tracked. The copied `.meta.json` is the evidence that the description-match picked the right file.
- Rejected: copying the `.jsonl` alone (throws away the correlation evidence); recording the source path without copying (the record dies when the session transcript is pruned).

### `gate_blinding` under the new topology

- Decision: keep three checks — an `mcp__quarry__` tool name anywhere in the transcript (fatal), any filesystem path into `repo_root` (fatal), and a bare "quarry" mention outside a `tool_result` payload (fatal, with a mention confined to a `tool_result` staying non-fatal as today). Drop the `/tmp/quarry-bench` literal check. Never treat the session scratch directory as a leak, since it is legitimately the subagent's own cwd.
- Rationale: `/tmp/quarry-bench` is a CLI-era artifact of the sibling suite; no such binary exists in this suite, so the check can never fire and reads as coverage that is not there. The repo-root check stays meaningful precisely because the session cwd is now a neutral scratch dir. Detection is kept even though the `none` arm's agent definition grants no quarry tools: the README's position is that blinding is enforced by detection rather than construction, and construction failing silently is the whole reason a second layer exists.
- Rejected: keeping the dead `/tmp/quarry-bench` check; dropping `gate_blinding` on the grounds that the allowlist makes exposure structurally impossible.

### Two preflight probes, not one

- Decision: the preflight stage runs two throwaway dispatches, each recorded into `probe.json`, and halts the matrix before any paid run if either fails.
  1. **Allowlist probe** — an agent definition granting `Read/Grep/Glob/Bash` but not `mcp__quarry__impact`, in a session where the quarry server *is* declared, asked to call `impact` and report verbatim what happened. Records `allowlist_blocks`.
  2. **Deny-list probe** — an agent definition that *does* grant `mcp__quarry__impact`, in a session whose `settings.json` denies it, asked the same question. Records `denylist_blocks`.
- Rationale: the layers are only independently established if each is probed with the other neutralised. A probe that exercises only the allowlist cannot distinguish "primary works, backup silently broken" from "primary broken, backup catching it" — the call fails identically either way. That defeats the point of having a backup: a layer that degrades unnoticed provides nothing on the day the primary also fails. One extra throwaway dispatch per matrix run is trivial next to a leaked tool call contaminating the exact confound this suite exists to rule out.
- Note: the probe's role has changed from #008's. It no longer establishes "`permissions.deny` blocks" as the single load-bearing premise; it establishes that both enforcement layers independently bound a subagent.
- Rejected: a single allowlist-only probe; no probe at all (an allowlist is a config file that can fail to load, not a structural guarantee).

### Scorer dispatch

- Decision: one Agent call per run. The scorer uses a generated agent definition pinned to the `opus` alias at `high` effort with **no tools at all**. The three-input blinded contract (redacted answer, `_meta`-stripped fasit, task text) is unchanged. `ladderbench redact` prints the scorer prompt; the session dispatches it; `ladderbench record-score` consumes the reply.
- Rationale: a scorer with zero tools cannot wander into the repo and un-blind itself, which is strictly stronger than the old `--strict-mcp-config` isolation. One call per run keeps the scorer from ever seeing two answers side by side.
- Rejected: one call per config scoring all 3 reps together (lets the scorer grade reps relative to each other, which the blinding forbids); keeping `claude -p` for scoring (contradicts the brief).

### The cold cell

- Decision: retained, as three dedicated short sessions run after all 15 config sessions. Each cold session builds its own fresh worktree from `cold_worktree_template` with `{n}` substituted, generates its own `.mcp.json` pointing at that worktree, clears the resolved state directory, asserts `gate_cold_before`, runs no warm-up, and removes the worktree afterwards. Disposition logic (`confirmed-cold` / `partial` / `not-run` / `no-daemon-signal`, with `not_run_causes`) ports unchanged.
- Rationale: the MCP server's `--target-dir` is fixed at session start, so a fresh worktree per repetition forces a fresh session per repetition. Dropping the cell would silently remove one of the three comparison types `summarize` already models — not acceptable for a port meant to reach parity.
- Rejected: dropping the cold cell; re-pointing one session's target dir between repetitions (not possible).

### Daemon warm-up via the Go MCP SDK

- Decision: `ladderbench warm` performs the stdio `initialize` + `tools/call` against a freshly spawned `quarry-mcp` using `github.com/modelcontextprotocol/go-sdk`, already a direct dependency. `WARM_UP_TOOL` stays `workspace_symbol`, and the post-condition stays "a `daemon.json` now exists at the resolved state directory".
- Rationale: the Python version hand-rolled JSON-RPC framing only because it was constrained to the standard library; that constraint does not exist in Go, and the repo already links this SDK for the server side. Warming via the `quarry` CLI would resolve the same state directory but exercise a different code path than the one under measurement, weakening the warm/cold comparison.
- Rejected: a hand-rolled Go JSON-RPC stdio client; warming via the `quarry` CLI.

### Single-flight dispatch, mechanically enforced

- Decision: the skill mandates exactly one Agent call in flight at a time. `ladderbench ingest` fails loud when it is asked to ingest a run whose predecessor in the config's repetition order has no `run.json`.
- Rationale: the README's design rationale forbids concurrent runs (CPU, shared daemon, and rate-limit contention all make wall-clock incomparable across configs). A rule that exists only as prose fails silently — corrupted wall-clock numbers with nothing flagging them.
- Rejected: stating the rule without enforcing it; allowing concurrency for scorer calls only (defensible in isolation, but it turns a simple invariant into a conditional one for a payoff not worth the enforcement complexity).

### `ladder.yaml` additions

- Decision: add `run_effort: medium` and `session_dir_template` (e.g. `/tmp/ladder-session-{config_id}`). `require_pins` extends to reject an unset `run_effort` alongside `run_model`, `max_turns`, `scorer.model`, and `scorer.effort`. `run_model` continues to ship `null` by design.
- Rationale: both are operator-tunable and reproducibility-relevant, so they belong in the single declarative source of truth rather than hardcoded in Go.
- Rejected: hardcoding either in Go; adding a `scorer.effort` companion (it already exists).

### Portability

- Decision: use `path/filepath` and `os.UserCacheDir` throughout so nothing is structurally Linux-only, but make no attempt to run the matrix on Windows and say so plainly in the README.
- Rationale: the brief's Windows motivation is about killing per-invocation interpreter startup cost on the user's development machine, which a Go binary fixes regardless of where the matrix runs. The matrix itself targets a pinned Linux checkout at Linux paths.
- Rejected: full Windows support (scope nobody asked for); loose `/`-joined literals (costs nothing to avoid, buys nothing by keeping).

### Python removal and documentation

- Decision: delete `bench/loomyard-eval/ladder/scripts/` and `bench/loomyard-eval/ladder/tests/` in this task's **final batch**, once the Go port has parity and its own tests pass. Move `tests/fixtures/` content to `internal/ladder/testdata/` as part of the porting batches. Rewrite `bench/loomyard-eval/ladder/README.md` in place as the self-contained protocol document, refresh `ladder.yaml`'s header comments, and remove the `bench/loomyard-eval/ladder/` clause from `CLAUDE.md`'s grandfather note in the same final batch.
- Rationale: keeping the Python as a live reference during the port beats porting blind against the README and git history. Deleting last closes out the grandfather note in the same commit that makes it untrue. A separate `docs/` protocol file would duplicate the skill body for no reader who is not already looking at one or the other; leaving the README stale would track prose describing a harness that no longer exists.
- Rejected: deleting the Python up front; keeping both trees; documenting only in the skill.

### Scope boundary: harness only

- Decision: this task delivers the harness, its tests, the skill, and the documentation. It does not run the 45-run matrix, does not run either preflight probe for real, and does not produce `conclusion.md`. Matrix execution is a separate follow-up task.
- Rationale: this matches #008's own precedent, where matrix execution was pulled out for the same reason — a multi-hour supervised run should not gate a code port's mergeability. The synthetic end-to-end test already covers the integration risk that spending real probe dollars would be buying.
- Rejected: harness plus probes only; harness plus the full matrix and conclusion.

## Technical context

### Repository shape

Go module `github.com/Knatte18/quarry`, Go 1.26.
Existing binaries are `cmd/quarry` (a thin wrapper over `internal/cli.RunCLI`) and `cmd/quarry-mcp`.
The CLI is cobra-based; `internal/cli/cli.go` is the single place the command tree is defined, and its package doc is the model for how this repo documents a command surface.
Direct dependencies already available and relevant here: `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, and `github.com/modelcontextprotocol/go-sdk`.
There is no `_codeguide/` and no `CONSTRAINTS.md` in this repo.

`.gitignore` already carries `bench/loomyard-eval/ladder/results/**/raw/`, so per-run artifacts (transcripts, answers, usage, scores, run markers) are untracked while `probe.json`, `cold_cell.json`, `summary.json`, and `conclusion.md` stay tracked.

### What is being ported, module by module

The Python source of truth is `bench/loomyard-eval/ladder/scripts/`.
Read each file before porting its Go counterpart; the docstrings carry rationale that the ported Go doc comments must preserve.

- `ladder_config.py` (442 lines) — `QUARRY_TOOLS` validation constant, `MCP_PREFIX`, `DAEMON_BACKED_TOOLS` (every canonical tool except `toc_dir`/`toc_file`), the `Ladder`/`LadderConfig`/`TaskEntry`/`ScorerConfig` shapes, `load_ladder` with its nine validation rules, `config_by_id`, `control_for`, `warm_counterpart_for`, `require_pins`, `deny_list_for`, `settings_document_for`, `write_settings`, the three preamble constants plus `_TOOL_DESCRIPTIONS` and `preamble_for`, and `extract_fenced_json` with its `first`/`last` selector.
- `extract_usage.py` (208 lines) — transcript reading, `iter_tool_use_blocks` / `iter_tool_uses`, `init_event`, `result_event`, the `_BASH_GREP_RE` leading-command-word pattern, and `extract_usage`. `init_event` and `result_event` have no counterpart in the new transcript format and are dropped; everything else ports.
- `gates.py` (624 lines) — `GateFinding` / `GateReport`, `gate_denied_tools_not_used`, `gate_no_target_override`, `gate_model_pinned` with its `[1m]`-suffix normalisation, `gate_blinding` with its `tool_result`-redaction pass, `workspace_key`, `user_cache_dir`, `resolve_state_dir`, `daemon_state_file`, `daemon_alive`, `gate_cold_before`, `used_daemon_backed_tool`, `gate_cold_after`, `clear_state_dir`, `wait_for_daemon_exit`, `gate_worktree_neutralised`, `observe_worktree_dirtied`, `run_dir`, `is_complete`, `invalidate`, `write_run_json`, `run_gates`, and `gate_run_complete_artifacts`.
- `score_run.py` (358 lines) — the redaction alternation built most-specific-first, `redact_text`, `redact_answer`, `write_redacted`, `EXPLORATION_RULE`, `IMPACT_RULE`, `strip_fasit_meta`, and `build_scorer_prompt`. `run_scorer_client` is replaced by session dispatch; `score_run`'s injected-`runner` seam becomes the `redact` / `record-score` subcommand pair.
- `summarize.py` (469 lines) — `METRICS`, `GREP_METRICS`, `load_runs` with its `tokens.<class>` flattening, `_median`, `summarise_cell`, `ranges_disjoint`, `compare_rung_to_control`, `compare_rungs`, `compare_warm_cold`, `build_summary`, `write_summary`, and the non-zero-exit-on-incomplete behaviour.
- `run_ladder.py` (1133 lines) — worktree lifecycle (`neutralise_worktree`, `build_worktree`, `restore_worktree`, `remove_worktree`, `ensure_task_worktrees`), `build_server`, `mcp_config_document`, `write_run_inputs`, `warm_daemon`, `run_env`, `task_text_for` with its load-bearing section boundary, `schema_for`, `plan_runs` / `main_runs` / `cold_runs` / `pending_runs`, `MAX_ATTEMPTS`, the run-outcome shape, and the cold-cell disposition logic. The `claude -p` argv assembly (`build_argv`, `launch_run`) and the subprocess drivers (`run_matrix`, `run_cold_cell`'s dispatch loop, `run_stage`, `run_probe`'s dispatch) are replaced by the skill plus the subcommand surface.

### The subagent transcript format

This is the single most important new fact for the port, and it was verified against real transcripts on this machine.

Location: `~/.claude/projects/<slugified-session-cwd>/<session-id>/subagents/agent-<id>.jsonl`, with a sibling `agent-<id>.meta.json`.
The project directory name is the session's cwd with path separators replaced by `-` (e.g. `/tmp/ladder-session-a5-bundle` → `-tmp-ladder-session-a5-bundle`).
Since sessions are per-config with distinct scratch dirs, each config's search space is one directory.
`ingest` should glob `<projects-root>/<slug>/*/subagents/*.meta.json` rather than requiring the session id as an argument.

`meta.json` shape (verified):

```json
{"agentType":"general-purpose","description":"Benchmark task 04 — Agent A (with quarry)","toolUseId":"toolu_01SFoW21JgioRXn3iXii6oTK","spawnDepth":1}
```

`agent-<id>.jsonl` is newline-delimited JSON, one record per line.
Records carry `parentUuid`, `isSidechain: true`, `agentId`, `uuid`, `timestamp` (RFC 3339 with milliseconds), `type`, and `message`.
Observed `type` values across a real run: `user`, `assistant`, `attachment`, `deferred_tools_delta`, `skill_listing`.
The first record is the `user` record carrying the dispatched prompt as a plain string in `message.content`.

Assistant records carry `message.model` (e.g. `claude-sonnet-5`), a top-level `effort` field (e.g. `medium`), a `message.content` array of `text` / `thinking` / `tool_use` blocks, and `message.usage` with `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens` — the same four classes `extract_usage` already sums.
User records carry `tool_result` blocks under `message.content` keyed by `tool_use_id`, plus a sibling top-level `toolUseResult` field holding the raw result payload.

Critically: there is **no** `system`/`init` record and **no** terminal `result` record.
The transcript simply ends at the last assistant message.
This is what forces the metric changes decided above.

The answer is still the last fenced ```json block in the final assistant record's `text` content — `extract_fenced_json(text, "last")` ports unchanged.

### Session launch and agent-definition discovery

`prepare-session` writes into the scratch dir: `.mcp.json` (a single server named `quarry`, command = the built `quarry-mcp` absolute path, args = `--target-dir <task worktree>`), `.claude/settings.json` (the derived `permissions.allow` / `permissions.deny`), and `.claude/agents/<config-id>.md` (the run agent definition) plus `.claude/agents/<config-id>-scorer.md` (the zero-tool scorer definition).
It then prints the exact `claude` launch command for the operator to run.

**Known implementation risk:** Claude Code's `--setting-sources` flag and its agent-definition discovery interact, and the exact flag combination that isolates settings while still loading project-local agent definitions must be established empirically during implementation.
If project-local agent discovery turns out to be suppressed by the isolation flags, the fallback is to write the definitions into `~/.claude/agents/` under a `ladder-<config-id>` namespace and have `prepare-session` clean them up.
This must be settled by an actual `claude` launch during the batch that builds `prepare-session`, not assumed.

### `ladder.yaml` as it stands

15 configs across two ladders: Ladder A (`a0-none` … `a5-bundle`, task `01-reed-geometry-exploration`, exploration schema) and Ladder B (`b0-none` … `b7-bundle`, task `04-shedadapters-shuttle-impact`, impact schema), plus the single cold config `a5-bundle-cold` with `cold: true` and `warm_counterpart: a5-bundle`.
`reps: 3`, `max_turns: 60`, `run_model: null` (operator sets it), `scorer: {model: claude-opus-5, effort: high}`.
Both tasks pin SHA `975578cda8d6f3a81580bd4e73725e060211b766` off `source_repo: /home/knatte/Code/loomyard/wts/loomyard`, with worktrees at `/tmp/loomyard-eval-01` and `/tmp/loomyard-eval-04`, and `cold_worktree_template: /tmp/loomyard-eval-01-cold-{n}`.
The file is the single declarative source of truth and stays so.

### Results tree layout

`<results_root>/` holds tracked `probe.json`, `cold_cell.json`, `summary.json`, and (later, in the follow-up task) `conclusion.md`.
`<results_root>/raw/<config-id>/<n>/` holds the per-run artifacts, gitignored: `transcript.jsonl`, `transcript.meta.json`, `answer.json`, `answer.redacted.json`, `usage.json`, `score.json`, `run.json`, plus the session's generated `settings.json` / `mcp.json` copies if the port keeps recording them.
`run.json` is written last and is the sole definition of a complete run (`is_complete`), which is what makes resume safe.

### Subcommand surface

Ten subcommands, split at each session boundary because only a live session can invoke the Agent Tool:

- `prepare-session` — ensure task worktrees at their pins, build `quarry-mcp`, write the scratch dir's `.mcp.json` / `settings.json` / agent definitions, print the launch command.
- `next-run` — resume-aware; print the next pending `(config, rep)` for this config, its full prompt, and its agent-definition name.
- `warm` — the daemon warm-up call plus its `daemon.json` post-condition. Skipped for the cold config.
- `ingest` — locate the transcript by description, copy it and its meta in, extract `usage.json`, parse `answer.json`, run the gates, print the outcome (`complete-pending-scoring` / `truncated` / `failed`).
- `redact` — write `answer.redacted.json`, print the scorer prompt.
- `record-score` — validate the scorer reply, write `score.json`, run `gate_run_complete_artifacts`, write `run.json`.
- `restore-worktree` — the unconditional post-run `reset --hard` + `clean -fdx` + re-neutralise.
- `probe-record` — consume a probe dispatch's transcript and write/extend `probe.json`; halt on a failing layer.
- `cold-cell` — finalise the cold cell's disposition record.
- `summarize` — build and write `summary.json`, non-zero exit when any cell is incomplete.

### Gotchas carried over from the Python

- `task_text_for`'s section boundary is load-bearing, not tidiness. Task 01's next section is its fasit leads and task 04's names the real callers and the `burler.go:373` decoy outright; an extractor that over-reads pastes the answer key into every prompt.
- `deny_list_for` must always derive from `ladder.quarry_tools` through the `mcp__quarry__` prefixing helper, never from a literal, so a mutated already-loaded ladder still produces a correct deny-list. The Python suite has an explicit drift-guard test for this.
- `impact` is deliberately excluded from redaction's bare-name pass because it is an ordinary English word every Ladder-B answer's prose uses; only its `mcp__quarry__impact` and CLI-invocation forms are redacted. The alternation must stay ordered most-specific-first.
- `bash_grep_count` and `grep_tool_count` are never merged; `grep_fallback_total` is their sum and never substituted for either.
- `gate_cold_before` keys on daemon *liveness* (pid alive), not on `daemon.json`'s existence — neither the file nor the state directory is removed when a daemon exits.
- `observe_worktree_dirtied` must run before `restore-worktree`, because the restore is precisely what erases the evidence.
- `ranges_disjoint` treats ranges that touch at a single shared value as **not** disjoint.
- Grep metrics are excluded from every rung-vs-control comparison (the control's preamble differs in steering as well as tools) but remain eligible for rung-vs-rung.

## Testing

The Python suite's own division holds: pure units are unit-tested, and the dispatch layer is exercised by actually running the matrix, never by a mock.
Under the new design the "dispatch layer" is the skill plus the session, which no Go test can drive — so the Go tests cover strictly more of the binary than the pytest suite covered of the Python, because argv assembly and subprocess launching are gone.

Package layout for tests: `bench/loomyard-eval/ladder/internal/ladder/*_test.go`, table-driven, with fixtures under `bench/loomyard-eval/ladder/internal/ladder/testdata/`.

**TDD candidates — write the test first for each of these:**

- Ladder loading and validation. Every one of `load_ladder`'s nine rejection rules gets a case: duplicate id, ladder outside `a`/`b`, unknown task key, unknown allowed tool, `quarry_tools` drift from the canonical seven, zero controls on a ladder, more than one control on a ladder, more than one cold config, `warm_counterpart` on a non-cold config, cold config without a `warm_counterpart`, and a `warm_counterpart` naming an unknown id or a cold config. Plus `require_pins` rejecting each unset pin including the new `run_effort`.
- Deny-list and settings derivation, including the drift-guard case where a loaded ladder's `quarry_tools` is mutated in memory and the deny-list must track it.
- Agent-definition generation: the allowlist for a `none` config contains no `mcp__quarry__*` name and no `Task`; the allowlist for a rung contains exactly its `allowed` tools prefixed, plus the four base tools; the scorer definition grants nothing.
- Preamble generation for both shapes (`none` control reproducing the committed Agent B text with `<TARGET_DIR>` substituted; a rung's generated MCP-shaped body listing exactly its allowed tools in canonical order).
- Fenced-JSON extraction, `first` and `last` selectors, and the no-block case.
- Usage extraction from a fixture subagent transcript: independent summation of all four token classes, `num_turns` from assistant-record count, `duration_ms` from first/last timestamps, `tool_uses_breakdown`, `quarry_tool_uses`, the separate `bash_grep_count` / `grep_tool_count` counters with the leading-command-word distinction (a `grep` inside a path must not count), `denied_tool_attempts` from denial-shaped errored tool results, and `granted_tools` read from a generated definition.
- Every gate, each with a passing and a failing fixture: denied tools used, target/buildTags override, model pinning including the `[1m]` suffix normalisation, blinding in all three of its outcomes (fatal `mcp__quarry__`, fatal repo-root path, fatal bare-quarry outside a `tool_result`, non-fatal bare-quarry confined to a `tool_result`), the new post-hoc `max_turns` ceiling, worktree neutralisation, cold-before and all three cold-after outcomes, and the complete-artifacts gate.
- Redaction: every alternation branch, the `impact` exemption, the case-insensitivity, the adjacent-token collapse, and structural fields left untouched.
- Scorer-prompt assembly: the three inputs are present, `_meta` is stripped from the fasit, and the config id, ladder, allowed set, and transcript are all absent.
- Summarization: `_median` at odd and even lengths, `ranges_disjoint` including the touching-at-one-value case, the grep-metric exclusion from rung-vs-control only, the same-ladder guard on `compare_rungs`, warm-vs-cold suppression on `not-run` / `partial` / all-`cold_no_daemon_backed_call`, the incomplete-cell list, and the non-zero exit.
- Run-state bookkeeping: `is_complete` false without `run.json` and false when `state != "complete"`, `invalidate`'s lowest-unused-index rename, and resume filtering that skips completed pairs.
- Transcript correlation: exactly one description match succeeds, zero matches errors, two matches errors.
- Single-flight enforcement: ingesting rep 2 while rep 1 has no `run.json` errors.

**The synthetic end-to-end test** drives `ingest` → gates → `redact` → `record-score` (with a canned scorer reply, no dispatch) → `summarize` over a hand-written subagent transcript for a small synthetic ladder, and asserts the final `summary.json`.
Its purpose is catching field-name drift across the four stages' handoffs, which per-unit tests structurally cannot.

Existing fixtures to move from `bench/loomyard-eval/ladder/tests/fixtures/`: `none-target-origin-mention.jsonl`, `errored-tool-result.jsonl`, `targetdir-override.jsonl`, `cold-native-fallback.jsonl`, `denied-attempt.jsonl`, `bundle-mixed-tools.jsonl`, `zero-tool-calls.jsonl`.
Each must be **re-shaped** into the subagent transcript format described above, not copied verbatim — they are currently `claude -p` stream-json.

Verification per the `golang:golang-build` conventions: `go build ./...`, `go vet ./...`, `go test ./...` from the repo root, with `CGO_ENABLED=1` where the tree-sitter-linked packages are involved.

## Q&A log

- **Q:** Where does the Go code live, and how many binaries? **A:** One binary at `bench/loomyard-eval/ladder/cmd/ladderbench` with cobra subcommands, plus `internal/ladder/`. Bench tooling stays out of quarry's own `cmd/` namespace.
- **Q:** What drives the session-side orchestration loop? **A:** A repo-tracked skill at `.claude/skills/ladder-run/SKILL.md`. Millhouse was rejected outright as unrelated to this harness.
- **Q:** What is the orchestrating session's working directory? **A:** A disposable neutral scratch dir — the only choice keeping both `gate_blinding`'s repo-root check and `gate_worktree_neutralised` meaningful.
- **Q:** How is a rung's tool exposure enforced? **A:** Generated per-config agent definitions with a `tools:` allowlist, plus the derived deny-list in session settings as a backup. The Agent Tool has no per-call `--settings` equivalent, so per-rung restriction has to be static subagent-type definitions.
- **Q:** What happens to the Python tree? **A:** Deleted in the final batch, once Go has parity and its own tests pass. Keeps a working reference during the port rather than porting blind.
- **Q:** `cost_usd` has no source under Agent dispatch. **A:** Dropped entirely. A synthesised number from a drifting price table is worse than no number.
- **Q:** How are `duration_ms` / `wall_clock_ms` / `num_turns` derived? **A:** `duration_ms` from first/last timestamp, `num_turns` from assistant-record count, `wall_clock_ms` dropped as a duplicate of `duration_ms`.
- **Q:** `--max-turns` has no Agent-tool equivalent. **A:** Becomes a post-hoc fatal gate with identical truncate/halt semantics. Live supervision is a backstop, not a replacement for a recorded gate — a run that quietly ran long must come out flagged, not merely "not killed".
- **Q:** `denied_tool_attempts` and `advertised_tools` sources? **A:** Denials from `is_error` tool-result blocks matching the permission-denial shape; `advertised_tools` renamed to `granted_tools` and sourced from the generated agent definition. The rename makes the provenance change explicit instead of silently swapping what the field means.
- **Q:** How does the Go tool find a run's transcript? **A:** Unique per-call `description` matched against `subagents/*.meta.json`, hard error on zero or multiple matches. Does not bet on an id field's format, and beats a newest-mtime guess.
- **Q:** Scoring dispatch shape? **A:** One Agent call per run, zero-tool scorer definition pinned opus/high. Batching reps would let the scorer grade them relative to each other, which blinding forbids.
- **Q:** One preflight probe or two? **A:** Two. A single allowlist probe cannot tell "primary works, backup silently broken" from "primary broken, backup catching it" — both look identical. A backup layer that degrades unnoticed provides nothing on the day the primary fails.
- **Q:** How does the cold cell work under the session model? **A:** Three dedicated sessions, one per repetition, after all config sessions. Dropping it would silently remove a comparison type `summarize` already models.
- **Q:** Warm-up implementation? **A:** `go-sdk`, already a direct dependency. The Python hand-rolled JSON-RPC only because it was stdlib-constrained; warming via the CLI would exercise a different code path than the one being measured.
- **Q:** Go test coverage? **A:** Per-unit table-driven tests plus a synthetic end-to-end test. Per-unit-only cannot catch field-name drift across the extract→gate→score→summarize handoffs.
- **Q:** `ladderbench` subcommand surface? **A:** Ten subcommands split at each session boundary. Folding scoring into `ingest` is a structural non-starter: only a live session can invoke the Agent Tool, never a subprocess.
- **Q:** Model pinning between `ladder.yaml` and the agent definitions? **A:** One pinned id per role, Go maps id→alias, `gate_model_pinned` verifies the dispatched agent reported it. A wrong mapping surfaces empirically on the first run.
- **Q:** Transcript custody? **A:** Copy both the `.jsonl` and its `.meta.json` into the run directory and record the source path. The meta is what proves the description-match picked the right file.
- **Q:** `gate_blinding` under the new topology? **A:** Keep the three real checks, drop the dead `/tmp/quarry-bench` literal, never treat the scratch dir as a leak. A check that can never fire reads as coverage that is not there.
- **Q:** Documentation scope? **A:** README rewritten in place, `ladder.yaml` header refreshed, `CLAUDE.md`'s grandfather clause dropped in the final batch. A separate `docs/` file would duplicate the skill body.
- **Q:** Session granularity, given that `settings.json` is per session? **A:** One session per config — 18 total. It is the only shape where both enforcement layers are genuinely per-rung, and 18 manual launches is the honest cost of the enforcement model already committed to.
- **Q:** Does this task run the matrix? **A:** No — harness only. Matches #008's precedent; a multi-hour supervised run should not gate a code port's mergeability, and the synthetic end-to-end test already covers the integration risk.
- **Q:** Sequential dispatch? **A:** Mechanically enforced single-flight, with `ingest` failing loud on an out-of-order transcript. A prose-only rule fails silently and corrupts wall-clock numbers with nothing flagging it.
- **Q:** New `ladder.yaml` fields? **A:** `run_effort` and `session_dir_template`, with `require_pins` extended to cover `run_effort`. Both are operator-tunable and reproducibility-relevant, so they belong in the single source of truth.
- **Q:** OS portability? **A:** Portable path handling, no Windows execution support, stated plainly. The Windows motivation was per-invocation interpreter startup on the dev machine, which a Go binary fixes regardless of where the matrix runs.
