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
- Docs: `docs/rewrite-plan.md` §5 paragraph, `internal/cli/usage.go` help text, `README.md`'s
  verb list.

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

### Accepted declaration forms

- Decision: every top-level declaration shape the Go extractor emits is accepted — function,
  method, type (struct, interface, alias, named), const, var — in either their head-only or
  their complete form. Two shapes are explicit non-goals:
  1. **Interface methods.** An interface method's extracted signature is a bare method element
     (`Read(p []byte) (int, error)`), which is not a top-level declaration in any wrapping, and
     its owner is the enclosing interface's name, which is not present in the fragment at all. A
     Create card that adds an interface method creates it as part of the interface's own type
     declaration.
  2. **Multi-name const/var specs.** `const X, Y = 1, 2` is one declaration yielding two symbols,
     so it is rejected by the exactly-one rule above, correctly and by design.
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
  `codeForResolveResult` and `codeForExpandAnswer` are: `exitOK` when `ID` is non-empty,
  `exitNegative` when `Error` is non-empty, `exitInternal` for the shape where neither is set
  (unreachable; spelled so it cannot route to zero).
- Rationale: the D2 rule stated in `docs/rewrite-plan.md` §5 — a negative answer renders its
  payload with exit 1, and the error envelope is only for usage and internal errors with no
  payload. A malformed declaration is a negative answer *about the declaration*, which the maker
  answers with a reason; it is not a usage error.
- The payload is written to stdout before the exit code is computed, matching `runResolve`.

### Text view

- Decision: `RenderNameText(r NameResult) string`, one line, no trailing whitespace, exactly one
  trailing `"\n"`:
  - success: `<id> <kind>`
  - failure: `<target> error <reason>: <message>`
- Rationale: the failure line is `RenderResolveText`'s own empty-status branch verbatim, including
  its `normalizeProse` call on the message. The success line drops the target because
  `RenderResolveText`'s success branches already drop it in favour of the id — for a one-per-call
  CLI the input is on the command line beside the output.
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

Three things this pins down for the plan. First, the completion retry's trigger is real:
`type T struct` reports a partial parse *and* yields zero symbols, and appending `" {}"` produces
exactly the id the complete form does. Second, a bodyless func and a bodyless method both parse
clean, so the retry never fires for them. Third, the unit in every id above is the parameter
`u/v`, never the synthetic clause `q` — the unit-independence property, observed rather than
argued.

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
  complete forms produce. Assert the complete form and the head-only form agree.
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
   `Kind` — the same harvest `assertSymbolRoundTrip` already performs.
2. Partition the harvest into *in-contract* and *excluded*, by the two declared non-goals:
   interface methods (a `KindMethod` symbol whose owner names an interface — detectable because
   its signature is not a `func` declaration), and symbols sharing a signature with another symbol
   in the same unit and span (multi-name specs).
3. For every in-contract symbol, call the maker with `(unit, signature)` and assert the returned
   `id` equals the symbol's real id and the returned `kind` equals its real kind. **Zero misses,
   zero extras.**
4. For every excluded symbol, assert the maker returns a per-entry *error* — never a wrong id.
   This is what keeps the exclusions honest rather than a silent skip.
5. Report both counts in the failure message, and fail if the in-contract count is implausibly
   small (a partition bug that excludes everything would otherwise pass vacuously).

**`internal/cli` — verb wiring.**

- `parseArgs` table rows: `name` accepted; `--unit` required and present; `--unit` rejected for
  `toc`/`resolve`/`expand`; `--depth`/`--symbols`/`--no-symbols`/`--root` rejected for `name`;
  `--text` accepted; exactly one target enforced.
- `codeForNameResult` table test, direct, like the three existing mappers.
- `Run` end-to-end on a machine with no repository root reachable — asserts `quarry name` still
  answers, proving the dispatch happens before root resolution. Use the existing scratch-tree
  helper for a directory outside any repository if one is reachable; otherwise assert the
  ordering structurally.
- `usageText` assertions extended for the `name` rows.

**Goldens — `internal/cli/testdata/name/`, JSON and text for each of the six required cases:**
method, free function, type, method on a receiver type that does not exist, malformed declaration
(per-entry parse error), multi-symbol input (per-entry reject). Both views per case, so twelve
files. They are machine-independent — the maker reads no repository — so unlike the existing
Loomyard goldens they run everywhere with no environment gate. `docs/research/output-formats/after/`
is a frozen research record and must not be added to.

**Docs.**

- `docs/rewrite-plan.md` §5: one paragraph for `name`, in the style of the `resolve` and `expand`
  paragraphs — the contract, the batch-versus-CLI split, the same-extractor property, and the
  two non-goals.
- `internal/cli/usage.go`: the usage block gains a `name` line; the flags list gains `--unit` marked
  `name` only; nothing else moves.
- `README.md` line 4: the verb list becomes `toc`, `resolve`, `expand`, `name`.

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
- **Q:** Which docs change? **A:** [auto-pick] `docs/rewrite-plan.md` §5, `internal/cli/usage.go`, and `README.md`'s verb list. **Why:** the README names the query set in its opening paragraph, so leaving it at three verbs would go stale immediately.
