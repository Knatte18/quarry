# Batch: integration-and-docs

```yaml
task: "Add an MCP wrapper for quarry"
batch: "integration-and-docs"
number: 7
cards: 4
verify: go test ./internal/mcpserver/... && go test -tags lsp ./internal/mcpserver/...
depends-on: [6]
```

## Batch Scope

This batch closes the task: the tier-3 real-binary stdio test that is the only thing able to catch
a stray write corrupting the JSON-RPC frame stream, and the wiring and documentation that make the
server reachable — a committed `.mcp.json`, the `.gitignore` entry for the optional pre-built
binary, `docs/mcp-setup.md`, and the README pointer. It is one batch because every card depends on
the finished binary from batch 6 and on nothing else, and because the documentation cards describe
behaviour the tier-3 card exercises.

Batch-local decisions beyond `## Shared Decisions`:

- `.mcp.json` runs `go run ./cmd/quarry-mcp` with no `--target-dir` argument. The server takes its
  process working directory as the launch default and absolutises it at startup. Passing
  `--target-dir .` would resolve to the identical path but reads as if it pinned something, hiding
  that the value comes from the process working directory; omitting the flag makes the dependency
  visible, and the startup stderr line reports the resolved path either way.
- `.mcp.json` correctness is verified by dogfooding, not by a test.

## Cards

### Card 27: Add the tier-3 real-binary stdio test

- **Context:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/transport_test.go`
  - `cmd/quarry-mcp/main.go`
  - `internal/cli/refs_targetdir_lsp_test.go`
  - `internal/cli/assertnocallers_lsp_test.go`
  - `testdata/impactfixture/go.mod`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/stdio_lsp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/stdio_lsp_test.go` guarded by a `//go:build lsp`
  line and by `t.Skip` on `exec.LookPath("gopls")` carrying the same install hint
  `internal/cli/refs_targetdir_lsp_test.go` uses, matching the convention already established
  there. Its file comment states that this test exists to catch a stray log or cobra line
  corrupting the JSON-RPC stream, which an in-memory transport can never see, and that this is the
  risk that motivated the separate binary. The test builds `cmd/quarry-mcp` into a `t.TempDir()`
  with `go build -o`, then connects to it with `mcp.NewClient` over an `mcp.CommandTransport`
  pointing at the built binary with `--target-dir` set to the `testdata/impactfixture` tree, so no
  committed client configuration file is involved. Assert: the initialize handshake succeeds and
  `tools/list` returns the
  seven tools; at least one real `tools/call` against the fixture resolves through gopls and comes
  back `found`; one multi-entry call returns one entry per input in input order, proving array
  batching survives real serialization; and the binary's stdout carries only well-formed
  JSON-RPC frames and no other bytes. Use one mechanism for that last assertion, not a choice:
  spawn a second, separate child of the same built binary with `os/exec`, wired to explicit
  `StdinPipe`, `StdoutPipe`, and `StderrPipe`, write a hand-built `initialize` request followed by
  the `initialized` notification to its stdin as newline-delimited JSON, read every line it writes
  to stdout, and fail on any line that does not parse as a JSON object. `mcp.CommandTransport`
  cannot serve this assertion, because it owns the child's stdout pipe and its framed reader
  consumes it — the `CommandTransport` session is for the handshake, `tools/list`, and the two
  `tools/call` assertions only. That stdout-purity assertion is the reason this tier exists. The
  second child's stderr is expected to carry the startup target-directory line and must not be
  conflated with its stdout.
- **Commit:** `test(mcpserver): add the tier-3 real-binary stdio integration test`

### Card 28: Commit `.mcp.json` and ignore the optional built binary

- **Context:**
  - `cmd/quarry-mcp/main.go`
  - `README.md`
- **Edits:**
  - `.gitignore`
- **Creates:**
  - `.mcp.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `.mcp.json` at the repository root in the project-scope
  form: a single top-level `mcpServers` object whose one key is `quarry`, whose value sets
  `"command": "go"` and `"args": ["run", "./cmd/quarry-mcp"]`, with no `--target-dir` argument. The
  top-level key is `mcpServers` and not a bare server map: a project-scoped `.mcp.json` at a
  repository root always uses the `mcpServers` object. Pin it explicitly because a wrong top-level
  shape has no detector anywhere in this plan — `.mcp.json` correctness is verified by dogfooding
  rather than by a test. Add a plain `/quarry-mcp` line to `.gitignore` inside the existing
  hand-written block above the mill-managed section, with a one-line comment noting that it covers
  the optional `go build -o quarry-mcp ./cmd/quarry-mcp` warm-start alternative documented in
  the MCP setup document card 29 adds. No `!/quarry-mcp/` re-include is needed: the `/quarry` entry
  has one only
  because that name collides with the `quarry/` package directory, and `quarry-mcp` has no such
  collision.
- **Commit:** `chore: commit .mcp.json and ignore the built quarry-mcp binary`

### Card 29: Add `docs/mcp-setup.md`

- **Context:**
  - `.mcp.json`
  - `cmd/quarry-mcp/main.go`
  - `internal/quarryengine/cgoguard_nocgo.go`
  - `README.md`
- **Edits:** none
- **Creates:**
  - `docs/mcp-setup.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `docs/mcp-setup.md` covering four things. First, what the committed
  `.mcp.json` does: a Claude Code session opened in quarry connects to the server with no
  install step, after the one-time prompt Claude Code shows before trusting a project-scoped
  server, and the server takes its process working directory as the target-directory default,
  absolutising it once at startup and reporting the resolved path on stderr. Second, cold-start
  behaviour, stated as expected rather than as a bug: quarry requires `CGO_ENABLED=1` and a C
  toolchain because the tree-sitter package links C grammars, so the first `go run ./cmd/quarry-mcp`
  on a cold build cache is a cgo build that can exceed an MCP client's connect timeout — the first
  connect after a fresh clone or a cleared build cache may fail or hang, a retry once the build has
  completed succeeds, and every later launch is cache-fast. Third, the missing-toolchain failure
  mode: the build fails with the guard's own compile error naming
  `quarry_requires_CGO_ENABLED_1_with_a_C_toolchain` rather than a linker dump, which surfaces to
  the client as the server process exiting immediately with that message on stderr. Fourth, the
  warm-start alternative: `go build -o quarry-mcp ./cmd/quarry-mcp` and pointing the client at the
  built binary, with a note that `/quarry-mcp` is gitignored. Also list the four launch-only flags
  — `--target-dir`, `--config`, `--state-dir`, `--timeout` — and state that everything else is a
  tool parameter the model supplies per call.
- **Commit:** `docs: add MCP setup and cold-start notes`

### Card 30: Point the README at the MCP layer

- **Context:**
  - `docs/mcp-setup.md`
  - `.mcp.json`
- **Edits:**
  - `README.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a short section to `README.md` naming `cmd/quarry-mcp` as quarry's third
  exposure layer beside the engine and the CLI, giving its
  `go build -o quarry-mcp ./cmd/quarry-mcp` line beside the existing
  `go build -o quarry ./cmd/quarry` in the "Building and running" section, naming the seven tools,
  and linking to `docs/mcp-setup.md`. State that a Claude Code session opened in this repository
  connects through the committed `.mcp.json` once the one-time project-server trust prompt is
  accepted. Keep it short — the README currently
  documents the CLI as quarry's only entry point, and the defect being fixed is that the third
  layer is undiscoverable from the front door, not that the README lacks detail.
- **Commit:** `docs(readme): add a pointer to the MCP layer`

## Batch Tests

`verify:` runs `go test ./internal/mcpserver/...` untagged, then chains
`go test -tags lsp ./internal/mcpserver/...` so `stdio_lsp_test.go` is compiled. The tagged run
`t.Skip`s because `gopls` is not on `$PATH` here, so what it proves in this environment is that the
tier-3 test compiles against the finished binary and the SDK's client API — which is the failure
this batch could otherwise introduce silently. On a machine with `gopls` installed, the same
command runs the handshake, the real `tools/call`, the multi-entry call, and the stdout-purity
assertion for real. Cards 28, 29, and 30 have no runnable surface: `.mcp.json` correctness is
verified by dogfooding, and the two documentation files are prose.
