# Batch: naming-round-trip

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
batch: 'naming-round-trip'
number: 4
cards: 3
verify: go test ./internal/engine/ -run 'TestRoundTrip'
depends-on: [1]
```

## Batch Scope

This batch delivers the task's done-when gate: the prediction-equals-extraction round trip over the
pinned Loomyard checkout. It is one batch because all three cards serve one test — two of them widen
existing test helpers so the third can be written without a second, driftable harvest of the same
walk. It depends on batch 1 only: the round trip calls the engine's maker directly and needs neither
the facade nor the CLI.

Batch-local decision: the new test consumes the harvest half of the existing round-trip helper, not
the helper itself. That helper's second half is a per-unit span lookup over whole real files, which
answers a question the existing Loomyard round trip already asks and this test does not, at a cost
its own doc comment names as a live constraint against go test's default timeout at Loomyard scale.

## Cards

### Card 14: carry signature and kind through the one walk collector

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/walk.go`
  - `internal/engine/loomyard_test.go`
- **Edits:**
  - `internal/engine/roundtrip_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/engine/roundtrip_test.go`:

  Extend `roundTripSymbol` with three fields — `signature string`, `kind Kind`, and `lossy bool` —
  filling the first two in `collectWalkSymbols` from the walked symbol's own `Signature` and `Kind`,
  and the third from the enclosing `FileEntry`'s own `Lossy` flag. The third field exists because
  `fileEntry` sets `Lossy` on a partial parse and still populates that entry's symbols, so a symbol
  whose signature was cut from a broken file is in the harvest by construction; card 16 asserts that
  no such symbol is present rather than letting one silently fail its zero-misses check. Extending the existing
  struct is deliberate rather than writing a second walk collector: `TestRoundTrip_QuarryItself` and
  `TestRoundTrip_Loomyard` consume the same struct and simply ignore the two new fields, and one
  collector is what keeps both round trips reading a single harvest. A parallel collector would be a
  second harvest that can drift from the first — the same argument the maker itself rests on.

  Extract the harvest half of `assertSymbolRoundTrip` into a helper,
  `harvestWalkSymbols(t *testing.T, r *Repo) []roundTripSymbol`, doing exactly what that function's
  first block does today: call `TOC` on the repository root with `DepthAll` and symbols on, fatal on
  an error, run `collectWalkSymbols`, and fatal when the walk collected zero symbols. Have
  `assertSymbolRoundTrip` call it and keep its own behaviour otherwise unchanged, so there is still
  exactly one collector and exactly one place the walk is configured.

  Update the file header comment and `assertSymbolRoundTrip`'s doc comment to name the new helper and
  the three new fields, and to say the fields exist for the naming round trip that card 16 adds.
- **Commit:** `test(engine): carry signature and kind through the shared walk harvest`

### Card 15: let compareGolden take any value

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/loomyard_test.go`
- **Edits:**
  - `internal/engine/golden_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/engine/golden_test.go`: widen `compareGolden`'s third parameter from `DirAnswer` to
  `any`. Every existing call site still compiles unchanged, and the marshal, update-flag, and
  byte-compare block stays a single implementation — which is what makes a golden produced by one
  update run compare equal to itself on every later run, and exactly what a sibling helper
  duplicating that block would put at risk of drift.

  Update the function's doc comment to say it marshals any value, not a directory answer
  specifically, and to name the counts golden card 16 adds as the reason the parameter widened.

  This is a test-helper signature change and not a behaviour change: the additive constraint governs
  the product's envelopes, verbs and exit codes, none of which this touches. Say so in the doc
  comment, so a later reader auditing the additive rule does not have to guess whether it reaches
  test helpers.
- **Commit:** `test(engine): widen compareGolden's value parameter to any`

### Card 16: the prediction-equals-extraction round trip

- **Context:**
  - `internal/engine/name.go`
  - `internal/engine/roundtrip_test.go`
  - `internal/engine/golden_test.go`
  - `internal/engine/loomyard_test.go`
  - `internal/engine/answer.go`
  - `internal/engine/golang.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/naming_roundtrip_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/engine/naming_roundtrip_test.go` holding `TestRoundTrip_LoomyardNaming`, gated by
  `loomyardRepo` and skipped under `-short` like every other whole-repository pass in this package.
  It must not grow its own environment gate: `loomyardRepo` owns the skip-versus-fail asymmetry and
  the pin check. The test does five things, in this order.

  1. Harvest every symbol in the pinned checkout through `harvestWalkSymbols`, giving each one's
     unit, signature, id and kind. Do not call `assertSymbolRoundTrip` — its second half re-runs a
     per-unit span lookup this test does not need.

  2. Partition the harvest into in-contract and excluded, by the two declared non-goals. The
     partition key for the multi-name-spec exclusion is the 5-tuple of unit, file, start, end and
     signature: a spec's several names produce symbols agreeing on all five, so two or more harvested
     symbols sharing this tuple is exactly "these came from one spec". The file is in the key
     deliberately — build-tag twins in one unit can share a signature and a line span across two
     different files, and excluding those would then assert, in step 4, an error the maker will never
     produce, since each twin's fragment names exactly one symbol and answers normally.

     Interface methods are excluded by a separate rule: a symbol whose kind is `KindMethod` and whose
     signature does not begin with the `func` keyword is an interface method element rather than a
     method declaration. Populated interface *types* are in-contract, not excluded: a harvested
     interface type's signature is head-only, cut at the body, so it goes through the maker's
     completion retry and answers normally.

     A symbol harvested from a file the walk marked lossy is neither in-contract nor excluded, and
     gets no partition rule: instead, assert before partitioning that the harvest carries no such
     symbol, failing with the offending files named. The two exclusion rules above do not cover this
     case, and a signature cut from a file whose parse reported an error would otherwise land
     in-contract and fail the zero-misses check in step 3 for a reason that has nothing to do with
     the maker. The assertion is a premise check, not an exclusion: the checkout is pinned at a commit
     whose sources compile, so a lossy file there is itself the finding, and recording it as one is
     better than quietly partitioning it away.

  3. For every in-contract symbol, call `Name` with a `Declaration` holding that symbol's unit and
     its signature, and assert the returned id equals the symbol's real id and the returned kind
     equals its real kind. Zero misses, zero extras. Batch the calls rather than calling once per
     symbol, since the facade takes a slice and the whole point of the batch shape is that a caller
     hands it a whole harvest at once.

  4. For every excluded symbol, assert the maker returns a per-entry error and never an id. This is
     what keeps the exclusions honest rather than a silent skip.

  5. Guard against a vacuous pass with three pinned counts — the harvest total, the in-contract
     count, and the excluded count — compared through `compareGolden` against a counts golden, using
     the widened third parameter card 15 provides. Declare a small local struct with three integer
     fields and JSON tags for the three counts, and pass a value of it. Report all three counts in
     every failure message.

     Alongside the counts, assert one cheap structural floor that holds before any number is known:
     the in-contract count is greater than zero, and the excluded count is strictly less than the
     harvest total. A partition bug that sweeps everything into one side fails immediately on that
     floor, without waiting for numbers nobody can know yet.

     The counts golden is not committed by this plan. This machine has no Loomyard checkout, so
     every Loomyard-gated test here skips and the file cannot be produced; the first run on a machine
     with a checkout at the pin regenerates it. When the golden is missing, this test must fail with
     a message naming the exact regeneration command rather than with a bare read error, so the state
     is attributable rather than mysterious. `compareGolden` cannot supply that message — it fatals
     on the failed read itself, and card 15 changes only its parameter type — so this test owns the
     mechanism: before calling it, and only when the update flag is not set, stat the golden's path
     and fatal with the regeneration command when it is absent. Gating on the flag is what keeps the
     regeneration run itself from tripping the very check that exists to explain it. The golden's
     path and the command are:

     ```
     internal/engine/testdata/loomyard/naming-counts.json
     LADDER_LOOMYARD_REPO=<checkout pinned at 72c23d9> go test ./internal/engine/ -run TestRoundTrip_LoomyardNaming -update
     ```

  Runtime budget, to be stated in the test's own doc comment: the maker adds one parse per harvested
  symbol, two for a head that takes the completion retry, but each parses a three-line synthetic file
  rather than a real source file, so the cost is linear in the symbol count with a small constant
  against the existing helper's per-unit whole-file passes. If the run nonetheless lands near go
  test's default timeout, the mitigation is to raise the timeout for this one test. Sampling the
  harvest is not an option: it would silently weaken the zero-misses criterion this test exists to
  assert.
- **Commit:** `test(engine): the prediction-equals-extraction round trip over the pinned checkout`

## Batch Tests

`verify: go test ./internal/engine/ -run 'TestRoundTrip'` runs the two existing round trips plus the
new naming one. The `-run` narrowing is scoped to what this batch touches: cards 14 and 15 edit
helpers those three tests are the only consumers of, and `internal/engine` holds a large suite of
unrelated cases the repo-wide `pipeline.done_gate` already covers before the task is marked done.

On the machine this task is implemented on, `TestRoundTrip_Loomyard` and `TestRoundTrip_LoomyardNaming`
both skip for want of a checkout, so the batch's real green signal here is
`TestRoundTrip_QuarryItself` — which does exercise cards 14 and 15's edits, since it consumes both
the widened collector and the extracted harvest helper. The naming round trip's own first real run
happens on a Loomyard-equipped machine, together with the counts golden's first generation.
