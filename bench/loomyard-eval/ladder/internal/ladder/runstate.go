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
