// name.go implements the glyph maker: predicting the id and kind a declaration head will get once
// it is actually written, by parsing it through the same Go Strategy the walk itself uses. It
// declares the maker's input shape (Declaration), its answer shape (NameResult), its closed reason
// vocabulary, and the exported batch entry point, Name. Nothing here writes to disk, opens a
// repository, or takes a root — see the plan's "the maker owns no naming logic" and "nothing is
// written to disk" Shared Decisions.

package engine

import (
	"errors"
	"fmt"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// Declaration is one input to Name: a unit the declaration will belong to, and the declaration head
// itself, verbatim. It carries no language field — the maker parses as Go and builds glyph.Go
// glyphs, matching resolveGlyphTarget, which already hardcodes the alphabet.
type Declaration struct {
	// Unit is the glyph unit the declaration will belong to.
	Unit string
	// Decl is the declaration head, verbatim.
	Decl string
}

// NameResult is Name's answer to one Declaration, positionally. Unit and Target always echo the
// input's Unit and Decl verbatim. ID and Kind are set only on success; Error and Reason only on
// failure. There is no "ok" key: the failure envelope's own presence is what a caller reads,
// so it can never disagree with an exit code beside it.
type NameResult struct {
	// Unit echoes the input Declaration.Unit verbatim. Always present.
	Unit string `json:"unit"`
	// Target echoes the input Declaration.Decl verbatim. Always present.
	Target string `json:"target"`
	// ID is the predicted glyph id, set only on success.
	ID string `json:"id,omitempty"`
	// Kind is the predicted declaration kind, set only on success.
	Kind Kind `json:"kind,omitempty"`
	// Error is a one-sentence failure message, set only on failure.
	Error string `json:"error,omitempty"`
	// Reason is the plain-word form of the failure, set only on failure. It is a plain string, not
	// a defined type, matching the same field on ResolveResult.
	Reason string `json:"reason,omitempty"`
}

// The four reason words the maker itself produces. A glyph.Reason word is also propagated verbatim
// through NameResult.Reason when the extracted id fails glyph.Parse — that word is not one of these
// four and carries no constant of its own here, since glyph.Reasons already enumerates it.
const (
	// NameReasonParse marks a declaration head that still does not parse after the completion retry.
	NameReasonParse = "parse"
	// NameReasonNoDeclaration marks a declaration head that declares no symbol.
	NameReasonNoDeclaration = "no_declaration"
	// NameReasonSeveralDeclarations marks a declaration head that declares more than one symbol.
	NameReasonSeveralDeclarations = "several_declarations"
	// NameReasonInternal marks a failure with no cause in the caller's own input: an unwired
	// grammar, a tree-sitter error, or an extractor invariant violated.
	NameReasonInternal = "internal"
)

// NameReasons lists all four maker-owned reason values, in the same order as the constant block
// above. Go cannot reflect over package-level constants, so this slice is the only way a test can
// enumerate the vocabulary. Adding a constant means adding it here in the same edit, exactly as
// glyph.Reasons does.
var NameReasons = []string{
	NameReasonParse,
	NameReasonNoDeclaration,
	NameReasonSeveralDeclarations,
	NameReasonInternal,
}

// Name predicts the id and kind for every declaration in decls, positionally: the returned slice
// has exactly len(decls) elements, and element i answers decls[i]. It allocates with
// make([]NameResult, 0, len(decls)) so an empty input returns an empty, non-nil slice.
//
// Name returns no error at all: with no I/O there is nothing that can fail batch-wide, and every
// failure mode is a property of one entry's own unit or fragment, carried in that entry's own
// NameResult.
func Name(decls []Declaration) []NameResult {
	results := make([]NameResult, 0, len(decls))
	for _, d := range decls {
		results = append(results, nameOne(d))
	}
	return results
}

// nameFailure builds the failure NameResult for d, echoing Unit and Target and setting Reason and
// Error. It exists so that echo is written once rather than at each of nameOne's failure sites.
func nameFailure(d Declaration, reason, errMsg string) NameResult {
	return NameResult{Unit: d.Unit, Target: d.Decl, Reason: reason, Error: errMsg}
}

// nameExtract wraps d's synthetic src in treesitter.WithTree("go", ...), records the partial flag,
// and — only when partial is false — appends strategy.Symbols(unit, root, src) to a slice it copies
// out. The callback never retains root; Symbol is a value struct of strings and ints, so copying
// the slice out is safe. A non-nil error from WithTree is returned unchanged.
func nameExtract(strategy Strategy, unit string, src []byte) (syms []Symbol, partial bool, err error) {
	err = treesitter.WithTree("go", src, func(root *ts.Node, isPartial bool) error {
		partial = isPartial
		if !partial {
			syms = append(syms, strategy.Symbols(unit, root, src)...)
		}
		return nil
	})
	return syms, partial, err
}

// nameOne predicts the id and kind for one Declaration, in this fixed order: look up the Go
// strategy, build the synthetic source, extract, retry once on a partial parse, count the
// resulting symbols, and validate the single symbol's id by round-tripping it through glyph.Parse.
func nameOne(d Declaration) NameResult {
	strategy, ok := StrategyFor("go")
	if !ok {
		err := fmt.Errorf("engine: no Strategy registered for language %q", "go")
		return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
	}

	src := []byte("package q\n\n" + d.Decl + "\n")
	syms, partial, err := nameExtract(strategy, d.Unit, src)
	if err != nil {
		return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
	}

	if partial {
		retrySrc := []byte("package q\n\n" + d.Decl + " {}" + "\n")
		syms, partial, err = nameExtract(strategy, d.Unit, retrySrc)
		if err != nil {
			return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
		}
		if partial {
			return nameFailure(d, NameReasonParse, "declaration does not parse")
		}
	}

	switch len(syms) {
	case 0:
		return nameFailure(d, NameReasonNoDeclaration, "declaration declares no symbol")
	case 1:
		// Falls through to the id round trip below.
	default:
		return nameFailure(d, NameReasonSeveralDeclarations,
			fmt.Sprintf("declaration declares %d symbols; exactly one is required", len(syms)))
	}

	sym := syms[0]
	parsed, err := glyph.Parse(glyph.Go, sym.ID)
	if err != nil {
		reason := ""
		var parseErr *glyph.ParseError
		if errors.As(err, &parseErr) {
			reason = string(parseErr.Reason)
		}
		return nameFailure(d, reason, err.Error())
	}
	if roundTripped := parsed.String(); roundTripped != sym.ID {
		internalErr := fmt.Errorf("engine: id round trip mismatch: %q parsed and printed back as %q", sym.ID, roundTripped)
		return nameFailure(d, NameReasonInternal, "internal error: "+internalErr.Error())
	}

	return NameResult{Unit: d.Unit, Target: d.Decl, ID: sym.ID, Kind: sym.Kind}
}
