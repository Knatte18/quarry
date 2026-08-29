"""
Tests for extract_usage.py against the tracked transcript fixtures under
tests/fixtures/. Each test targets one discussion-named extraction unit;
the fixtures themselves carry no assertions of their own beyond being
parseable, which every test here exercises on load.
"""
from pathlib import Path

import pytest

from extract_usage import (
    TranscriptError,
    extract_usage,
    init_event,
    iter_tool_uses,
    read_transcript,
    result_event,
)

FIXTURES_DIR = Path(__file__).resolve().parent / "fixtures"


def _load(name):
    return read_transcript(FIXTURES_DIR / name)


def test_per_tool_counting_across_mixed_quarry_and_non_quarry_tools():
    events = _load("bundle-mixed-tools.jsonl")
    usage = extract_usage(events, wall_clock_ms=46000)

    assert usage["tool_uses"] == 8
    assert usage["quarry_tool_uses"] == 2
    assert usage["tool_uses_breakdown"] == {
        "mcp__quarry__toc_file": 1,
        "mcp__quarry__workspace_symbol": 1,
        "Read": 2,
        "Grep": 1,
        "Bash": 3,
    }


def test_bash_grep_count_counts_only_matching_bash_commands():
    events = _load("bundle-mixed-tools.jsonl")
    usage = extract_usage(events, wall_clock_ms=46000)

    # Two Bash calls match ("grep -n ..." and "rg ..."); the native Grep
    # tool call and the unrelated "go build ./..." Bash call must not move it.
    assert usage["bash_grep_count"] == 2


def test_grep_tool_count_counts_only_the_native_grep_call():
    events = _load("bundle-mixed-tools.jsonl")
    usage = extract_usage(events, wall_clock_ms=46000)

    assert usage["grep_tool_count"] == 1


def test_grep_fallback_total_is_the_sum_and_differs_from_each_component():
    events = _load("bundle-mixed-tools.jsonl")
    usage = extract_usage(events, wall_clock_ms=46000)

    assert usage["grep_fallback_total"] == usage["bash_grep_count"] + usage["grep_tool_count"]
    assert usage["grep_fallback_total"] == 3
    assert usage["grep_fallback_total"] != usage["bash_grep_count"]
    assert usage["grep_fallback_total"] != usage["grep_tool_count"]


def test_denied_attempt_is_counted_as_an_attempt():
    events = _load("denied-attempt.jsonl")
    usage = extract_usage(events, wall_clock_ms=31000)

    assert usage["denied_tool_attempts"] == 1
    # advertised_tools does not carry the denied name -- the init event's
    # own tools array never contained mcp__quarry__impact.
    assert "mcp__quarry__impact" not in usage["advertised_tools"]


def test_every_token_class_extracted_separately_and_summed_across_events():
    events = _load("bundle-mixed-tools.jsonl")
    usage = extract_usage(events, wall_clock_ms=46000)

    # Fixture carries two assistant events with distinct usage objects:
    # (100, 50, 10, 5) and (200, 80, 20, 15).
    assert usage["tokens"] == {
        "input_tokens": 300,
        "output_tokens": 130,
        "cache_read_input_tokens": 30,
        "cache_creation_input_tokens": 20,
    }
    # None of the four classes was silently folded into another.
    assert len({usage["tokens"]["input_tokens"], usage["tokens"]["output_tokens"],
                usage["tokens"]["cache_read_input_tokens"], usage["tokens"]["cache_creation_input_tokens"]}) == 4


def test_zero_tool_calls_yields_zero_counts_and_empty_breakdown():
    events = _load("zero-tool-calls.jsonl")
    usage = extract_usage(events, wall_clock_ms=6000)

    assert usage["tool_uses"] == 0
    assert usage["tool_uses_breakdown"] == {}
    assert usage["quarry_tool_uses"] == 0
    assert usage["bash_grep_count"] == 0
    assert usage["grep_tool_count"] == 0
    assert usage["grep_fallback_total"] == 0


def test_errored_tool_result_parses_without_raising_and_still_counts_the_call():
    events = _load("errored-tool-result.jsonl")
    usage = extract_usage(events, wall_clock_ms=11000)

    assert usage["tool_uses"] == 1
    assert usage["tool_uses_breakdown"] == {"Read": 1}


def test_iter_tool_uses_yields_name_input_pairs_in_transcript_order():
    events = _load("bundle-mixed-tools.jsonl")

    names = [name for name, _ in iter_tool_uses(events)]
    assert names == [
        "mcp__quarry__toc_file",
        "mcp__quarry__workspace_symbol",
        "Read",
        "Read",
        "Grep",
        "Bash",
        "Bash",
        "Bash",
    ]


def test_extract_usage_carries_result_and_init_fields_through_verbatim():
    events = _load("bundle-mixed-tools.jsonl")
    usage = extract_usage(events, wall_clock_ms=46000)

    assert usage["duration_ms"] == 45000
    assert usage["wall_clock_ms"] == 46000
    assert usage["cost_usd"] == 0.42
    assert usage["num_turns"] == 4
    assert usage["result_subtype"] == "success"
    assert usage["result_is_error"] is False
    assert usage["model"] == "claude-opus-5"
    assert usage["session_id"] == "a5-bundle-001"


def test_read_transcript_raises_transcript_error_naming_the_line_number(tmp_path):
    transcript_path = tmp_path / "malformed.jsonl"
    transcript_path.write_text('{"type": "system", "subtype": "init"}\n{not valid json\n')

    with pytest.raises(TranscriptError, match="line 2"):
        read_transcript(transcript_path)


def test_init_event_raises_when_no_system_init_event_is_present():
    events = [{"type": "assistant", "message": {"content": []}}, {"type": "result", "duration_ms": 1}]

    with pytest.raises(TranscriptError):
        init_event(events)


def test_result_event_raises_when_terminal_result_event_is_absent():
    # The shape a run that died mid-way actually leaves behind: an init
    # event and some assistant activity, but no terminal result event.
    events = [
        {"type": "system", "subtype": "init", "tools": [], "session_id": "died-mid-way"},
        {"type": "assistant", "message": {"content": [{"type": "text", "text": "working..."}]}},
    ]

    with pytest.raises(TranscriptError):
        result_event(events)
