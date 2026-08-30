// Package ladder is the ported capability-ladder bench harness logic: loading and validating
// ladder.yaml, the quarry-mcp tool constants and deny-list/settings/preamble derivation, transcript
// usage and gate scoring, and the run-state and session-provisioning helpers the ladderbench
// subcommands drive. It lives under an internal/ prefix so Go's own internal/ visibility rule makes
// it unimportable from the product tree (see the plan's package-layout Shared Decision).
package ladder

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// QuarryTools holds the canonical seven client-side tool names quarry-mcp exposes, bare (without the
// mcp__quarry__ prefix). This is the VALIDATION constant: LoadLadder checks the ladder file's own
// quarry_tools list against it, so a ladder that drifts from quarry's real surface is rejected at load
// time rather than silently producing a wrong deny-list downstream.
var QuarryTools = []string{
	"toc_dir",
	"toc_file",
	"textDocument_definition",
	"textDocument_references",
	"workspace_symbol",
	"impact",
	"assert_no_callers",
}

// MCPPrefix is the client-side prefix a quarry-mcp tool name carries once exposed to a run.
const MCPPrefix = "mcp__quarry__"

// DaemonBackedTools holds every canonical tool name except toc_dir and toc_file, derived from
// QuarryTools rather than listed as a literal so it can never drift from the canonical set. toc_dir/
// toc_file are tree-sitter-backed and never start a daemon (see tocFileHandler/tocDirHandler in
// internal/mcpserver/tools_toc.go, which call effectiveTargetDir/tocPreflight directly and never
// resolveCall). Every other canonical tool routes through resolveCall/EnsureServer and can be used as
// a warmth signal for the cold cell.
var DaemonBackedTools = daemonBackedTools()

// daemonBackedTools computes DaemonBackedTools from QuarryTools at package init, so the exclusion of
// toc_dir/toc_file is derived rather than a second literal that could drift from the first.
func daemonBackedTools() []string {
	tools := make([]string, 0, len(QuarryTools))
	for _, tool := range QuarryTools {
		if tool == "toc_dir" || tool == "toc_file" {
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

// MCPName returns the client-side mcp__quarry__* name for a bare tool name.
func MCPName(tool string) string {
	return MCPPrefix + tool
}

// ConfigError is raised when ladder.yaml fails validation. Its message names the offending
// field/config id and the ladder file path.
type ConfigError struct {
	// Path is the ladder file that failed validation.
	Path string
	// Message names the specific rule that failed.
	Message string
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// LadderConfig is one row of the matrix: a single (ladder, task, tool-exposure) cell.
type LadderConfig struct {
	// ID is the unique config id, e.g. "a5-bundle".
	ID string `yaml:"id"`
	// Ladder is "a" or "b" -- which ladder this config belongs to.
	Ladder string `yaml:"ladder"`
	// Task is the key into Ladder.Tasks.
	Task string `yaml:"task"`
	// Allowed holds the bare quarry_tools names exposed to this config's agent; empty for a "none"
	// control.
	Allowed []string `yaml:"allowed"`
	// Cold is true only for the single cold-cell config.
	Cold bool `yaml:"cold"`
	// WarmCounterpart is the warm config id this cold config is contrasted against, empty for every
	// non-cold config.
	WarmCounterpart string `yaml:"warm_counterpart"`
}

// TaskEntry is one entry of Ladder.Tasks: everything needed to set up and score one of the two target
// tasks (task 01 exploration, task 04 impact).
type TaskEntry struct {
	// TaskFile is the path to the task's markdown prompt.
	TaskFile string `yaml:"task_file"`
	// PinnedSHA is the Loomyard commit the task's pinned worktree is built from.
	PinnedSHA string `yaml:"pinned_sha"`
	// Worktree is the warm-cell working directory for this task.
	Worktree string `yaml:"worktree"`
	// Schema names the answer schema this task's runs are scored against.
	Schema string `yaml:"schema"`
	// Fasit is the path to the task's committed reference answer.
	Fasit string `yaml:"fasit"`
}

// ScorerConfig is the pinned scoring client parameters, shared by every config.
type ScorerConfig struct {
	// Model is empty until the operator pins the scoring model id.
	Model string `yaml:"model"`
	// Effort is empty until the operator pins the scoring reasoning effort.
	Effort string `yaml:"effort"`
}

// Ladder is the fully loaded, validated contents of ladder.yaml.
type Ladder struct {
	// RunModel is the pinned model id for all 45 runs; nil until the operator sets it. A pointer so
	// RequirePins can distinguish an unset value from the empty string.
	RunModel *string `yaml:"run_model"`
	// Reps is the repetitions per config.
	Reps int `yaml:"reps"`
	// MaxTurns is the per-run turn ceiling, identical across all 45 runs; nil until the operator sets
	// it. Its meaning is the maximum number of assistant records in a run's subagent transcript. A
	// pointer so RequirePins can distinguish an unset value from zero.
	MaxTurns *int `yaml:"max_turns"`
	// RunEffort is the pinned reasoning-effort level for all 45 runs; empty until the operator sets it.
	RunEffort string `yaml:"run_effort"`
	// SessionDirTemplate is the per-repetition session working-directory path template.
	SessionDirTemplate string `yaml:"session_dir_template"`
	// Scorer carries the pinned scoring model/effort.
	Scorer ScorerConfig `yaml:"scorer"`
	// QuarryTools is the canonical seven tool names, as loaded (validated to equal package QuarryTools).
	QuarryTools []string `yaml:"quarry_tools"`
	// Tasks maps task slug to TaskEntry.
	Tasks map[string]TaskEntry `yaml:"tasks"`
	// SourceRepo is the path to the Loomyard checkout the pinned worktrees are built from.
	SourceRepo string `yaml:"source_repo"`
	// Configs holds all 15 LadderConfig rows.
	Configs []LadderConfig `yaml:"configs"`
	// ColdWorktreeTemplate is the per-repetition cold worktree path template.
	ColdWorktreeTemplate string `yaml:"cold_worktree_template"`
}

// LoadLadder loads and validates ladder.yaml, returning a *Ladder.
//
// It returns a *ConfigError naming the offending file and rule for any of: a duplicate config id; a
// ladder value outside "a"/"b"; a task key absent from tasks; an allowed entry not present in
// quarry_tools; a quarry_tools list that is not exactly the canonical seven; a ladder with zero or more
// than one config whose allowed is empty (the control); more than one cold: true config; a
// warm_counterpart set on a non-cold config; a cold config with no warm_counterpart; and a
// warm_counterpart naming an unknown id, a cold config, or the cold config itself.
func LoadLadder(path string) (*Ladder, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ladder file %s: %w", path, err)
	}

	var l Ladder
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse ladder file %s: %w", path, err)
	}

	if !stringSlicesEqual(l.QuarryTools, QuarryTools) {
		return nil, fail(path, "quarry_tools must be exactly the canonical seven %v, got %v", QuarryTools, l.QuarryTools)
	}

	seenIDs := make(map[string]bool, len(l.Configs))
	for _, config := range l.Configs {
		if seenIDs[config.ID] {
			return nil, fail(path, "duplicate config id %q", config.ID)
		}
		seenIDs[config.ID] = true

		if config.Ladder != "a" && config.Ladder != "b" {
			return nil, fail(path, "config %q has ladder %q, must be 'a' or 'b'", config.ID, config.Ladder)
		}

		if _, ok := l.Tasks[config.Task]; !ok {
			return nil, fail(path, "config %q references unknown task %q", config.ID, config.Task)
		}

		for _, tool := range config.Allowed {
			if !stringSliceContains(l.QuarryTools, tool) {
				return nil, fail(path, "config %q allows unknown tool %q", config.ID, tool)
			}
		}
	}

	// Each of ladder "a" and "b" must carry exactly one control config -- the one with empty Allowed.
	for _, ladderName := range []string{"a", "b"} {
		controlCount := 0
		for _, config := range l.Configs {
			if config.Ladder == ladderName && len(config.Allowed) == 0 {
				controlCount++
			}
		}
		if controlCount == 0 {
			return nil, fail(path, "ladder %q has no control config (empty allowed)", ladderName)
		}
		if controlCount > 1 {
			return nil, fail(path, "ladder %q has %d control configs (empty allowed); must have exactly one", ladderName, controlCount)
		}
	}

	coldCount := 0
	for _, config := range l.Configs {
		if config.Cold {
			coldCount++
		}
	}
	if coldCount > 1 {
		return nil, fail(path, "found %d configs with cold: true; must have at most one", coldCount)
	}

	byID := make(map[string]LadderConfig, len(l.Configs))
	for _, config := range l.Configs {
		byID[config.ID] = config
	}
	for _, config := range l.Configs {
		if config.WarmCounterpart != "" && !config.Cold {
			return nil, fail(path, "config %q sets warm_counterpart but is not cold", config.ID)
		}
		if config.Cold && config.WarmCounterpart == "" {
			return nil, fail(path, "cold config %q has no warm_counterpart", config.ID)
		}
		if config.Cold {
			target, ok := byID[config.WarmCounterpart]
			if !ok {
				return nil, fail(path, "cold config %q names unknown warm_counterpart %q", config.ID, config.WarmCounterpart)
			} else if target.Cold {
				return nil, fail(path, "cold config %q names a cold config as warm_counterpart: %q", config.ID, config.WarmCounterpart)
			}
		}
	}

	return &l, nil
}

// fail builds a *ConfigError naming path and the formatted message, mirroring the Python module's
// _fail wrapper.
func fail(path, format string, args ...any) error {
	return &ConfigError{Path: path, Message: fmt.Sprintf(format, args...)}
}

// stringSlicesEqual reports whether a and b hold the same strings in the same order.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// stringSliceContains reports whether s holds v.
func stringSliceContains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}
