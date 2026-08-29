"""
Tests for ladder_config.py: loading and validating ladder.yaml, control and
warm-counterpart resolution, per-config deny-list and settings-document
derivation, and per-rung preamble generation.

Usage:
    uv run --no-project --with pytest --with pyyaml python -m pytest \
        bench/loomyard-eval/ladder/tests/test_ladder_config.py -q
"""
import copy
import json
from dataclasses import replace
from pathlib import Path

import pytest
import yaml

import ladder_config as lc

LADDER_YAML = Path(__file__).resolve().parent.parent / "ladder.yaml"


def _raw_ladder_dict():
    """Loads ladder.yaml as a plain dict, for tests that mutate one field
    and re-dump it to a temp file to exercise a specific validation raise."""
    with open(LADDER_YAML) as f:
        return yaml.safe_load(f)


def _write_ladder(tmp_path, raw):
    path = tmp_path / "ladder.yaml"
    with open(path, "w") as f:
        yaml.safe_dump(raw, f)
    return path


""" LOADING """


def test_load_ladder_succeeds_with_15_configs():
    ladder = lc.load_ladder(LADDER_YAML)
    assert len(ladder.configs) == 15


def test_load_ladder_raises_on_duplicate_id(tmp_path):
    raw = _raw_ladder_dict()
    raw["configs"][1]["id"] = raw["configs"][0]["id"]
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_bad_ladder_value(tmp_path):
    raw = _raw_ladder_dict()
    raw["configs"][0]["ladder"] = "c"
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_unknown_task_key(tmp_path):
    raw = _raw_ladder_dict()
    raw["configs"][0]["task"] = "not-a-real-task"
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_allowed_entry_outside_quarry_tools(tmp_path):
    raw = _raw_ladder_dict()
    raw["configs"][0]["allowed"] = ["not_a_real_tool"]
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_non_canonical_quarry_tools(tmp_path):
    raw = _raw_ladder_dict()
    raw["quarry_tools"] = raw["quarry_tools"][:-1]
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_when_a_ladder_has_no_control(tmp_path):
    raw = _raw_ladder_dict()
    raw["configs"] = [c for c in raw["configs"] if c["id"] != "a0-none"]
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_when_a_ladder_has_two_controls(tmp_path):
    raw = _raw_ladder_dict()
    extra_control = copy.deepcopy(next(c for c in raw["configs"] if c["id"] == "a0-none"))
    extra_control["id"] = "a0-none-again"
    raw["configs"].append(extra_control)
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_second_cold_config(tmp_path):
    raw = _raw_ladder_dict()
    extra_cold = copy.deepcopy(next(c for c in raw["configs"] if c["id"] == "a5-bundle-cold"))
    extra_cold["id"] = "a5-bundle-cold-again"
    raw["configs"].append(extra_cold)
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_warm_counterpart_on_non_cold_config(tmp_path):
    raw = _raw_ladder_dict()
    non_cold = next(c for c in raw["configs"] if c["id"] == "a1-toc-file")
    non_cold["warm_counterpart"] = "a5-bundle"
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_cold_config_missing_warm_counterpart(tmp_path):
    raw = _raw_ladder_dict()
    cold = next(c for c in raw["configs"] if c["id"] == "a5-bundle-cold")
    del cold["warm_counterpart"]
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_warm_counterpart_naming_unknown_id(tmp_path):
    raw = _raw_ladder_dict()
    cold = next(c for c in raw["configs"] if c["id"] == "a5-bundle-cold")
    cold["warm_counterpart"] = "not-a-real-id"
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_warm_counterpart_naming_a_cold_config(tmp_path):
    raw = _raw_ladder_dict()
    other_cold = copy.deepcopy(next(c for c in raw["configs"] if c["id"] == "b0-none"))
    other_cold["id"] = "b0-none-cold"
    other_cold["cold"] = True
    other_cold["warm_counterpart"] = "b7-bundle"
    raw["configs"] = [c for c in raw["configs"] if c["id"] != "a5-bundle-cold"]
    raw["configs"].append(other_cold)
    cold = next(c for c in raw["configs"] if c["id"] == "b0-none-cold")
    cold["warm_counterpart"] = "b0-none-cold"
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


def test_load_ladder_raises_on_warm_counterpart_naming_itself(tmp_path):
    raw = _raw_ladder_dict()
    cold = next(c for c in raw["configs"] if c["id"] == "a5-bundle-cold")
    cold["warm_counterpart"] = "a5-bundle-cold"
    path = _write_ladder(tmp_path, raw)
    with pytest.raises(lc.LadderConfigError):
        lc.load_ladder(path)


""" RESOLUTION """


def test_control_for_resolves_within_each_ladder():
    ladder = lc.load_ladder(LADDER_YAML)
    a5 = lc.config_by_id(ladder, "a5-bundle")
    b5 = lc.config_by_id(ladder, "b5-impact")
    assert lc.control_for(ladder, a5).id == "a0-none"
    assert lc.control_for(ladder, b5).id == "b0-none"


def test_warm_counterpart_for_resolves_the_cold_configs_warm_cell():
    ladder = lc.load_ladder(LADDER_YAML)
    cold = lc.config_by_id(ladder, "a5-bundle-cold")
    assert lc.warm_counterpart_for(ladder, cold).id == "a5-bundle"


""" PINS """


def test_require_pins_raises_while_run_model_is_null():
    ladder = lc.load_ladder(LADDER_YAML)
    assert ladder.run_model is None
    with pytest.raises(lc.LadderConfigError):
        lc.require_pins(ladder)


def test_require_pins_raises_on_blanked_max_turns():
    ladder = lc.load_ladder(LADDER_YAML)
    ladder = replace(ladder, run_model="claude-opus-5", max_turns=None)
    with pytest.raises(lc.LadderConfigError):
        lc.require_pins(ladder)


def test_require_pins_raises_on_blanked_scorer_model():
    ladder = lc.load_ladder(LADDER_YAML)
    ladder = replace(ladder, run_model="claude-opus-5", scorer=replace(ladder.scorer, model=None))
    with pytest.raises(lc.LadderConfigError):
        lc.require_pins(ladder)


def test_require_pins_raises_on_blanked_scorer_effort():
    ladder = lc.load_ladder(LADDER_YAML)
    ladder = replace(ladder, run_model="claude-opus-5", scorer=replace(ladder.scorer, effort=None))
    with pytest.raises(lc.LadderConfigError):
        lc.require_pins(ladder)


def test_require_pins_passes_once_all_four_carry_values():
    ladder = lc.load_ladder(LADDER_YAML)
    ladder = replace(ladder, run_model="claude-opus-5")
    lc.require_pins(ladder)
