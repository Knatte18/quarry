// transcript.go reads and iterates a run's subagent transcript: an agent-<id>.jsonl file the session
// captures while a dispatched subagent works one matrix cell. It ports scripts/extract_usage.py's
// read_transcript, iter_tool_use_blocks, and iter_tool_uses onto the subagent transcript shape, which
// carries no system/init record and no terminal result record -- the whole run is a stream of records
// keyed by the dispatching session's own sidechain, not a self-contained claude -p invocation.

package ladder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// MessageUsage is the token-accounting object attached to one assistant Record's Message. It is named
// MessageUsage rather than Usage because the package-level Usage struct (see usage.go) models the
// emitted usage.json document and the two names would otherwise collide.
type MessageUsage struct {
	// InputTokens is this record's own input token count -- summed across every assistant record by
	// ExtractUsage, never derived from another token class.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is this record's own output token count.
	OutputTokens int `json:"output_tokens"`
	// CacheReadInputTokens is this record's own cache-read token count.
	CacheReadInputTokens int `json:"cache_read_input_tokens"`
	// CacheCreationInputTokens is this record's own cache-creation token count.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ContentBlock is one entry of a Message's Content slice. Its Type selects which of the remaining
// fields are populated: "text" and "thinking" blocks carry Text; "tool_use" blocks carry ToolUseID
// (the block's own id field), Name, and Input; "tool_result" blocks carry ToolUseID (the id of the
// tool_use they answer), IsError, and a nested Content slice of the result's own text blocks.
type ContentBlock struct {
	// Type is "text", "thinking", "tool_use", or "tool_result".
	Type string `json:"type"`
	// Text is the block's prose, populated for "text" and "thinking" blocks.
	Text string `json:"text,omitempty"`
	// Name is the tool name, populated for "tool_use" blocks.
	Name string `json:"name,omitempty"`
	// Input is the tool call's argument map, populated for "tool_use" blocks.
	Input map[string]any `json:"input,omitempty"`
	// ToolUseID is the id field on a "tool_use" block, or the tool_use_id field naming the call being
	// answered on a "tool_result" block.
	ToolUseID string `json:"tool_use_id,omitempty"`
	// IsError is set on a "tool_result" block whose call failed.
	IsError bool `json:"is_error,omitempty"`
	// Content is a "tool_result" block's own nested content, typically one or more "text" blocks
	// carrying the tool's output or error text.
	Content []ContentBlock `json:"content,omitempty"`
}

// Message is the assistant/user payload embedded in one Record.
type Message struct {
	// Model is the model id that produced this message, populated on assistant records.
	Model string `json:"model,omitempty"`
	// Content is the message's content blocks, in the order the model emitted them.
	Content []ContentBlock `json:"content"`
	// Usage is this record's own token accounting. Zero-valued on records that carry no usage, such as
	// a user record replaying a tool result.
	Usage MessageUsage `json:"usage"`
}

// Record is one line of an agent-<id>.jsonl subagent transcript. Unlike the claude -p stream-json
// format this suite's harness used to capture, a subagent transcript carries no system/init record --
// there is no advertised-tools/model/session_id envelope, which is why ExtractUsage takes the granted
// tool list as a parameter instead of reading it here -- and no terminal result record, so fields the
// old format read off the result envelope (duration, turn count, denial count) are now derived from the
// record stream itself.
type Record struct {
	// ParentUUID is the uuid of the record this one replies to, nil for the first record in the
	// transcript.
	ParentUUID *string `json:"parentUuid"`
	// IsSidechain is true for every record in a subagent transcript -- the dispatching session's own
	// main chain is never captured here.
	IsSidechain bool `json:"isSidechain"`
	// AgentID identifies the dispatched subagent this transcript belongs to.
	AgentID string `json:"agentId"`
	// UUID is this record's own identifier.
	UUID string `json:"uuid"`
	// Timestamp is this record's capture time, RFC 3339 with millisecond precision.
	Timestamp string `json:"timestamp"`
	// Type is "assistant" or "user".
	Type string `json:"type"`
	// Effort is the reasoning-effort level, populated on assistant records.
	Effort string `json:"effort,omitempty"`
	// ToolUseResult is the tool-specific structured result payload a client attaches alongside a
	// tool_result content block. Its shape varies per tool, so it is carried opaquely and not modelled
	// further by this package -- no card in this batch reads it.
	ToolUseResult json.RawMessage `json:"toolUseResult,omitempty"`
	// Message is this record's assistant/user payload.
	Message Message `json:"message"`
}

// TranscriptError is raised when an agent-<id>.jsonl file cannot be parsed as this suite's subagent
// transcript shape: a malformed line. Unlike the retired claude -p format, there is no missing
// system/init or terminal result condition to report here, since a subagent transcript never carries
// either.
type TranscriptError struct {
	// Path is the transcript file that failed to parse.
	Path string
	// Message describes the parse failure.
	Message string
}

// Error implements the error interface.
func (e *TranscriptError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ReadTranscript parses a subagent transcript file into a slice of Record, in the order the session
// wrote them. Blank lines are skipped. A line that is not valid JSON returns a *TranscriptError naming
// its 1-based line number -- the malformed line is treated as a hard failure rather than skipped, since
// a transcript that fails to parse is exactly the run a caller must not silently score.
func ReadTranscript(path string) ([]Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript %s: %w", path, err)
	}
	defer file.Close()

	var records []Record
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, &TranscriptError{
				Path:    path,
				Message: fmt.Sprintf("malformed JSON on line %d: %v", lineNumber, err),
			}
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read transcript %s: %w", path, err)
	}
	return records, nil
}

// IterToolUseBlocks returns (toolUseID, name, input) for every tool_use content block in every
// assistant record, in transcript order. gates.go's tool_use/tool_result correlation gates use the
// id-carrying form directly rather than re-parsing the transcript; IterToolUses below is a thin
// (name, input) view over it for callers that only need the token classes and breakdown counts.
func IterToolUseBlocks(records []Record) []ToolUseBlock {
	var blocks []ToolUseBlock
	for _, record := range records {
		if record.Type != "assistant" {
			continue
		}
		for _, block := range record.Message.Content {
			if block.Type != "tool_use" {
				continue
			}
			blocks = append(blocks, ToolUseBlock{
				ToolUseID: block.ToolUseID,
				Name:      block.Name,
				Input:     block.Input,
			})
		}
	}
	return blocks
}

// ToolUseBlock is the (id, name, input) view IterToolUseBlocks returns for one tool_use content block.
type ToolUseBlock struct {
	// ToolUseID is the block's own id, correlating it to the tool_result block that answers it.
	ToolUseID string
	// Name is the tool name.
	Name string
	// Input is the tool call's argument map.
	Input map[string]any
}

// IterToolUses returns (name, input) for every tool_use content block in every assistant record, in
// transcript order.
func IterToolUses(records []Record) []NamedToolUse {
	blocks := IterToolUseBlocks(records)
	uses := make([]NamedToolUse, 0, len(blocks))
	for _, block := range blocks {
		uses = append(uses, NamedToolUse{Name: block.Name, Input: block.Input})
	}
	return uses
}

// NamedToolUse is the (name, input) view IterToolUses returns for one tool_use content block.
type NamedToolUse struct {
	// Name is the tool name.
	Name string
	// Input is the tool call's argument map.
	Input map[string]any
}

// AssistantRecords returns the subset of records whose Type is "assistant", in transcript order.
func AssistantRecords(records []Record) []Record {
	var assistants []Record
	for _, record := range records {
		if record.Type == "assistant" {
			assistants = append(assistants, record)
		}
	}
	return assistants
}
