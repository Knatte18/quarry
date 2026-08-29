All 4 cards (27, 28, 29, 30) are committed and match the batch's declared `Commit:` messages exactly. Card-count self-check: 4 of 4 cards committed — all complete.

{"status":"success","commit_sha":"def1940b7a3e9e09e9b524e13ef4df15b47d88d0","session_id":"a3abd676-5d17-4237-a7a4-40ec6bc9231b","cards_done":[27,28,29,30]}

Summary of work done (all 4 of 4 cards committed):

- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/internal/mcpserver/stdio_lsp_test.go` — new tier-3 `//go:build lsp` test that builds `cmd/quarry-mcp` into a real binary, drives an `mcp.CommandTransport` session for handshake/tools-list/single-entry/multi-entry `textDocument_definition` calls against `testdata/impactfixture`, and spawns a second raw-pipe child process to assert every stdout line is well-formed JSON-RPC (commit `00452e6`).
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/.mcp.json` (new) and `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/.gitignore` (edited) — project-scoped MCP server declaration running `go run ./cmd/quarry-mcp` with no `--target-dir`, plus a `/quarry-mcp` ignore entry (commit `e1f2f4b`).
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/docs/mcp-setup.md` (new) — covers `.mcp.json` behaviour, cgo cold-start timeout risk, the missing-toolchain compile-error identifier, the warm-start binary alternative, and the four launch-only flags (commit `0a213f7`).
- `/home/knatte/Code/quarry/wts/quarry-mcp-wrapper/README.md` (edited) — new "Third exposure layer: MCP" section naming `cmd/quarry-mcp`, its build line, the seven tools, and a link to `docs/mcp-setup.md` (commit `def1940`).

Verify ran successfully: `go test ./internal/mcpserver/...` and `go test -tags lsp ./internal/mcpserver/...` both pass (tagged run skips live assertions since `gopls` is absent from `$PATH` in this environment, but compiles clean against the real binary and SDK client API). Working tree is clean of tracked in-scope modifications; the only untracked file is the brief itself, which is outside this batch's scope.
