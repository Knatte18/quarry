// main.go is the entire quarry binary: one call into cli.Run and nothing else. Everything below
// os.Exit is testable in-process through internal/cli, so the golden tests capture the binary's
// exact bytes without a build step or os/exec, and any logic added here would be the one part of
// the CLI no test covers. The binary is built as quarry at the repository root, which .gitignore
// already reserves — /quarry ignores the built binary while !/quarry/ keeps the facade package
// directory tracked — so no .gitignore change is needed and none is made.

package main

import (
	"os"

	"github.com/Knatte18/quarry/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
