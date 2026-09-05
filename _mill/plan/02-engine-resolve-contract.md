# Batch: engine-resolve-contract

```yaml
task: "Glyph self-form and the resolve contract (C1)"
batch: "engine-resolve-contract"
number: 2
cards: 9
verify: go test ./internal/engine/... ./quarry/...
depends-on: [1]
```

## Batch Scope

This batch makes `Resolve` take glyphs only. It deletes the engine's own `#`-containment classifier
(D4), routes a self glyph through the existing listing producer (D10, D10a), renames
`ResolveResult.Dir` to `ResolveResult.Listing` (D13), and gives the text renderer its listing branch
(D14). It is one batch because the rename breaks every reader of that field simultaneously, so the
engine, the facade's text renderer, and the two test files that name the field must move together or
the module stops compiling.

It depends on batch 1 for `Glyph.IsSelf`. It does not depend on batch 3: the two batches touch
disjoint files (`resolve.go`/`answer.go`/`walk.go`/`quarry/text.go` here, `expand.go`/`repo.go`/
`quarry/quarry.go` there) and may run in parallel.

The external interface batch 4 consumes: a `Resolve` that answers a bare path with a
`no_separator` rejection payload rather than a listing, and `ResolveResult.Listing`.

Batch-local decision beyond the overview's Shared Decisions: card 18 touches
`internal/cli/cli_test.go` for the field rename alone. It changes no assertion and no expectation —
the behavioural retargeting of that file belongs to batch 4. The edit is here only because leaving
it out would stop `internal/cli`'s test binary from compiling, which the overview's
green-compile decision forbids.

## Cards

### Card 10: `ResolveResult.Dir` becomes `ResolveResult.Listing`

- **Context:**
  - `docs/rewrite-plan.md`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename the `ResolveResult` field `Dir *DirAnswer` to `Listing *DirAnswer` and its
  JSON tag from `dir,omitempty` to `listing,omitempty`, keeping the field in its current position
  between `Candidates` and `Error` so the emitted object's key order is unchanged. Rewrite the
  field's doc comment: the block is a listing of one file's entry or of a directory's contents, which
  is why `listing` replaces `dir` — the old name claimed "directory" while also carrying single-file
  answers and repeated the inner key's word one level up. Do not rename `DirAnswer.Dir`, the inner
  repository-relative path string: it is shared with the `toc` answer, is out of scope, and `dir` is
  the right word for a path that is always a directory. `answer.go`'s own file header states that no
  field is added or renamed without a corresponding Shared Decision change; add a sentence recording
  that this task is that change, naming the `dir` to `listing` rename.
- **Commit:** `refactor(engine)!: rename ResolveResult.Dir to Listing`

### Card 11: a self glyph resolves through the listing producer

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/toc.go`
  - `glyph/glyph.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename `resolvePathTarget` to `resolveSelfTarget`. Its body is unchanged apart
  from the field rename card 10 made — it still calls `r.TOC` with `TOCOptions{Depth: 0, Symbols:
  &symbolsOff}` where `symbolsOff` is a local false, and it keeps its three-way disposition intact:
  a nil error is `StatusFound` with the answer's address in `Listing`, `ErrTargetNotFound` is
  `StatusNotFound` with no listing, `ErrTargetOutsideRepo` is the error field, and any other error
  fails the whole call. Preserve the explicit `Symbols: &symbolsOff`: relying on the per-target
  default would switch symbols on for a file target, and a self glyph answers where a thing is, not
  what is inside it. In `resolveGlyphTarget`, immediately after `glyph.Parse` succeeds and before
  `m.dirsOf` is read, branch on `g.IsSelf()`: call `resolveSelfTarget` with `g.Unit`, then set
  `Target` to the argument verbatim and `ID` to `g.String()` on the returned result, and set `Unit`
  by the same rule every other glyph uses — `StatusFound` when `m.dirsOf(g.Unit)` returns a non-empty
  directory list and `StatusNotFound` when it is empty — on the `StatusNotFound` disposition only,
  matching where `Unit` is set for a member glyph. Do not suppress `Unit`: the external test unit is
  the case that forces it, since `internal/logger_test#Foo` resolves through `unitDirs`' `_test`
  stripping while `internal/logger_test#` is a path that exists nowhere, and `status: not_found,
  unit: found` is the only complete answer for it. Rewrite `resolveSelfTarget`'s doc comment: it
  answers a self glyph, not a target with no `"#"`; it inherits the explicitly-named-gitignored-target
  rule, the never-follow-a-symlink rule and the empty-string-and-dot-mean-the-root rule from `TOC`
  for free; its `ErrTargetOutsideRepo` arm is now unreachable, because `checkGoUnit` rejects `.` and
  `..` segments before a unit can escape, and the arm is kept deliberately as a defensive sentinel
  translation rather than deleted. Extend `resolveGlyphTarget`'s doc comment with the self branch and
  with the `Unit` rule above. Do not change `statusForMatches`, `matchesFor`, or `unitDirs`.
- **Commit:** `feat(engine): resolve a self glyph through the listing producer`

### Card 12: the engine stops classifying targets

- **Context:**
  - `glyph/parse.go`
  - `glyph/errors.go`
- **Edits:**
  - `internal/engine/resolve.go`
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Delete the `isGlyphTarget` function and its doc comment from
  `internal/engine/resolve.go`. In `resolve`, drop the `if isGlyphTarget(target)` branch so every
  target goes to `resolveGlyphTarget` unconditionally, and rewrite `resolve`'s doc comment, which
  today narrates that split. A bare path now fails `glyph.Parse` with `ReasonNoSeparator` and becomes
  the pre-resolution rejection shape `resolveGlyphTarget` already produces: `Target`, `Error`,
  `Reason`, no `Status`, and a nil error so the other targets in the call are still answered. Write
  no separator check of any kind in this package to replace the deleted one. Remove the `strings`
  import if this deletion leaves it unused. In `internal/engine/resolve_test.go`, delete
  `TestIsGlyphTarget` and its doc comment in the same edit: it tables the deleted function over
  `"#"`-containing and `"#"`-free targets, and neither the function nor the split it asserts
  survives. Change nothing else in that file — card 16 owns the behavioural rows.
- **Commit:** `refactor(engine)!: delete isGlyphTarget and route every target to the grammar`

### Card 13: `SpansOf` says what a self glyph does there

- **Context:**
  - `glyph/glyph.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add one sentence to `SpansOf`'s doc comment: a self glyph reaches this function's
  own inline owner-and-name filter, matches nothing, and so returns the existing empty slice with a
  nil error — `SpansOf` has no status vocabulary, and "nothing matches" is exactly what an empty
  slice with a nil error means there. Change no line of `SpansOf`'s code. It is the function the
  round-trip test is written against, so it must not be edited for behaviour, and its deliberate
  duplicate of the `matchesFor` filter stays as it is.
- **Commit:** `docs(engine): record a self glyph's disposition in SpansOf`

### Card 14: `unitSpellable`'s rejection list gains the new reason

- **Context:**
  - `glyph/errors.go`
  - `glyph/parse.go`
- **Edits:**
  - `internal/engine/walk.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** `unitSpellable` probes `unit + "#x"` through `glyph.Parse`. For a unit whose own
  name carries a `"#"` that probe string now has two of them and is rejected as
  `multiple_separators` where it was previously rejected as `member_bad_rune`. The reason changes and
  the boolean does not, so the emitted answer is byte-identical and the round trip stays true by
  construction. Update `unitSpellable`'s doc comment, which lists the rejections it covers, to name
  `multiple_separators` and to state that asymmetry explicitly: naming such a directory as a target
  is an error under the new contract, while encountering one below a listed target is still a silent
  listing with no symbols, because the contract governs what a caller may name and this function
  governs what the walk may mint. Do not edit `walkDir`, `unitFor`, or any other line of this file —
  the walk's traversal, ignore handling and answer shape are out of scope.
- **Commit:** `docs(engine): record the new probe rejection in unitSpellable`

### Card 15: the text renderer's listing branch

- **Context:**
  - `internal/engine/answer.go`
- **Edits:**
  - `quarry/text.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Insert a new branch in `RenderResolveText` between the existing `r.Status == ""`
  branch and the existing `r.ID != ""` branch, keyed on `r.Listing != nil`. Order is the whole of it:
  the glyph branch fires on a non-empty `ID`, which a self glyph now also carries, and would print a
  bare status line with no listing. The new branch emits line 1 as `r.ID`, a space and the status
  word, falling back to `r.Target` when `r.ID` is empty so line 1 can never begin with a space — the
  same external-caller courtesy the first branch's empty-error guard already extends. It then emits
  the block `strings.Join(dirBlocks(*r.Listing), "\n")` only when `r.Status == StatusFound`; write
  that status guard explicitly, or a hand-built `not_found` carrying a listing would print a
  directory block under a negative status. Keep the existing `default` arm but reduce it to line 1
  alone — `r.Target`, a space, the status word — since the engine can no longer produce a path result
  without an `ID`; document it as the guard for an externally constructed value. Rename the two
  `r.Dir` references to `r.Listing`. Rewrite the numbered branch list in `RenderResolveText`'s doc
  comment so branch 3 describes the listing branch and branch 4 the reduced default, and record that
  a self glyph whose status is `not_found` has a nil listing and falls to the glyph branch, which
  prints the identifier, ` not_found`, and the unit suffix — the unit field is always set on that
  answer, so the suffix is always present. Change no other renderer.
- **Commit:** `feat(quarry): render a self glyph's listing block`

### Card 16: engine tests for the self form and the rejections

- **Context:**
  - `internal/engine/resolve.go`
  - `internal/engine/answer.go`
  - `internal/engine/testdata/tree/pkg/export_test.go`
  - `internal/engine/testdata/tree/pkg/doc.go`
  - `glyph/errors.go`
- **Edits:**
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rewrite `TestResolve_PathTargets` for the self form, renaming it to say so. Its
  rows: `internal/engine/testdata/tree/pkg/doc.go#` returns `StatusFound` with a `Listing` holding
  exactly one `FileEntry` whose `Symbols` is nil, and whose enclosing directory is
  `internal/engine/testdata/tree/pkg`; `internal/engine/testdata/tree/pkg#` returns `StatusFound`
  with that directory's populated `Listing`; a self glyph naming neither a file nor a directory
  returns `StatusNotFound` with a nil `Listing` and `Unit` equal to `StatusNotFound`. On all three,
  `ID` is the argument with its trailing `"#"` intact and `Target` is the argument verbatim. Add the
  external-test-unit row that the `Unit` rule turns on:
  `internal/engine/testdata/tree/pkg_test#` returns `StatusNotFound` with a nil `Listing` and `Unit`
  equal to `StatusFound`, because `unitDirs` strips the `_test` suffix and finds
  `internal/engine/testdata/tree/pkg` while no `pkg_test` directory exists on disk; assert the
  companion row `internal/engine/testdata/tree/pkg_test#TestExported` still resolving beside it, so
  the documented asymmetry is visible in one test. Add a bare-path row asserting an empty `Status`
  with `Reason` equal to `no_separator`, and a two-separator row asserting an empty `Status` with
  `Reason` equal to `multiple_separators`. Add a multi-target `Resolve` call mixing a member glyph, a
  self glyph and a bare path, asserting three positionally correct results and a nil error — the
  rejection taints only itself. Rename any `Dir` field reference in this file to `Listing`. Use the
  existing fixture trees under `internal/engine/testdata/`; do not add a fixture.
  Do not edit `internal/engine/roundtrip_test.go`, and do not edit `internal/engine/glyph_test.go`:
  both must pass unchanged and are this task's regression gate on the walk.
- **Commit:** `test(engine): pin the self form, the rejections and the mixed target list`

### Card 17: facade tests for the rename and the listing branch

- **Context:**
  - `quarry/text.go`
  - `internal/engine/answer.go`
  - `glyph/errors.go`
- **Edits:**
  - `quarry/text_test.go`
  - `quarry/repo_test.go`
  - `quarry/render_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `quarry/text_test.go`, extend `TestRenderResolveText` with a self-glyph
  `found` row emitting the identifier, a space, `found`, then the directory block; a self-glyph
  `not_found` row emitting the identifier, ` not_found (unit found)`; and a second `not_found` row
  emitting ` (unit not_found)` for a path whose unit does not exist either. Both `not_found` rows
  must agree with the `Unit` values card 16 pins on the engine side. Add the two guard rows the new
  branch's explicit guards exist for: a hand-built `not_found` carrying a non-nil `Listing` emits
  line 1 only, with no directory block; and a hand-built value carrying a `Listing` and an empty
  `ID` emits the target on line 1, never a line beginning with a space. Add a row for the reduced
  `default` arm: a hand-built value with no `ID` and no `Listing` still emits line 1 alone. Assert
  the renderer's standing invariants on every new case — no trailing whitespace on any line, and
  exactly one closing newline. In `quarry/repo_test.go`, retarget `TestRepoResolve_PathTarget`: it
  resolves a bare path today, which card 12 turns into a `no_separator` rejection with an empty
  status, so its argument becomes that path's self glyph with a trailing `"#"`, its `Dir` field
  reference becomes `Listing`, and the test is renamed off the word "PathTarget", which no longer
  describes what it asserts. In `quarry/render_test.go`, rename the two `ResolveResult` composite
  literals that set `Dir` to set `Listing` — one in the round-trip table's path row, one in
  `TestRenderResolveJSON_KeyOrder` — and change the pinned key list in that second test so its last
  entry is the `listing` key rather than the `dir` one. That key list is pinned to `ResolveResult`'s
  own field declaration order, which card 10 leaves unchanged, so only the key's spelling moves.
  Both files break the `quarry` test binary's compile until this card lands, which is why they are
  named here rather than left to a later batch.
- **Commit:** `test(quarry): pin the listing branch and its guards`

### Card 18: keep `internal/cli`'s test binary compiling

- **Context:**
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename every `Dir` field reference on a resolve result in
  `internal/cli/cli_test.go` to `Listing`, so the package's test binary still compiles after card
  10's rename. This is a mechanical rename and nothing else: do not retarget any case, do not change
  any expected exit code, and do not touch the bare-path resolve rows, which batch 4 rewrites. The
  behavioural failures those rows now produce are the bounded red window the overview records, and
  hiding them with a skip here would hide exactly the regressions batch 4 must fix.
- **Commit:** `test(cli): follow the ResolveResult.Listing rename`

## Batch Tests

`verify: go test ./internal/engine/... ./quarry/...` covers both packages this batch changes
behaviour in: `internal/engine`'s `resolve_test.go` (rewritten here), `answer_test.go`,
`walk_test.go`, `roundtrip_test.go` and `glyph_test.go`, and `quarry`'s `text_test.go`,
`repo_test.go` and `render_test.go`. `internal/cli` is deliberately outside the scope: card 18
restores its compile, but its behavioural rows stay red until batch 4, so including it here would
make this batch's `verify:` fail for a reason this batch does not own.

`internal/engine/roundtrip_test.go` and `internal/engine/glyph_test.go` are the regression gate on
the walk: both must pass unchanged, which is what proves the new grammar rule changed
`unitSpellable`'s reason without changing its answer. The Loomyard-gated tests in this package skip,
since `LADDER_LOOMYARD_REPO` is unset for every batch but batch 5.
