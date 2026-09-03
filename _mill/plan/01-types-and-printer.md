# Batch: types-and-printer

```yaml
task: "The glyph package (T1)"
batch: "types-and-printer"
number: 1
cards: 3
verify: go test ./glyph/
depends-on: []
```

## Batch Scope

This batch creates the package and everything that does not depend on parsing: the package doc, the
`Language`/`Glyph` types with the total `String()` printer, the closed `Reason` vocabulary with its
`*ParseError`, and the printer's own tests. It is one batch because those four files are a single
readable unit — roughly 250 lines of Go and one test file — and because none of them needs the
parser to compile or to be tested.

Card order is chosen so every commit compiles on its own: `glyph/glyph.go` declares `Language`, and
`glyph/errors.go`'s `ParseError.Lang` field refers to it, so the type lands first.

The external interface batch 2 consumes is exactly: the `Language` type with its single `Go`
constant, the `Glyph` struct's five fields, the `Reason` constants and `Reasons` slice, and the
`*ParseError` struct and its four fields. Batch 2 adds no field and renames nothing here.

Batch-local decisions beyond the overview's Shared Decisions:

- The `Reason` constants are named `Reason` + the UpperCamel of their string value
  (`unsupported_language` becomes `ReasonUnsupportedLanguage`), matching the toc package's
  `Kind`/`KindFunction` convention.
- `Error()` composes its message from a package-level `map[Reason]string` of phrases, so the sixteen
  messages differ from one another by construction and wording can be changed in one place.
- `String()` has a value receiver, `func (g Glyph) String() string`, so a `Glyph` value and a
  `*Glyph` both satisfy `fmt.Stringer`.

## Cards

### Card 1: package doc, Language, Glyph and the total String printer

- **Context:**
  - `docs/glyph.md`
  - `internal/quarryengine/toc/doc.go`
  - `internal/quarryengine/toc/types.go`
- **Edits:** none
- **Creates:**
  - `glyph/doc.go`
  - `glyph/glyph.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  Create the package `glyph` at the repository root, import path
  `github.com/Knatte18/quarry/glyph`. Both files declare `package glyph`.

  `glyph/doc.go` holds only the package doc comment and the package clause. The doc comment states:
  what a glyph is (`unit "#" member`, one name for one source symbol); that the package is pure Go,
  standard library only, no cgo, so anything can import it without the engine; that `Parse` is the
  syntactic check and reads no source; that `String` is a total pure printer that never validates;
  and that `Go` is the only `Language` implemented today. Name `docs/glyph.md` in prose as the
  contract this package implements.

  `glyph/glyph.go` declares, in this order:

  1. `type Language string`, with a doc comment saying the zero value is deliberately not a valid
     language so a forgotten argument is an error at the first call rather than a silent Go.
  2. `const Go Language = "go"`, the only `Language` value this package defines. Do not declare
     `Python`, `CSharp`, or any other language constant, and do not stub them.
  3. `type Glyph struct { Lang Language; Unit string; Owner []string; Name string; Params []string }`
     — this field set and these names are fixed and are not redesigned. Doc comments state: `Unit`
     is the unit half; `Owner` is the enclosing type chain, outermost first, and is nil for a
     package-level Go name; `Name` is the symbol's own name; `Params` is nil for Go always, and its
     nil-versus-non-nil state is what decides whether `String` prints parentheses. The type's doc
     comment also states that a `Glyph` built by hand rather than by `Parse` is the builder's
     responsibility: the package exports no constructor and no `Validate` method, and `String` does
     not check the value it is given.
  4. `func (g Glyph) String() string`, a total pure printer: it never returns an error, never panics
     and never validates. It returns
     `g.Unit + "#" + strings.Join(parts, ".") + params`, where `parts` is a freshly allocated slice
     holding every element of `g.Owner` in order followed by `g.Name`, and `params` is the empty
     string when `g.Params` is nil and `"(" + strings.Join(g.Params, ",") + ")"` otherwise —
     including when `g.Params` is a non-nil empty slice, which prints `()`.

     Build `parts` with `make([]string, 0, len(g.Owner)+1)` followed by two `append` calls. Do not
     write `append(g.Owner, g.Name)`: that can write into the caller's backing array and corrupt a
     `Glyph` the caller still holds.

  This package exports no `New` function and no `Glyph.Validate` method. Imports in `glyph/glyph.go`
  are limited to `strings`; `glyph/doc.go` imports nothing.

- **Commit:** `feat(glyph): add the package doc, Language, Glyph and the String printer`

### Card 2: the closed Reason vocabulary and ParseError

- **Context:**
  - `docs/glyph.md`
  - `glyph/glyph.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/toc/types.go`
- **Edits:** none
- **Creates:**
  - `glyph/errors.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  `glyph/errors.go` declares `package glyph` and holds, in this order:

  1. `type Reason string`, with a doc comment saying it is the closed vocabulary a `ParseError`'s
     `Reason` field is drawn from and that no value outside the constant block below is ever
     produced.
  2. One grouped `const` block with exactly these sixteen constants, in this order, each carrying the
     explicit `Reason` type on its own line — as `internal/quarryengine/toc/types.go` does for every
     `Kind` constant — and each with its own doc comment naming what it fires on:
     `ReasonUnsupportedLanguage Reason = "unsupported_language"`,
     `ReasonInvalidUTF8 Reason = "invalid_utf8"`, `ReasonNoSeparator Reason = "no_separator"`,
     `ReasonUnitEmpty Reason = "unit_empty"`,
     `ReasonUnitEmptySegment Reason = "unit_empty_segment"`,
     `ReasonUnitDotSegment Reason = "unit_dot_segment"`,
     `ReasonUnitBadRune Reason = "unit_bad_rune"`,
     `ReasonMemberEmpty Reason = "member_empty"`,
     `ReasonMemberEmptyComponent Reason = "member_empty_component"`,
     `ReasonMemberTooDeep Reason = "member_too_deep"`,
     `ReasonMemberNotIdentifier Reason = "member_not_identifier"`,
     `ReasonMemberKeyword Reason = "member_keyword"`,
     `ReasonMemberTypeParams Reason = "member_type_params"`,
     `ReasonMemberParens Reason = "member_parens"`,
     `ReasonMemberPointer Reason = "member_pointer"`,
     `ReasonMemberBadRune Reason = "member_bad_rune"`.
     No seventeenth constant. Add none for the root-package unit or for whitespace; both are covered
     by the constants above.
  3. Immediately below the constant block, `var Reasons = []Reason{...}` listing all sixteen in that
     same order, with a doc comment saying Go cannot reflect over package-level constants, so this
     slice is the only way a test or an exhaustive caller can enumerate the vocabulary, and that
     adding a constant means adding it here in the same edit.
  4. An unexported `var reasonText = map[Reason]string{...}` giving each of the sixteen a short
     phrase naming what was wrong, e.g. `ReasonMemberPointer` maps to a phrase saying a receiver's
     pointer-ness is not part of a glyph, and `ReasonNoSeparator` maps to a phrase saying a glyph
     needs a `#` and a repository-relative path is not a glyph. All sixteen phrases must differ from
     one another.
  5. `type ParseError struct { Lang Language; Input string; Reason Reason; Detail string }` with a
     doc comment saying callers use `errors.As` and switch on `Reason`, and that `Detail` carries the
     offending segment, component or rune where one exists and is empty otherwise — a blank `Detail`
     carries no meaning and is never a discriminator.
  6. `func (e *ParseError) Error() string`, composing a complete message from `Reason`, `Lang` and
     `Input` alone, appending `Detail` in parentheses only when it is non-empty, and falling back to
     the raw `Reason` string when `reasonText` has no entry. The message must be non-empty for every
     `Reason` in `Reasons`, and the sixteen messages must be pairwise distinct for one fixed `Lang`,
     `Input` and `Detail`.

  Imports in this file are limited to `fmt`. This file must compile against `Language` as card 1
  declared it; the type is not redeclared here.

- **Commit:** `feat(glyph): add the closed ParseError reason vocabulary`

### Card 3: printer tests

- **Context:**
  - `docs/glyph.md`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `internal/quarryengine/toc/toc_test.go`
- **Edits:** none
- **Creates:**
  - `glyph/string_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  `glyph/string_test.go` opens with a file-level comment naming the tables it holds — matching
  `internal/quarryengine/toc/toc_test.go`'s own opening comment — then declares `package glyph` and
  holds a table test over hand-built `Glyph` values asserting the exact printed string. Imports are
  limited to `testing`. Cases, at minimum:

  - a package-level Go name: `Glyph{Lang: Go, Unit: "internal/logger", Name: "stderrHandlerSnapshot"}`
    prints `internal/logger#stderrHandlerSnapshot`;
  - a method with a one-element `Owner`:
    `Glyph{Lang: Go, Unit: "internal/reedengine/render", Owner: []string{"Renderer"}, Name: "Draw"}`
    prints `internal/reedengine/render#Renderer.Draw`;
  - nil `Params` prints no parentheses (covered by the two cases above, asserted explicitly in the
    case names);
  - a non-nil empty `Params`, `[]string{}`, prints `()` — for example
    `Loomyard.Engine.Layout#Renderer.Draw()`;
  - a populated `Params`, `[]string{"int", "string"}`, prints `(int,string)` — comma-separated, no
    spaces — for example `Loomyard.Engine.Layout#Renderer.Draw(int,string)`. This case is a printer
    test only and has no `Parse` counterpart; it fixes the C#-shaped rule now, while it is free;
  - a two-element `Owner` under `Lang: Go`:
    `Glyph{Lang: Go, Unit: "internal/logger", Owner: []string{"Outer", "Inner"}, Name: "handle"}`
    prints `internal/logger#Outer.Inner.handle`. Assert the printed string, not merely that the call
    returns: this is a string the Go alphabet would reject, and printing it anyway is the pure-printer
    contract made visible;
  - the zero `Glyph`, asserting only that `String()` returns without panicking.

  Add one further test asserting that `String()` does not mutate its receiver's `Owner` slice: build
  an `Owner` slice with spare capacity — `owner := make([]string, 1, 4); owner[0] = "Renderer"` —
  call `String()`, then assert `len(owner) == 1` and `owner[0] == "Renderer"` and that a second
  `String()` call returns the same string as the first.

  Do not add round-trip cases here; batch 2 adds them once the accept table exists.

- **Commit:** `test(glyph): cover the String printer and the Params nil-versus-empty rule`

## Batch Tests

`verify: go test ./glyph/` runs the one test file this batch creates, `glyph/string_test.go`. Scope
is the new package only: nothing else in the repository imports it yet, so no other package's tests
can be affected. The module-wide `verify:` in the overview —
`go vet ./... && golangci-lint run ./glyph/...` —
runs afterwards at the batch boundary and catches vet and lint regressions across the repository,
including in the new files.

`glyph/errors.go` has no test of its own in this batch; its `Error()` assertions and the `Reasons`
completeness test live in `glyph/golang_test.go` in batch 2, where the reject tables they range over
are declared. Until then `errors.go` is covered by compilation only, which is deliberate rather than
an omission.
