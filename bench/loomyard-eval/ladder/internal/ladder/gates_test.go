package ladder

import (
	"strings"
	"testing"
)

func TestGateReport_Passed(t *testing.T) {
	passing := GateFinding{Gate: "x", Message: "ok", Fatal: false}
	failing := GateFinding{Gate: "y", Message: "bad", Fatal: true}

	if !(GateReport{Findings: []GateFinding{passing}}).Passed() {
		t.Error("GateReport{passing}.Passed() = false; want true")
	}
	if (GateReport{Findings: []GateFinding{passing, failing}}).Passed() {
		t.Error("GateReport{passing, failing}.Passed() = true; want false")
	}
}

func TestGateReport_FatalAndNonFatalFindings(t *testing.T) {
	passing := GateFinding{Gate: "x", Message: "ok", Fatal: false}
	failing := GateFinding{Gate: "y", Message: "bad", Fatal: true}
	report := GateReport{Findings: []GateFinding{passing, failing}}

	fatal := report.FatalFindings()
	if len(fatal) != 1 || fatal[0].Gate != "y" {
		t.Errorf("report.FatalFindings() = %+v; want [%+v]", fatal, failing)
	}

	nonFatal := report.NonFatalFindings()
	if len(nonFatal) != 1 || nonFatal[0].Gate != "x" {
		t.Errorf("report.NonFatalFindings() = %+v; want [%+v]", nonFatal, passing)
	}
}

func TestToolResultsByID(t *testing.T) {
	records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
	}

	results := toolResultsByID(records)
	if len(results) != 8 {
		t.Fatalf("toolResultsByID() has %d entries; want 8", len(results))
	}
	result, ok := results["tu_1"]
	if !ok {
		t.Fatal(`toolResultsByID()["tu_1"] missing`)
	}
	if result.Type != "tool_result" || result.IsError {
		t.Errorf("toolResultsByID()[%q] = %+v; want a non-error tool_result", "tu_1", result)
	}
}

func TestGateDeniedToolsNotUsed(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		deniedNames []string
		wantCount   int
	}{
		{"passes when the denied tool was never called", "bundle-mixed-tools.jsonl", []string{"mcp__quarry__impact"}, 0},
		{"passes when the denied call errored", "denied-attempt.jsonl", []string{"mcp__quarry__impact"}, 0},
		{"fails when a denied tool succeeded", "bundle-mixed-tools.jsonl", []string{"mcp__quarry__toc_file"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := ReadTranscript("testdata/" + tt.fixture)
			if err != nil {
				t.Fatalf("ReadTranscript(%q) = _, %v; want nil error", tt.fixture, err)
			}
			findings := GateDeniedToolsNotUsed(records, tt.deniedNames)
			if len(findings) != tt.wantCount {
				t.Fatalf("GateDeniedToolsNotUsed() = %+v; want %d finding(s)", findings, tt.wantCount)
			}
			for _, finding := range findings {
				if !finding.Fatal || finding.Gate != "denied_tools_not_used" {
					t.Errorf("finding = %+v; want fatal denied_tools_not_used", finding)
				}
			}
		})
	}
}

func TestGateNoTargetOverride(t *testing.T) {
	t.Run("passes when no override key is carried", func(t *testing.T) {
		records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}
		if findings := GateNoTargetOverride(records); len(findings) != 0 {
			t.Errorf("GateNoTargetOverride() = %+v; want empty", findings)
		}
	})

	t.Run("fails once per offending targetDir and buildTags key", func(t *testing.T) {
		records, err := ReadTranscript("testdata/targetdir-override.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}
		findings := GateNoTargetOverride(records)
		if len(findings) != 2 {
			t.Fatalf("GateNoTargetOverride() = %+v; want 2 findings", findings)
		}
		var sawTargetDir, sawBuildTags bool
		for _, finding := range findings {
			if !finding.Fatal {
				t.Errorf("finding = %+v; want fatal", finding)
			}
			if strings.Contains(finding.Message, "targetDir") {
				sawTargetDir = true
			}
			if strings.Contains(finding.Message, "buildTags") {
				sawBuildTags = true
			}
		}
		if !sawTargetDir || !sawBuildTags {
			t.Errorf("findings = %+v; want one naming targetDir and one naming buildTags", findings)
		}
	})
}

func assistantRecordsWithModel(model string) []Record {
	return []Record{
		{Type: "assistant", Message: Message{Model: model}},
	}
}

func TestGateModelPinned(t *testing.T) {
	t.Run("passes on exact match", func(t *testing.T) {
		findings := GateModelPinned(assistantRecordsWithModel("claude-opus-5"), "claude-opus-5")
		if len(findings) != 0 {
			t.Errorf("GateModelPinned() = %+v; want empty", findings)
		}
	})

	t.Run("passes on bracketed context-window suffix", func(t *testing.T) {
		findings := GateModelPinned(assistantRecordsWithModel("claude-opus-5[1m]"), "claude-opus-5")
		if len(findings) != 0 {
			t.Errorf("GateModelPinned() = %+v; want empty", findings)
		}
	})

	t.Run("fails on mismatch", func(t *testing.T) {
		findings := GateModelPinned(assistantRecordsWithModel("claude-sonnet-5"), "claude-opus-5")
		if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "model_pinned" {
			t.Errorf("GateModelPinned() = %+v; want one fatal model_pinned finding", findings)
		}
	})

	t.Run("fails on no assistant record without error or panic", func(t *testing.T) {
		records := []Record{{Type: "user"}}
		findings := GateModelPinned(records, "claude-opus-5")
		if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "model_pinned" {
			t.Errorf("GateModelPinned() = %+v; want one fatal model_pinned finding", findings)
		}
	})
}

func TestGateBlinding(t *testing.T) {
	const repoRoot = "/home/user/quarry"

	t.Run("fatal on an mcp__quarry__ tool name", func(t *testing.T) {
		records := []Record{
			{
				Type: "assistant",
				Message: Message{
					Content: []ContentBlock{{Type: "tool_use", ToolUseID: "tu_1", Name: "mcp__quarry__toc_file"}},
				},
			},
		}
		findings := GateBlinding(records, repoRoot)
		if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "blinding" {
			t.Errorf("GateBlinding() = %+v; want one fatal blinding finding", findings)
		}
	})

	t.Run("fatal on a repo-root path", func(t *testing.T) {
		records := []Record{
			{
				Type: "assistant",
				Message: Message{
					Content: []ContentBlock{{Type: "text", Text: "Reading " + repoRoot + "/internal/foo.go"}},
				},
			},
		}
		findings := GateBlinding(records, repoRoot)
		if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "blinding" {
			t.Errorf("GateBlinding() = %+v; want one fatal blinding finding", findings)
		}
	})

	t.Run("fatal on a bare mention outside a tool_result", func(t *testing.T) {
		records := []Record{
			{
				Type: "assistant",
				Message: Message{
					Content: []ContentBlock{{Type: "text", Text: "This project is quarry."}},
				},
			},
		}
		findings := GateBlinding(records, repoRoot)
		if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "blinding" {
			t.Errorf("GateBlinding() = %+v; want one fatal blinding finding", findings)
		}
	})

	t.Run("non-fatal observation for a mention confined to a tool_result", func(t *testing.T) {
		records, err := ReadTranscript("testdata/none-target-origin-mention.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}
		findings := GateBlinding(records, repoRoot)
		if len(findings) != 1 || findings[0].Fatal || findings[0].Gate != "target_origin_quarry_mention" {
			t.Errorf("GateBlinding() = %+v; want one non-fatal target_origin_quarry_mention finding", findings)
		}
	})

	t.Run("short-circuits: an unconditional check plus a bare mention yields no bare-mention finding", func(t *testing.T) {
		records := []Record{
			{
				Type: "assistant",
				Message: Message{
					Content: []ContentBlock{
						{Type: "tool_use", ToolUseID: "tu_1", Name: "mcp__quarry__toc_file"},
						{Type: "text", Text: "This is a bare mention of quarry too."},
					},
				},
			},
		}
		findings := GateBlinding(records, repoRoot)
		if len(findings) != 1 {
			t.Fatalf("GateBlinding() = %+v; want exactly one finding from the short-circuit", findings)
		}
		if findings[0].Gate != "blinding" || !findings[0].Fatal {
			t.Errorf("findings[0] = %+v; want the fatal mcp__quarry__ finding, not a bare-mention finding", findings[0])
		}
	})
}

func TestUsedDaemonBackedTool(t *testing.T) {
	t.Run("false for a toc-only transcript", func(t *testing.T) {
		records, err := ReadTranscript("testdata/cold-native-fallback.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}
		if UsedDaemonBackedTool(records) {
			t.Error("UsedDaemonBackedTool() = true; want false")
		}
	})

	t.Run("true for a daemon-backed call", func(t *testing.T) {
		records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}
		if !UsedDaemonBackedTool(records) {
			t.Error("UsedDaemonBackedTool() = false; want true")
		}
	})
}

func assistantRecordsCount(n int) []Record {
	records := make([]Record, n)
	for i := range records {
		records[i] = Record{Type: "assistant"}
	}
	return records
}

func TestGateMaxTurns(t *testing.T) {
	tests := []struct {
		name      string
		turns     int
		maxTurns  int
		wantFatal bool
	}{
		{"at the limit passes", 5, 5, false},
		{"one below the limit passes", 4, 5, false},
		{"one above the limit is fatal", 6, 5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := GateMaxTurns(assistantRecordsCount(tt.turns), tt.maxTurns)
			if tt.wantFatal {
				if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "max_turns" {
					t.Errorf("GateMaxTurns() = %+v; want one fatal max_turns finding", findings)
				}
			} else if len(findings) != 0 {
				t.Errorf("GateMaxTurns() = %+v; want empty", findings)
			}
		})
	}
}

func TestGateRunPrompt(t *testing.T) {
	task := "> Explain the thing."
	good := []Record{{Type: "user", Message: Message{Content: ContentBlocks{{Type: "text", Text: PARALLEL_OPENING + "\n\nbody\n\n" + task + "\n\nschema"}}}}}
	if f := GateRunPrompt(good, task); len(f) != 0 {
		t.Errorf("good prompt flagged: %+v", f)
	}
	description := []Record{{Type: "user", Message: Message{Content: ContentBlocks{{Type: "text", Text: "ladderbench run a1-toc-file rep 4 attempt 1"}}}}}
	if f := GateRunPrompt(description, task); len(f) != 1 || !f[0].Fatal || f[0].Gate != "run_prompt" {
		t.Errorf("dispatch-description prompt not flagged: %+v", f)
	}
	noTask := []Record{{Type: "user", Message: Message{Content: ContentBlocks{{Type: "text", Text: PARALLEL_OPENING + " something else"}}}}}
	if f := GateRunPrompt(noTask, task); len(f) != 1 || !f[0].Fatal {
		t.Errorf("prompt without task text not flagged: %+v", f)
	}
	if f := GateRunPrompt(description, ""); len(f) != 0 {
		t.Errorf("empty taskText must disable the gate: %+v", f)
	}
	if f := GateRunPrompt(nil, task); len(f) != 1 || !f[0].Fatal {
		t.Errorf("no user record not flagged: %+v", f)
	}
}
