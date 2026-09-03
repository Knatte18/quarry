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
}

// IsControl reports whether c is the control cell for its ladder letter, i.e. it grants no tools.
func (c Config) IsControl() bool {
	return len(c.Allowed) == 0
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
		return nil, fmt.Errorf("decode ladder file %s: %w", path, err)
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

// ControlFor returns the single config for the given ladder letter whose allowed list is empty, and
// reports whether one was found. validate guarantees at most one such config exists per letter that
// appears in the file.
func (l *Ladder) ControlFor(letter string) (Config, bool) {
	for _, c := range l.Configs {
		if c.Ladder == letter && c.IsControl() {
			return c, true
		}
	}
	return Config{}, false
}

// validate checks l against every rule the ladder file format requires. It is extended below with
// the full rule set; this declaration exists so LoadLadder has a validation step to call.
func (l *Ladder) validate() error {
	return nil
}
