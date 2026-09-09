# Batch: engine-vocabulary

```yaml
task: 'Batch-answer contract: per-target coverage + fail-closed Status helpers'
batch: engine-vocabulary
number: 1
cards: 3
verify: go test ./internal/engine/ ./quarry/
depends-on: []
```

## Batch Scope

This batch delivers the fail-closed `Status` vocabulary half of the task: the exported `Statuses`
enumeration, the `Status.Known()` predicate, and the `ResolveResult.Rejected()` marker in
`internal/engine/answer.go`, their engine-side tests, and the one facade edit that surfaces
`Statuses` to an external importer. It is one batch because all three symbols are declared in one
file, are tested together against one vocabulary, and the facade edit is a single `var` line that is
meaningless without them.

The external interface batch 3 consumes is exactly two methods — `ResolveResult.Rejected()` and
(not used in this repository, but shipped) `Status.Known()` — plus `quarry.Statuses`. Batch 3 uses
`Rejected()` at `quarry/text.go` and `internal/cli/cli.go`; it uses neither `Known()` nor
`Statuses`, deliberately, because both in-repository switches are already total.

This batch has no batch-local decisions that differ from the overview's Shared Decisions.

## Cards

### Card 1: `Statuses`, `Status.Known()`, and `ResolveResult.Rejected()`

- **Context:**
  - `internal/engine/name.go`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add three declarations to `internal/engine/answer.go`, plus a clause in the file header comment.

  1. Immediately after the `Status` constant block (the block ending with `StatusMultipart`), add:

     ```go
     var Statuses = []Status{StatusFound, StatusNotFound, StatusAmbiguous, StatusMultipart}
     ```

     Its doc comment must state that the slice lists all four `Status` values in the same order as
     the constant block above, that Go cannot reflect over package-level constants so the slice is
     the only way a test or a caller enumerating the vocabulary can range over it, and that adding a
     constant means adding it here in the same edit. Mirror the register and the same-edit sentence
     of `NameReasons`'s own doc comment in `internal/engine/name.go`.

  2. Immediately after `Statuses`, add the method Known() on `Status`, with exactly this body:

     ```go
     switch s {
     case StatusFound, StatusNotFound, StatusAmbiguous, StatusMultipart:
     	return true
     default:
     	return false
     }
     ```

     The receiver is a value receiver named `s`; the signature is `func (s Status) Known() bool`.
     The body is a switch over the four constants. Do not implement it as a range over `Statuses`.
     The doc comment must state that Known() reports whether `s` is one of the four values the
     constant block declares, that it is false for the empty string because the empty string is the
     pre-resolution rejection marker rather than a resolution outcome, and — as the rejected
     alternative — that ranging `Statuses` was rejected because `Statuses` is an exported slice a
     caller can mutate, which would make the predicate's truth set changeable at a distance, and
     because it would turn the vocabulary truth-table test into a tautology. Note in the same
     comment that the empty-string answer is load-bearing for `ResolveResult.Unit` and
     `ExpandAnswer.Unit`, which are absent on every status but `not_found`.

  3. Immediately after the `ResolveResult` struct's closing brace, add:

     ```go
     func (r ResolveResult) Rejected() bool { return r.Status == "" }
     ```

     The doc comment must state that `Rejected` reports whether `r` is a pre-resolution rejection of
     the target string itself rather than a resolution outcome, cite the struct's own documented
     rule that `Status` and `Error` are never both set and that `Status` is absent exactly when the
     target never reached resolution, and name the rejected alternative — testing `Error != ""` —
     as the derived marker rather than the primary one, false for a rejection whose message happened
     to be empty where `Status == ""` still holds.

  Extend the file header comment's enumeration of what the file declares so it names the new
  `Statuses` var and the two methods. Add no struct field and no JSON tag: the header's own closed
  emitted-key-set rule must stay satisfied without a Shared Decision change.
- **Commit:** `feat(engine): add Statuses, Status.Known and ResolveResult.Rejected`

### Card 2: engine-side vocabulary tests

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/name_test.go`
- **Edits:**
  - `internal/engine/answer_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add three tests to `internal/engine/answer_test.go`, in package `engine`.

  1. `TestStatus_Completeness` — mirror `TestName_ReasonCompleteness` in
     `internal/engine/name_test.go` structurally: build a locally written `want` set holding the
     four `Status` constants, fail when `len(Statuses)` differs from `len(want)`, fail on a repeated
     element, fail on an element absent from `want`, and fail on a `want` element absent from
     `Statuses`. The `want` set is written out as four literal map entries in the test; it is not
     derived from `Statuses`, because deriving it is what would make the test assert nothing.

  2. `TestStatus_Known` — a truth table. True: every element of `Statuses`, ranged over rather than
     restated as four literals. False: `Status("")`, and at least two plausible-looking bogus values
     such as `Status("found ")` and `Status("FOUND")`, pinning that `Known` does no trimming and no
     case folding. Note in the test's own doc comment that the range over `Statuses` is a genuine
     cross-check only because Known()'s body is an independent switch, so the test fails when a
     constant is added to one enumeration and not the other.

  3. `TestResolveResult_Rejected` — a truth table over hand-built `ResolveResult` values, no
     filesystem. True for a value whose `Status` is empty and whose `Error` is set. False for a
     value carrying each of the four statuses in turn. Include one further false case: a value whose
     `Status` is `StatusNotFound` and whose `Error` is also set, so the test pins that `Rejected`
     reads `Status` and not `Error`.

  Follow the file's existing table-test style: a `tests` slice of anonymous structs with a `name`
  field, driven through `t.Run`, with `t.Errorf` messages in the `got = X; want Y` shape the
  surrounding tests use. The three test names above follow the file's own `TestSubject_Case` shape,
  which every existing test in it carries; keep them exactly as prescribed.

  Extend `internal/engine/answer_test.go`'s own file header comment, which enumerates what the file
  pins, so it names the new subjects too: the `Status` vocabulary's completeness and the two
  fail-closed predicates. The file's header is the local convention for saying what a test file
  covers, and leaving it stale while adding three new subjects would be the same drift card 1
  closes for `internal/engine/answer.go`'s own header.
- **Commit:** `test(engine): cover Statuses completeness, Known and Rejected`

### Card 3: surface `Statuses` through the facade, with an identity test

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/name.go`
  - `quarry/repo_test.go`
- **Edits:**
  - `quarry/quarry.go`
- **Creates:**
  - `quarry/quarry_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add to `quarry/quarry.go`, immediately after the `Status` constant block that ends with
  `StatusMultipart`:

  ```go
  var Statuses = engine.Statuses
  ```

  Its doc comment must say it is the engine's own value and not a copy, so a caller enumerating the
  vocabulary and the engine's own test are reading one slice — the same sentence and the same reason
  the existing `NameReasons` var at the bottom of the same file already carries. Add no type alias
  and no constant: `Status` and the four constants are already aliased in this file, and a
  package-level var is the one thing a type alias does not carry across.

  Create `quarry/quarry_test.go` in package `quarry`, with a file header comment in the register of
  the package's other test files, holding one test — `TestFacadeVocabulariesAreEngineValues` — that
  asserts three things. `quarry/repo_test.go` is the in-package precedent for importing
  `github.com/Knatte18/quarry/internal/engine` from a test in this package; follow it.

  1. `Statuses` is the engine's own slice, not a copy: assert both that `len(Statuses)` equals
     `len(engine.Statuses)` and that `&Statuses[0] == &engine.Statuses[0]`, guarding the address
     comparison behind a non-empty length check so an empty slice fails with a readable message
     rather than panicking on the index.
  2. `NameReasons` is the engine's own slice, by the identical pair of assertions against
     `engine.NameReasons`. This assertion does not exist anywhere today — the sibling vocabulary the
     new var is modelled on is currently covered engine-side only — and it is added here so both
     vocabularies are pinned at the facade by one test.
  3. The two new methods are reachable through the aliased types without importing the engine:
     assign `Status("found").Known()` and `ResolveResult{}.Rejected()` to `bool` variables named in
     the test, using the `quarry` package's own `Status` and `ResolveResult` spellings, and assert
     their values are `true` and `true` respectively. This is a compile-level assertion first — it
     is what proves an external importer gets the methods through the alias — and a value assertion
     second.
- **Commit:** `feat(quarry): surface Statuses as the engine's own slice`

## Batch Tests

`verify: go test ./internal/engine/ ./quarry/` runs both packages this batch touches. It covers the
three new tests in `internal/engine/answer_test.go` (card 2) and the new
`quarry/quarry_test.go` (card 3), and it re-runs both packages' existing suites so the additive
declarations are proved not to have broken anything already passing — in particular
`internal/engine/answer_test.go`'s existing JSON-shape tests, which are the guard that no emitted
key was added.

The scope is deliberately two packages rather than the whole module: this batch declares one var
and two methods and edits no call site, so no third package's behaviour can change. The overview's
module-wide `go vet ./...` runs at the batch boundary and catches any compile-level fallout beyond
these two packages.
