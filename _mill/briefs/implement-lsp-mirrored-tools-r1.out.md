Verify passes, no uncommitted tracked changes, all 4 commits match the batch's 4 cards' `Commit:` messages exactly. 4 of 4 cards committed — all complete.

{"status":"success","commit_sha":"14a32eb0bcc16fe817f925d6bf19c09a3f2df087","session_id":"0a5ccb39-795c-4965-ad8a-d107a8345a95","cards_done":[13,14,15,16]}

Summary: All 4 cards in batch 03-lsp-mirrored-tools were completed and committed (4 of 4 cards committed — verified against the batch file's `## Cards` section and the commit log since batch-start commit `6d17967`):

- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/lspentry.go` and `lspentry_test.go` (card 13) — LSP entry parsing (`lspEntry`, `textDocumentIdentifier`, `lspPosition`), the three-form query union, and `runTargets`. Fixed a stack-overflow bug during implementation: the plan's "type alias lspEntry" wording had to be a Go *defined type* (`type lspEntryAlias lspEntry`, no `=`), not a true type alias, or `UnmarshalJSON` recurses infinitely via method promotion.
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_lsp.go` (card 14) — `textDocument_definition` and `textDocument_references`, sharing `resolveLSPEntry`.
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_symbol.go` (card 15) — `workspace_symbol`.
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/tools_lsp_test.go` (card 16) — handler-level tests via facade-seam stubs.
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/mcpserver.go` — edited to register all three tools from `NewServer`.

`go test ./internal/mcpserver/...` passes; working tree is clean.

{"status":"success","commit_sha":"14a32eb0bcc16fe817f925d6bf19c09a3f2df087","session_id":"0a5ccb39-795c-4965-ad8a-d107a8345a95","cards_done":[13,14,15,16]}
