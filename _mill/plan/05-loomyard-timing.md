# Batch: loomyard-timing

```yaml
task: "resolve + expand (T4)"
batch: "loomyard-timing"
number: 5
cards: 5
verify: CGO_ENABLED=1 go test ./internal/engine/
depends-on: [3, 4]
```

## Batch Scope

This batch closes the task: the environment-gated timing assertion that turns the plan's twenty
glyphs across five units under 150 ms into a test, the benchmark kept beside it so a regression is
measurable rather than merely detectable, the spot check that shows `expand` collecting members from
more than one file of a real repository, and the two mechanical follow-throughs the task's Scope
names and closes — widening T3's Loomyard gate helper by one parameter type so a benchmark can call
it, and correcting the one sentence in T3's type-symbol doc comment that describes an `expand` this
task has now decided differently.

It is last because the timing test calls `Resolve` and the spot check calls `Expand`, so both verbs
must exist. It is one batch because all five cards are small, share the Loomyard gate as their
subject, and share one risk: a change to a T3 file. Both changes to T3 files are behaviour-preserving
by construction and each is its own card so a reviewer sees it in isolation, and each of the three
test artefacts is its own card and its own commit for the same reason.

**Batch prerequisite — an environment step, not a deliverable.** Cards 20, 21 and 22 all measure
against a checkout at the commit `loomyard_test.go`'s own pin names, and card 20 reads its twenty
glyph strings off it. The gate helper reads one thing and one thing only: the `LADDER_LOOMYARD_REPO`
environment variable. So before card 20 starts, resolve a path to a checkout at the pinned commit and
export it for every test run in this batch. Nothing is written and nothing is committed by this step —
no file in this repository reads that path, and no tracked file may carry it. If no checkout at the
pinned commit can be found on the host, stop and report that as the blocker rather than inventing
glyph strings: the list must be read off the real checkout or it is not a measurement, and none of the
three cards can be validated without it.

## Cards

### Card 19: widen the Loomyard gate to testing.TB

- **Context:**
  - `internal/engine/golden_test.go`
  - `internal/engine/roundtrip_test.go`
- **Edits:**
  - `internal/engine/loomyard_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Change `loomyardRepo`'s parameter type in `internal/engine/loomyard_test.go` from
  `*testing.T` to `testing.TB`, keeping the parameter's name as `t`. That is what makes this one line:
  the body's five calls — one helper marker, one skip, two formatted skips and one fatal — are every
  one of them methods on `testing.TB`, so renaming the parameter would rewrite the whole body for
  nothing. Change nothing else in the function, its doc comment or the file, and confirm every
  existing caller still compiles: the goldens and the round trip pass a `*testing.T`, which satisfies
  `testing.TB`.

  Card 21's benchmark is the reason for the widening. Duplicating the pin check into a second file was
  rejected: two implementations of "is this the right Loomyard" is exactly the drift that makes a
  done-criterion silently unverifiable when one is updated and the other is not.
- **Commit:** `refactor(engine): widen the Loomyard gate helper to testing.TB`

### Card 20: the twenty-glyph list and the timing assertion

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/loomyard_test.go`
  - `internal/engine/repo.go`
  - `internal/engine/resolve.go`
  - `internal/engine/roundtrip_test.go`
  - `internal/engine/toc_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/loomyard_timing_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/engine/loomyard_timing_test.go` with a file header comment in
  this package's register, the glyph list, and one test. The batch's Loomyard checkout prerequisite,
  stated in the Batch Scope above, is resolved before this card starts; this card reads its glyph
  strings off that checkout.

  Declare a package-level `var loomyardTwentyGlyphs = []string{...}` of exactly twenty glyph strings,
  four each from five Loomyard packages of differing size, every one of them read off the pinned
  checkout at implementation time. At least one of the five must be a large package — the plan's own
  measurement of a single glyph in a 35-file package is where four glyphs in one package is what an
  ungrouped implementation would blow the budget on, and the grouping guarantee is what the number
  depends on. Comment the list with which packages it draws from and that the pin is what keeps it
  from going stale silently.

  `TestResolve_TwentyGlyphsUnder150ms` checks `testing.Short()` and skips with a reason **first**,
  before calling `loomyardRepo` — the same order `TestRoundTrip_Loomyard` uses in
  `internal/engine/roundtrip_test.go`, and the order that matters: `loomyardRepo` *fails* rather than
  skips on a wrongly-pinned checkout, so gating on the checkout first would make this test fail under
  `-short` on a machine where the established Loomyard test skips. It then opens the checkout with the
  existing `openRepo` helper, as every other Loomyard-gated test in this package does, rather than
  calling `Open` and hand-rolling the error check. It resolves all twenty glyphs asserting every
  result's `Status` is `StatusFound` — a drifted
  glyph list would otherwise turn the criterion green by timing twenty misses — and times one
  `Resolve` call over the same twenty glyphs five times with `time.Since`, asserting the minimum
  elapsed is under 150 ms and reporting all five durations in the failure message. The minimum is the
  floor the criterion is about; a single run would fail on an unrelated build in another worktree, and
  an average or a percentile would measure the machine's load rather than the code.
- **Commit:** `test(engine): assert the twenty-glyph resolve timing against Loomyard`

### Card 21: the resolve benchmark

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/loomyard_test.go`
  - `internal/engine/repo.go`
  - `internal/engine/resolve.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/loomyard_timing_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `BenchmarkResolveTwentyGlyphs` to `internal/engine/loomyard_timing_test.go`,
  beside the timing test it mirrors. It calls the same `loomyardRepo` gate — passing a `*testing.B`,
  which is the reason card 19 widened that helper's parameter to `testing.TB` — opens the checkout
  once outside the timed loop with `Open` directly — not the `openRepo` helper cards 20 and 22 use,
  which takes a `*testing.T` a benchmark does not have; widening that helper too is not among this
  task's two named scope exceptions, so the benchmark calls `Open` and fails on its error with
  `b.Fatalf` — calls `b.ResetTimer`, and calls `Resolve` over
  `loomyardTwentyGlyphs` once per iteration, failing the benchmark on any error. It reuses the
  package-level glyph list card 20 declared and does not restate it.

  The benchmark is kept beside the assertion rather than replacing it, so a regression is measurable
  rather than merely detectable. It cannot replace the assertion: `go test` does not run benchmarks,
  so a criterion asserted only in a benchmark would never be checked by the gate.
- **Commit:** `test(engine): keep the twenty-glyph resolve call as a benchmark`

### Card 22: the expand spot check over a real repository

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
  - `internal/engine/loomyard_test.go`
  - `internal/engine/repo.go`
  - `internal/engine/roundtrip_test.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/loomyard_timing_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestExpand_LoomyardMembersAcrossFiles` to
  `internal/engine/loomyard_timing_test.go`. It checks `testing.Short()` and skips first, then calls
  `loomyardRepo` — the same gate order card 20 uses and for the same reason — opens the checkout with
  the existing `openRepo` helper, and expands one Loomyard type whose methods are known to live in more
  than one file at the pinned commit. Assert `Status` is `StatusFound`, `Head` is non-nil, `Members` is
  non-empty, and the
  members carry at least two distinct `File` values.

  That last assertion is the property the verb exists for and the one no committed fixture proves as
  convincingly — a fixture built to have members in two files proves the filter runs, where a real
  repository proves it finds what a reader would have had to open several files to see. Name the chosen
  type and the files its methods live in, at the pinned commit, in a comment on the test.
- **Commit:** `test(engine): spot-check expand's cross-file members against Loomyard`

### Card 23: correct the stale expand sentence in golang.go

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Correct one sentence in `goUngroupedTypeSymbol`'s doc comment in
  `internal/engine/golang.go`. It currently reads, across three wrapped comment lines, `because the
  later expand verb renders a type's head by omitting the lines its own member symbols cover, and
  subtracting those spans from the head range is that consumer's job, not this extractor's.` Replace
  that sentence with `because the expand verb emits a type's head as this one contiguous span
  alongside its member symbols' own spans, and subtracting those member spans from the head range is
  that consumer's job, not this extractor's.`, re-wrapping the comment lines to the file's existing
  width.

  This is a comment about this task's own behaviour, written before this task had decided it: the
  sentence's second half already agrees with what `Expand` does, and only the claim that the verb
  renders a subtraction describes something that will not exist. Change no code in this file, no
  other comment in it, and no other declaration. The whole card is one doc comment sentence, kept
  separate so a reviewer sees it in isolation.
- **Commit:** `docs(engine): correct the stale expand sentence in goUngroupedTypeSymbol`

## Batch Tests

`verify:` runs the whole `internal/engine` package suite. Card 19's widening and card 23's comment
correction are both behaviour-preserving, and the suite passing unchanged is precisely their
assertion — card 19's in particular, since a caller that no longer compiles is the only way that
change can go wrong. The three artefacts of cards 20, 21 and 22 are environment-gated: on a host with
no Loomyard checkout they skip with a reason and the suite is green, and on a host with the pinned
checkout and `LADDER_LOOMYARD_REPO` exported they run for real. The implementer must run the suite
both ways before committing each of those three cards — once with the variable unset, confirming the
skip, and once with it set to the pinned checkout, confirming card 20's twenty glyphs resolve `found`
and its timing assertion passes, card 21's benchmark runs under
`go test -bench . -run '^$' ./internal/engine/`, and card 22's spot check finds members in more than
one file — and state both outcomes in each commit body. A test that only ever skips is not a test.
Run the suite once under `-short` as well, confirming cards 20 and 22 skip on that flag alone rather
than reaching the checkout gate.
