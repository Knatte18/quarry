# Batch: stream-and-metrics

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "stream-and-metrics"
number: 2
cards: 3
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1]
```

## Batch Scope

This batch turns a tee'd `stream-json` transcript into the per-rep numbers every downstream
consumer reads: the record types and the tolerant line parser, and the accounting layer over them.
It is one batch because the accounting rules are inseparable from the record shapes they reduce —
the grouping rule, the maximum-output rule and the tool-block walk are all statements about the
same structs. The external interface later batches consume is `ParseTranscript`, the `Record`
family, `SessionInit`, `ResultRecord` and `ComputeMetrics`.

Batch-local decision: the parser never fails on an unrecognised record type. Claude Code emits
record types this harness does not model (a rate-limit event, for one) and a transcript is
evidence that has already been paid for; discarding a whole rep because a new record type appeared
would be the worst possible trade.

## Cards

### Card 8: stream-json record types and the tolerant parser

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `stream.go` declaring the record shapes verified on
  this host, each with `json` tags. `Record` carries `Type`, `Subtype`, `UUID`, `Timestamp`,
  `ParentToolUseID`, an embedded `Message` and the raw bytes of the original line so a gate can
  re-marshal a whole transcript without loss. `Message` carries `ID`, `Model`, `Usage
  MessageUsage` and `Content []ContentBlock`. `MessageUsage` carries `InputTokens`,
  `OutputTokens`, `CacheReadInputTokens`, `CacheCreationInputTokens`. `ContentBlock` carries
  `Type`, `Text`, `Name`, `ID`, `ToolUseID`, `Input json.RawMessage` and `Content
  json.RawMessage` — one struct covering `text`, `tool_use` and `tool_result` blocks, discriminated
  by `Type`. `SessionInit` carries `Tools []string`, `MCPServers`, `Model`, `PermissionMode`,
  `ClaudeCodeVersion`, `MemoryPaths`, `Skills`, `SlashCommands` and `SessionID`; model
  `MemoryPaths` as a `map[string]string` so the auto-memory entry is readable by key rather than
  by position. `ResultRecord` carries `NumTurns`, `DurationMS`, `DurationAPIMS`, `TotalCostUSD`,
  `TerminalReason`, `StopReason`, `IsError` and `PermissionDenials []json.RawMessage`. Implement
  `ParseTranscript(r io.Reader) (*Transcript, error)` reading line-delimited JSON with a
  `bufio.Scanner` whose buffer is raised to at least 8 MiB (a single assistant record carrying a
  large tool result exceeds the default 64 KiB limit and would otherwise truncate the run's
  evidence silently); it decodes each line's `type` field first and skips any type it does not
  model rather than erroring, and it returns a `Transcript` holding the ordered `Record` slice, the
  first `system`/`init` record decoded into `SessionInit`, the final `result` record decoded into
  `ResultRecord`, and every raw line. A line that is not valid JSON at all is an error naming the
  line number — that is a truncated file, not an unknown record type. Implement
  `(*Transcript) MarshalAll() ([]byte, error)`, concatenating every raw line, which is what gate 2
  scans.
- **Commit:** `feat(ladder): add stream-json record types and a tolerant transcript parser`

### Card 9: the per-rep accounting layer

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `metrics.go` declaring a `Metrics` struct with
  `json` tags matching the field names the discussion fixes, and
  `ComputeMetrics(t *Transcript, mcpPrefix string) Metrics`. From the final result record copy
  `num_turns`, `duration_ms`, `duration_api_ms`, `total_cost_usd`, `terminal_reason`,
  `stop_reason`, `is_error` and `permission_denials` (both the count and the entries). Token counts
  are summed over assistant records **grouped by consecutive equal `message.id`**, never from the
  final result record's usage and never from `modelUsage`: port V1's `assistantCallGroups` and
  `perCallUsage` from `origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/usage.go` — a new
  group starts when the id is empty, differs from the previous id, or no group exists yet, and
  within a group the input and both cache figures are taken from any one record while
  `output_tokens` is the **maximum** across the group. Report `input_tokens`, `output_tokens`,
  `cache_read_input_tokens` and `cache_creation_input_tokens` separately and additionally report
  `input_tokens_total` as their input-plus-both-caches sum, alongside the three and never in place
  of them. Walk every `tool_use` content block for `tool_uses`, `tool_uses_breakdown` as a
  name-to-count map, and `quarry_tool_uses` as the count of uses whose name has the `mcpPrefix`
  argument as its prefix — take the prefix as a parameter, never a literal. Count `grep_tool_count`
  as native `Grep` uses and `bash_grep_count` as `Bash` uses whose decoded `command` input matches
  V1's regexp copied verbatim from that same `usage.go`, and report `grep_fallback_total` as their
  sum. Add the three metrics with no V1 counterpart: `tool_result_bytes` as the total UTF-8 byte
  length of every `tool_result` block's text content, `tool_result_bytes_breakdown` keyed by the
  tool name of the `tool_use` the block's tool-use id refers to, and `read_bytes` as the `Read`
  subset of that total. Record `model` from the first assistant record's message model. Add an
  `Effort` field that `ComputeMetrics` leaves empty and the run loop stamps, since the CLI does not
  echo the flag. Do not compute turn counts or durations by reconstructing them from timestamps —
  the result record reports both — and do not emit V1's retired
  `denied_tool_attempts`, `agent_id` or `transcript_source` fields.
- **Commit:** `feat(ladder): compute per-rep metrics from the transcript stream`

### Card 10: transcript fixtures and accounting tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/grouped-usage.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/max-turns.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/tool-bytes.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/stream_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** hand-author three small line-delimited fixtures, each opening with a
  `system`/`init` record and closing with a `result` record, and each carrying no absolute host
  path. The grouped-usage fixture must contain at least two assistant records sharing one message
  id with differing output-token values and a second group with its own id, chosen so that naive
  per-record summing over-counts and the expected totals are unambiguous; it must also contain one
  record of an unmodelled type between two real ones. The max-turns fixture's result record carries
  a max-turns terminal reason and no fenced answer. The tool-bytes fixture carries `tool_use`
  blocks for `Read`, `Grep` and `Bash` with matching `tool_result` blocks of known byte lengths,
  one `Bash` command that is a leading-word grep, one `Bash` command containing the substring
  `ripgrepping` that must not count, one `Bash` command of the piped `cat x | grep y` shape that
  must count, and one tool use whose name carries the `mcp__quarry__` prefix. Write `stream_test.go`
  asserting the unmodelled record type is skipped rather than erroring, that a truncated final line
  produces an error naming the line number, that a record longer than the scanner's default buffer
  is read whole, and that re-marshalling round-trips every raw line. Write `metrics_test.go`
  asserting per fixture: the grouping rule's totals and the maximum-output rule, the grep counts and
  the leading-command-word distinction, the byte metrics and the `Read` subset, the prefixed
  tool-use count under a prefix passed as an argument and a second assertion under a different
  prefix proving nothing is hardcoded, a zero-tool-call transcript reporting zeroes rather than
  absent fields, and the max-turns fixture's terminal reason surviving into the metrics.
- **Commit:** `test(ladder): cover transcript parsing and the accounting rules`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers this batch through `stream_test.go` and
`metrics_test.go` plus batch 1's two files, all in the one package this task creates. The three
committed fixtures are the whole evidence base for the accounting rules and are deliberately small
enough to read by hand — the grouping rule is the one V1 accounting rule that was load-bearing, and
a fixture nobody can verify by eye would not prove it.
