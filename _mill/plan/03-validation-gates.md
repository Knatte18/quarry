# Batch: validation-gates

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "validation-gates"
number: 3
cards: 3
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_gates.py -q
depends-on: [1, 2]
```

## Batch Scope

This batch delivers `gates.py`: every per-run validation gate, as pure predicates over a parsed transcript, a filesystem state, and a run directory. The discussion makes these verification steps over produced data rather than unit tests of the runs, and names each one as separately testable — which is why they live in their own module instead of inside the dispatch layer.

The external interface batch 6 consumes is `gates.py`'s `run_gates(...) -> GateReport`, plus the individual predicates it composes. `run_ladder.py` calls `run_gates` once per run and writes `run.json` only when the report passes; it never re-implements a gate.

Batch-local decision: a gate returns a structured `GateFinding` rather than raising, so one run can fail several gates and report all of them. `GateReport.passed` is `True` only when no finding has `fatal: True`; findings that are observations (`worktree_dirtied`, `target_origin_quarry_mention`) are carried on the report with `fatal: False` and never block.

## Cards

### Card 8: transcript-derived gates

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/ladder/tests/fixtures/bundle-mixed-tools.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/denied-attempt.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/targetdir-override.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/none-target-origin-mention.jsonl`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Define the frozen dataclasses `GateFinding` (fields `gate`, `message`, `fatal`) and `GateReport` (fields `findings`, plus a `passed` property). Then the transcript gates, each taking the parsed event list and returning a list of findings:
  - `gate_denied_tools_not_used(events, denied_names)` — fatal when any `tool_use` block names a tool in that config's deny-list **and** its matching `tool_result` did not error. A denied name that appears only as a rejected attempt is not a violation; it is the `denied_tool_attempts` metric.
  - `gate_no_target_override(events)` — fatal when any `mcp__quarry__*` tool call's `input` carries a `targetDir` or a `buildTags` key. A run that retargets breaks both the pinned-worktree constraint and the cold cell's daemon key.
  - `gate_model_pinned(events, run_model)` — fatal when the init event's `model` does not match the pinned id. Match by normalising away a trailing bracketed context-window suffix, so a pinned `claude-opus-5` is satisfied by a reported `claude-opus-5[1m]` but not by any other id.
  - `gate_blinding(events, repo_root)` — applies only to a config whose `allowed` is empty. Fatal when the transcript contains an `mcp__quarry__` tool name, the literal `/tmp/quarry-bench`, or any filesystem path into `repo_root`. A bare `quarry` mention whose only occurrence is inside a `tool_result` payload is **not** fatal: it records a non-fatal `target_origin_quarry_mention` finding instead, because the target codebase mentions quarry in its own tracked files and a bare-string gate would halt the matrix over the target's own prose.

  Tests assert each gate against the fixtures: the bundle fixture passes the denial and override gates; the `targetdir-override` fixture fails `gate_no_target_override` on both the `targetDir` and the `buildTags` call; the `denied-attempt` fixture passes `gate_denied_tools_not_used` because the attempt did not succeed while still being counted by the extractor; a model mismatch is fatal while the bracketed-suffix form is not; the `none-target-origin-mention` fixture passes `gate_blinding` and carries the non-fatal observation; and a `none`-shaped transcript containing an `mcp__quarry__` name, a `/tmp/quarry-bench` string, or a path under `repo_root` fails it, asserted as three separate cases.
- **Commit:** `feat(bench): add transcript-derived validation gates`

### Card 9: filesystem and daemon-state gates

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/fixtures/cold-native-fallback.jsonl`
  - `internal/cli/paths.go`
  - `internal/quarryengine/daemon/daemonstate.go`
  - `internal/quarryengine/daemon/ensureserver.go`
  - `internal/mcpserver/tools_toc.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add:
  - `workspace_key(target_dir)` — a Python re-derivation of quarry's own keying: the target directory's base name, a hyphen, then the first 12 hex characters of the SHA-256 digest of the cleaned absolute path. Mirrors `workspaceKey` in the Go source listed in Context.
  - `user_cache_dir()` — the base directory quarry's own default tier resolves to, with Go's `os.UserCacheDir` semantics reproduced explicitly: `$XDG_CACHE_HOME` when set and non-empty, otherwise `~/.cache`. This is the only place the production `cache_dir` argument is derived; no card passes a hand-written path.
  - `resolve_state_dir(target_dir, cache_dir)` — `<cache_dir>/quarry/<workspace_key>`, matching the third precedence tier of `ResolveStateDir`. The suite never passes `--state-dir` and always clears `QUARRY_STATE_DIR`, so the two higher tiers are deliberately not modelled. It also models **no** `tags-<hex>` segment, because the suite clears `QUARRY_BUILD_TAGS` and never sets `buildTags` on a call. Both of those are load-bearing rather than incidental: `ResolveStateDir` appends a `tags-<hex>` segment at every tier when the tag set is non-empty, and the MCP server resolves its tag set through `cli.ResolveBuildTags`, which falls back to `$QUARRY_BUILD_TAGS` — so an operator with that variable set would move every daemon state file one segment deeper, and every cold-cell gate would read an empty directory and declare a fatal native fallback that never happened. The function therefore takes the environment it is resolving **for** as an explicit `env` argument rather than reading `os.environ`, and raises `GateError` when that mapping carries either variable. In production the caller passes `run_ladder.run_env()`, which has already scrubbed both, so the raise never fires on a correct call — it exists to stop the gate being asked about an environment the runs were not launched with, which would silently resolve a key that is not the one in use. The harness's own process may legitimately have either variable exported; that is not what the gate resolves against.
  - `daemon_state_file(state_dir, lang)` — `<state_dir>/<lang>/daemon.json`, with `lang` defaulting to `go`.
  - `gate_cold_before(target_dir, cache_dir)` — fatal when the resolved `daemon.json` exists before a cold run starts, since the daemon is already warm and the run cannot be reported as cold.
  - `used_daemon_backed_tool(events)` — `True` when the transcript contains at least one `tool_use` block whose name is `mcp_name(t)` for a `t` in `DAEMON_BACKED_TOOLS`. `toc_file` and `toc_dir` are deliberately excluded: their handlers reach the tree-sitter path directly and never `EnsureServer`, so a toc call starts no daemon and writes no state.
  - `gate_cold_after(events, target_dir, cache_dir)` — three outcomes, not two. When `used_daemon_backed_tool` is `False`, the gate does not apply: it returns a single non-fatal `cold_no_daemon_backed_call` observation, and the run stands as valid while carrying no warmth signal. When it is `True` and a `daemon.json` exists, the gate passes — that is the positive confirmation the connection was supervised. When it is `True` and no `daemon.json` exists, the gate is fatal: the native fallback was taken, on which path the shared daemon address is not a function of the state directory at all, so the run is invalidated rather than reported as cold. Conflating the first and third outcomes would invalidate perfectly good runs and could report an environment limitation that does not exist.
  - `daemon_pid(target_dir, cache_dir, lang)` — the `pid` field of the resolved `daemon.json`, or `None` when the file is absent.
  - `wait_for_daemon_exit(target_dir, cache_dir, timeout_s, lang)` — poll that pid's liveness with `os.kill(pid, 0)` until the process is gone, returning immediately when `daemon_pid` is `None`, and raising `GateError` naming the pid and the bound when `timeout_s` elapses first. The pid is the liveness signal because neither `daemon.json` nor the state directory is removed on exit — only `daemon.sock` is — so file presence says nothing about whether the daemon is still running, while `daemon.json`'s recorded pid is exactly what quarry's own staleness check reads. Callers pass a bound derived from the daemon's own 10-minute idle timeout plus a margin.
  - `gate_worktree_neutralised(worktree)` — fatal when `CLAUDE.md`, `CONSTRAINTS.md`, or `.claude/` exists in the task worktree.
  - `observe_worktree_dirtied(worktree)` — returns a non-fatal finding carrying `True` or `False` from `git -C <worktree> status --porcelain` being non-empty. Recorded, never gated. It is called from inside `run_gates`, which runs before the worktree is restored; an observation taken after `restore_worktree` would be `False` for every run, since the restore is precisely what erases the evidence.

  Tests use `tmp_path` for both the worktree and the cache dir: `workspace_key` is deterministic and differs between two distinct paths sharing a basename; `resolve_state_dir` raises when it is passed an `env` mapping carrying `QUARRY_STATE_DIR` and, as a separate case, one carrying `QUARRY_BUILD_TAGS`, and resolves normally for a scrubbed mapping even while the ambient process environment has both set; `user_cache_dir` honours `$XDG_CACHE_HOME` when set and falls back to `~/.cache` when it is not, asserted with monkeypatch; `gate_cold_before` passes on an empty cache dir and fails once the `daemon.json` is created at the resolved location; `gate_worktree_neutralised` fails for each of the three ambient-context entries independently and passes when none is present; and `observe_worktree_dirtied` is non-fatal in both outcomes.

  `gate_cold_after` is asserted on all three of its outcomes, each as its own test: a transcript with a `workspace_symbol` call and no `daemon.json` is fatal; the same transcript with the `daemon.json` present passes; and the committed `cold-native-fallback.jsonl` fixture — whose calls are toc-only — is neither, returning the non-fatal `cold_no_daemon_backed_call` observation against that same empty state. `used_daemon_backed_tool` is asserted `False` for a toc-only transcript and `True` for one containing any of the five daemon-backed names. `wait_for_daemon_exit` returns immediately when no `daemon.json` exists, returns once a pid that was alive has exited, and raises `GateError` on a timeout against a pid that stays alive.
- **Commit:** `feat(bench): add worktree and cold-daemon state gates`

### Card 10: run-state, invalidation, and the composed gate report

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add:
  - `run_dir(results_root, config_id, n)` — `<results_root>/raw/<config_id>/<n>/`.
  - `is_complete(run_dir)` — `True` only when `run.json` exists in that directory **and** parses with `state == "complete"`. A directory holding `answer.json` and `usage.json` but no `score.json` is by construction not complete, because `run.json` is written last.
  - `invalidate(run_dir)` — delete `run.json`, then move the directory aside to a sibling `<n>.invalid-<k>/` with `k` the lowest unused index, leaving the discarded attempt inspectable. Returns the new path.
  - `write_run_json(run_dir, payload)` — write the terminal-state marker with `state: "complete"`, the config id, the repetition index, the resolved run model, the gate report's non-fatal observations, and the timestamp. Called only after the answer parsed, `usage.json` was extracted, every fatal gate passed, and `score.json` exists.
  - `run_gates(events, config, run_model, repo_root, worktree, run_dir, cache_dir)` — compose the card 8 and card 9 gates into one `GateReport`, applying the cold-cell gates only when `config.cold` is true. It runs **before** scoring, so it requires only `answer.json` and `usage.json` on disk and never asserts on a scoring artifact.
  - `gate_run_complete_artifacts(run_dir)` — a separate, later gate requiring all four of `answer.json`, `answer.redacted.json`, `usage.json`, and `score.json`. It is deliberately not part of `run_gates`: two of those files are written by the scorer, which runs after `run_gates`, so folding it in would make every run fail a fatal gate before the scorer had a chance to write them, and the matrix would halt on the third attempt of its first run. The caller invokes this one after scoring and immediately before `write_run_json`.

  Tests: a run directory with a `state: "complete"` `run.json` is skipped by `is_complete` while one without it is not; a directory whose `run.json` records any other state is not complete; a directory holding an answer and usage but no `score.json` is not complete; `invalidate` removes `run.json`, moves the directory to `<n>.invalid-1/` without destroying its contents, and a second invalidation of a re-created directory lands on `<n>.invalid-2/`; `run_gates` returns a failing report when any fatal gate fails and a passing report carrying the non-fatal observations when none does; and `run_gates` passes on a directory holding only `answer.json` and `usage.json` — the state that actually exists when it runs — while `gate_run_complete_artifacts` fails on that same directory and passes once the two scoring artifacts are added.
- **Commit:** `feat(bench): add run-state, invalidation, and the composed gate report`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_gates.py`, the only test file this batch creates. It covers all three cards' gates: the transcript gates against the committed fixtures, the filesystem and daemon-state gates against `tmp_path` fixtures, and the run-state helpers against synthesised run directories. No live model call, no network, and no real daemon is involved — the daemon gates are asserted against the state artifact's presence, which is the only source-grounded signal available without modifying quarry.
