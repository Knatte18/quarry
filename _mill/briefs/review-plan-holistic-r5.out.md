MILL_REVIEW_BEGIN
# Review: Ladder, toc rerun (T7) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (self-assessed; no independent means to verify the build)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] Server-connect invalidation is unbounded across reps
**Location:** batch 2 card 5 (wiring), batch 3 card 7 (termination rule)
**Issue:** Routing a non-connected server into `InvalidateRep` bounds retries per repetition at `MaxAttempts` = 3 but not across the cell's 5 repetitions, so a *systematic* connect failure — the defect class this batch exists for, whose `run.go` wiring the batch admits is first exercised by the matrix itself — spends up to 15 measured `claude -p` calls inside one invocation, against a whole-task budget of 10, and only becomes visible after the invocation returns.
**Fix:** State a cross-repetition stop for consecutive server-connect invalidations — either the `repOutcome.abortRun` disposition `run.go` already carries for a tainted memory-path scan, or a card-7 driver rule that kills the background invocation after N consecutive `.invalid-*` renames on the same cell.

### [BLOCKING:design] No disposition for a transcript with no session-init record
**Location:** batch 2 card 5
**Issue:** `CheckServerConnected` is specified as taking "the decoded `SessionInit`", and its call site is inside the attempt loop, where `transcript.Init` is nil for a transcript that carries no init line — `run.go` nil-guards `transcript.Init` at both of its existing uses, and the attempt loop's accept branch only tests `t.Result`. The card's `gates_test.go` case list (connected / other status / server absent / control) has no nil-init case, so the specified behaviour is undefined and the natural implementation dereferences nil.
**Fix:** State whether an absent init record is a finding or a pass, and add that case to the card's test list.

### [NIT:scope] New fingerprint status map and diff clause get no test
**Location:** batch 2 card 3 (with batch 2 `## Batch Tests`)
**Issue:** Card 3 adds `SessionFingerprint.MCPServerStatuses` and a new status clause in `diffSessionFingerprint`, but no card adds a case for either; `provenance_test.go`'s `TestCompareFingerprints` builds `SessionFingerprint` from named-field literals, so it stays green while exercising neither. The batch's stated gate (card 4) covers only `ParseTranscript` decoding.
**Fix:** Give card 3 or card 4 a case asserting `NewSessionFingerprint` populates the map from the retyped field and that `diffSessionFingerprint` reports a status-only difference.

### [NIT:consistency] New fixture sits outside the package's testdata layout
**Location:** batch 2 card 4 `Creates:`; overview `## All Files Touched`
**Issue:** Every transcript fixture in the package lives under `testdata/transcripts/`, and `metrics_test.go`'s shared `parseTranscriptFixture` helper hardcodes that prefix; the card creates `testdata/session-init-mcp-connected.jsonl` at the `testdata/` root, so the new case cannot use the existing helper and the fixture directory stops being the one place fixtures live.
**Fix:** Put the fixture under `testdata/transcripts/` in both the card and `## All Files Touched`.

### [NIT:consistency] `the-code-under-test-is-not-edited` forbids what batch 2 does
**Location:** overview `## Shared Decisions`, `the-code-under-test-is-not-edited`
**Issue:** The decision reads "no card in this plan edits `internal/`, `quarry/`, `cmd/`", but batch 2's cards edit `bench/loomyard-eval/ladder/internal/ladder/{stream,provenance,gates,run,runstate}.go` and card 7 drives `bench/loomyard-eval/ladder/cmd/ladder`; only the rationale ("those are the code under test") disambiguates, and the card an implementer reads is not the rationale.
**Fix:** Qualify the three paths as repository-root `internal/`, `quarry/`, `cmd/`, excluding the harness tree under `bench/loomyard-eval/ladder/`.

## Verdict

REQUEST_CHANGES
Card 5's retry budget and nil-init disposition need stating; three smaller consistency and coverage gaps.
MILL_REVIEW_END
