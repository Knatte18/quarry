# quarry

Quarry is being rewritten around one identifier — the glyph — and three
queries: `toc`, `resolve`, `expand`. Extraction is tree-sitter only, and
Go only.

The rewrite is specified in two documents:

- [`docs/rewrite-plan.md`](docs/rewrite-plan.md) — what is deleted, what is
  built, in what order.
- [`docs/glyph.md`](docs/glyph.md) — the identifier contract.

Version 1 — the seven-verb CLI, the MCP server and the language-server
layer — is frozen on branch `v1-final`. Nothing is merged from it.

## Building

The tree-sitter backend is a cgo binding, so `go build`, `go run` and
`go test` require `CGO_ENABLED=1` and a C toolchain. `go build ./cmd/quarry`
and `go build ./cmd/quarry-mcp` build the CLI and the MCP server; `go test
./...` runs the extractors' tests.

## MCP server

`cmd/quarry-mcp` is a stdio MCP server over the same engine the `quarry`
command-line tool uses. It exposes exactly one tool, `toc`, which answers a
table-of-contents query for a directory or a file.

The repository ships a ready-to-use configuration at `.mcp.json`, so a
client that reads project-scope MCP configuration needs no setup at all.
That configuration invokes the server with `go run ./cmd/quarry-mcp`, which
recompiles on every server start; because this binary links tree-sitter
through cgo, the first start after a change takes seconds rather than
milliseconds.

For a faster local start, build the binary once:

```
go build -o quarry-mcp ./cmd/quarry-mcp
```

and point the configuration at it instead of `go run`:

```json
"command": "./quarry-mcp"
```
