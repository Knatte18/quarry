//go:build lsp

// stdio_lsp_test.go is this package's tier-3 test: it builds cmd/quarry-mcp into a real binary and
// talks to it over real OS pipes, the one thing an in-memory transport (transport_test.go,
// transport_errors_test.go) can never exercise. Those tier-2 tests prove the handler and schema
// logic against a real mcp.Client, but they connect over mcp.NewInMemoryTransports(), so a stray
// fmt.Println, a leftover log line, or a cobra side effect that writes to os.Stdout would never
// show up: nothing in that path actually shares a byte stream with anything else. A real stdio
// child process does share exactly one stream — its stdout — between the framed MCP protocol and
// every other thing the process might accidentally write there, and that is the risk that
// motivated building this binary as a separate, cobra-free process in the first place
// (cmd/quarry-mcp/main.go's own file comment). This file is the only test that can catch a
// regression there.
//
// The tier-2 handshake, tools/list, and tools/call assertions run over one mcp.CommandTransport
// session (newRealBinaryClient below): CommandTransport owns the child's stdout pipe and its framed
// reader consumes it, so it cannot also serve the stdout-purity assertion. That assertion instead
// spawns a second, separate child of the same built binary, wired to explicit StdinPipe/
// StdoutPipe/StderrPipe, and drives a hand-built initialize/initialized handshake over those pipes
// directly, reading every line the child writes to stdout and failing on any line that does not
// parse as a JSON object.

package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// repoRoot returns this worktree's module root, scanning up from this file's own location —
// internal/mcpserver/stdio_lsp_test.go is two directories below the repo root — mirroring
// internal/cli/assertnocallers_lsp_test.go's own repoRoot helper, which this package cannot import
// (per the facade-seam-usage Shared Decision, internal/mcpserver reaches quarry only through
// quarry/facade.go, and repoRoot is a same-package test helper, not part of that seam).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("repoRoot: could not determine this file's own location")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// buildQuarryMCPBinary builds cmd/quarry-mcp into a fresh t.TempDir() and returns the built
// binary's path, failing t on any build error.
func buildQuarryMCPBinary(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "quarry-mcp")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/quarry-mcp")
	cmd.Dir = repoRoot(t)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build -o %s ./cmd/quarry-mcp: %v\n%s", binPath, err, stderr.String())
	}
	return binPath
}

// newRealBinaryClient connects a real mcp.Client to binPath (built by buildQuarryMCPBinary) over an
// mcp.CommandTransport, passing --target-dir fixtureRoot on the command line so no committed client
// configuration file is involved. The session is closed via t.Cleanup.
func newRealBinaryClient(t *testing.T, binPath, fixtureRoot string) *mcp.ClientSession {
	t.Helper()

	transport := &mcp.CommandTransport{Command: exec.Command(binPath, "--target-dir", fixtureRoot)}
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-lsp-test-client", Version: "0.0.0"}, nil)

	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client.Connect(real binary) error = %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// TestRealBinary_HandshakeToolsListAndCalls_Integration proves the built quarry-mcp binary — not the
// in-process server transport_test.go and transport_errors_test.go exercise — completes a real MCP
// handshake, lists the same seven tools, and resolves a real textDocument_definition call (single
// and multi-entry) through a live gopls against the on-disk impact fixture.
func TestRealBinary_HandshakeToolsListAndCalls_Integration(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found on $PATH; install with: go install golang.org/x/tools/gopls@latest")
	}

	binPath := buildQuarryMCPBinary(t)
	fixtureRoot := filepath.Join(repoRoot(t), "testdata", "impactfixture")

	session := newRealBinaryClient(t, binPath, fixtureRoot)

	toolsRes, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error = %v", err)
	}
	if len(toolsRes.Tools) != len(wantToolNames) {
		t.Fatalf("len(toolsRes.Tools) = %d (%v); want %d (%v)", len(toolsRes.Tools), toolNames(toolsRes.Tools), len(wantToolNames), wantToolNames)
	}
	for _, name := range wantToolNames {
		toolByName(t, toolsRes.Tools, name)
	}

	singleRes, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"ApplyDiscount"}]}`),
	})
	if err != nil {
		t.Fatalf("CallTool(textDocument_definition, single entry) error = %v", err)
	}
	if singleRes.IsError {
		t.Fatalf("CallTool(textDocument_definition, single entry) IsError = true; content: %+v", singleRes.Content)
	}
	var singleOut definitionOutput
	structuredContentTo(t, singleRes, &singleOut)
	if len(singleOut.Results) != 1 {
		t.Fatalf("len(singleOut.Results) = %d; want 1", len(singleOut.Results))
	}
	if singleOut.Results[0].Status != statusFound {
		t.Errorf("singleOut.Results[0].Status = %q; want %q (resolved through a real gopls)", singleOut.Results[0].Status, statusFound)
	}

	multiRes, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"ApplyDiscount"},{"symbol":"ProcessRefund"}]}`),
	})
	if err != nil {
		t.Fatalf("CallTool(textDocument_definition, multi entry) error = %v", err)
	}
	if multiRes.IsError {
		t.Fatalf("CallTool(textDocument_definition, multi entry) IsError = true; content: %+v", multiRes.Content)
	}
	var multiOut definitionOutput
	structuredContentTo(t, multiRes, &multiOut)
	if len(multiOut.Results) != 2 {
		t.Fatalf("len(multiOut.Results) = %d; want 2 (array batching survives real serialization)", len(multiOut.Results))
	}
	wantSymbols := []string{"ApplyDiscount", "ProcessRefund"}
	for i, wantSymbol := range wantSymbols {
		target, ok := multiOut.Results[i].Target.(map[string]any)
		if !ok {
			t.Fatalf("multiOut.Results[%d].Target = %#v; want map[string]any", i, multiOut.Results[i].Target)
		}
		if target["symbol"] != wantSymbol {
			t.Errorf("multiOut.Results[%d].Target[\"symbol\"] = %v; want %q (input order)", i, target["symbol"], wantSymbol)
		}
		if multiOut.Results[i].Status != statusFound {
			t.Errorf("multiOut.Results[%d].Status = %q; want %q", i, multiOut.Results[i].Status, statusFound)
		}
	}
}

// jsonRPCLine builds one newline-delimited JSON-RPC 2.0 request or notification line: id is nil for
// a notification (the "id" key is omitted entirely, per spec) and a request id otherwise.
func jsonRPCLine(t *testing.T, id any, method string, params any) []byte {
	t.Helper()

	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		msg["params"] = params
	}
	if id != nil {
		msg["id"] = id
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("jsonRPCLine(%s): json.Marshal error = %v", method, err)
	}
	return append(data, '\n')
}

// TestRealBinary_StdoutCarriesOnlyJSONRPCFrames is the stdout-purity assertion this tier exists for:
// it spawns a second, separate child of the built binary, wired to explicit StdinPipe/StdoutPipe/
// StderrPipe, writes a hand-built initialize request followed by the initialized notification to its
// stdin as newline-delimited JSON, then reads every line the child writes to stdout and fails on any
// line that does not parse as a JSON object. mcp.CommandTransport cannot serve this assertion because
// it owns the child's stdout pipe and its framed reader consumes it. The child's stderr is read
// concurrently and is expected to carry the startup target-directory line — it must never be
// conflated with stdout.
func TestRealBinary_StdoutCarriesOnlyJSONRPCFrames(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found on $PATH; install with: go install golang.org/x/tools/gopls@latest")
	}

	binPath := buildQuarryMCPBinary(t)
	fixtureRoot := filepath.Join(repoRoot(t), "testdata", "impactfixture")

	cmd := exec.Command(binPath, "--target-dir", fixtureRoot)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe error = %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe error = %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe error = %v", err)
	}

	// Drain stderr concurrently so a full pipe buffer can never deadlock the child; the lines
	// collected here are asserted after the child exits, kept entirely separate from stdout.
	stderrLines := make(chan []string, 1)
	go func() {
		var lines []string
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		stderrLines <- lines
	}()

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start error = %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	stdoutReader := bufio.NewReader(stdout)

	if _, err := stdin.Write(jsonRPCLine(t, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "stdio-purity-test-client", "version": "0.0.0"},
	})); err != nil {
		t.Fatalf("write initialize request: %v", err)
	}

	initLine, err := readLineWithTimeout(t, stdoutReader, 30*time.Second)
	if err != nil {
		t.Fatalf("read initialize response: %v", err)
	}
	assertJSONObjectLine(t, initLine)

	if _, err := stdin.Write(jsonRPCLine(t, nil, "notifications/initialized", map[string]any{})); err != nil {
		t.Fatalf("write initialized notification: %v", err)
	}

	// Closing stdin is this transport's own documented shutdown signal (mcp.CommandTransport's own
	// Close does the same); the child then exits and stdout reaches EOF, so every remaining line it
	// wrote can be drained and checked without an arbitrary read deadline.
	if err := stdin.Close(); err != nil {
		t.Fatalf("stdin.Close error = %v", err)
	}

	for {
		line, err := stdoutReader.ReadString('\n')
		if len(line) > 0 {
			assertJSONObjectLine(t, []byte(line))
		}
		if err != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("cmd.Wait error = %v", err)
	}

	lines := <-stderrLines
	found := false
	for _, l := range lines {
		if bytes.Contains([]byte(l), []byte("resolved target directory")) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("child stderr lines = %v; want a line reporting the resolved target directory", lines)
	}
}

// readLineWithTimeout reads one newline-terminated line from r, failing t if none arrives within d.
func readLineWithTimeout(t *testing.T, r *bufio.Reader, d time.Duration) ([]byte, error) {
	t.Helper()

	type result struct {
		line []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		line, err := r.ReadBytes('\n')
		done <- result{line, err}
	}()

	select {
	case res := <-done:
		return res.line, res.err
	case <-time.After(d):
		return nil, fmt.Errorf("no line read within %s", d)
	}
}

// assertJSONObjectLine fails t unless line's trimmed bytes unmarshal into a JSON object — the
// stdout-purity contract this tier exists to check.
func assertJSONObjectLine(t *testing.T, line []byte) {
	t.Helper()

	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return
	}
	var v map[string]any
	if err := json.Unmarshal(trimmed, &v); err != nil {
		t.Fatalf("stdout line is not a well-formed JSON object: %v; line: %q", err, trimmed)
	}
}
