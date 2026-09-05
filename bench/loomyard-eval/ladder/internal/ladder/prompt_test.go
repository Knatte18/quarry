package ladder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const task01TaskText = "Explain how a reed session's terminal geometry is reconciled when the\n" +
	"operator's terminal window changes size or when re-attaching to an\n" +
	"existing session. Your explanation must cover:\n\n" +
	"1. Where/how the current terminal size is read.\n" +
	"2. How that size reaches the session's stored or live geometry state.\n" +
	"3. Any point where a previously-known geometry is compared against a\n" +
	"   freshly-read one (reconciliation), and what happens when they differ.\n" +
	"4. Which files and functions form this path, end to end.\n\n" +
	"Scope your answer to `internal/reedengine` and `internal/reedcli`."

const task01SchemaBlock = "```json\n" +
	"{\n" +
	"  \"relevant_files\": [\"internal/reedengine/geometry.go\", \"...\"],\n" +
	"  \"key_symbols\": [\n" +
	"    {\"name\": \"FuncOrTypeName\", \"file\": \"path/to/file.go\", \"role\": \"one sentence\"}\n" +
	"  ],\n" +
	"  \"summary\": \"3-6 sentences explaining how the mechanism works end to end\",\n" +
	"  \"confidence\": \"high|medium|low\",\n" +
	"  \"open_questions\": [\"anything left uncertain, if any\"]\n" +
	"}\n" +
	"```"

const task04TaskText = "You are about to change the `Shuttle` interface's `Run` method in\n" +
	"`internal/shedadapters/singlellm.go`, adding a `context.Context` first\n" +
	"parameter:\n\n" +
	"```go\n" +
	"// before\n" +
	"Run(shuttleengine.Spec) (shuttleengine.Result, error)\n" +
	"// after\n" +
	"Run(ctx context.Context, spec shuttleengine.Spec) (shuttleengine.Result, error)\n" +
	"```\n\n" +
	"Before making this change, identify every real call site **within\n" +
	"`internal/shedadapters`** that invokes this specific interface method and\n" +
	"would need a `context.Context` argument threaded in to keep compiling.\n\n" +
	"Some other call sites in this same package call a *different* method that\n" +
	"also happens to be named `Run` — those must not be listed, since changing\n" +
	"`Shuttle.Run`'s signature does not affect them. For every call site you\n" +
	"list, say how you confirmed it actually resolves to `Shuttle.Run` and not\n" +
	"a different method of the same name."

const task04SchemaBlock = "```json\n" +
	"{\n" +
	"  \"callers_to_update\": [\n" +
	"    {\"file\": \"internal/shedadapters/....go\", \"line\": N, \"evidence\": \"how you confirmed this resolves to Shuttle.Run specifically\"}\n" +
	"  ],\n" +
	"  \"excluded_lookalikes\": [\n" +
	"    {\"file\": \"internal/shedadapters/....go\", \"line\": N, \"reason\": \"why this same-named call site is NOT Shuttle.Run\"}\n" +
	"  ],\n" +
	"  \"confidence\": \"high|medium|low\",\n" +
	"  \"open_questions\": [\"anything left uncertain, if any\"]\n" +
	"}\n" +
	"```"

const task02TaskText = "Map how a build artifact flows through the \"shed\" pipeline: from\n" +
	"`internal/shedbuild`, through `internal/shedadapters`, to\n" +
	"`internal/shedcheck`. Your answer must identify:\n\n" +
	"1. The type(s) that represent a build artifact as it crosses each package\n" +
	"   boundary — same type reused throughout, or does each package have its\n" +
	"   own representation with a conversion step?\n" +
	"2. The specific function(s) at each handoff point (shedbuild →\n" +
	"   shedadapters, shedadapters → shedcheck).\n" +
	"3. What `shedadapters` actually contributes to the pipeline — its role,\n" +
	"   not just its existence.\n\n" +
	"Scope your answer to these three packages."

// task02SchemaBlock and task06SchemaBlock carry the exploration schema block with the placeholder
// example value, per the overview's neutral-schema-example-values-in-the-new-task-files decision --
// they must stay byte-identical to each other and must never carry a real Loomyard package path.
const task02SchemaBlock = "```json\n" +
	"{\n" +
	"  \"relevant_files\": [\"path/to/file.go\", \"...\"],\n" +
	"  \"key_symbols\": [\n" +
	"    {\"name\": \"FuncOrTypeName\", \"file\": \"path/to/file.go\", \"role\": \"one sentence\"}\n" +
	"  ],\n" +
	"  \"summary\": \"3-6 sentences explaining how the mechanism works end to end\",\n" +
	"  \"confidence\": \"high|medium|low\",\n" +
	"  \"open_questions\": [\"anything left uncertain, if any\"]\n" +
	"}\n" +
	"```"

const task06TaskText = "This repository is unfamiliar to you. Somewhere in it, each module keeps an\n" +
	"on-disk YAML configuration file that must stay in sync with that module's\n" +
	"built-in template as the template's own set of keys changes over time --\n" +
	"keys can be added to or removed from a template between releases, and an\n" +
	"existing on-disk file must be reconciled against that change rather than\n" +
	"silently drifting from it.\n\n" +
	"Find this mechanism and explain how it works. Your explanation must cover:\n\n" +
	"1. Which package(s) implement the actual reconciliation logic -- computing\n" +
	"   which keys were added to or removed from the template relative to an\n" +
	"   existing on-disk file -- and which package(s) own the registry of module\n" +
	"   names and their default templates.\n" +
	"2. What entry points (CLI commands, or exported functions a CLI command\n" +
	"   calls) trigger this reconciliation, and which package(s) those entry\n" +
	"   points live in.\n" +
	"3. Any module whose config is handled as a special case rather than by the\n" +
	"   ordinary per-module logic, and why.\n" +
	"4. Which files and functions form this path end to end, from an entry point\n" +
	"   down to the lowest-level key-comparison logic."

const task06SchemaBlock = "```json\n" +
	"{\n" +
	"  \"relevant_files\": [\"path/to/file.go\", \"...\"],\n" +
	"  \"key_symbols\": [\n" +
	"    {\"name\": \"FuncOrTypeName\", \"file\": \"path/to/file.go\", \"role\": \"one sentence\"}\n" +
	"  ],\n" +
	"  \"summary\": \"3-6 sentences explaining how the mechanism works end to end\",\n" +
	"  \"confidence\": \"high|medium|low\",\n" +
	"  \"open_questions\": [\"anything left uncertain, if any\"]\n" +
	"}\n" +
	"```"

func TestLoadTaskFile_RealTaskFiles(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		wantTaskText      string
		wantSchemaBlock   string
		droppedSubstrings []string // drawn from the setup/scope/scorer-notes sections; must not leak
	}{
		{
			name:            "exploration task 01",
			path:            "../../../tasks/01-reed-geometry-exploration.md",
			wantTaskText:    task01TaskText,
			wantSchemaBlock: task01SchemaBlock,
			droppedSubstrings: []string{
				"PINNED_SHA",                         // setup section
				"worktree add /tmp/loomyard-eval-01", // setup section
				"tmux-backed",                        // scope section
				"windowsize.go",                      // scorer-notes section
				"8b14f32b",                           // scorer-notes section
			},
		},
		{
			name:            "impact task 04",
			path:            "../../../tasks/04-shedadapters-shuttle-impact.md",
			wantTaskText:    task04TaskText,
			wantSchemaBlock: task04SchemaBlock,
			droppedSubstrings: []string{
				"975578cda8d6f3a81580bd4e73725e060211b766", // setup section
				"worktree add /tmp/loomyard-eval-04",       // setup section
				"BurlerRunner.Run",                         // scorer-notes section
				"singlellm.go:143",                         // scorer-notes section
				"bouncer.go:466",                           // scorer-notes section
			},
		},
		{
			name:            "exploration task 02",
			path:            "../../../tasks/02-shedadapters-exploration.md",
			wantTaskText:    task02TaskText,
			wantSchemaBlock: task02SchemaBlock,
			droppedSubstrings: []string{
				"worktree add /tmp/loomyard-eval-02",       // setup section
				"975578cda8d6f3a81580bd4e73725e060211b766", // setup section
				"8.4k lines",      // scope section
				"genuinely open",  // notes section
				"degenerate case", // notes section
			},
		},
		{
			name:            "exploration task 06",
			path:            "../../../tasks/06-loomyard-cold-start-orientation.md",
			wantTaskText:    task06TaskText,
			wantSchemaBlock: task06SchemaBlock,
			droppedSubstrings: []string{
				"worktree add /tmp/loomyard-eval-06",       // setup section
				"975578cda8d6f3a81580bd4e73725e060211b766", // setup section
				"value under test",                         // scope section
				"internal/configsync",                      // notes section
				"No subject swap was needed",               // notes section
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc, err := LoadTaskFile(tt.path)
			if err != nil {
				t.Fatalf("LoadTaskFile(%q) returned error: %v", tt.path, err)
			}
			if tc.TaskText != tt.wantTaskText {
				t.Errorf("LoadTaskFile(%q).TaskText = %q; want %q", tt.path, tc.TaskText, tt.wantTaskText)
			}
			if tc.SchemaBlock != tt.wantSchemaBlock {
				t.Errorf("LoadTaskFile(%q).SchemaBlock = %q; want %q", tt.path, tc.SchemaBlock, tt.wantSchemaBlock)
			}

			combined := tc.TaskText + "\n" + tc.SchemaBlock
			for _, s := range tt.droppedSubstrings {
				if strings.Contains(combined, s) {
					t.Errorf("LoadTaskFile(%q) result contains %q, which is drawn from the setup/scope/scorer-notes section and must not leak into the extracted content", tt.path, s)
				}
			}
		})
	}
}

// TestLoadTaskFile_NewFilesShareANeutralSchemaBlock asserts the overview's
// neutral-schema-example-values-in-the-new-task-files decision: task 02's and task 06's schema
// blocks are byte-identical to each other, and neither carries the substring "internal/" -- the
// standing guard that no real Loomyard package path re-enters either new prompt through the schema
// block, whether by a later edit or by a subject swap.
func TestLoadTaskFile_NewFilesShareANeutralSchemaBlock(t *testing.T) {
	if task02SchemaBlock != task06SchemaBlock {
		t.Errorf("task02SchemaBlock != task06SchemaBlock:\n%q\n%q", task02SchemaBlock, task06SchemaBlock)
	}
	for name, block := range map[string]string{
		"task02SchemaBlock": task02SchemaBlock,
		"task06SchemaBlock": task06SchemaBlock,
	} {
		if strings.Contains(block, "internal/") {
			t.Errorf("%s contains \"internal/\"; want a neutral placeholder path only", name)
		}
	}
}

func TestLoadTaskFile_NoSchemaHeading(t *testing.T) {
	path := "testdata/tasks/no-schema-heading.md"
	_, err := LoadTaskFile(path)
	if err == nil {
		t.Fatalf("LoadTaskFile(%q) = nil error; want a hard error naming the file", path)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("LoadTaskFile(%q) error = %v; want it to name the file", path, err)
	}
}

func TestRenderPrompt_SixPartsInOrder(t *testing.T) {
	target := TaskContent{
		TaskText:    "TASKTEXTMARKER",
		SchemaBlock: "SCHEMABLOCKMARKER",
	}
	toolNames := append([]string{}, BuiltinTools...)

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames, "")

	parts := []string{
		PARALLEL_OPENING,
		"/tmp/target-dir",
		"TASKTEXTMARKER",
		PARALLEL_BLOCK,
		closingSentence,
		"SCHEMABLOCKMARKER",
	}

	lastIdx := -1
	for _, part := range parts {
		idx := strings.Index(prompt, part)
		if idx == -1 {
			t.Fatalf("RenderPrompt() does not contain %q", part)
		}
		if idx <= lastIdx {
			t.Errorf("RenderPrompt() part %q appears out of order (index %d, previous %d)", part, idx, lastIdx)
		}
		lastIdx = idx
	}
}

func TestRenderPrompt_ControlCellNamesNoQuarryOrTOC(t *testing.T) {
	target := TaskContent{TaskText: "task text", SchemaBlock: "```json\n{}\n```"}
	toolNames := append([]string{}, BuiltinTools...)

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames, "")

	if MatchesBareToken(prompt, "quarry") {
		t.Errorf("RenderPrompt() for a control cell contains the word quarry:\n%s", prompt)
	}
	if MatchesBareToken(prompt, "toc") {
		t.Errorf("RenderPrompt() for a control cell contains the token toc:\n%s", prompt)
	}
}

func TestRenderPrompt_GrantedCellListsPrefixedToolName(t *testing.T) {
	target := TaskContent{TaskText: "task text", SchemaBlock: "```json\n{}\n```"}
	toolNames := append(append([]string{}, BuiltinTools...), "mcp__quarry__toc_dir")

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames, "")

	if !strings.Contains(prompt, "mcp__quarry__toc_dir") {
		t.Errorf("RenderPrompt() for a granted cell does not list its granted tool name mcp__quarry__toc_dir:\n%s", prompt)
	}
}

func TestRenderPrompt_CardLandsAfterTaskTextBeforeParallelBlock(t *testing.T) {
	target := TaskContent{TaskText: "TASKTEXTMARKER", SchemaBlock: "SCHEMABLOCKMARKER"}
	toolNames := append([]string{}, BuiltinTools...)

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames, "CARDMARKER")

	parts := []string{
		PARALLEL_OPENING,
		"/tmp/target-dir",
		"TASKTEXTMARKER",
		"CARDMARKER",
		PARALLEL_BLOCK,
		closingSentence,
		"SCHEMABLOCKMARKER",
	}

	lastIdx := -1
	for _, part := range parts {
		idx := strings.Index(prompt, part)
		if idx == -1 {
			t.Fatalf("RenderPrompt() does not contain %q", part)
		}
		if idx <= lastIdx {
			t.Errorf("RenderPrompt() part %q appears out of order (index %d, previous %d)", part, idx, lastIdx)
		}
		lastIdx = idx
	}
}

// TestRenderPrompt_NoCardIsByteIdenticalToTodaysOutput asserts the
// backwards-compatibility-is-a-tested-property decision: rendering the same TaskContent, target
// directory and tool list with an empty card produces exactly the six-section prompt RenderPrompt
// rendered before the card parameter existed, byte for byte. The golden string is spelled out here
// independently of renderBody's own implementation, so a bug in the join or in the conditional
// card-insertion is what this test would catch, not a self-referential match against the same code
// path it exercises.
func TestRenderPrompt_NoCardIsByteIdenticalToTodaysOutput(t *testing.T) {
	target := TaskContent{TaskText: "TASKTEXTMARKER", SchemaBlock: "SCHEMABLOCKMARKER"}
	targetDir := "/tmp/target-dir"
	toolNames := append([]string{}, BuiltinTools...)

	golden := strings.Join([]string{
		PARALLEL_OPENING,
		"You are working on a code task in the codebase at /tmp/target-dir. You have access to\n" +
			"the following tools: Read, Grep, Glob, Bash. Explore as needed to answer thoroughly and\n" +
			"correctly.",
		"TASKTEXTMARKER",
		PARALLEL_BLOCK,
		closingSentence,
		"SCHEMABLOCKMARKER",
	}, "\n\n")

	got := RenderPrompt(target, targetDir, toolNames, "")
	if got != golden {
		t.Errorf("RenderPrompt() with an empty card = %q; want golden %q", got, golden)
	}
}

// TestLoadCardFile covers a card read back with trailing newlines trimmed, and a missing path
// returning an error naming the file.
func TestLoadCardFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "card.md")
	if err := os.WriteFile(path, []byte("card text\n\n"), 0o644); err != nil {
		t.Fatalf("write card file %s: %v", path, err)
	}

	got, err := LoadCardFile(path)
	if err != nil {
		t.Fatalf("LoadCardFile(%q) error = %v", path, err)
	}
	if got != "card text" {
		t.Errorf("LoadCardFile(%q) = %q; want %q", path, got, "card text")
	}

	missing := filepath.Join(dir, "missing.md")
	if _, err := LoadCardFile(missing); err == nil {
		t.Fatalf("LoadCardFile(%q) = nil error; want a hard error naming the file", missing)
	} else if !strings.Contains(err.Error(), missing) {
		t.Errorf("LoadCardFile(%q) error = %v; want it to name the file", missing, err)
	}
}
