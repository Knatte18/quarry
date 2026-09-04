# Plan: MCP, thin (T6)

```yaml
task: "MCP, thin (T6)"
slug: "mcp-thin"
approved: false
started: "20260904-085038"
parent: "main"
root: ""
verify: go vet ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: repopath-extraction
    file: 01-repopath-extraction.md
    depends-on: []
    verify: go test ./internal/repopath/... ./internal/cli/...
  - number: 2
    name: mcp-server
    file: 02-mcp-server.md
    depends-on: [1]
    verify: go build ./... && go test ./internal/mcpserver/...
  - number: 3
    name: mcp-server-tests
    file: 03-mcp-server-tests.md
    depends-on: [2]
    verify: go test ./internal/mcpserver/...
  - number: 4
    name: docs-and-config
    file: 04-docs-and-config.md
    depends-on: [2]
    verify: go build ./cmd/quarry-mcp
```

## Shared Decisions

### Decision: the facade is the only engine dependency

- **Decision:** no file under `internal/mcpserver/` or `cmd/quarry-mcp/` — production or `_test.go` —
  may import `github.com/Knatte18/quarry/internal/engine` or any package below it. Every engine type,
  constant and sentinel reaches these packages through `github.com/Knatte18/quarry/quarry`.
  `internal/repopath` obeys the same rule (it imports `quarry` for `ErrTargetOutsideRepo` only).
  A layering test in batch 2 enforces it mechanically for both MCP packages.
- **Rationale:** discussion D10 and the task body state it as a hard constraint; `internal/cli`
  already obeys it, and review alone is not a mechanism.
- **Applies to:** all batches

### Decision: stdout belongs entirely to the MCP transport

- **Decision:** nothing in `internal/mcpserver` or `cmd/quarry-mcp` writes to `os.Stdout`. Startup
  progress and fatal startup errors go to `os.Stderr`, and a fatal startup error exits non-zero.
- **Rationale:** discussion D12. A stray stdout write corrupts the framed JSON-RPC stream, which is
  the one way this binary fails catastrophically and silently.
- **Applies to:** all batches

### Decision: the MCP surface says exactly what the CLI says

- **Decision:** success bytes are `quarry.RenderJSON(answer)` verbatim; failure bytes are
  `quarry.RenderErrorJSON(msg)` verbatim, with `isError` set. Failure wording reuses the CLI's own
  sentences where the condition is the same (`target outside repository: <target as given>`,
  `target not found: <repository-relative path>`, `internal error: <err>`). The one deliberate
  divergence is the `depth` rejection message, because `-1` is valid on this surface and `"all"` does
  not exist on it.
- **Rationale:** discussion D7, D8. "Mirror of the CLI" has to mean the same bytes, not a paraphrase.
- **Applies to:** mcp-server, mcp-server-tests

### Decision: no `outputSchema`, therefore no `structuredContent`

- **Decision:** the `toc` tool is registered through `mcp.AddTool[tocInput, any]` with an explicit
  `InputSchema` and a nil `OutputSchema`, and the handler returns a nil `Out` value. The go-sdk emits
  `structuredContent` for any tool that declares an `outputSchema`, so omitting it is an
  implementation requirement, not a default. Two assertions guard it: the `tools/list` test asserts
  the tool carries no `outputSchema`, and a happy-path test asserts the call result carries no
  `structuredContent`.
- **Rationale:** discussion D7 and its "Implementation consequence" paragraph. Note the token-cost
  argument for this decision is refuted by `docs/research/mcp-surface.md` and must not be repeated:
  the decision stands on payload-shape simplicity alone.
- **Applies to:** mcp-server, mcp-server-tests

### Decision: the tool's prose is fixed, not authored

- **Decision:** the `toc` tool description and its three property descriptions are the exact ASCII
  strings quoted in card 8, reproduced verbatim as Go string constants and pinned by exact-string
  assertions in the `tools/list` test. The implementer does not reword, reflow, or "improve" them.
- **Rationale:** discussion D5a. That prose is the granted ladder cell's entire prompt cost and the
  thing that decides whether the agent calls the tool at all — it is T7's independent variable, so it
  belongs in the reviewed artefact rather than in implementation judgement.
- **Applies to:** mcp-server, mcp-server-tests

### Decision: tests never touch the system temp directory

- **Decision:** no test added or moved by this plan calls `t.TempDir()`, `t.Chdir`, `os.Chdir`, or
  writes anywhere under `/tmp`. Fixture trees are built with a per-package `writeScratchTree` copy
  writing under the repository's gitignored `.scratch/`; committed fixture trees live under the
  package's own `testdata/`.
- **Rationale:** the repo's standing test rule, restated in discussion D11. `internal/cli`,
  `internal/engine` and `quarry` each already carry their own copy of the helper for exactly this
  reason — Go test helpers are not importable across packages, so a per-package copy is the
  documented convention here, not duplication to be avoided.
- **Applies to:** repopath-extraction, mcp-server-tests

### Decision: renames go through `git mv`

- **Decision:** every `Moves:` pair in this plan is performed with `git mv <old> <new>` first,
  followed only by surgical edits (package clause, identifier exports, import lines). No moved file
  is rewritten from scratch.
- **Rationale:** it preserves git rename history and keeps the review diff to the lines that actually
  changed — which is what makes the "no observable CLI behaviour change" claim reviewable.
- **Applies to:** repopath-extraction

### Decision: the ladder harness is read-only input to this task

- **Decision:** nothing under `bench/loomyard-eval/ladder/` is edited by any batch. In particular
  `ladder-toc.yaml` keeps its empty `server.args`. The names `quarry` (server) and `toc` (tool) are
  external contracts fixed by that file and by `MCPPrefix()`, not choices this plan makes.
- **Rationale:** discussion Scope Out and the Constraints section. Any other spelling means the
  granted cell's tool is never allowed and T7 measures nothing.
- **Applies to:** all batches

### Decision: the repo-wide gate runs once, at the end, not at every batch boundary

- **Decision:** no batch's `verify:` and not the module-wide `verify:` runs the discussion's stated
  gate `go test ./... && golangci-lint run`. Per-batch commands are package-scoped, the module-wide
  command is `go vet ./...`, and the full gate runs once at the end of the task — it is already
  configured as this hub's `pipeline.done_gate`, which mill-go runs from the repository root before
  marking the task done, so a regression in a package no batch scoped to is caught there.
- **Rationale:** this repository's suite links tree-sitter through cgo and includes the engine's whole
  extraction corpus; running it plus the linter after every implementer round and every fixer round —
  which is what a batch `verify:` means — would cost minutes per round for a task whose own code lives
  in three packages. `go vet ./...` at each batch boundary is the cheap cross-package compile check
  that catches what package-scoped tests cannot, which is the module-wide `verify:` field's stated
  purpose.
- **Applies to:** all batches

### Decision: the live §9a probe is out of `go test`

- **Decision:** the automated protocol tests in batch 3 are the whole of this plan's runnable gate.
  The live `claude -p` probe (discussion D13 item 2, D14) is operator work performed after this plan
  merges; it is not a card, not a test, and not part of `pipeline.done_gate`.
- **Rationale:** a `go test` that shells out to `claude -p` costs money on every gate run, needs
  network and a logged-in CLI, and would make `go test ./...` unrunnable as the repo's gate.
- **Applies to:** all batches

## All Files Touched

- `.gitignore`
- `.mcp.json`
- `README.md`
- `cmd/quarry-mcp/main.go`
- `go.mod`
- `go.sum`
- `internal/cli/cli.go`
- `internal/cli/message_test.go`
- `internal/mcpserver/doc.go`
- `internal/mcpserver/fixture_test.go`
- `internal/mcpserver/layering_test.go`
- `internal/mcpserver/mcpserver.go`
- `internal/mcpserver/root.go`
- `internal/mcpserver/root_test.go`
- `internal/mcpserver/testdata/golden/toc-dir-depth-all.json`
- `internal/mcpserver/testdata/golden/toc-dir-depth1.json`
- `internal/mcpserver/testdata/golden/toc-dir-symbols-true.json`
- `internal/mcpserver/testdata/golden/toc-dir.json`
- `internal/mcpserver/testdata/golden/toc-file-symbols-false.json`
- `internal/mcpserver/testdata/golden/toc-file.json`
- `internal/mcpserver/testdata/repo/alpha/alpha.go`
- `internal/mcpserver/testdata/repo/alpha/doc.go`
- `internal/mcpserver/testdata/repo/alpha/sub/leaf.go`
- `internal/mcpserver/toc.go`
- `internal/mcpserver/toc_defaults_test.go`
- `internal/mcpserver/toc_depth_test.go`
- `internal/mcpserver/toc_errors_test.go`
- `internal/mcpserver/toc_golden_test.go`
- `internal/mcpserver/tools_test.go`
- `internal/repopath/doc.go`
- `internal/repopath/root.go`
- `internal/repopath/root_test.go`
- `internal/repopath/scratchtree_test.go`
- `internal/repopath/target.go`
- `internal/repopath/target_test.go`
