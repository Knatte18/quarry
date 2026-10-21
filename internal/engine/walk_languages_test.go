// walk_languages_test.go covers the walk's per-language unit rules: which language supplies a directory's package, how a Go vote ignores other languages' files, and which files list symbols.

package engine

import (
	"reflect"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// entrySymbolIDs returns the ids of entry's symbols, nil when the entry carries no symbol list.
func entrySymbolIDs(entry FileEntry) []string {
	if entry.Symbols == nil {
		return nil
	}
	ids := make([]string, len(*entry.Symbols))
	for i, sym := range *entry.Symbols {
		ids[i] = sym.ID
	}
	return ids
}

// tocWithSymbols answers the TOC of target with symbols requested, failing the test on error.
func tocWithSymbols(t *testing.T, r *Repo, target string) DirAnswer {
	t.Helper()
	got, err := r.TOC(target, TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC(%q) returned error: %v", target, err)
	}
	return got
}

const (
	walkGoDoc      = "// Package p is the mixed directory's Go package.\npackage p\n\nfunc A() {}\n"
	walkGoPlain    = "package p\n\nfunc B() {}\n"
	walkGoExternal = "package p_test\n\nfunc C() {}\n"
)

// walkCSharpSource returns a C# file declaring one class in the given namespace.
func walkCSharpSource(namespace, class string) string {
	return "namespace " + namespace + "\n{\n    public class " + class + " { }\n}\n"
}

func TestWalk_PerLanguageUnits(t *testing.T) {
	t.Parallel()

	t.Run("GoIdsUnchangedAndCSharpFilesCarryTheirNamespace", func(t *testing.T) {
		t.Parallel()
		goOnly := openScratchRepo(t, "walk-units-go-only", map[string]string{
			"pkg/doc.go":    walkGoDoc,
			"pkg/b.go":      walkGoPlain,
			"pkg/c_test.go": walkGoExternal,
		})
		mixed := openScratchRepo(t, "walk-units-mixed", map[string]string{
			"pkg/doc.go":    walkGoDoc,
			"pkg/b.go":      walkGoPlain,
			"pkg/c_test.go": walkGoExternal,
			"pkg/X.cs":      walkCSharpSource("N", "X"),
			"pkg/Y.cs":      walkCSharpSource("N.Sub", "Y"),
		})

		want := tocWithSymbols(t, goOnly, "pkg")
		got := tocWithSymbols(t, mixed, "pkg")

		for _, name := range []string{"doc.go", "b.go", "c_test.go"} {
			gotIDs := entrySymbolIDs(entryByName(t, got.Files, name))
			wantIDs := entrySymbolIDs(entryByName(t, want.Files, name))
			if !reflect.DeepEqual(gotIDs, wantIDs) {
				t.Errorf("%s symbol ids = %v; want %v, the ids the directory gives without its C# files", name, gotIDs, wantIDs)
			}
		}
		if got.Package != "p" || got.Language != "go" {
			t.Errorf("answer package/language = %q/%q; want p/go from the Go majority", got.Package, got.Language)
		}
		if got.Doc != "Package p is the mixed directory's Go package." {
			t.Errorf("answer Doc = %q; want the Go package documentation", got.Doc)
		}

		x := entryByName(t, got.Files, "X.cs")
		if ids := entrySymbolIDs(x); !reflect.DeepEqual(ids, []string{"N#X"}) {
			t.Errorf("X.cs symbol ids = %v; want [N#X]", ids)
		}
		if x.Package != "N" || x.Language != "csharp" {
			t.Errorf("X.cs package/language = %q/%q; want the deviation keys N/csharp", x.Package, x.Language)
		}
		y := entryByName(t, got.Files, "Y.cs")
		if ids := entrySymbolIDs(y); !reflect.DeepEqual(ids, []string{"N.Sub#Y"}) {
			t.Errorf("Y.cs symbol ids = %v; want [N.Sub#Y]", ids)
		}
		for _, sym := range *y.Symbols {
			if sym.File != "" {
				t.Errorf("Y.cs symbol %s File = %q; want empty inside a toc answer", sym.ID, sym.File)
			}
		}
	})

	t.Run("CSharpMajorityTakesTheMostCommonFirstNamespace", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-csharp-majority", map[string]string{
			"lib/doc.go": walkGoDoc,
			"lib/One.cs": walkCSharpSource("A", "One"),
			"lib/Two.cs": walkCSharpSource("A", "Two"),
			"lib/Cee.cs": walkCSharpSource("B", "Cee"),
		})
		got := tocWithSymbols(t, r, "lib")
		if got.Package != "A" || got.Language != "csharp" {
			t.Errorf("answer package/language = %q/%q; want A/csharp", got.Package, got.Language)
		}
		if got.Doc != "Package p is the mixed directory's Go package." {
			t.Errorf("answer Doc = %q; want the Go package documentation although C# supplies the package", got.Doc)
		}
		one := entryByName(t, got.Files, "One.cs")
		if one.Package != "" || one.Language != "" {
			t.Errorf("One.cs package/language = %q/%q; want no deviation keys", one.Package, one.Language)
		}
		cee := entryByName(t, got.Files, "Cee.cs")
		if cee.Package != "B" || cee.Language != "" {
			t.Errorf("Cee.cs package/language = %q/%q; want package B and no language key", cee.Package, cee.Language)
		}
		goFile := entryByName(t, got.Files, "doc.go")
		if goFile.Package != "p" || goFile.Language != "go" {
			t.Errorf("doc.go package/language = %q/%q; want the deviation keys p/go", goFile.Package, goFile.Language)
		}
	})

	t.Run("CSharpNamespaceTieTakesTheSmallerName", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-namespace-tie", map[string]string{
			"lib/Zed.cs":   walkCSharpSource("Z", "Zed"),
			"lib/Alpha.cs": walkCSharpSource("A", "Alpha"),
		})
		got := tocWithSymbols(t, r, "lib")
		if got.Package != "A" || got.Language != "csharp" {
			t.Errorf("answer package/language = %q/%q; want A/csharp", got.Package, got.Language)
		}
	})

	t.Run("LanguageTieTakesTheSmallerLanguageName", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-language-tie", map[string]string{
			"lib/g.go": walkGoPlain,
			"lib/N.cs": walkCSharpSource("N", "N"),
		})
		got := tocWithSymbols(t, r, "lib")
		if got.Package != "N" || got.Language != "csharp" {
			t.Errorf("answer package/language = %q/%q; want N/csharp, since csharp sorts before go", got.Package, got.Language)
		}
		goFile := entryByName(t, got.Files, "g.go")
		if goFile.Package != "p" || goFile.Language != "go" {
			t.Errorf("g.go package/language = %q/%q; want the deviation keys p/go", goFile.Package, goFile.Language)
		}
	})

	t.Run("FileTargetAnswersTheDirectoryFacts", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-file-target", map[string]string{
			"pkg/doc.go":    walkGoDoc,
			"pkg/b.go":      walkGoPlain,
			"pkg/c_test.go": walkGoExternal,
			"pkg/X.cs":      walkCSharpSource("N", "X"),
			"pkg/Y.cs":      walkCSharpSource("N", "Y"),
		})
		dir := tocWithSymbols(t, r, "pkg")
		file := tocWithSymbols(t, r, "pkg/X.cs")
		if file.Package != dir.Package || file.Language != dir.Language || file.Doc != dir.Doc {
			t.Errorf("file target facts = %q/%q/%q; want the directory's %q/%q/%q",
				file.Package, file.Language, file.Doc, dir.Package, dir.Language, dir.Doc)
		}
		if len(file.Files) != 1 || !reflect.DeepEqual(file.Files[0], entryByName(t, dir.Files, "X.cs")) {
			t.Errorf("file target Files = %+v; want only the X.cs entry the directory lists", file.Files)
		}
	})

	t.Run("CSharpListsSymbolsWhereGoCannot", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-unspellable-directories", map[string]string{
			"Root.cs":         walkCSharpSource("N", "Root"),
			"with space/W.cs": walkCSharpSource("N", "W"),
			"with space/g.go": walkGoPlain,
			"root.go":         walkGoPlain,
		})
		root := tocWithSymbols(t, r, ".")
		if ids := entrySymbolIDs(entryByName(t, root.Files, "Root.cs")); !reflect.DeepEqual(ids, []string{"N#Root"}) {
			t.Errorf("Root.cs symbol ids = %v; want [N#Root] at the repository root", ids)
		}
		if entryByName(t, root.Files, "root.go").Symbols != nil {
			t.Errorf("root.go lists symbols; a Go file at the repository root has an unspellable unit")
		}

		spaced := tocWithSymbols(t, r, "with space")
		if ids := entrySymbolIDs(entryByName(t, spaced.Files, "W.cs")); !reflect.DeepEqual(ids, []string{"N#W"}) {
			t.Errorf("W.cs symbol ids = %v; want [N#W] in a directory whose name holds a space", ids)
		}
		if entryByName(t, spaced.Files, "g.go").Symbols != nil {
			t.Errorf("g.go lists symbols; a Go file in a directory whose name holds a space has an unspellable unit")
		}
	})

	t.Run("DroppedDeclarationMakesTheFileLossy", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-dropped-id", map[string]string{
			"pkg/D.cs": "namespace N\n{\n    public class Café { }\n    public class Fine { }\n}\n",
		})
		got := tocWithSymbols(t, r, "pkg/D.cs")
		entry := got.Files[0]
		if !entry.Lossy {
			t.Errorf("D.cs Lossy = false; want true for a file whose strategy dropped an id")
		}
		if ids := entrySymbolIDs(entry); !reflect.DeepEqual(ids, []string{"N#Fine"}) {
			t.Errorf("D.cs symbol ids = %v; want only [N#Fine]", ids)
		}
	})

	t.Run("GoGlyphNamingACSharpTypeResolvesNothing", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "walk-units-go-glyph-vs-csharp", map[string]string{
			"pkg/g.go":   walkGoPlain,
			"pkg/Foo.cs": walkCSharpSource("N", "Foo"),
		})
		got, err := r.SpansOf(glyph.Glyph{Lang: glyph.Go, Unit: "pkg", Name: "Foo"})
		if err != nil {
			t.Fatalf("SpansOf returned error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("SpansOf(pkg#Foo) = %+v; want no spans for a name only a C# type declares", got)
		}
		got, err = r.SpansOf(glyph.Glyph{Lang: glyph.Go, Unit: "pkg", Name: "B"})
		if err != nil || len(got) != 1 {
			t.Errorf("SpansOf(pkg#B) = %+v, %v; want the one Go span", got, err)
		}
	})
}
