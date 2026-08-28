// seam_enforcement_test.go enforces the engine/CLI seam across the whole engine tree: production
// code under internal/quarryengine/ (recursively, across every subpackage — the root leaf, lsp,
// registry, daemon, daemon/daemontest, query, treesitter, toc, and impact) and under quarry/ (the
// thin facade) never imports internal/output, cobra, or any internal/*cli package — internal/cli is
// the sole place engine results become JSON.
// It is a BANNED LIST, not an allowlist — the engine draws on the shared-infrastructure layer as
// freely as any other engine package, and this check covers direct imports only, never the
// transitive closure.
// It widens what was originally a single-directory guard on the flat quarry/ package into a
// two-tree walk, since the engine-repackage move split that one package into the eight-package DAG
// under internal/quarryengine plus the quarry/ facade.

package quarryengine

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEngineSeamInvariant_BannedImports verifies that no non-test file anywhere under
// internal/quarryengine/ or quarry/ imports internal/output, cobra, or any internal/*cli
// package.
func TestEngineSeamInvariant_BannedImports(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine quarry source directory location")
	}
	// This file sits at internal/quarryengine/seam_enforcement_test.go, so its own directory is
	// the root of the first tree to walk; the module root (two levels up) is where the second
	// tree, quarry/, lives.
	quarryengineRoot := filepath.Dir(file)
	moduleRoot := filepath.Dir(filepath.Dir(quarryengineRoot))
	quarryRoot := filepath.Join(moduleRoot, "quarry")

	var failures []string
	parsedCount := 0
	visitedDirs := make(map[string]bool)

	walkTree := func(root string) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				visitedDirs[path] = true
				return nil
			}
			name := d.Name()
			if strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
				return nil
			}

			fset := token.NewFileSet()
			astFile, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Logf("warning: failed to parse %s: %v", path, err)
				return nil
			}
			parsedCount++

			for _, imp := range astFile.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)

				if importPath == "github.com/Knatte18/quarry/internal/output" {
					failures = append(failures, name+": banned internal/output import (engine must stay io.Writer/exit-code-free)")
					continue
				}
				if strings.Contains(importPath, "spf13/cobra") {
					failures = append(failures, name+": banned cobra import (engine must stay cobra-free)")
					continue
				}
				if strings.Contains(importPath, "/internal/") && strings.HasSuffix(importPath, "cli") {
					failures = append(failures, name+": banned internal/*cli import (cli imports engine, never the reverse)")
					continue
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	walkTree(quarryengineRoot)
	if _, err := os.Stat(quarryRoot); err != nil {
		t.Fatalf("quarry facade directory not found at %s: %v", quarryRoot, err)
	}
	walkTree(quarryRoot)

	if parsedCount == 0 {
		t.Fatal("scanned zero non-test .go files across the engine tree; the seam check cannot go green by finding nothing to check")
	}

	// A package added later and silently skipped by this walk must not let the guard go green
	// by finding nothing to check against it: internal/quarryengine itself, lsp, registry,
	// daemon, daemon/daemontest, query, treesitter, toc, and impact make nine engine directories,
	// and quarry/ makes ten across both trees — this walk covers both trees, unlike
	// layering_test.go's engine-only walk, so its floor is deliberately kept below the real count
	// rather than raised to match it exactly.
	const minPackageDirs = 8
	if len(visitedDirs) < minPackageDirs {
		t.Fatalf("walk visited %d distinct directories; want at least %d, proving the walk actually covers the whole engine tree rather than silently skipping a package", len(visitedDirs), minPackageDirs)
	}

	if len(failures) > 0 {
		t.Errorf("the engine/CLI seam is violated; banned imports found: %v", failures)
	}
}
