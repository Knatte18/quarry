"""
Extracts per-run benchmark metrics from a captured
`claude -p --output-format stream-json --verbose` transcript, producing
exactly the mapping the ladder suite's `usage.json` commits to disk. Token
classes are summed across `assistant` events rather than read off the
result envelope's final-iteration-only `usage` object (see the plan's
"token classes are summed from assistant events" Shared Decision), and
`bash_grep_count` / `grep_tool_count` are kept in the exact separate shape
#006's own definitions used.

Usage:
    from extract_usage import read_transcript, iter_tool_uses, extract_usage
    events = read_transcript("raw/a5-bundle/1/transcript.jsonl")
    usage = extract_usage(events, wall_clock_ms=123456)
"""
import argparse
import json
import re
import sys

from ladder_config import MCP_PREFIX

# Matches a Bash `command` string invoking grep or ripgrep as a leading
# command word -- not merely containing the substring "grep" somewhere
# unrelated (e.g. inside a path). #006's own definition (README "Dispatch
# protocol" step 4) greps the transcript's Bash "command" fields for this
# shape, so bash_grep_count is held to it exactly.
_BASH_GREP_RE = re.compile(r"(^|[|&;\s])(grep|rg)\b")


class TranscriptError(Exception):
    """
    Raised when a transcript JSONL file cannot be parsed as this suite's
    stream-json shape: a malformed line, a missing system/init event, or a
    missing terminal result event.
    """


def read_transcript(path):
    """
    Parses a captured stream-json transcript into a list of event dicts, in
    the order the client emitted them.

    Blank lines are skipped. A line that is not valid JSON raises
    TranscriptError naming its 1-based line number.
    """
    events = []
    with open(path) as f:
        for line_number, raw_line in enumerate(f, start=1):
            line = raw_line.strip()
            if not line:
                continue
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError as exc:
                raise TranscriptError(f"{path}: malformed JSON on line {line_number}: {exc}") from exc
    return events


def iter_tool_uses(events):
    """
    Yields (name, input_dict) for every tool_use content block in every
    assistant event, in transcript order.
    """
    for event in events:
        if event.get("type") != "assistant":
            continue
        for block in event["message"]["content"]:
            if block.get("type") == "tool_use":
                yield block["name"], block.get("input", {})


def init_event(events):
    """
    Returns the single system/init event.

    Raises TranscriptError when the transcript carries none.
    """
    for event in events:
        if event.get("type") == "system" and event.get("subtype") == "init":
            return event
    raise TranscriptError("transcript has no system/init event")


def result_event(events):
    """
    Returns the terminal result event.

    Raises TranscriptError when the transcript carries none -- the shape a
    run that died mid-way actually leaves behind.
    """
    for event in events:
        if event.get("type") == "result":
            return event
    raise TranscriptError("transcript has no terminal result event")


def _is_bash_grep_command(command):
    """
    True when a Bash tool call's command string invokes grep or rg as a
    command word, matching #006's exact Bash-only grep-fallback definition.
    """
    return bool(_BASH_GREP_RE.search(command))


def extract_usage(events, wall_clock_ms, transcript_path=None):
    """
    Builds the usage.json mapping for one run from its parsed transcript
    events and the harness-measured wall_clock_ms.

    Every token class (input_tokens, output_tokens,
    cache_read_input_tokens, cache_creation_input_tokens) is summed
    independently across every assistant event's message.usage -- none is
    derived from another, and the result event's own usage object is kept
    only as result_usage, for cross-checking. bash_grep_count and
    grep_tool_count are counted strictly separately per #006's definitions;
    grep_fallback_total is their sum and never substituted for either.

    Args:
        transcript_path: recorded verbatim into the returned mapping's
            "transcript" field, so a caller that already knows the path
            (e.g. the CLI entry point below) does not have to re-derive it.

    Returns:
        The usage.json mapping: duration_ms, wall_clock_ms, tokens,
        result_usage, cost_usd, num_turns, tool_uses,
        tool_uses_breakdown, quarry_tool_uses, bash_grep_count,
        grep_tool_count, grep_fallback_total, denied_tool_attempts,
        result_subtype, result_is_error, advertised_tools, model,
        session_id, and transcript.
    """
    init = init_event(events)
    result = result_event(events)

    tokens = {
        "input_tokens": 0,
        "output_tokens": 0,
        "cache_read_input_tokens": 0,
        "cache_creation_input_tokens": 0,
    }
    for event in events:
        if event.get("type") != "assistant":
            continue
        event_usage = event["message"].get("usage", {})
        for token_class in tokens:
            tokens[token_class] += event_usage.get(token_class, 0)

    tool_uses_breakdown = {}
    bash_grep_count = 0
    grep_tool_count = 0
    for name, tool_input in iter_tool_uses(events):
        tool_uses_breakdown[name] = tool_uses_breakdown.get(name, 0) + 1
        if name == "Bash" and _is_bash_grep_command(tool_input.get("command", "")):
            bash_grep_count += 1
        elif name == "Grep":
            grep_tool_count += 1

    tool_uses = sum(tool_uses_breakdown.values())
    quarry_tool_uses = sum(count for name, count in tool_uses_breakdown.items() if name.startswith(MCP_PREFIX))

    return {
        "duration_ms": result["duration_ms"],
        "wall_clock_ms": wall_clock_ms,
        "tokens": tokens,
        "result_usage": result.get("usage"),
        "cost_usd": result.get("total_cost_usd"),
        "num_turns": result["num_turns"],
        "tool_uses": tool_uses,
        "tool_uses_breakdown": tool_uses_breakdown,
        "quarry_tool_uses": quarry_tool_uses,
        "bash_grep_count": bash_grep_count,
        "grep_tool_count": grep_tool_count,
        "grep_fallback_total": bash_grep_count + grep_tool_count,
        "denied_tool_attempts": len(result.get("permission_denials", [])),
        "result_subtype": result.get("subtype"),
        "result_is_error": result.get("is_error"),
        "advertised_tools": init.get("tools"),
        "model": init.get("model"),
        "session_id": init.get("session_id"),
        "transcript": transcript_path,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Extract per-run benchmark metrics from a stream-json transcript.")
    parser.add_argument("transcript", help="path to a captured stream-json transcript.jsonl")
    parser.add_argument("--wall-clock-ms", type=int, default=None, help="harness-measured wall-clock duration in milliseconds")
    args = parser.parse_args()

    transcript_events = read_transcript(args.transcript)
    usage_mapping = extract_usage(transcript_events, wall_clock_ms=args.wall_clock_ms, transcript_path=args.transcript)
    json.dump(usage_mapping, sys.stdout, indent=2)
    sys.stdout.write("\n")
