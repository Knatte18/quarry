// correlate.go builds the unique per-dispatch description a session's subagent transcript is correlated
// against, locates the metadata/transcript pair Claude Code wrote for that dispatch under its own
// projects tree, and copies both into a run directory so the results tree stays self-contained. This
// replaces the retired claude -p client's own captured-stdout transcript: under agent dispatch, the
// transcript instead lands wherever Claude Code's own session-transcript machinery writes it, and this
// file is the one place that goes looking for it.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// transcriptSearchPollInterval is LocateTranscript's poll interval while waiting for a matched
// metadata's sibling transcript to be flushed to disk.
const transcriptSearchPollInterval = 50 * time.Millisecond

// DispatchDescription returns the unique per-call description one dispatch is correlated against: the
// config id, the repetition, and the attempt index, in that fixed order. This is the single derivation
// site for the description -- next-run derives this same string (taking the attempt index from
// NextAttempt) to print the dispatching agent's own description, and ingest re-derives it identically to
// locate the transcript that dispatch produced, so the two commands provably agree on what a given
// attempt's subagent was told to describe itself as.
func DispatchDescription(configID string, rep, attempt int) string {
	return fmt.Sprintf("ladderbench run %s rep %d attempt %d", configID, rep, attempt)
}

// defaultProjectsRoot resolves ~/.claude/projects, the fixed root LocateTranscript searches under when
// its own projectsRoot argument is empty.
func defaultProjectsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ladder: default projects root: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// mangleProjectDir replaces every path separator and every literal dot in cwd with a hyphen, mirroring
// Claude Code's own project-directory naming convention under its projects root. Verified empirically
// against a real scratch directory containing a hidden-directory component (".scratch"): Claude Code's
// own on-disk project directory name has two consecutive hyphens there (one for the preceding "/", one
// for the "."), not a literal dot -- a plain separator-only replacement leaves the dot in place and never
// matches what Claude Code actually wrote to disk.
func mangleProjectDir(cwd string) string {
	replacer := strings.NewReplacer(string(filepath.Separator), "-", ".", "-")
	return replacer.Replace(cwd)
}

// subagentMetadata is the shape of one agent-<id>.meta.json file: only its description field is read
// here, the field LocateTranscript matches against.
type subagentMetadata struct {
	// Description is the dispatching call's own description string, matched exactly against
	// LocateTranscript's description argument.
	Description string `json:"description"`
}

// transcriptPathForMeta derives the agent-<id>.jsonl transcript path that sits beside a matched
// agent-<id>.meta.json file, by trimming the ".meta.json" suffix and substituting ".jsonl".
func transcriptPathForMeta(metaPath string) string {
	base := strings.TrimSuffix(filepath.Base(metaPath), ".meta.json")
	return filepath.Join(filepath.Dir(metaPath), base+".jsonl")
}

// LocateTranscript finds the one subagent metadata file, under projectsRoot's mangled sessionDir project
// directory, whose recorded description matches description exactly, and returns its path alongside the
// path of its sibling agent-<id>.jsonl transcript.
//
// projectsRoot defaults to ~/.claude/projects when empty. The search glob is
// <projects-root>/<mangled-sessionDir>/*/subagents/*.meta.json -- one wildcard segment for the session
// id, then the fixed subagents directory this session's own scratch directory writes under. The
// session-id wildcard deliberately spans every session ever launched from that scratch directory: because
// each run session has its own scratch directory and description embeds the config id, the repetition,
// and the attempt index, a second match means two dispatches genuinely shared one description, which is
// exactly the collision the multiple-match error below exists to catch.
//
// Zero matches and more than one match are both hard errors -- neither ever falls back to a
// newest-mtime guess, which would silently pick the wrong transcript under any concurrent dispatch.
// Exactly one match whose sibling transcript is not yet present on disk is re-checked at
// transcriptSearchPollInterval until wait elapses, then hard-errors naming the matched metadata path and
// the description -- a not-yet-flushed transcript is not itself a collision or a missing dispatch, only
// a race this bounded wait absorbs.
func LocateTranscript(projectsRoot, sessionDir, description string, wait time.Duration) (transcriptPath, metaPath string, err error) {
	root := projectsRoot
	if root == "" {
		root, err = defaultProjectsRoot()
		if err != nil {
			return "", "", err
		}
	}

	pattern := filepath.Join(root, mangleProjectDir(sessionDir), "*", "subagents", "*.meta.json")
	candidates, err := filepath.Glob(pattern)
	if err != nil {
		return "", "", fmt.Errorf("ladder: locate transcript: glob %s: %w", pattern, err)
	}

	var matched []string
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			return "", "", fmt.Errorf("ladder: locate transcript: read %s: %w", candidate, err)
		}
		var meta subagentMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return "", "", fmt.Errorf("ladder: locate transcript: parse %s: %w", candidate, err)
		}
		if meta.Description == description {
			matched = append(matched, candidate)
		}
	}

	if len(matched) == 0 {
		return "", "", fmt.Errorf("ladder: locate transcript: no subagent metadata under %s matched description %q", pattern, description)
	}
	if len(matched) > 1 {
		return "", "", fmt.Errorf("ladder: locate transcript: %d subagent metadata files matched description %q: %v", len(matched), description, matched)
	}

	metaPath = matched[0]
	transcriptPath = transcriptPathForMeta(metaPath)

	deadline := time.Now().Add(wait)
	for {
		if _, statErr := os.Stat(transcriptPath); statErr == nil {
			return transcriptPath, metaPath, nil
		}
		if time.Now().After(deadline) {
			return "", "", fmt.Errorf("ladder: locate transcript: matched metadata %s (description %q) but its transcript %s never appeared within %s", metaPath, description, transcriptPath, wait)
		}
		time.Sleep(transcriptSearchPollInterval)
	}
}

// copyIntoRunDir copies src's bytes to <runDir>/<destName>.
func copyIntoRunDir(src, runDir, destName string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("ladder: copy transcript custody: read %s: %w", src, err)
	}
	dest := filepath.Join(runDir, destName)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("ladder: copy transcript custody: write %s: %w", dest, err)
	}
	return nil
}

// CopyTranscriptCustody copies transcriptPath and metaPath into runDir as transcript.jsonl and
// transcript.meta.json, the fixed names GateRunCompleteArtifacts and the redaction/scoring steps require
// by. This is what keeps the results tree self-contained and lets it survive session-transcript pruning:
// the metadata copy remains on disk as the evidence that LocateTranscript's match picked the right file,
// even after the session's own projects-tree copy is gone.
func CopyTranscriptCustody(transcriptPath, metaPath, runDir string) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("ladder: copy transcript custody: mkdir %s: %w", runDir, err)
	}
	if err := copyIntoRunDir(transcriptPath, runDir, "transcript.jsonl"); err != nil {
		return err
	}
	if err := copyIntoRunDir(metaPath, runDir, "transcript.meta.json"); err != nil {
		return err
	}
	return nil
}
