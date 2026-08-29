// result.go declares the per-entry status vocabulary, the reference/symbol field shapes, and the
// error classifiers every handler shares — the wire-facing counterparts of the batchStatus
// constants and classifyLookupError/classifySymbolError/classifyTOCError already defined in
// internal/cli/cli.go and internal/cli/toc.go for the CLI's own JSON envelope.

package mcpserver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Knatte18/quarry/quarry"
)

// The per-entry status vocabulary every tool's "results" array entry carries under "status".
const (
	statusFound     = "found"
	statusNotFound  = "not_found"
	statusAmbiguous = "ambiguous"
	statusError     = "error"
)

// resolutionComplete is the "resolution" field value a found entry carries when the language
// server has resolved the query exhaustively, mirroring the CLI's own "resolution":"complete"
// trust marker.
const resolutionComplete = "complete"

// referenceField is one reference's wire shape: a file and a 0-based-or-1-based line/character
// pair, depending on which converter built it. Its JSON tags match referenceFields in
// internal/cli/cli.go exactly, so an LSP-mirrored tool's per-entry reference shape is identical to
// the CLI's own.
type referenceField struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
}

// symbolField is one workspace/symbol match's wire shape. Kind is an int, not a string: it is the
// numeric LSP SymbolKind quarry.SymbolMatch.Kind carries unchanged, and symbolMatchFields in
// internal/cli/cli.go emits it the same way — typing it string here would silently change the JSON
// type the CLI emits for the identical query. Its JSON tags match symbolMatchFields exactly.
type symbolField struct {
	Name      string `json:"name"`
	Kind      int    `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
}

// referenceFieldsWire converts refs to the wire shape an LSP-mirrored tool declares on both axes,
// applying toZeroBased to both Line and Character so the result leaves in the 0-based convention
// those tools use. The returned slice is always non-nil, even for an empty input, so it marshals as
// "[]" rather than "null".
func referenceFieldsWire(refs []quarry.Reference) []referenceField {
	fields := make([]referenceField, len(refs))
	for i, r := range refs {
		fields[i] = referenceField{File: r.File, Line: toZeroBased(r.Line), Character: toZeroBased(r.Character)}
	}
	return fields
}

// referenceFieldsNative converts refs to the wire shape a quarry-native tool declares: identical
// to referenceFieldsWire except no conversion is applied, since quarry-native tools stay in
// quarry.Reference's own 1-based convention. The returned slice is always non-nil, even for an
// empty input.
func referenceFieldsNative(refs []quarry.Reference) []referenceField {
	fields := make([]referenceField, len(refs))
	for i, r := range refs {
		fields[i] = referenceField{File: r.File, Line: r.Line, Character: r.Character}
	}
	return fields
}

// symbolFieldsWire converts matches to the wire shape an LSP-mirrored tool declares, applying
// toZeroBased to both Line and Character exactly as referenceFieldsWire does. The returned slice is
// always non-nil, even for an empty input.
func symbolFieldsWire(matches []quarry.SymbolMatch) []symbolField {
	fields := make([]symbolField, len(matches))
	for i, m := range matches {
		fields[i] = symbolField{Name: m.Name, Kind: m.Kind, File: m.File, Line: toZeroBased(m.Line), Character: toZeroBased(m.Character)}
	}
	return fields
}

// classifyLSPError maps a References/Definition/Callers-family error to a status, an optional
// candidate list, and an optional message, implementing the exact branches
// classifyLookupError (internal/cli/cli.go) uses, nil branch included.
//
// A nil err yields statusFound with no candidates and no message — stated explicitly so a caller
// that hands classifyLSPError a nil error never falls through to the else branch and dereferences
// it.
func classifyLSPError(err error) (status string, candidates []string, message string) {
	if err == nil {
		return statusFound, nil, ""
	}

	var ambiguous *quarry.ErrAmbiguousSymbol
	if errors.As(err, &ambiguous) {
		return statusAmbiguous, ambiguous.Candidates, ""
	}

	if errors.Is(err, quarry.ErrSymbolNotFoundSentinel) {
		return statusNotFound, nil, ""
	}

	return statusError, nil, err.Error()
}

// classifySymbolError maps a Symbol outcome to a status and an optional message, implementing
// classifySymbolError's own two predicates from internal/cli/cli.go — no ambiguous branch, unlike
// classifyLSPError. symbolFromClient (internal/quarryengine/query/symbol.go) deliberately returns
// every candidate rather than collapsing multiple matches to quarry.ErrAmbiguousSymbol, so
// quarry.Symbol never produces that error; an ambiguous branch here would be dead code that also
// forces a "candidates" key onto a tool whose CLI counterpart cannot emit one.
func classifySymbolError(err error) (status string, message string) {
	if err == nil {
		return statusFound, ""
	}
	if errors.Is(err, quarry.ErrSymbolNotFoundSentinel) {
		return statusNotFound, ""
	}
	return statusError, err.Error()
}

// classifyTOCError maps a toc outcome to a status and a message, implementing toc's own rule
// (internal/cli/toc.go's classifyTOCError) rather than borrowing classifyLSPError's LSP
// predicates: toc uses no language server, so applying those predicates here would report "error"
// where the CLI reports "not_found" for a missing file. quarry.ErrLanguageUnsupported yields a
// message worded from quarry.TOCImplemented() exactly as internal/cli/toc.go words it; anything
// else yields err.Error() unchanged.
func classifyTOCError(err error) (status string, message string) {
	if errors.Is(err, quarry.ErrLanguageUnsupported) {
		return statusError, fmt.Sprintf("toc: language not yet supported; quarry can currently read: %s", strings.Join(quarry.TOCImplemented(), ", "))
	}
	return statusError, err.Error()
}

// rewordMarshalFailure reworks a StructToFields-style failure (which carries a literal "toc: "
// prefix, since internal/cli.StructToFields was written for the toc verbs) into an
// impact-specific message, mirroring rewordImpactMarshalFailure in internal/cli/impact.go.
func rewordMarshalFailure(err error) string {
	return fmt.Sprintf("impact: %s", strings.TrimPrefix(err.Error(), "toc: "))
}
