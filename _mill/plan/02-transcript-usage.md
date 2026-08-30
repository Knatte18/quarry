# Batch: transcript-usage

```yaml
task: Port the capability-ladder bench harness to Go
batch: transcript-usage
number: 2
cards: 5
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1]
```

## Rename mechanic

For each `Moves:` pair the implementer MUST:

1. Run `git mv <old> <new>` FIRST, before making any other change to the moved file.
2. Make ONLY surgical edits — touch only the lines that must change after the move.
3. Use a full-file `Creates:` entry only for genuinely new files that have no predecessor.
4. Never write the relocated file from scratch and delete the original — that breaks git rename history
   and inflates review diffs.

The seven fixtures moved by card 9 are an exception to point 2 in content only, not in mechanic: they
are `claude -p` stream-json today and must be re-shaped into the subagent transcript format, so their
bodies change substantially. The `git mv` still comes first, so the rename is recorded as a rename.

## Batch Scope

Ports `extract_usage.py` onto the new subagent transcript format: the record model, transcript reading,
the tool-use iterators, and `ExtractUsage` rebuilt around the metric partition the discussion fixes.
It also relocates the seven existing test fixtures into the Go package's `testdata/` and re-shapes them
into subagent transcript records, which every later gate batch reuses.

The external interface later batches consume is `Record`, `ReadTranscript`, `IterToolUses`,
`AssistantRecords`, `ExtractUsage`, and `DenialShapePattern`.

Batch-local decision: `ExtractUsage` takes the granted-tool list as a parameter rather than reading the
generated agent definition itself. The definition generator lands in a later batch, and threading the
list in from the caller keeps this package free of a dependency cycle while preserving the
"extracted, never self-reported" rule — the value still comes from a harness-generated file.

## Cards

### Card 8: Subagent transcript record model and readers

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Define a `Record` struct for one line of an `agent-<id>.jsonl` subagent transcript,
  carrying `ParentUUID`, `IsSidechain`, `AgentID`, `UUID`, `Timestamp` (RFC 3339 with milliseconds),
  `Type`, `Effort`, `ToolUseResult`, and a `Message` sub-struct with `Model`, `Content` (a slice of
  content blocks typed `text` / `thinking` / `tool_use` / `tool_result`), and `Usage` with
  `InputTokens`, `OutputTokens`, `CacheReadInputTokens`, `CacheCreationInputTokens`. Content blocks
  must retain the fields the gates need: `Type`, `Text`, `Name`, `Input`, `ToolUseID`, `IsError`, and
  `Content`. Port `read_transcript` as `ReadTranscript(path string) ([]Record, error)` and
  `TranscriptError` as an exported `TranscriptError` type, keeping the Python's behaviour of erroring
  on a malformed line rather than skipping it. Port `iter_tool_use_blocks` as `IterToolUseBlocks` and
  `iter_tool_uses` as `IterToolUses` returning name/input pairs in transcript order. Add
  `AssistantRecords(records []Record) []Record` filtering on `Type == "assistant"`. Do not port
  `init_event` or `result_event` — the subagent transcript has no `system`/`init` record and no
  terminal `result` record, and each Go doc comment for the fields that used to come from them must say
  so. Test reading a small inline transcript, the malformed-line error, and iterator ordering.
- **Commit:** `feat(ladder): add subagent transcript record model and readers`

### Card 9: Relocate and re-shape the transcript fixtures

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/tests/test_extract_usage.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `bench/loomyard-eval/ladder/tests/fixtures/none-target-origin-mention.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/none-target-origin-mention.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/errored-tool-result.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/errored-tool-result.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/targetdir-override.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/targetdir-override.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/cold-native-fallback.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/cold-native-fallback.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/denied-attempt.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/denied-attempt.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/bundle-mixed-tools.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/bundle-mixed-tools.jsonl`
  - `bench/loomyard-eval/ladder/tests/fixtures/zero-tool-calls.jsonl` -> `bench/loomyard-eval/ladder/internal/ladder/testdata/zero-tool-calls.jsonl`
- **Requirements:** After the `git mv`, re-shape each fixture from `claude -p` stream-json into the
  subagent transcript format the `Record` type models: drop the `system`/`init` and terminal `result`
  records entirely, set `isSidechain: true` and an `agentId` on every record, give each record an RFC
  3339 timestamp with milliseconds and a `uuid`, move the model onto each assistant record's
  `message.model`, and add a top-level `effort` to assistant records. Preserve each fixture's
  distinguishing property, which is the only reason it exists: the origin-mention fixture keeps its
  bare-quarry mention outside a tool result, the errored-tool-result fixture keeps its `is_error` tool
  result with a non-denial error text, the override fixture keeps its `--target-dir` override attempt,
  the cold-native-fallback fixture keeps a transcript with no daemon-backed tool call, the
  denied-attempt fixture keeps an `is_error` tool result whose text is permission-denial shaped, the
  bundle-mixed-tools fixture keeps its mix of quarry, Bash-grep, and Grep calls, and the
  zero-tool-calls fixture keeps its empty tool-use set. Do not delete the Python test files that
  currently read these fixtures — the documentation batch removes the Python tree wholesale, and
  keeping it readable until then is deliberate.
- **Commit:** `refactor(ladder): move transcript fixtures to Go testdata and re-shape them`

### Card 10: ExtractUsage — the fields whose definitions survive

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Define a `Usage` struct serialising to the `usage.json` shape, and start
  `ExtractUsage`. This card lands the fields whose definitions are unchanged by the dispatch swap: the
  four token classes summed independently across every assistant record's `message.usage` — none
  derived from another; `ToolUses`; `ToolUsesBreakdown`; `QuarryToolUses` counted by the `MCPPrefix`
  prefix; `BashGrepCount`; `GrepToolCount`; and `GrepFallbackTotal`. Port `_BASH_GREP_RE` as an
  unexported `bashGrepRe` and `_is_bash_grep_command` as `isBashGrepCommand`, keeping the
  leading-command-word semantics so a `grep` appearing inside a path does not count. `BashGrepCount`
  and `GrepToolCount` are counted strictly separately and are never merged; `GrepFallbackTotal` is
  their sum and is never substituted for either — say so in the doc comment. Test independent summation
  of all four token classes, the breakdown, the quarry count, and the leading-command-word distinction.
- **Commit:** `feat(ladder): port ExtractUsage token and tool-use metrics`

### Card 11: ExtractUsage — the changed, added, and dropped fields

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Complete `ExtractUsage` with the signature
  `ExtractUsage(records []Record, transcriptPath, transcriptSource string, grantedTools []string) (Usage, error)`.
  Changed fields: `DurationMs` is the last record's timestamp minus the first's; `NumTurns` is the count
  of assistant records; `Model` comes from the assistant records' `message.model`; `DeniedToolAttempts`
  is the count of `tool_result` blocks with `is_error` set whose text matches a permission-denial shape.
  Added fields: `Effort` from the assistant records' top-level `effort`, `AgentID`, `TranscriptSource`,
  and `DeniedToolAttemptsProvisional` which is always emitted `true` by this port. Renamed field:
  `advertised_tools` becomes `GrantedTools`, populated from the `grantedTools` parameter rather than
  from the transcript. Dropped entirely, with no field emitted and no placeholder null: `cost_usd`,
  `wall_clock_ms`, `result_usage`, `result_subtype`, `result_is_error`, and `session_id`. Hold the
  denial-shape match in one named exported constant `DenialShapePattern`, whose doc comment states
  that it is unvalidated against a real denial record because this task dispatches nothing that
  provokes one, that `DeniedToolAttemptsProvisional` exists to flag exactly that, and that the
  follow-up matrix task's deny-list probe is what clears it. Test the timestamp-derived duration, the
  assistant-record turn count, the denial count against the reshaped denied-attempt and
  errored-tool-result fixtures — the latter must count zero — and that no dropped field appears in the
  serialised JSON.
- **Commit:** `feat(ladder): rebuild ExtractUsage fields for subagent transcripts`

### Card 12: Fixture-driven usage extraction test

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/bundle-mixed-tools.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/zero-tool-calls.jsonl`
  - `bench/loomyard-eval/ladder/tests/test_extract_usage.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/usage_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add table-driven cases running `ExtractUsage` end to end over the reshaped
  fixtures, asserting the full `Usage` value for the mixed-tools fixture and the zero-tool-call case,
  and asserting that `GrantedTools` reflects the parameter passed rather than anything in the
  transcript. Carry over every assertion the Python suite made that still has meaning under the new
  format; where an assertion covered a dropped field, drop the assertion rather than substituting a
  near-equivalent.
- **Commit:** `test(ladder): cover ExtractUsage against the reshaped fixtures`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `transcript_test.go` and `usage_test.go`
alongside the batch-1 tests, all inside the ladder subtree.
