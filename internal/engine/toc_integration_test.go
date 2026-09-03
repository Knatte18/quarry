// toc_integration_test.go runs TOCFile against a real file in this repository —
// internal/engine/treesitter/treesitter.go — rather than a synthetic fixture, so the
// extraction pipeline is proven against source nobody wrote to satisfy this package's own tests. It
// is hermetic (reads one file, writes nothing, spawns nothing) and belongs in the default test tier
// alongside the rest of the package.

package engine

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestTOCFile_RepositoryFile_TreesitterPackage runs TOCFile against
// internal/engine/treesitter/treesitter.go, chosen because it is small, stable, and carries a
// file header plus three well-documented functions. It asserts the symbol names, kinds, and range
// ordering loosely enough to survive an ordinary edit to that file — a test that has to be updated
// every time an unrelated comment is reflowed is a test that gets deleted.
func TestTOCFile_RepositoryFile_TreesitterPackage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine this test file's own source location")
	}
	// This file sits at internal/engine/toc_integration_test.go, so the module root is three
	// levels up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	targetPath := filepath.Join(moduleRoot, "internal", "engine", "treesitter", "treesitter.go")

	got, err := TOCFile(targetPath, "")
	if err != nil {
		t.Fatalf("TOCFile(%q, \"\", ...) returned error: %v", targetPath, err)
	}

	if got.Language != "go" {
		t.Errorf("Language = %q; want %q", got.Language, "go")
	}
	if got.Package != "treesitter" {
		t.Errorf("Package = %q; want %q", got.Package, "treesitter")
	}
	if got.Partial {
		t.Error("Partial = true; want false for a real, well-formed repository file")
	}
	if got.Header == "" {
		t.Error("Header is empty; want the file's first header paragraph")
	}

	wantNames := map[string]bool{"Supported": true, "Languages": true, "WithTree": true}
	found := make(map[string]bool, len(wantNames))
	for _, sym := range got.Symbols {
		if !wantNames[sym.Name] {
			continue
		}
		found[sym.Name] = true
		if sym.Kind != KindFunction {
			t.Errorf("symbol %q Kind = %q; want %q", sym.Name, sym.Kind, KindFunction)
		}
		if sym.Signature == "" {
			t.Errorf("symbol %q Signature is empty; want non-empty", sym.Name)
		}
		if sym.Docstring == "" {
			t.Errorf("symbol %q Docstring is empty; want non-empty", sym.Name)
		}
		if !(sym.Start <= sym.SigEnd && sym.SigEnd <= sym.End) {
			t.Errorf("symbol %q: Start=%d SigEnd=%d End=%d; want Start <= SigEnd <= End", sym.Name, sym.Start, sym.SigEnd, sym.End)
		}
	}
	for name := range wantNames {
		if !found[name] {
			t.Errorf("expected exported function %q not found in Symbols", name)
		}
	}

	// Start values are ascending across every symbol in the file, not just the three named ones.
	for i := 1; i < len(got.Symbols); i++ {
		if got.Symbols[i-1].Start >= got.Symbols[i].Start {
			t.Errorf("Symbols[%d].Start = %d >= Symbols[%d].Start = %d; want ascending order",
				i-1, got.Symbols[i-1].Start, i, got.Symbols[i].Start)
		}
	}
}
