# Batch: usage-extraction

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "usage-extraction"
number: 2
cards: 2
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_extract_usage.py -q
depends-on: [1]
```

## Batch Scope

This batch delivers the suite's metric extractor and the tracked transcript fixtures it is written against. It is the discussion's strongest TDD candidate: its entire job is turning a run's `stream-json` transcript into the numbers `#006` proved cannot be trusted when produced by hand. Fixtures come first so the extractor is written against committed inputs rather than against a live run.

The external interface batches 3, 5, and 6 consume is `extract_usage.py`'s module-level functions: `read_transcript`, `iter_tool_uses`, and `extract_usage`. `gates.py` (batch 3) reuses `read_transcript` and `iter_tool_uses` rather than re-parsing the transcript itself.

Batch-local decision: the transcript event shapes this batch parses are the ones the installed client actually emits, confirmed while planning against `claude -p --output-format stream-json --verbose`. A `system`/`init` event carries `tools`, `mcp_servers`, `model`, `permissionMode`, `cwd`, and `session_id`. An `assistant` event carries `message.model`, `message.usage`, and `message.content[]` with `type: "tool_use"` blocks bearing `name` and `input`. A `user` event carries `message.content[]` with `type: "tool_result"` blocks bearing `is_error`. The terminal `result` event carries `duration_ms`, `duration_api_ms`, `num_turns`, `total_cost_usd`, `usage`, `modelUsage`, `permission_denials`, `is_error`, `subtype`, and `result`. This is the only transcript format the suite parses.

## Cards

### Card 6: commit hand-trimmed transcript fixtures

- **Context:**
  - `bench/loomyard-eval/ladder/tests/conftest.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/tests/fixtures/bundle-mixed-tools.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/zero-tool-calls.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/denied-attempt.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/errored-tool-result.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/targetdir-override.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/none-target-origin-mention.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/cold-native-fallback.jsonl`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Each fixture is a small hand-authored JSONL file — a `system`/`init` event, a handful of `assistant`/`user` events, and one terminal `result` event — matching the event shapes named in this batch's scope. They are test inputs, not captures of real sessions, and every one must be small enough to read in full.
  - `bundle-mixed-tools.jsonl`: an `a5-bundle`-shaped run mixing quarry and non-quarry tool calls — at least one `mcp__quarry__toc_file`, one `mcp__quarry__workspace_symbol`, two `Read`, one native `Grep`, one `Bash` whose `input.command` contains `grep -n`, and one `Bash` whose command contains `rg ` and one whose command contains neither. Two `assistant` events carrying distinct `message.usage` objects, so summation across events is exercised rather than a single-event read.
  - `zero-tool-calls.jsonl`: a run whose assistant events contain only `text` blocks.
  - `denied-attempt.jsonl`: a run whose terminal `result` event carries a non-empty `permission_denials` array naming a `mcp__quarry__impact` attempt, and whose `init` event's `tools` array does not contain that name.
  - `errored-tool-result.jsonl`: a run containing a `tool_result` block with `is_error: true`.
  - `targetdir-override.jsonl`: a run containing an `mcp__quarry__impact` tool call whose `input` carries a `targetDir` key, and a second containing `buildTags`.
  - `none-target-origin-mention.jsonl`: a `none`-shaped run — empty `mcp_servers`, no `mcp__quarry__*` name anywhere — whose transcript contains the word `quarry` only inside a `tool_result` payload that reads as file content from the task worktree.
  - `cold-native-fallback.jsonl`: an `a5-bundle-cold`-shaped run that completed normally using **only** `mcp__quarry__toc_file` and `mcp__quarry__toc_dir` quarry calls, plus non-quarry calls. It carries no call to any daemon-backed tool, which is exactly the third cold-cell outcome batch 3's gate must distinguish from a native fallback: the toc handlers never start a daemon, so this run writes no daemon state and yet is not a fallback.
- **Commit:** `test(bench): add tracked transcript fixtures for the ladder suite`

### Card 7: extract per-run metrics from a transcript

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/ladder/tests/fixtures/bundle-mixed-tools.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/zero-tool-calls.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/denied-attempt.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/errored-tool-result.jsonl`
  - `bench/loomyard-eval/scripts/gen_compact_toc.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/tests/test_extract_usage.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests against the committed fixtures first, then the extractor. Module-level functions:
  - `read_transcript(path)` — parse the JSONL into a list of event dicts, skipping blank lines and raising `TranscriptError` naming the line number on a malformed line.
  - `iter_tool_uses(events)` — yield `(name, input_dict)` for every `tool_use` block in every `assistant` event, in transcript order.
  - `init_event(events)` and `result_event(events)` — return the single `system`/`init` and terminal `result` event, raising `TranscriptError` when either is absent.
  - `extract_usage(events, wall_clock_ms)` — return the `usage.json` mapping.

  The returned mapping's fields, each named exactly as the discussion names them: `duration_ms` (the result event's own `duration_ms`), `wall_clock_ms` (the harness-measured value passed in), `tokens` (a nested mapping with `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, each summed across every assistant event's `message.usage` and none of them derived from any other), `result_usage` (the result event's `usage` object verbatim, for cross-checking only), `cost_usd` (the result event's `total_cost_usd`, or `null` when absent), `num_turns`, `tool_uses` (the total count of `tool_use` blocks), `tool_uses_breakdown` (a per-tool-name count covering every tool invoked, quarry and non-quarry alike), `quarry_tool_uses` (the subtotal over names beginning with `MCP_PREFIX`), `bash_grep_count`, `grep_tool_count`, `grep_fallback_total`, `denied_tool_attempts`, `advertised_tools` (the init event's `tools` array), `model` (the init event's `model`), `session_id`, and `transcript` (the transcript path).

  `bash_grep_count` counts `Bash` tool calls whose `input.command` matches a `grep` or `rg` invocation and **nothing else** — a native `Grep` tool call must never increment it, since this field is held to `#006`'s exact Bash-only definition. `grep_tool_count` counts native `Grep` tool calls and nothing else. `grep_fallback_total` is derived as the sum of the two and is never substituted for either. `denied_tool_attempts` is the length of the result event's `permission_denials` array; an attempt counts as an attempt, never as a successful call.

  Tests, one per discussion-named scenario: per-tool counting across mixed quarry and non-quarry tools; `bash_grep_count` counting only the matching Bash commands and being unmoved by the native `Grep` call; `grep_tool_count` counting only the native `Grep` call; `grep_fallback_total` equalling their sum and differing from each; a denied attempt counted as an attempt; every token class extracted separately, summed across both assistant events, with none silently summed into another; a run with zero tool calls yielding zero counts and an empty breakdown; and a transcript containing an errored tool result parsing without raising and still counting that call. One further test per documented raise, since the enumerated scenarios otherwise exercise only well-formed input: `read_transcript` raises `TranscriptError` naming the offending line number on a malformed line, `init_event` raises when no `system`/`init` event is present, and `result_event` raises when the terminal `result` event is absent — the last being the shape a run that died mid-way actually leaves behind. Add a CLI entry point under `if __name__ == "__main__":` taking a transcript path and an optional `--wall-clock-ms` and writing the mapping as JSON to stdout.
- **Commit:** `feat(bench): extract per-run benchmark metrics from a run transcript`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_extract_usage.py`, the only test file this batch creates. It covers every extraction unit against the fixtures card 6 commits; the fixtures themselves have no assertions of their own beyond being parseable, which the extractor tests exercise on every load. The scope is one file because this batch adds no shared helper that other test files import — batch 3 imports `extract_usage`'s functions but ships its own test file and its own `verify:` scope.
