// toc_test.go covers Repo.TOC: the header first-paragraph truncation, package resolution, sigend
// ordering, invalid UTF-8, partial parses, file ordering, subdirectory identity-only answers,
// non-code file listing, empty directories, the test/generated flags, and unreadable files. Every
// fixture lives under .scratch/engine-tests/ via writeScratchTree, per the fixture-split Shared
// Decision; t.TempDir() is never used here.

package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// openRepo opens root as a Repo, failing the test on error.
func openRepo(t *testing.T, root string) *Repo {
	t.Helper()
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", root, err)
	}
	return r
}

// openScratchRepo writes files under .scratch/engine-tests/<name>/ via writeScratchTree and opens
// the resulting tree as a Repo.
func openScratchRepo(t *testing.T, name string, files map[string]string) *Repo {
	t.Helper()
	root := writeScratchTree(t, name, files)
	return openRepo(t, root)
}

// entryByName returns the FileEntry named name from files, failing the test if it is absent.
func entryByName(t *testing.T, files []FileEntry, name string) FileEntry {
	t.Helper()
	for _, f := range files {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no FileEntry named %q in %+v", name, files)
	return FileEntry{}
}

// boolPtr returns a pointer to b, for TOCOptions.Symbols.
func boolPtr(b bool) *bool {
	return &b
}

// TestRepoTOC_GoFileWithSymbols asserts the full file-entry shape for a Go file target carrying
// two symbols. The fixture sits under a "pkg" subdirectory, not the scratch root itself: the
// repository root's own glyph unit is "" (dirRel == "."), and "" is never spellable
// (checkGoUnit's ReasonUnitEmpty), so a root-level file could never carry symbols at all.
func TestRepoTOC_GoFileWithSymbols(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"// Foo does a thing.\n" +
		"func Foo() {}\n" +
		"\n" +
		"// Bar does another thing.\n" +
		"func Bar() {}\n"
	r := openScratchRepo(t, "toc-go-file-with-symbols", map[string]string{"pkg/foo.go": src})

	got, err := r.TOC("pkg/foo.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", "pkg/foo.go", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Files) = %d; want 1", len(got.Files))
	}
	entry := got.Files[0]
	if entry.Symbols == nil {
		t.Fatal("Symbols == nil; want a non-nil pointer for a file target's default")
	}
	symbols := *entry.Symbols
	if len(symbols) != 2 {
		t.Fatalf("len(Symbols) = %d; want 2", len(symbols))
	}
	wantIDs := []string{"pkg#Foo", "pkg#Bar"}
	for i, want := range wantIDs {
		if symbols[i].ID != want {
			t.Errorf("Symbols[%d].ID = %q; want %q", i, symbols[i].ID, want)
		}
	}
	for i := 1; i < len(symbols); i++ {
		if symbols[i-1].Start >= symbols[i].Start {
			t.Errorf("Symbols[%d].Start = %d >= Symbols[%d].Start = %d; want ascending", i-1, symbols[i-1].Start, i, symbols[i].Start)
		}
	}
}

// TestRepoTOC_HeaderTruncation covers the header-truncation cases, over both a file target and a
// directory target: a multi-line header with a blank line inside it (first paragraph only, no
// declarations), and a header with no blank line (returned whole).
func TestRepoTOC_HeaderTruncation(t *testing.T) {
	t.Run("MultiParagraphHeaderNoDeclarations", func(t *testing.T) {
		src := "// Package p does things.\n" +
			"// This is still the first paragraph.\n" +
			"//\n" +
			"// This is a second paragraph that must not appear.\n" +
			"package p\n"
		// Under "pkg/", not the scratch root: the root's own unit ("") is never spellable, and this
		// subtest asserts the spellable-unit shape — a non-nil, empty Symbols slice — not the
		// unspellable one, which glyph_test.go covers separately.
		r := openScratchRepo(t, "toc-header-multi-paragraph", map[string]string{"pkg/doc.go": src})

		got, err := r.TOC("pkg/doc.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		want := "Package p does things.\nThis is still the first paragraph."
		entry := got.Files[0]
		if entry.Header != want {
			t.Errorf("Header = %q; want %q", entry.Header, want)
		}
		if entry.Symbols == nil || len(*entry.Symbols) != 0 {
			t.Errorf("Symbols = %v; want a non-nil, empty slice", entry.Symbols)
		}
	})

	t.Run("HeaderWithNoBlankLineReturnedWhole", func(t *testing.T) {
		src := "// Package p does things.\n" +
			"// on two lines.\n" +
			"package p\n"
		r := openScratchRepo(t, "toc-header-no-blank-line", map[string]string{"doc.go": src})

		got, err := r.TOC("doc.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		want := "Package p does things.\non two lines."
		if got.Files[0].Header != want {
			t.Errorf("Header = %q; want %q", got.Files[0].Header, want)
		}
	})

	t.Run("DirectoryTargetHeaderFirstParagraphOnly", func(t *testing.T) {
		src := "// First paragraph line one.\n" +
			"//\n" +
			"// Second paragraph must not appear.\n" +
			"package p\n"
		r := openScratchRepo(t, "toc-dir-header-first-paragraph", map[string]string{"doc.go": src})

		got, err := r.TOC(".", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		want := "First paragraph line one."
		if got.Files[0].Header != want {
			t.Errorf("Header = %q; want %q", got.Files[0].Header, want)
		}
	})
}

// TestRepoTOC_Package asserts the directory's Package matches a lone file's package clause, and is
// empty when the clause is lost under a partial parse.
func TestRepoTOC_Package(t *testing.T) {
	t.Run("NormalFile", func(t *testing.T) {
		r := openScratchRepo(t, "toc-package-normal", map[string]string{"foo.go": "package mypkg\n\nfunc F() {}\n"})
		got, err := r.TOC("foo.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if got.Package != "mypkg" {
			t.Errorf("Package = %q; want %q", got.Package, "mypkg")
		}
		if got.Files[0].Package != "" {
			t.Errorf("Files[0].Package = %q; want empty — it matches the directory's own package", got.Files[0].Package)
		}
	})

	t.Run("PackageClauseLostUnderPartialParse", func(t *testing.T) {
		// A file whose very first token is broken swallows the package clause itself under
		// tree-sitter's error recovery, so it casts no vote and the directory has no package.
		src := "packag mypkg\n\nfunc F() {}\n"
		r := openScratchRepo(t, "toc-package-lost", map[string]string{"foo.go": src})
		got, err := r.TOC("foo.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if got.Package != "" {
			t.Errorf("Package = %q; want empty for a fixture whose package clause is lost", got.Package)
		}
	})
}

// TestRepoTOC_SigEndOrderingInvariant asserts, across every symbol of a multi-symbol fixture, that
// a symbol with a body satisfies Start <= SigEnd <= End and a bodyless symbol has SigEnd == 0.
func TestRepoTOC_SigEndOrderingInvariant(t *testing.T) {
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
	r := openScratchRepo(t, "toc-sigend-ordering", map[string]string{"pkg/foo.go": src})
	got, err := r.TOC("pkg/foo.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	symbols := *got.Files[0].Symbols
	if len(symbols) != 3 {
		t.Fatalf("len(Symbols) = %d; want 3", len(symbols))
	}
	for _, sym := range symbols {
		if sym.SigEnd == 0 {
			continue
		}
		if !(sym.Start <= sym.SigEnd && sym.SigEnd <= sym.End) {
			t.Errorf("symbol %q: Start=%d SigEnd=%d End=%d; want Start <= SigEnd <= End", sym.ID, sym.Start, sym.SigEnd, sym.End)
		}
	}
	if symbols[1].SigEnd != 0 {
		t.Errorf("bodyless symbol %q SigEnd = %d; want 0", symbols[1].ID, symbols[1].SigEnd)
	}
}

// TestRepoTOC_InvalidUTF8 asserts a file target whose bytes are not valid UTF-8 answers with no
// top-level error, an Error naming the file, and no Header or Lossy.
func TestRepoTOC_InvalidUTF8(t *testing.T) {
	// 0xFF is never valid as a standalone UTF-8 byte; writeScratchTree writes strings, so this
	// fixture is created directly to carry an invalid byte writeScratchTree's map cannot.
	root := writeScratchTree(t, "toc-invalid-utf8", map[string]string{"placeholder.txt": "x"})
	path := filepath.Join(root, "foo.go")
	if err := os.WriteFile(path, []byte("package p\n\xff\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, ...) failed: %v", path, err)
	}
	r := openRepo(t, root)

	got, err := r.TOC("foo.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v; want nil — invalid UTF-8 is an Error entry, not a top-level failure", err)
	}
	entry := got.Files[0]
	if entry.Error == "" {
		t.Fatal("Error is empty; want it set for invalid UTF-8")
	}
	if !strings.Contains(entry.Error, "foo.go") {
		t.Errorf("Error %q does not name foo.go", entry.Error)
	}
	if entry.Header != "" || entry.Lossy {
		t.Errorf("Header=%q Lossy=%v; want both unset", entry.Header, entry.Lossy)
	}
}

// TestRepoTOC_PartialParseReturnsSurvivingSymbols asserts a file target with a syntax error that
// swallows a later declaration still reports Lossy true with the surviving symbols returned.
func TestRepoTOC_PartialParseReturnsSurvivingSymbols(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"func Broken(\n" +
		"\n" +
		"func Recovered() {}\n"
	r := openScratchRepo(t, "toc-partial-parse", map[string]string{"pkg/foo.go": src})
	got, err := r.TOC("pkg/foo.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	entry := got.Files[0]
	if !entry.Lossy {
		t.Error("Lossy = false; want true for a deliberately broken fixture")
	}
	if entry.Symbols == nil || len(*entry.Symbols) == 0 {
		t.Error("Symbols is empty; want the surviving symbols returned")
	}
}

// TestRepoTOC_FileOrdering asserts a directory's Files come back in lexicographic order by base
// filename regardless of creation order, with the directory's own package and language carried at
// the directory level and omitted per file since every file agrees with it.
func TestRepoTOC_FileOrdering(t *testing.T) {
	r := openScratchRepo(t, "toc-file-ordering", map[string]string{
		"zebra.go":  "package p\n\nfunc F() {}\n",
		"apple.go":  "package p\n\nfunc G() {}\n",
		"middle.go": "package p\n\nfunc H() {}\n",
	})

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if got.Package != "p" || got.Language != "go" {
		t.Errorf("Package=%q Language=%q; want %q, %q", got.Package, got.Language, "p", "go")
	}
	if len(got.Files) != 3 {
		t.Fatalf("len(Files) = %d; want 3", len(got.Files))
	}
	wantOrder := []string{"apple.go", "middle.go", "zebra.go"}
	for i, name := range wantOrder {
		if got.Files[i].Name != name {
			t.Errorf("Files[%d].Name = %q; want %q", i, got.Files[i].Name, name)
		}
		if got.Files[i].Language != "" || got.Files[i].Package != "" {
			t.Errorf("Files[%d] Language=%q Package=%q; want both empty, matching the directory", i, got.Files[i].Language, got.Files[i].Package)
		}
	}
}

// TestRepoTOC_SubdirectoryNotListedNotRecursed asserts a subdirectory never appears in Files, is
// listed in Dirs as an identity-plus-doc answer, and carries no Files or Dirs of its own at
// Depth 0.
func TestRepoTOC_SubdirectoryNotListedNotRecursed(t *testing.T) {
	r := openScratchRepo(t, "toc-subdir-not-recursed", map[string]string{
		"top.go":        "package p\n",
		"sub/nested.go": "package p\n",
	})

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(got.Files) != 1 || got.Files[0].Name != "top.go" {
		t.Fatalf("Files = %+v; want only top.go", got.Files)
	}
	if len(got.Dirs) != 1 || got.Dirs[0].Dir != "sub" {
		t.Fatalf("Dirs = %+v; want one identity-only entry for \"sub\"", got.Dirs)
	}
	if len(got.Dirs[0].Files) != 0 || len(got.Dirs[0].Dirs) != 0 {
		t.Errorf("Dirs[0] Files=%v Dirs=%v; want both absent at the depth cut", got.Dirs[0].Files, got.Dirs[0].Dirs)
	}
}

// TestRepoTOC_SubdirectoryNamesListedSortedNoContentsLeak asserts multiple subdirectories are
// reported by Dir name only, sorted lexicographically regardless of creation order, with none of
// their own contents leaking into the parent's Files.
func TestRepoTOC_SubdirectoryNamesListedSortedNoContentsLeak(t *testing.T) {
	r := openScratchRepo(t, "toc-subdir-sorted", map[string]string{
		"zebra/nested.go":  "package p\n",
		"apple/nested.go":  "package p\n",
		"middle/nested.go": "package p\n",
		"top.go":           "package p\n",
	})

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(got.Dirs) != 3 {
		t.Fatalf("len(Dirs) = %d; want 3", len(got.Dirs))
	}
	wantDirs := []string{"apple", "middle", "zebra"}
	for i, name := range wantDirs {
		if got.Dirs[i].Dir != name {
			t.Errorf("Dirs[%d].Dir = %q; want %q", i, got.Dirs[i].Dir, name)
		}
	}
	if len(got.Files) != 1 || got.Files[0].Name != "top.go" {
		t.Errorf("Files = %+v; want only top.go, no nested.go from any subdirectory", got.Files)
	}
}

// TestRepoTOC_EmptyDirectory asserts a directory with no entry at all returns an answer with no
// Files, no Dirs, and no error.
func TestRepoTOC_EmptyDirectory(t *testing.T) {
	root := writeScratchTree(t, "toc-empty-directory", map[string]string{"keep.txt": "x"})
	emptyDir := filepath.Join(root, "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) failed: %v", emptyDir, err)
	}
	r := openRepo(t, root)

	got, err := r.TOC("empty", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(got.Files) != 0 {
		t.Errorf("len(Files) = %d; want 0", len(got.Files))
	}
	if len(got.Dirs) != 0 {
		t.Errorf("len(Dirs) = %d; want 0", len(got.Dirs))
	}
	if got.Package != "" || got.Doc != "" {
		t.Errorf("Package=%q Doc=%q; want both empty", got.Package, got.Doc)
	}
}

// TestRepoTOC_NonCodeFileIsListedWithHeaderNoSymbols asserts a file whose extension maps to no
// language is now listed — the walk lists every non-gitignored file, not only code files — with a
// header from its own table and no Symbols, even when the query explicitly requests symbols.
func TestRepoTOC_NonCodeFileIsListedWithHeaderNoSymbols(t *testing.T) {
	// Under "pkg/", not the scratch root: the root's own unit ("") is never spellable, and top.go
	// needs a spellable unit here to prove Symbols is populated when requested.
	r := openScratchRepo(t, "toc-non-code-file-listed", map[string]string{
		"pkg/top.go":    "package p\n",
		"pkg/README.md": "# Title\n\nSome prose.\n",
	})

	got, err := r.TOC("pkg", TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(got.Files) != 2 {
		t.Fatalf("len(Files) = %d; want 2", len(got.Files))
	}
	readme := entryByName(t, got.Files, "README.md")
	if readme.Header == "" {
		t.Error("README.md Header is empty; want its markdown header rule to apply")
	}
	if readme.Symbols != nil {
		t.Errorf("README.md Symbols = %v; want nil — a non-code file never gets Symbols", readme.Symbols)
	}
	top := entryByName(t, got.Files, "top.go")
	if top.Symbols == nil {
		t.Error("top.go Symbols == nil; want a non-nil pointer since Symbols was requested")
	}
}

// TestRepoTOC_TestFlag asserts a test-suffixed file gets Test true and a plain file gets Test
// false, as plain bools rather than V1's pointer.
func TestRepoTOC_TestFlag(t *testing.T) {
	r := openScratchRepo(t, "toc-test-flag", map[string]string{
		"foo_test.go": "package p\n\nfunc TestFoo(t *testing.T) {}\n",
		"foo.go":      "package p\n\nfunc Foo() {}\n",
	})

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	testEntry := entryByName(t, got.Files, "foo_test.go")
	if !testEntry.Test {
		t.Error("foo_test.go Test = false; want true")
	}
	plainEntry := entryByName(t, got.Files, "foo.go")
	if plainEntry.Test {
		t.Error("foo.go Test = true; want false")
	}
}

// TestRepoTOC_GeneratedFlagAndHeaderAfterBanner asserts a generated Go file gets Generated true,
// and Header is the block after the banner, not the banner itself.
func TestRepoTOC_GeneratedFlagAndHeaderAfterBanner(t *testing.T) {
	src := "// Code generated by mockgen. DO NOT EDIT.\n" +
		"\n" +
		"// Real header text.\n" +
		"package p\n"
	r := openScratchRepo(t, "toc-generated-flag", map[string]string{"gen.go": src})

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	entry := entryByName(t, got.Files, "gen.go")
	if !entry.Generated {
		t.Error("Generated = false; want true")
	}
	if entry.Header != "Real header text." {
		t.Errorf("Header = %q; want %q", entry.Header, "Real header text.")
	}
}

// TestRepoTOC_UnreadableFileIsListedWithErrorOthersUnaffected asserts a file made unreadable via
// chmod is still listed, with Error set and Header/Lossy unset, while every other file in the
// directory is unaffected. Skipped when running as a user for whom chmod 0000 has no effect (e.g.
// root).
func TestRepoTOC_UnreadableFileIsListedWithErrorOthersUnaffected(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as a privileged user for whom chmod 0000 does not block reads")
	}
	root := writeScratchTree(t, "toc-unreadable-file", map[string]string{
		"readable.go":   "package p\n\nfunc F() {}\n",
		"unreadable.go": "package p\n",
	})
	unreadablePath := filepath.Join(root, "unreadable.go")
	if err := os.Chmod(unreadablePath, 0o000); err != nil {
		t.Fatalf("os.Chmod(%q, 0000) failed: %v", unreadablePath, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadablePath, 0o644)
	})
	if _, err := os.ReadFile(unreadablePath); err == nil {
		t.Skip("chmod 0000 did not block reads in this environment")
	}
	r := openRepo(t, root)

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	unreadable := entryByName(t, got.Files, "unreadable.go")
	if unreadable.Error == "" {
		t.Error("unreadable.go Error is empty; want it set")
	}
	if unreadable.Header != "" || unreadable.Lossy {
		t.Errorf("unreadable.go Header=%q Lossy=%v; want both unset", unreadable.Header, unreadable.Lossy)
	}
	readable := entryByName(t, got.Files, "readable.go")
	if readable.Error != "" {
		t.Errorf("readable.go Error = %q; want empty, unaffected by the unreadable sibling", readable.Error)
	}
}

// TestImplemented_MatchesRegisteredStrategies asserts Implemented() returns exactly the languages
// with a concrete Strategy, in sorted order — a guard against a strategy silently failing to
// register itself, or a stray extra one leaking in from a test.
func TestImplemented_MatchesRegisteredStrategies(t *testing.T) {
	want := []string{"go"}
	got := Implemented()
	if len(got) != len(want) {
		t.Fatalf("Implemented() = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Implemented() = %v; want %v", got, want)
			break
		}
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
