# Batch: cli-run-commands

```yaml
task: Port the capability-ladder bench harness to Go
batch: cli-run-commands
number: 12
cards: 9
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [7, 11]
```

## Batch Scope

Completes the subcommand surface: transcript correlation and custody, then `ingest`, `invalidate`,
`redact`, `record-score`, `probe-record`, `cold-cell`, and `summarize`. After this batch the whole
protocol is expressible as a sequence of `ladderbench` calls from a live session, with the session
supplying only the dispatch.

Batch-local decision: correlation matches on a unique per-call description rather than on any agent-id
field. The description form is fixed here as the config id, the repetition, and the attempt index, so
retries cannot collide, and zero matches or more than one match is a hard error rather than a fallback —
a newest-mtime guess would silently pick the wrong transcript under any concurrent dispatch.

## Cards

### Card 57: Transcript correlation and custody

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/correlate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/correlate_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `DispatchDescription(configID string, rep, attempt int) string` producing the
  unique per-call description, and
  `LocateTranscript(projectsRoot, sessionDir, description string, wait time.Duration) (transcriptPath, metaPath string, err error)`.
  The search space is fixed here and no card may widen it: the projects root defaults to
  `~/.claude/projects`, the project directory is the session's working-directory path with every path
  separator replaced by a hyphen, and the glob is
  `<projects-root>/<mangled-cwd>/*/subagents/*.meta.json` — one wildcard segment for the session id,
  then the fixed subagents directory. The transcript is the `agent-<id>.jsonl` sibling of the matched
  `agent-<id>.meta.json`. Select the metadata file whose recorded description matches exactly. The
  session-id wildcard deliberately spans every session ever launched from that scratch directory:
  because each run session has its own scratch directory and the description embeds the config, the
  repetition, and the attempt index, a second match means two dispatches genuinely shared one
  description, which is exactly the collision the multiple-match error exists to catch. Zero matches errors and more than one match errors; neither ever
  falls back. Exactly one match whose sibling transcript is absent is its own case: re-check on a short
  bounded wait for a not-yet-flushed transcript, then hard-error naming the matched metadata path and
  the description. Add `CopyTranscriptCustody(transcriptPath, metaPath, runDir string) error` copying
  both files into the run directory as the run's transcript and its metadata, so the results tree stays
  self-contained and survives session-transcript pruning, and the metadata copy remains the evidence
  that the match picked the right file. Test the project-directory derivation, a single match, zero
  matches, two matches, a match whose transcript appears within the wait, a match whose transcript never
  appears, and that custody copies both files.
- **Commit:** `feat(ladder): correlate and take custody of a run's subagent transcript`

### Card 58: ingest

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/correlate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/ingest.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/ingest_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `ingest` subcommand performing, in this order: enforce the single-flight
  predicate; build this attempt's correlation description with `DispatchDescription`, taking the
  attempt index from `NextAttempt` — the same derivation `next-run` uses, so the two commands provably
  build the same string; locate the transcript by that description and copy it and its metadata into the run directory;
  copy the session's launch inputs — its settings document, its run agent definition, and its server
  declaration when the config has one — into the run directory, at parity with what the Python wrote
  per run; extract usage and serialise it to `<run_dir>/usage.json`; parse the answer from the last fenced
  block of the final assistant record, taking `ExtractFencedJSON`'s `inner` half, and write it to
  `<run_dir>/answer.json`. Usage extraction takes the granted-tool list produced by
  `GrantedToolsFromDefinition` over the copied agent definition — that reader is its only source, so
  the recorded list always comes from a harness-generated file rather than from anything the agent
  said. Reading the transcript uses `ReadTranscript`, and the dirtiness observation uses
  `ObserveWorktreeDirtied`, whose result is what `RunGates` takes as its injected observation. These two writes are named explicitly because `ingest` is their only write site anywhere
  in the plan, and both the complete-artifacts gate and the redaction step require the files by name;
  then take the worktree dirtiness observation;
  run the gates; assemble the ingest record with `NewIngestRecord` and write it on success; and print the outcome as ingested, truncated, or
  failed. `ingest` enforces the full pin set before running the gates, which is what makes the turn
  ceiling readable — the ceiling value ships blank and the gate would otherwise compare against nothing.
  The dirtiness observation is taken before anything restores the worktree, because the restore
  is what erases the evidence. `ingest` never destroys evidence on failure — invalidation is a separate
  command — and a truncated outcome is never retried. Test the ordering constraint that the observation
  precedes the marker write, the three outcomes, and that a single-flight violation errors before any
  file is copied.
- **Commit:** `feat(ladderbench): add ingest`

### Card 59: invalidate

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/invalidate.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/invalidate_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `invalidate` subcommand renaming a failed run directory aside and printing
  the next attempt index, erroring when the attempt ceiling is already reached — which is the matrix
  halt. Its help text must state that it is deliberately separate from `ingest`, because `ingest` has
  to be able to report a failure without destroying the evidence of it. Test the printed index and the
  ceiling error.
- **Commit:** `feat(ladderbench): add invalidate`

### Card 60: redact

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/task.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/redact.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/redact_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `redact` subcommand writing the redacted answer beside the original and
  printing the assembled scorer prompt on standard output for the session to dispatch. Its help text
  must state that the printed prompt embeds the task's unstripped answer key, which is why it is only
  ever run inside the dedicated scoring session and never in a session that also hosts a run agent.
  `redact` enforces the full pin set. Test that the redacted file is written, that the original is left
  byte-identical, and that the printed prompt is the assembled scorer prompt.
- **Commit:** `feat(ladderbench): add redact`

### Card 61: record-score

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/recordscore.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/recordscore_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `record-score` subcommand consuming a scorer reply, validating it, stamping
  the pinned scorer model and effort into the score record, writing it, running the complete-artifacts
  gate, and writing `run.json` last. It assembles the run marker's payload by reading the run
  directory's ingest record with `ReadIngestRecord` and passing it to `RunJSONPayload`, rather than
  building the mapping inline. This is the only path by which an observation taken in the run session
  reaches the marker the summariser and the cold-cell disposition both read, and keeping the assembly in
  the library is what lets the synthetic end-to-end test cover that handoff — without it every
  observation would be stranded in the run session. Its help text must state that `run.json` is written last and
  remains the sole definition of a complete run. `record-score` enforces the full pin set. Test that an
  invalid reply is rejected before anything is written, that the artifacts gate failing prevents the run
  marker, and that a successful run leaves the directory complete.
- **Commit:** `feat(ladderbench): add record-score`

### Card 62: probe-record

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/correlate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/proberecord.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/proberecord_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `probe-record` subcommand consuming one probe dispatch's transcript and
  writing or extending `probe.json` at the results root with the boolean keys `allowlist_blocks` and
  `denylist_blocks`, halting on either being false. Each probe kind writes only its own key, so a second
  invocation extends the existing document rather than replacing it. The deny-list probe additionally
  captures the verbatim text of the errored tool result it provoked into the same document under
  `denial_shape_observed` — its help text must state that this is what the follow-up task checks the
  provisional denial pattern against before clearing the provisional marker. Test writing each probe's
  half, extending an existing record rather than overwriting the other half, halting on a false layer,
  and capturing the observed shape verbatim.
- **Commit:** `feat(ladderbench): add probe-record`

### Card 63: Cold-cell disposition logic

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession.go`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/coldcell.go`
  - `bench/loomyard-eval/ladder/internal/ladder/coldcell_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_rep_disposition_from_run_json` as an unexported per-repetition disposition
  helper and the disposition half of `run_cold_cell` as
  `ColdCellDisposition(l *Ladder, resultsRoot string) (ColdCellRecord, error)`, producing the same four
  outcomes the Python produces — confirmed-cold, partial, not-run, and no-daemon-signal — with the same
  not-run cause distinction between a live daemon before start and an exhausted native fallback. The
  Python tracked the live-daemon cause in the driver's own memory, which no longer exists: under the
  session split that cause arises in a session-preparation abort in a different process. It is
  therefore read from the `cold_abort.json` records the cold preparation path leaves inside the
  `<n>.invalid-<k>` sibling directories, matching on that file's `cause` key, and the
  exhausted-fallback cause from the completed runs' observations. Both causes occurring for one cell
  must still produce a reason text naming both. Do not
  port the dispatch loop: dispatch happens in a session. Test all four outcomes and the case where both
  not-run causes occur, where the reason text must name both.
- **Commit:** `feat(ladder): port the cold-cell disposition logic`

### Card 64: cold-cell

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/coldcell.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/coldcell.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/coldcell_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `cold-cell` subcommand. With `--teardown --rep`, it removes that
  repetition's disposable worktree unconditionally whatever the run's outcome and waits for that
  worktree's daemon to exit. With no flags it finalises and writes the cold-cell record at the results
  root. `cold-cell` enforces the full pin set. Test that teardown removes the worktree even after a
  failed run, and that the bare form writes the record.
- **Commit:** `feat(ladderbench): add cold-cell`

### Card 65: summarize

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/summarize.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/summarize_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `summarize` subcommand building and writing the summary at the results root
  and exiting non-zero when any cell is incomplete, naming the incomplete cells on standard error.
  `summarize` enforces the full pin set. Test the zero-exit complete path and the non-zero incomplete
  path.
- **Commit:** `feat(ladderbench): add summarize`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `correlate_test.go`, `coldcell_test.go`, and
every new command-package test file, plus every earlier test file in the ladder subtree. Correlation
tests build synthetic subagent metadata trees in the test's temp directory, which is the only way to
exercise the zero-match, two-match, and missing-sibling cases deterministically.
