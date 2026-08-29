// facade.go declares the package-level function variables every handler calls the quarry facade
// through, mirroring the userConfigDir/userCacheDir seam convention in internal/cli/paths.go: a
// test can substitute a stub for any of the seven, so the five otherwise-untestable
// language-server-backed handlers still get per-status and error-mapping coverage without a live
// gopls. No behaviour lives here or in quarry/facade.go — the facade is behaviour-free, and
// quarry/facade_test.go enforces that mechanically.

package mcpserver

import "github.com/Knatte18/quarry/quarry"

// definitionFn calls quarry.Definition. Tests substitute a stub to exercise definitionCommand's
// handler without a live language server.
var definitionFn = quarry.Definition

// referencesFn calls quarry.References. Tests substitute a stub to exercise refsCommand's handler
// without a live language server.
var referencesFn = quarry.References

// symbolFn calls quarry.Symbol. Tests substitute a stub to exercise symbolCommand's handler
// without a live language server.
var symbolFn = quarry.Symbol

// callersFn calls quarry.Callers. Tests substitute a stub to exercise the callers/assert-no-callers
// handler without a live language server.
var callersFn = quarry.Callers

// impactFn calls quarry.Impact. Tests substitute a stub to exercise the impact handler without a
// live language server.
var impactFn = quarry.Impact

// tocFileFn calls quarry.TOCFile. Tests substitute a stub to exercise the toc-file handler.
var tocFileFn = quarry.TOCFile

// tocDirFn calls quarry.TOCDir. Tests substitute a stub to exercise the toc-dir handler.
var tocDirFn = quarry.TOCDir
