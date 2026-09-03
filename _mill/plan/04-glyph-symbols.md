# Batch: glyph-symbols

```yaml
task: "Engine core (T3)"
batch: "glyph-symbols"
number: 4
cards: 8
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
depends-on: [3]
```

## Batch Scope

This batch moves the engine onto the glyph. `Symbol` gains its `glyph.Glyph`, its `id`, its `file`
and its head span and loses `Name` and `Owner`; `Kind` widens to the five values glyph.md's
Go table names; the Go declaration walk grows package-level `const` and `var` and interface
methods, strips type parameters from a generic receiver's owner, and skips the blank identifier; and the walk derives each file's unit, including the external `_test` unit and the
unspellable-unit exclusion.

After this batch a `toc` answer emits §4's symbol entry exactly — `id`, `kind`, `start`, `sigend`,
`end`, `signature`, `doc` — which is what makes batch 6's goldens meaningful and batch 5's round
trip possible.

The external interface batch 5 consumes: `Symbol` with `Glyph`/`ID`/`File`/`HeadStart`/`HeadEnd`,
`Strategy.Symbols(unit string, root *ts.Node, src []byte) []Symbol`, and the walk's unit derivation
seam `unitFor`.

Batch-local decision: the unspellable-unit check asks `glyph.Parse` and believes the answer rather
than restating any alphabet rule in the engine. The `glyph` package is read and imported here and is
never modified — it is the one implementation of the grammar.

## Cards

### Card 23: Symbol carries the glyph

- **Context:**
  - `glyph/glyph.go`
  - `glyph/doc.go`
  - `docs/glyph.md`
  - `docs/rewrite-plan.md`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Replace the `Symbol` struct with:

  ```go
  type Symbol struct {
      Glyph     glyph.Glyph `json:"-"`
      ID        string      `json:"id"`
      Kind      Kind        `json:"kind"`
      File      string      `json:"file,omitempty"`
      Start     int         `json:"start"`
      SigEnd    int         `json:"sigend,omitempty"`
      End       int         `json:"end"`
      Signature string      `json:"signature"`
      Doc       string      `json:"doc,omitempty"`
      HeadStart int         `json:"-"`
      HeadEnd   int         `json:"-"`
  }
  ```

  `Name` and `Owner` are removed: a caller reads `sym.Glyph.Name` and `sym.Glyph.Owner`, and the doc
  comment must say that two parallel spellings of one identity is exactly the drift the
  one-implementation-of-the-grammar rule exists to prevent. `Docstring` is renamed `Doc` to match
  §4's key. `ID` is stored at build time rather than computed in a custom `MarshalJSON`, so the
  emitted key set stays a plain struct with plain tags; say so. `File` is empty — and therefore
  omitted — inside a `toc` answer, where the symbol already sits in its file entry, and is filled by
  the span lookup batch 5 adds and by the later resolve and expand verbs, whose entries do span
  files. `HeadStart`/`HeadEnd` are JSON-hidden, populated only for `KindType`, and for Go equal the
  type declaration's own span — for **every** Go type, interfaces included; the doc comment must say
  that the subtraction of member spans from the head is the consumer's job, not the extractor's,
  which is why one span pair suffices and no discontiguous span type is needed.

  Widen `Kind` to the closed set `KindFunction`, `KindMethod`, `KindType`, `KindConst`, `KindVar`
  with the string values `function`, `method`, `type`, `const`, `var`, and update the comment that
  currently says three values. Add the `glyph` import.
- **Commit:** `feat(engine): key Symbol by its glyph`

### Card 24: Strategy.Symbols takes the unit

- **Context:**
  - `glyph/glyph.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/strategy.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Change the interface method to
  `Symbols(unit string, root *ts.Node, src []byte) []Symbol`. Its doc comment must state that `unit`
  is the glyph unit every symbol in this file belongs to, that the strategy builds each symbol's
  `Glyph` and `ID` from it, and that a strategy never derives the unit itself — the walk owns unit
  identity because it is a directory-level fact, not a file-level one. Delete the sentence promising
  that `Docstring` is untrimmed and sentence trimming is the entry point's job; there is no trimming
  any more.
- **Commit:** `feat(engine): pass the glyph unit into Strategy.Symbols`

### Card 25: The walk derives the unit

- **Context:**
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `glyph/errors.go`
  - `docs/glyph.md`
  - `internal/engine/strategy.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/walk.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `func unitFor(dirRel, dirPkg, fileClause string) string`: a file whose
  package clause is exactly `dirPkg + "_test"` belongs to the unit `dirRel + "_test"`; every other
  `.go` file belongs to `dirRel`. Deriving the external-test unit from that exact spelling rather
  than from any `_test` suffix is what keeps a package legitimately named `mytest` or `httptest`
  from being split, and a `_test.go` file declaring the package itself belongs to the package's own
  unit — the discriminator is the clause, never the filename. Say both in the comment.

  The repository root is the exception to the suffix rule: `dirRel` is `"."` there, and `unitFor`
  returns the empty string for **both** branches, so a root-level external test package does not
  become the unit `"._test"`. Left unhandled it would, and `glyph.Parse` accepts `._test` as a legal
  segment — so root-level `_test` files would be emitted under a unit while their siblings in the
  same directory are excluded as unspellable. The empty string keeps the root wholly out, which is
  the same open gap the next paragraph records, not a second rule.

  Add `func (r *Repo) unitSpellable(unit string) bool`: one call to `glyph.Parse(glyph.Go, unit +
  "#x")`, made once per directory before any extraction, returning whether it succeeded. A directory
  whose unit the Go alphabet cannot spell is still **listed** — its files get `Name`, `Header`,
  `Test` and `Generated` like any other file — but no file entry in it carries `Symbols`. This
  covers every rejection the alphabet makes with one rule rather than restating any of them: the
  repository root's empty unit, a `.` or `..` segment, and a segment holding a space, a backslash or
  an ASCII control rune. State in the comment that emitting nothing is the honest answer to a name
  the contract cannot spell, that the alternative would be minting a unit spelling `glyph.Parse`
  rejects, and that what the repository root's unit should spell is an open gap in the identifier
  contract rather than something the engine decides.

  Thread both through `fileEntry`: it takes the file's unit and passes it to `Strategy.Symbols`, and
  it leaves `Symbols` nil when the directory's unit is unspellable, whatever `wantSymbols` says.
- **Commit:** `feat(engine): derive each file's glyph unit in the walk`

### Card 26: Glyphs for functions, methods and types

- **Context:**
  - `glyph/glyph.go`
  - `glyph/golang.go`
  - `docs/glyph.md`
  - `internal/engine/nodes.go`
  - `internal/engine/answer.go`
  - `internal/engine/text.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Change `Symbols` on `goStrategy` to the new signature and thread `unit` down to
  every builder. Replace `goDeclSymbol`'s `name`/`owner` string parameters with the construction of
  a `glyph.Glyph{Lang: glyph.Go, Unit: unit, Owner: owner, Name: name}` — `Owner` nil for a
  package-level name and a one-element slice for a method — setting `Symbol.Glyph` and
  `Symbol.ID = g.String()`. Rename the `Docstring` assignment to `Doc`. Do the same in
  `goUngroupedTypeSymbol` and `goGroupedTypeSymbol`, and set `HeadStart`/`HeadEnd` on both to the
  same `Start`/`End` the type symbol already carries.

  Fix `goReceiverTypeName` for generic receivers: when the receiver's type, after unwrapping a
  `pointer_type`, is a `generic_type`, take that node's `type` field's identifier rather than the
  node's whole text — otherwise `func (b *Box[T]) M()` yields the owner `Box[T]` and an id the
  grammar rejects with a type-parameters reason. Type *names* are already bare, so this is the one
  place the rule has to be applied; say so in the comment and cite the contract's own wording that
  type parameters are not part of a glyph.

  Skip any declaration whose name is `_`, in every builder: `var _ = ...`, `var _ Iface =
  (*T)(nil)` and `type _ struct{}` name nothing a plan can target, and one glyph for the blank
  identifier would collapse many declarations the way `init` does but without `init`'s defined
  meaning. Record in the comment that the identifier contract is silent here and this is quarry's
  own choice, made so that every emitted id stays addressable.

  Several `func init()` in one package all carry the one id `<unit>#init` and are listed separately
  with their own spans — that falls out of building the glyph from the name, so state it in the
  comment rather than special-casing it.
- **Commit:** `feat(engine): build glyphs for Go functions, methods and types`

### Card 27: Package-level const and var

- **Context:**
  - `glyph/glyph.go`
  - `docs/glyph.md`
  - `internal/engine/nodes.go`
  - `internal/engine/answer.go`
  - `internal/engine/text.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Handle `const_declaration` and `var_declaration` children of `source_file` in
  `Symbols`, emitting one symbol per declared **name** with `KindConst` or `KindVar`. Confirm the
  node kind names against the pinned tree-sitter-go v0.25.0 grammar before writing the walk rather
  than assuming them. Reuse `goDeclIsGrouped`'s literal-`(`-child test to tell the grouped and
  ungrouped forms apart, for the same reason it is used on types — a single-spec group is legal and
  a spec-count test would misroute it.

  Per shape, and stated explicitly because the round trip compares two readings of one walk and
  cannot catch a wrong reading:

  | shape | Start / End | Signature | SigEnd |
  |---|---|---|---|
  | ungrouped `const X = 1`, `var x int` | the declaration's span, doc via `CommentBlockAbove` on the declaration, since it is the node with `source_file` siblings | `SignatureCut(decl, nil, src)` — the whole declaration text trimmed, carrying the keyword | `0` |
  | grouped, one spec | the spec's span, doc via `CommentBlockAbove` on the spec | the keyword prepended to the spec's own text, mirroring the grouped type's rule so grouped and ungrouped render identically | `0` |
  | several names in one spec | one symbol per name, all sharing that spec's span, doc and signature text | as its row above | `0` |
  | a bare `iota` spec with no type and no value | the spec's span | the keyword plus the spec text, verbatim, never synthesised from the preceding spec | `0` |

  In every row `End` is the node's own last line and `Start` is the first line of the comment block
  `CommentBlockAbove` attaches when it returns one, and the node's own first line otherwise — the
  rule the existing declaration builder already implements. `SigEnd` is `0` for all of them because
  none has a body-bearing child, which `omitempty` turns into an absent key. `HeadStart`/`HeadEnd`
  stay zero: they are populated for types only. Several names in one spec produce distinct glyphs
  over identical spans, which is fine and needs no special case.
- **Commit:** `feat(engine): list package-level const and var declarations`

### Card 28: Interface methods and the head span

- **Context:**
  - `glyph/glyph.go`
  - `docs/glyph.md`
  - `internal/engine/nodes.go`
  - `internal/engine/answer.go`
  - `internal/engine/text.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** List a `method_elem` inside an `interface_type` as a `KindMethod` symbol whose
  owner is the interface's type name. Confirm the node kind name against the pinned tree-sitter-go
  v0.25.0 grammar first. Its span is the `method_elem`'s own, its doc comes from
  `CommentBlockAbove` on the `method_elem`, its signature is the `method_elem`'s own text, and its
  `SigEnd` is `0`.

  Only one kind of `interface_type` is walked: the one that is the `type` of a file-scope
  `type_spec` or `type_alias` — the interface named by a `type X interface { ... }` declaration. An
  **anonymous** interface, in a struct field, a parameter, a return type, a `var`, or a generic
  constraint, is never descended into and its methods are never listed: it has no type name to own
  them, and the identifier contract excludes struct fields and local declarations outright. An
  *embedded* interface is a `type_elem` rather than a `method_elem`, so it falls out of the walk
  naturally rather than by a special case, and the embedded name is not a member of the embedder.
  State the bound and its reason in the comment: without it the walk emits owner-less or
  wrongly-owned glyphs, and the round trip — two readings of one walk — cannot catch that.

  The interface type symbol's own `HeadStart`/`HeadEnd` are the type declaration's span, exactly as
  for a struct: an interface is the one Go type whose declaration contains its own members, so the
  member spans lie inside the head range and the later expand verb renders the head by omitting the
  lines its member symbols cover. Say so where `HeadStart` is set.
- **Commit:** `feat(engine): list interface methods and set the type head span`

### Card 29: Port the Go strategy tests

- **Context:**
  - `internal/engine/golang.go`
  - `internal/engine/answer.go`
  - `internal/engine/walk.go`
  - `glyph/glyph.go`
- **Edits:**
  - `internal/engine/golang_test.go`
  - `internal/engine/toc_test.go`
  - `internal/engine/answer_test.go`
  - `internal/engine/toc_integration_test.go`
  - `internal/engine/classify_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Every existing assertion on a symbol's `Name` or `Owner` becomes an assertion on
  its `ID`, and every assertion on `Docstring` becomes one on `Doc`. `TestGoStrategy_Symbols`,
  `TestGoStrategy_SymbolsAscendingByStart` and `TestGoStrategy_SigEnd` all call `Symbols` with an
  explicit unit argument. Keep the cases' existing coverage — a package-level `func`, a method with a
  pointer and a value receiver, an ungrouped and a grouped `type`, and a type alias whose absent
  `sigend` is asserted through the marshalled JSON. In `toc_test.go`, `answer_test.go` and
  `toc_integration_test.go`, update the symbol assertions the same way; the integration test's three
  expected function names become their three expected ids. In `classify_test.go`, `fakeStrategy`'s
  `Symbols` method gains the `unit string` parameter so it still satisfies the interface — without
  it the package does not build.
- **Commit:** `test(engine): port the Go strategy tests onto the glyph-keyed symbol`

### Card 30: Glyph, widening and unit tests

- **Context:**
  - `internal/engine/golang.go`
  - `internal/engine/walk.go`
  - `internal/engine/answer.go`
  - `internal/engine/scratchtree_test.go`
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `glyph/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/glyph_test.go`
  - `internal/engine/testdata/glyphs/decls.go`
  - `internal/engine/testdata/glyphs/iface.go`
  - `internal/engine/testdata/glyphs/generic.go`
  - `internal/engine/testdata/glyphs/blank.go`
  - `internal/engine/testdata/glyphs/inits.go`
  - `internal/engine/testdata/units/root.go`
  - `internal/engine/testdata/units/test data/pkg/spaced.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** The fixture files under `testdata/glyphs/` declare one package and between them
  carry: the five `const`/`var` shapes of card 27's table; a named interface with two documented
  methods, an anonymous interface in a struct field, one in a parameter and one in a generic
  constraint, and an embedded interface; a generic type with both a value-receiver and a
  pointer-receiver method; a `var _ = ...`, a `var _ Iface = (*T)(nil)` and a `type _ struct{}`; and
  three `func init()`. Under `testdata/units/`, a file in a directory whose name contains a space gives the bad-rune
  case, and `root.go` — a `.go` file sitting directly in `testdata/units/` — gives the empty-unit
  case, which is only reachable when that directory is itself the repository root.

  `glyph_test.go` asserts:

  - Glyph assignment: a package-level function's id, a method's `owner.name` id, an interface
    method's owner being the interface's type name, and — the case that would otherwise slip —
    a method on a generic type whose owner is the bare type name and never the parameterised one,
    asserted by round-tripping the id through `glyph.Parse` and `String`.
  - The blank identifier: every `_`-named declaration is **absent** from the symbol list.
  - `init`: three symbols, one id, spans in file-then-line order.
  - The `const`/`var` shapes: each asserted on id, span, signature and the absence of `sigend`.
  - The interface walk's scope: the named interface's methods listed with the right owner, and none
    of the anonymous or embedded cases contributing a symbol.
  - The interface head span: the type symbol's `HeadStart`/`HeadEnd` cover the whole declaration and
    every member symbol's span lies inside that range.
  - Unspellable units, two cases with different entry points. The bad-rune case is queried from
    the quarry root as usual: the space-bearing directory's file is listed with its header and
    carries no `symbols`. The empty-unit case cannot be reached that way — from the quarry root
    that file's unit is a perfectly legal path — so the test `Open`s the `testdata/units` directory
    as its **own** repository root and queries `"."`, where the unit genuinely is the empty string:
    the file is listed with its header and carries no `symbols`. Assert in both cases that the
    entry is otherwise unchanged, and that `SpansOf` returns nothing for any name in either.
- **Commit:** `test(engine): cover glyph assignment, the widened walk and unspellable units`

## Batch Tests

`verify:` is the same build-then-test pair the earlier batches use.

The batch's own new coverage is `glyph_test.go` over the `testdata/glyphs/` and `testdata/units/`
fixtures. `golang_test.go`, `toc_test.go`, `answer_test.go` and `toc_integration_test.go` are ported
onto the new symbol in card 29 and must keep passing; `repo_test.go`, `walk_test.go`,
`ignore_test.go`, `headers_test.go`, `text_test.go`, `extension_test.go`, `classify_test.go` and
`treesitter/treesitter_test.go` must pass untouched, which is what proves this batch changed symbol
identity and nothing about the walk's structure.

Note that `testdata/units/test data/pkg/spaced.go` sits under a directory whose name contains a
space on purpose — that is the fixture, and it must survive as a committed path.
