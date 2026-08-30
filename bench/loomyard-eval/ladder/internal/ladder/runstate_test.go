package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIngestMarker_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	rec := IngestRecord{
		ConfigID: "a5-bundle",
		Rep:      1,
		Attempt:  1,
		Observations: []GateFinding{
			{Gate: "worktree_dirtied", Fatal: false, Message: "worktree dirtied: false"},
		},
	}

	if HasIngest(dir) {
		t.Fatal("HasIngest() = true before WriteIngestJSON; want false")
	}
	if err := WriteIngestJSON(dir, rec); err != nil {
		t.Fatalf("WriteIngestJSON() = %v; want nil error", err)
	}
	if !HasIngest(dir) {
		t.Fatal("HasIngest() = false after WriteIngestJSON; want true")
	}

	got, err := ReadIngestRecord(dir)
	if err != nil {
		t.Fatalf("ReadIngestRecord() = _, %v; want nil error", err)
	}
	if got.ConfigID != rec.ConfigID || got.Rep != rec.Rep || got.Attempt != rec.Attempt {
		t.Errorf("ReadIngestRecord() = %+v; want %+v", got, rec)
	}
	if len(got.Observations) != 1 || got.Observations[0].Gate != "worktree_dirtied" {
		t.Errorf("ReadIngestRecord().Observations = %+v; want one worktree_dirtied finding", got.Observations)
	}
}

func TestNewIngestRecord_AssemblesFromNonFatalFindings(t *testing.T) {
	report := GateReport{Findings: []GateFinding{
		{Gate: "worktree_dirtied", Fatal: false, Message: "worktree dirtied: true"},
	}}
	rec := NewIngestRecord("a5-bundle", 2, 1, report)
	if rec.ConfigID != "a5-bundle" || rec.Rep != 2 || rec.Attempt != 1 {
		t.Errorf("NewIngestRecord() = %+v; want config_id=a5-bundle rep=2 attempt=1", rec)
	}
	if len(rec.Observations) != 1 || rec.Observations[0].Gate != "worktree_dirtied" {
		t.Errorf("NewIngestRecord().Observations = %+v; want one worktree_dirtied finding", rec.Observations)
	}
}

func TestRunJSONPayload_EmitsBothStructuredAndLiftedShapes(t *testing.T) {
	rec := IngestRecord{
		ConfigID: "a5-bundle",
		Rep:      1,
		Attempt:  1,
		Observations: []GateFinding{
			{Gate: "worktree_dirtied", Fatal: false, Message: "worktree dirtied: true"},
			{Gate: "cold_no_daemon_backed_call", Fatal: false, Message: "no daemon-backed tool call observed"},
		},
	}

	payload := RunJSONPayload(rec, "claude-opus-5")

	if payload["config_id"] != "a5-bundle" || payload["n"] != 1 || payload["model"] != "claude-opus-5" {
		t.Errorf("payload = %+v; want config_id/n/model set from rec and runModel", payload)
	}

	observations, ok := payload["observations"].([]map[string]string)
	if !ok || len(observations) != 2 {
		t.Fatalf("payload[observations] = %+v; want 2 structured entries", payload["observations"])
	}

	if payload["worktree_dirtied"] != true {
		t.Errorf("payload[worktree_dirtied] = %v; want true", payload["worktree_dirtied"])
	}
	if payload["cold_no_daemon_backed_call"] != true {
		t.Errorf("payload[cold_no_daemon_backed_call] = %v; want true", payload["cold_no_daemon_backed_call"])
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal(payload) = _, %v; want nil error", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(payload) = %v; want nil error", err)
	}
	decodedObservations, ok := decoded["observations"].([]any)
	if !ok || len(decodedObservations) != 2 {
		t.Fatalf("decoded[observations] = %+v; want 2 entries", decoded["observations"])
	}
	if decoded["worktree_dirtied"] != true || decoded["cold_no_daemon_backed_call"] != true {
		t.Errorf("decoded lifted keys = %+v; want both true, agreeing with the structured list", decoded)
	}
}

func TestRunJSONPayload_WorktreeDirtiedFalseLiftsFalse(t *testing.T) {
	rec := IngestRecord{
		ConfigID:     "a5-bundle",
		Rep:          1,
		Attempt:      1,
		Observations: []GateFinding{{Gate: "worktree_dirtied", Fatal: false, Message: "worktree dirtied: false"}},
	}
	payload := RunJSONPayload(rec, "claude-opus-5")
	if payload["worktree_dirtied"] != false {
		t.Errorf("payload[worktree_dirtied] = %v; want false", payload["worktree_dirtied"])
	}
}

func TestInvalidate_PicksLowestUnusedIndex(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeJSONFile(t, filepath.Join(dir, "answer.json"), map[string]any{})

	next, err := Invalidate(dir)
	if err != nil {
		t.Fatalf("Invalidate() = _, %v; want nil error", err)
	}
	if next != 2 {
		t.Errorf("Invalidate() next attempt = %d; want 2", next)
	}
	first := filepath.Join(root, "1.invalid-1")
	if _, err := os.Stat(first); err != nil {
		t.Errorf("Invalidate() did not create %s: %v", first, err)
	}
	if _, err := os.Stat(filepath.Join(first, "answer.json")); err != nil {
		t.Errorf("Invalidate() lost answer.json in the moved directory: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("original run dir %s still exists after Invalidate()", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	next, err = Invalidate(dir)
	if err != nil {
		t.Fatalf("Invalidate() (second) = _, %v; want nil error", err)
	}
	if next != 3 {
		t.Errorf("Invalidate() (second) next attempt = %d; want 3", next)
	}
	if _, err := os.Stat(filepath.Join(root, "1.invalid-2")); err != nil {
		t.Errorf("Invalidate() (second) did not create 1.invalid-2: %v", err)
	}
}

func TestInvalidate_ErrorsAtTheCeiling(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "1")

	for i := 0; i < MaxAttempts-1; i++ {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, err := Invalidate(dir); err != nil {
			t.Fatalf("Invalidate() attempt %d = _, %v; want nil error", i+1, err)
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := Invalidate(dir); err == nil {
		t.Fatal("Invalidate() at MaxAttempts = nil error; want an error")
	}
	if _, err := os.Stat(filepath.Join(root, fmt.Sprintf("1.invalid-%d", MaxAttempts))); err != nil {
		t.Errorf("Invalidate() at the ceiling did not still create the %d-th invalid sibling: %v", MaxAttempts, err)
	}
}

func TestNextAttempt_DerivesFromExistingInvalidSiblings(t *testing.T) {
	root := t.TempDir()

	attempt, err := NextAttempt(root, "a5-bundle", 1)
	if err != nil {
		t.Fatalf("NextAttempt() with no invalid siblings = _, %v; want nil error", err)
	}
	if attempt != 1 {
		t.Errorf("NextAttempt() with no invalid siblings = %d; want 1", attempt)
	}

	dir := RunDirPath(root, "a5-bundle", 1)
	for k := 1; k <= 2; k++ {
		if err := os.MkdirAll(invalidSiblingPath(dir, k), 0o755); err != nil {
			t.Fatalf("mkdir invalid sibling %d: %v", k, err)
		}
	}
	attempt, err = NextAttempt(root, "a5-bundle", 1)
	if err != nil {
		t.Fatalf("NextAttempt() with 2 invalid siblings = _, %v; want nil error", err)
	}
	if attempt != 3 {
		t.Errorf("NextAttempt() with 2 invalid siblings = %d; want 3", attempt)
	}
}

func TestRunDirPath_BuildsExpectedPath(t *testing.T) {
	got := RunDirPath("/results/2026-08-29", "a5-bundle", 1)
	want := filepath.Join("/results/2026-08-29", "raw", "a5-bundle", "1")
	if got != want {
		t.Errorf("RunDirPath() = %q; want %q", got, want)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%v) = _, %v; want nil error", value, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%s) = %v; want nil error", path, err)
	}
}

func TestIsComplete_TrueForStateComplete(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "run.json"), map[string]any{"state": "complete"})
	if !IsComplete(dir) {
		t.Error("IsComplete() = false; want true")
	}
}

func TestIsComplete_FalseWithoutRunJSON(t *testing.T) {
	dir := t.TempDir()
	if IsComplete(dir) {
		t.Error("IsComplete() = true; want false")
	}
}

func TestIsComplete_FalseForOtherState(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "run.json"), map[string]any{"state": "invalidated"})
	if IsComplete(dir) {
		t.Error("IsComplete() = true; want false")
	}
}

func TestIsComplete_FalseWithAnswerAndUsageButNoScore(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "answer.json"), map[string]any{})
	writeJSONFile(t, filepath.Join(dir, "usage.json"), map[string]any{})
	if IsComplete(dir) {
		t.Error("IsComplete() = true; want false")
	}
}

func TestWriteRunJSON_RoundTripsThroughIsComplete(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "1")
	payload := map[string]any{"config_id": "a5-bundle", "n": 1, "model": "claude-opus-5", "observations": []any{}}

	record, err := WriteRunJSON(dir, payload)
	if err != nil {
		t.Fatalf("WriteRunJSON() = _, %v; want nil error", err)
	}
	if record["state"] != "complete" {
		t.Errorf("record[state] = %v; want complete", record["state"])
	}
	if record["timestamp"] == nil || record["timestamp"] == "" {
		t.Error("record[timestamp] is empty; want a stamped timestamp")
	}
	if record["config_id"] != "a5-bundle" {
		t.Errorf("record[config_id] = %v; want a5-bundle", record["config_id"])
	}

	if !IsComplete(dir) {
		t.Error("IsComplete() = false after WriteRunJSON; want true")
	}

	onDisk, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(run.json) = _, %v; want nil error", err)
	}
	var onDiskRecord map[string]any
	if err := json.Unmarshal(onDisk, &onDiskRecord); err != nil {
		t.Fatalf("json.Unmarshal(run.json) = %v; want nil error", err)
	}
	if onDiskRecord["state"] != "complete" || onDiskRecord["config_id"] != "a5-bundle" {
		t.Errorf("on-disk run.json = %+v; want state=complete config_id=a5-bundle", onDiskRecord)
	}
}
