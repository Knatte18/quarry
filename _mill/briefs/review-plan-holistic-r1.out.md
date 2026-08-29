MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: plan/
date: 2026-08-29
```

## Findings

### [BLOCKING:design] Embedded `entryRaw` hijacks every entry's decode
**Location:** batch 3 card 13 (and cards 15, 17 which inherit it)
**Issue:** `entryRaw` carries an `UnmarshalJSON` method and is *embedded* in `lspEntry`, `symbolEntry`, and `nativeEntry`; Go promotes that method to the outer type, so `json.Unmarshal` into an entry sets `raw` and leaves `TextDocument`, `Position`, `Symbol`, `Within`, `File`, `Line`, `Character`, `Except`, and `Query` at their zero values — every handler would then see an empty entry and `lspEntry.query`/`nativeEntry.query` would always return the "no accepted form" error.
**Fix:** state the raw-capture mechanic in a form that does not promote `UnmarshalJSON` — e.g. each entry type declares its own `UnmarshalJSON` that stores the bytes and then decodes into a local alias type, or `Targets` is declared `[]json.RawMessage` at the input level and each entry decoded explicitly.

### [BLOCKING:consistency] `symbolField.Kind` typed `string`, source is `int`
**Location:** batch 2 card 9
**Issue:** the card declares `type symbolField struct { ...; Kind string; ... }` while requiring the JSON tags to match `symbolMatchFields` in `internal/cli/cli.go:793`, which emits `m.Kind` from `query.SymbolMatch.Kind int` (`internal/quarryengine/query/symbol.go:27`) — the LSP numeric SymbolKind.
**Fix:** declare `Kind int` so `workspace_symbol`'s `kind` stays the same JSON number the CLI emits, or state explicitly that the MCP layer converts the numeric kind to a name and where that mapping lives.

### [BLOCKING:decision] toc marshal-failure disposition unstated
**Location:** batch 5 card 22
**Issue:** `cli.StructToFields` (`toc.go:401`) and `cli.TOCDirEntries` (`toc.go:376`) both return an error, and the CLI maps each to `statusError` with the raw `err.Error()` (`toc.go:317-320`, `352-355`); card 22 names both calls but gives no disposition for their error return, while card 18 explicitly reroutes the analogous impact failure through `rewordMarshalFailure`.
**Fix:** state that a `StructToFields`/`TOCDirEntries` failure is that entry's `statusError` carrying the error verbatim, and that `rewordMarshalFailure` is `impact`-only.

### [NIT:consistency] `jsonschema-go` cannot become a direct v0.4.3 requirement as written
**Location:** batch 2 card 6 / card 10
**Issue:** card 6 requires `github.com/google/jsonschema-go v0.4.3` as a *direct* requirement but prescribes only `go get github.com/modelcontextprotocol/go-sdk@v1.7.0`, which pins whatever version the SDK selects and leaves it indirect; card 10 is its first importer and lists no `go.mod`/`go.sum` in `Edits:`.
**Fix:** either name `go get github.com/google/jsonschema-go@v0.4.3` in card 6, or drop the pinned version and add `go.mod`/`go.sum` to card 10's `Edits:`.

### [NIT:consistency] `workspace_symbol` gains an `ambiguous` branch the CLI has not
**Location:** batch 3 card 15
**Issue:** the card routes symbol outcomes through `classifyLSPError`, which card 9 defines as `classifyLookupError`'s predicates including `*quarry.ErrAmbiguousSymbol`; `classifySymbolError` (`cli.go:963`) has no ambiguous branch, and the card reasons explicitly about that function for the `resolution` key but not for this divergence.
**Fix:** state the added `ambiguous`/`candidates` disposition for `workspace_symbol` as a deliberate divergence with its rationale, as batch 4 does for `assert_no_callers`'s added `not_found`.

### [NIT:design] Card 25 asserts wording the SDK owns
**Location:** batch 6 card 25
**Issue:** "assert each message names the bound and the received length" asserts on a validation message produced by the SDK's schema validator for `minItems`/`maxItems`, which the plan neither generates nor specifies.
**Fix:** narrow the assertion to the observable contract — the result's error flag set and no handler run — or state where the message text comes from.

### [NIT:scope] Card 10 Context omits `internal/cli/tocconfig.go`
**Location:** batch 2 card 10
**Issue:** `docSentences.value()`'s "decimal string form of a number, the string unchanged for a string" rule is derived from `parseDocSentences`'s accepted grammar in `internal/cli/tocconfig.go:83`, which the Requirements name but the card's `Context:` (only `internal/mcpserver/mcpserver.go`) does not list.
**Fix:** add `internal/cli/tocconfig.go` to card 10's `Context:`.

## Verdict

REQUEST_CHANGES
Entry decoding is broken by design; two type/disposition mismatches with the CLI need pinning.
MILL_REVIEW_END
