# Batch: layout

```yaml
task: "Engine core (T3)"
batch: "layout"
number: 1
cards: 7
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
depends-on: []
```

## Rename mechanic

For each `Moves:` pair the implementer MUST:

1. Run `git mv <old> <new>` FIRST, before making any other change to the moved file.
2. Make ONLY surgical edits — touch only the lines that must change after the move (package
   declaration, imports, identifier retargeting, doc-comment path references).
3. Use a full-file `Creates:` entry only for genuinely new files that have no predecessor.
4. Never write the relocated file from scratch and delete the original — that breaks git rename
   history and inflates review diffs.

## Batch Scope

This batch settles the package layout and removes the two features the rewrite deletes
outright, with **no change to any extraction rule**. After it, `internal/quarryengine` and its
`toc` subpackage are gone; there is one package `internal/engine`, one subpackage
`internal/engine/treesitter`, and a new dependency-only package `internal/cgoguard`. `TOCFile` and
`TOCDir` still exist with their current behaviour minus the doc-sentence option, and every ported
test still passes — that is this batch's whole safety argument, and it is why the answer-shape and
symbol-model rewrites are deliberately not in it.

The external interface the next batches consume is the package path
`github.com/Knatte18/quarry/internal/engine` and the file-per-concern layout inside it: batch 2
adds `ignore.go`/`headers.go` beside these files, batch 3 rewrites `answer.go`/`toc.go` and adds
`repo.go`/`walk.go`, and batch 4 rewrites `golang.go`'s walk.

Batch-local decision: `sentences_test.go` is deleted whole rather than partly ported. The
discussion says its `FirstParagraph` cases move to `text_test.go`, but on disk those cases live in
`comments_test.go` (which becomes `text_test.go` by rename), and `sentences_test.go` holds only
`TestFirstSentences`, whose subject this batch deletes.

## Cards

### Card 1: The cgo guard becomes its own package

- **Context:**
  - `internal/quarryengine/doc.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `internal/quarryengine/cgoguard.go` -> `internal/cgoguard/cgoguard.go`
  - `internal/quarryengine/cgoguard_nocgo.go` -> `internal/cgoguard/cgoguard_nocgo.go`
- **Requirements:** Change both files' package clause from `package quarryengine` to
  `package cgoguard`. Add a package doc comment to `cgoguard.go` stating that this
  package declares nothing, imports nothing, and exists only so that a `CGO_ENABLED=0` build fails
  with a readable message before the raw cgo linker error, and that it is blank-imported by
  `internal/engine/treesitter` so it is strictly earlier in the build graph than anything that links
  tree-sitter. Rewrite the two files' existing header comments the same way: the old rationale
  ("both files sit in this root package … because under CGO_ENABLED=0 the treesitter package itself
  cannot compile") is now wrong, because the engine package itself transitively imports cgo, so a
  guard left there would be unreachable exactly when it is needed. Keep the `//go:build cgo` and
  `//go:build !cgo` constraints, keep the "do not delete this file" warning, and keep the
  `var _ = quarry_requires_CGO_ENABLED_1_with_a_C_toolchain` line verbatim.
- **Commit:** `refactor(cgoguard): move the cgo build guard into its own package`

### Card 2: The treesitter subpackage moves under internal/engine

- **Context:**
  - `internal/cgoguard/cgoguard.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `internal/quarryengine/treesitter/treesitter.go` -> `internal/engine/treesitter/treesitter.go`
  - `internal/quarryengine/treesitter/treesitter_test.go` -> `internal/engine/treesitter/treesitter_test.go`
- **Requirements:** Keep `package treesitter`. Add the blank import
  `_ "github.com/Knatte18/quarry/internal/cgoguard"` to the import block of `treesitter.go`, with a
  comment saying it is what makes the guard a build-graph dependency of every package that links
  tree-sitter. Update the file header comment's reference to the old package path
  so it names the new one. `treesitter_test.go` needs no change beyond the move.
- **Commit:** `refactor(treesitter): move under internal/engine and depend on cgoguard`

### Card 3: The engine root files move and the two package comments merge

- **Context:**
  - `internal/quarryengine/toc/doc.go`
- **Edits:** none
- **Creates:** none
- **Deletes:**
  - `internal/quarryengine/toc/doc.go`
- **Moves:**
  - `internal/quarryengine/doc.go` -> `internal/engine/doc.go`
  - `internal/quarryengine/errors.go` -> `internal/engine/errors.go`
- **Requirements:** Change both files' package clause to `package engine`. Rewrite `doc.go`'s package comment as the merge of the two old ones: it names the one
  package that holds the extraction engine, states that `treesitter` is the parse-and-release seam
  and `cgoguard` the build guard, and states that this package returns typed results and typed
  errors only — it never emits JSON, never decides an exit code, and never resolves a caller's cwd.
  The merged comment must NOT carry the "# The sentence-boundary rule" section: `FirstSentences` is
  deleted in card 6. Keep `ErrLanguageUnsupported` and its doc comment unchanged in this card — it is
  rewritten in batch 5 when it gains its one caller.
- **Commit:** `refactor(engine): move the root package files and merge the package comments`

### Card 4: The toc sources move into internal/engine

- **Context:**
  - `internal/engine/doc.go`
  - `internal/engine/errors.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `internal/quarryengine/toc/types.go` -> `internal/engine/answer.go`
  - `internal/quarryengine/toc/toc.go` -> `internal/engine/toc.go`
  - `internal/quarryengine/toc/strategy.go` -> `internal/engine/strategy.go`
  - `internal/quarryengine/toc/golang.go` -> `internal/engine/golang.go`
  - `internal/quarryengine/toc/nodes.go` -> `internal/engine/nodes.go`
  - `internal/quarryengine/toc/classify.go` -> `internal/engine/classify.go`
  - `internal/quarryengine/toc/extension.go` -> `internal/engine/extension.go`
  - `internal/quarryengine/toc/comments.go` -> `internal/engine/text.go`
- **Requirements:** Change every moved file's package clause from `package toc` to `package engine`.
  In `toc.go`, drop the `"github.com/Knatte18/quarry/internal/quarryengine"` import
  and refer to `ErrLanguageUnsupported` unqualified, and change the
  `"github.com/Knatte18/quarry/internal/quarryengine/treesitter"` import to
  `"github.com/Knatte18/quarry/internal/engine/treesitter"`. In the `Register` panic message in `strategy.go`, change the `"toc: duplicate Strategy registration for language "`
  prefix to `"engine: "`. Update each moved file's own header comment where it names the old package
  or the old path — `answer.go`'s header still says "types.go declares the toc
  package's result and option types" and `text.go`'s still says "comments.go holds
  …". The two remaining sources under the old directory, which card 6 deletes outright, are not
  moved and are not touched by this card.
- **Commit:** `refactor(engine): move the toc sources into the engine package`

### Card 5: The tests move with their subjects

- **Context:**
  - `internal/engine/toc.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `internal/quarryengine/toc/toc_test.go` -> `internal/engine/toc_test.go`
  - `internal/quarryengine/toc/golang_test.go` -> `internal/engine/golang_test.go`
  - `internal/quarryengine/toc/classify_test.go` -> `internal/engine/classify_test.go`
  - `internal/quarryengine/toc/comments_test.go` -> `internal/engine/text_test.go`
  - `internal/quarryengine/toc/extension_test.go` -> `internal/engine/extension_test.go`
  - `internal/quarryengine/toc/toc_integration_test.go` -> `internal/engine/toc_integration_test.go`
- **Requirements:** Change every moved test file's package clause to `package engine`. Retarget the
  imports the moves invalidate: `toc_test.go` imports both
  `"github.com/Knatte18/quarry/internal/quarryengine"` and
  `"github.com/Knatte18/quarry/internal/quarryengine/treesitter"`, and `golang_test.go` imports the
  latter — neither package path exists after cards 2 and 3. Drop the first import from
  `toc_test.go` and refer to `ErrLanguageUnsupported` unqualified at its one use site; change the
  second to `"github.com/Knatte18/quarry/internal/engine/treesitter"` in both files. In
  `toc_integration_test.go`: the module-root climb from `runtime.Caller(0)` is now
  three `filepath.Dir` calls rather than four, because the file moved up one directory level; the
  target path becomes `internal`, `engine`, `treesitter`, `treesitter.go`; and the header comment's
  reference to the old path is updated. Its loose assertions on symbol names, kinds and range
  ordering stay loose — do not tighten them.
- **Commit:** `refactor(engine): move the package tests alongside their subjects`

### Card 6: Delete the compact view and doc-sentence trimming

- **Context:**
  - `internal/engine/text.go`
  - `internal/engine/doc.go`
- **Edits:**
  - `internal/engine/answer.go`
  - `internal/engine/toc.go`
  - `internal/engine/toc_test.go`
  - `internal/engine/toc_integration_test.go`
- **Creates:** none
- **Deletes:**
  - `internal/quarryengine/toc/compact.go`
  - `internal/quarryengine/toc/compact_test.go`
  - `internal/quarryengine/toc/sentences.go`
  - `internal/quarryengine/toc/sentences_test.go`
- **Moves:** none
- **Requirements:** Delete the four files above. From `internal/engine/answer.go` delete the
  `AllSentences` constant and the `Options` type. From `internal/engine/toc.go` delete
  `applyDocSentences` and change `TOCFile`'s signature to `TOCFile(path string, langOverride string)
  (FileTOC, error)`, dropping the `opts` parameter and the `applyDocSentences` call; rewrite the part
  of `TOCFile`'s doc comment that documents `opts.DocSentences` so it states instead that a symbol's
  `Docstring` is always the complete, untrimmed docstring. From `internal/engine/toc_test.go` delete
  `TestTOCFile_DocSentencesPolicy`,
  `TestTOCFile_DocSentencesDoesNotSplitOnAbbreviationOrBacktickIdentifier` and
  `TestTOCFile_RangesStableAcrossDocSentences`, and update every remaining `TOCFile` call site to the
  two-argument form. Update `internal/engine/toc_integration_test.go`'s single `TOCFile` call the
  same way; its `Docstring` assertion stays as it is. `FirstParagraph` in `internal/engine/text.go`
  is kept and is not touched by this card.
- **Commit:** `refactor(engine): delete the compact view and doc-sentence trimming`

### Card 7: README names the three real queries

- **Context:**
  - `docs/rewrite-plan.md`
- **Edits:**
  - `README.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Replace the stale verb list "`map`, `resolve`, `members`" with "`toc`,
  `resolve`, `expand`". Change nothing else in the file.
- **Commit:** `docs(readme): name the three queries the rewrite actually has`

## Batch Tests

`verify:` runs `CGO_ENABLED=1 go build ./...` followed by `CGO_ENABLED=1 go test ./internal/...`,
which covers `internal/engine`, `internal/engine/treesitter` and `internal/cgoguard`. `glyph/` is
deliberately outside the scope: this task never modifies it.

No test is added in this batch. Its whole verification is that the ported tests —
`toc_test.go`, `golang_test.go`, `classify_test.go`, `text_test.go`, `extension_test.go`,
`toc_integration_test.go` and `treesitter/treesitter_test.go` — still pass unchanged apart from the
`TOCFile` signature and the deleted doc-sentence cases. A behavioural difference introduced by the
move would show up there.

`go build ./...` is part of `verify:` rather than left to the test run because the cgo guard
(`internal/cgoguard`) has no test files at all: a build is the only thing that exercises it.
