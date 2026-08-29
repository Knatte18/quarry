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
  - `build_worktree(ladder, path, sha)` — `git -C <source_repo> worktree add <path> <sha>`, then immediately `neutralise_worktree`, then assert `gate_worktree_neutralised` passes. Raises `HarnessError` when the worktree already exists at that path, so a stale worktree is never silently reused.
  - `neutralise_worktree(path)` — delete `CLAUDE.md`, `CONSTRAINTS.md`, and the `.claude/` directory from the disposable worktree. This is a mutation of the disposable checkout only; the live source checkout is never touched.
  - `restore_worktree(path)` — `git -C <path> reset --hard` followed by `git -C <path> clean -fdx`, then `neutralise_worktree` again, since `clean -fdx` restores the ambient-context files the neutralisation removed. Called unconditionally after every main-matrix run.
  - `remove_worktree(ladder, path)` — `git -C <source_repo> worktree remove --force <path>`.
  - `ensure_task_worktrees(ladder)` — build each task's shared worktree once, at the path and pin its `tasks:` entry declares, and return a mapping from task key to path.

  The restore-then-re-neutralise ordering is load-bearing: `git clean -fdx` would otherwise leave the target's own `CLAUDE.md`, `CONSTRAINTS.md`, and `.claude/` back in place for the next run, re-introducing the uncontrolled prompt prefix and the second permissions source the neutralisation exists to remove.

  Card 22 writes this card's tests; nothing here is asserted yet beyond what that card covers.
- **Commit:** `feat(bench): manage disposable task worktrees for the ladder`

### Card 17: server binary, per-run MCP config, and warm-up

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `docs/mcp-setup.md`
  - `cmd/quarry-mcp/main.go`
  - `.mcp.json`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add:
  - `build_server(repo_root)` — `go build -o quarry-mcp ./cmd/quarry-mcp` from the repo root with `CGO_ENABLED=1`, returning the absolute path to the built binary. Raises `HarnessError` with the compiler output when the build fails, naming the CGO toolchain requirement, since a missing C toolchain fails at compile time. The warm-start path is used rather than the committed `go run` form so a cold build cache cannot make a run's first connection exceed the client's connect timeout.
  - `mcp_config_document(server_path, target_dir)` — the `--mcp-config` mapping declaring a single server named `quarry`, with `command` the built binary's absolute path and `args` an explicit `--target-dir <target_dir>`.
  - `write_run_inputs(ladder, config, run_dir, target_dir, server_path)` — materialise that run's settings JSON and, for a config with a non-empty allowed set, its MCP config JSON, into the run directory. A config whose allowed set is empty gets **no** MCP config file and is launched with no `--mcp-config` at all: the quarry server is never declared to it, because a declared server named `quarry` exposing an `mcp__quarry__*` namespace is itself the structural leak the blinding forbids.
  - `warm_daemon(server_path, target_dir, env)` — start the server against the target dir, issue one cheap tool call, and shut it down, so the main matrix's runs all execute against a pre-warmed daemon. Never called for a config with `cold: true`.
  - `run_env()` — the environment every subprocess inherits, with `QUARRY_STATE_DIR` removed. `--state-dir` is never passed anywhere in this module. Both take precedence over the per-path keying the cold cell depends on.
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

  The probe runs with the same `--setting-sources ""`, `--strict-mcp-config`, and non-interactive permission mode as every matrix run, so what it establishes is true of the matrix and not of a differently-configured invocation.
- **Commit:** `feat(bench): probe permission-deny semantics before the matrix starts`

### Card 19: execute one run end to end

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/README.md`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add:
  - `task_text_for(ladder, task_key)` — extract the task file's `<TASK TEXT>` block and the task's output schema, so the preamble is assembled from the committed task file rather than from a copy.
  - `build_argv(ladder, config, run_dir, target_dir)` — the full `claude` argv: `-p`, the generated preamble as the prompt, `--model` the pinned run model, `--output-format stream-json`, `--verbose`, `--setting-sources ""`, `--strict-mcp-config`, `--settings <run_dir>/settings.json`, `--permission-mode dontAsk`, and `--mcp-config <run_dir>/mcp.json` only when the config's allowed set is non-empty. The process working directory is the run's task worktree, never this repository.
  - `execute_run(ladder, config, n, run_dir, target_dir, server_path, repo_root)` — the per-run pipeline, in this order: materialise inputs, record a monotonic start time, launch the subprocess capturing its stdout stream directly to `transcript.jsonl` as it runs, record wall clock, parse the final fenced json block out of the result event's `result` field into `answer.json`, extract `usage.json`, run `run_gates`, invoke `score_run`, and only then `write_run_json`. Any failing fatal gate, an unparseable answer, or a scoring failure returns a failed outcome carrying the findings, and no `run.json` is written.

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
  - `plan_runs(ladder)` — the ordered list of `(config, n)` pairs for the whole matrix: every non-cold config's `reps` repetitions first, then the cold config's. Order is deterministic and the cold cell is always last, because a supervised cold run leaves a daemon resident for the client's idle timeout against a worktree the harness then deletes, and running the cell first would leave those daemons resident through the opening of the main matrix, loading the very timing figures the separation rule reads.
  - `pending_runs(ladder, results_root)` — `plan_runs` filtered through `is_complete`, so a re-invocation skips finished runs rather than re-running the matrix.
  - `MAX_ATTEMPTS = 3` and `run_matrix(...)` — iterate `pending_runs` one at a time, never concurrently. For each: execute; on success continue; on failure `invalidate` the run directory and retry, up to `MAX_ATTEMPTS` attempts for that run. On the third failure, halt the **whole** matrix — not just that cell — reporting which gate failed and why, and leaving every completed run intact so the matrix resumes once the cause is fixed. After every main-matrix run, successful or not, call `restore_worktree` unconditionally and record the `worktree_dirtied` observation for that run; dirtying is never itself a failure.

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
- **Requirements:** Add `run_cold_cell(ladder, results_root, server_path, repo_root)`, executed after the entire main matrix. For each of the cold config's repetitions:
  - Build a fresh worktree at `cold_worktree_template` with `{n}` substituted, so each run has a distinct absolute target-dir path and therefore a distinct daemon key. The path is never reused: the worktree is removed immediately after its run.
  - Run **no** warm-up step.
  - Before launching, assert `gate_cold_before`. A pre-existing `daemon.json` under the resolved state directory means the daemon is already warm; the run is invalidated rather than reported as cold.
  - Execute the run through the same `execute_run` pipeline as every other run.
  - After the run, assert `gate_cold_after`. Its absence means the native fallback was taken — on that path the shared daemon address is not a function of the state directory at all, so a distinct worktree buys nothing and three "cold" runs could all have joined one warm daemon. Such a run is invalidated, never reported as cold.
  - Before starting the next repetition, wait for the previous run's daemon to exit, so no cold run is timed alongside a resident daemon from its predecessor.
  - Remove the worktree.

  When the supervised strategy proves unavailable on the operator's machine — every attempt failing `gate_cold_after` — the cold cell is reported as **not run** rather than reported with numbers that cannot be trusted, and this disposition is distinct from the matrix-wide halt: it is an environment limitation, not a fault. `run_cold_cell` returns that disposition to its caller rather than raising.
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
- **Requirements:** Add the CLI under `if __name__ == "__main__":` taking a ladder path and a results root, with the documented order of operations: require the run model, build the server, build the task worktrees, run the probe, run the main matrix, run the cold cell, and print where `summarize.py` should be pointed. It refuses to start when the run model is unset, halting with a message naming the field and the ladder file.

  Then the tests, covering only the pure logic:
  - `plan_runs` yields 45 pairs, three per config, with all three cold-cell pairs strictly last and every main-matrix pair before them.
  - `pending_runs` skips a run whose directory carries a `state: "complete"` `run.json`, includes one whose directory is absent, and includes one whose directory holds an answer and usage but no `score.json`.
  - `build_argv` includes `--mcp-config` for a config with a non-empty allowed set and omits it entirely for one whose allowed set is empty; includes `--setting-sources`, `--strict-mcp-config`, `--output-format stream-json`, and the pinned `--model`; and never includes `--state-dir`.
  - `run_env` removes `QUARRY_STATE_DIR` from an environment that had it set.
  - `mcp_config_document` declares exactly one server, named `quarry`, whose args carry the run's own target dir.
  - The attempt accounting: a run failing three times halts the whole matrix rather than skipping the cell, asserted against an injected executor that always fails, and a run failing twice then succeeding leaves the matrix running, asserted against an injected executor that fails twice.

  No test in this file launches a subprocess, builds a worktree, or makes a model call; every one of them drives injected callables.
- **Commit:** `feat(bench): add the ladder CLI entry point and orchestration tests`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_run_ladder.py`, the only test file this batch creates. Its scope is deliberately narrow: run ordering and the cold-cell-last rule, resume decisions, argv assembly, environment scrubbing, MCP-config shape, and attempt accounting. The dispatch layer, the worktree builds, the server build, the probe, and the cold-daemon assertions are exercised by actually running the matrix in batch 8, which is where the discussion puts them — mocking a model inside the unit suite would assert the mock, not the harness.
