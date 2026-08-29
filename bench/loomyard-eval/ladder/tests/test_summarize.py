"""
Tests for summarize.py: per-config medians and ranges, the three
disjoint-range comparison types, and the tracked summary.json this batch
emits.

Every unit is pure over a synthetic results tree built in tmp_path, so no
real run artifacts and no live model call are needed.
"""
import json
import shutil
from pathlib import Path

import pytest
import summarize
from gates import run_dir
from ladder_config import config_by_id, load_ladder

LADDER_PATH = Path(__file__).resolve().parent.parent / "ladder.yaml"


@pytest.fixture
def ladder():
    return load_ladder(LADDER_PATH)


def _write_run(results_root, config_id, n, *, usage=None, score=None, run_extra=None, state="complete"):
    """
    Writes a synthetic run directory at <results_root>/raw/<config_id>/<n>/
    carrying usage.json, score.json, and run.json, with reasonable defaults
    for every field summarize.py reads.
    """
    directory = run_dir(results_root, config_id, n)
    directory.mkdir(parents=True, exist_ok=True)

    full_usage = {
        "duration_ms": 1000,
        "wall_clock_ms": 1200,
        "num_turns": 5,
        "tool_uses": 3,
        "quarry_tool_uses": 2,
        "tokens": {
            "input_tokens": 100,
            "output_tokens": 50,
            "cache_read_input_tokens": 10,
            "cache_creation_input_tokens": 5,
        },
        "cost_usd": 0.05,
        "bash_grep_count": 0,
        "grep_tool_count": 0,
        "grep_fallback_total": 0,
        "denied_tool_attempts": 0,
    }
    if usage:
        full_usage.update(usage)
    with open(directory / "usage.json", "w") as f:
        json.dump(full_usage, f)

    full_score = {"recall": 0.5, "precision": 0.5}
    if score:
        full_score.update(score)
    with open(directory / "score.json", "w") as f:
        json.dump(full_score, f)

    full_run = {"config_id": config_id, "n": n, "state": state}
    if run_extra:
        full_run.update(run_extra)
    with open(directory / "run.json", "w") as f:
        json.dump(full_run, f)

    return directory


def _cell_from_metric(config_id, metric, values):
    """Builds a Cell via summarise_cell over a single metric's values --
    keeps comparison-builder tests focused on one metric at a time."""
    runs = [{metric: value} for value in values]
    return summarize.summarise_cell(config_id, runs, len(values))


""" CARD 13: per-config medians, ranges, and completeness """


def test_summarise_cell_odd_run_count_takes_middle_value():
    cell = summarize.summarise_cell("a1-toc-file", [{"num_turns": 9}, {"num_turns": 1}, {"num_turns": 5}], 3)
    assert cell.stats["num_turns"]["median"] == 5
    assert cell.complete is True


def test_summarise_cell_even_run_count_takes_mean_of_middle_two():
    cell = summarize.summarise_cell("a1-toc-file", [{"num_turns": 1}, {"num_turns": 3}], 2)
    assert cell.stats["num_turns"]["median"] == 2.0


def test_summarise_cell_with_two_of_three_runs_is_incomplete():
    cell = summarize.summarise_cell("a1-toc-file", [{"num_turns": 1}, {"num_turns": 3}], 3)
    assert cell.complete is False
    assert cell.stats["num_turns"]["n"] == 2


def test_summarise_cell_metric_missing_from_one_run_reduces_only_that_metrics_n():
    runs = [{"num_turns": 5, "cost_usd": 1.0}, {"num_turns": 7}]
    cell = summarize.summarise_cell("a1-toc-file", runs, 2)
    assert cell.stats["num_turns"]["n"] == 2
    assert cell.stats["cost_usd"]["n"] == 1


def test_load_runs_ignores_a_run_whose_run_json_is_not_complete(tmp_path):
    _write_run(tmp_path, "a1-toc-file", 1)
    _write_run(tmp_path, "a1-toc-file", 2, state="running")
    _write_run(tmp_path, "a1-toc-file", 3)

    runs = summarize.load_runs(tmp_path, "a1-toc-file", 3)

    assert len(runs) == 2


def test_load_runs_flattens_tokens_to_top_level(tmp_path):
    _write_run(tmp_path, "a1-toc-file", 1)

    runs = summarize.load_runs(tmp_path, "a1-toc-file", 1)

    assert runs[0]["input_tokens"] == 100
    assert runs[0]["output_tokens"] == 50
    assert runs[0]["cache_read_input_tokens"] == 10
    assert runs[0]["cache_creation_input_tokens"] == 5
    assert "tokens" not in runs[0]


def test_load_runs_carries_run_json_observations(tmp_path):
    _write_run(tmp_path, "a1-toc-file", 1, run_extra={"worktree_dirtied": True, "target_origin_quarry_mention": False})

    runs = summarize.load_runs(tmp_path, "a1-toc-file", 1)

    assert runs[0]["worktree_dirtied"] is True
    assert runs[0]["target_origin_quarry_mention"] is False


""" CARD 14: the three disjoint-range comparison types """


def test_ranges_disjoint_true_for_non_overlapping():
    assert summarize.ranges_disjoint((1, 5), (6, 9)) is True


def test_ranges_disjoint_false_for_overlapping():
    assert summarize.ranges_disjoint((1, 5), (3, 9)) is False


def test_ranges_disjoint_touching_at_one_endpoint_is_not_disjoint():
    assert summarize.ranges_disjoint((1, 5), (5, 9)) is False


def test_compare_rung_to_control_overlapping_ranges_are_not_separated(ladder):
    a1 = config_by_id(ladder, "a1-toc-file")
    cells = {
        "a1-toc-file": _cell_from_metric("a1-toc-file", "duration_ms", [10, 12, 14]),
        "a0-none": _cell_from_metric("a0-none", "duration_ms", [12, 13, 15]),
    }
    comparisons = summarize.compare_rung_to_control(ladder, cells, a1)
    assert len(comparisons) == 1
    assert comparisons[0].kind == "rung-vs-control"
    assert comparisons[0].separated is False


def test_compare_rung_to_control_disjoint_ranges_are_separated(ladder):
    a1 = config_by_id(ladder, "a1-toc-file")
    cells = {
        "a1-toc-file": _cell_from_metric("a1-toc-file", "duration_ms", [10, 12, 14]),
        "a0-none": _cell_from_metric("a0-none", "duration_ms", [20, 22, 24]),
    }
    comparisons = summarize.compare_rung_to_control(ladder, cells, a1)
    assert comparisons[0].separated is True


def test_compare_rungs_overlapping_and_disjoint_are_asserted_independently(ladder):
    overlapping_cells = {
        "a1-toc-file": _cell_from_metric("a1-toc-file", "duration_ms", [10, 12, 14]),
        "a2-toc-dir": _cell_from_metric("a2-toc-dir", "duration_ms", [11, 13, 15]),
    }
    overlapping = summarize.compare_rungs(ladder, overlapping_cells, "a1-toc-file", "a2-toc-dir")
    assert overlapping[0].kind == "rung-vs-rung"
    assert overlapping[0].separated is False

    disjoint_cells = {
        "a1-toc-file": _cell_from_metric("a1-toc-file", "duration_ms", [10, 12, 14]),
        "a2-toc-dir": _cell_from_metric("a2-toc-dir", "duration_ms", [30, 32, 34]),
    }
    disjoint = summarize.compare_rungs(ladder, disjoint_cells, "a1-toc-file", "a2-toc-dir")
    assert disjoint[0].separated is True


def test_rung_vs_control_excludes_grep_metrics_but_rung_vs_rung_does_not(ladder):
    def cell(config_id):
        runs = [{"duration_ms": v, "bash_grep_count": g} for v, g in zip([10, 12, 14], [1, 2, 3])]
        return summarize.summarise_cell(config_id, runs, 3)

    a1 = config_by_id(ladder, "a1-toc-file")
    cells = {
        "a1-toc-file": cell("a1-toc-file"),
        "a2-toc-dir": cell("a2-toc-dir"),
        "a0-none": cell("a0-none"),
    }

    control_comparisons = summarize.compare_rung_to_control(ladder, cells, a1)
    assert all(comparison.metric not in summarize.GREP_METRICS for comparison in control_comparisons)

    rung_comparisons = summarize.compare_rungs(ladder, cells, "a1-toc-file", "a2-toc-dir")
    assert any(comparison.metric == "bash_grep_count" for comparison in rung_comparisons)


def test_compare_rungs_raises_across_ladders(ladder):
    cells = {
        "a1-toc-file": _cell_from_metric("a1-toc-file", "duration_ms", [1, 2, 3]),
        "b1-symbol": _cell_from_metric("b1-symbol", "duration_ms", [1, 2, 3]),
    }
    with pytest.raises(summarize.SummarizeError):
        summarize.compare_rungs(ladder, cells, "a1-toc-file", "b1-symbol")


def test_compare_rung_to_control_resolves_ladder_b_control_never_a(ladder):
    b1 = config_by_id(ladder, "b1-symbol")
    cells = {
        "b1-symbol": _cell_from_metric("b1-symbol", "duration_ms", [1, 2, 3]),
        "b0-none": _cell_from_metric("b0-none", "duration_ms", [4, 5, 6]),
        "a0-none": _cell_from_metric("a0-none", "duration_ms", [100, 200, 300]),
    }
    comparisons = summarize.compare_rung_to_control(ladder, cells, b1)
    assert comparisons[0].right == "b0-none"
    assert all(comparison.right != "a0-none" for comparison in comparisons)


def test_compare_warm_cold_resolves_warm_side_through_warm_counterpart_field(ladder):
    cells = {
        "a5-bundle-cold": _cell_from_metric("a5-bundle-cold", "duration_ms", [1, 2, 3]),
        "a5-bundle": _cell_from_metric("a5-bundle", "duration_ms", [100, 110, 120]),
    }
    comparisons = summarize.compare_warm_cold(ladder, cells, "confirmed-cold")
    assert len(comparisons) == 1
    assert comparisons[0].kind == "warm-vs-cold"
    assert {comparisons[0].left, comparisons[0].right} == {"a5-bundle-cold", "a5-bundle"}


def test_compare_warm_cold_emits_nothing_when_every_cold_run_lacks_daemon_signal(ladder):
    runs = [{"duration_ms": v, "cold_no_daemon_backed_call": True} for v in [1, 2, 3]]
    cells = {
        "a5-bundle-cold": summarize.summarise_cell("a5-bundle-cold", runs, 3),
        "a5-bundle": _cell_from_metric("a5-bundle", "duration_ms", [10, 11, 12]),
    }
    assert summarize.compare_warm_cold(ladder, cells, "confirmed-cold") == []


@pytest.mark.parametrize("disposition", ["not-run", "partial"])
def test_compare_warm_cold_emits_nothing_for_not_run_or_partial_disposition(ladder, disposition):
    cells = {
        "a5-bundle-cold": _cell_from_metric("a5-bundle-cold", "duration_ms", [1, 2, 3]),
        "a5-bundle": _cell_from_metric("a5-bundle", "duration_ms", [10, 11, 12]),
    }
    assert summarize.compare_warm_cold(ladder, cells, disposition) == []


def test_incomplete_cell_yields_no_comparison_at_all(ladder):
    a1 = config_by_id(ladder, "a1-toc-file")
    cells = {
        "a1-toc-file": summarize.summarise_cell("a1-toc-file", [{"duration_ms": 1}], 3),
        "a0-none": _cell_from_metric("a0-none", "duration_ms", [1, 2, 3]),
    }
    assert summarize.compare_rung_to_control(ladder, cells, a1) == []


""" CARD 15: emit the tracked summary.json """


_DEFAULT_COLD_CELL_PAYLOAD = {"disposition": "confirmed-cold", "confirmed_cold_reps": 3}
_DEFAULT_PROBE_PAYLOAD = {"denied_tools_advertised": True}

# Sentinel distinguishing "caller did not pass this argument" from an
# explicit None -- callers pass cold_cell_payload=None / probe_payload=None
# to simulate the file genuinely being absent.
_UNSET = object()


def _write_full_matrix(
    results_root,
    ladder,
    *,
    decoy_admitted_config=None,
    worktree_dirtied_config=None,
    target_origin_config=None,
    cold_cell_payload=_UNSET,
    probe_payload=_UNSET,
):
    """
    Writes ladder.reps complete runs for every config in ladder, plus
    cold_cell.json and probe.json, so build_summary sees a fully complete
    matrix. Individual tests perturb specific runs afterwards (removing a
    run.json to make a cell short, deleting the cold cell's directory to
    simulate a driver that never ran it).
    """
    if cold_cell_payload is _UNSET:
        cold_cell_payload = _DEFAULT_COLD_CELL_PAYLOAD
    if probe_payload is _UNSET:
        probe_payload = _DEFAULT_PROBE_PAYLOAD
    for config in ladder.configs:
        task = ladder.tasks[config.task]
        score_extra = {"decoy_admitted": False, "lookalikes_matched": 0} if task.schema == "impact" else {"summary_matches": True}
        for n in range(1, ladder.reps + 1):
            score = dict(score_extra)
            if config.id == decoy_admitted_config and n == 1:
                score["decoy_admitted"] = True

            run_extra = {}
            if config.id == worktree_dirtied_config and n == 1:
                run_extra["worktree_dirtied"] = True
            if config.id == target_origin_config and n == 1:
                run_extra["target_origin_quarry_mention"] = True
            if config.cold:
                run_extra.setdefault("cold_no_daemon_backed_call", False)

            _write_run(results_root, config.id, n, score=score, run_extra=run_extra)

    if cold_cell_payload is not None:
        with open(Path(results_root) / "cold_cell.json", "w") as f:
            json.dump(cold_cell_payload, f)
    if probe_payload is not None:
        with open(Path(results_root) / "probe.json", "w") as f:
            json.dump(probe_payload, f)


def test_build_summary_meta_records_pinned_model_and_scorer(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    summary = summarize.build_summary(ladder, tmp_path)
    assert summary["_meta"]["scorer"] == {"model": ladder.scorer.model, "effort": ladder.scorer.effort}
    assert summary["_meta"]["reps"] == ladder.reps


def test_build_summary_every_config_id_appears_in_cells(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    summary = summarize.build_summary(ladder, tmp_path)
    assert set(summary["cells"].keys()) == {config.id for config in ladder.configs}


def test_build_summary_incomplete_lists_exactly_the_short_cells(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    (run_dir(tmp_path, "a3-toc-pair", 2) / "run.json").unlink()

    summary = summarize.build_summary(ladder, tmp_path)

    assert summary["incomplete"] == ["a3-toc-pair"]


def test_build_summary_comparisons_contains_all_three_kinds(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    summary = summarize.build_summary(ladder, tmp_path)
    kinds = {comparison["kind"] for comparison in summary["comparisons"]}
    assert kinds == {"rung-vs-control", "rung-vs-rung", "warm-vs-cold"}


def test_build_summary_aggregates_worktree_dirtied_decoy_admitted_and_daemon_backed(tmp_path, ladder):
    _write_full_matrix(
        tmp_path,
        ladder,
        decoy_admitted_config="b6-assert-no-callers",
        worktree_dirtied_config="a1-toc-file",
        target_origin_config="a0-none",
    )
    summary = summarize.build_summary(ladder, tmp_path)

    assert summary["cells"]["b6-assert-no-callers"]["decoy_admitted_count"] == 1
    assert summary["cells"]["a1-toc-file"]["worktree_dirtied_count"] == 1
    assert summary["cells"]["a0-none"]["target_origin_quarry_mention_count"] == 1
    assert summary["cells"]["a5-bundle-cold"]["daemon_backed_runs"] == ladder.reps


def test_build_summary_summary_matches_carried_through_verbatim(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    summary = summarize.build_summary(ladder, tmp_path)
    assert summary["cells"]["a1-toc-file"]["summary_matches"] == [True, True, True]


def test_build_summary_denied_tool_attempts_reported_follows_probe_record(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder, probe_payload={"denied_tools_advertised": False})
    summary = summarize.build_summary(ladder, tmp_path)
    assert summary["_meta"]["denied_tool_attempts_reported"] is False


def test_build_summary_not_run_cold_cell_absent_from_incomplete_and_cli_exits_zero(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder, cold_cell_payload={"disposition": "not-run", "confirmed_cold_reps": 0})
    shutil.rmtree(run_dir(tmp_path, "a5-bundle-cold", 1).parent)

    summary = summarize.build_summary(ladder, tmp_path)

    assert "a5-bundle-cold" not in summary["incomplete"]
    assert summarize._exit_code_for_summary(summary) == 0


def test_build_summary_partial_cold_cell_absent_from_incomplete_and_no_warm_cold_comparison(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder, cold_cell_payload={"disposition": "partial", "confirmed_cold_reps": 1})
    shutil.rmtree(run_dir(tmp_path, "a5-bundle-cold", 1).parent)

    summary = summarize.build_summary(ladder, tmp_path)

    assert "a5-bundle-cold" not in summary["incomplete"]
    assert not any(comparison["kind"] == "warm-vs-cold" for comparison in summary["comparisons"])
    assert summarize._exit_code_for_summary(summary) == 0


def test_build_summary_short_cold_cell_for_other_reason_is_incomplete(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    (run_dir(tmp_path, "a5-bundle-cold", 2) / "run.json").unlink()

    summary = summarize.build_summary(ladder, tmp_path)

    assert "a5-bundle-cold" in summary["incomplete"]
    assert summarize._exit_code_for_summary(summary) == 1


def test_build_summary_absent_cold_cell_json_and_probe_json_do_not_raise(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder, cold_cell_payload=None, probe_payload=None)
    shutil.rmtree(run_dir(tmp_path, "a5-bundle-cold", 1).parent)

    summary = summarize.build_summary(ladder, tmp_path)

    assert summary["_meta"]["cold_disposition"] == "unknown"
    assert summary["_meta"]["denied_tool_attempts_reported"] is None
    assert "a5-bundle-cold" in summary["incomplete"]


def test_write_summary_writes_sorted_keys_json_with_trailing_newline(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    summarize.write_summary(ladder, tmp_path)

    content = (tmp_path / "summary.json").read_text()

    assert content.endswith("\n")
    parsed = json.loads(content)
    assert set(parsed.keys()) == {"_meta", "cells", "comparisons", "incomplete"}


def test_main_exit_code_is_nonzero_when_a_cell_is_incomplete(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)
    (run_dir(tmp_path, "a3-toc-pair", 2) / "run.json").unlink()

    assert summarize.main([str(LADDER_PATH), str(tmp_path)]) == 1


def test_main_exit_code_is_zero_when_no_cell_is_incomplete(tmp_path, ladder):
    _write_full_matrix(tmp_path, ladder)

    assert summarize.main([str(LADDER_PATH), str(tmp_path)]) == 0
