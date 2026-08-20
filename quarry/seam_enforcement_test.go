// seam_enforcement_test.go enforces quarry's own engine/CLI seam: production code in the public
// quarry package never imports the CLI package, cobra, or the output-envelope package —
// internal/cli is the sole place engine results become JSON.
// It is a BANNED LIST, not an allowlist — quarry draws on the shared-infrastructure layer as freely
// as any other engine package, and this check covers direct imports only, never the transitive
// closure.

package quarry

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEngineSeamInvariant_BannedImports verifies that no non-test file in the quarry package
// imports internal/output, cobra, or any internal/*cli package.
func TestEngineSeamInvariant_BannedImports(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine quarry source directory location")
	}
	pkgDir := filepath.Dir(file)

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("read quarry directory: %v", err)
	}

	var failures []string
	parsedCount := 0

	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), "_test.go") || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(pkgDir, entry.Name())

		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Logf("warning: failed to parse %s: %v", path, err)
			continue
		}
		parsedCount++

		for _, imp := range astFile.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			if importPath == "github.com/Knatte18/quarry/internal/output" {
				failures = append(failures, entry.Name()+": banned internal/output import (engine must stay io.Writer/exit-code-free)")
				continue
			}
			if strings.Contains(importPath, "spf13/cobra") {
				failures = append(failures, entry.Name()+": banned cobra import (engine must stay cobra-free)")
				continue
			}
			if strings.Contains(importPath, "/internal/") && strings.HasSuffix(importPath, "cli") {
				failures = append(failures, entry.Name()+": banned internal/*cli import (cli imports engine, never the reverse)")
				continue
			}
		}
	}

	if parsedCount == 0 {
		t.Fatal("scanned zero non-test .go files in the quarry package; the seam check cannot go green by finding nothing to check")
	}

	if len(failures) > 0 {
		t.Errorf("quarry's engine/CLI seam violated; banned imports found: %v", failures)
	}
}
