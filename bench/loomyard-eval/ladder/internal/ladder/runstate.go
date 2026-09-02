// runstate.go ports the run-directory bookkeeping half of scripts/gates.py and the run/ingest driving
// logic of scripts/run_ladder.py: the run directory path, run.json's completeness definition, and the
// terminal-state marker write itself. run.json keeps its existing meaning untouched under the session
// split: it is still written last, in the scoring session, and is still the sole definition of a
// complete run. The ingest.json marker, invalidation, attempt bookkeeping, resume filtering, the
// single-flight predicate, and the aggregating gate report land in this same file across the batch's
// remaining cards, since the run/ingest/run-marker handoff is the seam this package exists to protect.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunDirPath returns <resultsRoot>/raw/<configID>/<n>/, the directory one run's artifacts live under.
func RunDirPath(resultsRoot, configID string, n int) string {
	return filepath.Join(resultsRoot, "raw", configID, fmt.Sprintf("%d", n))
}

// runJSONRecord is the shape read back off run.json by IsComplete: only the two fields IsComplete's own
// definition of completeness depends on. write_run_json's payload carries more fields than this, but
// IsComplete needs none of them.
type runJSONRecord struct {
	State string `json:"state"`
}

// IsComplete is true only when run.json exists in runDir and parses with state == "complete". A
// directory holding answer.json and usage.json but no score.json is by construction not complete,
// because run.json is written last -- see WriteRunJSON's doc comment. This definition is untouched by
// the session split: run session resume, scoring session resume, and matrix completeness are three
// different questions the ingest marker and PendingRuns/PendingScoring answer instead (see this file's
// later cards), but "is this run's artifact set the matrix's own definition of finished" is still this
// function and no other.
func IsComplete(runDir string) bool {
	data, err := os.ReadFile(filepath.Join(runDir, "run.json"))
	if err != nil {
		return false
	}
	var record runJSONRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return false
	}
	return record.State == "complete"
}

// WriteRunJSON writes the terminal-state marker to <runDir>/run.json: payload (assembled by the caller
// -- see RunJSONPayload) plus a stamped state: "complete" and a UTC timestamp. Called only after the
// answer parsed, usage.json was extracted, every fatal gate passed, and score.json exists -- and, under
// the session split, only from the scoring session, after gate_run_complete_artifacts's own artifact set
// (which includes ingest.json) is satisfied.
//
// Returns the full written record, so a caller does not have to re-read the file it just wrote.
func WriteRunJSON(runDir string, payload map[string]any) (map[string]any, error) {
	record := make(map[string]any, len(payload)+2)
	for key, value := range payload {
		record[key] = value
	}
	record["state"] = "complete"
	record["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("ladder: WriteRunJSON: mkdir %s: %w", runDir, err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("ladder: WriteRunJSON: marshal record: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(runDir, "run.json"), data, 0o644); err != nil {
		return nil, fmt.Errorf("ladder: WriteRunJSON: write run.json in %s: %w", runDir, err)
	}
	return record, nil
}

// IngestRecord is the ingest.json marker's shape: the config id, the repetition, the attempt index,
// and the gate report's non-fatal observations, as taken in the run session. Under the split session
// model this is the only path by which an observation taken while dispatching a run reaches the marker
// the summariser and the cold-cell disposition read (see RunJSONPayload) -- the scoring session that
// writes run.json never re-runs the gates that produced them.
type IngestRecord struct {
	// ConfigID is the LadderConfig this run belongs to.
	ConfigID string `json:"config_id"`
	// Rep is the 1-based repetition index.
	Rep int `json:"rep"`
	// Attempt is the 1-based attempt index this ingest reflects.
	Attempt int `json:"attempt"`
	// Observations holds the gate report's non-fatal findings taken during the run session.
	Observations []GateFinding `json:"observations"`
}

// NewIngestRecord assembles an IngestRecord from a gate report, carrying only its non-fatal findings --
// a report reaching this call is presumed already checked for Passed(), so its NonFatalFindings() is the
// whole of what is worth recording.
func NewIngestRecord(configID string, rep, attempt int, report GateReport) IngestRecord {
	return IngestRecord{
		ConfigID:     configID,
		Rep:          rep,
		Attempt:      attempt,
		Observations: report.NonFatalFindings(),
	}
}

// WriteIngestJSON serialises rec as indented JSON to <runDir>/ingest.json, with a trailing newline. This
// is the split session model's own marker: the run session writes it once dispatch, gating, and usage
// extraction all succeeded, ahead of (and independent from) the scoring session's later run.json write.
func WriteIngestJSON(runDir string, rec IngestRecord) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("ladder: WriteIngestJSON: mkdir %s: %w", runDir, err)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("ladder: WriteIngestJSON: marshal record: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(runDir, "ingest.json"), data, 0o644); err != nil {
		return fmt.Errorf("ladder: WriteIngestJSON: write ingest.json in %s: %w", runDir, err)
	}
	return nil
}

// HasIngest reports whether runDir carries an ingest.json marker.
func HasIngest(runDir string) bool {
	_, err := os.Stat(filepath.Join(runDir, "ingest.json"))
	return err == nil
}

// ReadIngestRecord reads and parses <runDir>/ingest.json.
func ReadIngestRecord(runDir string) (IngestRecord, error) {
	data, err := os.ReadFile(filepath.Join(runDir, "ingest.json"))
	if err != nil {
		return IngestRecord{}, fmt.Errorf("ladder: ReadIngestRecord: read ingest.json in %s: %w", runDir, err)
	}
	var rec IngestRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return IngestRecord{}, fmt.Errorf("ladder: ReadIngestRecord: unmarshal ingest.json in %s: %w", runDir, err)
	}
	return rec, nil
}

// namedLiftedObservationGates is the fixed set of non-fatal gate names RunJSONPayload lifts to a
// top-level boolean key of the same name, alongside the structured observations list. This is the
// suite's entire set of non-fatal findings, drawn from every gate that ever produces one:
// GateBlinding's target_origin_quarry_mention, ObserveWorktreeDirtied's worktree_dirtied, and
// GateColdAfter's cold_no_daemon_backed_call.
var namedLiftedObservationGates = map[string]bool{
	"worktree_dirtied":             true,
	"cold_no_daemon_backed_call":   true,
	"target_origin_quarry_mention": true,
}

// liftedObservationValue derives the boolean RunJSONPayload lifts a named observation to. worktree_dirtied
// is the one named gate whose finding fires unconditionally with either outcome, so its boolean is parsed
// out of the message ObserveWorktreeDirtied writes ("worktree dirtied: true"/"worktree dirtied: false");
// the other two named gates only ever produce a finding when the condition they observe held, so their
// mere presence in the observations list already means true.
func liftedObservationValue(finding GateFinding) bool {
	if finding.Gate == "worktree_dirtied" {
		return strings.HasSuffix(finding.Message, "true")
	}
	return true
}

// RunJSONPayload builds the payload WriteRunJSON writes into run.json from an assembled IngestRecord and
// the pinned run model.
//
// It emits the observations in both shapes, and this is a deliberate repair of a broken chain rather
// than a port: the Python wrote them only as a structured list of gate/message pairs, while its own
// summariser lifted them by looking for top-level keys named after each gate -- keys nothing in the
// Python ever wrote, so the lift never fired and every metric downstream of it (worktree_dirtied_count,
// target_origin_quarry_mention_count, daemon_backed_runs) was dead. RunJSONPayload therefore writes the
// structured "observations" list, which the cold-cell disposition reads by gate name, and lifts each of
// the three namedLiftedObservationGates to a top-level key of that same name carrying its boolean value,
// which is what the summariser reads. Emitting only the Python's shape would port a dead lift forward
// rather than fix it.
func RunJSONPayload(rec IngestRecord, runModel string) map[string]any {
	observations := make([]map[string]string, 0, len(rec.Observations))
	for _, finding := range rec.Observations {
		observations = append(observations, map[string]string{"gate": finding.Gate, "message": finding.Message})
	}

	payload := map[string]any{
		"config_id":    rec.ConfigID,
		"n":            rec.Rep,
		"model":        runModel,
		"observations": observations,
	}
	for _, finding := range rec.Observations {
		if namedLiftedObservationGates[finding.Gate] {
			payload[finding.Gate] = liftedObservationValue(finding)
		}
	}
	return payload
}

// MaxAttempts is the attempt cap every run and cold-cell repetition is retried against, ported from the
// Python's MAX_ATTEMPTS.
const MaxAttempts = 3

// invalidSiblingPath returns the path of runDir's k-th invalidated sibling: <runDir>.invalid-<k>.
func invalidSiblingPath(runDir string, k int) string {
	dir := filepath.Clean(runDir)
	return filepath.Join(filepath.Dir(dir), fmt.Sprintf("%s.invalid-%d", filepath.Base(dir), k))
}

// countInvalidSiblings counts runDir's existing <runDir>.invalid-<k> siblings, starting at k=1 and
// stopping at the first missing index -- this is NextAttempt's derivation site, and Invalidate's own
// search for the lowest unused index scans the same sequence.
func countInvalidSiblings(runDir string) int {
	count := 0
	for {
		if _, err := os.Stat(invalidSiblingPath(runDir, count+1)); err != nil {
			return count
		}
		count++
	}
}

// Invalidate renames runDir aside to the lowest unused <runDir>.invalid-<k> sibling -- taking
// ingest.json with it, since the whole directory moves, unlike the Python port's invalidate, which
// deleted run.json in place before the move; nothing here needs a comparable selective deletion, since
// no caller ever reads run.json or ingest.json out of an invalidated directory again.
//
// Returns the next attempt index (the sibling count after this rename, plus one). Errors, after still
// performing the rename, once that index would exceed MaxAttempts: the rename still lands the
// MaxAttempts-th invalid sibling on disk, which is what lets CheckSingleFlight recognise an exhausted
// attempt record purely from what NextAttempt already counts, and the error is what stops a caller from
// treating the (nonexistent) MaxAttempts+1-th attempt as valid.
func Invalidate(runDir string) (int, error) {
	dir := filepath.Clean(runDir)
	k := countInvalidSiblings(dir) + 1
	if k > MaxAttempts {
		return 0, fmt.Errorf("ladder: Invalidate: %s already carries %d invalid siblings, at MaxAttempts (%d)", dir, k-1, MaxAttempts)
	}

	candidate := invalidSiblingPath(dir, k)
	if err := os.Rename(dir, candidate); err != nil {
		return 0, fmt.Errorf("ladder: Invalidate: rename %s to %s: %w", dir, candidate, err)
	}
	if k >= MaxAttempts {
		return 0, fmt.Errorf("ladder: Invalidate: %s exhausted MaxAttempts (%d) after this invalidation", dir, MaxAttempts)
	}
	return k + 1, nil
}

// NextAttempt derives the current attempt index for (resultsRoot, configID, n) by counting existing
// <n>.invalid-<k> siblings and adding one, so the index a run correlation description embeds has one
// derivation site on disk rather than living in a session's memory across a resume.
func NextAttempt(resultsRoot, configID string, n int) (int, error) {
	dir := RunDirPath(resultsRoot, configID, n)
	count := countInvalidSiblings(dir)
	if count >= MaxAttempts {
		return 0, fmt.Errorf("ladder: NextAttempt: config %q rep %d has exhausted MaxAttempts (%d)", configID, n, MaxAttempts)
	}
	return count + 1, nil
}

// RunPair names one (config, repetition) cell of the matrix -- the run-state batch's own pair type,
// reused rather than duplicated by every later batch that enumerates or filters the matrix (plan_runs,
// PendingRuns/PendingScoring here, and the CLI drivers alike).
type RunPair struct {
	// Config is the LadderConfig this pair belongs to.
	Config LadderConfig
	// N is the 1-based repetition index.
	N int
}

// runJSONExists reports whether runDir carries a run.json file, independent of IsComplete's stricter
// state == "complete" check -- PendingScoring's own resume rule is "run.json absent", not "not yet
// complete", so it reads this rather than IsComplete's negation.
func runJSONExists(runDir string) bool {
	_, err := os.Stat(filepath.Join(runDir, "run.json"))
	return err == nil
}

// PendingRuns filters pairs to those a run session still has work to do: the absence of the ingest
// marker, rather than run.json's absence as the Python's own pending_runs read. Under the session split
// a run session's job ends once ingest.json exists; scoring, and run.json's write, are the scoring
// session's job (see PendingScoring), so filtering a run session's resume on run.json would leave it
// re-dispatching a run whose ingest already succeeded. Preserves pairs's own order -- callers are
// expected to pass an already config-then-repetition-ordered slice (see PlanRuns/MainRuns/ColdRuns).
func PendingRuns(resultsRoot string, pairs []RunPair) []RunPair {
	var pending []RunPair
	for _, pair := range pairs {
		dir := RunDirPath(resultsRoot, pair.Config.ID, pair.N)
		if !HasIngest(dir) {
			pending = append(pending, pair)
		}
	}
	return pending
}

// PendingScoring filters pairs to those a scoring session still has work to do: the ingest marker
// present and run.json absent. A pair with neither is a run session's own pending work (see
// PendingRuns); a pair with both is already complete. Preserves pairs's own order, matching PendingRuns.
func PendingScoring(resultsRoot string, pairs []RunPair) []RunPair {
	var pending []RunPair
	for _, pair := range pairs {
		dir := RunDirPath(resultsRoot, pair.Config.ID, pair.N)
		if HasIngest(dir) && !runJSONExists(dir) {
			pending = append(pending, pair)
		}
	}
	return pending
}

// CheckSingleFlight fails when repetition n of configID is being ingested while repetition n-1 of the
// same config has none of: an ingest marker, a run.json, or an exhausted attempt record -- concretely,
// MaxAttempts <n-1>.invalid-<k> sibling directories present, the same on-disk siblings NextAttempt
// counts and the only artifact recording exhaustion, since invalidation past the ceiling errors rather
// than writing a marker of its own. n == 1 always passes, since there is no n-1 to check.
//
// This predicate holds across sessions rather than merely within one: everything it reads is on disk, so
// a caller in a freshly started session sees the same answer a caller mid-session would have seen.
func CheckSingleFlight(resultsRoot, configID string, n int) error {
	if n <= 1 {
		return nil
	}
	prevDir := RunDirPath(resultsRoot, configID, n-1)
	if HasIngest(prevDir) {
		return nil
	}
	if runJSONExists(prevDir) {
		return nil
	}
	if countInvalidSiblings(prevDir) >= MaxAttempts {
		return nil
	}
	return fmt.Errorf(
		"ladder: CheckSingleFlight: config %q rep %d cannot start: rep %d has neither an ingest marker, "+
			"a run.json, nor an exhausted attempt record (%d invalid siblings)",
		configID, n, n-1, MaxAttempts,
	)
}

// RunGates composes the transcript gates and the filesystem/daemon-state gates into one GateReport, in
// the same order the Python's run_gates used, with two additions: GateMaxTurns, and a deliberate
// signature difference from the Python's run_gates(events, ladder, config, run_model, repo_root,
// worktree, run_dir, cache_dir, env) in three respects.
//
// It derives deniedNames through DenyListFor(l, c) rather than accepting a precomputed list, so the
// suite keeps one derivation site for them -- passing a precomputed list in would invite a second one,
// exactly what the single-source deny-list decision exists to prevent. It drops the Python's a_run_dir
// parameter rather than carrying it forward for symmetry: the Python's own run_gates never inspected it
// either (see its docstring), so threading an unused parameter through the port would only carry the
// dead weight forward. And it takes dirtied as an input rather than computing it internally, so the
// caller can take that observation (via ObserveWorktreeDirtied) before restoring the worktree -- the
// restore is precisely what erases the evidence a call made from inside RunGates could no longer see.
//
// Applies GateBlinding only when c.Allowed is empty, and GateColdAfter only when c.Cold is true --
// GateColdBefore is a separate precondition the caller checks before starting an attempt, not part of
// this composed report.
func RunGates(records []Record, l *Ladder, c LadderConfig, runModel, repoRoot, worktree string, maxTurns int, dirtied GateFinding, cacheDir string, env []string, taskText string) GateReport {
	deniedNames := DenyListFor(l, c)

	var findings []GateFinding
	findings = append(findings, GateRunPrompt(records, taskText)...)
	findings = append(findings, GateDeniedToolsNotUsed(records, deniedNames)...)
	findings = append(findings, GateNoTargetOverride(records)...)
	findings = append(findings, GateModelPinned(records, runModel)...)
	findings = append(findings, GateMaxTurns(records, maxTurns)...)
	if len(c.Allowed) == 0 {
		findings = append(findings, GateBlinding(records, repoRoot)...)
	}
	findings = append(findings, GateWorktreeNeutralised(worktree)...)
	findings = append(findings, dirtied)
	if c.Cold {
		findings = append(findings, GateColdAfter(records, worktree, cacheDir, env)...)
	}

	return GateReport{Findings: findings}
}

// runCompleteArtifactNames is the fixed set of files GateRunCompleteArtifacts requires, in the order it
// checks them. All seven are unconditional: the copied launch inputs (settings.json, and mcp.json when
// the config's allowed set is non-empty) are deliberately excluded, because a declared server named
// "quarry" among them exists only for a config whose allowed set is non-empty, and this gate's own
// signature carries no config to make that per-config distinction -- requiring mcp.json unconditionally
// would fail every blinded "none" control run, which never writes one by design (see
// write_run_inputs/WriteRunInputs).
var runCompleteArtifactNames = []string{
	"answer.json",
	"answer.redacted.json",
	"usage.json",
	"score.json",
	"ingest.json",
	"transcript.jsonl",
	"transcript.meta.json",
}

// GateRunCompleteArtifacts is a separate, later gate requiring all seven of runCompleteArtifactNames --
// updated from the Python's four to the new results layout's own complete-artifact set, which adds
// ingest.json (the session split's own run-session marker), transcript.jsonl, and transcript.meta.json.
//
// Deliberately not part of RunGates: two of its required files are written by the scorer, which runs
// after RunGates, so folding it in would make every run fail a fatal gate before the scorer had a chance
// to write them. Like the Python's gate_run_complete_artifacts, this stays a gate the caller invokes
// separately after scoring and immediately before the run marker is written.
func GateRunCompleteArtifacts(runDir string) []GateFinding {
	var findings []GateFinding
	for _, name := range runCompleteArtifactNames {
		if _, err := os.Stat(filepath.Join(runDir, name)); err != nil {
			findings = append(findings, GateFinding{
				Gate:    "run_complete_artifacts",
				Fatal:   true,
				Message: fmt.Sprintf("%s missing from %s", name, runDir),
			})
		}
	}
	return findings
}
