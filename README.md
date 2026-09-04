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
`go test` require `CGO_ENABLED=1` and a C toolchain. There is no command
to build yet; `go test ./...` runs the extractors' tests.
