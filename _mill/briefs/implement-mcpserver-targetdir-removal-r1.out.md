All 5 cards are committed (matching the 5 declared `Commit:` messages), no uncommitted tracked changes, verify (`go build ./...` and `go test ./internal/mcpserver/...`) both pass.

Card-count self-check: 5 of 5 cards committed — all complete.

```json
{"status":"success","commit_sha":"011acc7c66d3e844e9afbc0da2ec3e3f9e823e35","session_id":"da3fbefc-0dab-41fa-abf0-545ef7a9997a","cards_done":[1,2,3,4,5]}
```

Summary of work:
- Card 1: `internal/mcpserver/transport_test.go` — extended `TestToolsList_PerToolParameterMatrix` and added `TestCallTool_TargetDirIsRejectedAsWholeCallError`.
- Card 2: `internal/mcpserver/callcontext.go`, `callcontext_test.go`, `tools_lsp.go`, `tools_symbol.go`, `tools_impact.go`, `tools_assert.go`, `tools_toc.go` — removed `TargetDir` input fields, deleted `effectiveTargetDir`, changed `resolveCall` signature, updated call sites and tests.
- Card 3: `internal/mcpserver/nativeentry.go`, `lspentry.go` — reworded `jsonschema` tag text.
- Card 4: `internal/mcpserver/mcpserver.go`, `callcontext.go`, `translate.go`, `tools_lsp.go`, `tools_impact.go`, `tools_assert.go`, `tools_toc.go`, `nativeentry.go`, `lspentry.go` — corrected all stale Go doc comments.
- Card 5: `internal/mcpserver/tools_toc_test.go` — added `TestTOCFileHandler_AbsoluteTargetResolvesOutsideLaunchRoot` and corrected two stale comments.

Verify: `go build ./...` and `go test ./internal/mcpserver/...` both pass (full suite green, no skipped or weakened tests).

```json
{"status":"success","commit_sha":"011acc7c66d3e844e9afbc0da2ec3e3f9e823e35","session_id":"da3fbefc-0dab-41fa-abf0-545ef7a9997a","cards_done":[1,2,3,4,5]}
```
