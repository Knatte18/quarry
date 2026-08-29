package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestTOCPreflight_InvalidLangFailsWholeCall asserts an unrecognised lang value produces an error,
// which the caller turns into a whole-call failure.
func TestTOCPreflight_InvalidLangFailsWholeCall(t *testing.T) {
	if _, err := tocPreflight("not-a-real-language", docSentences{}); err == nil {
		t.Errorf("tocPreflight(%q, ...) error = nil; want an error naming the valid --lang values", "not-a-real-language")
	}
}

// TestTOCPreflight_InvalidDocSentencesFailsWholeCall asserts an invalid docSentences value produces
// an error even though lang is valid.
func TestTOCPreflight_InvalidDocSentencesFailsWholeCall(t *testing.T) {
	doc := mustUnmarshalDocSentences(t, `"not-a-number"`)
	if _, err := tocPreflight("", doc); err == nil {
		t.Errorf("tocPreflight(\"\", %+v) error = nil; want an error naming the valid doc-sentences forms", doc)
	}
}

// TestTOCPreflight_DocSentencesAsNumberAndAsAllBothPass asserts docSentences given as a JSON number
// and as the string "all" both reach cli.ParseDocSentences as a string and both pass.
func TestTOCPreflight_DocSentencesAsNumberAndAsAllBothPass(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"Number", "3", "3"},
		{"All", `"all"`, "all"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustUnmarshalDocSentences(t, tt.raw)
			got, err := tocPreflight("", doc)
			if err != nil {
				t.Fatalf("tocPreflight(\"\", %+v) error = %v", doc, err)
			}
			if got != tt.want {
				t.Errorf("tocPreflight(\"\", %+v) = %q; want %q", doc, got, tt.want)
			}
		})
	}
}

// TestTOCPreflight_UnsetDocSentencesSkipsParse asserts an unset docSentences returns an empty
// string without reaching cli.ParseDocSentences at all — an invalid value would fail this test
// only if the skip did not happen, since docSentences{} carries no value to parse.
func TestTOCPreflight_UnsetDocSentencesSkipsParse(t *testing.T) {
	got, err := tocPreflight("", docSentences{})
	if err != nil {
		t.Fatalf("tocPreflight(\"\", docSentences{}) error = %v; want nil", err)
	}
	if got != "" {
		t.Errorf("tocPreflight(\"\", docSentences{}) = %q; want \"\" (unset docSentences skips the parse)", got)
	}
}

// TestTOCStat_MissingPathIsNotFound asserts a missing path yields statusNotFound in both
// directions.
func TestTOCStat_MissingPathIsNotFound(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	for _, wantDir := range []bool{false, true} {
		status, _, err := tocStat(missing, wantDir)
		if err == nil {
			t.Fatalf("tocStat(%q, %v) error = nil; want non-nil for a missing path", missing, wantDir)
		}
		if status != statusNotFound {
			t.Errorf("tocStat(%q, %v) status = %q; want %q", missing, wantDir, status, statusNotFound)
		}
	}
}

// TestTOCStat_WrongTypeIsErrorInEachDirection asserts a directory passed where a file is wanted,
// and a file passed where a directory is wanted, both yield statusError.
func TestTOCStat_WrongTypeIsErrorInEachDirection(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.go")
	if err := os.WriteFile(file, []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", file, err)
	}

	status, message, err := tocStat(dir, false)
	if err == nil {
		t.Fatalf("tocStat(%q, false) error = nil; want non-nil for a directory passed where a file is wanted", dir)
	}
	if status != statusError {
		t.Errorf("tocStat(%q, false) status = %q; want %q", dir, status, statusError)
	}
	if message == "" {
		t.Errorf("tocStat(%q, false) message = \"\"; want a non-empty message", dir)
	}

	status, message, err = tocStat(file, true)
	if err == nil {
		t.Fatalf("tocStat(%q, true) error = nil; want non-nil for a file passed where a directory is wanted", file)
	}
	if status != statusError {
		t.Errorf("tocStat(%q, true) status = %q; want %q", file, status, statusError)
	}
	if message == "" {
		t.Errorf("tocStat(%q, true) message = \"\"; want a non-empty message", file)
	}
}

// TestTOCStat_MatchingTypeReturnsNilError asserts a stat that succeeds with a matching type
// returns a nil error, signalling the caller should proceed to its facade call.
func TestTOCStat_MatchingTypeReturnsNilError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.go")
	if err := os.WriteFile(file, []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", file, err)
	}

	if _, _, err := tocStat(file, false); err != nil {
		t.Errorf("tocStat(%q, false) error = %v; want nil", file, err)
	}
	if _, _, err := tocStat(dir, true); err != nil {
		t.Errorf("tocStat(%q, true) error = %v; want nil", dir, err)
	}
}

// mustUnmarshalDocSentences decodes raw into a docSentences value via its own UnmarshalJSON, so
// these tests exercise the same decode path a real MCP call goes through.
func mustUnmarshalDocSentences(t *testing.T, raw string) docSentences {
	t.Helper()
	var doc docSentences
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", raw, err)
	}
	return doc
}
