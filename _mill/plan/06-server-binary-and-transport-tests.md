# Batch: server-binary-and-transport-tests

```yaml
task: "Add an MCP wrapper for quarry"
batch: "server-binary-and-transport-tests"
number: 6
cards: 3
verify: go test ./internal/mcpserver/... ./cmd/...
depends-on: [5]
```

## Batch Scope

This batch turns the seven registered tools into a runnable binary and proves the whole surface
over a real MCP session. It is one batch because the tier-2 tests exercise every tool at once
through one client — tool listing, the per-tool parameter matrix, array batching, the whole-call
versus per-entry split, and the `resolution` marker's presence and absence are all cross-tool
assertions that cannot be split without duplicating the client harness.

The external interface batch 7 consumes: the built `cmd/quarry-mcp` binary and its flag set.

Batch-local decisions beyond `## Shared Decisions`:

- `cmd/quarry-mcp/main.go` parses flags with the standard library `flag` package, never with
  `cobra`. The stdout-purity constraint is enforced by never constructing or running a
  `cobra.Command` in this process, not by cobra being absent from the binary — `internal/mcpserver`
  imports `internal/cli`, so cobra and `internal/output` are linked in either way.
- The server holds no global lock, so concurrent `tools/call` requests proceed independently. The
  single-flight hazard in the engine's client is per-client, and no client is cached in-process, so
  two independent calls cannot trigger it. A process-wide mutex would give unbounded head-of-line
  blocking for a 64-entry call with a per-entry timeout and no whole-call deadline.

## Cards

### Card 24: Add the `cmd/quarry-mcp` binary

- **Context:**
  - `cmd/quarry/main.go`
  - `internal/mcpserver/mcpserver.go`
  - `internal/cli/cli.go`
- **Edits:** none
- **Creates:**
  - `cmd/quarry-mcp/main.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `cmd/quarry-mcp/main.go` as a thin entry point holding no resolution or
  tool logic of its own. Its file comment states that stdio MCP cannot tolerate anything else
  writing to stdout, which is why this is a separate binary from `cmd/quarry`, and that the
  guarantee it provides is that no CLI command ever runs in this process — no `cobra.Command` is
  constructed or executed, so no `output.Ok` or `output.Err` call site is reachable. Parse four
  launch-only flags with the standard library `flag` package: `--target-dir` (default empty),
  `--config` (default empty), `--state-dir` (default empty), and `--timeout` (a
  `time.Duration`, default `30 * time.Second`, matching the CLI's own default). Call
  `mcpserver.ResolveLaunchTargetDir` on the `--target-dir` value; on error, write it to
  `os.Stderr` and exit non-zero rather than starting. On success, write one line to `os.Stderr`
  naming the resolved absolute target directory. Build the `mcpserver.Config`, call
  `mcpserver.NewServer`, and run it with `server.Run(context.Background(), &mcp.StdioTransport{})`,
  writing any error to `os.Stderr` and exiting non-zero. Nothing in this file writes to
  `os.Stdout`.
- **Commit:** `feat(quarry-mcp): add the stdio MCP server binary`

### Card 25: Tier-2 transport tests for listing, schemas, and batching

- **Context:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/schema.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/transport_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/transport_test.go` wiring an `mcp.Client` and the
  server from `NewServer` over `mcp.NewInMemoryTransports()`, connecting the server before the
  client, with the facade seam variables replaced by stubs so no gopls is required and
  `Config.StateDir` pinned to a `t.TempDir()`. Provide one helper that builds the connected pair so
  each test does not repeat the wiring. Assert: `tools/list` returns exactly the seven tools named
  `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `toc_file`, `toc_dir`,
  `impact`, and `assert_no_callers`, each carrying both an input and an output schema; the per-tool
  parameter matrix holds, by inspecting each listed tool's input schema properties —
  `workspace_symbol` declares no `within` on its entries, neither toc tool declares `buildTags`,
  `toc_dir` declares no `docSentences`, `impact`'s entry schema declares no `except` while
  `assert_no_callers`'s does, and `noVerify` appears on `assert_no_callers` alone; a
  malformed **call** is rejected before any handler runs and comes back with the result's error
  flag set — assert this for a zero-length `targets` array, a 65-entry `targets` array, and a
  `targets` value of the wrong JSON type. Assert only the observable contract for these — the
  result's error flag set and no handler run, checked by a facade stub that records whether it
  was called — never the validation message's wording, which the SDK's schema validator owns
  and this plan neither generates nor specifies. What matters is that the call is rejected
  whole rather than silently truncated to the cap or returned as an empty `results` array; a malformed **entry** instead yields that entry's
  `status: "error"` with every other entry's result intact — assert both in the same test file,
  since conflating the two is the failure mode; a multi-entry call returns one entry per input in
  input order; every entry carries `target` and `status`; a mixed batch validates against the
  tool's declared output schema without any entry being rejected; and `structuredContent` and the
  text content block carry the same payload.
- **Commit:** `test(mcpserver): add tier-2 transport tests for listing, schemas, and batching`

### Card 26: Tier-2 transport tests for error mapping and call isolation

- **Context:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/transport_test.go`
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/callcontext.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/transport_errors_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/transport_errors_test.go` reusing the connected-pair
  helper from `internal/mcpserver/transport_test.go`. Assert: a mixed call — one resolvable target,
  one not-found, one ambiguous — comes back with the result's error flag unset and three distinct
  per-entry statuses with the good result intact, which is the regression that matters most under
  array batching; `resolution: "complete"` is present on `found` entries of
  `textDocument_definition`, `textDocument_references`, and `impact` and absent from
  `workspace_symbol`, `assert_no_callers`, `toc_file`, and `toc_dir` — assert both halves, since
  the failure mode is adding it everywhere; the toc whole-call split holds, by pointing
  `Config.ConfigPath` at a malformed `servers.yaml` fixture and asserting an LSP-backed tool fails
  the whole call while `toc_file` and `toc_dir` still succeed, and separately that an invalid
  `lang` or an invalid `docSentences` fails a toc call wholly; an `assert_no_callers` entry with a
  relative `except` path exempts the intended file, which silently never matches when the path is
  resolved against the process working directory; two concurrent `tools/call` requests both
  complete correctly and neither blocks the other, proving no global mutex; and the `targetDir` a
  handler uses is absolute even when the launch default came from a relative process working
  directory, asserted by observing the `quarry.Options.TargetDir` a stub receives.
- **Commit:** `test(mcpserver): add tier-2 transport tests for error mapping and call isolation`

## Batch Tests

`verify: go test ./internal/mcpserver/... ./cmd/...` runs the two new tier-2 test files plus every
earlier test in `internal/mcpserver`, and compiles both `cmd/quarry` and `cmd/quarry-mcp`
(`./cmd/...` holds no test files, so its value here is the build). Scope covers exactly the
packages this batch touches. Tier 2 is where the schema and envelope contract is proven, because
an in-memory transport exercises the SDK's real validation and serialization path while the facade
seam keeps every assertion reachable without gopls; what it cannot see — a stray write corrupting
the stdout frame stream — is what batch 7's tier-3 test exists for.
