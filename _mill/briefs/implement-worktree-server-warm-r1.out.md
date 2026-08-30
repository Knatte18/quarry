{"status":"success","commit_sha":"3b0b5021e5edc418dcdacf7a44e129431ba243d4","session_id":"d5f362dc-b406-4453-8b44-fd6360bc0f09","cards_done":[35,36,37,38]}

Summary of work (all 4 cards committed, matches the batch's declared total of 4):

- `bench/loomyard-eval/ladder/internal/ladder/worktree.go` - added `HarnessError`, `GitRunner`, `NeutraliseWorktree`, `BuildWorktree`, `RestoreWorktree`, `RemoveWorktree`, `EnsureTaskWorktrees` (card 35, commit `c8e513c`)
- `bench/loomyard-eval/ladder/internal/ladder/worktree_test.go` - matching tests
- `bench/loomyard-eval/ladder/internal/ladder/server.go` (new) - `Builder`, `BuildServer`, `MCPConfigDocument` (cards 36+37, combined commit `5758bb8`)
- `bench/loomyard-eval/ladder/internal/ladder/server_test.go` (new) - matching tests
- `bench/loomyard-eval/ladder/internal/ladder/warm.go` (new) - `WarmUpTool`, `warmUpTimeout`, `Warm` built on `github.com/modelcontextprotocol/go-sdk/mcp` (card 38, commit `3b0b502`)
- `bench/loomyard-eval/ladder/internal/ladder/warm_test.go` (new) - tests the post-condition failure path by re-execing the test binary itself as a fake MCP stdio server via `TestMain` + an env-var switch, so no real quarry-mcp binary is spawned

`go test ./bench/loomyard-eval/ladder/...` and `go vet ./bench/loomyard-eval/ladder/...` both pass; `gofmt -l` reports no drift; working tree is clean; all commits pushed to `port-ladder-bench-to-go`.
