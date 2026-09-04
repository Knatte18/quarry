// toc_integration_test.go runs Repo.TOC against a real file in this repository —
// internal/engine/treesitter/treesitter.go — rather than a synthetic fixture, so the
// extraction pipeline is proven against source nobody wrote to satisfy this package's own tests. It
// is hermetic (reads files under the module root, writes nothing, spawns nothing) and belongs in
// the default test tier alongside the rest of the package.

package engine

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestRepoTOC_RepositoryFile_TreesitterPackage opens the module root and runs TOC against
// internal/engine/treesitter/treesitter.go, chosen because it is small, stable, and carries a
// file header plus three well-documented functions. It asserts the symbol names, kinds, and range
// ordering loosely enough to survive an ordinary edit to that file — a test that has to be updated
// every time an unrelated comment is reflowed is a test that gets deleted.
func TestRepoTOC_RepositoryFile_TreesitterPackage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine this test file's own source location")
	}
	// This file sits at internal/engine/toc_integration_test.go, so the module root is three
	// levels up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	targetRel := filepath.ToSlash(filepath.Join("internal", "engine", "treesitter", "treesitter.go"))

	r, err := Open(moduleRoot)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", moduleRoot, err)
	}
	got, err := r.TOC(targetRel, TOCOptions{})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", targetRel, err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Files) = %d; want 1", len(got.Files))
	}
	entry := got.Files[0]

	if got.Language != "go" {
		t.Errorf("Language = %q; want %q", got.Language, "go")
	}
	if got.Package != "treesitter" {
		t.Errorf("Package = %q; want %q", got.Package, "treesitter")
	}
	if entry.Lossy {
		t.Error("Lossy = true; want false for a real, well-formed repository file")
	}
	if entry.Header == "" {
		t.Error("Header is empty; want the file's first header paragraph")
	}
	if entry.Symbols == nil {
		t.Fatal("Symbols == nil; want a non-nil pointer for a file target's default")
	}
	symbols := *entry.Symbols

	// The treesitter package's own unit is its repository-relative directory, so each expected
	// function's id is that unit followed by "#" and the bare function name.
	wantIDs := map[string]bool{
		"internal/engine/treesitter#Supported": true,
		"internal/engine/treesitter#Languages": true,
		"internal/engine/treesitter#WithTree":  true,
	}
	found := make(map[string]bool, len(wantIDs))
	for _, sym := range symbols {
		if !wantIDs[sym.ID] {
			continue
		}
		found[sym.ID] = true
		if sym.Kind != KindFunction {
			t.Errorf("symbol %q Kind = %q; want %q", sym.ID, sym.Kind, KindFunction)
		}
		if sym.Signature == "" {
			t.Errorf("symbol %q Signature is empty; want non-empty", sym.ID)
		}
		if sym.Doc == "" {
			t.Errorf("symbol %q Doc is empty; want non-empty", sym.ID)
		}
		if !(sym.Start <= sym.SigEnd && sym.SigEnd <= sym.End) {
			t.Errorf("symbol %q: Start=%d SigEnd=%d End=%d; want Start <= SigEnd <= End", sym.ID, sym.Start, sym.SigEnd, sym.End)
		}
	}
	for wantID := range wantIDs {
		if !found[wantID] {
			t.Errorf("expected exported function id %q not found in Symbols", wantID)
		}
	}

	// Start values are ascending across every symbol in the file, not just the three named ones.
	for i := 1; i < len(symbols); i++ {
		if symbols[i-1].Start >= symbols[i].Start {
			t.Errorf("Symbols[%d].Start = %d >= Symbols[%d].Start = %d; want ascending order",
				i-1, symbols[i-1].Start, i, symbols[i].Start)
		}
	}
}
