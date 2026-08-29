# Plan: Add an MCP wrapper for quarry

```yaml
task: "Add an MCP wrapper for quarry"
slug: "quarry-mcp-wrapper"
approved: true
started: "20260829-065134"
parent: "main"
root: ""
verify: go build ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: export-cli-helpers
    file: 01-export-cli-helpers.md
    depends-on: []
    verify: go test ./internal/cli/... && go test -tags lsp ./internal/cli/... && go test ./internal/quarryengine/query/...
  - number: 2
    name: mcpserver-foundation
    file: 02-mcpserver-foundation.md
    depends-on: [1]
    verify: go test ./internal/mcpserver/...
  - number: 3
    name: lsp-mirrored-tools
    file: 03-lsp-mirrored-tools.md
    depends-on: [2]
    verify: go test ./internal/mcpserver/...
  - number: 4
    name: quarry-native-lsp-tools
    file: 04-quarry-native-lsp-tools.md
    depends-on: [3]
    verify: go test ./internal/mcpserver/...
  - number: 5
    name: toc-tools
    file: 05-toc-tools.md
    depends-on: [4]
    verify: go test ./internal/mcpserver/...
  - number: 6
    name: server-binary-and-transport-tests
    file: 06-server-binary-and-transport-tests.md
    depends-on: [5]
    verify: go test ./internal/mcpserver/... ./cmd/...
  - number: 7
    name: integration-and-docs
    file: 07-integration-and-docs.md
    depends-on: [6]
    verify: go test ./internal/mcpserver/... && go test -tags lsp ./internal/mcpserver/...
```

## Shared Decisions

_Cross-cutting decisions every batch inherits._

### Decision: array-parameter-name

- **Decision:** every tool's array input parameter is named `targets`, and every tool's output
  envelope is `{"results": [...]}` with each entry carrying `target` and `status`.
- **Rationale:** `_mill/discussion.md` pins the envelope (`results`, per-entry `target`) and the
  array-batching shape but never names the input parameter. `targets` in / `results` out is
  symmetric with the per-entry `target` echo, and one name across all seven tools means the model
  never has to remember a per-tool spelling.
- **Applies to:** all batches

### Decision: schema-derivation-and-patching

- **Decision:** input and output schemas are derived from Go types with
  `jsonschema.For[T]` (`github.com/google/jsonschema-go`), called with the package's own
  `jsonschema.ForOptions` value — never `nil`, because the `docSentences` type schema is registered
  through its `TypeSchemas` map and inference would otherwise reduce that type to a property-less
  object — then patched by the shared helpers
  in `internal/mcpserver/schema.go` before being assigned to `mcp.Tool.InputSchema` /
  `mcp.Tool.OutputSchema`. Three patches are mandatory and are the reason raw inference is never
  used as-is:
  1. `AdditionalProperties` is cleared (set to `nil`) on every object schema reachable from the
     `targets` element schema and on every object schema in an output schema. Struct inference sets
     `AdditionalProperties` to the false schema, which would turn a stray or wrong-tool property on
     one entry into a whole-call validation failure — exactly the outcome `input-schema-strictness`
     and `error-mapping` forbid. Call-wide properties keep whatever inference produced, because
     `input-schema-strictness` deliberately makes call-level violations a whole-call failure.
  2. On the `targets` property schema, `Types` is cleared and `Type` is set to `"array"`, and
     `MinItems`/`MaxItems` are set to `1` and `64` via `jsonschema.Ptr`. Slice inference emits
     `Types: ["null", "array"]`, which would let `{"targets": null}` bypass `minItems` entirely.
  3. Every entry-object property, and every property of a nested entry object, is optional —
     achieved with `omitempty` (and pointer element types where the zero value is meaningful, such
     as a `position`'s `line`/`character`) so inference never adds them to `required`. The legal
     combinations are described in the property descriptions and validated per entry by the
     handler.
- **Rationale:** the SDK validates input against the tool's input schema before the handler runs
  and returns a `CallToolResult` with `IsError` set on failure; it validates the handler's typed
  output against the output schema and fails the call when validation fails. Both behaviours are
  desirable only for the call-level rules, so the schema must be patched to stop them applying at
  entry level.
- **Applies to:** all batches

### Decision: whole-call-failure-mechanics

- **Decision:** a whole-call failure is expressed by returning a non-nil `error` from the
  `mcp.ToolHandlerFor` function. The SDK packs it into `CallToolResult` with `IsError` set, which
  is the `isError: true` the discussion's `error-mapping` section specifies. Handlers never
  construct `CallToolResult.IsError` by hand. A per-entry failure is never an `error` return — it
  is an entry in `results` with `status: "error"`.
- **Rationale:** one mechanism, and it keeps the pre-flight-versus-per-entry split visible in the
  handler's own control flow.
- **Applies to:** batches 3, 4, 5, 6

### Decision: renaming-is-rename-only

- **Decision:** batch 1 changes identifier spelling and call sites only. No signature, no body, no
  doc-comment semantics, no output shape, no flag, and no exit code changes anywhere in
  `internal/cli`.
- **Rationale:** `_mill/discussion.md`'s Constraints section allows exactly one edit to
  `internal/cli` — exporting the thirteen named helpers — and defines that as renaming the
  identifier and updating its call sites.
- **Applies to:** export-cli-helpers

### Decision: engine-comment-references

- **Decision:** three files under `internal/quarryengine/` mention `filterUnexpectedCallers` in
  prose comments only. Batch 1 updates that prose to the new exported spelling. No Go statement in
  any `internal/quarryengine/` file is touched.
- **Rationale:** the discussion's "No engine changes" scope item is about engine logic and the
  `character`-unit asymmetry. Leaving a stale identifier name in three doc comments would be a
  documentation defect introduced by this task, and correcting the prose costs nothing and changes
  no behaviour.
- **Applies to:** export-cli-helpers

### Decision: no-mill-config-change

- **Decision:** `pipeline.done_gate` stays at its current `go test ./...` and `mill-config.yaml`
  is not edited by this plan.
- **Rationale:** the repo-wide test command is already configured and meaningful. Neither
  `golangci-lint` nor `gopls` is installed on this machine, so a lint-extended `done_gate` could
  not be verified to exit 0 at the worktree tip, and adding an unverifiable gate would block every
  future task in this hub.
- **Applies to:** all batches

### Decision: verify-scope-and-cgo

- **Decision:** per-batch `verify:` commands are package-scoped `go test` invocations, never
  `go test ./...`. The module-wide `verify: go build ./...` in this file's frontmatter is the
  cross-package gate run at each batch boundary.
- **Rationale:** quarry is a cgo build (`internal/quarryengine/cgoguard_nocgo.go` fails a
  `CGO_ENABLED=0` build on purpose), so the first compile in a cold cache is expensive; scoping
  each batch keeps the repeated per-round cost proportional to what the batch touched, while
  `go build ./...` still catches a cross-package regression at the batch that introduced it.
- **Applies to:** all batches

### Decision: lsp-tagged-verify-chaining

- **Decision:** a batch that edits or creates a `//go:build lsp` file chains a second
  `go test -tags lsp <same packages>` invocation onto its `verify:` command rather than
  comma-joining a tag onto the first.
- **Rationale:** the tagged tests `t.Skip` when `gopls` is absent from `$PATH` (which it is on this
  machine), so the tagged run is a compile-and-skip gate. Chaining a second invocation keeps the
  untagged run's semantics identical to every other batch's.
- **Applies to:** export-cli-helpers, integration-and-docs

### Decision: facade-seam-usage

- **Decision:** every facade call inside `internal/mcpserver` goes through the package-level
  function variables declared in `internal/mcpserver/facade.go` — `definitionFn`, `referencesFn`,
  `symbolFn`, `callersFn`, `impactFn`, `tocFileFn`, `tocDirFn`. No handler calls `quarry.Definition`
  (or any sibling) directly.
- **Rationale:** `_mill/discussion.md`'s `facade-seam-for-tests` decision. Five of the seven
  handlers are otherwise untestable without a live gopls, which would push every per-entry status
  and error-mapping assertion behind the `//go:build lsp` gate.
- **Applies to:** batches 2, 3, 4, 5, 6

## All Files Touched

- `.gitignore`
- `.mcp.json`
- `README.md`
- `cmd/quarry-mcp/main.go`
- `docs/mcp-setup.md`
- `go.mod`
- `go.sum`
- `internal/cli/assertnocallers_lsp_test.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/impact.go`
- `internal/cli/impact_test.go`
- `internal/cli/paths.go`
- `internal/cli/paths_test.go`
- `internal/cli/resolve_test.go`
- `internal/cli/toc.go`
- `internal/cli/toc_test.go`
- `internal/cli/tocconfig.go`
- `internal/cli/tocconfig_test.go`
- `internal/mcpserver/callcontext.go`
- `internal/mcpserver/callcontext_test.go`
- `internal/mcpserver/facade.go`
- `internal/mcpserver/layering_test.go`
- `internal/mcpserver/lspentry.go`
- `internal/mcpserver/lspentry_test.go`
- `internal/mcpserver/mcpserver.go`
- `internal/mcpserver/nativeentry.go`
- `internal/mcpserver/nativeentry_test.go`
- `internal/mcpserver/result.go`
- `internal/mcpserver/result_test.go`
- `internal/mcpserver/schema.go`
- `internal/mcpserver/schema_test.go`
- `internal/mcpserver/stdio_lsp_test.go`
- `internal/mcpserver/tocentry.go`
- `internal/mcpserver/tocentry_test.go`
- `internal/mcpserver/tools_assert.go`
- `internal/mcpserver/tools_assert_test.go`
- `internal/mcpserver/tools_impact.go`
- `internal/mcpserver/tools_impact_test.go`
- `internal/mcpserver/tools_lsp.go`
- `internal/mcpserver/tools_lsp_test.go`
- `internal/mcpserver/tools_symbol.go`
- `internal/mcpserver/tools_toc.go`
- `internal/mcpserver/tools_toc_test.go`
- `internal/mcpserver/translate.go`
- `internal/mcpserver/translate_test.go`
- `internal/mcpserver/transport_errors_test.go`
- `internal/mcpserver/transport_test.go`
- `internal/quarryengine/impact/impact.go`
- `internal/quarryengine/query/callers.go`
- `internal/quarryengine/query/callers_test.go`
