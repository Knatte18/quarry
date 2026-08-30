// usage.go ports scripts/extract_usage.py's extract_usage onto the subagent transcript format:
// summing token classes and counting tool uses across a run's Record stream rather than reading the
// retired claude -p result envelope's final-iteration-only usage object (see the plan's "token classes
// are summed from assistant events" Shared Decision), keeping bash_grep_count and grep_tool_count in
// the exact separate shape #006's own definitions used.

package ladder

import (
	"regexp"
	"strings"
)

// bashGrepRe matches a Bash tool call's "command" string invoking grep or ripgrep as a leading command
// word -- not merely containing the substring "grep" somewhere unrelated (e.g. inside a path). #006's
// own definition (README "Dispatch protocol" step 4) greps the transcript's Bash "command" fields for
// this shape, so BashGrepCount is held to it exactly.
var bashGrepRe = regexp.MustCompile(`(^|[|&;\s])(grep|rg)\b`)

// isBashGrepCommand reports whether a Bash tool call's command string invokes grep or rg as a command
// word, matching #006's exact Bash-only grep-fallback definition.
func isBashGrepCommand(command string) bool {
	return bashGrepRe.MatchString(command)
}

// TokenUsage is the four independently-summed token classes carried on a run's Usage.
type TokenUsage struct {
	// InputTokens is the sum of every assistant record's message.usage.input_tokens.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is the sum of every assistant record's message.usage.output_tokens.
	OutputTokens int `json:"output_tokens"`
	// CacheReadInputTokens is the sum of every assistant record's message.usage.cache_read_input_tokens.
	CacheReadInputTokens int `json:"cache_read_input_tokens"`
	// CacheCreationInputTokens is the sum of every assistant record's
	// message.usage.cache_creation_input_tokens.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// Usage is the usage.json document ExtractUsage builds for one run. Its field set intentionally
// differs from the retired claude -p port: fields the result envelope used to carry
// (cost_usd, wall_clock_ms, result_usage, result_subtype, result_is_error, session_id) have no
// counterpart, because the subagent transcript has no terminal result record to read them from.
type Usage struct {
	// Tokens holds the four token classes, each summed independently across every assistant record's
	// message.usage -- none derived from another.
	Tokens TokenUsage `json:"tokens"`
	// ToolUses is the total count of tool_use content blocks across the transcript.
	ToolUses int `json:"tool_uses"`
	// ToolUsesBreakdown maps tool name to the number of times it was called.
	ToolUsesBreakdown map[string]int `json:"tool_uses_breakdown"`
	// QuarryToolUses is the count of tool uses whose name carries the MCPPrefix prefix.
	QuarryToolUses int `json:"quarry_tool_uses"`
	// BashGrepCount is the count of Bash tool calls whose command matches bashGrepRe. Counted strictly
	// separately from GrepToolCount and never merged with it.
	BashGrepCount int `json:"bash_grep_count"`
	// GrepToolCount is the count of native Grep tool calls. Counted strictly separately from
	// BashGrepCount and never merged with it.
	GrepToolCount int `json:"grep_tool_count"`
	// GrepFallbackTotal is BashGrepCount + GrepToolCount. It is their sum for reporting purposes only
	// and is never substituted for either individual count.
	GrepFallbackTotal int `json:"grep_fallback_total"`
	// Transcript is the path to the run's transcript, naming the copy inside the run directory rather
	// than a harness-captured file -- the session that dispatched the subagent is what wrote it there.
	Transcript string `json:"transcript"`
}

// ExtractUsage builds the usage.json document for one run from its parsed transcript records,
// transcriptPath (the copy inside the run directory), and grantedTools (the tool list the run's agent
// definition exposed).
func ExtractUsage(records []Record, transcriptPath string, grantedTools []string) (Usage, error) {
	usage := Usage{
		ToolUsesBreakdown: map[string]int{},
		Transcript:        transcriptPath,
	}

	for _, record := range AssistantRecords(records) {
		recordUsage := record.Message.Usage
		usage.Tokens.InputTokens += recordUsage.InputTokens
		usage.Tokens.OutputTokens += recordUsage.OutputTokens
		usage.Tokens.CacheReadInputTokens += recordUsage.CacheReadInputTokens
		usage.Tokens.CacheCreationInputTokens += recordUsage.CacheCreationInputTokens
	}

	for _, use := range IterToolUses(records) {
		usage.ToolUsesBreakdown[use.Name]++
		if use.Name == "Bash" {
			if command, ok := use.Input["command"].(string); ok && isBashGrepCommand(command) {
				usage.BashGrepCount++
			}
		} else if use.Name == "Grep" {
			usage.GrepToolCount++
		}
		if strings.HasPrefix(use.Name, MCPPrefix) {
			usage.QuarryToolUses++
		}
	}

	for _, count := range usage.ToolUsesBreakdown {
		usage.ToolUses += count
	}
	usage.GrepFallbackTotal = usage.BashGrepCount + usage.GrepToolCount

	return usage, nil
}
