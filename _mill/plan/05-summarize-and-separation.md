# Batch: summarize-and-separation

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "summarize-and-separation"
number: 5
cards: 3
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_summarize.py -q
depends-on: [1, 2, 3, 4]
```

## Batch Scope

This batch delivers `summarize.py`: per-config medians and full ranges over every metric, the disjoint-range separation rule implemented and tested separately for each of the three comparison types the discussion defines, and the tracked `summary.json` that backs every claim the conclusion will make. It depends on batch 1 for config identity and control resolution, on batch 3 for run completeness (`load_runs` calls `gates.is_complete` rather than re-deriving what a complete run is, so there is exactly one definition of it in the suite), and on batches 2 and 4 for the artifact schemas it reads: every metric it summarises is a field `extract_usage` writes into `usage.json` or `score_run` writes into `score.json`, so those two modules define the shape this batch consumes.

The external interface batch 8 consumes is `summarize.py`'s CLI, invoked once after the matrix completes to write `summary.json`. Batch 9 reads that file and nothing else when writing the conclusion.

Batch-local decision: a comparison is a first-class value, not a formatted string. `Comparison` carries the two cell ids, the metric, both medians, both ranges, and a `separated` boolean, so the conclusion cites structured records and every claim in it is mechanically checkable against the tracked numbers.

## Cards

### Card 13: per-config medians, ranges, and completeness

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/scripts/gen_compact_toc.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add module-level `METRICS` — the ordered list of summarised metric names drawn from a run's `usage.json` and `score.json`: `duration_ms`, `wall_clock_ms`, `num_turns`, `tool_uses`, `quarry_tool_uses`, `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `cost_usd`, `bash_grep_count`, `grep_tool_count`, `grep_fallback_total`, `denied_tool_attempts`, `recall`, `precision`, and `lookalikes_matched` — and `GREP_METRICS`, the subset `bash_grep_count`, `grep_tool_count`, and `grep_fallback_total`.

  Two of `score.json`'s fields are deliberately **not** in `METRICS`, because a median and a range are meaningless for them, and each gets its own per-cell treatment instead:
  - `decoy_admitted` is a boolean, summarised as a per-cell `decoy_admitted_count` — how many of that config's runs listed the fasit's lookalike as a real caller. It is reported as its own column and never folded into precision, exactly as the impact rule requires: task 04's own scoring notes call admitting `burler.go:373` materially worse than missing a real caller, and averaging it into a ratio would hide the single finding the task was built to surface.
  - `summary_matches` is a qualitative judgement, carried into the cell verbatim as the list of its runs' values rather than reduced to a number.

  `lookalikes_matched` is a count and so does summarise normally. Card 13's own "a metric absent from a run" sentence refers to exactly these per-schema differences: `recall` and `precision` exist on both schemas, `lookalikes_matched` only on impact runs, and `summary_matches` only on exploration runs.

  The four token classes are **flattened out of `usage.json`'s nested `tokens` mapping**: `extract_usage` writes them under `tokens.input_tokens` and siblings, and `METRICS` names them flat, so `load_runs` lifts each `tokens.<class>` to a top-level key of the per-run mapping. Every other metric is already top-level in `usage.json` or `score.json` and is copied across unchanged.

  `wall_clock_ms` is summarised alongside `duration_ms` because the sequential-dispatch rationale rests on wall-clock comparability and the two measure different spans — the client's own view of the run, and the harness's measurement around the whole subprocess including startup. `denied_tool_attempts` is summarised because the summary's own metadata carries a boolean saying whether it means anything; a metadata flag qualifying a metric the summary never reports would qualify nothing.

  Add:
  - `load_runs(results_root, config_id, reps)` — for each `raw/<config_id>/<n>/` directory, call `gates.is_complete` and skip it when that returns `False`; for the rest, read `usage.json`, `score.json`, **and** `run.json`, returning a list of flattened per-run mappings carrying that run's metrics and the observations `run.json` records — `worktree_dirtied`, `target_origin_quarry_mention`, and `cold_no_daemon_backed_call`. Completeness is never re-derived here; `gates.is_complete` is the single definition.
  - `Cell` — a dataclass carrying `config_id`, `runs` (the loaded list), `complete` (`len(runs) == reps`), and `stats`, a mapping from metric name to a record of `median`, `min`, `max`, and `n`.
  - `summarise_cell(config_id, runs, reps)` — build the `Cell`. Median is the true median: the middle value at odd `n`, the mean of the two middle values at even `n`. A metric absent from a run (`cost_usd` when the envelope omitted it, the impact-only fields on a Ladder-A run) is skipped for that run and its `n` reflects the runs that carried it.
  - Cells that are not complete are still summarised, but every downstream comparison involving them is refused; the cell records `complete: false` so the summary reports it as incomplete rather than silently summarising from two runs.

  Tests: an odd run count takes the middle value and an even count the mean of the two middle; a config with only two of three runs present is `complete: false` and its stats record `n: 2`; a metric missing from one run reduces that metric's `n` without affecting the others; and `load_runs` ignores a run directory whose `run.json` does not record `state: "complete"`.
- **Commit:** `feat(bench): summarise per-config medians and ranges`

### Card 14: the three disjoint-range comparison types

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add the `Comparison` dataclass (`kind`, `left`, `right`, `metric`, `left_median`, `right_median`, `left_range`, `right_range`, `separated`) and `ranges_disjoint(a, b)` — `True` only when the two closed ranges do not overlap at all; ranges that touch at a single value are not disjoint.

  Add one builder per comparison type, each refusing to emit a comparison when either cell is incomplete:
  - `compare_rung_to_control(ladder, cells, config)` — compares a rung against its own ladder's control, resolved through `control_for` rather than by parsing the id. Emits comparisons for every metric in `METRICS` **except** those in `GREP_METRICS`, which are excluded from this comparison type entirely because the control's preamble differs in steering as well as tools. `kind` is `rung-vs-control`.
  - `compare_rungs(ladder, cells, left_id, right_id)` — compares two configs in the same ladder, both with a non-empty allowed set. All metrics including the grep metrics are eligible, since these preambles are identical except for the tool list. `kind` is `rung-vs-rung`. Raises `SummarizeError` when the two configs are not in the same ladder.
  - `compare_warm_cold(ladder, cells, cold_disposition)` — compares the cold cell against the warm cell named by its own `warm_counterpart` field, resolved through `warm_counterpart_for`, all metrics eligible. `kind` is `warm-vs-cold`. The disposition is an explicit argument, supplied by `build_summary` from the record it already reads, rather than something this function goes to disk for. It emits no comparison when that disposition is `not-run` or `partial` — a disjoint-range claim at `n = 3` cannot be made from fewer than three cold runs — and none when the cold cell's runs recorded `cold_no_daemon_backed_call` for every repetition: none of them started a daemon, so there is no warmth contrast to draw and reporting one would read a difference that cannot be about warmth.

  A rung is never compared against the other ladder's control, and the builders make that structural: `compare_rung_to_control` resolves the control from the config's own `ladder` field, and `compare_rungs` raises across ladders.

  Tests, asserted independently for each of the three kinds: overlapping ranges produce `separated: false` and disjoint ones `separated: true`; a rung-vs-control comparison set contains no entry for any metric in `GREP_METRICS` while a rung-vs-rung set over the same cells does; `compare_rungs` raises for a Ladder-A and a Ladder-B config; `compare_rung_to_control` on a Ladder-B rung resolves `b0-none` and never `a0-none`; `compare_warm_cold` resolves its warm side through the declared `warm_counterpart` field and emits nothing when every cold run recorded `cold_no_daemon_backed_call`; an incomplete cell on either side yields no comparison at all; and two ranges touching at one endpoint are not separated.
- **Commit:** `feat(bench): implement the three disjoint-range comparison types`

### Card 15: emit the tracked `summary.json`

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add `build_summary(ladder, results_root)` returning the full mapping written to `summary.json`:
  - `_meta` — the pinned run model, the pinned scorer mapping, `reps`, the results-root date segment, the number of configs, and the cold cell's disposition, read from `<results_root>/cold_cell.json` as written by the cold-cell driver, or the literal `"unknown"` when that file is absent — one of `confirmed-cold`, `not-run`, `no-daemon-signal`, or `partial`, with the count of confirmed-cold repetitions — so the summary states in its own metadata why a warm-versus-cold comparison is present or absent; and a `denied_tool_attempts_reported` boolean — `null` when the probe record is absent — read from `<results_root>/probe.json` — the probe record at the results root, not in any run directory — as the value of that file's **`denied_tools_advertised`** key, which is the field name the probe writer uses. When denied tools are hidden from the model the metric can never be non-zero, so the flag is false and the summary says the column carries no information.
  - `cells` — every config id mapped to its `Cell` stats plus `complete`, plus `decoy_admitted_count`, the verbatim `summary_matches` list, `worktree_dirtied_count`, `target_origin_quarry_mention_count`, and `daemon_backed_runs` aggregated from the observations `load_runs` reads out of each run's `run.json`, so a cluster of dirtied runs is visible per config rather than only per run, and the cold cell's warmth signal — or its absence — is visible in the tracked numbers.
  - `comparisons` — every comparison the three builders produce, as a flat list of `Comparison` records.
  - `incomplete` — the ids of every cell that is not complete.

  Neither missing file raises: summarising after a `--stage main`-only run is a legitimate intermediate state, and a summariser that refused to run on it would be unusable exactly when an operator wants to look at the matrix so far. A `"unknown"` cold disposition means the cold cell has not run yet, so its cell is treated as incomplete in the ordinary way rather than excused. A cold cell whose `cold_cell.json` records `not-run` or `partial` is **excluded from the `incomplete` list**: both are legitimate terminal states of the cold-cell driver rather than interrupted runs, and without that record neither is distinguishable from a cell the operator stopped half way. Any other cell with fewer than `reps` complete runs is incomplete as before.

  Add `write_summary(ladder, results_root)` serialising it with sorted keys and a trailing newline, and a CLI under `if __name__ == "__main__":` taking the ladder path and the results root and writing `summary.json` into that root. The CLI exits non-zero, naming the incomplete cells, when any cell is incomplete — a summary of a partial matrix is written but must not be mistaken for a finished one — and exits zero when the only short cell is a `not-run` cold cell.

  Tests build a synthetic results tree under `tmp_path` with complete and incomplete cells and assert: `_meta` records the pinned model and scorer; every config id appears in `cells`; `incomplete` lists exactly the short cells; `comparisons` contains all three kinds; `worktree_dirtied_count`, `decoy_admitted_count`, and `daemon_backed_runs` aggregate the per-run observations and score fields; `summary_matches` is carried through verbatim rather than reduced; `denied_tool_attempts_reported` follows the probe record and qualifies a metric the cells actually carry; a `not-run` cold cell and a `partial` one are both absent from `incomplete` and neither makes the CLI exit non-zero, while a cold cell short for any other reason does; `compare_warm_cold` emits nothing for a `partial` cell; an absent `cold_cell.json` yields `"unknown"` and an absent `probe.json` yields a null flag, both without raising, and the cold cell is then incomplete in the ordinary way; and the CLI exit code is non-zero when a cell is incomplete and zero when none is.
- **Commit:** `feat(bench): emit the tracked ladder summary.json`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_summarize.py`, the only test file this batch creates. Every unit is pure over a synthetic results tree built in `tmp_path`, so no real run artifacts and no live model call are needed. The discussion's summarize scenarios map one-to-one onto the tests named in cards 13, 14, and 15: even and odd run counts, overlapping versus disjoint ranges asserted independently for all three comparison kinds, grep-metric exclusion for rung-vs-control only, no cross-ladder control resolution, and an incomplete cell reported rather than summarised from two runs.
