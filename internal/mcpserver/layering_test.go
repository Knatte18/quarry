// layering_test.go enforces the facade-only import constraint internal/quarryengine/layering_test.go
// leaves purely conventional for this package: that file's layeringTable polices only rows under
// internal/quarryengine/..., with no row for internal/mcpserver, so without a test here the
// facade-seam-usage Shared Decision — every engine identifier this package needs reaches it through
// github.com/Knatte18/quarry/quarry, never a direct internal/quarryengine/... import — would be
// convention-only while the analogous engine rule is mechanical.

package mcpserver

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// bannedImportPrefix is the one import path prefix no file in this package may import: the engine
// tree itself. github.com/Knatte18/quarry/quarry (the facade) and
// github.com/Knatte18/quarry/internal/cli (the resolution helpers) are both allowed and are not
// checked against this prefix.
const bannedImportPrefix = "github.com/Knatte18/quarry/internal/quarryengine"

// TestLayeringInvariant_FacadeOnly walks every .go file — production and _test.go alike — in this
// package's own directory, parses its import block with parser.ImportsOnly, and fails on any
// import path carrying the bannedImportPrefix.
func TestLayeringInvariant_FacadeOnly(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine internal/mcpserver source directory location")
	}
	pkgDir := filepath.Dir(file)

	var failures []string
	parsedCount := 0

	err := filepath.WalkDir(pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != pkgDir {
				// This package has no subdirectories today; skip defensively rather than
				// descend into one that might appear later with its own import rules.
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		parsedCount++

		for _, imp := range astFile.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == bannedImportPrefix || strings.HasPrefix(importPath, bannedImportPrefix+"/") {
				failures = append(failures, d.Name()+": disallowed import "+importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", pkgDir, err)
	}

	if parsedCount == 0 {
		t.Fatal("scanned zero .go files under internal/mcpserver; the layering check cannot go green by finding nothing to check")
	}

	if len(failures) > 0 {
		t.Errorf("the internal/mcpserver facade-only layering invariant is violated: %v", failures)
	}
}
