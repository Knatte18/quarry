package ladder

import (
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

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames)

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

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames)

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

	prompt := RenderPrompt(target, "/tmp/target-dir", toolNames)

	if !strings.Contains(prompt, "mcp__quarry__toc_dir") {
		t.Errorf("RenderPrompt() for a granted cell does not list its granted tool name mcp__quarry__toc_dir:\n%s", prompt)
	}
}
