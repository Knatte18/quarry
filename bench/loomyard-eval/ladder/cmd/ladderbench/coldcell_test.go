package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func TestRunColdCellTeardown_RemovesWorktreeEvenAfterAFailedRun(t *testing.T) {
	l := mustLoadLadderFixture(t)
	sourceRepo := t.TempDir()
	t.Setenv("LADDER_LOOMYARD_REPO", sourceRepo)

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	// runColdCellTeardown takes no notion of the preceding attempt's outcome at all -- it is called
	// once after any outcome (complete, failed, or truncated) and always removes the worktree. This
	// simulates the failed-run case by simply calling it with nothing about outcome threaded through.
	var out bytes.Buffer
	if err := runColdCellTeardown(&out, l, 2, git); err != nil {
		t.Fatalf("runColdCellTeardown() error = %v; want nil", err)
	}

	wantTargetDir := coldWorktreeDir(l, 2)
	found := false
	for _, call := range calls {
		if len(call) >= 4 && call[0] == "-C" && call[1] == sourceRepo && call[2] == "worktree" && call[3] == "remove" {
			for _, arg := range call {
				if arg == wantTargetDir {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("git calls = %v; want a worktree remove call naming %s", calls, wantTargetDir)
	}
	if !strings.Contains(out.String(), wantTargetDir) {
		t.Errorf("runColdCellTeardown() output = %q; want it to name the torn-down worktree", out.String())
	}
}

func TestRunColdCellFinalize_WritesTheDispositionRecord(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	coldConfig, err := ladder.ConfigByID(l, "a5-bundle-cold")
	if err != nil {
		t.Fatalf("ConfigByID: %v", err)
	}
	for n := 1; n <= l.Reps; n++ {
		runDir := ladder.RunDirPath(resultsRoot, coldConfig.ID, n)
		if _, err := ladder.WriteRunJSON(runDir, map[string]any{"observations": []map[string]string{}}); err != nil {
			t.Fatalf("WriteRunJSON(rep %d): %v", n, err)
		}
	}

	var out bytes.Buffer
	if err := runColdCellFinalize(&out, l, resultsRoot); err != nil {
		t.Fatalf("runColdCellFinalize() error = %v; want nil", err)
	}

	data, err := os.ReadFile(filepath.Join(resultsRoot, "cold_cell.json"))
	if err != nil {
		t.Fatalf("read cold_cell.json: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("unmarshal cold_cell.json: %v", err)
	}
	if record["disposition"] != "confirmed-cold" {
		t.Errorf("cold_cell.json disposition = %v; want confirmed-cold", record["disposition"])
	}
	if !strings.Contains(out.String(), "confirmed-cold") {
		t.Errorf("runColdCellFinalize() output = %q; want it to report the disposition", out.String())
	}
}
