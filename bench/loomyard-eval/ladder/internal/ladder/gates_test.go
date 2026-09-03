package ladder

import (
	"os"
	"strings"
	"testing"
)

// parseTranscriptFromLines joins lines with newlines and parses the result as a stream-json
// transcript, failing the test on any parse error.
func parseTranscriptFromLines(t *testing.T, lines ...string) *Transcript {
	t.Helper()
	transcript, err := ParseTranscript(strings.NewReader(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatalf("ParseTranscript() error = %v, want nil", err)
	}
	return transcript
}

// systemInitLine is a minimal "system"/"init" record shared by every synthetic transcript this file
// builds.
const systemInitLine = `{"type":"system","subtype":"init","uuid":"init-1","timestamp":"2026-01-01T00:00:00Z","session_id":"sess-1","model":"claude-x","tools":["Bash","Glob","Grep","Read"],"mcp_servers":[],"permissionMode":"default","claude_code_version":"2.1.236","memory_paths":{},"skills":[],"slash_commands":[]}`

// resultLine is a minimal "result" record shared by every synthetic transcript this file builds.
const resultLine = `{"type":"result","subtype":"success","uuid":"r1","timestamp":"2026-01-01T00:00:02Z","num_turns":1,"duration_ms":100,"duration_api_ms":80,"total_cost_usd":0.01,"terminal_reason":"completed","stop_reason":"end_turn","is_error":false,"permission_denials":[]}`

func TestCheckBlinding_CheckA_FatalOnLeakedPrefixFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/transcripts/leaked-prefix.jsonl")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	transcript, err := ParseTranscript(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("ParseTranscript() error = %v", err)
	}

	findings := CheckBlinding(transcript, BlindingInput{
		MCPPrefix:      "mcp__quarry__",
		QuarryRepoRoot: "/nonexistent-repo-root",
	})

	if len(findings) != 1 {
		t.Fatalf("CheckBlinding() returned %d findings, want 1", len(findings))
	}
	if !findings[0].Fatal {
		t.Errorf("CheckBlinding() finding.Fatal = false, want true")
	}
	if findings[0].Gate != "control_blinding_mcp_prefix" {
		t.Errorf("CheckBlinding() finding.Gate = %q, want %q", findings[0].Gate, "control_blinding_mcp_prefix")
	}
}

func TestCheckBlinding_CheckA_UsesSuppliedPrefixNotLiteral(t *testing.T) {
	transcript := parseTranscriptFromLines(t,
		systemInitLine,
		`{"type":"assistant","uuid":"a1","timestamp":"2026-01-01T00:00:01Z","message":{"id":"msg_1","model":"claude-x","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","id":"tu1","name":"mcp__widget__foo","input":{}}]}}`,
		resultLine,
	)

	findings := CheckBlinding(transcript, BlindingInput{
		MCPPrefix:      "mcp__widget__",
		QuarryRepoRoot: "/nonexistent-repo-root",
	})

	if len(findings) != 1 {
		t.Fatalf("CheckBlinding() returned %d findings, want 1", len(findings))
	}
	if !findings[0].Fatal {
		t.Errorf("CheckBlinding() finding.Fatal = false, want true")
	}
	if !strings.Contains(findings[0].Message, "mcp__widget__") {
		t.Errorf("CheckBlinding() finding.Message = %q, want it to name the supplied prefix mcp__widget__", findings[0].Message)
	}

	// The same transcript against a prefix that is not present must produce no check (a) finding
	// and no target_origin_quarry_mention observation either, since the transcript names no bare
	// "quarry" token at all.
	clean := CheckBlinding(transcript, BlindingInput{
		MCPPrefix:      "mcp__quarry__",
		QuarryRepoRoot: "/nonexistent-repo-root",
	})
	if clean != nil {
		t.Errorf("CheckBlinding() with an absent prefix = %+v, want nil", clean)
	}
}

func TestCheckBlinding_CheckB_FatalOnSuppliedRepoRoot(t *testing.T) {
	transcript := parseTranscriptFromLines(t,
		systemInitLine,
		`{"type":"assistant","uuid":"a1","timestamp":"2026-01-01T00:00:01Z","message":{"id":"msg_1","model":"claude-x","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","id":"tu1","name":"Read","input":{"file_path":"/custom/repo/root/README.md"}}]}}`,
		resultLine,
	)

	findings := CheckBlinding(transcript, BlindingInput{
		MCPPrefix:      "mcp__doesnotappear__",
		QuarryRepoRoot: "/custom/repo/root",
	})

	if len(findings) != 1 {
		t.Fatalf("CheckBlinding() returned %d findings, want 1", len(findings))
	}
	if !findings[0].Fatal {
		t.Errorf("CheckBlinding() finding.Fatal = false, want true")
	}
	if findings[0].Gate != "control_blinding_repo_root" {
		t.Errorf("CheckBlinding() finding.Gate = %q, want %q", findings[0].Gate, "control_blinding_repo_root")
	}
}

func TestCheckBlinding_CheckC_NonFatalObservationOnFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/transcripts/target-origin-quarry.jsonl")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	transcript, err := ParseTranscript(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("ParseTranscript() error = %v", err)
	}

	findings := CheckBlinding(transcript, BlindingInput{
		MCPPrefix:      "mcp__quarry__",
		QuarryRepoRoot: "/nonexistent-repo-root",
	})

	if len(findings) != 1 {
		t.Fatalf("CheckBlinding() returned %d findings, want 1", len(findings))
	}
	got := findings[0]
	if got.Fatal {
		t.Errorf("CheckBlinding() finding.Fatal = true, want false")
	}
	if got.Gate != "target_origin_quarry_mention" {
		t.Errorf("CheckBlinding() finding.Gate = %q, want %q", got.Gate, "target_origin_quarry_mention")
	}
	if got.Count != 2 {
		t.Errorf("CheckBlinding() finding.Count = %d, want 2 (one tool_result occurrence, one assistant-text occurrence)", got.Count)
	}
	if !strings.Contains(got.Message, "survives-tool-result-redaction=true") {
		t.Errorf("CheckBlinding() finding.Message = %q, want it to report the token surviving tool_result redaction (it also appears in assistant prose)", got.Message)
	}
}

// TestCheckBlinding_CheckC_NeverFatal asserts that check (c) is non-fatal for every combination of
// the two antecedent booleans and of the token appearing inside or outside a tool_result block --
// the point being that no input makes it fatal, so a location-based fatal branch cannot come back as
// code.
func TestCheckBlinding_CheckC_NeverFatal(t *testing.T) {
	insideOnly := []string{
		systemInitLine,
		`{"type":"assistant","uuid":"a1","timestamp":"2026-01-01T00:00:01Z","message":{"id":"msg_1","model":"claude-x","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","id":"tu1","name":"Read","input":{"file_path":"README.md"}}]}}`,
		`{"type":"user","uuid":"u1","timestamp":"2026-01-01T00:00:02Z","message":{"content":[{"type":"tool_result","tool_use_id":"tu1","content":"the word quarry appears only here"}]}}`,
		resultLine,
	}
	outsideOnly := []string{
		systemInitLine,
		`{"type":"assistant","uuid":"a1","timestamp":"2026-01-01T00:00:01Z","message":{"id":"msg_1","model":"claude-x","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"content":[{"type":"text","text":"quarry appears only in my own prose here"}]}}`,
		resultLine,
	}

	tests := []struct {
		name                      string
		lines                     []string
		tokenInTargetTrackedFiles bool
		tokenInAutoLoadedContext  bool
	}{
		{"inside_falsefalse", insideOnly, false, false},
		{"inside_truefalse", insideOnly, true, false},
		{"inside_falsetrue", insideOnly, false, true},
		{"inside_truetrue", insideOnly, true, true},
		{"outside_falsefalse", outsideOnly, false, false},
		{"outside_truefalse", outsideOnly, true, false},
		{"outside_falsetrue", outsideOnly, false, true},
		{"outside_truetrue", outsideOnly, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcript := parseTranscriptFromLines(t, tt.lines...)
			findings := CheckBlinding(transcript, BlindingInput{
				MCPPrefix:                 "mcp__absent__",
				QuarryRepoRoot:            "/absent-repo-root",
				TokenInTargetTrackedFiles: tt.tokenInTargetTrackedFiles,
				TokenInAutoLoadedContext:  tt.tokenInAutoLoadedContext,
			})
			for _, f := range findings {
				if f.Fatal {
					t.Errorf("CheckBlinding() finding %+v is fatal, want check (c) to never be fatal", f)
				}
			}
		})
	}
}

func TestCheckRenderedControlPrompt(t *testing.T) {
	tc, err := LoadTaskFile("../../../tasks/01-reed-geometry-exploration.md")
	if err != nil {
		t.Fatalf("LoadTaskFile() error = %v", err)
	}
	prompt := RenderPrompt(tc, "/tmp/target-dir", BuiltinTools)

	// The server name is deliberately not the word "quarry", so a case relying on it fails if the
	// implementation re-derives the name from the prefix or leans on a hardcoded default instead
	// of reading the supplied field.
	in := BlindingInput{ServerName: "widget", MCPPrefix: "mcp__widget__"}
	quarryTools := []string{"toc"}

	tests := []struct {
		name      string
		prompt    string
		wantFatal bool
	}{
		{"contains_bare_quarry", prompt + "\n\nquarry", true},
		{"contains_tool_token", prompt + "\n\ntoc", true},
		{"contains_mcp_prefix", prompt + "\n\nmcp__widget__toc", true},
		{"contains_bare_server_name", prompt + "\n\nwidget", true},
		{"real_control_prompt_passes", prompt, false},
		{"contains_protocol_passes", prompt + "\n\nprotocol", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckRenderedControlPrompt(tt.prompt, in, quarryTools)
			if tt.wantFatal && (got == nil || !got.Fatal) {
				t.Errorf("CheckRenderedControlPrompt() = %+v, want a fatal finding", got)
			}
			if !tt.wantFatal && got != nil {
				t.Errorf("CheckRenderedControlPrompt() = %+v, want nil", got)
			}
		})
	}
}

func TestCheckGrantedToolUsed(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		perRepUses  []int
		wantFinding bool
	}{
		{
			name:        "granted_cell_all_reps_zero",
			cfg:         Config{ID: "a2-toc-dir", Allowed: []string{"toc"}},
			perRepUses:  []int{0, 0, 0},
			wantFinding: true,
		},
		{
			name:        "granted_cell_one_rep_nonzero",
			cfg:         Config{ID: "a2-toc-dir", Allowed: []string{"toc"}},
			perRepUses:  []int{0, 3, 0},
			wantFinding: false,
		},
		{
			name:        "control_cell_zero_uses",
			cfg:         Config{ID: "a0-none", Allowed: nil},
			perRepUses:  []int{0, 0, 0},
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckGrantedToolUsed(tt.cfg, tt.perRepUses)
			if tt.wantFinding {
				if got == nil {
					t.Fatalf("CheckGrantedToolUsed() = nil, want a non-fatal finding")
				}
				if got.Fatal {
					t.Errorf("CheckGrantedToolUsed() finding.Fatal = true, want false")
				}
				wantMessage := "!! " + tt.cfg.ID + ": tool-granted config whose agent never called a granted tool in any repetition -- this cell measures the tool's prompt cost, not the tool"
				if got.Message != wantMessage {
					t.Errorf("CheckGrantedToolUsed() finding.Message = %q, want %q", got.Message, wantMessage)
				}
			} else if got != nil {
				t.Errorf("CheckGrantedToolUsed() = %+v, want nil", got)
			}
		})
	}
}

func TestCheckWorktreeDirtied(t *testing.T) {
	if got := CheckWorktreeDirtied(""); got != nil {
		t.Errorf("CheckWorktreeDirtied(\"\") = %+v, want nil", got)
	}
	got := CheckWorktreeDirtied(" M some/file.go\n")
	if got == nil {
		t.Fatalf("CheckWorktreeDirtied(non-empty) = nil, want a non-fatal finding")
	}
	if got.Fatal {
		t.Errorf("CheckWorktreeDirtied() finding.Fatal = true, want false")
	}
}
