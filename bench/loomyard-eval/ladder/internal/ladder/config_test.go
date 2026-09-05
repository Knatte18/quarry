package ladder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLadderFile writes contents to a ladder.yaml under a fresh t.TempDir() and returns its path.
func writeLadderFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ladder.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write ladder file: %v", err)
	}
	return path
}

// minimalValid is a complete, accepted ladder file: one task, one control config, one server block.
const minimalValid = `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer:
  model: claude-opus-5
  effort: high
quarry_tools:
  - toc
server:
  name: quarry
  build: ./cmd/quarry-mcp
tasks:
  t1:
    task_file: tasks/t1.md
    pinned_sha: abc123
    schema: exploration
    fasit: tasks/t1.fasit.json
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - id: a0-none
    ladder: a
    task: t1
    allowed: []
`

func TestLoadLadder_Accepted(t *testing.T) {
	path := writeLadderFile(t, minimalValid)
	l, err := LoadLadder(path)
	if err != nil {
		t.Fatalf("LoadLadder(%s) = %v; want no error", path, err)
	}
	if len(l.Configs) != 1 || l.Configs[0].ID != "a0-none" {
		t.Errorf("LoadLadder(%s).Configs = %+v; want one config a0-none", path, l.Configs)
	}
}

func TestLoadLadder_NoServerBlockLoadsCleanly(t *testing.T) {
	contents := `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer:
  model: claude-opus-5
  effort: high
quarry_tools:
  - toc
tasks:
  t1:
    task_file: tasks/t1.md
    pinned_sha: abc123
    schema: exploration
    fasit: tasks/t1.fasit.json
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - id: a0-none
    ladder: a
    task: t1
    allowed: []
  - id: a1-toc
    ladder: a
    task: t1
    allowed: [toc]
`
	path := writeLadderFile(t, contents)
	l, err := LoadLadder(path)
	if err != nil {
		t.Fatalf("LoadLadder(%s) = %v; want no error for a config granting tools with no server block", path, err)
	}
	if l.Server != nil {
		t.Errorf("LoadLadder(%s).Server = %+v; want nil", path, l.Server)
	}
}

// TestLoadLadder_ThreeToolLessConfigsOneExplicitControl accepts three tool-less configs under one
// ladder letter when exactly one sets control: true, since Control now overrides the default
// len(Allowed) == 0 rule that would otherwise call all three controls.
func TestLoadLadder_ThreeToolLessConfigsOneExplicitControl(t *testing.T) {
	contents := `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: [], control: true}
  - {id: a1-none, ladder: a, task: t1, allowed: [], control: false}
  - {id: a2-none, ladder: a, task: t1, allowed: [], control: false}
`
	path := writeLadderFile(t, contents)
	l, err := LoadLadder(path)
	if err != nil {
		t.Fatalf("LoadLadder(%s) = %v; want no error", path, err)
	}
	c, ok := l.ControlFor("a")
	if !ok || c.ID != "a0-none" {
		t.Errorf(`ControlFor("a") = %+v, %v; want a0-none, true`, c, ok)
	}
}

func TestLoadLadder_Rejected(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		wantErr  string
	}{
		{
			name: "DuplicateConfigID",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
  - {id: a0-none, ladder: a, task: t1, allowed: [toc]}
`,
			wantErr: "duplicate id",
		},
		{
			name: "ConfigNamesUnknownTask",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: does-not-exist, allowed: []}
`,
			wantErr: "is not a key in tasks",
		},
		{
			name: "AllowedEntryOutsideToolList",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
  - {id: a1-extra, ladder: a, task: t1, allowed: [not-a-tool]}
`,
			wantErr: "is not in quarry_tools",
		},
		{
			name: "WrongSourceRepoLiteral",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: /some/absolute/path
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "source_repo",
		},
		{
			name: "LadderLetterWithZeroControls",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a1-toc, ladder: a, task: t1, allowed: [toc]}
`,
			wantErr: "expected exactly one control",
		},
		{
			name: "LadderLetterWithTwoControls",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
  - {id: a1-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "expected exactly one control",
		},
		{
			name: "ThreeToolLessConfigsTwoExplicitControls",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: [], control: true}
  - {id: a1-none, ladder: a, task: t1, allowed: [], control: true}
  - {id: a2-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "expected exactly one control",
		},
		{
			name: "ThreeToolLessConfigsNoExplicitControl",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
  - {id: a1-none, ladder: a, task: t1, allowed: []}
  - {id: a2-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "expected exactly one control",
		},
		{
			name: "TaskWithUnrecognisedSchema",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: bogus, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: `must be "exploration" or "impact"`,
		},
		{
			name: "RetiredKeyWorktree",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json, worktree: /tmp/x}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "worktree",
		},
		{
			name: "RetiredKeySessionDirTemplate",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
session_dir_template: .scratch/{config_id}-{n}
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "session_dir_template",
		},
		{
			name: "RetiredKeyCold",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: [], cold: false}
`,
			wantErr: "cold",
		},
		{
			name: "RetiredKeyWarmCounterpart",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: [], warm_counterpart: a1-none}
`,
			wantErr: "warm_counterpart",
		},
		{
			name: "RetiredKeyColdWorktreeTemplate",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
cold_worktree_template: .scratch/cold-{config_id}
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "cold_worktree_template",
		},
		{
			name: "RetiredKeyAnnex",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: [], annex: precomputed-toc}
`,
			wantErr: "annex",
		},
		{
			name: "RetiredKeyAnnexes",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
annexes:
  - precomputed-toc
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "annexes",
		},
		{
			name: "RetiredKeyTocFormat",
			contents: `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer: {model: claude-opus-5, effort: high}
quarry_tools: [toc]
toc_format: compact
tasks:
  t1: {task_file: tasks/t1.md, pinned_sha: abc123, schema: exploration, fasit: tasks/t1.fasit.json}
source_repo: env:LADDER_LOOMYARD_REPO
configs:
  - {id: a0-none, ladder: a, task: t1, allowed: []}
`,
			wantErr: "toc_format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeLadderFile(t, tt.contents)
			_, err := LoadLadder(path)
			if err == nil {
				t.Fatalf("LoadLadder(%s) = nil error; want error containing %q", path, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadLadder(%s) error = %q; want it to contain %q", path, err.Error(), tt.wantErr)
			}
		})
	}
}

// boolPtr returns a pointer to b, for constructing Config.Control values in table-driven tests.
func boolPtr(b bool) *bool {
	return &b
}

// TestConfigIsControlAndGrantsTools covers the four Control-defaulting cases and asserts that
// GrantsTools reads only Allowed in every one of them, so the two predicates are independent.
func TestConfigIsControlAndGrantsTools(t *testing.T) {
	tests := []struct {
		name        string
		control     *bool
		allowed     []string
		wantControl bool
		wantGrants  bool
	}{
		{"UnsetEmptyAllowed", nil, nil, true, false},
		{"UnsetNonEmptyAllowed", nil, []string{"toc"}, false, true},
		{"ExplicitTrueOverridesEmptyAllowed", boolPtr(true), nil, true, false},
		{"ExplicitFalseOverridesEmptyAllowed", boolPtr(false), nil, false, false},
		{"ExplicitTrueOverridesNonEmptyAllowed", boolPtr(true), []string{"toc"}, true, true},
		{"ExplicitFalseOverridesNonEmptyAllowed", boolPtr(false), []string{"toc"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{Control: tt.control, Allowed: tt.allowed}
			if got := c.IsControl(); got != tt.wantControl {
				t.Errorf("IsControl() = %v; want %v", got, tt.wantControl)
			}
			if got := c.GrantsTools(); got != tt.wantGrants {
				t.Errorf("GrantsTools() = %v; want %v", got, tt.wantGrants)
			}
		})
	}
}

// TestLoadLadder_ControlFieldDefaulting drives the same Control-defaulting matrix through
// LoadLadder, so the yaml decoding of control: is covered and not only the Config method.
func TestLoadLadder_ControlFieldDefaulting(t *testing.T) {
	tests := []struct {
		name        string
		controlLine string
		allowedLine string
		wantControl bool
	}{
		{"UnsetEmptyAllowed", "", "allowed: []", true},
		{"UnsetNonEmptyAllowed", "", "allowed: [toc]", false},
		{"ExplicitTrueOverridesEmptyAllowed", "control: true", "allowed: []", true},
		{"ExplicitFalseOverridesEmptyAllowed", "control: false", "allowed: []", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeLadderFile(t, controlDefaultingFixture(tt.controlLine, tt.allowedLine, tt.wantControl))
			l, err := LoadLadder(path)
			if err != nil {
				t.Fatalf("LoadLadder(%s) = %v; want no error", path, err)
			}
			testConfig, ok := l.ConfigByID("a0-under-test")
			if !ok {
				t.Fatalf("ConfigByID(a0-under-test) not found")
			}
			if got := testConfig.IsControl(); got != tt.wantControl {
				t.Errorf("Configs[a0-under-test].IsControl() = %v; want %v", got, tt.wantControl)
			}
		})
	}
}

// controlDefaultingFixture builds a two-config ladder file: the config under test, carrying
// controlLine (a "control: <bool>" yaml line, or empty to omit the key) and allowedLine (an
// "allowed: [...]" yaml line), plus a companion control config present only when the config under
// test is not itself expected to be the control -- otherwise validate's one-control-per-letter rule
// would reject the file before the assertion under test ever runs. Used by
// TestLoadLadder_ControlFieldDefaulting.
func controlDefaultingFixture(controlLine, allowedLine string, wantControl bool) string {
	entry := "  - id: a0-under-test\n    ladder: a\n    task: t1\n    " + allowedLine + "\n"
	if controlLine != "" {
		entry += "    " + controlLine + "\n"
	}
	if !wantControl {
		entry += "  - {id: a1-companion-control, ladder: a, task: t1, allowed: [], control: true}\n"
	}
	return `
run_model: claude-sonnet-5
reps: 1
run_effort: medium
max_turns: 10
scorer:
  model: claude-opus-5
  effort: high
quarry_tools:
  - toc
tasks:
  t1:
    task_file: tasks/t1.md
    pinned_sha: abc123
    schema: exploration
    fasit: tasks/t1.fasit.json
source_repo: env:LADDER_LOOMYARD_REPO
configs:
` + entry
}

// TestLoadLadder_RealTocFile loads the tracked, migrated ladder-toc.yaml and asserts the shape the
// breadth matrix depends on: all eight cell ids, the single-entry tool list, the four controls, the
// two new task entries' schema and shared pin, and the MCP prefix with and without a declared server
// block.
func TestLoadLadder_RealTocFile(t *testing.T) {
	l, err := LoadLadder("../../ladder-toc.yaml")
	if err != nil {
		t.Fatalf("LoadLadder(ladder-toc.yaml) = %v; want no error", err)
	}

	wantIDs := map[string]bool{
		"a0-none": true, "a2-toc-dir": true,
		"b0-none": true, "b8-toc-dir": true,
		"c0-none": true, "c1-toc-dir": true,
		"d0-none": true, "d1-toc-dir": true,
	}
	if len(l.Configs) != len(wantIDs) {
		t.Fatalf("len(Configs) = %d; want %d", len(l.Configs), len(wantIDs))
	}
	for _, c := range l.Configs {
		if !wantIDs[c.ID] {
			t.Errorf("unexpected config id %q", c.ID)
		}
	}

	if len(l.QuarryTools) != 1 || l.QuarryTools[0] != "toc" {
		t.Errorf("QuarryTools = %v; want [toc]", l.QuarryTools)
	}

	if c, ok := l.ControlFor("a"); !ok || c.ID != "a0-none" {
		t.Errorf(`ControlFor("a") = %+v, %v; want a0-none, true`, c, ok)
	}
	if c, ok := l.ControlFor("b"); !ok || c.ID != "b0-none" {
		t.Errorf(`ControlFor("b") = %+v, %v; want b0-none, true`, c, ok)
	}
	if c, ok := l.ControlFor("c"); !ok || c.ID != "c0-none" {
		t.Errorf(`ControlFor("c") = %+v, %v; want c0-none, true`, c, ok)
	}
	if c, ok := l.ControlFor("d"); !ok || c.ID != "d0-none" {
		t.Errorf(`ControlFor("d") = %+v, %v; want d0-none, true`, c, ok)
	}

	const sharedPin = "975578cda8d6f3a81580bd4e73725e060211b766"
	for _, taskID := range []string{"02-shedadapters-exploration", "06-loomyard-cold-start-orientation"} {
		task, ok := l.Tasks[taskID]
		if !ok {
			t.Errorf("Tasks[%q] not found", taskID)
			continue
		}
		if task.Schema != "exploration" {
			t.Errorf("Tasks[%q].Schema = %q; want %q", taskID, task.Schema, "exploration")
		}
		if task.PinnedSHA != sharedPin {
			t.Errorf("Tasks[%q].PinnedSHA = %q; want %q", taskID, task.PinnedSHA, sharedPin)
		}
	}

	if got := l.MCPPrefix(); got != "mcp__quarry__" {
		t.Errorf("MCPPrefix() = %q; want %q", got, "mcp__quarry__")
	}

	noServer := Ladder{}
	if got := noServer.MCPPrefix(); got != "mcp__quarry__" {
		t.Errorf("MCPPrefix() with no server block = %q; want %q", got, "mcp__quarry__")
	}
}
