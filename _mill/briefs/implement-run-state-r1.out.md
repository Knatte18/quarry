All 5 cards (22, 23, 24, 25, 26) committed, matching the batch's 5 declared cards exactly. 5 of 5 cards committed - all complete, verified via `go test ./bench/loomyard-eval/ladder/...` passing.

Summary of work:
- `bench/loomyard-eval/ladder/internal/ladder/worktree.go` / `worktree_test.go` - `RunGit`, `GateWorktreeNeutralised`, `ObserveWorktreeDirtied`
- `bench/loomyard-eval/ladder/internal/ladder/runstate.go` / `runstate_test.go` - `RunDirPath`, `IsComplete`, `WriteRunJSON`, `IngestRecord`/`NewIngestRecord`/`WriteIngestJSON`/`HasIngest`/`ReadIngestRecord`, `RunJSONPayload` (with the dead-lift repair to top-level boolean observation keys), `MaxAttempts`, `Invalidate`, `NextAttempt`, `RunPair`, `PendingRuns`, `PendingScoring`, `CheckSingleFlight`, `RunGates`, `GateRunCompleteArtifacts`

{"status":"success","commit_sha":"c9fc59856bb458736bb7665f8cbf67fdb65f7b1f","session_id":"50726992-c718-4c34-8568-7142c87bc071","cards_done":[22,23,24,25,26]}
