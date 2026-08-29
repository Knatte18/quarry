package mcpserver

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

func TestNativeEntry_Query_PositionForm(t *testing.T) {
	line, char := 42, 8
	entry := nativeEntry{File: "foo.go", Line: &line, Character: &char}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	want := quarry.Query{Pos: &quarry.Position{File: "/target/foo.go", Line: 42, Character: 8}}
	if got.Pos == nil || *got.Pos != *want.Pos {
		t.Errorf("entry.query(\"/target\") = %+v; want %+v (1-based, no ±1 conversion)", got, want)
	}
}

func TestNativeEntry_Query_SymbolAloneForm(t *testing.T) {
	entry := nativeEntry{Symbol: "Foo"}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	want := quarry.Query{Symbol: "Foo"}
	if got != want {
		t.Errorf("entry.query(\"/target\") = %+v; want %+v", got, want)
	}
}

func TestNativeEntry_Query_FilePlusSymbolForm(t *testing.T) {
	entry := nativeEntry{File: "foo.go", Symbol: "Bar"}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	want := quarry.Query{InFile: &quarry.InFileQuery{File: "/target/foo.go", Name: "Bar"}}
	if got.InFile == nil || *got.InFile != *want.InFile {
		t.Errorf("entry.query(\"/target\") = %+v; want %+v", got, want)
	}
}

// TestNativeEntry_Query_FileURIPrefixNotStripped asserts a "file://" prefix on File is left
// unchanged, unlike lspEntry.query's resolveEntryFile, which strips it.
func TestNativeEntry_Query_FileURIPrefixNotStripped(t *testing.T) {
	entry := nativeEntry{Symbol: "Bar", File: "file:///abs/foo.go"}

	got, err := entry.query("/target")
	if err != nil {
		t.Fatalf("entry.query(\"/target\") error = %v", err)
	}
	if got.InFile == nil {
		t.Fatalf("entry.query(\"/target\").InFile = nil; want non-nil")
	}
	want := "/target/file:/abs/foo.go"
	if got.InFile.File != want {
		t.Errorf("entry.query(\"/target\").InFile.File = %q; want %q (file:// prefix not stripped, joined as an ordinary relative path)", got.InFile.File, want)
	}
}

// TestNativeEntry_Query_IllegalCombinations asserts every combination outside the three legal
// forms returns an error rather than a silent guess.
func TestNativeEntry_Query_IllegalCombinations(t *testing.T) {
	line, char := 1, 1

	tests := []struct {
		name  string
		entry nativeEntry
	}{
		{"PositionWithoutFile", nativeEntry{Line: &line, Character: &char}},
		{"NeitherSymbolNorPosition", nativeEntry{}},
		{"FileAlone", nativeEntry{File: "foo.go"}},
		{"BothSymbolAndPosition", nativeEntry{
			Symbol:    "Foo",
			File:      "foo.go",
			Line:      &line,
			Character: &char,
		}},
		{"PositionMissingLine", nativeEntry{File: "foo.go", Character: &char}},
		{"PositionMissingCharacter", nativeEntry{File: "foo.go", Line: &line}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.entry.query("/target"); err == nil {
				t.Errorf("nativeEntry(%+v).query(\"/target\") error = nil; want an error naming the three accepted forms", tt.entry)
			}
		})
	}
}

// TestNativeEntry_UnmarshalJSON_PopulatesFieldsAndRaw asserts UnmarshalJSON populates every
// exported field while also preserving the original bytes in raw.
func TestNativeEntry_UnmarshalJSON_PopulatesFieldsAndRaw(t *testing.T) {
	data := []byte(`{"file":"foo.go","line":3,"character":7,"within":"internal/foo","except":["a.go"]}`)

	var entry nativeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", data, err)
	}

	if entry.File != "foo.go" {
		t.Errorf("entry.File = %q; want %q", entry.File, "foo.go")
	}
	if entry.Line == nil || *entry.Line != 3 {
		t.Errorf("entry.Line = %v; want 3", entry.Line)
	}
	if entry.Character == nil || *entry.Character != 7 {
		t.Errorf("entry.Character = %v; want 7", entry.Character)
	}
	if entry.Within != "internal/foo" {
		t.Errorf("entry.Within = %q; want %q", entry.Within, "internal/foo")
	}
	if len(entry.Except) != 1 || entry.Except[0] != "a.go" {
		t.Errorf("entry.Except = %v; want [\"a.go\"]", entry.Except)
	}
	if string(entry.raw) != string(data) {
		t.Errorf("entry.raw = %s; want %s", entry.raw, data)
	}
}

// TestExceptSet_ResolvesAgainstTargetDirNotProcessCwd asserts exceptSet resolves a relative except
// path against the given targetDir, not the process working directory — the regression this
// function exists to guard against per nativeentry.go's own doc comment.
func TestExceptSet_ResolvesAgainstTargetDirNotProcessCwd(t *testing.T) {
	targetDir := "/some/project"

	got := exceptSet(targetDir, []string{"wrapper.go", "sub/other.go"})

	want := map[string]bool{
		filepath.Clean("/some/project/wrapper.go"):   true,
		filepath.Clean("/some/project/sub/other.go"): true,
	}
	if len(got) != len(want) {
		t.Fatalf("exceptSet(...) = %v; want %v", got, want)
	}
	for k := range want {
		if !got[k] {
			t.Errorf("exceptSet(...) missing key %q resolved against targetDir %q", k, targetDir)
		}
	}
}

// TestExceptSet_AbsolutePathPassesThroughCleaned asserts an already-absolute except path is used
// unchanged (cleaned), never joined onto targetDir.
func TestExceptSet_AbsolutePathPassesThroughCleaned(t *testing.T) {
	got := exceptSet("/some/project", []string{"/elsewhere/wrapper.go"})

	want := filepath.Clean("/elsewhere/wrapper.go")
	if !got[want] {
		t.Errorf("exceptSet(...) = %v; want key %q", got, want)
	}
}
