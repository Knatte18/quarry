# Batch: quarry-native-lsp-tools

```yaml
task: "Add an MCP wrapper for quarry"
batch: "quarry-native-lsp-tools"
number: 4
cards: 4
verify: go test ./internal/mcpserver/...
depends-on: [3]
```

## Batch Scope

This batch delivers the two language-server-backed tools that keep quarry's own vocabulary rather
than LSP's: `impact` and `assert_no_callers`. It is one batch because both take the same flat entry
union — plain paths, 1-based line and column, no URI form and no `±1` conversion — and both need
the same per-entry `within` handling; `assert_no_callers` adds per-entry `except` and a call-wide
`noVerify` on top.

The external interface batch 6 consumes: the `nativeEntry` flat union and the two `register*`
functions now called from `NewServer`.

Batch-local decisions beyond `## Shared Decisions`:

- `impact`'s marshalled `quarry.ImpactResult` is nested under a `result` wrapper key rather than
  flattened into the entry. `impact.Result.Target` marshals to a top-level `target`, and flattening
  it the way `runBatch` does in `internal/cli/cli.go` would have it overwrite the envelope's echoed
  input `target`. The envelope's `target` always wins and always means "the input entry, echoed".
  The CLI does not hit this because its identity key is `symbol`.
- A violation is not a status. An `assert_no_callers` entry whose symbol resolved is
  `status: "found"` whether or not it has violating callers, and it carries `violation` and
  `callers` alongside. `violation: false` is emitted explicitly on the clean case rather than
  omitted, so the field never has to be inferred from an empty array. A violation never sets
  tool-level `isError` — it is an answer to the question asked.
- `assert_no_callers` gains a `not_found` status the CLI has no counterpart for. The CLI's
  `emitAmbiguousOrError` sends `ErrSymbolNotFound` to a plain error envelope because
  `assert-no-callers` is `cobra.ExactArgs(1)` and has no batch envelope at all; `not_found` here is
  an intended addition arriving with the batch envelope, using the same status vocabulary as the
  other six tools.

## Cards

### Card 17: Add flat quarry-native entry parsing

- **Context:**
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/translate.go`
  - `internal/mcpserver/schema.go`
  - `internal/cli/cli.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/nativeentry_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/nativeentry.go` declaring
  `type nativeEntry struct { entryRaw; File string; Line *int; Character *int; Symbol string; Within string; Except []string }`
  with json tags `file,omitempty`, `line,omitempty`, `character,omitempty`, `symbol,omitempty`,
  `within,omitempty`, `except,omitempty`, each with a `jsonschema` tag describing the property and
  naming which entry forms it participates in. Declare
  `func (e nativeEntry) query(targetDir string) (quarry.Query, error)` implementing the flat
  three-form union: `File` plus `Line` plus `Character` yields a `quarry.Query` whose `Pos` is a
  `quarry.Position` with `File` set to `cli.AbsOrJoin(targetDir, e.File)` and `Line`/`Character`
  taken as given — 1-based on this tool, with neither `toOneBased` nor `stripFileURI` applied;
  `Symbol` alone yields `quarry.Query{Symbol: e.Symbol}`; `File` plus `Symbol` yields
  `quarry.Query{InFile: &quarry.InFileQuery{File: cli.AbsOrJoin(targetDir, e.File), Name: e.Symbol}}`.
  Every other combination returns an error naming the three accepted forms. Declare
  `func exceptSet(targetDir string, except []string) map[string]bool` reproducing the inline
  composition in `assertNoCallersCommand`'s `RunE` in `internal/cli/cli.go` exactly: each path is
  resolved with `cli.AbsOrJoin` against the effective absolute target directory — never the process
  working directory — then `filepath.Clean`ed, and the cleaned paths are the map's keys. Document
  that `cli.FilterUnexpectedCallers` compares this map against `filepath.Clean(r.File)` on
  already-absolute `quarry.Reference.File` values, so a base mismatch makes every exemption
  silently fail to match and turns a sanctioned wrapper into a reported violation; and that this is
  reimplemented rather than exported because `FilterUnexpectedCallers` takes the built map, leaving
  no shared function to reuse. Create `internal/mcpserver/nativeentry_test.go` covering each legal
  form, each illegal combination, that a `file://` prefix is not stripped and no `±1` conversion is
  applied on this shape, and that `exceptSet` resolves a relative path against the given target
  directory rather than the process working directory.
- **Commit:** `feat(mcpserver): add the flat quarry-native entry union`

### Card 18: Add the `impact` tool

- **Context:**
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/result.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/facade.go`
  - `internal/cli/impact.go`
  - `internal/cli/toc.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/mcpserver/mcpserver.go`
- **Creates:**
  - `internal/mcpserver/tools_impact.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_impact.go` declaring
  `type impactInput struct { Targets []nativeEntry; Lang string; BuildTags string; TargetDir string }`
  with json tags `targets`, `lang,omitempty`, `buildTags,omitempty`, `targetDir,omitempty`, and
  `type impactEntry struct { Target any; Status string; Resolution string; Result map[string]any; Candidates []string; Error string }`
  with json tags `target`, `status`, `resolution,omitempty`, `result,omitempty`,
  `candidates,omitempty`, `error,omitempty`, plus
  `type impactOutput struct { Results []impactEntry }`. Declare
  `registerImpactTool(s *mcp.Server, cfg Config) error` following the registration shape card 14
  established. The handler resolves the call context once, then runs `runTargets` over
  `in.Targets`. Per entry it reports `unknownEntryKeys(entry.raw, "file", "line", "character", "symbol", "within")`
  as `status: statusError` — `except` is not accepted by this tool — then parses with
  `nativeEntry.query`, then calls `impactFn` with `callContext.options(in.Lang, query)`. On a nil
  error it applies `cli.FilterImpactWithin(result, entry.Within, ctx.TargetDir)` when
  `entry.Within` is non-empty, marshals the result with `cli.StructToFields`, and returns
  `status: statusFound` with the marshalled map under `Result` and `resolution: resolutionComplete`
  on the entry; a `cli.StructToFields` failure is that entry's `status: statusError` with
  `rewordMarshalFailure(err)` as the message, matching `classifyImpactError`'s own disposition for
  the same failure. A non-nil facade error routes through `classifyLSPError`. Positions inside the
  marshalled result are left exactly as the engine produced them — 1-based, unconverted. The tool's
  `Description` opens with a sentence stating that `line` and `character` are 1-based on this tool,
  then names the three accepted entry forms and states that the marshalled result is nested under
  `result` so it cannot collide with the echoed `target`. Edit `NewServer` in
  `internal/mcpserver/mcpserver.go` to call `registerImpactTool(s, cfg)` and return its error.
- **Commit:** `feat(mcpserver): add the impact tool`

### Card 19: Add the `assert_no_callers` tool

- **Context:**
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/result.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/facade.go`
  - `internal/cli/cli.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/mcpserver/mcpserver.go`
- **Creates:**
  - `internal/mcpserver/tools_assert.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_assert.go` declaring
  `type assertInput struct { Targets []nativeEntry; Lang string; BuildTags string; NoVerify bool; TargetDir string }`
  with json tags `targets`, `lang,omitempty`, `buildTags,omitempty`, `noVerify,omitempty`,
  `targetDir,omitempty` — `noVerify` documented as a whole-check mode, which is why it is call-wide
  while `within` and `except` are per-entry — and
  `type assertEntry struct { Target any; Status string; Violation *bool; Callers []referenceField; Candidates []string; Error string }`
  with json tags `target`, `status`, `violation,omitempty`, `callers,omitempty`,
  `candidates,omitempty`, `error,omitempty`, plus
  `type assertOutput struct { Results []assertEntry }`. `Violation` is a pointer so `false` is
  emitted explicitly on a clean `found` entry rather than dropped by `omitempty`. Declare
  `registerAssertTool(s *mcp.Server, cfg Config) error` following card 14's registration shape.
  Per entry the handler reports
  `unknownEntryKeys(entry.raw, "file", "line", "character", "symbol", "within", "except")` as
  `status: statusError`, parses with `nativeEntry.query`, builds
  `opts := callContext.options(in.Lang, query)` and sets `opts.SkipVerification = in.NoVerify`,
  then calls `callersFn`. On a non-nil error it routes through `classifyLSPError`, so an ambiguous
  symbol is `ambiguous` and `quarry.ErrSymbolNotFoundSentinel` is `not_found`. On success it
  applies `cli.FilterWithin(refs, entry.Within, ctx.TargetDir)` when `entry.Within` is non-empty,
  builds the exemption map with `exceptSet(ctx.TargetDir, entry.Except)`, calls
  `cli.FilterUnexpectedCallers(refs, declRefs, exemptions)`, and returns `status: statusFound` with
  `Violation` pointing at whether any violation remains and `Callers` carrying
  `referenceFieldsNative(violations)` — a non-nil empty slice when none remain, so it marshals as
  `[]`. No `resolution` key is ever emitted, matching the CLI, and no violation ever produces a
  whole-call failure. The tool's `Description` opens with a sentence stating that `line` and
  `character` are 1-based on this tool, then states that `except` and `within` are per-entry
  because each names paths sanctioned for that one symbol. Edit `NewServer` in
  `internal/mcpserver/mcpserver.go` to call `registerAssertTool(s, cfg)` and return its error.
- **Commit:** `feat(mcpserver): add the assert_no_callers tool`

### Card 20: Handler unit tests for `impact` and `assert_no_callers`

- **Context:**
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/result.go`
  - `quarry/facade.go`
  - `internal/cli/impact.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/tools_impact_test.go`
  - `internal/mcpserver/tools_assert_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create both test files driving the handlers directly with the facade seam
  variables replaced by stubs, restored with `t.Cleanup`, and `Config.StateDir` pinned to a
  `t.TempDir()`. `internal/mcpserver/tools_impact_test.go` asserts: an entry's envelope `target` is
  the echoed input while the marshalled result's own `target` is reachable under `result`, so
  neither overwrites the other; a `found` entry carries `resolution: "complete"`; per-entry
  `within` filters that entry's callers while leaving another entry's untouched; a stub whose
  result cannot be marshalled produces a per-entry `error` message beginning `impact: ` and never
  `toc: `; and positions in the result stay 1-based. `internal/mcpserver/tools_assert_test.go`
  asserts: a clean check is `status: "found"` with `violation: false` present and an empty
  `callers` array; a check with violations is `status: "found"` with `violation: true` and
  populated `callers`; neither sets a whole-call failure; a relative `except` path exempts the
  intended file, which is the regression that appears when the path is resolved against the process
  working directory instead; one entry's `except` never affects another entry's check; call-wide
  `noVerify` reaches `quarry.Options.SkipVerification` while the default leaves it false; and
  `quarry.ErrSymbolNotFoundSentinel` yields `status: "not_found"` rather than `error`.
- **Commit:** `test(mcpserver): cover the impact and assert_no_callers handlers`

## Batch Tests

`verify: go test ./internal/mcpserver/...` runs `nativeentry_test.go`, `tools_impact_test.go`, and
`tools_assert_test.go` alongside every earlier test in the package, which must keep passing because
cards 18 and 19 both edit `NewServer`. Scope is the one package these cards touch. The
`except`-base assertion and the `result`-wrapper assertion are the two regressions with no cheap
detection anywhere else: the first silently converts a sanctioned wrapper into a reported
violation, and the second silently destroys entry attributability.
