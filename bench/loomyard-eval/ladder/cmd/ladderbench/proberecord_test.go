package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// writeProbeFixtureTranscript writes a subagent metadata/transcript pair for kind's probe session,
// carrying one call to a quarry tool whose matching tool_result errors (blocked) or not, per blocked.
// When blocked, the tool_result's text is deniedText.
func writeProbeFixtureTranscript(t *testing.T, l *ladder.Ladder, projectsRoot, kind string, blocked bool, deniedText string) {
	t.Helper()

	configID, err := probeRecordSessionConfigID(kind)
	if err != nil {
		t.Fatalf("probeRecordSessionConfigID(%q): %v", kind, err)
	}
	scratchDir, err := ladder.SessionDir(l, configID, 1)
	if err != nil {
		t.Fatalf("SessionDir: %v", err)
	}
	description := ladder.DispatchDescription(configID, 1, 1)

	subagentsDir := filepath.Join(projectsRoot, mangleProjectDirForTest(scratchDir), "sess-1", "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", subagentsDir, err)
	}
	metaData, err := json.Marshal(map[string]any{"description": description})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subagentsDir, "agent-fixture.meta.json"), metaData, 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	resultContent := []ladder.ContentBlock{{Type: "text", Text: "allowed"}}
	if blocked {
		resultContent = []ladder.ContentBlock{{Type: "text", Text: deniedText}}
	}
	assistantRecord := ladder.Record{
		IsSidechain: true,
		AgentID:     "fixture",
		UUID:        "uuid-a",
		Timestamp:   time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		Type:        "assistant",
		Message: ladder.Message{
			Model: ingestTestModel,
			Content: []ladder.ContentBlock{
				{Type: "tool_use", ToolUseID: "call-1", Name: "mcp__quarry__impact", Input: map[string]any{}},
			},
		},
	}
	userRecord := ladder.Record{
		IsSidechain: true,
		AgentID:     "fixture",
		UUID:        "uuid-b",
		Timestamp:   time.Date(2026, 8, 30, 12, 0, 1, 0, time.UTC).Format(time.RFC3339Nano),
		Type:        "user",
		Message: ladder.Message{
			Content: []ladder.ContentBlock{
				{Type: "tool_result", ToolUseID: "call-1", IsError: blocked, Content: resultContent},
			},
		},
	}

	var lines []string
	for _, record := range []ladder.Record{assistantRecord, userRecord} {
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		lines = append(lines, string(data))
	}
	transcriptPath := filepath.Join(subagentsDir, "agent-fixture.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
}

func newProbeRecordFixtureLadder(t *testing.T) *ladder.Ladder {
	t.Helper()
	l := mustLoadLadderFixture(t)
	l.SessionDirTemplate = filepath.Join(t.TempDir(), "session-{config_id}-{n}")
	return l
}

func TestRunProbeRecord_AllowlistProbeWritesItsOwnKey(t *testing.T) {
	l := newProbeRecordFixtureLadder(t)
	resultsRoot := t.TempDir()
	projectsRoot := t.TempDir()
	writeProbeFixtureTranscript(t, l, projectsRoot, ladder.ProbeKindAllowlist, true, "permission denied by the user")

	var out bytes.Buffer
	if err := runProbeRecord(&out, l, resultsRoot, ladder.ProbeKindAllowlist, projectsRoot, 2*time.Second); err != nil {
		t.Fatalf("runProbeRecord() error = %v; want nil", err)
	}

	record := mustReadProbeJSON(t, resultsRoot)
	if record["allowlist_blocks"] != true {
		t.Errorf("probe.json allowlist_blocks = %v; want true", record["allowlist_blocks"])
	}
	if _, ok := record["denylist_blocks"]; ok {
		t.Errorf("probe.json = %+v; allowlist probe must not write denylist_blocks", record)
	}
}

func TestRunProbeRecord_DenylistProbeCapturesDeniedTextAndExtendsExistingRecord(t *testing.T) {
	l := newProbeRecordFixtureLadder(t)
	resultsRoot := t.TempDir()
	projectsRoot := t.TempDir()

	// The allowlist probe records first, establishing the half denylist-record must not clobber.
	writeProbeFixtureTranscript(t, l, projectsRoot, ladder.ProbeKindAllowlist, true, "n/a")
	var firstOut bytes.Buffer
	if err := runProbeRecord(&firstOut, l, resultsRoot, ladder.ProbeKindAllowlist, projectsRoot, 2*time.Second); err != nil {
		t.Fatalf("runProbeRecord(allowlist) error = %v; want nil", err)
	}

	writeProbeFixtureTranscript(t, l, projectsRoot, ladder.ProbeKindDenylist, true, "permission denied by the user")
	var secondOut bytes.Buffer
	if err := runProbeRecord(&secondOut, l, resultsRoot, ladder.ProbeKindDenylist, projectsRoot, 2*time.Second); err != nil {
		t.Fatalf("runProbeRecord(denylist) error = %v; want nil", err)
	}

	record := mustReadProbeJSON(t, resultsRoot)
	if record["allowlist_blocks"] != true {
		t.Errorf("probe.json allowlist_blocks = %v; want it preserved as true from the first invocation", record["allowlist_blocks"])
	}
	if record["denylist_blocks"] != true {
		t.Errorf("probe.json denylist_blocks = %v; want true", record["denylist_blocks"])
	}
	if record["denial_shape_observed"] != "permission denied by the user" {
		t.Errorf("probe.json denial_shape_observed = %v; want the verbatim denied tool result text", record["denial_shape_observed"])
	}
}

func TestRunProbeRecord_HaltsOnFalseLayer(t *testing.T) {
	l := newProbeRecordFixtureLadder(t)
	resultsRoot := t.TempDir()
	projectsRoot := t.TempDir()
	writeProbeFixtureTranscript(t, l, projectsRoot, ladder.ProbeKindDenylist, false, "")

	var out bytes.Buffer
	err := runProbeRecord(&out, l, resultsRoot, ladder.ProbeKindDenylist, projectsRoot, 2*time.Second)
	if err == nil {
		t.Fatal("runProbeRecord() error = nil; want an error when the probe observed no block")
	}

	record := mustReadProbeJSON(t, resultsRoot)
	if record["denylist_blocks"] != false {
		t.Errorf("probe.json denylist_blocks = %v; want false written before halting", record["denylist_blocks"])
	}
}

// mustReadProbeJSON reads and parses <resultsRoot>/probe.json.
func mustReadProbeJSON(t *testing.T, resultsRoot string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(resultsRoot, "probe.json"))
	if err != nil {
		t.Fatalf("read probe.json: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("unmarshal probe.json: %v", err)
	}
	return record
}
