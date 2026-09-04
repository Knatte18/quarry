// toc_errors_test.go pins the two failure envelopes tocResult builds by reusing the CLI's own
// wording — a target that does not exist, and a target that escapes the repository — and the one
// case that is deliberately not a failure: a target that is a broken symbolic link. Both the handler
// (toc.go) and the engine (internal/engine/repo.go, internal/engine/toc.go) stat with os.Lstat and
// never os.Stat, so a symbolic link named directly as the target is answered as a file rather than
// followed. Asserting a failure envelope for that case would be pinning the opposite of the rule this
// case exists to protect.
//
// A symbolic link is not something the committed fixture repository can portably hold, so the third
// case builds its own tree under this repository's gitignored scratch directory with root_test.go's
// writeScratchTree, opens it as its own repository, and skips — rather than fails — on a platform
// that cannot create a symbolic link.

package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/quarry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTOCErrors_TargetNotFound(t *testing.T) {
	client := connectedClient(t, fixtureRepoRoot(t))

	const target = "alpha/does-not-exist.go"
	got, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toc",
		Arguments: map[string]any{"target": target},
	})
	if err != nil {
		t.Fatalf("CallTool(target=%q): %v", target, err)
	}
	if !got.IsError {
		t.Fatalf("CallTool(target=%q): IsError = false; want true", target)
	}
	if len(got.Content) != 1 {
		t.Fatalf("CallTool(target=%q): %d content blocks; want 1", target, len(got.Content))
	}
	text, ok := got.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(target=%q): content block is a %T; want *mcp.TextContent", target, got.Content[0])
	}
	want := string(quarry.RenderErrorJSON("target not found: " + target))
	if text.Text != want {
		t.Errorf("CallTool(target=%q) text = %q; want %q", target, text.Text, want)
	}
}

func TestTOCErrors_TargetOutsideRepository(t *testing.T) {
	client := connectedClient(t, fixtureRepoRoot(t))

	const target = "../outside"
	got, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toc",
		Arguments: map[string]any{"target": target},
	})
	if err != nil {
		t.Fatalf("CallTool(target=%q): %v", target, err)
	}
	if !got.IsError {
		t.Fatalf("CallTool(target=%q): IsError = false; want true", target)
	}
	if len(got.Content) != 1 {
		t.Fatalf("CallTool(target=%q): %d content blocks; want 1", target, len(got.Content))
	}
	text, ok := got.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(target=%q): content block is a %T; want *mcp.TextContent", target, got.Content[0])
	}
	want := string(quarry.RenderErrorJSON("target outside repository: " + target))
	if text.Text != want {
		t.Errorf("CallTool(target=%q) text = %q; want %q", target, text.Text, want)
	}
}

func TestTOCErrors_BrokenSymlinkNeverFollowed(t *testing.T) {
	root := writeScratchTree(t, "toc-errors-broken-symlink", map[string]string{
		"placeholder.go": "package placeholder\n",
	})

	linkPath := filepath.Join(root, "broken-link")
	if err := os.Symlink(filepath.Join(root, "does-not-exist"), linkPath); err != nil {
		t.Skipf("os.Symlink unsupported on this platform: %v", err)
	}

	client := connectedClient(t, root)
	answer := callTOC(t, client, "broken-link", nil)

	var found bool
	for _, fe := range answer.Files {
		if fe.Name == "broken-link" {
			found = true
		}
	}
	if !found {
		t.Errorf("broken symlink: Files = %+v; want an entry named %q", answer.Files, "broken-link")
	}
}
