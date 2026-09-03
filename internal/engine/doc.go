// Package engine is the extraction engine: the one package that walks a parsed source tree into
// typed symbol and table-of-contents results. treesitter (internal/engine/treesitter) is its
// parse-and-release seam, and cgoguard (internal/cgoguard) is the build guard that fails a
// CGO_ENABLED=0 build with a readable message before treesitter's cgo linker error would.
//
// This package returns typed results and typed errors only. It never emits JSON, never decides an
// exit code, and never resolves a caller's cwd.
package engine
