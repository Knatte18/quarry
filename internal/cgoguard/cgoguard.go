// cgoguard.go is the cgo half of a matched pair with cgoguard_nocgo.go: this file compiles only
// under a cgo-enabled build and carries no code, so the pair reads together as one guard with zero
// cost when cgo is available. This package declares nothing, imports nothing, and exists only so
// that a CGO_ENABLED=0 build fails with a readable message before the raw cgo linker error. It is
// blank-imported by internal/engine/treesitter, so it is strictly earlier in the build graph than
// anything that links tree-sitter. The guard cannot live inside internal/engine/treesitter itself,
// or inside internal/engine (which transitively imports treesitter), because under CGO_ENABLED=0
// those packages cannot compile at all -- a guard placed there would be unreachable exactly when it
// is needed.

//go:build cgo

package cgoguard

// This file intentionally declares nothing. Its only purpose is to exist as the cgo-side twin of
// cgoguard_nocgo.go's build-time failure, so a reader sees the guard is a matched pair rather than
// a stray file.
