// runstate.go declares the on-disk contract for one repetition: the raw repetition directory path,
// the six per-repetition file names every writer and reader in this package references rather than
// re-spelling, the run.json payload itself, and the completeness predicate and invalid-rep rename
// that let a killed run be resumed. It is deliberately separate from run.go, the loop that drives
// it, so the report path can read a results root without touching the loop at all.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// The six per-repetition file names, the single place they are spelled. Every writer and reader in
// this package references these constants rather than a literal.
const (
	// TranscriptFile is the tee'd stream-json transcript.
	TranscriptFile = "transcript.jsonl"
	// AnswerFile is the decoded fenced answer.
	AnswerFile = "answer.json"
	// RedactedAnswerFile is the answer after the scorer's redaction.
	RedactedAnswerFile = "answer.redacted.json"
	// UsageFile is the computed metrics, diagnostic only.
	UsageFile = "usage.json"
	// ScoreFile is the scorer's parsed reply, or the unscored stand-in.
	ScoreFile = "score.json"
	// RunStateFile is the repetition state.
	RunStateFile = "run.json"
)

// MaxAttempts is the ceiling on how many times one repetition is retried -- shared by the
// invalid-repetition rename this file implements and the scorer-only retry run.go implements -- so
// the two retry paths never drift to different limits.
const MaxAttempts = 3

// RepDir returns the raw repetition directory for cellID's repetition rep, under resultsRoot:
// <results-root>/raw/<cell>/<rep>.
func RepDir(resultsRoot, cellID string, rep int) string {
	return filepath.Join(resultsRoot, "raw", cellID, fmt.Sprintf("%d", rep))
}

// RunState is the run.json payload: one repetition's own disposition and the facts a consumer that
// never reads the ladder file needs to recompute gate 1 and account for the rep. ServerName and
// MCPPrefix are carried here on purpose -- the prefixed-tool-use count, and therefore gate 1, must be
// recomputable without the ladder file.
type RunState struct {
	// State is this repetition's disposition, e.g. "complete".
	State string `json:"state"`
	// ConfigID is the cell id this repetition belongs to.
	ConfigID string `json:"config_id"`
	// Ladder is the ladder letter this repetition's cell belongs to.
	Ladder string `json:"ladder"`
	// Task is the task id this repetition's cell runs.
	Task string `json:"task"`
	// Allowed is the tool subset this repetition's cell grants.
	Allowed []string `json:"allowed"`
	// IsControl reports whether this repetition's cell is its ladder letter's control.
	IsControl bool `json:"is_control"`
	// ControlForLadder is the ladder letter this repetition's cell is the control for, empty for a
	// non-control cell.
	ControlForLadder string `json:"control_for_ladder"`
	// ServerName is the MCP server's own name.
	ServerName string `json:"server_name"`
	// MCPPrefix is the tool-name prefix the MCP server registers its tools under.
	MCPPrefix string `json:"mcp_prefix"`
	// Rep is this repetition's own number.
	Rep int `json:"rep"`
	// Model is the model this repetition ran at.
	Model string `json:"model"`
	// Effort is the reasoning-effort level this repetition ran at.
	Effort string `json:"effort"`
	// MaxTurns is the turn ceiling this repetition ran under.
	MaxTurns int `json:"max_turns"`
	// Scored reports whether this repetition produced a real scorer reply, as opposed to the
	// unscored stand-in.
	Scored bool `json:"scored"`
	// ScoreSkipReason names why this repetition was not scored, empty when Scored is true.
	ScoreSkipReason string `json:"score_skip_reason"`
	// Observations is every non-fatal finding this repetition accumulated, in the order they were
	// recorded.
	Observations []Finding `json:"observations"`
	// BlindingFailed reports whether a fatal gate finding discarded this repetition. A complete
	// repetition with this flag set does not satisfy RepIsComplete, so the next invocation
	// re-attempts it once the operator fixes the cause.
	BlindingFailed bool `json:"blinding_failed"`
	// MaxTurnsHit reports whether this repetition ended at its turn ceiling.
	MaxTurnsHit bool `json:"max_turns_hit"`
}

// WriteRunState writes s as dir/run.json.
func WriteRunState(dir string, s RunState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("write run state %s: %w", dir, err)
	}
	path := filepath.Join(dir, RunStateFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write run state %s: %w", path, err)
	}
	return nil
}

// ReadRunState reads and decodes dir/run.json.
func ReadRunState(dir string) (RunState, error) {
	path := filepath.Join(dir, RunStateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return RunState{}, fmt.Errorf("read run state %s: %w", path, err)
	}
	var s RunState
	if err := json.Unmarshal(data, &s); err != nil {
		return RunState{}, fmt.Errorf("read run state %s: %w", path, err)
	}
	return s, nil
}

// RepIsComplete reports whether the repetition directory at dir satisfies resume: its run.json parses,
// its state field is the literal "complete", and its blinding-failed flag is false. A repetition
// discarded for blinding is written as complete so its transcript and reason survive on disk, yet does
// not satisfy this predicate -- the next invocation re-attempts it once the operator fixes the cause.
func RepIsComplete(dir string) bool {
	s, err := ReadRunState(dir)
	if err != nil {
		return false
	}
	return s.State == "complete" && !s.BlindingFailed
}

// InvalidateRep renames the repetition directory at dir to the same path suffixed with a dot, the
// word "invalid", a dash and the next unused attempt number starting at one, and returns how many
// such directories now exist (including the one just renamed).
func InvalidateRep(dir string) (attempts int, err error) {
	n := 0
	for {
		n++
		target := fmt.Sprintf("%s.invalid-%d", dir, n)
		if _, statErr := os.Stat(target); statErr == nil {
			continue
		} else if !os.IsNotExist(statErr) {
			return 0, fmt.Errorf("invalidate rep %s: %w", dir, statErr)
		}
		if err := os.Rename(dir, target); err != nil {
			return 0, fmt.Errorf("invalidate rep %s: %w", dir, err)
		}
		return n, nil
	}
}
