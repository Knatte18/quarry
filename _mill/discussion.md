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

- A single new Go binary `bench/loomyard-eval/ladder/cmd/ladderbench` (cobra, eleven subcommands) plus its logic package `bench/loomyard-eval/ladder/internal/ladder/`.
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
- Delivery: the skill is tracked in this repo at `.claude/skills/ladder-run/SKILL.md` (the repo has no `.claude/` today; this task creates it), but every session launches from a `/tmp/ladder-session-*` scratch dir, which is a different project scope — so `prepare-session` **installs** it to `~/.claude/skills/ladder-run/SKILL.md` (user scope) for every session type uniformly. It is deliberately **not** copied into the scratch dir: the skill body names quarry throughout, and a `none` session's scratch dir is the blinded agent's own cwd (see the tool-exposure decision's scratch-dir rule).
- Rejected: a runbook section in the README that the operator pastes into a session (protocol becomes unenforced prose); a millhouse skill (separate repo, and this harness is quarry-specific); relying on project-scope discovery from the repo (the session's cwd is not the repo); copying the skill into the scratch dir (leaks into every blinded arm's cwd).

### One session per config

- Decision: **17** short Claude Code sessions total — one per each of the **14 non-cold** configs, plus one per each of the **3 cold-cell repetitions** of the single `a5-bundle-cold` config. Each is launched by `ladderbench prepare-session --config-id <id>` (cold sessions additionally take `--rep <n>`).
- Arithmetic, matching `ladder.yaml` and `plan_runs`: 15 configs total = 14 non-cold + 1 cold. A warm session runs that config's 3 repetitions, so it dispatches 3 run agents and 3 scorer agents, interleaved per the session loop below — 14 × 3 = 42 main-matrix runs. A cold session runs exactly **one** repetition, so it dispatches 1 run agent and 1 scorer agent — 3 × 1 = 3 cold runs. 42 + 3 = 45 runs.
- Rationale: `settings.json` is one file per session, so a session hosting several configs can carry only one deny-list. The two-layer enforcement model below (agent-def allowlist as primary, session deny-list as backup) is only genuinely per-rung when the session is per-config. 17 manual launches is the honest cost of that model. Supervision also stays granular, which is the point of the whole dispatch swap.
- Rejected: per-ladder sessions with the deny-list reduced to `Task` only (drops the backup layer, and the second preflight probe would then be testing a layer that does not exist); per-ladder sessions with a per-rung deny-list written but unenforceable (a check that cannot fire reads as coverage that is not there).

### The session's working directory is a neutral scratch dir

- Decision: each session runs from a disposable scratch directory derived from `ladder.yaml`'s `session_dir_template` (e.g. `/tmp/ladder-session-a5-bundle`). The `ladderbench` binary and the results root are addressed by absolute path. The generated `.mcp.json`, `settings.json`, and agent definition live in that scratch dir.
- Rationale: it is the only cwd choice that keeps both `gate_blinding`'s repo-root check and `gate_worktree_neutralised` meaningful. Running from the quarry repo would make `repo_root` the subagents' own cwd, gutting the blinding check for the `none` arms. Running from the pinned task worktree would require writing `.claude/` into it, which `gate_worktree_neutralised` forbids and which `git clean -fdx` would delete after every run anyway.
- Rejected: cwd = the quarry repo; cwd = the pinned task worktree.

### Tool exposure is enforced by a generated agent definition, with the deny-list as backup

- Decision: each config gets a generated agent definition whose `tools:` frontmatter is an **allowlist** — `Read`, `Grep`, `Glob`, `Bash`, plus that config's `allowed` quarry tools under their `mcp__quarry__*` client-side names. The session's `settings.json` additionally carries the derived `permissions.deny` list (every canonical tool not in `allowed`, plus `Task`) as a second layer.
- Rationale: the Agent Tool has no per-call `--settings` equivalent, so per-rung restriction has to be a static subagent-type definition. An allowlist is structurally stronger than a deny-list: the `none` arm never sees the `mcp__quarry__*` namespace at all, rather than seeing it and being denied. The deny-list backup exists so a definition that fails to load does not silently promote a rung to the full bundle.
- Note: `Task` denial is now structural too — an allowlist that omits `Task` means a run agent cannot spawn its own subagents, which is what the old uniform `Task` deny existed to guarantee.
- **A `none` control's session gets no `.mcp.json` at all.** `write_run_inputs`'s existing rule ports verbatim to the session level: `prepare-session` writes `.mcp.json` only when the config's `allowed` set is non-empty, and a `none` session is launched with no `--mcp-config` flag whatsoever. The quarry server is never declared to a blinded arm, because a declared server named `quarry` exposing an `mcp__quarry__*` namespace is itself the structural leak the blinding forbids (`ladder/README.md` "Enforcement"). This is what makes the claim "the `none` arm never sees the `mcp__quarry__*` namespace at all" literally true rather than merely allowlist-mediated: the namespace does not exist in that session.
- **What a `none` session's scratch directory may contain.** The blinded run agent has `Read`/`Grep`/`Glob`/`Bash` and its cwd *is* the scratch dir, so anything written there is one `cat` away from the transcript — and `gate_blinding` makes any `mcp__quarry__` substring anywhere in the transcript fatal. A `none` session's scratch dir therefore contains only: the agent definition (whose allowlist is `Read`/`Grep`/`Glob`/`Bash`, naming nothing), a `settings.json` whose `permissions.deny` is **`["Task"]` and nothing else**, and no `.mcp.json`. The `ladder-run` skill is installed at **`~/.claude/skills/ladder-run/`** (user scope) for every session type uniformly, never copied into a scratch dir, because its body names quarry throughout.
- This reverses the "denies names nothing declares, costs nothing" reading: for a `none` config the deny-list backup guards nothing (no server is declared and no quarry tool is in the allowlist) while leaking everything, so it is omitted. For every rung, `settings.json` carries that config's full derived deny-list plus `Task`, exactly as `settings_document_for` derives it — those sessions are not blinded, so there is nothing to leak.
- Residual limit, stated plainly: `~/.claude/skills/ladder-run/` is still reachable by `Bash` from any cwd. This is a hygiene measure that removes the leak from the agent's own working directory, not a structural guarantee — the same class of limit `ladder/README.md` already acknowledges for the `none` arms' blinding generally. `gate_blinding` remains the detector.
- Rejected: agent definitions alone (nothing catches a definition that failed to load); session-level deny only with one generic agent type (settings are not per-subagent).

### One pinned model id per role in `ladder.yaml`

- Decision: `ladder.yaml` stores exactly one pinned model id per role. `scorer.model` ships committed as `claude-opus-5`; `run_model` continues to **ship `null`** and is set by the operator before the matrix starts — `claude-sonnet-5` is the value the user intends to supply, an example here, never a value this task commits to the file. Go maps whichever id is present to the agent-definition alias (`sonnet` / `opus`). `gate_model_pinned` continues to verify that the dispatched agent actually reported the pinned id, now read from the subagent transcript's assistant records rather than from a `system/init` event.
- Rationale: a single source of truth, and the id→alias mapping is verified empirically by the first run rather than sitting unchecked. Storing both representations gives two fields nothing can check agree; storing only the alias leaves the gate with nothing concrete to compare against.
- Rejected: storing id and alias side by side; storing only the alias.

### Metrics: what survives, changes, and dies under Agent dispatch

- Decision: `usage.json` is rebuilt around the subagent transcript. The partition below is exhaustive over every field `extract_usage` writes today.
  - **Survives unchanged in definition:** the four token classes summed independently across assistant records; `tool_uses`; `tool_uses_breakdown`; `quarry_tool_uses`; `bash_grep_count`; `grep_tool_count`; `grep_fallback_total`; `transcript` (still the path to the run's transcript, now the copy inside the run directory); `model` (same meaning, source moves from the `system/init` event to the assistant records' `message.model`).
  - **Changed:** `duration_ms` becomes last-minus-first record timestamp; `num_turns` becomes the count of `assistant` records; `denied_tool_attempts` becomes the count of `tool_result` blocks marked `is_error` whose text matches a permission-denial shape (**provisional — see below**); `advertised_tools` is **renamed to `granted_tools`** and sourced from the run's generated agent definition rather than from the transcript.
  - **Dropped entirely:** `cost_usd`, `wall_clock_ms`, `result_usage`, `result_subtype`, `result_is_error`, `session_id`.
  - **Added:** `effort`, `agent_id`, `transcript_source` (the original `~/.claude/projects/.../agent-<id>.jsonl` path the run-directory `transcript` was copied from).
- Rationale: the subagent transcript has no `result` envelope and no `system/init` event, so `total_cost_usd`, `num_turns`, `permission_denials`, `subtype`, and the advertised-tool list have no source. A cost figure synthesised from a drifting price table is worse than no number — the field's whole point was that it came from the client. `wall_clock_ms` and `duration_ms` would now be the same measurement under two names, since there is no harness process wrapping the dispatch. The `advertised_tools` → `granted_tools` rename makes the provenance change explicit instead of silently swapping what a field means; the "extracted, never self-reported" rule still holds, because `granted_tools` comes from a file the harness itself generated, not from the agent.
- Consequence: `summarize`'s `METRICS` tuple loses `wall_clock_ms` and `cost_usd`; everything else in it is unchanged.
- Rejected: computing `cost_usd` from a pinned price table; keeping `cost_usd` as a permanent `null`; keeping `wall_clock_ms` as an orchestrator-measured value (a model session's timing is coarse and unreproducible).

### `denied_tool_attempts` ships provisional until a real denial record is observed

- Decision: the field replaces the structured `permission_denials` array with a pattern match over errored `tool_result` text, held in a single named constant in `internal/ladder/`. Because this task runs no dispatch of any kind, **no real denial record has been observed**, and the pattern is unvalidated against reality. It is therefore marked provisional: `usage.json` carries `denied_tool_attempts_provisional: true`, and `summarize` propagates that marker onto the metric's stats record so no reader mistakes an unvalidated count for a measured one.
- Validation path: the follow-up matrix task's **deny-list probe** is by construction the one dispatch that produces a genuine denial record. `probe-record` captures the offending `tool_result`'s verbatim text into `probe.json` as `denial_shape_observed`, and the follow-up task's first job is to check the constant against it and clear the provisional marker. Until then the field is recorded but not to be relied on.
- Rationale: the honest options were to ship an unvalidated pattern silently, to drop the field, or to ship it flagged. Dropping it loses the only signal that a rung tried to reach past its allowlist — the exact thing the two-layer enforcement model exists to detect. Shipping it silently would be the dead-check problem inverted: a number that reads as measured but was never checked against the shape it claims to match.
- Rejected: dropping `denied_tool_attempts`; shipping the pattern unflagged; quoting a denial shape from memory (nothing in the transcripts read during exploration contained one, and a fabricated shape is worse than an acknowledged gap).

### Environment scrubbing under Agent dispatch

- Decision: `run_env`'s scrub survives in three places, because the harness no longer owns the process that spawns `quarry-mcp`.
  1. The generated `.mcp.json`'s server entry carries an explicit `"env"` block setting `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` to the empty string, so the server Claude Code spawns sees them as unset. `QUARRY_CONFIG` is deliberately **not** touched — it selects the `servers.yaml` overlay naming the language-server command, and clearing it on a machine that needs an overlay would stop the server starting at all.
  2. `prepare-session` hard-fails when either variable is set non-empty in the operator's own shell, naming the variable — a belt-and-braces precondition, since whether an MCP `env` block replaces or augments the inherited environment must be established empirically (see the implementation risk in Technical context).
  3. Every Go gate that calls `resolve_state_dir` resolves against a reconstructed environment — `os.Environ()` with both keys forced to the empty string — exactly mirroring what `run_env` produced. `resolve_state_dir`'s hard error is on a **non-empty value**, not on key presence, matching the Python's `env.get()` truthiness check and quarry's own treatment of `""` as unset in `internal/cli/paths.go`. A key present with an empty value passes.
- Rationale: both variables take precedence over `workspaceKey` outright, and a non-empty tag set appends a `tags-<hex>` segment at every tier. A cold cell whose worktree-keyed state directory can be silently overridden by either variable cannot support a "this daemon is fresh because this path never ran before" claim — the cold cell's entire argument rests on the scrub. Losing the wrapping harness process is exactly why it now needs an explicit application point rather than an implicit one.
- Rejected: relying on the `.mcp.json` `env` block alone (its inherit-versus-replace semantics are unverified); relying on the `prepare-session` precondition alone (it checks the operator's shell at launch, not the environment the server actually receives); dropping the scrub (would invalidate the cold cell).

### The per-repetition session loop, and who owns retries

- Decision: a session's loop is **fully serialized per repetition**, not batched into a run phase followed by a scoring phase. For each repetition `n` of the session's config:
  `warm` (skipped for cold) → dispatch the run agent → `ingest` → `redact` → dispatch the scorer agent → `record-score` → `restore-worktree` → next repetition.
- Rationale: `run.json` is written last, by `record-score`. A "3 run agents, then 3 scorer agents" order would make rep 2's `ingest` fail its single-flight check while rep 1 is still unscored. Serializing per repetition is the only order that satisfies both the single-flight predicate and the sequential-dispatch rationale, and it is also what keeps the operator's supervision unit equal to one run.
- Retry ownership: the **skill** owns the retry loop; the Go side owns the ceiling and the attempt index. An eleventh subcommand, `invalidate`, performs `gates.invalidate`'s rename-aside on the run directory and returns the next attempt index; it errors when the ceiling (`MAX_ATTEMPTS = 3`) is already reached, which is the matrix halt. `next-run` reports the current attempt index for a pending repetition by counting existing `<n>.invalid-<k>` siblings, so the index the correlation `description` embeds has a single derivation site rather than being tracked in the session's head.
- Truncation keeps its distinct handling: a `"truncated"` outcome from `ingest` is never retried and halts immediately, since `max_turns` is a matrix-wide constant a second attempt would hit identically.
- Rejected: batching run dispatches ahead of scoring (breaks the single-flight predicate); leaving attempt bookkeeping to the session's own memory (no single derivation site, and a resumed session would lose it); folding `invalidate` into `ingest` (`ingest` must be able to report a failure without destroying the evidence).

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
- Both probe sessions are materialised by `prepare-session --probe allowlist|denylist`, which generates that probe's bespoke agent definition and `settings.json` into its own scratch dir. Generating the probe inputs is in scope for this task; **dispatching** them is not — that is the follow-up matrix task's first step.
- Rationale: the layers are only independently established if each is probed with the other neutralised. A probe that exercises only the allowlist cannot distinguish "primary works, backup silently broken" from "primary broken, backup catching it" — the call fails identically either way. That defeats the point of having a backup: a layer that degrades unnoticed provides nothing on the day the primary also fails. One extra throwaway dispatch per matrix run is trivial next to a leaked tool call contaminating the exact confound this suite exists to rule out.
- The deny-list probe additionally captures the verbatim text of the errored `tool_result` it provokes into `probe.json` as `denial_shape_observed`, which is what clears `denied_tool_attempts`'s provisional marker (see that decision above).
- Note: the probe's role has changed from #008's. It no longer establishes "`permissions.deny` blocks" as the single load-bearing premise; it establishes that both enforcement layers independently bound a subagent.
- Rejected: a single allowlist-only probe; no probe at all (an allowlist is a config file that can fail to load, not a structural guarantee).

### Scorer dispatch

- Decision: one Agent call per run. The scorer uses a generated agent definition pinned to the `opus` alias at `high` effort with **no tools at all**. The three-input blinded contract (redacted answer, `_meta`-stripped fasit, task text) is unchanged. `ladderbench redact` prints the scorer prompt; the session dispatches it; `ladderbench record-score` consumes the reply.
- Rationale: a scorer with zero tools cannot wander into the repo and un-blind itself, which is strictly stronger than the old `--strict-mcp-config` isolation. One call per run keeps the scorer from ever seeing two answers side by side.
- Rejected: one call per config scoring all 3 reps together (lets the scorer grade reps relative to each other, which the blinding forbids); keeping `claude -p` for scoring (contradicts the brief).

### The cold cell

- Decision: retained, as three dedicated short sessions run after all 14 warm sessions. Disposition logic (`confirmed-cold` / `partial` / `not-run` / `no-daemon-signal`, with `not_run_causes`) ports unchanged.
- Ownership of every cold-cell step, so none is left without a subcommand:
  - `prepare-session --config-id a5-bundle-cold --rep <n>` drains the daemons the warm sessions left resident on both task worktrees (`wait_for_daemon_exit` at `DAEMON_EXIT_TIMEOUT_S = 660`), builds the fresh worktree from `cold_worktree_template` with `{n}` substituted, clears the resolved state directory, asserts `gate_cold_before`, and writes that session's `.mcp.json` pointing at the new worktree. The drain and the clear are re-run per attempt, not once per repetition, so a failed attempt's leftover `daemon.json` cannot fail the precondition deterministically.
  - The session loop skips `warm` entirely for a cold config.
  - `cold-cell --teardown --rep <n>` removes the worktree (`remove_worktree`, unconditional, whatever the outcome) and waits for that worktree's daemon to exit.
  - `cold-cell` with no flags, run once after all three cold sessions, finalises `cold_cell.json`.
  - `restore-worktree` is **not** used by cold sessions: it resets and re-neutralises a persistent worktree, which is the opposite of a cold repetition's disposable one.
- Rationale: the MCP server's `--target-dir` is fixed at session start, so a fresh worktree per repetition forces a fresh session per repetition. Dropping the cell would silently remove one of the three comparison types `summarize` already models — not acceptable for a port meant to reach parity.
- Rejected: dropping the cold cell; re-pointing one session's target dir between repetitions (not possible).

### Daemon warm-up via the Go MCP SDK

- Decision: `ladderbench warm` performs the stdio `initialize` + `tools/call` against a freshly spawned `quarry-mcp` using `github.com/modelcontextprotocol/go-sdk`, already a direct dependency. `WARM_UP_TOOL` stays `workspace_symbol`, and the post-condition stays "a `daemon.json` now exists at the resolved state directory".
- Rationale: the Python version hand-rolled JSON-RPC framing only because it was constrained to the standard library; that constraint does not exist in Go, and the repo already links this SDK for the server side. Warming via the `quarry` CLI would resolve the same state directory but exercise a different code path than the one under measurement, weakening the warm/cold comparison.
- Rejected: a hand-rolled Go JSON-RPC stdio client; warming via the `quarry` CLI.

### Single-flight dispatch, mechanically enforced

- Decision: the skill mandates exactly one Agent call in flight at a time. `ladderbench ingest` fails loud when it is asked to ingest repetition `n` while repetition `n-1` of the same config has neither a `run.json` nor an exhausted attempt record — a predicate the per-repetition session loop above satisfies by construction, and which a batched "all runs then all scores" order would violate immediately.
- Rationale: the README's design rationale forbids concurrent runs (CPU, shared daemon, and rate-limit contention all make wall-clock incomparable across configs). A rule that exists only as prose fails silently — corrupted wall-clock numbers with nothing flagging them.
- Rejected: stating the rule without enforcing it; allowing concurrency for scorer calls only (defensible in isolation, but it turns a simple invariant into a conditional one for a payoff not worth the enforcement complexity).

### `ladder.yaml` additions

- Decision: add `run_effort: medium` and `session_dir_template: /tmp/ladder-session-{config_id}-{n}`. `require_pins` extends to reject an unset `run_effort` alongside `run_model`, `max_turns`, `scorer.model`, and `scorer.effort`. `run_model` continues to ship `null` by design.
- `{n}` is the **session index within the config**, not the repetition index in general: `1` for a warm session (a config has exactly one), and the repetition index for a cold session (a cold config has one session per repetition). Without it the three cold sessions would share one scratch dir, since they share the config id `a5-bundle-cold` — which would break both "each session runs from a disposable scratch directory" and `ingest`'s assumption that a session's transcript search space is one directory. `prepare-session` takes `--rep <n>` for a cold config and defaults `{n}` to `1` otherwise.
- The two probe sessions use the same template with `{config_id}` substituted as `probe-allowlist` / `probe-denylist` and `{n}` as `1`.
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
- `run_ladder.py` (1133 lines) — worktree lifecycle (`neutralise_worktree`, `build_worktree`, `restore_worktree`, `remove_worktree`, `ensure_task_worktrees`), `build_server`, `mcp_config_document`, `write_run_inputs`, `warm_daemon`, `run_env` (now applied at the three points named in the environment-scrubbing decision rather than around a subprocess launch), `task_text_for` with its load-bearing section boundary, `schema_for`, `plan_runs` / `main_runs` / `cold_runs` / `pending_runs`, `MAX_ATTEMPTS`, the run-outcome shape, and the cold-cell disposition logic. The `claude -p` argv assembly (`build_argv`, `launch_run`) and the subprocess drivers (`run_matrix`, `run_cold_cell`'s dispatch loop, `run_stage`, `run_probe`'s dispatch) are replaced by the skill plus the subcommand surface.

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

`prepare-session` writes into the scratch dir:

- `.mcp.json` — **only when the config's `allowed` set is non-empty**: a single server named `quarry`, `command` = the built `quarry-mcp` absolute path, `args` = `["--target-dir", "<task worktree>"]`, and an `env` block setting `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` to the empty string. A `none` config gets no file and is launched with no `--mcp-config`.
- `.claude/settings.json` — `permissions.allow` (`Read`, `Grep`, `Glob`, `Bash`) and `permissions.deny`: that config's derived quarry deny-list plus `Task` for a rung, and **`["Task"]` alone** for a `none` config.
- `.claude/agents/<config-id>.md` — the run agent definition, and `.claude/agents/<config-id>-scorer.md` — the zero-tool scorer definition.

Outside the scratch dir it installs `~/.claude/skills/ladder-run/SKILL.md` from the repo copy — never into the scratch dir, since a `none` session's scratch dir is the blinded agent's own cwd.
It also hard-fails when `QUARRY_STATE_DIR` or `QUARRY_BUILD_TAGS` is set non-empty in the operator's shell, and then prints the exact `claude` launch command for the operator to run.

**Known implementation risks, all to be settled by an actual `claude` launch during the batch that builds `prepare-session`, never assumed:**

1. Claude Code's `--setting-sources` flag interacts with project-local **agent-definition** discovery, and with user-scope **skill** discovery. The flag combination that isolates settings while still loading both must be established empirically. Fallback if project-local agent discovery is suppressed: write the definitions into `~/.claude/agents/` under a `ladder-<config-id>` namespace, with `prepare-session` responsible for cleaning them up. If user-scope skill discovery is also suppressed, the operator invokes the protocol by reading the installed `SKILL.md` path directly — the skill must never be relocated into a scratch dir to work around this.
2. Whether an MCP server entry's `env` block **replaces** or **augments** the inherited environment. If it augments, setting both keys to the empty string is sufficient; if it replaces, `QUARRY_CONFIG` must be forwarded explicitly in the same block or the server will not start on a machine that needs a `servers.yaml` overlay. The `prepare-session` precondition check exists partly as cover for this uncertainty.

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

Eleven subcommands, split at each session boundary because only a live session can invoke the Agent Tool:

- `prepare-session` — ensure task worktrees at their pins, build `quarry-mcp`, check the environment precondition, write the scratch dir's `.mcp.json` (config permitting) / `settings.json` / agent definitions / skill copy, print the launch command.
- `next-run` — resume-aware; print the next pending `(config, rep)` for this config, its current **attempt index** (derived by counting `<n>.invalid-<k>` siblings), its full prompt, and its agent-definition name.
- `warm` — the daemon warm-up call plus its `daemon.json` post-condition. Skipped for the cold config.
- `ingest` — locate the transcript by description, copy it and its meta in, extract `usage.json`, parse `answer.json`, run the gates, enforce the single-flight predicate, print the outcome (`complete-pending-scoring` / `truncated` / `failed`). Never destroys evidence.
- `invalidate` — rename the failed run directory aside to `<n>.invalid-<k>` and return the next attempt index; errors when `MAX_ATTEMPTS` is already exhausted, which is the matrix halt.
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
- Session-input generation: a `none` config produces **no** `.mcp.json` while a rung produces one; a `none` config's `settings.json` denies `Task` only while a rung's carries its full derived deny-list plus `Task`; the generated `.mcp.json` carries an `env` block emptying `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` while leaving `QUARRY_CONFIG` alone; the `prepare-session` environment precondition rejects each variable when set non-empty and accepts an empty or absent one.
- Scratch-dir path derivation: `session_dir_template` substitutes `{config_id}` and `{n}`, giving `1` for a warm config, the repetition index for a cold one, and distinct paths for the three cold sessions and the two probe sessions.
- `resolve_state_dir`'s environment check rejects a non-empty value and accepts both an absent key and a key set to the empty string.
- Attempt bookkeeping: `next-run`'s attempt index is 1 with no invalid siblings and `k+1` with `k` of them; `invalidate` picks the lowest unused index and errors once `MAX_ATTEMPTS` is exhausted.
- Single-flight: ingesting rep `n` errors while rep `n-1` has neither a `run.json` nor an exhausted attempt record, and succeeds once either is present.
- Preamble generation for both shapes (`none` control reproducing the committed Agent B text with `<TARGET_DIR>` substituted; a rung's generated MCP-shaped body listing exactly its allowed tools in canonical order).
- Fenced-JSON extraction, `first` and `last` selectors, and the no-block case.
- Usage extraction from a fixture subagent transcript: independent summation of all four token classes, `num_turns` from assistant-record count, `duration_ms` from first/last timestamps, `tool_uses_breakdown`, `quarry_tool_uses`, the separate `bash_grep_count` / `grep_tool_count` counters with the leading-command-word distinction (a `grep` inside a path must not count), `denied_tool_attempts` from denial-shaped errored tool results, and `granted_tools` read from a generated definition.
- Every gate, each with a passing and a failing fixture: denied tools used, target/buildTags override, model pinning including the `[1m]` suffix normalisation, blinding in all three of its outcomes (fatal `mcp__quarry__`, fatal repo-root path, fatal bare-quarry outside a `tool_result`, non-fatal bare-quarry confined to a `tool_result`), the new post-hoc `max_turns` ceiling, worktree neutralisation, cold-before and all three cold-after outcomes, and the complete-artifacts gate.
- Redaction: every alternation branch, the `impact` exemption, the case-insensitivity, the adjacent-token collapse, and structural fields left untouched.
- Scorer-prompt assembly: the three inputs are present, `_meta` is stripped from the fasit, and the config id, ladder, allowed set, and transcript are all absent.
- Summarization: `_median` at odd and even lengths, `ranges_disjoint` including the touching-at-one-value case, the grep-metric exclusion from rung-vs-control only, the same-ladder guard on `compare_rungs`, warm-vs-cold suppression on `not-run` / `partial` / all-`cold_no_daemon_backed_call`, the incomplete-cell list, and the non-zero exit.
- Run-state bookkeeping: `is_complete` false without `run.json` and false when `state != "complete"`, `invalidate`'s lowest-unused-index rename, and resume filtering that skips completed pairs.
- Transcript correlation: exactly one description match succeeds, zero matches errors, two matches errors.

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
- **Q:** `ladderbench` subcommand surface? **A:** Eleven subcommands split at each session boundary. Folding scoring into `ingest` is a structural non-starter: only a live session can invoke the Agent Tool, never a subprocess.
- **Q:** Model pinning between `ladder.yaml` and the agent definitions? **A:** One pinned id per role, Go maps id→alias, `gate_model_pinned` verifies the dispatched agent reported it. A wrong mapping surfaces empirically on the first run.
- **Q:** Transcript custody? **A:** Copy both the `.jsonl` and its `.meta.json` into the run directory and record the source path. The meta is what proves the description-match picked the right file.
- **Q:** `gate_blinding` under the new topology? **A:** Keep the three real checks, drop the dead `/tmp/quarry-bench` literal, never treat the scratch dir as a leak. A check that can never fire reads as coverage that is not there.
- **Q:** Documentation scope? **A:** README rewritten in place, `ladder.yaml` header refreshed, `CLAUDE.md`'s grandfather clause dropped in the final batch. A separate `docs/` file would duplicate the skill body.
- **Q:** Session granularity, given that `settings.json` is per session? **A:** One session per config — 17 total (14 warm, 3 cold). It is the only shape where both enforcement layers are genuinely per-rung, and 17 manual launches is the honest cost of the enforcement model already committed to.
- **Q:** Does this task run the matrix? **A:** No — harness only. Matches #008's precedent; a multi-hour supervised run should not gate a code port's mergeability, and the synthetic end-to-end test already covers the integration risk.
- **Q:** Sequential dispatch? **A:** Mechanically enforced single-flight, with `ingest` failing loud on an out-of-order transcript. A prose-only rule fails silently and corrupts wall-clock numbers with nothing flagging it.
- **Q:** New `ladder.yaml` fields? **A:** `run_effort` and `session_dir_template`, with `require_pins` extended to cover `run_effort`. Both are operator-tunable and reproducibility-relevant, so they belong in the single source of truth.
- **Q:** OS portability? **A:** Portable path handling, no Windows execution support, stated plainly. The Windows motivation was per-invocation interpreter startup on the dev machine, which a Go binary fixes regardless of where the matrix runs.
- **Q:** How does a repo-tracked skill reach a session launched from `/tmp`? **A:** [auto-pick] `prepare-session` copies `SKILL.md` into the scratch dir's `.claude/skills/`. **Why:** the scratch dir is a different project scope than the repo, so project-local discovery cannot reach it; the `~/.claude/skills/` install is the documented fallback if the isolation flags suppress project-local discovery.
- **Q:** Does a `none` session get a `.mcp.json`? **A:** [auto-pick] No — `write_run_inputs`'s rule ports verbatim to the session level. **Why:** a declared server named `quarry` exposing an `mcp__quarry__*` namespace is itself the structural leak the blinding forbids; omitting the file is what makes "the `none` arm never sees the namespace" literally true.
- **Q:** How many sessions? **A:** [auto-pick] 17 — 14 warm (3 reps each) plus 3 cold (1 rep each). **Why:** `ladder.yaml` holds 14 non-cold configs plus one cold config whose 3 repetitions each need their own worktree; the earlier "18" double-counted `a5-bundle-cold`.
- **Q:** What is a session's step order? **A:** [auto-pick] Fully serialized per repetition — warm, dispatch, `ingest`, `redact`, score, `record-score`, `restore-worktree`, next rep. **Why:** `run.json` is written last by `record-score`, so batching all runs before all scoring would make rep 2's `ingest` fail its own single-flight check.
- **Q:** Who owns retry and `MAX_ATTEMPTS`? **A:** [auto-pick] The skill owns the loop; an eleventh subcommand `invalidate` owns the rename-aside and the ceiling, and `next-run` derives the attempt index. **Why:** the attempt index is embedded in the correlation description, so it needs one derivation site on disk rather than living in the session's memory across a resume.
- **Q:** Where does `run_env`'s scrub apply now that no harness process wraps the dispatch? **A:** [auto-pick] Three points — an `env` block in the generated `.mcp.json`, a `prepare-session` precondition on the operator's shell, and the reconstructed environment the Go gates resolve against. **Why:** the cold cell's whole argument rests on the state directory being a function of the worktree path; a single unverified application point is not enough given the MCP `env` inherit-versus-replace semantics are unestablished.
- **Q:** `denied_tool_attempts` matches a denial shape nobody has observed. **A:** [auto-pick] Ship it flagged provisional, and have the deny-list probe capture the real shape into `probe.json` for the follow-up task to validate against. **Why:** dropping it loses the only signal a rung reached past its allowlist, and shipping it silently would be a number that reads as measured but was never checked.
- **Q:** The blinded agent's cwd is the scratch dir, which was to hold a deny-list naming all seven quarry tools and a skill describing the ladder. **A:** [auto-pick] A `none` session's scratch dir holds only its neutral agent definition and a `Task`-only `settings.json`; the skill installs at `~/.claude/skills/` for every session type. **Why:** `gate_blinding` makes any `mcp__quarry__` substring fatal, so one `cat` of the session's own settings would both leak and burn an attempt — and for a `none` config the deny-list guards nothing anyway, since no server is declared. The earlier "costs nothing" claim was wrong.
- **Q:** Which subcommand owns the cold cell's drain, build, clear, and remove steps? **A:** [auto-pick] `prepare-session --config-id a5-bundle-cold --rep <n>` owns drain/build/clear/`gate_cold_before`; `cold-cell --teardown --rep <n>` owns removal and the post-run wait; bare `cold-cell` finalises the disposition. **Why:** `restore-worktree` resets a persistent worktree, which is the opposite of a cold repetition's disposable one, so it cannot absorb them.
- **Q:** `session_dir_template` keyed on `{config_id}` alone collides across the three cold sessions. **A:** [auto-pick] Template becomes `/tmp/ladder-session-{config_id}-{n}`, where `{n}` is the session index within the config — 1 for a warm session, the repetition index for a cold one. **Why:** the three cold sessions share one config id, and a shared scratch dir breaks both the disposability claim and `ingest`'s one-directory transcript search space.
- **Q:** Nothing generates the two probe sessions' inputs. **A:** [auto-pick] `prepare-session --probe allowlist|denylist` materialises each. **Why:** generating the inputs is mechanical work that belongs with the rest of `prepare-session`; only the dispatch is deferred to the follow-up task.
- **Q:** `model` and `transcript` were missing from the metric partition. **A:** [auto-pick] Both survive — `model` with its source moved to the assistant records, `transcript` now naming the in-run-directory copy, with `transcript_source` added for the original path. **Why:** the partition is presented as exhaustive over today's `usage.json`, so an omission reads as an undecided field.
