package ladder

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsBashGrepCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"leading grep", "grep -n Foo internal/foo.go", true},
		{"leading rg", "rg Foo internal/", true},
		{"piped grep", "cat foo.go | grep Foo", true},
		{"grep after semicolon", "go build ./...; grep -n Foo .", true},
		{"grep inside a path is not a command word", "cat /var/grep-logs/output.txt", false},
		{"unrelated command", "go build ./...", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBashGrepCommand(tt.command)
			if got != tt.want {
				t.Errorf("isBashGrepCommand(%q) = %v; want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestExtractUsage_SumsTokenClassesIndependently(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	want := TokenUsage{InputTokens: 300, OutputTokens: 130, CacheReadInputTokens: 30, CacheCreationInputTokens: 20}
	if usage.Tokens != want {
		t.Errorf("usage.Tokens = %+v; want %+v", usage.Tokens, want)
	}
	seen := map[int]bool{
		usage.Tokens.InputTokens:              true,
		usage.Tokens.OutputTokens:             true,
		usage.Tokens.CacheReadInputTokens:     true,
		usage.Tokens.CacheCreationInputTokens: true,
	}
	if len(seen) != 4 {
		t.Errorf("token classes collided into fewer than 4 distinct values: %+v", usage.Tokens)
	}
}

func TestExtractUsage_ToolUsesBreakdownAndQuarryCount(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	if usage.ToolUses != 8 {
		t.Errorf("usage.ToolUses = %d; want 8", usage.ToolUses)
	}
	if usage.QuarryToolUses != 2 {
		t.Errorf("usage.QuarryToolUses = %d; want 2", usage.QuarryToolUses)
	}
	wantBreakdown := map[string]int{
		"mcp__quarry__toc_file":         1,
		"mcp__quarry__workspace_symbol": 1,
		"Read":                          2,
		"Grep":                          1,
		"Bash":                          3,
	}
	if len(usage.ToolUsesBreakdown) != len(wantBreakdown) {
		t.Fatalf("usage.ToolUsesBreakdown = %+v; want %+v", usage.ToolUsesBreakdown, wantBreakdown)
	}
	for name, count := range wantBreakdown {
		if usage.ToolUsesBreakdown[name] != count {
			t.Errorf("usage.ToolUsesBreakdown[%q] = %d; want %d", name, usage.ToolUsesBreakdown[name], count)
		}
	}
}

func TestExtractUsage_BashGrepAndGrepToolCountsStaySeparate(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	// Two Bash calls match ("grep -n ..." and "rg ..."); the native Grep tool call and the unrelated
	// "go build ./..." Bash call must not move it.
	if usage.BashGrepCount != 2 {
		t.Errorf("usage.BashGrepCount = %d; want 2", usage.BashGrepCount)
	}
	if usage.GrepToolCount != 1 {
		t.Errorf("usage.GrepToolCount = %d; want 1", usage.GrepToolCount)
	}
	if usage.GrepFallbackTotal != usage.BashGrepCount+usage.GrepToolCount {
		t.Errorf("usage.GrepFallbackTotal = %d; want BashGrepCount + GrepToolCount = %d", usage.GrepFallbackTotal, usage.BashGrepCount+usage.GrepToolCount)
	}
	if usage.GrepFallbackTotal == usage.BashGrepCount || usage.GrepFallbackTotal == usage.GrepToolCount {
		t.Errorf("usage.GrepFallbackTotal = %d must never equal either component (%d, %d)", usage.GrepFallbackTotal, usage.BashGrepCount, usage.GrepToolCount)
	}
}

func TestExtractUsage_ZeroToolCallsYieldsZeroCountsAndEmptyBreakdown(t *testing.T) {
	records, err := ReadTranscript("testdata/zero-tool-calls.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	if usage.ToolUses != 0 {
		t.Errorf("usage.ToolUses = %d; want 0", usage.ToolUses)
	}
	if len(usage.ToolUsesBreakdown) != 0 {
		t.Errorf("usage.ToolUsesBreakdown = %+v; want empty", usage.ToolUsesBreakdown)
	}
	if usage.QuarryToolUses != 0 || usage.BashGrepCount != 0 || usage.GrepToolCount != 0 || usage.GrepFallbackTotal != 0 {
		t.Errorf("usage = %+v; want all tool-use counters zero", usage)
	}
}

func TestExtractUsage_DurationIsDerivedFromFirstAndLastTimestamp(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	// Fixture spans 2026-08-30T09:00:00.000Z to 2026-08-30T09:00:10.000Z.
	if usage.DurationMs != 10000 {
		t.Errorf("usage.DurationMs = %d; want 10000", usage.DurationMs)
	}
}

func TestExtractUsage_NumTurnsCountsAssistantRecords(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	// Fixture carries three assistant records.
	if usage.NumTurns != 3 {
		t.Errorf("usage.NumTurns = %d; want 3", usage.NumTurns)
	}
	if usage.Model != "claude-opus-5" {
		t.Errorf("usage.Model = %q; want %q", usage.Model, "claude-opus-5")
	}
	if usage.Effort != "medium" {
		t.Errorf("usage.Effort = %q; want %q", usage.Effort, "medium")
	}
	if usage.AgentID != "agent-a5-bundle-1" {
		t.Errorf("usage.AgentID = %q; want %q", usage.AgentID, "agent-a5-bundle-1")
	}
}

func TestExtractUsage_DeniedAttemptFixtureCountsOneDenial(t *testing.T) {
	records, err := ReadTranscript("testdata/denied-attempt.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	if usage.DeniedToolAttempts != 1 {
		t.Errorf("usage.DeniedToolAttempts = %d; want 1", usage.DeniedToolAttempts)
	}
	if !usage.DeniedToolAttemptsProvisional {
		t.Error("usage.DeniedToolAttemptsProvisional = false; want true")
	}
}

func TestExtractUsage_ErroredToolResultFixtureCountsZeroDenials(t *testing.T) {
	// The errored-tool-result fixture's is_error tool result is a plain "file not found" error, not
	// permission-denial shaped -- it must not be counted as a denied attempt.
	records, err := ReadTranscript("testdata/errored-tool-result.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	if usage.DeniedToolAttempts != 0 {
		t.Errorf("usage.DeniedToolAttempts = %d; want 0", usage.DeniedToolAttempts)
	}
}

func TestExtractUsage_SerialisedJSONCarriesNoDroppedField(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	usage, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", nil)
	if err != nil {
		t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
	}

	encoded, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("json.Marshal(usage) = _, %v; want nil error", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded) = %v; want nil error", err)
	}

	for _, dropped := range []string{"cost_usd", "wall_clock_ms", "result_usage", "result_subtype", "result_is_error", "session_id"} {
		if _, present := decoded[dropped]; present {
			t.Errorf("serialised Usage carries dropped field %q; want it absent", dropped)
		}
	}
}

func TestExtractUsage_EndToEndOverReshapedFixtures(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		grantedTools []string
		want         Usage
	}{
		{
			name:         "bundle-mixed-tools",
			fixture:      "bundle-mixed-tools.jsonl",
			grantedTools: []string{"toc_file", "workspace_symbol", "impact"},
			want: Usage{
				Tokens:                        TokenUsage{InputTokens: 300, OutputTokens: 130, CacheReadInputTokens: 30, CacheCreationInputTokens: 20},
				ToolUses:                      8,
				ToolUsesBreakdown:             map[string]int{"mcp__quarry__toc_file": 1, "mcp__quarry__workspace_symbol": 1, "Read": 2, "Grep": 1, "Bash": 3},
				QuarryToolUses:                2,
				BashGrepCount:                 2,
				GrepToolCount:                 1,
				GrepFallbackTotal:             3,
				Transcript:                    "run/transcript.jsonl",
				DurationMs:                    10000,
				NumTurns:                      3,
				Model:                         "claude-opus-5",
				Effort:                        "medium",
				AgentID:                       "agent-a5-bundle-1",
				TranscriptSource:              "agent-1/agent-1.jsonl",
				GrantedTools:                  []string{"toc_file", "workspace_symbol", "impact"},
				DeniedToolAttempts:            0,
				DeniedToolAttemptsProvisional: true,
			},
		},
		{
			// The zero-tool-calls fixture's granted tools list intentionally names tools that never
			// appear as a tool_use in the transcript, proving GrantedTools reflects the parameter
			// rather than anything the transcript itself carries.
			name:         "zero-tool-calls",
			fixture:      "zero-tool-calls.jsonl",
			grantedTools: []string{"toc_dir", "toc_file"},
			want: Usage{
				Tokens:                        TokenUsage{InputTokens: 50, OutputTokens: 20, CacheReadInputTokens: 0, CacheCreationInputTokens: 0},
				ToolUses:                      0,
				ToolUsesBreakdown:             map[string]int{},
				QuarryToolUses:                0,
				BashGrepCount:                 0,
				GrepToolCount:                 0,
				GrepFallbackTotal:             0,
				Transcript:                    "run/transcript.jsonl",
				DurationMs:                    0,
				NumTurns:                      1,
				Model:                         "claude-opus-5",
				Effort:                        "medium",
				AgentID:                       "agent-zero-1",
				TranscriptSource:              "agent-1/agent-1.jsonl",
				GrantedTools:                  []string{"toc_dir", "toc_file"},
				DeniedToolAttempts:            0,
				DeniedToolAttemptsProvisional: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := ReadTranscript("testdata/" + tt.fixture)
			if err != nil {
				t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", tt.fixture, err)
			}

			got, err := ExtractUsage(records, "run/transcript.jsonl", "agent-1/agent-1.jsonl", tt.grantedTools)
			if err != nil {
				t.Fatalf("ExtractUsage() = _, %v; want nil error", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExtractUsage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
