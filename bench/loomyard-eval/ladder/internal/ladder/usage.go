// usage.go ports scripts/extract_usage.py's extract_usage onto the subagent transcript format:
// summing token classes and counting tool uses across a run's Record stream rather than reading the
// retired claude -p result envelope's final-iteration-only usage object (see the plan's "token classes
// are summed from assistant events" Shared Decision), keeping bash_grep_count and grep_tool_count in
// the exact separate shape #006's own definitions used.

package ladder

import (
	"regexp"
	"strings"
	"time"
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

// DenialShapePattern is the case-insensitive regex ExtractUsage matches a tool_result block's text
// against to decide whether an is_error result was a permission denial. It is unvalidated against a
// real denial record, because nothing this task dispatches provokes a genuine permission-denied tool
// result -- that is exactly what DeniedToolAttemptsProvisional exists to flag on every Usage this port
// produces. The follow-up matrix task's deny-list probe is what clears it, by running an actual denied
// dispatch and confirming this pattern still matches the real shape.
const DenialShapePattern = `(?i)(permission denied|not permitted|access denied|denied by (the )?user)`

// denialShapeRe is DenialShapePattern compiled once at package init.
var denialShapeRe = regexp.MustCompile(DenialShapePattern)

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
	// DurationMs is the last record's Timestamp minus the first's, in milliseconds. The retired
	// claude -p port read this off the result envelope's own duration_ms; a subagent transcript has no
	// such envelope, so it is derived from the record stream instead.
	DurationMs int64 `json:"duration_ms"`
	// NumTurns is the count of assistant records. The retired port read this off the result envelope's
	// own num_turns; here it is counted directly, since assistant records are the turns.
	NumTurns int `json:"num_turns"`
	// Model is the model id carried on the assistant records' message.model. The retired port read
	// this off the system/init event, which a subagent transcript does not have.
	Model string `json:"model"`
	// Effort is the reasoning-effort level carried on the assistant records' top-level effort field.
	// This field has no counterpart in the retired claude -p port.
	Effort string `json:"effort"`
	// AgentID identifies the dispatched subagent this Usage was extracted from. This field has no
	// counterpart in the retired claude -p port, which captured one transcript per top-level
	// invocation and so had no agent id to carry.
	AgentID string `json:"agent_id"`
	// TranscriptSource names where the transcript this Usage was extracted from actually came from --
	// distinct from Transcript, which is the path to the copy committed inside the run directory. This
	// field has no counterpart in the retired claude -p port.
	TranscriptSource string `json:"transcript_source"`
	// GrantedTools is the caller-supplied list of tools the run's generated agent definition exposed.
	// Renamed from the retired port's advertised_tools and populated from ExtractUsage's grantedTools
	// parameter rather than read off the transcript, preserving the "extracted, never self-reported"
	// rule: the value still comes from a harness-generated file, just not this one.
	GrantedTools []string `json:"granted_tools"`
	// DeniedToolAttempts is the count of tool_result blocks with IsError set whose text matches
	// DenialShapePattern.
	DeniedToolAttempts int `json:"denied_tool_attempts"`
	// DeniedToolAttemptsProvisional is always true in this port. It flags that DeniedToolAttempts and
	// DenialShapePattern are unvalidated against a real permission-denial tool result, because nothing
	// this task dispatches provokes one; see DenialShapePattern's doc comment.
	DeniedToolAttemptsProvisional bool `json:"denied_tool_attempts_provisional"`
}

// ExtractUsage builds the usage.json document for one run from its parsed transcript records,
// transcriptPath (the copy inside the run directory), transcriptSource (where that copy actually came
// from), and grantedTools (the tool list the run's agent definition exposed).
func ExtractUsage(records []Record, transcriptPath, transcriptSource string, grantedTools []string) (Usage, error) {
	usage := Usage{
		ToolUsesBreakdown:             map[string]int{},
		Transcript:                    transcriptPath,
		TranscriptSource:              transcriptSource,
		GrantedTools:                  grantedTools,
		DeniedToolAttemptsProvisional: true,
	}

	assistantRecords := AssistantRecords(records)
	usage.NumTurns = len(assistantRecords)
	for _, record := range assistantRecords {
		recordUsage := record.Message.Usage
		usage.Tokens.InputTokens += recordUsage.InputTokens
		usage.Tokens.OutputTokens += recordUsage.OutputTokens
		usage.Tokens.CacheReadInputTokens += recordUsage.CacheReadInputTokens
		usage.Tokens.CacheCreationInputTokens += recordUsage.CacheCreationInputTokens
		if usage.Model == "" && record.Message.Model != "" {
			usage.Model = record.Message.Model
		}
		if usage.Effort == "" && record.Effort != "" {
			usage.Effort = record.Effort
		}
	}

	if len(records) > 0 {
		usage.AgentID = records[0].AgentID
		if duration, ok := recordSpanMs(records[0], records[len(records)-1]); ok {
			usage.DurationMs = duration
		}
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
	usage.DeniedToolAttempts = countDeniedToolAttempts(records)

	return usage, nil
}

// recordSpanMs returns the duration between first and last's Timestamp fields in milliseconds, and
// whether both parsed successfully as RFC 3339.
func recordSpanMs(first, last Record) (int64, bool) {
	firstTime, err := time.Parse(time.RFC3339Nano, first.Timestamp)
	if err != nil {
		return 0, false
	}
	lastTime, err := time.Parse(time.RFC3339Nano, last.Timestamp)
	if err != nil {
		return 0, false
	}
	return lastTime.Sub(firstTime).Milliseconds(), true
}

// countDeniedToolAttempts counts every tool_result content block, across every record, whose IsError is
// set and whose flattened text matches DenialShapePattern.
func countDeniedToolAttempts(records []Record) int {
	count := 0
	for _, record := range records {
		for _, block := range record.Message.Content {
			if block.Type != "tool_result" || !block.IsError {
				continue
			}
			if denialShapeRe.MatchString(toolResultText(block)) {
				count++
			}
		}
	}
	return count
}

// toolResultText concatenates the text of every "text" block in a tool_result content block's own
// nested Content, in order.
func toolResultText(block ContentBlock) string {
	var text strings.Builder
	for _, inner := range block.Content {
		if inner.Type == "text" {
			text.WriteString(inner.Text)
		}
	}
	return text.String()
}
