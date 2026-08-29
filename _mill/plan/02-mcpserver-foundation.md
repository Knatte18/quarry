# Batch: mcpserver-foundation

```yaml
task: "Add an MCP wrapper for quarry"
batch: "mcpserver-foundation"
number: 2
cards: 7
verify: go test ./internal/mcpserver/...
depends-on: [1]
```

## Batch Scope

This batch creates the `internal/mcpserver` package and everything the seven handlers stand on:
the module dependency, the server constructor and its launch configuration, the facade seam, the
translation layer, the shared per-entry result vocabulary, the schema derivation-and-patching
helpers, the per-call context resolution that mirrors `internal/cli`'s `resolveContext` and
`buildOptions`, and the layering test that keeps the package off `internal/quarryengine`. It is one
batch because none of these pieces is independently useful and every one of them is consumed by
every handler batch that follows.

The external interface batches 3, 4, and 5 consume: `Config`, `NewServer`, the seven `*Fn`
variables, `resolveEntryFile`/`toOneBased`/`toZeroBased`, the `status*` constants and the
`classifyLSPError`/`classifySymbolError`/`classifyTOCError` helpers, `inputSchemaFor`/`outputSchemaFor`/`unknownEntryKeys`/`dropEntryProperty`, the `docSentences`
type, and `effectiveTargetDir`/`resolveCall`.

Batch-local decision beyond `## Shared Decisions`: `NewServer` registers no tools yet. Each
handler batch adds its own `register*` call to `NewServer` in `internal/mcpserver/mcpserver.go`,
which is why batches 3, 4, and 5 form a chain rather than a fan-out.

## Cards

### Card 6: Create the `internal/mcpserver` package and add the MCP SDK dependency

- **Context:**
  - `cmd/quarry/main.go`
  - `internal/cli/cwdcontext.go`
  - `quarry/facade.go`
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:**
  - `internal/mcpserver/mcpserver.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `github.com/modelcontextprotocol/go-sdk v1.7.0` as a direct requirement by
  running `go get github.com/modelcontextprotocol/go-sdk@v1.7.0`, then `go mod tidy` after the new
  file below exists so the requirement is justified by a real import. Do not acquire
  `github.com/google/jsonschema-go` here: nothing imports it until card 10, so `go mod tidy` in
  this card would demote it straight back to `// indirect`. Card 10 owns that acquisition, in the
  same card as its first import. Create `internal/mcpserver/mcpserver.go` with a package doc comment stating that
  this package binds `quarry/facade.go` onto MCP tools and imports `internal/cli` for resolution
  helpers but never `internal/quarryengine`. Declare `const serverVersion = "0.1.0"`. Declare
  `const minTargets = 1` and `const maxTargets = 64`. Declare
  `type Config struct { TargetDir string; ConfigPath string; StateDir string; Timeout time.Duration }`
  documenting each field as a launch-only value the model never sees. Declare
  `func ResolveLaunchTargetDir(flagValue string) (string, error)` which returns
  `filepath.Abs(flagValue)` when `flagValue` is non-empty and `os.Getwd()` otherwise — `os.Getwd`
  already returns an absolute path, so no second absolutisation is needed — wrapping either
  function's error as `mcpserver: resolve target dir: %w`; document that this runs exactly once,
  at server startup, before any handler can run, and that every downstream consumer therefore only
  ever sees an absolute path. Declare
  `func NewServer(cfg Config) (*mcp.Server, error)` which asserts `filepath.IsAbs(cfg.TargetDir)`
  and returns an error naming the field when it is not, then constructs and returns
  `mcp.NewServer(&mcp.Implementation{Name: "quarry", Version: serverVersion}, nil)`. `NewServer`
  registers no tools in this batch; later batches add `register*` calls to it. Nothing in this
  package writes to `os.Stdout`.
- **Commit:** `feat(mcpserver): add package skeleton and the MCP Go SDK dependency`

### Card 7: Declare the facade seam

- **Context:**
  - `quarry/facade.go`
  - `quarry/facade_test.go`
  - `internal/cli/paths.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/facade.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/facade.go` declaring exactly seven package-level
  function variables, each defaulting to its facade function: `definitionFn = quarry.Definition`,
  `referencesFn = quarry.References`, `symbolFn = quarry.Symbol`, `callersFn = quarry.Callers`,
  `impactFn = quarry.Impact`, `tocFileFn = quarry.TOCFile`, `tocDirFn = quarry.TOCDir`. Document
  that these exist so tests in this package can substitute stubs, mirroring the `userConfigDir` and
  `userCacheDir` seam convention in `internal/cli/paths.go`, and that no behaviour may be added
  here or in `quarry/facade.go` — the facade is behaviour-free and `quarry/facade_test.go` enforces
  that mechanically.
- **Commit:** `feat(mcpserver): add the facade function-variable seam`

### Card 8: Add the translation layer

- **Context:**
  - `internal/cli/cli.go`
  - `internal/quarryengine/lsp/wire.go`
  - `internal/quarryengine/query/refs.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/translate.go`
  - `internal/mcpserver/translate_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/translate.go` with four functions.
  `stripFileURI(s string) string` removes a leading `file://` prefix and returns everything else
  unchanged; a `file:///abs/path` input yields `/abs/path`. `resolveEntryFile(targetDir, raw string) string`
  applies `stripFileURI` and then `cli.AbsOrJoin(targetDir, ...)`; document that `targetDir` is
  always absolute by the time this is called and that `quarry.Query.Pos.File` and
  `quarry.InFileQuery.File` must be absolute because `References` turns them into `file://` URIs
  with no further resolution. `toOneBased(v int) int` returns `v + 1` and
  `toZeroBased(v int) int` returns `v - 1`; both are applied to the `line` and the `character` of
  an LSP-mirrored tool's position, because LSP is 0-based on both axes while `quarry.Position` is
  1-based on both. Both carry a comment naming the engine's `character`-unit asymmetry —
  `internal/quarryengine/lsp/wire.go` converts a 1-based byte column into a 0-based UTF-16
  character inbound while `internal/quarryengine/query/refs.go` does a naive `+1` outbound — and
  stating that this layer deliberately reproduces the CLI's existing naive behaviour rather than
  fixing it locally, because a returned `character` must round-trip straight back into a following
  call. Neither conversion reads a file. Create
  `internal/mcpserver/translate_test.go` covering `file://` stripping and plain-path passthrough,
  `resolveEntryFile` against an absolute and a relative input, the `±1` conversion in both
  directions, round-trip stability, and an explicitly-asserted non-ASCII case that pins the current
  naive behaviour as deliberate rather than accidental.
- **Commit:** `feat(mcpserver): add the URI and character translation layer`

### Card 9: Add the shared per-entry result vocabulary

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/impact.go`
  - `internal/cli/toc.go`
  - `internal/quarryengine/query/symbol.go`
  - `internal/mcpserver/translate.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/result.go`
  - `internal/mcpserver/result_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/result.go` declaring
  `const statusFound = "found"`, `statusNotFound = "not_found"`, `statusAmbiguous = "ambiguous"`,
  and `statusError = "error"`, plus `const resolutionComplete = "complete"`. Declare
  `type referenceField struct { File string; Line int; Character int }` and
  `type symbolField struct { Name string; Kind int; File string; Line int; Character int }` —
  `Kind` is an `int` because `quarry.SymbolMatch.Kind` is the numeric LSP `SymbolKind` and
  `symbolMatchFields` in `internal/cli/cli.go` emits it unchanged, so typing it `string` here
  would silently change the JSON type the CLI emits for the identical query —
  both with lower-case JSON tags matching `referenceFields` and `symbolMatchFields` in
  `internal/cli/cli.go`. Declare `referenceFieldsWire(refs []quarry.Reference) []referenceField`
  and `symbolFieldsWire(matches []quarry.SymbolMatch) []symbolField`, each applying
  `toZeroBased` to both the `Line` and the `Character` value so results leave in the 0-based
  convention the LSP-mirrored tools declare on both axes, and each returning a non-nil empty slice
  rather than nil so an empty
  result marshals as `[]`. Declare `referenceFieldsNative(refs []quarry.Reference) []referenceField`
  which is identical except that it applies no conversion, for the quarry-native tools. Declare
  `classifyLSPError(err error) (status string, candidates []string, message string)` implementing
  exactly the branches `classifyLookupError` uses, nil branch included: a nil `err` yields
  `statusFound` with no candidates and no message — stated explicitly so a caller that hands it a
  nil error does not fall through to the else branch and dereference it; `errors.As(err, &ambiguous)` against
  `*quarry.ErrAmbiguousSymbol` yields `statusAmbiguous` with the candidates;
  `errors.Is(err, quarry.ErrSymbolNotFoundSentinel)` yields `statusNotFound` with no message;
  anything else yields `statusError` carrying `err.Error()`. Declare
  `classifySymbolError(err error) (status string, message string)` implementing
  `classifySymbolError`'s own two predicates from `internal/cli/cli.go` instead — nil yields
  `statusFound`, `errors.Is(err, quarry.ErrSymbolNotFoundSentinel)` yields `statusNotFound`,
  and anything else yields `statusError` — with no `ambiguous` branch, because
  `symbolFromClient` in `internal/quarryengine/query/symbol.go` deliberately returns every
  match rather than collapsing multiple candidates to `quarry.ErrAmbiguousSymbol`, so
  `quarry.Symbol` never produces that error and an `ambiguous` branch here would be dead code
  that also forces a `candidates` key onto a tool whose CLI counterpart cannot emit one.
  Declare `classifyTOCError(err error) (status string, message string)` implementing toc's own rule
  instead: `errors.Is(err, quarry.ErrLanguageUnsupported)` yields `statusError` with a message
  worded from `quarry.TOCImplemented()` exactly as `internal/cli/toc.go` words it, and anything
  else yields `statusError` with `err.Error()`; document that the LSP predicates must never be
  applied to toc, because toc uses no language server and applying them would report `error` where
  the CLI reports `not_found` for a missing file. Declare
  `rewordMarshalFailure(err error) string` returning `fmt.Sprintf("impact: %s", strings.TrimPrefix(err.Error(), "toc: "))`,
  mirroring `rewordImpactMarshalFailure` in `internal/cli/impact.go`. Create
  `internal/mcpserver/result_test.go` asserting each predicate against a constructed sentinel and
  wrapped sentinel, that `classifySymbolError` has no `ambiguous` branch, that the toc
  classifier does not borrow the LSP predicates, that the wire
  converters subtract one and the native converters do not, that both return `[]` rather than nil
  for an empty input, and that `rewordMarshalFailure` replaces the `toc: ` prefix.
- **Commit:** `feat(mcpserver): add the shared per-entry status and field vocabulary`

### Card 10: Add schema derivation and patching

- **Context:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/cli/tocconfig.go`
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:**
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/schema_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Run `go get github.com/google/jsonschema-go@v0.4.3` and `go mod tidy` in this
  card, not in card 6: this card holds the package's first direct import of `jsonschema-go`, so
  running the acquisition here is what makes it a direct requirement that survives tidy. Create
  `internal/mcpserver/schema.go` implementing the
  `schema-derivation-and-patching` Shared Decision against
  `github.com/google/jsonschema-go/jsonschema`. Declare
  `type docSentences struct { raw json.RawMessage }` with an `UnmarshalJSON` method that accepts a
  JSON number or a JSON string and rejects anything else, and a method
  `value() (string, bool)` returning the decimal string form of a number, the string unchanged for
  a string, and `false` for the zero value, so `{"docSentences": 3}` and `{"docSentences": "all"}`
  both reach `cli.ParseDocSentences` as a string. Declare
  `var docSentencesSchema = &jsonschema.Schema{Types: []string{"integer", "string"}}` and register
  it through `jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{reflect.TypeFor[docSentences](): docSentencesSchema}}`.
  Declare `clearAdditionalProperties(s *jsonschema.Schema)` walking `Properties`, `Items`, and
  `AdditionalProperties` recursively and setting each visited object schema's
  `AdditionalProperties` to nil, guarding against a nil schema and against revisiting a schema
  pointer twice. Declare `inputSchemaFor[T any]() (*jsonschema.Schema, error)` which calls
  `jsonschema.For[T]` with the options above, then locates `Properties["targets"]`, sets its
  `Types` to nil and its `Type` to `"array"`, sets `MinItems` to `jsonschema.Ptr(minTargets)` and
  `MaxItems` to `jsonschema.Ptr(maxTargets)`, and calls `clearAdditionalProperties` on its `Items`
  only — the call-wide properties keep whatever inference produced, so a call-level violation stays
  a whole-call failure. Return an error naming `targets` when the property is absent. Declare
  `outputSchemaFor[T any]() (*jsonschema.Schema, error)` which calls `jsonschema.For[T]` with the
  same `jsonschema.ForOptions` value `inputSchemaFor` uses — never `nil`, so the two helpers name
  one call form, harmless here because no output type embeds `docSentences` — and then
  `clearAdditionalProperties` on the whole tree, so a mixed batch never fails output validation.
  Declare `dropEntryProperty(s *jsonschema.Schema, name string)` which removes `name` from the
  `targets` item schema's `Properties` map and from its `PropertyOrder`, so a tool whose entry Go
  type carries a field it does not accept never advertises that field in its published schema.
  Declare `unknownEntryKeys(raw json.RawMessage, allowed ...string) []string` which unmarshals
  `raw` into a `map[string]json.RawMessage` and returns the sorted key names absent from `allowed`,
  or nil when `raw` is not a JSON object; document that this exists because clearing
  `additionalProperties` on entry schemas is what makes a wrong-tool property a per-entry error
  instead of a whole-call failure, so the handler needs its own detection point for one. Create
  `internal/mcpserver/schema_test.go` asserting, against a small local fixture type, that a
  derived input schema has `type: "array"` and no `"null"` member on `targets`, that `minItems` is
  1 and `maxItems` is 64, that no object schema under `targets` carries `additionalProperties`,
  that a call-wide property's own inferred constraints survive, that no entry-object property is
  listed in `required`, that a derived output schema carries `additionalProperties` nowhere, that
  `docSentences` unmarshals from both `3` and `"all"` and rejects `true`, and that
  `unknownEntryKeys` reports a key absent from `allowed`, reports nothing for an entry using only
  allowed keys, and returns nil for a non-object input; and that `dropEntryProperty` removes the
  named property from both `Properties` and `PropertyOrder` and leaves every other property intact.
- **Commit:** `feat(mcpserver): add schema derivation and the permissive-entry patches`

### Card 11: Add per-call context and options resolution

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/paths.go`
  - `internal/cli/toc.go`
  - `quarry/facade.go`
  - `internal/mcpserver/mcpserver.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/callcontext_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/callcontext.go` declaring
  `type callContext struct { Registry quarry.Registry; TargetDir string; StateDir string; BuildTags []string; Timeout time.Duration }`
  and `func resolveCall(cfg Config, targetDirOverride, buildTags string) (callContext, error)`.
  Declare `func effectiveTargetDir(cfg Config, override string) (string, error)`: when `override`
  is non-empty it is passed through `filepath.Abs` immediately, before it is used for anything, and
  any error is returned wrapped as `mcpserver: resolve targetDir: %w`; otherwise `cfg.TargetDir` is
  returned unchanged, already absolute from `ResolveLaunchTargetDir`. Document that this is the
  only place a per-call override becomes absolute, and that the toc tools call it directly without
  going through `resolveCall`, because `tocFileCommand` and `tocDirCommand` never call
  `resolveContext`, `quarry.LoadRegistry`, or `ResolveStateDir` — so a malformed `servers.yaml`
  must not fail a toc call.
  `resolveCall` calls `effectiveTargetDir` first, then follows `resolveContext`'s sequence in
  `internal/cli/cli.go` exactly and in the same order, through the exported helpers:
  `cli.ResolveBuildTags(buildTags)`, then `cli.ResolveConfigPath(cfg.ConfigPath)`, then
  `quarry.LoadRegistry` on that path, then `cli.ResolveStateDir(cfg.StateDir, absTargetDir, tags)`. `callContext.Timeout` is carried
  straight from `cfg.Timeout` — the launch-only `--timeout` value — so it is a per-entry
  deadline for each entry's facade call rather than a whole-call budget, exactly as the CLI
  applies `Options.Timeout` per invocation.
  Document that using these helpers rather than a local copy is what keeps the state-directory key
  bit-for-bit identical to the CLI's, and that a divergence silently spawns a second gopls daemon
  and forfeits warm-daemon reuse. Declare
  `func (c callContext) options(lang string, query quarry.Query) quarry.Options` returning a
  `quarry.Options` populated exactly as `buildOptions` populates it — `Registry`, `TargetDir`,
  `StateDir`, `Lang`, `Query`, `Timeout`, `BuildTags` — and leaving `SkipVerification` at its zero
  value so the default is "verify". Create `internal/mcpserver/callcontext_test.go` asserting that
  a relative `targetDirOverride` is absolutised, that the launch default is used when the override
  is empty, that the derived `StateDir` equals what `cli.ResolveStateDir` returns for the identical
  inputs so drift fails the build, that a non-empty build-tag set appends the `tags-` segment,
  that `options` leaves `SkipVerification` false, and that `options` carries `cfg.Timeout` through
  to `quarry.Options.Timeout` in full for every entry rather than dividing it across a batch —
  build the `quarry.Options` for several entries of one call and assert each carries the same,
  undivided value, which is `batching-execution-model`'s per-entry-timeout rule. Pin `Config.StateDir` explicitly in these tests
  rather than relying on the machine's user cache directory, because the `userCacheDir` seam is
  reachable only from inside `internal/cli`.
- **Commit:** `feat(mcpserver): add per-call context and options resolution`

### Card 12: Add the layering test

- **Context:**
  - `internal/quarryengine/layering_test.go`
  - `internal/mcpserver/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/layering_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/layering_test.go` with a test that walks every
  non-test and test Go file in the `internal/mcpserver` package directory, parses its import block
  with `go/parser` in `parser.ImportsOnly` mode, and fails on any import path with the
  `github.com/Knatte18/quarry/internal/quarryengine` prefix. `github.com/Knatte18/quarry/quarry`
  and `github.com/Knatte18/quarry/internal/cli` are both allowed. Document that
  `internal/quarryengine/layering_test.go` polices only rows under `internal/quarryengine/...` and
  has no row for this package, so without this test the facade-only constraint would be
  convention-only while the analogous engine rule is mechanical.
- **Commit:** `test(mcpserver): enforce facade-only imports with a layering test`

## Batch Tests

`verify: go test ./internal/mcpserver/...` runs the four test files this batch creates:
`translate_test.go`, `result_test.go`, `schema_test.go`, `callcontext_test.go`, plus
`layering_test.go`. Scope is the new package only — nothing outside `internal/mcpserver` changes
behaviour in this batch, and the module-wide `go build ./...` at the batch boundary is what proves
the new `go.mod` requirement did not break another package's build. The translation layer is
written test-first: it is the single strongest TDD candidate in the task, its inputs and outputs
are pure values, and a regression there silently corrupts every position the server ever returns.
