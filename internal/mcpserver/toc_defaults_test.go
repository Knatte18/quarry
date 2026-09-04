// toc_defaults_test.go pins two absent-property defaults, each with its own named test: the
// pointer-versus-bool trap in the absent-symbols default, and the absent-depth default. An
// implementation that maps an absent symbols property to false rather than to nil — the engine's
// per-target default — passes every other test in this batch, including every golden in
// toc_golden_test.go, since none of those cases omits the property; only a test that specifically
// calls the tool without it can catch the trap. The absent-depth case is a separate failure mode
// from a separate cause and gets its own assertion for the same reason.
//
// wireFileEntry and wireDirAnswer below decode just enough of the toc payload's JSON shape for these
// two assertions, and are reused by every later file in this batch that needs to read a payload
// rather than compare it against a golden. This package cannot import the engine types that shape
// is defined by, per this task's facade-only rule, so these are a local, test-only mirror of the
// wire contract — read off the decoded JSON, per card 12's note, not off internal/engine/answer.go.

package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// wireFileEntry decodes the fields these tests need from one Files entry: Name to locate the entry,
// and Symbols as a json.RawMessage so its presence — not its content — is what "populated" or
// "absent" means. A non-nil, possibly empty pointer marshals as a present key, per
// internal/engine/answer.go's FileEntry.Symbols doc comment; a plain bool field here would collapse
// that distinction back into the very trap this test exists to catch.
type wireFileEntry struct {
	Name    string          `json:"name"`
	Symbols json.RawMessage `json:"symbols,omitempty"`
}

// wireDirAnswer decodes just enough of a toc payload to read Dir, Files and Dirs.
type wireDirAnswer struct {
	Dir   string          `json:"dir"`
	Files []wireFileEntry `json:"files,omitempty"`
	Dirs  []wireDirAnswer `json:"dirs,omitempty"`
}

// callTOC calls the toc tool through client for target with the arguments in extra merged alongside
// "target", fails the test if the call errors or its result carries the error flag, and decodes the
// result's one text content block as a wireDirAnswer.
func callTOC(t *testing.T, client *mcp.ClientSession, target string, extra map[string]any) wireDirAnswer {
	t.Helper()

	args := map[string]any{"target": target}
	for k, v := range extra {
		args[k] = v
	}

	got, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "toc", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%q, %v): %v", target, extra, err)
	}
	if got.IsError {
		t.Fatalf("CallTool(%q, %v): IsError = true; content: %v", target, extra, got.Content)
	}
	if len(got.Content) != 1 {
		t.Fatalf("CallTool(%q, %v): %d content blocks; want 1", target, extra, len(got.Content))
	}
	text, ok := got.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(%q, %v): content block is a %T; want *mcp.TextContent", target, extra, got.Content[0])
	}

	var answer wireDirAnswer
	if err := json.Unmarshal([]byte(text.Text), &answer); err != nil {
		t.Fatalf("CallTool(%q, %v): decoding payload: %v", target, extra, err)
	}
	return answer
}

func TestAbsentSymbols_FileDefaultOn_DirDefaultOff(t *testing.T) {
	client := connectedClient(t, fixtureRepoRoot(t))

	fileAnswer := callTOC(t, client, "alpha/alpha.go", nil)
	if len(fileAnswer.Files) != 1 {
		t.Fatalf("file target: %d files; want 1", len(fileAnswer.Files))
	}
	if fileAnswer.Files[0].Symbols == nil {
		t.Errorf("file target with symbols omitted: Files[0].Symbols is absent; want populated (the per-target default is on for a file target)")
	}

	dirAnswer := callTOC(t, client, "alpha", nil)
	if len(dirAnswer.Files) == 0 {
		t.Fatal("dir target: 0 files; want at least one, to make the absent-symbols assertion meaningful")
	}
	for _, fe := range dirAnswer.Files {
		if fe.Symbols != nil {
			t.Errorf("dir target with symbols omitted: Files[%q].Symbols is populated; want absent (the per-target default is off for a directory target)", fe.Name)
		}
	}
}

func TestAbsentDepth_BehavesAsZero(t *testing.T) {
	client := connectedClient(t, fixtureRepoRoot(t))

	answer := callTOC(t, client, "alpha", nil)

	wantFiles := map[string]bool{"doc.go": false, "alpha.go": false}
	for _, fe := range answer.Files {
		if _, ok := wantFiles[fe.Name]; ok {
			wantFiles[fe.Name] = true
		}
	}
	for name, seen := range wantFiles {
		if !seen {
			t.Errorf("absent depth: %q not listed among alpha's own files: %+v", name, answer.Files)
		}
	}

	if len(answer.Dirs) != 1 || answer.Dirs[0].Dir != "alpha/sub" {
		t.Fatalf("absent depth: Dirs = %+v; want exactly one entry named %q", answer.Dirs, "alpha/sub")
	}
	if len(answer.Dirs[0].Files) != 0 || len(answer.Dirs[0].Dirs) != 0 {
		t.Errorf("absent depth: alpha/sub carries Files=%+v Dirs=%+v; want both empty, since depth 0 names a subdirectory without descending into it",
			answer.Dirs[0].Files, answer.Dirs[0].Dirs)
	}
}
