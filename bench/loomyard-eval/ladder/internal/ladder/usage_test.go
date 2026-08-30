package ladder

import "testing"

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

	usage, err := ExtractUsage(records, "run/transcript.jsonl", nil)
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

	usage, err := ExtractUsage(records, "run/transcript.jsonl", nil)
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

	usage, err := ExtractUsage(records, "run/transcript.jsonl", nil)
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

	usage, err := ExtractUsage(records, "run/transcript.jsonl", nil)
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
