# Batch: run-gate-and-subcommand

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: run-gate-and-subcommand
number: 5
cards: 3
verify: go build ./bench/loomyard-eval/ladder/... && go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestE2E|TestPreMatrix'
depends-on: [2, 4]
```

## Batch Scope

This batch closes the loop between the pack and the matrix: the run entry point gains a pre-rep-1
verification that every selected cell's card exists and that the pack in the pack cell's card is the
pack provenance recorded, and the command-line layer gains the third subcommand that produces that
pack. It is one batch because the gate and the command are the two halves of one contract — the
command writes a hash the gate reads — and because a gate with no way to satisfy it, or a command
whose output nothing checks, is not a shippable half.

It depends on the sweep batch because it edits the same function in the run loop's own file, and on
the pack batch because the gate calls the block extractor and hash the pack batch defines.

Batch-local decision: the gate runs only when the loaded ladder file declares a pack cell. Every
existing ladder file declares none, so the gate is inert for them and no committed results root
changes behaviour. The card-existence half runs for every selected config that declares a card,
whether or not the file has a pack cell.

## Cards

### Card 21: Verify the cards and the pack hash before rep 1

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add an unexported helper in `run.go` — name it `verifyCardsAndPack` — taking the loaded ladder, the
  selected configs, the merged provenance record and the quarry repository root, and returning an
  error. Call it from `Run` immediately after the first provenance write and before the server-build
  block, so a finding costs no API call and no server build.
  It performs two checks in this order.
  First, for every selected config declaring a non-empty card, stat the card path resolved against
  the quarry repository root exactly as the run loop resolves it, and return an error naming the cell
  and the path when it is missing or unreadable. A typo'd card path would otherwise first surface
  partway through rep 1, after the other cells had already spent API calls, and the check costs one
  stat per cell.
  Second, when the ladder file declares a pack cell, compare exactly one thing: the block hash of
  that cell's card equals the pack hash the provenance record's pack block carries. Read the card,
  extract its sentinel-delimited block, hash it, and compare. A record with no pack block at all is
  an error telling the operator to run the pack subcommand first. A mismatch is an error naming both
  hashes. When the file declares no pack cell this whole check is skipped, which is what keeps every
  existing ladder file and every committed results root behaving exactly as it does today.
  The gate deliberately does not compare the quarry repository's commit or dirty flag against the
  pack block's own record of them. Write that as a comment on the helper, with both reasons: the
  merged record derives its top-level commit and dirty flag from the latest invocation, so the two
  sides differ after any commit between the pack and the run — and committing the generated card is
  exactly such a commit, and the intended workflow, so a gate on it would brick the root for doing
  the right thing; and the dirty flag is vacuously true on both sides the moment the pack writes a
  tracked card, so a gate on it would always be satisfied. What enforces the never-edit-the-code-
  under-test rule is the per-invocation record, which makes a mid-matrix edit visible and auditable
  afterwards, not a gate.
  State the operator workflow in the same comment so it is not rediscovered: run the pack, inspect
  its output, commit the generated card together with the ladder file and the task and fasit files,
  then start the matrix. Committing between the pack and the run is expected and is not a freshness
  violation.
- **Commit:** `feat(ladder): verify cards and the pack hash before the first repetition`

### Card 22: Add the pack subcommand

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a `packCommand` function alongside the two existing subcommand functions, built the same way:
  its own flag set with three flags — a required config path, a required results root, and an
  optional claude binary path defaulting to the same value the run subcommand's own flag defaults to.
  It resolves the quarry repository root from the process's working directory exactly as the run
  subcommand does, builds the pack options, and calls the pack entry point. It returns no non-zero
  exit signal of its own: a pack either succeeds or errors.
  Wire it into the dispatch switch as a third case, and add it to both usage strings — the one
  printed when no subcommand is given and the one printed for an unknown subcommand.
  Rewrite this file's header comment. It currently says the harness parses flags for exactly two
  subcommands and ends by asserting that because this harness drives its own loop, there is nothing
  left for a third subcommand to do. That reasoning is superseded, and the replacement must say why
  rather than merely dropping the sentence: the pack subcommand is not a step of the run loop, it
  produces an artefact the prompt consumes, which is the distinction the original sentence was
  drawing and the reason the original conclusion no longer follows.
- **Commit:** `feat(ladder): add the pack subcommand`

### Card 23: Test the pre-rep-1 verification

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add four subtests to the end-to-end suite, all reusing its existing fixture helpers.
  `MissingCardFailsBeforeDispatch`: a selected config whose card path does not exist fails, the error
  names the cell and the path, and no repetition directory was created — the assertion that it
  failed before any dispatch rather than during rep 1.
  `EditedPackFailsBeforeDispatch`: a results root whose provenance carries a pack block, whose pack
  cell's card block has been edited after generation, fails with an error naming both hashes and
  creates no repetition directory.
  `MatchingPackProceeds`: the same fixture with the card left as generated runs to completion.
  `ForeignQuarryCommitStillProceeds`: a results root whose top-level quarry commit differs from the
  pack block's own recorded commit still runs. This is the anti-regression for the decision that the
  freshness gate must not key on repository state; without it, the obvious "surely we should also
  check the commit matches" change would pass review and then brick the root on the first commit
  between the pack and the run.
  Add one assertion, in whichever existing subtest is cheapest, that a ladder file declaring no pack
  cell runs with no provenance pack block present and no error — the inertness guarantee that keeps
  every existing ladder file working.
- **Commit:** `test(ladder): cover the pre-repetition card and pack verification`

## Batch Tests

`verify:` first builds the whole harness tree, which is the only gate on the new subcommand — the
command-line layer has no test file of its own, by the same reasoning that leaves it thin — and then
runs `go test` against the harness package with `-run 'TestE2E|TestPreMatrix'`.

The end-to-end suite is where the gate is observable: it is the only place a real run reaches the
point just after the provenance write with a real results root, a real ladder file and a real card on
disk. The pre-matrix tests are included because the gate reads a card path resolved the same way the
task file path is, and the pre-matrix suite is what keeps the real committed ladder file's own paths
honest.

The pack surface's own tests are outside this batch's pattern deliberately: they belong to the batch
that introduced them and are re-run by the module-wide gate and by `pipeline.done_gate` at the end of
the task.
