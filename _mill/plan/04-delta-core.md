# Batch: delta-core

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'delta-core'
number: 4
cards: 8
verify: go test ./internal/engine/ -run 'TestDelta'
depends-on: [3]
```

## Batch Scope

This batch is the pure core: `(*engine.Repo).Delta`, which takes a batch of entries carrying two
byte slices and two units each, extracts both sides with the same extractor the existing verbs use,
compares the two symbol tables, and returns the delta.
It knows nothing about git, revisions or directories, and it reads nothing outside its arguments —
the receiver is present only for symmetry with the engine's other query methods.

It is one batch because the five implementation cards are one algorithm read top to bottom: entries
become tables, tables become created/deleted/modified, the leftovers become renames on two tiers,
and the whole answer is then put in a total order.
Splitting them would leave a half-classified answer no test could sensibly assert against.

The discussion's own closing entry records that its exact-tier machinery — conditions 1 to 7 — is
the part most likely to still carry a defect, and that its evidence tier has been settled since the
first review round.
Card 19 is therefore the card to read the discussion against most carefully, and card 23 is the test
card that exists to pin it.

The external interface batch 6 consumes: `(*Repo).Delta(entries []DeltaEntry) (DeltaAnswer, error)`,
whose error is non-nil only for a failure of the call as a whole and never for one entry.

Batch-local decision: extraction, comparison and both tiers live in one new
`internal/engine/delta.go`, with the three test cards in three separate files so a failure names its
theme in its filename.

## Cards

### Card 17: per-entry extraction and the files echo

- **Context:**
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta_tokens.go`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/walk.go`
  - `internal/engine/strategy.go`
  - `internal/engine/extension.go`
  - `internal/engine/treesitter/treesitter.go`
  - `internal/engine/glyph_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create the delta core file with a file comment stating that this is the pure
  core, that it never parses a textual diff, and that correctness lives entirely in the symbol-table
  comparison with git's only job being to avoid extracting files that did not change.
  Declare `(*Repo).Delta(entries []DeltaEntry) (DeltaAnswer, error)` and implement, in this card,
  only its per-entry extraction pass and the `files` block it produces.
  For each entry, in the input's own order, produce exactly one `DeltaFile`, so a caller can index
  the echo against what it submitted.
  A non-empty `Refusal` short-circuits the entry: no parse on either side, disposition error, and
  the refusal string as the entry's error message.
  An extension that resolves to no registered strategy gives disposition unsupported, contributes no
  symbols on either side, and is not an error.
  Otherwise the sides that exist are extracted: a nil before slice with a non-nil after slice gives
  added, a non-nil before with a nil after gives removed, and two non-nil slices give changed.
  Each side is parsed inside one call of the engine's parse seam, and inside that same callback the
  symbols and — for the later cards — their token streams are built into values owning their own
  memory, because the seam invalidates its node the moment the callback returns.
  A parse the grammar reports an error on sets the corresponding lossy flag on the file echo for
  that side independently and still contributes its surviving symbols, exactly as the existing walk
  already does.
  A failure to extract a side at all sets disposition error with a message and contributes no
  symbols from that entry; a failing entry never fails the batch, so the whole call still returns a
  nil error and the answer is still a success.
  Set each symbol's `File` field from the entry's own path, on each side independently — the
  strategy leaves it empty, and the file-change dimension, a modified entry's before block and a
  rename pair's two symbols all read it, so all three would silently compare empty strings
  otherwise.
  Honour the unspellable-unit rule: an entry whose supplied unit for a side is not spellable by the
  glyph grammar contributes no symbols from that side, the same rule the walk applies through
  `unitSpellable`.
  Both slices being nil is a caller error in the entry rather than a state to classify; give it
  disposition error with a message naming the path.
- **Commit:** `feat(engine): add the delta core's per-entry extraction and files echo`

### Card 18: the symbol table, the comparison tuple, and created, deleted and modified

- **Context:**
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta_tokens.go`
  - `internal/engine/answer.go`
  - `internal/engine/golang.go`
- **Edits:**
  - `internal/engine/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Build one symbol table per side over the whole batch, keyed by the pair of a
  symbol's id and its kind.
  Including the kind in the key is what makes a const replaced by a var of the same name a delete
  plus a create — two different declarations — rather than a modification.
  Reduce every occurrence under a key to one comparison tuple of four values: a hash of its body
  token stream, its signature text, its doc text, and its file.
  Those are the same four dimensions the changed array is drawn from, so the comparison and the
  reporting cannot cover different ground.
  A key present only on the after side contributes every one of its symbols to created; a key
  present only on the before side contributes every one of its symbols to deleted.
  A key present on both sides is compared as a multiset of comparison tuples — never of body hashes
  alone.
  Equal multisets means unchanged, and such a symbol appears nowhere in the answer; that is what
  makes a symbol whose only difference is a line shift invisible to the delta, which is the whole
  point of glyph identity.
  Any difference produces exactly one modified entry for that key, whose changed array is the union
  of the dimensions that differ across the multiset difference, emitted in the closed vocabulary's
  own declaration order — body, signature, doc, file — never in the order the dimensions were
  discovered in.
  A doc-only change to one of two same-named declarations therefore reports a changed array naming
  the doc dimension rather than vanishing.
  The entry carries after as every after-side occurrence's full symbol and before as every
  before-side occurrence's file, start, sigend and end; both are arrays for every key, so the
  ordinary single-occurrence case is simply length one and a multiplicity change is visible from the
  two lengths without any count field.
  The before array is independently sorted and is aligned with the after array positionally by
  nothing: a key held by several declarations has no per-occurrence identity to align by, and
  minting one would emit an identity the glyph contract does not define.
- **Commit:** `feat(engine): compare symbol tables into created, deleted and modified`

### Card 19: the exact-tier rename classifier

- **Context:**
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta_tokens.go`
  - `internal/engine/answer.go`
  - `internal/engine/golang.go`
  - `internal/engine/glyph_test.go`
- **Edits:**
  - `internal/engine/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Pair a deleted symbol with a created symbol at the exact tier when, and only
  when, all seven of the discussion's conditions hold — read that decision in full before writing
  this card, since it is the part of the design its own closing entry marks as highest-risk.
  In summary: the two must share a glyph unit, taking the deleted symbol's unit from the before side
  and the created symbol's from the after side, so a cross-unit move is never paired; they must
  share an owner chain and a kind; their names must differ, since identical names are the modified
  case; their body token streams must be identical modulo the renamed identifier and their signature
  token streams must be identical modulo the renamed identifier under the same node-based rule,
  never a textual substitution; exactly one such created symbol must exist for that deleted symbol
  and exactly one such deleted symbol for that created symbol; and both symbols must have a
  non-empty body stream, with neither side's file having parsed partially.
  When several candidates satisfy the first five conditions for one symbol, none of the involved
  symbols is asserted — they all fall through to the evidence tier, because several matches means
  nothing was chosen.
  The non-empty-body condition is what keeps every const, var, type alias and interface method
  element out of this tier: all of them have an empty body stream on both sides, so the body
  condition would otherwise be vacuously satisfied by any two of them and two unrelated constants
  would be asserted as a rename.
  The lossy condition is there for the same reason from the other direction: a truncated table
  manufactures spurious deleted entries, and a spurious delete is exactly the input that turns an
  assertion into a confident lie.
  An asserted pair removes its two constituents — the deleted symbol from deleted and the created
  symbol from created — so the pair becomes the only surviving record of either, which is why it
  carries both symbols in full rather than their ids.
  A key held by several declarations on either side is never a rename candidate on either tier.
  Classification is relative to the supplied batch and never to the unit as it exists on disk: a
  rename whose other half was not in the batch is a plain deleted or created symbol with no
  candidate, because quarry cannot pair against a symbol it was never given.
  The two symbols may live in different files.
- **Commit:** `feat(engine): add the exact-tier rename classifier`

### Card 20: the evidence-tier candidates and their signals

- **Context:**
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta_tokens.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Report a deleted-created pair as an evidence-tier candidate when it satisfies
  the first three exact-tier conditions — same unit, same owner chain, same kind, different name —
  and is not part of an asserted exact pair.
  Structural facts alone gate the candidate set: there is no similarity threshold, no floor and no
  cut-off anywhere, so the query has no tuning knob.
  The deleted symbol stays in deleted and every candidate created symbol stays in created; the
  candidate block only cross-references them and attaches signals, because suppressing the create
  and the delete for a candidate quarry has not resolved would be a silent pick in disguise.
  Fill each candidate's five signals: whether the two signature token streams are identical modulo
  the renamed identifier, computed over the streams under the node-based rule and never over the
  verbatim signature text; the body token similarity coefficient; the two body stream lengths; and
  whether the two doc texts are identical.
  Emit no composite score of any kind.
  Sort candidates within one entry by body token similarity descending, then by created id
  ascending — a deterministic ordering, not a ranking.
  A pair demoted from the exact tier by the ambiguity condition, by an empty body stream or by a
  lossy side lands here with its signals filled like any other candidate; a demoted bodyless pair
  will read as identical signatures modulo the name and a similarity of exactly one over two empty
  streams, which is correct and is precisely why it is reported rather than asserted.
- **Commit:** `feat(engine): add the evidence-tier rename candidates and their signals`

### Card 21: total ordering of every array in the answer

- **Context:**
  - `internal/engine/delta_answer.go`
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `internal/engine/golang.go`
- **Edits:**
  - `internal/engine/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Put every array in the answer into the total order the discussion's ordering
  decision fixes, and document each rule at the point it is applied as ordering and never as
  ranking.
  The created array, the deleted array and a modified entry's after array sort by file ascending,
  then start ascending, then id ascending, then kind ascending.
  The first two are the same rule `symbolsOfUnit` already applies so a symbol list reads the same way
  wherever a caller meets it; the last two are the total tie-break, and they are load-bearing rather
  than decorative, because the const and var builders give every name declared in one spec the same
  start and end, so file-then-start alone cannot separate them and a stable sort would then preserve
  whatever order the table's map iteration happened to produce.
  The pair of id and kind is unique because it is the table key.
  A modified entry's changed array is in the closed vocabulary's declaration order.
  The modified array sorts by id ascending then kind ascending — the table key itself, so the order
  does not depend on which occurrence was seen first.
  The renamed array sorts by the from symbol's id ascending then the to symbol's id ascending.
  The rename candidates array sorts by the deleted symbol's id ascending then its kind ascending,
  with the candidates inside each entry keeping the order card 20 gives them.
  A modified entry's before array is independently sorted by file then start, aligned with the after
  array by nothing.
  The files array keeps the input batch's own order and is never sorted.
  Go randomises map iteration, so an answer assembled by ranging over the table is
  non-deterministic without every one of these rules; the seven committed goldens batch 9 requires
  cannot be byte-stable otherwise, and a pipeline diffing two delta outputs would see phantom
  changes.
- **Commit:** `feat(engine): put every delta answer array in its stated total order`

### Card 22: table tests for dispositions, created, deleted and modified

- **Context:**
  - `internal/engine/delta.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta_tokens.go`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/repo_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestDelta_Dispositions` and `TestDelta_CreatedDeletedModified` in package
  `engine`, every case built from string literals in the test file with no fixture tree.
  The dispositions test covers: a file added, a file removed, a file changed, an entry whose
  extension has no strategy giving unsupported with no symbols and no error, an entry carrying a
  pre-set refusal asserted skipped without any parse on either side and surfacing as an error
  disposition with that exact message in its own input position, an entry whose bytes are not valid
  source giving an error disposition while every other entry in the same batch still contributes its
  symbols and the call still returns a nil error, an empty batch, and a syntactically broken after
  side asserted to set the after-side lossy flag while still contributing its surviving symbols.
  The created-deleted-modified test covers: created only; deleted only; modified only; a symbol
  whose only change is its line position, asserted absent from every array in the answer; each
  changed dimension in isolation — body only, signature only, doc only, and the same symbol moved
  between two files of one unit for the file dimension — and one combination; a body reformatted
  with whitespace and comment changes only, asserted not modified; a body changing an increment to a
  decrement and one changing a plus to a minus, each asserted modified naming the body dimension; a
  struct with a changed field and a type alias changed to a different underlying type, each asserted
  modified, since the nil-body branch must not collapse to an empty comparison; adding a method to
  an interface, asserted to produce both a created method symbol owned by the interface and a
  modified entry for the interface type; a const replaced by a var of the same name, asserted one
  create and one delete and never a modification; two same-named parameterless functions before and
  after differing only in the doc of one of them, asserted modified naming the doc dimension, and
  likewise for a signature-only difference; the before and after arrays asserted length one for an
  ordinary symbol and length two for a two-occurrence key, with a multiplicity change visible from
  the lengths; and an entry whose supplied unit is unspellable contributing no symbols.
- **Commit:** `test(engine): assert delta dispositions and the created, deleted and modified arrays`

### Card 23: table tests for both rename tiers

- **Context:**
  - `internal/engine/delta.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta_tokens.go`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/repo_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta_rename_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestDelta_ExactTier` and `TestDelta_EvidenceTier` in package `engine`, over
  in-memory sources only.
  The exact-tier test covers: a rename whose bodies are identical modulo the identifier, including a
  recursive self-call, asserted present in renamed with both constituents absent from created and
  deleted, and with the pair's two symbols each carrying an id, a kind, a file and spans, since the
  pair is the only surviving record of either location; the same rename with one extra statement in
  the body, asserted demoted to the candidate array with the create and the delete both still
  present; the same rename whose bodies differ only in an anonymous operator token, asserted demoted
  and never present in renamed; two deleted symbols each pairing exactly with one created symbol,
  asserted to produce no renamed entry at all and candidate entries instead; a cross-file rename
  inside one unit, asserted exact; a cross-unit rename, asserted paired on neither tier and reported
  as a plain create plus delete; a rename whose other half lies outside the supplied batch, asserted
  a plain deleted symbol with no candidate at all, which is what proves classification is relative
  to the batch; a rename involving a file whose after side parsed partially, asserted demoted to the
  candidate array; and two unrelated interfaces of the same owner and kind, one deleted and one
  created, asserted not an exact-tier rename — a case a node-containment stream definition would get
  wrong.
  The evidence-tier test covers: a deleted constant and a created constant of different names,
  asserted absent from renamed but present as a candidate whose signature signal is true and whose
  similarity is exactly one over two empty streams, and the same for a renamed var, a renamed type
  alias and a renamed interface method — without the non-empty-body condition every one of these is
  a false assertion; a method renamed such that a textual substitution would corrupt its receiver,
  asserted to carry a true signature signal; that the deleted symbol and every candidate created
  symbol remain in deleted and created respectively; that no composite score field exists on any
  candidate; and that a key held by several declarations is never a candidate on either tier.
- **Commit:** `test(engine): assert the exact and evidence rename tiers`

### Card 24: ordering determinism test

- **Context:**
  - `internal/engine/delta.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/repo_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/delta_order_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestDelta_Ordering` in package `engine`, over a batch large enough that
  Go's randomised map iteration would surface if the ordering were not total.
  It asserts that repeated calls on the identical batch produce a byte-identical marshalled answer,
  and it must include a source declaring several names in one const spec, since those symbols share
  one start and one end and are exactly the case the id-and-kind tie-break exists for — a
  file-then-start rule alone cannot separate them.
  It also asserts each array's own stated rule directly rather than only asserting stability: the
  created and deleted arrays in file-then-start-then-id-then-kind order, the modified array in
  id-then-kind order, the renamed array in from-id-then-to-id order, the candidate array in deleted
  id order, and the files array in the input batch's own order.
  Assert one modified entry's changed array is in the closed vocabulary's declaration order for a
  case whose dimensions would plausibly be discovered in a different order.
  Repeat the call enough times that a non-total order fails reliably rather than occasionally.
- **Commit:** `test(engine): assert the delta answer's total ordering and determinism`

## Batch Tests

`verify:` runs `go test ./internal/engine/ -run 'TestDelta'`, which selects the five test functions
cards 22, 23 and 24 add across `internal/engine/delta_test.go`,
`internal/engine/delta_rename_test.go` and `internal/engine/delta_order_test.go`, and nothing else —
no existing test in this package carries that prefix.
The pattern is deliberately by prefix rather than by full name, because the three test cards between
them cover more than thirty scenarios and grouping them into five functions with sub-tests is the
readable shape; a `-run` naming each sub-test would have to be edited on every added case.
Every case is in-memory source, so the whole batch runs without a repository, a fixture tree or a
git binary, which is the property the pure core was designed for.
