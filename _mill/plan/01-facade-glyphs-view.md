# Batch: facade-glyphs-view

```yaml
task: 'P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)'
batch: 'facade-glyphs-view'
number: 1
cards: 5
verify: go test ./quarry/
depends-on: []
```

## Batch Scope

This batch delivers the whole glyphs view as facade code, with no CLI surface at all: the new
`quarry/view.go` file holding the projected answer type, the pure projection over a complete
`DirAnswer`, and the view's two renderers; the `Glyphs` query method and its frozen options on
`Repo`; and the package doc comment's updated counts. Nothing under `internal/engine/` and nothing
under `internal/cli/` is touched. It is one batch because the four code cards are one file plus one
short method, they share one set of read dependencies (`internal/engine/answer.go`'s tag
spellings, `quarry/text.go`'s `joinRel` and existing line grammar, `quarry/render.go`'s
`renderJSON`), and the projection cannot be reviewed sensibly apart from the two renderers that
consume it.

The external interface batch 2 consumes is exactly four exported names:
`quarry.GlyphsAnswer`, `quarry.GlyphView`, `quarry.RenderGlyphsJSON`, `quarry.RenderGlyphsText`,
plus `quarry.GlyphsOptions` for batch 2's drift test and `(*quarry.Repo).Glyphs` for Go callers.

Batch-local decisions beyond `## Shared Decisions` in the overview: none. Every decision this batch
implements is either in the overview or in `_mill/discussion.md`'s `## Decisions`.

## Cards

### Card 1: the projected answer type and the pure projection

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
  - `quarry/text.go`
  - `quarry/scratchtree_test.go`
  - `quarry/render_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/view.go`
  - `quarry/view_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/view.go` with a file-level comment in this package's existing
  register, stating that the file holds the glyphs view — the flat, non-recursive projection of a
  complete table-of-contents answer, plus that view's own two renderers — and that the projection
  is applied after the query returns so extraction underneath stays complete and `internal/engine/`
  is never reached by a view concept.

  Declare `GlyphsAnswer`, exactly as the discussion's `glyphs-answer-shape` decision fixes
  it: three fields, `Target string`, `Symbols []Symbol`, `Incomplete []string`, and **no JSON
  struct tags on any of them**. Its doc comment must say why there are no tags: the wire shape is
  the unexported envelope card 2 adds, so there is exactly one description of the emitted key set,
  and an untagged struct makes a caller's direct `json.Marshal` of the answer visibly a different
  document rather than a plausible one.

  Declare `func GlyphView(target string, a DirAnswer) GlyphsAnswer`. It is pure over `a`: it reads
  `a` and returns a new value, and must not mutate `a` or share a backing array with any slice
  inside it. Behaviour:
  - `Target` is `target` verbatim, echoed with no normalisation of any kind. The doc comment states
    that the callers normalise — the CLI passes the already-repository-relative form, and `Glyphs`
    maps an empty target to `"."` before both its query and its echo — so one query never has two
    spellings in its own answer.
  - `Symbols` is every symbol in `a`'s whole tree in depth-first order, matching `dirBlocks`'s own
    order in `quarry/text.go`: this directory's `Files` in slice order, each file's `*Symbols` in
    slice order, then each entry of `Dirs` in slice order, recursively. A `FileEntry` whose
    `Symbols` pointer is nil contributes nothing and is not an error; a non-nil pointer to an empty
    slice likewise contributes nothing.
  - Each contributed symbol is a copy with `Doc`, `Signature` and `SigEnd` set to their zero values
    and `File` set to `joinRel(<enclosing DirAnswer's Dir>, <enclosing FileEntry's Name>)`, reusing
    the existing unexported `joinRel` in `quarry/text.go` rather than re-deriving the rule. Every
    other field, `Glyph` included, is carried across untouched.
  - `Incomplete` is the `joinRel`-joined path of every `FileEntry` in the whole tree whose `Error`
    is non-empty or whose `Lossy` is true, sorted with `sort.Strings` (or `slices.Sort`), and left
    nil when there are none. The doc comment states the invariant and its scope, per the discussion's
    `incomplete-is-explicit` decision: "an absent symbol line means the symbol is not in the target"
    holds for the frozen `glyphs` preset, which is `--depth all`, and there only — a depth-cut
    answer is truncated by construction and contributes nothing to `Incomplete`, because a
    depth-cut `DirAnswer` with no `Files` and no `Dirs` is indistinguishable from a genuinely empty
    leaf directory and this function has nothing to detect.

  Create `quarry/view_test.go` with a file-level comment naming what it covers, and table tests for
  `GlyphView` over hand-built `DirAnswer` values only — no filesystem, no `Open`, no `TOC`, matching
  `quarry/render_test.go`'s own posture. Cases: a single-file answer; an answer nested two
  directories deep, asserting the depth-first order; a file entry with a nil `Symbols`; a file entry
  with a non-nil pointer to an empty slice; entries with `Lossy` set and with `Error` set, both
  landing in `Incomplete` and the result sorted; `File` filled through `joinRel` including the
  `Dir == "."` root case, where the joined path is the bare file name; `Doc`, `Signature` and
  `SigEnd` cleared on every returned symbol while a non-empty input `Signature` is left untouched in
  the input value; and an explicit no-mutation case that builds an input, calls `GlyphView`, and
  asserts the input's own symbols still carry their original `Doc`/`Signature`/`SigEnd`/`File`.
- **Commit:** `feat(quarry): GlyphsAnswer and the pure GlyphView projection`

### Card 2: the glyphs view's JSON renderer and its shadow envelope

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
  - `quarry/render_test.go`
- **Edits:**
  - `quarry/view.go`
  - `quarry/view_test.go`
  - `quarry/render.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add to `quarry/view.go` the two unexported types and the one exported renderer
  the discussion's `glyphs-json-shadow-struct` decision fixes, verbatim in field set, field
  order and tag spelling:

  `glyphSymbol`, with `ID string` tagged `json:"id"`, `Kind Kind` tagged `json:"kind"`, `File
  string` tagged `json:"file"`, `Start int` tagged `json:"start"`, and `End int` tagged
  `json:"end"` — five keys, in `Symbol`'s own declaration order, and note that `File` here carries
  **no** `omitempty`, unlike `Symbol.File`'s own tag, because in this view the field is always
  filled.

  `glyphsEnvelope`, with `Target string` tagged `json:"target"`, `Symbols []glyphSymbol` tagged
  `json:"symbols"` with **no** `omitempty`, and `Incomplete []string` tagged
  `json:"incomplete,omitempty"`.

  `func RenderGlyphsJSON(a GlyphsAnswer) ([]byte, error)`, which builds one `glyphsEnvelope` —
  copying `Target` and `Incomplete` across, allocating `make([]glyphSymbol, 0, len(a.Symbols))` so
  a zero-symbol answer emits `"symbols": []` and never `"symbols": null`, and mapping each `Symbol`
  into a `glyphSymbol` — and encodes it through the existing unexported `renderJSON` helper in
  `quarry/render.go`, never through its own `json.Encoder`, so the two-space indent,
  no-HTML-escaping, single-trailing-newline byte contract cannot drift from the other success
  renderers'.

  The doc comments must state the reasoning, not just the shape: that `Symbol.Signature` is tagged
  `json:"signature"` with no `omitempty` in `internal/engine/answer.go`, so clearing the field in
  `GlyphView` would still emit `"signature": ""` and the promised key set is unreachable by
  clearing fields alone; that adding `omitempty` there was rejected because it would change a key
  set three verbs share and that file declares closed; and that a custom `MarshalJSON` was rejected
  because `quarry/render.go`'s own header records that these types deliberately carry no methods.

  `quarry/render.go` is edited by this card for one reason only: its file header enumerates the
  renderers the facade exports and calls them "the four successful envelopes, sharing one
  unexported encoder configuration in renderJSON". After this card there is a fifth success
  renderer sharing that same configuration, declared in `quarry/view.go` rather than here. Correct
  the enumeration and say where the fifth lives, so a reader of that header is not told the sharing
  set is closed at four. Change nothing else in the file — no renderer's code, no tag, no
  `renderJSON` behaviour.

  Add cases to `quarry/view_test.go`, asserting on the rendered bytes rather than a decoded map so
  key order is checkable: the envelope's key set and order (`target`, `symbols`, `incomplete`);
  `incomplete` absent when the slice is empty and present when it is not; a zero-symbol answer
  emitting `"symbols": []` as its own named case, since a nil slice would render `null` here and
  nothing else in the suite would catch it; each symbol object carrying exactly `id`, `kind`,
  `file`, `start`, `end` in that order; and one named case whose input symbol carried a **non-empty**
  `Signature` before `GlyphView` cleared it, asserting `signature`, `doc` and `sigend` are all
  absent from the rendered bytes — a symbol with an empty signature would prove nothing here. Also
  assert the shared byte contract: two-space indentation, exactly one trailing newline, and a `<`
  in a symbol id or target left unescaped.
- **Commit:** `feat(quarry): RenderGlyphsJSON and the glyphs view's shadow envelope`

### Card 3: the glyphs view's text renderer

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
  - `quarry/text.go`
  - `quarry/text_test.go`
- **Edits:**
  - `quarry/view.go`
  - `quarry/view_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `func RenderGlyphsText(a GlyphsAnswer) string` to `quarry/view.go`,
  implementing the discussion's `text-line-shape` and `incomplete-is-explicit` decisions
  exactly:
  - one line per entry of `a.Symbols`, in slice order, spelled
    `<File>:<Start>-<End> <Kind> <ID>` — file first, then a colon, the span with an ASCII hyphen,
    a space, the kind word, a space, the id, and a newline. No directory line, no file line, no
    header, no docstring, no signature, and no `(sig ...)` clause.
  - when `a.Incomplete` is non-empty, a single blank line follows the symbol lines, then one line
    per path spelled `[incomplete] <path>` in the slice's own order. When `a.Incomplete` is empty,
    neither the block nor its blank line is emitted.
  - the byte contract is this renderer's own, stated in its doc comment rather than borrowed from
    `RenderText` in `quarry/text.go`: no trailing whitespace on any line; every non-empty rendering
    ends with exactly one `"\n"`; and an answer with no symbols and no incomplete files renders as
    the empty string `""`, never as `"\n"`. The doc comment says why the contract is restated —
    `RenderText`'s own "ends with exactly one newline" rule cannot describe an empty rendering, and
    emitting a bare newline would put a blank line on a caller's stdout that says nothing.

  Do not reuse `writeSymbolLine` from `quarry/text.go` and do not add a suppression parameter to
  it: its grammar puts the span first with no file prefix inside a toc answer, and one function
  serving two grammars is what its own doc comment says it is not. Say so in a sentence in
  `RenderGlyphsText`'s doc comment.

  Add cases to `quarry/view_test.go`: the line grammar for each of the five kinds; an answer with
  no symbols and no incomplete files asserting exactly `""`; an answer with symbols and no
  incomplete files; an answer with no symbols but one or more incomplete files, asserting the block
  is present and that the leading blank line does not produce a leading empty line before any
  symbol line exists; an answer with both; and an assertion over every case that no line ends in
  whitespace and that a non-empty result ends in exactly one newline.
- **Commit:** `feat(quarry): RenderGlyphsText, the glyphs view's line grammar`

### Card 4: the Glyphs query and its frozen options

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
  - `quarry/view.go`
  - `quarry/scratchtree_test.go`
  - `quarry/repo_test.go`
- **Edits:**
  - `quarry/repo.go`
  - `quarry/view_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add to `quarry/repo.go`:

  `func GlyphsOptions() TOCOptions`, returning the frozen options the `glyphs` preset expands to —
  `Depth: DepthAll` and `Symbols` pointing to a `true`. It returns a fresh value on each call, so a
  caller cannot mutate a shared one through the returned pointer. Its doc comment states why it is
  exported rather than unexported: `internal/cli/`'s drift test parses the CLI's own preset tokens
  and compares the parse against this value, and that test cannot live in this package, because
  `internal/cli/` imports `quarry` and the reverse import would be a cycle — see the overview's
  `glyphs-options-is-exported` Shared Decision.

  `func (r *Repo) Glyphs(target string) (GlyphsAnswer, error)`, implemented as: map an empty
  `target` to `"."`; call `r.TOC(<normalised target>, GlyphsOptions())`; on a non-nil error return
  the zero `GlyphsAnswer` and that error unchanged, so `errors.Is(err, ErrTargetNotFound)` keeps
  working through it exactly as it does through `TOC`; otherwise return
  `GlyphView(<normalised target>, answer)`. Its doc comment states why the normalisation happens
  here and not in `GlyphView`: `TOC` accepts `""` and `"."` as the same query, so without it one
  query would have two spellings in its own answer's `target` field, and `GlyphView` is deliberately
  a verbatim echo of what it is handed.

  Add to `quarry/view_test.go` the facade-side drift assertion: build a small fixture tree with
  `writeScratchTree`, `Open` it, and assert `Glyphs(target)` is deep-equal to
  `GlyphView(target, <the answer from TOC(target, GlyphsOptions())>)` computed in the test, for a
  directory target. Add a second case asserting `Glyphs("")` and `Glyphs(".")` both return
  `Target == "."`. Add a third for the error branch, which nothing else in this plan reaches — the
  CLI's own glyphs path calls `TOC` and `GlyphView` directly and never calls this method, so if
  this case is not written the pass-through contract is untested everywhere: `Glyphs` on a target
  that does not exist in the fixture must return the zero `GlyphsAnswer` and an error satisfying
  `errors.Is(err, ErrTargetNotFound)`, which is the whole reason the method returns the engine's
  error unchanged rather than wrapping it. Do not add a Loomyard checkout gate to this package — see the overview's
  `the-facade-drift-test-uses-a-scratch-tree` Shared Decision.

  This card also makes `quarry/repo.go`'s own file header stale, and it is updated in this same
  card: the header says the file declares Repo, Open, "and the TOC query that delegates to the
  engine unchanged", which no longer describes a file that also declares a query composing TOC with
  a projection, and a frozen option value that exists to be compared against the CLI's preset. Say
  both, in the register the rest of that header uses.
- **Commit:** `feat(quarry): Repo.Glyphs and the exported GlyphsOptions`

### Card 5: the package doc comment's counts

- **Context:**
  - `quarry/render.go`
  - `quarry/repo.go`
  - `quarry/text.go`
  - `quarry/view.go`
  - `quarry/name.go`
- **Edits:**
  - `quarry/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** `quarry/doc.go`'s package comment currently says the package "exposes four
  queries, not one: TOC, Resolve, Expand and Name" and that it "owns nine renderers". Both counts
  are now wrong. Update the query sentence to five queries, naming `Glyphs` alongside the existing
  four and stating that it is a method for the same reason `TOC` is — it reads the repository —
  and that unlike the other three method queries it does not delegate to the engine unchanged: it
  is `TOC` under frozen options followed by a pure projection, which is the one place this package
  adds behaviour of its own rather than only re-shaping. Update the renderer sentence to eleven,
  adding `RenderGlyphsJSON` to the JSON group and `RenderGlyphsText` to the text group, and state
  that `RenderGlyphsJSON` shares the same encoder configuration as the other JSON success renderers
  while `RenderGlyphsText` states its own byte contract rather than the shared one, because an
  empty glyphs answer renders as the empty string.

  A third sentence in the same comment must change with them, and editing only the two counts would
  leave the file self-contradictory: the second paragraph opens by stating that the package adds no
  behaviour of its own, immediately before the sentence this card is adding `Glyphs` to. Rewrite
  that opening so the claim is scoped to what still holds, and be precise about what "the answer
  types" now covers: every type reached through the four delegating queries is an alias for an
  engine type, and those four add no filtering, re-shaping or defaulting — but `GlyphsAnswer` is
  this package's own defined struct, not an alias, so "the answer types are the engine's" would be
  stale the day it is written. Say "the engine's types, plus this package's one projected answer
  type". Do not weaken the sentence into vagueness: the no-added-behaviour posture is a real
  property of the other four queries and is why the aliases work at all.

  The same paragraph's renderer sentence carries a second claim this task falsifies: it calls the
  renderers "the only code it owns beyond the queries themselves", which `GlyphView`, `glyphSymbol`
  and `glyphsEnvelope` are not. Dispose of that clause explicitly rather than editing the count
  around it — the projection is the package's own code, and saying so is the honest form of the
  same point that sentence was making.
- **Commit:** `docs(quarry): update the package comment's query and renderer counts`

## Batch Tests

`verify: go test ./quarry/` runs this package's whole test binary, which is the correct scope: the
batch's four code cards all land in `quarry/view.go` and `quarry/repo.go`, and the new
`quarry/view_test.go` is where every assertion for them lives. The existing
`quarry/render_test.go`, `quarry/text_test.go`, `quarry/repo_test.go` and `quarry/name_test.go`
run alongside as the regression gate for this batch's one real regression risk — that a change to
a shared helper (`renderJSON`, `joinRel`) made to suit the new view breaks an existing renderer.
The package's tests take well under a second, so nothing here needs `-run` scoping.

No test in this batch needs a Loomyard checkout: every assertion is over hand-built `DirAnswer`
values or a `writeScratchTree` fixture, so the batch is fully verified on any machine.
