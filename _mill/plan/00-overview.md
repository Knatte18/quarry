# Plan: Add `impact` verb for caller-context lookup

```yaml
task: "Add `impact` verb for caller-context lookup"
slug: "impact-verb"
approved: false
started: "20260828-090600"
parent: "main"
root: ""
verify: go build ./... && go vet ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: engine-impact-package
    file: 01-engine-impact-package.md
    depends-on: []
    verify: go build ./... && go test ./internal/quarryengine/...
  - number: 2
    name: facade-reexports
    file: 02-facade-reexports.md
    depends-on: [1]
    verify: go build ./... && go test ./quarry/...
  - number: 3
    name: cli-impact-verb
    file: 03-cli-impact-verb.md
    depends-on: [2]
    verify: go build ./... && go test ./internal/cli/...
  - number: 4
    name: docs-and-live-tier
    file: 04-docs-and-live-tier.md
    depends-on: [3]
    verify: go build ./... && go vet -tags lsp ./internal/cli/ && go test ./internal/quarryengine/ ./internal/cli/...
```

## Shared Decisions

### Decision: exact-identifier-set-is-fixed-here

- **Decision:** the engine package is `internal/quarryengine/impact`, exporting `Impact(ctx context.Context, opts query.Options) (Result, error)` plus the result types `Result`, `Target`, `Definition`, `Caller`, and `Range`.
  Their facade aliases are `ImpactResult`, `ImpactTarget`, `ImpactDefinition`, `ImpactCaller`, and `ImpactRange`.
  No options type and no options alias is added — `Impact` takes the existing `query.Options` (already re-exported as `quarry.Options`), and nothing is added to `query.Options`.
- **Rationale:** `quarry/facade_test.go`'s alias-pair, `init()` round-trip, and blank-identifier blocks are hand-enumerated and need exact identifiers.
  Fixing them here rather than leaving them to the implementer keeps batches 1 and 2 from drifting apart.
- **Applies to:** all batches

### Decision: json-key-disposition

- **Decision:** the emitted key set is fixed entirely by json struct tags on the `impact` package's result types, and `internal/cli` marshals them through the existing `structToFields` helper rather than hand-building maps.
  `ok` is injected by `output.Ok` and `resolution` is added CLI-side; neither is a field on `Result`.
  Exhaustively: on `Result`, `callers` is always present (an empty, non-nil array when there are none) and `target`/`definition` are `omitempty`.
  All five identity fields (`kind`, `name`, `owner`, `package`, `signature`) are `omitempty` on both `Target` and `Caller`.
  On `Definition`, `file` and `line` are always present, and `start_line`/`sigend_line`/`end_line`/`error` are `omitempty`.
  On a caller entry, `file` and `call_site_line` are always present, and `call_site_character`/`enclosing_range`/`error` are `omitempty`.
  On `Range`, `start_line` and `end_line` are always present (the object is omitted wholesale otherwise) and `sigend_line` is `omitempty`.
- **Rationale:** `structToFields` already honours every `omitempty` rule in one place, and pinning the key set by struct tags is the same discipline `toc/types.go`'s header comment records.
- **Rationale for the two range shapes:** `definition` carries its three range fields flat while a caller entry nests the identical triple under `enclosing_range`.
  The asymmetry is deliberate and load-bearing, not an oversight.
  On a caller entry the range is present-or-absent **as a unit**, and its absence is a meaningful, documented signal ("file-scope reference — no enclosing declaration") that must be distinguishable while `file` and `call_site_line` stay present;
  a pointer to a nested object expresses that one invariant in one place, whereas three flat `omitempty` line fields would scatter it across three tags and make a legitimately-absent range indistinguishable from three independently-zero numbers.
  On `definition` the three fields are individually omittable by design — outcomes 2 and 3 of the `three-outcome-degradation-rule` drop them while `file` and `line` remain — so there is no unit to nest, and nesting would add a level that is empty in exactly the cases the flat form already covers.
  This is also the shape `discussion.md`'s `json-key-set` decision fixes verbatim, so changing `definition` to a nested object would contradict a documented design decision rather than merely restyle the output.
- **Applies to:** engine-impact-package, cli-impact-verb

### Decision: identity-object-is-keyed-target-not-symbol

- **Decision:** the top-level identity object is keyed `target`, never `symbol`, in both the single-argument and the batch-entry shape.
  `runBatch` is not modified.
- **Rationale:** `runBatch` builds each batch entry as `{"symbol": arg, "status": ...}` and then merges the per-entry fields over it, so a field named `symbol` would overwrite the batch envelope's own query-string key — the key the shared batch contract identifies entries by across all existing verbs.
- **Applies to:** engine-impact-package, cli-impact-verb

### Decision: three-outcome-degradation-rule

- **Decision:** one enclosing-declaration lookup rule serves both the resolved symbol's own declaration site and every caller call site.
  It has three outcomes: (1) an enclosing toc symbol was found — range fields present, no `error`; (2) the file parsed but no listable declaration covers that line — range fields omitted, `error` **absent** (the file-scope outcome, not a failure); (3) the file could not be parsed at all (unreadable, invalid UTF-8, or no registered toc `Strategy`) — range fields omitted, `error` present.
  On the `definition` side, `target` is present only in outcome 1.
- **Rationale:** `toc.Kind`'s vocabulary is function/method/type only, so a package-level `var`/`const` or a struct field genuinely lands in outcome 2; folding it into outcome 3 would report a correct answer as a failure.
- **Applies to:** engine-impact-package, cli-impact-verb

### Decision: verification-is-best-effort-and-resolution-means-what-it-means-on-refs

- **Decision:** `Impact` never sets `SkipVerification` but never claims verification ran, and adds no verified/degraded signal of its own.
  `"resolution":"complete"` on `impact` means exactly what it already means on `refs` — *the language server returned every reference for the query as given* — and asserts nothing about per-caller verification having run, nor about enclosing ranges having resolved.
  That wording goes verbatim into the verb's Long help.
- **Rationale:** `refs` already carries the marker on a set that is never verified, so reading it as a verification claim would misread the existing contract.
  Making `impact` assert more would require changing `query.Callers`' return values, which is out of scope.
- **Applies to:** engine-impact-package, cli-impact-verb

### Decision: seam-splits-what-is-reused-and-what-is-re-implemented

- **Decision:** `resolveContext`, `buildOptions`, `parseQuery`, `inFileQuery`, `isWithinDir`, `structToFields`, `runBatch`, `batchStatus`/`statusRank`, `CwdFrom`, and `SetExit` are reused unchanged, with nothing added parallel to them.
  Four helpers are deliberately *not* on that list because all four are typed to `[]quarry.Reference` while `impact`'s result is its own struct: `emitLookupResult` and `classifyLookupError` gain small `impact`-typed counterparts in `internal/cli`; declaration exclusion is re-implemented engine-side inside `internal/quarryengine/impact` (it cannot reuse `filterUnexpectedCallers` at all, since the seam guard bans the engine from importing any `internal/*cli` package); and `--within` is applied CLI-side over `impact`'s own entry type.
- **Rationale:** the engine/CLI seam forces the declaration-exclusion duplication, and the type difference forces the other three.
  Consequences for the SDK contract: `quarry.Impact` returns a caller list that **already excludes** declaration sites, and is **not** `--within`-filtered.
- **Applies to:** engine-impact-package, cli-impact-verb

### Decision: doc-site-ownership-by-touching-batch

- **Decision:** the rule is "update every comment, doc block, or hand-enumerated test table that lists the engine package set or the verb set".
  Each site is owned by the batch that already edits that file, so no file is edited by two batches: `internal/quarryengine/layering_test.go` by batch 1; `quarry/facade.go` and `quarry/facade_test.go` by batch 2; `internal/cli/cli.go` and `internal/cli/cli_test.go` by batch 3; `README.md`, `internal/quarryengine/doc.go`, and `internal/quarryengine/seam_enforcement_test.go` by batch 4.
  The phrase "seven-package DAG" occurs in exactly three files — `internal/quarryengine/doc.go` line 50, `quarry/facade.go`'s header line 3, and `internal/quarryengine/seam_enforcement_test.go`'s header line 10 — but grepping only for that phrase is what originally missed sites that carry no such phrase, so batch 4 closes with a doc-audit sweep over the whole rule, not the phrase.
- **Rationale:** one file per batch keeps the DAG chain conflict-free and puts each doc edit next to the code change that made it stale.
- **Applies to:** all batches

### Decision: fixtures-live-at-repo-root-testdata

- **Decision:** the new fixture tree is repo-root `testdata/impactfixture/`, carrying its own `go.mod`, following `testdata/clockfixture`'s shape.
- **Rationale:** not stylistic — `internal/quarryengine/layering_test.go` walks *every* `.go` file under `internal/quarryengine/` with no `testdata` exemption, so a Go fixture at `internal/quarryengine/impact/testdata/` fails that guard with "no layering row declared".
- **Applies to:** engine-impact-package, docs-and-live-tier

### Decision: parse-cache-is-per-call-and-proven-by-a-counting-seam

- **Decision:** `Impact` keeps a per-call, absolute-path-keyed cache of toc results (both a successful `toc.FileTOC` and any per-file error), reached through an injectable parse function.
  It is not a package-level cache and does not survive the call.
  The cache is asserted by injecting a counting parse function and requiring exactly one parse per distinct file — never by comparing results and never by timing.
- **Rationale:** N callers in one file must parse that file once.
  A package-level cache would contradict `toc`'s documented "spawns no daemon and caches nothing" property.
  Identical results are consistent with N parses, so the identical-result proxy cannot prove the claim at all.
- **Applies to:** engine-impact-package

### Decision: parse-loop-cancellation-scope

- **Decision:** the per-caller-file loop checks `ctx.Err()` before each *cache-miss* parse and returns that error when the context is already cancelled.
  It never interrupts a parse already in flight.
  `--timeout` deliberately does not cover the parse phase and gains no new phase in its help text; no second timeout flag is introduced.
- **Rationale:** `toc.TOCFile` takes no `ctx`, so a real per-parse deadline would mean changing a shared engine signature.
  A between-files check is the honest bound actually available, and silently widening `--timeout` would make its existing help text false.
- **Applies to:** engine-impact-package

### Decision: go-verify-commands-carry-no-pythonpath-prefix

- **Decision:** every `verify:` in this plan is a native Go command with no `PYTHONPATH= ` prefix, scoped to the packages the batch touches, plus a `go build ./...` compile gate.
  The module-wide overview `verify:` is `go build ./... && go vet ./...`.
  `pipeline.done_gate` in `mill-config.yaml` is already `go test ./...` and is not changed by this task; `golangci-lint` is not installed in this worktree, so no lint command is added to the done gate.
- **Rationale:** the `PYTHONPATH= ` rule is a Python/mill-project rule; this is a Go module.
  The batch-scoped test commands keep each round cheap while `done_gate` catches anything outside their scope.
- **Applies to:** all batches

## All Files Touched

- `README.md`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/impact.go`
- `internal/cli/impact_lsp_test.go`
- `internal/cli/impact_test.go`
- `internal/quarryengine/doc.go`
- `internal/quarryengine/impact/enclosing.go`
- `internal/quarryengine/impact/enclosing_test.go`
- `internal/quarryengine/impact/impact.go`
- `internal/quarryengine/impact/impact_test.go`
- `internal/quarryengine/impact/types.go`
- `internal/quarryengine/layering_test.go`
- `internal/quarryengine/seam_enforcement_test.go`
- `quarry/facade.go`
- `quarry/facade_test.go`
- `testdata/impactfixture/billing/invoice.go`
- `testdata/impactfixture/go.mod`
- `testdata/impactfixture/pyfixture/shapes.py`
- `testdata/impactfixture/refund/refund.go`
- `testdata/impactfixture/tsfixture/client.ts`
