// toc_depth_test.go pins depth -1 as accepted and recursing to the fixture tree's bottom, depth -2
// and depth -7 as rejected over the protocol without pinning the rejection's wording there, and the
// wording itself by calling tocResult directly — this batch's one deliberate departure from
// protocol-only testing, stated in the Batch Scope.
//
// This is the load-bearing one of this batch's schema tests, and the two layers deserve to be named
// precisely: the schema's minimum is what actually rejects a protocol call carrying depth -2 or
// depth -7 — mcp.AddTool's generated wrapper validates arguments against the input schema before
// tocResult's handler ever runs, so tocResult's own rejection is unreachable over the wire while that
// minimum is in force. tocResult's own check is what owns the wording, and what would still reject if
// the schema's minimum were ever dropped or this function were reached from in-process code that
// bypasses the SDK wrapper. Either layer is needed at all because the engine's directory walk
// (internal/engine/walk.go) decrements depth with no floor and stops only at zero or at the
// whole-tree sentinel — an unvalidated negative depth is an unbounded walk that returns a
// plausible-looking answer rather than an error, a defect that would reach T7 as a cost measurement
// rather than as a failure.

package mcpserver

import (
	"context"
	"fmt"
	"testing"

	"github.com/Knatte18/quarry/quarry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDepthAll_RecursesToBottom(t *testing.T) {
	client := connectedClient(t, fixtureRepoRoot(t))

	answer := callTOC(t, client, "alpha", map[string]any{"depth": -1})
	if len(answer.Dirs) != 1 || answer.Dirs[0].Dir != "alpha/sub" {
		t.Fatalf("depth -1: Dirs = %+v; want exactly one entry named %q", answer.Dirs, "alpha/sub")
	}

	var foundLeaf bool
	for _, fe := range answer.Dirs[0].Files {
		if fe.Name == "leaf.go" {
			foundLeaf = true
		}
	}
	if !foundLeaf {
		t.Errorf("depth -1: alpha/sub's files = %+v; want leaf.go among them", answer.Dirs[0].Files)
	}
}

func TestDepthBelowMinusOne_RejectedOverProtocol(t *testing.T) {
	client := connectedClient(t, fixtureRepoRoot(t))

	for _, depth := range []int{-2, -7} {
		t.Run(fmt.Sprintf("depth=%d", depth), func(t *testing.T) {
			got, err := client.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "toc",
				Arguments: map[string]any{"target": "alpha", "depth": depth},
			})
			if err != nil {
				t.Fatalf("CallTool(depth=%d): %v", depth, err)
			}
			if !got.IsError {
				t.Fatalf("CallTool(depth=%d): IsError = false; want true", depth)
			}
			if len(got.Content) != 1 {
				t.Fatalf("CallTool(depth=%d): %d content blocks; want 1", depth, len(got.Content))
			}
			text, ok := got.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("CallTool(depth=%d): content block is a %T; want *mcp.TextContent", depth, got.Content[0])
			}
			if text.Text == "" {
				t.Errorf("CallTool(depth=%d): text is empty; want the SDK's own arguments-validation message", depth)
			}
			// The wording is the SDK's own arguments-validation message, not tocResult's own — it is
			// deliberately not asserted here; see this file's header comment and
			// TestDepthBelowMinusOne_RejectionWording below, which asserts it against tocResult
			// directly instead.
		})
	}
}

func TestDepthBelowMinusOne_RejectionWording(t *testing.T) {
	root := fixtureRepoRoot(t)
	repo, err := quarry.Open(root)
	if err != nil {
		t.Fatalf("quarry.Open(%q): %v", root, err)
	}

	for _, depth := range []int{-2, -7} {
		t.Run(fmt.Sprintf("depth=%d", depth), func(t *testing.T) {
			got := tocResult(repo, root, tocInput{Target: "alpha", Depth: depth})
			if !got.IsError {
				t.Fatalf("tocResult(depth=%d): IsError = false; want true", depth)
			}
			if len(got.Content) != 1 {
				t.Fatalf("tocResult(depth=%d): %d content blocks; want 1", depth, len(got.Content))
			}
			text, ok := got.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("tocResult(depth=%d): content block is a %T; want *mcp.TextContent", depth, got.Content[0])
			}
			wantMsg := fmt.Sprintf("--depth must be -1 (whole tree) or a non-negative integer, got %d", depth)
			want := string(quarry.RenderErrorJSON(wantMsg))
			if text.Text != want {
				t.Errorf("tocResult(depth=%d) text = %q; want %q", depth, text.Text, want)
			}
		})
	}
}
