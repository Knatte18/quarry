package mcpserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// stubCallersFn returns a callersFn-shaped stub keyed on the query's own identity (symbol name,
// in-file name, or position file:line:character), following stubLookupFn's own convention in
// tools_lsp_test.go. It also records opts.SkipVerification for each call, keyed the same way, so a
// test can assert what the handler actually passed through.
func stubCallersFn(t *testing.T, responses map[string]struct {
	refs     []quarry.Reference
	declRefs []quarry.Reference
	err      error
}, skipVerificationSeen map[string]bool) func(context.Context, quarry.Options) ([]quarry.Reference, []quarry.Reference, error) {
	t.Helper()
	return func(_ context.Context, opts quarry.Options) ([]quarry.Reference, []quarry.Reference, error) {
		key := queryKey(opts.Query)
		if skipVerificationSeen != nil {
			skipVerificationSeen[key] = opts.SkipVerification
		}
		r, ok := responses[key]
		if !ok {
			t.Fatalf("stub callers fn: no response configured for query key %q (query %+v)", key, opts.Query)
		}
		return r.refs, r.declRefs, r.err
	}
}

// TestAssertHandler_CleanCheck_IsFoundWithViolationFalseAndEmptyCallers asserts a clean check
// (no unexpected callers) is statusFound with violation:false present and an empty callers array.
func TestAssertHandler_CleanCheck_IsFoundWithViolationFalseAndEmptyCallers(t *testing.T) {
	cfg := newTestConfig(t)
	declFile := filepath.Join(cfg.TargetDir, "decl.go")

	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{
		"Foo": {
			refs:     []quarry.Reference{{File: declFile, Line: 1, Character: 1}},
			declRefs: []quarry.Reference{{File: declFile, Line: 1, Character: 1}},
		},
	}, nil))

	in := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := assertHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}

	entry := out.Results[0]
	if entry.Status != statusFound {
		t.Fatalf("entry.Status = %q; want %q", entry.Status, statusFound)
	}
	if entry.Violation == nil || *entry.Violation {
		t.Errorf("entry.Violation = %v; want a non-nil pointer to false", entry.Violation)
	}
	if entry.Callers == nil {
		t.Errorf("entry.Callers = nil; want a non-nil empty slice")
	}
	if len(entry.Callers) != 0 {
		t.Errorf("entry.Callers = %v; want empty", entry.Callers)
	}

	// The runtime half of the non-nil-slice guarantee: the marshaled clean entry must still carry
	// "callers": []. omitempty would drop the empty-but-non-nil slice; the omitzero tag keeps it.
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal(entry) error = %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(data, &asMap); err != nil {
		t.Fatalf("json.Unmarshal(data) error = %v", err)
	}
	callers, ok := asMap["callers"].([]any)
	if !ok {
		t.Fatalf("marshaled clean entry \"callers\" = %v; want a present, empty JSON array", asMap["callers"])
	}
	if len(callers) != 0 {
		t.Errorf("marshaled clean entry \"callers\" = %v; want empty", callers)
	}
}

// TestAssertHandler_ViolatingCheck_IsFoundWithViolationTrueAndPopulatedCallers asserts a check with
// violations is statusFound with violation:true and populated callers, and that this never sets a
// whole-call failure.
func TestAssertHandler_ViolatingCheck_IsFoundWithViolationTrueAndPopulatedCallers(t *testing.T) {
	cfg := newTestConfig(t)
	declFile := filepath.Join(cfg.TargetDir, "decl.go")
	callerFile := filepath.Join(cfg.TargetDir, "caller.go")

	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{
		"Foo": {
			refs: []quarry.Reference{
				{File: declFile, Line: 1, Character: 1},
				{File: callerFile, Line: 5, Character: 3},
			},
			declRefs: []quarry.Reference{{File: declFile, Line: 1, Character: 1}},
		},
	}, nil))

	in := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Foo"}]}`)

	_, out, err := assertHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v; want nil (a violation never sets a whole-call failure)", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}

	entry := out.Results[0]
	if entry.Status != statusFound {
		t.Fatalf("entry.Status = %q; want %q", entry.Status, statusFound)
	}
	if entry.Violation == nil || !*entry.Violation {
		t.Errorf("entry.Violation = %v; want a non-nil pointer to true", entry.Violation)
	}
	if len(entry.Callers) != 1 || entry.Callers[0].File != callerFile {
		t.Errorf("entry.Callers = %v; want exactly the one unexpected caller %q", entry.Callers, callerFile)
	}
}

// TestAssertHandler_RelativeExceptResolvesAgainstTargetDir asserts a relative "except" path exempts
// the intended file, the regression that appears when the path is resolved against the process
// working directory instead of the call's own target directory.
func TestAssertHandler_RelativeExceptResolvesAgainstTargetDir(t *testing.T) {
	cfg := newTestConfig(t)
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

	in := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Foo","except":["wrapper.go"]}]}`)

	_, out, err := assertHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v", err)
	}
	entry := out.Results[0]
	if entry.Violation == nil || *entry.Violation {
		t.Errorf("entry.Violation = %v; want a non-nil pointer to false (relative except resolved against targetDir must exempt wrapper.go)", entry.Violation)
	}
	if len(entry.Callers) != 0 {
		t.Errorf("entry.Callers = %v; want empty (wrapper.go exempted)", entry.Callers)
	}
}

// TestAssertHandler_OneEntrysExceptNeverAffectsAnother asserts one entry's "except" never exempts
// another entry's check.
func TestAssertHandler_OneEntrysExceptNeverAffectsAnother(t *testing.T) {
	cfg := newTestConfig(t)
	declFile := filepath.Join(cfg.TargetDir, "decl.go")
	wrapperFile := filepath.Join(cfg.TargetDir, "wrapper.go")

	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{
		"Exempted": {
			refs: []quarry.Reference{
				{File: declFile, Line: 1, Character: 1},
				{File: wrapperFile, Line: 2, Character: 2},
			},
			declRefs: []quarry.Reference{{File: declFile, Line: 1, Character: 1}},
		},
		"NotExempted": {
			refs: []quarry.Reference{
				{File: declFile, Line: 1, Character: 1},
				{File: wrapperFile, Line: 2, Character: 2},
			},
			declRefs: []quarry.Reference{{File: declFile, Line: 1, Character: 1}},
		},
	}, nil))

	in := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Exempted","except":["wrapper.go"]},{"symbol":"NotExempted"}]}`)

	_, out, err := assertHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("len(out.Results) = %d; want 2", len(out.Results))
	}

	exempted := out.Results[0]
	if exempted.Violation == nil || *exempted.Violation {
		t.Errorf("exempted entry Violation = %v; want a non-nil pointer to false", exempted.Violation)
	}

	notExempted := out.Results[1]
	if notExempted.Violation == nil || !*notExempted.Violation {
		t.Errorf("not-exempted entry Violation = %v; want a non-nil pointer to true (this entry's own except is empty)", notExempted.Violation)
	}
}

// TestAssertHandler_CallWideNoVerifyReachesSkipVerification asserts the call-wide "noVerify" reaches
// quarry.Options.SkipVerification while the default leaves it false.
func TestAssertHandler_CallWideNoVerifyReachesSkipVerification(t *testing.T) {
	cfg := newTestConfig(t)
	seen := make(map[string]bool)

	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{
		"Foo": {},
		"Bar": {},
	}, seen))

	inNoVerify := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Foo"}],"noVerify":true}`)
	if _, _, err := assertHandler(cfg)(context.Background(), nil, inNoVerify); err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v", err)
	}
	if !seen["Foo"] {
		t.Errorf("opts.SkipVerification for %q = %v; want true (noVerify:true)", "Foo", seen["Foo"])
	}

	inDefault := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Bar"}]}`)
	if _, _, err := assertHandler(cfg)(context.Background(), nil, inDefault); err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v", err)
	}
	if seen["Bar"] {
		t.Errorf("opts.SkipVerification for %q = %v; want false (default)", "Bar", seen["Bar"])
	}
}

// TestAssertHandler_SymbolNotFoundYieldsNotFoundStatus asserts quarry.ErrSymbolNotFoundSentinel
// yields status:"not_found" rather than "error".
func TestAssertHandler_SymbolNotFoundYieldsNotFoundStatus(t *testing.T) {
	cfg := newTestConfig(t)
	withStubbedFacade(t, &callersFn, stubCallersFn(t, map[string]struct {
		refs     []quarry.Reference
		declRefs []quarry.Reference
		err      error
	}{
		"Missing": {err: quarry.ErrSymbolNotFoundSentinel},
	}, nil))

	in := mustUnmarshal[assertInput](t, `{"targets":[{"symbol":"Missing"}]}`)

	_, out, err := assertHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("assertHandler(cfg)(...) error = %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	if got := out.Results[0].Status; got != statusNotFound {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusNotFound)
	}
}
