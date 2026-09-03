// score.go assembles and validates the pinned, three-input scoring exchange that turns one run's
// answer into a score.json, and blinds an answer to which rung of the capability ladder produced it
// before that answer ever reaches the scorer.
//
// The scoring rules, ExplorationRule and ImpactRule, and the required-field extraction that derives
// its validator from a rule's own fenced example, port V1's score.go verbatim. RedactAnswer is new:
// it operates on the answer as already-rendered JSON text rather than a decoded map, and builds its
// alternation from the ladder file's own tool list and server name via match.go's shared token
// matcher, so it agrees with gate 2's check (c) about what counts as a giveaway token instead of
// re-deriving the notion separately.

package ladder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RedactionPlaceholder is the fixed placeholder RedactAnswer writes over every redacted occurrence.
const RedactionPlaceholder = "<redacted>"

// RedactionInput carries every fact RedactAnswer cannot compute from the answer text alone: the
// ladder file's tool list, the server name, the MCP prefix, the quarry repository root path and the
// task worktree path. The tool list and server name come from the loaded ladder file rather than a
// hardcoded list, so a redaction alternation can never drift from what the file actually grants.
type RedactionInput struct {
	// QuarryTools is the ladder file's full tool list.
	QuarryTools []string
	// ServerName is the MCP server's own name.
	ServerName string
	// MCPPrefix is the tool-name prefix the MCP server registers its tools under.
	MCPPrefix string
	// QuarryRepoRoot is the quarry repository's root path.
	QuarryRepoRoot string
	// TaskWorktreePath is the pinned target worktree's path.
	TaskWorktreePath string
}

// RedactAnswer returns a copy of answer with every giveaway token replaced by RedactionPlaceholder.
// The bare-token half of the alternation is built through match.go's BareTokenAlternation over every
// entry of in.QuarryTools plus the bare server name -- the server name is included deliberately:
// without it, an answer whose prose names the server identifies the arm, and it is the same token
// gate 2's check (c) treats as leakage, so the two must agree. The MCP prefix and the two paths are
// applied as case-sensitive composed-string replacements, since word-boundary matching does not apply
// to a path or a prefix.
func RedactAnswer(answer string, in RedactionInput) string {
	redacted := answer

	bareTokens := append(append([]string{}, in.QuarryTools...), in.ServerName)
	if pattern := BareTokenAlternation(bareTokens); pattern != nil {
		redacted = pattern.ReplaceAllString(redacted, RedactionPlaceholder)
	}

	redacted = replaceComposedString(redacted, in.MCPPrefix)
	redacted = replaceComposedString(redacted, in.QuarryRepoRoot)
	redacted = replaceComposedString(redacted, in.TaskWorktreePath)

	return redacted
}

// replaceComposedString replaces every case-sensitive occurrence of s in text with
// RedactionPlaceholder, leaving text unchanged when s is empty -- strings.ReplaceAll would otherwise
// insert the placeholder between every rune of text for an empty old string.
func replaceComposedString(text, s string) string {
	if s == "" {
		return text
	}
	return strings.ReplaceAll(text, s, RedactionPlaceholder)
}

// ExplorationRule is reproduced byte for byte from the committed benchmark README's Scoring section,
// adapted from the three-arm A/B/C labelling to the blind fasit/answer pairing this scorer actually
// sees.
const ExplorationRule = "Exploration scoring rule:\n" +
	"\n" +
	"recall = (the fasit's relevant_files/key_symbols also present in the\n" +
	"answer's) / (the fasit's total); precision = (the answer's entries\n" +
	"corroborated by the fasit) / (the answer's total). Also judge qualitatively\n" +
	"whether the answer's summary describes the same actual mechanism the fasit\n" +
	"found, not just whether file names overlap.\n" +
	"\n" +
	"Reply with ONLY a fenced json code block, no other trailing prose after it:\n" +
	"```json\n" +
	"{\"recall\": <float 0.0-1.0>, \"precision\": <float 0.0-1.0>, \"summary_matches\": <true|false>}\n" +
	"```\n"

// ImpactRule is reproduced byte for byte from the committed benchmark README's Scoring section,
// adapted from the three-arm A/B/C labelling to the blind fasit/answer pairing this scorer actually
// sees.
const ImpactRule = "Impact-analysis scoring rule:\n" +
	"\n" +
	"recall = (the fasit's callers_to_update entries matched on file AND line --\n" +
	"a line must denote the same call site, not merely the same file -- also\n" +
	"present in the answer's) / (the fasit's total); precision = (the answer's\n" +
	"callers_to_update entries corroborated by the fasit) / (the answer's\n" +
	"total). decoy_admitted is true when the answer's callers_to_update contains\n" +
	"a call site the fasit lists under excluded_lookalikes -- report this as its\n" +
	"own field, never folded into precision. lookalikes_matched is the count of\n" +
	"the answer's excluded_lookalikes the fasit also names -- credited, never\n" +
	"required, so an answer naming none loses no points for it.\n" +
	"\n" +
	"Reply with ONLY a fenced json code block, no other trailing prose after it:\n" +
	"```json\n" +
	"{\"recall\": <float 0.0-1.0>, \"precision\": <float 0.0-1.0>, \"decoy_admitted\": <true|false>, \"lookalikes_matched\": <int>}\n" +
	"```\n"

// ruleBySchema maps a task's schema to the fixed scoring rule text a scorer prompt for that schema
// carries.
var ruleBySchema = map[string]string{
	"exploration": ExplorationRule,
	"impact":      ImpactRule,
}

// StripFasitMeta returns a shallow copy of fasit with its top-level "_meta" key removed and every
// other field left verbatim. _meta is identical across every run of a task and carries no per-config
// signal, but its role/see_also text names quarry and the scorer document -- scoring-irrelevant, so
// it is dropped rather than left to sit unexplained next to an answer deliberately redacted of the
// same words.
func StripFasitMeta(fasit map[string]any) map[string]any {
	stripped := make(map[string]any, len(fasit))
	for key, value := range fasit {
		if key == "_meta" {
			continue
		}
		stripped[key] = value
	}
	return stripped
}

// BuildScorerPrompt assembles the scorer's prompt from exactly rule, taskText, fasit and
// redactedAnswer, in V1's order: the rule, a task heading with taskText, a reference-fasit heading
// with the _meta-stripped fasit as a fenced json block, and an answer heading with redactedAnswer --
// already-rendered, already-redacted JSON text -- as a fenced json block. It never embeds a config
// id, a ladder letter, an allowed-tool list, the transcript, or any other run's answer: the scorer
// must not learn which rung it is grading.
func BuildScorerPrompt(rule string, taskText string, fasit map[string]any, redactedAnswer string) (string, error) {
	strippedFasit := StripFasitMeta(fasit)
	fasitJSON, err := json.MarshalIndent(strippedFasit, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build scorer prompt: marshal stripped fasit: %w", err)
	}

	return fmt.Sprintf(
		"%s\n## Task\n\n%s\n\n## Reference fasit\n\n```json\n%s\n```\n\n## Answer to score\n\n```json\n%s\n```\n",
		rule, taskText, fasitJSON, redactedAnswer,
	), nil
}

// ruleFieldPattern matches a JSON object key in a rule's declared example reply shape, e.g. the
// "recall" in {"recall": <float 0.0-1.0>, ...}. The rule's example block is not itself valid JSON --
// its values are placeholders like <float 0.0-1.0> -- so this reads field names directly out of the
// text rather than decoding it.
var ruleFieldPattern = regexp.MustCompile(`"(\w+)":`)

// scoreFieldsFromRule returns every field name a rule's declared example reply shape names, in the
// order they appear, so ParseScorerReply's required-field set is derived from the rule text itself
// rather than a hand-written list that could drift from it.
func scoreFieldsFromRule(rule string) []string {
	_, inner, err := ExtractFencedJSON(rule, "first")
	if err != nil {
		// rule is always one of the two package-level constants above, both of which carry a fenced
		// json example by construction; a mismatch here is a programming error in this file, not a
		// runtime condition callers can recover from.
		panic(fmt.Sprintf("scoring rule carries no fenced json example: %v", err))
	}
	matches := ruleFieldPattern.FindAllStringSubmatch(inner, -1)
	fields := make([]string, 0, len(matches))
	for _, match := range matches {
		fields = append(fields, match[1])
	}
	return fields
}

// requiredScoreFieldsBySchema maps a task schema to the field names ParseScorerReply requires a
// decoded reply to carry, derived from ruleBySchema's own declared output shape rather than a
// hand-written list so the rules stay the single source of truth.
var requiredScoreFieldsBySchema = map[string][]string{
	"exploration": scoreFieldsFromRule(ExplorationRule),
	"impact":      scoreFieldsFromRule(ImpactRule),
}

// ScoreRecord is the score.json shape: the scorer's own metrics for the task's schema, or the
// unscored stand-in UnscoredRecord produces.
type ScoreRecord map[string]any

// UnscoredRecord returns the score.json stand-in ScoreRecord for a rep that was never scored -- a rep
// that hit the turn ceiling, or one whose scorer dispatch failed -- carrying a false "scored" flag and
// reason naming why.
func UnscoredRecord(reason string) ScoreRecord {
	return ScoreRecord{
		"scored": false,
		"reason": reason,
	}
}

// ParseScorerReply decodes the first fenced json code block out of a scorer reply for the given task
// schema, reusing ExtractFencedJSON rather than compiling a second fence pattern, and requires every
// field that schema's rule declares -- a reply missing one silently drops a cell's measurement
// otherwise.
func ParseScorerReply(reply, schema string) (ScoreRecord, error) {
	_, inner, err := ExtractFencedJSON(reply, "first")
	if err != nil {
		return nil, fmt.Errorf("parse scorer reply: %w", err)
	}

	var record ScoreRecord
	if err := json.Unmarshal([]byte(inner), &record); err != nil {
		return nil, fmt.Errorf("parse scorer reply: decode fenced json block: %w", err)
	}

	required, ok := requiredScoreFieldsBySchema[schema]
	if !ok {
		return nil, fmt.Errorf("parse scorer reply: unknown schema %q", schema)
	}
	for _, field := range required {
		if _, present := record[field]; !present {
			return nil, fmt.Errorf("parse scorer reply: missing required field %q for schema %q", field, schema)
		}
	}

	return record, nil
}
