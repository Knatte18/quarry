# Batch: resolve

```yaml
task: "resolve + expand (T4)"
batch: "resolve"
number: 3
cards: 9
verify: CGO_ENABLED=1 go test ./internal/engine/
depends-on: [1, 2]
```

## Batch Scope

This batch delivers the whole `resolve` verb: the per-call unit memo that makes "each unit is parsed
once" true and observable, the pure status decision function, the glyph branch, the path branch, the
exported `Repo.Resolve` that wires them together, and every test docs/glyph.md §5's status
vocabulary needs. It all lives in `internal/engine/resolve.go` and `internal/engine/resolve_test.go`,
growing the two files T3 wrote for the span lookup rather than adding a package — the engine is one
package with files per concern, never a package per verb.

It is one batch because every card after the first reads the memo or the decision function the first
two add, and because splitting the implementation from its tests would leave a batch with nothing to
verify. Batch 4 consumes three things from here: `unitMemo` with `newUnitMemo`, `symbolsOf` and
`dirsOf`; the `statusForMatches` decision function; and `matchesFor`, the owner-chain-and-name filter.

Batch-local decision, differing from nothing in the overview but worth stating once: cards 6 through
9 each add a function that compiles on its own. Go permits an unused unexported function, so the
decision function and the two branch helpers land before the exported entry point that calls them,
and every card in this batch leaves the package building and the existing suite green.

## Cards

### Card 5: the per-call unit memo

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/ignore.go`
  - `internal/engine/repo.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the per-call memo to `internal/engine/resolve.go`, placed after `unitDirs`
  and `dirExists` and before `dirChainBelowRoot`, so the memo sits beside the two functions it
  wraps.

  Declare `type unitDirsResult struct { dirs []string; collision bool }` — one memoised `unitDirs`
  answer, carrying no error because `unitDirs` returns none: it is two `dirExists` calls, each an
  `os.Lstat` whose failure is reported as "not a directory".

  Declare `type unitMemo struct { repo *Repo; ig *ignoreSet; symbols map[string][]Symbol; dirs
  map[string]unitDirsResult; parses int }`. Document `parses` as the count of `symbolsOfUnit` calls
  made, read by tests and never by production code — it is the seam that makes the grouping
  guarantee observable, since a map's entry count is true by construction and wall-clock is the very
  thing the guarantee's test must be independent of.

  Declare `func newUnitMemo(r *Repo) (*unitMemo, error)`. It builds the ignore set exactly as
  `SpansOf` does — `newIgnoreSet(r.root)` followed by one `extend(".")`, wrapping an extend failure
  with the same message shape `SpansOf` uses — and returns a memo with both maps made non-nil. That
  one set carrying the root's own patterns and nothing below them is precisely what `symbolsOfUnit`'s
  doc comment requires of its caller; `symbolsOfUnit` extends and trims per directory itself.

  Declare `func (m *unitMemo) symbolsOf(unit string) ([]Symbol, error)`. On a hit in `m.symbols` it
  returns the memoised slice and a nil error. On a miss it increments `m.parses`, calls
  `m.repo.symbolsOfUnit(unit, m.ig)`, returns the error unmemoised on failure, and otherwise stores
  and returns the result. Increment `parses` before the call, not after a successful return, so the
  counter means what its doc comment says: calls made.

  Declare `func (m *unitMemo) dirsOf(unit string) unitDirsResult`, memoising `m.repo.unitDirs(unit)`
  with no error return for the reason recorded on `unitDirsResult`.

  Document on `unitMemo` that it is a local of the exported entry point and dies with it: nothing is
  stored on `Repo`, because a memo that cannot outlive the call it was built in cannot go stale,
  which is what keeps this inside the engine's no-cache rule while still turning twenty lookups over
  five units into five parses instead of twenty. Document that `symbolsOf` is the only site in either
  verb that calls `symbolsOfUnit`, and that `SpansOf` is deliberately not used by either verb: it is
  the per-glyph wrapper, and calling it per target would re-parse each unit once per target.
- **Commit:** `feat(engine): add the per-call unit memo behind resolve and expand`

### Card 6: the target split and the status decision function

- **Context:**
  - `docs/glyph.md`
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add two pure functions to `internal/engine/resolve.go`, after the memo and before
  `dirChainBelowRoot`. Neither reads the filesystem and neither is called yet; cards 7 through 9 wire
  them in.

  `func isGlyphTarget(target string) bool` reports whether target is a glyph rather than a path, by
  `strings.Contains(target, "#")` and nothing else. A target containing `#` is a glyph and is handed
  to `glyph.Parse`, which decides whether it is a well-formed one; everything else is a
  repository-relative path. The verb never guesses which was meant, and the split deliberately does
  not pre-empt any of the grammar's own rejections — `#x`, with an empty unit, is a glyph target that
  `glyph.Parse` then rejects, not a path.

  `func statusForMatches(g glyph.Glyph, matches []Symbol, collision bool) Status` is the whole
  decision the verb turns on, expressed once so `resolve` and `expand` can never disagree about what
  a glyph resolves to. Its rows, in this exact order:

  1. `len(matches) == 0` returns `StatusNotFound`. Tested first, so a unit collision with no match in
     either directory is `not_found` rather than an `ambiguous` with nothing to be ambiguous
     between — and that `not_found` carries `unit: found`, which is what a plan card creating the
     first declaration of an existing unit needs.
  2. `collision` returns `StatusAmbiguous`. Tested before every row below, so a single match under a
     collision is ambiguous rather than found: a `found` whose glyph string names two different units
     is exactly the failure the literal-first unit lookup exists to prevent. Several `init` under a
     collision is ambiguous too, not multipart — the glyph does not unambiguously name one unit, and
     multipart would assert that it does.
  3. `len(matches) == 1` returns `StatusFound`.
  4. `len(g.Owner) == 0 && g.Name == "init"` returns `StatusMultipart`. This is the only Go glyph
     that names one symbol the language lets be declared in several places, and the discriminator is
     the glyph's own name rather than any property of the declarations — decidable without reading
     build tags, which the engine has no business interpreting. A build-tagged pair of `init`
     functions is still `init` and still multipart.
  5. Otherwise `StatusAmbiguous`.

  Document why the function does not evaluate build constraints: doing so would make the answer
  depend on a `GOOS` and `GOARCH` the engine does not know and the caller did not state, and
  reporting both candidates with their files is the honest answer where guessing one is a silent
  pick. Document that the caller places the matches — `Symbols` for `found` and `multipart`,
  `Candidates` for `ambiguous`, neither for `not_found` — so the status and the placement have one
  source.
- **Commit:** `feat(engine): add the target split and the resolve status decision function`

### Card 7: the glyph branch

- **Context:**
  - `docs/glyph.md`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add two functions to `internal/engine/resolve.go`, after `statusForMatches`.

  `func matchesFor(symbols []Symbol, g glyph.Glyph) []Symbol` returns every symbol whose owner chain
  and name equal the glyph's, reusing the existing `sameOwner` helper, preserving the input order and
  therefore the file-then-line order `symbolsOfUnit` established. It is the filter `SpansOf` performs
  inline today, written once here so the two new verbs share one copy.

  `SpansOf` keeps its own inline copy of that filter: do not change `SpansOf` to call `matchesFor`,
  and change no line of its code — card 9 re-tenses one clause of its doc comment and that is the only
  edit it takes in this plan. That leaves two implementations of one filter in this file, and
  the duplication is deliberate rather than overlooked — record it in `matchesFor`'s own doc comment,
  naming `SpansOf`'s inline loop as the second copy and stating both halves of the reason. First, this
  task's scope permits exactly two edits to code T3 wrote for its own purposes, both named and both
  elsewhere; a third, however behaviour-preserving, is a scope change and not this plan's to make.
  Second, `SpansOf` is what T3's round trip is written against, so a refactor of it would put a test
  this task must keep passing untouched onto code this task rewrote. Whether the two collapse into one
  is a question for whichever later task next edits `SpansOf` for its own reasons.

  `func (r *Repo) resolveGlyphTarget(target string, m *unitMemo) (ResolveResult, error)` answers one
  glyph target:

  Parse with `glyph.Parse(glyph.Go, target)`. The alphabet is `glyph.Go`, hardcoded; state in a
  comment that a multi-alphabet dispatch is where a second language enters, and that nothing about
  the answer's key set depends on it. On a parse failure, extract the `*glyph.ParseError` with
  `errors.As` and return a `ResolveResult` whose `Target` is the argument, whose `Error` is the
  error's `Error()` text, whose `Reason` is `string(parseErr.Reason)`, and whose `Status` is left
  empty — with a nil error, so the call continues and the other targets are still answered. The four
  statuses are resolution outcomes and a string that is not a glyph was never searched for; answering
  `not_found` would tell a validator the name is free when the truth is that it is unspellable.
  Should `errors.As` ever fail — `glyph.Parse` returns nothing else, so it cannot — leave `Reason`
  empty rather than panicking on a nil pointer.

  Otherwise set `Target` to the argument and `ID` to the parsed glyph's `String()`. Read
  `m.dirsOf(g.Unit)` for the collision flag and the directory list, then `m.symbolsOf(g.Unit)`. An
  error from `symbolsOf` is returned as this function's error and fails the whole call: an engine
  read failure is not an answer about a glyph, and a unit that failed to read would otherwise answer
  `not_found` for every glyph in it, which a done-check reads as success. A unit whose directory does
  not exist is not that case and never reaches it — `unitDirs` returns an empty slice and
  `symbolsOfUnit` over zero directories returns an empty slice and a nil error.

  Filter with `matchesFor`, switch on `statusForMatches(g, matches, res.collision)`, and populate:
  `StatusFound` and `StatusMultipart` set `Symbols` to the matches; `StatusAmbiguous` sets
  `Candidates` to the matches; `StatusNotFound` sets neither and sets `Unit` to `StatusFound` when
  the memoised directory list is non-empty and to `StatusNotFound` when it is empty. `Unit` is set on
  `not_found` and on nothing else — docs/glyph.md §5 attaches it to the miss, and a `found` that also
  said so would be clutter.

  Document, where `Unit` is derived, that `unit: found` is directory existence and nothing more, so
  it is `unitDirs`'s answer restated nowhere: deriving it any other way would be a second
  unit-to-directory implementation in one package, drifting on the corner case the literal-first rule
  was written for. Record the consequence — a `_test` unit whose stripped directory exists but which
  holds no external test package at all reports `unit: found`, because the directory is there, and a
  card creating that package's first file is exactly the case that wants it. Record there too the
  third of this task's contract gaps, closed by nobody: `dirExists` uses `os.Lstat`, which refuses a
  symlink only in the path's final component, so a unit reached through an intermediate symlinked
  directory resolves here while the walk never descends that directory and so never lists those
  declarations. The behaviour is inherited unchanged and deliberately not narrowed — changing
  `dirExists` would be a change to the walk's inverse, and whether a unit may be reached through a
  link at all is a statement docs/glyph.md does not make.

  Do not restate the first gap here. `unitDirs`'s own doc comment already records it, almost verbatim —
  that docs/glyph.md §2 gives the external test unit its pseudo-path without saying what happens when a
  real directory spells the same string, and that checking both and reporting the collision is quarry's
  chosen behaviour rather than the contract's. Card 9 re-tenses that comment and keeps that statement;
  it is the surviving copy. At the collision read, write one sentence pointing to it and adding only
  what is new here: that this verb promotes the reported collision to `ambiguous`, which is the half of
  the gap `unitDirs` could not state because the status type did not exist when it was written.
- **Commit:** `feat(engine): add resolve's glyph branch and its status disposition`

### Card 8: the path branch

- **Context:**
  - `docs/glyph.md`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/toc.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `func (r *Repo) resolvePathTarget(target string) (ResolveResult, error)` to
  `internal/engine/resolve.go`, after `resolveGlyphTarget`.

  It answers a target with no `#` as a repository-relative path by calling `r.TOC` with
  `TOCOptions{Depth: 0, Symbols: &symbolsOff}` where `symbolsOff` is a local `false`. Reusing the
  directory answer rather than restating the rule is what makes the explicitly-named-gitignored-target
  rule, the never-follow-a-symlink rule and the empty-string-and-dot-mean-the-root rule hold here for
  free and keeps them from drifting.

  Disposition, in this order: a nil error sets `Status` to `StatusFound` and `Dir` to the address of
  the returned `DirAnswer`; `errors.Is(err, ErrTargetNotFound)` sets `Status` to `StatusNotFound`
  with `Dir` absent; `errors.Is(err, ErrTargetOutsideRepo)` returns an entry with `Status` empty and
  `Error` set to the error's text and `Reason` empty, the second and last member of the per-entry
  error domain; any other error is returned as this function's error and fails the whole call. `TOC`'s
  two sentinels are the only errors this verb converts into an answer. `Target` is always set to the
  argument; `ID` and `Unit` are never set on a path result, because a path has no glyph and belongs
  to no unit.

  Document why symbols are switched off explicitly rather than left to the per-target default, which
  would turn them on for a file target: this verb answers where a thing is and whether it exists, and
  what is inside it is the table-of-contents question. A plan card whose target is a Markdown page has
  no symbols to want, and paying a tree-sitter parse per Go path target inside a call measured against
  a 150 ms budget would be a cost with no consumer. Document that a file target's answer is its
  enclosing directory's answer holding exactly that one file entry — the shape `TOC` already produces
  — so the caller can read the package and language a bare file entry would not carry.
- **Commit:** `feat(engine): add resolve's path branch over the directory answer`

### Card 9: Repo.Resolve

- **Context:**
  - `docs/glyph.md`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the exported entry point and its unexported worker to
  `internal/engine/resolve.go`, after `resolvePathTarget` and before `SpansOf`, and update the file's
  own header comment.

  `func (r *Repo) Resolve(targets []string) ([]ResolveResult, error)` is a thin shell: it builds one
  `unitMemo` with `newUnitMemo`, returning that error if it fails, and delegates to
  `func (r *Repo) resolve(targets []string, m *unitMemo) ([]ResolveResult, error)`. The unexported
  form takes the memo so a test can construct one, pass it in, and read `parses` afterwards; the
  exported form never exposes it.

  `resolve` allocates a result slice of exactly `len(targets)`, and for each target in order calls
  `resolveGlyphTarget` when `isGlyphTarget` reports true and `resolvePathTarget` otherwise. Any error
  from either fails the whole call: return a nil slice and that error, unwrapped further only if it
  does not already name the target or unit it was reading. A `nil` targets slice yields an empty,
  non-nil result slice and a nil error.

  Document the contract on `Resolve`: the returned slice has exactly `len(targets)` elements and
  element `i` answers `targets[i]`; a target repeated twice is answered twice. That positional 1:1
  mapping is what a caller resolving every glyph of a draft in one call needs to map each answer back
  to the card that asked for it, and it is the only shape where a duplicate or a malformed target
  cannot silently vanish. Document the error boundary exhaustively: a `ResolveResult` expresses either
  a resolution outcome or a pre-resolution rejection of the target string, never an engine failure;
  every engine error other than the two `TOC` sentinels the path branch converts fails the whole call,
  and losing the other answers is right precisely because an engine failure makes the whole answer
  untrustworthy, unlike a malformed target, which taints only itself. Document that "grouped by unit"
  is an execution property and not the output shape — the answer is flat, and each distinct unit is
  parsed exactly once per call by the memo — with the reason the output is not grouped: a per-unit
  group's natural key is `unit` holding a path, and docs/glyph.md §5 already spells `unit` as a key
  holding `found` or `not_found`. Document the ordering guarantee: results in argument order, and
  within a result `Symbols` and `Candidates` by file then by start line, file comparison being the raw
  repository-relative forward-slash string under `sort.Strings` semantics with no case folding and no
  locale. No caller sorts.

  Correct every comment in `internal/engine/resolve.go` that speaks of `resolve` as a verb that does
  not exist yet. There are five such clauses across four comments, and this card owns all of them, on
  the same ground card 2 owns `internal/engine/answer.go`'s: a comment that describes its own file
  wrongly is worse than the churn of fixing it, and this file is one the task edits as its own work
  rather than under a scope exception. Four of the five are re-tensings with no change of substance;
  the first is not, and is the one to get right. Change no code and no other comment while doing it.

  1. The header comment's sentence saying `Repo.SpansOf` is "the public, per-glyph wrapper the rest of
     this task's later verbs (resolve, expand) are built on". That is not merely mistensed — this plan
     builds neither verb on `SpansOf`. Both go through `symbolsOfUnit` behind the memo, precisely so
     each unit is parsed once per call, and card 5 states that `symbolsOf` is the only site in either
     verb that calls `symbolsOfUnit`. Rewrite the sentence to say what is true: `SpansOf` is the
     single-glyph convenience the round trip is written against, and `Resolve` and `Expand` are built
     on `symbolsOfUnit` through the per-call memo. Leaving the sentence standing would ship a claim the
     plan's own design contradicts.
  2. The header comment's closing clauses, which say the file stops short of a status vocabulary and
     that the collision flag is carried for a later verb to promote. Rewrite them to describe what the
     file now holds — `SpansOf` still returning an empty slice with no status, and `Resolve` above it
     holding the vocabulary and promoting the collision — without touching the file's description of
     `unitDirs`, `symbolsOfUnit` or `symbolsOfDir`.
  3. `symbolsOfUnit`'s doc comment, which says "the later resolve verb needs the same grouping anyway
     for the many glyphs one card can name". Re-tense that clause: the verb exists, in this file, and
     the memo is where the grouping is realised.
  4. `unitDirs`'s doc comment, which says the later `resolve` verb promotes the collision into an
     `ambiguous` status when it builds the status vocabulary, and that the flag lives on an unexported
     return so the task records the fact without inventing a status type that is not its to design.
     Re-tense both clauses: `Resolve` in this same file now reads the flag and promotes it, and the
     status type now exists in `internal/engine/answer.go`. Keep this comment's statement of the
     identifier-contract gap — that docs/glyph.md §2 gives the external test unit its pseudo-path
     without saying what happens when a real directory spells it — exactly as it stands. This is the
     surviving copy of that gap in this file; card 7 points at it rather than repeating it.
  5. `SpansOf`'s doc comment, which contrasts its own empty-slice-no-status result with "the later
     resolve verb's" four statuses. Re-tense that one clause only. Card 7's instruction not to change
     `SpansOf` is about its code and its behaviour, not its doc comment: the function keeps its inline
     filter, its signature and every line of its body.
- **Commit:** `feat(engine): add Repo.Resolve over the glyph and path branches`

### Card 10: decision-function table tests

- **Context:**
  - `glyph/glyph.go`
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add two table-driven tests to `internal/engine/resolve_test.go`. Neither parses a
  file or reads the filesystem: these are the pure decisions the verb turns on, and they are asserted
  directly rather than through a fixture.

  `TestStatusForMatches` tables `statusForMatches` over, at minimum: zero matches with no collision;
  zero matches with a collision; one match with no collision; one match with a collision; three `init`
  matches with no collision; three `init` matches with a collision; two non-`init` package-level
  matches; two matches with a collision; and two matches for an owned name such as a method glyph.
  The two rows that pin the check order are zero-matches-under-collision, which must be `not_found`
  and not `ambiguous`, and several-`init`-under-collision, which must be `ambiguous` and not
  `multipart`. Build the `[]Symbol` values by hand — the function reads only the slice's length — and
  build the `glyph.Glyph` values by hand too, since only `Owner` and `Name` are read.

  `TestIsGlyphTarget` tables `isGlyphTarget` over at least `a/b#C`, `a/b`, `#x`, `a#b#c`, `Makefile`,
  `notes.rst`, the empty string, and `.`, asserting true for every target containing `#` and false for
  every other.
  State in the test's doc comment that `#x` is a glyph target the grammar then rejects, not a path —
  the split does not pre-empt the alphabet's own rules.

  This card also owns `internal/engine/resolve_test.go`'s header comment, on the same ground cards 2
  and 9 own their files': it currently says the file covers `Repo.unitDirs`, `Repo.symbolsOfUnit`'s
  ignore filtering and the public `Repo.SpansOf`, and describes a committed-versus-`.scratch` fixture
  split drawn for those three. Cards 10 through 13 add fourteen test functions for `Resolve`, a second
  collision tree and a permissions fixture, so both halves stop describing the file. Extend the
  enumeration to name `Repo.Resolve` and its status vocabulary alongside the three already there, and
  extend the fixture-split paragraph to cover the new run-time trees. Card 10 owns it rather than 11,
  12 or 13 because it is the first card to touch the file, so no intermediate commit leaves the header
  describing a file it no longer matches.
- **Commit:** `test(engine): table the resolve status decision and the target split`

### Card 11: the four statuses over committed fixtures

- **Context:**
  - `docs/glyph.md`
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `internal/engine/testdata/glyphs/inits.go`
  - `internal/engine/testdata/methods/aardvark.go`
  - `internal/engine/testdata/methods/widget.go`
  - `internal/engine/testdata/tags/linux.go`
  - `internal/engine/testdata/tags/other.go`
  - `internal/engine/testdata/tree/pkg/alpha.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add fixture-driven status tests to `internal/engine/resolve_test.go`, all opened
  through the existing `openQuarryRoot` helper so the committed fixtures are addressed by their real
  repository-relative paths.

  `TestResolve_Found` resolves a package-level function in the tree fixture package and a method on
  `Widget` in the methods fixture package. Each answers exactly one result with `Status` equal to
  `StatusFound`, exactly one entry in `Symbols`, `Candidates` absent, `Unit` empty, and `ID` equal to
  `Target` — for Go the two are always byte-identical on a non-error result, because the Go alphabet
  normalises nothing, so this assertion is parity with `Symbol.ID` and must not chase a normalisation
  case the language cannot produce. Assert the symbol's `File`, `Start`, `SigEnd`, `End` and
  `Signature` against the values the walk reports, and assert `Doc` carries the declaration's own doc
  comment.

  `TestResolve_Multipart` resolves the bare `init` glyph of the glyphs fixture package: one result,
  `Status` equal to `StatusMultipart`, three entries in `Symbols` in file-then-line order,
  `Candidates` absent.

  `TestResolve_AmbiguousBuildTags` resolves the duplicated function glyph of the tags fixture package:
  one result, `Status` equal to `StatusAmbiguous`, both declarations in `Candidates` with their two
  distinct `File` values, and `Symbols` absent. Assert the same for the duplicated type glyph.

  `TestResolve_NotFoundBothWays` resolves a name that does not exist inside the tree fixture package,
  asserting `Status` equal to `StatusNotFound` with `Unit` equal to `StatusFound`; and a glyph whose
  unit directory does not exist at all, asserting `Status` equal to `StatusNotFound` with `Unit` equal
  to `StatusNotFound`. Marshal one result of each kind with `encoding/json` and assert on the emitted
  JSON, not only on the struct: the `unit` key must be spelled with the values `found` and
  `not_found`; `target`, `id` and `status` must all be present, `id` carrying the glyph's own string
  form; and `symbols`, `candidates`, `dir`, `error` and `reason` must all be absent. `id` is present on
  every successfully parsed glyph target, a miss included — card 7 sets it the moment the parse
  succeeds, and its `omitempty` exists for the path branch, which has no glyph, not for the miss.

  Marshal two more results in this same test, so `ResolveResult`'s keys are observed present and not
  only absent: a `found` result over the same fixture `TestResolve_Found` uses, asserting `symbols`
  present under exactly that spelling with one entry and `candidates` absent; and an `ambiguous` result
  over the tags fixture, asserting `candidates` present with two entries and `symbols` absent. A key
  observed only in its absent state is a key whose spelling nothing checks — `go vet` reads tag syntax
  and never a tag's name, so `symbol` for `symbols` would pass every struct-level assertion in this
  plan. That leaves `dir`, `error` and `reason` unobserved in their present state; card 13 marshals
  those, and between the two cards every one of `ResolveResult`'s nine keys is seen both ways.

  Write each test's doc comment in this file's established register, naming what it asserts and why
  that assertion is the one that matters.
- **Commit:** `test(engine): assert every resolve status over the committed fixtures`

### Card 12: collision, grouping and the error boundary

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/ignore.go`
  - `internal/engine/repo.go`
  - `internal/engine/resolve.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the tests that need a run-time tree, built with the existing
  `openScratchRepo` helper under `.scratch/`. `t.TempDir` is banned outright — it writes to the system
  temporary directory — and a committed tree cannot express the collision, because the existing
  `TestSpansOf_LiteralFirst` asserts the committed `foo_test` fixture has no sibling, and committing
  one would break it.

  Build one shared collision tree, named distinctly from the existing collision test's tree so the two
  cannot collide on disk: a directory `foo` holding a file with package clause `foo` and a second file
  with clause `foo_test`, plus a literal directory `foo_test` holding a file with clause `foo_test`.
  The unit `foo_test` then resolves to both directories. Declare in the two files that unit reaches —
  the external test file under `foo` and the file under `foo_test` — a function of the same name, a
  type of the same name, and, in the `foo_test` directory's file only, a second type declared nowhere
  else. Their repository-relative paths sort with the `foo` directory's file first even though
  `unitDirs` returns the literal `foo_test` directory first, which is what makes the closing sort
  load-bearing.

  `TestResolve_AmbiguousCollision` asserts three dispositions over that tree: the doubly-declared
  function glyph answers `StatusAmbiguous` with both declarations in `Candidates`; the singly-declared
  type glyph, matching exactly once, still answers `StatusAmbiguous` rather than `StatusFound`,
  because a found whose glyph string names two different units is the failure literal-first exists to
  prevent; and a name declared in neither directory answers `StatusNotFound` with `Unit` equal to
  `StatusFound`, since both directories exist and only the member is missing.

  `TestResolve_CandidatesOrdered` asserts, over the same tree, that the doubly-declared glyph's
  `Candidates` come back ordered by file then start line — the `foo` directory's file first — even
  though `symbolsOfUnit` appended the literal `foo_test` directory's symbols first. This is the one
  place the engine's sort is genuinely load-bearing: a fixture built to perturb a single directory's
  read order would assert nothing, because `os.ReadDir` is documented to return entries already sorted
  by filename.

  `TestResolve_ParsesEachUnitOnce` constructs a `unitMemo` with `newUnitMemo`, calls
  `r.resolve(targets, m)` directly with at least eight targets spread over three distinct units — at
  least two glyphs per unit, and one unit named four times — and asserts `m.parses` equals the number
  of distinct units, not the number of targets. Asserting the memo map's length instead would be true
  by construction and prove nothing. Use the committed fixture packages for the units so the test
  needs no tree of its own.

  `TestResolve_UnitDirectoryMissingIsNotAnError` asserts a glyph whose unit directory does not exist
  returns a result rather than failing the call: a non-nil slice of length one, `StatusNotFound` with
  `Unit` equal to `StatusNotFound`, and a nil call error. This is the assertion that stops a future
  change turning a missing directory into a call failure and making the create-a-new-unit case
  unanswerable.

  `TestResolve_ReadFailureFailsTheCall` builds a small tree, makes the unit's directory unreadable
  with `os.Chmod` mid-test, and asserts `Resolve` returns a nil slice and a non-nil error that is
  neither `ErrTargetNotFound` nor `ErrTargetOutsideRepo` under `errors.Is`, rather than a `not_found`
  entry. Restore the mode in a `t.Cleanup` so the tree can be removed. Skip with the reason, rather
  than weakening the assertion, when the host cannot revoke read permission — running as root, or a
  filesystem without the bit — detected by re-reading the directory after the chmod and finding it
  still readable.
- **Commit:** `test(engine): assert the collision, grouping and error-boundary rules`

### Card 13: argument order, path targets and error entries

- **Context:**
  - `glyph/errors.go`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/resolve.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/testdata/tree/README.md`
  - `internal/engine/toc.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the remaining `resolve` tests to `internal/engine/resolve_test.go`.

  `TestResolve_ArgumentOrderAndArity` calls `Resolve` once with a mixed slice — a resolvable glyph, a
  path, a malformed glyph, and the first glyph repeated — and asserts exactly `len(targets)` results,
  each result's `Target` equal to its own argument at the same index, and the repeated target answered
  twice with equal results. Assert a nil targets slice returns an empty, non-nil slice and a nil
  error.

  `TestResolve_PathTargets` asserts, over the committed tree fixture: an existing file answers
  `StatusFound` with `Dir` non-nil carrying the enclosing directory's `Dir`, `Package` and `Language`
  and exactly one entry in `Files`, whose `Symbols` field is nil because symbols are switched off; an
  existing directory answers `StatusFound` with its own `Files` populated and every entry's `Symbols`
  nil; a path that does not exist answers `StatusNotFound` with `Dir` absent and `Unit` empty; and an
  absolute path and a `..`-escaping path each answer with `Status` empty, `Error` non-empty and
  `Reason` empty. Over a run-time tree built with `openScratchRepo` holding a `.gitignore` that
  excludes a file that exists, assert the excluded file is still answered when named explicitly — the
  ignore filter exists so a listing is not noise, not to make a file unaddressable.

  Marshal the existing-file result with `encoding/json` and assert `dir` present under exactly that
  spelling, carrying the directory answer's own keys, with `id` and `unit` absent — the path branch is
  the reason both carry `omitempty`, and this is the only marshal that observes `dir` present.

  `TestResolve_MalformedGlyphEntries` asserts one entry per distinct grammar rejection the engine can
  reach, choosing target strings that produce `ReasonMemberTooDeep`, `ReasonUnitBadRune`,
  `ReasonUnitDotSegment` and `ReasonMemberKeyword`. Each answers with `Status` empty, `Error`
  non-empty, and `Reason` equal to the grammar's own word for that rejection. Marshal one of those
  entries with `encoding/json` and assert `error` and `reason` present under exactly those spellings
  and `status` absent — the only marshal that observes those three keys in that state, and the one that
  pins that a rejected target emits no status word at all. Assert in the same test
  that a call mixing one malformed target with two valid ones still answers the two valid ones
  normally and returns a nil call error, which is the whole reason the rejection is carried per entry
  rather than raised as the call's error.
- **Commit:** `test(engine): assert resolve's arity, path targets and error entries`

## Batch Tests

`verify:` runs the whole `internal/engine` package suite. That is the right scope, not an over-broad
one: this batch's cards edit two files in that package and nothing outside it, the package's whole
suite runs in well under a second on this host, and every T3 test in it is a must-keep-passing gate
for a batch that grows a file T3 wrote. The new coverage is the four `Test` functions of card 11, the
five of card 12 and the three of card 13, plus card 10's two table tests — between them one assertion
per row of the status vocabulary, the argument-order and arity contract, both halves of the
`unit: found`/`unit: not_found` distinction with its marshalled-JSON spelling, both members of the
per-entry error domain, the parse-once guarantee read off `unitMemo.parses`, the load-bearing sort
over the collision union, and both sides of the error boundary. The Loomyard timing assertion that
completes this verb's coverage is batch 5's, because it needs a checkout this batch does not.
