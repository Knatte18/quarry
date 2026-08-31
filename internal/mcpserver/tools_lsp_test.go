package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// stubDefinitionFn returns a definitionFn/referencesFn-shaped stub keyed on the query's own identity
// (symbol name, in-file name, or position file:line:character), so a single stub can drive a
// multi-entry call whose entries resolve to different outcomes.
func stubLookupFn(t *testing.T, responses map[string]struct {
	refs []quarry.Reference
	err  error
}) func(context.Context, quarry.Options) ([]quarry.Reference, error) {
	t.Helper()
	return func(_ context.Context, opts quarry.Options) ([]quarry.Reference, error) {
		key := queryKey(opts.Query)
		r, ok := responses[key]
		if !ok {
			t.Fatalf("stub lookup fn: no response configured for query key %q (query %+v)", key, opts.Query)
		}
		return r.refs, r.err
	}
}

// queryKey derives a stable lookup key from a quarry.Query, matching however the test itself built
// the query: the symbol name for a bare-symbol query, "file:line:character" for a position query, or
// "file/name" for an in-file query.
func queryKey(q quarry.Query) string {
	switch {
	case q.Pos != nil:
		return fmt.Sprintf("%s:%d:%d", q.Pos.File, q.Pos.Line, q.Pos.Character)
	case q.InFile != nil:
		return q.InFile.File + "/" + q.InFile.Name
	default:
		return q.Symbol
	}
}

// withStubbedFacade replaces the package-level facade seam variable target points at with stub for
// the duration of t, restoring the original value via t.Cleanup.
func withStubbedFacade[F any](t *testing.T, target *F, stub F) {
	t.Helper()
	original := *target
	*target = stub
	t.Cleanup(func() { *target = original })
}

// newTestConfig returns a Config pinned to fresh t.TempDir()s for both TargetDir and StateDir, so no
// machine-global cache directory is touched by a test that drives a handler through resolveCall.
func newTestConfig(t *testing.T) Config {
	t.Helper()
	return Config{TargetDir: t.TempDir(), StateDir: t.TempDir()}
}

func mustUnmarshal[T any](t *testing.T, data string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", data, err)
	}
	return v
}

func TestDefinitionHandler_ThreeEntryMixedCall(t *testing.T) {
	cfg := newTestConfig(t)
	ambiguous := &quarry.ErrAmbiguousSymbol{Symbol: "Ambiguous", Candidates: []string{"a.go:1:1", "b.go:2:2"}}
	withStubbedFacade(t, &definitionFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Found":     {refs: []quarry.Reference{{File: "a.go", Line: 5, Character: 10}}},
		"Missing":   {err: quarry.ErrSymbolNotFoundSentinel},
		"Ambiguous": {err: ambiguous},
	}))

	in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Found"},{"symbol":"Missing"},{"symbol":"Ambiguous"}]}`)

	_, out, err := definitionHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("definitionHandler(cfg)(...) error = %v; want nil (per-entry failures never become a whole-call error)", err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("len(out.Results) = %d; want 3", len(out.Results))
	}

	wantStatuses := []string{statusFound, statusNotFound, statusAmbiguous}
	for i, want := range wantStatuses {
		if got := out.Results[i].Status; got != want {
			t.Errorf("out.Results[%d].Status = %q; want %q", i, got, want)
		}
	}

	found := out.Results[0]
	if found.Resolution != resolutionComplete {
		t.Errorf("found entry Resolution = %q; want %q", found.Resolution, resolutionComplete)
	}
	wantDef := referenceField{File: "a.go", Line: 4, Character: 9}
	if len(found.Definitions) != 1 || found.Definitions[0] != wantDef {
		t.Errorf("found entry Definitions = %v; want [%v] (0-based on both axes)", found.Definitions, wantDef)
	}

	ambiguousEntry := out.Results[2]
	if len(ambiguousEntry.Candidates) != 2 {
		t.Errorf("ambiguous entry Candidates = %v; want 2 candidates", ambiguousEntry.Candidates)
	}

	for i, entry := range out.Results {
		wantTarget := mustUnmarshal[map[string]any](t, []string{`{"symbol":"Found"}`, `{"symbol":"Missing"}`, `{"symbol":"Ambiguous"}`}[i])
		gotTarget, ok := entry.Target.(map[string]any)
		if !ok {
			t.Fatalf("out.Results[%d].Target = %#v (%T); want map[string]any", i, entry.Target, entry.Target)
		}
		if gotTarget["symbol"] != wantTarget["symbol"] {
			t.Errorf("out.Results[%d].Target = %v; want %v", i, gotTarget, wantTarget)
		}
	}
}

func TestReferencesHandler_FoundEntryCarriesResolutionComplete(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &referencesFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Foo": {refs: []quarry.Reference{{File: "a.go", Line: 1, Character: 1}}},
	}))

	in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := referencesHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("referencesHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	if out.Results[0].Resolution != resolutionComplete {
		t.Errorf("out.Results[0].Resolution = %q; want %q", out.Results[0].Resolution, resolutionComplete)
	}
}

func TestSymbolHandler_FoundEntryCarriesNoResolutionAndNoCandidates(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &symbolFn, func(_ context.Context, opts quarry.Options) ([]quarry.SymbolMatch, error) {
		if opts.Query.Symbol != "Foo" {
			t.Fatalf("symbolFn called with Query.Symbol = %q; want \"Foo\"", opts.Query.Symbol)
		}
		return []quarry.SymbolMatch{{Name: "Foo", Kind: 12, File: "a.go", Line: 3, Character: 4}}, nil
	})

	in := mustUnmarshal[symbolInput](t, `{"targets":[{"query":"Foo"}]}`)

	_, out, err := symbolHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("symbolHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	got := out.Results[0]
	if got.Status != statusFound {
		t.Fatalf("out.Results[0].Status = %q; want %q", got.Status, statusFound)
	}
	wantSym := symbolField{Name: "Foo", Kind: 12, File: "a.go", Line: 2, Character: 3}
	if len(got.Symbols) != 1 || got.Symbols[0] != wantSym {
		t.Errorf("out.Results[0].Symbols = %v; want [%v] (0-based on both axes)", got.Symbols, wantSym)
	}
	// symbolMatchEntry has no Resolution or Candidates field at all — the compiler itself enforces
	// this; asserting JSON output carries neither key is the runtime half of that guarantee.
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal(got) error = %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(data, &asMap); err != nil {
		t.Fatalf("json.Unmarshal(data) error = %v", err)
	}
	if _, ok := asMap["resolution"]; ok {
		t.Error("marshaled workspace_symbol found entry carries \"resolution\"; want it absent")
	}
	if _, ok := asMap["candidates"]; ok {
		t.Error("marshaled workspace_symbol found entry carries \"candidates\"; want it absent")
	}
}

// TestDefinitionHandler_PositionEntryWithNoTextDocument_IsThatEntrysErrorOnly asserts a position
// entry missing textDocument is that entry's own status: "error" while its siblings still return
// their results.
func TestDefinitionHandler_PositionEntryWithNoTextDocument_IsThatEntrysErrorOnly(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &definitionFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Before": {refs: []quarry.Reference{{File: "a.go", Line: 1, Character: 1}}},
		"After":  {refs: []quarry.Reference{{File: "b.go", Line: 2, Character: 2}}},
	}))

	in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Before"},{"position":{"line":0,"character":0}},{"symbol":"After"}]}`)

	_, out, err := definitionHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("definitionHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("len(out.Results) = %d; want 3", len(out.Results))
	}
	if out.Results[0].Status != statusFound {
		t.Errorf("out.Results[0].Status = %q; want %q (sibling of the bad entry)", out.Results[0].Status, statusFound)
	}
	if out.Results[1].Status != statusError {
		t.Errorf("out.Results[1].Status = %q; want %q (position with no textDocument)", out.Results[1].Status, statusError)
	}
	if out.Results[2].Status != statusFound {
		t.Errorf("out.Results[2].Status = %q; want %q (sibling of the bad entry)", out.Results[2].Status, statusFound)
	}
}

// TestSymbolHandler_TextDocumentOrPositionKey_IsThatEntrysError asserts a workspace_symbol entry
// carrying "textDocument" or "position" is that entry's own status: "error", and that the facade
// stub is never invoked with an empty-string search for either.
func TestSymbolHandler_TextDocumentOrPositionKey_IsThatEntrysError(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &symbolFn, func(_ context.Context, opts quarry.Options) ([]quarry.SymbolMatch, error) {
		if opts.Query.Symbol == "" {
			t.Fatal("symbolFn called with an empty-string search; want it never called for an illegal entry")
		}
		return []quarry.SymbolMatch{{Name: opts.Query.Symbol, File: "a.go", Line: 1, Character: 1}}, nil
	})

	in := mustUnmarshal[symbolInput](t, `{"targets":[
		{"query":"Good"},
		{"query":"Bad1","textDocument":{"uri":"x.go"}},
		{"query":"Bad2","position":{"line":0,"character":0}}
	]}`)

	_, out, err := symbolHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("symbolHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("len(out.Results) = %d; want 3", len(out.Results))
	}
	if out.Results[0].Status != statusFound {
		t.Errorf("out.Results[0].Status = %q; want %q", out.Results[0].Status, statusFound)
	}
	if out.Results[1].Status != statusError {
		t.Errorf("out.Results[1].Status = %q; want %q (entry carries textDocument)", out.Results[1].Status, statusError)
	}
	if out.Results[2].Status != statusError {
		t.Errorf("out.Results[2].Status = %q; want %q (entry carries position)", out.Results[2].Status, statusError)
	}
}

// TestReferencesHandler_PerEntryWithinFiltersOnlyThatEntry asserts a per-entry "within" filters only
// that entry's own references, leaving a sibling entry with no "within" unaffected.
func TestReferencesHandler_PerEntryWithinFiltersOnlyThatEntry(t *testing.T) {
	cfg := newTestConfig(t)
	inFile := filepath.Join(cfg.TargetDir, "in", "a.go")
	outFile := filepath.Join(cfg.TargetDir, "out", "b.go")

	withStubbedFacade(t, &referencesFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Scoped":   {refs: []quarry.Reference{{File: inFile, Line: 1, Character: 1}, {File: outFile, Line: 2, Character: 2}}},
		"Unscoped": {refs: []quarry.Reference{{File: inFile, Line: 1, Character: 1}, {File: outFile, Line: 2, Character: 2}}},
	}))

	in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Scoped","within":"in"},{"symbol":"Unscoped"}]}`)

	_, out, err := referencesHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("referencesHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("len(out.Results) = %d; want 2", len(out.Results))
	}

	scoped := out.Results[0]
	if len(scoped.References) != 1 {
		t.Fatalf("scoped entry References = %v; want exactly the one reference within \"in\"", scoped.References)
	}

	unscoped := out.Results[1]
	if len(unscoped.References) != 2 {
		t.Errorf("unscoped entry References = %v; want both references (no \"within\" filter applied)", unscoped.References)
	}
}

// TestFoundEntryWithZeroResults_StillMarshalsEmptyResultsKey asserts a statusFound entry whose
// facade call returned no locations still emits its results key as "[]" on the wire. The language
// server answers an empty location list (not an error) for a position that names no identifier, so
// without the key an agent cannot tell "found, nothing there" apart from a malformed result — the
// exact shape observed in the ladder benchmark's b2-definition/3 run before this was fixed.
// omitempty would drop the empty-but-non-nil slice referenceFieldsWire guarantees; the omitzero tag
// is what keeps it.
func TestFoundEntryWithZeroResults_StillMarshalsEmptyResultsKey(t *testing.T) {
	cfg := newTestConfig(t)

	empty := map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Nothing": {refs: nil, err: nil},
	}

	tests := []struct {
		name       string
		resultsKey string
		marshal    func(t *testing.T) []byte
	}{
		{
			name:       "Definition",
			resultsKey: "definitions",
			marshal: func(t *testing.T) []byte {
				t.Helper()
				withStubbedFacade(t, &definitionFn, stubLookupFn(t, empty))
				in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Nothing"}]}`)
				_, out, err := definitionHandler(cfg)(context.Background(), nil, in)
				if err != nil {
					t.Fatalf("definitionHandler(cfg)(...) error = %v", err)
				}
				data, err := json.Marshal(out.Results[0])
				if err != nil {
					t.Fatalf("json.Marshal(out.Results[0]) error = %v", err)
				}
				return data
			},
		},
		{
			name:       "References",
			resultsKey: "references",
			marshal: func(t *testing.T) []byte {
				t.Helper()
				withStubbedFacade(t, &referencesFn, stubLookupFn(t, empty))
				in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Nothing"}]}`)
				_, out, err := referencesHandler(cfg)(context.Background(), nil, in)
				if err != nil {
					t.Fatalf("referencesHandler(cfg)(...) error = %v", err)
				}
				data, err := json.Marshal(out.Results[0])
				if err != nil {
					t.Fatalf("json.Marshal(out.Results[0]) error = %v", err)
				}
				return data
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.marshal(t)
			var asMap map[string]any
			if err := json.Unmarshal(data, &asMap); err != nil {
				t.Fatalf("json.Unmarshal(%s) error = %v", data, err)
			}
			if asMap["status"] != statusFound {
				t.Fatalf("marshaled entry status = %v; want %q", asMap["status"], statusFound)
			}
			results, ok := asMap[tt.resultsKey].([]any)
			if !ok {
				t.Fatalf("marshaled entry %q = %v; want a present, empty JSON array", tt.resultsKey, asMap[tt.resultsKey])
			}
			if len(results) != 0 {
				t.Errorf("marshaled entry %q = %v; want empty", tt.resultsKey, results)
			}
		})
	}
}

// TestDefinitionHandler_SpawnTimeoutIsPerEntryErrorNotWholeCallFailure asserts a stub returning
// quarry.ErrServerSpawnTimeoutSentinel produces a per-entry status: "error" rather than a whole-call
// failure, since a daemon spawn failure surfaces only from a per-entry facade call.
func TestDefinitionHandler_SpawnTimeoutIsPerEntryErrorNotWholeCallFailure(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &definitionFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{
		"Foo": {err: quarry.ErrServerSpawnTimeoutSentinel},
	}))

	in := mustUnmarshal[lspInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := definitionHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("definitionHandler(cfg)(...) error = %v; want nil (a daemon spawn failure is a per-entry outcome, not a whole-call failure)", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	if out.Results[0].Status != statusError {
		t.Errorf("out.Results[0].Status = %q; want %q", out.Results[0].Status, statusError)
	}
}
