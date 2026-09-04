// layering_test.go mechanically enforces two of this task's rules over internal/mcpserver's own
// files and cmd/quarry-mcp's files: the facade-only rule (no file here imports internal/engine, or
// any package below it, directly) and the stdout rule (no production file here writes to standard
// output).
//
// This tree carries no import check of any kind before this file: the facade-only rule holds by
// convention everywhere it holds today, including in internal/cli. This is the first place the rule
// is mechanical rather than one row added to an existing mechanism — there is no engine-side
// layering test in this tree for this file to extend, and the predecessor these checks may recall
// is V1's, on the frozen branch, reference material rather than something this test extends.
//
// Be precise about what the import check is and is not: it scans each file's own import block, so
// it catches a direct engine import only. It cannot catch the engine reached through an
// intermediate package, and it deliberately does not try — a transitive check would fail
// immediately and permanently, since the quarry facade these packages are required to depend on
// imports the engine itself, which is the whole point of the facade. The residual gap: an engine
// dependency introduced through some future intermediate internal/ package would pass this check.
//
// The stdout check is deliberately blunt: it matches on the identifiers os.Stdout, fmt.Print,
// fmt.Println, fmt.Printf, print, and println, not on reachability. A legitimate future need to
// name os.Stdout — there is none today — is expected to argue with this test rather than slip past
// it.

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

// forbiddenEngineImport and its path-prefix form are the one import this task's two packages must
// never carry directly.
const forbiddenEngineImport = "github.com/Knatte18/quarry/internal/engine"

// checkedDirs returns the two directories this file's checks walk: this package's own directory
// and cmd/quarry-mcp's, located from runtime.Caller(0) rather than from the working directory, so
// this test does not depend on where it is run from.
func checkedDirs(t *testing.T) (mcpserverDir, quarryMCPDir string) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("checkedDirs: runtime.Caller(0) failed to resolve this file's path")
	}
	mcpserverDir = filepath.Dir(thisFile)
	// mcpserverDir is .../internal/mcpserver; the module root is two directories up.
	moduleRoot := filepath.Dir(filepath.Dir(mcpserverDir))
	quarryMCPDir = filepath.Join(moduleRoot, "cmd", "quarry-mcp")
	return mcpserverDir, quarryMCPDir
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

// TestFacadeOnly_NoDirectEngineImport walks every .go file — production and _test.go alike — in
// internal/mcpserver and cmd/quarry-mcp, and fails on any import path equal to
// forbiddenEngineImport or carrying it as a path prefix. The facade-only rule binds the test files
// too, so _test.go files are not excluded from this walk.
func TestFacadeOnly_NoDirectEngineImport(t *testing.T) {
	mcpserverDir, quarryMCPDir := checkedDirs(t)

	fset := token.NewFileSet()
	checked := 0
	for _, dir := range []string{mcpserverDir, quarryMCPDir} {
		for _, path := range goFilesIn(t, dir) {
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parser.ParseFile(%q, ImportsOnly): %v", path, err)
			}
			checked++
			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if importPath == forbiddenEngineImport || strings.HasPrefix(importPath, forbiddenEngineImport+"/") {
					t.Errorf("%s: imports %q directly; every engine identifier must reach this package through github.com/Knatte18/quarry/quarry",
						path, importPath)
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
	mcpserverDir, quarryMCPDir := checkedDirs(t)

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
