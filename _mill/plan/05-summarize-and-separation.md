# Batch: summarize-and-separation

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "summarize-and-separation"
number: 5
cards: 3
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_summarize.py -q
depends-on: [1]
```

## Batch Scope

This batch delivers `summarize.py`: per-config medians and full ranges over every metric, the disjoint-range separation rule implemented and tested separately for each of the three comparison types the discussion defines, and the tracked `summary.json` that backs every claim the conclusion will make. It depends only on batch 1 because it reads the ladder for config identity and control resolution, and reads run artifacts off disk rather than through any other module.

The external interface batch 8 consumes is `summarize.py`'s CLI, invoked once after the matrix completes to write `summary.json`. Batch 9 reads that file and nothing else when writing the conclusion.

Batch-local decision: a comparison is a first-class value, not a formatted string. `Comparison` carries the two cell ids, the metric, both medians, both ranges, and a `separated` boolean, so the conclusion cites structured records and every claim in it is mechanically checkable against the tracked numbers.

## Cards

### Card 13: per-config medians, ranges, and completeness

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/scripts/gen_compact_toc.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add module-level `METRICS` — the ordered list of summarised metric names drawn from a run's `usage.json` and `score.json`: `duration_ms`, `num_turns`, `tool_uses`, `quarry_tool_uses`, `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `cost_usd`, `bash_grep_count`, `grep_tool_count`, `grep_fallback_total`, `recall`, and `precision` — and `GREP_METRICS`, the subset `bash_grep_count`, `grep_tool_count`, and `grep_fallback_total`.

  Add:
  - `load_runs(results_root, config_id, reps)` — read each complete run's `usage.json` and `score.json` under `raw/<config_id>/<n>/`, returning a list of flattened per-run metric mappings. A run directory that is not complete contributes nothing.
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
  - `compare_warm_cold(ladder, cells)` — compares the warm bundle cell against the cold cell, all metrics eligible. `kind` is `warm-vs-cold`.

  A rung is never compared against the other ladder's control, and the builders make that structural: `compare_rung_to_control` resolves the control from the config's own `ladder` field, and `compare_rungs` raises across ladders.

  Tests, asserted independently for each of the three kinds: overlapping ranges produce `separated: false` and disjoint ones `separated: true`; a rung-vs-control comparison set contains no entry for any metric in `GREP_METRICS` while a rung-vs-rung set over the same cells does; `compare_rungs` raises for a Ladder-A and a Ladder-B config; `compare_rung_to_control` on a Ladder-B rung resolves `b0-none` and never `a0-none`; an incomplete cell on either side yields no comparison at all; and two ranges touching at one endpoint are not separated.
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
  - `_meta` — the pinned run model, the pinned scorer mapping, `reps`, the results-root date segment, the number of configs, and a `denied_tool_attempts_reported` boolean read from the run directory's `probe.json`, so the summary states whether that metric carries information or was dropped.
  - `cells` — every config id mapped to its `Cell` stats plus `complete`, plus `worktree_dirtied_count` and `target_origin_quarry_mention_count` aggregated from the runs' recorded observations, so a cluster of dirtied runs is visible per config rather than only per run.
  - `comparisons` — every comparison the three builders produce, as a flat list of `Comparison` records.
  - `incomplete` — the ids of every cell that is not complete.

  Add `write_summary(ladder, results_root)` serialising it with sorted keys and a trailing newline, and a CLI under `if __name__ == "__main__":` taking the ladder path and the results root and writing `summary.json` into that root. The CLI exits non-zero, naming the incomplete cells, when any cell is incomplete — a summary of a partial matrix is written but must not be mistaken for a finished one.

  Tests build a synthetic results tree under `tmp_path` with complete and incomplete cells and assert: `_meta` records the pinned model and scorer; every config id appears in `cells`; `incomplete` lists exactly the short cells; `comparisons` contains all three kinds; `worktree_dirtied_count` aggregates the per-run observations; `denied_tool_attempts_reported` follows the probe record; and the CLI exit code is non-zero when a cell is incomplete and zero when none is.
- **Commit:** `feat(bench): emit the tracked ladder summary.json`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_summarize.py`, the only test file this batch creates. Every unit is pure over a synthetic results tree built in `tmp_path`, so no real run artifacts and no live model call are needed. The discussion's summarize scenarios map one-to-one onto the tests named in cards 13, 14, and 15: even and odd run counts, overlapping versus disjoint ranges asserted independently for all three comparison kinds, grep-metric exclusion for rung-vs-control only, no cross-ladder control resolution, and an incomplete cell reported rather than summarised from two runs.
