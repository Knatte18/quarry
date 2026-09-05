# Batch: expand-gate-and-sentinel

```yaml
task: "Glyph self-form and the resolve contract (C1)"
batch: "expand-gate-and-sentinel"
number: 3
cards: 4
verify: go test ./internal/engine/... ./quarry/...
depends-on: [2]
```

## Batch Scope

This batch adds the two engine-side values the CLI, the MCP server and `internal/repopath` need in
batch 4: `SelfGlyphError`, the typed failure `expand` returns for a self glyph (D11), and
`ErrTargetHasSeparator`, the sentinel `repoRelTarget` will return for a `"#"`-bearing path target
(D8). Both are aliased through the facade in the same batch so `errors.As` and `errors.Is` stay
transitive for an external caller, which is the whole point of declaring them here rather than in
the packages that raise them.

It depends on batch 2, and through it on batch 1 for `Glyph.IsSelf`. The files it edits are disjoint
from batch 2's, so the dependency is sequencing rather than a data dependency: both batches share the
same `verify:` scope over `internal/engine` and `quarry`, and running them in sequence keeps each
batch's verify a statement about that batch alone.

The external interface batch 4 consumes: `quarry.ErrTargetHasSeparator` and
`quarry.SelfGlyphError`.

Batch-local decision beyond the overview's Shared Decisions: `ErrTargetHasSeparator` is declared in
`internal/engine/repo.go` beside `ErrTargetOutsideRepo` and `ErrTargetNotFound`, not in
`internal/engine/errors.go`. That file's header reads that it holds the one error sentinel the
engine's subpackages share; a second sentinel there would falsify the header and force an unrelated
rewrite, while `repo.go` already houses the two target sentinels and needs no header change at all.

## Cards

### Card 19: `expand` of a self glyph is a typed failure

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `glyph/glyph.go`
- **Edits:**
  - `internal/engine/expand.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `SelfGlyphError` to `internal/engine/expand.go`, beside `NotATypeError` and
  in the same shape: a struct with an `ID string` field and an `Error` method returning
  `engine: expand <id>: not a type, a self glyph names the unit or file itself`. In the unexported
  `expand` worker, return the zero `ExpandAnswer` and a `*SelfGlyphError` carrying `g.String()`
  immediately after `glyph.Parse` succeeds and before `m.dirsOf` and `m.symbolsOf` are called, when
  `g.IsSelf()` reports true — a gate before resolution means no directory is read for a question that
  has no answer, which is what makes the memo's `parses` counter observably zero for this case.
  `codeForExpandError` in `internal/cli` must map it to the negative exit; that mapping is batch 4's
  card and is not written here. The type's doc comment records why this is a typed error rather than
  a sixth `Kind` value: `Kind` is documented as the closed vocabulary a `Symbol`'s kind is drawn from
  and the five values `toc` emits, a self glyph is not a symbol, and widening a symbol vocabulary to
  describe a non-symbol would falsify that comment. It also records why the answer is not
  `not_found`: a self glyph matches no symbol, so the answer would fall out as `not_found`, which
  says the name is free when the truth is that the verb does not apply. Do not change
  `NotATypeError`, the kind gate, or any other line of the worker.
- **Commit:** `feat(engine): reject expand of a self glyph with a typed error`

### Card 20: the target-separator sentinel

- **Context:**
  - `internal/engine/errors.go`
- **Edits:**
  - `internal/engine/repo.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare `ErrTargetHasSeparator` in `internal/engine/repo.go`, immediately after
  `ErrTargetNotFound`, with the text `engine: target contains the glyph separator "#"` — the
  `engine: target ...` style its two neighbours already use. Its doc comment states that it is
  returned when any segment of a cleaned repository-relative path target contains a `"#"`, that
  callers wrap it the same way the other two are wrapped so `errors.Is` still succeeds after
  wrapping, and that the raiser is `internal/repopath`'s `repoRelTarget` rather than `resolveTarget`.
  Do not declare it in `internal/engine/errors.go`: that file's header reads that it holds the one
  error sentinel the engine's subpackages share, and a second sentinel there would falsify the
  header. Do not add the reject itself here — `resolveTarget` is unchanged by this task, and
  `internal/repopath` is where both path-taking callers already normalise.
- **Commit:** `feat(engine): add the ErrTargetHasSeparator sentinel`

### Card 21: alias both values through the facade

- **Context:**
  - `internal/engine/repo.go`
  - `internal/engine/expand.go`
- **Edits:**
  - `quarry/quarry.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `type SelfGlyphError = engine.SelfGlyphError` beside the existing
  `NotATypeError` alias, so `errors.As` against `*quarry.SelfGlyphError` succeeds for a caller that
  never imports the engine — the same transitivity argument that alias's own doc comment already
  makes. Add `var ErrTargetHasSeparator = engine.ErrTargetHasSeparator` beside `ErrTargetOutsideRepo`
  and `ErrTargetNotFound`, as the same value and not a copy, which is what keeps `errors.Is`
  transitive. Both get doc comments in the style of their neighbours, citing the same reason.
- **Commit:** `feat(quarry): alias SelfGlyphError and ErrTargetHasSeparator`

### Card 22: `expand_test.go` — the self gate and its no-read proof

- **Context:**
  - `internal/engine/expand.go`
  - `internal/engine/resolve.go`
  - `glyph/errors.go`
- **Edits:**
  - `internal/engine/expand_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a case asserting that expanding a self glyph naming an existing package
  directory returns the zero `ExpandAnswer` and an error that `errors.As` reaches as a
  `*SelfGlyphError` whose `ID` is the argument with its trailing `"#"` intact. Assert that no
  directory was read, by constructing a `unitMemo` with `newUnitMemo`, passing it to the unexported
  `expand` worker, and reading its `parses` counter as zero afterwards — that counter is the existing
  seam for exactly this kind of claim, and it is what proves the gate sits before the directory work
  rather than after it. Assert the error's message text too, so the sentence is pinned rather than
  only its type. Leave the existing `TestExpand_MalformedTarget` rows alone: the no-separator row
  still asserts `ReasonNoSeparator`, which batch 1 did not change, and its doc comment's claim that
  `expand` writes no separator check of its own is still true and becomes truer in batch 4.
- **Commit:** `test(engine): pin the self-glyph expand gate and its zero reads`

## Batch Tests

`verify: go test ./internal/engine/... ./quarry/...` runs both packages this batch touches. The
new coverage is in `internal/engine/expand_test.go`; `quarry`'s own tests exercise the two aliases
only by compiling against them, which is the whole of what an alias can be wrong about — an alias
that names a different value fails to compile, and one that copies rather than aliases would break
`errors.Is`, which card 22's `errors.As` assertion and batch 4's `errors.Is` assertions cover
between them.

`internal/cli` stays outside this batch's scope for the same reason it stays outside batch 2's: its
behavioural resolve rows are red until batch 4 rewrites them, and including the package here would
make this batch's `verify:` fail for a reason this batch does not own.
