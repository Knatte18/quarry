// main.go is quarry-mcp's binary entry point. It is a separate binary from cmd/quarry because
// stdio MCP cannot tolerate anything else writing to stdout — that stream is reserved entirely for
// the framed MCP protocol — and this process guarantees exactly that: no cobra.Command is ever
// constructed or executed here, so no internal/output.Ok or internal/output.Err call site is
// reachable from this process, and nothing in this file writes to os.Stdout.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/internal/mcpserver"
)

func main() {
	targetDir := flag.String("target-dir", "", "project directory to detect the language in and root the server at (default: cwd)")
	configPath := flag.String("config", "", "explicit path to a servers.yaml overlay, overriding $QUARRY_CONFIG and the user config directory default")
	stateDir := flag.String("state-dir", "", "explicit daemon state directory, overriding $QUARRY_STATE_DIR and the user cache directory default")
	timeout := flag.Duration("timeout", 30*time.Second, "deadline applied per entry's facade call")
	flag.Parse()

	resolvedTargetDir, err := mcpserver.ResolveLaunchTargetDir(*targetDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "quarry-mcp: resolved target directory %s\n", resolvedTargetDir)

	server, err := mcpserver.NewServer(mcpserver.Config{
		TargetDir:  resolvedTargetDir,
		ConfigPath: *configPath,
		StateDir:   *stateDir,
		Timeout:    *timeout,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
