// lspclient_guard_test.go is a narrow file-scoped guard on internal/quarryengine/lsp/lspclient.go:
// the stdio LSP client carries no dependency beyond the standard library and the single
// internal/quarryengine leaf package it needs for quarryengine.Logger and quarryengine.ErrServerTimeout.
// It guards this one file only — it must never be generalized into a per-file allowed-set table,
// which would be an allowlist through the back door; the widened rule is expressed as one hardcoded
// import path, not a table, because this file has exactly one legitimate first-party dependency.

package lsp

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// allowedNonStdlibImport is the single first-party import path lspclient.go may use beyond the
// standard library. Every other non-stdlib import, first-party or third-party, still fails this
// guard.
const allowedNonStdlibImport = "github.com/Knatte18/quarry/internal/quarryengine"

// TestLSPClientGuard_StdlibOnly verifies that lspclient.go imports only the standard library plus
// allowedNonStdlibImport.
func TestLSPClientGuard_StdlibOnly(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine quarry source directory location")
	}
	pkgDir := filepath.Dir(file)
	target := filepath.Join(pkgDir, "lspclient.go")

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("lspclient.go not found at %s; the guard's hardcoded target path must track the file: %v", target, err)
	}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, target, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse lspclient.go: %v", err)
	}

	var failures []string

	for _, imp := range astFile.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		if importPath == allowedNonStdlibImport {
			continue
		}

		// A stdlib import path has no '.' in its first path segment
		// (e.g. "fmt", "os", "go/parser") — a domain that would need a
		// registered TLD (e.g. "github.com/...") always contains one.
		firstSegment := importPath
		if idx := strings.IndexByte(importPath, '/'); idx >= 0 {
			firstSegment = importPath[:idx]
		}
		isStdlib := !strings.Contains(firstSegment, ".")

		if isStdlib {
			continue
		}

		failures = append(failures, importPath)
	}

	if len(failures) > 0 {
		t.Errorf("lspclient.go must import only the standard library plus %q; found: %v", allowedNonStdlibImport, failures)
	}
}
