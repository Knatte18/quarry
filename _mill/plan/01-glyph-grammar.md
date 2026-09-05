# Batch: glyph-grammar

```yaml
task: "Glyph self-form and the resolve contract (C1)"
batch: "glyph-grammar"
number: 1
cards: 9
verify: go test ./glyph/...
depends-on: []
```

## Batch Scope

This batch delivers the whole grammar change inside package `glyph` and nothing else: the self form
as an empty member (D1, D2), the exactly-one-`#` rule with its own rejection reason (D3), the
actionable bare-path message (D4), and the `Self` compose constructor (D20). It is one batch because
`glyph/` is pure Go with no dependencies, every file is small, and the four decisions are one
grammar: `Parse` gains a count check and an empty-member accept, `errors.go` gains one constant and
loses another, and `Self` is `Parse` plus one concatenation. Nothing outside `glyph/` compiles
against anything this batch adds, so it lands green on its own.

The external interface batches 2, 3 and 5 consume: `Glyph.IsSelf() bool`,
`Self(lang Language, path string) (Glyph, error)`, `ReasonMultipleSeparators`, and the absence of
`ReasonMemberEmpty`.

Batch-local decision beyond the overview's Shared Decisions: `String()` is not edited. It already
prints `Unit + "#"` for the genuine self value, and that it needs no edit is the evidence the
representation is right (D1). Card 8 pins that with a test rather than leaving it to reading.

## Cards

### Card 1: exactly one `#`, with its own rejection reason

- **Context:**
  - `docs/glyph.md`
- **Edits:**
  - `glyph/parse.go`
  - `glyph/errors.go`
  - `glyph/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `ReasonMultipleSeparators Reason = "multiple_separators"` to the constant
  block in `glyph/errors.go`, to `Reasons` in the same order as the constant block, and to
  `reasonText` with the phrase `a glyph has exactly one "#"; a unit or member component may not
  contain one`. Its constant doc comment states that it fires when the input carries more than one
  `"#"`. In `Parse`, between the `utf8.ValidString` check and the `splitGlyph` call, count the `"#"`
  runes in `s`; a count greater than one returns the zero `Glyph` and a `*ParseError` with
  `Reason: ReasonMultipleSeparators` and `Detail` set to `s`. `splitGlyph` is then reached only when
  the count is exactly one, so its `ok == false` arm covers only the zero case; keep `strings.Cut`
  and add a sentence to `splitGlyph`'s doc comment recording that `Parse` now calls it only after
  the count check. Rewrite `checkGoUnit`'s doc comment in `glyph/golang.go`: the sentence beginning
  "There is no `"#"` check here, because the split that produced unit already consumed the first
  `"#"`" is replaced by one citing the count rule in `Parse` instead. Do not add a `"#"` rune check
  inside `checkGoUnit` — `unitSpellable` in the engine probes `unit + "#x"`, and a unit-level rune
  check would report the wrong reason for that probe. `errors.go`'s header sentences that count the
  vocabulary ("The sixteen Reason values", "Reasons lists all sixteen Reason values") stay at
  sixteen only once card 2 has removed `ReasonMemberEmpty`; card 2 owns that arithmetic, so leave
  both counts alone here.
- **Commit:** `feat(glyph): reject more than one "#" with its own reason`

### Card 2: the empty member is the self form, and `member_empty` leaves the vocabulary

- **Context:**
  - `glyph/parse.go`
  - `glyph/glyph.go`
- **Edits:**
  - `glyph/golang.go`
  - `glyph/errors.go`
  - `glyph/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `parseGo`, after the `checkGoUnit` call succeeds and before the
  `checkGoMember` call, return `Glyph{Lang: Go, Unit: unit}, nil` when `member` is the empty string.
  Update `parseGo`'s doc comment to state that the empty member is the self form and is admitted
  after the unit half is validated, so `#`, `.#` and `a//b#` are still rejects. Delete the
  `member == ""` branch in `checkGoMember` that returns `ReasonMemberEmpty`; leave the pointer,
  parens and bracket scans above it in place and do not disturb the ordering comment further down
  that function, which exists for the 7a-before-7b/7c rule. Delete the `ReasonMemberEmpty` constant,
  its entry in `Reasons`, and its `reasonText` entry from `glyph/errors.go`; the vocabulary stays at
  sixteen values because card 1 added one, so `errors.go`'s two counting sentences ("The sixteen
  Reason values a ParseError ever carries" and "Reasons lists all sixteen Reason values") remain
  correct and are not renumbered. Widen the package summary in `glyph/doc.go`: its opening sentence
  says a glyph names "one source symbol", which the self form falsifies — a self glyph names a
  directory or a file, neither of which is a source symbol — so the summary reads "names one symbol,
  unit or file" instead.
- **Commit:** `feat(glyph): accept the empty member as the self form`

### Card 3: `IsSelf` predicate

- **Context:**
  - `glyph/parse.go`
- **Edits:**
  - `glyph/glyph.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `func (g Glyph) IsSelf() bool` to `glyph/glyph.go`, returning true when
  `len(g.Owner) == 0` and `g.Name == ""` and `g.Params == nil`. Its doc comment must record why all
  three fields are tested and not merely two: `String` prints `"()"` for a non-nil empty `Params`
  because its nil-versus-non-nil state, not its length, decides the parentheses, so a hand-built
  `Glyph{Unit: "a", Params: []string{}}` would report true while printing `a#()`, breaking the
  property that removing the trailing `"#"` yields the plain path. Do not edit `String`: it already
  emits `Unit + "#"` for the genuine self value, which is the canonical form the contract wants.
- **Commit:** `feat(glyph): add the IsSelf predicate`

### Card 4: the bare-path rejection names its fix

- **Context:**
  - `docs/glyph.md`
- **Edits:**
  - `glyph/parse.go`
  - `glyph/errors.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `Parse`, the zero-separator arm sets `Detail` to `s + "#"` alongside
  `Reason: ReasonNoSeparator`. Rewrite `reasonText[ReasonNoSeparator]` to `a glyph needs a "#"; a
  path is addressed as its own glyph by appending one to its repository-relative form`. The clause
  "to its repository-relative form" is required and not decorative: `Detail` echoes the caller's
  argument verbatim, so a cwd-relative argument shows a spelling that is not a working glyph, and
  without the clause the hint misleads exactly the caller it is aimed at. Composed by
  `ParseError.Error`, the whole rendered line for the input `internal/logger` is
  `glyph: parse "internal/logger" as go: a glyph needs a "#"; a path is addressed as its own glyph by
  appending one to its repository-relative form (internal/logger#)`, and that sentence is
  authoritative — card 6 pins it verbatim. Rewrite `ReasonNoSeparator`'s own constant doc comment,
  which currently says a glyph "needs a unit and a member": the member is optional now, so the
  comment states that the input carries no `"#"` at all. Rewrite both copies of the `Detail`
  sentence — the one on `ParseError`'s type doc comment and the duplicate on the `Detail` field
  itself — which today say `Detail` carries "the offending segment, component or rune where one
  exists": it now also carries a suggested spelling for `no_separator` and the whole input for
  `multiple_separators`, so both copies must say so or they contradict each other.
- **Commit:** `feat(glyph): make the no-separator rejection name its fix`

### Card 5: the `Self` compose constructor

- **Context:**
  - `glyph/parse.go`
  - `glyph/errors.go`
- **Edits:**
  - `glyph/glyph.go`
- **Creates:**
  - `glyph/self.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `glyph/self.go` holding `func Self(lang Language, path string) (Glyph,
  error)`. It validates by delegation, never by a second copy of the unit alphabet: it builds
  `path + "#"` and returns `Parse(lang, path+"#")` directly, so a `"#"` inside `path` surfaces as
  `multiple_separators`, an empty `path` as `unit_empty`, and every other unit rule — dot segments,
  empty segments, bad runes, unsupported language — comes along unchanged and can never drift from
  `Parse`'s. The one concatenation in the whole system lives inside this function; no consumer
  performs it. `lang` is a parameter and is never defaulted to `Go`: `Language`'s zero value is
  deliberately not a valid language so a forgotten argument fails at the first call, and hardcoding
  `Go` would reintroduce exactly that silent default. The file header comment names the file's one
  function and states the delegation rule. In `glyph/glyph.go`, the `Glyph` type's doc comment
  currently says "this package exports no constructor and no Validate method"; `Self` falsifies it,
  so rewrite that sentence to name `Self` as the one constructor and to keep the standing warning
  that a `Glyph` built by hand rather than by `Parse` or `Self` is the builder's responsibility and
  that `String` does not check the value it is given. This sentence lives on the `Glyph` doc comment
  in `glyph/glyph.go`. Do not edit `glyph/doc.go` for this sentence: that file carries no such claim,
  so an edit aimed there would change nothing and leave the false one standing.
- **Commit:** `feat(glyph): add the Self compose constructor`

### Card 6: `parse_test.go` — the self accept table, the reject table, and the retargeted split test

- **Context:**
  - `glyph/parse.go`
  - `glyph/errors.go`
  - `glyph/glyph.go`
- **Edits:**
  - `glyph/parse_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a self-form accept table whose rows are `internal/reedengine/render#`,
  `internal/reedengine/render/focus.go#`, `cmd/lyx#`, and
  `internal/engine/testdata/tree/pkg_test#`. Each row asserts the returned `Glyph` has the expected
  `Unit`, a nil `Owner`, an empty `Name`, a nil `Params`, and that `IsSelf` reports true. Add a
  reject table asserting the `Reason` value — never the message text — for `internal/logger` as
  `ReasonNoSeparator` with `Detail` equal to `internal/logger#`; `logger.go` as `ReasonNoSeparator`
  with `Detail` equal to `logger.go#`, the cwd-relative row that shows the repository-relative
  clause doing its work on the exact input it exists for; `a#b#c` and `a#b#` as
  `ReasonMultipleSeparators`; `#` as `ReasonUnitEmpty`; `.#` and `a/../b#` as
  `ReasonUnitDotSegment`; `a//b#` as `ReasonUnitEmptySegment`. Add exactly one message-shape test,
  on `no_separator`, pinning card 4's authoritative sentence verbatim including the
  repository-relative clause and the parenthesised `Detail`. Retarget `TestSplitGlyph_FirstHash`:
  its `splitGlyph` assertions on `internal/logger#a#b` still hold, since `splitGlyph` still cuts at
  the first `"#"`, but its `Parse` assertion must now expect `ReasonMultipleSeparators` rather than
  `ReasonMemberBadRune`, and its doc comment must say that the second `"#"` is now caught by
  `Parse`'s count check before the split rather than reaching the Go member validator. Rename the
  test if its current name asserts the retired behaviour.
- **Commit:** `test(glyph): table the self form, the new rejects, and the retargeted split`

### Card 7: `golang_test.go` — the reject rows the grammar change moves

- **Context:**
  - `glyph/errors.go`
  - `glyph/golang.go`
  - `glyph/parse_test.go`
- **Edits:**
  - `glyph/golang_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `goReject`, delete the `member empty` row for `internal/logger#`, which is now
  an accept. Retarget the `second hash reaches the member validator` row for `internal/logger#a#b`
  to `reason: ReasonMultipleSeparators` with `detail` equal to the whole input string
  `internal/logger#a#b`, and rename the row to name the count rule instead of the member validator.
  Set `detail` on all three `ReasonNoSeparator` rows — `internal/logger`, the empty string, and
  `internal/reedengine/render.Renderer.Draw` — to the input with `"#"` appended, since card 4 makes
  `Detail` non-empty for that reason; the empty-string row's expected detail is therefore the single
  character `#`. Update the table's preamble comment: its closing sentence explains that the §7
  dotted spelling "has no `"#"`, so it fails the split", which stays true, but the sentence about a
  second `"#"` reaching the member checks does not — rewrite whichever sentences describe the old
  separator handling. `TestReasons_Completeness` must keep passing: it fails for any `Reason` in
  `Reasons` that no row of `goReject` or `parseReject` names, so `ReasonMultipleSeparators` needs the
  retargeted row above and `ReasonMemberEmpty` must no longer appear in either table.
- **Commit:** `test(glyph): retarget the reject rows the count rule and self form move`

### Card 8: `string_test.go` — the self form round trip

- **Context:**
  - `glyph/glyph.go`
- **Edits:**
  - `glyph/string_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Assert that `Glyph{Lang: Go, Unit: "a/b"}.String()` is exactly `a/b#`, verifying
  by test rather than by reading that `String` needs no change for the self form — it builds `parts`
  from `Owner` plus `Name`, and a future edit that started trimming empties would break this
  silently. Extend the existing parse-then-string round trip so every self-form row card 6 tables is
  also asserted here: parsing the string and printing the result returns the identical string.
- **Commit:** `test(glyph): pin String's self form and its round trip`

### Card 9: `self_test.go` — the compose round trip and the mirrored rejects

- **Context:**
  - `glyph/self.go`
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `glyph/errors.go`
- **Edits:** none
- **Creates:**
  - `glyph/self_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Group (a), the compose-then-parse round trip in both directions. For every path
  in a table holding `internal/logger`, `internal/logger/logger.go`, `cmd/lyx` and
  `internal/engine/testdata/tree/pkg_test`: `Self(Go, p)` returns a `Glyph` for which `IsSelf`
  reports true, whose `String` is exactly `p` with `"#"` appended, and which is
  `reflect.DeepEqual` to the `Glyph` `Parse(Go, p+"#")` returns; and parsing that string back yields
  a `Glyph` whose path — its `String` with the trailing `"#"` removed — is byte-identical to `p`.
  Assert the path identity, not merely that parsing succeeds. Group (b), the reject cases mirroring
  the count rule: `Self(Go, "a#b")` gives `ReasonMultipleSeparators`; `Self(Go, "")` gives
  `ReasonUnitEmpty`; `Self(Go, ".")` and `Self(Go, "a/../b")` give `ReasonUnitDotSegment`;
  `Self(Go, "a//b")` gives `ReasonUnitEmptySegment`; `Self(Language("python"), "x")` gives
  `ReasonUnsupportedLanguage`. Drive group (b) and the equivalent `Parse` calls from one shared
  table so that a unit rule added to `Parse` without a matching `Self` row is visible — that shared
  table is the strongest available proof that `Self` delegates rather than duplicates. Assert the
  `Reason` value via `errors.As` on a `*ParseError`, and that the returned `Glyph` is the zero value
  on every reject. The file header comment states that this file is the executable form of the
  compose direction's contract.
- **Commit:** `test(glyph): pin Self's round trip and its delegated rejects`

## Batch Tests

`verify: go test ./glyph/...` runs the whole `glyph` package, which is exactly what this batch
touches: `parse_test.go`, `golang_test.go`, `string_test.go` and the new `self_test.go`. The package
is pure Go with no dependencies and no fixtures, so the run is sub-second — there is no cheaper
scope worth naming, and no Loomyard checkout is involved.

The regression gates inside that run are `TestReasons_Completeness`, which fails if the vocabulary
and the reject tables drift apart, and the parse-then-string round trip, which fails if `String`
stops being total over the self value.
