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
	// Shared Decision, never treated as an allowlist anywhere in this suite.
	Allow []string `json:"allow"`
	// Deny is config's quarry deny-list plus "Task", denied uniformly across all 45 runs so a
	// dispatched subagent's tool calls can never produce an undercounted transcript.
	Deny []string `json:"deny"`
}

// SettingsDocument is the full settings mapping a run is launched with.
type SettingsDocument struct {
	Permissions Permissions `json:"permissions"`
}

// SettingsDocumentFor returns the full settings document a run of config is launched with.
// permissions.allow is fixed to Read/Grep/Glob/Bash. For a config whose Allowed is non-empty,
// permissions.deny is config's quarry deny-list plus "Task". For a config whose Allowed is empty (a
// blinded "none" control), permissions.deny is exactly ["Task"] and nothing else -- no quarry name may
// appear in a blinded config's settings document, because that document sits in the blinded agent's
// own cwd.
func SettingsDocumentFor(l *Ladder, config LadderConfig) SettingsDocument {
	var deny []string
	if len(config.Allowed) == 0 {
		// A blinded "none" control: no quarry name may appear in its settings document, since that
		// document sits in the blinded agent's own cwd.
		deny = []string{"Task"}
	} else {
		deny = append(deny, DenyListFor(l, config)...)
		deny = append(deny, "Task")
		sort.Strings(deny)
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
