# Batch: expand

```yaml
task: "resolve + expand (T4)"
batch: "expand"
number: 4
cards: 5
verify: CGO_ENABLED=1 go test ./internal/engine/
depends-on: [1, 2, 3]
```

## Batch Scope

This batch delivers the `expand` verb in a new `internal/engine/expand.go` and its tests in a new
`internal/engine/expand_test.go`: the typed `*NotATypeError`, the full disposition of a match set,
the head read from the type symbol's own head span, and the members collected by owner chain across
the unit's files. It is a new file rather than an addition to `internal/engine/resolve.go` because
these are two verbs with two answers and no shared helper beyond the memo and the decision function
batch 3 already lifted; it is not a new package, because the engine is one package with files per
concern.

It depends on batch 3 for three things it reuses rather than reimplements — `unitMemo` with
`newUnitMemo`, `symbolsOf` and `dirsOf`; `statusForMatches`, so the two verbs can never disagree
about what a glyph resolves to; and `matchesFor`, the owner-chain-and-name filter — and on batch 2
for the two fixture packages its tests read.

Batch-local decision: `Expand`'s failures split by what they say. A glyph naming a non-type is a
caller-actionable answer about the caller's own glyph and gets a struct carrying the kind. A type
symbol arriving with no head span says the engine is internally inconsistent, and gets a plain
`fmt.Errorf` — there is no status for that, nothing a caller can do differently, and a struct would
invite branching on a condition that must never occur.

## Cards

### Card 14: NotATypeError

- **Context:**
  - `docs/glyph.md`
  - `docs/rewrite-plan.md`
  - `internal/engine/answer.go`
  - `internal/engine/errors.go`
  - `internal/engine/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/expand.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/engine/expand.go` with the package clause, a file header comment
  in this package's register describing the file as the home of the `expand` verb and its one typed
  failure, and the error type.

  Declare `type NotATypeError struct { ID string; Kind Kind }` and
  `func (e *NotATypeError) Error() string` producing the message
  `engine: expand <id>: not a type, kind <kind>` with the struct's two fields substituted.

  Document why it is a struct rather than a bare sentinel: docs/rewrite-plan.md's `expand` rule —
  the glyph must name a type, and on any other kind the answer names the kind — is what requires the
  kind to be carried, and a caller mapping engine failures to a status word and an exit code needs it
  without parsing a message. That is the same argument that split `ErrTargetOutsideRepo` from
  `ErrTargetNotFound` in `internal/engine/repo.go`. Cite docs/rewrite-plan.md, not docs/glyph.md,
  for that rule: glyph.md §5 is the status vocabulary and says nothing about `expand`'s kind gate.
  Document that it is returned as the verb's `error` and never carried inside `ExpandAnswer`: an
  `ok`-plus-kind pair inside the payload would duplicate the envelope a later task owns, inside the
  data. Document that it is never returned under a unit collision, because the glyph does not
  unambiguously name anything there and naming a kind would be a claim the answer cannot support.

  The file must compile and vet cleanly on its own; `Expand` itself is card 15.
- **Commit:** `feat(engine): add expand.go and the typed NotATypeError`

### Card 15: Repo.Expand

- **Context:**
  - `docs/glyph.md`
  - `docs/rewrite-plan.md`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/expand.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `func (r *Repo) Expand(target string) (ExpandAnswer, error)` and its
  unexported worker `func (r *Repo) expand(target string, m *unitMemo) (ExpandAnswer, error)` to
  `internal/engine/expand.go`. `Expand` builds one `unitMemo` with `newUnitMemo`, returning that
  error if it fails, and delegates — the same shell-plus-worker shape `Resolve` has, for the same
  reason: the memo is a local that dies with the call, and a test can construct one and pass it in.

  `expand` parses with `glyph.Parse(glyph.Go, target)` and, on failure, returns the zero
  `ExpandAnswer` and the parse error wrapped as `engine: expand <target>: <err>` with `%w`, so
  `errors.As` still reaches the `*glyph.ParseError`. This includes a target with no `#`, which the
  grammar already rejects as its no-separator reason: `expand` writes no separator check of its own,
  because every alphabet question is one `glyph.Parse` call and a hand-rolled check would return a
  different error type for one of the grammar's rejection reasons than for all the others. Unlike
  `resolve`, `expand` answers one target, so there is no other answer to protect and no reason to move
  the failure into the payload.

  It then sets `ID` to the parsed glyph's `String()`, reads `m.dirsOf(g.Unit)` for the directory list
  and the collision flag, reads `m.symbolsOf(g.Unit)` — an error from which is returned as the call's
  error — and filters with `matchesFor`.

  It then calls `statusForMatches(g, matches, dirs.collision)`. That call is not optional and no row
  of it is restated here: it is the one place a match set becomes a status, and `expand` calling it is
  what stops the two verbs disagreeing about what a glyph resolves to. `expand` adds a kind gate
  *between* that function's zero-and-collision rows and its remaining ones, and maps the four returned
  values onto its own answer like this, in exactly this order:

  1. `StatusNotFound`: `Status` is `StatusNotFound`, `Unit` is `StatusFound` when the memoised
     directory list is non-empty and `StatusNotFound` when it is empty, and the answer carries no head,
     members or candidates. This is the shared function's own first row, evaluated before its collision
     row, which is why a collision with no match in either directory answers `not_found` with
     `unit: found` here as it does in `resolve` — `ambiguous` with an empty `Candidates` would have
     nothing to be ambiguous between and would hide the `unit: found` a card creating the first
     declaration of an existing unit needs.
  2. `StatusAmbiguous` **and** the collision flag is set: `Status` is `StatusAmbiguous`, `Candidates`
     is the match set, and there is no head and no members. Checked before the kind gate below, so a
     single type match under a collision answers `ambiguous` here exactly as it does in `resolve`, and
     `*NotATypeError` is never returned under a collision — the glyph does not unambiguously name
     anything, so naming a kind would be a claim the answer cannot support. Testing the flag rather
     than the status alone is what separates this row from row 4: the shared function returns
     `StatusAmbiguous` for a collision and for a plain multi-match alike, and only the collision case
     short-circuits the kind gate. `expand` reads the flag from `dirsOf` itself and never infers it
     from the match set, which cannot show it: `symbolsOfUnit` returns the union either way.
  3. The kind gate, applied to whatever the shared function returned other than the two rows above —
     `StatusFound`, `StatusMultipart`, or a non-collision `StatusAmbiguous`. When no match in the set
     is `KindType`, whatever the count, return the zero `ExpandAnswer` and a `*NotATypeError` carrying
     the glyph's string form and the kind of the first match in file-then-line order. Keying the gate
     on no match being a type rather than on the match count is what gives the several-declaration
     `init` glyph a defined answer: the shared function calls that set `StatusMultipart`, a status
     `ExpandAnswer` does not admit, and the gate catches it because `init`'s kind is function. It also
     catches a single non-type match, which the shared function calls `StatusFound`.
  4. The set holds at least one type and the shared function did not return `StatusFound`: `Status` is
     `StatusAmbiguous` and `Candidates` is the match set. This covers both several type declarations
     under one glyph — docs/glyph.md §5's build-tag duplicates, which are the only way a Go type
     multiplies, since docs/rewrite-plan.md says a Go type never splits and only `init` does — and a
     mixed set naming a type and something else, where choosing between them would be a silent pick.
  5. `StatusFound` and the single match is a type: `Status` is `StatusFound`, with the head and
     members below.

  The head is the matched type's own `Symbol`, copied by value, with two substitutions and nothing
  else: `Start` becomes the symbol's `HeadStart` and `End` becomes its `HeadEnd`. Every other field —
  the id, kind, file, sigend, signature and doc — is the symbol's own, because one symbol entry is
  what all three verbs return for a symbol and `expand` emits no shape of its own. Assign its address
  to `Head`. If the matched `KindType` symbol has `HeadStart == 0`, return the zero `ExpandAnswer` and
  a plain `fmt.Errorf` naming the glyph's string form and stating that a type symbol carries no head
  span: that is an invariant violation in the walk, and a silent fallback to `Start` and `End` would
  hide it behind an answer that happens to be right for Go.

  The members are every symbol in the unit's symbol slice — not the match set — with
  `len(s.Glyph.Owner) > 0 && s.Glyph.Owner[0] == g.Name`. The type symbol itself has no owner and is
  excluded by that filter, so it is never both head and member. The slice `symbolsOf` returns is
  already ordered by file then start line and the filter preserves it, so no second sort is needed and
  none is written. Leave `Members` nil when the filter selects nothing, so `omitempty` drops the key:
  a type with no members is `found` with a head and nothing else, not an error and not a `not_found` —
  the type exists and consists of its head.

  Document, on the members filter, that matching on the first owner rather than the whole chain is the
  general form: in Go the chain is at most one element because the grammar rejects a deeper member
  outright, so the two are the same today and the general form is what a nested-type language needs.
  Document that interface methods are members by this same rule with no special case, since the walk
  gives a method element the interface's own type name as its owner, and that they sort into place
  inside the head range by file and line. Document that only the glyph's own unit is searched: the
  external test unit is a different unit and cannot declare methods on this unit's types. Document, on
  the head, why the span is read from the head fields rather than re-derived: for Go the two pairs are
  identical so nothing observable changes today, and for the first language whose head is a strict
  subset of its declaration `expand` needs no edit. Document that docs/rewrite-plan.md's "the class
  span minus its member spans" — its phrase, in the three-queries section, not docs/glyph.md's —
  describes what a reader ends up reading, not arithmetic this verb performs —
  for a Go struct the subtraction is empty, and for an interface the answer already carries every
  member's start and end, so a consumer wanting only the non-member lines has what it needs and the
  engine emits one contiguous head entry rather than a discontiguous span type the closed symbol shape
  does not have.
- **Commit:** `feat(engine): add Repo.Expand with its head, members and disposition table`

### Card 16: expand's found answers

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
  - `internal/engine/resolve_test.go`
  - `internal/engine/testdata/glyphs/decls.go`
  - `internal/engine/testdata/glyphs/iface.go`
  - `internal/engine/testdata/methods/aardvark.go`
  - `internal/engine/testdata/methods/widget.go`
  - `internal/engine/toc_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/expand_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/engine/expand_test.go` with a file header comment in this
  package's register, and the three tests that assert a successful expansion. Open the repository with
  the existing `openQuarryRoot` helper declared in `internal/engine/resolve_test.go` — it is in the
  same package, so no new helper is needed and none is written.

  `TestExpand_Struct` expands the `Widget` type of the methods fixture package. Assert `Status` equal
  to `StatusFound`; `Unit` empty; `Candidates` absent; `Head` non-nil with `Kind` equal to `KindType`,
  `ID` equal to the answer's own `ID`, `File` naming the file that declares the type, and `Start` and
  `End` equal to that symbol's head span as the walk reports it — read the expected pair from a
  `SpansOf` lookup of the same glyph in the test itself rather than hard-coding line numbers, so the
  assertion survives an edit to the fixture's comments. Assert `Members` holds all three methods, from
  both files, ordered by file then start line, with the sibling file's two methods first, and that no
  member's `ID` equals the head's.

  `TestExpand_Interface` expands the named interface of the glyphs fixture package. Assert `Status`
  equal to `StatusFound`, that `Members` are exactly that interface's own two methods and not the
  embedded interface's method, that every member's `Start` and `End` lie within the head's `Start` and
  `End` inclusive, and that the members are ordered by start line. The containment assertion is the
  shape claim: the head is one contiguous span, and a consumer that wants only its non-member lines
  subtracts the member spans the same answer carries.

  `TestExpand_TypeWithoutMembers` expands the defined scalar type of the glyphs fixture package.
  Assert `Status` equal to `StatusFound`, `Head` non-nil, and `Members` nil — a type that consists of
  its head is a found answer, not a miss.
- **Commit:** `test(engine): assert expand's head and member collection`

### Card 17: expand's failure rows

- **Context:**
  - `glyph/errors.go`
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
  - `internal/engine/resolve_test.go`
  - `internal/engine/testdata/glyphs/decls.go`
  - `internal/engine/testdata/glyphs/inits.go`
  - `internal/engine/testdata/tags/linux.go`
  - `internal/engine/testdata/tags/other.go`
- **Edits:**
  - `internal/engine/expand_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add one test per remaining row of the disposition table to
  `internal/engine/expand_test.go`.

  `TestExpand_NotAType` asserts, with `errors.As`, that a package-level function glyph and a
  package-level const glyph each return a `*NotATypeError` whose `Kind` is the kind the walk reports,
  and that the returned answer is the zero value. It then asserts the case a match-count gate would
  miss: the bare `init` glyph of the glyphs fixture package, which matches three declarations, returns
  a `*NotATypeError` with a function kind — not a multipart answer, which the type does not admit, and
  not an ambiguous one.

  `TestExpand_AmbiguousBuildTags` asserts that the duplicated type glyph of the tags fixture package
  answers `StatusAmbiguous` with both declarations in `Candidates` and no head and no members, and
  that the mixed glyph — a type in one file and a function in the other — answers `StatusAmbiguous`
  too rather than a `*NotATypeError`, because the set holds a type and choosing between the two would
  be a silent pick.

  `TestExpand_MalformedTarget` asserts that a target the grammar rejects and a target with no `#` each
  return a non-nil error and the zero answer, that `errors.As` reaches a `*glyph.ParseError` in both
  cases, and that the no-separator case carries the grammar's own no-separator reason — proof that
  `expand` writes no separator check of its own.

  `TestExpand_NotFound` asserts that a name that does not exist inside an existing unit answers
  `StatusNotFound` with `Unit` equal to `StatusFound` and a nil error, and that a glyph whose unit
  directory does not exist answers `StatusNotFound` with `Unit` equal to `StatusNotFound` and a nil
  error. A miss is a legitimate answer with a status, never a failure.
- **Commit:** `test(engine): assert expand's not-a-type, ambiguous and miss rows`

### Card 18: expand under a collision, and the two verbs agreeing

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
  - `internal/engine/resolve.go`
  - `internal/engine/resolve_test.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/expand_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestExpand_Collision` to `internal/engine/expand_test.go`, over a run-time
  tree built with the existing `openScratchRepo` helper and named distinctly from every other scratch
  tree in this package. Build it with the same shape batch 3's collision test uses — a directory whose
  external test file and a literally-named sibling directory both belong to one unit — declaring in
  the two files that unit reaches a type of the same name in both, and a second type in the
  literally-named directory's file only.

  Assert three things. The doubly-declared type glyph answers `StatusAmbiguous` with both declarations
  in `Candidates`, `Head` nil and `Members` nil. The singly-declared type glyph — one match, and a
  type — answers `StatusAmbiguous` too rather than `StatusFound`, which is the row that keeps the two
  verbs from disagreeing. And `Candidates` come back ordered by file then start line even though the
  literally-named directory's symbols were appended first.

  Call `Resolve` on the same two glyphs in this same test and assert both verbs report the same status
  and the same candidate ids in the same order, so the agreement is shown rather than assumed. Assert
  in the same test that a name declared in neither directory answers `StatusNotFound` with `Unit`
  equal to `StatusFound`, the zero-match row that is evaluated before the collision row.
- **Commit:** `test(engine): assert expand under a unit collision and its agreement with resolve`

## Batch Tests

`verify:` runs the whole `internal/engine` package suite, for the same reason batch 3 does: this
batch's files live in that package, the suite runs in well under a second, and every existing test in
it is a must-keep-passing gate. The new coverage is one test per row of the disposition table — the
zero-match row and its two `unit` values, the collision row on both a single and a doubly-declared
type, the not-a-type row for a function, a const and the several-declaration `init` glyph that a
match-count gate would miss, the ambiguous row for both a type-only and a mixed-kind match set, and
the found row over a struct with members across two files, an interface whose members lie inside its
head span, and a type with no members at all — plus the malformed-target row and the cross-verb
agreement assertion. The one property no committed fixture proves convincingly, members drawn from
more than one file of a real repository, is batch 5's Loomyard spot check.
