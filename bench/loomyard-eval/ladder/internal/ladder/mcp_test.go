package ladder

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mcpTestLadder() *Ladder {
	return &Ladder{
		Server: &ServerSpec{
			Name:  "quarry",
			Build: "./cmd/quarry-mcp",
			Args:  []string{"--target", "{target_dir}", "--verbose"},
			Env:   map[string]string{"FOO": "bar"},
		},
	}
}

func TestMCPConfigDocument_Control(t *testing.T) {
	l := mcpTestLadder()
	cfg := Config{ID: "a0-none", Allowed: nil}

	got, err := MCPConfigDocument(l, cfg, "/bin/server", "/worktree")
	if err != nil {
		t.Fatalf("MCPConfigDocument() = %v; want no error", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("Unmarshal() = %v", err)
	}
	servers, ok := doc["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("MCPConfigDocument() has no mcpServers map: %s", got)
	}
	if len(servers) != 0 {
		t.Errorf("MCPConfigDocument() for a control cell has %d servers; want 0", len(servers))
	}
	if len(doc) != 1 {
		t.Errorf("MCPConfigDocument() for a control cell has %d top-level keys; want exactly mcpServers", len(doc))
	}
}

func TestMCPConfigDocument_Granted(t *testing.T) {
	l := mcpTestLadder()
	cfg := Config{ID: "a3-full", Allowed: []string{"toc"}}

	got, err := MCPConfigDocument(l, cfg, "/bin/server", "/pinned/worktree")
	if err != nil {
		t.Fatalf("MCPConfigDocument() = %v; want no error", err)
	}

	var doc struct {
		MCPServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("Unmarshal() = %v", err)
	}
	if len(doc.MCPServers) != 1 {
		t.Fatalf("MCPConfigDocument() for a granted cell has %d servers; want exactly 1", len(doc.MCPServers))
	}
	server, ok := doc.MCPServers["quarry"]
	if !ok {
		t.Fatalf("MCPConfigDocument() does not name the server %q: %s", "quarry", got)
	}
	if server.Command != "/bin/server" {
		t.Errorf("server.Command = %q; want %q", server.Command, "/bin/server")
	}
	wantArgs := []string{"--target", "/pinned/worktree", "--verbose"}
	if len(server.Args) != len(wantArgs) {
		t.Fatalf("server.Args = %v; want %v", server.Args, wantArgs)
	}
	for i := range wantArgs {
		if server.Args[i] != wantArgs[i] {
			t.Errorf("server.Args[%d] = %q; want %q", i, server.Args[i], wantArgs[i])
		}
	}
	if server.Env["FOO"] != "bar" {
		t.Errorf("server.Env[FOO] = %q; want %q", server.Env["FOO"], "bar")
	}
}

func TestMCPConfigDocument_GrantedNoServerBlockErrors(t *testing.T) {
	l := &Ladder{}
	cfg := Config{ID: "a3-full", Allowed: []string{"toc"}}

	_, err := MCPConfigDocument(l, cfg, "/bin/server", "/pinned/worktree")
	if err == nil {
		t.Fatal("MCPConfigDocument() = nil error; want an error naming the cell id")
	}
	if !strings.Contains(err.Error(), cfg.ID) {
		t.Errorf("MCPConfigDocument() error = %q; want it to name %q", err, cfg.ID)
	}
}

func TestBuildServer(t *testing.T) {
	quarryRepoRoot := t.TempDir()
	outPath := filepath.Join(t.TempDir(), "server-bin")
	if err := os.WriteFile(outPath, []byte("fake binary contents"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	envBefore, hadBefore := os.LookupEnv("CGO_ENABLED")

	r := &recordingRunner{}
	sha, err := BuildServer(context.Background(), r, quarryRepoRoot, "./cmd/quarry-mcp", outPath)
	if err != nil {
		t.Fatalf("BuildServer() = %v; want no error", err)
	}
	if sha == "" {
		t.Error("BuildServer() returned an empty sha256")
	}

	if len(r.calls) != 1 {
		t.Fatalf("BuildServer() made %d calls; want 1", len(r.calls))
	}
	call := r.calls[0]
	if call.Dir != quarryRepoRoot {
		t.Errorf("BuildServer() command Dir = %q; want %q", call.Dir, quarryRepoRoot)
	}
	if call.Env["CGO_ENABLED"] != "1" {
		t.Errorf("BuildServer() command Env[CGO_ENABLED] = %q; want %q", call.Env["CGO_ENABLED"], "1")
	}

	envAfter, hadAfter := os.LookupEnv("CGO_ENABLED")
	if hadBefore != hadAfter || envBefore != envAfter {
		t.Errorf("BuildServer() changed the harness process's own CGO_ENABLED: before=(%q,%v) after=(%q,%v)", envBefore, hadBefore, envAfter, hadAfter)
	}
}
