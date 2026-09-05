# Batch: pack-generation

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: pack-generation
number: 4
cards: 5
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestRenderKickstartPack|TestPackBlock|TestPack_'
depends-on: [1, 3]
```

## Batch Scope

This batch delivers the whole pack-generation surface in one new file: the line renderer that turns a
batched resolve into the block that goes in the prompt, the sentinel-delimited read and write that
put that block into a card file idempotently, and the `Pack` entry point that ties them to the run
lock, the pinned worktree, the facade and the provenance record. It is one batch because the three
pieces are meaningless apart — the renderer's output is defined by where it is written, and the
hashes the provenance block records are computed over exactly the bytes the writer wrote.

This is also the batched facade's first real exercise: `Pack` makes exactly one
`(*quarry.Repo).Resolve` call over the full glyph list, which is the performance property the facade
exists to preserve. No cell calls the facade at run time.

The external interface batch 5 and 7 consume is: the two sentinel constants, `ExtractPackBlock`,
`PackBlockSHA256` and the exported `Pack` with its `PackOptions`.

Batch-local decision: `Pack` does not restore the pinned worktree when it finishes. It only reads
through the facade, so it dirties nothing, and restoring would discard state that some other holder
of the worktree put there rather than state the pack caused. The run lock, held for the whole
command, is what keeps a pack and a run off one pinned worktree at the same time.

## Cards

### Card 16: Render the pack lines from a batched resolve

- **Context:**
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `quarry/text.go`
  - `internal/engine/answer.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `pack.go` with a header comment stating what the file is for: generating the kick-start
  pack that a pack cell's card carries, and the sentinel-delimited block protocol that lets `run`
  verify the pack in the prompt is the pack provenance recorded.
  Add `RenderKickstartPack(results []quarry.ResolveResult) (string, error)`. It emits two lines per
  result, in the slice's own order — which is the ladder file's `pack_targets` order, since the
  facade answers positionally:

  ```
  <target> → <file> <start>-<end>
      <signature>
  ```

  The first line's fields are the result's own `Target` verbatim, its single symbol's repository-
  relative file, and that symbol's start and end lines joined by a hyphen. The second line is
  indented by exactly four spaces and carries the symbol's signature with its internal newlines
  collapsed: split the signature on newlines, trim each resulting line's surrounding whitespace, and
  join with a single space. The rendered block ends with no trailing newline.
  The docstring is never emitted, and neither is the signature-end line. This is the treatment's
  definition, not a formatting preference: emitting the docstring would turn "the agent knows where
  things are" into "the agent knows where things are and has a slab of prose", and a win would then
  be uninterpretable. The existing renderer in the facade's own text layer emits the docstring, which
  is exactly why this function exists instead of a call to it. State that in the doc comment.
  Any result whose status is not the found status, and any result carrying a non-empty pre-resolution
  error, is fatal: return an error naming the offending target and what came back for it. Not-found,
  ambiguous, multipart and a pre-resolution error are all fatal, with no partial output — a pack
  missing one glyph is a different treatment from the one the cards describe.
  A found result carries exactly one symbol; treat a found result with no symbols as the same class
  of fatal error rather than indexing into an empty slice.
- **Commit:** `feat(ladder): render kick-start pack lines from a batched resolve`

### Card 17: Read and write the sentinel-delimited card block

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add two exported constants, `PackSentinelBegin` and `PackSentinelEnd`, spelled as markdown comment
  lines so they are invisible in a rendered card while remaining a literal, greppable line in the
  prompt text. Document that they must appear on their own lines, exactly once each, in that order.
  Add `ExtractPackBlock(cardText string) (string, error)`, returning the text strictly between the
  two sentinel lines — neither sentinel line included — with leading and trailing blank lines removed
  and no trailing newline. A card missing either sentinel, carrying either more than once, or
  carrying them in the wrong order, is an error naming which condition failed. A card whose block is
  empty is not an error: that is the state an authored card starts in, before the pack is generated
  into it.
  Add `WritePackIntoCard(cardText, pack string) (string, error)`, returning the card with the region
  between the sentinels replaced by `pack`. It preserves everything outside the sentinels byte for
  byte, and it is idempotent: writing the same pack twice yields the same text. A card missing its
  sentinels is an error, never a silent append — appending would produce a card whose prompt text
  disagrees with the hash provenance records, which is the one failure this whole mechanism exists to
  make impossible.
  Add `PackBlockSHA256(block string) string`, returning the hex sha256 of the block's bytes, computed
  by calling the package's existing `sha256Hex` helper rather than by spelling the same computation a
  second time.
  Both the writer and the run-time gate go through this one function, so the two cannot drift.
- **Commit:** `feat(ladder): read and write the sentinel-delimited pack block`

### Card 18: Add the Pack entry point

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `quarry/repo.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `PackOptions` carrying `LadderFilePath`, `ResultsRoot`, `ClaudeBinPath`, `QuarryRepoStart` and
  `Runner`, documented field by field, mirroring the run options struct's own field set minus the
  cell selection and repetition override, which have no meaning here.
  Add `Pack(ctx context.Context, opts PackOptions) error` performing these steps in this order:
  1. load the ladder file;
  2. resolve the quarry repository root, the target repository and the worktree root exactly as the
     run entry point does;
  3. acquire the same advisory run lock against the worktree root and the results root, with the
     release deferred, so a pack and a run can never touch one pinned worktree concurrently. The
     lock is exclusive-create and is never reaped automatically, so a pack that dies leaves the same
     operator-cleared stale lock a dead run does — the existing, documented behaviour, and not
     something this command works around;
  4. find the file's single pack cell — the config whose pack flag is set — and error naming the
     ladder file when there is none. Its own card path is the file the pack is written into; there
     is deliberately no flag for it, so the card written can never disagree with the card the matrix
     renders;
  5. prepare the pinned worktree for that cell's task, exactly as a run does;
  6. collect the invocation, with exactly the inputs a run of this ladder file would use: the quarry
     repository root, the ladder file path, the target repository path, the file's server name, the
     claude binary path, the file's own repetition count as the effective count, and every config id
     in the file as the selected cells. This is why the claude binary path is an option rather than a
     constant: collection probes the binary's version and aborts when it cannot be executed, so the
     pack must be able to name the same binary the run will;
  7. open the pinned worktree through the facade and make exactly one batched resolve call over the
     ladder file's whole glyph list;
  8. render the pack, halting on the first target that did not resolve found;
  9. read the pack cell's card, write the rendered pack between its sentinels, and write the result
     back;
  10. marshal the resolve results as indented JSON and write them to `pack-resolve.json` under the
      results root, creating the root when absent;
  11. read the existing provenance record, merge the collected invocation into it, set the
      `kickstart_pack` block on the merged record, and write it.
  The block's generated-at value is the invocation's own write time, so the pack invocation is
  identifiable afterwards by that timestamp matching; its quarry commit, quarry dirty flag and
  target-repository commit are copied from the same invocation; its targets are the ladder file's
  glyph list; its pack hash is the block hash of what step 9 wrote; its resolve hash is the hex
  sha256 of the bytes step 10 wrote; its card file is the pack cell's own repository-relative card
  value.
  Write a real, complete invocation rather than a pack-only stub, and say why in a comment: the merge
  refuses a record whose ladder file, target-repository hash, server name or effective repetition
  count differs from the existing one, and the run entry point additionally refuses a root whose
  effective repetition count differs from its own. A stub leaving those at zero fails both checks on
  the very first run, before rep 1, every time.
  Accept the two consequences deliberately and record them in the doc comment: the record carries one
  invocation that ran no repetitions, and the effective repetition count is pinned from the pack
  onward, so a later per-run repetition override against the same root is refused — which is the
  correct behaviour under a locked n, not a limitation to work around.
  Do not restore the pinned worktree on the way out. The pack only reads.
- **Commit:** `feat(ladder): add the pack entry point`

### Card 19: Test the renderer and the block protocol

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `quarry/quarry.go`
  - `internal/engine/answer.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `TestRenderKickstartPack_GoldenBlock`, driving a fabricated slice of resolve results — no
  repository involved, the renderer is pure — and comparing the whole rendered string against a
  golden block held in the test. The golden must exercise: order preservation across at least three
  entries, absence of the docstring for a symbol that has one, and a multi-line signature collapsed
  onto one line.
  Add `TestRenderKickstartPack_FatalStatuses`, a table with one case per fatal input — not found,
  ambiguous, multipart, a pre-resolution error, and a found result carrying no symbols — each
  asserting an error naming the offending target and no partial output.
  Add `TestPackBlockRoundTrip`: writing a pack into a card between its sentinels preserves everything
  outside them byte for byte, extracting it returns exactly what was written, writing the same pack
  twice yields the same file, and a card whose block is empty extracts to the empty string without
  error.
  Add `TestPackBlockErrors`, a table over a card with no sentinels, one with only the begin sentinel,
  one with only the end sentinel, one carrying a sentinel twice, and one carrying them in the wrong
  order — each asserting an error rather than a silent append or a silent empty read.
  Add `TestPackBlockSHA256_MatchesWrittenBlock`, asserting the hash of the extracted block equals the
  hash of the string handed to the writer, which is the invariant the run-time gate depends on.
- **Commit:** `test(ladder): cover the pack renderer and the block protocol`

### Card 20: Test the Pack entry point end to end

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  These tests live in the same package as the end-to-end suite, so its existing fixture helpers — the
  synthetic quarry repository, the synthetic target repository, the worktree root and the fake claude
  binary — are callable directly and must be reused rather than reimplemented.
  Add `TestPack_EndToEnd`: a synthetic target repository holding one small Go package with two
  exported declarations, a ladder file naming those two as its glyph list with one pack cell whose
  card carries the two sentinels and an empty block, and a fresh results root that does not yet
  exist. Assert that the card comes back with both glyphs rendered between its sentinels; that the
  resolve JSON file was written; that the provenance record exists, carries a pack block whose pack
  hash equals the hash of the card's extracted block, whose targets equal the ladder file's glyph
  list, and whose card file is the ladder file's own repository-relative value; and that the record
  carries exactly one invocation. The results root not existing beforehand is part of the assertion —
  it is the pack command's own use of the provenance mkdir fix.
  Add `TestPack_UnresolvableGlyphIsFatal`: the same fixture with one glyph that does not exist in the
  synthetic package. Assert the returned error names that glyph, that the card is left byte for byte
  unchanged, and that no provenance record was written. A half-written card is the one state the
  whole sentinel-and-hash mechanism cannot recover from.
  Add `TestPack_LockHeld`: with the advisory lock file already present under the worktree root,
  assert the pack fails with the message naming the existing holder's process id and results root,
  and that it does so before touching the card. Add the release half as an assertion inside the two
  tests above rather than as a third test — the successful run and the glyph-failure run must both
  leave no lock file behind.
- **Commit:** `test(ladder): cover the pack entry point, its failures and its lock`

## Batch Tests

`verify:` runs `go test` against `./bench/loomyard-eval/ladder/internal/ladder/` with
`-run 'TestRenderKickstartPack|TestPackBlock|TestPack_'`. The three alternatives cover exactly the
three surfaces this batch adds: the renderer, the sentinel block protocol, and the `Pack` entry
point. The trailing underscore in the third keeps the pattern off unrelated future tests whose names
begin with the same word.

`TestPack_EndToEnd` and `TestPack_UnresolvableGlyphIsFatal` exercise the real facade against a real
temporary repository and the real `git worktree add` through the production runner, so they are the
only offline proof that the batched resolve, the card write, the resolve-JSON write and the
provenance write agree with each other. They also cover the provenance mkdir fix from a second call
site, which is why this batch depends on the provenance batch rather than merely following it.

The module-wide `go build ./...` at the batch boundary is what catches the new import of the facade
package from the harness package — the first time the harness imports it — including any accidental
import cycle.
