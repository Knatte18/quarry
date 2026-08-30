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
