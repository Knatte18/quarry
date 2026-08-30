package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/* REDACTION */

func TestRedactText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "PreservesImpactAsCommonNoun",
			text: "the impact of this refactor is limited to one package",
			want: "the impact of this refactor is limited to one package",
		},
		{
			name: "RedactsMCPPrefixedName",
			text: "called mcp__quarry__toc_file to inspect the directory",
			want: "called <tool> to inspect the directory",
		},
		{
			name: "RedactsMCPPrefixedImpact",
			text: "corroborated by mcp__quarry__impact",
			want: "corroborated by <tool>",
		},
		{
			name: "RedactsBareToolNameExceptImpact",
			text: "ran toc_dir against the workspace",
			want: "ran <tool> against the workspace",
		},
		{
			name: "DoesNotRedactBareImpact",
			text: "the impact is that ctx must be threaded through",
			want: "the impact is that ctx must be threaded through",
		},
		{
			name: "RedactsQuarryCLIShellForm",
			text: "ran quarry impact on singlellm.go:39:2 to confirm",
			want: "ran <tool> on singlellm.go:39:2 to confirm",
		},
		{
			name: "RedactsTmpQuarryBenchPath",
			text: "fixtures live under /tmp/quarry-bench for this run",
			want: "fixtures live under <tool> for this run",
		},
		{
			name: "RedactsTargetDirFlag",
			text: "invoked with --target-dir=/tmp/loomyard-eval-01 set",
			want: "invoked with <tool> set",
		},
		{
			name: "RedactsBareQuarryWord",
			text: "quarry's toc_file tool found nothing",
			want: "<tool>'s <tool> tool found nothing",
		},
		{
			// "QUARRY TOC_FILE" is consumed as one unit by the "quarry <verb>" CLI shell-form
			// alternative, since TOC_FILE matches its [A-Za-z_]+ tail.
			name: "IsCaseInsensitive",
			text: "QUARRY TOC_FILE returned nothing",
			want: "<tool> returned nothing",
		},
		{
			name: "CollapsesAdjacentRedactionTokens",
			text: "mcp__quarry__toc_dir mcp__quarry__toc_file returned nothing",
			want: "<tool> returned nothing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactText(tt.text); got != tt.want {
				t.Errorf("RedactText(%q) = %q; want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestRedactText_MostSpecificFormClaimsMatchFirst(t *testing.T) {
	// "quarry impact" must be consumed by the CLI shell-form alternative as one unit, rather than the
	// bare "quarry" alternative matching first and leaving "impact" unredacted.
	text := "ran quarry impact against the tree"
	redacted := RedactText(text)
	if strings.Contains(strings.ToLower(redacted), "impact") {
		t.Errorf("RedactText(%q) = %q; still contains impact", text, redacted)
	}
	if strings.Contains(strings.ToLower(redacted), "quarry") {
		t.Errorf("RedactText(%q) = %q; still contains quarry", text, redacted)
	}
}

var impactAnswer = map[string]any{
	"callers_to_update": []any{
		map[string]any{
			"file": "internal/shedadapters/singlellm.go",
			"line": float64(143),
			"evidence": "ran quarry impact on singlellm.go:39:2 (--within internal/shedadapters), " +
				"corroborated by mcp__quarry__impact. The impact of this change is that " +
				"ctx must be threaded through the call chain.",
		},
	},
	"excluded_lookalikes": []any{
		map[string]any{
			"file":   "internal/shedadapters/burler.go",
			"line":   float64(373),
			"reason": "Resolves to a different interface, confirmed with mcp__quarry__textDocument_definition, not quarry impact.",
		},
	},
	"open_questions": []any{"Does mcp__quarry__workspace_symbol reach this via bouncer.go too?"},
	"confidence":     "high",
}

var explorationAnswer = map[string]any{
	"relevant_files": []any{"internal/reedcli/attach.go", "internal/reedengine/attach.go"},
	"key_symbols": []any{
		map[string]any{
			"name": "attachCmd",
			"file": "internal/reedcli/attach.go",
			"role": "Reads terminal size via mcp__quarry__toc_file and calls Engine.AttachArgv.",
		},
	},
	"summary":        "Uses quarry's toc_file and workspace_symbol tools plus /tmp/quarry-bench fixtures to trace geometry.",
	"confidence":     "high",
	"open_questions": []any{"Does mcp__quarry__impact ever get called here?"},
}

func TestRedactAnswer_PreservesCommonNounImpactAndStripsToolProvenance(t *testing.T) {
	redacted := RedactAnswer(impactAnswer)
	callers := redacted["callers_to_update"].([]any)
	evidence := callers[0].(map[string]any)["evidence"].(string)

	if !strings.Contains(evidence, "The impact of this change is that ctx must be threaded through the call chain.") {
		t.Errorf("evidence = %q; want the common-noun impact sentence preserved", evidence)
	}
	if strings.Contains(evidence, "mcp__quarry__impact") {
		t.Errorf("evidence = %q; want no mcp__quarry__impact", evidence)
	}
	if strings.Contains(strings.ToLower(evidence), "quarry") {
		t.Errorf("evidence = %q; want no quarry", evidence)
	}
	if !strings.Contains(evidence, RedactionToken) {
		t.Errorf("evidence = %q; want at least one %s", evidence, RedactionToken)
	}

	caller := callers[0].(map[string]any)
	if caller["file"] != "internal/shedadapters/singlellm.go" {
		t.Errorf("caller[file] = %v; want internal/shedadapters/singlellm.go", caller["file"])
	}
	if caller["line"] != float64(143) {
		t.Errorf("caller[line] = %v; want 143", caller["line"])
	}
	if redacted["confidence"] != "high" {
		t.Errorf("confidence = %v; want high", redacted["confidence"])
	}
}

func TestRedactAnswer_PreservesRelevantFilesAndRedactsSummary(t *testing.T) {
	redacted := RedactAnswer(explorationAnswer)

	gotFiles, _ := json.Marshal(redacted["relevant_files"])
	wantFiles, _ := json.Marshal(explorationAnswer["relevant_files"])
	if string(gotFiles) != string(wantFiles) {
		t.Errorf("relevant_files = %s; want %s", gotFiles, wantFiles)
	}
	if redacted["summary"] == explorationAnswer["summary"] {
		t.Errorf("summary was not redacted: %v", redacted["summary"])
	}
	if strings.Contains(strings.ToLower(redacted["summary"].(string)), "quarry") {
		t.Errorf("summary = %q; want no quarry", redacted["summary"])
	}
}

func TestRedactAnswer_LeavesNoQuarryTraceAnywhere(t *testing.T) {
	for _, answer := range []map[string]any{impactAnswer, explorationAnswer} {
		blob, err := json.Marshal(RedactAnswer(answer))
		if err != nil {
			t.Fatalf("json.Marshal(RedactAnswer(answer)) = _, %v; want nil error", err)
		}
		lower := strings.ToLower(string(blob))
		if strings.Contains(lower, "quarry") {
			t.Errorf("redacted answer still contains quarry: %s", blob)
		}
	}
}

func TestWriteRedacted_LeavesOriginalAnswerByteIdenticalAndWritesRedactedCopy(t *testing.T) {
	runDir := t.TempDir()
	originalBytes, err := json.MarshalIndent(impactAnswer, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent(impactAnswer) = _, %v; want nil error", err)
	}
	originalBytes = append(originalBytes, '\n')
	if err := os.WriteFile(filepath.Join(runDir, "answer.json"), originalBytes, 0o644); err != nil {
		t.Fatalf("os.WriteFile(answer.json) = %v; want nil error", err)
	}

	redacted, err := WriteRedacted(runDir)
	if err != nil {
		t.Fatalf("WriteRedacted(%q) = _, %v; want nil error", runDir, err)
	}

	gotOriginal, err := os.ReadFile(filepath.Join(runDir, "answer.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(answer.json) = _, %v; want nil error", err)
	}
	if string(gotOriginal) != string(originalBytes) {
		t.Errorf("answer.json was modified: got %s; want %s", gotOriginal, originalBytes)
	}

	onDiskBytes, err := os.ReadFile(filepath.Join(runDir, "answer.redacted.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(answer.redacted.json) = _, %v; want nil error", err)
	}
	var onDisk map[string]any
	if err := json.Unmarshal(onDiskBytes, &onDisk); err != nil {
		t.Fatalf("json.Unmarshal(answer.redacted.json) = %v; want nil error", err)
	}
	gotJSON, _ := json.Marshal(onDisk)
	wantJSON, _ := json.Marshal(redacted)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("on-disk redacted answer = %s; want %s", gotJSON, wantJSON)
	}
	if strings.Contains(strings.ToLower(string(onDiskBytes)), "quarry") {
		t.Errorf("answer.redacted.json still contains quarry: %s", onDiskBytes)
	}
}

/* SCORING RULES, FASIT STRIPPING, AND PROMPT ASSEMBLY */

func testLadder() *Ladder {
	tasks := map[string]TaskEntry{
		"01-reed-geometry-exploration": {
			TaskFile:  "bench/loomyard-eval/tasks/01-reed-geometry-exploration.md",
			PinnedSHA: "975578cda8d6f3a81580bd4e73725e060211b766",
			Worktree:  "/tmp/loomyard-eval-01",
			Schema:    "exploration",
			Fasit:     "testdata/fasit-exploration.json",
		},
		"04-shedadapters-shuttle-impact": {
			TaskFile:  "bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md",
			PinnedSHA: "975578cda8d6f3a81580bd4e73725e060211b766",
			Worktree:  "/tmp/loomyard-eval-04",
			Schema:    "impact",
			Fasit:     "testdata/fasit-impact.json",
		},
	}
	configs := []LadderConfig{
		{ID: "a5-bundle", Ladder: "a", Task: "01-reed-geometry-exploration", Allowed: []string{"toc_dir", "toc_file", "workspace_symbol"}},
		{ID: "b7-bundle", Ladder: "b", Task: "04-shedadapters-shuttle-impact", Allowed: []string{"impact", "assert_no_callers"}},
	}
	return &Ladder{
		Reps:                 3,
		Scorer:               ScorerConfig{Model: "claude-opus-5", Effort: "high"},
		QuarryTools:          QuarryTools,
		Tasks:                tasks,
		SourceRepo:           "/home/knatte/Code/loomyard/wts/loomyard",
		ColdWorktreeTemplate: "/tmp/loomyard-eval-01-cold-{n}",
		Configs:              configs,
	}
}

func configByID(l *Ladder, id string) LadderConfig {
	for _, c := range l.Configs {
		if c.ID == id {
			return c
		}
	}
	panic("no config " + id)
}

var impactFasit = map[string]any{
	"_meta": map[string]any{
		"role":     "reference/fasit agent",
		"see_also": "scorecard.md",
	},
	"callers_to_update": []any{
		map[string]any{
			"file":     "internal/shedadapters/singlellm.go",
			"line":     float64(143),
			"evidence": "singlellm.go:143 calls Shuttle.Send directly, which the interface change renames.",
		},
	},
	"excluded_lookalikes": []any{},
}

var explorationFasit = map[string]any{
	"_meta": map[string]any{
		"role": "reference/fasit agent",
	},
	"relevant_files": []any{"internal/reedcli/attach.go"},
	"key_symbols":    []any{},
}

func TestBuildScorerPrompt_NeverLeaksRunIdentity(t *testing.T) {
	l := testLadder()
	config := configByID(l, "b7-bundle")
	redactedAnswer := RedactAnswer(impactAnswer)
	taskText := "Analyze the fallout of the Shuttle interface change across shedadapters."

	prompt, err := BuildScorerPrompt(l, config, redactedAnswer, impactFasit, taskText)
	if err != nil {
		t.Fatalf("BuildScorerPrompt() = _, %v; want nil error", err)
	}

	for _, forbidden := range []string{"b7-bundle", "assert_no_callers", "ladder: b", "ladder b"} {
		if strings.Contains(strings.ToLower(prompt), strings.ToLower(forbidden)) {
			t.Errorf("prompt leaks run identity: contains %q", forbidden)
		}
	}
	if !strings.Contains(prompt, taskText) {
		t.Errorf("prompt does not contain task text")
	}
	answerJSON, _ := json.MarshalIndent(redactedAnswer, "", "  ")
	if !strings.Contains(prompt, string(answerJSON)) {
		t.Errorf("prompt does not contain the redacted answer json")
	}
}

func TestBuildScorerPrompt_StripsFasitMetaButKeepsEvidenceVerbatim(t *testing.T) {
	l := testLadder()
	config := configByID(l, "b7-bundle")
	redactedAnswer := RedactAnswer(impactAnswer)

	prompt, err := BuildScorerPrompt(l, config, redactedAnswer, impactFasit, "task text")
	if err != nil {
		t.Fatalf("BuildScorerPrompt() = _, %v; want nil error", err)
	}

	if strings.Contains(prompt, "_meta") {
		t.Errorf("prompt still contains _meta")
	}
	if strings.Contains(prompt, "reference/fasit agent") {
		t.Errorf("prompt still contains the _meta role text")
	}
	callers := impactFasit["callers_to_update"].([]any)
	wantEvidence := callers[0].(map[string]any)["evidence"].(string)
	if !strings.Contains(prompt, wantEvidence) {
		t.Errorf("prompt does not contain the fasit's verbatim evidence text")
	}
}

func TestStripFasitMeta_RemovesOnlyMeta(t *testing.T) {
	stripped := StripFasitMeta(impactFasit)
	if _, present := stripped["_meta"]; present {
		t.Errorf("StripFasitMeta left _meta present")
	}
	if _, present := stripped["callers_to_update"]; !present {
		t.Errorf("StripFasitMeta dropped callers_to_update")
	}
}

func TestBuildScorerPrompt_SelectsTemplateByTaskSchema(t *testing.T) {
	l := testLadder()

	explorationPrompt, err := BuildScorerPrompt(l, configByID(l, "a5-bundle"), RedactAnswer(explorationAnswer), explorationFasit, "task text")
	if err != nil {
		t.Fatalf("BuildScorerPrompt(exploration) = _, %v; want nil error", err)
	}
	if !strings.Contains(explorationPrompt, strings.TrimSpace(ExplorationRule)) {
		t.Errorf("exploration prompt does not carry ExplorationRule")
	}
	if strings.Contains(explorationPrompt, strings.TrimSpace(ImpactRule)) {
		t.Errorf("exploration prompt unexpectedly carries ImpactRule")
	}

	impactPrompt, err := BuildScorerPrompt(l, configByID(l, "b7-bundle"), RedactAnswer(impactAnswer), impactFasit, "task text")
	if err != nil {
		t.Fatalf("BuildScorerPrompt(impact) = _, %v; want nil error", err)
	}
	if !strings.Contains(impactPrompt, strings.TrimSpace(ImpactRule)) {
		t.Errorf("impact prompt does not carry ImpactRule")
	}
	if strings.Contains(impactPrompt, strings.TrimSpace(ExplorationRule)) {
		t.Errorf("impact prompt unexpectedly carries ExplorationRule")
	}
}

/* SCORER REPLY PARSING */

func TestParseScorerReply_WellFormedReplyForEachSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		reply  string
	}{
		{
			name:   "Exploration",
			schema: "exploration",
			reply:  "```json\n{\"recall\": 0.5, \"precision\": 0.75, \"summary_matches\": true}\n```",
		},
		{
			name:   "Impact",
			schema: "impact",
			reply:  "```json\n{\"recall\": 1.0, \"precision\": 1.0, \"decoy_admitted\": false, \"lookalikes_matched\": 0}\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := ParseScorerReply(tt.reply, tt.schema)
			if err != nil {
				t.Fatalf("ParseScorerReply(%q, %q) = _, %v; want nil error", tt.reply, tt.schema, err)
			}
			if _, present := record["recall"]; !present {
				t.Errorf("record missing recall")
			}
			if _, present := record["precision"]; !present {
				t.Errorf("record missing precision")
			}
		})
	}
}

func TestParseScorerReply_NoFencedBlock(t *testing.T) {
	_, err := ParseScorerReply("I refuse to answer.", "impact")
	if err == nil {
		t.Fatalf("ParseScorerReply() = _, nil; want ScoringError")
	}
	if _, ok := err.(*ScoringError); !ok {
		t.Errorf("ParseScorerReply() error type = %T; want *ScoringError", err)
	}
}

func TestParseScorerReply_UnparseableFencedBlock(t *testing.T) {
	_, err := ParseScorerReply("```json\nnot valid json\n```", "impact")
	if err == nil {
		t.Fatalf("ParseScorerReply() = _, nil; want ScoringError")
	}
	if _, ok := err.(*ScoringError); !ok {
		t.Errorf("ParseScorerReply() error type = %T; want *ScoringError", err)
	}
}

func TestParseScorerReply_MissingSchemaSpecificRequiredField(t *testing.T) {
	// decoy_admitted and lookalikes_matched are impact-schema-specific fields; a reply lacking them
	// must be rejected even though it carries recall/precision.
	reply := "```json\n{\"recall\": 0.5, \"precision\": 0.5}\n```"
	_, err := ParseScorerReply(reply, "impact")
	if err == nil {
		t.Fatalf("ParseScorerReply() = _, nil; want ScoringError for missing decoy_admitted/lookalikes_matched")
	}
}

func TestParseScorerReply_MissingSummariserReadMetric(t *testing.T) {
	// recall/precision are read by the summariser independently of schema; a reply lacking one must be
	// rejected even for the exploration schema.
	reply := "```json\n{\"precision\": 0.5, \"summary_matches\": true}\n```"
	_, err := ParseScorerReply(reply, "exploration")
	if err == nil {
		t.Fatalf("ParseScorerReply() = _, nil; want ScoringError for missing recall")
	}
}
