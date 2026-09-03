All 3 cards (25, 26, 27) committed and matched against the batch file's declared `## Cards`, batch verify (`go test ./bench/loomyard-eval/ladder/...`) passes. 3 of 3 cards committed — all complete.

{"status":"success","commit_sha":"05d98605d8bd38541a02f5b458415367a7bdb2b0","session_id":"30a1c6e8-9aba-4f36-ae6e-1e2613063a09","cards_done":[25,26,27]}

Summary of work: implemented batch 06-run-loop in full — 3 of 3 cards committed.

- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/runstate.go` (Card 25): `RepDir`, the six per-rep filename constants, `RunState`, `WriteRunState`/`ReadRunState`, `RepIsComplete`, `InvalidateRep`, `MaxAttempts`.
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/run.go` (Card 26): `RunOptions`/`Run` driving the sequential cell-minor loop (startup resolution, advisory lock, provenance merge/write, lazy server build, per-rep worktree/prompt/gate-d/dispatch/gate-2/answer/redaction/scoring/six-file-write/restore), `RunScorer` for the second measured-binary invocation, and the five-way failure disposition (infra/formatting retry to `MaxAttempts`, max-turns completion, fatal gate-2 discard, scorer-only retry, and a memory-path-taint abort-the-whole-run path modeled "exactly like a blinding failure").
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/runstate_test.go` (Card 27): covers `RepDir`'s path shape, a full `RunState` round-trip including `Observations`, all five `RepIsComplete` cases (with the blinding-failed-but-complete case that must return false), and `InvalidateRep`'s sequential `.invalid-<k>` naming against `MaxAttempts=3`.

`go build ./bench/loomyard-eval/ladder/...`, `go vet`, and `go test ./bench/loomyard-eval/ladder/...` all pass clean; no uncommitted tracked changes remain.

{"status":"success","commit_sha":"05d98605d8bd38541a02f5b458415367a7bdb2b0","session_id":"30a1c6e8-9aba-4f36-ae6e-1e2613063a09","cards_done":[25,26,27]}
