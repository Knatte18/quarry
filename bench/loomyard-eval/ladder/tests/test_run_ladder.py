"""
Tests for run_ladder.py: the pure planning and resume logic -- run
ordering, attempt accounting, the resume skip decision, argv assembly,
environment scrubbing, and task-text/schema extraction with their section
boundaries. Every test drives an injected `git=` or `executor=` seam, or
monkeypatches a module-level dispatch function; none launches a
subprocess, builds a real worktree, or makes a model call.
"""
import dataclasses
import json
from collections import Counter
from pathlib import Path
from types import SimpleNamespace

import pytest
import run_ladder
from gates import GateFinding
from ladder_config import (
    DAEMON_BACKED_TOOLS,
    Ladder,
    LadderConfig,
    ScorerConfig,
    TaskEntry,
    config_by_id,
    load_ladder,
)
from run_ladder import (
    MAX_ATTEMPTS,
    WARM_UP_TOOL,
    HarnessError,
    RunOutcome,
    _run_probe_if_needed,
    build_argv,
    cold_runs,
    ensure_task_worktrees,
    main_runs,
    mcp_config_document,
    pending_runs,
    plan_runs,
    run_env,
    run_matrix,
    run_stage,
    task_text_for,
)

LADDER_PATH = Path(__file__).resolve().parent.parent / "ladder.yaml"


@pytest.fixture
def real_ladder():
    return load_ladder(str(LADDER_PATH))


""" CARD 20: plan_runs / main_runs / cold_runs / pending_runs """


def test_plan_runs_yields_45_pairs_three_per_config_with_cold_strictly_last(real_ladder):
    pairs = plan_runs(real_ladder)
    assert len(pairs) == 45

    counts = Counter(config.id for config, _n in pairs)
    assert all(count == 3 for count in counts.values())

    cold_flags = [config.cold for config, _n in pairs]
    first_cold_index = cold_flags.index(True)
    assert all(not flag for flag in cold_flags[:first_cold_index])
    assert all(flag for flag in cold_flags[first_cold_index:])


def test_main_runs_and_cold_runs_partition_disjointly(real_ladder):
    mains = main_runs(real_ladder)
    colds = cold_runs(real_ladder)

    assert len(mains) == 42
    assert len(colds) == 3
    assert all(not config.cold for config, _n in mains)
    assert all(config.cold for config, _n in colds)

    main_pairs = {(config.id, n) for config, n in mains}
    cold_pairs = {(config.id, n) for config, n in colds}
    assert main_pairs.isdisjoint(cold_pairs)


def test_pending_runs_skips_complete_includes_absent_and_partial(tmp_path):
    complete_dir = tmp_path / "raw" / "a" / "1"
    complete_dir.mkdir(parents=True)
    (complete_dir / "run.json").write_text(json.dumps({"state": "complete"}))

    partial_dir = tmp_path / "raw" / "a" / "2"
    partial_dir.mkdir(parents=True)
    (partial_dir / "answer.json").write_text("{}")
    (partial_dir / "usage.json").write_text("{}")

    config = SimpleNamespace(id="a")
    pairs = [(config, 1), (config, 2), (config, 3)]

    pending = pending_runs(pairs, tmp_path)

    assert (config, 1) not in pending
    assert (config, 2) in pending
    assert (config, 3) in pending


""" CARD 19/22: build_argv, run_env """


def test_build_argv_includes_mcp_config_for_nonempty_allowed(tmp_path, real_ladder):
    ladder = dataclasses.replace(real_ladder, run_model="claude-opus-5-test")
    config = config_by_id(ladder, "a5-bundle")

    argv = build_argv(ladder, config, tmp_path, "/tmp/target")

    assert "--mcp-config" in argv
    assert str(tmp_path / "mcp.json") in argv


def test_build_argv_omits_mcp_config_for_empty_allowed(tmp_path, real_ladder):
    ladder = dataclasses.replace(real_ladder, run_model="claude-opus-5-test")
    config = config_by_id(ladder, "a0-none")

    argv = build_argv(ladder, config, tmp_path, "/tmp/target")

    assert "--mcp-config" not in argv


def test_build_argv_includes_expected_flags_and_never_state_dir(tmp_path, real_ladder):
    ladder = dataclasses.replace(real_ladder, run_model="claude-opus-5-test")
    config = config_by_id(ladder, "a0-none")

    argv = build_argv(ladder, config, tmp_path, "/tmp/target")

    assert "--setting-sources" in argv
    assert "--strict-mcp-config" in argv
    assert "--output-format" in argv
    assert "stream-json" in argv
    assert "--model" in argv
    assert "claude-opus-5-test" in argv
    assert "--max-turns" in argv
    assert str(ladder.max_turns) in argv
    assert "--state-dir" not in argv


def test_run_env_scrubs_state_dir_and_build_tags_leaves_config(monkeypatch):
    monkeypatch.setenv("QUARRY_STATE_DIR", "/should-be-removed")
    monkeypatch.setenv("QUARRY_BUILD_TAGS", "integration")
    monkeypatch.setenv("QUARRY_CONFIG", "/keep/this")

    env = run_env()

    assert "QUARRY_STATE_DIR" not in env
    assert "QUARRY_BUILD_TAGS" not in env
    assert env["QUARRY_CONFIG"] == "/keep/this"


""" CARD 19: task_text_for """


def test_task_text_for_extracts_task01_bounded_text_without_the_fasit_leads(real_ladder):
    text = task_text_for(real_ladder, "01-reed-geometry-exploration")
    assert text.startswith("Explain how a reed session's terminal geometry")
    assert "burler.go:373" not in text
    assert "fasit" not in text.lower()


def test_task_text_for_extracts_task04_bounded_text_without_the_scoring_notes(real_ladder):
    text = task_text_for(real_ladder, "04-shedadapters-shuttle-impact")
    assert text.startswith("You are about to change the `Shuttle` interface")
    assert "burler.go:373" not in text
    assert "fasit" not in text.lower()


def test_task_text_for_raises_when_heading_absent(tmp_path, real_ladder):
    bogus_task_file = tmp_path / "bogus.md"
    bogus_task_file.write_text("# No task text heading here\n")
    bogus_task = TaskEntry(
        task_file=str(bogus_task_file), pinned_sha="deadbeef", worktree=str(tmp_path / "wt"),
        schema="exploration", fasit="unused",
    )
    ladder = dataclasses.replace(real_ladder, tasks={**real_ladder.tasks, "bogus": bogus_task})

    with pytest.raises(HarnessError):
        task_text_for(ladder, "bogus")


""" CARD 16/22: ensure_task_worktrees """


def _ladder_with_single_task_worktree(real_ladder, task_key, worktree_path):
    task = dataclasses.replace(real_ladder.tasks[task_key], worktree=str(worktree_path))
    return dataclasses.replace(real_ladder, tasks={task_key: task})


def test_ensure_task_worktrees_builds_a_missing_worktree_without_creating_one(tmp_path, real_ladder):
    task_key = next(iter(real_ladder.tasks))
    missing_path = tmp_path / "missing-worktree"
    ladder = _ladder_with_single_task_worktree(real_ladder, task_key, missing_path)

    calls = []

    def fake_git(args):
        calls.append(list(args))
        return ""

    worktrees = ensure_task_worktrees(ladder, git=fake_git)

    assert worktrees[task_key] == missing_path
    assert not missing_path.exists()
    assert any("add" in call for call in calls)


def test_ensure_task_worktrees_adopts_existing_worktree_at_matching_pin(tmp_path, real_ladder):
    task_key = next(iter(real_ladder.tasks))
    existing_path = tmp_path / "existing-worktree"
    existing_path.mkdir()
    ladder = _ladder_with_single_task_worktree(real_ladder, task_key, existing_path)
    pin = ladder.tasks[task_key].pinned_sha

    def fake_git(args):
        if "rev-parse" in args:
            return pin + "\n"
        return ""

    worktrees = ensure_task_worktrees(ladder, git=fake_git)

    assert worktrees[task_key] == existing_path


def test_ensure_task_worktrees_raises_when_existing_worktree_is_at_the_wrong_pin(tmp_path, real_ladder):
    task_key = next(iter(real_ladder.tasks))
    existing_path = tmp_path / "existing-worktree"
    existing_path.mkdir()
    ladder = _ladder_with_single_task_worktree(real_ladder, task_key, existing_path)

    def fake_git(args):
        if "rev-parse" in args:
            return "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n"
        return ""

    with pytest.raises(HarnessError):
        ensure_task_worktrees(ladder, git=fake_git)


""" CARD 22: the probe skip decision and the --stage selector """


def test_run_probe_if_needed_skips_when_results_root_already_holds_a_passing_probe(tmp_path):
    (tmp_path / "probe.json").write_text(json.dumps({"denial_blocks": True}))
    calls = []

    def fake_prober(*args, **kwargs):
        calls.append((args, kwargs))
        return {}

    ladder = SimpleNamespace(tasks={"t": None})
    result = _run_probe_if_needed(ladder, "/repo", tmp_path, {"t": "/wt"}, "/bin/quarry-mcp", prober=fake_prober)

    assert calls == []
    assert result == {"denial_blocks": True}


def test_run_probe_if_needed_runs_when_no_probe_json_present(tmp_path):
    calls = []

    def fake_prober(ladder, repo_root, results_root, target_dir, server_path):
        calls.append((ladder, repo_root, results_root, target_dir, server_path))
        return {"denial_blocks": True}

    ladder = SimpleNamespace(tasks={"t": None})
    result = _run_probe_if_needed(ladder, "/repo", tmp_path, {"t": "/wt"}, "/bin/quarry-mcp", prober=fake_prober)

    assert len(calls) == 1
    assert result == {"denial_blocks": True}


def _run_stage_with_recording_drivers(tmp_path, stage):
    calls = []

    def fake_prober(*args, **kwargs):
        calls.append("probe")
        return {"denial_blocks": True}

    def fake_matrix_runner(*args, **kwargs):
        calls.append("main")

    def fake_cold_runner(*args, **kwargs):
        calls.append("cold")

    ladder = SimpleNamespace(tasks={"t": None})
    run_stage(
        ladder, tmp_path, {"t": "/wt"}, "/bin/server", "/repo", "/cache", stage,
        prober=fake_prober, matrix_runner=fake_matrix_runner, cold_runner=fake_cold_runner,
    )
    return calls


def test_run_stage_probe_runs_only_the_probe(tmp_path):
    assert _run_stage_with_recording_drivers(tmp_path, "probe") == ["probe"]


def test_run_stage_main_runs_only_the_matrix(tmp_path):
    assert _run_stage_with_recording_drivers(tmp_path, "main") == ["main"]


def test_run_stage_cold_runs_only_the_cold_cell(tmp_path):
    assert _run_stage_with_recording_drivers(tmp_path, "cold") == ["cold"]


def test_run_stage_all_runs_probe_then_main_then_cold_in_order(tmp_path):
    assert _run_stage_with_recording_drivers(tmp_path, "all") == ["probe", "main", "cold"]


def test_warm_up_tool_is_daemon_backed():
    assert WARM_UP_TOOL in DAEMON_BACKED_TOOLS


def test_mcp_config_document_declares_a_single_quarry_server_with_the_runs_target_dir():
    document = mcp_config_document("/abs/quarry-mcp", "/tmp/target")

    assert list(document["mcpServers"].keys()) == ["quarry"]
    server = document["mcpServers"]["quarry"]
    assert server["command"] == "/abs/quarry-mcp"
    assert "--target-dir" in server["args"]
    assert "/tmp/target" in server["args"]


""" CARD 20: attempt accounting """


def _minimal_ladder_with_one_main_pair():
    config = LadderConfig(id="x0-only", ladder="a", task="t", allowed=(), cold=False)
    task = TaskEntry(task_file="unused", pinned_sha="deadbeef", worktree="/tmp/unused-worktree", schema="exploration", fasit="unused")
    ladder = Ladder(
        run_model="claude-opus-5-test",
        reps=1,
        max_turns=10,
        scorer=ScorerConfig(model="claude-opus-5-test", effort="high"),
        quarry_tools=DAEMON_BACKED_TOOLS + ("toc_dir", "toc_file"),
        tasks={"t": task},
        source_repo="/unused",
        cold_worktree_template="/tmp/x-{n}",
        configs=(config,),
    )
    return ladder, config


def test_run_matrix_calls_warm_daemon_once_per_dispatch_and_never_for_a_cold_pair(tmp_path, monkeypatch):
    ladder, config = _minimal_ladder_with_one_main_pair()
    worktrees = {"t": "/tmp/unused-worktree"}

    warm_calls = []
    monkeypatch.setattr(run_ladder, "warm_daemon", lambda *args, **kwargs: warm_calls.append(args))
    monkeypatch.setattr(run_ladder, "restore_worktree", lambda *args, **kwargs: None)

    def fake_execute_run(ladder_arg, config_arg, n, a_run_dir, target_dir, server_path, repo_root, cache_dir, executor=None):
        return RunOutcome(status="complete", config_id=config_arg.id, n=n, run_json={"state": "complete"})

    monkeypatch.setattr(run_ladder, "execute_run", fake_execute_run)

    run_matrix(ladder, tmp_path / "results", worktrees, "/bin/server", "/repo", "/cache")

    assert len(warm_calls) == 1
    assert config.cold is False


def test_run_matrix_halts_after_three_consecutive_failures(tmp_path, monkeypatch):
    ladder, _config = _minimal_ladder_with_one_main_pair()
    worktrees = {"t": "/tmp/unused-worktree"}

    monkeypatch.setattr(run_ladder, "warm_daemon", lambda *args, **kwargs: None)
    monkeypatch.setattr(run_ladder, "restore_worktree", lambda *args, **kwargs: None)
    monkeypatch.setattr(run_ladder, "invalidate", lambda a_run_dir: a_run_dir)

    attempts = []

    def always_failing_execute_run(ladder_arg, config_arg, n, a_run_dir, target_dir, server_path, repo_root, cache_dir, executor=None):
        attempts.append(1)
        return RunOutcome(
            status="failed", config_id=config_arg.id, n=n, findings=(GateFinding("x", "boom", fatal=True),)
        )

    monkeypatch.setattr(run_ladder, "execute_run", always_failing_execute_run)

    with pytest.raises(HarnessError):
        run_matrix(ladder, tmp_path / "results", worktrees, "/bin/server", "/repo", "/cache")

    assert len(attempts) == MAX_ATTEMPTS


def test_run_matrix_recovers_after_two_failures_and_keeps_running(tmp_path, monkeypatch):
    ladder, _config = _minimal_ladder_with_one_main_pair()
    worktrees = {"t": "/tmp/unused-worktree"}

    monkeypatch.setattr(run_ladder, "warm_daemon", lambda *args, **kwargs: None)
    monkeypatch.setattr(run_ladder, "restore_worktree", lambda *args, **kwargs: None)
    monkeypatch.setattr(run_ladder, "invalidate", lambda a_run_dir: a_run_dir)

    attempts = []

    def flaky_execute_run(ladder_arg, config_arg, n, a_run_dir, target_dir, server_path, repo_root, cache_dir, executor=None):
        attempts.append(1)
        if len(attempts) < MAX_ATTEMPTS:
            return RunOutcome(
                status="failed", config_id=config_arg.id, n=n, findings=(GateFinding("x", "boom", fatal=True),)
            )
        return RunOutcome(status="complete", config_id=config_arg.id, n=n, run_json={"state": "complete"})

    monkeypatch.setattr(run_ladder, "execute_run", flaky_execute_run)

    run_matrix(ladder, tmp_path / "results", worktrees, "/bin/server", "/repo", "/cache")

    assert len(attempts) == MAX_ATTEMPTS
