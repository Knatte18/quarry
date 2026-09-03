3 of 3 cards committed (cards 8, 9, 10), matching the batch's declared card count of 3. Verify passed. No uncommitted tracked changes.

{"status":"success","commit_sha":"980ed238de273e4bf228910233b789f556e48b1d","session_id":"5425f920-16ea-4f41-a8a9-bd03a44f685f","cards_done":[8,9,10]}

Summary: implemented all 3 cards of batch `02-stream-and-metrics`:

- `bench/loomyard-eval/ladder/internal/ladder/stream.go` — record types (`Record`, `Message`, `MessageUsage`, `ContentBlock`, `SessionInit`, `ResultRecord`) and `ParseTranscript` (8 MiB scanner buffer, skips unmodelled types, errors naming the line number on truly malformed JSON) plus `Transcript.MarshalAll`.
- `bench/loomyard-eval/ladder/internal/ladder/metrics.go` — `Metrics` struct and `ComputeMetrics`, porting V1's `assistantCallGroups`/`perCallUsage` grouping rule and the leading-word grep regex verbatim from `origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/usage.go`, plus the new byte metrics (`tool_result_bytes`, `tool_result_bytes_breakdown`, `read_bytes`).
- `bench/loomyard-eval/ladder/internal/ladder/stream_test.go`, `metrics_test.go`, and three fixtures under `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/` (`grouped-usage.jsonl`, `max-turns.jsonl`, `tool-bytes.jsonl`).

`go test ./bench/loomyard-eval/ladder/...` passes. `go vet` clean, `gofmt -l` clean. No files outside the batch's declared `Edits:`/`Creates:` were touched.

{"status":"success","commit_sha":"980ed238de273e4bf228910233b789f556e48b1d","session_id":"5425f920-16ea-4f41-a8a9-bd03a44f685f","cards_done":[8,9,10]}
