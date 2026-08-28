# Batch: engine-impact-package

```yaml
task: "Add `impact` verb for caller-context lookup"
batch: "engine-impact-package"
number: 1
cards: 7
verify: go build ./... && go test ./internal/quarryengine/...
depends-on: []
```

## Batch Scope

This batch delivers the whole engine half of the feature: a new `internal/quarryengine/impact`
package sitting at the top of the engine DAG (it imports the engine root, `query`, and `toc`), the
repo-root fixture tree its file-level tests parse, and the `internal/quarryengine/layering_test.go`
row plus `minPackageDirs` bump the DAG guard requires the moment the package directory exists.
It is one batch because the layering guard fails as soon as the new package directory appears
without a table row, so the package and its guard row cannot land in separate batches without a red
`verify:` in between.

The external interface batch 2 consumes is exactly:
`impact.Impact(ctx context.Context, opts query.Options) (impact.Result, error)` plus the exported
types `impact.Result`, `impact.Target`, `impact.Definition`, `impact.Caller`, and `impact.Range`.

Batch-local decision beyond `## Shared Decisions`: the package is split into three production files —
`types.go` (result types and json tags only), `enclosing.go` (the pure enclosing-symbol selection
plus the per-call parse cache), and `impact.go` (the package doc comment, the `buildResult` assembly
seam, and the `Impact` entry point). That split is what makes the pure selection function and the
assembly seam testable with no filesystem and no LSP, exactly as `query`'s own
`callersFromClient`/`symbolFromClient` seams are structured.

## Cards

### Card 1: Repo-root impact fixture tree

- **Context:**
  - `testdata/clockfixture/go.mod`
  - `testdata/clockfixture/builder/poll.go`
  - `internal/quarryengine/toc/python.go`
- **Edits:** none
- **Creates:**
  - `testdata/impactfixture/go.mod`
  - `testdata/impactfixture/billing/invoice.go`
  - `testdata/impactfixture/refund/refund.go`
  - `testdata/impactfixture/pyfixture/shapes.py`
  - `testdata/impactfixture/tsfixture/client.ts`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create a fixture tree mirroring `testdata/clockfixture`'s shape: its own module file declaring
  `module impactfixture` and the same `go 1.26` line the repo's own module file uses, so the tree is
  a self-contained module gopls can resolve and the parent module's `./...` patterns never compile
  it.
  Every declaration below carries a real doc comment, because the docstring-inclusive range is the
  central property the tests assert.

  In package `billing`: a type `Invoice` with an exported method `ApplyDiscount` whose doc comment is
  at least two lines long and sits immediately above the `func` line, so `toc.Symbol.Start` for
  `ApplyDiscount` is strictly less than its `func` line.
  Also declare a package-level `var DefaultRate` whose initializer expression is a plain literal —
  this is the file-scope target that exercises outcome 2 of the
  `three-outcome-degradation-rule` Shared Decision, since `toc.Kind` has no vocabulary for a
  package-level `var`.

  In package `refund`, importing `billing`: an exported function `ProcessRefund` with a doc comment,
  whose body calls `ApplyDiscount` on an `Invoice` value **twice**, on two distinct lines.
  Two call sites inside one enclosing function is what proves the one-entry-per-call-site rule and
  what makes the per-file parse cache observable.
  Also add a second exported function `Reconcile`, with a doc comment, that calls `ApplyDiscount`
  once, so the caller set spans two enclosing declarations within one file.

  In the Python fixture: a module declaring a class `Shape` containing a method `area`, both with
  docstrings. This is the class-and-method overlap the greatest-`Start` tie-break exists for; Go
  produces no overlapping ranges, so a Go fixture alone cannot exercise that rule.

  In the TypeScript fixture: any small, syntactically valid TypeScript declaration.
  Its only job is to be a file whose extension resolves to a language with no registered toc
  `Strategy`, exercising the per-entry degradation path.
- **Commit:** `test(fixtures): add repo-root impactfixture tree for the impact verb`

### Card 2: impact result types

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/query/refs.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/impact/types.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Declare `package impact` and the five exported result types, with json tags exactly as the
  `json-key-disposition` Shared Decision fixes them.

  `Range` carries `StartLine int` tagged `start_line`, `SigEndLine int` tagged
  `sigend_line,omitempty`, and `EndLine int` tagged `end_line`.
  Zero is `toc.Symbol.SigEnd`'s documented absent marker, which is why only that one is omitempty.

  `Target` carries `Kind toc.Kind` tagged `kind,omitempty`, `Name string` tagged `name,omitempty`,
  `Owner string` tagged `owner,omitempty`, `Package string` tagged `package,omitempty`, and
  `Signature string` tagged `signature,omitempty` — all five omitempty.

  `Definition` carries `File string` tagged `file`, `Line int` tagged `line`, `StartLine int` tagged
  `start_line,omitempty`, `SigEndLine int` tagged `sigend_line,omitempty`, `EndLine int` tagged
  `end_line,omitempty`, and `Error string` tagged `error,omitempty`.

  `Caller` carries `File string` tagged `file`, `CallSiteLine int` tagged `call_site_line`,
  `CallSiteCharacter int` tagged `call_site_character,omitempty`, the same five identity fields
  `Target` carries with the same omitempty tags, `EnclosingRange *Range` tagged
  `enclosing_range,omitempty`, and `Error string` tagged `error,omitempty`.
  `EnclosingRange` is a pointer specifically so the whole object is omitted rather than emitted as a
  zero-valued triple.

  `Result` carries `Target *Target` tagged `target,omitempty`, `Definition *Definition` tagged
  `definition,omitempty`, and `Callers []Caller` tagged `callers` with no omitempty, since an empty
  caller list must marshal as `[]` and never as `null`.

  Give each exported type and each non-obvious field a doc comment recording the contract it carries
  — in particular that the absence of `enclosing_range` on a caller entry means "file-scope
  reference: no enclosing declaration", not "lookup failed", and that a caller entry's `error` is the
  parse-or-language-failure case instead.
  Import only `internal/quarryengine/toc`, for `toc.Kind`.
- **Commit:** `feat(impact): add the impact package's result types`

### Card 3: Layering guard row for the impact package

- **Context:**
  - `internal/quarryengine/impact/types.go`
- **Edits:**
  - `internal/quarryengine/layering_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `impactPkg = rootPkg + "/impact"` to the import-path const block and correct that block's
  leading comment from "eight" to "nine" import paths.

  Add two `layeringTable` rows, one production and one test, both
  `{pkgDir: "impact", allowed: pathSet(rootPkg, queryPkg, tocPkg)}` with `isTestRow: false` and
  `isTestRow: true` respectively.
  Extend the `layeringTable` comment's enumeration of allowed directions with the new package's
  direction, stated the same way the existing entries are.

  Raise `minPackageDirs` from 8 to 9 and update its comment's directory enumeration to include the
  new package directory, keeping the comment's existing "this walk covers only the engine tree, so
  minPackageDirs here is the exact directory count" reasoning intact.

  Do not touch the walk logic, `allowedFor`, or the daemontest allowance — this card changes the
  table and its comments only.
- **Commit:** `test(layering): add the impact package row and raise minPackageDirs to nine`

### Card 4: Enclosing-symbol selection and the per-call parse cache

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/toc.go`
  - `internal/quarryengine/impact/types.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/impact/enclosing.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add the pure selection function
  `enclosingSymbol(symbols []toc.Symbol, line int) (toc.Symbol, bool)`.
  It returns the symbol with the greatest `Start` among those satisfying `Start <= line <= End`
  (bounds inclusive at both ends), and `false` when no symbol matches.
  State in its doc comment that the greatest-`Start` rule, not slice order, is the contract: `toc`
  documents `Symbols` as source-ordered by `Start`, which makes "last match wins" and "greatest Start
  wins" equivalent today, but only the latter survives a future ordering change.
  Record that the overlap case this rule exists for is Python and C# class-and-method nesting, where
  a `KindType` symbol spans every `KindMethod` symbol inside it, and that a grouped Go type
  declaration is *not* an overlap case because each spec's range is computed from the spec itself.

  Add the per-call parse cache. Declare an unexported struct type holding a parse function of type
  `func(path string) (toc.FileTOC, error)` and a `map[string]` cache whose value records both the
  successful result and any per-file error, keyed by the absolute path.
  Give it a constructor taking the parse function, where a nil argument selects the production
  parse — `toc.TOCFile(path, "", toc.Options{DocSentences: 0})`.
  `DocSentences: 0` is deliberate: `impact` emits no docstring text, and per `toc.TOCFile`'s own
  contract `Start`, `SigEnd`, and `End` are never affected by that setting.
  Give it a `resolve` method returning the cached result-or-error for a path, parsing exactly once
  per distinct path, and a predicate reporting whether a path is already cached — the cancellation
  check in card 5 needs to distinguish a cache hit from a cache miss.
  The injectable parse function is the counting seam card 7's cache test asserts through; it exists
  for that reason and its doc comment must say so.

  The cache is per-instance and per-call only.
  Do not add a package-level cache: that would contradict `toc`'s documented "spawns no daemon and
  caches nothing" property and would go stale against on-disk edits between calls.
- **Commit:** `feat(impact): add enclosing-symbol selection and the per-call parse cache`

### Card 5: Result assembly and the Impact entry point

- **Context:**
  - `internal/quarryengine/query/callers.go`
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/impact/types.go`
  - `internal/quarryengine/impact/enclosing.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/impact/impact.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Write the package doc comment at the top of this file: `impact` composes `query.Callers`' verified
  caller set with `toc`'s declaration ranges, sits above both in the engine DAG, introduces no
  options type of its own, and issues no LSP request `query.Callers` does not already issue.

  Add the assembly seam
  `buildResult(ctx context.Context, callers, declaration []query.Reference, resolver <the card 4 cache type>) (Result, error)`.
  It is separate from the entry point for the same reason `query`'s own `callersFromClient` seam is:
  the assembly is testable against a hand-built reference set with no LSP.

  Assembly rules, in order:

  Build a set of the `declaration` entries and use it to exclude the symbol's own declaration
  site(s) from the reported callers — the same set-membership rule `internal/cli`'s
  `filterUnexpectedCallers` applies for `assert-no-callers`, re-implemented here because the seam
  guard bans the engine from importing any `internal/*cli` package.
  Exclude only that set: a recursive call inside the target's own body is an ordinary caller whose
  enclosing symbol is the target itself, and must be kept.

  `target` and `definition`: when `declaration` is empty, omit both and still return the caller list
  with a nil error — their joint absence means "the language server returned no definition for the
  query position".
  Otherwise take `declaration[0]` (the set is already sorted, so this is deterministic), set the
  definition's `File` and `Line` from it, and run it through the same enclosing lookup every caller
  uses, applying the three outcomes of the `three-outcome-degradation-rule` Shared Decision:
  a resolver error fills the definition's `Error` and leaves `target` nil; no matching enclosing
  symbol leaves both the range fields and `Error` unset and `target` nil; a match fills
  `StartLine`/`SigEndLine`/`EndLine` and builds `target` from that same `toc.Symbol` plus the parsed
  file's declared package.
  The `target` object has exactly one provenance — that `toc.Symbol` — never the query string and
  never an LSP candidate.

  `callers`: initialise the slice as empty-but-non-nil so it marshals as `[]`.
  For each surviving reference, in input order, set `File`, `CallSiteLine`, and
  `CallSiteCharacter` from the reference (`query.Reference.File` is already absolute — do not
  re-resolve it against a working directory), then resolve its enclosing declaration through the same
  lookup: a resolver error sets the entry's `Error` and leaves `EnclosingRange` nil; no matching
  symbol leaves both unset; a match fills `EnclosingRange` and the identity fields.
  A file-scope entry still keeps `Package`, since the file parsed successfully and its declared
  package is known — only the symbol-level fields are absent there.
  Emit one entry per call site: two calls to the target inside one enclosing function produce two
  entries with identical enclosing ranges.

  Cancellation: immediately before resolving a reference whose file is not already cached, return
  `ctx.Err()` if it is non-nil.
  Do not attempt to interrupt a parse already in flight, and do not thread a deadline into
  `toc.TOCFile`.

  Sort the assembled entries by file, then line, then character, with a stable sort, so the ordering
  guarantee is local to this function rather than inherited from the caller.

  Add the entry point `Impact(ctx context.Context, opts query.Options) (Result, error)`: call
  `query.Callers(ctx, opts)` leaving `SkipVerification` at its zero value, return the zero `Result`
  and the error unchanged on failure, and otherwise delegate to `buildResult` with a freshly
  constructed cache.
  Its doc comment must record, per the
  `verification-is-best-effort-and-resolution-means-what-it-means-on-refs` Shared Decision, that it
  never sets `SkipVerification` but makes no claim that verification ran, and that the returned
  caller list already excludes declaration sites while being unfiltered by any directory scope.
- **Commit:** `feat(impact): add result assembly and the Impact entry point`

### Card 6: Enclosing-symbol selection unit tests

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/impact/enclosing.go`
  - `internal/quarryengine/toc/toc_test.go`
  - `internal/cli/assertnocallers_lsp_test.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/impact/enclosing_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Table-driven tests for `enclosingSymbol` over hand-built `toc.Symbol` slices, with no filesystem
  and no LSP. Cover, one case each: a line inside a function body resolves to that function; a line
  on the declaration's first line, a line on the docstring's first line, and a line on the last line
  all match, proving both bounds are inclusive; a line in the blank gap between two declarations does
  not match; a line before the first declaration (the package clause or imports region) does not
  match; overlapping ranges resolve to the greatest-`Start` symbol; and an empty symbol slice does
  not match.

  Add file-level tests for the card 4 parse cache against the card 1 fixture tree, resolving the
  repo root from the test file's own location the way
  `internal/cli/assertnocallers_lsp_test.go`'s `repoRoot` helper does, rather than from a working
  directory. Assert: the Go fixture's `ApplyDiscount` resolves from a line inside its body to a
  symbol whose `Start` is its docstring's first line and strictly less than its `func` line; a line
  at the package-level `var` in that same file resolves to no enclosing symbol; a nonexistent path
  yields a resolver error; the TypeScript fixture yields a resolver error naming an unsupported
  language; and the Python fixture resolves a line inside the method to the *method*, not the
  enclosing class.

  Assert the cache with the injected counting parse function from card 4: resolving the same path
  several times parses exactly once, and two distinct paths parse exactly twice.
  Do not assert on elapsed time, and do not use identical results as a proxy for a single parse —
  identical results are consistent with N parses and cannot establish the claim.
- **Commit:** `test(impact): cover enclosing-symbol selection and the parse cache`

### Card 7: Result assembly tests

- **Context:**
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/impact/types.go`
  - `internal/quarryengine/impact/enclosing.go`
  - `internal/quarryengine/impact/impact.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/impact/impact_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Test `buildResult` directly against hand-built `query.Reference` slices and an injected parse
  function, never through `Impact` — `Impact` needs a live language server and is proved by the
  live-tier test in batch 4 instead.

  Cover: every entry in the declaration set is excluded from the reported callers; a recursive
  self-call whose enclosing symbol is the target itself is retained; two call sites inside one
  enclosing function produce two entries with equal enclosing ranges and distinct call-site lines;
  entries come back sorted by file, then line, then character even when the input is not; and an
  empty caller set marshals to `[]` rather than `null` when the result is passed through
  `encoding/json`.

  Cover the empty-declaration-set-with-nil-error shape as its own case: both `target` and
  `definition` are omitted, the caller list is still returned, and the error is nil.

  Cover all three definition outcomes as three separate cases, never folded together:
  resolved (range fields and `target` present, no `error`); parsed but with no enclosing symbol,
  using a package-level `var` target (range fields omitted, `error` **absent**, `target` omitted);
  and no toc strategy for the declaring file (`file` and `line` present alongside `error`, no range
  fields, `target` omitted).
  The middle case is the one that distinguishes the two non-resolving outcomes and must be asserted
  separately from the third.

  Cover a caller file whose language has no registered strategy: the entry is still emitted with its
  call-site line, carries a per-entry `error`, has no enclosing range, and the call itself succeeds.

  Cover cancellation: a context already cancelled before the first cache-miss parse makes
  `buildResult` return that error rather than a partial result.
- **Commit:** `test(impact): cover result assembly, degradation shapes, and cancellation`

## Batch Tests

`verify: go build ./... && go test ./internal/quarryengine/...` compiles the whole module (catching
any signature drift the new package introduces) and runs every engine-tree test package.
That scope is deliberate rather than narrowed to the new package alone: this batch's card 3 edits
`internal/quarryengine/layering_test.go`, which lives in the engine *root* package, and the walk it
performs is precisely what must be re-run once a new package directory appears.
`internal/quarryengine/seam_enforcement_test.go`'s own guard runs in the same package and re-walks
both trees, confirming the new package imports neither `internal/output`, nor cobra, nor any
`internal/*cli` package — the check that would otherwise only be discovered in a later batch.

New test files: `internal/quarryengine/impact/enclosing_test.go` (card 6) and
`internal/quarryengine/impact/impact_test.go` (card 7).
Edited test file: `internal/quarryengine/layering_test.go` (card 3).

The temporary check the discussion asks for — verifying the layering guard still *fails* when the
new package imports something disallowed — is a local, uncommitted experiment the implementer runs
once by hand and reverts; it is deliberately not a committed test, since a committed version would
have to import a disallowed package to be meaningful.
