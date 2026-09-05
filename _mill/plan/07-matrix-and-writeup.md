# Batch: matrix-and-writeup

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: matrix-and-writeup
number: 7
cards: 4
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'
depends-on: [5, 6]
```

## Batch Scope

This batch runs the measurement and writes it up: generate the pack and inspect it, run the 3 × 10
matrix, compute the predeclared test by hand into a conclusion, and update the roadmap. It is one
batch because the four steps are strictly sequential against one results root and share one date, and
because splitting them would let the roadmap point at a root whose conclusion had not been written.

Two things about this batch are unlike every other batch in this plan. First, it is the only one that
spends real money: thirty measured invocations plus up to thirty scorer invocations against the live
API, at the run model and turn ceiling the ladder file pins. Second, most of its output is not code,
so its `verify:` is a regression net over the committed artefacts rather than a test of the work
itself — the work is checked by the procedures the cards name.

The results root is `bench/loomyard-eval/ladder/results/<YYYY-MM-DD>-kickstart/`, the date being the
day the pack command is run. The plan spells that date as `2026-09-05` in every path below. If the
pack is run on a different day, use the actual date and apply the same substitution to the results
root's own path, to the conclusion's heading, and to the roadmap line — the date is a fact about the
run, not a plan constant.

Batch-local decision: once rep 1 has run, nothing about the treatment may change. The glyph list, the
three cards, the task file, the fasit and n are frozen for the whole root. Every correction the
substitution rule allows happens before rep 0 or not at all.

## Cards

### Card 29: Generate the pack and inspect it

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/cards/07-e0-names.md`
  - `bench/loomyard-eval/cards/07-e2-files.md`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
- **Edits:**
  - `bench/loomyard-eval/cards/07-e1-pack.md`
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/pack-resolve.json`
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/provenance.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Run the pack subcommand against the new ladder file and the new results root, through the same
  bare go-run entry point the ladder file's own header documents.
  Then inspect its output before anything else happens. Confirm that all nine glyphs resolved found —
  the command halts on the first one that did not, so a clean exit is that confirmation — and read the
  rendered block in the treatment card to check that each line names a plausible file and span and
  that no docstring leaked into it.
  If a glyph did not resolve found, do not weaken the gate and do not edit the target repository.
  Replace the offending glyph with another symbol from the same package and the same mechanism —
  the discussion names three candidates held in reserve — and then, in this order: edit the glyph
  list in the ladder file; hand-edit the `Uses:` list in all three cards, since the pack command
  rewrites only the treatment card's own block and the other two would otherwise keep naming a symbol
  the treatment no longer lists, which is an arm difference in precisely the dimension under test;
  re-derive the descriptive card's file list from the new glyph set, deduplicated, since the
  substitute may live in a file no other glyph does or may vacate one; re-run the pack command;
  record the substitution and its reason in the task file's notes section; and re-run the fasit's
  cross-check. All of that happens before rep 0 or not at all.
  Check by eye that the three cards' `Uses:` lists are still identical after any such edit. The
  offline suite checks this too, so run it as the mechanical backstop rather than as a substitute for
  looking.
  Then commit the generated card together with the pack's two output files. Committing between the
  pack and the run is the expected workflow and is not a freshness violation: the run gate compares
  only the pack hash, never repository state, precisely so that doing the right thing here does not
  brick the root.
  The raw tree under the results root stays untracked, per the existing ignore rule scoped to that
  directory; only the tracked files named above are committed.
- **Commit:** `feat(bench): generate the kick-start pack for the e1 card`

### Card 30: Run the 3 × 10 matrix

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/provenance.json`
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/table.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Run the run subcommand against the new ladder file and the same results root, with no cell
  selection and no repetition override, so all three cells run at the file's own predeclared
  repetition count. The pre-repetition verification runs first and must pass; a failure there is a
  stop, not something to work around by regenerating the pack after the cards have been committed.
  The run is resumable. If it dies part way, re-run the same command against the same results root —
  completed repetitions are skipped and the invocation is appended to the provenance record. Do not
  pass a repetition override to a resumed run: the effective count is pinned from the pack onward and
  a differing value is refused, which is the correct behaviour under a locked n.
  When it finishes, read the printed table and the summary and record, per cell, the four
  dispositions the accounting distinguishes: passed the gate, failed the gate, ended at the turn
  ceiling, and unscored for any other reason. The last two are excluded from recall and precision but
  are real cost measurements and stay in the cost sample. A repetition the harness discarded before
  it produced a transcript — a fatal blinding finding, or an attempt loop exhausted to incomplete —
  is not a disposition of a measured repetition at all and is counted separately as a harness event.
  Every repetition that produced a transcript stays in the cost sample. Do not drop repetitions after
  seeing their scores, do not add repetitions after seeing any result, and do not re-run an arm
  because its numbers look wrong. Each of those is optional stopping through a side door, and the
  whole value of a predeclared rule is that it survives contact with the numbers.
  Commit the summary, the table and the updated provenance record. The raw tree stays untracked.
- **Commit:** `feat(bench): run the kick-start 3x10 matrix`

### Card 31: Write the conclusion

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-ladder-d/conclusion.md`
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/table.txt`
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/provenance.json`
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/conclusion.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Write the conclusion following the existing ladder-d conclusion's structure: the run's own
  identity and invocation facts, a statement that this root stands alone and is not pooled with any
  other, the per-cell medians and ranges quoted verbatim from the summary rather than re-derived, the
  predeclared rule restated, the per-repetition table, the rank sums, the U values, and a verdict.
  Restate the predeclared rule before the numbers, transcribed rather than reinterpreted: primary
  comparison is the treatment against the control, primary metrics are turns and cost, one-sided
  Mann–Whitney U with the alternative that the treatment is lower, ten per arm, alpha of five
  percent, critical U at or below twenty-seven. Reject the null for a metric only when its U is at or
  below that value. The descriptive arm gets medians and ranges only and no test.
  Show the arithmetic. The per-repetition values come from each repetition's own usage file; the
  ranks are computed with ties averaged; the two rank sums are checked against the identity that they
  total N(N+1)/2 and that the two U values total the product of the two sample sizes. Recompute the
  whole thing a second time from the per-repetition table before committing — that second pass is the
  only check on it, since no code computes it.
  Report, per arm and regardless of whether any threshold is crossed, the turn-ceiling count and the
  unscored count, so a clean matrix is visibly clean. Apply the predeclared readings: more than two
  of ten at the ceiling in any arm means the turns test is reported as censored and said so in the
  verdict line with cost carrying the primary comparison, and no re-run, re-sample or dropped arm;
  more than two of ten gate failures in any arm means that arm's cost numbers are called suspect and
  still reported; more than two of ten unscored in any arm means that arm's correctness accounting is
  called incomplete while its cost metrics stand.
  State plainly that recall and precision are descriptive only and are never compared across arms,
  and why: the treatment's card names the seven files verbatim in the prompt, so its file recall is
  inflated by construction. This is a known property of the design, stated here as such rather than
  discovered afterwards.
  State the other known asymmetry too: the treatment minus the descriptive arm is not a clean spans
  contrast, because the treatment also carries the signature and the parallel-read instruction.
  Secondary observations — read bytes, wall time, and recall of the listed symbols in the answer —
  are reported and never tested.
  Write the same shape of document whichever way the result falls. A negative answer is a valid,
  publishable answer, and if the treatment does not separate either, the surface has now been
  measured from both directions and the parked condition is closed twice over; if it does separate,
  the win belongs to pre-resolution rather than to a tool, which is a different product from the one
  the parked task assumed. Say which of those two the numbers support, in the verdict, in one
  sentence.
- **Commit:** `docs(bench): write the kick-start matrix conclusion`

### Card 32: Update the roadmap

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/conclusion.md`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:**
  - `docs/roadmap.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Two edits, one commit.
  Strike the small-and-independent bullet about creating the results root before writing the
  provenance record. It is done: the create now happens inside the write itself, which is why the
  bullet's own symptom — a fresh results path failing on rep 0 — cannot recur.
  Add one line to the standing-rule paragraph at the top, alongside the two conclusions it already
  cites, pointing at the new results root's conclusion and saying in one clause what it measured and
  what it found. That paragraph is the file's measured record; this is the third entry in it.
  Update the file's own "updated" date line to the date of this change.
  Change nothing else. In particular, whether the parked task stays parked is the operator's call and
  is already recorded as such; this card reports a measurement and does not decide that question.
- **Commit:** `docs: record the kick-start measurement in the roadmap`

## Batch Tests

`verify:` runs `go test` against `./bench/loomyard-eval/ladder/internal/ladder/` with
`-run 'TestPreMatrix'`. That is a regression net, not a test of this batch's work: the pre-matrix
suite reads the committed ladder file, the three cards and the fasit, so it catches any edit the
substitution rule makes in card 29 that breaks the identical-`Uses:`-lists property, the sentinel
protocol, the fasit's shape or the three prompts' blindness. It cannot be a test of the measurement
itself, because the measurement is thirty live API calls.

The work in this batch is verified by procedure instead, and each procedure is named in its own card:
the pack command's own pre-rep-0 gate is what confirms every glyph resolves found; the run's
pre-repetition verification is what confirms the pack in the prompt is the pack provenance recorded;
and the conclusion's arithmetic is confirmed by recomputing it a second time from the per-repetition
table, with the two rank-sum identities as the check.

The repository-wide gate configured for this task — the full test suite followed by the linter — runs
at the end of the task and is what confirms nothing in the seven batches regressed anything outside
the harness package.
