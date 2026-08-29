// Package mcpserver binds quarry/facade.go onto MCP tools.
// It imports internal/cli for the resolution helpers (config path, state directory, build tags)
// that mirror the CLI's own per-call setup, but it never imports internal/quarryengine directly —
// every engine identifier it needs reaches it through the quarry facade, exactly as internal/cli
// already does. layering_test.go enforces the facade-only rule mechanically.
package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverVersion is the MCP server's own version string, reported to a connecting client as part
// of its Implementation identity. It tracks this package's own development, not quarry's module
// version.
const serverVersion = "0.1.0"

// minTargets and maxTargets bound every tool's "targets" array parameter: at least one entry is
// required, and at most 64 may be submitted in a single call.
const (
	minTargets = 1
	maxTargets = 64
)

// Config holds the launch-only values NewServer needs to start the MCP server. Every field here is
// resolved exactly once, at server startup, before any handler can run — a handler never sees these
// raw values, only what resolveCall derives from them per call.
type Config struct {
	// TargetDir is the default project directory to detect the language in and root the server at,
	// used whenever a call omits its own targetDir override. It is always absolute by the time
	// NewServer runs.
	TargetDir string
	// ConfigPath is the explicit servers.yaml overlay path, mirroring the CLI's --config flag.
	// An empty value defers to cli.ResolveConfigPath's own precedence.
	ConfigPath string
	// StateDir is the explicit daemon state directory, mirroring the CLI's --state-dir flag.
	// An empty value defers to cli.ResolveStateDir's own precedence.
	StateDir string
	// Timeout is the deadline applied per entry's facade call, mirroring the CLI's --timeout flag.
	Timeout time.Duration
}

// ResolveLaunchTargetDir resolves the server's default target directory at startup: flagValue,
// absolutised, when non-empty, or the process's current working directory otherwise.
// os.Getwd already returns an absolute path, so no second absolutisation is applied to it.
// This runs exactly once, at server startup, before any handler can run, so every downstream
// consumer of Config.TargetDir only ever sees an absolute path.
func ResolveLaunchTargetDir(flagValue string) (string, error) {
	if flagValue != "" {
		abs, err := filepath.Abs(flagValue)
		if err != nil {
			return "", fmt.Errorf("mcpserver: resolve target dir: %w", err)
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("mcpserver: resolve target dir: %w", err)
	}
	return cwd, nil
}

// NewServer constructs quarry's MCP server for cfg. cfg.TargetDir must already be absolute —
// ResolveLaunchTargetDir is what callers use to guarantee that — and NewServer returns an error
// naming the field otherwise, rather than silently resolving it again.
//
// NewServer registers no tools of its own; each handler batch adds its own register* call here.
// Nothing in this package writes to os.Stdout — that stream is reserved for the MCP transport.
func NewServer(cfg Config) (*mcp.Server, error) {
	if !filepath.IsAbs(cfg.TargetDir) {
		return nil, fmt.Errorf("mcpserver: NewServer: cfg.TargetDir must be absolute, got %q", cfg.TargetDir)
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "quarry", Version: serverVersion}, nil)

	if err := registerLSPTools(s, cfg); err != nil {
		return nil, err
	}

	return s, nil
}
