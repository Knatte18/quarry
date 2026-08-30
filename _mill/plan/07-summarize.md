# Batch: summarize

```yaml
task: Port the capability-ladder bench harness to Go
batch: summarize
number: 7
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [5]
```

## Batch Scope

Ports `summarize.py` in full: the metric tuples with the two dropped entries, per-cell loading and
median statistics, the disjoint-range comparison machinery, the three comparison families, and summary
building with its non-zero exit on an incomplete cell. It also threads the provisional marker on
`denied_tool_attempts` from a run's usage record onto the summarised cell's stats record.

The external interface the CLI batches consume is `BuildSummary`, `WriteSummary`, and
`SummaryExitCode`.

Batch-local decision: `wall_clock_ms` and `cost_usd` leave the metric tuple rather than surviving as
permanent nulls. Neither has a source under Agent dispatch, and a metric column that is always null
reads as a measurement that failed rather than one that was never taken.

## Cards

### Card 31: Metric constants, cell loading, and per-cell statistics

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `METRICS` as `Metrics`, dropping `wall_clock_ms` and `cost_usd` and keeping
  every remaining entry's definition unchanged; port `GREP_METRICS` as `GrepMetrics` and
  `_RUN_JSON_OBSERVATIONS` as an unexported `runJSONObservations`. Port `SummarizeError` as an exported
  `SummarizeError` type, `Cell` as a struct, `load_runs` as `LoadRuns` keeping its
  `tokens.<class>` flattening, `_median` as an unexported `median`, and `summarise_cell` as
  `SummariseCell`. `LoadRuns` reads only `run.json`-complete runs — the ingest marker is invisible to
  summarisation. Its observation lift reads the top-level keys the run-marker payload now writes, which
  is what makes the lift fire at all; in the Python those keys are never written, so the lift and every
  metric downstream of it are dead. Note that in the doc comment so a later reader does not
  "restore" the Python's behaviour. Add the one shape change the new metric partition requires: a cell's
  `denied_tool_attempts` stats record carries a provisional marker through from the runs it summarised,
  set when any contributing run's usage record carries it. Test the median at odd and even lengths, the
  token flattening, the incomplete-cell path, and the provisional marker propagating in and clearing
  out.
- **Commit:** `feat(ladder): port metric constants, cell loading, and cell statistics`

### Card 32: Disjoint ranges and the comparison value type

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `Comparison` as a struct, `ranges_disjoint` as
  `RangesDisjoint(a, b [2]float64) bool`, and `_build_comparison` as an unexported `buildComparison`.
  `RangesDisjoint` treats ranges that touch at a single shared value as **not** disjoint — the doc
  comment must state that, and a test case must pin it. Test disjoint, overlapping, and
  touching-at-one-value ranges, plus `buildComparison` over two synthetic cells.
- **Commit:** `feat(ladder): port disjoint-range comparison machinery`

### Card 33: The three comparison families

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `compare_rung_to_control` as `CompareRungToControl`, `compare_rungs` as
  `CompareRungs`, and `compare_warm_cold` as `CompareWarmCold`. The grep metrics are excluded from
  every rung-versus-control comparison, because the control's preamble differs in steering as well as
  in tools, but they stay eligible for rung-versus-rung — encode that asymmetry explicitly rather than
  as a shared filter, and record the reason in the doc comment. `CompareRungs` keeps its same-ladder
  guard. `CompareWarmCold` keeps its suppression on a cold disposition that is not-run or partial, and
  on the all-runs-had-no-daemon-backed-call case. Test the grep exclusion applying to rung-versus-control
  and not to rung-versus-rung, the same-ladder guard rejecting a cross-ladder pair, and each warm-cold
  suppression condition.
- **Commit:** `feat(ladder): port the three comparison families`

### Card 34: Summary building, writing, and the incomplete exit code

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_read_json_or_default` as an unexported `readJSONOrDefault`, `build_summary`
  as `BuildSummary`, with one field dropped: the summary's `denied_tool_attempts_reported` meta flag
  goes away rather than being retargeted. Its source was the probe record's advertised-tools key, which
  was derived from the client's own advertised tool list — a list that has no counterpart under agent
  dispatch, which is the same reason the transcript-sourced advertised-tools field was replaced by a
  definition-sourced granted-tools field. The deny-list probe's generated definition now grants the
  denied tool by construction, so the question the flag asked is answered by the definition rather than
  observed, and the probe record's observed-denial-shape key is the real successor signal. Record that
  reasoning in the doc comment rather than leaving the field reading a key nothing writes. Port
  `write_summary` as `WriteSummary`, and `_exit_code_for_summary` as
  `SummaryExitCode`, keeping the non-zero exit when any cell is incomplete and the incomplete-cell list
  the summary carries. Do not port `main` — the command-line entry point is a cobra subcommand added in
  a later batch. Test a summary built over a synthetic complete results tree, one with an incomplete
  cell producing the non-zero code and naming the cell, and that writing round-trips.
- **Commit:** `feat(ladder): port summary building and the incomplete exit code`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `summarize_test.go` plus every other test
file in the ladder subtree. The summary tests build synthetic results trees in the test's temp
directory; no committed results tree is read.
