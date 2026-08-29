"""
Summarises the quarry-mcp capability ladder's completed matrix into the
tracked `summary.json`: per-config medians and full ranges over every
metric `extract_usage.py` and `score_run.py` write, and the disjoint-range
separation rule applied independently to the three comparison types the
discussion defines -- rung-vs-control, rung-vs-rung, and warm-vs-cold.

Completeness is never re-derived here: `load_runs` calls `gates.is_complete`
so there is exactly one definition of a complete run across the suite.

Usage:
    from summarize import build_summary, write_summary
    write_summary(ladder, results_root)
"""
import argparse
import json
import sys
from dataclasses import asdict, dataclass
from pathlib import Path

from gates import is_complete, run_dir
from ladder_config import config_by_id, control_for, load_ladder, warm_counterpart_for

# Ordered metric names summarised into a median/min/max/n record. Drawn from
# every run's usage.json and score.json, except decoy_admitted and
# summary_matches, which are qualitative or boolean and get their own
# per-cell treatment in build_summary instead of a meaningless median.
METRICS = (
    "duration_ms",
    "wall_clock_ms",
    "num_turns",
    "tool_uses",
    "quarry_tool_uses",
    "input_tokens",
    "output_tokens",
    "cache_read_input_tokens",
    "cache_creation_input_tokens",
    "cost_usd",
    "bash_grep_count",
    "grep_tool_count",
    "grep_fallback_total",
    "denied_tool_attempts",
    "recall",
    "precision",
    "lookalikes_matched",
)

# The subset of METRICS excluded from a rung-vs-control comparison: the
# control's preamble differs in steering as well as tools, so a grep-usage
# gap between it and a rung cannot be attributed to the tool exposure alone.
GREP_METRICS = ("bash_grep_count", "grep_tool_count", "grep_fallback_total")

# run.json observations load_runs lifts onto each per-run mapping, recorded
# by the harness as non-fatal gate findings rather than summarised metrics.
_RUN_JSON_OBSERVATIONS = ("worktree_dirtied", "target_origin_quarry_mention", "cold_no_daemon_backed_call")


class SummarizeError(Exception):
    """Raised when a comparison is requested across configs that cannot be
    compared -- e.g. compare_rungs given two configs from different
    ladders."""


""" PER-CONFIG MEDIANS, RANGES, AND COMPLETENESS """


@dataclass(frozen=True)
class Cell:
    """One config's summarised results.

    Instance variables:
        config_id: the LadderConfig id this cell belongs to.
        runs: the list of flattened per-run mappings load_runs read.
        complete: True only when len(runs) == the config's reps.
        stats: mapping from METRICS name to a record of median/min/max/n,
            present only for metrics carried by at least one run.
    """

    config_id: str
    runs: list
    complete: bool
    stats: dict


def load_runs(results_root, config_id, reps):
    """
    Reads every complete run of config_id under
    <results_root>/raw/<config_id>/<n>/ for n in 1..reps.

    A run directory is skipped when gates.is_complete returns False for it
    -- this is the suite's single definition of completeness, never
    re-derived here. For each complete run, usage.json, score.json, and
    run.json are merged into one flat mapping: usage.json's tokens.<class>
    fields are lifted to top-level <class> keys, every other usage.json and
    score.json field is copied unchanged, and run.json's worktree_dirtied,
    target_origin_quarry_mention, and cold_no_daemon_backed_call
    observations are carried across when present.

    Returns:
        A list of flattened per-run mappings, in repetition order, one per
        complete run found.
    """
    runs = []
    for n in range(1, reps + 1):
        a_run_dir = run_dir(results_root, config_id, n)
        if not is_complete(a_run_dir):
            continue

        with open(a_run_dir / "usage.json") as f:
            usage = json.load(f)
        with open(a_run_dir / "score.json") as f:
            score = json.load(f)
        with open(a_run_dir / "run.json") as f:
            run_record = json.load(f)

        merged = {}
        for key, value in usage.items():
            if key == "tokens":
                merged.update(value)
            else:
                merged[key] = value
        merged.update(score)
        for observation in _RUN_JSON_OBSERVATIONS:
            if observation in run_record:
                merged[observation] = run_record[observation]

        runs.append(merged)
    return runs


def _median(values):
    """
    The true median of values: the middle value at odd length, the mean of
    the two middle values at even length.
    """
    ordered = sorted(values)
    count = len(ordered)
    midpoint = count // 2
    if count % 2 == 1:
        return ordered[midpoint]
    return (ordered[midpoint - 1] + ordered[midpoint]) / 2


def summarise_cell(config_id, runs, reps):
    """
    Builds the Cell for config_id from its already-loaded runs.

    A metric absent from a run -- cost_usd when the envelope omitted it, the
    impact-only fields on a Ladder-A run -- is skipped for that run, so its
    stats record's n reflects only the runs that carried it rather than the
    full run count.
    """
    stats = {}
    for metric in METRICS:
        values = [run[metric] for run in runs if metric in run and run[metric] is not None]
        if not values:
            continue
        stats[metric] = {
            "median": _median(values),
            "min": min(values),
            "max": max(values),
            "n": len(values),
        }
    return Cell(config_id=config_id, runs=runs, complete=len(runs) == reps, stats=stats)


""" THE THREE DISJOINT-RANGE COMPARISON TYPES """


@dataclass(frozen=True)
class Comparison:
    """
    One structured, mechanically checkable comparison between two cells over
    one metric.

    Instance variables:
        kind: "rung-vs-control", "rung-vs-rung", or "warm-vs-cold".
        left: the left-hand config id.
        right: the right-hand config id.
        metric: the METRICS name compared.
        left_median, right_median: each side's median.
        left_range, right_range: each side's (min, max) tuple.
        separated: True only when left_range and right_range do not overlap
            at all.
    """

    kind: str
    left: str
    right: str
    metric: str
    left_median: float
    right_median: float
    left_range: tuple
    right_range: tuple
    separated: bool


def ranges_disjoint(a, b):
    """
    True only when the two closed ranges (min, max) do not overlap at all.
    Ranges that touch at a single value -- one's max equals the other's min
    -- are not disjoint.
    """
    a_min, a_max = a
    b_min, b_max = b
    return a_max < b_min or b_max < a_min


def _build_comparison(kind, left_cell, right_cell, metric):
    """Returns one Comparison for metric between left_cell and right_cell,
    or None when either side carries no stats for that metric."""
    left_stats = left_cell.stats.get(metric)
    right_stats = right_cell.stats.get(metric)
    if left_stats is None or right_stats is None:
        return None

    left_range = (left_stats["min"], left_stats["max"])
    right_range = (right_stats["min"], right_stats["max"])
    return Comparison(
        kind=kind,
        left=left_cell.config_id,
        right=right_cell.config_id,
        metric=metric,
        left_median=left_stats["median"],
        right_median=right_stats["median"],
        left_range=left_range,
        right_range=right_range,
        separated=ranges_disjoint(left_range, right_range),
    )


def compare_rung_to_control(ladder, cells, config):
    """
    Compares config against its own ladder's control, resolved through
    control_for rather than by parsing config.id. Emits a comparison for
    every METRICS entry except GREP_METRICS, since the control's preamble
    differs in steering as well as tools. Emits nothing when either cell is
    incomplete.
    """
    control = control_for(ladder, config)
    rung_cell = cells[config.id]
    control_cell = cells[control.id]
    if not rung_cell.complete or not control_cell.complete:
        return []

    comparisons = []
    for metric in METRICS:
        if metric in GREP_METRICS:
            continue
        comparison = _build_comparison("rung-vs-control", rung_cell, control_cell, metric)
        if comparison is not None:
            comparisons.append(comparison)
    return comparisons


def compare_rungs(ladder, cells, left_id, right_id):
    """
    Compares two configs in the same ladder, both with a non-empty allowed
    set expected. All METRICS are eligible, including the grep metrics,
    since these preambles are identical except for the tool list.

    Raises SummarizeError when left_id and right_id are not in the same
    ladder. Emits nothing when either cell is incomplete.
    """
    left_config = config_by_id(ladder, left_id)
    right_config = config_by_id(ladder, right_id)
    if left_config.ladder != right_config.ladder:
        raise SummarizeError(f"compare_rungs: {left_id!r} and {right_id!r} are not in the same ladder")

    left_cell = cells[left_id]
    right_cell = cells[right_id]
    if not left_cell.complete or not right_cell.complete:
        return []

    comparisons = []
    for metric in METRICS:
        comparison = _build_comparison("rung-vs-rung", left_cell, right_cell, metric)
        if comparison is not None:
            comparisons.append(comparison)
    return comparisons


def compare_warm_cold(ladder, cells, cold_disposition):
    """
    Compares the ladder's cold cell against the warm cell named by its own
    warm_counterpart field, resolved through warm_counterpart_for.

    Emits nothing when cold_disposition is "not-run" or "partial" -- a
    disjoint-range claim at n = reps cannot be made from fewer than reps
    cold runs -- when either cell is incomplete, or when every cold run
    recorded cold_no_daemon_backed_call: none of them started a daemon, so
    there is no warmth contrast to draw.
    """
    if cold_disposition in ("not-run", "partial"):
        return []

    cold_config = next(config for config in ladder.configs if config.cold)
    warm_config = warm_counterpart_for(ladder, cold_config)
    cold_cell = cells[cold_config.id]
    warm_cell = cells[warm_config.id]
    if not cold_cell.complete or not warm_cell.complete:
        return []
    if cold_cell.runs and all(run.get("cold_no_daemon_backed_call") for run in cold_cell.runs):
        return []

    comparisons = []
    for metric in METRICS:
        comparison = _build_comparison("warm-vs-cold", cold_cell, warm_cell, metric)
        if comparison is not None:
            comparisons.append(comparison)
    return comparisons


""" THE TRACKED summary.json """


def _read_json_or_default(path, default):
    """Returns the parsed JSON at path, or default when path does not
    exist. Neither missing file this module reads (cold_cell.json,
    probe.json) raises -- summarising a partial matrix is a legitimate
    intermediate state."""
    path = Path(path)
    if not path.exists():
        return default
    with open(path) as f:
        return json.load(f)


def build_summary(ladder, results_root):
    """
    Builds the full mapping written to summary.json: _meta, cells,
    comparisons, and incomplete.

    _meta carries the pinned run model, the pinned scorer mapping, reps,
    the results-root date segment, the number of configs, the cold cell's
    disposition (from cold_cell.json, or "unknown" when absent), and a
    denied_tool_attempts_reported flag (from probe.json's
    denied_tools_advertised key, or None when probe.json is absent).

    cells maps every config id to its Cell stats plus complete,
    decoy_admitted_count, the verbatim summary_matches list,
    worktree_dirtied_count, target_origin_quarry_mention_count, and
    daemon_backed_runs, aggregated from load_runs's per-run observations.

    comparisons is every Comparison the three builders produce, as plain
    dicts. rung-vs-control comparisons are built for every config with a
    non-empty allowed set that is not the cold config; rung-vs-rung
    comparisons are built for every pair of such configs within the same
    ladder; warm-vs-cold is built once, for the ladder's single cold cell.

    incomplete lists the ids of every cell that is not complete, except a
    cold cell whose cold_cell.json records "not-run" or "partial" -- both
    are legitimate terminal states of the cold-cell driver, not interrupted
    runs.
    """
    results_root = Path(results_root)

    cells = {}
    for config in ladder.configs:
        runs = load_runs(results_root, config.id, ladder.reps)
        cells[config.id] = summarise_cell(config.id, runs, ladder.reps)

    cold_cell_record = _read_json_or_default(results_root / "cold_cell.json", {})
    cold_disposition = cold_cell_record.get("disposition", "unknown")
    cold_confirmed_cold_reps = cold_cell_record.get("confirmed_cold_reps")

    probe_record = _read_json_or_default(results_root / "probe.json", None)
    denied_tool_attempts_reported = None if probe_record is None else probe_record.get("denied_tools_advertised")

    meta = {
        "run_model": ladder.run_model,
        "scorer": {"model": ladder.scorer.model, "effort": ladder.scorer.effort},
        "reps": ladder.reps,
        "results_root_date": results_root.name,
        "num_configs": len(ladder.configs),
        "cold_disposition": cold_disposition,
        "cold_confirmed_cold_reps": cold_confirmed_cold_reps,
        "denied_tool_attempts_reported": denied_tool_attempts_reported,
    }

    cell_records = {}
    for config in ladder.configs:
        cell = cells[config.id]
        runs = cell.runs
        cell_records[config.id] = {
            "stats": cell.stats,
            "complete": cell.complete,
            "decoy_admitted_count": sum(1 for run in runs if run.get("decoy_admitted") is True),
            "summary_matches": [run["summary_matches"] for run in runs if "summary_matches" in run],
            "worktree_dirtied_count": sum(1 for run in runs if run.get("worktree_dirtied") is True),
            "target_origin_quarry_mention_count": sum(
                1 for run in runs if run.get("target_origin_quarry_mention") is True
            ),
            "daemon_backed_runs": sum(1 for run in runs if not run.get("cold_no_daemon_backed_call", False)),
        }

    non_control_configs = [config for config in ladder.configs if config.allowed and not config.cold]

    comparisons = []
    for config in non_control_configs:
        comparisons.extend(compare_rung_to_control(ladder, cells, config))
    for ladder_name in ("a", "b"):
        rung_ids = [config.id for config in non_control_configs if config.ladder == ladder_name]
        for i, left_id in enumerate(rung_ids):
            for right_id in rung_ids[i + 1 :]:
                comparisons.extend(compare_rungs(ladder, cells, left_id, right_id))
    comparisons.extend(compare_warm_cold(ladder, cells, cold_disposition))

    incomplete = []
    for config in ladder.configs:
        cell = cells[config.id]
        if cell.complete:
            continue
        if config.cold and cold_disposition in ("not-run", "partial"):
            continue
        incomplete.append(config.id)

    return {
        "_meta": meta,
        "cells": cell_records,
        "comparisons": [asdict(comparison) for comparison in comparisons],
        "incomplete": incomplete,
    }


def write_summary(ladder, results_root):
    """
    Serialises build_summary(ladder, results_root) as summary.json into
    results_root, with sorted keys and a trailing newline.

    Returns the summary mapping, so callers (including the CLI below) do
    not have to re-read the file they just wrote.
    """
    results_root = Path(results_root)
    summary = build_summary(ladder, results_root)
    with open(results_root / "summary.json", "w") as f:
        json.dump(summary, f, indent=2, sort_keys=True)
        f.write("\n")
    return summary


def _exit_code_for_summary(summary):
    """1 when summary's incomplete list is non-empty, 0 otherwise -- a
    summary of a partial matrix is written but must not be mistaken for a
    finished one."""
    return 1 if summary["incomplete"] else 0


def main(argv=None):
    """
    Loads the ladder at argv[0], writes summary.json into the results root
    at argv[1], and returns the process exit code: non-zero, after naming
    the incomplete cells on stderr, when any cell is incomplete; zero
    otherwise (including when the only short cell is a not-run cold cell).
    """
    parser = argparse.ArgumentParser(description="Summarise a quarry-mcp capability ladder results tree into summary.json.")
    parser.add_argument("ladder", help="path to ladder.yaml")
    parser.add_argument("results_root", help="path to the dated results directory, e.g. results/2026-08-29")
    args = parser.parse_args(argv)

    loaded_ladder = load_ladder(args.ladder)
    summary = write_summary(loaded_ladder, args.results_root)
    if summary["incomplete"]:
        print(f"incomplete cells: {', '.join(summary['incomplete'])}", file=sys.stderr)
    return _exit_code_for_summary(summary)


if __name__ == "__main__":
    sys.exit(main())
