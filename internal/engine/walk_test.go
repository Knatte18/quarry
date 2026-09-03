// walk_test.go covers walkDir against the committed fixture trees under testdata/ and against
// .scratch/ trees for the gitignore, symlink, and creation-order cases those committed trees
// cannot carry.

package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// repoModuleRoot returns this module's root directory, for tests that walk the committed fixture
// trees under internal/engine/testdata/ via the repository's own real path rather than a
// .scratch/ copy.
func repoModuleRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine this test file's own source location")
	}
	// This file sits at internal/engine/walk_test.go; the module root is two directories up.
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// openModuleRepo opens this module's own root as a Repo.
func openModuleRepo(t *testing.T) *Repo {
	t.Helper()
	return openRepo(t, repoModuleRoot(t))
}

// TestWalk_TreePkgDocFirstParagraphOnly asserts testdata/tree/pkg's directory Doc is doc.go's
// first paragraph only, per the package-doc rule.
func TestWalk_TreePkgDocFirstParagraphOnly(t *testing.T) {
	r := openModuleRepo(t)
	got, err := r.TOC("internal/engine/testdata/tree/pkg", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if got.Package != "pkg" {
		t.Errorf("Package = %q; want %q", got.Package, "pkg")
	}
	want := "Package pkg is the fixture package the walk's tie-break and package-doc tests read."
	if got.Doc != want {
		t.Errorf("Doc = %q; want %q", got.Doc, want)
	}
}

// TestWalk_TreeSubDocAbsent asserts testdata/tree/sub's directory Doc is absent: doc.go's leading
// comment is a file header, not prefixed "Package sub" immediately above the clause, so it is not
// package documentation.
func TestWalk_TreeSubDocAbsent(t *testing.T) {
	r := openModuleRepo(t)
	got, err := r.TOC("internal/engine/testdata/tree/sub", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if got.Package != "sub" {
		t.Errorf("Package = %q; want %q", got.Package, "sub")
	}
	if got.Doc != "" {
		t.Errorf("Doc = %q; want empty", got.Doc)
	}
}

// TestWalk_PackageDeviationKeyOnlyOnExternalTestFile asserts testdata/tree/pkg's per-file Package
// key appears only on export_test.go, whose "pkg_test" clause deviates from the directory's "pkg".
func TestWalk_PackageDeviationKeyOnlyOnExternalTestFile(t *testing.T) {
	r := openModuleRepo(t)
	got, err := r.TOC("internal/engine/testdata/tree/pkg", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	for _, f := range got.Files {
		if f.Name == "export_test.go" {
			if f.Package != "pkg_test" {
				t.Errorf("export_test.go Package = %q; want %q", f.Package, "pkg_test")
			}
			continue
		}
		if f.Package != "" {
			t.Errorf("%s Package = %q; want empty, matching the directory's own package", f.Name, f.Package)
		}
	}
}

// TestWalk_PackageLiterallyNamedHTTPTestIsNotSplit asserts a directory whose package is
// legitimately named "httptest" is not treated as a "_test"-suffixed deviation: the suffix test is
// exact, not a substring match.
func TestWalk_PackageLiterallyNamedHTTPTestIsNotSplit(t *testing.T) {
	r := openScratchRepo(t, "walk-httptest-package", map[string]string{
		"a.go": "package httptest\n\nfunc A() {}\n",
		"b.go": "package httptest\n\nfunc B() {}\n",
	})
	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if got.Package != "httptest" {
		t.Errorf("Package = %q; want %q", got.Package, "httptest")
	}
	for _, f := range got.Files {
		if f.Package != "" {
			t.Errorf("%s Package = %q; want empty — both files agree with the directory", f.Name, f.Package)
		}
	}
}

// TestWalk_TieBreakPicksLexicographicallySmallerClauseDeterministically asserts testdata/tiebreak's
// even split between "alpha" and "main" resolves to "alpha" — the lexicographically smaller clause
// — identically across repeated runs.
func TestWalk_TieBreakPicksLexicographicallySmallerClauseDeterministically(t *testing.T) {
	r := openModuleRepo(t)
	for i := 0; i < 5; i++ {
		got, err := r.TOC("internal/engine/testdata/tiebreak", TOCOptions{})
		if err != nil {
			t.Fatalf("run %d: TOC returned error: %v", i, err)
		}
		if got.Package != "alpha" {
			t.Errorf("run %d: Package = %q; want %q", i, got.Package, "alpha")
		}
	}
}

// TestWalk_OrderingAndSymbolSourceOrder asserts Files and Dirs come back sorted regardless of
// creation order, and that a file's Symbols stay in source (declaration) order rather than being
// resorted.
func TestWalk_OrderingAndSymbolSourceOrder(t *testing.T) {
	root := writeScratchTree(t, "walk-ordering", map[string]string{
		"zebra.go":         "package p\n\nfunc Zeta() {}\n\nfunc Alpha() {}\n",
		"apple.go":         "package p\n",
		"middle/nested.go": "package p\n",
		"alsomid/n.go":     "package p\n",
	})
	r := openRepo(t, root)

	got, err := r.TOC(".", TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	wantFiles := []string{"apple.go", "zebra.go"}
	if len(got.Files) != len(wantFiles) {
		t.Fatalf("Files = %+v; want %v", got.Files, wantFiles)
	}
	for i, name := range wantFiles {
		if got.Files[i].Name != name {
			t.Errorf("Files[%d].Name = %q; want %q", i, got.Files[i].Name, name)
		}
	}
	wantDirs := []string{"alsomid", "middle"}
	if len(got.Dirs) != len(wantDirs) {
		t.Fatalf("Dirs = %+v; want %v", got.Dirs, wantDirs)
	}
	for i, name := range wantDirs {
		if got.Dirs[i].Dir != name {
			t.Errorf("Dirs[%d].Dir = %q; want %q", i, got.Dirs[i].Dir, name)
		}
	}

	zebra := entryByName(t, got.Files, "zebra.go")
	symbols := *zebra.Symbols
	if len(symbols) != 2 || symbols[0].Name != "Zeta" || symbols[1].Name != "Alpha" {
		t.Errorf("zebra.go Symbols = %+v; want [Zeta, Alpha] in source order", symbols)
	}
}

// TestWalk_SymlinksAreNameOnlyAndDepthAllTerminates asserts a symlink to a directory, a symlink to
// a file, and a symlink cycle are each a name-only entry, that DepthAll terminates despite the
// cycle, and that nothing behind a link is listed or parsed.
func TestWalk_SymlinksAreNameOnlyAndDepthAllTerminates(t *testing.T) {
	root := writeScratchTree(t, "walk-symlinks", map[string]string{
		"real.go":        "package p\n\nfunc F() {}\n",
		"reald/inner.go": "package p\n\nfunc Inner() {}\n",
	})
	if err := os.Symlink(filepath.Join(root, "reald"), filepath.Join(root, "linkdir")); err != nil {
		t.Fatalf("os.Symlink(reald, linkdir) failed: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "real.go"), filepath.Join(root, "linkfile.go")); err != nil {
		t.Fatalf("os.Symlink(real.go, linkfile.go) failed: %v", err)
	}
	if err := os.Symlink(root, filepath.Join(root, "cycle")); err != nil {
		t.Fatalf("os.Symlink(root, cycle) failed: %v", err)
	}
	r := openRepo(t, root)

	got, err := r.TOC(".", TOCOptions{Depth: DepthAll})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}

	for _, name := range []string{"linkdir", "linkfile.go", "cycle"} {
		entry := entryByName(t, got.Files, name)
		if entry.Header != "" || entry.Symbols != nil || entry.Error != "" || entry.Lossy {
			t.Errorf("%s = %+v; want a name-only entry, never opened or descended into", name, entry)
		}
	}

	for _, dir := range got.Dirs {
		if dir.Dir == "linkdir" || dir.Dir == "cycle" {
			t.Errorf("Dirs contains %q; a symlink must never appear as a directory entry", dir.Dir)
		}
	}
	if len(got.Dirs) != 1 || got.Dirs[0].Dir != "reald" {
		t.Fatalf("Dirs = %+v; want exactly the real \"reald\" directory", got.Dirs)
	}
	if len(got.Dirs[0].Files) != 1 || got.Dirs[0].Files[0].Name != "inner.go" {
		t.Errorf("reald.Files = %+v; want exactly inner.go, read once through its real path", got.Dirs[0].Files)
	}
}

// TestWalk_DescendantGitignoreTwoLevelsDownHonoredUnderDepthAll asserts a .gitignore two
// directory levels below the query root is honoured when DepthAll recurses that far.
func TestWalk_DescendantGitignoreTwoLevelsDownHonoredUnderDepthAll(t *testing.T) {
	root := writeScratchTree(t, "walk-descendant-gitignore", map[string]string{
		"a/b/.gitignore": "secret.go\n",
		"a/b/secret.go":  "package p\n",
		"a/b/visible.go": "package p\n",
	})
	r := openRepo(t, root)

	got, err := r.TOC(".", TOCOptions{Depth: DepthAll})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(got.Dirs) != 1 || got.Dirs[0].Dir != "a" {
		t.Fatalf("Dirs = %+v; want exactly \"a\"", got.Dirs)
	}
	aDir := got.Dirs[0]
	if len(aDir.Dirs) != 1 || aDir.Dirs[0].Dir != "a/b" {
		t.Fatalf("a.Dirs = %+v; want exactly \"a/b\"", aDir.Dirs)
	}
	bDir := aDir.Dirs[0]
	// .gitignore itself is a real, non-excluded file and is listed like any other; secret.go is
	// the one entry its own pattern excludes.
	for _, f := range bDir.Files {
		if f.Name == "secret.go" {
			t.Errorf("a/b.Files = %+v; secret.go must be excluded by its own directory's .gitignore", bDir.Files)
		}
	}
	if len(bDir.Files) != 2 {
		t.Errorf("a/b.Files = %+v; want exactly .gitignore and visible.go", bDir.Files)
	}
}
