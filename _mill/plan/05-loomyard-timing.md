# Batch: loomyard-timing

```yaml
task: "resolve + expand (T4)"
batch: "loomyard-timing"
number: 5
cards: 3
verify: CGO_ENABLED=1 go test ./internal/engine/
depends-on: [3, 4]
```

## Batch Scope

This batch closes the task: the environment-gated timing assertion that turns the plan's twenty
glyphs across five units under 150 ms into a test, the benchmark kept beside it so a regression is
measurable rather than merely detectable, and the two mechanical follow-throughs the task's Scope
names and closes — widening T3's Loomyard gate helper by one parameter type so a benchmark can call
it, and correcting the one sentence in T3's type-symbol doc comment that describes an `expand` this
task has now decided differently.

It is last because the timing test calls `Resolve` and the spot check calls `Expand`, so both verbs
must exist. It is one batch because all three cards are small, share the Loomyard gate as their
subject, and share one risk: a change to a T3 file. Both changes to T3 files are behaviour-preserving
by construction and each is its own card so a reviewer sees it in isolation.

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

  Card 20's benchmark is the reason for the widening. Duplicating the pin check into a second file was
  rejected: two implementations of "is this the right Loomyard" is exactly the drift that makes a
  done-criterion silently unverifiable when one is updated and the other is not.
- **Commit:** `refactor(engine): widen the Loomyard gate helper to testing.TB`

### Card 20: the Loomyard timing test, its benchmark and the expand spot check

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
  - `internal/engine/loomyard_test.go`
  - `internal/engine/resolve.go`
  - `internal/engine/toc_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/loomyard_timing_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/engine/loomyard_timing_test.go` with a file header comment in
  this package's register, and three artefacts, all gated through the existing `loomyardRepo` helper
  so the pin rule is not restated: it skips when the environment names no checkout and fails when it
  names one at the wrong commit.

  Before writing the glyph list, resolve a Loomyard checkout on this host. Read
  `LADDER_LOOMYARD_REPO` from the environment first; when it is unset, read simple key-equals-value
  lines from the gitignored `.scratch/ladder.env` beneath the repository root, the same file and the
  same precedence the ladder harness uses. When neither names a checkout at the pinned commit, locate
  one on the host and write its path into `.scratch/ladder.env`, which is gitignored and therefore
  carries no machine path into a tracked file. If no checkout at the pinned commit can be found at
  all, stop and report it rather than inventing glyph strings: the list must be read off the real
  checkout or it is not a measurement.

  Declare a package-level `var loomyardTwentyGlyphs = []string{...}` of exactly twenty glyph strings,
  four each from five Loomyard packages of differing size, every one of them read off the pinned
  checkout at implementation time. At least one of the five must be a large package — the plan's own
  measurement of a single glyph in a 35-file package is where four glyphs in one package is what an
  ungrouped implementation would blow the budget on, and the grouping guarantee is what the number
  depends on. Comment the list with which packages it draws from and that the pin is what keeps it
  from going stale silently.

  `TestResolve_TwentyGlyphsUnder150ms` calls `loomyardRepo`, skips when `testing.Short()` is set,
  opens the checkout with `Open`, and first resolves all twenty glyphs asserting every result's
  `Status` is `StatusFound` — a drifted glyph list would otherwise turn the criterion green by timing
  twenty misses. It then times one `Resolve` call over the same twenty glyphs five times with
  `time.Since` and asserts the minimum elapsed is under 150 ms, reporting all five durations in the
  failure message. The minimum is the floor the criterion is about; a single run would fail on an
  unrelated build in another worktree, and an average or a percentile would measure the machine's
  load rather than the code.

  `BenchmarkResolveTwentyGlyphs` calls the same `loomyardRepo` gate — the reason card 19 widened it —
  opens the checkout once outside the timed loop, resets the timer, and calls `Resolve` over the same
  twenty glyphs once per iteration, failing the benchmark on any error.

  `TestExpand_LoomyardMembersAcrossFiles` calls the same gate, skips under `testing.Short()`, and
  expands one Loomyard type whose methods are known to live in more than one file at the pinned
  commit, asserting `Status` is `StatusFound`, that `Members` is non-empty, and that the members carry
  at least two distinct `File` values. That is the property the verb exists for and the one no
  committed fixture proves as convincingly. Name the chosen type in a comment alongside the glyph
  list.
- **Commit:** `test(engine): assert the twenty-glyph resolve timing against Loomyard`

### Card 21: correct the stale expand sentence in golang.go

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

`verify:` runs the whole `internal/engine` package suite. Card 19's widening and card 21's comment
correction are both behaviour-preserving, and the suite passing unchanged is precisely their
assertion — card 19's in particular, since a caller that no longer compiles is the only way that
change can go wrong. Card 20's three artefacts are environment-gated: on a host with no Loomyard
checkout they skip with a reason and the suite is green, and on a host with the pinned checkout and
`LADDER_LOOMYARD_REPO` exported they run for real. The implementer must run the suite both ways
before committing card 20 — once with the variable unset, confirming the skips, and once with it set
to the pinned checkout, confirming the twenty glyphs resolve `found`, the timing assertion passes,
the benchmark runs under `go test -bench . -run '^$' ./internal/engine/`, and the expand spot check
finds members in more than one file — and state both outcomes in the commit body. A test that only
ever skips is not a test.
