// toc_test.go covers TOCFile: language resolution (extension-based and langOverride), the
// docstring-trimming policy driven by Options.DocSentences, the header first-paragraph truncation,
// range stability across DocSentences values, and every error route (unsupported extension,
// designed-but-unimplemented language, nonexistent path, invalid UTF-8, and a partial parse). Every
// fixture is written into a t.TempDir(), since TOCFile is the first code in this package that
// touches disk.

package toc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine"
)

// writeTempFile writes content to name inside a fresh t.TempDir() and returns the file's full path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, ...) failed: %v", path, err)
	}
	return path
}

// TestTOCFile_GoFileWithSymbols asserts the full FileTOC shape for a Go file carrying two symbols.
func TestTOCFile_GoFileWithSymbols(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"// Foo does a thing.\n" +
		"func Foo() {}\n" +
		"\n" +
		"// Bar does another thing.\n" +
		"func Bar() {}\n"
	path := writeTempFile(t, "foo.go", src)

	got, err := TOCFile(path, "", Options{DocSentences: 1})
	if err != nil {
		t.Fatalf("TOCFile(%q, \"\", ...) returned error: %v", path, err)
	}
	if got.Language != "go" {
		t.Errorf("Language = %q; want %q", got.Language, "go")
	}
	if got.Partial {
		t.Error("Partial = true; want false")
	}
	if len(got.Symbols) != 2 {
		t.Fatalf("len(Symbols) = %d; want 2", len(got.Symbols))
	}
	if got.Symbols[0].Name != "Foo" || got.Symbols[1].Name != "Bar" {
		t.Errorf("Symbols names = [%q, %q]; want [\"Foo\", \"Bar\"]", got.Symbols[0].Name, got.Symbols[1].Name)
	}
	for i := 1; i < len(got.Symbols); i++ {
		if got.Symbols[i-1].Start >= got.Symbols[i].Start {
			t.Errorf("Symbols[%d].Start = %d >= Symbols[%d].Start = %d; want ascending", i-1, got.Symbols[i-1].Start, i, got.Symbols[i].Start)
		}
	}
}

// TestTOCFile_HeaderTruncation covers the header-truncation cases: a multi-line header with a blank
// line inside it (first paragraph only, no declarations), and a header with no blank line (returned
// whole).
func TestTOCFile_HeaderTruncation(t *testing.T) {
	t.Run("MultiParagraphHeaderNoDeclarations", func(t *testing.T) {
		src := "// Package p does things.\n" +
			"// This is still the first paragraph.\n" +
			"//\n" +
			"// This is a second paragraph that must not appear.\n" +
			"package p\n"
		path := writeTempFile(t, "doc.go", src)

		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Package p does things.\nThis is still the first paragraph."
		if got.Header != want {
			t.Errorf("Header = %q; want %q", got.Header, want)
		}
		if len(got.Symbols) != 0 {
			t.Errorf("len(Symbols) = %d; want 0", len(got.Symbols))
		}
	})

	t.Run("HeaderWithNoBlankLineReturnedWhole", func(t *testing.T) {
		src := "// Package p does things.\n" +
			"// on two lines.\n" +
			"package p\n"
		path := writeTempFile(t, "doc.go", src)

		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Package p does things.\non two lines."
		if got.Header != want {
			t.Errorf("Header = %q; want %q", got.Header, want)
		}
	})
}

// TestTOCFile_DocSentencesPolicy covers Options.DocSentences: the default of 1, AllSentences, 0, and
// an N larger than the sentence count.
func TestTOCFile_DocSentencesPolicy(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"// Foo does a thing. It has a second sentence. And a third.\n" +
		"func Foo() {}\n"
	path := writeTempFile(t, "foo.go", src)

	t.Run("DefaultOneSentence", func(t *testing.T) {
		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Foo does a thing."
		if got.Symbols[0].Docstring != want {
			t.Errorf("Docstring = %q; want %q", got.Symbols[0].Docstring, want)
		}
	})

	t.Run("AllSentences", func(t *testing.T) {
		got, err := TOCFile(path, "", Options{DocSentences: AllSentences})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Foo does a thing. It has a second sentence. And a third."
		if got.Symbols[0].Docstring != want {
			t.Errorf("Docstring = %q; want %q", got.Symbols[0].Docstring, want)
		}
	})

	t.Run("ZeroOmitsEveryDocstring", func(t *testing.T) {
		got, err := TOCFile(path, "", Options{DocSentences: 0})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		for i, sym := range got.Symbols {
			if sym.Docstring != "" {
				t.Errorf("Symbols[%d].Docstring = %q; want empty", i, sym.Docstring)
			}
		}
	})

	t.Run("NLargerThanSentenceCountReturnsWholeNoError", func(t *testing.T) {
		got, err := TOCFile(path, "", Options{DocSentences: 100})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Foo does a thing. It has a second sentence. And a third."
		if got.Symbols[0].Docstring != want {
			t.Errorf("Docstring = %q; want %q", got.Symbols[0].Docstring, want)
		}
	})
}

// TestTOCFile_DocSentencesDoesNotSplitOnAbbreviationOrBacktickIdentifier covers two of
// FirstSentences' exclusion shapes, exercised through TOCFile under the default DocSentences of 1.
func TestTOCFile_DocSentencesDoesNotSplitOnAbbreviationOrBacktickIdentifier(t *testing.T) {
	t.Run("AbbreviationInFirstSentence", func(t *testing.T) {
		src := "package p\n" +
			"\n" +
			"// Foo handles e.g. this case and returns. Second sentence.\n" +
			"func Foo() {}\n"
		path := writeTempFile(t, "foo.go", src)

		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Foo handles e.g. this case and returns."
		if got.Symbols[0].Docstring != want {
			t.Errorf("Docstring = %q; want %q", got.Symbols[0].Docstring, want)
		}
	})

	t.Run("BacktickQuotedDottedIdentifier", func(t *testing.T) {
		src := "package p\n" +
			"\n" +
			"// Foo calls `a.b.c` internally. Second sentence.\n" +
			"func Foo() {}\n"
		path := writeTempFile(t, "foo.go", src)

		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Foo calls `a.b.c` internally."
		if got.Symbols[0].Docstring != want {
			t.Errorf("Docstring = %q; want %q", got.Symbols[0].Docstring, want)
		}
	})
}

// TestTOCFile_RangesStableAcrossDocSentences asserts Start, SigEnd, and End are identical across
// DocSentences values 0, 1, and AllSentences for the same fixture — no range shrinks with the
// emitted text.
func TestTOCFile_RangesStableAcrossDocSentences(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"// Foo does a thing. It has a second sentence.\n" +
		"func Foo() {\n" +
		"\treturn\n" +
		"}\n"
	path := writeTempFile(t, "foo.go", src)

	var results []FileTOC
	for _, n := range []int{0, 1, AllSentences} {
		got, err := TOCFile(path, "", Options{DocSentences: n})
		if err != nil {
			t.Fatalf("TOCFile(DocSentences: %d) returned error: %v", n, err)
		}
		results = append(results, got)
	}

	want := results[0].Symbols[0]
	for i, got := range results {
		sym := got.Symbols[0]
		if sym.Start != want.Start || sym.SigEnd != want.SigEnd || sym.End != want.End {
			t.Errorf("results[%d].Symbols[0] ranges = {Start:%d SigEnd:%d End:%d}; want {Start:%d SigEnd:%d End:%d}",
				i, sym.Start, sym.SigEnd, sym.End, want.Start, want.SigEnd, want.End)
		}
	}
}

// TestTOCFile_Package asserts Package matches the fixture's package clause, and is empty when the
// package clause is lost under a Partial parse.
func TestTOCFile_Package(t *testing.T) {
	t.Run("NormalFile", func(t *testing.T) {
		path := writeTempFile(t, "foo.go", "package mypkg\n\nfunc F() {}\n")
		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		if got.Package != "mypkg" {
			t.Errorf("Package = %q; want %q", got.Package, "mypkg")
		}
	})

	t.Run("PackageClauseLostUnderPartialParse", func(t *testing.T) {
		// A file whose very first token is broken swallows the package clause itself under
		// tree-sitter's error recovery.
		src := "packag mypkg\n\nfunc F() {}\n"
		path := writeTempFile(t, "foo.go", src)
		got, err := TOCFile(path, "", Options{DocSentences: 1})
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		if got.Package != "" {
			t.Errorf("Package = %q; want empty for a fixture whose package clause is lost", got.Package)
		}
	})
}

// TestTOCFile_SigEndOrderingInvariant asserts, across every symbol of a multi-symbol fixture, that a
// symbol with a body satisfies Start <= SigEnd <= End and a bodyless symbol has SigEnd == 0 — the
// property a consumer relies on.
func TestTOCFile_SigEndOrderingInvariant(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"func WithBody() {\n" +
		"\treturn\n" +
		"}\n" +
		"\n" +
		"type Alias = int\n" +
		"\n" +
		"type S struct {\n" +
		"\tX int\n" +
		"}\n"
	path := writeTempFile(t, "foo.go", src)
	got, err := TOCFile(path, "", Options{DocSentences: 1})
	if err != nil {
		t.Fatalf("TOCFile returned error: %v", err)
	}
	if len(got.Symbols) != 3 {
		t.Fatalf("len(Symbols) = %d; want 3", len(got.Symbols))
	}
	for _, sym := range got.Symbols {
		if sym.SigEnd == 0 {
			continue
		}
		if !(sym.Start <= sym.SigEnd && sym.SigEnd <= sym.End) {
			t.Errorf("symbol %q: Start=%d SigEnd=%d End=%d; want Start <= SigEnd <= End", sym.Name, sym.Start, sym.SigEnd, sym.End)
		}
	}
	if got.Symbols[1].SigEnd != 0 {
		t.Errorf("bodyless symbol %q SigEnd = %d; want 0", got.Symbols[1].Name, got.Symbols[1].SigEnd)
	}
}

// TestTOCFile_UnsupportedExtension asserts a .md file returns a wrapped ErrLanguageUnsupported.
func TestTOCFile_UnsupportedExtension(t *testing.T) {
	path := writeTempFile(t, "readme.md", "# Title\n")
	_, err := TOCFile(path, "", Options{DocSentences: 1})
	if !errors.Is(err, quarryengine.ErrLanguageUnsupported) {
		t.Errorf("TOCFile(%q) error = %v; want errors.Is(err, ErrLanguageUnsupported)", path, err)
	}
}

// TestTOCFile_DesignedButUnimplementedLanguage asserts a .rs file returns the same wrapped
// ErrLanguageUnsupported, proving a designed-but-unimplemented language is not a silent empty
// result.
func TestTOCFile_DesignedButUnimplementedLanguage(t *testing.T) {
	path := writeTempFile(t, "main.rs", "fn main() {}\n")
	_, err := TOCFile(path, "", Options{DocSentences: 1})
	if !errors.Is(err, quarryengine.ErrLanguageUnsupported) {
		t.Errorf("TOCFile(%q) error = %v; want errors.Is(err, ErrLanguageUnsupported)", path, err)
	}
}

// TestTOCFile_LangOverrideWinsOverExtensionMismatch asserts a langOverride of "go" on a .py file
// parses with the Go grammar and does not error on the extension mismatch.
func TestTOCFile_LangOverrideWinsOverExtensionMismatch(t *testing.T) {
	path := writeTempFile(t, "foo.py", "package p\n\nfunc Foo() {}\n")
	got, err := TOCFile(path, "go", Options{DocSentences: 1})
	if err != nil {
		t.Fatalf("TOCFile(%q, \"go\", ...) returned error: %v", path, err)
	}
	if got.Language != "go" {
		t.Errorf("Language = %q; want %q", got.Language, "go")
	}
}

// TestTOCFile_NonexistentPath asserts a nonexistent path returns a wrapped os error.
func TestTOCFile_NonexistentPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.go")
	_, err := TOCFile(path, "", Options{DocSentences: 1})
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("TOCFile(%q) error = %v; want errors.Is(err, os.ErrNotExist)", path, err)
	}
}

// TestTOCFile_InvalidUTF8 asserts a file whose bytes are not valid UTF-8 returns an error naming the
// path, and not a Partial result.
func TestTOCFile_InvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	// 0xFF is never valid as a standalone UTF-8 byte.
	if err := os.WriteFile(path, []byte("package p\n\xff\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, ...) failed: %v", path, err)
	}
	_, err := TOCFile(path, "", Options{DocSentences: 1})
	if err == nil {
		t.Fatal("TOCFile returned nil error for invalid UTF-8 content; want an error naming the path")
	}
	if got := err.Error(); !strings.Contains(got, path) {
		t.Errorf("error %q does not name the path %q", got, path)
	}
}

// TestTOCFile_PartialParseReturnsSurvivingSymbols asserts a file with a syntax error that swallows a
// later declaration still reports Partial true with the surviving symbols returned.
func TestTOCFile_PartialParseReturnsSurvivingSymbols(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"func Broken(\n" +
		"\n" +
		"func Recovered() {}\n"
	path := writeTempFile(t, "foo.go", src)
	got, err := TOCFile(path, "", Options{DocSentences: 1})
	if err != nil {
		t.Fatalf("TOCFile returned error: %v", err)
	}
	if !got.Partial {
		t.Error("Partial = false; want true for a deliberately broken fixture")
	}
	if len(got.Symbols) == 0 {
		t.Error("len(Symbols) = 0; want the surviving symbols returned")
	}
}
