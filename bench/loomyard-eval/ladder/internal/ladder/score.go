// score.go blinds a run's answer to which rung of the capability ladder produced it, and assembles and
// validates the pinned, three-input scoring exchange that turns one run's answer into a score.json.
//
// Blinding has two parts: RedactText/RedactAnswer/WriteRedacted strip every trace of tool provenance --
// client-side mcp__quarry__* names, bare canonical tool names, the word "quarry", and CLI invocation
// forms -- out of an answer's free-text fields before it ever reaches the scorer. "impact" is the one
// canonical tool name excluded from the bare-name pass because it is also an ordinary English word every
// Ladder-B answer's prose legitimately uses; only its mcp__quarry__impact and CLI-invocation forms are
// redacted.
//
// BuildScorerPrompt assembles a prompt from exactly three inputs -- the redacted answer, the
// _meta-stripped fasit, and the task text -- plus the fixed scoring rule for the task's schema. Unlike
// the ported Python's score_run, this package never dispatches that prompt itself: the subprocess
// scorer client (run_scorer_client/score_run's dispatch half) has no counterpart here, because dispatch
// moves to the session. The redact / record-score subcommand pair the CLI batches wire up takes
// score_run's injected-runner seam's place, with ParseScorerReply validating the reply a session-driven
// dispatch hands back.
//
// The fasit's own free-text fields are left verbatim by StripFasitMeta: it is one fixed file, identical
// across every rung of its task, so it cannot leak which config is being graded, and its evidence/reason
// text is what the scoring rules match a run's own entries against.
package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RedactionToken is the placeholder every redacted tool-provenance mention is replaced with.
const RedactionToken = "<tool>"

// bareToolNamesExceptImpact holds the canonical tool names redacted on their own, bare, unprefixed
// form. "impact" is deliberately excluded -- see the package doc comment and buildRedactionPattern.
var bareToolNamesExceptImpact = computeBareToolNamesExceptImpact()

// computeBareToolNamesExceptImpact derives bareToolNamesExceptImpact from QuarryTools at package init,
// so the exclusion of "impact" is derived rather than a second literal that could drift from the first.
func computeBareToolNamesExceptImpact() []string {
	tools := make([]string, 0, len(QuarryTools))
	for _, tool := range QuarryTools {
		if tool == "impact" {
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

// buildRedactionPattern builds the single compiled regex RedactText runs, as an alternation ordered
// from most to least specific so a more specific form (e.g. a full mcp__quarry__* name, or the
// "quarry <verb>" CLI shell form that also catches impact's CLI invocation) claims a match before the
// bare "quarry" fallback gets a chance to.
func buildRedactionPattern() *regexp.Regexp {
	alternatives := make([]string, 0, len(QuarryTools)*2+4)
	for _, tool := range QuarryTools {
		alternatives = append(alternatives, `\b`+regexp.QuoteMeta(MCPName(tool))+`\b`)
	}
	// "quarry <verb>" CLI shell form, e.g. "quarry impact ..." or "quarry toc_file ...". Consumed as
	// one unit, which is also the only path that redacts impact's CLI-invocation form (impact itself
	// is excluded from the bare-name pass below).
	alternatives = append(alternatives, `\bquarry\s+[A-Za-z_]+\b`)
	for _, tool := range bareToolNamesExceptImpact {
		alternatives = append(alternatives, `\b`+regexp.QuoteMeta(tool)+`\b`)
	}
	alternatives = append(alternatives, regexp.QuoteMeta("/tmp/quarry-bench"))
	alternatives = append(alternatives, `--target-dir(?:[= ]\S+)?`)
	// The bare word "quarry" on its own, last so every more specific alternative above gets first
	// refusal at the same starting position.
	alternatives = append(alternatives, `\bquarry\b`)
	return regexp.MustCompile(`(?i)` + strings.Join(alternatives, "|"))
}

// redactionPattern is the compiled alternation RedactText runs.
var redactionPattern = buildRedactionPattern()

// adjacentTokenRun collapses a run of RedactionTokens produced by adjacent matches from one phrase
// (e.g. two adjacent mcp__quarry__* names) into a single token, so the redacted prose stays readable
// instead of stuttering.
var adjacentTokenRun = regexp.MustCompile(regexp.QuoteMeta(RedactionToken) + `(?:\s+` + regexp.QuoteMeta(RedactionToken) + `)+`)

// RedactText replaces every case-insensitive occurrence of tool provenance in text with RedactionToken:
// every mcp__quarry__* client-side name, every bare canonical quarry tool name except "impact", the
// word "quarry", and CLI invocation forms (the literal "/tmp/quarry-bench" path, a "quarry <verb>"
// shell form, and a "--target-dir"-style flag).
//
// A run of adjacent tokens produced by one phrase collapses into a single token.
func RedactText(text string) string {
	redacted := redactionPattern.ReplaceAllString(text, RedactionToken)
	return adjacentTokenRun.ReplaceAllString(redacted, RedactionToken)
}

// deepCopyJSONValue returns a deep copy of a value produced by json.Unmarshal into an any -- a tree of
// map[string]any, []any, and scalars. It never errors because that decoded shape has no cycles and no
// non-copyable types.
func deepCopyJSONValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		copied := make(map[string]any, len(val))
		for k, item := range val {
			copied[k] = deepCopyJSONValue(item)
		}
		return copied
	case []any:
		copied := make([]any, len(val))
		for i, item := range val {
			copied[i] = deepCopyJSONValue(item)
		}
		return copied
	default:
		return val
	}
}

// RedactAnswer returns a deep copy of answer with RedactText applied to every free-text field a scorer
// would otherwise read tool provenance out of.
//
// Exploration answers get summary, open_questions, and each key_symbols[].role redacted. Impact answers
// get open_questions, each callers_to_update[].evidence, and each excluded_lookalikes[].reason
// redacted. Structural fields -- relevant_files, every file, every line, every name, and confidence --
// are returned untouched.
func RedactAnswer(answer map[string]any) map[string]any {
	redacted := deepCopyJSONValue(answer).(map[string]any)

	if summary, ok := redacted["summary"].(string); ok {
		redacted["summary"] = RedactText(summary)
	}
	if questions, ok := redacted["open_questions"].([]any); ok {
		for i, question := range questions {
			if text, ok := question.(string); ok {
				questions[i] = RedactText(text)
			}
		}
	}
	if symbols, ok := redacted["key_symbols"].([]any); ok {
		for _, entry := range symbols {
			if symbol, ok := entry.(map[string]any); ok {
				if role, ok := symbol["role"].(string); ok {
					symbol["role"] = RedactText(role)
				}
			}
		}
	}
	if callers, ok := redacted["callers_to_update"].([]any); ok {
		for _, entry := range callers {
			if caller, ok := entry.(map[string]any); ok {
				if evidence, ok := caller["evidence"].(string); ok {
					caller["evidence"] = RedactText(evidence)
				}
			}
		}
	}
	if lookalikes, ok := redacted["excluded_lookalikes"].([]any); ok {
		for _, entry := range lookalikes {
			if lookalike, ok := entry.(map[string]any); ok {
				if reason, ok := lookalike["reason"].(string); ok {
					lookalike["reason"] = RedactText(reason)
				}
			}
		}
	}

	return redacted
}

// WriteRedacted reads answer.json from runDir, writes the redacted copy beside it as
// answer.redacted.json, and leaves the original byte-identical.
//
// Returns the redacted answer mapping.
func WriteRedacted(runDir string) (map[string]any, error) {
	answerPath := filepath.Join(runDir, "answer.json")
	data, err := os.ReadFile(answerPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", answerPath, err)
	}

	var answer map[string]any
	if err := json.Unmarshal(data, &answer); err != nil {
		return nil, fmt.Errorf("parse %s: %w", answerPath, err)
	}

	redacted := RedactAnswer(answer)

	redactedJSON, err := json.MarshalIndent(redacted, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal redacted answer: %w", err)
	}
	redactedJSON = append(redactedJSON, '\n')

	redactedPath := filepath.Join(runDir, "answer.redacted.json")
	if err := os.WriteFile(redactedPath, redactedJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", redactedPath, err)
	}

	return redacted, nil
}

// ScoringError is raised when a scorer reply carries no parseable fenced json block, or its decoded
// score record is missing a field required for its task schema. The second case has no Python
// counterpart -- see ParseScorerReply's doc comment.
type ScoringError struct {
	// Message is the human-readable failure description.
	Message string
}

// Error implements the error interface.
func (e *ScoringError) Error() string {
	return e.Message
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

// ImpactRule is reproduced byte for byte from the committed benchmark README's Scoring section, adapted
// from the three-arm A/B/C labelling to the blind fasit/answer pairing this scorer actually sees.
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

// StripFasitMeta returns a shallow copy of a loaded fasit mapping with its top-level _meta block
// removed.
//
// _meta is identical across every run of a task and carries no per-config signal, but its
// role/see_also text names quarry and scorecard.md -- scoring-irrelevant, so it is dropped rather than
// left to sit unexplained next to an answer deliberately redacted of the same words. Every other field,
// including the free-text evidence/reason strings the scoring rules match a run's entries against, is
// left verbatim: the fasit is one fixed file, identical across all 45 runs of its task, so it cannot
// tell the scorer which config it is grading, and redacting it would damage the very fields
// recall/precision are computed from.
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

// BuildScorerPrompt assembles the scorer's prompt from exactly three inputs plus the fixed rule for the
// task's schema: the redacted answer, the _meta-stripped fasit, and the task text.
//
// Never embeds c.ID, c.Ladder, c.Allowed, the transcript, or any other run's answer -- the scorer must
// not learn which rung it is grading. The task text is included because the exploration rule cannot
// judge the summary without it, and it is identical across a ladder's rungs.
func BuildScorerPrompt(l *Ladder, c LadderConfig, redactedAnswer, fasit map[string]any, taskText string) (string, error) {
	task, ok := l.Tasks[c.Task]
	if !ok {
		return "", fmt.Errorf("build scorer prompt: config %q references unknown task %q", c.ID, c.Task)
	}
	rule, ok := ruleBySchema[task.Schema]
	if !ok {
		return "", fmt.Errorf("build scorer prompt: task %q has unknown schema %q", c.Task, task.Schema)
	}

	strippedFasit := StripFasitMeta(fasit)
	fasitJSON, err := json.MarshalIndent(strippedFasit, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal stripped fasit: %w", err)
	}
	answerJSON, err := json.MarshalIndent(redactedAnswer, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal redacted answer: %w", err)
	}

	return fmt.Sprintf(
		"%s\n## Task\n\n%s\n\n## Reference fasit\n\n```json\n%s\n```\n\n## Answer to score\n\n```json\n%s\n```\n",
		rule, taskText, fasitJSON, answerJSON,
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
		// rule is one of the two package-level constants above, both of which carry a fenced json
		// example by construction; a mismatch here is a programming error in this file, not a
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

// requiredScoreFieldsBySchema maps a task schema to the field names ParseScorerReply requires a decoded
// reply to carry, derived from ruleBySchema's own declared output shape rather than a hand-written list
// so the rules stay the single source.
var requiredScoreFieldsBySchema = map[string][]string{
	"exploration": scoreFieldsFromRule(ExplorationRule),
	"impact":      scoreFieldsFromRule(ImpactRule),
}

// ScoreRecord is the score.json shape: the scorer's own metrics for the task's schema, plus the pinned
// scorer model, effort, and prompt_template (the task schema the template was chosen from) that the
// record-score subcommand stamps in after ParseScorerReply decodes the scorer's reply.
type ScoreRecord map[string]any

// ParseScorerReply decodes the first fenced json code block out of a scorer reply for the given task
// schema.
//
// This ports _extract_fenced_json, reusing ExtractFencedJSON rather than compiling a second fence
// pattern, and taking its inner half -- the decode-ready content -- never the fenced block. It then adds
// validation the Python original does not perform: the Python decodes and returns whatever the fence
// carries, so a reply missing a field would reach score.json and surface later as an absent metric with
// nothing naming the cause. ParseScorerReply instead requires every field schema's rule declares --
// independently of which fields the summariser treats as optional -- because a missing one silently
// drops a cell's measurement.
//
// It does not dispatch anything: run_scorer_client and score_run's dispatch half have no counterpart
// here (see the package doc comment) -- dispatch happens in a live session, never in a subprocess.
func ParseScorerReply(reply, schema string) (ScoreRecord, error) {
	_, inner, err := ExtractFencedJSON(reply, "first")
	if err != nil {
		return nil, &ScoringError{Message: "scorer reply carried no fenced json block"}
	}

	var record ScoreRecord
	if err := json.Unmarshal([]byte(inner), &record); err != nil {
		return nil, &ScoringError{Message: fmt.Sprintf("scorer reply's fenced json block did not parse: %v", err)}
	}

	required, ok := requiredScoreFieldsBySchema[schema]
	if !ok {
		return nil, &ScoringError{Message: fmt.Sprintf("parse scorer reply: unknown schema %q", schema)}
	}
	for _, field := range required {
		if _, present := record[field]; !present {
			return nil, &ScoringError{Message: fmt.Sprintf("scorer reply missing required field %q for schema %q", field, schema)}
		}
	}

	return record, nil
}
