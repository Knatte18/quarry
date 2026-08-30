package ladder

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestModelAlias(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    string
		wantErr bool
	}{
		{"opus", "claude-opus-5", "opus", false},
		{"sonnet", "claude-sonnet-5", "sonnet", false},
		{"unmapped", "claude-haiku-5", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ModelAlias(tt.modelID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ModelAlias(%q) error = %v; wantErr %t", tt.modelID, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ModelAlias(%q) = %q; want %q", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestRunAgentDefinition_BlindedConfigNamesNoQuarry(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	name, body, err := RunAgentDefinition(l, a0, "claude-opus-5")
	if err != nil {
		t.Fatalf("RunAgentDefinition(l, a0, ...) = _, _, %v; want nil error", err)
	}
	combined := strings.ToLower(name + "\n" + body)
	if strings.Contains(combined, "quarry") {
		t.Errorf("RunAgentDefinition(l, a0, ...) name/body contains %q, case-insensitive; want none", "quarry")
	}
	if strings.Contains(body, MCPPrefix) {
		t.Errorf("RunAgentDefinition(l, a0, ...) body contains prefixed name %q; want none", MCPPrefix)
	}
}

func TestRunAgentDefinition_AllowlistIsExactlyAllowedToolsPlusBaseFour(t *testing.T) {
	l := mustLoadLadder(t)
	b5, err := ConfigByID(l, "b5-impact")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b5-impact", err)
	}
	name, body, err := RunAgentDefinition(l, b5, "claude-opus-5")
	if err != nil {
		t.Fatalf("RunAgentDefinition(l, b5, ...) = _, _, %v; want nil error", err)
	}

	tools, err := extractToolsFromDoc(t, body)
	if err != nil {
		t.Fatalf("extractToolsFromDoc: %v", err)
	}

	want := append([]string{}, baseRunTools...)
	for _, tool := range b5.Allowed {
		want = append(want, MCPName(tool))
	}
	sort.Strings(want)
	sort.Strings(tools)
	if !reflect.DeepEqual(tools, want) {
		t.Errorf("RunAgentDefinition(l, b5, ...) tools = %v; want %v", tools, want)
	}
	if name != b5.ID {
		t.Errorf("RunAgentDefinition(l, b5, ...) name = %q; want %q", name, b5.ID)
	}
}

func TestRunAgentDefinition_NeverGrantsTask(t *testing.T) {
	l := mustLoadLadder(t)
	for _, config := range l.Configs {
		_, body, err := RunAgentDefinition(l, config, "claude-opus-5")
		if err != nil {
			t.Fatalf("RunAgentDefinition(l, %q, ...) = _, _, %v; want nil error", config.ID, err)
		}
		tools, err := extractToolsFromDoc(t, body)
		if err != nil {
			t.Fatalf("extractToolsFromDoc(%q): %v", config.ID, err)
		}
		if stringSliceContains(tools, "Task") {
			t.Errorf("RunAgentDefinition(l, %q, ...) tools = %v; want no \"Task\"", config.ID, tools)
		}
	}
}

func TestRunAgentDefinition_UnmappedModelIDErrors(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	if _, _, err := RunAgentDefinition(l, a0, "unknown-model"); err == nil {
		t.Error("RunAgentDefinition(l, a0, \"unknown-model\") = _, _, nil; want an error")
	}
}

// extractToolsFromDoc writes body to a temp file and reads its tools: frontmatter back through
// GrantedToolsFromDefinition, so this test file exercises the generator and the reader through the same
// round trip every other caller does.
func extractToolsFromDoc(t *testing.T, body string) ([]string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "def.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return GrantedToolsFromDefinition(path)
}

func TestGrantedToolsFromDefinition_RoundTripsBlindedAndRung(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	b5, err := ConfigByID(l, "b5-impact")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b5-impact", err)
	}

	for _, config := range []LadderConfig{a0, b5} {
		_, body, err := RunAgentDefinition(l, config, "claude-opus-5")
		if err != nil {
			t.Fatalf("RunAgentDefinition(l, %q, ...) = _, _, %v; want nil error", config.ID, err)
		}
		tools, err := extractToolsFromDoc(t, body)
		if err != nil {
			t.Fatalf("GrantedToolsFromDefinition round trip for %q: %v", config.ID, err)
		}

		want := append([]string{}, baseRunTools...)
		for _, tool := range config.Allowed {
			want = append(want, MCPName(tool))
		}
		sort.Strings(want)
		sort.Strings(tools)
		if !reflect.DeepEqual(tools, want) {
			t.Errorf("round trip for %q tools = %v; want %v", config.ID, tools, want)
		}
	}
}

func TestGrantedToolsFromDefinition_ErrorsOnMalformedInput(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "missing.md")
	if _, err := GrantedToolsFromDefinition(missing); err == nil {
		t.Error("GrantedToolsFromDefinition(missing file) = _, nil; want an error")
	}

	noFrontmatter := filepath.Join(dir, "no-frontmatter.md")
	if err := os.WriteFile(noFrontmatter, []byte("just a body, no frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", noFrontmatter, err)
	}
	if _, err := GrantedToolsFromDefinition(noFrontmatter); err == nil {
		t.Error("GrantedToolsFromDefinition(no frontmatter) = _, nil; want an error")
	}

	unparseable := filepath.Join(dir, "unparseable.md")
	if err := os.WriteFile(unparseable, []byte("---\nname: [this is not valid yaml\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", unparseable, err)
	}
	if _, err := GrantedToolsFromDefinition(unparseable); err == nil {
		t.Error("GrantedToolsFromDefinition(unparseable) = _, nil; want an error")
	}

	missingKey := filepath.Join(dir, "missing-key.md")
	if err := os.WriteFile(missingKey, []byte("---\nname: x\ndescription: y\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", missingKey, err)
	}
	if _, err := GrantedToolsFromDefinition(missingKey); err == nil {
		t.Error("GrantedToolsFromDefinition(missing tools key) = _, nil; want an error")
	}
}
