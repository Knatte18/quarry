# Batch: harness-mcp-init-fix

```yaml
task: "Ladder, toc rerun (T7)"
batch: "harness-mcp-init-fix"
number: 2
cards: 3
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1]
```

## Batch Scope

The harness cannot measure a rung cell today. `SessionInit.MCPServers` is typed `[]string`, but
Claude Code 2.1.236 emits objects in the session-init record, so `ParseTranscript` fails on line 1
of every transcript whose session actually loaded an MCP server. The operator's informal run of this
very matrix showed `a2-toc-dir` going 0-for-6 — every attempt invalidated, the cell incomplete —
while the measured agent runs themselves succeeded. Control cells pass only because their MCP config
is empty and `[]` unmarshals into either shape, which is why the defect survived T2: the e2e tests'
synthetic transcripts never carried a connected server, so nothing in the suite has ever exercised
the real init shape.

This batch is the harness fix `## Shared Decisions`' `harness-fixes-restart-the-matrix` anticipates,
brought forward: the defect is known before the matrix starts rather than discovered during it, so it
costs a batch rather than an abandoned results root. It sits between the pre-matrix gates and the
matrix because the fix must be committed before repetition 1 — the clean-tree rule makes that a hard
ordering, and a fix landing mid-matrix would void the root.

The batch is scoped to `bench/loomyard-eval/ladder/` and touches nothing under test. Card 3 retypes
the field and threads the status through provenance; card 4 pins the real wire shape in a regression
test; card 5 makes a cell that was granted a server but did not get a working one fail loudly instead
of measuring a silently toolless run. The interface batch 3 consumes is a harness that can parse a
rung cell's transcript at all.

## Cards

### Card 3: Retype `SessionInit.MCPServers` and record the server status in the fingerprint

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/live_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `stream.go`, add an exported `MCPServerStatus` struct with two string fields,
  `Name` (json tag `name`) and `Status` (json tag `status`), and retype `SessionInit.MCPServers` from
  `[]string` to `[]MCPServerStatus`, updating its doc comment — it is an advertised server list with
  per-server connection status, no longer a name list. **The typed struct is required rather than
  `json.RawMessage`:** the names have a live consumer, so discarding them would silently disable
  fingerprint drift detection.
  In `provenance.go`, `NewSessionFingerprint` currently copies the field wholesale with
  `append([]string(nil), init.MCPServers...)`. Keep `SessionFingerprint.MCPServers` typed `[]string`
  and populate it with the `Name` values in the order the record lists them, so `provenance.json`'s
  existing shape is unchanged and no committed root becomes unreadable. Add a second field,
  `MCPServerStatuses map[string]string` with json tag `mcp_server_statuses`, mapping each name to its
  status, and populate it in the same function. Extend `diffSessionFingerprint` to report a
  difference in the status map alongside its existing `mcp_servers` clause, so a server that was
  connected for one repetition and not for another shows up as fingerprint drift rather than
  vanishing. Follow that function's existing per-field style: a guard, an appended
  `fmt.Sprintf` clause naming the json field and both sides.
  Read `live_test.go` to confirm the retype does not break its assertion — it tests
  `len(transcript.Init.MCPServers) != 0` for a control cell, which is length-only and survives the
  change. Read `run.go` only to confirm `NewSessionFingerprint`'s call site needs no edit. Do not
  change either file in this card.
- **Commit:** `fix(ladder): type session-init mcp_servers as objects and record server status`

### Card 4: Pin the real 2.1.236 init shape in a regression test

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream_test.go`
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/session-init-mcp-connected.jsonl`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a regression case whose init line is **copied verbatim from a real transcript
  this host produced**, never hand-written — a hand-written line would pin the shape someone believed
  the CLI emits, which is the assumption that produced this defect. Save the captured line as the new
  `testdata` fixture and load it from the test.
  **Obtaining the line, in priority order.** (a) If the operator supplies one of the informal run's
  invalidated `a2-toc-dir` transcripts, take the first line of that file. (b) Otherwise capture one:
  build the server, write an MCP config naming it via the same code path the harness uses, and run a
  single trivial `claude -p` turn under that config with stream-json output, keeping the first
  record. That is one cheap turn, not a measured repetition — do not use the pre-matrix live test for
  this, which costs a full 60-turn repetition and runs after this batch anyway. Whichever route is
  taken, record in the test's comment which one it was and the CLI version the line came from.
  The line must have a **non-empty** `mcp_servers` array carrying at least one object with `name` and
  `status` — an empty array reproduces exactly the control-cell case that already passed and would
  leave the defect uncovered. Assert that `ParseTranscript` succeeds on it, that the decoded
  `SessionInit.MCPServers` has the expected length, and that the first entry's `Name` and `Status`
  match the fixture. Follow the file's existing table-test convention rather than introducing a new
  one.
  Keep any existing case that exercises the empty-array shape, or add one if none exists: both shapes
  are real, and the control cells depend on the empty one continuing to work.
- **Commit:** `test(ladder): pin the real session-init mcp_servers shape from a captured transcript`

### Card 5: Invalidate a granted repetition whose server did not connect

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** A cell granted a server that never connected measures a toolless run. Averaging
  that into `a2-toc-dir`'s median would corrupt the exact number this task exists to produce, and
  nothing catches it today: `CheckGrantedToolUsed` fires only per cell, only after every repetition
  has run, and is never fatal.
  In `gates.go`, add `CheckServerConnected` following the file's existing gate shape — it returns
  `*Finding`, `nil` when there is nothing to report. Give it the decoded `SessionInit` and the
  expected server name. It reports a finding when the cell was granted a server and the init record
  either does not list that server at all or lists it with a status other than `connected`. Include
  the observed status, or its absence, in the finding's message. A control cell — one whose allowed
  list is empty — is never its concern; keep that predicate on the caller's side, matching how the
  blinding checks are gated by `IsControl` at the call site rather than inside the check.
  In `run.go`, wire it as an **infrastructure failure, not a fatal finding.** Place the check inside
  the per-attempt loop, against the candidate transcript's `Init`, before the loop accepts that
  transcript and breaks. On a finding, do not break: let control fall through to the existing
  `InvalidateRep` call and its `attempts >= MaxAttempts` ceiling, exactly as a non-zero exit or an
  unparseable stream already does. **This is deliberate and the reason must not be lost:**
  `InvalidateRep` increments a counter derived from the `.invalid-<k>` directories on disk, so the
  existing ceiling bounds the retries for free, whereas the `writeCompleteState` blinding-failed path
  increments no counter at all and would re-attempt without limit. A server that failed to connect is
  also genuinely transient in a way a blinding failure is not — the next attempt may well succeed.
  Record the finding in the repetition's observations so the write-up can report it.
  Add table cases to `gates_test.go` covering: a granted cell whose server is `connected` (no
  finding), one whose server reports another status (a finding naming that status), one whose init
  record omits the server entirely (a finding), and a control cell, which the caller-side predicate
  keeps out — assert the check's own behaviour for the granted cases and leave the control gating to
  the call site.
- **Commit:** `feat(ladder): invalidate a granted repetition whose mcp server did not connect`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` — the harness's own offline suite, scoped to the
only tree this batch changes. It runs against the fake-runner layer under `testdata/`, needs no
network and no API budget, and skips the guarded live test because `LADDER_LIVE_TEST` is unset in the
verify environment. Card 4's new fixture case and card 5's new gate cases both land inside that
scope, as do the existing `provenance_test.go` fingerprint cases card 3's change must keep green.

The batch's own gate is card 4's regression case: it is the only test in the suite that has ever seen
a real connected-server init record, and it is what stops this defect from returning the next time
the CLI's wire format shifts. The one thing offline tests cannot prove is that the captured line
still matches what the CLI emits today — that is what the pre-matrix live test in card 6 checks,
before ten repetitions depend on it.
