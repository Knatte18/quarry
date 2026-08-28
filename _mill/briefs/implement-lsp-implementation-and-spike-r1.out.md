4 of 4 cards committed, matching the batch's declared card count exactly. Verify (`go vet -tags lsp ./...` and `go test ./internal/quarryengine/lsp/`) passed. No uncommitted tracked changes remain.

{"status":"success","commit_sha":"ed14849afa97dbffa3d529bbd7af8941c8e3eece","session_id":"bebd2108-1868-41a6-9edd-19b895196de4","cards_done":[1,2,3,4]}

Summary: all 4 of 4 cards in batch `01-lsp-implementation-and-spike` are committed and pushed to `gopls-query-precision`.

- `internal/quarryengine/lsp/lspclient.go`, `internal/quarryengine/lsp/lspclient_test.go` (commit `0d3b6df`) — added `ImplementationProvider` capability, `SupportsImplementation()`, and `Implementation()` (reusing `parseDefinitionResult`), plus fake-server tests.
- `testdata/clockfixture/{go.mod,builder/poll.go,runner/tick.go,sched/wait.go}` (commit `3506e6a`) — self-contained fixture module with three structurally-identical, unrelated `clock` interfaces.
- `internal/quarryengine/query/implementation_spike_lsp_test.go` (commit `253231b`) — `TestImplementationWidening_Spike`, a `//go:build lsp` measurement harness logging definition/implementation/references results plus per-reference definitions at the fixture's two query positions.
- `docs/implementation-widening-spike.md` (commit `ed14849`) — spike results run against pinned gopls v0.23.0; recorded `mode: directional`, `references-unfiltered: 7`, `references-verified: 7`, `callers-verified: 6`.

Verify command (`go vet -tags lsp ./... && go test ./internal/quarryengine/lsp/`) passed cleanly; `gofmt -l` showed no drift across the repo (excluding the ignored `testdata/` tree, which itself is also clean).

{"status":"success","commit_sha":"ed14849afa97dbffa3d529bbd7af8941c8e3eece","session_id":"bebd2108-1868-41a6-9edd-19b895196de4","cards_done":[1,2,3,4]}
