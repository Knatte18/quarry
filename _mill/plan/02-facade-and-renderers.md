# Batch: facade-and-renderers

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
batch: 'facade-and-renderers'
number: 2
cards: 5
verify: go test ./quarry/
depends-on: [1]
```

## Batch Scope

This batch exposes the maker through the public facade: the type and constant aliases that make the
engine's new shapes nameable from outside the module, the package-level batch entry point, and the
two renderers the CLI will call. It is one batch because every card is a small edit to one small
package that adds no behaviour of its own — the facade delegates, and the renderers delegate to the
one encoder configuration that already exists. The external interface batch 3 consumes is
`quarry.Name`, `quarry.Declaration`, `quarry.NameResult`, the four reason constants, and
`RenderNameJSON` / `RenderNameText`.

Batch-local decision: `Name` is a package-level function, not a `(*Repo)` method. The maker reads
nothing, so a `*Repo` receiver would claim a repository dependency the query does not have and would
force a caller holding only a fragment to first open a directory that has nothing to do with the
answer. The alias-types-carry-no-methods rule that governs the renderers is a separate matter and is
not the reason here: `Repo` is locally defined and carries methods perfectly well.

## Cards

### Card 3: alias the maker's shapes and reasons into the facade

- **Context:**
  - `internal/engine/name.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `quarry/quarry.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add to `quarry/quarry.go`, in the file's established alias style and with the same
  reached-through-an-alias rationale its existing doc comments state:

  `type Declaration = engine.Declaration` and `type NameResult = engine.NameResult`, each with a doc
  comment saying it is an alias, not a defined type, so an external importer can name the maker's
  input and answer shapes without importing the engine.

  A constant block aliasing the four maker-owned reason words —
  `NameReasonParse = engine.NameReasonParse` and the same for `NameReasonNoDeclaration`,
  `NameReasonSeveralDeclarations`, and `NameReasonInternal` — with a block doc comment matching the
  shape of the existing `Kind` and `Status` constant blocks in this file.

  A `NameReasons = engine.NameReasons` var, aliasing the enumerating slice as the same value rather
  than a copy, so a caller enumerating the vocabulary and the engine's own test are reading one
  slice. Its doc comment states that it is the engine's own value, not a copy, following the
  argument the error-sentinel vars in this file already make.

  `Kind` already has an alias here and needs none added; `NameResult.Kind` is that same type.
- **Commit:** `feat(quarry): alias the maker's Declaration, NameResult, and reason vocabulary`

### Card 4: the package-level Name entry point

- **Context:**
  - `internal/engine/name.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `quarry/doc.go`
- **Edits:** none
- **Creates:**
  - `quarry/name.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `quarry/name.go` with a file header comment in this package's style, declaring exactly one
  function: `func Name(decls []Declaration) []NameResult`, delegating to `engine.Name(decls)` and
  returning its result unchanged — no filtering, no re-shaping, no defaulting, matching the delegation
  posture every other query in this package already keeps.

  Its doc comment states three things. First, that it is package-level rather than a `Repo` method
  because the maker performs no I/O and a receiver would claim a repository dependency the query does
  not have. Second, that it returns no error: with no I/O nothing can fail batch-wide, every failure
  is a property of one entry and is carried in that entry's own `NameResult`, and a returned error
  would have no value to carry while inviting a caller to abandon results that are perfectly good.
  Third, that the result is positional and always the same length as the input, and that an empty
  input returns an empty, non-nil slice.

  The batch shape is kept for the same reason the resolve query keeps its multi-target signature: a
  Go caller batches a whole plan's declarations in one call, while the command line keeps one
  invocation, one answer, one exit code.
- **Commit:** `feat(quarry): package-level Name, the batched facade over the glyph maker`

### Card 5: RenderNameJSON

- **Context:**
  - `internal/engine/name.go`
  - `quarry/quarry.go`
- **Edits:**
  - `quarry/render.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `func RenderNameJSON(r NameResult) ([]byte, error)` to `quarry/render.go`, delegating to
  `renderJSON` exactly as the three existing success renderers do, so its two-space indent, single
  trailing newline, and disabled HTML escaping cannot drift from theirs. Its doc comment says the key
  order within the object is `NameResult`'s own field declaration order, so no hand-written
  marshaller is needed.

  The signature takes one result, not a batch, matching the single-result shape the resolve renderer
  already has. The doc comment states explicitly that there is no JSON view of a whole batch and none
  is added: the batch exists for one consumer, which calls the facade in-process and reads Go values,
  and no CLI path can produce a batch, so a slice renderer would have no caller on either side.

  In the same edit, update this file's own header comment: it currently names
  `RenderJSON, RenderResolveJSON and RenderExpandJSON, the three successful envelopes`. Widen the
  count and the list to four, naming `RenderNameJSON` alongside them. The header's second sentence,
  about alias types carrying no methods, stays true as written and needs no edit.
- **Commit:** `feat(quarry): RenderNameJSON, the maker's success envelope`

### Card 6: RenderNameText

- **Context:**
  - `internal/engine/name.go`
  - `quarry/quarry.go`
  - `quarry/render.go`
- **Edits:**
  - `quarry/text.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `func RenderNameText(r NameResult) string` to `quarry/text.go`. Like the other text renderers
  it cannot fail and returns no error; the returned string has no trailing whitespace on any line and
  ends with exactly one newline. It has two branches:

  Success, when `r.ID` is non-empty: one line, `r.ID`, a space, and the kind. The target is dropped
  because the existing resolve renderer's success branches already drop it in favour of the id, and
  for a one-per-call CLI the input is on the command line beside the output.

  Failure otherwise: one line, built from `normalizeProse(r.Target)` followed by the literal
  ` error` with no trailing space, then — only when `r.Reason` is non-empty — a space followed by
  `r.Reason`, then — only when `normalizeProse(r.Error)` is non-empty — `: ` followed by that
  normalised message. Spelling the segments this way rather than as a trailing-space ` error ` is
  what keeps an empty-reason value from emitting trailing whitespace, and it is the exact segment
  split the resolve renderer's own empty-status branch already uses. Follow that branch's
  degenerate-value guards: this renderer promises the no-trailing-whitespace invariant for every
  input, including an externally constructed zero-ish value the facade never produces.

  The echoed target goes through `normalizeProse` too, not only the message. A declaration head
  legitimately spans lines — an ungrouped var's signature is the whole declaration text — so echoing
  it raw would break the one-record-per-line invariant the whole text format rests on. This is not a
  new rule: `normalizeProse`'s own doc comment already states it is applied to every signature before
  printing, and the echoed target is a signature. Say so in the new function's doc comment, and say
  that the JSON view deliberately keeps the byte-verbatim echo with its newlines intact, because a
  JSON string has no line invariant to protect.

  The doc comment also states that this is a CLI view of one result and that there is no text
  rendering of a whole batch.
- **Commit:** `feat(quarry): RenderNameText, the maker's one-line text view`

### Card 7: facade and renderer tests

- **Context:**
  - `internal/engine/name.go`
  - `quarry/name.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `quarry/quarry.go`
  - `quarry/render_test.go`
  - `quarry/text_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/name_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `quarry/name_test.go` covering three things, in the style the existing renderer tests in
  this package use.

  Delegation: `Name` returns what the engine returns, asserted on a small batch mixing one success
  and one failure — positional order preserved, length equal to the input, each entry's unit and
  target echoed byte-identically. Assert an empty input returns an empty, non-nil slice.

  `RenderNameJSON`: the byte contract, asserted the way the existing JSON renderer tests assert it —
  two-space indent, exactly one trailing newline, no HTML escaping. Assert the emitted key set for a
  success payload is `unit`, `target`, `id`, `kind` and no other key, and for a failure payload is
  `unit`, `target`, `error`, `reason` and no other key, so an accidentally-populated field fails the
  test rather than passing silently. Assert no `ok` key appears on either.

  `RenderNameText`: the success line and the failure line, byte for byte, including that each ends
  with exactly one newline and carries no trailing whitespace.

  The multi-line divergence, as one test asserting both halves in one place: a declaration head
  spanning lines rendered through `RenderNameText` contains exactly one newline, at the end, while
  the same input rendered through `RenderNameJSON` still carries the head's line breaks in its
  `target` value. Assert the JSON half after unmarshalling the rendered bytes, never against the raw
  bytes: `renderJSON` disables HTML escaping only, so the encoder still emits a newline as the
  two-character escape sequence, and a raw-byte assertion would be asserting the wrong thing. Pinning
  both halves together is what makes the divergence deliberate rather than a discrepancy someone
  later "fixes".
- **Commit:** `test(quarry): facade delegation, both name renderers, and the multi-line divergence`

## Batch Tests

`verify: go test ./quarry/` runs this package's whole suite — the new `quarry/name_test.go` plus the
existing renderer, text, and repo tests. No `-run` narrowing is used here: the package is small, its
suite is fast, and three of this batch's five cards edit files the existing tests already cover, so
running them is the point rather than an overreach.
