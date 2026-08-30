// e2e_test.go drives the synthetic end-to-end run this package cannot otherwise catch drift in: the
// per-unit tests in this package each assert a stage's output against its own expectation, never
// against the next stage's expectation, so a field renamed on one side of a handoff and not the other
// passes every per-unit test while silently breaking the matrix. This test instead drives the real
// cross-stage assemblers -- NewIngestRecord (gate report to ingest record) and RunJSONPayload (ingest
// record to run-marker payload) -- over one hand-written subagent transcript, through ingest, gating,
// redaction/scoring, and summarisation, in that order, and asserts the final summary in full. No
// dispatch of any kind happens: the scorer reply is a literal in this file, and the transcript was
// written by hand (see testdata/e2e-transcript.jsonl).

package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// e2eLadderYAMLPath is the synthetic ladder this test loads, distinct from the committed ladder.yaml
// every other test in this package loads through mustLoadLadder.
const e2eLadderYAMLPath = "testdata/e2e-ladder.yaml"

// e2eFinalAnswerFromRecords parses the answer from the last fenced json block of the final assistant
// record in records, mirroring cmd/ladderbench/ingest.go's own parseFinalAnswer. That function lives in
// the main package and is not importable here; this is not a second definition of the extraction rule so
// much as the same three-line call reproduced at the one other place this package's own doc comment
// says a real caller performs it, since driving ladder.ExtractFencedJSON directly (rather than
// reimplementing fence-matching) is what keeps this test bound to the same rule the CLI uses.
func e2eFinalAnswerFromRecords(t *testing.T, records []Record) map[string]any {
	t.Helper()
	assistants := AssistantRecords(records)
	if len(assistants) == 0 {
		t.Fatal("e2eFinalAnswerFromRecords: transcript carries no assistant record")
	}
	final := assistants[len(assistants)-1]

	var text string
	for _, block := range final.Message.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	_, inner, err := ExtractFencedJSON(text, "last")
	if err != nil {
		t.Fatalf("ExtractFencedJSON(final assistant text, last) = _, _, %v; want nil error", err)
	}
	var answer map[string]any
	if err := json.Unmarshal([]byte(inner), &answer); err != nil {
		t.Fatalf("json.Unmarshal(final answer) = %v; want nil error", err)
	}
	return answer
}

// TestSyntheticEndToEnd_MatchesFieldNamesAcrossEveryStageHandoff drives one run of the e2e-ladder.yaml
// fixture's a1-rung config through ingest, gating, redaction/scoring, and summarisation, and asserts the
// final summary in full -- so a field renamed on one side of any of those stages' handoffs fails this
// test rather than silently dropping out of a field-by-field comparison.
func TestSyntheticEndToEnd_MatchesFieldNamesAcrossEveryStageHandoff(t *testing.T) {
	l, err := LoadLadder(e2eLadderYAMLPath)
	if err != nil {
		t.Fatalf("LoadLadder(%q) = _, %v; want nil error", e2eLadderYAMLPath, err)
	}
	if err := RequirePins(l); err != nil {
		t.Fatalf("RequirePins(fixture ladder) = %v; want nil error -- every pin should ship set", err)
	}

	config, err := ConfigByID(l, "a1-rung")
	if err != nil {
		t.Fatalf("ConfigByID(a1-rung) = _, %v; want nil error", err)
	}

	resultsRoot := t.TempDir()
	runDir := RunDirPath(resultsRoot, config.ID, 1)

	worktree := t.TempDir()
	initGitRepo(t, worktree)
	// Dirty the worktree before ObserveWorktreeDirtied runs, so this run's non-fatal worktree_dirtied
	// observation fires true -- the one non-fatal gate-time finding this test carries all the way to the
	// summarised cell (see the final assertion below).
	if err := os.WriteFile(filepath.Join(worktree, "scratch.txt"), []byte("scratch\n"), 0o644); err != nil {
		t.Fatalf("write scratch file into worktree: %v", err)
	}

	// STAGE 1: take transcript custody, then extract usage and the final answer from the run-directory
	// copy -- never the testdata original -- since that copy is what a real caller reads.

	metaPath := filepath.Join(t.TempDir(), "agent-e2e-1.meta.json")
	if err := os.WriteFile(metaPath, []byte(`{"description":"ladderbench run a1-rung rep 1 attempt 1"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write synthetic transcript metadata: %v", err)
	}
	const transcriptSource = "testdata/e2e-transcript.jsonl"
	if err := CopyTranscriptCustody(transcriptSource, metaPath, runDir); err != nil {
		t.Fatalf("CopyTranscriptCustody() = %v; want nil error", err)
	}

	copiedTranscriptPath := filepath.Join(runDir, "transcript.jsonl")
	records, err := ReadTranscript(copiedTranscriptPath)
	if err != nil {
		t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", copiedTranscriptPath, err)
	}

	usage, err := ExtractUsage(records, copiedTranscriptPath, transcriptSource, config.Allowed)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}
	if usage.Tokens.InputTokens == 0 || usage.Tokens.OutputTokens == 0 || usage.Tokens.CacheReadInputTokens == 0 || usage.Tokens.CacheCreationInputTokens == 0 {
		t.Fatalf("ExtractUsage() tokens = %+v; want all four classes non-zero", usage.Tokens)
	}
	if !usage.DeniedToolAttemptsProvisional {
		t.Error("ExtractUsage().DeniedToolAttemptsProvisional = false; want true (this port is always provisional)")
	}
	writeJSONFile(t, filepath.Join(runDir, "usage.json"), usage)

	answer := e2eFinalAnswerFromRecords(t, records)
	writeJSONFile(t, filepath.Join(runDir, "answer.json"), answer)

	// STAGE 2: the gates. The dirtiness observation is taken before anything could restore the worktree
	// -- this test never restores it, but the ordering mirrors runIngest's own.

	dirtied := ObserveWorktreeDirtied(worktree)
	if dirtied.Message != "worktree dirtied: true" {
		t.Fatalf("ObserveWorktreeDirtied() = %+v; want the fixture's own dirtied-true message", dirtied)
	}

	report := RunGates(records, l, config, *l.RunModel, "/repo/root", worktree, *l.MaxTurns, dirtied, t.TempDir(), nil)
	if !report.Passed() {
		t.Fatalf("RunGates() = %+v; want a passing report", report.FatalFindings())
	}

	rec := NewIngestRecord(config.ID, 1, 1, report)
	if err := WriteIngestJSON(runDir, rec); err != nil {
		t.Fatalf("WriteIngestJSON() = %v; want nil error", err)
	}

	// STAGE 3: redact the answer, assemble the scorer prompt, parse a canned reply, and write score.json
	// -- stamped exactly as cmd/ladderbench/recordscore.go stamps a real reply.

	redactedAnswer, err := WriteRedacted(runDir)
	if err != nil {
		t.Fatalf("WriteRedacted() = _, %v; want nil error", err)
	}

	fasitPath := l.Tasks[config.Task].Fasit
	fasitData, err := os.ReadFile(fasitPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) = _, %v; want nil error", fasitPath, err)
	}
	var fasit map[string]any
	if err := json.Unmarshal(fasitData, &fasit); err != nil {
		t.Fatalf("json.Unmarshal(fasit) = %v; want nil error", err)
	}

	const taskText = "Locate Foo and describe where it is defined."
	prompt, err := BuildScorerPrompt(l, config, redactedAnswer, fasit, taskText)
	if err != nil {
		t.Fatalf("BuildScorerPrompt() = _, %v; want nil error", err)
	}
	if prompt == "" {
		t.Fatal("BuildScorerPrompt() = \"\"; want a non-empty prompt")
	}

	const scorerReply = "```json\n{\"recall\": 1.0, \"precision\": 1.0, \"summary_matches\": true}\n```\n"
	scoreRecord, err := ParseScorerReply(scorerReply, "exploration")
	if err != nil {
		t.Fatalf("ParseScorerReply() = _, %v; want nil error", err)
	}
	scoreRecord["model"] = l.Scorer.Model
	scoreRecord["effort"] = l.Scorer.Effort
	scoreRecord["prompt_template"] = "exploration"
	writeJSONFile(t, filepath.Join(runDir, "score.json"), scoreRecord)

	if findings := GateRunCompleteArtifacts(runDir); len(findings) != 0 {
		t.Fatalf("GateRunCompleteArtifacts() = %+v; want no findings once every artifact is written", findings)
	}

	// STAGE 4: read the ingest record back, build the run-marker payload from it, write the run marker,
	// and summarise.

	readBackRec, err := ReadIngestRecord(runDir)
	if err != nil {
		t.Fatalf("ReadIngestRecord() = _, %v; want nil error", err)
	}

	if _, err := WriteRunJSON(runDir, RunJSONPayload(readBackRec, *l.RunModel)); err != nil {
		t.Fatalf("WriteRunJSON() = _, %v; want nil error", err)
	}
	if !IsComplete(runDir) {
		t.Fatal("IsComplete(runDir) = false after WriteRunJSON; want true")
	}

	summary, err := WriteSummary(l, resultsRoot)
	if err != nil {
		t.Fatalf("WriteSummary() = _, %v; want nil error", err)
	}

	want := Summary{
		Meta: SummaryMeta{
			RunModel:              *l.RunModel,
			Scorer:                SummaryScorerMeta{Model: l.Scorer.Model, Effort: l.Scorer.Effort},
			Reps:                  l.Reps,
			ResultsRootDate:       filepath.Base(resultsRoot),
			NumConfigs:            len(l.Configs),
			ColdDisposition:       "unknown",
			ColdConfirmedColdReps: nil,
		},
		Cells: map[string]CellRecord{
			"a0-none":      {Stats: map[string]MetricStats{}, Complete: false},
			"a1-rung-cold": {Stats: map[string]MetricStats{}, Complete: false},
			"b0-none":      {Stats: map[string]MetricStats{}, Complete: false},
			"a1-rung": {
				Stats: map[string]MetricStats{
					"duration_ms":                 {Median: 10000, Min: 10000, Max: 10000, N: 1},
					"num_turns":                   {Median: 3, Min: 3, Max: 3, N: 1},
					"tool_uses":                   {Median: 3, Min: 3, Max: 3, N: 1},
					"quarry_tool_uses":            {Median: 2, Min: 2, Max: 2, N: 1},
					"input_tokens":                {Median: 300, Min: 300, Max: 300, N: 1},
					"output_tokens":               {Median: 150, Min: 150, Max: 150, N: 1},
					"cache_read_input_tokens":     {Median: 30, Min: 30, Max: 30, N: 1},
					"cache_creation_input_tokens": {Median: 20, Min: 20, Max: 20, N: 1},
					"bash_grep_count":             {Median: 1, Min: 1, Max: 1, N: 1},
					"grep_tool_count":             {Median: 0, Min: 0, Max: 0, N: 1},
					"grep_fallback_total":         {Median: 1, Min: 1, Max: 1, N: 1},
					"denied_tool_attempts":        {Median: 0, Min: 0, Max: 0, N: 1, Provisional: true},
					"recall":                      {Median: 1, Min: 1, Max: 1, N: 1},
					"precision":                   {Median: 1, Min: 1, Max: 1, N: 1},
				},
				Complete:                       true,
				DecoyAdmittedCount:             0,
				SummaryMatches:                 []any{true},
				WorktreeDirtiedCount:           1,
				TargetOriginQuarryMentionCount: 0,
				DaemonBackedRuns:               1,
			},
		},
		Comparisons: nil,
		Incomplete:  []string{"a0-none", "a1-rung-cold", "b0-none"},
	}

	if diff := cmp.Diff(want, summary); diff != "" {
		t.Errorf("WriteSummary() mismatch (-want +got):\n%s", diff)
	}

	// The provisional denial marker survives from the usage record (ExtractUsage's
	// DeniedToolAttemptsProvisional) all the way onto the summarised cell's own denied_tool_attempts
	// stats record -- asserted again here by name, on top of the full-value comparison above, since it
	// is one of the two handoffs this test exists to catch drift in.
	if !summary.Cells["a1-rung"].Stats["denied_tool_attempts"].Provisional {
		t.Error("summary.Cells[a1-rung].Stats[denied_tool_attempts].Provisional = false; want true")
	}
	// The non-fatal worktree_dirtied observation taken at gate time survives the full gate-report ->
	// ingest-record -> run-marker -> summary chain, which no per-unit test spans.
	if summary.Cells["a1-rung"].WorktreeDirtiedCount != 1 {
		t.Errorf("summary.Cells[a1-rung].WorktreeDirtiedCount = %d; want 1", summary.Cells["a1-rung"].WorktreeDirtiedCount)
	}

	onDisk, err := os.ReadFile(filepath.Join(resultsRoot, "summary.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(summary.json) = _, %v; want nil error", err)
	}
	var onDiskSummary Summary
	if err := json.Unmarshal(onDisk, &onDiskSummary); err != nil {
		t.Fatalf("json.Unmarshal(summary.json) = %v; want nil error", err)
	}
	if diff := cmp.Diff(want, onDiskSummary); diff != "" {
		t.Errorf("on-disk summary.json mismatch (-want +got):\n%s", diff)
	}
}
