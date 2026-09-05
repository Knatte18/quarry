# Batch: engine-maker

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
batch: 'engine-maker'
number: 1
cards: 2
verify: go test ./internal/engine/ -run 'TestName'
depends-on: []
```

## Batch Scope

This batch delivers the maker itself: one new file in `internal/engine` holding the input and answer
shapes, the closed reason vocabulary, and the function that turns a unit plus a declaration head into
a glyph id and kind — plus its table tests. It is one batch because every part of it is a property of
one file that reads nothing from disk and depends on nothing outside the engine's existing extraction
seam. The external interface the next batch consumes is exactly three names: `Declaration`,
`NameResult`, and `Name`, plus the four exported reason constants and the slice enumerating them.

Batch-local decision: the maker's per-entry worker is unexported (`nameOne`), so the package's only
exported entry point is the batch function. That mirrors `resolveGlyphTarget` sitting behind
`Resolve`.

## Cards

### Card 1: the maker, its answer shape, and its reason vocabulary

- **Context:**
  - `internal/engine/strategy.go`
  - `internal/engine/golang.go`
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `internal/engine/treesitter/treesitter.go`
  - `glyph/errors.go`
  - `glyph/parse.go`
  - `glyph/glyph.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/name.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/engine/name.go` with a file header comment in this package's established style,
  declaring the following, in this order.

  `Declaration`, the input shape, with two exported string fields: `Unit`, the glyph unit the
  declaration will belong to, and `Decl`, the declaration head verbatim. It carries no language
  field: the maker parses as Go and builds `glyph.Go` glyphs, matching `resolveGlyphTarget`, which
  already hardcodes the alphabet.

  `NameResult`, the answer shape, positionally one per `Declaration`, with exactly these fields and
  JSON tags:

  ```go
  type NameResult struct {
      Unit   string `json:"unit"`
      Target string `json:"target"`
      ID     string `json:"id,omitempty"`
      Kind   Kind   `json:"kind,omitempty"`
      Error  string `json:"error,omitempty"`
      Reason string `json:"reason,omitempty"`
  }
  ```

  `Unit` and `Target` echo `Declaration.Unit` and `Declaration.Decl` verbatim and are always present.
  `ID` and `Kind` are set only on success; `Error` and `Reason` only on failure. There is no `ok`
  key: the failure envelope's `ok` is fixed as a key that appears only there, so it can never
  disagree with the exit code beside it. `Reason` is a plain string, not a defined type, matching the
  same field on `ResolveResult`.

  The four maker-owned reason words as exported untyped string constants —
  `NameReasonParse` = `parse`, `NameReasonNoDeclaration` = `no_declaration`,
  `NameReasonSeveralDeclarations` = `several_declarations`, `NameReasonInternal` = `internal` —
  followed by `NameReasons`, a `[]string` listing all four in the same order as the constant block.
  Go cannot reflect over package-level constants, so the slice is the only way a test can enumerate
  the vocabulary; the doc comment states that adding a constant means adding it here in the same
  edit, exactly as `glyph.Reasons` does.

  `Name(decls []Declaration) []NameResult`, exported, package-level, taking no receiver and returning
  no error. It allocates the result slice with `make([]NameResult, 0, len(decls))` so an empty input
  returns an empty, non-nil slice, and appends one `nameOne` result per input in order. It returns no
  error at all: with no I/O there is nothing that can fail batch-wide, and every failure mode is a
  property of one entry's own unit or fragment.

  `nameOne(d Declaration) NameResult`, unexported, performing these steps in this fixed order:

  1. Look up the Go strategy with `StrategyFor("go")`. A false second return value is the `internal`
     reason, with the sentence `internal error: ` followed by a constructed error naming the missing
     registration. Unreachable while Go is always wired; spelled anyway.
  2. Build the synthetic source as `"package q\n\n" + d.Decl + "\n"`. The placeholder clause is the
     fixed identifier `q` and is never derived from `d.Unit`: `Strategy.Symbols` takes the unit as a
     parameter and never reads the clause, and a unit whose last segment is not a valid Go identifier
     would break a derived one for no reason.
  3. Extract through a helper `nameExtract(strategy Strategy, unit string, src []byte) (syms []Symbol, partial bool, err error)`
     that calls `treesitter.WithTree("go", src, ...)` and, inside the callback, records the `partial`
     flag and — only when `partial` is false — appends `strategy.Symbols(unit, root, src)` to a slice
     it copies out. The callback never retains `root`; `Symbol` is a value struct of strings and
     ints, so copying the slice out is safe. A non-nil error from `WithTree` is returned unchanged.
  4. A non-nil error from step 3 is the `internal` reason, sentence `internal error: ` plus the
     error's own text.
  5. When `partial` is true, retry exactly once, building the source as
     `"package q\n\n" + d.Decl + " {}" + "\n"` and calling `nameExtract` again. A non-nil error from
     the retry is again `internal`. A retry that is still partial is the `parse` reason, sentence
     `declaration does not parse`. When the verbatim parse was clean, the retry never runs.
  6. The partial-parse disposition is decided before the count, and step 3 is what makes that
     structural rather than an ordering convention: `nameExtract` collects symbols only on a clean
     parse, so a still-partial retry returns none and the count check below is never reached for it.
     Keeping "malformed" and "declared the wrong number of things" as two crisply separate
     conditions, each with its own reason word, is the property this ordering protects.
  7. Count the symbols. Zero is `no_declaration`, sentence `declaration declares no symbol`. More
     than one is `several_declarations`, sentence
     `declaration declares N symbols; exactly one is required`, with N the actual count formatted
     into the sentence — the number is free, since the maker counted them to reject the entry.
  8. Validate the single symbol's id: call `glyph.Parse(glyph.Go, sym.ID)`. On an error, extract the
     `*glyph.ParseError` with `errors.As`, set `Reason` to `string(parseErr.Reason)` and `Error` to
     the error's own `Error()` text, whole — the one sentence that does name the target, since the
     glyph package composes it that way. Should `errors.As` fail, leave `Reason` empty rather than
     dereferencing a nil pointer, matching `resolveGlyphTarget`'s own guard. On a successful parse
     whose `String()` does not return `sym.ID` byte for byte, take the `internal` reason with a
     constructed error naming both spellings, per the overview's
     `a failed id round trip that is not a parse rejection is internal` decision.
  9. On success set `ID` to `sym.ID` and `Kind` to `sym.Kind`, and nothing else.

  Every returned `NameResult` sets `Unit` and `Target` from the input on every route, success and
  failure alike. Add a small unexported helper for building the failure result so the echo is written
  once rather than at each of the seven failure sites.

  Nothing here writes to disk, opens a repository, or takes a root.
- **Commit:** `feat(engine): the glyph maker — declaration head plus unit to glyph id and kind`

### Card 2: the maker's table tests

- **Context:**
  - `internal/engine/name.go`
  - `internal/engine/golang.go`
  - `internal/engine/golang_test.go`
  - `internal/engine/answer.go`
  - `glyph/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/name_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/engine/name_test.go`, table tests over `Name` directly, with no fixture on disk
  and no repository. Every test function name begins with `TestName_` so the batch's `verify:`
  `-run` pattern reaches all of them. Cover:

  One case per accepted declaration kind, each asserting the produced id and kind: free function,
  method, struct type, interface type, type alias, named type, ungrouped const, ungrouped var,
  grouped const member, grouped var member.

  A bare iota-continuation const member — a fragment of exactly `const B`, the signature the grouped
  const/var builder emits for a continuation spec in an iota block — as its own named case rather
  than a row folded into the grouped-const case. Measured to parse cleanly and yield one symbol. It
  gets its own case because an iota enum is the most common grouped shape in the pinned Loomyard
  checkout: if this regresses, the naming round trip fails on every enum at once, and the failure
  must be attributable to one named unit test rather than to a whole-repository walk.

  There is deliberately no `var B` counterpart row. A Go var spec always carries a type or an
  expression list, so a bare `var B` is not a shape `goGroupedConstOrVarSymbols` can ever emit — the
  iota continuation that makes `const B` real has no var equivalent — and the discussion's own
  measured table covers the const form only. The grouped-var member case listed above already covers
  the var side of the grouped shape with a fragment the extractor genuinely produces.

  Unit independence: a unit unrelated to the synthetic package clause — `internal/reedengine`, say —
  produces a glyph carrying that unit and not `q`. This is the test that proves the clause is inert.

  Nonexistent receiver type: a method on a type declared nowhere, such as
  `func (f *Focus) Reset() error`, answers normally with the id and kind. Name the case for the
  tree-sitter-does-not-type-check property it pins.

  Completion retry: `type T struct` and `type T interface` both answer, each with the same id its
  empty-bodied form (`type T struct {}` / `type T interface {}`) produces. Assert the head-only form
  and the empty-bodied form agree. A populated struct, `type S struct { F int }`, agrees with its
  head too and is asserted as its own row, so the asymmetry with an interface is pinned rather than
  assumed.

  Populated interface rejected: `type R interface { Read() error }` gives
  `NameReasonSeveralDeclarations`. Its own named case, because a reader of the accepted-forms list
  would expect it to work; the interface's own type symbol plus its method symbol are two.

  Malformed: a fragment that still fails after the retry gives `NameReasonParse`.

  Zero symbols: `func _()` and a comment-only fragment each give `NameReasonNoDeclaration`.

  Several symbols: `const X, Y = 1, 2` and two declarations in one fragment each give
  `NameReasonSeveralDeclarations`, and the error sentence carries the actual count.

  Bad unit: a unit containing a `#`, a unit with an empty segment, and a unit with a `..` segment
  each fail with the grammar's own reason word propagated as `Reason` — compared against the
  corresponding `glyph.Reason` constant converted to a string, never against a literal.

  Reason completeness: `NameReasons` contains each of the four constants exactly once and nothing
  else, mirroring the glyph package's own completeness test.

  Batch semantics: a mixed batch where one entry fails and the rest succeed asserts the result slice
  length equals the input length, that order is preserved positionally, and that every entry's `Unit`
  and `Target` echo is byte-identical to its input. A separate assertion that `Name(nil)` and
  `Name([]Declaration{})` each return an empty, non-nil slice.
- **Commit:** `test(engine): table tests for the glyph maker, its reasons, and its batch semantics`

## Batch Tests

`verify: go test ./internal/engine/ -run 'TestName'` runs exactly the new file's tests. The `-run`
narrowing is deliberate: `internal/engine` holds the repository's largest suite, including the
whole-repository round trip, and this batch touches none of it. Card 1 is compiled by the same
command, so a build break in `internal/engine/name.go` fails the batch before any assertion runs.
The full package suite is covered by `pipeline.done_gate`'s repo-wide `go test ./...` before the task
is marked done.
