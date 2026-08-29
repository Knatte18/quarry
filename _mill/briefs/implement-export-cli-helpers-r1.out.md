All 5 of 5 declared cards are committed, matching the 5 `Commit:` messages in the batch file exactly. Verify (all three sub-invocations) passed. No cards used `Commit: none`.

{"status":"success","commit_sha":"010187c71bb72765c1f47c59be7dfa7c17470694","session_id":"75982287-6523-41a9-98ac-05d429040078","cards_done":[1,2,3,4,5]}

Summary: 5 of 5 cards committed for batch `01-export-cli-helpers`. All thirteen `internal/cli` helpers named in the batch scope were exported via mechanical rename-only edits (identifier + call sites + comment prose + test names/messages), with `isWithinDir`, `loadTOCConfig`, `resolveTOCConfigPath`, `resolveTOCBaseDir`, `workspaceKey`, `buildTagsSegment`, and `classifyTOCError` deliberately left unexported per the batch's local decision. Files touched:

- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/paths.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/cli.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/impact.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/toc.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/tocconfig.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/paths_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/resolve_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/cli_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/assertnocallers_lsp_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/impact_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/tocconfig_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/cli/toc_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/quarryengine/query/callers.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/quarryengine/query/callers_test.go`
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/quarryengine/impact/impact.go`

`go build ./...`, `go test ./internal/cli/...`, `go test -tags lsp ./internal/cli/...`, and `go test ./internal/quarryengine/query/...` all pass. Working tree has no dirty tracked files (only the pre-existing untracked brief file `_mill/briefs/implement-export-cli-helpers-r1.md`, outside this batch's scope).

{"status":"success","commit_sha":"010187c71bb72765c1f47c59be7dfa7c17470694","session_id":"75982287-6523-41a9-98ac-05d429040078","cards_done":[1,2,3,4,5]}
