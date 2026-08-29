All 4 cards from batch 04 (quarry-native-lsp-tools) are committed: Card 17, 18, 19, and 20 — 4 of 4 cards complete. Verify (`go test ./internal/mcpserver/...`) passes, and the module-wide `go build ./...` gate also passes.

Files touched:
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/nativeentry.go` (new)
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/nativeentry_test.go` (new)
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_impact.go` (new)
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_impact_test.go` (new)
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_assert.go` (new)
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_assert_test.go` (new)
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/mcpserver.go` (edited, `NewServer` now registers `impact` and `assert_no_callers`)

{"status":"success","commit_sha":"42d17210fc26a67322ea9a3afc581c2633c7a484","session_id":"b33b3db3-3a54-4d5d-91af-ec8f0e7f12e5","cards_done":[17,18,19,20]}
