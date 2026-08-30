package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeSubagentMeta writes a minimal agent-<id>.meta.json carrying description at path.
func writeSubagentMeta(t *testing.T, path, description string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.Marshal(subagentMetadata{Description: description})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestMangleProjectDir(t *testing.T) {
	tests := []struct {
		name string
		cwd  string
		want string
	}{
		{"simple", "/home/user/scratch", "-home-user-scratch"},
		{"trailing_slash", "/home/user/scratch/", "-home-user-scratch-"},
		{"single_segment", "/scratch", "-scratch"},
		{"dotted_hidden_dir", "/home/user/.scratch/x", "-home-user--scratch-x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mangleProjectDir(tt.cwd)
			if got != tt.want {
				t.Errorf("mangleProjectDir(%q) = %q; want %q", tt.cwd, got, tt.want)
			}
		})
	}
}

func TestDispatchDescription(t *testing.T) {
	first := DispatchDescription("a5-bundle", 3, 1)
	second := DispatchDescription("a5-bundle", 3, 2)
	if first == second {
		t.Errorf("DispatchDescription must differ across attempts, got equal strings %q", first)
	}
	third := DispatchDescription("a5-bundle", 4, 1)
	if first == third {
		t.Errorf("DispatchDescription must differ across repetitions, got equal strings %q", first)
	}
}

// projectSubagentsDir returns <root>/<mangled sessionDir>/<sessionID>/subagents.
func projectSubagentsDir(root, sessionDir, sessionID string) string {
	return filepath.Join(root, mangleProjectDir(sessionDir), sessionID, "subagents")
}

func TestLocateTranscript_SingleMatch(t *testing.T) {
	root := t.TempDir()
	sessionDir := "/home/user/scratch/a5-bundle-1"
	subagentsDir := projectSubagentsDir(root, sessionDir, "sess-1")

	metaPath := filepath.Join(subagentsDir, "agent-abc123.meta.json")
	writeSubagentMeta(t, metaPath, "ladderbench run a5-bundle rep 1 attempt 1")
	transcriptPath := filepath.Join(subagentsDir, "agent-abc123.jsonl")
	if err := os.WriteFile(transcriptPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	gotTranscript, gotMeta, err := LocateTranscript(root, sessionDir, "ladderbench run a5-bundle rep 1 attempt 1", time.Second)
	if err != nil {
		t.Fatalf("LocateTranscript: unexpected error: %v", err)
	}
	if gotTranscript != transcriptPath {
		t.Errorf("LocateTranscript transcript = %q; want %q", gotTranscript, transcriptPath)
	}
	if gotMeta != metaPath {
		t.Errorf("LocateTranscript meta = %q; want %q", gotMeta, metaPath)
	}
}

func TestLocateTranscript_ZeroMatches(t *testing.T) {
	root := t.TempDir()
	sessionDir := "/home/user/scratch/a5-bundle-1"
	subagentsDir := projectSubagentsDir(root, sessionDir, "sess-1")
	writeSubagentMeta(t, filepath.Join(subagentsDir, "agent-abc123.meta.json"), "some other description")

	_, _, err := LocateTranscript(root, sessionDir, "ladderbench run a5-bundle rep 1 attempt 1", time.Millisecond)
	if err == nil {
		t.Fatal("LocateTranscript: expected error for zero matches, got nil")
	}
}

func TestLocateTranscript_TwoMatches(t *testing.T) {
	root := t.TempDir()
	sessionDir := "/home/user/scratch/a5-bundle-1"
	description := "ladderbench run a5-bundle rep 1 attempt 1"

	subagentsDirA := projectSubagentsDir(root, sessionDir, "sess-1")
	writeSubagentMeta(t, filepath.Join(subagentsDirA, "agent-abc123.meta.json"), description)
	subagentsDirB := projectSubagentsDir(root, sessionDir, "sess-2")
	writeSubagentMeta(t, filepath.Join(subagentsDirB, "agent-def456.meta.json"), description)

	_, _, err := LocateTranscript(root, sessionDir, description, time.Millisecond)
	if err == nil {
		t.Fatal("LocateTranscript: expected error for two matches, got nil")
	}
}

func TestLocateTranscript_TranscriptAppearsWithinWait(t *testing.T) {
	root := t.TempDir()
	sessionDir := "/home/user/scratch/a5-bundle-1"
	subagentsDir := projectSubagentsDir(root, sessionDir, "sess-1")
	description := "ladderbench run a5-bundle rep 1 attempt 1"
	metaPath := filepath.Join(subagentsDir, "agent-abc123.meta.json")
	writeSubagentMeta(t, metaPath, description)
	transcriptPath := filepath.Join(subagentsDir, "agent-abc123.jsonl")

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(transcriptPath, []byte("{}\n"), 0o644)
	}()

	gotTranscript, gotMeta, err := LocateTranscript(root, sessionDir, description, time.Second)
	if err != nil {
		t.Fatalf("LocateTranscript: unexpected error: %v", err)
	}
	if gotTranscript != transcriptPath || gotMeta != metaPath {
		t.Errorf("LocateTranscript = (%q, %q); want (%q, %q)", gotTranscript, gotMeta, transcriptPath, metaPath)
	}
}

func TestLocateTranscript_TranscriptNeverAppears(t *testing.T) {
	root := t.TempDir()
	sessionDir := "/home/user/scratch/a5-bundle-1"
	subagentsDir := projectSubagentsDir(root, sessionDir, "sess-1")
	description := "ladderbench run a5-bundle rep 1 attempt 1"
	writeSubagentMeta(t, filepath.Join(subagentsDir, "agent-abc123.meta.json"), description)

	_, _, err := LocateTranscript(root, sessionDir, description, 150*time.Millisecond)
	if err == nil {
		t.Fatal("LocateTranscript: expected error when transcript never appears, got nil")
	}
}

func TestCopyTranscriptCustody(t *testing.T) {
	root := t.TempDir()
	sessionDir := "/home/user/scratch/a5-bundle-1"
	subagentsDir := projectSubagentsDir(root, sessionDir, "sess-1")

	transcriptPath := filepath.Join(subagentsDir, "agent-abc123.jsonl")
	metaPath := filepath.Join(subagentsDir, "agent-abc123.meta.json")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", subagentsDir, err)
	}
	if err := os.WriteFile(transcriptPath, []byte("transcript-content\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.WriteFile(metaPath, []byte(`{"description":"x"}`), 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	runDir := t.TempDir()
	if err := CopyTranscriptCustody(transcriptPath, metaPath, runDir); err != nil {
		t.Fatalf("CopyTranscriptCustody: unexpected error: %v", err)
	}

	gotTranscript, err := os.ReadFile(filepath.Join(runDir, "transcript.jsonl"))
	if err != nil {
		t.Fatalf("read copied transcript: %v", err)
	}
	if string(gotTranscript) != "transcript-content\n" {
		t.Errorf("copied transcript = %q; want %q", gotTranscript, "transcript-content\n")
	}
	gotMeta, err := os.ReadFile(filepath.Join(runDir, "transcript.meta.json"))
	if err != nil {
		t.Fatalf("read copied meta: %v", err)
	}
	if string(gotMeta) != `{"description":"x"}` {
		t.Errorf("copied meta = %q; want %q", gotMeta, `{"description":"x"}`)
	}
}
