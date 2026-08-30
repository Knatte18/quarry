package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// writeMinimalCompleteRun writes the minimal set of files ladder.IsComplete/LoadRuns need to treat
// configID's n-th repetition as a complete run: empty usage.json and score.json objects, and a run.json
// stamped complete via ladder.WriteRunJSON.
func writeMinimalCompleteRun(t *testing.T, resultsRoot, configID string, n int) {
	t.Helper()
	runDir := ladder.RunDirPath(resultsRoot, configID, n)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}
	for _, name := range []string{"usage.json", "score.json"} {
		if err := os.WriteFile(filepath.Join(runDir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if _, err := ladder.WriteRunJSON(runDir, map[string]any{"observations": []map[string]string{}}); err != nil {
		t.Fatalf("WriteRunJSON(%s rep %d): %v", configID, n, err)
	}
}

func TestRunSummarize_ZeroExitOnAFullyCompleteMatrix(t *testing.T) {
	l := mustLoadLadderFixture(t)
	l.Reps = 1
	resultsRoot := t.TempDir()

	for _, config := range l.Configs {
		writeMinimalCompleteRun(t, resultsRoot, config.ID, 1)
	}

	var out bytes.Buffer
	if err := runSummarize(&out, l, resultsRoot); err != nil {
		t.Fatalf("runSummarize() error = %v; want nil for a fully complete matrix", err)
	}

	if _, err := os.Stat(filepath.Join(resultsRoot, "summary.json")); err != nil {
		t.Errorf("runSummarize() did not write summary.json: %v", err)
	}
}

func TestRunSummarize_NonZeroAndNamesIncompleteCells(t *testing.T) {
	l := mustLoadLadderFixture(t)
	l.Reps = 1
	resultsRoot := t.TempDir()

	for _, config := range l.Configs {
		if config.ID == "a0-none" {
			continue // Left incomplete deliberately.
		}
		writeMinimalCompleteRun(t, resultsRoot, config.ID, 1)
	}

	var out bytes.Buffer
	err := runSummarize(&out, l, resultsRoot)
	if err == nil {
		t.Fatal("runSummarize() error = nil; want a non-nil error for an incomplete matrix")
	}
	if !strings.Contains(err.Error(), "a0-none") {
		t.Errorf("runSummarize() error = %v; want it to name the incomplete cell a0-none", err)
	}

	if _, statErr := os.Stat(filepath.Join(resultsRoot, "summary.json")); statErr != nil {
		t.Errorf("runSummarize() did not write summary.json for a partial matrix: %v", statErr)
	}
}
