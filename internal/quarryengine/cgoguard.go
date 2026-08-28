// cgoguard.go is the cgo half of a matched pair with cgoguard_nocgo.go: this file compiles only
// under a cgo-enabled build and carries no code, so the pair reads together as one guard with zero
// cost when cgo is available. Both files sit in this root package, not in
// internal/quarryengine/treesitter, because under CGO_ENABLED=0 the treesitter package itself
// cannot compile at all -- it transitively imports cgo through go-tree-sitter -- so a guard placed
// there would be unreachable exactly when it is needed. This root package imports no cgo, so its
// guard still builds and runs under CGO_ENABLED=0, and it is the first
// internal/quarryengine/... package `go test ./...` and `go build ./...` reach.

//go:build cgo

package quarryengine

// This file intentionally declares nothing. Its only purpose is to exist as the cgo-side twin of
// cgoguard_nocgo.go's build-time failure, so a reader sees the guard is a matched pair rather than
// a stray file.
