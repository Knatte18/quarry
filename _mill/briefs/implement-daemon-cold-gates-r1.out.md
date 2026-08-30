All 4 cards (18-21) of the daemon-cold-gates batch are complete: 4 of 4 cards committed (combined into a single commit since all four touch the same two files, `bench/loomyard-eval/ladder/internal/ladder/daemon.go` and `daemon_test.go`, per the plan's combined-commit allowance). Verify (`go test ./bench/loomyard-eval/ladder/...`) passes, `go vet ./bench/loomyard-eval/ladder/...` is clean, gofmt/goimports report no drift, and the working tree has no uncommitted tracked changes.

Key files:
- `bench/loomyard-eval/ladder/internal/ladder/daemon.go` - `ScrubbedEnv`, `WorkspaceKey`, `UserCacheDir`, `ResolveStateDir`, `DaemonStateFile`, `pidAlive`, `readDaemonState`, `DaemonAlive`, `DaemonPID`, `ClearStateDir`, `WaitForDaemonExit`, `DaemonExitTimeout`, `GateColdBefore`, `GateColdAfter`.
- `bench/loomyard-eval/ladder/internal/ladder/daemon_test.go` - table/subtest coverage for every one of the above, using `testdata/cold-native-fallback.jsonl` for the no-daemon-backed-call case and `testdata/bundle-mixed-tools.jsonl` for the daemon-backed cases.

Notable design decision (not literally specified by the card text, so flagging it): since `GateColdBefore`/`GateColdAfter`/`DaemonAlive`/`DaemonPID` have no error return in their card-specified signatures, but `ResolveStateDir` and JSON parsing can fail, `readDaemonState` panics on a `GateError` or malformed `daemon.json`, mirroring the Python port's own uncaught-exception propagation through `_read_daemon_state`'s callers. This is documented in the doc comments.

{"status":"success","commit_sha":"98a1466942c32778fd02dfca397bce7e8e23c490","session_id":"7de998b4-871c-41c1-aaa4-bc3bca17ef28","cards_done":[18,19,20,21]}
