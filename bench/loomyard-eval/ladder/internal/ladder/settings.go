// settings.go derives, from a loaded Ladder and one of its LadderConfig rows, the permissions
// deny-list and the full settings document a run is launched with.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// DenyListFor returns the sorted list of client-side mcp__quarry__* names for every canonical tool in
// l.QuarryTools not in config.Allowed. Always derived through MCPName from l.QuarryTools -- never
// assembled from a literal -- so a config's deny-list tracks l.QuarryTools even for a deliberately
// mutated, already-loaded Ladder (see TestDenyListFor_TracksAMutatedQuarryTools).
func DenyListFor(l *Ladder, config LadderConfig) []string {
	deny := make([]string, 0, len(l.QuarryTools))
	for _, tool := range l.QuarryTools {
		if !stringSliceContains(config.Allowed, tool) {
			deny = append(deny, MCPName(tool))
		}
	}
	sort.Strings(deny)
	return deny
}

// Permissions is the "permissions" block of a run's settings document.
type Permissions struct {
	// Allow is fixed to Read/Grep/Glob/Bash for every config -- prompt-avoidance only, per the plan's
	// Shared Decision, never treated as an allowlist anywhere in this suite. This intentionally still
	// lists Grep/Glob even though baseRunTools (agentdef.go) does not grant them: this field only
	// suppresses permission prompts for tools the agent actually has, so the two entries are simply
	// inert here, not contradictory -- and this list only takes effect at all when the session's
	// scratch directory sits under an already-trusted ancestor (see ladder.yaml's session_dir_template
	// comment); a fresh, untrusted /tmp directory has this entire field silently ignored by Claude Code.
	Allow []string `json:"allow"`
	// Deny is config's quarry deny-list. "Task" is deliberately not included here -- see
	// SettingsDocumentFor's doc comment for why a session-wide deny of Task is incompatible with this
	// architecture, confirmed by an actual live dispatch attempt against the generated document.
	Deny []string `json:"deny"`
}

// SettingsDocument is the full settings mapping a run is launched with.
type SettingsDocument struct {
	Permissions Permissions `json:"permissions"`
}

// SettingsDocumentFor returns the full settings document a run of config is launched with.
// permissions.allow is fixed to Read/Grep/Glob/Bash. permissions.deny is config's quarry deny-list only
// (empty for a blinded "none" control, which declares no server at all).
//
// "Task" was originally included in this deny-list uniformly, as a backup against a run agent
// definition that fails to load and falls back to a broader tool set that could recursively spawn
// subagents. That reasoning only holds for the *dispatched* subagent -- but permissions.deny applies to
// the whole session document, including the top-level context the operator's own live session runs in
// before it has dispatched anything. A live run against the generated settings.json confirmed this
// directly: with Task denied, the operator's own session has no Agent Tool available at all, so it
// cannot dispatch the run agent in the first place -- the backup layer as originally written makes the
// architecture inoperable, not merely redundant. The structural protection this was meant to back up
// (a run agent definition's tools: frontmatter never lists Task, so a successfully loaded definition
// cannot spawn subagents regardless of settings.json) still stands on its own.
func SettingsDocumentFor(l *Ladder, config LadderConfig) SettingsDocument {
	// A blinded "none" control denies nothing: DenyListFor(l, config) on an empty Allowed set would
	// return every quarry name, which is exactly the leak this branch exists to prevent -- no quarry
	// name may appear anywhere in a blinded session's own files, since that session declares no server
	// and its scratch directory is the blinded agent's own cwd.
	var deny []string
	if len(config.Allowed) == 0 {
		deny = []string{}
	} else {
		deny = DenyListFor(l, config)
	}
	return SettingsDocument{
		Permissions: Permissions{
			Allow: []string{"Read", "Grep", "Glob", "Bash"},
			Deny:  deny,
		},
	}
}

// WriteSettings serialises SettingsDocumentFor(l, config) as indented JSON to path, with a trailing
// newline.
func WriteSettings(l *Ladder, config LadderConfig, path string) error {
	data, err := json.MarshalIndent(SettingsDocumentFor(l, config), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings document: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings document to %s: %w", path, err)
	}
	return nil
}
