// main.go is quarry's binary entry point: a thin wrapper around cli.RunCLI, with no flag parsing,
// path resolution, or cobra construction of its own. All of that belongs to cli.Command() —
// duplicating any of it here would give quarry two places where its CLI surface is defined.
// cmd/quarry-mcp is the intended future peer: it will consume the same quarry engine package
// through a different front-end (an MCP server), not by shelling out to this binary.

package main

import (
	"os"

	"github.com/Knatte18/quarry/internal/cli"
)

func main() {
	os.Exit(cli.RunCLI(os.Stdout, os.Args[1:]))
}
