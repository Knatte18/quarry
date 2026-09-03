//go:build ignore

package main

// Tool is a fixture function on the "main" side of the tie. The walk parses this file's text
// regardless of the build constraint above — it is not the Go toolchain — so this clause still
// votes.
func Tool() {}
