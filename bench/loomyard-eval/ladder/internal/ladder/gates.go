// gates.go implements the two gates that survived the V1-to-headless rewrite: gate 1, the
// tool-granted-but-unused observation, and gate 2, control-cell blinding. Both are judgment layers
// over a finished rep's transcript, and both use match.go's shared token matcher so their notion of
// "the giveaway token appeared" cannot drift apart. Neither gate touches a worktree, resolves an
// environment variable, or runs a process: every fact a gate cannot compute from the transcript
// arrives as an explicit field on BlindingInput, so the gates stay pure and directly table-testable.

package ladder

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// redactedToolResultContent is the placeholder a tool_result block's nested content is replaced with
// when CheckBlinding's check (c) tests whether a token survives redaction of every tool_result
// payload — see checkTargetOriginQuarryMention.
const redactedToolResultContent = "REDACTED"

// Finding is one gate's or observation's verdict against a rep or a cell. Fatal reports whether the
// finding aborts the rep before further API spend; Count is optional and populated only by findings
// that carry an occurrence count, such as the target_origin_quarry_mention observation.
type Finding struct {
	// Gate names which gate or observation produced this finding.
	Gate string
	// Fatal reports whether this finding aborts the rep.
	Fatal bool
	// Message is the human-readable finding text.
	Message string
	// Count is the finding's occurrence count, when it has one; zero otherwise.
	Count int
}

// BlindingInput carries every fact CheckBlinding and CheckRenderedControlPrompt cannot compute from
// the transcript or prompt text alone. ServerName is its own field rather than something derived by
// trimming MCPPrefix, matching how score.go's RedactionInput carries the two separately: the two
// consumers must agree on the giveaway token, and re-deriving it in one of them is how they stop
// agreeing.
type BlindingInput struct {
	// MCPPrefix is the tool-name prefix the MCP server registers its tools under.
	MCPPrefix string
	// ServerName is the MCP server's own name.
	ServerName string
	// QuarryRepoRoot is the quarry repository's root path.
	QuarryRepoRoot string
	// TokenInTargetTrackedFiles reports whether the bare token "quarry" appears in the pinned
	// worktree's tracked files.
	TokenInTargetTrackedFiles bool
	// TokenInAutoLoadedContext reports whether the bare token "quarry" appears in the session's
	// auto-loaded project context.
	TokenInAutoLoadedContext bool
}

// CheckGrantedToolUsed is gate 1: it applies per cell and is never fatal. When cfg grants a non-empty
// tool subset and the maximum prefixed-tool-use count across its reps is zero, the config measured
// only the tool's prompt cost and never the tool itself — that is worth flagging but never worth
// aborting on, since every rep already ran to completion. It returns nil for a control cell (an empty
// allowed list) and for a granted cell where at least one rep used a granted tool.
func CheckGrantedToolUsed(cfg Config, perRepQuarryToolUses []int) *Finding {
	if len(cfg.Allowed) == 0 {
		return nil
	}
	max := 0
	for _, n := range perRepQuarryToolUses {
		if n > max {
			max = n
		}
	}
	if max > 0 {
		return nil
	}
	return &Finding{
		Gate:  "granted_tool_used",
		Fatal: false,
		Message: fmt.Sprintf(
			"!! %s: tool-granted config whose agent never called a granted tool in any repetition -- this cell measures the tool's prompt cost, not the tool",
			cfg.ID,
		),
	}
}

// CheckBlinding is gate 2: it applies per rep, only for a control cell (an empty allowed list), and
// runs checks (a), (b) and (c) in order over the whole transcript re-marshalled to JSON via
// Transcript.MarshalAll -- every record and every field, the session-init record's working directory
// included, never a selected subset -- short-circuiting on the first fatal finding. Check (a) tests
// for the MCP prefix, check (b) for the quarry repository root path, both fatal; check (c),
// checkTargetOriginQuarryMention, is the always-non-fatal target_origin_quarry_mention observation
// and runs only when neither (a) nor (b) fired. It returns nil when none of the three checks produce
// a finding.
func CheckBlinding(t *Transcript, in BlindingInput) []Finding {
	marshalled, err := t.MarshalAll()
	if err != nil {
		// MarshalAll never errors today -- see stream.go -- but a future change that makes it
		// fallible must not silently skip blinding checks. Surface a fatal finding rather than
		// panicking, since a gate is not the place to introduce a new failure mode.
		return []Finding{{
			Gate:    "control_blinding_marshal",
			Fatal:   true,
			Message: fmt.Sprintf("!! could not marshal transcript for blinding checks: %v", err),
		}}
	}
	text := string(marshalled)

	if MatchesComposedString(text, in.MCPPrefix) {
		return []Finding{{
			Gate:    "control_blinding_mcp_prefix",
			Fatal:   true,
			Message: fmt.Sprintf("!! control-cell transcript contains the MCP prefix %q", in.MCPPrefix),
		}}
	}
	if MatchesComposedString(text, in.QuarryRepoRoot) {
		return []Finding{{
			Gate:    "control_blinding_repo_root",
			Fatal:   true,
			Message: fmt.Sprintf("!! control-cell transcript contains the quarry repository root path %q", in.QuarryRepoRoot),
		}}
	}
	if f := checkTargetOriginQuarryMention(t, in); f != nil {
		return []Finding{*f}
	}
	return nil
}

// lineTypeProbe decodes just the "type" field of a raw transcript line, so
// checkTargetOriginQuarryMention can attribute an occurrence to its record type without risking a
// decode error against the fuller Record struct for a line of an unmodelled type.
type lineTypeProbe struct {
	Type string `json:"type"`
}

// checkTargetOriginQuarryMention is check (c): it records the always-non-fatal
// target_origin_quarry_mention observation when the bare token "quarry" appears anywhere in t,
// carrying the occurrence count and the record types it appeared in. This check must never set the
// Fatal flag under any input -- a location-based fatal branch here is unreachable at the pinned
// commit and an unreachable check reads as protection while providing none. To let a reader tell an
// expected mention from a surprising one, the message names which antecedents held: whether the
// token also appears in an earlier tool_result payload -- computed by re-marshalling with every
// tool_result block's nested content replaced by redactedToolResultContent and testing whether the
// token survives -- and the two booleans the caller supplies. It returns nil when the token does not
// appear at all.
func checkTargetOriginQuarryMention(t *Transcript, in BlindingInput) *Finding {
	total := 0
	typesSeen := map[string]bool{}
	for _, line := range t.Lines {
		count := len(BareTokenPattern("quarry").FindAllString(string(line), -1))
		if count == 0 {
			continue
		}
		total += count
		var probe lineTypeProbe
		if err := json.Unmarshal(line, &probe); err == nil {
			typesSeen[probe.Type] = true
		}
	}
	if total == 0 {
		return nil
	}

	survivesRedaction := false
	for _, line := range t.Lines {
		redacted := redactToolResultContent(line)
		if MatchesBareToken(string(redacted), "quarry") {
			survivesRedaction = true
			break
		}
	}

	types := make([]string, 0, len(typesSeen))
	for typ := range typesSeen {
		types = append(types, typ)
	}
	sort.Strings(types)

	return &Finding{
		Gate:  "target_origin_quarry_mention",
		Fatal: false,
		Count: total,
		Message: fmt.Sprintf(
			"target_origin_quarry_mention: the bare token \"quarry\" appeared %d time(s) in record types %v; "+
				"survives-tool-result-redaction=%v token-in-target-tracked-files=%v token-in-auto-loaded-context=%v",
			total, types, survivesRedaction, in.TokenInTargetTrackedFiles, in.TokenInAutoLoadedContext,
		),
	}
}

// redactToolResultContent returns a copy of raw, one transcript line, with every tool_result content
// block's nested Content payload replaced by redactedToolResultContent. It is used only to test
// whether a token survives outside a tool_result payload -- see checkTargetOriginQuarryMention -- and
// never to decide a verdict on its own. A line this cannot decode, or one that carries no tool_result
// block, is returned unchanged.
func redactToolResultContent(raw []byte) []byte {
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return raw
	}
	message, ok := generic["message"].(map[string]any)
	if !ok {
		return raw
	}
	content, ok := message["content"].([]any)
	if !ok {
		return raw
	}
	changed := false
	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if block["type"] == "tool_result" {
			block["content"] = redactedToolResultContent
			changed = true
		}
	}
	if !changed {
		return raw
	}
	out, err := json.Marshal(generic)
	if err != nil {
		return raw
	}
	return out
}

// CheckRenderedControlPrompt is check (d): it applies per rep, before dispatch, and is fatal. The
// fully rendered prompt for a control cell must contain neither the bare token "quarry", nor the
// server name, nor any entry of quarryTools -- each matched with the shared bare-token matcher, which
// is what keeps a three-character tool name from matching ordinary prose -- nor the MCP prefix as a
// composed string. A failure fails the rep without spending an API call. It returns nil when prompt
// carries none of these.
func CheckRenderedControlPrompt(prompt string, in BlindingInput, quarryTools []string) *Finding {
	if MatchesBareToken(prompt, "quarry") {
		return &Finding{
			Gate:    "rendered_control_prompt",
			Fatal:   true,
			Message: "!! rendered control-cell prompt contains the bare token \"quarry\"",
		}
	}
	if MatchesBareToken(prompt, in.ServerName) {
		return &Finding{
			Gate:    "rendered_control_prompt",
			Fatal:   true,
			Message: fmt.Sprintf("!! rendered control-cell prompt contains the bare server name %q", in.ServerName),
		}
	}
	for _, tool := range quarryTools {
		if MatchesBareToken(prompt, tool) {
			return &Finding{
				Gate:    "rendered_control_prompt",
				Fatal:   true,
				Message: fmt.Sprintf("!! rendered control-cell prompt contains the tool token %q", tool),
			}
		}
	}
	if MatchesComposedString(prompt, in.MCPPrefix) {
		return &Finding{
			Gate:    "rendered_control_prompt",
			Fatal:   true,
			Message: fmt.Sprintf("!! rendered control-cell prompt contains the MCP prefix %q", in.MCPPrefix),
		}
	}
	return nil
}

// CheckWorktreeDirtied returns the always-non-fatal worktree_dirtied observation when porcelain, a
// worktree's `git status --porcelain` output, is non-empty. It returns nil for a clean worktree.
func CheckWorktreeDirtied(porcelain string) *Finding {
	if porcelain == "" {
		return nil
	}
	return &Finding{
		Gate:  "worktree_dirtied",
		Fatal: false,
		Message: fmt.Sprintf(
			"worktree_dirtied: the worktree was not clean after the rep:\n%s",
			strings.TrimRight(porcelain, "\n"),
		),
	}
}
