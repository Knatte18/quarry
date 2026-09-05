# Batch: delta-renderers

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'delta-renderers'
number: 7
cards: 4
verify: go test ./quarry/ -run 'TestRenderDelta'
depends-on: [6]
```

## Batch Scope

This batch adds the delta verb's two renderers to the facade: the JSON renderer, sharing the one
encoder configuration every existing success renderer delegates to, and the lossless text view,
following the conventions the three existing text renderers already establish.
It depends on batch 6 because both take the wrapped git answer, which that batch declares.

Both renderers take the wrapped form rather than the bare core answer, since the command line is
their only caller and it always has revisions in hand.
A Go caller holding a bare answer from the pure path wraps it with empty revisions to render it, or
reads the struct directly, which is what a facade consumer does anyway; say so in the doc comments
rather than adding a second pair of renderers.

The external interface batch 8 consumes: a JSON renderer returning bytes and an error, and a text
renderer returning a string and no error, matching the shapes the existing renderers for the other
three verbs already have.

Batch-local decision: the two renderers are added to the existing renderer files rather than to a
new one, because both of those files own a byte contract stated once at the top and shared by every
function in them; a delta renderer in a third file would be the first place that contract could
drift.

## Cards

### Card 38: the delta JSON renderer

- **Context:**
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/engine/delta_answer.go`
  - `quarry/doc.go`
- **Edits:**
  - `quarry/render.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `RenderDeltaJSON`, taking the wrapped git answer and returning bytes and an
  error, delegating to the file's one unexported encoder helper exactly as the three existing
  success renderers do.
  Delegating is not optional: that helper is the single place the two-space indent, the one trailing
  newline and the disabled HTML escaping are configured, and a renderer building its own encoder
  would be the first place the byte contract could differ between verbs.
  Key order within each object is the encoded type's own field declaration order, so no hand-written
  marshaller is needed here or anywhere else in the delta path.
  Do not add the failure envelope's marker key to this renderer's output: that key is present only
  on the failure path, and an empty delta is a successful answer rather than a negative one.
  Update this file's own header comment, which enumerates the renderers it declares.
- **Commit:** `feat(quarry): add RenderDeltaJSON`

### Card 39: the delta text view

- **Context:**
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/answer.go`
  - `quarry/doc.go`
- **Edits:**
  - `quarry/text.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `RenderDeltaText`, taking the wrapped git answer and returning a string,
  with no error return, matching the three existing text renderers in this file.
  It must satisfy the same two invariants they promise and state: no trailing whitespace on any
  line, and exactly one newline at the end.
  It must be lossless — every field of the answer reachable in the JSON view must be recoverable
  from the text view, since the text view is fixed as lossless and this discussion requires goldens
  in both.
  Reuse this file's existing helpers rather than reimplementing their rules: the prose normaliser
  for every doc, signature and error value, so an arbitrary multi-line message cannot break the
  one-record-per-line property the whole format rests on, and the existing symbol-line writer for
  every symbol the delta emits, so a symbol reads the same way here as it does in the other three
  views.
  A symbol in a delta always carries a file, since the core fills that field on both sides, so the
  symbol line's file prefix is always present here — unlike inside a table-of-contents answer, where
  the field is empty by design.
  **The shared symbol-line writer does not emit a symbol's kind**, so this renderer must spell the
  kind itself wherever a symbol or a table key appears: beside each created and deleted symbol,
  beside each of a renamed pair's two symbols, and on each modified entry and each candidate entry,
  whose own kind fields are part of the key rather than of a symbol.
  This is a correctness requirement rather than a presentational choice: the kind is in the JSON view
  with no other carrier, the text view is fixed as lossless, and a const replaced by a var of the
  same name — which the core reports as one create and one delete carrying an identical identifier —
  would otherwise render as two indistinguishable lines.
  Do not widen the shared writer to emit the kind: it is byte-pinned by the existing goldens and
  tests of the other three verbs, and this task changes no existing output.
  **A modified entry's before array cannot use the shared writer at all** and needs its own line
  grammar, stated here: its elements are location blocks carrying a file, a start, a signature end
  and an end, with no identifier, signature or docstring for the shared writer to emit.
  Give the location line a grammar close enough to the symbol line that the two read as one family —
  the same file prefix and the same span and signature-end conventions, including omitting the
  signature-end clause when it is zero, which is the engine's own marker for a declaration with no
  body — and identify the entry it belongs to from the modified entry's own identifier and kind,
  which are printed once for the entry rather than repeated per location.
  This is the one shape in the delta whose text form has no existing precedent, so spell it fully in
  the doc comment rather than leaving it to the implementer.
  Choose a block-per-section grammar covering the files echo with each entry's disposition, lossy
  flags and error; the created, deleted and modified arrays; the renamed pairs; and the candidate
  entries with every signal spelled by name.
  Emit the two revisions, with an explicit word for the working-tree side rather than an empty
  field.
  Fix the grammar in this function's own doc comment to the character, as the existing renderers'
  doc comments do, since that comment is the specification the goldens are read against.
  Preserve every array's order exactly as the answer carries it; a renderer that sorted would defeat
  the total ordering the core establishes.
- **Commit:** `feat(quarry): add the lossless RenderDeltaText view`

### Card 40: JSON renderer key-order and byte-contract tests

- **Context:**
  - `quarry/render.go`
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/engine/delta_answer.go`
- **Edits:**
  - `quarry/render_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestRenderDeltaJSON_KeyOrder` and `TestRenderDeltaJSON_ByteContract`,
  mirroring the shape of the existing expand renderer's key-order test in this file.
  The key-order test asserts the top-level key order is the two revisions followed by the files
  echo, created, deleted, modified, renamed and the candidates, and asserts the key order inside a
  file echo entry, a modified entry, a renamed pair, a candidate entry and a signals block.
  The byte-contract test asserts two-space indentation, exactly one trailing newline, and that a
  signature containing angle brackets and an ampersand survives unescaped, which is what the
  disabled HTML escaping is for.
  Assert that the after-side revision marshals as an explicit null when the after side is the
  working tree and as a string when it is a revision, since the presence of that key is the
  statement that distinguishes the two.
  Assert that the flags omitted when false are absent from an ordinary entry and present when set.
- **Commit:** `test(quarry): assert RenderDeltaJSON key order and byte contract`

### Card 41: text view tests

- **Context:**
  - `quarry/text.go`
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `quarry/text_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestRenderDeltaText` and `TestRenderDeltaText_Lossless`, mirroring the
  shape of the existing expand text renderer's test in this file.
  The first asserts the exact rendered string for a hand-built answer exercising every section —
  a file echo carrying each disposition and both lossy flags, a created symbol, a deleted symbol, a
  modified entry naming several changed dimensions with a multi-occurrence before and after array, a
  renamed pair, and a candidate entry with every signal — and asserts the two invariants the
  renderer promises: no trailing whitespace on any line and exactly one closing newline.
  The second asserts losslessness structurally rather than by eye: every value present in the JSON
  view of the same answer appears somewhere in the text view, including each signal's value and each
  changed dimension's word.
  Include the case that makes the kind load-bearing — a const replaced by a var of the same name,
  giving one created and one deleted symbol with an identical identifier and differing kinds —
  asserted to render as two distinguishable records, since the shared symbol-line writer emits no
  kind and card 39 requires this renderer to spell it.
  Include a case whose doc text spans several lines and contains runs of whitespace, asserted
  collapsed to single spaces by the shared prose normaliser, since an un-normalised value would
  break the one-record-per-line property.
  Include a case whose after-side revision is the working tree, asserted to render the explicit word
  rather than an empty field.
- **Commit:** `test(quarry): assert the delta text view and its losslessness`

## Batch Tests

`verify:` runs `go test ./quarry/ -run 'TestRenderDelta'`, selecting exactly the four test functions
cards 40 and 41 add to `quarry/render_test.go` and `quarry/text_test.go`; no existing test in this
package carries that prefix, and the existing renderer tests for the other three verbs are
untouched by this batch.
Both cards build their answers by hand in the test file rather than by running a delta, so this
batch needs neither a git binary nor a fixture repository: the renderers are pure functions of a
value, and testing them through a delta would be testing batch 4 again.
The committed goldens for the seven required cases are batch 9's job, and they exercise these same
two renderers end to end against a real answer.
