package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// validExplorationReply is a scorer reply satisfying task 01's exploration schema
// (ladder.ExplorationRule's own declared field set).
const validExplorationReply = "```json\n{\"recall\": 0.8, \"precision\": 0.9, \"summary_matches\": true}\n```\n"

// recordScoreFixtureLadder loads the committed ladder.yaml pinned for record-score.
func recordScoreFixtureLadder(t *testing.T) *ladder.Ladder {
	t.Helper()
	l := mustLoadLadderFixture(t)
	l.RunModel = strPtr(ingestTestModel)
	maxTurns := 10
	l.MaxTurns = &maxTurns
	l.RunEffort = "medium"
	l.Scorer.Model = ingestTestModel
	l.Scorer.Effort = "high"
	return l
}

// writeCompleteArtifactsExcept writes every file ladder.GateRunCompleteArtifacts requires except score.json
// (record-score's own job) and every name listed in except, into runDir.
func writeCompleteArtifactsExcept(t *testing.T, runDir string, except ...string) {
	t.Helper()
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}
	skip := map[string]bool{"score.json": true}
	for _, name := range except {
		skip[name] = true
	}
	for _, name := range []string{"answer.json", "answer.redacted.json", "usage.json", "ingest.json", "transcript.jsonl", "transcript.meta.json"} {
		if skip[name] {
			continue
		}
		if err := os.WriteFile(filepath.Join(runDir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if !skip["ingest.json"] {
		if err := ladder.WriteIngestJSON(runDir, ladder.IngestRecord{ConfigID: "a0-none", Rep: 1, Attempt: 1}); err != nil {
			t.Fatalf("WriteIngestJSON: %v", err)
		}
	}
}

func TestRunRecordScore_InvalidReplyRejectedBeforeAnythingIsWritten(t *testing.T) {
	l := recordScoreFixtureLadder(t)
	resultsRoot := t.TempDir()
	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	writeCompleteArtifactsExcept(t, runDir)

	var out bytes.Buffer
	err := runRecordScore(&out, l, resultsRoot, "a0-none", 1, "no fenced json block here")
	if err == nil {
		t.Fatal("runRecordScore() error = nil; want an error for an invalid reply")
	}

	if _, statErr := os.Stat(filepath.Join(runDir, "score.json")); !os.IsNotExist(statErr) {
		t.Error("runRecordScore() wrote score.json despite an invalid reply")
	}
	if _, statErr := os.Stat(filepath.Join(runDir, "run.json")); !os.IsNotExist(statErr) {
		t.Error("runRecordScore() wrote run.json despite an invalid reply")
	}
}

func TestRunRecordScore_ArtifactsGateFailurePreventsRunMarker(t *testing.T) {
	l := recordScoreFixtureLadder(t)
	resultsRoot := t.TempDir()
	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	// Omit transcript.meta.json so the complete-artifacts gate fails after score.json is written.
	writeCompleteArtifactsExcept(t, runDir, "transcript.meta.json")

	var out bytes.Buffer
	if err := runRecordScore(&out, l, resultsRoot, "a0-none", 1, validExplorationReply); err != nil {
		t.Fatalf("runRecordScore() error = %v; want nil", err)
	}

	if _, err := os.Stat(filepath.Join(runDir, "score.json")); err != nil {
		t.Errorf("runRecordScore() did not write score.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "run.json")); !os.IsNotExist(err) {
		t.Error("runRecordScore() wrote run.json despite an incomplete artifact set")
	}
	if ladder.IsComplete(runDir) {
		t.Error("runRecordScore() left the run directory reporting complete despite a failing artifacts gate")
	}
}

func TestRunRecordScore_SuccessfulRunLeavesDirectoryComplete(t *testing.T) {
	l := recordScoreFixtureLadder(t)
	resultsRoot := t.TempDir()
	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	writeCompleteArtifactsExcept(t, runDir)

	var out bytes.Buffer
	if err := runRecordScore(&out, l, resultsRoot, "a0-none", 1, validExplorationReply); err != nil {
		t.Fatalf("runRecordScore() error = %v; want nil", err)
	}

	if !ladder.IsComplete(runDir) {
		t.Error("runRecordScore() did not leave the run directory complete")
	}

	scoreData, err := os.ReadFile(filepath.Join(runDir, "score.json"))
	if err != nil {
		t.Fatalf("read score.json: %v", err)
	}
	if !bytes.Contains(scoreData, []byte(`"model": "`+ingestTestModel+`"`)) {
		t.Errorf("score.json = %s; want it to carry the pinned scorer model", scoreData)
	}
	if !bytes.Contains(scoreData, []byte(`"prompt_template": "exploration"`)) {
		t.Errorf("score.json = %s; want it to carry prompt_template exploration", scoreData)
	}
}

func TestRunRecordScore_UnknownConfigErrors(t *testing.T) {
	l := recordScoreFixtureLadder(t)
	resultsRoot := t.TempDir()

	var out bytes.Buffer
	if err := runRecordScore(&out, l, resultsRoot, "does-not-exist", 1, validExplorationReply); err == nil {
		t.Error("runRecordScore() = nil; want an error for an unknown config id")
	}
}
