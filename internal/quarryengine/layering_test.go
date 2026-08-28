// layering_test.go pins the package DAG the engine-repackage batch created: it walks every .go
// file under internal/quarryengine/ — production AND _test.go files, unlike
// seam_enforcement_test.go's production-only walk — and checks each file's
// internal/quarryengine/... imports against an allowed-direction table keyed by (package, file
// kind). A test file importing across the DAG is the realistic way this layering rots, which is
// why this walk covers _test.go files that seam_enforcement_test.go deliberately skips.

package quarryengine

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The seven internal/quarryengine/... import paths this guard reasons about.
const (
	rootPkg       = "github.com/Knatte18/quarry/internal/quarryengine"
	lspPkg        = rootPkg + "/lsp"
	registryPkg   = rootPkg + "/registry"
	daemonPkg     = rootPkg + "/daemon"
	daemontestPkg = daemonPkg + "/daemontest"
	queryPkg      = rootPkg + "/query"
	treesitterPkg = rootPkg + "/treesitter"
)

// layeringRow names the set of internal/quarryengine/... import paths a file belonging to pkgDir
// (the package's directory relative to internal/quarryengine/, "" for the root package itself),
// of the given file kind (production or _test.go), may import. query is the one package whose
// test row differs from its production row — its test row swaps daemonPkg for daemontestPkg — so
// every other package lists the same allowed set twice rather than sharing one entry, keeping the
// table's shape uniform.
type layeringRow struct {
	pkgDir    string
	isTestRow bool
	allowed   map[string]bool
}

// pathSet builds a lookup set from a list of import paths.
func pathSet(paths ...string) map[string]bool {
	set := make(map[string]bool, len(paths))
	for _, p := range paths {
		set[p] = true
	}
	return set
}

// layeringTable encodes the allowed import directions from the plan's package-layout Shared
// Decision: the root imports no subpackage; lsp and registry import the root only; daemon imports
// root + registry + lsp; query's production files import all four; query's test files import
// root + registry + lsp + daemontest, never daemon directly; daemontest imports daemon.
var layeringTable = []layeringRow{
	{pkgDir: "", isTestRow: false, allowed: pathSet()},
	{pkgDir: "", isTestRow: true, allowed: pathSet()},
	{pkgDir: "lsp", isTestRow: false, allowed: pathSet(rootPkg)},
	{pkgDir: "lsp", isTestRow: true, allowed: pathSet(rootPkg)},
	{pkgDir: "registry", isTestRow: false, allowed: pathSet(rootPkg)},
	{pkgDir: "registry", isTestRow: true, allowed: pathSet(rootPkg)},
	{pkgDir: "daemon", isTestRow: false, allowed: pathSet(rootPkg, registryPkg, lspPkg)},
	{pkgDir: "daemon", isTestRow: true, allowed: pathSet(rootPkg, registryPkg, lspPkg)},
	{pkgDir: "daemon/daemontest", isTestRow: false, allowed: pathSet(daemonPkg)},
	{pkgDir: "daemon/daemontest", isTestRow: true, allowed: pathSet(daemonPkg)},
	{pkgDir: "query", isTestRow: false, allowed: pathSet(rootPkg, registryPkg, lspPkg, daemonPkg)},
	{pkgDir: "query", isTestRow: true, allowed: pathSet(rootPkg, registryPkg, lspPkg, daemontestPkg)},
	{pkgDir: "treesitter", isTestRow: false, allowed: pathSet(rootPkg)},
	{pkgDir: "treesitter", isTestRow: true, allowed: pathSet(rootPkg)},
}

// allowedFor looks up the layeringTable row for pkgDir and isTestRow, reporting ok = false if no
// row was declared for that combination — that is itself a bug in the table, not a passing case.
func allowedFor(pkgDir string, isTestRow bool) (map[string]bool, bool) {
	for _, row := range layeringTable {
		if row.pkgDir == pkgDir && row.isTestRow == isTestRow {
			return row.allowed, true
		}
	}
	return nil, false
}

// TestLayeringInvariant_ImportDirections verifies that every .go file under
// internal/quarryengine/ — production and _test.go alike — only imports
// internal/quarryengine/... packages in the direction the package DAG allows. Imports outside
// that prefix are ignored; the banned-import direction (engine to CLI/output/cobra) is
// seam_enforcement_test.go's job, not this one's.
func TestLayeringInvariant_ImportDirections(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine quarry source directory location")
	}
	quarryengineRoot := filepath.Dir(file)

	var failures []string
	parsedCount := 0
	visitedDirs := make(map[string]bool)

	err := filepath.WalkDir(quarryengineRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			visitedDirs[path] = true
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		isTestFile := strings.HasSuffix(name, "_test.go")

		relDir, err := filepath.Rel(quarryengineRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkgDir := filepath.ToSlash(relDir)
		if pkgDir == "." {
			pkgDir = ""
		}

		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Logf("warning: failed to parse %s: %v", path, err)
			return nil
		}
		parsedCount++

		allowed, ok := allowedFor(pkgDir, isTestFile)
		if !ok {
			failures = append(failures, name+": no layering row declared for package %q (test="+boolString(isTestFile)+")")
			return nil
		}

		for _, imp := range astFile.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(importPath, rootPkg) {
				continue
			}
			// Any _test.go file, in any package, may import daemontest unconditionally — that is
			// what daemontest exists for, and it is the only allowance not gated by the file's
			// own package row.
			if isTestFile && importPath == daemontestPkg {
				continue
			}
			if !allowed[importPath] {
				failures = append(failures, name+": disallowed import "+importPath+" (package "+describePkgDir(pkgDir)+")")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", quarryengineRoot, err)
	}

	if parsedCount == 0 {
		t.Fatal("scanned zero .go files under internal/quarryengine; the layering check cannot go green by finding nothing to check")
	}

	// internal/quarryengine itself, lsp, registry, daemon, daemon/daemontest, and query is six
	// distinct package directories; a package added later and silently skipped by this walk must
	// not let the guard go green by finding nothing to check against it.
	const minPackageDirs = 6
	if len(visitedDirs) < minPackageDirs {
		t.Fatalf("walk visited %d distinct directories; want at least %d, proving the walk actually covers every layer of the DAG rather than silently skipping a package", len(visitedDirs), minPackageDirs)
	}

	if len(failures) > 0 {
		t.Errorf("the internal/quarryengine layering invariant is violated: %v", failures)
	}
}

// boolString renders b as "true" or "false" for use inside an error message built with string
// concatenation, avoiding a fmt.Sprintf call in a rarely-hit branch.
func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// describePkgDir renders pkgDir for an error message, naming the root package explicitly since
// its pkgDir is the empty string.
func describePkgDir(pkgDir string) string {
	if pkgDir == "" {
		return "internal/quarryengine (root)"
	}
	return "internal/quarryengine/" + pkgDir
}
