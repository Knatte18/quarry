package ladder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTranscript writes lines (already-JSON-encoded, one per line) to a temp file and returns its
// path.
func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent-1.jsonl")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) = %v; want nil error", path, err)
	}
	return path
}

func TestReadTranscript_ParsesRecordsInOrder(t *testing.T) {
	path := writeTranscript(t,
		`{"parentUuid":null,"isSidechain":true,"agentId":"agent-1","uuid":"u1","timestamp":"2026-08-30T11:00:00.000Z","type":"assistant","effort":"medium","message":{"model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"content":[{"type":"text","text":"hello"}]}}`,
		``,
		`{"parentUuid":"u1","isSidechain":true,"agentId":"agent-1","uuid":"u2","timestamp":"2026-08-30T11:00:01.000Z","type":"user","message":{"content":[]}}`,
	)

	records, err := ReadTranscript(path)
	if err != nil {
		t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", path, err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d; want 2", len(records))
	}
	if records[0].UUID != "u1" || records[0].Type != "assistant" {
		t.Errorf("records[0] = %+v; want uuid u1, type assistant", records[0])
	}
	if records[1].UUID != "u2" || records[1].Type != "user" {
		t.Errorf("records[1] = %+v; want uuid u2, type user", records[1])
	}
	if records[0].Message.Content[0].Text != "hello" {
		t.Errorf("records[0].Message.Content[0].Text = %q; want %q", records[0].Message.Content[0].Text, "hello")
	}
}

func TestReadTranscript_MalformedLineReturnsTranscriptErrorNamingLineNumber(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"assistant","message":{"content":[]}}`,
		`{not valid json`,
	)

	_, err := ReadTranscript(path)
	if err == nil {
		t.Fatal("ReadTranscript() = _, nil; want an error")
	}
	transcriptErr, ok := err.(*TranscriptError)
	if !ok {
		t.Fatalf("ReadTranscript() error type = %T; want *TranscriptError", err)
	}
	if !strings.Contains(transcriptErr.Message, "line 2") {
		t.Errorf("TranscriptError.Message = %q; want it to name line 2", transcriptErr.Message)
	}
}

func TestIterToolUseBlocks_YieldsIDNameInputInTranscriptOrder(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"thinking out loud"},{"type":"tool_use","tool_use_id":"tu_1","name":"Read","input":{"file_path":"go.mod"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu_1","is_error":false,"content":[{"type":"text","text":"module foo"}]}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","tool_use_id":"tu_2","name":"Grep","input":{"pattern":"foo"}}]}}`,
	)

	records, err := ReadTranscript(path)
	if err != nil {
		t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", path, err)
	}

	blocks := IterToolUseBlocks(records)
	if len(blocks) != 2 {
		t.Fatalf("len(IterToolUseBlocks(records)) = %d; want 2", len(blocks))
	}
	if blocks[0].ToolUseID != "tu_1" || blocks[0].Name != "Read" || blocks[0].Input["file_path"] != "go.mod" {
		t.Errorf("blocks[0] = %+v; want tu_1/Read/{file_path: go.mod}", blocks[0])
	}
	if blocks[1].ToolUseID != "tu_2" || blocks[1].Name != "Grep" {
		t.Errorf("blocks[1] = %+v; want tu_2/Grep", blocks[1])
	}
}

func TestIterToolUses_YieldsNameInputPairsOnly(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","tool_use_id":"tu_1","name":"Read","input":{"file_path":"go.mod"}}]}}`,
	)
	records, err := ReadTranscript(path)
	if err != nil {
		t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", path, err)
	}

	uses := IterToolUses(records)
	if len(uses) != 1 {
		t.Fatalf("len(IterToolUses(records)) = %d; want 1", len(uses))
	}
	if uses[0].Name != "Read" || uses[0].Input["file_path"] != "go.mod" {
		t.Errorf("uses[0] = %+v; want Read/{file_path: go.mod}", uses[0])
	}
}

func TestAssistantRecords_FiltersOnType(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"assistant","uuid":"u1","message":{"content":[]}}`,
		`{"type":"user","uuid":"u2","message":{"content":[]}}`,
		`{"type":"assistant","uuid":"u3","message":{"content":[]}}`,
	)
	records, err := ReadTranscript(path)
	if err != nil {
		t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", path, err)
	}

	assistants := AssistantRecords(records)
	if len(assistants) != 2 {
		t.Fatalf("len(AssistantRecords(records)) = %d; want 2", len(assistants))
	}
	if assistants[0].UUID != "u1" || assistants[1].UUID != "u3" {
		t.Errorf("assistants = %+v; want uuids u1, u3", assistants)
	}
}
