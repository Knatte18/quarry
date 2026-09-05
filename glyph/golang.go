// golang.go implements the Go alphabet: the syntax parseGo checks a unit and member half against
// when Lang is Go, per docs/glyph.md §2 and §3.

package glyph

import (
	"fmt"
	"strings"
	"unicode"
)

// goKeywords holds Go's twenty-five reserved words. A member component that spells one of these is
// rejected with ReasonMemberKeyword; predeclared names such as len, string and nil are not
// keywords and are not in this set.
var goKeywords = map[string]struct{}{
	"break":       {},
	"case":        {},
	"chan":        {},
	"const":       {},
	"continue":    {},
	"default":     {},
	"defer":       {},
	"else":        {},
	"fallthrough": {},
	"for":         {},
	"func":        {},
	"go":          {},
	"goto":        {},
	"if":          {},
	"import":      {},
	"interface":   {},
	"map":         {},
	"package":     {},
	"range":       {},
	"return":      {},
	"select":      {},
	"struct":      {},
	"switch":      {},
	"type":        {},
	"var":         {},
}

// isASCIIControl reports whether r is an ASCII control character: below 0x20 or equal to 0x7f.
func isASCIIControl(r rune) bool {
	return r < 0x20 || r == 0x7f
}

// parseGo checks unit and member against the Go alphabet and, on success, returns the Glyph they
// name. input is the original whole string Parse was given, carried through so every *ParseError
// this function returns has Input set to the whole string rather than a half. parseGo validates
// the unit half first and the member half second, so a string failing both reports the unit's
// reason.
func parseGo(input, unit, member string) (Glyph, error) {
	if err := checkGoUnit(input, unit); err != nil {
		return Glyph{}, err
	}
	owner, name, err := checkGoMember(input, member)
	if err != nil {
		return Glyph{}, err
	}
	return Glyph{Lang: Go, Unit: unit, Owner: owner, Name: name, Params: nil}, nil
}

// checkGoUnit validates unit against the Go alphabet's unit rules, returning a *ParseError with
// Lang: Go and Input: input on the first rule unit fails. There is no "#" check here, because
// Parse's own count rule has already rejected any input carrying more than one "#" before unit was
// split out, and no "/" check inside a segment, because "/" is the segment separator itself.
func checkGoUnit(input, unit string) error {
	if unit == "" {
		return &ParseError{Lang: Go, Input: input, Reason: ReasonUnitEmpty}
	}

	for _, segment := range strings.Split(unit, "/") {
		if segment == "" {
			return &ParseError{Lang: Go, Input: input, Reason: ReasonUnitEmptySegment}
		}
		if segment == "." || segment == ".." {
			return &ParseError{Lang: Go, Input: input, Reason: ReasonUnitDotSegment, Detail: segment}
		}
		for _, r := range segment {
			if r == '\\' || isASCIIControl(r) || unicode.IsSpace(r) {
				return &ParseError{Lang: Go, Input: input, Reason: ReasonUnitBadRune, Detail: fmt.Sprintf("%q", r)}
			}
		}
	}

	return nil
}

// checkGoMember validates member against the Go alphabet's member rules and, on success, splits it
// into owner (the enclosing type chain, nil for a package-level name) and name (the symbol's own
// name). It returns a *ParseError with Lang: Go and Input: input on the first rule member fails.
func checkGoMember(input, member string) (owner []string, name string, err error) {
	if r, ok := firstOccurrence(member, '*'); ok {
		return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberPointer, Detail: fmt.Sprintf("%q", r)}
	}
	if r, ok := firstOccurrence(member, '(', ')'); ok {
		return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberParens, Detail: fmt.Sprintf("%q", r)}
	}
	if r, ok := firstOccurrence(member, '[', ']'); ok {
		return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberTypeParams, Detail: fmt.Sprintf("%q", r)}
	}
	if member == "" {
		return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberEmpty}
	}

	components := strings.Split(member, ".")
	for _, c := range components {
		if c == "" {
			return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberEmptyComponent}
		}
	}
	if len(components) >= 3 {
		return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberTooDeep, Detail: member}
	}

	for _, c := range components {
		// 7a must run before 7b and 7c: every rune it covers would also fail the identifier test, so
		// ReasonMemberBadRune could otherwise never fire.
		if r, ok := firstBadMemberComponentRune(c); ok {
			return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberBadRune, Detail: fmt.Sprintf("%q", r)}
		}
		if _, isKeyword := goKeywords[c]; isKeyword {
			return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberKeyword, Detail: c}
		}
		if !isGoIdentifier(c) {
			return nil, "", &ParseError{Lang: Go, Input: input, Reason: ReasonMemberNotIdentifier, Detail: c}
		}
	}

	if len(components) == 2 {
		return []string{components[0]}, components[1], nil
	}
	return nil, components[0], nil
}

// firstOccurrence scans s left to right and returns the first rune that equals any of targets.
func firstOccurrence(s string, targets ...rune) (rune, bool) {
	for _, r := range s {
		for _, t := range targets {
			if r == t {
				return r, true
			}
		}
	}
	return 0, false
}

// firstBadMemberComponentRune scans a member component left to right and returns the first rune
// that is "#", "/", an ASCII control character, or a rune satisfying unicode.IsSpace.
func firstBadMemberComponentRune(c string) (rune, bool) {
	for _, r := range c {
		if r == '#' || r == '/' || isASCIIControl(r) || unicode.IsSpace(r) {
			return r, true
		}
	}
	return 0, false
}

// isGoIdentifier reports whether c is a Go identifier: first rune "_" or unicode.IsLetter; every
// later rune "_", unicode.IsLetter or unicode.IsDigit. c is never empty when this is called.
func isGoIdentifier(c string) bool {
	for i, r := range c {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
