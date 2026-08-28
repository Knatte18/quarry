// cgoguard_nocgo.go fails a CGO_ENABLED=0 build at compile time with a readable message, instead
// of letting the build proceed until a tree-sitter-importing package hits the raw cgo linker
// error. See cgoguard.go for why this pair of files lives in the engine root package rather than
// in internal/quarryengine/treesitter.
//
// The failure mode here is deliberately a compile error, not a _test.go assertion: a test only
// runs under `go test`, so a CGO_ENABLED=0 `go build ./...`, `go vet ./...` (this task's own
// top-level verify command), or `go run` would sail past a test-only guard and hit the linker
// error this file exists to replace. Do not "fix" the undefined-identifier error below by deleting
// this file -- that removes the guard entirely, and the linker dump it prevents is far less
// actionable than the message the identifier itself spells out.

//go:build !cgo

package quarryengine

// var _ deliberately references an undeclared identifier so the compiler's "undefined:" error
// names the fix directly: quarry requires CGO_ENABLED=1 and a C toolchain to build, because the
// treesitter package links against tree-sitter's C grammars.
var _ = quarry_requires_CGO_ENABLED_1_with_a_C_toolchain
