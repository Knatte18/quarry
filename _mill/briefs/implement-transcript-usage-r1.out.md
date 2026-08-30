No tracked in-scope dirty files. All 5 cards (8, 9, 10, 11, 12) have matching commits in the log. Card-count check: 5 of 5 cards committed, all complete.

{"status":"success","commit_sha":"3a1bad9389c44d68b9d5498639c747a368acfcdd","session_id":"6fb400e0-0147-44db-8466-abf0847c3f21","cards_done":[8,9,10,11,12]}

All 5 of 5 cards (Card 8 through Card 12) declared in this batch were committed this turn, and `go test ./bench/loomyard-eval/ladder/...` passes.

Summary of work:
- `bench/loomyard-eval/ladder/internal/ladder/transcript.go` / `transcript_test.go` (new) — `Record`/`Message`/`ContentBlock`/`MessageUsage` subagent-transcript model, `ReadTranscript`, `IterToolUseBlocks`, `IterToolUses`, `AssistantRecords`.
- `bench/loomyard-eval/ladder/internal/ladder/testdata/*.jsonl` — the seven fixtures moved via `git mv` from `bench/loomyard-eval/ladder/tests/fixtures/` and re-shaped into the subagent transcript format, each preserving its distinguishing property.
- `bench/loomyard-eval/ladder/internal/ladder/usage.go` / `usage_test.go` — `Usage`/`TokenUsage`, `DenialShapePattern`, and `ExtractUsage(records, transcriptPath, transcriptSource, grantedTools)`.
- `_mill/plan/02-transcript-usage.md` — extended card 12's `Edits:` to name `go.mod`/`go.sum` before adding `github.com/google/go-cmp` as a direct dependency for the full-struct comparison test.
- `go.mod` — added `github.com/google/go-cmp v0.7.0` as a direct dependency.

`go vet ./...` and `gofmt -l` on all touched files are both clean.

{"status":"success","commit_sha":"3a1bad9389c44d68b9d5498639c747a368acfcdd","session_id":"6fb400e0-0147-44db-8466-abf0847c3f21","cards_done":[8,9,10,11,12]}
