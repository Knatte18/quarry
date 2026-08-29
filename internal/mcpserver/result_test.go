package mcpserver

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

func TestClassifyLSPError(t *testing.T) {
	sentinelWrapped := fmt.Errorf("wrapped: %w", quarry.ErrSymbolNotFoundSentinel)
	ambiguous := &quarry.ErrAmbiguousSymbol{Symbol: "Foo", Candidates: []string{"a.go:1:1", "b.go:2:2"}}
	other := errors.New("boom")

	tests := []struct {
		name           string
		err            error
		wantStatus     string
		wantCandidates []string
		wantMessage    string
	}{
		{"Nil", nil, statusFound, nil, ""},
		{"AmbiguousSentinel", ambiguous, statusAmbiguous, ambiguous.Candidates, ""},
		{"NotFoundSentinel", quarry.ErrSymbolNotFoundSentinel, statusNotFound, nil, ""},
		{"WrappedNotFoundSentinel", sentinelWrapped, statusNotFound, nil, ""},
		{"OtherError", other, statusError, nil, other.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, candidates, message := classifyLSPError(tt.err)
			if status != tt.wantStatus {
				t.Errorf("classifyLSPError(%v) status = %q; want %q", tt.err, status, tt.wantStatus)
			}
			if len(candidates) != len(tt.wantCandidates) {
				t.Errorf("classifyLSPError(%v) candidates = %v; want %v", tt.err, candidates, tt.wantCandidates)
			}
			if message != tt.wantMessage {
				t.Errorf("classifyLSPError(%v) message = %q; want %q", tt.err, message, tt.wantMessage)
			}
		})
	}
}

// TestClassifySymbolError_NoAmbiguousBranch asserts classifySymbolError has no ambiguous branch:
// an *quarry.ErrAmbiguousSymbol, which classifyLSPError maps to statusAmbiguous, falls through to
// classifySymbolError's catch-all statusError branch instead.
func TestClassifySymbolError_NoAmbiguousBranch(t *testing.T) {
	ambiguous := &quarry.ErrAmbiguousSymbol{Symbol: "Foo", Candidates: []string{"a.go:1:1", "b.go:2:2"}}

	tests := []struct {
		name        string
		err         error
		wantStatus  string
		wantMessage string
	}{
		{"Nil", nil, statusFound, ""},
		{"NotFoundSentinel", quarry.ErrSymbolNotFoundSentinel, statusNotFound, ""},
		{"AmbiguousFallsThroughToError", ambiguous, statusError, ambiguous.Error()},
		{"OtherError", errors.New("boom"), statusError, "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, message := classifySymbolError(tt.err)
			if status != tt.wantStatus {
				t.Errorf("classifySymbolError(%v) status = %q; want %q", tt.err, status, tt.wantStatus)
			}
			if message != tt.wantMessage {
				t.Errorf("classifySymbolError(%v) message = %q; want %q", tt.err, message, tt.wantMessage)
			}
		})
	}
}

// TestClassifyTOCError_DoesNotBorrowLSPPredicates asserts the toc classifier applies its own rule
// rather than classifyLSPError's predicates: quarry.ErrSymbolNotFoundSentinel — which
// classifyLSPError maps to statusNotFound — must still map to statusError here, because toc uses
// no language server and has no "not_found" outcome of its own kind to distinguish it from.
func TestClassifyTOCError_DoesNotBorrowLSPPredicates(t *testing.T) {
	status, message := classifyTOCError(quarry.ErrSymbolNotFoundSentinel)
	if status != statusError {
		t.Errorf("classifyTOCError(ErrSymbolNotFoundSentinel) status = %q; want %q (toc must not borrow the LSP not-found predicate)", status, statusError)
	}
	if message != quarry.ErrSymbolNotFoundSentinel.Error() {
		t.Errorf("classifyTOCError(ErrSymbolNotFoundSentinel) message = %q; want %q", message, quarry.ErrSymbolNotFoundSentinel.Error())
	}
}

func TestClassifyTOCError_LanguageUnsupported(t *testing.T) {
	wrapped := fmt.Errorf("toc: resolve language: %w", quarry.ErrLanguageUnsupported)
	status, message := classifyTOCError(wrapped)
	if status != statusError {
		t.Errorf("classifyTOCError(wrapped ErrLanguageUnsupported) status = %q; want %q", status, statusError)
	}
	want := fmt.Sprintf("toc: language not yet supported; quarry can currently read: %s", strings.Join(quarry.TOCImplemented(), ", "))
	if message != want {
		t.Errorf("classifyTOCError(wrapped ErrLanguageUnsupported) message = %q; want %q", message, want)
	}
}

func TestReferenceFieldsWire(t *testing.T) {
	refs := []quarry.Reference{{File: "a.go", Line: 5, Character: 10}}
	got := referenceFieldsWire(refs)
	want := referenceField{File: "a.go", Line: 4, Character: 9}
	if len(got) != 1 || got[0] != want {
		t.Errorf("referenceFieldsWire(%v) = %v; want [%v]", refs, got, want)
	}
}

func TestReferenceFieldsNative(t *testing.T) {
	refs := []quarry.Reference{{File: "a.go", Line: 5, Character: 10}}
	got := referenceFieldsNative(refs)
	want := referenceField{File: "a.go", Line: 5, Character: 10}
	if len(got) != 1 || got[0] != want {
		t.Errorf("referenceFieldsNative(%v) = %v; want [%v]", refs, got, want)
	}
}

// TestReferenceFields_EmptyInputYieldsNonNil asserts both converters return a non-nil empty slice
// for an empty input, so an empty result marshals as "[]" rather than "null".
func TestReferenceFields_EmptyInputYieldsNonNil(t *testing.T) {
	if got := referenceFieldsWire(nil); got == nil {
		t.Error("referenceFieldsWire(nil) = nil; want a non-nil empty slice")
	}
	if got := referenceFieldsNative(nil); got == nil {
		t.Error("referenceFieldsNative(nil) = nil; want a non-nil empty slice")
	}
}

func TestSymbolFieldsWire(t *testing.T) {
	matches := []quarry.SymbolMatch{{Name: "Foo", Kind: 12, File: "a.go", Line: 5, Character: 10}}
	got := symbolFieldsWire(matches)
	want := symbolField{Name: "Foo", Kind: 12, File: "a.go", Line: 4, Character: 9}
	if len(got) != 1 || got[0] != want {
		t.Errorf("symbolFieldsWire(%v) = %v; want [%v]", matches, got, want)
	}
}

func TestSymbolFieldsWire_EmptyInputYieldsNonNil(t *testing.T) {
	if got := symbolFieldsWire(nil); got == nil {
		t.Error("symbolFieldsWire(nil) = nil; want a non-nil empty slice")
	}
}

func TestRewordMarshalFailure(t *testing.T) {
	err := errors.New("toc: marshal result: boom")
	got := rewordMarshalFailure(err)
	want := "impact: marshal result: boom"
	if got != want {
		t.Errorf("rewordMarshalFailure(%v) = %q; want %q", err, got, want)
	}
}
