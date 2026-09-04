package ladder

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestParseTranscript_SkipsUnmodelledType asserts that a record whose type ParseTranscript does
// not model (here "rate_limit_event") is skipped rather than causing an error, and that it is
// excluded from Records but still preserved as a raw line.
func TestParseTranscript_SkipsUnmodelledType(t *testing.T) {
	data, err := os.ReadFile("testdata/transcripts/grouped-usage.jsonl")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	transcript, err := ParseTranscript(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseTranscript() error = %v, want nil", err)
	}

	for _, record := range transcript.Records {
		if record.Type == "rate_limit_event" {
			t.Errorf("Records contains an unmodelled type %q, want it skipped", record.Type)
		}
	}

	found := false
	for _, line := range transcript.Lines {
		if bytes.Contains(line, []byte(`"rate_limit_event"`)) {
			found = true
		}
	}
	if !found {
		t.Errorf("Lines does not contain the unmodelled-type line, want it preserved")
	}
}

// TestParseTranscript_TruncatedLineErrors asserts that a line which is not valid JSON at all
// produces an error naming its 1-based line number, distinguishing a truncated file from an
// unknown record type.
func TestParseTranscript_TruncatedLineErrors(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"system","subtype":"init"}`,
		`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"hi"}]}}`,
		`{"type":"result","truncated`, // line 3: deliberately malformed
	}, "\n")

	_, err := ParseTranscript(strings.NewReader(input))
	if err == nil {
		t.Fatalf("ParseTranscript() error = nil, want an error naming line 3")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("ParseTranscript() error = %q, want it to name line 3", err.Error())
	}
}

// TestParseTranscript_LargeLineReadWhole asserts that a record longer than the scanner's default
// 64 KiB buffer is read in full rather than truncated or dropped.
func TestParseTranscript_LargeLineReadWhole(t *testing.T) {
	padding := strings.Repeat("x", 200*1024) // well past the default 64 KiB scanner limit
	line := fmt.Sprintf(
		`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"%s"}]}}`,
		padding,
	)

	transcript, err := ParseTranscript(strings.NewReader(line))
	if err != nil {
		t.Fatalf("ParseTranscript() error = %v, want nil", err)
	}
	if len(transcript.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(transcript.Records))
	}
	got := transcript.Records[0].Message.Content[0].Text
	if got != padding {
		t.Errorf("large text block was not read whole: got %d bytes, want %d", len(got), len(padding))
	}
}

// TestParseTranscript_SessionInitMCPServers pins the session-init record's "mcp_servers" shape
// against two real fixtures: a connected server (an object list) and a control cell (an empty list).
// The connected-server fixture is copied verbatim, apart from four scrubbed machine paths and
// identifiers, from a real invalidated a2-toc-dir repetition run on this host against
// claude_code_version 2.1.236 -- see .scratch/orch-evidence/a2-invalid-real-transcript.jsonl, which
// is what first showed this harness that Claude Code emits mcp_servers as a list of
// {"name","status"} objects rather than a list of names, the defect this test guards against.
func TestParseTranscript_SessionInitMCPServers(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		wantLen        int
		wantFirstName  string
		wantFirstState string
	}{
		{
			name:           "ConnectedServer",
			fixture:        "session-init-mcp-connected.jsonl",
			wantLen:        1,
			wantFirstName:  "quarry",
			wantFirstState: "connected",
		},
		{
			name:    "ControlCellEmptyList",
			fixture: "grouped-usage.jsonl",
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/transcripts/" + tt.fixture)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			transcript, err := ParseTranscript(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("ParseTranscript() error = %v, want nil", err)
			}
			if transcript.Init == nil {
				t.Fatal("transcript.Init = nil, want a decoded session-init record")
			}
			if len(transcript.Init.MCPServers) != tt.wantLen {
				t.Fatalf("len(Init.MCPServers) = %d, want %d", len(transcript.Init.MCPServers), tt.wantLen)
			}
			if tt.wantLen == 0 {
				return
			}
			got := transcript.Init.MCPServers[0]
			if got.Name != tt.wantFirstName || got.Status != tt.wantFirstState {
				t.Errorf("Init.MCPServers[0] = %+v, want {Name: %q, Status: %q}", got, tt.wantFirstName, tt.wantFirstState)
			}
		})
	}
}

// TestTranscript_MarshalAllRoundTrips asserts that MarshalAll reproduces every raw line of a real
// fixture losslessly.
func TestTranscript_MarshalAllRoundTrips(t *testing.T) {
	data, err := os.ReadFile("testdata/transcripts/tool-bytes.jsonl")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	transcript, err := ParseTranscript(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseTranscript() error = %v", err)
	}

	marshalled, err := transcript.MarshalAll()
	if err != nil {
		t.Fatalf("MarshalAll() error = %v", err)
	}

	wantLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	gotLines := strings.Split(strings.TrimRight(string(marshalled), "\n"), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("MarshalAll() produced %d lines, want %d", len(gotLines), len(wantLines))
	}
	for i := range wantLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("MarshalAll() line %d = %q, want %q", i+1, gotLines[i], wantLines[i])
		}
	}
}
