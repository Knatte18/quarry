// coldcell.go ports the disposition half of scripts/run_ladder.py's run_cold_cell: reading the cold
// cell's own completed runs and invalidated attempts off disk and reducing them to the same four
// dispositions the Python produced -- confirmed-cold, partial, not-run, and no-daemon-signal. The
// dispatch loop that drove those attempts has no counterpart here: dispatch happens in a session (see
// the plan's own Shared Decision), so this file only ever reads what a run or cold-session-preparation
// abort already left on disk.
//
// The Python tracked a not-run repetition's live-daemon cause in the driver's own in-process memory,
// which no longer exists under the session split: that cause now arises inside a session-preparation
// abort running in a different process entirely (the ladderbench prepare-session cold path). It is
// therefore read back from the cold_abort.json record that abort leaves inside the repetition's
// <n>.invalid-<k> sibling directory, matching on that file's own cause key, rather than carried forward
// in memory the way the Python held it.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// coldAbortFilename is cold_abort.json's fixed name, mirroring the ladderbench prepare-session cold
// path's own unexported constant of the same name -- that command lives in a different package and
// cannot be referenced directly, so this is coldcell.go's own copy of the same literal.
const coldAbortFilename = "cold_abort.json"

// Not-run cause tokens ColdCellDisposition records under a ColdCellRecord's NotRunCauses, mirroring the
// Python's own rep_not_run_cause literals.
const (
	// coldAbortCauseLiveDaemon names a repetition whose every attempt was discarded by a
	// session-preparation abort because a daemon was already alive before the attempt even started.
	coldAbortCauseLiveDaemon = "live_daemon_before_start"
	// coldAbortCauseNativeFallbackExhausted names a repetition that exhausted MaxAttempts on the
	// native-fallback branch: every attempt completed the transcript but the cold-after gate found no
	// daemon.json, so ingest failed and each attempt was invalidated in turn.
	coldAbortCauseNativeFallbackExhausted = "native_fallback_exhausted"
)

// coldAbortRecord is cold_abort.json's own shape, read here only for its cause field -- the run-session
// preparation path (ladderbench prepare-session) is this file's sole writer.
type coldAbortRecord struct {
	Cause string `json:"cause"`
}

// completedRunObservations is the subset of run.json ColdCellDisposition reads: its own observations
// list, keyed only by gate name here -- the same shape RunJSONPayload writes (runstate.go).
type completedRunObservations struct {
	Observations []struct {
		Gate string `json:"gate"`
	} `json:"observations"`
}

// completedRepDisposition reads a completed cold-cell repetition's run.json and returns "cold" when its
// gate observations confirm a supervised connection, or "no_daemon_signal" when it recorded
// cold_no_daemon_backed_call -- ports _rep_disposition_from_run_json's own gate-name match.
func completedRepDisposition(runDir string) (string, error) {
	path := filepath.Join(runDir, "run.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ladder: cold cell disposition: read %s: %w", path, err)
	}
	var record completedRunObservations
	if err := json.Unmarshal(data, &record); err != nil {
		return "", fmt.Errorf("ladder: cold cell disposition: parse %s: %w", path, err)
	}
	for _, observation := range record.Observations {
		if observation.Gate == "cold_no_daemon_backed_call" {
			return "no_daemon_signal", nil
		}
	}
	return "cold", nil
}

// notRunCauseFromInvalidSiblings scans runDir's <n>.invalid-<k> siblings, k from 1 to MaxAttempts, for a
// cold_abort.json recording coldAbortCauseLiveDaemon. Finding one anywhere in the sequence means a
// session-preparation abort discarded at least one attempt before it ever ran; finding none means every
// invalidated attempt instead ran to completion and failed the cold-after gate on the native-fallback
// branch, which leaves no cold_abort.json of its own -- so the absence of the marker is itself the
// second cause's own positive signal, not merely a default.
func notRunCauseFromInvalidSiblings(runDir string) (string, error) {
	for k := 1; k <= MaxAttempts; k++ {
		path := filepath.Join(invalidSiblingPath(runDir, k), coldAbortFilename)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("ladder: cold cell disposition: read %s: %w", path, err)
		}
		var record coldAbortRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return "", fmt.Errorf("ladder: cold cell disposition: parse %s: %w", path, err)
		}
		if record.Cause == coldAbortCauseLiveDaemon {
			return coldAbortCauseLiveDaemon, nil
		}
	}
	return coldAbortCauseNativeFallbackExhausted, nil
}

// repDisposition returns configID's n-th repetition's own disposition status ("cold", "no_daemon_signal",
// or "not_run") and, only for "not_run", its cause -- purely from what is already on disk, since the
// dispatch loop that produced it has no counterpart in this file.
func repDisposition(resultsRoot, configID string, n int) (status, cause string, err error) {
	runDir := RunDirPath(resultsRoot, configID, n)
	if IsComplete(runDir) {
		status, err := completedRepDisposition(runDir)
		return status, "", err
	}
	cause, err = notRunCauseFromInvalidSiblings(runDir)
	if err != nil {
		return "", "", err
	}
	return "not_run", cause, nil
}

// notRunReasonText describes why the not-run repetitions never confirmed cold, naming both causes only
// when both actually occurred -- mirroring run_cold_cell's own _not_run_reps_reason, kept as a
// separately tracked pair so the reason text stays accurate rather than reporting both alike.
func notRunReasonText(liveDaemon, nativeFallback int) string {
	switch {
	case liveDaemon == 0:
		return fmt.Sprintf("%d exhausted attempts on the native-fallback branch", nativeFallback)
	case nativeFallback == 0:
		return fmt.Sprintf("%d found a live daemon before every attempt started", liveDaemon)
	default:
		return fmt.Sprintf(
			"%d exhausted attempts on the native-fallback branch, %d found a live daemon before every attempt started",
			nativeFallback, liveDaemon,
		)
	}
}

// ColdCellRecord is cold_cell.json's shape: the overall disposition and its human-readable reason, the
// count of confirmed-cold repetitions out of the total, and the per-repetition status and not-run-cause
// breakdowns -- ported field for field from run_cold_cell's own written record.
type ColdCellRecord struct {
	// Disposition is one of "confirmed-cold", "partial", "not-run", or "no-daemon-signal".
	Disposition string `json:"disposition"`
	// Reason is the human-readable explanation for Disposition.
	Reason string `json:"reason"`
	// ConfirmedColdReps is the count of repetitions whose own status is "cold".
	ConfirmedColdReps int `json:"confirmed_cold_reps"`
	// Reps is the cold config's total repetition count.
	Reps int `json:"reps"`
	// PerRepetition maps each repetition's 1-based index (as a decimal string, matching Python's own
	// int-keyed dict serialising to string JSON object keys) to its own status: "cold",
	// "no_daemon_signal", or "not_run".
	PerRepetition map[string]string `json:"per_repetition"`
	// NotRunCauses maps a "not_run" repetition's index (as a decimal string) to its cause:
	// coldAbortCauseLiveDaemon or coldAbortCauseNativeFallbackExhausted.
	NotRunCauses map[string]string `json:"not_run_causes"`
}

// ColdCellDisposition builds l's cold cell's own ColdCellRecord by reading, for each of its Reps
// repetitions, whatever a prior run or cold-session-preparation abort already left on disk -- ported from
// the disposition half of run_cold_cell. It does not port that function's dispatch loop: dispatch happens
// in a session, never here.
//
// The four dispositions match run_cold_cell's own rules exactly: "confirmed-cold" when at least one
// repetition is cold and none are not-run; "partial" when at least one of each; "not-run" when every
// repetition is not-run; and "no-daemon-signal" otherwise -- every repetition completed validly, none
// confirmed cold, and none is not-run, meaning every one recorded no_daemon_signal.
func ColdCellDisposition(l *Ladder, resultsRoot string) (ColdCellRecord, error) {
	var coldConfig LadderConfig
	found := false
	for _, config := range l.Configs {
		if config.Cold {
			coldConfig = config
			found = true
			break
		}
	}
	if !found {
		return ColdCellRecord{}, fmt.Errorf("ladder: cold cell disposition: no cold config found in ladder")
	}

	perRepetition := make(map[string]string, l.Reps)
	notRunCauses := make(map[string]string)
	confirmedCold := 0
	notRun := 0
	notRunLiveDaemon := 0
	notRunNativeFallback := 0

	for n := 1; n <= l.Reps; n++ {
		status, cause, err := repDisposition(resultsRoot, coldConfig.ID, n)
		if err != nil {
			return ColdCellRecord{}, err
		}
		key := strconv.Itoa(n)
		perRepetition[key] = status

		switch status {
		case "cold":
			confirmedCold++
		case "not_run":
			notRun++
			notRunCauses[key] = cause
			if cause == coldAbortCauseLiveDaemon {
				notRunLiveDaemon++
			} else {
				notRunNativeFallback++
			}
		}
	}

	totalReps := l.Reps
	var disposition, reason string
	switch {
	case confirmedCold >= 1 && notRun == 0:
		disposition = "confirmed-cold"
		reason = fmt.Sprintf("%d of %d repetitions confirmed cold", confirmedCold, totalReps)
	case confirmedCold >= 1 && notRun >= 1:
		disposition = "partial"
		reason = fmt.Sprintf(
			"%d of %d repetitions confirmed cold; %d did not: %s",
			confirmedCold, totalReps, notRun, notRunReasonText(notRunLiveDaemon, notRunNativeFallback),
		)
	case confirmedCold == 0 && notRun == totalReps:
		disposition = "not-run"
		if notRunLiveDaemon == 0 {
			reason = "the supervised daemon strategy is unavailable on this machine -- every repetition exhausted its attempts on the native-fallback branch"
		} else {
			reason = fmt.Sprintf("no repetition confirmed cold: %s", notRunReasonText(notRunLiveDaemon, notRunNativeFallback))
		}
	default:
		disposition = "no-daemon-signal"
		reason = "every repetition completed validly but none invoked a daemon-backed tool"
	}

	return ColdCellRecord{
		Disposition:       disposition,
		Reason:            reason,
		ConfirmedColdReps: confirmedCold,
		Reps:              totalReps,
		PerRepetition:     perRepetition,
		NotRunCauses:      notRunCauses,
	}, nil
}

// WriteColdCellRecord serialises record as indented JSON with a trailing newline to
// <resultsRoot>/cold_cell.json.
func WriteColdCellRecord(resultsRoot string, record ColdCellRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("ladder: write cold cell record: marshal: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(resultsRoot, "cold_cell.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("ladder: write cold cell record: write %s: %w", path, err)
	}
	return nil
}
