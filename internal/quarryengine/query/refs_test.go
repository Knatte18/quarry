// refs_test.go is the untagged, spawn-free counterpart to refs_integration_test.go: it exercises
// References's error-mapping paths that do not require a real language server.
// exec.LookPath failing for a nonexistent binary happens before any subprocess is spawned, so this
// test needs no //go:build integration tag and no installed language server.
// It builds clients over newPipeTransportPair/fakeServer, a fake-transport harness duplicated
// from internal/quarryengine/lsp/lspclient_test.go, since package lsp's own copy is a
// _test.go-only helper and therefore not importable from here, per the test-support-helpers
// Shared Decision.

package query

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/daemon/daemontest"
	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

// pipeTransport wires client and server sides over two io.Pipes.
type pipeTransport struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (p pipeTransport) Close() error {
	var err error
	for _, c := range p.closers {
		if cerr := c.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// newPipeTransportPair returns two linked transports for client and server.
func newPipeTransportPair() (client, server pipeTransport) {
	clientReadServerWrite, serverWriteClientRead := io.Pipe()
	serverReadClientWrite, clientWriteServerRead := io.Pipe()

	client = pipeTransport{
		Reader:  clientReadServerWrite,
		Writer:  clientWriteServerRead,
		closers: []io.Closer{clientReadServerWrite, clientWriteServerRead},
	}
	server = pipeTransport{
		Reader:  serverReadClientWrite,
		Writer:  serverWriteClientRead,
		closers: []io.Closer{serverReadClientWrite, serverWriteClientRead},
	}
	return client, server
}

// fakeServer reads and writes Content-Length-framed JSON-RPC messages.
type fakeServer struct {
	r *bufio.Reader
	w io.Writer
}

func newFakeServer(rw io.ReadWriter) *fakeServer {
	return &fakeServer{r: bufio.NewReader(rw), w: rw}
}

// fakeServerMessage mirrors lsp's wire message shape for testing.
type fakeServerMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// readMessage reads one Content-Length-framed message, reporting errors via t.Errorf.
func (s *fakeServer) readMessage(t *testing.T) (msg fakeServerMessage, ok bool) {
	t.Helper()
	contentLength := -1
	for {
		line, err := s.r.ReadString('\n')
		if err != nil {
			t.Errorf("fakeServer: read header: %v", err)
			return fakeServerMessage{}, false
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, cut := strings.Cut(line, ":"); cut && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				t.Errorf("fakeServer: parse Content-Length: %v", err)
				return fakeServerMessage{}, false
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		t.Errorf("fakeServer: message missing Content-Length header")
		return fakeServerMessage{}, false
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.r, body); err != nil {
		t.Errorf("fakeServer: read body: %v", err)
		return fakeServerMessage{}, false
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Errorf("fakeServer: unmarshal body: %v", err)
		return fakeServerMessage{}, false
	}
	return msg, true
}

// writeMessage frames and writes a value, reporting errors via t.Errorf.
func (s *fakeServer) writeMessage(t *testing.T, v any) bool {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Errorf("fakeServer: marshal: %v", err)
		return false
	}
	if _, err := fmt.Fprintf(s.w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		t.Errorf("fakeServer: write header: %v", err)
		return false
	}
	if _, err := s.w.Write(body); err != nil {
		t.Errorf("fakeServer: write body: %v", err)
		return false
	}
	return true
}

// respond writes a success response for a request ID.
func (s *fakeServer) respond(t *testing.T, id json.RawMessage, result any) bool {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Errorf("fakeServer: marshal result: %v", err)
		return false
	}
	return s.writeMessage(t, fakeServerMessage{JSONRPC: "2.0", ID: id, Result: raw})
}

// TestCollectInFileMatches tests the collectInFileMatches helper on a hierarchical symbol tree.
func TestCollectInFileMatches(t *testing.T) {
	tree := []lsp.DocumentSymbol{
		{
			Name: "Reader",
			Children: []lsp.DocumentSymbol{
				{Name: "Open", SelectionRange: lsp.Range{Start: lsp.Position{Line: 1, Character: 1}}},
				{Name: "Close", SelectionRange: lsp.Range{Start: lsp.Position{Line: 2, Character: 1}}},
			},
		},
		{
			Name: "Writer",
			Children: []lsp.DocumentSymbol{
				{Name: "Open", SelectionRange: lsp.Range{Start: lsp.Position{Line: 10, Character: 1}}},
			},
		},
		{Name: "TopLevelFunc", SelectionRange: lsp.Range{Start: lsp.Position{Line: 20, Character: 1}}},
	}

	tests := []struct {
		name      string
		query     string
		wantLines []int
	}{
		{name: "TopLevelMatch", query: "TopLevelFunc", wantLines: []int{20}},
		{name: "NestedChildMatch", query: "Close", wantLines: []int{2}},
		{name: "NoMatch", query: "NoSuchSymbol", wantLines: nil},
		{name: "TwoSameNameMatchesAcrossTypes", query: "Open", wantLines: []int{1, 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := collectInFileMatches(tree, tt.query)
			if len(matches) != len(tt.wantLines) {
				t.Fatalf("collectInFileMatches(tree, %q) returned %d matches; want %d", tt.query, len(matches), len(tt.wantLines))
			}
			for i, line := range tt.wantLines {
				if got := matches[i].SelectionRange.Start.Line; got != line {
					t.Errorf("collectInFileMatches(tree, %q)[%d].SelectionRange.Start.Line = %d; want %d", tt.query, i, got, line)
				}
			}
		})
	}
}

// TestResolvePosition_InFileSingleMatchReturnsSelectionRangeStart verifies resolvePosition returns
// selection range starts for InFile matches.
func TestResolvePosition_InFileSingleMatchReturnsSelectionRangeStart(t *testing.T) {
	clientTransport, serverTransport := newPipeTransportPair()
	defer clientTransport.Close()
	defer serverTransport.Close()

	client := lsp.NewClientFromRW(clientTransport)
	server := newFakeServer(serverTransport)

	done := make(chan struct{})
	go func() {
		defer close(done)
		initReq, ok := server.readMessage(t)
		if !ok {
			return
		}
		if !server.respond(t, initReq.ID, map[string]any{
			"capabilities": map[string]any{"documentSymbolProvider": true},
		}) {
			return
		}
		server.readMessage(t) // initialized notification

		req, ok := server.readMessage(t)
		if !ok {
			return
		}
		if req.Method != "textDocument/documentSymbol" {
			t.Errorf("fakeServer: got request method %q; want %q", req.Method, "textDocument/documentSymbol")
			return
		}
		server.respond(t, req.ID, []map[string]any{
			{
				"name": "Open",
				"kind": 6,
				"range": map[string]any{
					"start": map[string]any{"line": 6, "character": 0},
					"end":   map[string]any{"line": 8, "character": 1},
				},
				"selectionRange": map[string]any{
					"start": map[string]any{"line": 6, "character": 15},
					"end":   map[string]any{"line": 6, "character": 19},
				},
			},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, "file:///tmp/example"); err != nil {
		t.Fatalf("initialize() returned unexpected error: %v", err)
	}

	opts := Options{
		TargetDir: "/tmp/example",
		Query:     Query{InFile: &InFileQuery{File: "/tmp/example/foo.go", Name: "Open"}},
		Timeout:   5 * time.Second,
	}
	fileURI, pos, err := resolvePosition(ctx, client, opts, "go", registry.Entry{Command: []string{"gopls"}})
	<-done
	if err != nil {
		t.Fatalf("resolvePosition() returned unexpected error: %v", err)
	}
	if fileURI != "file:///tmp/example/foo.go" {
		t.Errorf("resolvePosition() fileURI = %q; want %q", fileURI, "file:///tmp/example/foo.go")
	}
	wantPos := lsp.Position{Line: 6, Character: 15}
	if pos != wantPos {
		t.Errorf("resolvePosition() pos = %+v; want %+v (the match's SelectionRange.Start)", pos, wantPos)
	}
}

// TestResolvePosition_InFileZeroMatchesYieldsErrSymbolNotFound asserts a documentSymbol result with
// no exact-name match maps to quarryengine.ErrSymbolNotFoundSentinel.
func TestResolvePosition_InFileZeroMatchesYieldsErrSymbolNotFound(t *testing.T) {
	clientTransport, serverTransport := newPipeTransportPair()
	defer clientTransport.Close()
	defer serverTransport.Close()

	client := lsp.NewClientFromRW(clientTransport)
	server := newFakeServer(serverTransport)

	done := make(chan struct{})
	go func() {
		defer close(done)
		initReq, ok := server.readMessage(t)
		if !ok {
			return
		}
		if !server.respond(t, initReq.ID, map[string]any{
			"capabilities": map[string]any{"documentSymbolProvider": true},
		}) {
			return
		}
		server.readMessage(t) // initialized notification

		req, ok := server.readMessage(t)
		if !ok {
			return
		}
		if req.Method != "textDocument/documentSymbol" {
			t.Errorf("fakeServer: got request method %q; want %q", req.Method, "textDocument/documentSymbol")
			return
		}
		server.respond(t, req.ID, []map[string]any{})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, "file:///tmp/example"); err != nil {
		t.Fatalf("initialize() returned unexpected error: %v", err)
	}

	opts := Options{
		TargetDir: "/tmp/example",
		Query:     Query{InFile: &InFileQuery{File: "/tmp/example/foo.go", Name: "NoSuchSymbol"}},
		Timeout:   5 * time.Second,
	}
	_, _, err := resolvePosition(ctx, client, opts, "go", registry.Entry{Command: []string{"gopls"}})
	<-done
	if !errors.Is(err, quarryengine.ErrSymbolNotFoundSentinel) {
		t.Errorf("resolvePosition() err = %v; want errors.Is(err, quarryengine.ErrSymbolNotFoundSentinel)", err)
	}
}

// TestResolvePosition_InFileMultipleMatchesYieldsErrAmbiguousSymbol asserts a documentSymbol result
// with two exact-name matches (e.g.
// a same-named method on two distinct types in the same file) maps to quarryengine.ErrAmbiguousSymbolSentinel,
// with Candidates formatted as file:line:col strings.
func TestResolvePosition_InFileMultipleMatchesYieldsErrAmbiguousSymbol(t *testing.T) {
	clientTransport, serverTransport := newPipeTransportPair()
	defer clientTransport.Close()
	defer serverTransport.Close()

	client := lsp.NewClientFromRW(clientTransport)
	server := newFakeServer(serverTransport)

	done := make(chan struct{})
	go func() {
		defer close(done)
		initReq, ok := server.readMessage(t)
		if !ok {
			return
		}
		if !server.respond(t, initReq.ID, map[string]any{
			"capabilities": map[string]any{"documentSymbolProvider": true},
		}) {
			return
		}
		server.readMessage(t) // initialized notification

		req, ok := server.readMessage(t)
		if !ok {
			return
		}
		if req.Method != "textDocument/documentSymbol" {
			t.Errorf("fakeServer: got request method %q; want %q", req.Method, "textDocument/documentSymbol")
			return
		}
		server.respond(t, req.ID, []map[string]any{
			{
				"name": "Reader",
				"kind": 23,
				"range": map[string]any{
					"start": map[string]any{"line": 0, "character": 0},
					"end":   map[string]any{"line": 10, "character": 1},
				},
				"selectionRange": map[string]any{
					"start": map[string]any{"line": 0, "character": 5},
					"end":   map[string]any{"line": 0, "character": 11},
				},
				"children": []map[string]any{
					{
						"name": "Open",
						"kind": 6,
						"range": map[string]any{
							"start": map[string]any{"line": 2, "character": 0},
							"end":   map[string]any{"line": 4, "character": 1},
						},
						"selectionRange": map[string]any{
							"start": map[string]any{"line": 2, "character": 15},
							"end":   map[string]any{"line": 2, "character": 19},
						},
					},
				},
			},
			{
				"name": "Writer",
				"kind": 23,
				"range": map[string]any{
					"start": map[string]any{"line": 12, "character": 0},
					"end":   map[string]any{"line": 20, "character": 1},
				},
				"selectionRange": map[string]any{
					"start": map[string]any{"line": 12, "character": 5},
					"end":   map[string]any{"line": 12, "character": 11},
				},
				"children": []map[string]any{
					{
						"name": "Open",
						"kind": 6,
						"range": map[string]any{
							"start": map[string]any{"line": 14, "character": 0},
							"end":   map[string]any{"line": 16, "character": 1},
						},
						"selectionRange": map[string]any{
							"start": map[string]any{"line": 14, "character": 15},
							"end":   map[string]any{"line": 14, "character": 19},
						},
					},
				},
			},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, "file:///tmp/example"); err != nil {
		t.Fatalf("initialize() returned unexpected error: %v", err)
	}

	opts := Options{
		TargetDir: "/tmp/example",
		Query:     Query{InFile: &InFileQuery{File: "/tmp/example/foo.go", Name: "Open"}},
		Timeout:   5 * time.Second,
	}
	_, _, err := resolvePosition(ctx, client, opts, "go", registry.Entry{Command: []string{"gopls"}})
	<-done
	var ambiguous *quarryengine.ErrAmbiguousSymbol
	if !errors.As(err, &ambiguous) {
		t.Fatalf("resolvePosition() err = %v; want *quarryengine.ErrAmbiguousSymbol", err)
	}
	if !errors.Is(err, quarryengine.ErrAmbiguousSymbolSentinel) {
		t.Errorf("resolvePosition() err = %v; want errors.Is(err, quarryengine.ErrAmbiguousSymbolSentinel)", err)
	}
	want := []string{"/tmp/example/foo.go:3:16", "/tmp/example/foo.go:15:16"}
	if len(ambiguous.Candidates) != len(want) {
		t.Fatalf("resolvePosition() ambiguous.Candidates = %v; want %v", ambiguous.Candidates, want)
	}
	for i := range want {
		if ambiguous.Candidates[i] != want[i] {
			t.Errorf("resolvePosition() ambiguous.Candidates[%d] = %q; want %q", i, ambiguous.Candidates[i], want[i])
		}
	}
}

// TestResolvePosition_InFileUnsupportedDocumentSymbolNeverSendsRequest asserts that when the
// server's initialize response omits documentSymbolProvider, resolvePosition's InFile branch
// returns ErrResolverUnsupported and never issues a textDocument/documentSymbol request at all —
// mirroring symbol_test.go's TestSymbolFromClient_UnsupportedWorkspaceSymbolNeverSendsRequest
// precedent for the workspace/symbol path.
func TestResolvePosition_InFileUnsupportedDocumentSymbolNeverSendsRequest(t *testing.T) {
	clientTransport, serverTransport := newPipeTransportPair()

	client := lsp.NewClientFromRW(clientTransport)
	server := newFakeServer(serverTransport)

	initDone := make(chan struct{})
	go func() {
		defer close(initDone)
		req, ok := server.readMessage(t)
		if !ok {
			return
		}
		if req.Method != "initialize" {
			t.Errorf("fakeServer: got request method %q; want %q", req.Method, "initialize")
			return
		}
		// Deliberately omit documentSymbolProvider so
		// client.SupportsDocumentSymbol() reports false.
		if !server.respond(t, req.ID, map[string]any{"capabilities": map[string]any{}}) {
			return
		}
		server.readMessage(t) // initialized notification
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, "file:///tmp/example"); err != nil {
		t.Fatalf("initialize() returned unexpected error: %v", err)
	}
	<-initDone

	if client.SupportsDocumentSymbol() {
		t.Fatal("supportsDocumentSymbol() = true; want false (server omitted documentSymbolProvider)")
	}

	// Guard the documentSymbol phase: this goroutine blocks on the first
	// byte of any further message. If resolvePosition's InFile branch
	// wrongly calls client.DocumentSymbols anyway, that byte arrives and
	// fails the test; otherwise the transport close below unblocks the read
	// with a transport-closed error, the quiet path this goroutine takes
	// when nothing was ever sent.
	unexpectedRequest := make(chan struct{})
	go func() {
		defer close(unexpectedRequest)
		if _, err := server.r.ReadByte(); err == nil {
			t.Error("fakeServer: received an unexpected byte after an unsupported-capability resolvePosition InFile call; want no request sent at all")
		}
	}()

	opts := Options{
		TargetDir: "/tmp/example",
		Query:     Query{InFile: &InFileQuery{File: "/tmp/example/foo.go", Name: "Open"}},
		Timeout:   5 * time.Second,
	}
	_, _, err := resolvePosition(ctx, client, opts, "go", registry.Entry{Command: []string{"gopls"}})
	if !errors.Is(err, quarryengine.ErrResolverUnsupportedSentinel) {
		t.Errorf("resolvePosition() err = %v; want errors.Is(err, quarryengine.ErrResolverUnsupportedSentinel)", err)
	}

	clientTransport.Close()
	serverTransport.Close()
	<-unexpectedRequest
}

// TestReferences_HasNativeDaemonRoutesThroughEnsureServer proves that a registry entry with
// HasNativeDaemon: true takes the daemon.EnsureServer path, not the legacy newLSPClient(entry.Command)
// path — without spawning a real gopls.
// It swaps installGoToolchain for a fake that always fails with a distinct, recognizable error,
// then asserts References's returned error wraps that exact sentinel: only reachable if the call
// chain was References -> lookup -> acquireConnection -> daemon.EnsureServer -> resolveGoToolchain -> the
// fake installer.
// daemon.EnsureServer resolves the toolchain directly and returns on failure before ever attempting
// ensureSupervised or ensureNative, so this fake-install failure never reaches either strategy.
// Had References instead taken the legacy path, it would fail with quarryengine.ErrServerNotFoundSentinel from a
// literal, unresolved "gopls" lookup on $PATH — a categorically different error this assertion
// distinguishes from.
// This is not a proof that a real gopls connection works end to end — that is
// ensureserver_integration_test.go.
func TestReferences_HasNativeDaemonRoutesThroughEnsureServer(t *testing.T) {
	daemontest.WithTempUserCacheDir(t)

	errFakeInstallRefused := errors.New("fake install refused")
	daemontest.WithFakeInstaller(t, func(ctx context.Context, version, destDir string) error {
		return errFakeInstallRefused
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := References(ctx, Options{
		Registry: registry.Registry{
			"go": {
				Command:         []string{"gopls"},
				PinnedVersion:   "v0.0.0-test",
				HasNativeDaemon: true,
			},
		},
		TargetDir: t.TempDir(),
		Lang:      "go",
		Query:     Query{Symbol: "X"},
		Timeout:   5 * time.Second,
	})
	if !errors.Is(err, errFakeInstallRefused) {
		t.Errorf("References() with HasNativeDaemon: true err = %v; want errors.Is(err, errFakeInstallRefused) (proving the daemon.EnsureServer -> resolveGoToolchain path was taken, not the legacy newLSPClient path)", err)
	}
}

// TestReferences_NonExistentServerBinaryYieldsErrServerNotFound points a synthetic registry entry's
// Command at a binary that cannot exist on $PATH and asserts References maps the resulting
// exec.LookPath failure to quarryengine.ErrServerNotFoundSentinel, mirroring the equivalent //go:build
// integration subtest in refs_integration_test.go but without any dependency on gopls being
// installed.
func TestReferences_NonExistentServerBinaryYieldsErrServerNotFound(t *testing.T) {
	dir := t.TempDir()
	reg := registry.Registry{
		"go": {
			Markers:     []string{"go.mod"},
			Match:       "any",
			Command:     []string{"quarry-nonexistent-binary-xyz"},
			InstallHint: "this binary is intentionally fake for the test",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := References(ctx, Options{
		Registry:  reg,
		TargetDir: dir,
		Lang:      "go",
		Query:     Query{Symbol: "Resolve"},
		Timeout:   5 * time.Second,
	})
	if !errors.Is(err, quarryengine.ErrServerNotFoundSentinel) {
		t.Errorf("References() with a non-existent server binary err = %v; want errors.Is(err, quarryengine.ErrServerNotFoundSentinel)", err)
	}
}
