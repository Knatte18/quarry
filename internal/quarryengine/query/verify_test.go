// verify_test.go covers verify.go's pure functions against hand-built lsp.Location values — no
// fake LSP server is needed since nothing under test performs I/O.

package query

import (
	"errors"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
)

// loc builds an lsp.Location at uri/line/character for table brevity below.
func loc(uri string, line, character int) lsp.Location {
	return lsp.Location{URI: uri, Range: lsp.Range{Start: lsp.Position{Line: line, Character: character}}}
}

// TestFilterVerifiedReferences covers the fail-closed predicate: a reference is dropped only when
// its own definition call positively disproves it (attempted, no error, non-empty, and none of its
// locations' keys is in the match set); every other case keeps the reference.
func TestFilterVerifiedReferences(t *testing.T) {
	matchSet := map[locationKey]bool{
		keyOf(loc("file:///a.go", 10, 2)): true,
	}

	tests := []struct {
		name     string
		refs     []lsp.Location
		matchSet map[locationKey]bool
		outcomes []verificationOutcome
		wantLen  int
	}{
		{
			name:     "DefinitionMatchesMatchSet_Kept",
			refs:     []lsp.Location{loc("file:///a.go", 30, 5)},
			matchSet: matchSet,
			outcomes: []verificationOutcome{{Attempted: true, Locations: []lsp.Location{loc("file:///a.go", 10, 2)}}},
			wantLen:  1,
		},
		{
			name:     "DefinitionPointsElsewhere_Dropped",
			refs:     []lsp.Location{loc("file:///a.go", 40, 5)},
			matchSet: matchSet,
			outcomes: []verificationOutcome{{Attempted: true, Locations: []lsp.Location{loc("file:///a.go", 99, 9)}}},
			wantLen:  0,
		},
		{
			name:     "DefinitionCallErrored_Kept",
			refs:     []lsp.Location{loc("file:///a.go", 50, 5)},
			matchSet: matchSet,
			outcomes: []verificationOutcome{{Attempted: true, Err: errors.New("boom")}},
			wantLen:  1,
		},
		{
			name:     "DefinitionReturnedEmpty_Kept",
			refs:     []lsp.Location{loc("file:///a.go", 60, 5)},
			matchSet: matchSet,
			outcomes: []verificationOutcome{{Attempted: true, Locations: nil}},
			wantLen:  1,
		},
		{
			name:     "NeverAttempted_Kept",
			refs:     []lsp.Location{loc("file:///a.go", 70, 5)},
			matchSet: matchSet,
			outcomes: []verificationOutcome{{Attempted: false}},
			wantLen:  1,
		},
		{
			name: "TwoSameNamedDeclarationsOneLineApart_PositionalNotFileLevel",
			refs: []lsp.Location{
				loc("file:///a.go", 1, 1), // matches the interface decl at line 10
				loc("file:///a.go", 2, 1), // resolves to a different, unrelated decl at line 11
			},
			matchSet: map[locationKey]bool{
				keyOf(loc("file:///a.go", 10, 2)): true,
			},
			outcomes: []verificationOutcome{
				{Attempted: true, Locations: []lsp.Location{loc("file:///a.go", 10, 2)}},
				{Attempted: true, Locations: []lsp.Location{loc("file:///a.go", 11, 2)}},
			},
			wantLen: 1,
		},
		{
			name: "EmptyMatchSetDropsEveryAttemptedNonMatchingReference",
			refs: []lsp.Location{
				loc("file:///a.go", 80, 5),
				loc("file:///a.go", 90, 5),
			},
			matchSet: map[locationKey]bool{},
			outcomes: []verificationOutcome{
				{Attempted: true, Locations: []lsp.Location{loc("file:///a.go", 1, 1)}},
				{Attempted: true, Locations: []lsp.Location{loc("file:///a.go", 2, 1)}},
			},
			// Verifying against an empty match set drops every attempted
			// reference whose definition returned a non-empty result — this
			// is exactly why callers.go must never call
			// filterVerifiedReferences with an empty match set and skips
			// verification entirely instead.
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterVerifiedReferences(tt.refs, tt.matchSet, tt.outcomes)
			if len(got) != tt.wantLen {
				t.Errorf("filterVerifiedReferences() returned %d references; want %d (got=%v)", len(got), tt.wantLen, got)
			}
		})
	}
}

// TestDeclarationMatchSet_Directional asserts declarationMatchSet returns every defLocs key plus
// only those implLocs keys present in interfaceDecl.
func TestDeclarationMatchSet_Directional(t *testing.T) {
	defLocs := []lsp.Location{loc("file:///a.go", 1, 1)}
	implLocs := []lsp.Location{
		loc("file:///b.go", 2, 2), // classified as an interface declaration: included
		loc("file:///c.go", 3, 3), // a concrete satisfier: excluded
	}
	interfaceDecl := map[locationKey]bool{
		keyOf(implLocs[0]): true,
	}

	got := declarationMatchSet(defLocs, implLocs, interfaceDecl)

	if !got[keyOf(defLocs[0])] {
		t.Error("declarationMatchSet() missing the definition-side key")
	}
	if !got[keyOf(implLocs[0])] {
		t.Error("declarationMatchSet() missing the interface-classified implementation key")
	}
	if got[keyOf(implLocs[1])] {
		t.Error("declarationMatchSet() unexpectedly includes the concrete (non-interface) implementation key")
	}
	if len(got) != 2 {
		t.Errorf("declarationMatchSet() returned %d keys; want 2", len(got))
	}
}

// TestIsInterfaceDeclaration covers each of the required classification cases: a position inside
// an interface's method returns true, a position inside a concrete type's method returns false, a
// position matching no symbol returns false, and a nested chain where the interface is an ancestor
// rather than the innermost symbol returns true.
func TestIsInterfaceDeclaration(t *testing.T) {
	tree := []lsp.DocumentSymbol{
		{
			Name:  "clock",
			Kind:  symbolKindInterface,
			Range: lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 4, Character: 1}},
			Children: []lsp.DocumentSymbol{
				{
					Name:  "Now",
					Kind:  6, // method
					Range: lsp.Range{Start: lsp.Position{Line: 1, Character: 2}, End: lsp.Position{Line: 1, Character: 20}},
				},
			},
		},
		{
			Name:  "realClock",
			Kind:  23, // struct
			Range: lsp.Range{Start: lsp.Position{Line: 10, Character: 0}, End: lsp.Position{Line: 14, Character: 1}},
			Children: []lsp.DocumentSymbol{
				{
					Name:  "(realClock).Now",
					Kind:  6,
					Range: lsp.Range{Start: lsp.Position{Line: 11, Character: 2}, End: lsp.Position{Line: 11, Character: 20}},
				},
			},
		},
	}

	tests := []struct {
		name string
		pos  lsp.Position
		want bool
	}{
		{name: "InsideInterfaceMethod_NestedAncestorIsInterface", pos: lsp.Position{Line: 1, Character: 5}, want: true},
		{name: "InsideConcreteTypeMethod", pos: lsp.Position{Line: 11, Character: 5}, want: false},
		{name: "NoSymbolMatches", pos: lsp.Position{Line: 100, Character: 5}, want: false},
		{name: "InterfaceItselfAsInnermostMatch", pos: lsp.Position{Line: 3, Character: 0}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInterfaceDeclaration(tree, tt.pos); got != tt.want {
				t.Errorf("isInterfaceDeclaration(tree, %+v) = %v; want %v", tt.pos, got, tt.want)
			}
		})
	}
}
