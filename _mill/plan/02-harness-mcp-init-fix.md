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
  - `bench/loomyard-eval/ladder/internal/ladder/provenance_test.go`
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
  **Cover both additions in `provenance_test.go`, because nothing existing does.**
  `TestCompareFingerprints` builds its fingerprints from named-field literals, so it stays green
  while exercising neither the new map nor the new diff clause. Add a case asserting
  `NewSessionFingerprint` populates `MCPServerStatuses` from a retyped init record — names as keys,
  statuses as values, with `MCPServers` still carrying the names in record order — and a
  `diffSessionFingerprint` case where two fingerprints agree on every field including the name list
  and differ only in a server's status, asserting the returned string reports that difference.
  Without the second case a server that silently stopped connecting mid-cell would read as no drift
  at all, which is the failure this field exists to prevent.
- **Commit:** `fix(ladder): type session-init mcp_servers as objects and record server status`

### Card 4: Pin the real 2.1.236 init shape in a regression test

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `.scratch/orch-evidence/init-line.json`
  - `.scratch/orch-evidence/a2-invalid-real-transcript.jsonl`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/tool-bytes.jsonl`
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/session-init-mcp-connected.jsonl`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a regression case whose init line is **copied verbatim from a real transcript
  this host produced**, never hand-written — a hand-written line would pin the shape someone believed
  the CLI emits, which is the assumption that produced this defect. Save the captured line as the new
  `testdata` fixture and load it from the test.
  **The line is already on disk — do not capture a new one.** The orchestrator supplied a real
  invalidated `a2-toc-dir` transcript from the informal run at
  `.scratch/orch-evidence/a2-invalid-real-transcript.jsonl`, with its init record extracted alone to
  `.scratch/orch-evidence/init-line.json`. That is a genuine 2.1.236 record from this host, produced
  by the matrix's own code path against a connected quarry server; it carries
  `"mcp_servers":[{"name":"quarry","status":"connected"}]`, `mcp__quarry__toc` in `tools`, and
  `"claude_code_version":"2.1.236"`. Use it. Do not spend a `claude -p` turn re-deriving evidence
  that already exists.
  Both evidence files live under a gitignored directory and do not travel with the branch. If they
  are absent — a fresh clone, or another machine — **stop and ask the operator for the line**; do not
  hand-write one and do not spend a measured call re-deriving it. Once this card commits the scrubbed
  fixture the evidence is no longer needed, because the fixture itself is the tracked record.
  **Scrub before committing, and scrub only what must be scrubbed.** The record carries machine
  paths and session identifiers that no tracked file in this repository may hold: `cwd` (an absolute
  ladder-worktree path), `session_id`, `uuid`, `memory_paths.auto` (an absolute auto-memory path —
  note it resolves into the Loomyard checkout, confirming whose memory `ScanMemoryPaths` walks), and
  `messaging_socket_path` (an absolute socket path carrying a pid). Replace each value with an
  obviously-synthetic stand-in of the same JSON type, and **keep every key present and every other
  value byte-verbatim** — above all `mcp_servers`, `tools`, `model`, `permissionMode` and
  `claude_code_version`, which are the fields the fixture exists to pin. Do not drop keys you think
  are irrelevant: the point of a captured fixture is that it carries the shape the CLI actually
  emits, including fields no current code reads. Record in the test's comment that the line came from
  a real invalidated `a2-toc-dir` repetition, name the CLI version, and state that only machine paths
  and identifiers were substituted.
  The line must have a **non-empty** `mcp_servers` array carrying at least one object with `name` and
  `status` — an empty array reproduces exactly the control-cell case that already passed and would
  leave the defect uncovered. Assert that `ParseTranscript` succeeds on it, that the decoded
  `SessionInit.MCPServers` has the expected length, and that the first entry's `Name` and `Status`
  match the fixture. Follow the file's existing table-test convention rather than introducing a new
  one.
  Keep any existing case that exercises the empty-array shape, or add one if none exists: both shapes
  are real, and the control cells depend on the empty one continuing to work.
  **Card 3's retype breaks one existing fixture, and this card owns repairing it.**
  `testdata/transcripts/tool-bytes.jsonl` carries `"mcp_servers":["quarry"]` on its init line — the
  old string-array shape — so after the retype `ParseTranscript` errors on it and takes down the
  cases in `stream_test.go` and `metrics_test.go` that load it. Convert that line's array to the
  object shape, changing nothing else about the fixture: it exists to exercise tool-result byte
  accounting, not the init record, so its other fields and every later line stay byte-identical. It
  is the only fixture with the string-array shape — `fakeclaude/main.go` hardcodes an empty array and
  every other transcript fixture carries `[]` — but re-check before assuming that still holds, and
  convert any other non-empty string-array init line the same way.
- **Commit:** `test(ladder): pin the real session-init mcp_servers shape from a captured transcript`

### Card 5: Invalidate a granted repetition whose server did not connect

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
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
  **A nil init record is a finding, not a pass, and the check must never dereference it.**
  `transcript.Init` is nil for a transcript carrying no init line — `run.go` nil-guards it at both of
  its existing uses, and the attempt loop's accept branch tests only the result record, so a nil can
  reach this call site. Take the pointer, guard it first, and return a finding when it is nil: for a
  granted cell, a transcript with no init record cannot show the server ever loaded, and treating an
  absence of evidence as a pass is the reasoning this whole batch exists to undo. Word that finding
  distinctly from the server-absent one, so the two are told apart in the reason file.
  In `run.go`, wire it as an **infrastructure failure, not a fatal finding.** Place the check inside
  the per-attempt loop, against the candidate transcript's `Init`, before the loop accepts that
  transcript and breaks. On a finding, do not break: let control fall through to the existing
  `InvalidateRep` call and its `attempts >= MaxAttempts` ceiling, exactly as a non-zero exit or an
  unparseable stream already does. **This is deliberate and the reason must not be lost:**
  `InvalidateRep` increments a counter derived from the `.invalid-<k>` directories on disk, so the
  existing ceiling bounds the retries for free, whereas the `writeCompleteState` blinding-failed path
  increments no counter at all and would re-attempt without limit. A server that failed to connect is
  also genuinely transient in a way a blinding failure is not — the next attempt may well succeed.
  **Persist the finding as a file, because there is nowhere else for it to go.** `run.go` has no
  logger and writes nothing to standard error, and this path never reaches `writeCompleteState`, so
  the repetition's `observations` slice — assembled after the attempt loop — is out of reach: an
  attempt-exhausted repetition returns with no state file at all. What the path does have is the
  attempt's own directory, which `InvalidateRep` **renames rather than deletes**, so anything written
  into it before the rename survives at `<root>/raw/<cell>/<rep>.invalid-<k>/` — the directories the
  matrix card already lists to decide whether a repetition is attempt-exhausted, and the write-up
  card already reads. Add a file-name constant to `runstate.go` alongside the existing per-repetition
  names, and write the finding into the attempt directory as that file immediately before calling
  `InvalidateRep`. Name the cell, the repetition, the expected server and the observed status, or its
  absence.
  **Bound the spend across repetitions, not just within one.** `MaxAttempts` caps retries at 3 *per
  repetition*, which for a five-repetition cell permits 15 measured `claude -p` calls — against a
  whole-task budget of 10 — and a systematic connect failure is exactly the case this check exists
  to catch, so that ceiling would be reached, not approached. Use the disposition `run.go` already
  carries for a tainted memory-path scan: when a repetition exhausts its attempts **and every one of
  those attempts failed this check**, return a `repOutcome` with `abortRun` set, stopping the whole
  invocation rather than moving to the next repetition. Three consecutive failures to connect the
  same server is a configuration or environment fault, not bad luck, and the next repetition will
  reproduce it at the cost of three more measured calls. That caps the blast radius of a systematic
  failure at 3 calls instead of 15, and surfaces it while the operator can still act rather than
  after the invocation returns. Keep the ordinary within-repetition retries intact: a single failed
  attempt followed by a successful one is transient and must not abort anything.
  Add table cases to `gates_test.go` covering: a granted cell whose server is `connected` (no
  finding), one whose server reports another status (a finding naming that status), one whose init
  record omits the server entirely (a finding), one passed a **nil** init record (a finding, and the
  case that proves the guard rather than a panic), and a control cell, which the caller-side
  predicate keeps out — assert the check's own behaviour for the granted cases and leave the control
  gating to the call site.
  **Make the fake emit a real server list, so the wiring is covered offline too.**
  `testdata/fakeclaude/main.go`'s `writeInit` hardcodes `"mcp_servers":[]` on every line it emits,
  which is demonstrably the wrong default for a rung-cell fake: the real init record this batch's
  fixture comes from carries both `mcp__quarry__toc` in `tools` and quarry in `mcp_servers` with
  status `connected`. `writeInit` already receives the granted tool list, so derive the server list
  from it — for each `mcp__<server>__<tool>` entry in `tools`, emit one `mcp_servers` object for
  `<server>` with status `connected`, deduplicated — and keep the empty array when the tool list has
  no prefixed entry, which is what a control cell must continue to produce. One source of truth, and
  every existing rung-cell case starts exercising the real shape without being rewritten.
  Add an environment override following the file's existing `FAKE_CLAUDE_*` convention that forces a
  named server to a status other than `connected`, and use it from `e2e_test.go` for one new case: a
  granted cell whose server never connects, asserting the repetition is invalidated rather than
  measured, that the reason file lands in the renamed attempt directory, and that exhausting the
  attempts aborts the invocation per the cross-repetition bound above.
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

**The wiring is covered offline, and that coverage is a late addition worth flagging.** An earlier
draft of this batch accepted a gap here: `testdata/fakeclaude/main.go` hardcoded `"mcp_servers":[]`,
so no fake-runner cell could present a non-connected server and card 5's `run.go` wiring was
reachable only from the live matrix. The orchestrator's real transcript settled it — a rung cell's
init record does carry the server, so the fake's empty array was simply the wrong default — and card
5 now derives the fake's server list from the granted tool list and adds an `e2e_test.go` case
driving the not-connected path end to end. The fake's control-cell behaviour is unchanged: no
prefixed tool, no server, empty array, exactly as before.
