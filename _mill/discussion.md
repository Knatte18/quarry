# Discussion: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
slug: glyph-maker
status: discussing
parent: main
```

## Problem

Loomyard's planner writes plans whose Create cards declare symbols that do not exist yet. The
orchestration design settled in Loomyard issue #226 forbids the LLM from ever spelling a glyph:
existing symbols are copied verbatim out of quarry answers, and new symbols carry a tentative
`plan:<expected-glyph>` handle that the pipeline canonicalizes. For that canonicalization the
pipeline needs to ask quarry a question quarry cannot answer today: *given a unit and a
declaration head I intend to write, what glyph will that declaration have?*

Why now: this is the first of `docs/roadmap.md` point 2's three plan-alphabet primitives. Its one
ordering constraint — C1's contract merge (`49304ca`), which changed the envelope and self-forms
this query emits — was satisfied 2026-09-05, so it is unblocked. The Loomyard adoption (issue
#226) runs in parallel and consumes this query.

The load-bearing property is not the query itself but *how* it is answered: the maker must wrap
the supplied declaration head in a synthetic in-memory file and run the **same** extractor that
later reads the real code. Prediction and eventual extraction are then the same function by
construction, and there is never a parallel naming rulebook that can drift from the extractor.

## Scope

**In:**

- A new engine-level maker: unit + declaration head in, glyph id + kind out, per entry.
- A batched facade entry point, `quarry.Name(decls []Declaration) []NameResult`, package-level.
- A new CLI verb, `name`, mirroring one target per call, with `--unit` and `--text`.
- JSON and text renderers for the new result shape, in `quarry/render.go` and `quarry/text.go`.
- The prediction≡extraction round trip over the pinned Loomyard checkout.
- Goldens (JSON and text) for the six cases the task names.
- Docs and every in-tree statement of the verb set or its count — enumerated by search, not by
  naming files; see the "Verb-set statements" inventory under Testing.

**Out:**

- No MCP tool. `internal/mcpserver` and `cmd/quarry-mcp` are not touched at all. The consumer is
  Loomyard's pipeline through the facade, never an LLM tool.
- No change to `glyph/`. That package stays cgo-free for Loomyard's own import; this query needs
  tree-sitter and therefore lives in `internal/engine`.
- No change to any existing verb, envelope, exit code, status vocabulary, or renderer behaviour.
  This is purely additive.
- No type checking of any kind. Tree-sitter does not resolve types and the maker must not try.
- Nothing written to disk. The synthetic file exists only as a `[]byte` handed to
  `treesitter.WithTree`.
- No knowledge of plans, cards, handles, or DAGs. Those are Loomyard's (issue #226).
- Interface methods and multi-name const/var specs are explicit non-goals — see the
  "Accepted declaration forms" decision.
- The roadmap's other two primitives (2a `glyphs` verb, 2c diff-to-symbols) are separate tasks.

## Decisions

### CLI verb name is `name`

- Decision: the CLI verb is `quarry name`, the facade function is `quarry.Name`, and the result
  type is `NameResult`.
- Rationale: the task body already reaches for `NameResult`, so `name` is the spelling the
  contract was drafted in. Quarry's verbs are imperative (`toc` excepted as a noun of long
  standing): "name this declaration" reads correctly.
- Rejected: `make` — collides with a Go builtin and says nothing about glyphs. `compose` — already
  taken by C1's self-form compose API in `glyph/`, and reusing the word for a different operation
  in the same repository would be a live confusion.

### The facade entry point is package-level, not a `Repo` method

- Decision: `func Name(decls []Declaration) []NameResult` is a package-level function in `quarry`,
  delegating to a package-level function in `internal/engine`. It takes no `*Repo`, no root, and
  performs no I/O.
- Rationale: the maker reads nothing. A `*Repo` receiver would claim a repository dependency the
  query does not have, and would force a caller with only a fragment to first open a directory
  that has nothing to do with the answer. `quarry/quarry.go`'s own alias-types-carry-no-methods
  decision already establishes that package-level functions are the facade's normal shape for
  anything not bound to an open repository.
- Rejected: a `(*Repo).Name` method for symmetry with `Resolve` and `Expand`. Symmetry is real but
  it would be symmetry with two queries that genuinely read the filesystem, which this one does
  not.

### The batch is the facade shape; the CLI is one per call

- Decision: the facade takes `[]Declaration` and returns `[]NameResult`, positionally, one result
  per input, always the same length as the input. The CLI accepts exactly one declaration per
  invocation and renders exactly one `NameResult`.
- Rationale: this is the resolve pattern, stated in `docs/rewrite-plan.md` §5 and implemented in
  `quarry/repo.go`'s `Resolve` doc comment: the Go caller batches a whole plan's Create
  declarations in one call, the command line keeps one invocation / one answer / one exit code.
- Rejected: a single-declaration facade with the caller looping — throws away the batch shape the
  consumer was designed around.

### One bad entry never fails the batch

- Decision: every failure is a per-entry failure carried in that entry's own `NameResult`. `Name`
  returns no error at all — no `([]NameResult, error)` pair.
- Rationale: with no I/O there is nothing that can fail batch-wide. Every failure mode is a
  property of one entry's own unit or fragment. A returned error would have no value to carry and
  would invite a caller to abandon results that are perfectly good.
- Rejected: `([]NameResult, error)` for symmetry with `Resolve`. `Resolve` has an error because it
  reads directories and that read can fail; this one does not.
- Corollary: an empty input slice returns an empty, non-nil slice, matching `symbolsOfUnit`'s and
  `SpansOf`'s existing empty-not-nil rule.

### `NameResult` shape: no `ok` key

- Decision:

  ```go
  // Declaration is one entry of a Name batch.
  type Declaration struct {
      Unit string // the glyph unit the declaration will belong to
      Decl string // the declaration head, verbatim
  }

  // NameResult is Name's answer for one Declaration, positionally.
  type NameResult struct {
      Unit   string `json:"unit"`             // echo of Declaration.Unit, verbatim
      Target string `json:"target"`           // echo of Declaration.Decl, verbatim
      ID     string `json:"id,omitempty"`     // the glyph, present only on success
      Kind   Kind   `json:"kind,omitempty"`   // the declaration kind, present only on success
      Error  string `json:"error,omitempty"`  // the rejection sentence, present only on failure
      Reason string `json:"reason,omitempty"` // the plain-word rejection reason, failure only
  }
  ```

- Rationale: `quarry/render.go`'s `RenderErrorJSON` doc comment fixes `ok` as a key that appears
  **only** on the failure envelope, precisely so it can never disagree with the exit code beside
  it. Putting an `ok` inside a success payload would break that invariant. The `ID`+`Kind` versus
  `Error`+`Reason` split is exactly `ResolveResult`'s own pre-resolution-rejection shape — status
  absent, `Error` and `Reason` set — so a reader already fluent in the resolve envelope reads this
  one with no new rule.
- Rejected: `ok bool` as the task body sketched. Rejected: reusing `Status`; `found` /
  `not_found` / `ambiguous` are answers about a repository the maker never looks at, and
  `not_found` for "your fragment declared nothing" would be actively misleading.
- The echo is flat (`unit` and `target` as two string keys), not nested under one `target` object:
  both halves are caller input, both are echoed verbatim, and a nested object would be the only
  nested echo in the whole envelope.
- `Kind` is the existing `engine.Kind` / `quarry.Kind`, aliased into the facade the way every
  other engine type already is. No new kind values.

### Go is implied; there is no language parameter

- Decision: `Declaration` carries no language field. The maker parses as Go and builds
  `glyph.Go` glyphs.
- Rationale: `resolveGlyphTarget` already parses every target as `glyph.Go` with no override, and
  `docs/rewrite-plan.md` §6 has Go as the only implemented alphabet. A language parameter today
  would have exactly one legal value.
- Rejected: an explicit `Lang` field for future-proofing — YAGNI; a second language adds it
  against a real case, exactly as the engine's other contract gaps are documented to.

### The synthetic file, and why the unit cannot leak from it

- Decision: the maker builds `"package q\n\n" + decl + "\n"` as a `[]byte`, hands it to
  `treesitter.WithTree("go", src, ...)`, and calls the registered Go `Strategy`'s
  `Symbols(unit, root, src)` with the unit taken from `Declaration.Unit`. The placeholder clause is
  the fixed identifier `q` and is never derived from the unit.
- Rationale: `Strategy.Symbols` takes the unit as a parameter and its doc comment states that a
  strategy never derives the unit itself — the clause is read only by `Strategy.Package`, which
  the maker never calls. A fixed placeholder makes the independence provable: a test passes a unit
  bearing no relation to `q` and asserts the emitted glyph carries the parameter's unit.
- Rejected: deriving the placeholder from the unit's last segment — it would make the two look
  coupled when they are not, and a unit whose last segment is not a valid Go identifier (a
  directory named `foo-bar`) would then produce an unparseable synthetic file for no reason.

### Same extractor, by construction — the anti-drift rule

- Decision: the maker calls `StrategyFor("go")` and that strategy's `Symbols` method. It contains
  no naming logic of its own: no receiver-type derivation, no owner-chain construction, no
  blank-name rule, no id formatting beyond taking the `Symbol.ID` the extractor already computed.
- Rationale: this is the task's load-bearing requirement. Any naming decision reimplemented here
  is a second rulebook that can drift from `golang.go`, and the entire value of the query is that
  it cannot.
- Enforcement: the round trip below is the mechanical proof. A reviewer reading the maker should
  be able to see that the only thing between the input and the output is the wrap, the parse, and
  a count.

### Bodyless heads get exactly one completion retry

- Decision: parse the fragment verbatim. If that parse reports an error (`treesitter.WithTree`'s
  `partial` flag is true), retry exactly once with `" {}"` appended to the fragment before the
  trailing newline. If the retry also reports an error, the entry fails with reason `parse`. If
  the verbatim parse is clean, the retry never runs.
- Rationale: the round-trip criterion feeds each extracted symbol's `Signature` field back in, and
  `SignatureCut` cuts a type's signature at its body's start byte — so an extracted struct's
  verbatim signature is `type T struct`, which is not on its own a parseable Go declaration. The
  same holds for `type T interface`. One appended `{}` closes both, and closes nothing else:
  a function head is legal Go without a body (`func F() error` parses as a bodyless declaration),
  a type alias parses whole, and const/var signatures are already complete declarations. This is
  a parse-recovery rule operating on bytes before extraction; it makes no naming decision and
  therefore does not violate the anti-drift rule above.
- Rejected: no completion at all — every struct and interface head, the two most common Create-card
  declarations there are, would be a per-entry error. Rejected: requiring the caller to supply a
  syntactically complete declaration — it pushes the completion rule into every caller, including
  the round trip, which is exactly the parallel rulebook the task forbids.
- Rejected: a ladder of several candidate wrappings tried in order. Two attempts, one documented
  rule, no ordering to reason about.

### Exactly one symbol, or the entry fails

- Decision: after a clean parse, count the symbols the strategy returns for the synthetic file.
  Exactly one is the answer. Zero fails with reason `no_declaration`. More than one fails with
  reason `several_declarations`.
- Rationale: the task's rule 5, and the repository's standing "never a silent pick" philosophy
  (`docs/rewrite-plan.md` §3, and the `ambiguous` status that exists for exactly this reason). A
  fragment declaring two things has two glyphs and the maker has no basis to choose.
- Note: a partial parse is decided *before* the count. A fragment whose parse reports an error is
  `parse`, even if it happens to yield one surviving symbol. This keeps "malformed" and "declared
  the wrong number of things" as two crisply separate conditions, each with its own golden.

### Validation is a glyph round trip, not a new validator

- Decision: after extraction, take the `Symbol.ID` the strategy computed and assert
  `glyph.Parse(glyph.Go, id)` succeeds and its `String()` returns the same bytes. On failure the
  entry fails with `Reason` set to the `*glyph.ParseError`'s own `Reason` word, as a plain string,
  and `Error` set to that error's `Error()` text.
- Rationale: this validates the caller's unit and the extracted name in one step, reusing the
  grammar that already owns both questions, with no new exported API in the cgo-free `glyph`
  package. It is the same assertion `TestRoundTrip_Loomyard` already makes over walked symbols.
  A caller passing a unit with a `#` in it, an empty segment, or a `..` segment is rejected here
  with the grammar's own vocabulary.
- Rejected: a new exported unit-only validator in `glyph/` — new public API in the package that
  Loomyard imports, for a check the existing parser already performs.
- `ResolveResult.Reason` is already documented as "deliberately a plain string, not glyph.Reason",
  so propagating the word as a string is the established convention, not a new one.

### The reason vocabulary

- Decision: a closed set of maker-owned words — `parse`, `no_declaration`, `several_declarations`,
  `internal` — plus, verbatim, any `glyph.Reason` word the validation step above propagates. The
  maker's own four live as a constant block in the engine with an enumerating slice beside them,
  mirroring `glyph.Reasons`, so a test can assert completeness.
- Rationale: closed vocabularies with an enumerating slice are this repository's established shape
  for exactly this (`glyph/errors.go`'s sixteen reasons and its `TestReasons_Completeness`).
- `internal` covers a `treesitter.WithTree` failure that is not a parse error — an unwired grammar
  or a nil tree. Unreachable with Go always wired; spelled anyway, because this repository spells
  its unreachable branches rather than letting one fall through to a success shape.
- Rejected: one `parse` word for every failure — the caller cannot then tell "your fragment is
  malformed" from "your fragment declared two things", which are different fixes.
- **The `Error` sentence for each reason**, since the text renderer prints it and two goldens pin
  its bytes. Each is single-line and **does not repeat the target**, which the text view already
  prints ahead of it (`<target> error <reason>: <message>`):

  | reason | `Error` sentence |
  | --- | --- |
  | `parse` | `declaration does not parse` |
  | `no_declaration` | `declaration declares no symbol` |
  | `several_declarations` | `declaration declares N symbols; exactly one is required` |
  | `internal` | `internal error: ` + the underlying error's own text |
  | *(propagated)* | the `*glyph.ParseError`'s own `Error()` text, whole |

  `several_declarations` carries the count because "two" and "eleven" are different mistakes and
  the number is free — the maker counted them to reject the entry.
  The propagated row is the one sentence that does name the target, since glyph composes it that
  way; that is the same exception `cli.go`'s own contract rule already grants `runExpand`'s
  `*glyph.ParseError` branch, on the grounds that `glyph` is a public package the contract names
  rather than an internal one whose name would leak.
  The `internal` row's sentence is the only one carrying a wrapped chain, matching exit 3's
  existing rule that it alone carries `err.Error()` whole behind one prefix.
- **The missing-`--unit` usage sentence** is `--unit is required for name`, following the existing
  per-verb rejection shape (`--depth is not valid for resolve`) rather than the value-shape one
  (`--unit requires a value`, which is what a bare `--unit` with no value gets from the existing
  `nextValue` path).

### Accepted declaration forms

- Decision: every top-level declaration shape the Go extractor emits is accepted — function,
  method, type (struct, interface, alias, named), const, var — in either their head-only or their
  complete form, **with one exception: an interface fragment must be head-only or carry an empty
  body.** `goTypeSymbols` (`golang.go:170,183`) appends `goInterfaceMethodSymbols` to the
  interface's own type symbol, so `type R interface { Read() error }` yields two symbols —
  measured — and is correctly rejected by the exactly-one rule as `several_declarations`. Both
  `type R interface` (through the completion retry) and `type R interface {}` yield exactly one,
  the type itself. Three shapes are therefore out of contract:
  1. **Interface methods, and any interface fragment carrying one.** An interface method's
     extracted signature is a bare method element (`Read(p []byte) (int, error)`), which is not a
     top-level declaration in any wrapping, and its owner is the enclosing interface's name, which
     is not present in the fragment at all. The same non-goal covers the *populated* interface
     declaration, for a different reason: it declares the type and every method at once, so it has
     several glyphs rather than one, and the maker names one declaration per entry.
     A Create card that adds an interface with methods asks the maker for the interface *type's*
     glyph with a head-only or empty-bodied fragment; the method glyphs are separate entries the
     maker cannot answer, since their owner is not derivable from the head alone.
  2. **Multi-name const/var specs.** `const X, Y = 1, 2` is one declaration yielding two symbols,
     so it is rejected by the exactly-one rule above, correctly and by design. `var A, B = 1, 2`
     behaves identically — measured.
- Note the asymmetry with a struct, and why it is not a defect: `type S struct { F int }` yields
  exactly one symbol, because Go struct fields are not listable declarations, while interface
  methods are. So the "complete form" is accepted for a struct and refused for an interface, and
  the rule is not "heads only" but the exactly-one rule applied uniformly. Nothing special-cases
  interfaces in the maker; the count does the work.
- Rationale: the input contract is "declaration head + unit". Anything whose glyph is not fully
  determined by those two things is out of contract, and inventing an owner parameter to admit
  interface methods would be designing for a requirement nobody has stated.
- Both non-goals are asserted in the round trip rather than silently skipped — see Testing.

### Tree-sitter does not type-check, and that is the feature

- Decision: a declaration whose receiver type does not exist anywhere — `func (f *Focus) Reset()
  error` where `Focus` is itself another Create card in the same plan — parses, extracts, and
  answers normally. No existence check of any kind is added.
- Rationale: this is precisely the case the query exists for. Loomyard's planner asks about symbols
  that do not exist yet, on types that do not exist yet.
- Covered by a dedicated test and a golden.

### CLI shape: `--unit` is required and verb-scoped

- Decision: `quarry name --unit <unit> "<declaration>"`. `--unit` is required for `name` and
  rejected for every other verb, with the message shape `parseArgs` already uses
  (`"--unit is not valid for toc"`). `--depth`, `--symbols` and `--no-symbols` are rejected for
  `name` the same way. `--text` is accepted.
- Rationale: keeps `parseArgs`' invariant that every verb takes exactly one positional target. The
  declaration is the target; the unit is a flag. Two positionals would be the only verb in the CLI
  with a different arity.
- A missing `--unit` is a usage error (exit 2) with usage text, since it is a malformed
  invocation, not a negative answer.

### `name` needs no repository root

- Decision: `Run` dispatches `name` immediately after the help check, before reading the working
  directory and before `repopath.ResolveRoot`. `--root` is rejected for `name` as not valid for
  the verb.
- Rationale: the verb reads nothing from the filesystem. Resolving a root first would make
  `quarry name` fail with "no repository root found above ..." in a directory where the answer is
  perfectly computable — a failure with no relationship to the question asked.
- Rejected: accepting `--root` and ignoring it — a flag that is accepted and does nothing is worse
  than one that is refused with a reason. Rejected: resolving the root like every other verb —
  buys uniformity at the price of a wrong failure.
- This means `Run`'s shared pipeline shrinks from "four steps every verb shares" to "two steps
  every verb shares, then two more for the three repository verbs". `Run`'s doc comment must be
  rewritten to say so; it currently states the four-step shape as universal.

### Exit codes

- Decision: a named `codeForNameResult(r NameResult) int`, table-testable exactly as
  `codeForResolveResult` and `codeForExpandAnswer` are: `exitOK` when `ID` is non-empty;
  **`exitInternal` when `Reason` is `internal`**; `exitNegative` for any other non-empty `Error`;
  `exitInternal` for the shape where neither `ID` nor `Error` is set (unreachable; spelled so it
  cannot route to zero).
- **The `internal` reason is not a negative answer, and the CLI must not render it as one.** An
  unwired grammar or a nil tree says nothing about the caller's declaration, so that entry takes
  `fail`'s path — the compact error envelope on stdout, the same sentence on stderr, exit 3, no
  payload — exactly as `usage.go`'s "3 internal error" line already promises and as the D2 rule
  requires (the error envelope is for usage and internal errors with no payload). Routing it
  through the payload path would make `quarry name` report exit 1 for a condition the help text
  says is exit 3.
- The facade keeps carrying `internal` as a per-entry reason regardless, because a batch must not
  lose the other entries' good answers to one entry's internal failure. The CLI, which renders one
  entry, is where the two diverge: `codeForNameResult` maps the reason to exit 3 correctly, but
  the *bytes* differ — the error envelope rather than the payload.
- **`runName`'s step order, stated explicitly, because the two rules below are otherwise
  contradictory:**
  1. Call the facade. Take the single result.
  2. **Check `Reason == internal` first, before rendering anything.** If it is, take `fail`'s path
     — compact error envelope on stdout, the same sentence on stderr, exit 3 — and return. No
     payload is written on this route.
  3. Otherwise render the payload (`RenderNameText` under `--text`, `RenderNameJSON` otherwise),
     write it to stdout, and only then compute and return `codeForNameResult(result)`. The
     payload-before-code order matters for the same reason it does in `runResolve`: a negative
     answer must be rendered, not replaced by the failure envelope.
  Read out of order — payload first, then the `internal` check — these two rules would emit a
  success payload *followed by* an error envelope on the same stdout. The check comes first.
- Rationale: the D2 rule stated in `docs/rewrite-plan.md` §5 — a negative answer renders its
  payload with exit 1, and the error envelope is only for usage and internal errors with no
  payload. A malformed declaration is a negative answer *about the declaration*, which the maker
  answers with a reason; it is not a usage error.
- On the payload route the bytes are written to stdout before the exit code is computed, matching
  `runResolve` — but that route is reached only after the `internal` check above, per `runName`'s
  stated step order.

### Text view

- Decision: `RenderNameText(r NameResult) string`, one line, no trailing whitespace, exactly one
  trailing `"\n"`:
  - success: `<id> <kind>`
  - failure: `<normalizeProse(target)> error <reason>: <normalizeProse(message)>`
- **The echoed target goes through `normalizeProse` too**, not only the message. A declaration head
  legitimately spans lines — an ungrouped var's signature is the whole declaration text, and a
  multi-line parameter list is part of a function's signature by `SignatureCut`'s own verbatim
  rule — so echoing it raw would break the one-record-per-line invariant the whole text format
  rests on. This is not a new rule: `normalizeProse`'s own doc comment states it is already applied
  to every `Signature` before printing, and the echoed target *is* a signature. Collapsing runs of
  whitespace to single spaces drops nothing: "prose intact" means nothing is truncated, not that
  source line breaks survive.
- The JSON view is unaffected — `target` there is the byte-verbatim echo, newlines included, since
  a JSON string has no line invariant to protect. A test asserts the two views differ in exactly
  this way for a multi-line head, so the divergence is deliberate and pinned rather than a
  discrepancy someone later "fixes".
- Rationale: the failure line is `RenderResolveText`'s own empty-status branch, extended by the one
  `normalizeProse` call its `Target` echo does not currently need (a resolve target is a glyph,
  which cannot contain whitespace; a declaration head can). The success line drops the target
  because `RenderResolveText`'s success branches already drop it in favour of the id — for a
  one-per-call CLI the input is on the command line beside the output.
- The text view is a CLI view of one result. There is no text rendering of a whole batch; the
  facade's consumer reads the struct.

### No MCP tool

- Decision: nothing is added to `internal/mcpserver` or `cmd/quarry-mcp`.
- Rationale: fixed by the task and by `docs/rewrite-plan.md` §9 — only tools a measured cell or
  the adoption's pipeline needs. This one is facade-first: the caller is Loomyard's pipeline.
- `internal/mcpserver/layering_test.go` is unaffected, since no file there changes.

## Technical context

**Where the pieces are.**

- `internal/engine/strategy.go` — the `Strategy` interface and `StrategyFor(lang)`. `Symbols(unit,
  root, src) []Symbol` is the one method the maker calls. Its doc comment is the authority for the
  unit-is-a-parameter rule.
- `internal/engine/golang.go` — the Go strategy: `goStrategy.Symbols` dispatches over
  `function_declaration`, `method_declaration`, `type_declaration`, `const_declaration`,
  `var_declaration`, and builds each `Symbol`'s `Glyph` and `ID`. The maker must not duplicate any
  of it.
- `internal/engine/treesitter/treesitter.go` — `WithTree(lang, src, fn)` owns the whole parse
  lifecycle and hands the callback `(root *ts.Node, partial bool)`. `partial` is
  `root.HasError()`, which is the completion-retry trigger. The callback must never retain `root`.
- `internal/engine/answer.go` — `Symbol` (the `ID` and `Kind` fields the maker reads), `Kind`,
  `Status`, `ResolveResult` (the envelope shape `NameResult` mirrors).
- `internal/engine/resolve.go:528` `symbolsOfDir` — the real-file path, for comparison: it reads
  bytes, calls `WithTree`, and calls `strategy.Symbols(unit, root, src)`. The maker is the same
  three lines with the bytes synthesised instead of read. Keeping the two visibly parallel is the
  point.
- `quarry/quarry.go` — alias declarations. `Declaration`, `NameResult` and the maker's reason
  constants need aliases here so an external caller can name them without importing the engine.
  Note the file's own rule: aliases carry no methods, so any helper is a package-level function.
- `quarry/repo.go` — `Resolve`'s doc comment is the model for the batch rationale.
- `quarry/render.go` — `renderJSON` is the single encoder configuration (two-space indent, one
  trailing newline, HTML escaping off). `RenderNameJSON` delegates to it like every other success
  renderer.
- `quarry/text.go:237` `RenderResolveText` — the branch structure and the `normalizeProse` call
  the new text renderer mirrors.
- `internal/cli/flags.go` — `parseArgs`, hand-rolled. The verb gate at the top and the flag
  `switch` both need a `name` row; the per-verb rejection message shape is
  `fmt.Sprintf("%s is not valid for %s", name, verb)`.
- `internal/cli/cli.go` — `Run`, the exit-code constants, `fail`, and the three `codeFor*`
  mappers. The verb dispatch `switch` and its `default` comment ("unreachable for every word other
  than the three verbs") both need updating to four.
- `internal/cli/usage.go` — `usageText`. ASCII only, no em dash, no typographic quotes; it is
  byte-compared in tests.
- `internal/engine/loomyard_test.go` — `loomyardRepo(t)` is the gate every Loomyard-dependent test
  goes through: it skips when `LADDER_LOOMYARD_REPO` is unset or not a directory, and **fails**
  when the checkout's HEAD does not start with `loomyardPin` (`72c23d9`). It also owns the
  `-update` flag. The new round trip uses this helper; it must not grow its own gate.
- `internal/engine/roundtrip_test.go` — `assertSymbolRoundTrip` and the existing
  `TestRoundTrip_Loomyard`, the T3-style pattern the new round trip follows, including the
  `testing.Short()` skip for a whole-repository walk.
- `internal/cli/glyph5_test.go` — the machine-independent fixture pattern (`writeScratchTree`) for
  cases that must run on a machine with no Loomyard checkout.

**Gotchas found during exploration.**

1. **A type's extracted signature does not parse on its own.** `goDeclSymbol` /
   `goUngroupedTypeSymbol` cut the signature at the body's start byte, so a struct's `Signature`
   is `type T struct`. This is the whole reason the completion retry exists. Anyone writing the
   round trip without it will see every struct and interface in Loomyard fail.
2. **Grouped const/var signatures are already complete — including bare iota continuations.**
   `goGroupedConstOrVarSymbols` prepends the keyword (`"const "` / `"var "`) to the spec's own
   text, so a grouped member's signature is `const A Kind = iota` for an explicit spec and just
   `const B` for a bare continuation spec in an iota block. **Both parse cleanly and yield exactly
   one symbol** — measured, not assumed (see the verification table below). This matters because
   an iota enum is the single most common grouped-const shape in Loomyard, so a bare continuation
   failing would have failed the round trip on every enum in the checkout. No special handling is
   needed for either. Ungrouped ones use `SignatureCut(decl, nil, src)`, the whole declaration
   text, which also parses.
3. **Interface-method signatures are not declarations.** `goInterfaceMethodSymbols` emits
   `KindMethod` symbols whose `Signature` is the method element text. There is no wrapping that
   recovers the right owner from that fragment alone. This is the first non-goal.
4. **Multi-name specs share one signature.** `goSpecNames` yields several names per spec, all
   carrying the same `Signature` string. Feeding that signature back produces several symbols and
   is correctly rejected. This is the second non-goal.
5. **A bodyless Go function is legal Go and parses cleanly** — the assembly-implementation form.
   So `func F() error` needs no completion, and the retry does not fire for it.
6. **Generic receivers.** `goReceiverTypeName` strips type parameters: `func (b *Box[T]) M()`
   yields owner `Box`, never `Box[T]`. The maker inherits this for free — and must, since the
   grammar rejects `member_type_params`.
7. **Blank names.** `goBlank` drops `_`-named declarations, so `func _()` yields zero symbols and
   the maker reports `no_declaration`. Correct, and worth a test.
8. **`init` is not special here.** `func init()` yields one symbol with the id `<unit>#init`.
   Several `init`s in one package collapse to one glyph in the *repository*, but a single fragment
   declares one, so the maker answers it. The multipart status is a resolve-time fact, not a
   naming-time one.

**Verified against tree-sitter-go, not assumed.** Every claim above about what parses was measured
during discussion by wrapping the fragment in `"package q\n\n" + frag + "\n"` and calling the
registered Go strategy's `Symbols("u/v", root, src)` — the exact shape the maker will use.
`partial` is `treesitter.WithTree`'s own flag:

| fragment | `partial` | symbols | ids |
| --- | --- | --- | --- |
| `const B` (bare iota continuation) | false | 1 | `u/v#B` const |
| `const A Kind = iota` | false | 1 | `u/v#A` const |
| `type T struct` | **true** | 0 | — |
| `type T struct {}` | false | 1 | `u/v#T` type |
| `func F() error` (bodyless) | false | 1 | `u/v#F` function |
| `func (f *Focus) Reset() error` | false | 1 | `u/v#Focus.Reset` method |
| `const X, Y = 1, 2` | false | 2 | `u/v#X`, `u/v#Y` const |
| `var A, B = 1, 2` | false | 2 | `u/v#A`, `u/v#B` var |
| `type R interface { Read() error }` | false | **2** | `u/v#R` type, `u/v#R.Read` method |
| `type R interface` | **true** | 0 | — |
| `type R interface {}` | false | 1 | `u/v#R` type |
| `var Long = map[string]int{\n\t"a": 1,\n}` | false | 1 | `u/v#Long` var, multi-line signature |

Five things this pins down for the plan. First, the completion retry's trigger is real:
`type T struct` reports a partial parse *and* yields zero symbols, and appending `" {}"` produces
exactly the id the complete form does. Second, a bodyless func and a bodyless method both parse
clean, so the retry never fires for them. Third, the unit in every id above is the parameter
`u/v`, never the synthetic clause `q` — the unit-independence property, observed rather than
argued. Fourth, a *populated* interface yields two symbols, which is why the accepted-forms
decision carves it out; a populated struct does not, because struct fields are not listable
declarations. Fifth, an ungrouped var's signature is the whole declaration text and can span
lines, which is what the text view's target echo has to survive.

## Constraints

- No `CONSTRAINTS.md` at the hub root; the constraints below are from `CLAUDE.md`, the task body,
  and exploration.
- **Go only. No Python** (`CLAUDE.md`).
- **Additive.** No existing envelope, verb, exit code, status value, or renderer behaviour
  changes. The one edit to existing behaviour is `Run`'s dispatch order, which changes nothing for
  the three existing verbs.
- **The `glyph` package stays cgo-free.** Nothing tree-sitter-shaped may be added to it. The
  maker lives in `internal/engine` behind `internal/cgoguard`'s existing guard.
- **Nothing is written to disk.** The synthetic file is a `[]byte` and nothing else. No temp file,
  no scratch directory, on any code path including tests of the maker itself.
- **The facade-only rule** holds for `internal/cli`: it imports `quarry`, never
  `internal/engine`. The new CLI code must reach the maker through `quarry.Name`.
- **`usageText` is ASCII only** and byte-compared; the `name` rows must not introduce an em dash
  or a typographic quote.
- **The JSON byte contract** is `renderJSON`'s: two-space indent, one trailing newline, HTML
  escaping off. The new renderer must delegate rather than build its own encoder.
- **Loomyard pin `72c23d9`.** Every Loomyard-dependent test goes through `loomyardRepo(t)` and
  inherits its skip-versus-fail asymmetry. Goldens regenerate only under `-update` against a
  checkout at that pin.
- `go test ./... && golangci-lint run` green.

## Testing

**`internal/engine` — the maker itself (TDD candidate; this is the core).**

Table tests over the maker function directly, no fixtures on disk, no repository:

- One case per accepted kind: free function, method, struct type, interface type, type alias,
  named type, ungrouped const, ungrouped var, grouped const member, grouped var member. Each
  asserts the produced `id` and `kind`.
- **Bare iota-continuation const member** — a fragment of exactly `const B`, the signature
  `goGroupedConstOrVarSymbols` emits for a continuation spec in an iota block. Measured to parse
  cleanly and yield one symbol (verification table above), and it is its own case rather than a
  row folded into the grouped-const case, because an iota enum is Loomyard's most common grouped
  shape: if this ever regresses, the round trip fails on every enum in the checkout at once, and
  the failure must be attributable to one named unit test rather than to a whole-repository walk.
  The `var` counterpart (`var B`) gets a row too.
- **Unit independence:** a unit unrelated to the synthetic package clause (`internal/reedengine`,
  say) produces a glyph carrying that unit. This is the test that proves the clause is inert.
- **Nonexistent receiver type:** a method on a type declared nowhere answers normally. Explicitly
  named as the "tree-sitter does not type-check" case.
- **Completion retry:** `type T struct` and `type T interface` both answer, with the same id their
  `{}`-bodied forms produce. Assert the head-only form and the *empty-bodied* form agree. The
  agreement assertion is against `type T struct {}` / `type T interface {}`, never against a
  populated interface: `type R interface { Read() error }` yields two symbols and is a
  `several_declarations` rejection, which gets its own case below rather than being folded in
  here. A *populated struct* (`type S struct { F int }`) does agree with its head, since struct
  fields are not listable declarations, and is asserted as its own row so the asymmetry is pinned
  rather than assumed.
- **Populated interface rejected:** `type R interface { Read() error }` gives
  `several_declarations` — the one place an otherwise-complete, perfectly valid Go declaration is
  refused. Its own named case, because a plan writer reading only the accepted-forms list would
  expect it to work.
- **Malformed:** a fragment that still fails after the retry gives reason `parse`.
- **Zero symbols:** `func _()`, and a fragment that is a comment only, give `no_declaration`.
- **Several symbols:** `const X, Y = 1, 2`, and two declarations in one fragment, give
  `several_declarations`.
- **Bad unit:** a unit with a `#`, an empty segment, and a `..` segment each fail with the
  grammar's own reason word propagated.
- **Reason completeness:** the enumerating slice matches the constant block, mirroring
  `TestReasons_Completeness`.
- **Batch semantics:** a mixed batch where one entry fails and the rest succeed — assert the
  result slice length equals the input length, order is preserved positionally, and every entry's
  `unit`/`target` echo is byte-identical to its input. Assert an empty input returns an empty,
  non-nil slice.

**`internal/engine` — the round trip (the done-when gate).**

`TestRoundTrip_LoomyardNaming` (name to taste), gated by `loomyardRepo(t)` and skipped under
`-short`:

1. Walk the whole pinned checkout and collect every symbol with its unit, `Signature`, `ID` and
   `Kind`. **The existing collector does not carry the data this needs:** `roundTripSymbol`
   (`internal/engine/roundtrip_test.go:33-37`) holds only `id`, `unit` and a `spanTuple` of
   `File`/`Start`/`SigEnd`/`End` — no `Signature`, no `Kind`. Disposition: **extend
   `roundTripSymbol` with `signature` and `kind` fields and fill them in `collectWalkSymbols`**,
   rather than writing a second walk collector. `TestRoundTrip_QuarryItself` and
   `TestRoundTrip_Loomyard` also consume that struct and simply ignore the two new fields; one
   collector keeps the two round trips reading the same harvest, which is the property that makes
   "prediction ≡ extraction" checkable at all. A parallel collector would be a second harvest that
   can drift from the first — the same argument the maker itself rests on.
   **Which walk this test consumes:** its own `TOC(".", TOCOptions{Depth: DepthAll, Symbols: on})`
   followed by `collectWalkSymbols` — the harvest half of `assertSymbolRoundTrip` — and **not**
   `assertSymbolRoundTrip` itself. That helper's second half is the per-unit span lookup, which
   costs one parse pass per unit directory over whole real files and answers a question
   (do the spans agree?) that `TestRoundTrip_Loomyard` already asks and this test does not. Extract
   the harvest into a small shared helper both call, so there is still exactly one collector.
   **Runtime budget, and why the timeout constraint is not violated:** `assertSymbolRoundTrip`'s
   doc comment (`roundtrip_test.go:100-104`) names go test's default timeout as a live constraint
   at Loomyard scale, so it must be answered rather than ignored. The maker adds one `WithTree`
   call per harvested symbol — two for a head that takes the completion retry — but each parses a
   three-line synthetic file, not a real source file, so the per-call cost is a parser
   construction plus a trivial parse rather than a whole-file parse. The shape of the cost is
   therefore linear in the *symbol* count with a small constant, against the existing helper's
   per-*unit* whole-file passes. The test is `-short`-skipped like every other whole-repository
   pass here. If the run nonetheless lands near the default timeout, the mitigation is to raise
   `-timeout` for this one test rather than to sample the harvest: sampling would silently weaken
   the zero-misses criterion, which is the one thing this test exists to assert.
2. Partition the harvest into *in-contract* and *excluded*, by the declared non-goals. The
   partition key for the multi-name-spec exclusion is the 5-tuple **(unit, File, Start, End,
   Signature)**: a spec's several names produce symbols agreeing on all five, so "two or more
   harvested symbols share this tuple" is exactly "these came from one spec". `File` is in the key
   deliberately — build-tag twins in one unit can share a signature *and* a line span across two
   different files, and excluding those would then assert (step 4) an error the maker will never
   produce, since each twin's fragment names exactly one symbol and answers normally.
   Interface methods are excluded by a separate rule: a `KindMethod` symbol whose `Signature` does
   not begin with the `func` keyword is an interface method element rather than a method
   declaration. Populated interface *types* are not excluded here — a harvested interface type's
   signature is head-only (`type R interface`, cut at the body), so it goes through the completion
   retry and answers in-contract.
3. For every in-contract symbol, call the maker with `(unit, signature)` and assert the returned
   `id` equals the symbol's real id and the returned `kind` equals its real kind. **Zero misses,
   zero extras.**
4. For every excluded symbol, assert the maker returns a per-entry *error* — never a wrong id.
   This is what keeps the exclusions honest rather than a silent skip.
5. Guard against a vacuous pass with **three pinned counts, not a fuzzy threshold**: the harvest
   total, the in-contract count, and the excluded count. They live in **a counts golden file,
   `internal/engine/testdata/loomyard/naming-counts.json`**, compared and rewritten by the same
   `compareGolden` mechanism (`golden_test.go:113-119`) the package's other Loomyard goldens use,
   under the existing `-update` flag. A Go `const` cannot be rewritten by a flag, so "constants
   regenerated under `-update`" — the earlier draft's phrasing — was two mechanisms that cannot
   both hold; a golden file is the half that keeps `-update` working, and `-update`'s own
   description already says "the Loomyard goldens under testdata/loomyard", which this file is.
   The checkout is pinned, so the numbers are stable, and any drift — an extractor change, a
   partition bug — fails loudly and attributably instead of passing under a threshold wide enough
   to hide it. A ratio or an "implausibly small" judgement would have to be chosen without knowing
   the repository, and two plan writers would choose two different gates.
   Alongside the constants, one cheap structural floor that holds before they are filled in on the
   first run: **the in-contract count must be greater than zero and the excluded count must be
   strictly less than the harvest total** — a partition bug that sweeps everything into one side
   fails immediately, without waiting for a number nobody can know yet.
   Report all three counts in every failure message.

**`internal/cli` — verb wiring.**

- `parseArgs` table rows: `name` accepted; `--unit` required and present; `--unit` rejected for
  `toc`/`resolve`/`expand`; `--depth`/`--symbols`/`--no-symbols`/`--root` rejected for `name`;
  `--text` accepted; exactly one target enforced.
- `codeForNameResult` table test, direct, like the three existing mappers.
- **`Run` from a directory inside no repository — `t.Chdir("/")`.** This is the direct proof that
  `name` dispatches before root resolution, and it is reachable: `Run` resolves the root from
  `os.Getwd()` (`cli.go:271`), the filesystem root has no `.git` above it, and Go 1.26 (`go.mod`)
  has `testing.T.Chdir`, which restores the working directory itself and fails a parallel test
  rather than corrupting one. The test asserts **both halves in one place**: from `/`,
  `quarry name --unit u/v "func F() error"` exits 0 with the right id, while `quarry toc .` exits
  2 with `"no repository root found above /; pass --root"`. One without the other proves nothing —
  the second half is what shows the first is not passing for some unrelated reason.
- Note the premise this replaces: an earlier draft proposed using the scratch-tree helper for a
  directory outside any repository. That is not achievable — `writeScratchTree`
  (`internal/cli/scratchtree_test.go:39`) builds under the module root's `.scratch/`, which is
  inside this repository, and `Run` never takes a path argument to resolve the root from.
  `cli.go:150-152` says as much: the no-root case is unreachable without changing the process
  working directory, "which these tests never do". `t.Chdir` is precisely the sanctioned way to
  change it, so this test makes that comment's parenthetical stale — update it in the same edit.
- No writes are involved: the test changes directory to `/` and reads nothing there, so the
  never-write-to-a-system-directory rule is untouched.
- `usageText` assertions extended for the `name` rows.
- **Multi-line head, both views:** a declaration head spanning lines (an ungrouped var with a
  composite literal, or a function with a multi-line parameter list) rendered under `--text` has
  exactly one `"\n"`, at the end; the same input rendered as JSON echoes `target` with its
  newlines intact. One test, asserting both halves, so the deliberate divergence is pinned in one
  place.

**Goldens — `internal/cli/testdata/name/`, JSON and text for each of the six required cases:**
method, free function, type, method on a receiver type that does not exist, malformed declaration
(per-entry parse error), multi-symbol input (per-entry reject). Both views per case, so twelve
files. They are machine-independent — the maker reads no repository — so unlike the existing
Loomyard goldens they run everywhere with no environment gate. `docs/research/output-formats/after/`
is a frozen research record and must not be added to.

`internal/cli` has no `testdata/` directory today, and its one golden helper is not reusable here,
so the harness has to be stated rather than assumed:

- **Compare/update helper.** A new `compareNameGolden(t, name, got string)` reading
  `testdata/name/<name>`, shaped exactly like `compareAfterGolden` (`after_test.go:189-211`):
  byte-for-byte comparison, or a write under the update flag. It cannot reuse
  `compareAfterGolden` itself — that function hard-codes the
  `../../docs/research/output-formats/after/` path, and its caller `TestAfterGoldens` is gated by
  `loomyardRepo(t)`, both wrong for a table that must run on a machine with no checkout.
- **The update flag is reused, not added.** `flag.Bool` panics on a duplicate name within one
  binary, and `internal/cli` already declares `-update` (`loomyard_test.go:29`). So the new table
  honours that same flag — which makes its description, *"regenerate the after/ goldens under
  docs/research/output-formats/after from the current LADDER_LOOMYARD_REPO checkout"*, wrong in
  both halves for these files: they are not under `after/` and need no checkout. **Widen that flag
  description and its doc comment, and add both to the docs inventory** — this is a fourth
  inventory site the earlier predicate would not have surfaced, since it counts no surface.
- **File body: the payload bytes and nothing else.** No `$ quarry ...` invocation header. The
  header on the `after/` goldens exists because those files are read as evidence documents in the
  research record; these are regression fixtures. It also matters concretely: a header would make
  each `.json` golden invalid JSON, and the engine's own `testdata/loomyard/*.json` goldens are
  pure payload with no header — that is the precedent a `testdata/` golden follows.
- **Exit code is pinned per row, in the table, not in the file.** Each of the six cases carries its
  expected exit code alongside its two golden names: 0 for the four answering cases, 1 for the
  malformed and multi-symbol rejections. Pinning the code in the table rather than in the file
  body keeps the golden byte-comparable against raw stdout.

**Docs and the verb-set inventory.**

This task adds a fourth verb, a fourth facade query, a fourth JSON success renderer and a fourth
text renderer. **The search predicate is therefore not "statements of the verb set" — that one
structurally cannot find a renderer-count or facade-method-count sentence, and missing five of
them was the second draft's error.** The predicate is: *any statement that counts or enumerates a
quarry surface this task extends* — verbs, facade query methods, renderers, envelopes. In
practice: search for the verb names together, for `Render` together with a number word, and for
the number words themselves (`three`, `seven`) across `*.go` and `*.md`, then read each hit and
decide whether the count is one this task changes. Every site below has a stated disposition; a
plan writer should re-run the search rather than trust this list to still be exhaustive.

Prose and contract:

- `docs/rewrite-plan.md` §5 — **add** one paragraph for `name`, in the style of the `resolve` and
  `expand` paragraphs: the contract, the batch-versus-CLI split, the same-extractor property, and
  the non-goals.
- `docs/rewrite-plan.md:12` — "three queries over one tree-sitter parse" → four. Note this sentence
  is the plan's opening framing, so the edit is a phrasing decision, not a number swap: `name` is
  a query over a *supplied fragment*, not over the tree. Recommended phrasing keeps "three queries
  over one tree-sitter parse" and adds `name` as a fourth query over the same extractor, so the
  original claim is not quietly falsified.
- `README.md:3` — "three" and the verb list → four, `toc`, `resolve`, `expand`, `name`.
- `internal/cli/doc.go:11,13` — "The command has three verbs" plus the per-verb sentences → add
  `name`'s: it takes a declaration head, which is neither a path nor a glyph.
- `internal/cli/doc.go:15-20` — **extend.** "A target is handed to the facade verbatim ... whenever
  the verb takes a glyph" is true of `name` for the same reason (no path arithmetic, no stat), but
  its stated condition excludes it; widen the condition from "whenever the verb takes a glyph" to
  "whenever the verb does not take a path". The paragraph's closing sentence, "'toc' is the only
  verb that still takes a path", stays true as written and needs no edit — worth stating so the
  next reader does not change it twice.
- `internal/cli/doc.go:22-29` — **extend.** The negative-answer paragraph says such an answer is
  "a payload carrying a status word (or, for the pre-resolution case, an error field of its own)".
  `name`'s negative payload carries `error`/`reason` and no status at all, which the parenthetical
  anticipates in shape but attributes only to `resolve`'s pre-resolution rejection; name the maker
  there explicitly. This paragraph also states what the `ok` key does and does not mean, which is
  the rule the no-`ok` decision above rests on — extending it keeps that rule's own statement true.
- `internal/cli/doc.go:31-39` — **keep, no edit.** The classification paragraph says `glyph.Parse`
  is the single classifier and that no surface tests a target for `"#"`. `name` adds no
  classification of any kind: its target is a fragment handed to the extractor, never tested for a
  separator, so the paragraph stays true. Listed here because a reader auditing the file will ask,
  and "no change" is a disposition.

Facade surface counts (the class the first two drafts' predicate could not reach):

- `quarry/doc.go:7` — "the package exposes three query methods, not one: TOC, Resolve and Expand"
  → four, naming `Name`. The sentence's rhetorical point ("not one") survives; only the count and
  the list change.
- `quarry/doc.go:11` — "The package owns seven renderers" → nine, once `RenderNameJSON` and
  `RenderNameText` land. The sentence goes on to enumerate them in three groups; `Name`'s two join
  the JSON-success group and the text group respectively.
- `quarry/doc.go:13-14` — "the three text renderers, RenderText, RenderResolveText and
  RenderExpandText" → four; and "The three JSON success renderers ... share one encoder
  configuration" → four. The second is load-bearing, not cosmetic: it is the sentence that states
  the byte-contract-cannot-drift property, and `RenderNameJSON` is inside that guarantee because
  it delegates to `renderJSON`.
- `quarry/render.go:2` — "the three successful envelopes" → four.

Help text and messages:

- `internal/cli/usage.go` — the usage block gains a `name` line; the flags list gains `--unit`
  marked `name` only. Also **the file's own doc comment at lines 11-13**, which explains the
  one-combined-flags-list layout in terms of "three per-verb sections" and "the three shared
  flags" — both counts change, and `--root` is no longer shared by every verb. The exit-code-1
  prose enumerating negative-answer causes gains the maker's own (a declaration that names no
  single symbol).
- `internal/cli/flags.go:61,66` — the two `"no verb given; expected: toc, resolve, or expand"`
  messages, byte-identical, both → include `name`. `flags.go:68` — the verb gate's
  `verb != "toc" && verb != "resolve" && verb != "expand"` chain. `flags.go:48` — the doc comment's
  "--text and --root are valid for all three verbs", which becomes false in two ways at once.
- `internal/cli/cli.go:180,302` — the two "unreachable for every word other than the three verbs"
  comments on `Run`'s dispatch `default`. `cli.go`'s `Run` doc comment also states the shared
  pipeline as "the four steps every verb shares", which the root-resolution decision above already
  requires rewriting.

Tests that pin the counts (they fail loudly, which is the point — none is a silent staleness):

- `internal/cli/flags_test.go:156,158` — the two expected message strings.
  `flags_test.go:218` — `TestParseArgs_ThreeVerbGate`, whose name and table both encode three.

Not touched:

- `internal/cli/after_test.go:4` — "The table spans three verbs" **stays true and is left alone.**
  That table drives the frozen `docs/research/output-formats/after/` goldens, which this task does
  not add to, so the comment neither goes stale nor fails; listing it among the loud failures
  would have been misleading. `name`'s goldens live in `internal/cli/testdata/name/` and are a
  separate table.
- `internal/cli/scratchtree_test.go:26,36` and `docs/rewrite-plan.md:147,172` — "three" counts
  directory levels, validation layers and harness rules, unrelated to any surface this task
  extends.
- `docs/research/output-formats/after/` — a frozen research record.

## Q&A log

- **Q:** CLI verb name — `name`, `make`, or `compose`? **A:** [auto-pick] `name`. **Why:** the task body already spells the result type `NameResult`; `make` collides with a Go builtin, and `compose` is taken by C1's self-form compose API in `glyph/`.
- **Q:** Facade entry point — package-level function or `*Repo` method? **A:** [auto-pick] package-level `quarry.Name`. **Why:** the maker performs no I/O, so a `*Repo` receiver would claim a repository dependency it does not have.
- **Q:** Per-entry success/failure discrimination — `ok` key, `Status`, or `id`+`error`? **A:** [auto-pick] `id`+`kind` on success, `error`+`reason` on failure, no `ok`. **Why:** `render.go` fixes `ok` as failure-envelope-only so it can never disagree with the exit code; this mirrors `ResolveResult`'s pre-resolution rejection exactly.
- **Q:** Echo shape — flat `unit`+`target`, or nested? **A:** [auto-pick] flat. **Why:** both halves are caller input echoed verbatim; a nested object would be the envelope's only nested echo.
- **Q:** Language parameter? **A:** [auto-pick] none, Go implied. **Why:** `resolve` already parses every target as `glyph.Go` with no override, and Go is the only implemented alphabet.
- **Q:** Synthetic file's package clause — fixed placeholder or derived from the unit? **A:** [auto-pick] fixed `package q`. **Why:** `Strategy.Symbols` takes the unit as a parameter and never reads the clause; a fixed placeholder makes that independence testable, and a unit whose last segment is not an identifier would break a derived one.
- **Q:** Bodyless heads such as `type T struct` — completion retry, or per-entry error? **A:** [auto-pick] one documented retry appending `" {}"` after a partial parse. **Why:** a type's extracted signature is cut at the body, so the round-trip criterion is unreachable without it; the retry operates on bytes and makes no naming decision.
- **Q:** Accepted declaration forms? **A:** [auto-pick] every top-level shape the Go extractor emits; interface methods and multi-name const/var specs are explicit non-goals. **Why:** an interface method's owner is not in its own head, so its glyph is not determined by (unit, head) — the stated input contract.
- **Q:** Unit and name validation — new `glyph` validator, or a round trip through the parser? **A:** [auto-pick] round-trip the produced id through `glyph.Parse`/`String`. **Why:** reuses the grammar that already owns both questions, adds no public API to the cgo-free package, and is the same assertion `TestRoundTrip_Loomyard` already makes.
- **Q:** Reason vocabulary? **A:** [auto-pick] `parse`, `no_declaration`, `several_declarations`, `internal`, plus propagated `glyph.Reason` words. **Why:** the caller must be able to tell "malformed" from "declared two things"; the closed-set-plus-enumerating-slice shape follows `glyph/errors.go`.
- **Q:** CLI invocation shape — `--unit` flag or two positionals? **A:** [auto-pick] `quarry name --unit <unit> "<decl>"`. **Why:** preserves `parseArgs`' exactly-one-target invariant across every verb.
- **Q:** Does `name` resolve a repository root? **A:** [auto-pick] no — dispatch before root resolution, and reject `--root` for the verb. **Why:** the verb reads nothing; requiring a root would fail a computable query with an unrelated error.
- **Q:** Exit codes? **A:** [auto-pick] named `codeForNameResult`: 0 with an id, 1 for a per-entry error with the payload rendered, 2 usage, 3 internal. **Why:** the D2 rule — a negative answer renders its payload with exit 1; the error envelope is for usage and internal errors only.
- **Q:** Text view? **A:** [auto-pick] `<id> <kind>` on success, `<target> error <reason>: <message>` on failure. **Why:** the failure line is `RenderResolveText`'s empty-status branch verbatim; its success branches already drop the target in favour of the id.
- **Q:** MCP tool? **A:** [auto-pick] none. **Why:** fixed by the task and §9 — facade-first, the caller is Loomyard's pipeline.
- **Q:** Facade error return? **A:** [auto-pick] none; `Name(decls) []NameResult`. **Why:** with no I/O nothing can fail batch-wide, and every failure is a property of one entry.
- **Q:** Empty batch? **A:** [auto-pick] empty, non-nil slice. **Why:** matches `symbolsOfUnit`'s and `SpansOf`'s existing rule.
- **Q:** Round-trip harvest scope over the pinned checkout? **A:** [auto-pick] every symbol except the two declared non-goals, with the exclusions counted and each asserted to produce a per-entry error. **Why:** a silent skip would let the 100 % criterion pass vacuously; asserting the error keeps the exclusions honest.
- **Q:** Goldens location? **A:** [auto-pick] `internal/cli/testdata/name/`. **Why:** `docs/research/output-formats/after/` is a frozen research record, and the roadmap already carries moving those out as a separate task.
- **Q:** Which docs change? **A:** [auto-pick, revised r2] every in-tree statement of the verb set or its count, enumerated by search — see the "Verb-set statements" inventory under Testing. **Why:** the first answer named three files from memory and missed nine sites, including two byte-identical usage messages, `usage.go`'s own layout doc comment, and a test whose *name* encodes the count.
- **Q:** How does the round trip harvest `Signature` and `Kind`? **A:** [r2] extend `roundTripSymbol` and `collectWalkSymbols`; do not write a second collector. **Why:** the existing struct carries neither field, and a parallel harvest could drift from the one the round trip is comparing against.
- **Q:** Is a complete interface declaration accepted? **A:** [r2] no — head-only or empty-bodied. **Why:** measured: `goTypeSymbols` appends the interface's method symbols, so a populated interface yields 1+N symbols and hits the exactly-one rule. A populated struct does agree with its head, because struct fields are not listable declarations.
- **Q:** What is the multi-name-spec partition key? **A:** [r2] the 5-tuple (unit, File, Start, End, Signature). **Why:** without `File`, build-tag twins sharing a signature and span across two files would be excluded and then asserted to fail, which they will not.
- **Q:** How is the vacuous-pass guard quantified? **A:** [r2] three counts pinned as constants against `72c23d9`, regenerated under `-update`, plus a structural floor that holds before they are known. **Why:** the checkout is pinned, so exact counts are stable and drift is loud; a ratio would have to be guessed without knowing the repository.
- **Q:** How is "`name` dispatches before root resolution" tested, given the scratch-tree helper builds inside the repository? **A:** [r3] `t.Chdir("/")`, asserting `name` exits 0 and `toc` exits 2 from there. **Why:** `Run` resolves the root from `os.Getwd()`, so only a working-directory change reaches the no-root state; Go 1.26 has `t.Chdir`, and it makes `cli.go:150-152`'s "which these tests never do" stale, to be updated in the same edit.
- **Q:** What exit code does the `internal` reason produce? **A:** [r3] exit 3 through `fail`'s error envelope, not exit 1 with a payload. **Why:** an unwired grammar says nothing about the caller's declaration, and `usage.go` already promises exit 3 for internal errors; the facade still carries it per-entry so a batch loses nothing.
- **Q:** Does the naming round trip call `assertSymbolRoundTrip`? **A:** [r3] no — its own `TOC` walk plus the shared collector. **Why:** that helper's second half re-runs a per-unit span lookup this test does not need, and its doc comment names go test's default timeout as a live constraint at Loomyard scale.
- **Q:** How are the new CLI goldens compared and regenerated? **A:** [r4] a new `compareNameGolden` helper over `testdata/name/`, honouring the package's existing `-update` flag, whose description must be widened. **Why:** `compareAfterGolden` hard-codes the frozen `after/` path and its caller is gated on a Loomyard checkout; `flag.Bool` panics on a duplicate name, so a second `-update` is impossible.
- **Q:** What do the golden files contain? **A:** [r4] the payload bytes only — no `$ quarry` header — with the exit code pinned in the table. **Why:** a header would make each `.json` golden invalid JSON, and the engine's own `testdata/loomyard/*.json` goldens are pure payload; `after/`'s header exists because those files are evidence documents.
- **Q:** How are the three round-trip counts pinned, given a `const` cannot be rewritten by a flag? **A:** [r4] a counts golden at `internal/engine/testdata/loomyard/naming-counts.json` through the existing `compareGolden`. **Why:** it is the half of "constants regenerated under -update" that can actually hold, and `-update`'s description already names that directory.
- **Q:** What are the `Error` sentences for the maker's four reasons? **A:** [r4] spelled as a table in the reason-vocabulary decision, none repeating the target. **Why:** the text renderer prints the target ahead of the message, and two goldens pin these bytes.
- **Q:** How does the text view survive a multi-line declaration head? **A:** [r2] `normalizeProse` the echoed target, as every `Signature` already is; JSON keeps the byte-verbatim echo. **Why:** an ungrouped var's signature is the whole declaration text and can span lines, which would break the one-record-per-line invariant.
