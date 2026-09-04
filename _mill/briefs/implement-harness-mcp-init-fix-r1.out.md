All 3 cards committed and verified: 3 of 3 cards committed, matching commit subjects exactly to the batch file's `Commit:` messages for cards 3, 4, and 5. `go test ./bench/loomyard-eval/ladder/...` passes, and the working tree has no uncommitted tracked changes.

{"status":"success","commit_sha":"cbcc7ab90459b879aba6ec67ee5383dd18c23b14","session_id":"c57ed069-873b-4ff5-9c45-46f23944a615","cards_done":[3,4,5]}

Summary of work (3 of 3 cards committed, all verified):

- **Card 3** (`0364aff`) — `bench/loomyard-eval/ladder/internal/ladder/stream.go`: added `MCPServerStatus{Name, Status}` and retyped `SessionInit.MCPServers` to `[]MCPServerStatus`. `provenance.go`: `SessionFingerprint` keeps `MCPServers []string` (names only, for `provenance.json` back-compat) and gains `MCPServerStatuses map[string]string`; `NewSessionFingerprint` and `diffSessionFingerprint` updated accordingly. New cases in `provenance_test.go`.
- **Card 4** (`881ca8a`) — added `testdata/transcripts/session-init-mcp-connected.jsonl`, a real 2.1.236 init record (from `.scratch/orch-evidence/init-line.json`) scrubbed of only `cwd`, `session_id`, `uuid`, `memory_paths.auto`, `messaging_socket_path`. New regression test `TestParseTranscript_SessionInitMCPServers` in `stream_test.go`. Converted `testdata/transcripts/tool-bytes.jsonl`'s init line from the old string-array `mcp_servers` shape to the object shape.
- **Card 5** (`cbcc7ab`) — `gates.go`: new `CheckServerConnected`. `runstate.go`: new `ServerConnectFailureFile` constant. `run.go`: wired into the attempt loop as a non-fatal, retried-then-InvalidateRep infrastructure failure, with a cross-repetition `abortRun` bound when every attempt failed this specific check, and the finding persisted into the attempt directory before the rename. `testdata/fakeclaude/main.go`: derives `mcp_servers` from the granted tool list and gains `FAKE_CLAUDE_SERVER_STATUS_OVERRIDE`. New table cases in `gates_test.go` and a full e2e case in `e2e_test.go` (`GrantedCellServerNeverConnects`).

Key files touched (all under `/home/knatte/Code/quarry/wts/ladder-toc-rerun/bench/loomyard-eval/ladder/internal/ladder/`): `stream.go`, `provenance.go`, `provenance_test.go`, `stream_test.go`, `gates.go`, `gates_test.go`, `run.go`, `runstate.go`, `e2e_test.go`, `testdata/fakeclaude/main.go`, `testdata/transcripts/tool-bytes.jsonl`, `testdata/transcripts/session-init-mcp-connected.jsonl` (new).

{"status":"success","commit_sha":"cbcc7ab90459b879aba6ec67ee5383dd18c23b14","session_id":"c57ed069-873b-4ff5-9c45-46f23944a615","cards_done":[3,4,5]}
