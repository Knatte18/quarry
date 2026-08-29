# Batch: lsp-mirrored-tools

```yaml
task: "Add an MCP wrapper for quarry"
batch: "lsp-mirrored-tools"
number: 3
cards: 4
verify: go test ./internal/mcpserver/...
depends-on: [2]
```

## Batch Scope

This batch delivers the three tools whose names and entry shapes mirror LSP verbatim:
`textDocument_definition`, `textDocument_references`, and `workspace_symbol`. It is one batch
because all three share one entry-parsing file, one call-wide parameter set
(`lang`, `buildTags`, `targetDir`), and one sequential batching loop; splitting them would fork
that shared parsing across two contexts.

The external interface batches 4, 5, and 6 consume: the per-type `raw json.RawMessage` plus
own-`UnmarshalJSON` convention for capturing an entry's original JSON, `runTargets` (the shared
sequential batching loop), and the two `register*` functions covering the three LSP-mirrored
tools, now called from `NewServer`.

Batch-local decisions beyond `## Shared Decisions`:

- The three LSP-mirrored tools are 0-based on both `line` and `character`, on input and on output.
  Every other tool in this plan is 1-based on both. Each tool's description string states its own
  convention in its first line, so the difference is never inferred.
- `workspace_symbol` declares exactly one entry property, `query`. A `textDocument`, `position`, or
  `symbol` key on one of its entries is that entry's own `status: "error"`, never a whole-call
  failure and never a silent empty-string search.

## Cards

### Card 13: Add LSP entry parsing and the shared batching loop

- **Context:**
  - `internal/mcpserver/translate.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/result.go`
  - `quarry/facade.go`
  - `internal/cli/cli.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/lspentry_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/lspentry.go` declaring
  the raw-capture convention every entry type in this package follows, in the one form that does
  not break decoding: each entry type carries its own unexported `raw json.RawMessage` field and
  its own `UnmarshalJSON` method, which records the incoming bytes into `raw` and then decodes into
  a locally-declared alias of the same struct — `type alias lspEntry` — so the method is not
  invoked recursively and every exported field is still populated. Do not put `UnmarshalJSON` on a
  shared embedded helper type: Go promotes a method from an embedded field to the outer type, so an
  embedded raw-capture type would hijack the outer entry's decode and leave every declared field at
  its zero value, making every entry parse as "no accepted form". Document that `raw` exists so the
  handler can both echo the input verbatim and detect keys the tool does not declare. Declare
  `type textDocumentIdentifier struct { URI string }` with json tag `uri,omitempty`,
  `type lspPosition struct { Line *int; Character *int }` with json tags `line,omitempty` and
  `character,omitempty` — pointers so a missing field is distinguishable from zero and neither is
  inferred as required — and
  `type lspEntry struct { raw json.RawMessage; TextDocument *textDocumentIdentifier; Position *lspPosition; Symbol string; Within string }`
  with `raw` unexported and therefore invisible to both `encoding/json` and schema inference, and
  json tags `textDocument,omitempty`, `position,omitempty`, `symbol,omitempty`,
  `within,omitempty`, each carrying a `jsonschema` tag describing the property and naming which
  entry forms it participates in. Declare
  `func (e lspEntry) query(targetDir string) (quarry.Query, error)` implementing the three-form
  union: `TextDocument` plus `Position` yields a `quarry.Query` with `Pos` set to a
  `quarry.Position` whose `File` is `resolveEntryFile(targetDir, e.TextDocument.URI)`, whose `Line`
  is `toOneBased(*e.Position.Line)`, and whose `Character` is `toOneBased(*e.Position.Character)`;
  `Symbol` alone yields `quarry.Query{Symbol: e.Symbol}`; `TextDocument` plus `Symbol` yields
  `quarry.Query{InFile: &quarry.InFileQuery{File: resolveEntryFile(targetDir, e.TextDocument.URI), Name: e.Symbol}}`.
  Every other combination — a `position` with no `textDocument`, neither `symbol` nor `position`,
  both `symbol` and `position`, a `position` missing `line` or `character`, an empty `uri`, an
  empty `symbol` — returns an error whose message names the three accepted forms for the tool.
  Declare
  `func runTargets[E any, R any](targets []E, one func(int, E) R) []R` which executes `one` for
  each target strictly sequentially in input order and returns exactly one result per input,
  always; document that entries are never run concurrently because every facade call acquires its
  own connection and a 64-entry array run concurrently would mean 64 simultaneous dials against the
  supervised daemon for no gain, and that sequential execution is also what makes the
  one-result-per-input-in-input-order contract trivially true. Create
  `internal/mcpserver/lspentry_test.go` covering each of the three legal forms mapping onto the
  right `quarry.Query` variant with the `+1` conversion applied to both axes, each illegal
  combination returning an error rather than a silent guess, `lspEntry.UnmarshalJSON` populating
  every exported field while also preserving the original bytes in `raw`, and `runTargets` returning results in input order with one result per input.
- **Commit:** `feat(mcpserver): add LSP entry parsing and the sequential batching loop`

### Card 14: Add `textDocument_definition` and `textDocument_references`

- **Context:**
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/result.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/translate.go`
  - `internal/cli/cli.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/mcpserver/mcpserver.go`
- **Creates:**
  - `internal/mcpserver/tools_lsp.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_lsp.go` declaring
  `type lspInput struct { Targets []lspEntry; Lang string; BuildTags string; TargetDir string }`
  with json tags `targets`, `lang,omitempty`, `buildTags,omitempty`, `targetDir,omitempty` and a
  `jsonschema` tag on each — `lang` documented as a servers.yaml registry key, validated through
  `quarry.LoadRegistry` and `quarry.DetectLanguage`. Declare
  `type definitionEntry struct { Target any; Status string; Resolution string; Definitions []referenceField; Candidates []string; Error string }`
  and `type referencesEntry struct { ... References []referenceField ... }` — identical except for
  the results key — with `target` and `status` as the only non-`omitempty` fields, and
  `type definitionOutput struct { Results []definitionEntry }` and
  `type referencesOutput struct { Results []referencesEntry }`, each with json tag `results`. No
  entry type declares a key its own tool cannot emit. Declare
  `registerLSPTools(s *mcp.Server, cfg Config) error` which builds each tool's input schema with
  `inputSchemaFor[lspInput]()` and its output schema with `outputSchemaFor[definitionOutput]()` /
  `outputSchemaFor[referencesOutput]()`, assigns them to the `mcp.Tool` value's `InputSchema` and
  `OutputSchema`, and registers each through `mcp.AddTool`. Each handler: calls
  `resolveCall(cfg, in.TargetDir, in.BuildTags)` and returns that error unchanged as a whole-call
  failure, then runs `runTargets` over `in.Targets`. Per entry it first reports
  `unknownEntryKeys(entry.raw, "textDocument", "position", "symbol", "within")` as
  `status: statusError`, then parses the entry with `lspEntry.query`, reporting a parse error as
  `status: statusError` too; on success it calls `definitionFn` or `referencesFn` with
  `callContext.options(in.Lang, query)`, applies `cli.FilterWithin(refs, entry.Within, ctx.TargetDir)`
  when `entry.Within` is non-empty, and maps the outcome through `classifyLSPError`. A `found`
  entry carries `resolution: resolutionComplete` and `referenceFieldsWire(...)` under
  `definitions` or `references`; the `target` key always carries the entry's own JSON decoded back
  into `any`, never a value derived from the result. Each tool's `Description` opens with a
  sentence stating that `line` and `character` are 0-based on this tool, then names the three
  accepted entry forms and the 64-entry cap. Edit `NewServer` in
  `internal/mcpserver/mcpserver.go` to call `registerLSPTools(s, cfg)` and return its error.
- **Commit:** `feat(mcpserver): add textDocument_definition and textDocument_references`

### Card 15: Add `workspace_symbol`

- **Context:**
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/result.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/facade.go`
  - `internal/cli/cli.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/mcpserver/mcpserver.go`
- **Creates:**
  - `internal/mcpserver/tools_symbol.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_symbol.go` declaring
  `type symbolEntry struct { raw json.RawMessage; Query string }`, following card 13's per-type
  `UnmarshalJSON` convention, with json tag `query,omitempty` on `Query` — the field
  name LSP's own `WorkspaceSymbolParams` uses — and no other entry property, so the derived schema
  declares only `query`. Declare
  `type symbolInput struct { Targets []symbolEntry; Lang string; BuildTags string; TargetDir string }`
  with the same call-wide set as `lspInput` and no `within` property, because the CLI registers
  none for `symbol` and `query.Symbol` has nothing to filter per-file against. Declare
  `type symbolMatchEntry struct { Target any; Status string; Symbols []symbolField; Error string }`
  — no `resolution` key, because `classifySymbolError` in `internal/cli/cli.go` never sets the
  marker and adding it would claim exhaustive language-server resolution the CLI does not
  claim; and no `candidates` key either, because this tool never emits an `ambiguous` status —
  and `type symbolOutput struct { Results []symbolMatchEntry }`. Declare
  `registerSymbolTool(s *mcp.Server, cfg Config) error` following card 14's registration shape.
  Per entry the handler calls `unknownEntryKeys(entry.raw, "query")` first and reports any key it
  finds as that entry's `status: statusError` with a message naming `query` as the only accepted
  property; an empty `query` is the same per-entry error. Only then does it call `symbolFn` with
  `callContext.options(in.Lang, quarry.Query{Symbol: entry.Query})` and map the outcome with
  `classifySymbolError` — not `classifyLSPError`, which would add an `ambiguous` branch
  `quarry.Symbol` never reaches — emitting `symbolFieldsWire(...)` under `symbols` on a `found` entry. The
  tool's `Description` opens with a sentence stating that results carry 0-based `line` and
  `character`, then states that `query` is the only accepted entry property. Edit `NewServer` in
  `internal/mcpserver/mcpserver.go` to call `registerSymbolTool(s, cfg)` and return its error.
- **Commit:** `feat(mcpserver): add workspace_symbol`

### Card 16: Handler unit tests for the three LSP-mirrored tools

- **Context:**
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/result.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/tools_lsp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_lsp_test.go` driving each handler directly
  with the facade seam variables replaced by stubs, restoring them with `t.Cleanup`, and pinning
  `Config.StateDir` to a `t.TempDir()` so no machine-global cache directory is touched. Assert: a
  three-entry mixed call returns three entries in input order with `found`, `not_found`, and
  `ambiguous` statuses and the good result intact; every entry carries `target` echoing its input
  and `status`; a `found` `textDocument_definition` and `textDocument_references` entry carries
  `resolution: "complete"` while a `found` `workspace_symbol` entry does not, and no `workspace_symbol` entry ever carries
  `candidates`; results carry
  0-based `line` and `character`; a `position` entry with no `textDocument` is that entry's
  `status: "error"` while its siblings still return their results; a `workspace_symbol` entry
  carrying `textDocument` and one carrying `position` are each that entry's `status: "error"` and
  neither produces an empty-string search; a per-entry `within` filters that entry's references
  without affecting another entry's; and a stub returning
  `quarry.ErrServerSpawnTimeoutSentinel` produces a per-entry `status: "error"` rather than a
  whole-call failure, because a daemon spawn failure surfaces only from a per-entry facade call.
- **Commit:** `test(mcpserver): cover the three LSP-mirrored tool handlers`

## Batch Tests

`verify: go test ./internal/mcpserver/...` runs the batch's two new test files —
`lspentry_test.go` and `tools_lsp_test.go` — alongside batch 2's suite, which must keep passing
because card 14 and card 15 both edit `NewServer`. Scope is the one package these cards touch. The
per-entry status and error-mapping assertions live here rather than in a `//go:build lsp` tier
precisely because the facade seam makes them reachable without gopls; that is the seam's whole
justification.
