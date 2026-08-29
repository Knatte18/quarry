"""
Tests for gates.py: the transcript gates against the committed fixtures,
the filesystem and daemon-state gates against tmp_path fixtures, and the
run-state helpers against synthesised run directories. No live model call,
no network, and no real daemon is involved -- the daemon gates are asserted
against the state artifact's presence, which is the only source-grounded
signal available without modifying quarry.
"""
import json
import os
import subprocess
import sys
import threading
import time
from pathlib import Path
from types import SimpleNamespace

import pytest
from extract_usage import read_transcript
from gates import (
    GateError,
    GateFinding,
    GateReport,
    clear_state_dir,
    daemon_state_file,
    gate_blinding,
    gate_cold_after,
    gate_cold_before,
    gate_denied_tools_not_used,
    gate_model_pinned,
    gate_no_target_override,
    gate_run_complete_artifacts,
    gate_worktree_neutralised,
    invalidate,
    is_complete,
    observe_worktree_dirtied,
    resolve_state_dir,
    run_dir,
    run_gates,
    used_daemon_backed_tool,
    user_cache_dir,
    wait_for_daemon_exit,
    workspace_key,
    write_run_json,
)

FIXTURES_DIR = Path(__file__).parent / "fixtures"


def load_fixture(name):
    return read_transcript(FIXTURES_DIR / name)


""" CARD 8: TRANSCRIPT-DERIVED GATES """


def test_gate_denied_tools_not_used_passes_bundle_fixture():
    events = load_fixture("bundle-mixed-tools.jsonl")
    assert gate_denied_tools_not_used(events, denied_names=["mcp__quarry__impact"]) == []


def test_gate_denied_tools_not_used_passes_denied_attempt_fixture():
    # impact was denied and never dispatched as a tool_use block at all --
    # it surfaces only in the result event's permission_denials, which is
    # extract_usage.py's denied_tool_attempts metric, not this gate's job.
    events = load_fixture("denied-attempt.jsonl")
    assert gate_denied_tools_not_used(events, denied_names=["mcp__quarry__impact"]) == []


def test_gate_denied_tools_not_used_fails_when_a_denied_tool_succeeds():
    events = load_fixture("bundle-mixed-tools.jsonl")
    findings = gate_denied_tools_not_used(events, denied_names=["mcp__quarry__toc_file"])
    assert len(findings) == 1
    assert findings[0].fatal is True
    assert findings[0].gate == "denied_tools_not_used"


def test_gate_no_target_override_passes_bundle_fixture():
    events = load_fixture("bundle-mixed-tools.jsonl")
    assert gate_no_target_override(events) == []


def test_gate_no_target_override_fails_targetdir_and_buildtags():
    events = load_fixture("targetdir-override.jsonl")
    findings = gate_no_target_override(events)
    assert len(findings) == 2
    assert all(finding.fatal for finding in findings)
    messages = " ".join(finding.message for finding in findings)
    assert "targetDir" in messages
    assert "buildTags" in messages


def _events_with_model(model_id):
    return [
        {"type": "system", "subtype": "init", "model": model_id, "session_id": "s"},
        {"type": "result", "subtype": "success", "is_error": False, "duration_ms": 1, "num_turns": 1, "session_id": "s"},
    ]


def test_gate_model_pinned_passes_on_exact_match():
    events = _events_with_model("claude-opus-5")
    assert gate_model_pinned(events, run_model="claude-opus-5") == []


def test_gate_model_pinned_passes_on_bracketed_context_window_suffix():
    events = _events_with_model("claude-opus-5[1m]")
    assert gate_model_pinned(events, run_model="claude-opus-5") == []


def test_gate_model_pinned_fails_on_mismatch():
    events = _events_with_model("claude-sonnet-5")
    findings = gate_model_pinned(events, run_model="claude-opus-5")
    assert len(findings) == 1
    assert findings[0].fatal is True


def test_gate_blinding_passes_with_non_fatal_observation_on_bare_mention():
    events = load_fixture("none-target-origin-mention.jsonl")
    findings = gate_blinding(events, repo_root="/home/user/quarry")
    assert all(not finding.fatal for finding in findings)
    assert any(finding.gate == "target_origin_quarry_mention" for finding in findings)


def test_gate_blinding_fails_on_mcp_quarry_tool_name():
    events = [
        {"type": "system", "subtype": "init", "model": "claude-opus-5", "session_id": "s"},
        {
            "type": "assistant",
            "message": {
                "content": [{"type": "tool_use", "id": "tu_1", "name": "mcp__quarry__toc_file", "input": {}}]
            },
        },
    ]
    findings = gate_blinding(events, repo_root="/home/user/quarry")
    assert any(finding.fatal for finding in findings)


def test_gate_blinding_fails_on_quarry_bench_literal():
    events = [
        {"type": "system", "subtype": "init", "model": "claude-opus-5", "session_id": "s"},
        {
            "type": "assistant",
            "message": {"content": [{"type": "text", "text": "Working in /tmp/quarry-bench/task-01"}]},
        },
    ]
    findings = gate_blinding(events, repo_root="/home/user/quarry")
    assert any(finding.fatal for finding in findings)


def test_gate_blinding_fails_on_repo_root_path():
    events = [
        {"type": "system", "subtype": "init", "model": "claude-opus-5", "session_id": "s"},
        {
            "type": "assistant",
            "message": {"content": [{"type": "text", "text": "Reading /home/user/quarry/internal/foo.go"}]},
        },
    ]
    findings = gate_blinding(events, repo_root="/home/user/quarry")
    assert any(finding.fatal for finding in findings)


""" CARD 9: FILESYSTEM AND DAEMON-STATE GATES """


def test_workspace_key_is_deterministic_and_differs_by_path(tmp_path):
    first = tmp_path / "app"
    second = tmp_path / "nested" / "app"
    first.mkdir()
    second.mkdir(parents=True)
    assert workspace_key(str(first)) == workspace_key(str(first))
    assert workspace_key(str(first)) != workspace_key(str(second))


def test_resolve_state_dir_raises_on_quarry_state_dir_env(tmp_path):
    with pytest.raises(GateError):
        resolve_state_dir(str(tmp_path), str(tmp_path), env={"QUARRY_STATE_DIR": "/somewhere"})


def test_resolve_state_dir_raises_on_quarry_build_tags_env(tmp_path):
    with pytest.raises(GateError):
        resolve_state_dir(str(tmp_path), str(tmp_path), env={"QUARRY_BUILD_TAGS": "integration"})


def test_resolve_state_dir_resolves_normally_against_scrubbed_env_despite_ambient(tmp_path, monkeypatch):
    monkeypatch.setenv("QUARRY_STATE_DIR", "/should-not-be-read")
    monkeypatch.setenv("QUARRY_BUILD_TAGS", "integration")
    resolved = resolve_state_dir(str(tmp_path), str(tmp_path), env={})
    assert resolved == os.path.join(str(tmp_path), "quarry", workspace_key(str(tmp_path)))


def test_user_cache_dir_honours_xdg_cache_home(monkeypatch, tmp_path):
    monkeypatch.setenv("XDG_CACHE_HOME", str(tmp_path))
    assert user_cache_dir() == str(tmp_path)


def test_user_cache_dir_falls_back_to_home_cache(monkeypatch):
    monkeypatch.delenv("XDG_CACHE_HOME", raising=False)
    assert user_cache_dir() == os.path.join(os.path.expanduser("~"), ".cache")


def _write_daemon_state(target_dir, cache_dir, pid, lang="go"):
    state_dir = resolve_state_dir(target_dir, cache_dir, env={})
    path = Path(daemon_state_file(state_dir, lang))
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps({"pid": pid, "address": "unix;x", "protocol_version": "1", "started_at": "now"}))


def test_gate_cold_before_passes_on_empty_cache_dir(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    assert gate_cold_before(str(target_dir), str(cache_dir), env={}) == []


def test_gate_cold_before_fails_when_daemon_alive(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    _write_daemon_state(str(target_dir), str(cache_dir), pid=os.getpid())
    findings = gate_cold_before(str(target_dir), str(cache_dir), env={})
    assert len(findings) == 1
    assert findings[0].fatal is True


def test_gate_cold_before_passes_when_daemon_json_present_but_pid_dead(tmp_path):
    # the state a previous failed attempt leaves behind -- daemon.json is
    # never removed on exit, only daemon.sock is.
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    dead_pid = _spawn_and_wait_for_exit()
    _write_daemon_state(str(target_dir), str(cache_dir), pid=dead_pid)
    assert gate_cold_before(str(target_dir), str(cache_dir), env={}) == []


def _spawn_and_wait_for_exit():
    """Spawns a process that exits immediately and returns its now-dead pid."""
    proc = subprocess.Popen([sys.executable, "-c", "pass"])
    proc.wait()
    return proc.pid


def test_clear_state_dir_removes_resolved_directory_and_leaves_sibling_untouched(tmp_path):
    target_dir = tmp_path / "worktree"
    sibling_dir = tmp_path / "sibling-worktree"
    target_dir.mkdir()
    sibling_dir.mkdir()
    cache_dir = tmp_path / "cache"
    _write_daemon_state(str(target_dir), str(cache_dir), pid=os.getpid())
    _write_daemon_state(str(sibling_dir), str(cache_dir), pid=os.getpid())

    clear_state_dir(str(target_dir), str(cache_dir), env={})

    assert not os.path.exists(resolve_state_dir(str(target_dir), str(cache_dir), env={}))
    assert os.path.exists(resolve_state_dir(str(sibling_dir), str(cache_dir), env={}))


@pytest.mark.parametrize("entry_name", ["CLAUDE.md", "CONSTRAINTS.md", ".claude"])
def test_gate_worktree_neutralised_fails_for_each_ambient_context_entry(tmp_path, entry_name):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    (worktree / entry_name).mkdir() if entry_name == ".claude" else (worktree / entry_name).write_text("x")
    findings = gate_worktree_neutralised(str(worktree))
    assert len(findings) == 1
    assert findings[0].fatal is True


def test_gate_worktree_neutralised_passes_when_none_present(tmp_path):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    assert gate_worktree_neutralised(str(worktree)) == []


def _init_git_repo(path):
    subprocess.run(["git", "-C", str(path), "init", "-q"], check=True)
    subprocess.run(["git", "-C", str(path), "config", "user.email", "test@example.com"], check=True)
    subprocess.run(["git", "-C", str(path), "config", "user.name", "test"], check=True)


def test_observe_worktree_dirtied_is_non_fatal_when_clean(tmp_path):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    _init_git_repo(worktree)
    finding = observe_worktree_dirtied(str(worktree))
    assert finding.fatal is False
    assert finding.gate == "worktree_dirtied"
    assert "False" in finding.message


def test_observe_worktree_dirtied_is_non_fatal_when_dirty(tmp_path):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    _init_git_repo(worktree)
    (worktree / "scratch.txt").write_text("dirty")
    finding = observe_worktree_dirtied(str(worktree))
    assert finding.fatal is False
    assert "True" in finding.message


def test_used_daemon_backed_tool_false_for_toc_only_transcript():
    events = load_fixture("cold-native-fallback.jsonl")
    assert used_daemon_backed_tool(events) is False


def test_used_daemon_backed_tool_true_for_daemon_backed_name():
    events = load_fixture("bundle-mixed-tools.jsonl")
    assert used_daemon_backed_tool(events) is True


def _events_with_workspace_symbol_call():
    return [
        {"type": "system", "subtype": "init", "model": "claude-opus-5", "session_id": "s"},
        {
            "type": "assistant",
            "message": {
                "content": [
                    {"type": "tool_use", "id": "tu_1", "name": "mcp__quarry__workspace_symbol", "input": {"targets": [{"query": "Foo"}]}}
                ]
            },
        },
    ]


def test_gate_cold_after_fatal_when_daemon_backed_used_and_no_daemon_json(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    findings = gate_cold_after(_events_with_workspace_symbol_call(), str(target_dir), str(cache_dir), env={})
    assert len(findings) == 1
    assert findings[0].fatal is True


def test_gate_cold_after_passes_when_daemon_backed_used_and_daemon_json_present(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    _write_daemon_state(str(target_dir), str(cache_dir), pid=os.getpid())
    assert gate_cold_after(_events_with_workspace_symbol_call(), str(target_dir), str(cache_dir), env={}) == []


def test_gate_cold_after_non_fatal_observation_for_toc_only_fixture(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    events = load_fixture("cold-native-fallback.jsonl")
    findings = gate_cold_after(events, str(target_dir), str(cache_dir), env={})
    assert len(findings) == 1
    assert findings[0].fatal is False
    assert findings[0].gate == "cold_no_daemon_backed_call"


def test_wait_for_daemon_exit_returns_immediately_when_no_daemon_json(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    wait_for_daemon_exit(str(target_dir), str(cache_dir), env={}, timeout_s=1)


def test_wait_for_daemon_exit_returns_once_a_live_pid_exits(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    proc = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(0.3)"])
    _write_daemon_state(str(target_dir), str(cache_dir), pid=proc.pid)
    # A direct child stays a zombie -- still "alive" to os.kill(pid, 0) --
    # until reaped, so a concurrent thread reaps it while the poll loop runs.
    reaper = threading.Thread(target=proc.wait)
    reaper.start()
    start = time.monotonic()
    wait_for_daemon_exit(str(target_dir), str(cache_dir), env={}, timeout_s=5)
    assert time.monotonic() - start < 5
    reaper.join()


def test_wait_for_daemon_exit_raises_on_timeout_against_a_pid_that_stays_alive(tmp_path):
    target_dir = tmp_path / "worktree"
    target_dir.mkdir()
    cache_dir = tmp_path / "cache"
    proc = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(5)"])
    try:
        _write_daemon_state(str(target_dir), str(cache_dir), pid=proc.pid)
        with pytest.raises(GateError):
            wait_for_daemon_exit(str(target_dir), str(cache_dir), env={}, timeout_s=0.3)
    finally:
        proc.kill()
        proc.wait()


""" CARD 10: RUN-STATE, INVALIDATION, AND THE COMPOSED GATE REPORT """


def test_run_dir_builds_expected_path(tmp_path):
    assert run_dir(tmp_path, "a5-bundle", 1) == tmp_path / "raw" / "a5-bundle" / "1"


def test_is_complete_true_for_state_complete(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "run.json").write_text(json.dumps({"state": "complete"}))
    assert is_complete(a_run_dir) is True


def test_is_complete_false_without_run_json(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    assert is_complete(a_run_dir) is False


def test_is_complete_false_for_other_state(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "run.json").write_text(json.dumps({"state": "invalidated"}))
    assert is_complete(a_run_dir) is False


def test_is_complete_false_with_answer_and_usage_but_no_score(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "answer.json").write_text("{}")
    (a_run_dir / "usage.json").write_text("{}")
    assert is_complete(a_run_dir) is False


def test_invalidate_moves_directory_aside_and_removes_run_json(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "run.json").write_text(json.dumps({"state": "complete"}))
    (a_run_dir / "answer.json").write_text("{}")

    new_path = invalidate(a_run_dir)

    assert new_path == tmp_path / "1.invalid-1"
    assert new_path.exists()
    assert not (new_path / "run.json").exists()
    assert (new_path / "answer.json").exists()
    assert not a_run_dir.exists()


def test_invalidate_second_time_lands_on_invalid_2(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "run.json").write_text(json.dumps({"state": "complete"}))
    invalidate(a_run_dir)

    a_run_dir.mkdir()
    (a_run_dir / "run.json").write_text(json.dumps({"state": "complete"}))
    second = invalidate(a_run_dir)

    assert second == tmp_path / "1.invalid-2"


def _fake_ladder_and_config(allowed=(), cold=False):
    ladder = SimpleNamespace(quarry_tools=("toc_file", "toc_dir", "workspace_symbol", "impact"))
    config = SimpleNamespace(allowed=allowed, cold=cold)
    return ladder, config


def test_run_gates_fails_when_a_fatal_gate_fails(tmp_path):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    _init_git_repo(worktree)
    ladder, config = _fake_ladder_and_config(allowed=("toc_file",))
    events = _events_with_model("claude-sonnet-5")  # mismatches the pinned run_model below
    report = run_gates(
        events, ladder, config, run_model="claude-opus-5", repo_root=str(tmp_path),
        worktree=str(worktree), a_run_dir=tmp_path / "1", cache_dir=str(tmp_path / "cache"), env={},
    )
    assert report.passed is False


def test_run_gates_passes_and_carries_non_fatal_observations(tmp_path):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    _init_git_repo(worktree)
    ladder, config = _fake_ladder_and_config(allowed=("toc_file",))
    events = _events_with_model("claude-opus-5")
    report = run_gates(
        events, ladder, config, run_model="claude-opus-5", repo_root=str(tmp_path),
        worktree=str(worktree), a_run_dir=tmp_path / "1", cache_dir=str(tmp_path / "cache"), env={},
    )
    assert report.passed is True
    assert any(finding.gate == "worktree_dirtied" for finding in report.findings)


def test_run_gates_passes_on_a_directory_holding_only_answer_and_usage(tmp_path):
    worktree = tmp_path / "worktree"
    worktree.mkdir()
    _init_git_repo(worktree)
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "answer.json").write_text("{}")
    (a_run_dir / "usage.json").write_text("{}")

    ladder, config = _fake_ladder_and_config(allowed=("toc_file",))
    events = _events_with_model("claude-opus-5")
    report = run_gates(
        events, ladder, config, run_model="claude-opus-5", repo_root=str(tmp_path),
        worktree=str(worktree), a_run_dir=a_run_dir, cache_dir=str(tmp_path / "cache"), env={},
    )
    assert report.passed is True


def test_gate_run_complete_artifacts_fails_on_directory_missing_scoring_files(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    (a_run_dir / "answer.json").write_text("{}")
    (a_run_dir / "usage.json").write_text("{}")
    findings = gate_run_complete_artifacts(a_run_dir)
    assert len(findings) == 2
    assert all(finding.fatal for finding in findings)


def test_gate_run_complete_artifacts_passes_once_all_four_files_exist(tmp_path):
    a_run_dir = tmp_path / "1"
    a_run_dir.mkdir()
    for name in ("answer.json", "answer.redacted.json", "usage.json", "score.json"):
        (a_run_dir / name).write_text("{}")
    assert gate_run_complete_artifacts(a_run_dir) == []


def test_write_run_json_stamps_complete_state_and_timestamp(tmp_path):
    a_run_dir = tmp_path / "1"
    payload = {"config_id": "a5-bundle", "rep": 1, "run_model": "claude-opus-5", "observations": []}
    record = write_run_json(a_run_dir, payload)

    assert record["state"] == "complete"
    assert "timestamp" in record
    on_disk = json.loads((a_run_dir / "run.json").read_text())
    assert on_disk["state"] == "complete"
    assert on_disk["config_id"] == "a5-bundle"


def test_gate_finding_and_report_shapes():
    passing_finding = GateFinding(gate="x", message="ok", fatal=False)
    failing_finding = GateFinding(gate="y", message="bad", fatal=True)

    assert GateReport(findings=(passing_finding,)).passed is True
    assert GateReport(findings=(passing_finding, failing_finding)).passed is False
