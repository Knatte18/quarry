package ladder

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestRequiredScoreFields_MatchRuleExample asserts that the derived required-field set for each rule
// equals exactly the field names that rule's own fenced example declares, so the rule text and its
// validator cannot drift.
func TestRequiredScoreFields_MatchRuleExample(t *testing.T) {
	tests := []struct {
		schema string
		rule   string
		want   []string
	}{
		{"exploration", ExplorationRule, []string{"recall", "precision", "summary_matches"}},
		{"impact", ImpactRule, []string{"recall", "precision", "decoy_admitted", "lookalikes_matched"}},
	}

	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			got := scoreFieldsFromRule(tt.rule)
			if len(got) != len(tt.want) {
				t.Fatalf("scoreFieldsFromRule(%s) = %v, want %v", tt.schema, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("scoreFieldsFromRule(%s)[%d] = %q, want %q", tt.schema, i, got[i], tt.want[i])
				}
			}

			require, ok := requiredScoreFieldsBySchema[tt.schema]
			if !ok {
				t.Fatalf("requiredScoreFieldsBySchema[%q] missing", tt.schema)
			}
			if len(require) != len(tt.want) {
				t.Errorf("requiredScoreFieldsBySchema[%q] = %v, want %v", tt.schema, require, tt.want)
			}
		})
	}
}

// TestStripFasitMeta_RealTask01Fasit asserts that meta stripping removes only the top-level "_meta"
// key and leaves every other field byte-identical, run against the real task 01 fasit.
func TestStripFasitMeta_RealTask01Fasit(t *testing.T) {
	data, err := os.ReadFile("../../../tasks/01-reed-geometry-exploration.fasit.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var fasit map[string]any
	if err := json.Unmarshal(data, &fasit); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := fasit["_meta"]; !ok {
		t.Fatalf("fixture fasit does not carry a _meta key, test setup is stale")
	}

	stripped := StripFasitMeta(fasit)

	if _, ok := stripped["_meta"]; ok {
		t.Errorf("StripFasitMeta() result still carries _meta")
	}
	if len(stripped) != len(fasit)-1 {
		t.Errorf("StripFasitMeta() removed %d keys, want exactly 1 (_meta)", len(fasit)-len(stripped))
	}
	for key, want := range fasit {
		if key == "_meta" {
			continue
		}
		got, ok := stripped[key]
		if !ok {
			t.Errorf("StripFasitMeta() dropped field %q, want it left verbatim", key)
			continue
		}
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("StripFasitMeta() field %q = %s, want byte-identical %s", key, gotJSON, wantJSON)
		}
	}
}

func TestParseScorerReply(t *testing.T) {
	t.Run("well_formed_reply_accepted", func(t *testing.T) {
		reply := "Here is my score:\n\n```json\n" +
			`{"recall": 0.5, "precision": 0.8, "summary_matches": true}` +
			"\n```\n"
		record, err := ParseScorerReply(reply, "exploration")
		if err != nil {
			t.Fatalf("ParseScorerReply() error = %v, want nil", err)
		}
		if record["recall"] != 0.5 {
			t.Errorf("ParseScorerReply() record[recall] = %v, want 0.5", record["recall"])
		}
	})

	t.Run("missing_required_field_rejected", func(t *testing.T) {
		reply := "```json\n" + `{"recall": 0.5, "precision": 0.8}` + "\n```\n"
		_, err := ParseScorerReply(reply, "exploration")
		if err == nil {
			t.Fatalf("ParseScorerReply() error = nil, want an error naming the missing field")
		}
		if !strings.Contains(err.Error(), "summary_matches") {
			t.Errorf("ParseScorerReply() error = %v, want it to name the missing field summary_matches", err)
		}
	})
}

func TestRedactAnswer(t *testing.T) {
	in := RedactionInput{
		QuarryTools:      []string{"toc"},
		ServerName:       "quarry",
		MCPPrefix:        "mcp__quarry__",
		QuarryRepoRoot:   "/home/example/quarry-repo",
		TaskWorktreePath: "/tmp/loomyard-eval-01",
	}

	answer := `The agent called mcp__quarry__toc and toc directly, ` +
		`mentioned the quarry server, read files under /home/example/quarry-repo and ` +
		`/tmp/loomyard-eval-01, and followed the standard protocol throughout.`

	redacted := RedactAnswer(answer, in)

	for _, mustGo := range []string{"mcp__quarry__toc", "/home/example/quarry-repo", "/tmp/loomyard-eval-01"} {
		if strings.Contains(redacted, mustGo) {
			t.Errorf("RedactAnswer() result still contains %q:\n%s", mustGo, redacted)
		}
	}
	// "toc" and "quarry" are checked as whole-word matches, not substrings: "protocol" legitimately
	// contains the substring "toc" and must survive redaction untouched.
	for _, bareToken := range []string{"toc", "quarry"} {
		if MatchesBareToken(redacted, bareToken) {
			t.Errorf("RedactAnswer() result still contains the bare token %q:\n%s", bareToken, redacted)
		}
	}
	if !strings.Contains(redacted, "protocol") {
		t.Errorf("RedactAnswer() result dropped the word \"protocol\", want it left intact:\n%s", redacted)
	}
	if !strings.Contains(redacted, RedactionPlaceholder) {
		t.Errorf("RedactAnswer() result does not contain the redaction placeholder at all:\n%s", redacted)
	}
}

func TestBuildScorerPrompt_FourPartsInOrderWithFencedBlocks(t *testing.T) {
	fasit := map[string]any{
		"_meta":          map[string]any{"role": "reference"},
		"relevant_files": []any{"FASITMARKERXYZ"},
	}
	redactedAnswer := `{"summary":"<redacted> did the work"}`

	prompt, err := BuildScorerPrompt(ExplorationRule, "TASKTEXTMARKER", fasit, redactedAnswer)
	if err != nil {
		t.Fatalf("BuildScorerPrompt() error = %v", err)
	}

	if strings.Contains(prompt, "_meta") {
		t.Errorf("BuildScorerPrompt() result still carries the fasit's _meta block:\n%s", prompt)
	}

	parts := []string{ExplorationRule, "## Task", "TASKTEXTMARKER", "## Reference fasit", "FASITMARKERXYZ", "## Answer to score", redactedAnswer}
	lastIdx := -1
	for _, part := range parts {
		idx := strings.Index(prompt, part)
		if idx == -1 {
			t.Fatalf("BuildScorerPrompt() does not contain %q:\n%s", part, prompt)
		}
		if idx <= lastIdx {
			t.Errorf("BuildScorerPrompt() part %q appears out of order (index %d, previous %d)", part, idx, lastIdx)
		}
		lastIdx = idx
	}

	fasitFenceIdx := strings.Index(prompt, "## Reference fasit")
	answerFenceIdx := strings.Index(prompt, "## Answer to score")
	fasitBlock := prompt[fasitFenceIdx:answerFenceIdx]
	if !strings.Contains(fasitBlock, "```json") {
		t.Errorf("BuildScorerPrompt() fasit section is not a fenced json block:\n%s", fasitBlock)
	}
	answerBlock := prompt[answerFenceIdx:]
	if !strings.Contains(answerBlock, "```json") {
		t.Errorf("BuildScorerPrompt() answer section is not a fenced json block:\n%s", answerBlock)
	}
}
