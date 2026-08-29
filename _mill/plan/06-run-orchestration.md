# Batch: run-orchestration

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "run-orchestration"
number: 6
cards: 7
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_run_ladder.py -q
depends-on: [1, 2, 3, 4, 5]
```

## Batch Scope

This batch delivers `run_ladder.py`, the harness entry point: worktree lifecycle, quarry-mcp build and per-run MCP config, the preflight probe, the per-run execution pipeline, the sequential matrix driver with resume and the three-attempt cap, and the cold-cell driver. It is one batch because every piece shares one execution model — one subprocess per run, one directory per run, one terminal marker per run — and splitting it would force that model to be re-established in two places.

It depends on all five earlier batches: it generates settings and preambles through `ladder_config.py`, parses transcripts through `extract_usage.py`, validates through `gates.py`, scores through `score_run.py`, and hands off to `summarize.py`. It re-implements none of them.

Batch-local decision: the dispatch layer is not mocked. Card 22's tests cover only the pure planning and resume logic — run ordering, attempt accounting, the resume skip decision, argv assembly — and never invoke a model. The discussion is explicit that dispatch is exercised by actually running the matrix (batch 8), not by a mock.

## Cards

### Card 16: task worktree lifecycle

- **Context:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
  - `bench/loomyard-eval/scripts/gen_compact_toc.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create the module with the standard docstring shape and add:
  - `build_worktree(ladder, path, sha, git=run_git)` — `git -C <source_repo> worktree add <path> <sha>`, then immediately `neutralise_worktree`, then assert `gate_worktree_neutralised` passes. Raises `HarnessError` when a directory already exists at that path, so a stale worktree is never silently reused. `ensure_task_worktrees` is the idempotent caller; nothing else calls this directly.
  - `neutralise_worktree(path)` — delete `CLAUDE.md`, `CONSTRAINTS.md`, and the `.claude/` directory from the disposable worktree. This is a mutation of the disposable checkout only; the live source checkout is never touched.
  - `restore_worktree(path, git=run_git)` — `git -C <path> reset --hard` followed by `git -C <path> clean -fdx`, then `neutralise_worktree` again, since `clean -fdx` restores the ambient-context files the neutralisation removed. Called unconditionally after every main-matrix run.
  - `remove_worktree(ladder, path, git=run_git)` — `git -C <source_repo> worktree remove --force <path>`.
  - `run_git(args)` — the single seam every git call in this module goes through, running `git` and returning its stdout. Every function below takes it as a `git=run_git` default parameter so card 22's tests can drive them against an injected runner without creating a real worktree.
  - `ensure_task_worktrees(ladder, git=run_git)` — return a mapping from task key to worktree path, idempotently, because the harness is re-invoked to resume and this runs on every invocation. For each task: when no directory exists at the declared path, `build_worktree` it. When one does exist, read `git -C <path> rev-parse HEAD` — if it equals the task's declared pin, adopt the existing worktree by calling `restore_worktree` on it and continue; if it does not, raise `HarnessError` naming both SHAs, since a worktree at the wrong pin would silently benchmark a different codebase. A resume must never halt merely because the worktrees it built last time are still there, and must never adopt one it cannot prove is at the right commit.

  The restore-then-re-neutralise ordering is load-bearing: `git clean -fdx` would otherwise leave the target's own `CLAUDE.md`, `CONSTRAINTS.md`, and `.claude/` back in place for the next run, re-introducing the uncontrolled prompt prefix and the second permissions source the neutralisation exists to remove.

  Card 22 writes this card's tests; nothing here is asserted yet beyond what that card covers.
- **Commit:** `feat(bench): manage disposable task worktrees for the ladder`

### Card 17: server binary, per-run MCP config, and warm-up

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `docs/mcp-setup.md`
  - `cmd/quarry-mcp/main.go`
  - `.mcp.json`
  - `internal/mcpserver/tools_toc.go`
  - `internal/quarryengine/daemon/ensureserver.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add:
  - `build_server(repo_root)` — `go build -o quarry-mcp ./cmd/quarry-mcp` from the repo root with `CGO_ENABLED=1`, returning the absolute path to the built binary. Raises `HarnessError` with the compiler output when the build fails, naming the CGO toolchain requirement, since a missing C toolchain fails at compile time. The warm-start path is used rather than the committed `go run` form so a cold build cache cannot make a run's first connection exceed the client's connect timeout.
  - `mcp_config_document(server_path, target_dir)` — the `--mcp-config` mapping declaring a single server named `quarry`, with `command` the built binary's absolute path and `args` an explicit `--target-dir <target_dir>`.
  - `write_run_inputs(ladder, config, run_dir, target_dir, server_path)` — materialise that run's settings JSON and, for a config with a non-empty allowed set, its MCP config JSON, into the run directory. A config whose allowed set is empty gets **no** MCP config file and is launched with no `--mcp-config` at all: the quarry server is never declared to it, because a declared server named `quarry` exposing an `mcp__quarry__*` namespace is itself the structural leak the blinding forbids.
  - `mcp_call(server_path, target_dir, tool, arguments, env, timeout_s)` — a minimal MCP stdio client, standard library only. Spawn the server binary with `--target-dir <target_dir>` and pipes on stdin and stdout; write a JSON-RPC `initialize` request, then an `initialized` notification, then a `tools/call` request naming `tool` and `arguments`, each as one JSON object followed by a newline; read newline-delimited JSON objects back off stdout until the response whose `id` matches the request arrives; then close stdin and terminate the process. Newline-delimited framing is what the server speaks: `cmd/quarry-mcp/main.go` runs the server over the SDK's `mcp.StdioTransport{}`, whose transport is documented as newline-delimited JSON. Raises `HarnessError` on a JSON-RPC error response, on a malformed line, or on `timeout_s` elapsing.
  - `WARM_UP_TOOL = "workspace_symbol"` and `warm_daemon(server_path, target_dir, env, cache_dir)` — one `mcp_call` against that tool with the arguments `{"targets": [{"query": "Run"}]}`, then an assertion that a `daemon.json` now exists at the state directory `gates.resolve_state_dir` derives for `target_dir`, raising `HarnessError` when it does not. The query needs no match: `workspace_symbol`'s handler calls `resolveCall` once for the whole call before resolving any target, so the daemon starts whether or not the query resolves — which is why the post-condition is the state file's existence and not the result payload. Without that assertion a malformed call would leave every "pre-warmed" run cold and silent. The warm-up tool must be daemon-backed: `toc_file` and `toc_dir` reach the tree-sitter path directly and never `EnsureServer`, so warming with a toc call would start no daemon at all and every "pre-warmed" main-matrix run would in fact be cold. `workspace_symbol` resolves a bare symbol project-wide, which is exactly the daemon start the main matrix needs to have already paid for. It is called by `run_matrix` immediately before each main-matrix run's dispatch — per run, not once per worktree, since the daemon self-expires after its idle timeout and a gap between runs would otherwise leave a nominally warm cell cold. Never called for a config with `cold: true`.
  - `run_env()` — the environment every subprocess inherits, with both `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` removed. `--state-dir` is never passed anywhere in this module, and no generated preamble permits a call-level `buildTags`. All three would move the resolved state directory off the per-path key the cold cell depends on: the first two take precedence over `workspaceKey` outright, and a non-empty tag set appends a `tags-<hex>` segment at every tier, which would make the cold-cell gates read an empty directory and report a native fallback that never happened. `QUARRY_CONFIG` is deliberately **not** scrubbed: it selects the `servers.yaml` overlay naming the language-server command, and clearing it on a machine that needs an overlay would stop the server starting at all. Its resolved value is recorded in the probe record instead, so a run environment that differs from a default one is visible rather than silent.
- **Commit:** `feat(bench): build the quarry-mcp server and materialise per-run launch inputs`

### Card 18: the preflight denial probe

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `run_probe(ladder, repo_root, results_root, target_dir, server_path)`, executed once before any matrix run. It launches one throwaway `claude -p` run with the quarry server declared, a deny-list naming `mcp__quarry__impact`, and a prompt instructing the agent to call that denied tool and report what happened. It writes `probe.json` into the results root recording:
  - `denial_blocks` — whether the denied call failed to succeed. Determined from the transcript: `True` when no `tool_use` block for the denied name has a non-errored `tool_result`. This is the load-bearing premise of the whole suite; when it is `False`, `run_probe` raises `HarnessError` and the matrix halts before a single paid run, because every rung would silently be the full bundle.
  - `denied_tools_advertised` — whether the denied name appears in the init event's `tools` array. When it does not, denied tools are hidden from the model, `denied_tool_attempts` can never be non-zero, and the field is dropped from the reported metrics rather than reported as a meaningless column of zeros. The recorded boolean is what `summarize.py` reads as `denied_tool_attempts_reported`.
  - `advertised_tools`, `session_id`, the resolved model, and the probe's own transcript path, so the answer is auditable rather than asserted.

  The probe runs with the same `--setting-sources ""`, `--strict-mcp-config`, `--max-turns`, and non-interactive permission mode as every matrix run, so what it establishes is true of the matrix and not of a differently-configured invocation.
- **Commit:** `feat(bench): probe permission-deny semantics before the matrix starts`

### Card 19: execute one run end to end

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/README.md`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add:
  - `task_text_for(ladder, task_key)` — extract the task's task-text block from its committed task file, so the preamble is assembled from the tracked source rather than from a copy. The section boundary is stated exactly, because over-reading it is the worst failure this harness can have: start at the line whose heading text is the task-text heading both task files use, take every following line up to but not including the next line beginning with `## `, strip a leading `> ` or `>` from each line, and strip surrounding blank lines. Raise `HarnessError` when the heading is absent or the extracted body is empty. The terminator is load-bearing rather than tidiness: task 01's very next section is its fasit leads, and task 04's task text is followed by its output schema and then by `## Notes for whoever scores this (ground truth — do not reveal to A/B/C)`, which names the three real callers and the `burler.go:373` decoy outright. An extractor that ran past the boundary would paste the answer key into all 45 runs' prompts and every correctness number in the matrix would be worthless. Add a test asserting the extracted text for both tasks contains neither `burler.go:373` nor the word `fasit`.
  - `schema_for(ladder, task_key)` — return the task's output schema, from the source that actually holds it, which differs per task and is not uniform. The impact schema is in task 04's own `## Output schema (impact-analysis tasks)` section. Task 01 has no schema section at all — its sections are Setup, Scope, the task text, and scoring notes — so the exploration schema comes from the `## Output schemas` section of the benchmark README named in this card's Context, under its "Exploration tasks" heading. Selection is driven by the task's declared `schema` field, never by guessing which file to read. Raises `HarnessError` when the named section is absent, rather than silently emitting a prompt with no schema in it.
  - `build_argv(ladder, config, run_dir, target_dir)` — the full `claude` argv: `-p`, the generated preamble as the prompt, `--model` the pinned run model, `--output-format stream-json`, `--verbose`, `--setting-sources ""`, `--strict-mcp-config`, `--settings <run_dir>/settings.json`, `--permission-mode dontAsk`, `--max-turns` from the ladder's `max_turns` field, and `--mcp-config <run_dir>/mcp.json` only when the config's allowed set is non-empty. The turn ceiling is identical across all 45 runs and the probe, so it bounds what a wandering run can cost without biasing any comparison; a matrix launched with no bound at all has no cost ceiling other than the operator noticing. The process working directory is the run's task worktree, never this repository.
  - `execute_run(ladder, config, n, run_dir, target_dir, server_path, repo_root, cache_dir, executor=launch_run)` — the per-run pipeline, in this order: materialise inputs; record a monotonic start time; launch the subprocess through `executor`, capturing its stdout stream directly to `transcript.jsonl` as it runs; record wall clock; parse the final fenced json block out of the result event's `result` field into `answer.json`; extract `usage.json`; run `run_gates`, which is where `observe_worktree_dirtied` is taken — before any restore, since the restore is what erases the evidence; invoke `score_run`; run `gate_run_complete_artifacts`; and only then `write_run_json`, carrying the gate report's non-fatal observations into it. `run_gates` runs before scoring and therefore asserts on no scoring artifact; `gate_run_complete_artifacts` is the separate later gate that requires all four files. Any failing fatal gate, an unparseable answer, or a scoring failure returns a failed outcome carrying the findings, and no `run.json` is written.
  - `launch_run(argv, cwd, env, transcript_path)` — the default `executor`: the one place this module starts a `claude` subprocess. It exists as a named seam so card 22's attempt-accounting tests drive an injected executor rather than a model.

  The transcript is captured from the stream as the process runs and is never located after the fact by `session_id`: the client's on-disk session layout is not something this suite controls, and streaming means a run that dies mid-way still leaves a diagnosable transcript.
- **Commit:** `feat(bench): execute one ladder run end to end`

### Card 20: the sequential matrix driver

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add:
  - `plan_runs(ladder)` — the ordered list of all 45 `(config, n)` pairs: every non-cold config's `reps` repetitions first, then the cold config's. It is the reporting view of the whole matrix; neither driver iterates it directly.
  - `main_runs(ladder)` and `cold_runs(ladder)` — the two disjoint partitions of `plan_runs`, by the config's `cold` field. `run_matrix` consumes only `main_runs` and `run_cold_cell` only `cold_runs`: a cold pair reaching the main driver would execute through the main pipeline — shared worktree, possibly warm daemon, none of the cold gates applied — and would then be `is_complete` by the time `run_cold_cell` looked, which would silently skip it and report warm numbers as cold. Order is deterministic and the cold cell is always last, because a supervised cold run leaves a daemon resident for the client's idle timeout against a worktree the harness then deletes, and running the cell first would leave those daemons resident through the opening of the main matrix, loading the very timing figures the separation rule reads.
  - `pending_runs(pairs, results_root)` — any pair list filtered through `gates.is_complete`, so a re-invocation skips finished runs rather than re-running the matrix. Both drivers call it with their own partition.
  - `MAX_ATTEMPTS = 3` and `run_matrix(ladder, results_root, worktrees, server_path, repo_root, cache_dir, executor=launch_run, git=run_git)` — iterate `pending_runs(main_runs(ladder), results_root)` one at a time, never concurrently. For each: call `warm_daemon` against that run's task worktree, then execute; on success continue; on failure `invalidate` the run directory and retry, up to `MAX_ATTEMPTS` attempts for that run. On the third failure, halt the **whole** matrix — not just that cell — reporting which gate failed and why, and leaving every completed run intact so the matrix resumes once the cause is fixed. After every main-matrix run, successful or not, call `restore_worktree` unconditionally. The `worktree_dirtied` observation is **not** taken here — `execute_run` already recorded it before the restore, and an observation taken after a `reset --hard` plus `clean -fdx` would be `False` for every run in the matrix. Dirtying is never itself a failure.

  Sequential dispatch is not an implementation convenience: concurrent runs contend for CPU, for the shared daemon, and for model-side rate limits, any of which would make wall-clock incomparable across configs and could manufacture or erase a separation.
- **Commit:** `feat(bench): drive the matrix sequentially with resume and an attempt cap`

### Card 21: the cold-cell driver

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `internal/quarryengine/daemon/ensureserver.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `run_cold_cell(ladder, results_root, server_path, repo_root, cache_dir, executor=launch_run, git=run_git)`, executed after the entire main matrix. For each of the cold config's repetitions:
  - Iterate `pending_runs(cold_runs(ladder), results_root)`, and give each repetition the same `MAX_ATTEMPTS` cap the main matrix uses — the cold cell has its own retry loop rather than borrowing the main driver's, because its per-attempt setup and teardown differ.
  - Build a fresh worktree at `cold_worktree_template` with `{n}` substituted, so each run has a distinct absolute target-dir path and therefore a distinct daemon key. On a failed attempt the worktree is removed before the next attempt rebuilds it at the same path: `build_worktree` raises when a directory already exists, so a retry that skipped the teardown would fail on the rebuild rather than retrying the run. The path is never reused across repetitions, and the worktree is removed after the final attempt whatever its outcome.
  - Run **no** warm-up step.
  - Before launching, assert `gate_cold_before`. A pre-existing `daemon.json` under the resolved state directory means the daemon is already warm; the run is invalidated rather than reported as cold.
  - Execute the run through the same `execute_run` pipeline as every other run.
  - After the run, assert `gate_cold_after` on its three outcomes. When the run used a daemon-backed tool and daemon state is present, it is a confirmed supervised cold run. When it used one and no state is present, the native fallback was taken — on that path the shared daemon address is not a function of the state directory at all, so a distinct worktree buys nothing and three "cold" runs could all have joined one warm daemon; the run is invalidated, never reported as cold. When it used none, the run is valid and carries the `cold_no_daemon_backed_call` observation into its `run.json`: it started no daemon, so it is neither a fallback nor a warmth measurement.
  - Before starting the next repetition, call `wait_for_daemon_exit` on the finished run's own state directory with a bound of `DAEMON_EXIT_TIMEOUT_S = 660` — the daemon's own 10-minute idle timeout plus a minute of margin — so no cold run is timed alongside a resident daemon from its predecessor. The wait keys on the pid recorded in that run's `daemon.json`, because neither the state file nor the state directory is removed on exit. A run that recorded `cold_no_daemon_backed_call` started no daemon, so the wait returns immediately.
  - Remove the worktree.

  Three dispositions, returned to the caller rather than raised. Confirmed-cold: at least one repetition was a supervised cold run. Not-run: the supervised strategy is unavailable on this machine, every attempt having failed `gate_cold_after` on the native-fallback branch — an environment limitation, not a fault, so the cell is reported as not run rather than with numbers that cannot be trusted, and the matrix is not halted. No-daemon-signal: all three repetitions completed validly but none invoked a daemon-backed tool, so the cell holds three good runs that say nothing about warmth. The third is distinct from the second precisely because the runs are valid and are still summarised — what is absent is the contrast, not the data.
- **Commit:** `feat(bench): run the cold-daemon comparison cell last, with coldness asserted`

### Card 22: orchestration tests and the CLI entry point

- **Context:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the CLI under `if __name__ == "__main__":` taking a ladder path and a results root, with the documented order of operations: require the run model, build the server, `ensure_task_worktrees`, run the probe unless a `probe.json` already present in the results root records a passing probe, run the main matrix, run the cold cell, and print where `summarize.py` should be pointed. The probe skip matters because the CLI runs end to end on every resume: without it, each resume would spend a paid run re-probing and would overwrite an already-committed probe record. It refuses to start when the run model is unset, halting with a message naming the field and the ladder file.

  Then the tests, covering only the pure logic:
  - `plan_runs` yields 45 pairs, three per config, with all three cold-cell pairs strictly last and every main-matrix pair before them; `main_runs` and `cold_runs` partition it disjointly into 42 and 3, and no cold pair appears in `main_runs`.
  - `pending_runs` skips a run whose directory carries a `state: "complete"` `run.json`, includes one whose directory is absent, and includes one whose directory holds an answer and usage but no `score.json`.
  - `build_argv` includes `--mcp-config` for a config with a non-empty allowed set and omits it entirely for one whose allowed set is empty; includes `--setting-sources`, `--strict-mcp-config`, `--output-format stream-json`, the pinned `--model`, and `--max-turns` carrying the ladder's value; and never includes `--state-dir`.
  - `run_env` removes both `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` from an environment that had them set, and leaves `QUARRY_CONFIG` in place.
  - `task_text_for` stops at the next `## ` heading, strips the `> ` blockquote prefix, and raises when the heading is absent — with a dedicated assertion that neither task's extracted text contains `burler.go:373` or the word `fasit`, since over-reading the boundary would leak the answer key into every run.
  - `ensure_task_worktrees` builds a missing worktree, adopts an existing one whose `HEAD` matches the declared pin, and raises when an existing one is at a different SHA — asserted against an injected git runner, with no real worktree created.
  - The probe is skipped when the results root already holds a `probe.json` recording a passing probe, and is run when it holds none.
  - `WARM_UP_TOOL` is a member of `DAEMON_BACKED_TOOLS`, so a future edit cannot quietly make the warm-up a toc call.
  - `run_matrix` calls `warm_daemon` once before each main-matrix dispatch and never for a cold pair, asserted against injected seams.
  - `mcp_config_document` declares exactly one server, named `quarry`, whose args carry the run's own target dir.
  - The attempt accounting: a run failing three times halts the whole matrix rather than skipping the cell, asserted against an injected executor that always fails, and a run failing twice then succeeding leaves the matrix running, asserted against an injected executor that fails twice.

  No test in this file launches a subprocess, builds a worktree, or makes a model call; every one of them drives injected callables.
- **Commit:** `feat(bench): add the ladder CLI entry point and orchestration tests`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_run_ladder.py`, the only test file this batch creates. Its scope is deliberately narrow: run ordering and the cold-cell-last rule, resume decisions, argv assembly, environment scrubbing, MCP-config shape, task-text and schema extraction with their section boundaries, and attempt accounting. Every one of them drives an injected `git=` or `executor=` seam, which is why cards 16, 19, 20, and 21 declare those parameters on their signatures rather than leaving the tests without a way in. The dispatch layer, the worktree builds, the server build, the probe, and the cold-daemon assertions are exercised by actually running the matrix in batch 8, which is where the discussion puts them — mocking a model inside the unit suite would assert the mock, not the harness.
