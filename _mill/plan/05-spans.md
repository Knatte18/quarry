# Batch: spans

```yaml
task: "Engine core (T3)"
batch: "spans"
number: 5
cards: 4
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
depends-on: [4]
```

## Batch Scope

This batch adds the internal span lookup (D16): the inverse of the walk, from a glyph back to the
declarations it names. It is what makes batch 6's round-trip criterion checkable, and it is the seam
the later `resolve` and `expand` verbs are built on — which is why it lives in `resolve.go`, the file
those verbs will grow, rather than in a file they would have to move.

It stops short of a status vocabulary on purpose: zero matches is an empty slice, not `not_found`.
The one fact the lookup cannot express without inventing a status type — a unit that names two
different directories — is carried on an unexported helper's second return instead, so it is
recorded now and promoted to a status later.

The external interface batch 6 consumes: `SpansOf`, `symbolsOfUnit` and `unitDirs`.

## Cards

### Card 31: Unit to directory, literal-first

- **Context:**
  - `glyph/glyph.go`
  - `docs/glyph.md`
  - `internal/engine/repo.go`
  - `internal/engine/walk.go`
  - `internal/engine/answer.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/resolve.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** New file `resolve.go`, `package engine`. Declare
  `func (r *Repo) unitDirs(unit string) (dirs []string, collision bool)`, the inverse of the walk's
  unit derivation, resolved literal-first:

  1. if the directory named exactly by `unit` exists under the root, it is a hit, and the search in
     it is restricted to files belonging to `unit` by the walk's own rule — excluding any file whose
     clause is the directory package's `_test` sibling, which belongs to `unit + "_test"`;
  2. if `unit` ends in `_test` and the directory named by `unit` with that suffix trimmed exists, it
     is a hit, and the search in it is restricted to files whose clause is exactly that directory
     package's `_test` sibling;
  3. neither existing means no directory at all.

  When **both** exist, both are returned and `collision` is true. `SpansOf` ignores `collision`; the
  later `resolve` verb promotes it into an `ambiguous` status when it builds the status vocabulary,
  and putting the flag on an unexported helper rather than on a public return means this task records
  the fact without inventing a status type that is not its to design.

  The comment must give the reason for literal-first: a directory literally named `foo_test/` is
  legal Go and the walk gives its declarations the unit `.../foo_test`, so an unconditional strip
  would send the lookup into a `foo/` directory that need not exist and one glyph string would name
  two different units. Record that the identifier contract gives the external test unit its
  pseudo-path without saying what happens when a real directory spells the same string, that this is
  quarry's chosen behaviour, and that amending the contract is not this task's to do. Note that
  neither repository the tests run against has such a directory, which is exactly why the rule must
  be right by construction rather than by test.
- **Commit:** `feat(engine): map a glyph unit back to its directories, literal-first`

### Card 32: symbolsOfUnit and SpansOf

- **Context:**
  - `glyph/glyph.go`
  - `glyph/parse.go`
  - `glyph/errors.go`
  - `internal/engine/repo.go`
  - `internal/engine/walk.go`
  - `internal/engine/ignore.go`
  - `internal/engine/golang.go`
  - `internal/engine/strategy.go`
  - `internal/engine/answer.go`
  - `internal/engine/errors.go`
  - `internal/engine/extension.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the unit-level primitive and the public lookup:

  - `func (r *Repo) symbolsOfUnit(unit string, ig *ignoreSet) ([]Symbol, error)` — parse each of the
    unit's `.go` files **once** and return every symbol in all of them, each with `File` set to the
    declaration's repository-relative, forward-slash path, ordered by file then start line. It
    resolves its directories through `unitDirs` and, when both exist, returns the union. In every
    branch the directory's `.go` files are filtered through the same ignore set the walk uses before
    being parsed: without that filter a gitignored `.go` file beside listed ones would contribute
    spans the walk never listed. A file that cannot be read, is not valid UTF-8, or belongs to a
    directory whose unit is unspellable contributes nothing — exactly as it contributes no symbols
    to a walk answer, which is what keeps the two readings equal. A parse the grammar reports an
    error on still contributes its surviving symbols, for the same reason.
  - `func (r *Repo) SpansOf(g glyph.Glyph) ([]Symbol, error)` — the public thin wrapper. It first
    returns `ErrLanguageUnsupported`, wrapped, when `g.Lang` is not `glyph.Go`, so the error names
    the real cause. It then validates `g` by round-tripping it through
    `glyph.Parse(g.Lang, g.String())` and returns the resulting parse error wrapped on failure — a
    `Glyph` is a plain struct a caller can build by hand, and that one check covers the empty unit, a
    dot segment, a member that is too deep and every other alphabet violation by calling the one
    implementation of the grammar rather than restating its rules here. It is also what makes the
    walk's claim that no root span is ever produced true rather than aspirational. It then builds a
    fresh `ignoreSet` for the root, extends it along the root-to-unit-directory chain, calls
    `symbolsOfUnit`, and filters by owner chain and name. Zero matches is an empty slice and a nil
    error — there is no status vocabulary here.

  Say in `symbolsOfUnit`'s doc comment why it is the primitive and `SpansOf` the wrapper rather than
  the other way round: a per-glyph lookup re-parses the whole unit directory and nothing is cached,
  so grouping by unit is what keeps a whole-repository check in seconds rather than minutes, and the
  later `resolve` verb needs the same grouping anyway.
- **Commit:** `feat(engine): add SpansOf and the unit-level symbol primitive`

### Card 33: ErrLanguageUnsupported gets its one caller

- **Context:**
  - `internal/engine/resolve.go`
  - `internal/engine/extension.go`
  - `internal/engine/toc.go`
- **Edits:**
  - `internal/engine/errors.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rewrite `ErrLanguageUnsupported`'s doc comment. Both of its original triggers are
  gone: every file is now listed regardless of language, so an unmapped extension is no longer an
  error, and there is no language override any more, so a language can no longer be requested. Its
  one remaining trigger is `SpansOf` called with a non-Go `Lang`, which is reachable because a
  `glyph.Glyph` is a struct a caller can build by hand with any language, so the engine needs a
  defined answer for a language it has no extractor for. Keep the wrapping convention note. Keep the
  sentinel itself and its name unchanged.
- **Commit:** `docs(engine): rewrite ErrLanguageUnsupported for its one remaining caller`

### Card 34: SpansOf tests

- **Context:**
  - `internal/engine/resolve.go`
  - `internal/engine/repo.go`
  - `internal/engine/answer.go`
  - `internal/engine/errors.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/testdata/tree/pkg/alpha.go`
  - `glyph/glyph.go`
  - `glyph/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/resolve_test.go`
  - `internal/engine/testdata/foo_test/literal.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** The fixture directory literally named `foo_test` holds one `package foo_test`
  file declaring one function, so the literal-first branch has something real to find.

  `resolve_test.go` asserts:

  - A hit: a glyph naming a declaration in the committed `tree/pkg` fixture returns exactly its
    spans, with `File` set to the repository-relative path.
  - A miss: a glyph whose unit exists but whose name does not returns an empty slice and a nil
    error.
  - The external-test unit: a glyph whose unit is the fixture package's `_test` sibling finds the
    declaration in the external test file and nothing from the package's own files, and a glyph in
    the package's own unit finds the reverse.
  - A glyph whose unit directory does not exist returns an empty slice and a nil error.
  - Literal-first: a glyph whose unit names the committed `foo_test` directory finds that
    directory's declaration, not one reached by stripping the suffix.
  - The collision: over a `.scratch/` tree holding both a `foo/` directory with an external test
    package and a literal `foo_test/` directory, `unitDirs` returns both directories with
    `collision` true, while `SpansOf` returns the union.
  - A non-Go `Lang` returns `ErrLanguageUnsupported`, asserted with `errors.Is`.
  - Argument validation: a hand-built glyph with an empty unit, one with a `..` segment and one with
    a three-component member are each rejected through the parse round-trip, with the grammar's own
    parse error surfacing via `errors.As` rather than a silent root-directory answer.
  - The ignore filter: over a `.scratch/` tree whose `.gitignore` excludes one `.go` file beside two
    listed ones, no span from the excluded file is returned.
- **Commit:** `test(engine): cover SpansOf, the unit lookup and its validation`

## Batch Tests

`verify:` is the same build-then-test pair the earlier batches use.

The batch's own coverage is `resolve_test.go`. It reuses the committed `testdata/tree/` fixtures
batch 3 added, adds one committed directory literally named `foo_test/`, and builds the collision and
ignore-filter cases as run-time `.scratch/` trees via `writeScratchTree`, since a committed tree
cannot hold a file its own `.gitignore` excludes.

Every test from batches 1–4 must keep passing untouched: this batch adds a new file and rewrites one
doc comment, and touches no walk rule.
