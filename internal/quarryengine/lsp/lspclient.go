// Package lsp implements Client, the generalized stdio LSP client speaking exactly the
// request/notification surface this engine needs (initialize, initialized, textDocument/references,
// textDocument/definition, textDocument/documentSymbol, workspace/symbol, shutdown, exit) — not the
// full LSP protocol, per the plan's references-only Shared Decision.
// No callHierarchy, no implementation — the spike's call-hierarchy recommendation (build it on
// TypesInfo.Uses/Defs, never syntactic *ast.CallExpr pattern-matching) does not translate to a
// language-agnostic LSP client at all, since LSP callers must accept whatever
// callHierarchy/incomingCalls a given server implements; that generalization is explicitly deferred
// (see quarryengine's own package doc comment, "What this engine deliberately does not do").
//
// Every request phase — initialize, the workspace/symbol resolver call, and
// textDocument/references or textDocument/definition — is bounded by its own
// context.WithTimeout(ctx, opts.Timeout) deadline (--timeout, default 30s). A phase that times out
// returns quarryengine.ErrServerTimeout (naming the stalled phase) and the caller hard-kills the
// subprocess (cmd.Process.Kill() via the client's Kill()) rather than attempting the graceful
// shutdown/exit handshake, which could itself re-block on a server that is already unresponsive. A
// phase that completes normally instead closes the server via the graceful handshake (Close()) —
// except under the daemon package's supervised EnsureServer strategy, whose connections are never
// closed at all; see daemon/doc.go's own package doc comment for why. This mirrors the spike's own
// timeout-closes-the-hang-failure-mode framing, generalized from "gopls hangs on initialize" to any
// of the request phases on any server.
//
// Position conversion (wire.go's ToPosition) is the one place caller-facing 1-based line/byte-column
// positions (file:line:col as parsed from a CLI argument) cross into LSP's wire format: 0-based
// line, UTF-16 code-unit column. The conversion re-reads the target file because the byte column and
// the UTF-16 offset only coincide on a pure-ASCII line — any multi-byte rune before the target
// column would otherwise misalign the position handed to the server. A workspace/symbol candidate's
// own returned position, by contrast, is already in LSP's wire shape and is used as-is with no round
// trip through the byte-column type, avoiding exactly that misalignment hazard on the resolver path.
//
// lspclient.go implements Client itself. It is ported from the recovered
// tools/codeintel-poc/gopls.go harness (git show 3b4dcf86), generalized to launch any command
// []string rather than a hardcoded "gopls" lookup, and with every request-level call threading a
// context.Context so a caller can bound it with a deadline that hard-kills the subprocess on expiry.
// The I/O is factored over an injectable transport for testability: the production constructor
// NewClient spawns a subprocess and wires its stdio, while the unexported NewClientFromRW seam
// builds a client over a caller-supplied io.ReadWriteCloser with no subprocess at all — the fake
// in-memory server in lspclient_test.go drives this seam.
package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
)

// lspError is the LSP/JSON-RPC error object shape.
type lspError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// lspMessage is the generic JSON-RPC-over-LSP envelope this client reads and sends.
type lspMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *lspError       `json:"error,omitempty"`
}

// SymbolInformation is the LSP wire shape for a workspace/symbol result.
type SymbolInformation struct {
	Name     string   `json:"name"`
	Kind     int      `json:"kind"`
	Location Location `json:"location"`
}

// DocumentSymbol is the LSP wire shape for a textDocument/documentSymbol result.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children"`
}

// capabilities reports the server's workspace/symbol and documentSymbol support.
type capabilities struct {
	WorkspaceSymbolProvider capabilityFlag `json:"workspaceSymbolProvider"`
	DocumentSymbolProvider  capabilityFlag `json:"documentSymbolProvider"`
}

// capabilityFlag normalizes LSP capability fields that may be bool or objects.
type capabilityFlag struct {
	Supported bool
}

// UnmarshalJSON accepts bool or JSON object for LSP capability fields.
func (f *capabilityFlag) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" {
		f.Supported = false
		return nil
	}
	if trimmed == "true" {
		f.Supported = true
		return nil
	}
	if trimmed == "false" {
		f.Supported = false
		return nil
	}
	// Any other well-formed JSON value (an options object) is present, so
	// the capability is advertised.
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal capability flag: %w", err)
	}
	f.Supported = true
	return nil
}

// Client drives a language-server subprocess or caller-supplied transport over Content-Length-framed LSP.
type Client struct {
	cmd      *exec.Cmd
	w        io.Writer
	stdout   *bufio.Reader
	closer   io.Closer
	nextID   int
	closed   bool
	caps     capabilities
	incoming chan lspReadResult
	// Lang is the language identifier this client's server was launched or
	// dialed for (e.g. "go"), set by the daemon package after construction —
	// production constructors here take no language argument, so the
	// caller assigns it once it is in scope — every test-constructed
	// client via NewClientFromRW leaves this at its zero value "". It
	// exists solely so Close()/Kill()'s diagnostic Warn calls can name
	// which language server misbehaved.
	Lang string
}

// lspReadResult is one readLoop iteration's outcome, delivered to whichever
// call() is currently selecting on Client.incoming.
type lspReadResult struct {
	msg *lspMessage
	err error
}

// readLoop is the client's single persistent reader: it calls readMessage
// in a tight loop for as long as the transport stays open, forwarding each
// result to incoming. It is started once per client (by both constructors)
// and runs until readMessage returns an error (transport closed) — at
// which point it sends that final error and exits. The send is
// deliberately not ctx-aware: incoming is shared across every call() the
// client ever makes, so no single call's context should be able to steal
// or drop a message meant for a different, still-pending call. If nothing
// is selecting on incoming when readLoop's final send happens (e.g. every
// caller has already given up), that send — and the goroutine — blocks
// forever; this is a bounded, one-goroutine-per-client leak accepted for
// the client's remaining process lifetime, not an unbounded one.
func (c *Client) readLoop() {
	for {
		msg, err := c.readMessage()
		c.incoming <- lspReadResult{msg: msg, err: err}
		if err != nil {
			return
		}
	}
}

// NewClient resolves command[0] on $PATH and spawns it with command[1:]
// as arguments, wiring its stdin/stdout for LSP framing. NewClient knows
// nothing of which language or install hint command belongs to — that is
// registry.Entry data the caller (refs.go) already has — so a LookPath
// failure is returned as a plain wrapped error (errors.Is(err,
// exec.ErrNotFound) still succeeds); the caller is responsible for
// translating that into the language- and install-hint-carrying
// *ErrServerNotFound. The subprocess's stderr is forwarded to this
// process's stderr so the server's own diagnostic logging is visible
// rather than silently discarded or deadlocking on a full pipe.
func NewClient(command []string) (*Client, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("quarry: empty launch command")
	}

	bin, err := exec.LookPath(command[0])
	if err != nil {
		return nil, fmt.Errorf("quarry: resolve %q on $PATH: %w", command[0], err)
	}

	cmd := exec.Command(bin, command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open %s stdin: %w", bin, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open %s stdout: %w", bin, err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}

	c := &Client{
		cmd:      cmd,
		w:        stdin,
		stdout:   bufio.NewReader(stdout),
		closer:   stdin,
		incoming: make(chan lspReadResult),
	}
	go c.readLoop()
	return c, nil
}

// NewClientFromRW builds an Client over a caller-supplied
// io.ReadWriteCloser transport with no subprocess. This is the seam
// lspclient_test.go's in-memory fake server drives: it lets the framing and
// protocol logic (writeMessage/readMessage, call/notify, the
// server-initiated-request handling, initialize/references/workspaceSymbol)
// be exercised without spawning a real language server.
func NewClientFromRW(rwc io.ReadWriteCloser) *Client {
	c := &Client{
		w:        rwc,
		stdout:   bufio.NewReader(rwc),
		closer:   rwc,
		incoming: make(chan lspReadResult),
	}
	go c.readLoop()
	return c
}

// NewClientDial dials network/address (a Unix socket path or a TCP
// address) and wraps the resulting connection with NewClientFromRW —
// net.Conn already satisfies io.ReadWriteCloser, so the dial-transport mode
// needs no framing, readLoop, or protocol code of its own; it reuses every
// piece of NewClientFromRW's already-tested behavior unchanged. This is
// the supervised-strategy constructor: it dials an already-running,
// externally-owned language server process rather than spawning one, so
// unlike NewClient the returned client's cmd field is nil — close()'s
// `if c.cmd != nil { c.cmd.Wait() }` branch is already a no-op for a dialed
// client, since close() guards on c.cmd == nil correctly. network/address
// are passed through verbatim to net.Dialer.DialContext; this function has
// no opinion on Unix-vs-TCP, the caller decides based on the daemon's
// recorded state-file address.
func NewClientDial(ctx context.Context, network, address string) (*Client, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("quarry: dial lsp server at %s %s: %w", network, address, err)
	}
	return NewClientFromRW(conn), nil
}

// writeMessage marshals v and frames it with the LSP Content-Length header,
// the wire shape every LSP implementation requires regardless of message
// kind (request, response, or notification).
func (c *Client) writeMessage(v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal lsp message: %w", err)
	}
	if _, err := fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return fmt.Errorf("write lsp header: %w", err)
	}
	if _, err := c.w.Write(body); err != nil {
		return fmt.Errorf("write lsp body: %w", err)
	}
	return nil
}

// readMessage reads one Content-Length-framed message from the transport.
// Header lines other than Content-Length (e.g. Content-Type) are accepted
// and ignored, per the LSP base protocol.
func (c *Client) readMessage() (*lspMessage, error) {
	contentLength := -1
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read lsp header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("parse Content-Length header %q: %w", line, err)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("lsp message missing Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, body); err != nil {
		return nil, fmt.Errorf("read lsp body (%d bytes): %w", contentLength, err)
	}

	var msg lspMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal lsp message: %w", err)
	}
	return &msg, nil
}

// Call sends a JSON-RPC request and blocks until either the matching
// response arrives on c.incoming or ctx is done, whichever comes first.
// incoming is fed by the client's single persistent readLoop goroutine
// (started once at construction, not per call) — see Client's doc
// comment for why sharing one reader across every call matters. While
// waiting, call also answers any server-initiated request it encounters in
// the meantime (e.g. client/registerCapability or workspace/configuration)
// with an empty success result — this client implements no client-side LSP
// capability of its own, so an honest empty response is the correct answer
// rather than leaving the server blocked waiting on a reply that will
// never come. Notifications and any other message not addressed to this
// call's id (e.g. a stale message for a call this client already gave up
// on after a previous timeout) are dropped silently and the wait
// continues. phase names the current request for quarryengine.ErrServerTimeout's Phase
// field if ctx expires first.
func (c *Client) Call(ctx context.Context, phase, method string, params any) (json.RawMessage, error) {
	// writeMessage below has no context awareness of its own: on a
	// pipe/subprocess-stdin transport a Write can block until something
	// reads it, so a ctx that is already expired before the write is even
	// attempted must be caught here rather than left to hang. Once the
	// write has actually started, the select loop below is what bounds the
	// remaining wait.
	if err := ctx.Err(); err != nil {
		return nil, &quarryengine.ErrServerTimeout{Phase: phase, Timeout: err.Error()}
	}

	c.nextID++
	id := c.nextID
	idBytes := []byte(strconv.Itoa(id))

	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	if err := c.writeMessage(req); err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, &quarryengine.ErrServerTimeout{Phase: phase, Timeout: ctx.Err().Error()}
		case r := <-c.incoming:
			if r.err != nil {
				return nil, fmt.Errorf("await response to %s: %w", method, r.err)
			}
			msg := r.msg

			if len(msg.ID) > 0 && bytes.Equal(msg.ID, idBytes) {
				if msg.Error != nil {
					return nil, fmt.Errorf("%s: lsp error %d: %s", method, msg.Error.Code, msg.Error.Message)
				}
				return msg.Result, nil
			}

			if msg.Method != "" && len(msg.ID) > 0 {
				if err := c.writeMessage(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(msg.ID), "result": nil}); err != nil {
					return nil, fmt.Errorf("answer server request %s: %w", msg.Method, err)
				}
			}
		}
	}
}

// Notify sends a JSON-RPC notification (a message with no id, expecting no
// response) — the shape "initialized" and "exit" require per the LSP spec.
func (c *Client) Notify(method string, params any) error {
	n := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		n["params"] = params
	}
	if err := c.writeMessage(n); err != nil {
		return fmt.Errorf("send %s notification: %w", method, err)
	}
	return nil
}

// Initialize sends the "initialize" request rooted at rootURI, retains the
// server's reported capabilities (at least workspaceSymbolProvider
// presence, via supportsWorkspaceSymbol), and then sends the "initialized"
// notification per the LSP handshake.
// It advertises textDocument.documentSymbol.hierarchicalDocumentSymbolSupport
// so gopls answers textDocument/documentSymbol with the DocumentSymbol[]
// (range/selectionRange) wire shape DocumentSymbol parses — without it,
// gopls falls back to the flat SymbolInformation[] shape, which unmarshals
// into a zero-valued SelectionRange for every result and silently breaks
// every InFile query.
func (c *Client) Initialize(ctx context.Context, rootURI string) error {
	raw, err := c.Call(ctx, "initialize", "initialize", map[string]any{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"documentSymbol": map[string]any{
					"hierarchicalDocumentSymbolSupport": true,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	var result struct {
		Capabilities capabilities `json:"capabilities"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("unmarshal initialize result: %w", err)
	}
	c.caps = result.Capabilities

	if err := c.Notify("initialized", map[string]any{}); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}
	return nil
}

// SupportsWorkspaceSymbol reports whether the server's initialize response
// advertised workspaceSymbolProvider. It is only meaningful after a
// successful initialize call.
func (c *Client) SupportsWorkspaceSymbol() bool {
	return c.caps.WorkspaceSymbolProvider.Supported
}

// SupportsDocumentSymbol reports whether the server's initialize response
// advertised documentSymbolProvider. It is only meaningful after a
// successful initialize call.
func (c *Client) SupportsDocumentSymbol() bool {
	return c.caps.DocumentSymbolProvider.Supported
}

// References issues one textDocument/references request (with
// includeDeclaration: true, so the declaration site is included alongside
// call sites) and returns the raw location list.
func (c *Client) References(ctx context.Context, fileURI string, pos Position) ([]Location, error) {
	raw, err := c.Call(ctx, "references", "textDocument/references", map[string]any{
		"textDocument": map[string]any{"uri": fileURI},
		"position":     pos,
		"context":      map[string]any{"includeDeclaration": true},
	})
	if err != nil {
		return nil, err
	}

	var locations []Location
	if err := json.Unmarshal(raw, &locations); err != nil {
		return nil, fmt.Errorf("unmarshal textDocument/references result: %w", err)
	}
	return locations, nil
}

// Definition issues one textDocument/definition request and returns the
// server's reported definition location(s), parsed via
// parseDefinitionResult since the LSP spec allows three distinct wire
// shapes for this method's response. Unlike references, no
// context.includeDeclaration parameter is sent — that field is specific to
// textDocument/references's request shape and does not exist on
// textDocument/definition's.
func (c *Client) Definition(ctx context.Context, fileURI string, pos Position) ([]Location, error) {
	raw, err := c.Call(ctx, "definition", "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": fileURI},
		"position":     pos,
	})
	if err != nil {
		return nil, err
	}
	return parseDefinitionResult(raw)
}

// parseDefinitionResult decodes a textDocument/definition response, whose
// LSP-specified result type is `Location | Location[] | LocationLink[] |
// null` — three distinct wire shapes one Go type cannot json.Unmarshal into
// directly. A null or empty result is a legitimate "zero definitions found"
// outcome at this layer (returned as (nil, nil), not an error); the caller
// visible outcome for an empty result is decided higher up, in Definition
// (definition.go), not here. gopls is known to report Location[] for this
// method (not LocationLink[]), so the LocationLink branch below is
// defensive per the LSP spec's stated possible shapes rather than something
// this task can empirically confirm exercises against the one real server
// V1 ships with — lspclient_test.go's table-driven coverage is what
// actually exercises it.
func parseDefinitionResult(raw json.RawMessage) ([]Location, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}

	// A bare Location object (not wrapped in an array) is the single-result
	// shape some servers use; JSON objects start with '{', arrays with '['.
	if trimmed[0] == '{' {
		var loc Location
		if err := json.Unmarshal(trimmed, &loc); err != nil {
			return nil, fmt.Errorf("unmarshal textDocument/definition bare Location result: %w", err)
		}
		return []Location{loc}, nil
	}

	var elements []json.RawMessage
	if err := json.Unmarshal(trimmed, &elements); err != nil {
		return nil, fmt.Errorf("unmarshal textDocument/definition result array: %w", err)
	}

	locations := make([]Location, len(elements))
	for i, elem := range elements {
		// probe carries both possible per-element shapes at once (Location's
		// uri/range and LocationLink's targetUri/targetSelectionRange) so a
		// single unmarshal determines which one the server actually sent.
		var probe struct {
			URI                  string `json:"uri"`
			Range                Range  `json:"range"`
			TargetURI            string `json:"targetUri"`
			TargetSelectionRange Range  `json:"targetSelectionRange"`
		}
		if err := json.Unmarshal(elem, &probe); err != nil {
			return nil, fmt.Errorf("unmarshal textDocument/definition result element %d: %w", i, err)
		}
		if probe.URI != "" {
			locations[i] = Location{URI: probe.URI, Range: probe.Range}
		} else {
			locations[i] = Location{URI: probe.TargetURI, Range: probe.TargetSelectionRange}
		}
	}
	return locations, nil
}

// WorkspaceSymbol issues one workspace/symbol query and returns the
// server's candidate matches, each carrying the symbol's name and
// declaration location.
func (c *Client) WorkspaceSymbol(ctx context.Context, query string) ([]SymbolInformation, error) {
	raw, err := c.Call(ctx, "workspace/symbol", "workspace/symbol", map[string]any{
		"query": query,
	})
	if err != nil {
		return nil, err
	}

	var symbols []SymbolInformation
	if err := json.Unmarshal(raw, &symbols); err != nil {
		return nil, fmt.Errorf("unmarshal workspace/symbol result: %w", err)
	}
	sort.Slice(symbols, func(i, j int) bool {
		return FormatLocation(symbols[i].Location) < FormatLocation(symbols[j].Location)
	})
	return symbols, nil
}

// DocumentSymbols issues one textDocument/documentSymbol request and returns
// the server's hierarchical DocumentSymbol[] result unchanged (children
// still nested under their parent), named in the plural since it returns a
// slice of DocumentSymbol. gopls returns this hierarchical shape,
// not the LSP spec's alternative flat SymbolInformation[] shape; parsing
// only the hierarchical shape is a deliberate scope choice (see
// DocumentSymbol's doc comment), not an oversight of the spec's other
// legal response shape.
func (c *Client) DocumentSymbols(ctx context.Context, fileURI string) ([]DocumentSymbol, error) {
	raw, err := c.Call(ctx, "documentSymbol", "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": fileURI},
	})
	if err != nil {
		return nil, err
	}

	var symbols []DocumentSymbol
	if err := json.Unmarshal(raw, &symbols); err != nil {
		return nil, fmt.Errorf("unmarshal textDocument/documentSymbol result: %w", err)
	}
	return symbols, nil
}

// Closed reports whether Close or Kill has already torn this client down.
func (c *Client) Closed() bool {
	return c.closed
}

// Close runs the graceful LSP shutdown handshake (shutdown request, exit
// notification) and waits for the subprocess to exit. It is best-effort and
// idempotent, for the normal end of a run: a failed shutdown RPC is logged
// rather than returned, since by the time close is called the caller has
// already gathered the result it needs and a clean process exit is a
// nice-to-have, not a correctness requirement. When the client was built
// with no subprocess (NewClientFromRW), close only closes the
// transport.
func (c *Client) Close() {
	if c.closed {
		return
	}
	c.closed = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Call(ctx, "shutdown", "shutdown", nil); err != nil {
		quarryengine.Logger.Warn("quarry: lsp shutdown request", "lang", c.Lang, "err", err)
	}
	if err := c.Notify("exit", nil); err != nil {
		quarryengine.Logger.Warn("quarry: lsp exit notification", "lang", c.Lang, "err", err)
	}
	c.closer.Close()
	if c.cmd != nil {
		if err := c.cmd.Wait(); err != nil {
			quarryengine.Logger.Warn("quarry: lsp process exit", "lang", c.Lang, "err", err)
		}
	}
}

// Kill hard-terminates the subprocess (cmd.Process.Kill(), then Wait to
// reap it) rather than attempting the graceful shutdown/exit handshake,
// which could itself re-block on a server that is already unresponsive —
// this is the timeout-path teardown per the plan's deadline-with-hard-kill
// Shared Decision. It guards on a nil *exec.Cmd: a client built over an
// injected transport (NewClientFromRW) has no subprocess to kill, so
// kill only closes the transport.
func (c *Client) Kill() {
	if c.closed {
		return
	}
	c.closed = true

	c.closer.Close()
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	if err := c.cmd.Process.Kill(); err != nil {
		quarryengine.Logger.Warn("quarry: kill lsp process", "lang", c.Lang, "pid", c.cmd.Process.Pid, "err", err)
	}
	if err := c.cmd.Wait(); err != nil {
		quarryengine.Logger.Warn("quarry: lsp process exit after kill", "lang", c.Lang, "pid", c.cmd.Process.Pid, "err", err)
	}
}
