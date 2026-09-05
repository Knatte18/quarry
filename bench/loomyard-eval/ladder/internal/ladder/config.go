// config.go declares the ladder configuration file's shape as Go structs and loads it with
// gopkg.in/yaml.v3. LoadLadder is the harness's one entry point for reading a ladder file; every
// other file in this package consumes the *Ladder it returns rather than parsing yaml itself.

package ladder

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BuiltinTools is the fixed set of built-in tools every cell, control and treatment alike, is
// granted: Read, Grep, Glob, Bash. The CLI's session-init record reports the same set back in
// sorted order, ["Bash","Glob","Grep","Read"]; a test asserting the reported list must compare
// against that sorted form, not this one.
var BuiltinTools = []string{"Read", "Grep", "Glob", "Bash"}

// Ladder is the decoded shape of a ladder configuration file: the run parameters shared by every
// cell, the scorer, the set of MCP tools the server may grant, the source repository, the optional
// MCP server declaration, the named tasks, and the configs (cells) that combine a task with an
// allowed tool subset.
type Ladder struct {
	RunModel    string          `yaml:"run_model"`
	RunEffort   string          `yaml:"run_effort"`
	MaxTurns    int             `yaml:"max_turns"`
	Reps        int             `yaml:"reps"`
	Scorer      ScorerSpec      `yaml:"scorer"`
	QuarryTools []string        `yaml:"quarry_tools"`
	SourceRepo  string          `yaml:"source_repo"`
	Server      *ServerSpec     `yaml:"server"`
	Tasks       map[string]Task `yaml:"tasks"`
	Configs     []Config        `yaml:"configs"`
	// PackTargets is the glyph list the pack subcommand resolves. It is a single top-level list
	// resolved once, which is why at most one config per file may set Pack.
	PackTargets []string `yaml:"pack_targets"`
}

// ScorerSpec names the model and effort the scoring pass runs at.
type ScorerSpec struct {
	Model  string `yaml:"model"`
	Effort string `yaml:"effort"`
}

// ServerSpec declares the MCP server a ladder's configs may grant tools from: its name, the build
// target that produces its binary, its argument list, and any environment entries to merge over the
// inherited environment when it runs.
type ServerSpec struct {
	Name  string            `yaml:"name"`
	Build string            `yaml:"build"`
	Args  []string          `yaml:"args"`
	Env   map[string]string `yaml:"env"`
}

// Task names one benchmark task: the task text file, the pinned commit of the repository it
// exercises, the fasit (expected-answer) file, and the schema the answer must satisfy.
type Task struct {
	TaskFile  string `yaml:"task_file"`
	PinnedSHA string `yaml:"pinned_sha"`
	Schema    string `yaml:"schema"`
	Fasit     string `yaml:"fasit"`
}

// Config is one cell of the ladder: an id, the ladder letter it belongs to, the task it runs, and
// the subset of the ladder's tool list it grants.
type Config struct {
	ID      string   `yaml:"id"`
	Ladder  string   `yaml:"ladder"`
	Task    string   `yaml:"task"`
	Allowed []string `yaml:"allowed"`
	// Control overrides which cell is its ladder letter's comparison baseline. A nil Control
	// defaults to today's behaviour, len(Allowed) == 0; an explicit false is distinguishable from
	// an omitted key, which is why this is a pointer rather than a plain bool.
	Control *bool `yaml:"control"`
	// Card is a repository-relative markdown file rendered into the prompt, resolved the same way
	// Task.TaskFile is. An empty value renders today's prompt unchanged.
	Card string `yaml:"card"`
	// Pack marks the one cell whose card the generated kick-start pack is written into.
	Pack bool `yaml:"pack"`
}

// IsControl reports whether c is its ladder letter's comparison baseline. When Control is set
// explicitly, that value wins; otherwise the default is today's behaviour, len(Allowed) == 0. This
// is a different question from GrantsTools: every cell of a ladder letter may grant no tools, in
// which case Control is the only way to pick one of them as the baseline.
func (c Config) IsControl() bool {
	if c.Control != nil {
		return *c.Control
	}
	return len(c.Allowed) == 0
}

// GrantsTools reports whether c has an MCP server attached, i.e. its allowed list is non-empty.
// This is a different question from IsControl: a cell can grant no tools without being its
// ladder letter's control, and vice versa.
func (c Config) GrantsTools() bool {
	return len(c.Allowed) > 0
}

// LoadLadder reads and decodes the ladder configuration file at path, rejecting any unrecognised
// key, then validates the decoded shape. A file that fails either step returns an error naming the
// offending key or id.
func LoadLadder(path string) (*Ladder, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ladder file %s: %w", path, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var l Ladder
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf("decode ladder file %s: %w", path, wrapRetiredKeyError(err))
	}

	if err := l.validate(); err != nil {
		return nil, fmt.Errorf("validate ladder file %s: %w", path, err)
	}

	return &l, nil
}

// ServerName returns the declared MCP server name, defaulting to "quarry" when the file declares no
// server block.
func (l *Ladder) ServerName() string {
	if l.Server != nil && l.Server.Name != "" {
		return l.Server.Name
	}
	return "quarry"
}

// MCPPrefix returns the tool-name prefix the MCP server registers its tools under, derived from
// ServerName.
func (l *Ladder) MCPPrefix() string {
	return "mcp__" + l.ServerName() + "__"
}

// ConfigByID returns the config with the given id and reports whether one was found.
func (l *Ladder) ConfigByID(id string) (Config, bool) {
	for _, c := range l.Configs {
		if c.ID == id {
			return c, true
		}
	}
	return Config{}, false
}

// ControlFor returns the single config for the given ladder letter that IsControl reports true for,
// and reports whether one was found. validate guarantees at most one such config exists per letter
// that appears in the file.
func (l *Ladder) ControlFor(letter string) (Config, bool) {
	for _, c := range l.Configs {
		if c.Ladder == letter && c.IsControl() {
			return c, true
		}
	}
	return Config{}, false
}

// retiredKeys are the yaml keys the V1 architecture used that this loader must refuse rather than
// silently accept. KnownFields(true) already turns any of these into a decode-time unknown-key
// error; wrapRetiredKeyError only recognises the ones on this list and names them explicitly, so a
// stale file fails legibly instead of with a bare "field not found" message.
var retiredKeys = []string{
	"worktree",
	"session_dir_template",
	"cold",
	"warm_counterpart",
	"cold_worktree_template",
	"annex",
	"annexes",
	"toc_format",
}

// wrapRetiredKeyError inspects a decode error for the name of a retired key and, when it finds one,
// wraps err with a message stating that the key was removed with the V1 architecture. A decode
// error naming any other unrecognised key is returned unchanged.
func wrapRetiredKeyError(err error) error {
	for _, key := range retiredKeys {
		if MatchesBareToken(err.Error(), key) {
			return fmt.Errorf("field %q was removed with the V1 architecture: %w", key, err)
		}
	}
	return err
}

// validate checks l against every rule the ladder file format requires: the kept-from-V1 rules
// (source_repo's literal value, config id uniqueness, task and tool-list references, one control
// per ladder letter that appears, and the non-zero run/scorer/task fields), the schema enumeration,
// and four new rules for pack, pack_targets and card: at most one config in the whole file may set
// Pack true; a Pack config must declare a non-empty Card; PackTargets is non-empty if and only if
// some config sets Pack true; and every PackTargets entry is non-empty and unique. Retired-key
// rejection happens earlier, at decode time, via wrapRetiredKeyError. validate is deliberately
// filesystem-free — it never opens Card, PackTargets, or any other referenced file; the sentinel
// check on a card's contents lives elsewhere, where the repository root is in hand.
func (l *Ladder) validate() error {
	if l.SourceRepo != "env:LADDER_LOOMYARD_REPO" {
		return fmt.Errorf("source_repo: must be the literal %q, got %q", "env:LADDER_LOOMYARD_REPO", l.SourceRepo)
	}
	if l.RunModel == "" {
		return fmt.Errorf("run_model: must be non-zero")
	}
	if l.RunEffort == "" {
		return fmt.Errorf("run_effort: must be non-zero")
	}
	if l.MaxTurns == 0 {
		return fmt.Errorf("max_turns: must be non-zero")
	}
	if l.Reps == 0 {
		return fmt.Errorf("reps: must be non-zero")
	}
	if l.Scorer.Model == "" {
		return fmt.Errorf("scorer.model: must be non-zero")
	}
	if l.Scorer.Effort == "" {
		return fmt.Errorf("scorer.effort: must be non-zero")
	}

	for name, task := range l.Tasks {
		if task.TaskFile == "" {
			return fmt.Errorf("tasks.%s.task_file: must be set", name)
		}
		if task.PinnedSHA == "" {
			return fmt.Errorf("tasks.%s.pinned_sha: must be set", name)
		}
		if task.Fasit == "" {
			return fmt.Errorf("tasks.%s.fasit: must be set", name)
		}
		if task.Schema != "exploration" && task.Schema != "impact" {
			return fmt.Errorf("tasks.%s.schema: must be \"exploration\" or \"impact\", got %q", name, task.Schema)
		}
	}

	toolSet := make(map[string]bool, len(l.QuarryTools))
	for _, t := range l.QuarryTools {
		toolSet[t] = true
	}

	seenIDs := make(map[string]bool, len(l.Configs))
	controlsByLetter := make(map[string]int)
	lettersSeen := make(map[string]bool)
	packCount := 0
	for _, c := range l.Configs {
		if seenIDs[c.ID] {
			return fmt.Errorf("configs: duplicate id %q", c.ID)
		}
		seenIDs[c.ID] = true
		lettersSeen[c.Ladder] = true

		if _, ok := l.Tasks[c.Task]; !ok {
			return fmt.Errorf("configs.%s.task: %q is not a key in tasks", c.ID, c.Task)
		}

		for _, a := range c.Allowed {
			if !toolSet[a] {
				return fmt.Errorf("configs.%s.allowed: %q is not in quarry_tools", c.ID, a)
			}
		}

		if c.Pack {
			packCount++
			if c.Card == "" {
				return fmt.Errorf("configs.%s.card: must be set when pack is true", c.ID)
			}
		}

		if c.IsControl() {
			controlsByLetter[c.Ladder]++
		}
	}

	for letter := range lettersSeen {
		if n := controlsByLetter[letter]; n != 1 {
			return fmt.Errorf("ladder %q: expected exactly one control, found %d", letter, n)
		}
	}

	if packCount > 1 {
		return fmt.Errorf("configs: at most one config may set pack, found %d", packCount)
	}
	if packCount == 0 && len(l.PackTargets) > 0 {
		return fmt.Errorf("pack_targets: set but no config sets pack")
	}
	if packCount > 0 && len(l.PackTargets) == 0 {
		return fmt.Errorf("pack_targets: must be non-empty when a config sets pack")
	}

	seenTargets := make(map[string]bool, len(l.PackTargets))
	for _, target := range l.PackTargets {
		if target == "" {
			return fmt.Errorf("pack_targets: entries must be non-empty")
		}
		if seenTargets[target] {
			return fmt.Errorf("pack_targets: duplicate entry %q", target)
		}
		seenTargets[target] = true
	}

	return nil
}
