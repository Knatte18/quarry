// transport_errors_test.go is transport_test.go's sibling tier-2 test file (card 26): error
// mapping and call isolation, over the same real mcp.Client/NewServer wiring, reusing
// newConnectedPair and newTransportTestConfig from transport_test.go.

package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/quarry"
)

// callToolStructured calls name with args on cs, fails t if the call itself errors or comes back
// with IsError set, and decodes the result's structuredContent into a map[string]any.
func callToolStructured(t *testing.T, cs *mcp.ClientSession, name, args string) map[string]any {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: json.RawMessage(args),
	})
	if err != nil {
		t.Fatalf("CallTool(%q) error = %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool(%q) res.IsError = true; want false", name)
	}
	var out map[string]any
	structuredContentTo(t, res, &out)
	return out
}

// firstResultEntry returns out["results"][0] as a map, failing t if the shape does not hold.
func firstResultEntry(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	results, ok := out["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("out[\"results\"] = %#v; want a non-empty array", out["results"])
	}
	entry, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("out[\"results\"][0] = %#v; want map[string]any", results[0])
	}
	return entry
}

// TestCallTool_MixedBatch_ThreeDistinctStatusesWithGoodResultIntact asserts a mixed call — one
// resolvable target, one not-found, one ambiguous — comes back with the result's error flag unset
// and three distinct per-entry statuses with the good result intact, the regression that matters
// most under array batching.
func TestCallTool_MixedBatch_ThreeDistinctStatusesWithGoodResultIntact(t *testing.T) {
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

	out := callToolStructured(t, cs, "textDocument_definition", `{"targets":[{"symbol":"Found"},{"symbol":"Missing"},{"symbol":"Ambiguous"}]}`)
	results, ok := out["results"].([]any)
	if !ok || len(results) != 3 {
		t.Fatalf("out[\"results\"] = %#v; want a 3-entry array", out["results"])
	}

	wantStatuses := []string{statusFound, statusNotFound, statusAmbiguous}
	for i, want := range wantStatuses {
		entry, ok := results[i].(map[string]any)
		if !ok {
			t.Fatalf("out[\"results\"][%d] = %#v; want map[string]any", i, results[i])
		}
		if got := entry["status"]; got != want {
			t.Errorf("out[\"results\"][%d][\"status\"] = %v; want %q", i, got, want)
		}
	}

	found, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("out[\"results\"][0] = %#v; want map[string]any", results[0])
	}
	defs, ok := found["definitions"].([]any)
	if !ok || len(defs) != 1 {
		t.Errorf("found entry \"definitions\" = %v; want exactly one entry (good result intact)", found["definitions"])
	}
}

// TestCallTool_ResolutionCompletePresenceMatrix asserts "resolution":"complete" is present on found
// entries of textDocument_definition, textDocument_references, and impact, and absent on
// workspace_symbol, assert_no_callers, toc_file, and toc_dir — both halves, since the failure mode
// is adding it everywhere.
func TestCallTool_ResolutionCompletePresenceMatrix(t *testing.T) {
	cfg := newTransportTestConfig(t)
	tocFile := filepath.Join(cfg.TargetDir, "a.go")
	if err := os.WriteFile(tocFile, []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", tocFile, err)
	}
	tocDir := filepath.Join(cfg.TargetDir, "sub")
	if err := os.MkdirAll(tocDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", tocDir, err)
	}

	withStubbedFacade(t, &definitionFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{"Foo": {refs: []quarry.Reference{{File: "a.go", Line: 1, Character: 1}}}}))
	withStubbedFacade(t, &referencesFn, stubLookupFn(t, map[string]struct {
		refs []quarry.Reference
		err  error
	}{"Foo": {refs: []quarry.Reference{{File: "a.go", Line: 1, Character: 1}}}}))
	withStubbedFacade(t, &impactFn, stubImpactFn(t, map[string]struct {
		result quarry.ImpactResult
		err    error
	}{"Foo": {result: quarry.ImpactResult{Callers: []quarry.ImpactCaller{}}}}))
	withStubbedFacade(t, &symbolFn, func(_ context.Context, _ quarry.Options) ([]quarry.SymbolMatch, error) {
		return []quarry.SymbolMatch{{Name: "Foo", File: "a.go", Line: 1, Character: 1}}, nil
	})
	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{"Foo": {}}, nil))

	cs := newConnectedPair(t, cfg)

	tests := []struct {
		tool        string
		args        string
		wantPresent bool
	}{
		{"textDocument_definition", `{"targets":[{"symbol":"Foo"}]}`, true},
		{"textDocument_references", `{"targets":[{"symbol":"Foo"}]}`, true},
		{"impact", `{"targets":[{"symbol":"Foo"}]}`, true},
		{"workspace_symbol", `{"targets":[{"query":"Foo"}]}`, false},
		{"assert_no_callers", `{"targets":[{"symbol":"Foo"}]}`, false},
		{"toc_file", `{"targets":["a.go"]}`, false},
		{"toc_dir", `{"targets":["sub"]}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			out := callToolStructured(t, cs, tt.tool, tt.args)
			entry := firstResultEntry(t, out)
			_, present := entry["resolution"]
			if present != tt.wantPresent {
				t.Errorf("%s found entry has \"resolution\" = %v; want %v", tt.tool, present, tt.wantPresent)
			}
		})
	}
}

// TestCallTool_MalformedServersYAML_LSPToolFailsWhileTOCToolsSucceed asserts the toc whole-call
// split: Config.ConfigPath pointing at a malformed servers.yaml fails an LSP-backed tool's whole
// call, while toc_file and toc_dir — which never load the registry — still succeed.
func TestCallTool_MalformedServersYAML_LSPToolFailsWhileTOCToolsSucceed(t *testing.T) {
	cfg := newTransportTestConfig(t)
	malformed := filepath.Join(t.TempDir(), "servers.yaml")
	if err := os.WriteFile(malformed, []byte("not: valid: yaml: [\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", malformed, err)
	}
	cfg.ConfigPath = malformed

	tocFile := filepath.Join(cfg.TargetDir, "a.go")
	if err := os.WriteFile(tocFile, []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", tocFile, err)
	}
	tocDir := filepath.Join(cfg.TargetDir, "sub")
	if err := os.MkdirAll(tocDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", tocDir, err)
	}

	cs := newConnectedPair(t, cfg)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"Foo"}]}`),
	})
	if err != nil {
		t.Fatalf("CallTool(textDocument_definition) error = %v", err)
	}
	if !res.IsError {
		t.Error("CallTool(textDocument_definition) res.IsError = false; want true (malformed servers.yaml fails the whole call)")
	}

	fileRes, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toc_file",
		Arguments: json.RawMessage(`{"targets":["a.go"]}`),
	})
	if err != nil {
		t.Fatalf("CallTool(toc_file) error = %v", err)
	}
	if fileRes.IsError {
		t.Error("CallTool(toc_file) res.IsError = true; want false (toc never loads the registry)")
	}

	dirRes, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toc_dir",
		Arguments: json.RawMessage(`{"targets":["sub"]}`),
	})
	if err != nil {
		t.Fatalf("CallTool(toc_dir) error = %v", err)
	}
	if dirRes.IsError {
		t.Error("CallTool(toc_dir) res.IsError = true; want false (toc never loads the registry)")
	}
}

// TestCallTool_TOCInvalidLangOrDocSentences_FailsWholeCall asserts an invalid lang or an invalid
// docSentences fails a toc call wholly, at the transport level.
func TestCallTool_TOCInvalidLangOrDocSentences_FailsWholeCall(t *testing.T) {
	cfg := newTransportTestConfig(t)
	tocFile := filepath.Join(cfg.TargetDir, "a.go")
	if err := os.WriteFile(tocFile, []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", tocFile, err)
	}
	cs := newConnectedPair(t, cfg)

	tests := []struct {
		name string
		tool string
		args string
	}{
		{"InvalidLang", "toc_file", `{"targets":["a.go"],"lang":"not-a-real-language"}`},
		{"InvalidDocSentences", "toc_file", `{"targets":["a.go"],"docSentences":"not-a-number"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tt.tool,
				Arguments: json.RawMessage(tt.args),
			})
			if err != nil {
				t.Fatalf("CallTool(%q) error = %v", tt.tool, err)
			}
			if !res.IsError {
				t.Errorf("CallTool(%q, %s) res.IsError = false; want true", tt.tool, tt.args)
			}
		})
	}
}

// TestCallTool_AssertNoCallers_RelativeExceptResolvesAgainstTargetDir asserts an assert_no_callers
// entry with a relative "except" path exempts the intended file at the transport level, the
// regression that silently never matches when the path is resolved against the process working
// directory instead of the call's own target directory.
func TestCallTool_AssertNoCallers_RelativeExceptResolvesAgainstTargetDir(t *testing.T) {
	cfg := newTransportTestConfig(t)
	declFile := filepath.Join(cfg.TargetDir, "decl.go")
	wrapperFile := filepath.Join(cfg.TargetDir, "wrapper.go")

	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{
		"Foo": {
			refs: []quarry.Reference{
				{File: declFile, Line: 1, Character: 1},
				{File: wrapperFile, Line: 2, Character: 2},
			},
			declRefs: []quarry.Reference{{File: declFile, Line: 1, Character: 1}},
		},
	}, nil))
	cs := newConnectedPair(t, cfg)

	out := callToolStructured(t, cs, "assert_no_callers", `{"targets":[{"symbol":"Foo","except":["wrapper.go"]}]}`)
	entry := firstResultEntry(t, out)
	if violation, ok := entry["violation"].(bool); !ok || violation {
		t.Errorf("entry[\"violation\"] = %v; want false (relative except resolved against targetDir must exempt wrapper.go)", entry["violation"])
	}
}

// TestCallTool_ConcurrentCallsBothCompleteIndependently asserts two concurrent tools/call requests
// both complete correctly and neither blocks the other, proving no global mutex serializes them: a
// facade stub for "Slow" blocks until it observes a concurrent "Fast" call has already completed.
func TestCallTool_ConcurrentCallsBothCompleteIndependently(t *testing.T) {
	cfg := newTransportTestConfig(t)
	fastDone := make(chan struct{})

	withStubbedFacade(t, &definitionFn, func(_ context.Context, opts quarry.Options) ([]quarry.Reference, error) {
		if opts.Query.Symbol == "Slow" {
			select {
			case <-fastDone:
			case <-time.After(5 * time.Second):
				t.Error("Slow call timed out waiting for the concurrent Fast call to complete; a global mutex would explain this")
			}
			return []quarry.Reference{{File: "slow.go", Line: 1, Character: 1}}, nil
		}
		return []quarry.Reference{{File: "fast.go", Line: 1, Character: 1}}, nil
	})
	cs := newConnectedPair(t, cfg)

	var wg sync.WaitGroup
	var slowErr, fastErr error
	var slowRes, fastRes *mcp.CallToolResult

	wg.Add(2)
	go func() {
		defer wg.Done()
		slowRes, slowErr = cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "textDocument_definition",
			Arguments: json.RawMessage(`{"targets":[{"symbol":"Slow"}]}`),
		})
	}()
	go func() {
		defer wg.Done()
		fastRes, fastErr = cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "textDocument_definition",
			Arguments: json.RawMessage(`{"targets":[{"symbol":"Fast"}]}`),
		})
		close(fastDone)
	}()
	wg.Wait()

	if fastErr != nil {
		t.Fatalf("Fast CallTool error = %v", fastErr)
	}
	if fastRes.IsError {
		t.Error("Fast CallTool res.IsError = true; want false")
	}
	if slowErr != nil {
		t.Fatalf("Slow CallTool error = %v", slowErr)
	}
	if slowRes.IsError {
		t.Error("Slow CallTool res.IsError = true; want false")
	}
}

// TestCallTool_TargetDirIsAbsoluteEvenFromRelativeProcessCwd asserts the targetDir a handler uses
// is absolute even when the launch default came from a relative process working directory,
// observed by watching the quarry.Options.TargetDir a stub receives.
func TestCallTool_TargetDirIsAbsoluteEvenFromRelativeProcessCwd(t *testing.T) {
	realCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	newCwd := t.TempDir()
	if err := os.Chdir(newCwd); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", newCwd, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(realCwd); err != nil {
			t.Fatalf("os.Chdir(%q) (restore) error = %v", realCwd, err)
		}
	})

	// ResolveLaunchTargetDir("") reads the process's own working directory, exactly as
	// cmd/quarry-mcp/main.go does at startup when --target-dir is omitted — this is the "launch
	// default came from a relative process working directory" scenario the card describes.
	launchTargetDir, err := ResolveLaunchTargetDir("")
	if err != nil {
		t.Fatalf("ResolveLaunchTargetDir(\"\") error = %v", err)
	}

	cfg := Config{TargetDir: launchTargetDir, StateDir: t.TempDir()}

	var gotTargetDir string
	withStubbedFacade(t, &definitionFn, func(_ context.Context, opts quarry.Options) ([]quarry.Reference, error) {
		gotTargetDir = opts.TargetDir
		return []quarry.Reference{{File: "a.go", Line: 1, Character: 1}}, nil
	})
	cs := newConnectedPair(t, cfg)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "textDocument_definition",
		Arguments: json.RawMessage(`{"targets":[{"symbol":"Foo"}]}`),
	})
	if err != nil {
		t.Fatalf("CallTool error = %v", err)
	}
	if res.IsError {
		t.Fatalf("res.IsError = true; want false")
	}
	if !filepath.IsAbs(gotTargetDir) {
		t.Errorf("opts.TargetDir = %q; want an absolute path", gotTargetDir)
	}
}
