package ladder

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestMCPConfigDocument_ToolLessNonControl is the regression for the crash card 6 fixes: a tool-less
// non-control cell -- an empty Allowed and an explicit control: false -- against a ladder whose
// Server is nil. Before the switch to GrantsTools, this input reached the granted branch and
// returned the "declares no server block" error, killing the run before rep 1.
func TestMCPConfigDocument_ToolLessNonControl(t *testing.T) {
	l := &Ladder{}
	notControl := false
	cfg := Config{ID: "e1-pack", Allowed: nil, Control: &notControl}

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
		t.Errorf("MCPConfigDocument() for a tool-less non-control cell has %d servers; want 0", len(servers))
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

// buildStubMCPServer builds testdata/stubmcp into a fresh temporary binary and returns its path.
func buildStubMCPServer(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "stubmcp")
	cmd := exec.Command("go", "build", "-o", binPath, "./testdata/stubmcp")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./testdata/stubmcp: %v\n%s", err, out)
	}
	return binPath
}

// rpcCall writes one JSON-RPC request line to w and reads exactly one response line back from r,
// returning the response's decoded "result" object.
func rpcCall(t *testing.T, w *bufio.Writer, r *bufio.Reader, id int, method string) map[string]any {
	t.Helper()
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request %s: %v", method, err)
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		t.Fatalf("write request %s: %v", method, err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("flush request %s: %v", method, err)
	}

	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read response to %s: %v", method, err)
	}
	var resp struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("decode response to %s: %v (line: %s)", method, err, line)
	}
	return resp.Result
}

// TestStubMCPServer_HandshakeMatchesGeneratedConfig builds the stub MCP server, generates the
// per-cell MCP configuration document through this package's own writer for a cell granting one of
// the stub's two advertised tools, launches the server using the command and arguments that document
// declares, completes the initialize and tools-list handshake, and asserts that the tool names the
// harness would pass as its allowlist -- the prefix applied to the cell's granted names -- correspond
// to tools the server actually advertises.
func TestStubMCPServer_HandshakeMatchesGeneratedConfig(t *testing.T) {
	binPath := buildStubMCPServer(t)

	l := &Ladder{
		QuarryTools: []string{"toc", "other"},
		Server: &ServerSpec{
			Name:  "quarry",
			Build: "./testdata/stubmcp",
		},
	}
	cfg := Config{ID: "a2-toc-dir", Ladder: "a", Task: "t", Allowed: []string{"toc"}}

	doc, err := MCPConfigDocument(l, cfg, binPath, t.TempDir())
	if err != nil {
		t.Fatalf("MCPConfigDocument() = %v; want no error", err)
	}
	var parsed struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(doc, &parsed); err != nil {
		t.Fatalf("Unmarshal(doc) = %v", err)
	}
	server, ok := parsed.MCPServers[l.ServerName()]
	if !ok {
		t.Fatalf("generated config names no %q server: %s", l.ServerName(), doc)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, server.Command, server.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() = %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() = %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start stub server declared by generated config: %v", err)
	}
	t.Cleanup(func() {
		stdin.Close()
		_ = cmd.Wait()
	})

	w := bufio.NewWriter(stdin)
	r := bufio.NewReader(stdout)

	rpcCall(t, w, r, 1, "initialize")
	toolsResult := rpcCall(t, w, r, 2, "tools/list")

	rawTools, ok := toolsResult["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list result carries no tools array: %v", toolsResult)
	}
	advertised := map[string]bool{}
	for _, rt := range rawTools {
		tool, ok := rt.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tool["name"].(string)
		advertised[name] = true
	}

	// This is the whole point: the harness's own allowlist, derived from the generated document's
	// declared command by way of grantedToolNames, must name tools the launched server actually
	// advertises.
	for _, name := range grantedToolNames(l, cfg) {
		if !strings.HasPrefix(name, l.MCPPrefix()) {
			continue
		}
		bare := strings.TrimPrefix(name, l.MCPPrefix())
		if !advertised[bare] {
			t.Errorf("harness allowlist entry %q (bare %q) is not advertised by the stub server: %v", name, bare, advertised)
		}
	}
	if len(advertised) == 0 {
		t.Fatal("stub server advertised no tools at all")
	}
}
