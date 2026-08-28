// enclosing_test.go covers enclosingSymbol's pure selection rule over hand-built toc.Symbol
// slices, and the fileCache's parse-once-per-path guarantee against the repo-root impactfixture
// tree.

package impact

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine/toc"
)

// TestEnclosingSymbol covers enclosingSymbol's selection rule over hand-built toc.Symbol slices,
// with no filesystem and no LSP.
func TestEnclosingSymbol(t *testing.T) {
	// docFunc carries a two-line-gap docstring (lines 1-2) ahead of its declaration line (3), a
	// signature ending on line 3, a body running through line 6, and a last line of 7 — enough
	// range to exercise every boundary in one symbol.
	docFunc := toc.Symbol{Name: "docFunc", Start: 1, SigEnd: 3, End: 7}

	tests := []struct {
		name     string
		symbols  []toc.Symbol
		line     int
		wantName string
		wantOK   bool
	}{
		{
			name:     "LineInsideFunctionBody",
			symbols:  []toc.Symbol{docFunc},
			line:     5,
			wantName: "docFunc",
			wantOK:   true,
		},
		{
			name:     "LineOnDeclarationFirstLine",
			symbols:  []toc.Symbol{docFunc},
			line:     3,
			wantName: "docFunc",
			wantOK:   true,
		},
		{
			name:     "LineOnDocstringFirstLine",
			symbols:  []toc.Symbol{docFunc},
			line:     1,
			wantName: "docFunc",
			wantOK:   true,
		},
		{
			name:     "LineOnLastLine",
			symbols:  []toc.Symbol{docFunc},
			line:     7,
			wantName: "docFunc",
			wantOK:   true,
		},
		{
			name: "LineInGapBetweenDeclarations",
			symbols: []toc.Symbol{
				{Name: "First", Start: 1, End: 5},
				{Name: "Second", Start: 10, End: 15},
			},
			line:   7,
			wantOK: false,
		},
		{
			name: "LineBeforeFirstDeclaration",
			symbols: []toc.Symbol{
				{Name: "First", Start: 5, End: 10},
			},
			line:   2,
			wantOK: false,
		},
		{
			name: "OverlappingRangesGreatestStartWins",
			symbols: []toc.Symbol{
				{Name: "Class", Start: 1, End: 20},
				{Name: "Method", Start: 5, End: 10},
			},
			line:     7,
			wantName: "Method",
			wantOK:   true,
		},
		{
			name:    "EmptySymbolSlice",
			symbols: nil,
			line:    5,
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := enclosingSymbol(tt.symbols, tt.line)
			if ok != tt.wantOK {
				t.Fatalf("enclosingSymbol(%v, %d) ok = %v; want %v", tt.symbols, tt.line, ok, tt.wantOK)
			}
			if ok && got.Name != tt.wantName {
				t.Errorf("enclosingSymbol(%v, %d) = %q; want %q", tt.symbols, tt.line, got.Name, tt.wantName)
			}
		})
	}
}

// repoRoot returns this worktree's module root, scanning up from this file's own location.
// internal/quarryengine/impact/enclosing_test.go sits three directories below the repo root
// (internal, quarryengine, impact), so reaching it takes four filepath.Dir calls: one to strip the
// filename itself, then one per intervening directory. This borrows the technique
// internal/cli/assertnocallers_lsp_test.go's repoRoot helper uses — runtime.Caller(0) walked up
// with filepath.Dir — not its literal three-call body, which is sized for that file's own,
// shallower location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("repoRoot: could not determine quarry source directory location")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

// TestFileCache_ImpactFixtureResolution covers fileCache.resolve and enclosingSymbol together
// against the repo-root impactfixture tree, proving the degradation outcomes the three-outcome
// rule describes.
func TestFileCache_ImpactFixtureResolution(t *testing.T) {
	fixtureRoot := filepath.Join(repoRoot(t), "testdata", "impactfixture")

	t.Run("GoMethodBodyResolvesToDocstringStartSymbol", func(t *testing.T) {
		cache := newFileCache(nil)
		path := filepath.Join(fixtureRoot, "billing", "invoice.go")
		fileTOC, err := cache.resolve(path)
		if err != nil {
			t.Fatalf("resolve(%q) returned error: %v", path, err)
		}
		// Line 21 is inside ApplyDiscount's body.
		sym, ok := enclosingSymbol(fileTOC.Symbols, 21)
		if !ok {
			t.Fatalf("enclosingSymbol(_, 21) ok = false; want true")
		}
		if sym.Name != "ApplyDiscount" {
			t.Errorf("enclosingSymbol(_, 21).Name = %q; want %q", sym.Name, "ApplyDiscount")
		}
		// Line 20 is the "func (inv *Invoice) ApplyDiscount(...)" line; Start must be strictly
		// less than it, since ApplyDiscount carries a doc comment.
		const funcLine = 20
		if sym.Start >= funcLine {
			t.Errorf("ApplyDiscount.Start = %d; want strictly less than func line %d", sym.Start, funcLine)
		}
	})

	t.Run("GoPackageLevelVarHasNoEnclosingSymbol", func(t *testing.T) {
		cache := newFileCache(nil)
		path := filepath.Join(fixtureRoot, "billing", "invoice.go")
		fileTOC, err := cache.resolve(path)
		if err != nil {
			t.Fatalf("resolve(%q) returned error: %v", path, err)
		}
		// Line 15 is "var DefaultRate = 0.05", which toc.Kind has no vocabulary for.
		if _, ok := enclosingSymbol(fileTOC.Symbols, 15); ok {
			t.Error("enclosingSymbol(_, 15) ok = true; want false for a package-level var line")
		}
	})

	t.Run("NonexistentPathYieldsResolverError", func(t *testing.T) {
		cache := newFileCache(nil)
		path := filepath.Join(fixtureRoot, "billing", "does-not-exist.go")
		if _, err := cache.resolve(path); err == nil {
			t.Error("resolve(nonexistent path) returned nil error; want a resolver error")
		}
	})

	t.Run("TypeScriptFixtureYieldsUnsupportedLanguageError", func(t *testing.T) {
		cache := newFileCache(nil)
		path := filepath.Join(fixtureRoot, "tsfixture", "client.ts")
		_, err := cache.resolve(path)
		if err == nil {
			t.Fatal("resolve(tsfixture) returned nil error; want an unsupported-language resolver error")
		}
		if !strings.Contains(err.Error(), "typescript") {
			t.Errorf("resolve(tsfixture) error = %q; want it to name the unsupported language %q", err.Error(), "typescript")
		}
	})

	t.Run("PythonMethodLineResolvesToMethodNotClass", func(t *testing.T) {
		cache := newFileCache(nil)
		path := filepath.Join(fixtureRoot, "pyfixture", "shapes.py")
		fileTOC, err := cache.resolve(path)
		if err != nil {
			t.Fatalf("resolve(%q) returned error: %v", path, err)
		}
		// Line 11 is inside area's docstring, itself inside Shape's own range — the
		// class-and-method overlap the greatest-Start tie-break exists for.
		sym, ok := enclosingSymbol(fileTOC.Symbols, 11)
		if !ok {
			t.Fatalf("enclosingSymbol(_, 11) ok = false; want true")
		}
		if sym.Name != "area" {
			t.Errorf("enclosingSymbol(_, 11).Name = %q; want %q (the nested method, not the enclosing class)", sym.Name, "area")
		}
	})
}

// TestFileCache_ParsesEachDistinctPathExactlyOnce injects a counting parseFunc and asserts the
// cache's one-parse-per-distinct-path guarantee, never comparing results or timing.
func TestFileCache_ParsesEachDistinctPathExactlyOnce(t *testing.T) {
	counts := make(map[string]int)
	cache := newFileCache(func(path string) (toc.FileTOC, error) {
		counts[path]++
		return toc.FileTOC{Language: "go"}, nil
	})

	if _, err := cache.resolve("/a.go"); err != nil {
		t.Fatalf("resolve(/a.go) returned error: %v", err)
	}
	if _, err := cache.resolve("/a.go"); err != nil {
		t.Fatalf("resolve(/a.go) returned error: %v", err)
	}
	if _, err := cache.resolve("/a.go"); err != nil {
		t.Fatalf("resolve(/a.go) returned error: %v", err)
	}
	if counts["/a.go"] != 1 {
		t.Errorf("parse count for /a.go = %d; want 1 after three resolve calls for the same path", counts["/a.go"])
	}

	if _, err := cache.resolve("/b.go"); err != nil {
		t.Fatalf("resolve(/b.go) returned error: %v", err)
	}
	if counts["/a.go"] != 1 || counts["/b.go"] != 1 {
		t.Errorf("parse counts = {/a.go: %d, /b.go: %d}; want {1, 1} after resolving two distinct paths", counts["/a.go"], counts["/b.go"])
	}
}
