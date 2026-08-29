package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

func TestLSPEntry_Query_PositionForm(t *testing.T) {
	line, char := 4, 9
	entry := lspEntry{
		TextDocument: &textDocumentIdentifier{URI: "foo.go"},
		Position:     &lspPosition{Line: &line, Character: &char},
	}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	want := quarry.Query{Pos: &quarry.Position{File: "/target/foo.go", Line: 5, Character: 10}}
	if got.Pos == nil || *got.Pos != *want.Pos {
		t.Errorf("entry.query(\"/target\") = %+v; want %+v", got, want)
	}
}

func TestLSPEntry_Query_SymbolAloneForm(t *testing.T) {
	entry := lspEntry{Symbol: "Foo"}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	want := quarry.Query{Symbol: "Foo"}
	if got != want {
		t.Errorf("entry.query(\"/target\") = %+v; want %+v", got, want)
	}
}

func TestLSPEntry_Query_TextDocumentPlusSymbolForm(t *testing.T) {
	entry := lspEntry{
		TextDocument: &textDocumentIdentifier{URI: "file:///abs/foo.go"},
		Symbol:       "Bar",
	}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	want := quarry.Query{InFile: &quarry.InFileQuery{File: "/abs/foo.go", Name: "Bar"}}
	if got.InFile == nil || *got.InFile != *want.InFile {
		t.Errorf("entry.query(\"/target\") = %+v; want %+v", got, want)
	}
}

// TestLSPEntry_Query_IllegalCombinations asserts every combination outside the three legal forms
// returns an error rather than a silent guess.
func TestLSPEntry_Query_IllegalCombinations(t *testing.T) {
	line, char := 0, 0

	tests := []struct {
		name  string
		entry lspEntry
	}{
		{"PositionWithoutTextDocument", lspEntry{Position: &lspPosition{Line: &line, Character: &char}}},
		{"NeitherSymbolNorPosition", lspEntry{}},
		{"BothSymbolAndPosition", lspEntry{
			Symbol:       "Foo",
			TextDocument: &textDocumentIdentifier{URI: "foo.go"},
			Position:     &lspPosition{Line: &line, Character: &char},
		}},
		{"PositionMissingLine", lspEntry{
			TextDocument: &textDocumentIdentifier{URI: "foo.go"},
			Position:     &lspPosition{Character: &char},
		}},
		{"PositionMissingCharacter", lspEntry{
			TextDocument: &textDocumentIdentifier{URI: "foo.go"},
			Position:     &lspPosition{Line: &line},
		}},
		{"EmptyURIWithPosition", lspEntry{
			TextDocument: &textDocumentIdentifier{URI: ""},
			Position:     &lspPosition{Line: &line, Character: &char},
		}},
		{"EmptyURIWithSymbol", lspEntry{
			TextDocument: &textDocumentIdentifier{URI: ""},
			Symbol:       "Foo",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.entry.query("/target"); err == nil {
				t.Errorf("lspEntry(%+v).query(\"/target\") error = nil; want an error naming the three accepted forms", tt.entry)
			}
		})
	}
}

// TestLSPEntry_UnmarshalJSON_PopulatesFieldsAndRaw asserts UnmarshalJSON populates every exported
// field while also preserving the original bytes in raw.
func TestLSPEntry_UnmarshalJSON_PopulatesFieldsAndRaw(t *testing.T) {
	data := []byte(`{"textDocument":{"uri":"foo.go"},"position":{"line":3,"character":7},"within":"internal/foo"}`)

	var entry lspEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", data, err)
	}

	if entry.TextDocument == nil || entry.TextDocument.URI != "foo.go" {
		t.Errorf("entry.TextDocument = %+v; want URI \"foo.go\"", entry.TextDocument)
	}
	if entry.Position == nil || entry.Position.Line == nil || *entry.Position.Line != 3 {
		t.Errorf("entry.Position.Line = %v; want 3", entry.Position)
	}
	if entry.Position == nil || entry.Position.Character == nil || *entry.Position.Character != 7 {
		t.Errorf("entry.Position.Character = %v; want 7", entry.Position)
	}
	if entry.Within != "internal/foo" {
		t.Errorf("entry.Within = %q; want %q", entry.Within, "internal/foo")
	}
	if string(entry.raw) != string(data) {
		t.Errorf("entry.raw = %s; want %s", entry.raw, data)
	}
}

func TestRunTargets_OrderPreservedOneResultPerInput(t *testing.T) {
	targets := []int{10, 20, 30, 40}

	got := runTargets(targets, func(i int, v int) int { return v + i })

	want := []int{10, 21, 32, 43}
	if len(got) != len(want) {
		t.Fatalf("runTargets(...) returned %d results; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("runTargets(...)[%d] = %d; want %d", i, got[i], want[i])
		}
	}
}
