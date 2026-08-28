// callers_test.go drives callersFromClient (callers.go) against a hand-built client over the
// in-memory pipe transport, mirroring symbol_test.go's fake-server construction. It never drives
// the exported Callers, which would require a spawn.
//
// docs/implementation-widening-spike.md records mode: directional, so every fake server in this
// file that is meant to let verification actually run advertises implementationProvider —
// declarationMatchSet (verify.go) scopes match-set inclusion by directory identity between the
// declaration- and implementation-side URIs, not by SymbolKind, so these fixtures distinguish
// kept from dropped references with distinct directory paths rather than fake documentSymbol
// responses.

package query

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine/daemon/daemontest"
	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
)

// verifiableCapabilities advertises implementationProvider, letting verification run all the way
// through the declaration match-set phase.
var verifiableCapabilities = map[string]any{
	"implementationProvider": true,
}

// locJSON builds the LSP wire JSON for a Location at uri/line/character, one character wide.
func locJSON(uri string, line, character int) map[string]any {
	return map[string]any{
		"uri": uri,
		"range": map[string]any{
			"start": map[string]any{"line": line, "character": character},
			"end":   map[string]any{"line": line, "character": character + 1},
		},
	}
}

// scriptedResponse is one expected request/response pair (or an error, or a deliberate stall) a
// scriptedFakeServer answers in order.
type scriptedResponse struct {
	method string
	result any
	errMsg string
	stall  bool
}

// driveScriptedFakeServer answers "initialize" with caps, consumes the "initialized" notification,
// then answers each subsequent request against script in order, failing the test via t.Errorf on
// any method mismatch. A stall step reads its request and returns without responding, leaving the
// corresponding client call blocked until its context deadline fires — used to drive the
// per-reference verification loop's own deadline. done closes once the script is exhausted (or the
// driver gives up on a mismatch or a stall).
func driveScriptedFakeServer(t *testing.T, server *fakeServer, caps map[string]any, script []scriptedResponse) (done chan struct{}) {
	t.Helper()
	done = make(chan struct{})
	go func() {
		defer close(done)
		initReq, ok := server.readMessage(t)
		if !ok {
			return
		}
		if initReq.Method != "initialize" {
			t.Errorf("fakeServer: got request method %q; want %q", initReq.Method, "initialize")
			return
		}
		if !server.respond(t, initReq.ID, map[string]any{"capabilities": caps}) {
			return
		}
		server.readMessage(t) // initialized notification

		for _, step := range script {
			req, ok := server.readMessage(t)
			if !ok {
				return
			}
			if req.Method != step.method {
				t.Errorf("fakeServer: got request method %q; want %q", req.Method, step.method)
				return
			}
			if step.stall {
				return
			}
			if step.errMsg != "" {
				respondWithError(t, server, req.ID, step.errMsg)
				continue
			}
			server.respond(t, req.ID, step.result)
		}
	}()
	return done
}

// respondWithError writes a JSON-RPC error response for id, mirroring fakeServer.respond's success
// shape but for the error branch fakeServerMessage's own type does not carry.
func respondWithError(t *testing.T, server *fakeServer, id json.RawMessage, message string) {
	t.Helper()
	server.writeMessage(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": -32000, "message": message},
	})
}

// callersFromClientResult carries callersFromClient's return values plus the timedOut flag it set
// through the pointer callers pass in.
type callersFromClientResult struct {
	references  []Reference
	declaration []Reference
	timedOut    bool
	err         error
}

// callCallersFromClient runs callersFromClient in a goroutine behind a 5-second wall-clock guard,
// so a pipeline bug that issues more LSP calls than the fake server's script expects fails the test
// promptly instead of hanging: the unread extra write blocks forever on the underlying io.Pipe,
// which no per-call context deadline can unblock (writeMessage is not ctx-aware).
func callCallersFromClient(t *testing.T, ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position, timeout time.Duration, skipVerification bool) callersFromClientResult {
	t.Helper()
	resultCh := make(chan callersFromClientResult, 1)
	go func() {
		timedOut := false
		refs, decl, err := callersFromClient(ctx, client, fileURI, pos, timeout, &timedOut, skipVerification)
		resultCh <- callersFromClientResult{references: refs, declaration: decl, timedOut: timedOut, err: err}
	}()
	select {
	case r := <-resultCh:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("callersFromClient did not return within 5s — likely issued an unexpected LSP call the fake server's script did not account for")
		return callersFromClientResult{}
	}
}

// newInitializedTestClient builds a client/fake-server pair over the in-memory pipe transport and
// drives script against it, returning the client ready for callersFromClient and a cleanup func the
// caller should defer.
func newInitializedTestClient(t *testing.T, caps map[string]any, script []scriptedResponse) (*lsp.Client, <-chan struct{}) {
	t.Helper()
	clientTransport, serverTransport := newPipeTransportPair()
	t.Cleanup(func() {
		clientTransport.Close()
		serverTransport.Close()
	})

	client := lsp.NewClientFromRW(clientTransport)
	server := newFakeServer(serverTransport)
	done := driveScriptedFakeServer(t, server, caps, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, "file:///tmp/example", nil); err != nil {
		t.Fatalf("initialize() returned unexpected error: %v", err)
	}
	return client, done
}

// TestCallersFromClient_EmptyDeclarationSkipsVerification_EmptyResult is the single most important
// test in this file: an empty declaration-side definition result is a silent success, not an
// error, and nothing intersects an empty set — verifying against one would drop every reference and
// turn the gate green, exactly the fail-open this whole design exists to prevent.
func TestCallersFromClient_EmptyDeclarationSkipsVerification_EmptyResult(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{}},
		{method: "textDocument/references", result: []map[string]any{
			locJSON(fileURI, 5, 2),
			locJSON("file:///tmp/example/caller.go", 20, 4),
		}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.declaration) != 0 {
		t.Errorf("callersFromClient() declaration = %v; want empty (definition returned no locations)", res.declaration)
	}
	if len(res.references) != 2 {
		t.Errorf("callersFromClient() returned %d references; want 2 (every reference kept, an empty declaration set is not verified against)", len(res.references))
	}
}

// TestCallersFromClient_DeclarationDefinitionErrorSkipsVerification asserts that a
// declaration-side textDocument/definition error is not propagated: verification is skipped,
// every reference is kept, and declaration comes back empty.
func TestCallersFromClient_DeclarationDefinitionErrorSkipsVerification(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}

	script := []scriptedResponse{
		{method: "textDocument/definition", errMsg: "gopls: internal error"},
		{method: "textDocument/references", result: []map[string]any{
			locJSON(fileURI, 5, 2),
			locJSON("file:///tmp/example/caller.go", 20, 4),
		}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.declaration) != 0 {
		t.Errorf("callersFromClient() declaration = %v; want empty (definition errored)", res.declaration)
	}
	if len(res.references) != 2 {
		t.Errorf("callersFromClient() returned %d references; want 2 (an errored declaration-side definition keeps every reference)", len(res.references))
	}
}

// TestCallersFromClient_ReferenceResolvingToSamePackageImplementationKept covers the property
// declarationMatchSet exists to preserve: a reference whose own definition resolves to an
// implementation location in the same directory as the declaration is kept, regardless of that
// implementation location's own SymbolKind (documentSymbol is never consulted).
func TestCallersFromClient_ReferenceResolvingToSamePackageImplementationKept(t *testing.T) {
	fileURI := "file:///tmp/example/builder/query.go"
	pos := lsp.Position{Line: 5, Character: 2}
	samePkgURI := "file:///tmp/example/builder/other.go"

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/implementation", result: []map[string]any{locJSON(samePkgURI, 20, 2)}},
		{method: "textDocument/references", result: []map[string]any{locJSON("file:///tmp/example/builder/caller.go", 40, 3)}},
		{method: "textDocument/definition", result: []map[string]any{locJSON(samePkgURI, 20, 2)}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.references) != 1 {
		t.Fatalf("callersFromClient() returned %d references; want 1 (kept: its own definition resolves to a same-directory implementation location)", len(res.references))
	}
}

// TestCallersFromClient_ReferenceResolvingToDifferentPackageImplementationDropped covers the fix
// itself: a reference whose own definition resolves to an implementation location in a different
// directory from the declaration is dropped — the cross-package leak issue #1 describes,
// regardless of that location's own SymbolKind.
func TestCallersFromClient_ReferenceResolvingToDifferentPackageImplementationDropped(t *testing.T) {
	fileURI := "file:///tmp/example/builder/query.go"
	pos := lsp.Position{Line: 5, Character: 2}
	otherPkgURI := "file:///tmp/example/runner/other.go"

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/implementation", result: []map[string]any{locJSON(otherPkgURI, 20, 2)}},
		{method: "textDocument/references", result: []map[string]any{locJSON("file:///tmp/example/runner/caller.go", 40, 3)}},
		{method: "textDocument/definition", result: []map[string]any{locJSON(otherPkgURI, 20, 2)}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.references) != 0 {
		t.Errorf("callersFromClient() returned %d references; want 0 (dropped: its own definition resolves to a different-directory implementation location)", len(res.references))
	}
}

// TestCallersFromClient_ReferenceMatchingNeitherHalfDropped covers the property that removes the
// unrelated, structurally-identical interfaces issue #1 measures: a reference whose own definition
// resolves to neither the definition-side location nor any same-directory implementation location
// is dropped.
func TestCallersFromClient_ReferenceMatchingNeitherHalfDropped(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}
	implURI := "file:///tmp/example/impl.go"
	unrelatedURI := "file:///tmp/example/unrelated.go"

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/implementation", result: []map[string]any{locJSON(implURI, 20, 2)}},
		{method: "textDocument/references", result: []map[string]any{locJSON("file:///tmp/example/caller.go", 40, 3)}},
		{method: "textDocument/definition", result: []map[string]any{locJSON(unrelatedURI, 99, 9)}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.references) != 0 {
		t.Errorf("callersFromClient() returned %d references; want 0 (its own definition resolves to a location in neither match-set half)", len(res.references))
	}
}

// TestCallersFromClient_NoImplementationProviderSkipsVerification asserts a server that does not
// advertise implementationProvider skips verification and keeps every reference — and never issues
// an implementation call at all, proven by the script's absence of one.
func TestCallersFromClient_NoImplementationProviderSkipsVerification(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}

	caps := map[string]any{}
	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/references", result: []map[string]any{
			locJSON(fileURI, 5, 2),
			locJSON("file:///tmp/example/caller.go", 40, 3),
		}},
	}
	client, done := newInitializedTestClient(t, caps, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.references) != 2 {
		t.Errorf("callersFromClient() returned %d references; want 2 (unadvertised implementationProvider skips verification)", len(res.references))
	}
}

// TestCallersFromClient_ImplementationCallErrorSkipsVerification asserts that an errored
// textDocument/implementation call also skips verification and keeps every reference.
func TestCallersFromClient_ImplementationCallErrorSkipsVerification(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/implementation", errMsg: "gopls: internal error"},
		{method: "textDocument/references", result: []map[string]any{
			locJSON(fileURI, 5, 2),
			locJSON("file:///tmp/example/caller.go", 40, 3),
		}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.references) != 2 {
		t.Errorf("callersFromClient() returned %d references; want 2 (an errored implementation call skips verification)", len(res.references))
	}
}

// TestCallersFromClient_SkipVerificationKeepsEveryReferenceNoPerRefCalls asserts that
// skipVerification keeps every reference and issues no per-reference definition calls at all — the
// script only accounts for the declaration-side definition and the references call, so any further
// call would exceed it and callCallersFromClient's wall-clock guard would fail the test.
func TestCallersFromClient_SkipVerificationKeepsEveryReferenceNoPerRefCalls(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/references", result: []map[string]any{
			locJSON(fileURI, 5, 2),
			locJSON("file:///tmp/example/caller.go", 40, 3),
		}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, true)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.references) != 2 {
		t.Errorf("callersFromClient() with SkipVerification returned %d references; want 2", len(res.references))
	}
}

// TestCallersFromClient_DeclarationIsDefinitionOnlyNotUnion asserts the returned declaration value
// is the definition-only result, never the widened match-set union: an implementation result that
// is not in the definition result must not appear in declaration, while the declaration site itself
// must both survive verification and be present in the returned reference set, since
// filterUnexpectedCallers depends on being able to remove it.
func TestCallersFromClient_DeclarationIsDefinitionOnlyNotUnion(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}
	declURI := "file:///tmp/example/query.go"
	implURI := "file:///tmp/example/impl.go"

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(declURI, 5, 2)}},
		{method: "textDocument/implementation", result: []map[string]any{locJSON(implURI, 20, 2)}},
		{method: "textDocument/references", result: []map[string]any{
			// includeDeclaration: true puts the declaration site itself in
			// the references result.
			locJSON(declURI, 5, 2),
		}},
		{method: "textDocument/definition", result: []map[string]any{locJSON(declURI, 5, 2)}},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 5*time.Second, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if len(res.declaration) != 1 || res.declaration[0].File != "/tmp/example/query.go" || res.declaration[0].Line != 6 {
		t.Errorf("callersFromClient() declaration = %+v; want exactly the definition-only result at query.go:6:3", res.declaration)
	}
	for _, d := range res.declaration {
		if d.File == "/tmp/example/impl.go" {
			t.Errorf("callersFromClient() declaration unexpectedly contains the implementation-only location %+v; declaration must never be the widened union", d)
		}
	}
	if len(res.references) != 1 {
		t.Fatalf("callersFromClient() returned %d references; want 1 (the declaration site itself, kept)", len(res.references))
	}
}

// TestCallersFromClient_VerificationPhaseDeadlineKeepsRemainingAndSetsTimedOut drives the
// per-reference verification loop's own deadline with a fake server that stalls on the first
// per-reference definition call and a short timeout: the pipeline must return a successful result
// with every remaining reference kept and the timed-out flag set — asserting both halves, since
// asserting only the return value would let a teardown regression through.
func TestCallersFromClient_VerificationPhaseDeadlineKeepsRemainingAndSetsTimedOut(t *testing.T) {
	fileURI := "file:///tmp/example/query.go"
	pos := lsp.Position{Line: 5, Character: 2}

	script := []scriptedResponse{
		{method: "textDocument/definition", result: []map[string]any{locJSON(fileURI, 5, 2)}},
		{method: "textDocument/implementation", result: []map[string]any{}},
		{method: "textDocument/references", result: []map[string]any{
			locJSON(fileURI, 5, 2),
			locJSON("file:///tmp/example/caller.go", 40, 3),
		}},
		{method: "textDocument/definition", stall: true},
	}
	client, done := newInitializedTestClient(t, verifiableCapabilities, script)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// A short phase timeout so the per-reference loop's own
	// context.WithTimeout deadline fires quickly against the stalled fake
	// server response above, without waiting anywhere near the 5s guard in
	// callCallersFromClient.
	res := callCallersFromClient(t, ctx, client, fileURI, pos, 200*time.Millisecond, false)
	<-done

	if res.err != nil {
		t.Fatalf("callersFromClient() returned unexpected error: %v", res.err)
	}
	if !res.timedOut {
		t.Error("callersFromClient() timedOut = false; want true (the per-reference verification loop's deadline expired)")
	}
	if len(res.references) != 2 {
		t.Errorf("callersFromClient() returned %d references; want 2 (every reference kept: the one never attempted, and the one whose call itself timed out)", len(res.references))
	}
}

// TestTeardownConnection_NativeTimedOutCloses asserts teardownConnection tears the client down for
// daemontest.ConnKindNative when timedOut is true, observable via the client's exported Closed
// accessor.
func TestTeardownConnection_NativeTimedOutCloses(t *testing.T) {
	client := newTeardownTestClient(t)
	teardownConnection(client, daemontest.ConnKindNative, true)
	if !client.Closed() {
		t.Error("Closed() = false after teardownConnection(ConnKindNative, timedOut=true); want true")
	}
}

// TestTeardownConnection_LegacyTimedOutCloses asserts teardownConnection tears the client down for
// daemontest.ConnKindLegacy when timedOut is true, mirroring the native case.
func TestTeardownConnection_LegacyTimedOutCloses(t *testing.T) {
	client := newTeardownTestClient(t)
	teardownConnection(client, daemontest.ConnKindLegacy, true)
	if !client.Closed() {
		t.Error("Closed() = false after teardownConnection(ConnKindLegacy, timedOut=true); want true")
	}
}

// TestTeardownConnection_SupervisedTimedOutLeavesOpen asserts teardownConnection leaves a
// daemontest.ConnKindSupervised connection neither killed nor closed even when timedOut is true —
// the flag is deliberately inert for the supervised strategy, whose daemon outlives this call by
// design.
func TestTeardownConnection_SupervisedTimedOutLeavesOpen(t *testing.T) {
	client := newTeardownTestClient(t)
	teardownConnection(client, daemontest.ConnKindSupervised, true)
	if client.Closed() {
		t.Error("Closed() = true after teardownConnection(ConnKindSupervised, timedOut=true); want false (supervised connections are never killed or closed)")
	}
}

// newTeardownTestClient builds a client over the in-memory pipe transport with no fake server
// driving it — teardownConnection's Kill()/Close() paths need no server response, since Kill only
// closes the transport and Close's shutdown RPC failure is logged, not fatal.
func newTeardownTestClient(t *testing.T) *lsp.Client {
	t.Helper()
	clientTransport, serverTransport := newPipeTransportPair()
	t.Cleanup(func() {
		clientTransport.Close()
		serverTransport.Close()
	})
	return lsp.NewClientFromRW(clientTransport)
}
