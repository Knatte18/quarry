package ladder

import (
	"bytes"
	"os"
	"testing"
)

// parseTranscriptFixture parses a fixture transcript from testdata/transcripts, failing the test
// on any parse error.
func parseTranscriptFixture(t *testing.T, name string) *Transcript {
	t.Helper()
	data, err := os.ReadFile("testdata/transcripts/" + name)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}
	transcript, err := ParseTranscript(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseTranscript(%s) error = %v", name, err)
	}
	return transcript
}

// TestComputeMetrics_GroupedUsage asserts the message.id grouping rule: two assistant records
// sharing one message id sum their input/cache figures once and take the maximum output_tokens,
// while a second group with its own id adds independently.
func TestComputeMetrics_GroupedUsage(t *testing.T) {
	transcript := parseTranscriptFixture(t, "grouped-usage.jsonl")
	m := ComputeMetrics(transcript, "mcp__quarry__")

	wantInput := 300       // 100 (group msg_1, once) + 200 (group msg_2)
	wantOutput := 80       // max(10, 50) + 30
	wantCacheRead := 15    // 5 + 10
	wantCacheCreation := 5 // 2 + 3
	wantInputTotal := 320  // 300 + 15 + 5

	if m.InputTokens != wantInput {
		t.Errorf("InputTokens = %d, want %d", m.InputTokens, wantInput)
	}
	if m.OutputTokens != wantOutput {
		t.Errorf("OutputTokens = %d, want %d", m.OutputTokens, wantOutput)
	}
	if m.CacheReadInputTokens != wantCacheRead {
		t.Errorf("CacheReadInputTokens = %d, want %d", m.CacheReadInputTokens, wantCacheRead)
	}
	if m.CacheCreationInputTokens != wantCacheCreation {
		t.Errorf("CacheCreationInputTokens = %d, want %d", m.CacheCreationInputTokens, wantCacheCreation)
	}
	if m.InputTokensTotal != wantInputTotal {
		t.Errorf("InputTokensTotal = %d, want %d", m.InputTokensTotal, wantInputTotal)
	}

	// A naive per-record sum over the msg_1 group would give 200, not 100 -- confirm the grouping
	// rule actually collapsed the duplicate.
	naiveInputSum := 100 + 100 + 200
	if m.InputTokens == naiveInputSum {
		t.Errorf("InputTokens = %d equals the naive per-record sum %d; grouping did not collapse the duplicate", m.InputTokens, naiveInputSum)
	}
}

// TestComputeMetrics_MaxTurnsTerminalReason asserts that a max-turns rep's terminal reason
// survives into the computed Metrics.
func TestComputeMetrics_MaxTurnsTerminalReason(t *testing.T) {
	transcript := parseTranscriptFixture(t, "max-turns.jsonl")
	m := ComputeMetrics(transcript, "mcp__quarry__")

	if m.TerminalReason != "max_turns" {
		t.Errorf("TerminalReason = %q, want %q", m.TerminalReason, "max_turns")
	}
}

// TestComputeMetrics_ZeroToolCallsReportsZeroes asserts that a transcript with no tool calls at
// all reports zero-valued tool metrics rather than absent fields.
func TestComputeMetrics_ZeroToolCallsReportsZeroes(t *testing.T) {
	transcript := parseTranscriptFixture(t, "max-turns.jsonl")
	m := ComputeMetrics(transcript, "mcp__quarry__")

	if m.ToolUses != 0 {
		t.Errorf("ToolUses = %d, want 0", m.ToolUses)
	}
	if m.QuarryToolUses != 0 {
		t.Errorf("QuarryToolUses = %d, want 0", m.QuarryToolUses)
	}
	if m.GrepToolCount != 0 {
		t.Errorf("GrepToolCount = %d, want 0", m.GrepToolCount)
	}
	if m.BashGrepCount != 0 {
		t.Errorf("BashGrepCount = %d, want 0", m.BashGrepCount)
	}
	if m.ToolResultBytes != 0 {
		t.Errorf("ToolResultBytes = %d, want 0", m.ToolResultBytes)
	}
	if m.ToolUsesBreakdown == nil {
		t.Errorf("ToolUsesBreakdown = nil, want a non-nil empty map")
	}
	if m.ToolResultBytesBreakdown == nil {
		t.Errorf("ToolResultBytesBreakdown = nil, want a non-nil empty map")
	}
}

// TestComputeMetrics_ToolBytesAndGrepCounts asserts the byte metrics, the Read subset, and the
// grep-fallback leading-command-word distinction, all against the tool-bytes fixture.
func TestComputeMetrics_ToolBytesAndGrepCounts(t *testing.T) {
	transcript := parseTranscriptFixture(t, "tool-bytes.jsonl")
	m := ComputeMetrics(transcript, "mcp__quarry__")

	if m.GrepToolCount != 1 {
		t.Errorf("GrepToolCount = %d, want 1", m.GrepToolCount)
	}
	if m.BashGrepCount != 2 {
		t.Errorf("BashGrepCount = %d, want 2 (leading-word grep and the piped cat|grep, not the ripgrepping substring)", m.BashGrepCount)
	}
	if m.GrepFallbackTotal != m.GrepToolCount+m.BashGrepCount {
		t.Errorf("GrepFallbackTotal = %d, want GrepToolCount+BashGrepCount = %d", m.GrepFallbackTotal, m.GrepToolCount+m.BashGrepCount)
	}

	wantToolResultBytes := len("0123456789") + len("abcdefghijklmno") + len("hello") + len("abc") + len("wxyz") + len("abcdef")
	if m.ToolResultBytes != wantToolResultBytes {
		t.Errorf("ToolResultBytes = %d, want %d", m.ToolResultBytes, wantToolResultBytes)
	}

	wantReadBytes := len("0123456789")
	if m.ReadBytes != wantReadBytes {
		t.Errorf("ReadBytes = %d, want %d", m.ReadBytes, wantReadBytes)
	}

	if got, want := m.ToolResultBytesBreakdown["Read"], len("0123456789"); got != want {
		t.Errorf("ToolResultBytesBreakdown[Read] = %d, want %d", got, want)
	}
	if got, want := m.ToolResultBytesBreakdown["Grep"], len("abcdefghijklmno"); got != want {
		t.Errorf("ToolResultBytesBreakdown[Grep] = %d, want %d", got, want)
	}
	wantBashBytes := len("hello") + len("abc") + len("wxyz")
	if got := m.ToolResultBytesBreakdown["Bash"]; got != wantBashBytes {
		t.Errorf("ToolResultBytesBreakdown[Bash] = %d, want %d", got, wantBashBytes)
	}
}

// TestComputeMetrics_QuarryToolUsesPrefixIsAnArgument asserts that QuarryToolUses is counted under
// the mcpPrefix argument rather than a hardcoded literal: a real prefix finds the one mcp tool use
// in the tool-bytes fixture, and a different prefix that matches nothing proves the value is not
// hardcoded to the real one.
func TestComputeMetrics_QuarryToolUsesPrefixIsAnArgument(t *testing.T) {
	transcript := parseTranscriptFixture(t, "tool-bytes.jsonl")

	under := ComputeMetrics(transcript, "mcp__quarry__")
	if under.QuarryToolUses != 1 {
		t.Errorf("QuarryToolUses under mcp__quarry__ = %d, want 1", under.QuarryToolUses)
	}

	underOther := ComputeMetrics(transcript, "mcp__other__")
	if underOther.QuarryToolUses != 0 {
		t.Errorf("QuarryToolUses under mcp__other__ = %d, want 0 -- the prefix must not be hardcoded", underOther.QuarryToolUses)
	}
}

// TestComputeMetrics_ResultRecordFields asserts that the result record's own fields (num_turns,
// duration_ms, duration_api_ms, total_cost_usd, stop_reason, is_error, permission_denials) copy
// through unmodified rather than being reconstructed from timestamps.
func TestComputeMetrics_ResultRecordFields(t *testing.T) {
	transcript := parseTranscriptFixture(t, "tool-bytes.jsonl")
	m := ComputeMetrics(transcript, "mcp__quarry__")

	if m.NumTurns != 6 {
		t.Errorf("NumTurns = %d, want 6", m.NumTurns)
	}
	if m.DurationMS != 2000 {
		t.Errorf("DurationMS = %d, want 2000", m.DurationMS)
	}
	if m.DurationAPIMS != 1500 {
		t.Errorf("DurationAPIMS = %d, want 1500", m.DurationAPIMS)
	}
	if m.TotalCostUSD != 0.1 {
		t.Errorf("TotalCostUSD = %v, want 0.1", m.TotalCostUSD)
	}
	if m.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", m.StopReason, "end_turn")
	}
	if m.IsError {
		t.Errorf("IsError = true, want false")
	}
	if m.PermissionDenialsCount != 0 {
		t.Errorf("PermissionDenialsCount = %d, want 0", m.PermissionDenialsCount)
	}
	if m.Model != "claude-x" {
		t.Errorf("Model = %q, want %q", m.Model, "claude-x")
	}
}
