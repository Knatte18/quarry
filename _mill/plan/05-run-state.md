# Batch: run-state

```yaml
task: Port the capability-ladder bench harness to Go
batch: run-state
number: 5
cards: 5
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [4]
```

## Batch Scope

Ports the run-directory bookkeeping and the two git-dependent gates, and adds the new `ingest.json`
marker the split session model needs. It finishes with the aggregating `RunGates` and the
complete-artifacts gate, so after this batch a run directory's full lifecycle — dispatched-and-gated,
then complete — is expressible without any session machinery.

The external interface later batches consume is `RunGit`, `GateWorktreeNeutralised`,
`ObserveWorktreeDirtied`, `RunDirPath`, `IsComplete`, `Invalidate`, `NextAttempt`, `WriteRunJSON`,
`WriteIngestJSON`, `HasIngest`, `PendingRuns`, `PendingScoring`, `CheckSingleFlight`, `RunGates`, and
`GateRunCompleteArtifacts`.

Batch-local decision: `run.json` keeps its existing meaning untouched — it is still written last and is
still the sole definition of a complete run. `ingest.json` is a strictly additional marker, because run
session resume, scoring session resume, and matrix completeness are three different questions and
collapsing them onto one marker is what breaks the single-flight predicate once scoring moves out of
the run session.

## Cards

### Card 22: Git helper and the two worktree gates

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `run_git` as `RunGit(args ...string) (string, error)`, keeping the Python's
  error-on-non-zero behaviour and capturing combined output into the error. Port
  `gate_worktree_neutralised` as `GateWorktreeNeutralised(worktree string) GateFinding` and
  `observe_worktree_dirtied` as `ObserveWorktreeDirtied(worktree string) GateFinding`, preserving each
  one's fatality and message shape. `ObserveWorktreeDirtied`'s doc comment must record that it has to
  run before the worktree restore, because the restore is precisely what erases the evidence. Test both
  gates against temporary git repositories created inside the test's own temp directory.
- **Commit:** `feat(ladder): port the git helper and the two worktree gates`

### Card 23: Run directory paths, completeness, and run.json

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `run_dir` as `RunDirPath(resultsRoot, configID string, n int) string`,
  `is_complete` as `IsComplete(runDir string) bool`, and `write_run_json` as `WriteRunJSON`.
  `IsComplete` keys on `run.json` alone and on its recorded state being complete — that definition is
  untouched by the session split, and the doc comment must say so explicitly so a later reader does not
  retarget it at the new marker. Test `IsComplete` false with no run marker, false when the recorded
  state is not complete, and true otherwise, and that `WriteRunJSON` round-trips through
  `IsComplete`.
- **Commit:** `feat(ladder): port run-directory paths, completeness, and run.json`

### Card 24: The ingest marker, invalidation, and attempt bookkeeping

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `WriteIngestJSON(runDir string, rec IngestRecord) error` and
  `HasIngest(runDir string) bool` for a new `ingest.json` marker whose `IngestRecord` carries the config
  id, the repetition, the attempt index, and the gate report's non-fatal observations. Port
  `MAX_ATTEMPTS` as an exported `MaxAttempts` constant holding 3, and `invalidate` as
  `Invalidate(runDir string) (int, error)`, which renames the run directory aside to the lowest unused
  `<n>.invalid-<k>` sibling — taking `ingest.json` with it, since the whole directory moves — and
  returns the next attempt index, erroring once `MaxAttempts` is exhausted. Add
  `NextAttempt(resultsRoot, configID string, n int) (int, error)` deriving the current attempt index by
  counting existing `<n>.invalid-<k>` siblings, so the index the correlation description embeds has one
  derivation site on disk rather than living in a session's memory across a resume. Test the marker
  round-trip, invalidation picking the lowest unused index, invalidation erroring at the ceiling, and
  the attempt index being 1 with no invalid siblings and `k+1` with `k` of them.
- **Commit:** `feat(ladder): add the ingest marker, invalidation, and attempt bookkeeping`

### Card 25: Resume filtering and the single-flight predicate

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `pending_runs` as `PendingRuns(pairs []RunPair, resultsRoot string) []RunPair`,
  filtering a run session's work on the absence of the ingest marker rather than on `run.json`, where
  `RunPair` names a config id and a repetition. Add
  `PendingScoring(resultsRoot string, pairs []RunPair) []RunPair` filtering on "ingest marker present,
  `run.json` absent", in config-then-repetition order. Add
  `CheckSingleFlight(resultsRoot, configID string, n int) error`, the predicate that fails when
  repetition `n` is being ingested while repetition `n-1` of the same config has none of an ingest
  marker, a `run.json`, or an exhausted attempt record. Its doc comment must state that the predicate
  holds across sessions rather than merely within one, because the marker it reads is on disk. Test run
  session resume treating an ingested repetition as done, scoring resume treating ingested-but-unscored
  as pending and both-present as done, and the single-flight predicate erroring for an out-of-order
  repetition and passing once any of the three conditions is met.
- **Commit:** `feat(ladder): add resume filtering and the single-flight predicate`

### Card 26: The aggregating gate runner and the complete-artifacts gate

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `run_gates` as `RunGates` with the same aggregation order the Python uses,
  extended to include `GateMaxTurns` and to take the dirtied observation as an input rather than
  computing it, so the caller can take that observation before restoring the worktree. Port
  `gate_run_complete_artifacts` as `GateRunCompleteArtifacts(runDir string) GateFinding`, updated to
  the artifact set the new results layout defines. Test the aggregate over a passing transcript and
  over a transcript that trips one fatal and one non-fatal gate, and the artifacts gate with a complete
  and an incomplete run directory.
- **Commit:** `feat(ladder): port the aggregating gate runner and artifacts gate`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `worktree_test.go` and `runstate_test.go`
plus every earlier test file in the ladder subtree. The git-dependent tests build throwaway
repositories under the test's temp directory rather than touching any real worktree.
