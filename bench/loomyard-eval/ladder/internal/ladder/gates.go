// gates.go ports the transcript-reading half of scripts/gates.py: pure predicates over a parsed
// subagent transcript that each return a slice of GateFinding rather than raising, so one run can fail
// several gates and report all of them at once. The environment-dependent gates (daemon liveness, the
// worktree filesystem checks) and the aggregating RunGates that composes every gate into one GateReport
// have no counterpart here -- they land in batches 4 and 5.

package ladder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// GateError is raised when a gate is asked to resolve state it cannot resolve safely. No gate in this
// file returns one -- the daemon-state and environment-resolution gates that do are ported in a later
// batch -- but the type is ported here alongside GateFinding and GateReport since every gate function's
// signature is written against it.
type GateError struct {
	// Message describes what state the gate could not resolve safely.
	Message string
}

// Error implements the error interface.
func (e *GateError) Error() string {
	return e.Message
}

// GateFinding is one observation from a single gate predicate.
type GateFinding struct {
	// Gate is the short name identifying which predicate produced this finding.
	Gate string
	// Fatal is true when this finding alone should invalidate the run.
	Fatal bool
	// Message is human-readable detail.
	Message string
}

// GateReport is the composed result of every gate RunGates applied to one run.
type GateReport struct {
	// Findings holds every GateFinding produced, fatal and non-fatal alike.
	Findings []GateFinding
}

// Passed reports whether no finding in this report is fatal. Ported from the Python dataclass's own
// passed property, the only accessor it exposed.
func (r GateReport) Passed() bool {
	for _, finding := range r.Findings {
		if finding.Fatal {
			return false
		}
	}
	return true
}

// FatalFindings returns the subset of r.Findings whose Fatal is true. This accessor has no Python
// counterpart; this port adds it for callers that report fatal and non-fatal findings separately.
func (r GateReport) FatalFindings() []GateFinding {
	var fatal []GateFinding
	for _, finding := range r.Findings {
		if finding.Fatal {
			fatal = append(fatal, finding)
		}
	}
	return fatal
}

// NonFatalFindings returns the subset of r.Findings whose Fatal is false. This accessor has no Python
// counterpart; this port adds it for callers that report fatal and non-fatal findings separately.
func (r GateReport) NonFatalFindings() []GateFinding {
	var nonFatal []GateFinding
	for _, finding := range r.Findings {
		if !finding.Fatal {
			nonFatal = append(nonFatal, finding)
		}
	}
	return nonFatal
}

// toolResultsByID maps a tool_use_id to its matching tool_result content block, across every user
// record's message content.
func toolResultsByID(records []Record) map[string]ContentBlock {
	results := make(map[string]ContentBlock)
	for _, record := range records {
		if record.Type != "user" {
			continue
		}
		for _, block := range record.Message.Content {
			if block.Type == "tool_result" {
				results[block.ToolUseID] = block
			}
		}
	}
	return results
}

// GateDeniedToolsNotUsed is fatal, once per offending call, when any tool_use block names a tool in
// deniedNames and its matching tool_result did not error. A denied name that appears only as a rejected
// attempt -- surfaced in a tool_result's error text, never as a tool_use block that went unanswered by
// an error -- is not a violation; it is the DeniedToolAttempts metric ExtractUsage already counts.
func GateDeniedToolsNotUsed(records []Record, deniedNames []string) []GateFinding {
	var findings []GateFinding
	resultsByID := toolResultsByID(records)
	for _, block := range IterToolUseBlocks(records) {
		if !stringSliceContains(deniedNames, block.Name) {
			continue
		}
		result, ok := resultsByID[block.ToolUseID]
		errored := ok && result.IsError
		if !errored {
			findings = append(findings, GateFinding{
				Gate:    "denied_tools_not_used",
				Fatal:   true,
				Message: fmt.Sprintf("denied tool %q was called and did not error", block.Name),
			})
		}
	}
	return findings
}

// targetOverrideKeys is the set of input keys GateNoTargetOverride checks, in the fixed order the
// Python gate checked them.
var targetOverrideKeys = []string{"targetDir", "buildTags"}

// GateNoTargetOverride is fatal, once per offending key, when any mcp__quarry__* tool call's input
// carries a targetDir or a buildTags key. A run that retargets breaks both the pinned-worktree
// constraint and the cold cell's daemon key.
func GateNoTargetOverride(records []Record) []GateFinding {
	var findings []GateFinding
	for _, block := range IterToolUseBlocks(records) {
		if !strings.HasPrefix(block.Name, MCPPrefix) {
			continue
		}
		for _, key := range targetOverrideKeys {
			if _, ok := block.Input[key]; ok {
				findings = append(findings, GateFinding{
					Gate:    "no_target_override",
					Fatal:   true,
					Message: fmt.Sprintf("%s call carried a %q key", block.Name, key),
				})
			}
		}
	}
	return findings
}

// contextWindowSuffixRe strips a trailing bracketed context-window suffix, e.g. "[1m]", from a reported
// model id -- see normaliseModelID.
var contextWindowSuffixRe = regexp.MustCompile(`\[[^\]]*\]$`)

// normaliseModelID drops a trailing bracketed context-window suffix from a model id, so a pinned
// "claude-opus-5" matches a reported "claude-opus-5[1m]".
func normaliseModelID(modelID string) string {
	return contextWindowSuffixRe.ReplaceAllString(modelID, "")
}

// GateModelPinned is fatal when the reported model does not match the pinned runModel, after
// normalising away a trailing bracketed context-window suffix on the reported id. The reported id is
// sourced from the first assistant record's message.model rather than from a system/init event -- a
// subagent transcript carries no such event at all (see transcript.go's Record doc comment). A
// transcript carrying no assistant record at all produces a fatal finding naming the absence, never an
// error return and never a panic: the Python port reached that state through an uncaught exception out
// of its own init-event lookup, which is not behaviour this port reproduces.
func GateModelPinned(records []Record, runModel string) []GateFinding {
	assistantRecords := AssistantRecords(records)
	if len(assistantRecords) == 0 {
		return []GateFinding{{
			Gate:    "model_pinned",
			Fatal:   true,
			Message: fmt.Sprintf("transcript carries no assistant record; cannot check against pinned run_model %q", runModel),
		}}
	}
	reportedModel := assistantRecords[0].Message.Model
	if normaliseModelID(reportedModel) != runModel {
		return []GateFinding{{
			Gate:    "model_pinned",
			Fatal:   true,
			Message: fmt.Sprintf("assistant record reported model %q, pinned run_model is %q", reportedModel, runModel),
		}}
	}
	return nil
}

// redactedPlaceholder replaces a tool_result block's own content in redactToolResultContent's output.
const redactedPlaceholder = "REDACTED"

// redactToolResultContent returns a copy of records with every tool_result block's nested Content
// replaced by a single placeholder text block, so GateBlinding can distinguish a token that appears only
// inside a tool_result payload from one that appears anywhere else in the transcript.
func redactToolResultContent(records []Record) []Record {
	redacted := make([]Record, len(records))
	for i, record := range records {
		if record.Type != "user" {
			redacted[i] = record
			continue
		}
		content := make([]ContentBlock, len(record.Message.Content))
		for j, block := range record.Message.Content {
			if block.Type == "tool_result" {
				block.Content = []ContentBlock{{Type: "text", Text: redactedPlaceholder}}
			}
			content[j] = block
		}
		record.Message.Content = content
		redacted[i] = record
	}
	return redacted
}

// GateBlinding applies only to a config whose allowed is empty (the caller's job to decide -- see
// RunGates). Fatal when the transcript contains an mcp__quarry__ tool name, or any filesystem path into
// repoRoot. The Python port's sibling literal check for "/tmp/quarry-bench" is dropped: no such binary
// exists in this suite, so the check could never fire and read as coverage that was not there. The
// identically-spelled branch in the redactor (see the redaction skill, batch 6) is a separate mechanism
// and is kept.
//
// Either unconditional check firing short-circuits the function: it returns immediately without
// evaluating the bare-mention check below, so this port never emits a finding the Python would not
// have. Only once neither has fired does a bare case-insensitive "quarry" mention get evaluated: a
// mention confined to a tool_result payload is not fatal -- it records a non-fatal
// target_origin_quarry_mention finding instead, because the target codebase mentions the word in its
// own tracked files and a bare-string gate would halt the matrix over the target's own prose. A mention
// found anywhere else in the transcript is fatal, since nothing besides a tool_result should ever have
// surfaced that word to a blinded agent. The session's own scratch directory is never treated as a leak
// by this gate -- it is legitimately the subagent's own cwd, not a path into repoRoot.
func GateBlinding(records []Record, repoRoot string) []GateFinding {
	var findings []GateFinding
	fullText := marshalRecordsForBlinding(records)

	if strings.Contains(fullText, MCPPrefix) {
		findings = append(findings, GateFinding{
			Gate:    "blinding",
			Fatal:   true,
			Message: "transcript contains an mcp__quarry__ tool name",
		})
	}
	if repoRoot != "" && strings.Contains(fullText, repoRoot) {
		findings = append(findings, GateFinding{
			Gate:    "blinding",
			Fatal:   true,
			Message: fmt.Sprintf("transcript contains a filesystem path into repo_root (%s)", repoRoot),
		})
	}
	if anyFatal(findings) {
		return findings
	}

	if !strings.Contains(strings.ToLower(fullText), "quarry") {
		return findings
	}
	nonToolResultText := marshalRecordsForBlinding(redactToolResultContent(records))
	if strings.Contains(strings.ToLower(nonToolResultText), "quarry") {
		findings = append(findings, GateFinding{
			Gate:    "blinding",
			Fatal:   true,
			Message: "quarry mentioned outside a tool_result payload",
		})
	} else {
		findings = append(findings, GateFinding{
			Gate:    "target_origin_quarry_mention",
			Fatal:   false,
			Message: "bare quarry mention confined to a tool_result payload",
		})
	}
	return findings
}

// anyFatal reports whether any finding in findings is fatal.
func anyFatal(findings []GateFinding) bool {
	for _, finding := range findings {
		if finding.Fatal {
			return true
		}
	}
	return false
}

// marshalRecordsForBlinding serialises records to JSON text for GateBlinding's substring checks,
// mirroring the Python gate's json.dumps(events) over the raw event list. Marshalling never fails on a
// []Record already produced by ReadTranscript, so a marshal error collapses to an empty string rather
// than a panic or a second error return threaded through every gate.
func marshalRecordsForBlinding(records []Record) string {
	encoded, err := json.Marshal(records)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// daemonBackedNames is the set of client-side mcp__quarry__* names derived from DaemonBackedTools
// through MCPName, computed once at package init.
var daemonBackedNames = daemonBackedNameSet()

// daemonBackedNameSet builds daemonBackedNames from DaemonBackedTools through MCPName rather than a
// literal, so it can never drift from the canonical daemon-backed set.
func daemonBackedNameSet() map[string]bool {
	names := make(map[string]bool, len(DaemonBackedTools))
	for _, tool := range DaemonBackedTools {
		names[MCPName(tool)] = true
	}
	return names
}

// UsedDaemonBackedTool reports whether the transcript contains at least one tool_use block whose name
// is mcp_name(t) for a t in DaemonBackedTools. toc_file and toc_dir are deliberately excluded from
// DaemonBackedTools: their handlers reach the tree-sitter path directly and never EnsureServer, so a toc
// call starts no daemon and writes no state.
func UsedDaemonBackedTool(records []Record) bool {
	for _, block := range IterToolUseBlocks(records) {
		if daemonBackedNames[block.Name] {
			return true
		}
	}
	return false
}

// GateMaxTurns is fatal when the count of assistant records in the transcript exceeds maxTurns,
// producing the same truncated outcome semantics the retired claude -p client's own --max-turns flag
// used to produce. This gate has no direct Python counterpart: the Agent Tool the session now dispatches
// through has no --max-turns equivalent, so nothing bounds a run mid-flight, and the ceiling is
// therefore evaluated post hoc against the finished transcript instead. Its basis also changed from the
// retired client's own turn accounting to assistant-record count -- which is why ladder.yaml's committed
// MaxTurns threshold ships blanked (nil) rather than carrying the old client's number forward.
func GateMaxTurns(records []Record, maxTurns int) []GateFinding {
	turnCount := len(AssistantRecords(records))
	if turnCount > maxTurns {
		return []GateFinding{{
			Gate:    "max_turns",
			Fatal:   true,
			Message: fmt.Sprintf("truncated: transcript carries %d assistant records, exceeding the max_turns ceiling of %d", turnCount, maxTurns),
		}}
	}
	return nil
}
