# Batch: delta-types-and-tokens

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'delta-types-and-tokens'
number: 3
cards: 6
verify: go test ./internal/engine/ -run 'TestTokenStreams|TestIdenticalModuloName|TestBodyTokenSimilarity'
depends-on: [1]
```

## Batch Scope

This batch declares the delta answer's whole type set and implements the two pure primitives every
later classification is built on: the token streams, extracted by byte range from a single walk of
the parse tree's leaves, and the identical-modulo-the-renamed-identifier predicate over them.
It depends on batch 1 because both streams are defined by byte range over the three offsets that
batch adds to `Symbol`.

The batch is one unit because the types and the primitives are read together: the signals block on a
rename candidate names the two stream lengths and the similarity coefficient, so declaring the type
without the function that fills it would leave the plan's most reviewed contract half-stated.
Nothing here compares two symbol tables — that is batch 4 — so this batch has no notion of created,
deleted, modified or renamed.

The external interface batch 4 consumes: `DeltaAnswer` and its member types; a function that, given
one file's parsed root, its source bytes and the symbols extracted from it, returns each symbol's
signature and body token streams; a predicate over two streams and two names; and a similarity
coefficient over two body streams.

Batch-local decision: the answer types live in a new `internal/engine/delta_answer.go` beside
`answer.go`, and the token machinery in a new `internal/engine/delta_tokens.go`.
Keeping them out of `answer.go` keeps that file's own header rule — every JSON tag in it is the
closed emitted key set of the three existing verbs — readable, while the new file states the same
rule for its own additive key set.

## Cards

### Card 11: declare the delta answer types

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/doc.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta_answer.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare, in package `engine`, the whole delta type set with the exact JSON key
  names the discussion's Decisions fix.
  `DeltaEntry` is the core's input and carries six fields: the entry's repository-relative path; its
  before and after byte slices, where nil means the file did not exist on that side and an
  empty-but-non-nil slice means an existing empty file; its before-side and after-side glyph units;
  and a refusal string.
  None of the six is emitted, since the input is not part of the answer.
  The disposition vocabulary is a closed string type with the five values added, removed, changed,
  unsupported and error.
  `ChangedDimension` is a closed string vocabulary with the four values body, signature, doc and
  file, declared in exactly that order, which is also the order a `changed` array is emitted in.
  `DeltaFile` carries `path`, `disposition`, an omitted-when-empty `error`, and `lossy_before` and
  `lossy_after` flags omitted when false.
  `SymbolLocation` carries `file`, `start`, `sigend` and `end`, mirroring the corresponding fields
  of `Symbol` so a before-side block reads the same way a symbol does.
  `ModifiedSymbol` carries `id`, `kind`, `changed`, `before` as a `[]SymbolLocation` and `after` as
  a `[]Symbol`; both arrays, always, so a key held by several declarations needs no second entry
  shape and the multiplicity is visible from the lengths.
  `RenamedPair` carries `from` and `to`, each a full `Symbol`.
  `RenameSignals` carries `signature_identical_modulo_name`, `body_token_similarity`,
  `body_tokens_before`, `body_tokens_after` and `doc_identical`, none omitted.
  `RenameCandidate` carries the created symbol's `id`, `kind`, `file` and its `signals`.
  `RenameCandidateEntry` carries one deleted symbol's `id` and `kind` plus its `candidates` array.
  `DeltaAnswer` carries `files`, `created`, `deleted`, `modified`, `renamed` and
  `rename_candidates`, in that declaration order, which fixes the emitted key order.
  Every type gets a doc comment; three of them carry a specific statement the discussion requires be
  recorded in the type's own doc rather than only in the plan.
  `RenameCandidateEntry`'s must state that the candidate ordering is deterministic ordering and
  explicitly not a ranking, not a recommendation and not a verdict, and that a directory with many
  deleted and many created symbols of one kind and owner can produce a large candidate block, which
  is accepted rather than capped because a cap is a threshold under another name.
  `RenamedPair`'s must state that an exact pair's constituents are removed from the created and
  deleted arrays, so the pair is the only surviving record of either symbol's location.
  `DeltaAnswer`'s must state the divergence from the listing rule of the table-of-contents verb: a
  file that is tracked and also matched by a gitignore pattern is kept in a delta batch, so this
  answer can report a symbol that verb never lists.
  Declare no method on any of these types and no custom marshaller anywhere.
- **Commit:** `feat(engine): declare the delta answer type set`

### Card 12: walk a file's leaves once and assign them to every containing symbol range

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/nodes.go`
  - `internal/engine/treesitter/treesitter.go`
  - `internal/engine/golang.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta_tokens.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare an unexported token type holding a node kind and a node text, both
  plain strings so the value owns its own memory, and an unexported stream type that is a slice of
  them.
  Owning the memory is a hard requirement, not a style choice: the parse seam's own doc comment
  forbids retaining a node past the callback's return, so a stream holding node pointers would be
  reading freed memory by the time it is compared.
  Declare an unexported function that takes a parsed root node, the file's source bytes and the
  symbols extracted from that file, and returns, per symbol, its signature stream and its body
  stream.
  It walks the tree's leaves exactly once, in source order, and assigns each leaf to **every**
  symbol range that contains it, never to just one: symbol spans nest and overlap by design, so an
  interface method element's leaves belong to the interface type's body stream and to that method
  symbol's own signature stream at the same time, and the several symbols declared by one const spec
  share one span verbatim and therefore share one identical stream.
  A leaf is a node with no children.
  Anonymous leaves are included, not only named ones — operators, keywords and punctuation are
  anonymous in the Go grammar, and a stream restricted to named leaves would omit every one of them,
  which would make two bodies differing only in an operator compare equal.
  A leaf belongs to a symbol's signature stream when its start byte lies in the half-open range from
  that symbol's `DeclStart` to its `BodyStart`, and to that symbol's body stream when its start byte
  lies in the half-open range from `BodyStart` to `DeclEnd`.
  A symbol whose `BodyStart` equals its `DeclEnd` therefore has an empty body stream, which is the
  intended outcome for every const, var, type alias and interface method element.
  Whitespace and line numbers contribute nothing, and neither stream ever includes a doc block,
  since the declaration node does not span it.
  Do not walk the tree a second time and do not look a node up per symbol; one walk with
  many assignments is the required shape and the reason the three byte offsets exist.
- **Commit:** `feat(engine): build per-symbol signature and body token streams from one leaf walk`

### Card 13: the identical-modulo-the-renamed-identifier predicate

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/delta_answer.go`
- **Edits:**
  - `internal/engine/delta_tokens.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare an unexported predicate taking two token streams and two names — the
  deleted symbol's name and the created symbol's name — and reporting whether the streams are
  identical modulo the renamed identifier.
  They are identical modulo the rename when they have the same length and, at every position, either
  the two tokens are equal in both kind and text, or both tokens are `identifier` nodes whose texts
  are respectively the deleted symbol's name and the created symbol's name.
  This is an exact structural test: there is no threshold, no tuning knob and no partial credit.
  The substitution is restricted to `identifier` nodes carrying exactly those two names, which is
  what keeps it exact — it admits the recursive self-call a renamed function makes and nothing else.
  An anonymous node can never be substituted, since it is not an `identifier` node, so including
  anonymous leaves in the stream widens what is compared without widening what may be substituted;
  say so in the doc comment, because that is the property that makes card 12's inclusion rule safe.
  The predicate is used for both the body streams and the signature streams under the same rule, and
  never as a textual substitution over a signature string: replacing the text `Run` with `Execute`
  inside a method head would also hit the `Runner` in its receiver, while a stream has real
  `identifier` nodes to key on.
- **Commit:** `feat(engine): add the identical-modulo-renamed-identifier stream predicate`

### Card 14: the body token similarity signal

- **Context:**
  - `internal/engine/delta_answer.go`
- **Edits:**
  - `internal/engine/delta_tokens.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare an unexported function returning the body token similarity signal for
  two body streams and the two symbol names: the Jaccard coefficient of the two streams treated as
  multisets of kind-and-text pairs, with the identifier bearing the symbol's own name normalised to
  one shared placeholder on both sides before the multisets are built.
  Two empty streams give exactly 1.0.
  The result is always in the closed interval from zero to one.
  Its doc comment must state that this value is a reported signal on a candidate quarry has
  explicitly declined to resolve, that no asserted outcome anywhere in the delta reads it, and that
  it is deliberately a cheap linear-time coefficient rather than an order-sensitive metric, because
  precision here would buy nothing quarry is allowed to spend it on.
  Choose the placeholder so it cannot collide with a real identifier text.
- **Commit:** `feat(engine): add the body token similarity signal`

### Card 15: token stream tests, including the two regressions the stream definition exists to prevent

- **Context:**
  - `internal/engine/delta_tokens.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/answer.go`
  - `internal/engine/golang.go`
  - `internal/engine/nodes.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta_tokens_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestTokenStreams_AnonymousLeavesIncluded`,
  `TestTokenStreams_ByteRangeSplit` and `TestTokenStreams_ManyToManyLeafAssignment` in package
  `engine`, all over in-memory Go source with no fixture files.
  The anonymous-leaves test asserts that a body changing an increment to a decrement, and one
  changing a plus to a minus, each produce different body streams — under a named-leaves-only rule
  both pairs would be identical, which is the regression this definition exists to prevent.
  The byte-range test asserts the two shapes a node-containment rule gets wrong: an interface type's
  body stream contains its method elements' leaves, not merely the opening brace, so adding a method
  to an interface changes its body stream; and a struct type's field leaves land in its body stream
  rather than its signature stream, so the signature stream stays byte-for-byte the span the
  signature cutter is given.
  The many-to-many test asserts an interface method element's leaves appear both in the enclosing
  interface type's body stream and in that method symbol's own signature stream, and that the
  several symbols declared by one const spec with two names share one identical stream.
- **Commit:** `test(engine): assert token stream extraction, byte-range split and leaf assignment`

### Card 16: predicate and similarity tests, including the receiver hazard

- **Context:**
  - `internal/engine/delta_tokens.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/answer.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:**
  - `internal/engine/delta_tokens_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestIdenticalModuloName` and `TestBodyTokenSimilarity` to the token stream
  test file card 15 created.
  `TestIdenticalModuloName` covers: two bodies identical but for the renamed identifier, including a
  recursive self-call, asserted identical; two bodies differing by one extra statement, asserted not
  identical; two bodies differing only in an anonymous operator token, asserted not identical; and
  the signature hazard the discussion names — a method renamed from one name to another whose
  receiver type shares a prefix with the old name, asserted identical modulo the rename, which a
  textual substitution over the signature string would get wrong by also rewriting the receiver.
  `TestBodyTokenSimilarity` covers: two empty streams giving exactly 1.0; two identical streams
  giving 1.0; two disjoint streams giving 0.0; and a partial overlap giving the coefficient computed
  by hand in the test case, so the value is pinned rather than merely bounded.
  Assert that every returned value lies in the closed unit interval.
- **Commit:** `test(engine): assert the modulo-name predicate and the similarity signal`

## Batch Tests

`verify:` runs `go test ./internal/engine/ -run 'TestTokenStreams|TestIdenticalModuloName|TestBodyTokenSimilarity'`,
selecting exactly the five test functions cards 15 and 16 add in
`internal/engine/delta_tokens_test.go`.
Every case is two Go source strings in the test file parsed through the engine's own parse seam, so
the batch needs no fixture tree and no repository.
Card 11's types have no runnable behaviour of their own and are covered indirectly here and directly
by the golden and key-order tests in batches 7 and 9; the module-wide `go vet ./...` at this batch's
boundary is what catches a type that does not compile or a struct tag that does not parse.
