// Package cli is the whole of the quarry command below os.Exit: flag parsing, calling
// internal/repopath to resolve a repository root and a cwd-relative target, the exit-code
// mapping, and the choice of renderer. cmd/quarry holds one line and nothing else, and this split
// exists so the golden tests can capture exactly the bytes the binary emits without building or
// exec'ing anything.
//
// This package is the only layer with a working directory — internal/engine deliberately
// performs no git discovery and no cwd resolution — and the two path frames never mix: input is
// interpreted where the user is, output is always repository-root relative with forward slashes.
package cli
