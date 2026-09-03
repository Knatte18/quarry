// metrics.go computes the per-rep accounting numbers this task's downstream consumers read, from
// a parsed Transcript. It ports V1's assistantCallGroups and perCallUsage grouping rule (see
// origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/usage.go) onto the stream-json record
// shapes stream.go declares, and adds the byte metrics stream.go's V1 counterpart never had.

package ladder

import (
	"encoding/json"
	"regexp"
	"strings"
)

// bashGrepRe matches a Bash tool call's "command" string invoking grep or ripgrep as a leading
// command word — not merely containing the substring "grep" somewhere unrelated (e.g. inside
// another word). Copied verbatim from V1's usage.go, per this batch's requirement that the grep
// regex not be re-derived.
var bashGrepRe = regexp.MustCompile(`(^|[|&;\s])(grep|rg)\b`)

// Metrics is the computed accounting for one rep: the result record's own fields, token counts
// summed over assistant records grouped by message id, tool-use counts and breakdowns, and the
// tool-result byte metrics.
type Metrics struct {
	// NumTurns is the result record's own reported turn count.
	NumTurns int `json:"num_turns"`
	// DurationMS is the result record's own wall-clock duration in milliseconds.
	DurationMS int64 `json:"duration_ms"`
	// DurationAPIMS is the result record's own cumulative API-call duration in milliseconds.
	DurationAPIMS int64 `json:"duration_api_ms"`
	// TotalCostUSD is the result record's own total measured cost.
	TotalCostUSD float64 `json:"total_cost_usd"`
	// TerminalReason is the result record's own terminal reason, e.g. a max-turns ceiling.
	TerminalReason string `json:"terminal_reason"`
	// StopReason is the result record's own model stop reason.
	StopReason string `json:"stop_reason"`
	// IsError is the result record's own error flag.
	IsError bool `json:"is_error"`
	// PermissionDenialsCount is the number of entries in PermissionDenials.
	PermissionDenialsCount int `json:"permission_denials_count"`
	// PermissionDenials is the result record's own raw permission-denial entries.
	PermissionDenials []json.RawMessage `json:"permission_denials"`

	// InputTokens is the sum, across API calls (assistant records grouped by message id), of each
	// call's own input_tokens.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is the sum, across API calls, of each call's maximum output_tokens snapshot —
	// never the final record's snapshot, which does not reliably carry the final count.
	OutputTokens int `json:"output_tokens"`
	// CacheReadInputTokens is the sum, across API calls, of each call's own
	// cache_read_input_tokens.
	CacheReadInputTokens int `json:"cache_read_input_tokens"`
	// CacheCreationInputTokens is the sum, across API calls, of each call's own
	// cache_creation_input_tokens.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	// InputTokensTotal is InputTokens + CacheReadInputTokens + CacheCreationInputTokens, reported
	// alongside the three token classes and never in place of them.
	InputTokensTotal int `json:"input_tokens_total"`

	// ToolUses is the total count of tool_use content blocks across the transcript.
	ToolUses int `json:"tool_uses"`
	// ToolUsesBreakdown maps tool name to the number of times it was called.
	ToolUsesBreakdown map[string]int `json:"tool_uses_breakdown"`
	// QuarryToolUses is the count of tool uses whose name carries the mcpPrefix argument as its
	// prefix.
	QuarryToolUses int `json:"quarry_tool_uses"`
	// GrepToolCount is the count of native Grep tool uses. Counted strictly separately from
	// BashGrepCount.
	GrepToolCount int `json:"grep_tool_count"`
	// BashGrepCount is the count of Bash tool uses whose command matches bashGrepRe. Counted
	// strictly separately from GrepToolCount.
	BashGrepCount int `json:"bash_grep_count"`
	// GrepFallbackTotal is GrepToolCount + BashGrepCount, reported for convenience only and never
	// substituted for either individual count.
	GrepFallbackTotal int `json:"grep_fallback_total"`

	// ToolResultBytes is the total UTF-8 byte length of every tool_result block's text content.
	ToolResultBytes int `json:"tool_result_bytes"`
	// ToolResultBytesBreakdown is ToolResultBytes keyed by the tool name of the tool_use block the
	// result answers.
	ToolResultBytesBreakdown map[string]int `json:"tool_result_bytes_breakdown"`
	// ReadBytes is the Read subset of ToolResultBytes.
	ReadBytes int `json:"read_bytes"`

	// Model is the first assistant record's message model.
	Model string `json:"model"`
	// Effort is the reasoning-effort level the run loop passed on the command line.
	// ComputeMetrics leaves it empty; the run loop stamps it, since the CLI does not echo the
	// flag.
	Effort string `json:"effort"`
}

// apiCall is one API call's assistant records: one or more consecutive Record entries sharing a
// non-empty message id, or a single record with an empty id.
type apiCall []Record

// groupAssistantCalls groups consecutive assistant records sharing one non-empty message id into
// one API call each, porting V1's assistantCallGroups verbatim. A new group starts when the id is
// empty, differs from the previous id, or no group exists yet — Claude Code writes one transcript
// record per content block, every record of one API call repeating that call's usage snapshot
// under one message id, so grouping by consecutive equal id is what stops naive per-record summing
// from over-counting.
func groupAssistantCalls(records []Record) []apiCall {
	var groups []apiCall
	lastID := ""
	for _, record := range records {
		if record.Type != "assistant" {
			continue
		}
		id := record.Message.ID
		if id == "" || id != lastID || len(groups) == 0 {
			groups = append(groups, nil)
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], record)
		lastID = id
	}
	return groups
}

// callUsage reduces one API call's records to that call's own token usage, porting V1's
// perCallUsage verbatim: input and cache figures are constant across a call's records, so any
// record supplies them, while output_tokens grows as the call streams and is taken as the maximum
// across the call's records.
func callUsage(call apiCall) MessageUsage {
	reduced := call[len(call)-1].Message.Usage
	for _, record := range call {
		if record.Message.Usage.OutputTokens > reduced.OutputTokens {
			reduced.OutputTokens = record.Message.Usage.OutputTokens
		}
	}
	return reduced
}

// isBashGrepCommand reports whether a Bash tool call's command string invokes grep or rg as a
// leading command word.
func isBashGrepCommand(command string) bool {
	return bashGrepRe.MatchString(command)
}

// toolUseInput decodes a tool_use block's raw Input as a JSON object, tolerating a block with no
// Input at all (e.g. a tool taking no arguments).
func toolUseInput(block ContentBlock) map[string]any {
	if len(block.Input) == 0 {
		return nil
	}
	var input map[string]any
	if err := json.Unmarshal(block.Input, &input); err != nil {
		return nil
	}
	return input
}

// toolResultText flattens a tool_result block's Content into its plain text: Claude Code writes it
// as either a bare JSON string or a nested array of content blocks, of which only "text" blocks
// contribute.
func toolResultText(block ContentBlock) string {
	if len(block.Content) == 0 {
		return ""
	}
	trimmed := strings.TrimSpace(string(block.Content))
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(block.Content, &text); err == nil {
			return text
		}
		return ""
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(block.Content, &blocks); err != nil {
		return ""
	}
	var b strings.Builder
	for _, inner := range blocks {
		if inner.Type == "text" {
			b.WriteString(inner.Text)
		}
	}
	return b.String()
}

// ComputeMetrics reduces t to the per-rep Metrics its downstream consumers read. mcpPrefix is the
// tool-name prefix the MCP server registers its tools under (see Ladder.MCPPrefix) — it is taken
// as a parameter rather than a literal so no accounting rule hardcodes a server name.
func ComputeMetrics(t *Transcript, mcpPrefix string) Metrics {
	m := Metrics{
		ToolUsesBreakdown:        map[string]int{},
		ToolResultBytesBreakdown: map[string]int{},
	}

	if t.Result != nil {
		m.NumTurns = t.Result.NumTurns
		m.DurationMS = t.Result.DurationMS
		m.DurationAPIMS = t.Result.DurationAPIMS
		m.TotalCostUSD = t.Result.TotalCostUSD
		m.TerminalReason = t.Result.TerminalReason
		m.StopReason = t.Result.StopReason
		m.IsError = t.Result.IsError
		m.PermissionDenials = t.Result.PermissionDenials
		m.PermissionDenialsCount = len(t.Result.PermissionDenials)
	}

	for _, call := range groupAssistantCalls(t.Records) {
		usage := callUsage(call)
		m.InputTokens += usage.InputTokens
		m.OutputTokens += usage.OutputTokens
		m.CacheReadInputTokens += usage.CacheReadInputTokens
		m.CacheCreationInputTokens += usage.CacheCreationInputTokens
	}
	m.InputTokensTotal = m.InputTokens + m.CacheReadInputTokens + m.CacheCreationInputTokens

	for _, record := range t.Records {
		if record.Type == "assistant" && m.Model == "" && record.Message.Model != "" {
			m.Model = record.Message.Model
		}
	}

	// toolUseNames maps a tool_use block's id to its tool name, so a later tool_result block can
	// attribute its bytes to the tool that produced them.
	toolUseNames := map[string]string{}
	for _, record := range t.Records {
		for _, block := range record.Message.Content {
			if block.Type != "tool_use" {
				continue
			}
			toolUseNames[block.ID] = block.Name

			m.ToolUses++
			m.ToolUsesBreakdown[block.Name]++
			if strings.HasPrefix(block.Name, mcpPrefix) {
				m.QuarryToolUses++
			}
			switch block.Name {
			case "Grep":
				m.GrepToolCount++
			case "Bash":
				if input := toolUseInput(block); input != nil {
					if command, ok := input["command"].(string); ok && isBashGrepCommand(command) {
						m.BashGrepCount++
					}
				}
			}
		}
	}
	m.GrepFallbackTotal = m.GrepToolCount + m.BashGrepCount

	for _, record := range t.Records {
		for _, block := range record.Message.Content {
			if block.Type != "tool_result" {
				continue
			}
			text := toolResultText(block)
			// len(text) is the UTF-8 byte length directly, since Go strings are UTF-8 encoded
			// byte sequences — no separate encoding step is needed.
			byteLen := len(text)
			m.ToolResultBytes += byteLen
			toolName := toolUseNames[block.ToolUseID]
			m.ToolResultBytesBreakdown[toolName] += byteLen
			if toolName == "Read" {
				m.ReadBytes += byteLen
			}
		}
	}

	return m
}
