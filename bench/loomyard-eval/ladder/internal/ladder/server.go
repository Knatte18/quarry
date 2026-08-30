// server.go ports the second half of scripts/run_ladder.py's "SERVER BINARY, PER-RUN MCP CONFIG, AND
// WARM-UP" section: building the quarry-mcp server binary once per invocation, and generating the
// per-run --mcp-config document that declares it.

package ladder

import (
	"fmt"
	"os"
	"path/filepath"
)

// Builder is the seam BuildServer invokes the actual build through: dir is the working directory the
// build runs in, env is the environment it runs with, and args is the command's argument vector.
// Mirroring the git runner the pinned-worktree lifecycle takes, this is an explicit parameter rather
// than a package-level variable, so a test substitutes it instead of mutating shared state.
type Builder func(dir string, env []string, args ...string) (string, error)

// BuildServer builds the quarry-mcp server binary at <repoRoot>/quarry-mcp with CGO_ENABLED=1 forced
// into the environment, returning its absolute path.
//
// The warm-start path (a built binary) is used rather than the committed `go run ./cmd/quarry-mcp` form
// so a cold build cache cannot make a run's first connection exceed the client's connect timeout.
// Returns a *HarnessError naming the CGO toolchain requirement when the build fails, since a missing C
// toolchain fails at compile time (the toc verbs' tree-sitter backend links C grammars) with output
// that otherwise reads as unrelated.
func BuildServer(repoRoot string, build Builder) (string, error) {
	binaryPath := filepath.Join(repoRoot, "quarry-mcp")
	// Appending, rather than pre-filtering an existing CGO_ENABLED entry out of os.Environ(), still
	// forces the value: exec.Cmd's own env resolution keeps only the last entry for a duplicate key,
	// which is what the Python port's dict-overwrite of build_env["CGO_ENABLED"] achieves too.
	env := append(os.Environ(), "CGO_ENABLED=1")

	output, err := build(repoRoot, env, "go", "build", "-o", binaryPath, "./cmd/quarry-mcp")
	if err != nil {
		return "", &HarnessError{Message: fmt.Sprintf(
			"build_server: go build ./cmd/quarry-mcp failed -- requires CGO_ENABLED=1 with a C toolchain:\n%s", output,
		)}
	}

	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return "", fmt.Errorf("ladder: build server: resolve absolute path for %s: %w", binaryPath, err)
	}
	return absPath, nil
}

// MCPConfigDocument is the --mcp-config mapping declaring a single server named "quarry", whose command
// is the built binary's absolute path and whose args carry an explicit --target-dir <targetDir>.
//
// Its env block sets QUARRY_STATE_DIR and QUARRY_BUILD_TAGS to the empty string, leaving QUARRY_CONFIG
// untouched -- the first of the three points the environment scrub now applies at (the other two are
// the harness's own process environment, scrubbed by ScrubbedEnv, and the warm-up client's spawn
// environment in warm.go). Whether declaring this block on a server entry replaces or augments the
// process environment the launching CLI would otherwise inherit into the spawned server is an open
// question this port does not resolve; see the plan's Shared Decision on the settings-source risk this
// was originally recorded alongside (that one has since been verified working -- see session.go's
// LaunchCommand doc comment -- this one has not).
func MCPConfigDocument(serverPath, targetDir string) map[string]any {
	return map[string]any{
		"mcpServers": map[string]any{
			"quarry": map[string]any{
				"command": serverPath,
				"args":    []string{"--target-dir", targetDir},
				"env": map[string]string{
					"QUARRY_STATE_DIR":  "",
					"QUARRY_BUILD_TAGS": "",
				},
			},
		},
	}
}
