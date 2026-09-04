// toc_golden_test.go golds the toc tool's payload bytes for six cases against the committed fixture
// repository, and adds the one assertion that goes further than any golden can: for the same target
// and the same options, the tool's own text and quarry.RenderJSON of the facade's own answer are
// identical bytes, which is what makes "a mirror of the CLI" testable rather than aspirational — a
// golden can agree with a wrong implementation forever, but this comparison cannot.
//
// The six committed goldens under testdata/golden/ are produced by running this package's tests once
// with "-update" against the fixture repository, never hand-written: a hand-written golden pins the
// wrong bytes and then passes forever. Run
// `go test ./internal/mcpserver/... -run TestGolden_TOC -update` after a deliberate change to the
// payload shape, and review the diff before committing the regenerated goldens.

package mcpserver

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/quarry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// updateGoldens is "-update", checked by TestGolden_TOC to decide whether to compare the tool's
// response against its committed golden or to rewrite that golden from the current run. Declared
// here, once, so no other file in this package needs its own flag.Bool for the same name.
var updateGoldens = flag.Bool("update", false, "regenerate this package's goldens under testdata/golden from the current fixture repository")

func TestGolden_TOC(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		depth      int
		symbolsSet bool
		symbolsVal bool
		golden     string
	}{
		{name: "dir-default", target: "alpha", golden: "toc-dir.json"},
		{name: "file-default", target: "alpha/alpha.go", golden: "toc-file.json"},
		{name: "root-depth1", target: "", depth: 1, golden: "toc-dir-depth1.json"},
		{name: "root-depth-all", target: "", depth: -1, golden: "toc-dir-depth-all.json"},
		{name: "dir-symbols-true", target: "alpha", symbolsSet: true, symbolsVal: true, golden: "toc-dir-symbols-true.json"},
		{name: "file-symbols-false", target: "alpha/alpha.go", symbolsSet: true, symbolsVal: false, golden: "toc-file-symbols-false.json"},
	}

	root := fixtureRepoRoot(t)
	client := connectedClient(t, root)
	repo, err := quarry.Open(root)
	if err != nil {
		t.Fatalf("quarry.Open(%q): %v", root, err)
	}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{"target": tt.target}
			if tt.depth != 0 {
				args["depth"] = tt.depth
			}
			if tt.symbolsSet {
				args["symbols"] = tt.symbolsVal
			}

			got, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "toc", Arguments: args})
			if err != nil {
				t.Fatalf("CallTool(%q, %v): %v", tt.target, args, err)
			}
			if got.IsError {
				t.Fatalf("CallTool(%q, %v): IsError = true; content: %v", tt.target, args, got.Content)
			}
			if len(got.Content) != 1 {
				t.Fatalf("CallTool(%q, %v): %d content blocks; want 1", tt.target, args, len(got.Content))
			}
			text, ok := got.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("CallTool(%q, %v): content block is a %T; want *mcp.TextContent", tt.target, args, got.Content[0])
			}
			if got.StructuredContent != nil {
				t.Errorf("CallTool(%q, %v): StructuredContent = %v; want nil", tt.target, args, got.StructuredContent)
			}

			compareTOCGolden(t, tt.golden, []byte(text.Text))

			opts := quarry.TOCOptions{Depth: tt.depth}
			if tt.symbolsSet {
				v := tt.symbolsVal
				opts.Symbols = &v
			}
			answer, err := repo.TOC(tt.target, opts)
			if err != nil {
				t.Fatalf("repo.TOC(%q, %+v): %v", tt.target, opts, err)
			}
			want, err := quarry.RenderJSON(answer)
			if err != nil {
				t.Fatalf("quarry.RenderJSON: %v", err)
			}
			if !bytes.Equal(want, []byte(text.Text)) {
				t.Errorf("toc tool text does not mirror quarry.RenderJSON(repo.TOC(%q, %+v)) (-want +got):\n--- want ---\n%s\n--- got ---\n%s",
					tt.target, opts, want, text.Text)
			}
		})
	}
}

// compareTOCGolden compares got byte-for-byte against the committed golden testdata/golden/name, or
// — under "-update" — rewrites that golden from got.
func compareTOCGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name)
	if *updateGoldens {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %q: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v", path, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden %q mismatch (-want +got):\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
