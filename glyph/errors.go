// errors.go declares the closed Reason vocabulary a ParseError's Reason field is drawn from, and
// ParseError itself.

package glyph

import "fmt"

// Reason is the closed vocabulary a ParseError's Reason field is drawn from. No value outside the
// constant block below is ever produced.
type Reason string

// The Reason values a ParseError ever carries. No constant outside this block is produced.
const (
	// ReasonUnsupportedLanguage fires when Parse is called with a Language it does not implement.
	ReasonUnsupportedLanguage Reason = "unsupported_language"
	// ReasonInvalidUTF8 fires when the input is not valid UTF-8.
	ReasonInvalidUTF8 Reason = "invalid_utf8"
	// ReasonNoSeparator fires when the input carries no "#" at all.
	ReasonNoSeparator Reason = "no_separator"
	// ReasonMultipleSeparators fires when the input carries more than one "#".
	ReasonMultipleSeparators Reason = "multiple_separators"
	// ReasonUnitEmpty fires when the unit half is the empty string.
	ReasonUnitEmpty Reason = "unit_empty"
	// ReasonUnitEmptySegment fires when a "/"-separated segment of the unit is empty.
	ReasonUnitEmptySegment Reason = "unit_empty_segment"
	// ReasonUnitDotSegment fires when a segment of the unit is "." or "..".
	ReasonUnitDotSegment Reason = "unit_dot_segment"
	// ReasonUnitBadRune fires when the unit contains a rune the language's alphabet does not allow.
	ReasonUnitBadRune Reason = "unit_bad_rune"
	// ReasonMemberEmptyComponent fires when a "."-separated component of the member is empty.
	ReasonMemberEmptyComponent Reason = "member_empty_component"
	// ReasonMemberTooDeep fires when the member has more "."-separated components than the
	// language's alphabet allows.
	ReasonMemberTooDeep Reason = "member_too_deep"
	// ReasonMemberNotIdentifier fires when a component of the member is not a valid identifier in
	// the language.
	ReasonMemberNotIdentifier Reason = "member_not_identifier"
	// ReasonMemberKeyword fires when a component of the member is a reserved keyword.
	ReasonMemberKeyword Reason = "member_keyword"
	// ReasonMemberTypeParams fires when a component of the member carries type parameters, which
	// are not part of a glyph.
	ReasonMemberTypeParams Reason = "member_type_params"
	// ReasonMemberParens fires when the member carries parentheses in a language whose alphabet
	// does not use them.
	ReasonMemberParens Reason = "member_parens"
	// ReasonMemberPointer fires when the member encodes a receiver's pointer-ness, which is not
	// part of a glyph.
	ReasonMemberPointer Reason = "member_pointer"
	// ReasonMemberBadRune fires when the member contains a rune the language's alphabet does not
	// allow.
	ReasonMemberBadRune Reason = "member_bad_rune"
	// ReasonUnitNotIdentifier fires when a "."-separated segment of a C# unit is not an identifier.
	ReasonUnitNotIdentifier Reason = "unit_not_identifier"
	// ReasonNeedlessEscape fires when a C# identifier carries a "@" in front of a name that is not a reserved keyword.
	ReasonNeedlessEscape Reason = "needless_escape"
	// ReasonNonCanonicalType fires when a C# type or parameter is not in its canonical spelling.
	// The Detail is the canonical spelling.
	ReasonNonCanonicalType Reason = "non_canonical_type"
	// ReasonUnbalancedBrackets fires when a C# member's "<>", "()" or "[]" do not close in order.
	ReasonUnbalancedBrackets Reason = "unbalanced_brackets"
	// ReasonIndexerBrackets fires when a C# indexer lacks its bracketed list, or square brackets follow any other name.
	ReasonIndexerBrackets Reason = "indexer_brackets"
	// ReasonMissingParens fires when a C# finalizer, static constructor or operator has no parameter list.
	ReasonMissingParens Reason = "missing_parens"
	// ReasonUnknownOperator fires when the token after "operator" is not an overloadable C# operator.
	ReasonUnknownOperator Reason = "unknown_operator"
)

// Reasons lists every Reason value, in the same order as the constant block above.
// Go cannot reflect over package-level constants, so this slice is the only way a test or an exhaustive caller can enumerate the vocabulary.
// Adding a constant means adding it here in the same edit.
var Reasons = []Reason{
	ReasonUnsupportedLanguage,
	ReasonInvalidUTF8,
	ReasonNoSeparator,
	ReasonMultipleSeparators,
	ReasonUnitEmpty,
	ReasonUnitEmptySegment,
	ReasonUnitDotSegment,
	ReasonUnitBadRune,
	ReasonMemberEmptyComponent,
	ReasonMemberTooDeep,
	ReasonMemberNotIdentifier,
	ReasonMemberKeyword,
	ReasonMemberTypeParams,
	ReasonMemberParens,
	ReasonMemberPointer,
	ReasonMemberBadRune,
	ReasonUnitNotIdentifier,
	ReasonNeedlessEscape,
	ReasonNonCanonicalType,
	ReasonUnbalancedBrackets,
	ReasonIndexerBrackets,
	ReasonMissingParens,
	ReasonUnknownOperator,
}

// reasonText gives each Reason a short phrase naming what was wrong. Every phrase differs from
// every other.
var reasonText = map[Reason]string{
	ReasonUnsupportedLanguage:  "this language is not implemented",
	ReasonInvalidUTF8:          "input is not valid UTF-8",
	ReasonNoSeparator:          "a glyph needs a \"#\"; a path is addressed as its own glyph by appending one to its repository-relative form",
	ReasonMultipleSeparators:   "a glyph has exactly one \"#\"; a unit or member component may not contain one",
	ReasonUnitEmpty:            "unit is empty",
	ReasonUnitEmptySegment:     "unit has an empty segment",
	ReasonUnitDotSegment:       "unit has a \".\" or \"..\" segment",
	ReasonUnitBadRune:          "unit contains a rune the language's alphabet does not allow",
	ReasonMemberEmptyComponent: "member has an empty component",
	ReasonMemberTooDeep:        "member has more components than the language allows",
	ReasonMemberNotIdentifier:  "member component is not a valid identifier",
	ReasonMemberKeyword:        "member component is a reserved keyword",
	ReasonMemberTypeParams:     "member component carries type parameters, which are not part of a glyph",
	ReasonMemberParens:         "member carries parentheses this language's alphabet does not use",
	ReasonMemberPointer:        "member encodes a receiver's pointer-ness, which is not part of a glyph",
	ReasonMemberBadRune:        "member contains a rune the language's alphabet does not allow",
	ReasonUnitNotIdentifier:    "unit segment is not a valid identifier",
	ReasonNeedlessEscape:       "identifier carries a \"@\" it does not need",
	ReasonNonCanonicalType:     "type is not in its canonical spelling",
	ReasonUnbalancedBrackets:   "member has brackets that do not close in order",
	ReasonIndexerBrackets:      "an indexer is spelled this[...] and no other name takes square brackets",
	ReasonMissingParens:        "this member needs a parameter list in parentheses",
	ReasonUnknownOperator:      "the token after \"operator\" is not an overloadable operator",
}

// ParseError reports why Parse rejected an input. Callers use errors.As and switch on Reason.
// Detail carries the offending segment, component or rune where one exists and is empty
// otherwise — a blank Detail carries no meaning and is never a discriminator. It also carries a
// suggested spelling for no_separator and the whole input for multiple_separators.
type ParseError struct {
	// Lang is the language Parse was asked to check the input against.
	Lang Language
	// Input is the exact string Parse was given.
	Input string
	// Reason is the closed vocabulary value naming why Parse rejected Input.
	Reason Reason
	// Detail carries the offending segment, component or rune where one exists, empty otherwise.
	// It also carries a suggested spelling for no_separator and the whole input for
	// multiple_separators.
	Detail string
}

// Error implements the error interface, composing a complete message from Reason, Lang and Input
// alone, appending Detail in parentheses only when it is non-empty, and falling back to the raw
// Reason string when reasonText has no entry.
func (e *ParseError) Error() string {
	text, ok := reasonText[e.Reason]
	if !ok {
		text = string(e.Reason)
	}
	msg := fmt.Sprintf("glyph: parse %q as %s: %s", e.Input, e.Lang, text)
	if e.Detail != "" {
		msg += fmt.Sprintf(" (%s)", e.Detail)
	}
	return msg
}
