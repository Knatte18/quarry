package ladder

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// coldConfigIDFixture is the committed ladder.yaml's own single cold config id.
const coldConfigIDFixture = "a5-bundle-cold"

// writeColdCompletedRun writes a complete run directory for the cold config's n-th repetition, recording
// a cold_no_daemon_backed_call observation when noDaemonSignal is true.
func writeColdCompletedRun(t *testing.T, resultsRoot string, n int, noDaemonSignal bool) {
	t.Helper()
	runDir := RunDirPath(resultsRoot, coldConfigIDFixture, n)
	var observations []map[string]string
	if noDaemonSignal {
		observations = []map[string]string{{"gate": "cold_no_daemon_backed_call", "message": "no daemon-backed tool call observed"}}
	}
	if _, err := WriteRunJSON(runDir, map[string]any{"observations": observations}); err != nil {
		t.Fatalf("WriteRunJSON(rep %d): %v", n, err)
	}
}

// writeColdNotRunRepetition writes runDir's siblings so it never completed: MaxAttempts invalid
// siblings, with a cold_abort.json recording coldAbortCauseLiveDaemon in the k-th sibling for every k in
// liveDaemonAttempts (1-based), and no cold_abort.json in the rest.
func writeColdNotRunRepetition(t *testing.T, resultsRoot string, n int, liveDaemonAttempts ...int) {
	t.Helper()
	runDir := RunDirPath(resultsRoot, coldConfigIDFixture, n)
	isLiveDaemon := make(map[int]bool, len(liveDaemonAttempts))
	for _, k := range liveDaemonAttempts {
		isLiveDaemon[k] = true
	}
	for k := 1; k <= MaxAttempts; k++ {
		siblingDir := invalidSiblingPath(runDir, k)
		if err := os.MkdirAll(siblingDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", siblingDir, err)
		}
		if isLiveDaemon[k] {
			data := []byte(`{"config_id":"` + coldConfigIDFixture + `","cause":"` + coldAbortCauseLiveDaemon + `"}`)
			if err := os.WriteFile(filepath.Join(siblingDir, coldAbortFilename), data, 0o644); err != nil {
				t.Fatalf("write cold_abort.json in %s: %v", siblingDir, err)
			}
		}
	}
}

func TestColdCellDisposition_ConfirmedCold(t *testing.T) {
	l := mustLoadLadder(t)
	resultsRoot := t.TempDir()

	for n := 1; n <= l.Reps; n++ {
		writeColdCompletedRun(t, resultsRoot, n, false)
	}

	record, err := ColdCellDisposition(l, resultsRoot)
	if err != nil {
		t.Fatalf("ColdCellDisposition() error = %v; want nil", err)
	}
	if record.Disposition != "confirmed-cold" {
		t.Errorf("Disposition = %q; want confirmed-cold", record.Disposition)
	}
	if record.ConfirmedColdReps != l.Reps {
		t.Errorf("ConfirmedColdReps = %d; want %d", record.ConfirmedColdReps, l.Reps)
	}
	if len(record.NotRunCauses) != 0 {
		t.Errorf("NotRunCauses = %+v; want empty", record.NotRunCauses)
	}
}

func TestColdCellDisposition_NoDaemonSignal(t *testing.T) {
	l := mustLoadLadder(t)
	resultsRoot := t.TempDir()

	for n := 1; n <= l.Reps; n++ {
		writeColdCompletedRun(t, resultsRoot, n, true)
	}

	record, err := ColdCellDisposition(l, resultsRoot)
	if err != nil {
		t.Fatalf("ColdCellDisposition() error = %v; want nil", err)
	}
	if record.Disposition != "no-daemon-signal" {
		t.Errorf("Disposition = %q; want no-daemon-signal", record.Disposition)
	}
	if record.ConfirmedColdReps != 0 {
		t.Errorf("ConfirmedColdReps = %d; want 0", record.ConfirmedColdReps)
	}
}

func TestColdCellDisposition_NotRun_NativeFallbackExhaustedOnly(t *testing.T) {
	l := mustLoadLadder(t)
	resultsRoot := t.TempDir()

	for n := 1; n <= l.Reps; n++ {
		writeColdNotRunRepetition(t, resultsRoot, n)
	}

	record, err := ColdCellDisposition(l, resultsRoot)
	if err != nil {
		t.Fatalf("ColdCellDisposition() error = %v; want nil", err)
	}
	if record.Disposition != "not-run" {
		t.Errorf("Disposition = %q; want not-run", record.Disposition)
	}
	if !strings.Contains(record.Reason, "native-fallback") {
		t.Errorf("Reason = %q; want it to name the native-fallback branch", record.Reason)
	}
	for n := 1; n <= l.Reps; n++ {
		key := repKey(n)
		if record.NotRunCauses[key] != coldAbortCauseNativeFallbackExhausted {
			t.Errorf("NotRunCauses[%q] = %q; want %q", key, record.NotRunCauses[key], coldAbortCauseNativeFallbackExhausted)
		}
	}
}

func TestColdCellDisposition_Partial(t *testing.T) {
	l := mustLoadLadder(t)
	resultsRoot := t.TempDir()

	writeColdCompletedRun(t, resultsRoot, 1, false)
	for n := 2; n <= l.Reps; n++ {
		writeColdNotRunRepetition(t, resultsRoot, n)
	}

	record, err := ColdCellDisposition(l, resultsRoot)
	if err != nil {
		t.Fatalf("ColdCellDisposition() error = %v; want nil", err)
	}
	if record.Disposition != "partial" {
		t.Errorf("Disposition = %q; want partial", record.Disposition)
	}
	if record.ConfirmedColdReps != 1 {
		t.Errorf("ConfirmedColdReps = %d; want 1", record.ConfirmedColdReps)
	}
}

func TestColdCellDisposition_BothNotRunCausesNamedInReason(t *testing.T) {
	l := mustLoadLadder(t)
	if l.Reps < 2 {
		t.Fatalf("fixture ladder.yaml reps = %d; this test needs at least 2", l.Reps)
	}
	resultsRoot := t.TempDir()

	// Repetition 1 exhausts on the native-fallback branch (no cold_abort.json anywhere); repetition 2
	// found a live daemon before its very first attempt.
	writeColdNotRunRepetition(t, resultsRoot, 1)
	writeColdNotRunRepetition(t, resultsRoot, 2, 1)
	for n := 3; n <= l.Reps; n++ {
		writeColdNotRunRepetition(t, resultsRoot, n)
	}

	record, err := ColdCellDisposition(l, resultsRoot)
	if err != nil {
		t.Fatalf("ColdCellDisposition() error = %v; want nil", err)
	}
	if record.Disposition != "not-run" {
		t.Errorf("Disposition = %q; want not-run", record.Disposition)
	}
	if !strings.Contains(record.Reason, "native-fallback") {
		t.Errorf("Reason = %q; want it to name the native-fallback cause", record.Reason)
	}
	if !strings.Contains(record.Reason, "live daemon") {
		t.Errorf("Reason = %q; want it to name the live-daemon cause", record.Reason)
	}
	if record.NotRunCauses[repKey(1)] != coldAbortCauseNativeFallbackExhausted {
		t.Errorf("NotRunCauses[1] = %q; want %q", record.NotRunCauses[repKey(1)], coldAbortCauseNativeFallbackExhausted)
	}
	if record.NotRunCauses[repKey(2)] != coldAbortCauseLiveDaemon {
		t.Errorf("NotRunCauses[2] = %q; want %q", record.NotRunCauses[repKey(2)], coldAbortCauseLiveDaemon)
	}
}

// repKey mirrors ColdCellDisposition's own decimal-string repetition key.
func repKey(n int) string {
	return strconv.Itoa(n)
}
