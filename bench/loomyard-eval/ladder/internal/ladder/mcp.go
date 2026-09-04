// mcp.go builds the per-cell MCP configuration document the claude binary reads at startup and
// lazily builds the MCP server binary the document points at. A control cell's document declares no
// server at all; a granted cell's document declares exactly one, with the pinned worktree path
// substituted into its argument list.

package ladder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// targetDirPlaceholder is the only placeholder MCPConfigDocument substitutes into a granted cell's
// server argument list. This is a new contract this plan defines: no ladder file in the tree uses it
// today, so the substitution has no consumer until the MCP-server task writes one. The migrated
// ladder file's header comment documents the same spelling.
const targetDirPlaceholder = "{target_dir}"

// mcpServerDoc is the shape of one server entry in an MCP configuration document.
type mcpServerDoc struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// mcpConfigDoc is the top-level shape MCPConfigDocument marshals: a map of server name to its
// declaration. A control cell's document has an empty map.
type mcpConfigDoc struct {
	MCPServers map[string]mcpServerDoc `json:"mcpServers"`
}

// MCPConfigDocument returns the JSON MCP configuration document for one cell. For a control -- a
// config whose allowed list is empty -- the returned document declares exactly an empty servers map
// and nothing else. For a granted cell, the document declares exactly one server under the ladder
// file's server name, with serverBinary as its command, the server block's argument list with every
// occurrence of the literal placeholder token "{target_dir}" replaced by targetDir, and the server
// block's environment map. "{target_dir}" is the only placeholder the argument list supports. A
// granted cell whose ladder file declares no server block is an error naming the cell id.
func MCPConfigDocument(l *Ladder, cfg Config, serverBinary, targetDir string) ([]byte, error) {
	if cfg.IsControl() {
		return json.MarshalIndent(mcpConfigDoc{MCPServers: map[string]mcpServerDoc{}}, "", "  ")
	}

	if l.Server == nil {
		return nil, fmt.Errorf("mcp config for cell %s: ladder file declares no server block", cfg.ID)
	}

	args := make([]string, len(l.Server.Args))
	for i, a := range l.Server.Args {
		args[i] = strings.ReplaceAll(a, targetDirPlaceholder, targetDir)
	}

	doc := mcpConfigDoc{
		MCPServers: map[string]mcpServerDoc{
			l.ServerName(): {
				Command: serverBinary,
				Args:    args,
				Env:     l.Server.Env,
			},
		},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// WriteMCPConfig writes doc under quarryRepoRoot's own .scratch/ladder/ directory, creating that
// directory when absent, and returns the written path. The join is done here rather than by the
// caller so the scratch location stays a single fact instead of one every caller re-spells; the
// invocation's own argument list is never echoed to a transcript, so a configuration path under the
// repository never reaches one.
func WriteMCPConfig(quarryRepoRoot, name string, doc []byte) (string, error) {
	dir := filepath.Join(quarryRepoRoot, ".scratch", "ladder")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("write mcp config %s: %w", name, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, doc, 0o644); err != nil {
		return "", fmt.Errorf("write mcp config %s: %w", path, err)
	}
	return path, nil
}

// BuildServer runs `go build` for buildTarget through r, with the command's working directory set to
// quarryRepoRoot -- the build target is a repository-relative path, so resolving it against the
// harness process's own working directory would make the build depend on where the operator happened
// to invoke it -- and CGO_ENABLED set to "1" in the command's own environment map rather than in the
// harness's process environment, because the target server links C grammars. It returns the hex
// sha256 of the produced binary at outPath.
func BuildServer(ctx context.Context, r Runner, quarryRepoRoot, buildTarget, outPath string) (string, error) {
	if err := r.Run(ctx, Cmd{
		Dir:  quarryRepoRoot,
		Name: "go",
		Args: []string{"build", "-o", outPath, buildTarget},
		Env:  map[string]string{"CGO_ENABLED": "1"},
	}); err != nil {
		return "", fmt.Errorf("build server %s: %w", buildTarget, err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		return "", fmt.Errorf("build server %s: read built binary: %w", buildTarget, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("build server %s: hash built binary: %w", buildTarget, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
