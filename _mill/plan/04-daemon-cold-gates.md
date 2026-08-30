# Batch: daemon-cold-gates

```yaml
task: Port the capability-ladder bench harness to Go
batch: daemon-cold-gates
number: 4
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [3]
```

## Batch Scope

Ports the environment-dependent half of `gates.py`: the environment scrub, state-directory resolution,
the daemon state file and liveness helpers, state-directory clearing, the bounded daemon-exit wait, and
the two cold-cell gates. This is the batch the cold cell's entire argument rests on, because the cold
claim is only as strong as the state directory being a pure function of the worktree path.

The external interface later batches consume is `ScrubbedEnv`, `ResolveStateDir`, `DaemonAlive`,
`DaemonPID`, `ClearStateDir`, `WaitForDaemonExit`, `GateColdBefore`, and `GateColdAfter`.

Batch-local decision: every function here that resolves a state directory takes an explicit environment
slice rather than reading the process environment, so `ScrubbedEnv` is applied at an explicit call site
rather than implicitly. The harness no longer owns the process that spawns the server, so an implicit
scrub would have nowhere to apply.

## Cards

### Card 18: Environment scrub, workspace key, and state-directory resolution

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `internal/cli/paths.go`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `run_env` as `ScrubbedEnv() []string`, returning `os.Environ()` with
  `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` each forced to the empty string rather than removed, and
  `QUARRY_CONFIG` left untouched. Its doc comment must carry the Python's rationale: both scrubbed
  variables take precedence over the workspace key outright and a non-empty tag set appends a
  `tags-<hex>` segment at every tier, while `QUARRY_CONFIG` selects the overlay naming the
  language-server command and clearing it would stop the server starting at all. Port `workspace_key`
  as `WorkspaceKey`, `user_cache_dir` as `UserCacheDir` built on `os.UserCacheDir`, and
  `resolve_state_dir` as `ResolveStateDir(targetDir, cacheDir string, env []string) (string, error)`.
  `ResolveStateDir` hard-errors on a **non-empty** value for either scrubbed variable, not on key
  presence — a key set to the empty string passes, matching the Python's truthiness check and quarry's
  own treatment of an empty value as unset in `internal/cli/paths.go`. Test that resolution rejects a
  non-empty value, accepts an absent key, and accepts a key set to the empty string, and that
  `ScrubbedEnv` empties both keys while preserving `QUARRY_CONFIG`.
- **Commit:** `feat(ladder): port the environment scrub and state-directory resolution`

### Card 19: Daemon state file and liveness

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `daemon_state_file` as `DaemonStateFile`, `_pid_alive` as an unexported
  `pidAlive`, `_read_daemon_state` as `readDaemonState`, `daemon_alive` as `DaemonAlive`, and
  `daemon_pid` as `DaemonPID`, each keeping the `lang` parameter defaulting to `go` and the Python's
  behaviour when the state file is absent or unreadable. `pidAlive` must use a signal-zero style
  liveness probe rather than parsing a process listing. Record in `DaemonAlive`'s doc comment that
  liveness is what the cold gate keys on, because neither the state file nor the state directory is
  removed when a daemon exits. Test liveness against the current process and against a pid that cannot
  be alive, and reading a state file that is absent, malformed, and well-formed.
- **Commit:** `feat(ladder): port daemon state-file reading and liveness`

### Card 20: State-directory clearing and the bounded daemon-exit wait

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `clear_state_dir` as `ClearStateDir`, `_DAEMON_EXIT_POLL_INTERVAL_S` as an
  unexported `daemonExitPollInterval`, and `wait_for_daemon_exit` as
  `WaitForDaemonExit(targetDir, cacheDir string, env []string, timeout time.Duration, lang string) error`.
  Port `DAEMON_EXIT_TIMEOUT_S` as an exported `DaemonExitTimeout` constant holding the same 660-second
  value. Test that clearing removes the resolved directory and is not an error when it is already
  absent, and that the wait returns promptly when no daemon is alive and returns a timeout error when
  one stays alive past the deadline.
- **Commit:** `feat(ladder): port state-directory clearing and the daemon-exit wait`

### Card 21: Cold-before and cold-after gates

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/cold-native-fallback.jsonl`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `gate_cold_before` as
  `GateColdBefore(targetDir, cacheDir string, env []string) []GateFinding` and `gate_cold_after` as
  `GateColdAfter(records []Record, targetDir, cacheDir string, env []string) []GateFinding`, keeping
  every outcome the Python produces, including the non-fatal no-daemon-backed-call observation that the
  cold-cell disposition logic later reads. `GateColdBefore` keys on daemon liveness, never on the state
  file's existence — its doc comment must say why. Test the cold-before gate passing with no live
  daemon and failing with one, and the cold-after gate in each of its outcomes, using the reshaped
  cold-native-fallback fixture for the no-daemon-backed-call case.
- **Commit:** `feat(ladder): port the cold-before and cold-after gates`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `daemon_test.go` plus every earlier test file
in the ladder subtree. The liveness and wait tests use the running test process and a synthetic dead
pid rather than spawning a real daemon, matching the Python suite's own division.
