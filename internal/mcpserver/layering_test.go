// layering_test.go mechanically enforces two of this task's rules: the facade-only rule (no file in
// internal/mcpserver, cmd/quarry-mcp, or internal/cli imports a forbidden internal package directly)
// and the stdout rule (no production file in internal/mcpserver or cmd/quarry-mcp writes to standard
// output).
//
// The import check's forbidden set is two paths, not one: internal/engine (checked over
// internal/mcpserver and cmd/quarry-mcp, as before) and internal/gitsrc (checked over those same two
// directories, plus internal/cli). internal/cli is scanned against the git plumbing path alone, never
// against the engine path — internal/cli legitimately imports neither today, but it is the package
// the facade-delta batch's own Shared Decision ("internal/cli reaches git error identity through the
// facade, never by importing internal/gitsrc") is actually about, and a convention no test states is
// one refactor from being lost. Adding the engine path to internal/cli's forbidden set would be a new
// rule this task did not decide; the header comment historically recorded that the facade-only rule
// holds there "by convention", and that convention is otherwise left exactly as it was.
//
// This tree carried no import check of any kind before this file first existed: the facade-only rule
// held by convention everywhere it held, including in internal/cli. This remains the only place the
// rule is mechanical rather than one row added to an existing mechanism — there is no engine-side
// layering test in this tree for this file to extend, and the predecessor these checks may recall
// is V1's, on the frozen branch, reference material rather than something this test extends.
//
// Be precise about what the import check is and is not: it scans each file's own import block, so it
// catches a direct import only. It cannot catch a forbidden package reached through an intermediate
// package, and it deliberately does not try — a transitive check would fail immediately and
// permanently, since the quarry facade every one of these packages is required to depend on imports
// both internal/engine and internal/gitsrc itself, which is the whole point of the facade. The
// residual gap, now covering both forbidden paths: a dependency on either introduced through some
// future intermediate internal/ package would pass this check.
//
// The stdout check's own scope is deliberately left untouched by the widening above: it still walks
// only internal/mcpserver and cmd/quarry-mcp, never internal/cli, because internal/cli legitimately
// writes to standard output — rendering the answer there is its entire job — so carrying this check
// onto it would fail immediately for a reason this task never decided. The two checks share one
// directory-locating helper (checkedDirs) but each builds its own list of directories to walk at its
// own call site, so widening one does not widen the other by construction. The stdout check is
// deliberately blunt in its own right: it matches on the identifiers os.Stdout, fmt.Print,
// fmt.Println, fmt.Printf, print, and println, not on reachability. A legitimate future need to name
// os.Stdout — there is none today — is expected to argue with this test rather than slip past it.

package mcpserver

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// forbiddenEngineImport and its path-prefix form are the one import internal/mcpserver and
// cmd/quarry-mcp must never carry directly.
const forbiddenEngineImport = "github.com/Knatte18/quarry/internal/engine"

// forbiddenGitsrcImport and its path-prefix form are the git plumbing package's import path:
// forbidden the same way forbiddenEngineImport is, over internal/mcpserver and cmd/quarry-mcp, and
// additionally over internal/cli, which must reach git error identity only through the facade (see
// this file's own header comment).
const forbiddenGitsrcImport = "github.com/Knatte18/quarry/internal/gitsrc"

// checkedDirs returns the three directories this file's checks may walk: this package's own
// directory, cmd/quarry-mcp's, and internal/cli's, located from runtime.Caller(0) rather than from
// the working directory, so this test does not depend on where it is run from. Each check below
// builds its own subset of these three at its own call site; checkedDirs itself does not decide
// which check sees which directory.
func checkedDirs(t *testing.T) (mcpserverDir, quarryMCPDir, cliDir string) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("checkedDirs: runtime.Caller(0) failed to resolve this file's path")
	}
	mcpserverDir = filepath.Dir(thisFile)
	// mcpserverDir is .../internal/mcpserver; the module root is two directories up.
	moduleRoot := filepath.Dir(filepath.Dir(mcpserverDir))
	quarryMCPDir = filepath.Join(moduleRoot, "cmd", "quarry-mcp")
	cliDir = filepath.Join(moduleRoot, "internal", "cli")
	return mcpserverDir, quarryMCPDir, cliDir
}

// goFilesIn returns the .go files directly inside dir, sorted by ReadDir's own order.
func goFilesIn(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("goFilesIn(%q): %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	return files
}

// importsForbidden reports whether importPath equals any of forbidden or carries any of them as a
// path prefix.
func importsForbidden(importPath string, forbidden []string) (string, bool) {
	for _, f := range forbidden {
		if importPath == f || strings.HasPrefix(importPath, f+"/") {
			return f, true
		}
	}
	return "", false
}

// TestFacadeOnly_NoDirectEngineImport walks every .go file — production and _test.go alike — in
// internal/mcpserver, cmd/quarry-mcp and internal/cli, and fails on a forbidden direct import: the
// engine path over internal/mcpserver and cmd/quarry-mcp, and the git plumbing path over all three,
// internal/cli included (see this file's own header comment for why internal/cli is scanned against
// the git plumbing path alone, never the engine path). The facade-only rule binds the test files
// too, so _test.go files are not excluded from this walk.
func TestFacadeOnly_NoDirectEngineImport(t *testing.T) {
	mcpserverDir, quarryMCPDir, cliDir := checkedDirs(t)

	forbiddenByDir := map[string][]string{
		mcpserverDir: {forbiddenEngineImport, forbiddenGitsrcImport},
		quarryMCPDir: {forbiddenEngineImport, forbiddenGitsrcImport},
		cliDir:       {forbiddenGitsrcImport},
	}

	fset := token.NewFileSet()
	checked := 0
	for _, dir := range []string{mcpserverDir, quarryMCPDir, cliDir} {
		forbidden := forbiddenByDir[dir]
		for _, path := range goFilesIn(t, dir) {
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parser.ParseFile(%q, ImportsOnly): %v", path, err)
			}
			checked++
			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if hit, ok := importsForbidden(importPath, forbidden); ok {
					t.Errorf("%s: imports %q directly; every identifier under %q must reach this package through github.com/Knatte18/quarry/quarry",
						path, importPath, hit)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("TestFacadeOnly_NoDirectEngineImport: parsed zero files; a check that finds nothing to check is worse than no check")
	}
}

// forbiddenStdoutIdentifiers is the stdout check's blunt identifier list: os.Stdout itself, and
// the fmt printing family and builtins that write to it by default.
var forbiddenStdoutIdentifiers = map[string]string{
	"Print":   "fmt",
	"Println": "fmt",
	"Printf":  "fmt",
}

// TestStdout_NoProductionWrite walks every production .go file (not _test.go — a test file is not
// part of the shipped binary's output behaviour) in internal/mcpserver and cmd/quarry-mcp, and
// fails on any reference to os.Stdout or a call to fmt.Print, fmt.Println, fmt.Printf, print, or
// println.
func TestStdout_NoProductionWrite(t *testing.T) {
	mcpserverDir, quarryMCPDir, _ := checkedDirs(t)

	fset := token.NewFileSet()
	checked := 0
	for _, dir := range []string{mcpserverDir, quarryMCPDir} {
		for _, path := range goFilesIn(t, dir) {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parser.ParseFile(%q): %v", path, err)
			}
			checked++
			ast.Inspect(f, func(n ast.Node) bool {
				switch expr := n.(type) {
				case *ast.SelectorExpr:
					pkgIdent, ok := expr.X.(*ast.Ident)
					if !ok {
						return true
					}
					if pkgIdent.Name == "os" && expr.Sel.Name == "Stdout" {
						t.Errorf("%s:%s: references os.Stdout; that stream is reserved for the framed MCP transport",
							path, fset.Position(expr.Pos()))
					}
					if wantPkg, ok := forbiddenStdoutIdentifiers[expr.Sel.Name]; ok && pkgIdent.Name == wantPkg {
						t.Errorf("%s:%s: calls %s.%s, which writes to standard output",
							path, fset.Position(expr.Pos()), wantPkg, expr.Sel.Name)
					}
				case *ast.Ident:
					if expr.Name == "print" || expr.Name == "println" {
						t.Errorf("%s:%s: calls the builtin %s, which writes to standard output",
							path, fset.Position(expr.Pos()), expr.Name)
					}
				}
				return true
			})
		}
	}
	if checked == 0 {
		t.Fatal("TestStdout_NoProductionWrite: parsed zero files; a check that finds nothing to check is worse than no check")
	}
}
