# Batch: facade-surface

```yaml
task: "Facade + CLI, resolve + expand (T5b)"
batch: "facade-surface"
number: 1
cards: 6
verify: go test ./quarry/
depends-on: []
```

## Batch Scope

This batch gives the `quarry/` facade the whole public surface for the two new verbs: the type aliases and status constants that make the engine's answer shapes nameable from outside the module, the two delegating query methods, and the four new renderers. It is one batch because every card is a small edit to one of five files that already exist and that all share one byte contract — the JSON encoder configuration and the text grammar — and because the renderers cannot be written without the aliases and cannot be tested without the methods.

The external interface the next two batches consume is exactly: `quarry.ResolveResult`, `quarry.ExpandAnswer`, `quarry.Status` with its four constants, `quarry.NotATypeError`, `(*quarry.Repo).Resolve`, `(*quarry.Repo).Expand`, `quarry.RenderResolveJSON`, `quarry.RenderExpandJSON`, `quarry.RenderResolveText`, `quarry.RenderExpandText`.

Batch-local decisions beyond the overview's Shared Decisions:

- The three existing exported renderers keep their exact current signatures and byte behaviour. The MCP transport task is being written against them in parallel, so a signature change here would break work in flight.
- No slice renderer is written. The CLI renders one result, never the slice, and nothing else needs one.
- The text-view grammar for the two new verbs is fixed to the character by cards 4 and 5, and every rule those cards do not state is inherited from the existing helpers in `quarry/text.go`: prose normalisation applied to every signature, doc and error string; no trailing whitespace on any line; the returned string ends with exactly one newline.

## Cards

### Card 1: alias the resolve/expand answer types and the four status constants

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
- **Edits:**
  - `quarry/quarry.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add four type aliases and one constant block to `quarry/quarry.go`, in the same style as the existing `DirAnswer`, `FileEntry`, `Symbol`, `Kind` and `TOCOptions` aliases and the existing five-value `Kind` constant block:

  ```go
  type ResolveResult = engine.ResolveResult
  type ExpandAnswer = engine.ExpandAnswer
  type Status = engine.Status
  type NotATypeError = engine.NotATypeError

  const (
      StatusFound     = engine.StatusFound
      StatusNotFound  = engine.StatusNotFound
      StatusAmbiguous = engine.StatusAmbiguous
      StatusMultipart = engine.StatusMultipart
  )
  ```

  Each alias carries its own doc comment stating, as the existing aliases do, that Go enforces the internal-package rule on import paths and not on types reached through an alias, so an external importer can name the type without importing the engine. The `Status` alias's comment states additionally that it is the closed per-entry vocabulary both result types draw from. The constant block carries one doc comment above it plus one comment per constant naming what that status means, mirroring the existing `Kind` block's shape.

  `NotATypeError` is aliased rather than re-declared so that `errors.As(err, &notType)` against `*quarry.NotATypeError` succeeds for a caller that never imports the engine — the same transitivity argument the existing sentinel variables' comments make for `errors.Is`. Say so in its doc comment.

  Add no new import: the file already imports the engine.
- **Commit:** `feat(quarry): alias ResolveResult, ExpandAnswer, Status and NotATypeError`

### Card 2: delegate Resolve and Expand through the facade

- **Context:**
  - `internal/engine/resolve.go`
  - `internal/engine/expand.go`
  - `quarry/quarry.go`
  - `quarry/scratchtree_test.go`
- **Edits:**
  - `quarry/repo.go`
  - `quarry/repo_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add two methods to `quarry/repo.go` beside the existing `TOC`:

  ```go
  func (r *Repo) Resolve(targets []string) ([]ResolveResult, error)
  func (r *Repo) Expand(target string) (ExpandAnswer, error)
  ```

  Each is a one-line delegation to `r.engine.Resolve` and `r.engine.Expand` respectively, returning the engine's own value and the engine's own error unchanged — no filtering, no re-shaping, no defaulting — exactly as `TOC` does today.

  `Resolve` keeps the engine's multi-target signature. The one-target rule this task's command line follows is the command line's rule, not the facade's: a Go caller batches many glyphs in one call and pays one parse per unit, which is the performance property the facade exists to preserve.

  Each method's doc comment states what it answers, that it returns the engine's own value and error unchanged, and — for `Expand` — that `errors.As(err, &notType)` against `*NotATypeError` therefore succeeds for a caller that never imports the engine.

  In `quarry/repo_test.go`, add tests beside the existing `TOC` tests, built with the same `writeScratchTree` helper those tests already use — but with one difference from their fixtures that this card must not get wrong: every Go file these new cases query must live in a **nested package directory**, not directly at the fixture root. The existing `TOC` fixtures put their Go files at the root, and a file directly under the root has the empty unit, which no glyph can spell; a glyph case built that way could never resolve. Name the nested directory in the fixture and spell every glyph against it.

  The cases:

  - `Resolve` over a glyph naming a free function in the fixture returns one result whose status is `StatusFound` and whose symbols hold that one declaration.
  - `Resolve` over a glyph whose member does not exist returns one result whose status is `StatusNotFound` and whose unit is `StatusFound`.
  - `Resolve` over a repository-relative path target returns one result whose status is `StatusFound` and whose directory answer is non-nil.
  - `Resolve` over a two-element target slice returns exactly two results, positionally.
  - `Expand` over a glyph naming a type returns `StatusFound` with a non-nil head.
  - `Expand` over a glyph naming a free function returns a non-nil error for which `errors.As(err, &notType)` against `*NotATypeError` succeeds, and the extracted value's `ID` and `Kind` fields are readable without importing the engine. This is the transitivity test, the analogue of the existing sentinel tests in this file.
- **Commit:** `feat(quarry): delegate Resolve and Expand to the engine`

### Card 3: one shared JSON encoder, two new exported JSON renderers

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
- **Edits:**
  - `quarry/render.go`
  - `quarry/render_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `quarry/render.go`, extract the encoder configuration currently inside `RenderJSON` into a new unexported helper:

  ```go
  func renderJSON(v any) ([]byte, error)
  ```

  It builds a `bytes.Buffer`, creates a `json.Encoder` over it, calls `SetEscapeHTML(false)` and `SetIndent("", "  ")`, encodes `v`, and returns the buffer's bytes. Rewrite `RenderJSON` to be a one-line delegation to it. `RenderJSON`'s exported signature, doc comment content and emitted bytes are unchanged; move the encoder-rationale prose into `renderJSON`'s own doc comment and leave `RenderJSON`'s comment naming what it renders and pointing at the shared helper for the byte contract.

  Add two exported renderers, each a one-line delegation to `renderJSON`:

  ```go
  func RenderResolveJSON(r ResolveResult) ([]byte, error)
  func RenderExpandJSON(a ExpandAnswer) ([]byte, error)
  ```

  Each carries a doc comment stating that it emits the same byte contract as `RenderJSON` — two-space indent, one trailing newline, no HTML escaping — and that key order within the object is the answer struct's own field declaration order, so no hand-written marshaller is needed.

  Do not change `RenderErrorJSON`: it deliberately uses a compact, unindented encoder, and routing it through the shared helper would change its bytes.

  Do not add a renderer taking a slice of results, and do not generalise `RenderJSON` to accept `any`: per-type exported renderers over one shared unexported encoder is the shape, so a type the plan's key set does not cover can never reach a renderer contracted to emit it.

  In `quarry/render_test.go`, add a table for each new renderer covering the payload shapes the two verbs actually produce: a found result, a multipart result, an ambiguous result carrying candidates, a not-found result with each unit value, a pre-resolution rejection carrying error and reason, and a path result carrying a directory answer; for expand, a found answer with head and members, a found answer with a head and no members, a not-found answer, and an ambiguous answer. Assert the same byte contract the existing `RenderJSON` tests assert — two-space indent, exactly one trailing newline, no HTML escaping, key order equal to struct declaration order, and no `ok` key — reusing the existing `assertKeyOrder` helper. For the not-found cases, assert that the symbols and candidates keys are absent rather than present as null or as an empty array.
- **Commit:** `refactor(quarry): share one JSON encoder and add the resolve/expand renderers`

### Card 4: the symbol line gains a file prefix, with a no-regression golden

- **Context:**
  - `internal/engine/answer.go`
- **Edits:**
  - `quarry/text.go`
  - `quarry/text_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `quarry/text.go`, change `writeSymbolLine` so it emits a leading `sym.File + ":"` before the start-end span, and only when `sym.File != ""`. Everything after that prefix is unchanged, so the line's full shape becomes:

  ```
  <file>:<start>-<end>[ (sig <start>-<sigend>)] <id>: <signature>
  ```

  with the optional following doc line, indented four spaces, unchanged.

  Extend `writeSymbolLine`'s doc comment to state the new prefix and why the change is invisible to the existing table-of-contents view: inside a toc answer the symbol's file field is always empty, because the symbol already sits in its own file's entry, so this is one grammar with one implementation rather than two.

  Make no other change to `quarry/text.go` in this card.

  In `quarry/text_test.go`, add two tests:

  - A regression golden asserting that rendering a directory answer whose symbols carry an empty file field produces a byte-identical string to the one the current implementation produces. Write it as a fixed expected string in the test, not as a comparison against a stored file.
  - A test that a symbol carrying a non-empty file field renders with the `<file>:` prefix, covering both the with-signature-end and without-signature-end forms.
- **Commit:** `feat(quarry): prefix a symbol line with its file when the symbol carries one`

### Card 5: the two text renderers

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `internal/engine/expand.go`
  - `quarry/quarry.go`
- **Edits:**
  - `quarry/text.go`
  - `quarry/text_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add two exported renderers to `quarry/text.go`, beside the existing `RenderText` and reusing the file's existing `dirBlocks`, `normalizeProse` and `writeSymbolLine` helpers rather than restating any of them:

  ```go
  func RenderResolveText(r ResolveResult) string
  func RenderExpandText(a ExpandAnswer) string
  ```

  Neither can fail and neither returns an error. Each returns a string with no trailing whitespace on any line, ending with exactly one newline.

  `RenderResolveText` has three branches, tested in this order:

  1. `r.Status == ""` — a pre-resolution rejection carried by the error field. One line and nothing else: the target as given, then a space, then the word `error`, then a space and the reason word only when the reason field is non-empty, then a colon, a space, and the normalised error string. With an empty reason this degenerates to `<target> error: <message>`. The identifier field is empty on this branch, which is why the line names the target rather than the identifier.
  2. `r.ID != ""` — a glyph target. On a not-found status, line 1 is the identifier, a space, the word `not_found`, a space, and the parenthesised clause `(unit found)` or `(unit not_found)` taken from the unit field, and nothing follows. On any other status, line 1 is the identifier, a space, and the status word; then one symbol line per entry, in order — the symbols slice for found and multipart, the candidates slice for ambiguous.
  3. otherwise — a path target. Line 1 is the target as given, a space, and the status word. On found, the directory-form block for the result's directory answer follows, starting on the line immediately after line 1, produced by joining `dirBlocks` of that answer with a newline — the same join the existing `RenderText` uses for its directory form. On not-found nothing follows, and no unit clause is emitted, because a path belongs to no unit.

     The directory answer is a pointer field, so guard it: when it is nil, emit line 1 alone and nothing else, whatever the status. That is unreachable for a value the engine produces — it always sets the pointer alongside a found path status — and it is spelled for the same reason branch 4 of the expand renderer below is: this is an exported renderer, an external caller can construct a found result with no directory answer, and dereferencing nil there would panic rather than honour the one-trailing-newline promise this renderer makes for every input.

  The directory form is used for a path result even when the target names a file: the engine answers a file path target with its enclosing directory's answer holding exactly one file entry, which the directory form renders losslessly. No file-versus-directory flag is plumbed into this renderer.

  `RenderExpandText` has four branches:

  1. not-found — one line: the identifier, a space, `not_found`, a space, and `(unit found)` or `(unit not_found)`; nothing follows.
  2. found — line 1 is the identifier, a space, `found`; then the head's symbol line; then, only when the members slice is non-empty, one blank line followed by one symbol line per member in order. The blank line is the same block separator `dirBlocks` already uses, so no new marker is invented and the head stays distinguishable from the members without a key.
  3. ambiguous — line 1 is the identifier, a space, `ambiguous`; then one symbol line per candidate in order, with no blank line.
  4. otherwise — a fall-through for any other status value, including the empty status of a zero answer and the multipart status this answer type does not admit. Emit one line: the identifier, a space, and the status word as it stands, then nothing. This branch is unreachable for any value the engine produces, and it exists for the same reason card 10's mappings spell an unreachable default: this is an exported renderer, an external caller can hand it a zero value, and the promise that the returned string ends with exactly one newline must hold for every input rather than for the three the engine happens to produce. State that in the doc comment.

  There is no text rendering of a not-a-type failure: that case takes the error path and produces no payload, and the existing rule that there is no text rendering of a failure already covers it.

  In `quarry/text_test.go`, add a table per renderer covering every branch listed above as an exact expected string, and assert the no-trailing-whitespace-and-one-newline property with the file's existing `assertNoTrailingWhitespaceAndOneNewline` helper. Include the expand renderer's fall-through branch in that table, driven by a zero answer value, and the resolve renderer's nil-directory-answer guard, driven by a found path result whose pointer is nil, so the one-newline promise is pinned for both. Include a path-target found case whose directory answer holds exactly one file entry — the shape the engine produces for a file path target — so the claim that a file target is rendered with the directory form is pinned here rather than only by the evidence goldens. These tests are written before the renderers: the grammar is fully specified above.
- **Commit:** `feat(quarry): add RenderResolveText and RenderExpandText`

### Card 6: describe the widened facade surface

- **Context:**
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `quarry/render.go`
  - `quarry/text.go`
- **Edits:**
  - `quarry/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Update the package doc comment in `quarry/doc.go` so it describes the surface as it now stands rather than the table-of-contents-only surface it describes today. It must state:

  - that the package now exposes three query methods, not one, and that the two new ones delegate to the engine unchanged exactly as the existing one does;
  - that the package owns seven renderers: the two existing JSON ones, the two new JSON ones, and the three text ones, with the three JSON success renderers sharing one encoder configuration;
  - that the `ok` key in the failure envelope marks that quarry could not answer, and never that the answer is negative — a negative resolution outcome is a payload with a status word, rendered by the ordinary renderer.

  Leave the existing statement about the phase-1 non-goals — no cache, no parser pool, no state beyond the repository root — in place and unchanged.
- **Commit:** `docs(quarry): describe the resolve and expand surface`

## Batch Tests

`verify: go test ./quarry/` runs the whole `quarry` package's test binary: `quarry/render_test.go`, `quarry/text_test.go`, `quarry/repo_test.go` and the shared `quarry/scratchtree_test.go` helper. That is the exact set of files this batch touches or depends on, and the package is small enough that scoping further would buy nothing.

The batch's own proof that the shared-helper refactors changed nothing is card 4's regression golden plus the existing `RenderJSON` and `RenderText` tests continuing to pass unchanged. The four committed table-of-contents evidence goldens are the second half of that proof and are re-compared in batch 4, which is where they are next exercised.
