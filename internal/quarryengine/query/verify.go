// verify.go implements the pure, transport-free half of Callers's verification pipeline
// (callers.go): the declaration-match set, the interface/concrete classification directional mode
// requires, and the fail-closed filter predicate that decides which references survive. Nothing in
// this file touches context, a *lsp.Client, or any other form of I/O, so all of it is unit-testable
// (verify_test.go) without a fake LSP server.

package query

import (
	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
)

// symbolKindInterface is the LSP SymbolKind wire value for an interface declaration. It is named
// here rather than left as a bare literal at isInterfaceDeclaration's comparison site.
const symbolKindInterface = 11

// locationKey is a location reduced to its comparable identity: the document URI plus the 0-based
// line and UTF-16 character its range starts at. All match-set membership and lookup happens in
// this LSP wire coordinate system, before any conversion to the public 1-based Reference type —
// Reference.Character and quarryengine.Position.Character do not share a coordinate system, and
// comparing after conversion to Reference would reintroduce the byte-column hazard
// internal/cli/cli.go's own declaration-exclusion comment already documents.
type locationKey struct {
	URI       string
	Line      int
	Character int
}

// keyOf reduces loc to its locationKey, built from its URI and its range's start position.
func keyOf(loc lsp.Location) locationKey {
	return locationKey{URI: loc.URI, Line: loc.Range.Start.Line, Character: loc.Range.Start.Character}
}

// verificationOutcome is one candidate reference's per-reference textDocument/definition result.
// Attempted is false when the phase deadline expired before that reference's own definition call
// was made, distinguishing "never tried" from "tried and got nothing" — both of which
// filterVerifiedReferences keeps, but for different reasons.
type verificationOutcome struct {
	Locations []lsp.Location
	Err       error
	Attempted bool
}

// filterVerifiedReferences returns the subset of refs that survive verification against matchSet,
// given outcomes — one verificationOutcome per entry in refs, at the same index. A reference at
// index i is dropped if and only if outcomes[i].Attempted is true, outcomes[i].Err is nil,
// outcomes[i].Locations is non-empty, and none of those locations' keys is present in matchSet.
// Every other case keeps the reference: this is the fail-closed rule expressed as one predicate
// rather than several early returns, so a future edit cannot half-invert it.
func filterVerifiedReferences(refs []lsp.Location, matchSet map[locationKey]bool, outcomes []verificationOutcome) []lsp.Location {
	var kept []lsp.Location
	for i, ref := range refs {
		outcome := outcomes[i]

		disproved := outcome.Attempted && outcome.Err == nil && len(outcome.Locations) > 0 && !anyKeyIn(outcome.Locations, matchSet)
		if disproved {
			continue
		}
		kept = append(kept, ref)
	}
	return kept
}

// anyKeyIn reports whether any of locs' keys is present in matchSet.
func anyKeyIn(locs []lsp.Location, matchSet map[locationKey]bool) bool {
	for _, loc := range locs {
		if matchSet[keyOf(loc)] {
			return true
		}
	}
	return false
}

// declarationMatchSet builds the directional-mode declaration match set: every defLocs key, plus
// only those implLocs keys present in interfaceDecl. This crosses into every structurally
// unrelated satisfier's declaration only when the query started on an interface method — the
// classification isInterfaceDeclaration performs — never when it started on a concrete method,
// which is why a caller must derive interfaceDecl from the query's own implementation results
// before calling this function, not guess the direction from the query itself.
func declarationMatchSet(defLocs, implLocs []lsp.Location, interfaceDecl map[locationKey]bool) map[locationKey]bool {
	matchSet := make(map[locationKey]bool, len(defLocs)+len(implLocs))
	for _, loc := range defLocs {
		matchSet[keyOf(loc)] = true
	}
	for _, loc := range implLocs {
		key := keyOf(loc)
		if interfaceDecl[key] {
			matchSet[key] = true
		}
	}
	return matchSet
}

// isInterfaceDeclaration walks symbols' hierarchical DocumentSymbol tree, follows the chain of
// symbols whose Range contains pos, and reports whether any symbol in that chain has Kind equal to
// symbolKindInterface. pos matching no symbol, or matching only a chain with no interface ancestor,
// reports false.
func isInterfaceDeclaration(symbols []lsp.DocumentSymbol, pos lsp.Position) bool {
	for _, sym := range symbols {
		if !rangeContains(sym.Range, pos) {
			continue
		}
		if sym.Kind == symbolKindInterface {
			return true
		}
		if isInterfaceDeclaration(sym.Children, pos) {
			return true
		}
		// pos falls inside this symbol's range but not inside any child's;
		// this symbol itself is not an interface, and no sibling range can
		// also contain pos (LSP ranges within one hierarchy level do not
		// overlap), so the search is done.
		return false
	}
	return false
}

// rangeContains reports whether r's half-open [Start, End) span contains pos, comparing line first
// and character only when pos falls on Start's or End's own line.
func rangeContains(r lsp.Range, pos lsp.Position) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character >= r.End.Character {
		return false
	}
	return true
}
