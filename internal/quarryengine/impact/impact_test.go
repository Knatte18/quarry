// impact_test.go covers buildResult directly against hand-built query.Reference slices and an
// injected parse function — never through Impact, which needs a live language server and is
// proved by the live-tier test in batch 4 instead.

package impact

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine/query"
	"github.com/Knatte18/quarry/internal/quarryengine/toc"
)

// fakeFile is one entry in a fakeParser's fixed table: either a FileTOC to return, or an error.
type fakeFile struct {
	toc toc.FileTOC
	err error
}

// fakeParser builds a parseFunc backed by a fixed path -> fakeFile table, counting how many times
// each path is actually parsed. A path with no table entry is a test-authoring error, so it fails
// the test immediately rather than returning a zero FileTOC silently.
type fakeParser struct {
	t      *testing.T
	files  map[string]fakeFile
	counts map[string]int
}

func newFakeParser(t *testing.T, files map[string]fakeFile) *fakeParser {
	t.Helper()
	return &fakeParser{t: t, files: files, counts: make(map[string]int)}
}

func (p *fakeParser) parse(path string) (toc.FileTOC, error) {
	p.counts[path]++
	f, ok := p.files[path]
	if !ok {
		p.t.Fatalf("fakeParser: no table entry for path %q", path)
	}
	return f.toc, f.err
}

// applyDiscountSymbol is the shared stand-in for billing.Invoice.ApplyDiscount: a docstring
// starting on line 17, a signature ending on line 20, and a body running through line 22 —
// matching the shape of the real impactfixture symbol these tests model.
var applyDiscountSymbol = toc.Symbol{
	Kind:      toc.KindMethod,
	Name:      "ApplyDiscount",
	Owner:     "Invoice",
	Signature: "func (inv *Invoice) ApplyDiscount(rate float64)",
	Start:     17,
	SigEnd:    20,
	End:       22,
}

// processRefundSymbol stands in for refund.ProcessRefund, the enclosing function two call sites
// share.
var processRefundSymbol = toc.Symbol{
	Kind:      toc.KindFunction,
	Name:      "ProcessRefund",
	Signature: "func ProcessRefund(inv *billing.Invoice)",
	Start:     10,
	SigEnd:    13,
	End:       16,
}

func TestBuildResult_ExcludesDeclarationSiteFromCallers(t *testing.T) {
	parser := newFakeParser(t, map[string]fakeFile{
		"/billing/invoice.go": {toc: toc.FileTOC{Package: "billing", Symbols: []toc.Symbol{applyDiscountSymbol}}},
		"/refund/refund.go":   {toc: toc.FileTOC{Package: "refund", Symbols: []toc.Symbol{processRefundSymbol}}},
	})
	declaration := []query.Reference{{File: "/billing/invoice.go", Line: 20, Character: 6}}
	callers := []query.Reference{
		declaration[0],
		{File: "/refund/refund.go", Line: 14, Character: 6},
	}

	got, err := buildResult(context.Background(), callers, declaration, newFileCache(parser.parse))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	if len(got.Callers) != 1 {
		t.Fatalf("len(Callers) = %d; want 1 (the declaration site excluded)", len(got.Callers))
	}
	if got.Callers[0].File != "/refund/refund.go" || got.Callers[0].CallSiteLine != 14 {
		t.Errorf("Callers[0] = %+v; want the refund.go:14 entry", got.Callers[0])
	}
}

func TestBuildResult_RecursiveSelfCallRetained(t *testing.T) {
	// selfCaller is a call inside ApplyDiscount's own body (a recursive call), not the declaration
	// site itself, so it must be kept and its enclosing symbol must be the target itself.
	parser := newFakeParser(t, map[string]fakeFile{
		"/billing/invoice.go": {toc: toc.FileTOC{Package: "billing", Symbols: []toc.Symbol{applyDiscountSymbol}}},
	})
	declaration := []query.Reference{{File: "/billing/invoice.go", Line: 20, Character: 6}}
	callers := []query.Reference{
		declaration[0],
		{File: "/billing/invoice.go", Line: 21, Character: 3},
	}

	got, err := buildResult(context.Background(), callers, declaration, newFileCache(parser.parse))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	if len(got.Callers) != 1 {
		t.Fatalf("len(Callers) = %d; want 1", len(got.Callers))
	}
	if got.Callers[0].Name != "ApplyDiscount" {
		t.Errorf("Callers[0].Name = %q; want %q (the recursive call's own enclosing symbol)", got.Callers[0].Name, "ApplyDiscount")
	}
}

func TestBuildResult_TwoCallSitesOneEnclosingFunction(t *testing.T) {
	parser := newFakeParser(t, map[string]fakeFile{
		"/refund/refund.go": {toc: toc.FileTOC{Package: "refund", Symbols: []toc.Symbol{processRefundSymbol}}},
	})
	callers := []query.Reference{
		{File: "/refund/refund.go", Line: 14, Character: 6},
		{File: "/refund/refund.go", Line: 15, Character: 6},
	}

	got, err := buildResult(context.Background(), callers, nil, newFileCache(parser.parse))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	if len(got.Callers) != 2 {
		t.Fatalf("len(Callers) = %d; want 2", len(got.Callers))
	}
	if got.Callers[0].CallSiteLine == got.Callers[1].CallSiteLine {
		t.Errorf("Callers[0].CallSiteLine == Callers[1].CallSiteLine == %d; want distinct call-site lines", got.Callers[0].CallSiteLine)
	}
	if got.Callers[0].EnclosingRange == nil || got.Callers[1].EnclosingRange == nil {
		t.Fatalf("EnclosingRange = %+v, %+v; want both non-nil", got.Callers[0].EnclosingRange, got.Callers[1].EnclosingRange)
	}
	if *got.Callers[0].EnclosingRange != *got.Callers[1].EnclosingRange {
		t.Errorf("EnclosingRange mismatch: %+v vs %+v; want equal ranges (same enclosing function)", *got.Callers[0].EnclosingRange, *got.Callers[1].EnclosingRange)
	}
}

func TestBuildResult_SortsByFileThenLineThenCharacter(t *testing.T) {
	parser := newFakeParser(t, map[string]fakeFile{
		"/b.go": {toc: toc.FileTOC{Package: "b", Symbols: nil}},
		"/a.go": {toc: toc.FileTOC{Package: "a", Symbols: nil}},
	})
	callers := []query.Reference{
		{File: "/b.go", Line: 5, Character: 2},
		{File: "/a.go", Line: 9, Character: 1},
		{File: "/a.go", Line: 3, Character: 4},
		{File: "/a.go", Line: 3, Character: 1},
	}

	got, err := buildResult(context.Background(), callers, nil, newFileCache(parser.parse))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	want := []query.Reference{
		{File: "/a.go", Line: 3, Character: 1},
		{File: "/a.go", Line: 3, Character: 4},
		{File: "/a.go", Line: 9, Character: 1},
		{File: "/b.go", Line: 5, Character: 2},
	}
	if len(got.Callers) != len(want) {
		t.Fatalf("len(Callers) = %d; want %d", len(got.Callers), len(want))
	}
	for i, w := range want {
		c := got.Callers[i]
		if c.File != w.File || c.CallSiteLine != w.Line || c.CallSiteCharacter != w.Character {
			t.Errorf("Callers[%d] = {%q, %d, %d}; want {%q, %d, %d}", i, c.File, c.CallSiteLine, c.CallSiteCharacter, w.File, w.Line, w.Character)
		}
	}
}

func TestBuildResult_EmptyCallerSetMarshalsToEmptyArray(t *testing.T) {
	got, err := buildResult(context.Background(), nil, nil, newFileCache(nil))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal(got) returned error: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("json.Unmarshal(marshaled Result) returned error: %v", err)
	}
	if got, want := string(raw["callers"]), "[]"; got != want {
		t.Errorf(`marshaled "callers" = %s; want %s`, got, want)
	}
}

func TestBuildResult_EmptyDeclarationSetOmitsTargetAndDefinition(t *testing.T) {
	callers := []query.Reference{{File: "/x.go", Line: 1, Character: 1}}
	parser := newFakeParser(t, map[string]fakeFile{
		"/x.go": {toc: toc.FileTOC{Package: "x", Symbols: nil}},
	})

	got, err := buildResult(context.Background(), callers, nil, newFileCache(parser.parse))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	if got.Target != nil {
		t.Errorf("Target = %+v; want nil", got.Target)
	}
	if got.Definition != nil {
		t.Errorf("Definition = %+v; want nil", got.Definition)
	}
	if len(got.Callers) != 1 {
		t.Errorf("len(Callers) = %d; want 1", len(got.Callers))
	}
}

// TestBuildResult_DefinitionOutcomes covers all three outcomes of the three-outcome-degradation-
// rule on the definition side, each as its own case.
func TestBuildResult_DefinitionOutcomes(t *testing.T) {
	t.Run("Resolved", func(t *testing.T) {
		parser := newFakeParser(t, map[string]fakeFile{
			"/billing/invoice.go": {toc: toc.FileTOC{Package: "billing", Symbols: []toc.Symbol{applyDiscountSymbol}}},
		})
		declaration := []query.Reference{{File: "/billing/invoice.go", Line: 20, Character: 6}}

		got, err := buildResult(context.Background(), nil, declaration, newFileCache(parser.parse))
		if err != nil {
			t.Fatalf("buildResult(...) returned error: %v", err)
		}
		if got.Target == nil {
			t.Fatal("Target = nil; want a resolved Target")
		}
		if got.Target.Name != "ApplyDiscount" || got.Target.Package != "billing" {
			t.Errorf("Target = %+v; want Name=ApplyDiscount, Package=billing", got.Target)
		}
		if got.Definition == nil {
			t.Fatal("Definition = nil; want a non-nil Definition")
		}
		if got.Definition.Error != "" {
			t.Errorf("Definition.Error = %q; want empty", got.Definition.Error)
		}
		if got.Definition.StartLine != applyDiscountSymbol.Start || got.Definition.EndLine != applyDiscountSymbol.End {
			t.Errorf("Definition range = {%d, %d}; want {%d, %d}", got.Definition.StartLine, got.Definition.EndLine, applyDiscountSymbol.Start, applyDiscountSymbol.End)
		}
	})

	t.Run("ParsedNoEnclosingSymbol", func(t *testing.T) {
		// Line 15 lands on a package-level var, which has no listable declaration covering it —
		// outcome 2, the file-scope outcome, not a failure.
		parser := newFakeParser(t, map[string]fakeFile{
			"/billing/invoice.go": {toc: toc.FileTOC{Package: "billing", Symbols: []toc.Symbol{applyDiscountSymbol}}},
		})
		declaration := []query.Reference{{File: "/billing/invoice.go", Line: 15, Character: 5}}

		got, err := buildResult(context.Background(), nil, declaration, newFileCache(parser.parse))
		if err != nil {
			t.Fatalf("buildResult(...) returned error: %v", err)
		}
		if got.Target != nil {
			t.Errorf("Target = %+v; want nil (no enclosing symbol found)", got.Target)
		}
		if got.Definition == nil {
			t.Fatal("Definition = nil; want a non-nil Definition")
		}
		if got.Definition.Error != "" {
			t.Errorf("Definition.Error = %q; want empty (file-scope is not a failure)", got.Definition.Error)
		}
		if got.Definition.StartLine != 0 || got.Definition.SigEndLine != 0 || got.Definition.EndLine != 0 {
			t.Errorf("Definition range = {%d, %d, %d}; want all zero", got.Definition.StartLine, got.Definition.SigEndLine, got.Definition.EndLine)
		}
		if got.Definition.File != "/billing/invoice.go" || got.Definition.Line != 15 {
			t.Errorf("Definition = {%q, %d}; want {/billing/invoice.go, 15}", got.Definition.File, got.Definition.Line)
		}
	})

	t.Run("NoTocStrategyForDeclaringFile", func(t *testing.T) {
		parser := newFakeParser(t, map[string]fakeFile{
			"/tsfixture/client.ts": {err: errors.New(`toc: /tsfixture/client.ts: language "typescript" has no toc strategy`)},
		})
		declaration := []query.Reference{{File: "/tsfixture/client.ts", Line: 3, Character: 1}}

		got, err := buildResult(context.Background(), nil, declaration, newFileCache(parser.parse))
		if err != nil {
			t.Fatalf("buildResult(...) returned error: %v", err)
		}
		if got.Target != nil {
			t.Errorf("Target = %+v; want nil", got.Target)
		}
		if got.Definition == nil {
			t.Fatal("Definition = nil; want a non-nil Definition")
		}
		if got.Definition.File != "/tsfixture/client.ts" || got.Definition.Line != 3 {
			t.Errorf("Definition = {%q, %d}; want {/tsfixture/client.ts, 3}", got.Definition.File, got.Definition.Line)
		}
		if got.Definition.Error == "" {
			t.Error("Definition.Error = \"\"; want a parse error naming the unsupported language")
		}
		if got.Definition.StartLine != 0 || got.Definition.SigEndLine != 0 || got.Definition.EndLine != 0 {
			t.Errorf("Definition range = {%d, %d, %d}; want all zero", got.Definition.StartLine, got.Definition.SigEndLine, got.Definition.EndLine)
		}
	})
}

func TestBuildResult_CallerFileWithNoRegisteredStrategy(t *testing.T) {
	parser := newFakeParser(t, map[string]fakeFile{
		"/tsfixture/client.ts": {err: errors.New(`toc: /tsfixture/client.ts: language "typescript" has no toc strategy`)},
	})
	callers := []query.Reference{{File: "/tsfixture/client.ts", Line: 3, Character: 1}}

	got, err := buildResult(context.Background(), callers, nil, newFileCache(parser.parse))
	if err != nil {
		t.Fatalf("buildResult(...) returned error: %v", err)
	}
	if len(got.Callers) != 1 {
		t.Fatalf("len(Callers) = %d; want 1", len(got.Callers))
	}
	c := got.Callers[0]
	if c.CallSiteLine != 3 {
		t.Errorf("CallSiteLine = %d; want 3", c.CallSiteLine)
	}
	if c.Error == "" {
		t.Error("Error = \"\"; want a per-entry parse error")
	}
	if c.EnclosingRange != nil {
		t.Errorf("EnclosingRange = %+v; want nil", c.EnclosingRange)
	}
}

// TestBuildResult_CancellationStopsAtFirstCallerCacheMiss covers cancellation: an already-
// cancelled context makes buildResult return that error at the first caller cache-miss parse, not
// at the definition-side lookup, which is deliberately never checked. The injected parse counter
// pins that ordering: the definition-side file is parsed exactly once (before the error surfaces),
// and no caller file is parsed at all.
func TestBuildResult_CancellationStopsAtFirstCallerCacheMiss(t *testing.T) {
	parser := newFakeParser(t, map[string]fakeFile{
		"/billing/invoice.go": {toc: toc.FileTOC{Package: "billing", Symbols: []toc.Symbol{applyDiscountSymbol}}},
		"/refund/refund.go":   {toc: toc.FileTOC{Package: "refund", Symbols: []toc.Symbol{processRefundSymbol}}},
	})
	declaration := []query.Reference{{File: "/billing/invoice.go", Line: 20, Character: 6}}
	callers := []query.Reference{
		declaration[0],
		{File: "/refund/refund.go", Line: 14, Character: 6},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := buildResult(ctx, callers, declaration, newFileCache(parser.parse))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("buildResult(...) error = %v; want context.Canceled", err)
	}
	if parser.counts["/billing/invoice.go"] != 1 {
		t.Errorf("parse count for /billing/invoice.go = %d; want 1 (the definition-side lookup, run before the cancellation check)", parser.counts["/billing/invoice.go"])
	}
	if parser.counts["/refund/refund.go"] != 0 {
		t.Errorf("parse count for /refund/refund.go = %d; want 0 (cancellation must stop the caller loop before this cache-miss parse)", parser.counts["/refund/refund.go"])
	}
}
