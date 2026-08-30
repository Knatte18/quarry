package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// writeRedactFixtureAnswer writes a minimal exploration answer.json into runDir, quarry-provenance-laden
// so a passing test proves RedactText actually ran rather than merely copying the file.
func writeRedactFixtureAnswer(t *testing.T, runDir string) []byte {
	t.Helper()
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}
	answer := map[string]any{
		"summary":        "used mcp__quarry__toc_file to explore the tree",
		"relevant_files": []string{"internal/reedengine/reed.go"},
		"key_symbols":    []any{},
		"open_questions": []string{},
		"confidence":     "high",
	}
	data, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		t.Fatalf("marshal answer: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "answer.json"), data, 0o644); err != nil {
		t.Fatalf("write answer.json: %v", err)
	}
	return data
}

func TestRunRedact_WritesRedactedFileAndLeavesOriginalByteIdentical(t *testing.T) {
	l := mustLoadLadderFixture(t)
	l.RunModel = strPtr(ingestTestModel)
	maxTurns := 10
	l.MaxTurns = &maxTurns
	l.RunEffort = "medium"
	l.Scorer.Model = ingestTestModel
	l.Scorer.Effort = "high"

	resultsRoot := t.TempDir()
	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	original := writeRedactFixtureAnswer(t, runDir)

	var out bytes.Buffer
	if err := runRedact(&out, l, repoRootFixture, resultsRoot, "a0-none", 1); err != nil {
		t.Fatalf("runRedact() error = %v; want nil", err)
	}

	afterOriginal, err := os.ReadFile(filepath.Join(runDir, "answer.json"))
	if err != nil {
		t.Fatalf("read answer.json: %v", err)
	}
	if !bytes.Equal(afterOriginal, original) {
		t.Errorf("runRedact() mutated the original answer.json; want it byte-identical")
	}

	redactedData, err := os.ReadFile(filepath.Join(runDir, "answer.redacted.json"))
	if err != nil {
		t.Fatalf("read answer.redacted.json: %v", err)
	}
	var redacted map[string]any
	if err := json.Unmarshal(redactedData, &redacted); err != nil {
		t.Fatalf("unmarshal answer.redacted.json: %v", err)
	}
	if summary, _ := redacted["summary"].(string); summary == "used mcp__quarry__toc_file to explore the tree" {
		t.Errorf("answer.redacted.json summary = %q; want tool provenance redacted", summary)
	}
}

func TestRunRedact_PrintsAssembledScorerPrompt(t *testing.T) {
	l := mustLoadLadderFixture(t)
	l.RunModel = strPtr(ingestTestModel)
	maxTurns := 10
	l.MaxTurns = &maxTurns
	l.RunEffort = "medium"
	l.Scorer.Model = ingestTestModel
	l.Scorer.Effort = "high"

	resultsRoot := t.TempDir()
	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	writeRedactFixtureAnswer(t, runDir)

	var out bytes.Buffer
	if err := runRedact(&out, l, repoRootFixture, resultsRoot, "a0-none", 1); err != nil {
		t.Fatalf("runRedact() error = %v; want nil", err)
	}

	config, err := ladder.ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID: %v", err)
	}
	taskText, err := ladder.TaskTextFor(l, repoRootFixture, config.Task)
	if err != nil {
		t.Fatalf("TaskTextFor: %v", err)
	}
	fasitPath := filepath.Join(repoRootFixture, l.Tasks[config.Task].Fasit)
	fasitData, err := os.ReadFile(fasitPath)
	if err != nil {
		t.Fatalf("read fasit: %v", err)
	}
	var fasit map[string]any
	if err := json.Unmarshal(fasitData, &fasit); err != nil {
		t.Fatalf("unmarshal fasit: %v", err)
	}
	redactedAnswer, err := ladder.WriteRedacted(runDir)
	if err != nil {
		t.Fatalf("WriteRedacted: %v", err)
	}
	wantPrompt, err := ladder.BuildScorerPrompt(l, config, redactedAnswer, fasit, taskText)
	if err != nil {
		t.Fatalf("BuildScorerPrompt: %v", err)
	}

	if out.String() != wantPrompt+"\n" {
		t.Errorf("runRedact() printed = %q; want the assembled scorer prompt %q", out.String(), wantPrompt+"\n")
	}
}

func TestRunRedact_UnknownConfigErrors(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	var out bytes.Buffer
	if err := runRedact(&out, l, repoRootFixture, resultsRoot, "does-not-exist", 1); err == nil {
		t.Error("runRedact() = nil; want an error for an unknown config id")
	}
}

// strPtr returns a pointer to s, for setting *ladder.Ladder.RunModel in a test fixture.
func strPtr(s string) *string { return &s }
