# Batch: engine-unit-exports

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'engine-unit-exports'
number: 2
cards: 5
verify: go test ./internal/engine/ -run 'TestPackageClause|TestUnitsForClauseMap|TestClauseMapForFiles|TestWalk|TestRepoTOC|TestResolve|TestExpand|TestSpansOf|TestRoundTrip'
depends-on: []
```

## Batch Scope

This batch opens the clause-map seam the facade needs: three exported engine entry points —
`PackageClause`, `(*Repo).ClauseMapForFiles` and `UnitsForClauseMap` — extracted from the existing
`dirPackage` and `unitFor` implementations rather than copied, with `dirPackage` refactored to call
the first and third so each rule keeps exactly one implementation.
Package `quarry` cannot reach `dirPackage` or `unitFor` — both are unexported — and it is the layer
that assembles a delta batch and must derive each side's units, so export is the only seam across
that boundary.

The batch is one unit because the extraction and the refactor are the same change: exporting the
helpers without rewiring `dirPackage` through them would create the second implementation this task
exists to avoid.
It is a root batch — it edits `internal/engine/walk.go` and creates a new file, sharing neither with
batch 1 nor batch 5.

The external interface batch 6 consumes: given a directory's repository-relative path and a
`map[base name]package clause`, `UnitsForClauseMap` returns the directory's dominant clause and a
function from a file's base name to that file's glyph unit; `PackageClause` turns one file's bytes
into its clause; `ClauseMapForFiles` builds the on-disk map for a supplied list of base names.

Batch-local decision: the three exports live in a new `internal/engine/units.go` rather than in
`walk.go`, so `walk.go`'s file comment — which describes the per-directory work `Repo.TOC` drives —
stays true, and the exported seam has a file whose own comment can state that it exists for a caller
outside this package.

## Cards

### Card 6: extract PackageClause and rewire dirPackage's bytes-to-clause step

- **Context:**
  - `internal/engine/strategy.go`
  - `internal/engine/extension.go`
  - `internal/engine/treesitter/treesitter.go`
  - `internal/engine/repo.go`
  - `internal/engine/doc.go`
  - `internal/engine/loomyard_timing_test.go`
- **Edits:**
  - `internal/engine/walk.go`
- **Creates:**
  - `internal/engine/units.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/engine/units.go` with a file comment stating that it holds the
  exported clause-and-unit seam package `quarry` needs, that every rule in it is the same rule the
  walk applies rather than a second copy, and that the unit is a directory-level fact no single
  file's content can establish.
  Declare `PackageClause(base string, src []byte) (clause string, ok bool)` in it.
  It resolves the language from the extension of `base` with `LanguageForExtension`, looks the
  strategy up with `StrategyFor`, parses `src` inside `treesitter.WithTree`, and returns the
  strategy's `Package` result.
  `ok` is false for exactly the four conditions the discussion fixes for this function, which are the
  same four under which `dirPackage` records no clause today: an extension with no language or no
  registered strategy, bytes that are not valid UTF-8, a `treesitter.WithTree` call that returns an
  error, and an empty clause string.
  The UTF-8 check belongs **inside** this function rather than in each caller: it is the one place
  both sides of a directory's clause vote pass through, and a caller-side check would hold only for
  the caller that reads from disk, leaving bytes fetched from a revision unchecked and letting one
  file vote on one side of a comparison only.
  Then rewire `dirPackage` in `walk.go` to call `PackageClause` for its per-file clause step instead
  of performing the language lookup, strategy lookup and parse inline.
  Keep an extension guard ahead of `dirPackage`'s own `os.ReadFile`, resolving the language from the
  base name's extension before reading anything, and call `PackageClause` only for a file whose
  extension resolves to one.
  That ordering is load-bearing and is the order the existing code already runs in: the language and
  strategy lookups come first and the read second, so a directory holding binaries, testdata or
  assets never reads them at all.
  Reading every entry before consulting its extension would leave the clauses map identical while
  making the walk's first pass read the whole tree — a regression the walk's own priced cost note
  records the budget for, and one the timing assertion in `internal/engine/loomyard_timing_test.go`
  selected by this batch's `verify:` would catch.
  `dirPackage` keeps its own read-failure handling where it is; the UTF-8 rejection now lives in
  `PackageClause` and the two together reproduce today's skip conditions exactly.
  `dirPackage`'s observable behaviour must not change: the same files vote, the same clauses map is
  returned, and the same tie-break runs.
- **Commit:** `refactor(engine): extract PackageClause and call it from dirPackage`

### Card 7: extract UnitsForClauseMap from the clause vote and the unit derivation

- **Context:**
  - `internal/engine/repo.go`
  - `internal/engine/glyph_test.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/units.go`
  - `internal/engine/walk.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare
  `UnitsForClauseMap(dirRel string, clauses map[string]string) (dirPkg string, unitOf func(base string) string)`
  in the units file.
  It runs the existing vote — the most common clause among files whose clause does not end in
  `"_test"`, falling back to the most common clause overall when every clause ends in `"_test"`,
  with the lexicographically smallest clause breaking a tie — and returns that as `dirPkg`, together
  with a function mapping a file's base name to the unit `unitFor` would derive for it from
  `dirRel`, `dirPkg` and that file's own clause.
  A base name absent from `clauses` maps to the unit a file with an empty clause would get; state
  that in the doc comment rather than leaving it to the reader.
  Move the vote itself out of `dirPackage` and the derivation out of `unitFor` so each rule exists
  once: `dirPackage` computes its clauses map and then returns `UnitsForClauseMap`'s `dirPkg`, and
  `unitFor` **stays where it is as a thin wrapper** delegating to the same derivation.
  Keep the wrapper rather than removing the function and rewriting its call sites: `unitFor` is
  called from `symbolsOfDir` in `internal/engine/resolve.go` as well as from this file, and that file
  is deliberately outside this batch's edit set — a wrapper keeps one implementation of the rule
  without widening the batch onto the resolve path.
  The discriminator stays the clause and never the filename — a file belongs to the external-test
  unit only when its clause is exactly `dirPkg` plus the `"_test"` suffix — and the repository root
  stays the documented exception returning the empty string for both branches.
  Update this file's own header comment in the same commit: it enumerates the per-directory work as
  `dirPackage`, the other unexported methods, and the unexported free function `unitFor`, and it must
  now say that the clause vote and the unit derivation live in the exported helpers this batch adds,
  with the two names here delegating to them.
  Preserve `mostCommonClause`'s existing tie-break exactly; its determinism is what keeps a
  directory's answer independent of directory-read order.
- **Commit:** `refactor(engine): extract UnitsForClauseMap from the clause vote and unit derivation`

### Card 8: add ClauseMapForFiles

- **Context:**
  - `internal/engine/repo.go`
  - `internal/engine/walk.go`
  - `internal/engine/extension.go`
- **Edits:**
  - `internal/engine/units.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare
  `(*Repo).ClauseMapForFiles(dirRel string, bases []string) (map[string]string, error)` in the units
  file.
  It reads each base name in `bases` from the directory `dirRel` under the repository root and
  records the clause `PackageClause` returns when that call reports ok.
  It does not check UTF-8 validity itself — card 6 puts that check inside `PackageClause`, so both
  this on-disk caller and the revision-side caller in batch 6 get it from one place — and it applies
  the same extension guard ahead of its own read that `dirPackage` does, so a non-source file named
  in `bases` is never read.
  It takes the file list rather than enumerating the directory itself, and its doc comment must say
  why: the caller chooses one enumeration rule and applies it to both sides of a comparison, so a
  directory's dominant clause cannot differ between the sides through a difference in which files
  were counted.
  A base name it cannot read, whose bytes are not valid UTF-8, that fails to parse, or whose clause
  is empty is skipped and records no clause — the same rule `dirPackage` already applies — and is
  explicitly **not** an error: a file listed in the index but deleted from the working tree is a
  routine input for the delta verb and must never fail the whole call.
  The `error` return is reserved for a failure of the call itself, such as the directory being
  unreadable; say so in the doc comment.
  It does not apply the engine's ignore set, and its doc comment must state that the caller's
  enumeration rule is what decides which files count.
- **Commit:** `feat(engine): add ClauseMapForFiles for a caller-supplied file list`

### Card 9: table tests for the three exported helpers

- **Context:**
  - `internal/engine/units.go`
  - `internal/engine/walk.go`
  - `internal/engine/repo.go`
  - `internal/engine/repo_test.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/units_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestPackageClause`, `TestUnitsForClauseMap` and `TestClauseMapForFiles` in
  package `engine`.
  `TestPackageClause` is a table over in-memory bytes covering all four conditions that make the ok
  flag false, plus the success case: a plain Go file, a file whose extension has no registered
  strategy, bytes that are not valid UTF-8, a file with no package clause at all, and a file whose
  content does not parse — asserting the clause and the ok flag for each.
  The invalid-UTF-8 row is the one that pins the check's location: it must be false here, in the
  function itself, rather than only in a caller.
  `TestUnitsForClauseMap` is a table over `map[base name]clause` inputs covering exactly the cases
  the discussion's rejected alternatives name, since they are the reason this helper exists: a plain
  package; a package with an external test file, where the two clauses must produce two distinct
  units; a package legitimately named `httptest`, asserted **not** split into a second unit; a tie
  between two equally common clauses, broken lexicographically; a directory in which every clause
  ends in the test suffix; and the repository root, whose unit is the empty string for both
  branches.
  `TestClauseMapForFiles` uses a directory built under `t.TempDir()` and asserts: a clause is
  recorded for each readable Go file; a base name naming a file that does not exist on disk is
  skipped with no clause recorded and no error returned; a file whose bytes are not valid UTF-8 is
  skipped the same way, through the check card 6 places inside `PackageClause`; and a base name whose
  extension resolves to no language is skipped without the file being read at all.
  Every case in these tables asserts against the helper's own return values only; none reads a
  committed fixture tree.
- **Commit:** `test(engine): table tests for PackageClause, UnitsForClauseMap and ClauseMapForFiles`

### Card 10: prove the refactor preserved the walk's behaviour

- **Context:**
  - `internal/engine/units.go`
  - `internal/engine/walk.go`
  - `internal/engine/walk_test.go`
  - `internal/engine/toc_test.go`
  - `internal/engine/resolve.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Verification-only card, producing no diff.
  Run the existing `internal/engine` walk, toc, resolve, expand, spans and round-trip suites and
  confirm every test passes unchanged.
  The resolve-side suites are named explicitly because they are the non-obvious half: `symbolsOfDir`
  in `internal/engine/resolve.go` consumes both of the functions cards 6 and 7 rewrite, so a
  behaviour change there surfaces on the resolve and expand paths rather than on the walk.
  Pay particular attention to the four cases that pin the rules those cards moved: the package
  deviation key appearing only on an external test file, the package literally named `httptest` not
  being split, the tie-break picking the lexicographically smaller clause deterministically, and the
  ordering-and-symbol-source-order case.
  If any of them now fails, the refactor changed behaviour and the correct response is to fix the
  refactor, never to adjust the test — these four are the pinned statement of the rules the two
  exported helpers were extracted from.
  Confirm as well that no file in this batch added an import of a package outside `internal/engine`
  and its existing dependencies.
- **Commit:** none

## Batch Tests

`verify:` runs
`go test ./internal/engine/ -run 'TestPackageClause|TestUnitsForClauseMap|TestClauseMapForFiles|TestWalk|TestRepoTOC|TestResolve|TestExpand|TestSpansOf|TestRoundTrip'`.
The first three patterns select card 9's new tables.
The rest are deliberately much broader than this batch's own additions, and each is load-bearing.
The walk and toc patterns cover where the clause vote and the unit derivation are already pinned —
the `httptest` case, the external-test deviation case, the lexicographic tie-break, and the whole
table-of-contents surface that reads a file's unit.
The resolve, expand, spans and round-trip patterns cover the non-obvious consumer: `symbolsOfDir` in
`internal/engine/resolve.go` calls both of the functions cards 6 and 7 rewrite, so a behaviour change
introduced by the refactor surfaces on those paths and would otherwise pass this batch's verify in
silence.
Card 10 names exactly these suites and every one of them is in this selection, so the refactor's
behaviour-preservation claim is checked at this batch's own boundary rather than deferred to the
repository-wide done gate.
