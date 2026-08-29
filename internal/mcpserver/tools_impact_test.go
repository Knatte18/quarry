package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// stubImpactFn returns an impactFn-shaped stub keyed on the query's own identity (symbol name,
// in-file name, or position file:line:character), following stubLookupFn's own convention in
// tools_lsp_test.go.
func stubImpactFn(t *testing.T, responses map[string]struct {
	result quarry.ImpactResult
	err    error
}) func(context.Context, quarry.Options) (quarry.ImpactResult, error) {
	t.Helper()
	return func(_ context.Context, opts quarry.Options) (quarry.ImpactResult, error) {
		key := queryKey(opts.Query)
		r, ok := responses[key]
		if !ok {
			t.Fatalf("stub impact fn: no response configured for query key %q (query %+v)", key, opts.Query)
		}
		return r.result, r.err
	}
}

// TestImpactHandler_EnvelopeTargetIsEchoedInputAndResultTargetIsNested asserts a found entry's
// envelope "target" is the echoed input while the marshalled result's own "target" is reachable
// under "result", so neither overwrites the other.
func TestImpactHandler_EnvelopeTargetIsEchoedInputAndResultTargetIsNested(t *testing.T) {
	cfg := newTestConfig(t)
	name := "Foo"
	withStubbedFacade(t, &impactFn, stubImpactFn(t, map[string]struct {
		result quarry.ImpactResult
		err    error
	}{
		"Foo": {result: quarry.ImpactResult{
			Target:  &quarry.ImpactTarget{Name: name},
			Callers: []quarry.ImpactCaller{},
		}},
	}))

	in := mustUnmarshal[impactInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := impactHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("impactHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}

	entry := out.Results[0]
	gotTarget, ok := entry.Target.(map[string]any)
	if !ok {
		t.Fatalf("entry.Target = %#v (%T); want map[string]any", entry.Target, entry.Target)
	}
	if gotTarget["symbol"] != "Foo" {
		t.Errorf("entry.Target = %v; want the echoed input {\"symbol\":\"Foo\"}", gotTarget)
	}

	resultTarget, ok := entry.Result["target"].(map[string]any)
	if !ok {
		t.Fatalf("entry.Result[\"target\"] = %#v; want map[string]any", entry.Result["target"])
	}
	if resultTarget["name"] != name {
		t.Errorf("entry.Result[\"target\"][\"name\"] = %v; want %q (nested under \"result\", not overwriting the envelope's own \"target\")", resultTarget["name"], name)
	}
}

// TestImpactHandler_FoundEntryCarriesResolutionComplete asserts a found entry carries
// "resolution":"complete".
func TestImpactHandler_FoundEntryCarriesResolutionComplete(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &impactFn, stubImpactFn(t, map[string]struct {
		result quarry.ImpactResult
		err    error
	}{
		"Foo": {result: quarry.ImpactResult{Callers: []quarry.ImpactCaller{}}},
	}))

	in := mustUnmarshal[impactInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := impactHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("impactHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	if got := out.Results[0].Resolution; got != resolutionComplete {
		t.Errorf("out.Results[0].Resolution = %q; want %q", got, resolutionComplete)
	}
}

// TestImpactHandler_PerEntryWithinFiltersOnlyThatEntry asserts a per-entry "within" filters that
// entry's own callers while leaving another entry's untouched.
func TestImpactHandler_PerEntryWithinFiltersOnlyThatEntry(t *testing.T) {
	cfg := newTestConfig(t)
	inFile := filepath.Join(cfg.TargetDir, "in", "a.go")
	outFile := filepath.Join(cfg.TargetDir, "out", "b.go")

	withStubbedFacade(t, &impactFn, stubImpactFn(t, map[string]struct {
		result quarry.ImpactResult
		err    error
	}{
		"Scoped": {result: quarry.ImpactResult{
			Callers: []quarry.ImpactCaller{{File: inFile, CallSiteLine: 1}, {File: outFile, CallSiteLine: 2}},
		}},
		"Unscoped": {result: quarry.ImpactResult{
			Callers: []quarry.ImpactCaller{{File: inFile, CallSiteLine: 1}, {File: outFile, CallSiteLine: 2}},
		}},
	}))

	in := mustUnmarshal[impactInput](t, `{"targets":[{"symbol":"Scoped","within":"in"},{"symbol":"Unscoped"}]}`)

	_, out, err := impactHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("impactHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("len(out.Results) = %d; want 2", len(out.Results))
	}

	scopedCallers, _ := out.Results[0].Result["callers"].([]any)
	if len(scopedCallers) != 1 {
		t.Errorf("scoped entry Result[\"callers\"] = %v; want exactly the one caller within \"in\"", scopedCallers)
	}

	unscopedCallers, _ := out.Results[1].Result["callers"].([]any)
	if len(unscopedCallers) != 2 {
		t.Errorf("unscoped entry Result[\"callers\"] = %v; want both callers (no \"within\" filter applied)", unscopedCallers)
	}
}

// TestRewordMarshalFailure_BeginsWithImpactNeverToc asserts rewordMarshalFailure reworks a
// StructToFields-style failure — which always carries a literal "toc: " prefix, since
// cli.StructToFields was written for the toc verbs — into a message beginning "impact: " and never
// "toc: ". This is resolveImpactEntry's own disposition for a marshal failure
// (cli.StructToFields(result) returning a non-nil error), exercised directly here because
// quarry.ImpactResult's field types (strings, ints, and pointers to structs of the same) always
// marshal successfully, so a stubbed impactFn can never itself trigger that branch through the full
// handler.
func TestRewordMarshalFailure_BeginsWithImpactNeverToc(t *testing.T) {
	msg := rewordMarshalFailure(fmt.Errorf("toc: marshal result: %w", errors.New("json: unsupported type: chan int")))
	if !strings.HasPrefix(msg, "impact: ") {
		t.Errorf("rewordMarshalFailure(...) = %q; want it to begin with %q", msg, "impact: ")
	}
	if strings.HasPrefix(msg, "toc: ") {
		t.Errorf("rewordMarshalFailure(...) = %q; want it to never begin with %q", msg, "toc: ")
	}
}

// TestImpactHandler_PositionsStay1Based asserts a found entry's marshalled result carries the
// engine's own 1-based line/character values unconverted.
func TestImpactHandler_PositionsStay1Based(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &impactFn, stubImpactFn(t, map[string]struct {
		result quarry.ImpactResult
		err    error
	}{
		"Foo": {result: quarry.ImpactResult{
			Callers: []quarry.ImpactCaller{{File: "a.go", CallSiteLine: 5, CallSiteCharacter: 10}},
		}},
	}))

	in := mustUnmarshal[impactInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := impactHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("impactHandler(cfg)(...) error = %v", err)
	}

	data, err := json.Marshal(out.Results[0].Result)
	if err != nil {
		t.Fatalf("json.Marshal(out.Results[0].Result) error = %v", err)
	}
	var decoded struct {
		Callers []struct {
			CallSiteLine      int `json:"call_site_line"`
			CallSiteCharacter int `json:"call_site_character"`
		} `json:"callers"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(...) error = %v", err)
	}
	if len(decoded.Callers) != 1 {
		t.Fatalf("len(decoded.Callers) = %d; want 1", len(decoded.Callers))
	}
	if decoded.Callers[0].CallSiteLine != 5 || decoded.Callers[0].CallSiteCharacter != 10 {
		t.Errorf("decoded.Callers[0] = %+v; want CallSiteLine=5 CallSiteCharacter=10 (1-based, unconverted)", decoded.Callers[0])
	}
}
