// Package quarryengine is the root of the extraction engine. It holds the cgo build guard
// (cgoguard.go, cgoguard_nocgo.go) and the error sentinel its subpackages share; the work is done
// in toc (symbol extraction over a parse tree) and treesitter (the parse-and-release seam).
package quarryengine
