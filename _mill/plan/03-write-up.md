# Batch: write-up

```yaml
task: "Ladder, toc rerun (T7)"
batch: "write-up"
number: 3
cards: 4
verify: null
depends-on: [2]
```

## Batch Scope

This batch turns the matrix's three machine artifacts into the two things the rewrite actually
needed from T7: a conclusion that either reproduces the toc separation or says honestly why not, and
a repository whose standing documents agree with it. It is one batch because all four cards read the
same small artifact set — the summary, the provenance record and the rendered table from this
results root — and because the two document edits are three short paragraphs each that would be
meaningless without the conclusion's own numbers in front of them. Card 5 runs first and is the only
defence against a summariser that is confidently wrong; card 6 writes the conclusion; cards 7 and 8
propagate it.

Batch-local decision: nothing in this batch re-derives a verdict the harness already computed. The
separation claim is quoted from the summary's comparison entries; the medians and ranges are quoted
from the same place; the table is the artifact a reader is pointed at.

## Cards

### Card 5: Hand-verify one `a2-toc-dir` repetition against its transcript

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/raw/a2-toc-dir/1/transcript.jsonl`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/raw/a2-toc-dir/1/usage.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/raw/a2-toc-dir/1/run.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/summary.json`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** A zero-diff verification card that repeats, once, the spot check T2's own
  done-when required: confirm the metrics match the transcript by hand. Pick the first complete,
  non-blinding-failed `a2-toc-dir` repetition — repetition 1 if it is complete, otherwise the lowest
  numbered one that is, adjusting the paths above accordingly. If the cell finished with **no**
  complete repetition at all — an outcome the termination rule explicitly permits — skip the check
  and report that it could not be performed and why; card 6 then records the same, rather than
  quoting a verification that never happened. From its transcript, count the turns,
  count the calls to the granted `toc` tool, and read the cache-read figure; compare all three
  against the same repetition's recorded usage and against that cell's entry in the summary. Report
  agreement or the exact disagreement — a mismatch is a finding the conclusion must carry, and it
  would invalidate the numbers every other card quotes. Read `metrics.go` to confirm which record
  the harness derives each figure from, in particular that metrics come from the assistant records
  rather than from the final result record's `modelUsage`, which includes the tool's own overhead;
  do not re-derive any number from `modelUsage`. Write the findings to a scratch note under
  .scratch/ so card 6 can quote them. Make no file change in this card.
- **Commit:** none

### Card 6: Write the conclusion

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/provenance.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/table.txt`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/probe.md`
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/.gitignore`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `_mill/discussion.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the conclusion in the shape of the `v1-final` toc conclusions: a header
  naming the ladder file, the reps, the host, the server commit and the run count; a metric table;
  one section per ladder; a "what this settles" section; and a closing section on what went wrong and
  what to do next. The prior record is read with
  `git show origin/v1-final:bench/loomyard-eval/ladder/results/2026-09-02-toc/conclusion.md` — that
  git object is an explicitly permitted read for this card, both as the shape to follow and as the
  source of the prior figures.
  **The headline claim** — does `a2-toc-dir` separate from `a0-none` on turns and cache-read — is
  taken from the summary's comparison entries and their `separated` field, quoted alongside
  `median [min–max]` for each metric. Read the comparison whose metric is
  `cache_read_input_tokens`, not `cache_read`: the short spelling is only the rendered table's column
  header. Report medians and ranges as well as the verdict, and treat `separated` as evidence rather
  than as the sole criterion — it is a strict no-overlap test on the min–max ranges, and at n=5 a
  real effect can be present without it firing.
  **Print the correctness sample size separately from the cost sample size.** The rendered table
  already carries `recall_n` and `precision_n` columns and the max-turns and unscored flags, and a
  cell can report turns and cache-read at n=5 while recall sits at n=2, because a scorer failure and
  a max-turns completion both satisfy the harness's completeness test while shrinking a statistic.
  Never write "recall unchanged" without naming the sample size behind it; an unqualified version of
  that sentence at n=2 would be the single most damaging line the conclusion could contain.
  **The prior `v1-final` figures go in one clearly-labelled prior-record section** naming the branch,
  the root and the reps they come from, and never in this root's metric table. The comparison this
  conclusion is entitled to make on cost is qualitative — same direction, same rough magnitude —
  because cost numbers compare only within one results root. Correctness metrics are the stated
  exception and may be compared by id across roots; say so where that is done.
  **Cover every one of these regardless of outcome:** any cell short of 5 repetitions and why; every
  repetition invalidated by gate 2 check (a) or check (b) and why; any check (d)
  `CheckRenderedControlPrompt` failure, which is deterministic and stops the run outright; any
  `ScanMemoryPaths` abort, naming which file tripped it and what was done; any gate 1 finding, which
  tells the reader whether `a2-toc-dir` ever called the tool it was granted — a cell that never did
  measured prompt cost only and the conclusion must say so; the `quarry_commit` equality check across
  every entry of the provenance record's `invocations` list, and the out-of-band server-hash readings
  from the run log, both stated as the values that were checked rather than as "no warning fired";
  the `quarry_dirty` flag and `quarry_dirty_files` list of **each** invocation entry, with the
  carve-out named — a resumed invocation records dirty by construction because the first
  invocation's own machine artifacts are untracked under the results root until they are committed
  after the run, and every listed path being inside that root is what makes the flag benign; any
  session-fingerprint drift; the number of invocations the matrix took and whether any repetition
  was attempt-exhausted; the result of card 5's hand verification, or that it could not be performed
  because the cell finished with no complete repetition; and whether `output_tokens` is usable on
  this host, since it was unusable on the V1 host.
  **State the two negative-coverage caveats plainly.** `WarnOnServerHashDrift` cannot fire after a
  resume — the harness rewrites the per-repetition hash map on every invocation and leaves the
  per-invocation copy empty, so the map holds one distinct hash by construction — and gate 2 check
  (c)'s bare-token observation is expected to fire and means nothing on its own. Silence from either
  is not evidence and the conclusion must not present it as such.
  **Record plan §11's decision:** the raw tree stays untracked, confirming the `results/*/raw/`
  entry the ladder directory's ignore file already carries, and give both reasons — the resolved
  auto-memory paths that no tracked file may hold, and the size of ten 60-turn transcripts that the
  committed artifacts already summarise. Name the five files this root commits.
  If the matrix restarted into a `-r2` root, name the harness fix and the abandoned root, and
  confirm the abandoned directory carries the marker card 4 wrote there naming the fix, the date and
  this root as its successor. If the run stopped on a defect in the code
  under test, record the finding and say that fixing it is a separate task.
- **Commit:** `bench(ladder): T7 toc conclusion for the 2026-09-04-toc results root`

### Card 7: Update the rewrite plan's open decision on the raw tree

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`
  - `bench/loomyard-eval/ladder/.gitignore`
- **Edits:**
  - `docs/rewrite-plan.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In §11, the section headed "## 11. Open decisions", replace the third bullet —
  the one reading "Whether the new harness's `results/**/raw/` is committed or ignored; T2 decides
  when it writes its first root" — with the decision and its reason: the raw tree is ignored, settled by this
  task's first results root, because resolved auto-memory directory paths would otherwise be
  committed in a repository whose rule is that no tracked file carries a machine path, and because
  the raw tree is fully summarised by the artifacts the root does commit. Name the results root the
  decision was made in. Change nothing else in that section — the type-checker bullet and the C#
  parameter-list bullet both stay open — and change no other section of the document, in particular
  not §12's task table.
- **Commit:** `docs(plan): settle the open decision on the harness raw tree`

### Card 8: Bring the handoff document up to date with what T7 measured

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/table.txt`
  - `docs/rewrite-plan.md`
- **Edits:**
  - `HANDOFF.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Four edits, no more.
  (0) **§2, "The decisions the plan rests on".** The bullet beginning "T5 was split; the critical
  path ran" ends "Only T6 and T7 remain on it." T7 has now run, so that sentence contradicts the §4
  rewrite below; correct it in place to say the critical path is finished, in one line, keeping the
  bullet's T5b clause and restating nothing about what T7 measured. This is the one exception to the
  "change no other section" rule below, and it exists because leaving it would make the document
  disagree with itself.
  (1) **§3, "What was measured, and still holds".** The prose above the table says the runs are on
  the `v1-final` branch and that nothing on the parent branch reproduces them, "(the T7 rerun will)".
  Replace that parenthetical with what T7 found, and update the table's first row — the directory
  table of contents row — so it cites this results root and this run's own figures alongside the
  prior ones, keeping the two clearly separated as the conclusion keeps them. If the rerun did not
  reproduce the separation, say that in the same place rather than leaving the old claim standing.
  (2) **§4, "Next".** T7 is no longer pending: the critical path is finished. Rewrite the sentence
  that describes T7 as needing T2 and T6 into a statement of what it ran and what remains — wave 6's
  type checker, and the cleanup and grooming items already listed, which stay as they are.
  (3) **§5, "Open decisions (plan §11)".** Remove the raw-tree bullet, which is now decided, and
  point the reader at the conclusion and at the plan document's own updated bullet. The other two
  bullets stay.
  Do not restate the conclusion's numbers anywhere beyond the §3 table row — the handoff document
  points at records, it does not duplicate them. Change no section other than the four named above.
- **Commit:** `docs(handoff): record the T7 toc rerun result and close the raw-tree decision`

## Batch Tests

`verify: null`. Every card in this batch either reads artifacts or writes markdown; there is no
runnable surface a per-round command could exercise, and a Go test command here would verify code
none of the four cards touches. The batch's checks are card 5 — the hand verification, which is the
only thing standing between a confidently wrong summariser and a conclusion that quotes it — and the
internal consistency requirement on card 6, that every number it prints be quotable from the
summary, the provenance record or the rendered table committed beside it. The repository-wide gate
is encoded as `pipeline.done_gate` in the hub's mill configuration, holding
`go test ./... && golangci-lint run` with the live-test guard unset; it runs from the repository root
after this batch and covers the tree as a whole.
