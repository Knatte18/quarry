# Batch: summarize-report-cli

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "summarize-report-cli"
number: 7
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [6]
```

## Batch Scope

This batch turns a results root into the numbers a reader acts on: the summariser that re-derives
every metric from the raw tree, the per-cell table, and the two-subcommand binary. It is one batch
because the summariser's output shape and the table's columns are one decision, and the binary is
thirty lines of flag parsing over both. After it, the harness is complete as a program; batch 8 is
the test layer that proves it. The external interface batch 8 consumes is `Summarize`,
`WriteSummary`, `RenderTable` and the two subcommands.

Batch-local decision: the summariser reads the raw tree only — every metric is recomputed from each
repetition's transcript, never taken from the metrics file that repetition wrote. The stored metrics
are diagnostic; recomputing is exactly what keeping the transcripts buys, and it is what makes an
accounting fix cost a re-report rather than a re-run.

## Cards

### Card 28: the summariser

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `summarize.go`. Declare `MetricStats` carrying the
  median, minimum, maximum and sample count for one metric; `CellRecord` carrying the cell id, the
  ladder letter, the task id, the allowed list, the control flag, a metric-name to statistics map,
  the three counters `max_turns_count`, `unscored_count` and `blinding_failed_count`, and the cell's
  gate-1 finding when present; and `Summary` carrying the cells, the comparisons, an `incomplete`
  slice, an `invalid` slice and a meta block naming the results root's **base name only** and
  nothing else — no operator-supplied path and no wall-clock write time, per the overview's
  no-machine-paths-in-tracked-output decision, which is also what lets batch 8 compare this file
  byte-for-byte against a golden.
  `Summarize(resultsRoot string) (*Summary, error)` walks the raw tree, reads each repetition's
  state file for its cell metadata — the ladder letter, the allowed list, the control flag and the
  MCP prefix all come from there, so no ladder file is consulted — recomputes that repetition's
  cost metrics from its own transcript using the prefix the state file carries, and reads that
  repetition's recall and precision from its score record under the exact keys `recall` and
  `precision` — both scoring rules declare those two keys identically in their own fenced examples,
  which is why the summariser needs no per-schema mapping, and a record lacking either (every
  unscored stand-in, which carries a false scored flag instead) is excluded from the correctness
  medians by the exclusion rules below rather than treated as a zero. The remaining score fields are
  schema-specific — the exploration rule declares one more, the impact rule two — and are left in
  the score records without being aggregated: they answer questions this task's comparison does not
  ask, and the discussion fixes recall and precision as the correctness metrics.
  The two inputs are asymmetric on purpose:
  every cost metric is recomputed, which is what keeping the transcripts buys, while recall and
  precision are the scorer's judgment and exist nowhere else, so the score record is their only
  source. The metrics file is never read — it is diagnostic, and a summariser that trusted it would
  carry an accounting bug forward across a re-report.
  Exclusion rules, all three distinct: a repetition whose blinding-failed flag is set contributes
  **nothing at all**, neither cost nor correctness, and increments the blinding-failed counter; a
  repetition that hit the turn ceiling contributes its cost metrics but is excluded from the recall
  and precision medians and increments the max-turns counter; a repetition whose score record
  carries a false scored flag for any other reason is likewise excluded from those two medians and
  increments the unscored counter. A cell's cost sample count and its correctness sample count may
  therefore legitimately differ, and both are reported.
  The set of repetitions a root *should* contain is the provenance record's selected cells crossed
  with its effective repetition count; a cell short of that count, counting only repetitions that are
  present **and** not void, is added to the incomplete slice. A results root whose provenance record
  is **absent** — the reader returns a nil record and no error for that case — is an error naming the
  missing file, not an empty incomplete list: the run subcommand writes that record before its first
  repetition, so a raw tree without one is a hand-assembled or truncated root, and reporting an empty
  incomplete list for it would make an unknowably short root read as finished. That is exactly the
  failure the merge-never-overwrite policy exists to prevent, and it must not reappear here. Any cell with a non-zero
  blinding-failed count is added to the invalid slice. Port V1's `Comparison` and `RangesDisjoint`
  from `origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/summarize.go` unchanged in
  substance — a comparison holds between a rung cell and its own ladder letter's control, and
  disjointness is non-overlapping minimum-maximum ranges, with no significance testing. Compute gate
  1 per cell by calling `CheckGrantedToolUsed` with a `Config` value synthesised from the run-state
  fields — the cell id, the ladder letter, the task id and the allowed list, which is every field
  gate 1 reads — and the recomputed prefixed-tool-use counts. Synthesising it is what lets the
  summariser call the same gate the run never calls while still never loading the ladder file.
  `WriteSummary` writes to the name held by a package-level constant declared in this file,
  `SummaryFile = "summary.json"`, the single spelling of that name, matching how card 23 pins the
  provenance file's name and card 29 the table's. Do not exclude grep metrics from
  comparison: V1 had to because its two arms were given different steering, and this harness renders
  one preamble for every cell. `WriteSummary(resultsRoot string, s *Summary) error` writes the
  summary file at the results root.
- **Commit:** `feat(ladder): summarize a results root by recomputing from raw transcripts`

### Card 29: the per-cell table

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `report.go` with `RenderTable(s *Summary, p
  *Provenance) string` and `WriteTable(resultsRoot, table string) error`. The table opens with a
  header block carrying the results root's **base name only** — never the operator-supplied path,
  and no wall-clock time anywhere in the file, for the same two reasons the summary carries
  neither — the effective repetition count, the CLI version from the
  provenance record, and the cache caveat stated in the harness's own words: the first repetition of
  a root pays cache creation while later repetitions read it, so cache-read and cache-creation
  figures are reported separately, never summed, and per-repetition numbers are not interchangeable
  — the median over repetitions is the honest statistic. Then one row per cell, columns for the cell
  id, the ladder letter, the sample count, and the median of turns, duration, cost, the two cache
  figures, output tokens, total input tokens, tool uses, prefixed tool uses, grep fallbacks, tool
  result bytes and read bytes, followed by recall and precision with their own sample count. Flag a
  cell in its row when its blinding-failed count is non-zero, when its max-turns or unscored count
  is non-zero, and when gate 1 fired. Below the rows, print each gate-1 finding verbatim, each
  comparison and its disjointness verdict, the incomplete list, the invalid list, any server-hash
  drift warning and any session-fingerprint drift observations. Use fixed-width column formatting;
  do not depend on a table library. Declare `TableFile = "table.txt"` as a package-level constant in
  this file, the single spelling of that name, matching how card 23 pins the provenance file's name
  and card 28 the summary's; `WriteTable` writes to it. Both subcommands print this string to
  standard output and write it to that file at the results root, so the printed and the written
  table are the same bytes.
- **Commit:** `feat(ladder): render the per-cell table with the cache caveat header`

### Card 30: the two-subcommand binary

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create a thin `package main` with exactly two subcommands and no third.
  The run subcommand's flags are spelled exactly `--config` (required, the ladder file), `--results`
  (required, the results root), `--cells` (optional, a comma-separated cell-id list), `--reps`
  (optional, a repetition override) and `--claude-bin` (optional, defaulting to the bare name
  `claude` resolved on the path). Those spellings are not free: the migrated ladder file's header
  comment documents the entry point using them, and the hand-run done-criterion invokes it verbatim,
  so a renamed flag silently breaks the documented command. The report subcommand takes `--results`
  and nothing else — **no ladder-file flag at all**, because a results root is self-describing:
  every repetition's state file carries its own cell metadata.
  The run subcommand resolves the quarry repository root
  by calling `ResolveQuarryRepoRoot` with the process working directory — the single producer of
  that path, created in batch 5 — calls the run entry point, then summarises, writes the summary, and
  prints and writes the table. The report subcommand re-derives the summary and the table from the
  raw tree without running or scoring anything, and writes **both** at the results root through the
  same `WriteSummary` and `WriteTable` calls the run subcommand uses, printing the table to standard
  output as well — re-deriving without rewriting would leave a stale summary beside a fresh table,
  which is the opposite of what the subcommand exists for.
  Exit non-zero when the run entry point reports an incomplete cell or a cell with a blinding
  failure; exit non-zero on any returned error, printing it to standard error. Keep all logic in the
  package the overview's layout decision names: this file parses flags, wires them and exits. Do not
  add a shell wrapper, a configuration-file loader for flags, or any further subcommand — V1's
  twelve existed because an external orchestrator drove the loop one step at a time.
- **Commit:** `feat(ladder): add the run and report subcommands`

### Card 31: summariser and table tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `summarize_test.go` building small raw trees under a temporary directory
  from in-test state files and short transcripts, asserting: median, minimum, maximum and sample
  count over an odd and an even number of values; the disjointness predicate for overlapping and
  non-overlapping ranges; the incomplete slice populated when a cell has fewer present, non-void
  repetitions than the provenance record's selected cells crossed with its effective repetition
  count, and empty when it has exactly that many; a raw tree with no provenance record at all
  producing an error naming the missing file rather than an empty incomplete list; and the three
  exclusion rules asserted
  independently — a blinding-failed repetition contributes to neither the cost nor the correctness
  medians, is not counted as present for the incomplete calculation, and puts its cell in the invalid
  slice; a max-turns repetition contributes cost but not recall or precision and increments its own
  counter; an unscored repetition does the same and increments the other counter. Assert that a cell
  whose cost sample count and correctness sample count differ reports both rather than one.
  Write `report_test.go` rendering a table from a constructed summary and provenance value,
  asserting the header carries the cache caveat and the CLI version, that a cell with a non-zero
  blinding-failed count is flagged in its row, that a gate-1 finding is printed verbatim below the
  rows, and that the incomplete and invalid lists appear. Do not compare against a golden file
  here — the golden comparison belongs with the fixture root in the next batch.
- **Commit:** `test(ladder): cover the summariser exclusions and the table rendering`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers this batch through `summarize_test.go`
and `report_test.go`, over raw trees the tests build in a temporary directory. The three exclusion
rules are asserted separately on purpose: they are three different dispositions that a single
combined test would let collapse into one, and the blinding-failed rule is the one whose failure
would silently corrupt the baseline every later contrast is measured against.
