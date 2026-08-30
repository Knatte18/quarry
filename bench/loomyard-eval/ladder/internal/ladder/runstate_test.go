package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
