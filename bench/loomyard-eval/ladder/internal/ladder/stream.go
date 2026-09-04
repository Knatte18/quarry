// stream.go declares the stream-json record shapes this harness models and a tolerant
// line-delimited parser over them. The parser never fails on a record type it does not model —
// Claude Code emits record types this package has no struct for (a rate-limit event, for one),
// and a transcript is evidence that has already been paid for; discarding a whole rep because a
// new record type appeared would be the worst possible trade. Only a line that is not valid JSON
// at all is an error, since that is a truncated file rather than an unknown record.

package ladder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// scannerBufferSize is the maximum line size ParseTranscript's bufio.Scanner accepts, raised well
// past the default 64 KiB: a single assistant record carrying a large tool result can exceed the
// default limit, and a scanner that hit it would truncate the run's evidence silently instead of
// erroring.
const scannerBufferSize = 8 * 1024 * 1024

// knownRecordTypes are the stream-json "type" values ParseTranscript decodes fully. A line whose
// type is not in this set is kept as a raw line for MarshalAll but is not decoded into a Record or
// appended to Transcript.Records — its shape is unverified against this package's structs, and
// decoding it anyway is exactly the silent-truncation risk this parser exists to avoid.
var knownRecordTypes = map[string]bool{
	"system":    true,
	"assistant": true,
	"user":      true,
	"result":    true,
}

// Record is one line of a stream-json transcript decoded into the fields this harness models. Raw
// carries the original line's bytes verbatim so a gate can re-marshal a whole transcript without
// loss.
type Record struct {
	// Type is the record's top-level "type" field, e.g. "system", "assistant", "user", "result".
	Type string `json:"type"`
	// Subtype further discriminates Type, e.g. "init" on the first "system" record.
	Subtype string `json:"subtype"`
	// UUID is this record's own identifier.
	UUID string `json:"uuid"`
	// Timestamp is this record's capture time.
	Timestamp string `json:"timestamp"`
	// ParentToolUseID is the id of the tool_use block this record answers, populated on a "user"
	// record carrying a tool_result.
	ParentToolUseID string `json:"parent_tool_use_id"`
	// Message is this record's assistant/user payload.
	Message Message `json:"message"`
	// Raw is the original line's bytes, unmodified. It is what MarshalAll re-emits and what a gate
	// scans for a leaked token, so it must never be re-derived from the decoded fields.
	Raw []byte `json:"-"`
}

// Message is the assistant/user message payload a Record carries.
type Message struct {
	// ID is the API message id. Claude Code writes one transcript record per content block, so
	// every record of one API call carries the same ID — it is the grouping key ComputeMetrics
	// aggregates token usage by.
	ID string `json:"id"`
	// Model is the model id that produced this message, populated on assistant records.
	Model string `json:"model"`
	// Usage is this record's own token accounting, repeating the API call's snapshot. Zero-valued
	// on records that carry none, such as a user record replaying a tool result.
	Usage MessageUsage `json:"usage"`
	// Content is the message's content blocks, in the order the model emitted them.
	Content []ContentBlock `json:"content"`
}

// MessageUsage is the four token classes one message's usage object reports.
type MessageUsage struct {
	// InputTokens is this record's own input token count.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is this record's own output token count. It grows as a call streams, so only
	// the maximum across a call's records is a reliable figure — see ComputeMetrics.
	OutputTokens int `json:"output_tokens"`
	// CacheReadInputTokens is this record's own cache-read token count.
	CacheReadInputTokens int `json:"cache_read_input_tokens"`
	// CacheCreationInputTokens is this record's own cache-creation token count.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ContentBlock is one entry of a Message's Content slice, covering the three shapes this package
// reads — "text", "tool_use" and "tool_result" — discriminated by Type. Input and Content are kept
// as raw JSON rather than decoded further because their shape depends on Type and, for Content, on
// whether the tool result is a bare string or a nested block list.
type ContentBlock struct {
	// Type is "text", "tool_use", or "tool_result".
	Type string `json:"type"`
	// Text is the block's prose, populated on a "text" block.
	Text string `json:"text"`
	// Name is the tool name, populated on a "tool_use" block.
	Name string `json:"name"`
	// ID is the block's own identifier, populated on a "tool_use" block.
	ID string `json:"id"`
	// ToolUseID is the id of the "tool_use" block this "tool_result" block answers.
	ToolUseID string `json:"tool_use_id"`
	// Input is a "tool_use" block's argument object, carried raw.
	Input json.RawMessage `json:"input"`
	// Content is a "tool_result" block's own result payload, carried raw because Claude Code
	// writes it as either a bare string or a nested content-block array.
	Content json.RawMessage `json:"content"`
}

// SessionInit is the first "system"/"init" record's payload: the session's advertised
// configuration.
type SessionInit struct {
	// Tools is the advertised built-in tool list.
	Tools []string `json:"tools"`
	// MCPServers is the advertised MCP server name list.
	MCPServers []string `json:"mcp_servers"`
	// Model is the session's model id.
	Model string `json:"model"`
	// PermissionMode is the session's permission mode, e.g. "default".
	PermissionMode string `json:"permissionMode"`
	// ClaudeCodeVersion is the CLI version that produced the transcript.
	ClaudeCodeVersion string `json:"claude_code_version"`
	// MemoryPaths maps an auto-memory kind (e.g. "auto") to its resolved path, so the entry is
	// readable by key rather than by position in an array.
	MemoryPaths map[string]string `json:"memory_paths"`
	// Skills is the advertised skill name list.
	Skills []string `json:"skills"`
	// SlashCommands is the advertised slash-command name list.
	SlashCommands []string `json:"slash_commands"`
	// SessionID is the session's identifier.
	SessionID string `json:"session_id"`
}

// ResultRecord is the final "result" record's payload.
type ResultRecord struct {
	// NumTurns is the CLI's own reported turn count.
	NumTurns int `json:"num_turns"`
	// DurationMS is the session's wall-clock duration in milliseconds.
	DurationMS int64 `json:"duration_ms"`
	// DurationAPIMS is the session's cumulative API-call duration in milliseconds.
	DurationAPIMS int64 `json:"duration_api_ms"`
	// TotalCostUSD is the session's total measured cost.
	TotalCostUSD float64 `json:"total_cost_usd"`
	// TerminalReason names why the session ended, e.g. a max-turns ceiling.
	TerminalReason string `json:"terminal_reason"`
	// StopReason is the model's own stop reason on the final turn.
	StopReason string `json:"stop_reason"`
	// IsError reports whether the session ended in an error state.
	IsError bool `json:"is_error"`
	// PermissionDenials is the raw list of permission-denial entries the session recorded, kept
	// unparsed because their shape carries no field this package's accounting needs beyond count
	// and presence.
	PermissionDenials []json.RawMessage `json:"permission_denials"`
}

// Transcript is the result of parsing one stream-json file: every recognised record in order, the
// decoded session-init and result payloads, and every raw line for lossless re-marshalling.
type Transcript struct {
	// Records is every recognised record, in transcript order.
	Records []Record
	// Init is the first "system"/"init" record's decoded payload, nil if the transcript carries
	// none.
	Init *SessionInit
	// Result is the final "result" record's decoded payload, nil if the transcript carries none.
	Result *ResultRecord
	// Lines is every raw line, in transcript order, including lines of an unmodelled type.
	Lines [][]byte
}

// typeProbe is the minimal shape ParseTranscript decodes first, to route a line without risking a
// decode error against this package's fuller Record struct for a type it does not model.
type typeProbe struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

// ParseTranscript reads r as line-delimited stream-json and returns the decoded Transcript. A line
// whose type is not in knownRecordTypes is skipped rather than treated as an error — see this
// file's header comment. A line that is not valid JSON at all is an error naming its 1-based line
// number, since that is a truncated file rather than an unknown record type.
func ParseTranscript(r io.Reader) (*Transcript, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerBufferSize)

	t := &Transcript{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		// Copy the line: scanner.Bytes() reuses its internal buffer on the next Scan call, and
		// both Record.Raw and Transcript.Lines must keep their own stable copy.
		raw := append([]byte(nil), line...)
		t.Lines = append(t.Lines, raw)

		var probe typeProbe
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, fmt.Errorf("parse transcript: invalid JSON on line %d: %w", lineNumber, err)
		}
		if !knownRecordTypes[probe.Type] {
			continue
		}

		var record Record
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, fmt.Errorf("parse transcript: invalid JSON on line %d: %w", lineNumber, err)
		}
		record.Raw = raw
		t.Records = append(t.Records, record)

		if t.Init == nil && probe.Type == "system" && probe.Subtype == "init" {
			var init SessionInit
			if err := json.Unmarshal(raw, &init); err != nil {
				return nil, fmt.Errorf("parse transcript: invalid JSON on line %d: %w", lineNumber, err)
			}
			t.Init = &init
		}
		if probe.Type == "result" {
			var result ResultRecord
			if err := json.Unmarshal(raw, &result); err != nil {
				return nil, fmt.Errorf("parse transcript: invalid JSON on line %d: %w", lineNumber, err)
			}
			t.Result = &result
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse transcript: %w", err)
	}

	return t, nil
}

// MarshalAll concatenates every raw line, each followed by a newline, reproducing the transcript
// file byte-for-byte modulo a possibly-absent trailing newline on the input. This is what gate 2
// scans for a leaked token, so it must round-trip every line — including one of an unmodelled type
// — losslessly.
func (t *Transcript) MarshalAll() ([]byte, error) {
	var buf bytes.Buffer
	for _, line := range t.Lines {
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}
