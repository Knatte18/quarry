// verify.go implements the pure, transport-free half of Callers's verification pipeline
// (callers.go): the declaration-match set and the fail-closed filter predicate that decides which
// references survive. Nothing in this file touches context, a *lsp.Client, or any other form of
// I/O, so all of it is unit-testable (verify_test.go) without a fake LSP server.

package query

import (
	"strings"

	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
)

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

// declarationMatchSet builds the package-scoped declaration match set: every defLocs key, plus
// every implLocs key whose URI shares a directory with some defLocs entry. This keeps every
// same-package satisfier — concrete or interface, gopls's own Kind for it is never consulted —
// while dropping every different-package satisfier regardless of its own kind. A call through the
// interface from another package still verifies despite that exclusion: it resolves via
// textDocument/definition to the interface's own declaration position, which is a defLocs entry
// and is unconditionally included.
func declarationMatchSet(defLocs, implLocs []lsp.Location) map[locationKey]bool {
	matchSet := make(map[locationKey]bool, len(defLocs)+len(implLocs))
	defDirs := make(map[string]bool, len(defLocs))
	for _, loc := range defLocs {
		matchSet[keyOf(loc)] = true
		defDirs[uriDir(loc.URI)] = true
	}
	for _, loc := range implLocs {
		if defDirs[uriDir(loc.URI)] {
			matchSet[keyOf(loc)] = true
		}
	}
	return matchSet
}

// uriDir returns uri with its final "/"-delimited path segment (the file name) dropped. LSP
// document URIs are always forward-slash, regardless of host OS, so this is a plain string
// operation on the wire form rather than filepath.Dir — and it is deliberately a directory-identity
// comparison, not a Go import-path or package-name lookup, since declarationMatchSet only ever
// compares two URIs already known to come from the same gopls workspace.
func uriDir(uri string) string {
	i := strings.LastIndex(uri, "/")
	if i < 0 {
		return uri
	}
	return uri[:i]
}
