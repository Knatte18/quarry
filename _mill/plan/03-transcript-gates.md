# Batch: transcript-gates

```yaml
task: Port the capability-ladder bench harness to Go
batch: transcript-gates
number: 3
cards: 5
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [2]
```

## Batch Scope

Ports the half of `gates.py` that reads only a transcript: the finding and report value types, the
denied-tool and target-override gates, model pinning with its context-window-suffix normalisation, the
three-check blinding gate, the daemon-backed-call observation, and the new post-hoc `max_turns` ceiling
that replaces the `claude -p` client's `--max-turns` flag.

The external interface later batches consume is `GateFinding`, `GateReport`, and each exported gate
function; the environment-dependent gates and the aggregating `RunGates` land in batches 4 and 5.

Batch-local decision: the `/tmp/quarry-bench` literal check is dropped from `GateBlinding`. No such
binary exists in this suite, so the check can never fire and reads as coverage that is not there. The
identically-spelled branch in the redactor is a separate mechanism and is kept — see batch 6.

## Cards

### Card 13: Gate finding and report value types

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `GateError` as an exported `GateError` type, `GateFinding` as a struct
  carrying the gate name, whether the finding is fatal, and its message, and `GateReport` as a struct
  holding a finding slice with the same aggregate accessors the Python dataclass exposes (whether any
  fatal finding is present, and the fatal and non-fatal subsets). Port `_tool_results_by_id` as an
  unexported `toolResultsByID(records []Record) map[string]ContentBlock`. Test the report accessors
  over mixed fatal and non-fatal findings, and the tool-result indexing.
- **Commit:** `feat(ladder): add gate finding and report value types`

### Card 14: Denied-tool and target-override gates

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/targetdir-override.jsonl`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `gate_denied_tools_not_used` as
  `GateDeniedToolsNotUsed(records []Record, deniedNames []string) []GateFinding` and
  `gate_no_target_override` as `GateNoTargetOverride(records []Record) []GateFinding`, both keeping their
  Python fatality and message shape. Both return a slice, matching the Python: the denied-tool gate emits
  one finding per offending call and the override gate one per offending key, so a singular return would
  silently collapse several violations into one. An empty slice is the passing result; there is no
  zero-value finding sentinel. Test each with a passing and a failing case; the override gate's
  failing case uses the reshaped override fixture.
- **Commit:** `feat(ladder): port denied-tool and target-override gates`

### Card 15: Model pinning gate

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_CONTEXT_WINDOW_SUFFIX_RE` and `_normalise_model_id` as an unexported
  `normaliseModelID`, and `gate_model_pinned` as
  `GateModelPinned(records []Record, runModel string) []GateFinding`, sourcing the reported id from the
  assistant records' `message.model` rather than from a `system`/`init` event — the doc comment must
  record that provenance change. The gate stays fatal. Test the matching case, the mismatching case,
  the `[1m]` context-window suffix normalising to a match, and a transcript with no assistant record at
  all.
- **Commit:** `feat(ladder): port the model-pinning gate onto assistant records`

### Card 16: Blinding gate

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/none-target-origin-mention.jsonl`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_redact_tool_result_content` as an unexported `redactToolResultContent` and
  `gate_blinding` as `GateBlinding(records []Record, repoRoot string) []GateFinding`, keeping exactly
  three checks, each fatal: an `mcp__quarry__` tool name anywhere in the transcript; any filesystem
  path into `repoRoot`; and a bare case-insensitive "quarry" mention outside a `tool_result` payload,
  with a mention confined to a `tool_result` staying non-fatal as it is today. Drop the
  sibling-suite binary-path literal check and record in the doc comment that it is dropped because no such
  binary exists in this suite, so the check could never fire. The session scratch directory is never
  treated as a leak — it is legitimately the subagent's own cwd — and the doc comment must say so.
  Test all four outcomes: fatal on an `mcp__quarry__` name, fatal on a repo-root path, fatal on a bare
  mention outside a tool result, and non-fatal for the reshaped origin-mention fixture whose mention is
  confined to a tool result.
- **Commit:** `feat(ladder): port the blinding gate and drop its dead literal check`

### Card 17: Daemon-backed-call observation and the post-hoc max_turns ceiling

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/cold-native-fallback.jsonl`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `used_daemon_backed_tool` as
  `UsedDaemonBackedTool(records []Record) bool`, deriving the tool set from `DaemonBackedTools` through
  `MCPName` rather than from a literal. Add a new `GateMaxTurns(records []Record, maxTurns int) []GateFinding`
  that is fatal when the count of assistant records exceeds `maxTurns`, producing the same `truncated`
  outcome semantics the client-side `--max-turns` flag used to produce. Its doc comment must state that
  the Agent Tool has no `--max-turns` equivalent so nothing bounds a run mid-flight, that the ceiling is
  therefore evaluated post hoc, and that the field's basis changed from the client's own turn
  accounting to assistant-record count — which is why the committed threshold was blanked. Test the
  observation against the reshaped cold-native-fallback fixture and a daemon-backed-call transcript,
  and the ceiling at, one below, and one above the limit.
- **Commit:** `feat(ladder): add the post-hoc max_turns gate and daemon-call observation`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `gates_test.go` plus every earlier test file
in the ladder subtree.
