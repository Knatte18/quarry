// main.go is the whole of the quarry-mcp binary. Nothing in this file writes to standard output —
// that stream carries framed MCP traffic only, and a stray write there is the one way this binary
// fails catastrophically and silently. Everything below is testable in-process through
// internal/mcpserver precisely because this file holds no logic: it resolves a root, opens a repo,
// constructs a server, and runs it.
//
// The standard-error line this file writes naming the resolved root serves interactive and
// operator use only: the ladder harness sets no standard-error sink on the measured process, so
// during a ladder run the line goes nowhere and a misrooting is observed instead from the answers
// themselves.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Knatte18/quarry/internal/mcpserver"
	"github.com/Knatte18/quarry/quarry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	root := flag.String("root", "", "repository root to serve; overrides discovery from the working directory")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	resolvedRoot, err := mcpserver.ResolveRoot(*root, cwd)
	if err != nil {
		fatal(err)
	}

	repo, err := quarry.Open(resolvedRoot)
	if err != nil {
		fatal(err)
	}

	fmt.Fprintln(os.Stderr, "quarry-mcp: serving "+resolvedRoot)

	server := mcpserver.NewServer(repo, resolvedRoot)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fatal(err)
	}
}

// fatal writes err's message to standard error and exits non-zero. Every failure this binary can
// have, whether before the transport starts or from the transport's own Run call once it is up,
// goes through this one disposition. A clean end of the client's session is not a failure and never
// reaches this function.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
