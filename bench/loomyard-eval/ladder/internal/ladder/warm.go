// warm.go ports scripts/run_ladder.py's daemon warm-up: one MCP tool call against the spawned
// quarry-mcp server, followed by an assertion that the call actually started a daemon. Unlike the
// Python port's hand-rolled JSON-RPC framing (mcp_call), Warm is built on the Go MCP SDK's stdio
// client, already a direct module dependency, because the Python framing exists only to work around
// the standard-library-only constraint that does not apply here.

package ladder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WarmUpTool is the tool Warm calls to pre-warm the daemon, mirroring the Python port's WARM_UP_TOOL.
// It must be daemon-backed: toc_dir and toc_file reach the tree-sitter path directly and never
// EnsureServer, so warming with either of them would start no daemon at all. The query behind this call
// needs no match: workspace_symbol's handler calls resolveCall once for the whole call before resolving
// any target, so the daemon starts whether or not the query resolves -- which is why Warm's
// post-condition is the state file's existence and not the call's result payload.
const WarmUpTool = "workspace_symbol"

// warmUpTimeout bounds Warm's tool call, mirroring the Python port's _WARM_UP_TIMEOUT_S.
const warmUpTimeout = 60 * time.Second

// Warm pre-warms the daemon for one main-matrix run: spawns serverPath rooted at targetDir with env as
// its environment, completes the MCP initialize handshake, calls WarmUpTool once, then asserts that a
// daemon.json now exists at the state directory ResolveStateDir resolves for targetDir and cacheDir --
// callers supply env from the scrub (ScrubbedEnv), which is what determines that resolved directory.
//
// Returns a *HarnessError when the post-condition fails: the warm-up call completed but no daemon.json
// appeared, meaning the call did not start a daemon. Called by the main-matrix driver immediately before
// each run's dispatch, per run rather than once per worktree, since the daemon self-expires after its
// idle timeout; never called for a config with cold: true.
func Warm(serverPath, targetDir string, env []string, cacheDir string) error {
	cmd := exec.Command(serverPath, "--target-dir", targetDir)
	cmd.Env = env

	client := mcp.NewClient(&mcp.Implementation{Name: "loomyard-eval-ladder", Version: "1.0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), warmUpTimeout)
	defer cancel()

	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return fmt.Errorf("ladder: warm: connect to %s: %w", serverPath, err)
	}
	defer session.Close()

	arguments, err := json.Marshal(map[string]any{"targets": []map[string]string{{"query": "Run"}}})
	if err != nil {
		return fmt.Errorf("ladder: warm: marshal %s arguments: %w", WarmUpTool, err)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: WarmUpTool, Arguments: json.RawMessage(arguments)})
	if err != nil {
		return fmt.Errorf("ladder: warm: call %s: %w", WarmUpTool, err)
	}
	if result.IsError {
		return &HarnessError{Message: fmt.Sprintf("warm: %s returned a tool error: %+v", WarmUpTool, result.Content)}
	}

	stateDir, err := ResolveStateDir(targetDir, cacheDir, env)
	if err != nil {
		return err
	}
	stateFile := DaemonStateFile(stateDir, daemonLang)
	if _, err := os.Stat(stateFile); err != nil {
		if os.IsNotExist(err) {
			return &HarnessError{Message: fmt.Sprintf(
				"warm_daemon: no daemon.json at %s after warming %s -- the warm-up call did not start a daemon", stateFile, targetDir,
			)}
		}
		return fmt.Errorf("ladder: warm: stat %s: %w", stateFile, err)
	}
	return nil
}
