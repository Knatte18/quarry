// name.go implements the glyph maker: predicting the id and kind a declaration head will get once
// it is actually written, by parsing it, in the declaration's language, through the same Strategy the walk itself uses. It
// declares the maker's input shape (Declaration), its answer shape (NameResult), its closed reason
// vocabulary, and the exported batch entry point, Name. Nothing here writes to disk, opens a
// repository, or takes a root — see the plan's "the maker owns no naming logic" and "nothing is
// written to disk" Shared Decisions.

package engine

import (
	"errors"
	"fmt"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// Declaration is one input to Name: the language the declaration is written in, a unit it will belong to, and the declaration head
// itself, verbatim.
type Declaration struct {
	// Lang is the language the declaration is written in, glyph.Go or glyph.CSharp.
	// The zero value is rejected rather than defaulted.
	Lang glyph.Language
	// Unit is the glyph unit the declaration will belong to.
	Unit string
	// Owner is the dotted owner chain of a C# member in glyph spelling, such as "Outer.Box<>".
	// A Go declaration carries no owner: a Go method states its receiver in the head.
	Owner string
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

// The reason words the maker itself produces.
// A glyph.Reason word is also propagated verbatim through NameResult.Reason.
// That happens when the extracted id fails glyph.Parse or the declaration's language is unsupported.
// Those words carry no constant of their own here, since glyph.Reasons already enumerates them.
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
	// NameReasonOwnerNotAllowed marks a Go declaration that carries an owner.
	NameReasonOwnerNotAllowed = "owner_not_allowed"
	// NameReasonInvalidContainer marks a C# declaration whose unit or owner is not a valid container.
	// It also marks a head that mints an id outside the requested one.
	NameReasonInvalidContainer = "invalid_container"
)

// NameReasons lists all maker-owned reason values, in the same order as the constant block
// above. Go cannot reflect over package-level constants, so this slice is the only way a test can
// enumerate the vocabulary. Adding a constant means adding it here in the same edit, exactly as
// glyph.Reasons does.
var NameReasons = []string{
	NameReasonParse,
	NameReasonNoDeclaration,
	NameReasonSeveralDeclarations,
	NameReasonInternal,
	NameReasonOwnerNotAllowed,
	NameReasonInvalidContainer,
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
	verifyNameCoverage(decls, results)
	return results
}

// verifyNameCoverage guards Name's positional contract: Name answers every declaration
// positionally, with Unit and Target echoing the input's Unit and Decl verbatim on every result —
// failures included, which is exactly what nameFailure exists to guarantee. It panics rather than
// returning an error, and here there is an extra reason beyond the unreachable-by-construction
// argument verifyResolveCoverage documents: Name returns no error at all by design, so an error
// return is not available without a breaking signature change. Unlike resolve, Name has no
// whole-call failure path to exclude — it has exactly one return statement — so the call below is
// unconditional.
//
// The per-index check compares Unit first and only reaches the Target comparison in the else
// branch, when Unit matched. That ordering, and the field name each message carries, are
// load-bearing: without the ordering a result diverging in both fields would have two equally
// valid messages and the choice between them would be arbitrary, and without the field label a
// Target-only divergence would print a bare got/want pair with nothing saying which field it
// described.
func verifyNameCoverage(decls []Declaration, results []NameResult) {
	if len(results) != len(decls) {
		panic(fmt.Sprintf("engine: name returned %d results for %d declarations", len(results), len(decls)))
	}
	for i, d := range decls {
		if results[i].Unit != d.Unit {
			panic(fmt.Sprintf("engine: name result %d echoes unit %q; want %q", i, results[i].Unit, d.Unit))
		} else if results[i].Target != d.Decl {
			panic(fmt.Sprintf("engine: name result %d echoes target %q; want %q", i, results[i].Target, d.Decl))
		}
	}
}

// nameFailure builds the failure NameResult for d, echoing Unit and Target and setting Reason and
// Error. It exists so that echo is written once rather than at each of nameOne's failure sites.
func nameFailure(d Declaration, reason, errMsg string) NameResult {
	return NameResult{Unit: d.Unit, Target: d.Decl, Reason: reason, Error: errMsg}
}

// nameExtract wraps d's synthetic src in treesitter.WithTree(lang, ...) and records the partial flag.
// Only when partial is false, it copies out what strategy.Symbols(unit, root, src) returns.
// That is the symbols and the ids the strategy dropped.
// The callback never retains root; Symbol is a value struct of strings and ints, so copying the slice out is safe.
// A non-nil error from WithTree is returned unchanged.
func nameExtract(strategy Strategy, lang, unit string, src []byte) (syms []Symbol, dropped []DroppedSymbol, partial bool, err error) {
	err = treesitter.WithTree(lang, src, func(root *ts.Node, isPartial bool) error {
		partial = isPartial
		if !partial {
			minted, droppedIDs := strategy.Symbols(unit, root, src)
			syms = append(syms, minted...)
			dropped = append(dropped, droppedIDs...)
		}
		return nil
	})
	return syms, dropped, partial, err
}

// nameLanguageFailure builds the unsupported_language failure for d.
// Its error names the language given, or that none was, and the values the maker accepts.
func nameLanguageFailure(d Declaration) NameResult {
	reason := string(glyph.ReasonUnsupportedLanguage)
	if d.Lang == "" {
		return nameFailure(d, reason, `language is unset; set Lang to "go" or "csharp"`)
	}
	return nameFailure(d, reason, fmt.Sprintf(`language %q is not supported; set Lang to "go" or "csharp"`, string(d.Lang)))
}

// nameOne predicts the id and kind for one Declaration.
// Its fixed order is:
// gate the language and owner, look up the Go strategy, build the synthetic source, extract, and retry once on a partial parse.
// It then counts the resulting symbols and validates the single symbol's id by round-tripping it through glyph.Parse.
// The gate runs before any source is synthesised.
// A language other than glyph.Go, the zero value included, is never defaulted.
// A Go declaration with an owner is rejected, since a Go method carries its receiver in the head.
func nameOne(d Declaration) NameResult {
	if d.Lang == glyph.CSharp {
		return nameCSharp(d)
	}
	if d.Lang != glyph.Go {
		return nameLanguageFailure(d)
	}
	if d.Owner != "" {
		return nameFailure(d, NameReasonOwnerNotAllowed,
			`a Go declaration takes no owner: leave Owner empty, since a Go method carries its receiver in the head; for a C# declaration, set Lang to "csharp" (CLI --lang csharp)`)
	}

	strategy, ok := StrategyFor("go")
	if !ok {
		err := fmt.Errorf("engine: no Strategy registered for language %q", "go")
		return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
	}

	src := []byte("package q\n\n" + d.Decl + "\n")
	syms, _, partial, err := nameExtract(strategy, "go", d.Unit, src)
	if err != nil {
		return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
	}

	if partial {
		retrySrc := []byte("package q\n\n" + d.Decl + " {}" + "\n")
		syms, _, partial, err = nameExtract(strategy, "go", d.Unit, retrySrc)
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

// nameCSharp predicts the id and kind for one C# Declaration.
// It validates the unit and owner chain and synthesises namespace and class shells around the head.
// It then extracts with the C# strategy and answers the one symbol the head declares.
// A head that does not parse whole is completed with ";" and then with " {}" before it is called unparseable.
// The minted id must sit in the requested unit and owner chain, so a head cannot mint outside its container.
func nameCSharp(d Declaration) NameResult {
	owner, err := nameCSharpContainer(d)
	if err != nil {
		return nameFailure(d, NameReasonInvalidContainer, err.Error())
	}

	strategy, ok := StrategyFor("csharp")
	if !ok {
		err := fmt.Errorf("engine: no Strategy registered for language %q", "csharp")
		return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
	}

	var syms []Symbol
	var dropped []DroppedSymbol
	var headStart, headEnd, completedEnd int
	partial := true
	for _, completion := range []string{"", ";", " {}"} {
		var src []byte
		src, headStart, headEnd = csharpShellSource(d.Unit, owner, d.Decl, completion)
		completedEnd = headEnd + len(completion)
		syms, dropped, partial, err = nameExtract(strategy, "csharp", d.Unit, src)
		if err != nil {
			return nameFailure(d, NameReasonInternal, "internal error: "+err.Error())
		}
		if !partial {
			break
		}
	}
	if partial {
		return nameFailure(d, NameReasonParse, nameOwnerHint(d, "declaration does not parse"))
	}

	for _, drop := range dropped {
		if drop.DeclStart < headStart || drop.DeclStart >= headEnd {
			continue
		}
		reason := ""
		var parseErr *glyph.ParseError
		if errors.As(drop.Err, &parseErr) {
			reason = string(parseErr.Reason)
		}
		return nameFailure(d, reason, drop.Err.Error())
	}

	var kept []Symbol
	for _, sym := range syms {
		if sym.DeclStart >= headStart && sym.DeclStart < headEnd && sym.DeclEnd <= completedEnd {
			kept = append(kept, sym)
		}
	}
	if len(kept) == 0 {
		return nameFailure(d, NameReasonNoDeclaration, nameOwnerHint(d, "declaration declares no symbol"))
	}
	sym, ok := outermostSymbol(kept)
	if !ok {
		return nameFailure(d, NameReasonSeveralDeclarations,
			fmt.Sprintf("declaration declares %d symbols; exactly one is required", len(kept)))
	}

	if err := checkMintedContainer(d, owner, sym); err != nil {
		return nameFailure(d, NameReasonInvalidContainer, err.Error())
	}

	parsed, err := glyph.Parse(glyph.CSharp, sym.ID)
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

// nameOwnerHint appends to message that a C# member head needs an owner, when d carries none.
func nameOwnerHint(d Declaration, message string) string {
	if d.Owner != "" {
		return message
	}
	return message + "; a C# member head needs Owner (CLI --owner)"
}

// nameCSharpContainer validates d's unit and owner chain and returns the owner's components.
// Each component must be a C# identifier with an optional arity suffix and no type arguments.
// That keeps braces, spaces and type arguments out of the synthesised shells' names.
func nameCSharpContainer(d Declaration) ([]string, error) {
	if _, err := glyph.Parse(glyph.CSharp, d.Unit+"#"); err != nil {
		return nil, fmt.Errorf("unit %q is not a C# unit: %w", d.Unit, err)
	}
	if d.Owner == "" {
		return nil, nil
	}
	components := splitOutsideAngles(d.Owner)
	for _, component := range components {
		_, suffix, generic := strings.Cut(component, "<")
		if generic && strings.Trim(strings.TrimSuffix(suffix, ">"), ",") != "" {
			return nil, fmt.Errorf("owner component %q carries type arguments; use an arity suffix such as Box<> or Dict<,>", component)
		}
		parsed, err := glyph.Parse(glyph.CSharp, "global#"+component+".m")
		if err != nil || len(parsed.Owner) != 1 || parsed.Owner[0] != component {
			return nil, fmt.Errorf("owner component %q is not a C# identifier with an optional arity suffix", component)
		}
	}
	return components, nil
}

// splitOutsideAngles splits s on "." outside "<>".
func splitOutsideAngles(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			depth--
		case '.':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// outermostSymbol returns the symbol whose span strictly contains every other symbol's span.
// It reports false when no symbol does, which includes two symbols sharing one span.
func outermostSymbol(symbols []Symbol) (Symbol, bool) {
	for i, candidate := range symbols {
		contains := true
		for j, other := range symbols {
			if i == j {
				continue
			}
			inside := candidate.DeclStart <= other.DeclStart && other.DeclEnd <= candidate.DeclEnd
			sameSpan := candidate.DeclStart == other.DeclStart && candidate.DeclEnd == other.DeclEnd
			if !inside || sameSpan {
				contains = false
				break
			}
		}
		if contains {
			return candidate, true
		}
	}
	return Symbol{}, false
}

// checkMintedContainer reports an error when sym's unit is not d's unit.
// It also reports one when sym's owner chain is not owner followed by the explicit-interface qualifier the head itself spells.
func checkMintedContainer(d Declaration, owner []string, sym Symbol) error {
	mismatch := fmt.Errorf("declaration mints %s outside the requested container %s#%s",
		sym.ID, d.Unit, strings.Join(owner, "."))
	if sym.Glyph.Unit != d.Unit || len(sym.Glyph.Owner) < len(owner) {
		return mismatch
	}
	for i, component := range owner {
		if sym.Glyph.Owner[i] != component {
			return mismatch
		}
	}
	if extra := sym.Glyph.Owner[len(owner):]; len(extra) > 0 {
		if sym.Kind == KindType || !strings.Contains(d.Decl, strings.Join(extra, ".")+".") {
			return mismatch
		}
	}
	return nil
}

// csharpShellSource builds the source the maker parses a C# head inside:
// a namespace shell from unit (none for "global") and one class shell per owner component.
// The head and then the completion are spliced into the innermost body.
// An arity suffix on an owner component becomes a declared type-parameter list.
// It returns the head's byte range in src, completion excluded.
func csharpShellSource(unit string, owner []string, head, completion string) (src []byte, headStart, headEnd int) {
	var b strings.Builder
	closers := 0
	if unit != "global" {
		b.WriteString("namespace " + unit + " {\n")
		closers++
	}
	for _, component := range owner {
		name, suffix, generic := strings.Cut(component, "<")
		b.WriteString("class " + name)
		if generic {
			parameters := make([]string, strings.Count(suffix, ",")+1)
			for i := range parameters {
				parameters[i] = fmt.Sprintf("T%d", i+1)
			}
			b.WriteString("<" + strings.Join(parameters, ",") + ">")
		}
		b.WriteString(" {\n")
		closers++
	}
	headStart = b.Len()
	b.WriteString(head)
	headEnd = b.Len()
	b.WriteString(completion + "\n")
	b.WriteString(strings.Repeat("}\n", closers))
	return []byte(b.String()), headStart, headEnd
}
