// toc_test.go covers TOCFile: language resolution (extension-based and langOverride), the header
// first-paragraph truncation, and every error route (unsupported extension, nonexistent path,
// invalid UTF-8, and a partial parse). Every fixture is written into a t.TempDir(), since TOCFile
// is the first code in this package that touches disk.
//
// It also covers TOCDir: single-level listing and ordering, per-file language resolution and
// langOverride restriction, the Error/Partial mutual-exclusion invariant, and every per-file failure
// route, over directories built in a t.TempDir().

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
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

	got, err := TOCFile(path, "")
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

		got, err := TOCFile(path, "")
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

		got, err := TOCFile(path, "")
		if err != nil {
			t.Fatalf("TOCFile returned error: %v", err)
		}
		want := "Package p does things.\non two lines."
		if got.Header != want {
			t.Errorf("Header = %q; want %q", got.Header, want)
		}
	})
}

// TestTOCFile_Package asserts Package matches the fixture's package clause, and is empty when the
// package clause is lost under a Partial parse.
func TestTOCFile_Package(t *testing.T) {
	t.Run("NormalFile", func(t *testing.T) {
		path := writeTempFile(t, "foo.go", "package mypkg\n\nfunc F() {}\n")
		got, err := TOCFile(path, "")
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
		got, err := TOCFile(path, "")
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
	got, err := TOCFile(path, "")
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
	_, err := TOCFile(path, "")
	if !errors.Is(err, ErrLanguageUnsupported) {
		t.Errorf("TOCFile(%q) error = %v; want errors.Is(err, ErrLanguageUnsupported)", path, err)
	}
}

// TestTOCFile_LangOverrideWinsOverExtensionMismatch asserts a langOverride of "go" on a .txt file
// parses with the Go grammar and does not error on the extension mismatch.
func TestTOCFile_LangOverrideWinsOverExtensionMismatch(t *testing.T) {
	path := writeTempFile(t, "foo.txt", "package p\n\nfunc Foo() {}\n")
	got, err := TOCFile(path, "go")
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
	_, err := TOCFile(path, "")
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
	_, err := TOCFile(path, "")
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
	got, err := TOCFile(path, "")
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

// writeDirFile writes content to name inside dir.
func writeDirFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, ...) failed: %v", path, err)
	}
}

// entryByName returns the DirEntry named name from files, failing the test if it is absent.
func entryByName(t *testing.T, files []DirEntry, name string) DirEntry {
	t.Helper()
	for _, f := range files {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no DirEntry named %q in %+v", name, files)
	return DirEntry{}
}

// TestTOCDir_FileOrdering asserts a directory's Files come back in lexicographic order by base
// filename regardless of creation order, each entry carrying its resolved language and package.
func TestTOCDir_FileOrdering(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "zebra.go", "package p\n\nfunc F() {}\n")
	writeDirFile(t, dir, "apple.go", "package p\n\nfunc G() {}\n")
	writeDirFile(t, dir, "middle.go", "package p\n\nfunc H() {}\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir(%q, \"\") returned error: %v", dir, err)
	}
	if len(got.Files) != 3 {
		t.Fatalf("len(Files) = %d; want 3", len(got.Files))
	}
	wantOrder := []string{"apple.go", "middle.go", "zebra.go"}
	for i, name := range wantOrder {
		if got.Files[i].Name != name {
			t.Errorf("Files[%d].Name = %q; want %q", i, got.Files[i].Name, name)
		}
		if got.Files[i].Language != "go" {
			t.Errorf("Files[%d].Language = %q; want %q", i, got.Files[i].Language, "go")
		}
		if got.Files[i].Package != "p" {
			t.Errorf("Files[%d].Package = %q; want %q", i, got.Files[i].Package, "p")
		}
	}
}

// TestTOCDir_SubdirectoryNotListedNotRecursed asserts a subdirectory never appears in Files and is
// never descended into, while its bare name is reported in Dirs.
func TestTOCDir_SubdirectoryNotListedNotRecursed(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "top.go", "package p\n")
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("os.Mkdir(%q) failed: %v", subdir, err)
	}
	writeDirFile(t, subdir, "nested.go", "package p\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Files) = %d; want 1", len(got.Files))
	}
	if got.Files[0].Name != "top.go" {
		t.Errorf("Files[0].Name = %q; want %q", got.Files[0].Name, "top.go")
	}
	if len(got.Dirs) != 1 || got.Dirs[0] != "sub" {
		t.Errorf("Dirs = %v; want [\"sub\"]", got.Dirs)
	}
}

// TestTOCDir_SubdirectoryNamesListedSortedNoContentsLeak asserts multiple subdirectories are
// reported by bare name only, sorted lexicographically regardless of creation order, with none of
// their own contents (files nested inside them) appearing anywhere in the result.
func TestTOCDir_SubdirectoryNamesListedSortedNoContentsLeak(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"zebra", "apple", "middle"} {
		subdir := filepath.Join(dir, name)
		if err := os.Mkdir(subdir, 0o755); err != nil {
			t.Fatalf("os.Mkdir(%q) failed: %v", subdir, err)
		}
		writeDirFile(t, subdir, "nested.go", "package p\n")
	}
	writeDirFile(t, dir, "top.go", "package p\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	wantDirs := []string{"apple", "middle", "zebra"}
	if !reflect.DeepEqual(got.Dirs, wantDirs) {
		t.Errorf("Dirs = %v; want %v", got.Dirs, wantDirs)
	}
	if len(got.Files) != 1 || got.Files[0].Name != "top.go" {
		t.Errorf("Files = %v; want only top.go, no nested.go from any subdirectory", got.Files)
	}
}

// TestTOCDir_NoSubdirectoryEmptyNonNilDirs asserts a directory with no subdirectory returns an
// empty, non-nil Dirs, mirroring Files' own empty-non-nil convention.
func TestTOCDir_NoSubdirectoryEmptyNonNilDirs(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "top.go", "package p\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	if got.Dirs == nil {
		t.Error("Dirs == nil; want empty, non-nil slice")
	}
	if len(got.Dirs) != 0 {
		t.Errorf("len(Dirs) = %d; want 0", len(got.Dirs))
	}
}

// TestTOCDir_NonCodeFileNotListed asserts a file whose extension maps to no language never appears.
func TestTOCDir_NonCodeFileNotListed(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "top.go", "package p\n")
	writeDirFile(t, dir, "README.md", "# Title\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Files) = %d; want 1", len(got.Files))
	}
	if got.Files[0].Name != "top.go" {
		t.Errorf("Files[0].Name = %q; want %q", got.Files[0].Name, "top.go")
	}
}

// TestTOCDir_NoSupportedFileEmptyNonNilFilesNilError asserts a directory with no supported file
// returns an empty, non-nil Files and a nil error.
func TestTOCDir_NoSupportedFileEmptyNonNilFilesNilError(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "README.md", "# Title\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	if got.Files == nil {
		t.Error("Files == nil; want empty, non-nil slice")
	}
	if len(got.Files) != 0 {
		t.Errorf("len(Files) = %d; want 0", len(got.Files))
	}
}

// TestTOCDir_HeaderFirstParagraphOnly asserts a file with a multi-paragraph header gets Header set
// to its first paragraph only.
func TestTOCDir_HeaderFirstParagraphOnly(t *testing.T) {
	dir := t.TempDir()
	src := "// First paragraph line one.\n" +
		"//\n" +
		"// Second paragraph must not appear.\n" +
		"package p\n"
	writeDirFile(t, dir, "doc.go", src)

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	want := "First paragraph line one."
	if got.Files[0].Header != want {
		t.Errorf("Header = %q; want %q", got.Files[0].Header, want)
	}
}

// TestTOCDir_TestPointer asserts a test-suffixed file gets Test pointing to true and a plain file
// gets Test pointing to false — a pointer in both cases, since Go has a test-file rule.
func TestTOCDir_TestPointer(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "foo_test.go", "package p\n\nfunc TestFoo(t *testing.T) {}\n")
	writeDirFile(t, dir, "foo.go", "package p\n\nfunc Foo() {}\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	testEntry := entryByName(t, got.Files, "foo_test.go")
	if testEntry.Test == nil || !*testEntry.Test {
		t.Errorf("foo_test.go Test = %v; want pointer to true", testEntry.Test)
	}
	plainEntry := entryByName(t, got.Files, "foo.go")
	if plainEntry.Test == nil || *plainEntry.Test {
		t.Errorf("foo.go Test = %v; want pointer to false", plainEntry.Test)
	}
}

// TestTOCDir_GeneratedPointerAndHeaderAfterBanner asserts a generated Go file gets Generated
// pointing to true, and Header is the block after the banner, not the banner itself.
func TestTOCDir_GeneratedPointerAndHeaderAfterBanner(t *testing.T) {
	dir := t.TempDir()
	src := "// Code generated by mockgen. DO NOT EDIT.\n" +
		"\n" +
		"// Real header text.\n" +
		"package p\n"
	writeDirFile(t, dir, "gen.go", src)

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	entry := entryByName(t, got.Files, "gen.go")
	if entry.Generated == nil || !*entry.Generated {
		t.Errorf("Generated = %v; want pointer to true", entry.Generated)
	}
	if entry.Header != "Real header text." {
		t.Errorf("Header = %q; want %q", entry.Header, "Real header text.")
	}
}

// TestTOCDir_UnreadableFileIsListedWithErrorOthersUnaffected asserts a file made unreadable via
// chmod is still listed, with Error set and Header/Partial unset, while every other file in the
// directory is unaffected. Skipped when running as a user for whom chmod 0000 has no effect (e.g.
// root).
func TestTOCDir_UnreadableFileIsListedWithErrorOthersUnaffected(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as a privileged user for whom chmod 0000 does not block reads")
	}
	dir := t.TempDir()
	writeDirFile(t, dir, "readable.go", "package p\n\nfunc F() {}\n")
	unreadablePath := filepath.Join(dir, "unreadable.go")
	if err := os.WriteFile(unreadablePath, []byte("package p\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, ...) failed: %v", unreadablePath, err)
	}
	if err := os.Chmod(unreadablePath, 0o000); err != nil {
		t.Fatalf("os.Chmod(%q, 0000) failed: %v", unreadablePath, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadablePath, 0o644)
	})

	if _, err := os.ReadFile(unreadablePath); err == nil {
		t.Skip("chmod 0000 did not block reads in this environment")
	}

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	unreadable := entryByName(t, got.Files, "unreadable.go")
	if unreadable.Error == "" {
		t.Error("unreadable.go Error is empty; want it set")
	}
	if unreadable.Header != "" || unreadable.Partial {
		t.Errorf("unreadable.go Header=%q Partial=%v; want both unset", unreadable.Header, unreadable.Partial)
	}
	readable := entryByName(t, got.Files, "readable.go")
	if readable.Error != "" {
		t.Errorf("readable.go Error = %q; want empty, unaffected by the unreadable sibling", readable.Error)
	}
}

// TestTOCDir_InvalidUTF8IsListedWithErrorNoPartial asserts a file whose bytes are not valid UTF-8 is
// listed with Error set and Partial unset.
func TestTOCDir_InvalidUTF8IsListedWithErrorNoPartial(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "bad.go", "package p\n\xff\n")

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	entry := entryByName(t, got.Files, "bad.go")
	if entry.Error == "" {
		t.Error("Error is empty; want it set for invalid UTF-8")
	}
	if entry.Partial {
		t.Error("Partial = true; want false")
	}
}

// TestTOCDir_SyntaxErrorIsListedWithPartialNoError asserts a file with a syntax error is listed with
// Partial true and Error empty, and that the two are never both set.
func TestTOCDir_SyntaxErrorIsListedWithPartialNoError(t *testing.T) {
	dir := t.TempDir()
	src := "package p\n" +
		"\n" +
		"func Broken(\n" +
		"\n" +
		"func Recovered() {}\n"
	writeDirFile(t, dir, "broken.go", src)

	got, err := TOCDir(dir, "")
	if err != nil {
		t.Fatalf("TOCDir returned error: %v", err)
	}
	entry := entryByName(t, got.Files, "broken.go")
	if !entry.Partial {
		t.Error("Partial = false; want true")
	}
	if entry.Error != "" {
		t.Errorf("Error = %q; want empty — Error and Partial are mutually exclusive", entry.Error)
	}
}

// TestTOCDir_LangOverrideRestrictsListing asserts a langOverride of "go" lists only the files with
// a Go extension.
func TestTOCDir_LangOverrideRestrictsListing(t *testing.T) {
	dir := t.TempDir()
	writeDirFile(t, dir, "a.go", "package p\n")
	writeDirFile(t, dir, "b.txt", "package p\n")

	got, err := TOCDir(dir, "go")
	if err != nil {
		t.Fatalf("TOCDir(dir, \"go\") returned error: %v", err)
	}
	if len(got.Files) != 1 || got.Files[0].Name != "a.go" {
		t.Fatalf("Files = %v; want only a.go", got.Files)
	}
}

// TestImplemented_MatchesRegisteredStrategies asserts Implemented() returns exactly the languages
// with a concrete Strategy, in sorted order — a guard against a strategy silently failing to
// register itself, or a stray extra one leaking in from a test.
func TestImplemented_MatchesRegisteredStrategies(t *testing.T) {
	want := []string{"go"}
	got := Implemented()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Implemented() = %v; want %v", got, want)
	}
}

// TestExtensionLanguages_AllHaveGrammars is the guard that matters: it fails the moment the
// extension map and the wired grammar set disagree, which is how a sixth language would otherwise
// get half-added — an extension resolving to a language name the backend cannot parse, surfacing as
// a confusing runtime error rather than a build-time one.
//
// It also asserts every name Implemented() reports appears in ExtensionLanguages(), so a
// strategy can never be registered under a name no extension resolves to.
func TestExtensionLanguages_AllHaveGrammars(t *testing.T) {
	for _, lang := range ExtensionLanguages() {
		if !treesitter.Supported(lang) {
			t.Errorf("ExtensionLanguages() includes %q, but treesitter.Supported(%q) = false", lang, lang)
		}
	}

	extensionLangs := make(map[string]bool)
	for _, lang := range ExtensionLanguages() {
		extensionLangs[lang] = true
	}
	for _, lang := range Implemented() {
		if !extensionLangs[lang] {
			t.Errorf("Implemented() includes %q, which is not in ExtensionLanguages()", lang)
		}
	}
}
