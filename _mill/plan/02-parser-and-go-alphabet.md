# Batch: parser-and-go-alphabet

```yaml
task: "The glyph package (T1)"
batch: "parser-and-go-alphabet"
number: 2
cards: 5
verify: go test ./glyph/
depends-on: [1]
```

## Batch Scope

This batch adds the parser and finishes the package: the language-free layer (language check, UTF-8
check, split at the first `#`, dispatch), the Go unit and member alphabets with their two fixed
precedence orders, the three test tables the spec supplies, the round trip driven from the accept
table, and a final zero-diff gate that proves the dependency and toolchain criteria. It is one batch
because the two implementation files are mutually dependent — `parse.go` dispatches into
`golang.go` — and because every test file here is a transcription of one table that the same reader
must hold in mind at once.

It consumes batch 1's `Reason` constants, `Reasons` slice, `*ParseError`, `Language`, `Go` and
`Glyph`, and adds no field to any of them.

Batch-local decisions beyond the overview's Shared Decisions:

- The split helper is `func splitGlyph(s string) (unit, member string, ok bool)`, unexported, and is
  a single `strings.Cut(s, "#")`. It is language-free and is what every future alphabet reuses.
- The Go alphabet's entry point is `func parseGo(input, unit, member string) (Glyph, error)`,
  unexported, in `glyph/golang.go`. `Parse` passes the original input through so `*ParseError`'s
  `Input` field is the whole string, not a half.
- Offending runes in a `Detail` are quoted with `fmt.Sprintf("%q", r)`, which renders a space as
  `' '`. This keeps the package's imports to `fmt`, `strings`, `unicode` and `unicode/utf8`; do not
  reach for `strconv`.
- The three tables use two named package-level struct types, not anonymous-struct literals: card 5
  declares `rejectCase` and card 6 declares `acceptCase`, each with a `section` field naming the
  `docs/glyph.md` section the case came from, so traceability is visible in the table rather than in
  a comment. Every table-driven test composes its `t.Run` subtest name from the row's `name` and
  `section` fields, so `section` is genuinely read — a write-only unexported struct field would be a
  plausible `unused` report under the lint command the overview's module-wide `verify:` runs.
  Naming `rejectCase` once is what lets card 6's completeness test range over card 5's
  `parseReject` and card 6's `goReject` in one loop: Go's type identity for anonymous structs is
  exact, so two independently written literals would not admit a shared loop variable.
- Every `*ParseError` the package constructs sets all four fields. `Lang` and `Input` are never left
  at their zero values on any reject path, and `Detail` is the exact value card 4's rules name — both
  reject tables carry a `detail` column and assert it on every row.
- Card 8 is a **zero-diff card**: it produces no commit and carries the literal `Commit: none`. That
  is a supported card shape, reserved here for the two done criteria no verify command can express —
  a dependency-list inspection and a read of the test files' import lines.

## Cards

### Card 4: the language-free layer and the Go alphabet

- **Context:**
  - `docs/glyph.md`
  - `glyph/doc.go`
  - `glyph/errors.go`
  - `glyph/glyph.go`
- **Edits:** none
- **Creates:**
  - `glyph/golang.go`
  - `glyph/parse.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  `glyph/parse.go` declares `func Parse(lang Language, s string) (Glyph, error)` and the unexported
  `splitGlyph`. Its imports are limited to `strings` and `unicode/utf8`. `Parse` runs these steps in
  exactly this order, returning `Glyph{}` and a `*ParseError` at the first failure:

  1. If `lang != Go`, return `ReasonUnsupportedLanguage` with `Detail` set to `string(lang)`. This
     check runs before `s` is inspected at all, so an input that fails both this and the split
     reports `ReasonUnsupportedLanguage`. The zero `Language` fails here like any other value.
  2. If `!utf8.ValidString(s)`, return `ReasonInvalidUTF8` with an empty `Detail`. This is the
     language-free layer's own check, before the split, so a bad byte on either side of the `#`
     reports it.
  3. `unit, member, ok := splitGlyph(s)`; when `ok` is false — the input contains no `#` at all —
     return `ReasonNoSeparator` with an empty `Detail`.
  4. Dispatch on `lang`. With `Go` the only value that reaches this point, the dispatch is a `switch`
     with a `case Go:` returning `parseGo(s, unit, member)`, so adding a language later adds one case
     and touches nothing else. The `switch` needs a trailing `return` after it — or a `default:` arm
     returning `ReasonUnsupportedLanguage` with `Detail` set to `string(lang)`, matching step 1's
     construction exactly — only to satisfy Go's terminating-statement rule. Step 1 has already
     rejected every non-`Go` value, so that arm can never fire; say so in a comment on it, so a later
     reader does not mistake it for a live second check.

  **Every `*ParseError` `Parse` returns sets all four fields**: `Lang` to the `lang` argument as
  given, `Input` to the original `s` as given, plus `Reason` and `Detail` as each step above
  specifies. This holds for the errors `parseGo` builds too — see below.

  `splitGlyph` returns `strings.Cut(s, "#")` unchanged: the split takes the **first** `#`, so a
  second `#` stays in the member half and becomes the Go alphabet's problem, not the skeleton's.

  `glyph/golang.go` declares `parseGo` and its helpers. Its imports are limited to `fmt`, `strings`
  and `unicode`. It validates the unit half first and the member half second, so a string failing
  both reports the unit's reason.

  Every `*ParseError` `parseGo` returns sets `Lang: Go` and `Input: input` — the whole original
  string, never a half — alongside the `Reason` and `Detail` each rule below names. `input` is the
  parameter that exists for exactly this purpose; leaving `Lang` or `Input` at its zero value on any
  of the fifteen Go reject paths is a defect, because `Error()` composes its message from `Reason`,
  `Lang` and `Input`.

  **The unit half**, checked in this order, stopping at the first failure:

  1. `unit == ""` — return `ReasonUnitEmpty`, `Detail` empty.
  2. Split on `/` and walk the segments left to right, stopping at the first failing segment:
     a. the segment is empty — `ReasonUnitEmptySegment`, `Detail` empty. This is what a leading `/`,
        a trailing `/` and a doubled `//` each produce;
     b. the segment is `.` or `..` — `ReasonUnitDotSegment`, `Detail` the segment;
     c. the segment contains a rune that is `\`, an ASCII control character, or satisfies
        `unicode.IsSpace` — `ReasonUnitBadRune`, `Detail` the offending rune quoted. Every other rune
        is allowed, including Unicode letters, digits, `.`, `-`, `+` and `~`, because a Go package
        directory may legitimately be named with any of them.

     There is no `#` check here and none is added: the split consumed the first `#`, so the unit half
     can never contain one, and a check that can never fire is worse than none. There is no `/`
     check either — `/` is the segment separator and cannot occur inside a segment.

  Because the walk is left-to-right and first-failure-wins, `internal/../lo gger#run` reports
  `ReasonUnitDotSegment` and `internal//lo gger#run` reports `ReasonUnitEmptySegment`: in both the
  offending earlier segment stops the walk before the segment holding the space is reached.

  **The member half**, checked in this order, stopping at the first failure:

  1. The whole member half contains `*` — `ReasonMemberPointer`, `Detail` the quoted rune.
  2. It contains `(` or `)` — `ReasonMemberParens`, `Detail` the first of the two found scanning left
     to right, quoted.
  3. It contains `[` or `]` — `ReasonMemberTypeParams`, `Detail` the first of the two found scanning
     left to right, quoted.
  4. `member == ""` — `ReasonMemberEmpty`, `Detail` empty.
  5. Split on `.`; if any component is empty — `ReasonMemberEmptyComponent`, `Detail` empty.
  6. If there are three or more components — `ReasonMemberTooDeep`, `Detail` the whole member half,
     since no single component is at fault.
  7. Then, per component, left to right, stopping at the first component that fails:
     a. the component contains `#`, `/`, an ASCII control character, or a rune satisfying
        `unicode.IsSpace` — `ReasonMemberBadRune`, `Detail` the offending rune quoted;
     b. the component is one of Go's twenty-five reserved keywords — `ReasonMemberKeyword`, `Detail`
        the component;
     c. the component is not a Go identifier — `ReasonMemberNotIdentifier`, `Detail` the component.

  Step 7a must run before 7c, not as a trailing fallback: every rune it covers would also fail the
  identifier test, so `ReasonMemberBadRune` could otherwise never fire.

  A Go identifier is: first rune `_` or `unicode.IsLetter`; every later rune `_`, `unicode.IsLetter`
  or `unicode.IsDigit`. Unicode, not ASCII — Go identifiers are Unicode and the extractor will emit
  them. `_` is a valid identifier and is accepted, because `func _()` and `var _ = ...` are legal Go
  declarations that must have a glyph.

  The keyword set is an unexported package-level `map[string]struct{}` (or
  `map[string]bool`) holding exactly Go's twenty-five reserved words: `break`, `case`, `chan`,
  `const`, `continue`, `default`, `defer`, `else`, `fallthrough`, `for`, `func`, `go`, `goto`, `if`,
  `import`, `interface`, `map`, `package`, `range`, `return`, `select`, `struct`, `switch`, `type`,
  `var`. Predeclared names such as `len`, `string` and `nil` are not keywords and are accepted.

  On success `parseGo` returns `Glyph{Lang: Go, Unit: unit, Owner: owner, Name: name, Params: nil}`,
  where `owner` is `[]string{components[0]}` when there are two components and nil when there is one,
  and `name` is the last component. `Params` is nil for every Go parse, without exception.

  An unexported helper `isASCIIControl(r rune) bool` returns true for `r < 0x20` and for `r == 0x7f`.

- **Commit:** `feat(glyph): add Parse, the language-free split and the Go alphabet`

### Card 5: the language-free layer's tests

- **Context:**
  - `docs/glyph.md`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `glyph/golang.go`
  - `glyph/parse.go`
  - `internal/quarryengine/toc/toc_test.go`
- **Edits:** none
- **Creates:**
  - `glyph/parse_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  `glyph/parse_test.go` opens with a file-level comment naming what it covers — the language-free
  layer — matching `internal/quarryengine/toc/toc_test.go`'s own opening comment, then declares
  `package glyph`: white-box, because `splitGlyph` is unexported.
  Imports are limited to `errors`, `reflect` and `testing`. It holds:

  1. A table test over `splitGlyph` covering every row of the `docs/glyph.md` §1 examples table,
     asserting the unit and member halves each row divides into. That includes the four Go rows and
     the three non-Go rows, which are the only non-Go examples in the spec that are whole glyphs:
     `loomyard.engine.layout#Beta.Inner.handle` divides into `loomyard.engine.layout` and
     `Beta.Inner.handle`; `Loomyard.Engine.Layout#Renderer.Draw(int)` and
     `Loomyard.Engine.Layout#Renderer.Title` divide at their own `#` likewise. Each case names the
     language the spec's table gives it, to make plain that the split needed none of them. The §2 and
     §3 non-Go examples are bare halves with no `#`; they are out of this task's reach and do not
     appear.
  2. A first-`#` test: a string carrying two `#` splits at the first, leaving the second in the
     member half; and `Parse(Go, "internal/logger#a#b")` then reports `ReasonMemberBadRune`, showing
     the second `#` reached the Go member validator rather than a pre-split check.
  3. A named package-level struct type, declared in this file and reused by card 6 rather than
     restated:

     ```go
     type rejectCase struct {
     	name    string
     	lang    Language
     	input   string
     	reason  Reason
     	detail  string
     	section string
     }
     ```

     and a package-level `var parseReject = []rejectCase{...}` holding every reject case declared in
     this file. Both must be package-level and named, because card 6's completeness test ranges over
     `parseReject` alongside its own `goReject` in one loop, from another file: Go's type identity
     for anonymous structs is exact, so a shared loop needs a shared named type. `section` names the
     `docs/glyph.md` section the case came from, or is empty for a case the spec does not write down.
     `detail` is the exact `ParseError.Detail` the row must produce, per card 4's rules — for these
     rows that is `string(lang)` for every `unsupported_language` case and the empty string for the
     `invalid_utf8` case.
     The rows are the `unsupported_language` cases of item 4 below, the reject-precedence case of
     item 5 and the `invalid_utf8` case of item 6. A test in this file drives the slice, asserting
     each row's `Reason` **and** `Detail`, and that the returned `Glyph` is the zero value.
  4. An `unsupported_language` group in `parseReject` covering `Language("python")`,
     `Language("csharp")`, `Language("")` and one arbitrary value, each asserting
     `ReasonUnsupportedLanguage` and that the returned `Glyph` is the zero value.
  5. A reject-precedence row in `parseReject`: `Parse(Language("python"), "no-hash")` — an input
     failing both the language check and the split — returns `ReasonUnsupportedLanguage`, not
     `ReasonNoSeparator`. This pins the order so a later refactor cannot silently swap the two
     checks.
  6. An `invalid_utf8`-precedes-the-split row in `parseReject`: an input that is both invalid UTF-8
     and missing a `#` reports `ReasonInvalidUTF8`, not `ReasonNoSeparator`.

  Every reject assertion extracts the reason with `errors.As` into a `*ParseError` and compares
  `Reason`; none asserts on message text. Every reject assertion also checks the returned `Glyph`
  equals `Glyph{}` with `reflect.DeepEqual`.

- **Commit:** `test(glyph): cover the language-free split, language check and reject precedence`

### Card 6: the Go alphabet's accept and reject tables

- **Context:**
  - `docs/glyph.md`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `glyph/golang.go`
  - `glyph/parse.go`
  - `glyph/parse_test.go`
  - `internal/quarryengine/toc/toc_test.go`
- **Edits:** none
- **Creates:**
  - `glyph/golang_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  `glyph/golang_test.go` opens with a file-level comment naming the tables it holds — matching
  `internal/quarryengine/toc/toc_test.go`'s own opening comment — then declares `package glyph`.
  Imports are limited to `errors`, `reflect` and `testing`.

  **The accept table** is a package-level `var goAccept = []acceptCase{...}`, where `acceptCase` is a
  named package-level struct type declared in this file with fields `name string`, `input string`,
  `want Glyph` and `section string`. Both are package-level, not function-local, because card 7's
  round-trip test in another file is driven from the slice. `section` names the `docs/glyph.md`
  section the case came from. Each case asserts the whole parsed `Glyph` with `reflect.DeepEqual`, so `Owner` being nil for
  a package-level name and `Params` being nil always are checked on every row. At minimum:

  - `internal/logger#stderrHandlerSnapshot` (§1) — nil `Owner`, `Name` `stderrHandlerSnapshot`;
  - `internal/logger#dualHandler.stderr` (§1) — `Owner` `["dualHandler"]`;
  - `internal/reedengine/render#Renderer.Draw` (§1);
  - `cmd/lyx#run` (§1);
  - `internal/shedrecipe#Lookup` (§7);
  - `internal/logger#init` (§3) — a plain identifier to the parser; its multiplicity is resolution's
    business;
  - `internal/logger_test#SomeName` (§2) — the external test package's unit is an ordinary path with
    no rule of its own;
  - `internal/logger#Box` (§3) — the canonical form of the type-parameter corner case;
  - `internal/logger#dualHandler.Handle` and `internal/logger#durableHandler.Handle` (§3), both
    present, with an assertion that the two parse to `Glyph` values differing in `Owner` alone —
    that is §3's point that the receiver type is half the key;
  - a single-segment unit, e.g. `glyph#Parse`;
  - a deep unit, e.g. `a/b/c/d/e#Name`;
  - a Unicode member identifier, e.g. `internal/logger#Ærlig`;
  - `_` as a member name, e.g. `internal/logger#_`;
  - a unit segment carrying `.`, `-`, `+` and `~`, e.g. `internal/go-lib.v2+x~1#Name`.

  **The reject table** is a package-level `var goReject = []rejectCase{...}`, reusing the
  `rejectCase` type card 5 declares rather than restating a shape, so the completeness test can range
  over `goReject` and `parseReject` in one loop. Every row's `lang` field is `Go`. Every case asserts
  the `Reason` **and** the `Detail` via `errors.As`, **and** that the returned `Glyph` equals
  `Glyph{}` — never the message text. The `detail` column below is the exact `ParseError.Detail`
  card 4's rules produce for that row, and the reject test asserts it string-for-string: the whole
  `Detail` contract is prescribed to the rune, so leaving it unasserted would let every one of those
  rules drift unnoticed. Note that a quoted rune is `fmt.Sprintf("%q", r)`'s output — a space renders
  as `' '`, a tab as `'\t'`. The rows, at minimum, are exactly these:

  | input | expected `Reason` | expected `Detail` |
  |---|---|---|
  | `internal/logger` | `ReasonNoSeparator` | `""` |
  | the empty string | `ReasonNoSeparator` | `""` |
  | `internal/reedengine/render.Renderer.Draw` | `ReasonNoSeparator` | `""` |
  | `#run` | `ReasonUnitEmpty` | `""` |
  | `/internal/logger#run` | `ReasonUnitEmptySegment` | `""` |
  | `internal/logger/#run` | `ReasonUnitEmptySegment` | `""` |
  | `internal//logger#run` | `ReasonUnitEmptySegment` | `""` |
  | `./internal/logger#run` | `ReasonUnitDotSegment` | `"."` |
  | `internal/../logger#run` | `ReasonUnitDotSegment` | `".."` |
  | `internal/../lo gger#run` | `ReasonUnitDotSegment` | `".."` |
  | `internal//lo gger#run` | `ReasonUnitEmptySegment` | `""` |
  | `internal\logger#run` | `ReasonUnitBadRune` | `` `'\\'` `` |
  | `internal/my logger#run` | `ReasonUnitBadRune` | `` `' '` `` |
  | a leading space before `internal/logger#run` | `ReasonUnitBadRune` | `` `' '` `` |
  | a horizontal tab inside the second unit segment | `ReasonUnitBadRune` | `` `'\t'` `` |
  | `internal/logger#` | `ReasonMemberEmpty` | `""` |
  | `internal/logger#.Handle` | `ReasonMemberEmptyComponent` | `""` |
  | `internal/logger#Handle.` | `ReasonMemberEmptyComponent` | `""` |
  | `internal/logger#A..b` | `ReasonMemberEmptyComponent` | `""` |
  | `internal/logger#A.B.c` | `ReasonMemberTooDeep` | `"A.B.c"` |
  | `internal/logger#1abc` | `ReasonMemberNotIdentifier` | `"1abc"` |
  | `internal/logger#a-b` | `ReasonMemberNotIdentifier` | `"a-b"` |
  | `internal/logger#func` | `ReasonMemberKeyword` | `"func"` |
  | `internal/logger#range` | `ReasonMemberKeyword` | `"range"` |
  | `internal/logger#Box[T]` | `ReasonMemberTypeParams` | `` `'['` `` |
  | `internal/logger#Renderer.Draw(int)` | `ReasonMemberParens` | `` `'('` `` |
  | `internal/logger#(*dualHandler).Handle` | `ReasonMemberPointer` | `` `'*'` `` |
  | `internal/logger#*dualHandler.Handle` | `ReasonMemberPointer` | `` `'*'` `` |
  | `internal/logger#a#b` | `ReasonMemberBadRune` | `` `'#'` `` |
  | `internal/logger#A .b` | `ReasonMemberBadRune` | `` `' '` `` |
  | a trailing space after `internal/logger#run` | `ReasonMemberBadRune` | `` `' '` `` |
  | a `0xff` byte inside the unit half | `ReasonInvalidUTF8` | `""` |
  | a `0xff` byte inside the member half | `ReasonInvalidUTF8` | `""` |

  The `ReasonUnsupportedLanguage` row lives in card 5's `parseReject` slice rather than here; the
  completeness test below therefore ranges over `goReject` and `parseReject` together. Both slices
  are package-level in the same `package glyph`, so this file reads card 5's without importing
  anything.

  Annotate the deliberately ambiguous rows in the table's own comments, since they are what the two
  precedence orders exist to settle: the doubly-invalid unit rows report the leftmost failing
  segment's reason; `(*dualHandler).Handle` carries both `*` and parentheses and reports
  `ReasonMemberPointer`; the §7 dotted spelling never reaches the member checks at all.

  **The completeness test** ranges over `Reasons` and fails for any element that no row of
  `goReject` or `parseReject` names. State in a comment what it does and does not guarantee: adding a
  seventeenth constant and listing it in `Reasons` fails until a reject case exists; adding one
  without listing it in `Reasons` is caught by review, not by any test.

  **A `ParseError` field test** covering the two fields the reject rows do not carry a column for:
  for one unit reject and one member reject — for example `internal/../logger#run` and
  `internal/logger#func` — assert that the `*ParseError` recovered with `errors.As` has `Lang` equal
  to `Go` and `Input` equal to the whole input string that was passed to `Parse`, not a half of it.
  Assert the same two fields for one `unsupported_language` case, where `Lang` is the rejected
  `Language` value rather than `Go`. `Detail` needs no separate test: the reject tables' own `detail`
  column asserts it on every row, in both files.

  **The `Error()` tests**, both ranging over `Reasons` so they stay complete as the vocabulary
  changes: every `Reason` produces a non-empty `Error()`, and the sixteen messages are pairwise
  distinct for one fixed `Lang`, `Input` and `Detail`. These are smoke assertions that pin the
  property without freezing wording; the reject rows continue to assert `Reason` only.

  **A case-sensitivity test**: `internal/Logger#Foo` and `internal/logger#foo` both parse, and their
  `Glyph` values differ; neither folds into the other.

- **Commit:** `test(glyph): cover the Go accept and reject tables and the Reason vocabulary`

### Card 7: the round trip, both directions

- **Context:**
  - `docs/glyph.md`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `glyph/golang.go`
  - `glyph/golang_test.go`
  - `glyph/parse.go`
- **Edits:**
  - `glyph/string_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  Append two round-trip tests to `glyph/string_test.go`, both driven from card 6's package-level
  `goAccept` table rather than a second hand-written list, and both total over it with no case
  carved out:

  1. For every accepted input `s`, `Parse(Go, s)` then `String()` returns exactly `s`.
  2. For every accepted input, parsing the printed form of the parsed `Glyph` yields a `Glyph` equal
     to the first, compared with `reflect.DeepEqual`.

  Because the Go alphabet accepts only the canonical spelling, this property is total: if a case
  needs an exception, the strictness rule has been broken somewhere and the implementation is what
  should change, not the test. Add `reflect` to the file's imports; leave the existing printer cases
  unchanged.

- **Commit:** `test(glyph): assert the Parse/String round trip in both directions`

### Card 8: dependency and toolchain gate

- **Context:**
  - `docs/glyph.md`
  - `glyph/golang_test.go`
  - `glyph/parse_test.go`
  - `glyph/string_test.go`
  - `go.mod`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  A zero-diff verification card. Run each of the following from the repository root and confirm the
  stated result; if any fails, the failure is a defect in an earlier card of this plan and is fixed
  there rather than worked around here.

  1. `go list -deps ./glyph` — every package printed is standard library, and no non-stdlib module
     appears anywhere in the output. This is the no-cgo, no-dependency proof, and it is about the
     transitive list rather than the four packages the source imports.
  2. Read the import lines of the three `_test.go` files in the new package and confirm none imports
     anything outside the standard library — `go list -deps` cannot see test imports, so this half of
     the rule is a read, not a command. In particular `github.com/google/go-cmp` must be absent.
  3. Confirm `go.mod` is unchanged by this task: no new `require` line, no new module.

  These three are exactly the checks the plan's verify commands do not already cover. `go test` is
  the batch's own `verify:`, `go vet ./...` and `golangci-lint run` are the overview's module-wide
  `verify:` at the same batch boundary, and `go test ./...` is the hub's done gate; none of them is
  repeated here.

  Produce no diff. This card exists so the done criteria are executed and recorded rather than
  assumed.

- **Commit:** none

## Batch Tests

`verify: go test ./glyph/` runs all three test files — `glyph/parse_test.go`,
`glyph/golang_test.go` and `glyph/string_test.go`. Scope is the new package only: nothing else in
the repository imports it, so no other package's tests can be affected, and the full repository suite
would add only the two pre-existing extractor packages that this batch cannot touch. The module-wide
`verify:` in the overview — `go vet ./... && golangci-lint run` — runs afterwards at the batch
boundary.

Card 8's checks are not Go tests on purpose: the dependency guarantee is a toolchain question, and
shelling out to `go list` from a unit test would be slow, environment-dependent and a duplicate of a
command the plan already runs.
