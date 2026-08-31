// transport_test.go is this package's tier-2 test suite: it wires a real mcp.Client against the
// server NewServer builds, connected over mcp.NewInMemoryTransports(), so every assertion here
// exercises the SDK's own schema validation and JSON serialization path rather than calling a
// handler function directly (the unit-test style every other *_test.go file in this package uses).
// The facade seam variables (internal/mcpserver/facade.go) are still stubbed, exactly as the
// handler-level tests stub them, so no gopls is required. This file covers tool listing, the
// per-tool input-schema parameter matrix, and array-batching's call-wide-versus-per-entry split;
// transport_errors_test.go (card 26) covers error mapping and call isolation, reusing
// newConnectedPair below.

package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/quarry"
)

// wantToolNames is the exact set of seven tools this server registers, in no particular order.
var wantToolNames = []string{
	"textDocument_definition",
	"textDocument_references",
	"workspace_symbol",
	"toc_file",
	"toc_dir",
	"impact",
	"assert_no_callers",
}

// newConnectedPair builds a server from cfg via NewServer, connects it over one half of
// mcp.NewInMemoryTransports(), then connects a client over the other half — the server is always
// connected first, so the client's initialize handshake has something to talk to. Both sessions are
// closed via t.Cleanup.
func newConnectedPair(t *testing.T, cfg Config) *mcp.ClientSession {
	t.Helper()

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer(%+v) error = %v", cfg, err)
	}

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect error = %v", err)
	}
	t.Cleanup(func() { serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "transport-test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect error = %v", err)
	}
	t.Cleanup(func() { clientSession.Close() })

	return clientSession
}

// newTransportTestConfig returns a Config pinned to a fresh t.TempDir() for both TargetDir and
// StateDir, mirroring newTestConfig (tools_lsp_test.go) for the transport-level tests in this file
// and in transport_errors_test.go.
func newTransportTestConfig(t *testing.T) Config {
	t.Helper()
	return Config{TargetDir: t.TempDir(), StateDir: t.TempDir()}
}

// failIfCalledLookupFn returns a definitionFn/referencesFn-shaped stub that fails t immediately if
// invoked, for asserting a malformed call is rejected before any handler runs.
func failIfCalledLookupFn(t *testing.T) func(context.Context, quarry.Options) ([]quarry.Reference, error) {
	t.Helper()
	return func(context.Context, quarry.Options) ([]quarry.Reference, error) {
		t.Fatal("facade fn called; want the malformed call rejected before any handler runs")
		return nil, nil
	}
}

// toolByName returns the *mcp.Tool named name from tools, failing t if absent.
func toolByName(t *testing.T, tools []*mcp.Tool, name string) *mcp.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("no tool named %q in %v", name, toolNames(tools))
	return nil
}

// toolNames extracts each tool's Name, for a failure message.
func toolNames(tools []*mcp.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

// schemaProperties navigates schema (a tool's InputSchema or OutputSchema, an any decoded from the
// wire as a map[string]any) down to its top-level "properties" map.
func schemaProperties(t *testing.T, schema any) map[string]any {
	t.Helper()
	m, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("schema = %#v (%T); want map[string]any", schema, schema)
	}
	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema[\"properties\"] = %#v; want map[string]any", m["properties"])
	}
	return props
}

// entryProperties navigates an input schema's call-wide properties down to the "targets" array
// property's item schema's own "properties" map — the per-entry property set.
func entryProperties(t *testing.T, callWideProps map[string]any) map[string]any {
	t.Helper()
	targets, ok := callWideProps["targets"].(map[string]any)
	if !ok {
		t.Fatalf("properties[\"targets\"] = %#v; want map[string]any", callWideProps["targets"])
	}
	items, ok := targets["items"].(map[string]any)
	if !ok {
		t.Fatalf("targets[\"items\"] = %#v; want map[string]any", targets["items"])
	}
	entryProps, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("items[\"properties\"] = %#v; want map[string]any", items["properties"])
	}
	return entryProps
}

// TestToolsList_ExactlySevenToolsWithBothSchemas asserts tools/list returns exactly the seven
// tools this server registers, each carrying both an input and an output schema.
func TestToolsList_ExactlySevenToolsWithBothSchemas(t *testing.T) {
	cs := newConnectedPair(t, newTransportTestConfig(t))

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error = %v", err)
	}
	if len(res.Tools) != len(wantToolNames) {
		t.Fatalf("len(res.Tools) = %d (%v); want %d (%v)", len(res.Tools), toolNames(res.Tools), len(wantToolNames), wantToolNames)
	}

	for _, name := range wantToolNames {
		tool := toolByName(t, res.Tools, name)
		if tool.InputSchema == nil {
			t.Errorf("tool %q InputSchema is nil; want a schema", name)
		}
		if tool.OutputSchema == nil {
			t.Errorf("tool %q OutputSchema is nil; want a schema", name)
		}
	}
}

// TestToolsList_PerToolParameterMatrix asserts the per-tool input-schema property matrix the plan's
// schema-derivation-and-patching decision and this batch's card describe, as amended by the ladder
// benchmark's findings: workspace_symbol declares exactly "query" and "within" on its entries (the
// within row was flipped after the benchmark showed unscoped workspace/symbol searches saturating
// the server's result cap with out-of-project noise), neither toc tool declares "buildTags",
// toc_dir declares no "docSentences", impact's entry schema declares no "except" while
// assert_no_callers's does, "noVerify" appears on assert_no_callers alone, and no tool declares a
// call-wide "targetDir".
func TestToolsList_PerToolParameterMatrix(t *testing.T) {
	cs := newConnectedPair(t, newTransportTestConfig(t))

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error = %v", err)
	}

	symbolEntryProps := entryProperties(t, schemaProperties(t, toolByName(t, res.Tools, "workspace_symbol").InputSchema))
	for _, want := range []string{"query", "within"} {
		if _, ok := symbolEntryProps[want]; !ok {
			t.Errorf("workspace_symbol entry schema does not declare %q; want it present", want)
		}
	}
	if len(symbolEntryProps) != 2 {
		t.Errorf("workspace_symbol entry schema declares %d properties; want exactly 2 (query, within)", len(symbolEntryProps))
	}

	for _, name := range []string{"toc_file", "toc_dir"} {
		props := schemaProperties(t, toolByName(t, res.Tools, name).InputSchema)
		if _, ok := props["buildTags"]; ok {
			t.Errorf("%s call-wide schema declares \"buildTags\"; want it absent", name)
		}
	}

	tocDirProps := schemaProperties(t, toolByName(t, res.Tools, "toc_dir").InputSchema)
	if _, ok := tocDirProps["docSentences"]; ok {
		t.Error("toc_dir call-wide schema declares \"docSentences\"; want it absent")
	}

	impactEntryProps := entryProperties(t, schemaProperties(t, toolByName(t, res.Tools, "impact").InputSchema))
	if _, ok := impactEntryProps["except"]; ok {
		t.Error("impact entry schema declares \"except\"; want it absent")
	}

	assertEntryProps := entryProperties(t, schemaProperties(t, toolByName(t, res.Tools, "assert_no_callers").InputSchema))
	if _, ok := assertEntryProps["except"]; !ok {
		t.Error("assert_no_callers entry schema does not declare \"except\"; want it present")
	}

	for _, name := range wantToolNames {
		props := schemaProperties(t, toolByName(t, res.Tools, name).InputSchema)
		_, hasNoVerify := props["noVerify"]
		wantNoVerify := name == "assert_no_callers"
		if hasNoVerify != wantNoVerify {
			t.Errorf("%s call-wide schema has \"noVerify\" = %v; want %v (assert_no_callers alone)", name, hasNoVerify, wantNoVerify)
		}
	}

	for _, name := range wantToolNames {
		props := schemaProperties(t, toolByName(t, res.Tools, name).InputSchema)
		if _, ok := props["targetDir"]; ok {
			t.Errorf("%s call-wide schema declares \"targetDir\"; want it absent", name)
		}
	}
}

// TestCallTool_TargetDirIsRejectedAsWholeCallError asserts that a call still sending "targetDir"
// fails as a whole-call error per the hard-removal-is-the-error-behaviour Shared Decision: the SDK's
// own call-wide additionalProperties: false rejects it before any handler runs, so the stubbed
// facade fn is never invoked and no per-entry "status": "error" result array comes back. This test
// does not assert the SDK validator's exact error string — only that the call failed as a whole.
func TestCallTool_TargetDirIsRejectedAsWholeCallError(t *testing.T) {
	cfg := newTransportTestConfig(t)
	withStubbedFacade(t, &definitionFn, failIfCalledLookupFn(t))
	cs := newConnectedPair(t, cfg)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"S"}],"targetDir":"/somewhere/else"}`),
	})
	if err != nil {
		t.Fatalf("CallTool error = %v; want a result with IsError set, not a protocol-level error", err)
	}
	if !res.IsError {
		t.Fatal("res.IsError = false; want true for a call carrying the removed \"targetDir\" property")
	}

	structuredData, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("json.Marshal(res.StructuredContent) error = %v", err)
	}
	var out definitionOutput
	if err := json.Unmarshal(structuredData, &out); err == nil && len(out.Results) > 0 {
		t.Errorf("out.Results = %v (from structuredContent = %s); want no decodable results array for a whole-call rejection", out.Results, structuredData)
	}
}

// TestCallTool_MalformedCall_RejectedBeforeHandlerRuns asserts a zero-length targets array, a
// 65-entry targets array, and a targets value of the wrong JSON type are each rejected before any
// handler runs: the result's error flag is set and the stubbed facade fn — which fails t if
// invoked — is never called. This asserts only the observable contract (rejected whole, error flag
// set, handler never run), never the SDK validator's message wording.
func TestCallTool_MalformedCall_RejectedBeforeHandlerRuns(t *testing.T) {
	cfg := newTransportTestConfig(t)

	sixtyFive := make([]string, 0, 65)
	for i := 0; i < 65; i++ {
		sixtyFive = append(sixtyFive, `{"symbol":"S"}`)
	}
	overCapTargets := "[" + joinJSON(sixtyFive) + "]"

	tests := []struct {
		name string
		args string
	}{
		{"ZeroLengthTargets", `{"targets":[]}`},
		{"SixtyFiveEntryTargets", `{"targets":` + overCapTargets + `}`},
		{"WrongTypeTargets", `{"targets":"not-an-array"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStubbedFacade(t, &definitionFn, failIfCalledLookupFn(t))
			cs := newConnectedPair(t, cfg)

			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "textDocument_definition",
				Arguments: json.RawMessage(tt.args),
			})
			if err != nil {
				t.Fatalf("CallTool error = %v; want a result with IsError set, not a protocol-level error", err)
			}
			if !res.IsError {
				t.Errorf("res.IsError = false; want true for a malformed call (%s)", tt.args)
			}
		})
	}
}

// joinJSON joins pre-formatted JSON object literals with commas, for building a targets array
// literal by hand.
func joinJSON(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ","
		}
		out += item
	}
	return out
}

// TestCallTool_MalformedEntryVsMalformedCall asserts the two failure modes this batch's card warns
// against conflating: a malformed entry (an unrecognized property) yields only that entry's own
// status: "error" with every sibling entry's result intact, while a malformed call (asserted above)
// is rejected whole. Both are asserted in this one test so a regression collapsing the two is
// caught here.
func TestCallTool_MalformedEntryVsMalformedCall(t *testing.T) {
	cfg := newTransportTestConfig(t)
	withStubbedFacade(t, &definitionFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Before": {refs: []quarry.Reference{{File: "a.go", Line: 1, Character: 1}}},
		"After":  {refs: []quarry.Reference{{File: "b.go", Line: 2, Character: 2}}},
	}))
	cs := newConnectedPair(t, cfg)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"Before"},{"symbol":"Bad","wrongProperty":"x"},{"symbol":"After"}]}`),
	})
	if err != nil {
		t.Fatalf("CallTool error = %v", err)
	}
	if res.IsError {
		t.Fatalf("res.IsError = true; want false (a per-entry violation must never fail the whole call)")
	}

	var out definitionOutput
	structuredContentTo(t, res, &out)
	if len(out.Results) != 3 {
		t.Fatalf("len(out.Results) = %d; want 3", len(out.Results))
	}
	if out.Results[0].Status != statusFound {
		t.Errorf("out.Results[0].Status = %q; want %q (sibling of the bad entry)", out.Results[0].Status, statusFound)
	}
	if out.Results[1].Status != statusError {
		t.Errorf("out.Results[1].Status = %q; want %q (the malformed entry)", out.Results[1].Status, statusError)
	}
	if out.Results[2].Status != statusFound {
		t.Errorf("out.Results[2].Status = %q; want %q (sibling of the bad entry)", out.Results[2].Status, statusFound)
	}
}

// structuredContentTo re-marshals res.StructuredContent (an any decoded from the wire) into out via
// a JSON round trip, since the client only ever sees the untyped decoded form.
func structuredContentTo(t *testing.T, res *mcp.CallToolResult, out any) {
	t.Helper()
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("json.Marshal(res.StructuredContent) error = %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("json.Unmarshal(structuredContent) error = %v", err)
	}
}

// TestCallTool_MultiEntryCall_OrderTargetStatusAndSchemaAndContentParity asserts the cross-cutting
// contract multi-entry array batching promises: one result entry per input in input order, every
// entry carries "target" and "status", the mixed batch validates against the tool's declared output
// schema without any entry being rejected (the SDK would fail the whole call with res.IsError if
// output validation failed), and structuredContent and the text content block carry the same
// payload.
func TestCallTool_MultiEntryCall_OrderTargetStatusAndSchemaAndContentParity(t *testing.T) {
	cfg := newTransportTestConfig(t)
	ambiguous := &quarry.ErrAmbiguousSymbol{Symbol: "Ambiguous", Candidates: []string{"a.go:1:1"}}
	withStubbedFacade(t, &definitionFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Found":     {refs: []quarry.Reference{{File: "a.go", Line: 5, Character: 10}}},
		"Missing":   {err: quarry.ErrSymbolNotFoundSentinel},
		"Ambiguous": {err: ambiguous},
	}))
	cs := newConnectedPair(t, cfg)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"Found"},{"symbol":"Missing"},{"symbol":"Ambiguous"}]}`),
	})
	if err != nil {
		t.Fatalf("CallTool error = %v", err)
	}
	if res.IsError {
		t.Fatalf("res.IsError = true; want false — a mixed batch of found/not_found/ambiguous entries must validate against the output schema and never fail the whole call")
	}

	var out definitionOutput
	structuredContentTo(t, res, &out)
	if len(out.Results) != 3 {
		t.Fatalf("len(out.Results) = %d; want 3", len(out.Results))
	}
	wantSymbols := []string{"Found", "Missing", "Ambiguous"}
	wantStatuses := []string{statusFound, statusNotFound, statusAmbiguous}
	for i := range out.Results {
		target, ok := out.Results[i].Target.(map[string]any)
		if !ok {
			t.Fatalf("out.Results[%d].Target = %#v; want map[string]any", i, out.Results[i].Target)
		}
		if target["symbol"] != wantSymbols[i] {
			t.Errorf("out.Results[%d].Target[\"symbol\"] = %v; want %q (input order)", i, target["symbol"], wantSymbols[i])
		}
		if out.Results[i].Status != wantStatuses[i] {
			t.Errorf("out.Results[%d].Status = %q; want %q", i, out.Results[i].Status, wantStatuses[i])
		}
	}

	if len(res.Content) != 1 {
		t.Fatalf("len(res.Content) = %d; want 1", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("res.Content[0] = %#v (%T); want *mcp.TextContent", res.Content[0], res.Content[0])
	}

	var fromText, fromStructured any
	if err := json.Unmarshal([]byte(text.Text), &fromText); err != nil {
		t.Fatalf("json.Unmarshal(text content) error = %v", err)
	}
	structuredData, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("json.Marshal(res.StructuredContent) error = %v", err)
	}
	if err := json.Unmarshal(structuredData, &fromStructured); err != nil {
		t.Fatalf("json.Unmarshal(structuredContent) error = %v", err)
	}

	fromTextJSON, _ := json.Marshal(fromText)
	fromStructuredJSON, _ := json.Marshal(fromStructured)
	if string(fromTextJSON) != string(fromStructuredJSON) {
		t.Errorf("text content payload = %s; want the same payload as structuredContent = %s", fromTextJSON, fromStructuredJSON)
	}
}
