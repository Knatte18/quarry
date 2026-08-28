# Discussion: Add `impact` verb for caller-context lookup

```yaml
task: Add `impact` verb for caller-context lookup
slug: impact-verb
status: discussing
parent: main
```

## Problem

`quarry refs <symbol>` answers "where is this symbol referenced" as a list of `file:line:col`
positions. That is enough to *locate* a caller but not enough to *act* on one: after a signature
change, rewriting a caller requires the full line range of the declaration the call sits inside,
not the single line the call occupies. Today that means a follow-up `Read` per caller file, usually
over far more of the file than needed, and the reader has to guess where the enclosing function
starts and ends.

A second consumer wants the same data programmatically. An orchestration layer that runs rewrite
tasks in parallel (across git worktrees) needs a dependency graph: which symbols' changes ripple
into which other symbols. "Who calls this symbol" *is* that edge set (`symbol → caller` means
"caller depends on symbol"), so a structured caller list with enclosing ranges is directly
assemblable into a DAG for topological sort — no separate dependency-analysis component needed.

Why now: both primitives this needs already exist and landed recently. `quarry.Callers`
(`internal/quarryengine/query/callers.go`) already produces a verified caller set that filters
gopls' interface-method conflation, and the toc verbs (commit `76b5fe5`) already extract
declaration ranges whose `Start` includes the preceding docstring. `impact` is the composition of
those two, not new analysis capability. GitHub issue #5 is the source; it is closed and consolidated
into this task.

## Scope

**In:**

- A new `quarry impact <symbol|file:line:col>` verb, exposed on the cobra command tree alongside
  `refs`/`definition`/`symbol`/`assert-no-callers`.
- A new engine package `internal/quarryengine/impact`, sitting at the top of the engine DAG and
  importing `query` (for the caller set) and `toc` (for enclosing-declaration ranges).
- Facade re-exports in `quarry/facade.go`: a one-line `Impact` delegation plus type aliases for the
  new result types. No options alias is added — `Impact` takes the existing `quarry.Options`.
- JSON output carrying the resolved symbol's own definition range and one entry per caller call
  site, each with the caller's enclosing declaration's full line range.
- Both the resolved symbol's `definition` range and each caller's `enclosing_range` include the
  docstring immediately preceding the declaration, per toc's "declaration + docstring as one range"
  rule.
- Batch mode (2+ positional arguments), symbol-keyed, sharing the existing `runBatch` driver and
  the existing `found/not_found/ambiguous/error` status vocabulary and exit-code ranking.
- Flag parity with `refs`: `--target-dir`, `--lang`, `--timeout`, `--in-file`, `--within`,
  `--build-tags`.
- Layering-table row for the new package (`internal/quarryengine/layering_test.go`) and the
  `minPackageDirs` bump the same file's guard requires.
- Extending `quarry/facade_test.go` by hand — it enumerates what it checks rather than deriving it
  (see Technical context): add the `Impact` engine/facade alias pairs for each new result type to
  the `var` block, add their round-trip assignments to `init()`, add the blank-identifier func-type
  assignment for `Impact`, and correct the stale "the fourteen blank-identifier assignments" and
  "twenty-one aliased types" counts in the surrounding comments.
- Doc updates. **The rule, not just the list:** update *every* comment, doc block, or hand-enumerated
  test table that lists the engine package set or the verb set. Derive the sites from that rule —
  do not grep for the phrase "seven-package DAG" and stop, which is how the list below was first
  built and how it first missed sites that carry no such phrase. The verified sites under that rule
  are:
  - `README.md` — the verb list, **and** line 17's "All four verbs accept `--build-tags`", which is
    already wrong today (it lists six verbs) and becomes wronger with a seventh.
  - `internal/quarryengine/doc.go` — **four** edits, not two: the "# The package layout" section's
    "seven-package DAG" (line 50); the engine/CLI-split paragraph's package list at lines 41-46; a
    *third* package enumeration at lines 29-30 ("this package and its lsp, registry, daemon, query,
    treesitter, and toc subpackages"), which carries no "seven-package" phrase and was missed by the
    original grep; and the package-layout bullet list at lines 53-92, which needs a **new `impact`
    bullet** describing the package and its allowed imports.
  - `quarry/facade.go`'s header comment — the second "seven-package DAG" occurrence (line 3).
  - `internal/cli/cli.go`'s package doc comment — enumerates every verb, the exit-code contract, and
    the per-verb batch identity key.
  - `internal/quarryengine/seam_enforcement_test.go` — the **header comment** carries both the
    package enumeration (lines 2-3) and the third "seven-package DAG" occurrence (line 10). Its
    `minPackageDirs` comment enumerates the packages too. Its
    `TestEngineSeamInvariant_BannedImports` doc comment carries **no** enumeration and needs no edit.
  - `internal/quarryengine/layering_test.go` — the const block's "eight import paths" comment, the
    `layeringTable` comment enumerating the allowed directions, and the `minPackageDirs` comment
    enumerating the package directories.
  - `internal/cli/cli_test.go` — `TestBuildTagsFlag_RegisteredOnAllFourVerbs` is a hand-enumerated
    four-verb table (`refs`, `definition`, `symbol`, `assert-no-callers`). It must gain an `impact`
    row, and its name and doc comment must be corrected off "AllFourVerbs". This is the same
    silent-under-coverage category as `quarry/facade_test.go`: without the row, `impact`'s
    `--build-tags` registration is untested, **and** the table's second loop — which asserts
    `--no-verify` is registered on `assert-no-callers` alone — is the mechanical enforcement of the
    Scope "Out" claim that `impact` gains no `--no-verify` flag. Adding the row is what makes that
    claim actually checked rather than merely asserted in prose.

**Out:**

- `--depth` / transitive impact (who calls the caller). Real future need, explicitly deferred by
  issue #5 to keep v1's output size and latency predictable. Do not build a depth parameter, a
  recursion guard, or a cycle detector "ready for later".
- The DAG or topological sort itself. That belongs in the orchestrator consuming this JSON.
- Cross-file "might be related" heuristics (file proximity, naming similarity). Weaker signals than
  a real caller edge; conflating them with `impact`'s output is explicitly rejected.
- Any human-readable / table / text rendering. JSON is the source of truth; a presentation layer is
  a separate concern.
- New LSP surface. `impact` issues no LSP request `Callers` does not already issue.
- Changes to `refs`, `definition`, `symbol`, `assert-no-callers`, or the `toc` verbs' own output.
- A `--no-verify` flag. That flag is `assert-no-callers`-only and stays that way; `impact` never
  sets `SkipVerification`. Note this is not the same as "impact always runs verified" — see the
  `verification-is-best-effort` decision for what `Callers` actually guarantees.
- Any engine-side change to `query.Callers`' signature or return values, including adding a
  verified/degraded signal it does not report today.
- A daemon, cache, or index for the tree-sitter side. `toc` deliberately caches nothing across
  calls; the only caching here is within a single `Impact` call (see the per-call parse cache
  decision).

## Decisions

### impact-lives-in-its-own-engine-package

- Decision: implement the composition in a new package
  `internal/quarryengine/impact`, exporting `Impact(ctx context.Context, opts query.Options)
  (Result, error)`. It introduces **no options type of its own**: `opts` is the existing
  `query.Options`, already re-exported as `quarry.Options`, passed through to `query.Callers`
  unchanged. Nothing is added to `query.Options` either — in particular there is no `Within` field,
  because `--within` is applied CLI-side (see Technical context). Only the result types are new.
- Decision: the exported result types are named, in the engine package, `impact.Result` (the whole
  envelope), `impact.Target` (the resolved symbol's identity — named for the JSON key, and to avoid
  reading as a second `toc.Symbol`), `impact.Definition`, `impact.Caller`, and `impact.Range` (the
  `start_line`/`sigend_line`/`end_line` triple, shared by `Definition` and `Caller.EnclosingRange`).
  Their facade aliases follow the repo's existing `TOC`-prefix convention: `ImpactResult`,
  `ImpactTarget`, `ImpactDefinition`, `ImpactCaller`, `ImpactRange`. These exact identifiers are
  what the `quarry/facade_test.go` alias-pair, `init()` round-trip, and blank-identifier blocks must
  be extended with, so they are fixed here rather than left to the implementer.
  Add a layering row
  `{pkgDir: "impact", allowed: pathSet(rootPkg, queryPkg, tocPkg)}` for both the production and test
  rows in `internal/quarryengine/layering_test.go`, and raise that file's `minPackageDirs` constant
  from 8 to 9.
- Rationale: `internal/quarryengine/layering_test.go` pins the engine DAG and forbids `query` from
  importing `toc` and vice versa. `impact` needs both, so it must be a new package above them. It
  also gives a non-CLI (SDK/orchestrator) consumer a typed entry point through `quarry.Impact`,
  which the issue's DAG-building consumer explicitly wants.
- Rejected: composing in `internal/cli` — that layer is documented as "the sole place engine
  results become JSON", and joining caller positions against tree-sitter ranges is real engine
  logic, not envelope mapping. Rejected: relaxing the layering table so `query` may import `toc` —
  that inverts the DAG's intent (query is the LSP layer, toc is the parse layer; neither is above
  the other) for one caller's convenience.

### enclosing-range-reuses-toc-symbols

- Decision: find a line's enclosing declaration by calling `toc.TOCFile(path, "", toc.Options{
  DocSentences: 0})` and selecting the symbol whose `Start <= line <= End`. When more than one
  symbol matches, take the one with the greatest `Start` (the innermost / latest-starting match).
- Rationale: `toc.Symbol.Start` is *already* the first line of the docstring when the declaration
  has one, and `End` is the last line of the whole declaration — exactly the docstring-inclusive
  range the brief requires, with no new logic. `Symbol.SigEnd` comes free alongside it. `TOCFile`
  also resolves the language from the file's own extension, so a caller file's language is handled
  per file with no extra plumbing. `DocSentences: 0` is used because `impact` emits no docstring
  text; per `toc.TOCFile`'s own contract, `Start`/`SigEnd`/`End` are never affected by that setting.
- Rationale for the greatest-`Start` tie-break: **class↔method nesting in Python and C#** is the
  case that guarantees overlap. `pythonClassSymbols` (`internal/quarryengine/toc/python.go`) emits a
  `KindType` Symbol spanning the whole class body *and* a `KindMethod` Symbol per method inside it,
  so every line in a Python method is covered by two symbols; the method is the useful answer. Go's
  `Strategy.Symbols` never descends into a function body, so nested closures cannot produce a false
  innermost match there. Note a grouped Go `type ( ... )` declaration is **not** an overlap case —
  `goGroupedTypeSymbol` computes each spec's range from the spec itself, never from the enclosing
  `type_declaration`, so grouped specs are siblings, not nested. The rule is stated as a rule rather
  than left to slice order because `Symbols` is documented as source-ordered by `Start`, which makes
  "last match wins" and "greatest Start wins" equivalent — but only the latter survives a future
  ordering change.
- Rejected: a new tree-sitter walk inside `toc` — it would have to reimplement `CommentBlockAbove`'s
  docstring-association rule (see `docs/toc-docstring-association.md`) and could drift from it.
  Rejected: LSP `textDocument/documentSymbol` — its ranges do not include the preceding docstring,
  which is the brief's central requirement, and it costs an extra round trip per caller file.

### callers-come-from-quarry-callers-verified

- Decision: the caller set comes from `query.Callers(ctx, opts)` with `SkipVerification` left false
  (the zero value). Its second return value, the declaration set, is used to exclude the symbol's
  own declaration site(s) from the reported callers, exactly as `internal/cli`'s
  `filterUnexpectedCallers` does for `assert-no-callers`.
- Rationale: `Callers` already runs the per-reference `textDocument/definition` verification that
  removes gopls' conservative interface-method conflation (documented at length in
  `refsCommand`'s Long help and `assert-no-callers`'). Using raw `References` would reintroduce that
  noise into a verb whose entire value is "these are the places you must rewrite". `Callers` also
  issues every LSP call on one connection, so `impact` costs one connection, not two.
- Rejected: `query.References` — noisier for no benefit. Rejected: exposing `--no-verify` on
  `impact` — that flag exists so `assert-no-callers` can reproduce its own pre-fix behaviour; a new
  verb has no such history to reproduce.

### verification-is-best-effort

- Decision: `impact` never sets `SkipVerification`, but it does **not** claim that verification ran.
  `callersFromClient` (`internal/quarryengine/query/callers.go`) silently sets `verify = false` —
  returning the raw, unfiltered reference set with a **nil** error — in four cases: the
  declaration-side `textDocument/definition` errored; that call returned zero locations;
  `client.SupportsImplementation()` reports false; or `textDocument/implementation` errored.
  `impact` accepts that behaviour as-is and adds no signal of its own.
- Decision: consequently `"resolution":"complete"` on `impact` means exactly what it already means
  on `refs` — *the language server returned every reference for the query as given* — and asserts
  nothing about per-caller verification having run, nor about enclosing ranges having resolved.
  Document that wording verbatim in the verb's Long help, next to the existing `--within`
  interface-method-conflation note, since that note is precisely the case where an unverified set is
  noisy.
- Rationale: `refs` already carries `"resolution":"complete"` on a set that is *never* verified, so
  reading the marker as a verification claim would be a misreading of the existing contract, not a
  new gap introduced here. Making `impact` alone assert more than `refs` does would need `Callers`
  to report whether it verified — an engine signature change, out of scope for this task.
- Rejected: surfacing a per-result `verified`/`degraded` field — requires changing `Callers`' return
  values, which no current caller needs and which would ripple into `assert-no-callers`. Rejected:
  suppressing `"resolution"` on `impact` — that would make `impact` the one verb without the trust
  marker, for a property the marker never claimed.

### resolved-symbol-definition-range

- Decision: the top-level `definition` field is built from the *first* entry of `Callers`'
  declaration set (sorted, so deterministic), run through the **same** enclosing-range lookup as
  every caller entry. It always carries `file` and `line` (the raw 1-based declaration line the LSP
  returned). Beyond that there are **three** distinct outcomes, mirroring the caller side's own
  two-way split rather than collapsing them into one "error" case:
  1. **Resolved** — an enclosing toc symbol was found. `start_line`/`sigend_line`/`end_line` present;
     no `error`; `target` present.
  2. **Parsed, but no enclosing symbol** — the file parsed fine and simply has no listable
     declaration covering that line. This is the *file-scope* outcome and is **not** an error, so it
     is symmetric with `no-enclosing-declaration-is-not-an-error`: the range keys are omitted,
     `error` is **absent**, and `target` is omitted. This is a real case, not a theoretical one:
     `toc.Kind`'s vocabulary is function/method/type only, so a package-level `var`/`const`, a
     struct field, or a Go `const` block target lands here.
  3. **Parse or language failure** — unreadable file, invalid UTF-8, or no registered toc
     `Strategy`. The range keys are omitted, `error` **is** present carrying the reason, and `target`
     is omitted.
- Decision: the top-level `target` object (the resolved symbol's own identity — `kind`, `name`,
  `owner`, `package`, `signature`) has exactly one provenance: it is the *same* `toc.Symbol` the
  `definition` enclosing lookup found. It is omitted entirely when that lookup found nothing. It is
  never derived from the query string, and never from the LSP `workspace/symbol` candidate.
- Decision: when `Callers` returns an **empty declaration set with a nil error**, both `target` and
  `definition` are omitted entirely and the call still succeeds with its caller list. Their joint
  absence means "the language server returned no definition for the query position". Both keys are
  therefore on the `omitempty` list. Precision on when this happens: `callersFromClient` assigns
  `declaration = toSortedReferences(defLocs)` *before* the implementation-side checks, so only the
  **two definition-side** silent-skip cases (`defErr != nil`, and `len(defLocs) == 0`) leave it
  empty. The other two cases from `verification-is-best-effort` (`!SupportsImplementation()`, and
  `implErr != nil`) are reached only when `len(defLocs) > 0`, so the declaration set is non-empty
  there and this branch does not apply — those two degrade verification only, never `definition`.
- Rationale: the declaration site is just another file position; running it through the same lookup
  and the same two degradation paths means there is exactly one rule to implement, one rule to test,
  and no shape the contract leaves undefined. Reusing `Callers`' already-performed
  `textDocument/definition` result costs nothing. Running it through the enclosing lookup is what
  makes the definition range docstring-inclusive, satisfying the brief's requirement that *both*
  ranges include the docstring.
- Rejected: a separate `query.Definition` call — a redundant LSP round trip on a second connection,
  and it would not fix the empty-set case (the same swallowed error produces it). Rejected: emitting
  `definition: null` or an error envelope for the empty-set case — the caller list is still a
  complete, useful answer, and failing the whole call over a missing definition would discard it.
  Rejected: emitting only the raw `file:line:col` — fails the brief.

### caller-identity-is-structured-not-a-qualified-string

- Decision: each caller entry carries the enclosing symbol's structured toc fields —
  `kind` (`function`/`method`/`type`), `name` (bare), `owner` (omitted when empty),
  `package` (omitted when empty), and `signature` — rather than a single synthesized dotted string.
- Rationale: issue #5's example shows `"symbol": "internal/billing.ProcessRefund"`, a Go import-path
  qualification. Quarry cannot derive an import path without `go/packages` (deliberately not wired,
  per `internal/quarryengine/doc.go`'s "what this engine deliberately does not do"), and the concept
  does not transfer to Python or C#. `toc.Symbol` already gives `Name`/`Owner`/`Kind`/`Signature`
  and `toc.FileTOC` gives the declared `Package` — all free, all honest. A consumer that wants a
  display string composes one itself; `toc.Symbol.Owner`'s own doc comment records this exact
  reasoning ("a caller composes the qualified form itself").
- Rationale for including `signature`: the human-facing consumer's question is "what do I need to
  rewrite", and the verbatim signature answers it without a Read. It is already computed by
  `TOCFile`; omitting it would be gratuitous.
- Rejected: synthesizing a qualified name — requires per-language qualification rules the engine
  does not have, and would be wrong (not merely absent) for the non-Go languages.

### one-entry-per-call-site

- Decision: emit one `callers[]` entry per call site, each with a singular `call_site_line` (and
  `call_site_character`), sorted by file then line then character. Two calls to the target inside one
  enclosing function produce two entries with identical `enclosing_range`.
- Rationale: this is the shape issue #5 specifies, it is lossless, and grouping by
  `(file, enclosing_range)` is a one-liner for the DAG consumer. The reverse — recovering individual
  call-site lines from a grouped shape — is impossible.
- Rejected: one entry per enclosing declaration with a `call_site_lines[]` array — deviates from the
  specified contract and discards nothing the consumer gains.

### no-enclosing-declaration-is-not-an-error

- Decision: a reference with no enclosing declaration (a package-level `var` initializer, a struct
  field type, an import line) is still emitted as a caller entry, with `call_site_line` present and
  the `enclosing_range` key and the identity fields (`kind`/`name`/`owner`/`signature`) all omitted.
  Documented: the absence of `enclosing_range` means "file-scope reference — no enclosing
  declaration", not "lookup failed".
- Rationale: it is a genuine dependency edge and dropping it would silently under-report impact,
  the one failure mode this verb must not have.
- Rejected: dropping such references — loses real edges. Rejected: treating it as an error — it is
  a correct, complete answer about a real reference.

### unsupported-caller-language-degrades-per-entry

- Decision: when a caller file's language has no registered toc `Strategy` (a `.ts` or `.rs` file),
  or the file is unreadable, or is not valid UTF-8, emit the caller entry with `call_site_line`, no
  `enclosing_range`, and a per-entry `error` string carrying the reason. The `impact` call itself
  still succeeds.
- Rationale: mirrors `toc dir`'s established rule — a file that cannot be parsed is "still listed,
  never skipped", and the listing never fails because of one bad file. A Go symbol called from a
  TypeScript file is a real edge worth reporting even without a range.
- Rejected: failing the whole command — one unsupported caller file would destroy an otherwise
  complete answer. Rejected: silently omitting the entry — same under-reporting failure as above.

### recursive-self-calls-are-kept

- Decision: exclude only the declaration set returned by `Callers`. A recursive call inside the
  target symbol's own body is reported as an ordinary caller whose enclosing symbol is the target
  itself.
- Rationale: a self-loop is a real edge; the consumer detects it by comparing the caller's identity
  to the query's, and a DAG builder needs to know about it explicitly (it is exactly the case that
  breaks a naive topological sort).
- Rejected: filtering self-calls — hides information the DAG consumer specifically needs.

### per-call-parse-cache

- Decision: `Impact` keeps a `map[string]` cache of per-file toc results (both the successful
  `FileTOC` and any per-file error) for the duration of a single `Impact` call, keyed by absolute
  path. It is not a package-level cache and does not survive the call.
- Rationale: N callers in one file must parse that file once, not N times. A package-level cache
  would contradict `toc`'s documented "spawns no daemon and caches nothing" property and would go
  stale against on-disk edits between calls.
- Rejected: parsing per caller — quadratic on a file with many call sites. Rejected: a process-wide
  cache — staleness hazard for a long-lived SDK consumer, and outside this task's scope.

### parse-loop-cancellation-and-timeout-scope

- Decision: the per-caller-file parse loop checks `ctx.Done()` **between files** (before each
  cache-miss parse) and returns `ctx.Err()` when the context is already cancelled. It does not
  attempt to interrupt a parse already in flight.
- Decision: `--timeout` deliberately does **not** cover the parse phase. Its help text already
  scopes it to the LSP request phases ("initialize, resolve, references"), and that stays literally
  true — no new phase is added to it, and no second timeout flag is introduced.
- Rationale: `toc.TOCFile` takes no `ctx` at all (`internal/quarryengine/toc/toc.go`), so there is
  nothing to thread a deadline into without changing toc's signature — which is out of scope and
  would affect the toc verbs. A between-files check is the honest bound that is actually available:
  it makes a cancelled `impact` stop promptly on a high-fan-in symbol (the verb's own target case,
  where the loop can run over many files) without pretending to a per-parse deadline the backend
  cannot enforce. Parse cost is linear in file size and the per-call cache bounds the loop to one
  parse per distinct file, so unbounded growth comes from file *count*, which the cancellation check
  covers.
- Rejected: threading `ctx` into `toc.TOCFile` — changes a shared engine signature for one new
  caller. Rejected: extending `--timeout` to wrap the parse loop — its documented meaning is
  per-LSP-phase, and silently widening it would make the existing help text false. Rejected: a
  separate `--parse-timeout` flag — a flag for a phase that cannot honour it mid-parse.

### cli-shape-mirrors-refs

- Decision: `impactCommand()` in `internal/cli/cli.go` mirrors `refsCommand()`: same
  cwd/target-dir/config/state-dir resolution via `CwdFrom` + `resolveContext` + `buildOptions`, and
  the same `buildQuery` closure *shape* — note `buildQuery` is not a shared helper to call: it is a
  per-command local closure, duplicated today at both `refsCommand` and `definitionCommand`, so
  `impact` re-creates it the same way (dispatching to `inFileQuery` when `--in-file` is set and
  `parseQuery` otherwise, which is what makes `--in-file` compose with batch mode and keeps a
  positional from ever being position-parsed under `--in-file`). Plus the same `--within` filter
  applied to the caller set, the
  same `--build-tags` handling, `Args: cobra.MinimumNArgs(1)`, and batch mode via the existing
  `runBatch` (symbol-keyed).
- Decision: exit-code contract identical to `refs`/`definition` — 0 found, 1 not found or engine
  error, 2 ambiguous (envelope carries `candidates`, `ok:true`). Batch mode ranks
  `found(0) < not_found(1) < ambiguous(2) < error(3)` via the existing `statusRank`.
- Decision: a successful lookup carries `"resolution":"complete"`, scoped exactly as the
  `verification-is-best-effort` decision defines it — the caller set is LSP-resolved and exhaustive
  for the query as given. It does not cover the enclosing ranges (tree-sitter-derived, absent per
  the two degradation decisions above) and it does not assert that per-caller verification ran.
- Rationale: consistency is the point; a verb that resolves symbols differently from its three
  siblings is a trap. `--within` is genuinely useful here for the same interface-method-scoping
  reason it exists on `refs`.
- Rejected: single-argument-only — every other verb supports batch mode and the DAG consumer
  querying many symbols is precisely the batch case. Rejected: omitting `--in-file` — it is the
  cheapest way to disambiguate a common name, and its absence would be an arbitrary gap.

### zero-callers-is-success

- Decision: a symbol that resolves but has no callers exits 0 with `"callers": []` (an empty,
  non-nil array, never `null`).
- Rationale: "nothing depends on this" is a true, useful answer, not a failure — and it is the
  answer a DAG builder needs for a leaf node. `toc`'s `FileTOC.Symbols` sets the same non-nil-slice
  precedent.
- Rejected: exit 1 / `not_found` — conflates "symbol does not exist" with "symbol has no callers",
  which are opposite facts.

### json-key-set

- Decision: the emitted key set is fixed by json tags on the `impact` package's result structs, and
  `internal/cli` marshals them through the existing `structToFields` helper (`internal/cli/toc.go`)
  rather than hand-building maps. Shape:

```json
{
  "ok": true,
  "resolution": "complete",
  "target": {"kind": "method", "name": "ApplyDiscount", "owner": "Invoice", "package": "billing", "signature": "func (i *Invoice) ApplyDiscount(pct float64) error"},
  "definition": {"file": "/abs/internal/billing/invoice.go", "line": 91, "start_line": 88, "sigend_line": 91, "end_line": 102},
  "callers": [
    {
      "file": "/abs/internal/billing/refund.go",
      "call_site_line": 15,
      "call_site_character": 21,
      "kind": "function",
      "name": "ProcessRefund",
      "package": "billing",
      "signature": "func ProcessRefund(id string) error",
      "enclosing_range": {"start_line": 8, "sigend_line": 10, "end_line": 28}
    }
  ]
}
```

- Decision: the top-level identity object is keyed `target`, **not** `symbol`. `runBatch`
  (`internal/cli/cli.go`) builds each batch entry as `{"symbol": arg, "status": ...}` and then
  merges the per-entry fields over it, so a field named `symbol` would overwrite the batch
  envelope's own query-string `symbol` key — the key the batch contract identifies entries by. No
  sibling verb's field set has this collision, so `runBatch` is not the thing to change. Renaming
  the identity object gives one shape for both arg-count modes: single-arg carries `target` at the
  envelope's top level, and a batch entry carries `symbol` (the query string, per the shared batch
  contract) alongside `target`, `definition`, and `callers`.
- Decision: the full per-key disposition, stated exhaustively so the two degraded shapes above are
  actually emittable:
  - **Envelope, split by which layer owns each key.** Only `target`, `definition`, and `callers` are
    struct-tag-fixed on `impact.Result`; `callers` is always present (an empty, non-nil array when
    there are none) and `target`/`definition` are `omitempty`. `ok` and `resolution` are **not**
    engine fields: `ok` is injected by `output.Ok`, and `resolution` is added CLI-side exactly as
    `emitLookupResult` adds it for `refs`. Putting `resolution` on `impact.Result` would leak a
    CLI-layer trust marker into `quarry.Impact`'s SDK contract and would violate the engine/CLI seam
    the whole package layout exists to keep.
  - **`target` / caller identity fields** (`kind`, `name`, `owner`, `package`, `signature`): *all
    five* are `omitempty`. `kind` and `name` being omitempty is what makes a file-scope caller entry
    emittable — without it the entry emits `"kind":"","name":""`. A file-scope entry **does** keep
    `package`, since the file parsed successfully and its declared package is known; only the
    symbol-level fields are absent.
  - **`definition`:** `file` and `line` always present when `definition` itself is; `start_line`,
    `sigend_line`, `end_line`, and `error` are all `omitempty`.
  - **Caller entry:** `file` and `call_site_line` always present; `call_site_character`,
    `enclosing_range`, and `error` are `omitempty`.
  - **`enclosing_range`:** `start_line` and `end_line` always present when the object itself is
    (the object is omitted wholesale otherwise); `sigend_line` is `omitempty`, zero being
    `toc.Symbol.SigEnd`'s documented absent marker.
- Decision: `file` paths are absolute, matching `refs`/`definition`/`assert-no-callers`
  (`referenceFields` emits `quarry.Reference.File` unchanged, which is always absolute). This
  deliberately differs from `toc dir`'s caller-relative `path` composition, which exists so an entry
  round-trips into `toc file`; `impact` has no such round-trip need and consistency with the other
  LSP-backed verbs wins.
- Rationale: `structToFields` already exists for exactly this, honours `omitempty` in one place, and
  keeps the key set pinned by struct tags rather than by scattered map literals — the same
  discipline `toc/types.go`'s header comment records.
- Rejected: hand-built `map[string]any` per call site — restates every omitempty rule by hand.

### sigend-is-included

- Decision: both `definition` and `enclosing_range` carry `sigend_line` alongside `start_line` and
  `end_line`, omitted when zero (the absent marker, e.g. a Go type alias with no body).
- Rationale: `toc.Symbol.SigEnd` is already computed; `start..sigend` lets a reader see a caller's
  docstring and signature without pulling its body, which is the documented two-phase discovery flow
  `toc file`'s help text recommends. Free, and directly serves the "what do I need to read" consumer.
- Rejected: start/end only — discards a useful, already-computed field for no saving.

## Technical context

**Existing primitives to reuse, do not reimplement:**

- `internal/quarryengine/query/callers.go` — `Callers(ctx, opts) ([]Reference, []Reference, error)`
  returns `(verified caller references, declaration set)`. It issues definition → implementation →
  references → per-reference definition, sequentially on one connection (`lsp.Client` is
  single-flight; concurrent calls on it drop each other's responses). Do not parallelise.
- `internal/quarryengine/query/refs.go` — `Reference{File string; Line, Character int}` with 1-based
  line and 1-based UTF-16 character. `Options` carries `Registry`, `TargetDir`, `StateDir`, `Lang`,
  `Query`, `Timeout`, `BuildTags`, `SkipVerification`. `Query` is one of `InFile`, `Pos`, `Symbol`.
- `internal/quarryengine/toc/toc.go` — `TOCFile(path, langOverride string, opts Options)
  (FileTOC, error)`. Language resolves from the extension when `langOverride` is empty. Returns a
  wrapped `quarryengine.ErrLanguageUnsupported` for an unknown extension or an unimplemented
  language; a plain error for a read failure or invalid UTF-8.
- `internal/quarryengine/toc/types.go` — `Symbol{Kind, Name, Owner, Signature, Docstring, Start,
  SigEnd, End}`; `FileTOC{Header, Language, Package, Symbols, Partial}`. `Start` is the docstring's
  first line when the docstring is a sibling of the declaration, the declaration's first line
  otherwise. All line numbers 1-based inclusive. `SigEnd == 0` means "no body at all".
- `internal/cli/cli.go` — `resolveContext`, `buildOptions`, `parseQuery`, `inFileQuery`,
  `isWithinDir`, `emitAmbiguousOrError`, `runBatch`, `batchStatus`/`statusRank`, `CwdFrom`,
  `SetExit`. Reuse all of these unchanged; add nothing parallel to them.
- **Deliberately *not* on that reuse list, because all four are typed to `[]quarry.Reference` and
  `impact`'s result is its own struct: `filterWithin`, `filterUnexpectedCallers`,
  `classifyLookupError`, and `emitLookupResult`.** "Add nothing parallel" applies to the first list,
  not to these — each needs a small `impact`-typed counterpart, and that is expected work, not
  duplication to avoid:
  - `emitLookupResult`'s single-arg envelope mapping and `classifyLookupError`'s batch mapping are
    reimplemented for `impact`'s result type, preserving the *same* three-way error routing
    verbatim: `*quarry.ErrAmbiguousSymbol` (via `errors.As`) → `candidates` + `ok:true` + exit 2 /
    `statusAmbiguous`; `quarry.ErrSymbolNotFoundSentinel` (via `errors.Is`) → exit 1 /
    `statusNotFound` with no extra fields; anything else → `output.Err` exit 1 / `statusError` with
    an `error` field. Do not invent a fourth branch or reorder the three.
  - The remaining two split by the seam instead of being reimplemented wholesale:
  - **Declaration exclusion runs engine-side, inside `internal/quarryengine/impact`.** It cannot
    reuse `filterUnexpectedCallers` at all — `seam_enforcement_test.go` bans the engine from
    importing any `internal/*cli` package. The impact package re-implements the same small
    set-membership rule over `query.Reference` values *before* building its own entries. This
    duplication is deliberate and forced by the seam, not an oversight to collapse later.
    Consequence, stated so the SDK contract is unambiguous: `quarry.Impact` returns a caller list
    that **already excludes** declaration sites.
  - **`--within` is applied CLI-side**, over `impact`'s own entry type. It reproduces
    `filterWithin`'s *three* within-value normalization steps before any comparison — join onto the
    resolved target directory when the flag value is relative, then `filepath.Abs`, then
    `filepath.Clean` — and only then calls the existing type-agnostic `isWithinDir` helper (which
    takes two strings) per entry. The normalization is load-bearing, not ceremony:
    `filterWithin`'s own comment records the failure mode — every compared path is absolute, so an
    un-normalized relative `--within internal/foo` makes `filepath.Rel` error and silently filters
    **every** caller out, producing an empty-but-successful answer. Copying the helper's rule
    without its normalization would reintroduce exactly that bug.
    Consequence: `quarry.Impact` is **not** `--within`-filtered; `--within` is a CLI flag with no
    engine option behind it.
- `internal/cli/toc.go` — `structToFields` for struct→`map[string]any` marshalling.
- `internal/output/output.go` — `Ok`, `Err`, `ErrFields`.

**Guards that will fail if the new package is added carelessly:**

- `internal/quarryengine/layering_test.go` — walks every `.go` file (production *and* `_test.go`)
  under `internal/quarryengine/` and fails on any intra-engine import not listed in
  `layeringTable`. A new package with no row fails with "no layering row declared". It also asserts
  `len(visitedDirs) >= minPackageDirs` (currently 8) — a new package directory must raise it to 9,
  or the guard silently under-covers.
- `internal/quarryengine/seam_enforcement_test.go` — the new package must not import
  `internal/output`, `spf13/cobra`, or any `internal/*cli` package. Its own `minPackageDirs` needs
  **no bump**: its comment states the floor is deliberately kept one below the real count (8, while
  the two-tree walk actually visits 9), so adding a package keeps it satisfied. Only its stale
  package *enumerations* in comments need updating (see the doc-update list).
- `quarry/facade_test.go` — **does not** enforce the alias/delegation property automatically. It is
  a hand-maintained set of compile-time checks over an *enumerated* list: a `var` block of
  engine/facade variable pairs, an `init()` that round-trip-assigns each pair, a `var` block of
  blank-identifier func-type assignments (one per delegating function), and
  `TestFacadeSentinels_Identity`'s table. A new facade declaration is covered by **nothing** until
  those blocks are extended by hand — the guard silently under-covers rather than failing. Extending
  them is in scope (see the Scope "In" list).

**Gotchas found during exploration:**

- Coordinate systems differ: `quarryengine.Position.Character` is a 1-based *byte* column while
  `quarry.Reference.Character` is a 1-based *UTF-16* column, so the two coincide only on a pure-ASCII
  line. This is why `assert-no-callers` excludes declarations using `Callers`' returned declaration
  set rather than the caller-supplied position — do the same. The *line* number is unambiguous in
  both, and the enclosing-range lookup uses only lines, so it is unaffected.
- `Reference.File` is already absolute (`trimFileURI` on the LSP URI); `toc.TOCFile` takes a plain
  path. No conversion needed, but do not re-resolve against cwd.
- `toc`'s Go strategy never descends into a function body, so a func literal or a type declared
  inside a body is not a listable symbol. A call site inside a closure therefore resolves to the
  enclosing top-level function — which is the desired answer.
- A grouped Go `type ( ... )` declaration produces one `Symbol` per spec, with ranges covering the
  spec only, never the group — so grouped specs do **not** overlap. The overlap that the
  greatest-`Start` tie-break actually exists for is Python/C# class↔method nesting, where a class
  Symbol spans every method Symbol inside it.
- `FileTOC.Partial` is true when the parse hit a syntax error. Decide nothing new here: propagate
  nothing, but do not treat `Partial` as an error — a partial parse still yields usable ranges, and
  a missing enclosing symbol in a partial file degrades to the existing "no enclosing declaration"
  path.
- `toc` is deliberately not reached through `resolveContext`/`buildOptions` in the CLI (see
  `internal/cli/toc.go`'s header comment) because it needs no registry and no state dir. `impact`
  *does* need both — it makes LSP calls — so it uses the `refs` path, not the `toc` path.
- The phrase "seven-package DAG" appears in three files, not one: `internal/quarryengine/doc.go`
  (line 50), `quarry/facade.go`'s header comment (line 3), and
  `internal/quarryengine/seam_enforcement_test.go`'s header comment (line 10). All three need
  updating to eight, and the latter two are easy to miss because the doc-update chore reads as a
  `doc.go`-only change. `doc.go` additionally enumerates the package set a *second* time at lines
  41-46, outside the "# The package layout" section. See the Scope "In" doc-update list for the
  full, precise inventory.
- `internal/cli/cli.go`'s package doc comment enumerates every verb and the batch-mode identity key
  per verb (`symbol`-keyed vs `path`-keyed). `impact` is symbol-keyed; add it there.

**Language support reality:** LSP-backed resolution works for whatever `registry` detects (Go has a
supervised daemon; Python/C#/TypeScript/Rust cold-spawn per call). Tree-sitter toc strategies exist
for Go, Python, and C# only. So an `impact` query in a TypeScript project resolves callers fine but
returns no enclosing ranges — the per-entry degradation path, not a failure.

## Testing

**`internal/quarryengine/impact` — pure unit tests, TDD candidates.** The enclosing-symbol selection
is a pure function over `[]toc.Symbol` and a line number; write it as one (e.g.
`enclosingSymbol(symbols []toc.Symbol, line int) (toc.Symbol, bool)`) and test it with hand-built
`toc.Symbol` slices, no filesystem and no LSP:

- line inside a function body → that function
- line on the declaration's first line, on the docstring's first line, and on the last line → all
  match (inclusive bounds at both ends)
- line in the blank gap between two declarations → no match
- line before the first declaration (package clause, imports) → no match
- overlapping ranges → greatest-`Start` wins
- empty symbol slice → no match

**`internal/quarryengine/impact` — file-level tests against fixtures.** Test the per-file
enclosing-range resolution and the per-call parse cache against real source files in a new
**repo-root** `testdata/impactfixture/` directory, following `testdata/clockfixture`'s shape.
Repo-root is not a stylistic preference: `layering_test.go` walks *every* `.go` file under
`internal/quarryengine/` with no `testdata` exemption, so a Go fixture at
`internal/quarryengine/impact/testdata/` fails the guard with "no layering row declared". Cover: a Go file
with a docstring'd method (assert `start_line` is the docstring line, not the `func` line); a file
where the reference is at package scope; an unreadable path; a `.ts` file (unsupported language →
per-entry error, no range); and — the case the greatest-`Start` tie-break exists for — a **Python**
fixture with a method inside a class, asserting a line inside the method resolves to the *method*,
not the enclosing class. Go fixtures alone cannot exercise that rule, since Go produces no
overlapping ranges.

For the parse cache, **inject a counting seam** and assert the parse count is exactly one when
several call sites share a file. This is the committed mechanism, not one of two options: identical
results do not prove a single parse, so the identical-result proxy cannot establish the
`per-call-parse-cache` decision's claim at all. Do not assert on timing.

**`internal/quarryengine/impact` — the assembly step.** `Impact` itself needs the LSP, so test the
assembly seam separately from the transport, the way `query`'s own `callersFromClient`/
`symbolFromClient` seams are structured: a function that takes an already-obtained
`(callers, declaration []query.Reference)` plus the file-resolution seam and produces the `Result`.
Cover: declaration-set exclusion, recursive self-call retention, two call sites in one function
producing two entries with equal `enclosing_range`, sort order, and — the two shapes the round-2
review found undefined — an **empty declaration set with a nil error** (both `target` and
`definition` omitted, callers still returned, no error) and all three `definition` outcomes:
**resolved**; **parsed but no enclosing symbol** (a package-level `var` target — ranges omitted,
`error` *absent*, `target` omitted); and **no toc strategy for the declaring file** (`definition`
present with `file` + `line` + `error`, no `start_line`/`end_line`, `target` omitted). The middle
case is the one that distinguishes the two non-resolving outcomes and must be asserted separately
from the third, not folded into it.

**`internal/cli` — CLI-level tests via `RunCLIIn`.** Follow `internal/cli/toc_test.go`'s and
`cli_test.go`'s existing shape: drive the command, decode the JSON envelope, assert on keys and exit
code. Cover: the `ok`/`resolution` envelope; `callers: []` on zero callers; ambiguity → exit 2 with
`candidates` and `ok:true`; not-found → exit 1; batch mode with mixed statuses → worst-status exit
code and per-entry `symbol`/`status` keys — and specifically that a batch entry's `symbol` key still
holds the **query string**, with the identity object present as `target`, proving the `runBatch`
merge-order collision is actually avoided; `--within` filtering with a **relative** flag value (not
only an absolute one — a relative value is what an un-normalized filter silently turns into an empty
result set); `--in-file` composing with batch
mode; `--build-tags` on a language with no tag template → error, not a silent no-op.

**`internal/cli` — live-tier test.** Add an `impact_lsp_test.go` guarded by `//go:build lsp` and
`exec.LookPath("gopls")` (via `t.Skip`), following `assertnocallers_lsp_test.go` exactly. This is
where the end-to-end claim is proved: a real symbol in `testdata/`, real gopls resolution, and the
assertion that each caller's `enclosing_range.start_line` lands on the caller's docstring line
rather than its `func` line. That docstring-inclusive assertion is the brief's central requirement
and cannot be proved against a fake server.

**Guard tests.** `layering_test.go` and `seam_enforcement_test.go` must pass unchanged in intent —
update their tables/constants as part of the work, and verify they still *fail* when the new package
imports something disallowed (a temporary local check, not a committed test).

**Verify command:** `go build ./... && go test ./...` for the plain tier; the `lsp`-tagged tier runs
separately on a machine with gopls and is not part of the default verify.

## Q&A log

- **Q:** Where does the impact logic live? **A:** [auto-pick] New `internal/quarryengine/impact` package. **Why:** the layering table forbids `query`↔`toc` imports, so a new top-of-DAG package is the only clean home, and it gives the SDK/orchestrator consumer a typed entry point.
- **Q:** How is a caller's enclosing declaration found? **A:** [auto-pick] Reuse `toc.TOCFile` symbols and select by `Start <= line <= End`. **Why:** `Symbol.Start` already includes the docstring, satisfying the brief for free, and reuses the already-tested docstring-association rule instead of duplicating it.
- **Q:** Which caller source — `Callers` or raw `References`? **A:** [auto-pick] `quarry.Callers`, verification on. **Why:** it already filters gopls' interface-method conflation and returns the declaration set needed for exclusion, on one connection.
- **Q:** How is the caller's own symbol identity emitted? **A:** [auto-pick] Structured toc fields (`kind`/`name`/`owner`/`package`/`signature`). **Why:** the issue's `internal/billing.ProcessRefund` example is a Go import path quarry cannot derive without `go/packages` and which does not transfer to Python or C#; fabricating one would be wrong, not merely absent.
- **Q:** Batch mode and flag set? **A:** [auto-pick] Mirror `refs` exactly, including symbol-keyed batch mode and the 0/1/2 exit contract. **Why:** a verb that resolves symbols differently from its three siblings is a trap, and the DAG consumer querying many symbols is precisely the batch case.
- **Q:** Multiple call sites inside one enclosing function? **A:** [auto-pick] One entry per call site, singular `call_site_line`. **Why:** it is the shape issue #5 specifies, it is lossless, and grouping is trivial for the consumer while un-grouping is impossible.
- **Q:** A reference with no enclosing declaration? **A:** [auto-pick] Emit the entry, omit `enclosing_range`, document the absence as "file-scope reference". **Why:** it is a real dependency edge; silently under-reporting impact is the one failure mode this verb must not have.
- **Q:** A caller file whose language has no toc strategy? **A:** [auto-pick] Per-entry `error`, no `enclosing_range`, call still succeeds. **Why:** mirrors `toc dir`'s "still listed, never skipped" rule; one unsupported file must not destroy an otherwise-complete answer.
- **Q:** Recursive self-calls? **A:** [auto-pick] Keep them; exclude only the declaration set. **Why:** a self-loop is a real edge and is exactly the case that breaks a naive topological sort, so the DAG consumer needs it explicitly.
- **Q:** Per-file parse cost? **A:** [auto-pick] Cache toc results per file within one `Impact` call only. **Why:** N callers in one file must parse once; a package-level cache would contradict toc's documented no-caching property and go stale against on-disk edits.
- **Q:** Does the `definition` range get its own LSP call? **A:** [auto-pick] No — reuse `Callers`' returned declaration set and run it through the same enclosing lookup. **Why:** the call has already been made; a second one would cost a redundant round trip on a second connection.
- **Q:** Are `sigend_line` values emitted? **A:** [auto-pick] Yes, on both `definition` and `enclosing_range`, omitted when zero. **Why:** `toc.Symbol.SigEnd` is already computed and `start..sigend` is the documented cheap way to read a symbol's docstring and signature without its body.
- **Q:** Zero callers — success or failure? **A:** [auto-pick] Success, exit 0, `"callers": []`. **Why:** "nothing depends on this" is the answer a DAG builder needs for a leaf node; exit 1 would conflate it with "symbol does not exist".
- **Q:** `--depth` / transitive impact? **A:** [auto-pick] Out of scope for v1. **Why:** issue #5 explicitly defers it to keep output size and latency predictable; building a recursion guard "ready for later" is speculative.
- **Q:** Human-readable rendering? **A:** [auto-pick] JSON only. **Why:** issue #5 states the JSON shape is the source of truth and any human rendering is a presentation layer on top.
- **Q:** Which docs change? **A:** [auto-pick] `README.md`'s verb list, `internal/quarryengine/doc.go`'s package-DAG section (seven → eight packages), `quarry/facade.go`'s header comment (same phrase), and `internal/cli/cli.go`'s package doc comment. **Why:** all four enumerate the verb or package set explicitly and would go stale; the repo's existing doc discipline treats that as a defect.
- **Q:** The top-level identity object was keyed `symbol`, which `runBatch` would overwrite with the query string — what shape wins? **A:** [auto-pick] Rename it to `target` in both arg-count modes; leave `runBatch` alone. **Why:** `symbol` is the shared batch contract's entry-identity key across all four existing verbs; changing `runBatch` would change them, while renaming one new field costs nothing and yields one shape for both modes.
- **Q:** `Callers` silently returns an unverified set on four swallowed-error paths — does `impact` claim verification ran? **A:** [auto-pick] No. Accept the silent-skip path and scope `"resolution":"complete"` to exactly what it means on `refs` — exhaustive for the query as given, no verification claim. **Why:** `refs` already carries that marker on a never-verified set, so reading it as a verification claim misreads the existing contract; asserting more would require an engine signature change to `Callers`, which is out of scope.
- **Q:** What do `target` and `definition` become when the declaration set comes back empty with a nil error? **A:** [auto-pick] Both omitted; the call still succeeds with its caller list. **Why:** the caller list is still a complete, useful answer, and failing the call over a missing definition would discard it; joint absence is an unambiguous signal.
- **Q:** Where do the top-level `target` fields come from? **A:** [auto-pick] The same `toc.Symbol` the `definition` enclosing lookup found — never the query string, never the `workspace/symbol` candidate. **Why:** one provenance means one rule to implement and test, and it keeps `target` consistent with every caller entry's identity fields.
- **Q:** Does `seam_enforcement_test.go`'s `minPackageDirs` need bumping too? **A:** [auto-pick] No. **Why:** its own comment records that the floor is deliberately one below the real count, so adding a package keeps it satisfied; only its stale comment enumerations need updating.
- **Q:** Where do the Go test fixtures live? **A:** [auto-pick] Repo-root `testdata/impactfixture/`. **Why:** `layering_test.go` walks every `.go` file under `internal/quarryengine/` with no `testdata` exemption, so an in-package fixture directory fails the guard outright.
- **Q:** Does `definition` get one degradation rule or two, given the caller side has a non-error "file-scope" outcome? **A:** [auto-pick] Three explicit outcomes — resolved / parsed-but-no-enclosing-symbol (no `error`) / parse-or-language failure (`error` present). **Why:** `toc.Kind` is function/method/type only, so a package-level `var` or `const` target genuinely lands in the middle case, and folding it into the error case would report a correct answer as a failure.
- **Q:** What are the engine and facade type names? **A:** [auto-pick] `impact.Result`/`Target`/`Definition`/`Caller`/`Range`, aliased as `ImpactResult`/`ImpactTarget`/`ImpactDefinition`/`ImpactCaller`/`ImpactRange`. **Why:** `facade_test.go`'s enumerated blocks need exact identifiers, and the `TOC`-prefix convention already sets the pattern; `Target` matches the JSON key and avoids reading as a second `toc.Symbol`.
- **Q:** Is the tree-sitter parse loop bounded by `--timeout`? **A:** [auto-pick] No — the loop checks `ctx.Done()` between files, and `--timeout` keeps its documented per-LSP-phase meaning. **Why:** `toc.TOCFile` takes no `ctx`, so a real deadline would mean changing a shared engine signature; a between-files check is the honest bound actually available, and silently widening `--timeout` would make its existing help text false.
- **Q:** Which overlap case justifies the greatest-`Start` tie-break? **A:** [auto-pick] Python/C# class↔method nesting, not grouped Go type declarations. **Why:** `goGroupedTypeSymbol` ranges each spec from the spec itself, so grouped specs are siblings; only a class Symbol genuinely spans its method Symbols, which is also why the fixture set needs a Python case and not only Go ones.
- **Q:** How is the parse cache asserted? **A:** [auto-pick] A counting seam, exactly one parse per distinct file. **Why:** identical results are consistent with N parses, so the identical-result proxy cannot prove the claim it was offered for.
- **Q:** Round 5 approved but carried demoted findings, so the convergence gate did not fire — run round 6? **A:** [operator] No: fix all six findings and proceed to Handoff. **Why:** operator override of the gate, taken with severity trending monotonically down — 2 BLOCKING in round 2, none in rounds 3-5, with the last rounds producing only demotions and contract precision. Recorded here because it is a deliberate deviation from the gate, not a gate that passed.
- **Q:** Absolute or caller-relative file paths in the output? **A:** [auto-pick] Absolute, matching `refs`/`definition`/`assert-no-callers`. **Why:** `toc dir`'s relative `path` exists so entries round-trip into `toc file`; `impact` has no such round-trip and consistency with the LSP-backed verbs wins.
