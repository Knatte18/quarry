// csharp.go implements the C# alphabet.
// It holds the syntax parseCSharp checks a unit and member half against when Lang is CSharp, per docs/glyph.md §2 and §3.

package glyph

import (
	"fmt"
	"strings"
	"unicode"
)

// csharpKeywords holds C#'s reserved keywords.
// A member component that spells one of these is rejected with ReasonMemberKeyword unless it carries the verbatim "@" prefix.
// The "@" is admitted only in front of one of these.
var csharpKeywords = map[string]struct{}{
	"abstract": {}, "as": {}, "base": {}, "bool": {}, "break": {}, "byte": {},
	"case": {}, "catch": {}, "char": {}, "checked": {}, "class": {}, "const": {},
	"continue": {}, "decimal": {}, "default": {}, "delegate": {}, "do": {}, "double": {},
	"else": {}, "enum": {}, "event": {}, "explicit": {}, "extern": {}, "false": {},
	"finally": {}, "fixed": {}, "float": {}, "for": {}, "foreach": {}, "goto": {},
	"if": {}, "implicit": {}, "in": {}, "int": {}, "interface": {}, "internal": {},
	"is": {}, "lock": {}, "long": {}, "namespace": {}, "new": {}, "null": {},
	"object": {}, "operator": {}, "out": {}, "override": {}, "params": {}, "private": {},
	"protected": {}, "public": {}, "readonly": {}, "ref": {}, "return": {}, "sbyte": {},
	"sealed": {}, "short": {}, "sizeof": {}, "stackalloc": {}, "static": {}, "string": {},
	"struct": {}, "switch": {}, "this": {}, "throw": {}, "true": {}, "try": {},
	"typeof": {}, "uint": {}, "ulong": {}, "unchecked": {}, "unsafe": {}, "ushort": {},
	"using": {}, "virtual": {}, "void": {}, "volatile": {}, "while": {},
}

// csharpOperators is the set of tokens C# lets a type overload, as written after "operator ".
var csharpOperators = map[string]struct{}{
	"+": {}, "-": {}, "!": {}, "~": {}, "++": {}, "--": {}, "true": {}, "false": {},
	"*": {}, "/": {}, "%": {}, "&": {}, "|": {}, "^": {}, "<<": {}, ">>": {}, ">>>": {},
	"==": {}, "!=": {}, "<": {}, ">": {}, "<=": {}, ">=": {},
}

// csharpCheckedOperators is the subset of csharpOperators that has a "checked" form.
var csharpCheckedOperators = map[string]struct{}{
	"+": {}, "-": {}, "*": {}, "/": {}, "++": {}, "--": {},
}

// csharpParamModifiers is the set of words a parameter spelling may carry before its type.
var csharpParamModifiers = map[string]struct{}{
	"ref": {}, "out": {}, "in": {}, "params": {}, "this": {}, "scoped": {}, "readonly": {},
}

// csharpConversionPrefixes are the two spellings a conversion operator's name starts with.
var csharpConversionPrefixes = []string{"implicit operator ", "explicit operator "}

// parseCSharp checks unit and member against the C# alphabet and, on success, returns the Glyph they name.
// input is the original whole string, carried through so every *ParseError has Input set to it.
// The unit is validated first and the member second, so a string failing both reports the unit's reason.
// An empty member is the self form.
func parseCSharp(input, unit, member string) (Glyph, error) {
	if err := checkCSharpUnit(input, unit); err != nil {
		return Glyph{}, err
	}
	if member == "" {
		return Glyph{Lang: CSharp, Unit: unit}, nil
	}
	owner, name, params, err := checkCSharpMember(input, member)
	if err != nil {
		return Glyph{}, err
	}
	return Glyph{Lang: CSharp, Unit: unit, Owner: owner, Name: name, Params: params}, nil
}

// csharpError builds the *ParseError every C# check returns.
func csharpError(input string, reason Reason, detail string) error {
	return &ParseError{Lang: CSharp, Input: input, Reason: reason, Detail: detail}
}

// checkCSharpUnit validates unit as "global" or identifiers joined by ".".
func checkCSharpUnit(input, unit string) error {
	if unit == "" {
		return csharpError(input, ReasonUnitEmpty, "")
	}
	for _, r := range unit {
		if r == '/' || r == ':' {
			return csharpError(input, ReasonUnitBadRune, fmt.Sprintf("%q", r))
		}
	}
	for _, segment := range strings.Split(unit, ".") {
		if segment == "" {
			return csharpError(input, ReasonUnitEmptySegment, "")
		}
		bare, escaped := strings.CutPrefix(segment, "@")
		if !isCSharpIdentifier(bare) {
			return csharpError(input, ReasonUnitNotIdentifier, segment)
		}
		if _, keyword := csharpKeywords[bare]; escaped && !keyword {
			return csharpError(input, ReasonNeedlessEscape, bare)
		}
	}
	return nil
}

// checkCSharpMember validates member and splits it into the owner chain, the name and the parameter list.
// The parameter list is nil when the member carries no parentheses or brackets.
func checkCSharpMember(input, member string) (owner []string, name string, params []string, err error) {
	if start, ok := findCSharpOperator(member); ok {
		return checkCSharpOperator(input, member, start)
	}
	if !balancedCSharpBrackets(member) {
		return nil, "", nil, csharpError(input, ReasonUnbalancedBrackets, member)
	}

	head, inner, open := cutCSharpParamList(member)
	components := splitCSharpTopLevel(head, '.')
	for _, component := range components {
		if component == "" {
			return nil, "", nil, csharpError(input, ReasonMemberEmptyComponent, "")
		}
	}
	name = components[len(components)-1]
	if (name == "this") != (open == '[') {
		return nil, "", nil, csharpError(input, ReasonIndexerBrackets, name)
	}
	if open == 0 && (strings.HasPrefix(name, "~") || strings.HasPrefix(name, "static ")) {
		return nil, "", nil, csharpError(input, ReasonMissingParens, name)
	}

	owner, err = checkCSharpOwner(input, components[:len(components)-1])
	if err != nil {
		return nil, "", nil, err
	}
	if err := checkCSharpName(input, name); err != nil {
		return nil, "", nil, err
	}
	if open != 0 {
		params, err = checkCSharpParams(input, inner)
		if err != nil {
			return nil, "", nil, err
		}
	}
	return owner, name, params, nil
}

// checkCSharpOperator validates a member whose name, starting at start, is an operator or a conversion operator.
// The operator token is never split or bracket-counted.
// Only the owner chain, a conversion's target type and the parameter list are checked for balance.
func checkCSharpOperator(input, member string, start int) (owner []string, name string, params []string, err error) {
	if start > 0 {
		ownerText := strings.TrimSuffix(member[:start], ".")
		if !balancedCSharpBrackets(ownerText) {
			return nil, "", nil, csharpError(input, ReasonUnbalancedBrackets, ownerText)
		}
		components := splitCSharpTopLevel(ownerText, '.')
		for _, component := range components {
			if component == "" {
				return nil, "", nil, csharpError(input, ReasonMemberEmptyComponent, "")
			}
		}
		owner, err = checkCSharpOwner(input, components)
		if err != nil {
			return nil, "", nil, err
		}
	}

	rest := member[start:]
	for _, prefix := range csharpConversionPrefixes {
		if strings.HasPrefix(rest, prefix) {
			return checkCSharpConversion(input, owner, prefix, rest[len(prefix):])
		}
	}

	token, list, hasList := strings.Cut(strings.TrimPrefix(rest, "operator "), "(")
	op, checked := strings.CutPrefix(token, "checked ")
	set := csharpOperators
	if checked {
		set = csharpCheckedOperators
	}
	if _, known := set[op]; !known {
		return nil, "", nil, csharpError(input, ReasonUnknownOperator, token)
	}
	name = "operator " + token
	if !hasList {
		return nil, "", nil, csharpError(input, ReasonMissingParens, name)
	}
	params, err = checkCSharpParenGroup(input, "("+list)
	if err != nil {
		return nil, "", nil, err
	}
	return owner, name, params, nil
}

// checkCSharpConversion validates the part of a conversion operator after its "implicit operator " or "explicit operator " prefix.
// That part is a canonical target type and a parenthesised parameter list.
func checkCSharpConversion(input string, owner []string, prefix, typeAndList string) ([]string, string, []string, error) {
	if !balancedCSharpBrackets(typeAndList) {
		return nil, "", nil, csharpError(input, ReasonUnbalancedBrackets, typeAndList)
	}
	head, inner, open := cutCSharpParamList(typeAndList)
	if open != '(' {
		return nil, "", nil, csharpError(input, ReasonMissingParens, prefix+typeAndList)
	}
	if head == "" {
		return nil, "", nil, csharpError(input, ReasonMemberEmptyComponent, "")
	}
	if canonical := CanonicalCSharpType(head); canonical != head {
		return nil, "", nil, csharpError(input, ReasonNonCanonicalType, canonical)
	}
	params, err := checkCSharpParams(input, inner)
	if err != nil {
		return nil, "", nil, err
	}
	return owner, prefix + head, params, nil
}

// checkCSharpParenGroup validates group, which starts with "(", as one parenthesised parameter list spanning the whole string.
// It returns the list's entries.
func checkCSharpParenGroup(input, group string) ([]string, error) {
	if !balancedCSharpBrackets(group) || !strings.HasSuffix(group, ")") {
		return nil, csharpError(input, ReasonUnbalancedBrackets, group)
	}
	head, inner, open := cutCSharpParamList(group)
	if open != '(' || head != "" {
		return nil, csharpError(input, ReasonUnbalancedBrackets, group)
	}
	return checkCSharpParams(input, inner)
}

// checkCSharpOwner validates each owner component as an identifier with an optional arity suffix or canonical type arguments.
func checkCSharpOwner(input string, components []string) ([]string, error) {
	if len(components) == 0 {
		return nil, nil
	}
	for _, component := range components {
		if err := checkCSharpGenericIdentifier(input, component); err != nil {
			return nil, err
		}
	}
	return components, nil
}

// checkCSharpName validates the member's last component.
// It is "this", a finalizer or static constructor spelling, or an identifier with an optional arity suffix.
func checkCSharpName(input, name string) error {
	switch {
	case name == "this":
		return nil
	case strings.HasPrefix(name, "~"):
		return checkCSharpIdentifier(input, strings.TrimPrefix(name, "~"))
	case strings.HasPrefix(name, "static "):
		return checkCSharpIdentifier(input, strings.TrimPrefix(name, "static "))
	}
	return checkCSharpGenericIdentifier(input, name)
}

// checkCSharpGenericIdentifier validates an identifier optionally followed by "<...>".
// The suffix is either an arity suffix (commas only) or canonical type arguments.
func checkCSharpGenericIdentifier(input, component string) error {
	base, suffix, generic := strings.Cut(component, "<")
	if err := checkCSharpIdentifier(input, base); err != nil {
		return err
	}
	if !generic {
		return nil
	}
	if !strings.HasSuffix(component, ">") {
		return csharpError(input, ReasonMemberNotIdentifier, component)
	}
	if canonical := CanonicalCSharpType(component); canonical != component {
		return csharpError(input, ReasonNonCanonicalType, canonical)
	}
	arguments := strings.TrimSuffix(suffix, ">")
	if strings.Trim(arguments, ",") == "" {
		return nil
	}
	for _, argument := range splitCSharpTopLevel(arguments, ',') {
		if argument == "" {
			return csharpError(input, ReasonMemberEmptyComponent, "")
		}
	}
	return nil
}

// checkCSharpIdentifier validates ident as a member identifier.
// A reserved keyword needs the "@" prefix, and any other identifier must not carry one.
func checkCSharpIdentifier(input, ident string) error {
	bare, escaped := strings.CutPrefix(ident, "@")
	if !isCSharpIdentifier(bare) {
		return csharpError(input, ReasonMemberNotIdentifier, ident)
	}
	_, keyword := csharpKeywords[bare]
	switch {
	case escaped && !keyword:
		return csharpError(input, ReasonNeedlessEscape, bare)
	case !escaped && keyword:
		return csharpError(input, ReasonMemberKeyword, ident)
	}
	return nil
}

// checkCSharpParams splits a parameter list's inner text into entries and requires each to be in its canonical spelling.
func checkCSharpParams(input, inner string) ([]string, error) {
	if inner == "" {
		return []string{}, nil
	}
	entries := splitCSharpTopLevel(inner, ',')
	params := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			return nil, csharpError(input, ReasonMemberEmptyComponent, "")
		}
		canonical := canonicalCSharpParam(entry)
		if canonical == "" {
			return nil, csharpError(input, ReasonMemberNotIdentifier, entry)
		}
		if canonical != entry {
			return nil, csharpError(input, ReasonNonCanonicalType, canonical)
		}
		params = append(params, entry)
	}
	return params, nil
}

// canonicalCSharpParam returns the canonical spelling of a parameter entry.
// That is its modifiers and its type, with any leading attribute lists, name and default value removed.
// It returns "" when the entry holds no type.
func canonicalCSharpParam(entry string) string {
	rest := strings.TrimSpace(entry)
	for strings.HasPrefix(rest, "[") {
		end := matchingCSharpBracket(rest, 0)
		if end < 0 {
			return ""
		}
		rest = strings.TrimSpace(rest[end+1:])
	}
	for i := 0; i < len(rest); i++ {
		if rest[i] == '=' && topLevelCSharpDepth(rest, i) == 0 {
			rest = strings.TrimSpace(rest[:i])
			break
		}
	}

	words := strings.Split(CanonicalCSharpType(rest), " ")
	modifiers := 0
	for modifiers < len(words) {
		if _, isModifier := csharpParamModifiers[words[modifiers]]; !isModifier {
			break
		}
		modifiers++
	}
	if modifiers == len(words) || words[modifiers] == "" {
		return ""
	}
	return strings.Join(words[:modifiers+1], " ")
}

// findCSharpOperator reports the index in member where an operator or conversion-operator name starts.
// That name is a top-level component beginning with "operator ", "implicit operator " or "explicit operator ".
func findCSharpOperator(member string) (start int, ok bool) {
	depth := 0
	for i := 0; i < len(member); i++ {
		switch member[i] {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		}
		if depth < 0 {
			return 0, false
		}
		if depth != 0 || (i > 0 && member[i-1] != '.') {
			continue
		}
		if strings.HasPrefix(member[i:], "operator ") {
			return i, true
		}
		for _, prefix := range csharpConversionPrefixes {
			if strings.HasPrefix(member[i:], prefix) {
				return i, true
			}
		}
	}
	return 0, false
}

// cutCSharpParamList splits a balanced member into the text before its trailing parenthesised or bracketed list and the list's inner text.
// open is '(' or '[' for the list found, and 0 when member ends with neither.
func cutCSharpParamList(member string) (head, inner string, open byte) {
	if member == "" {
		return "", "", 0
	}
	var closer, opener byte
	switch member[len(member)-1] {
	case ')':
		closer, opener = ')', '('
	case ']':
		closer, opener = ']', '['
	default:
		return member, "", 0
	}
	depth := 0
	for i := len(member) - 1; i >= 0; i-- {
		switch member[i] {
		case closer:
			depth++
		case opener:
			depth--
			if depth == 0 {
				return member[:i], member[i+1 : len(member)-1], opener
			}
		}
	}
	return member, "", 0
}

// matchingCSharpBracket returns the index of the "]" matching the "[" at s[start], or -1.
func matchingCSharpBracket(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// topLevelCSharpDepth returns how many "<", "(" or "[" are open in s before index i.
func topLevelCSharpDepth(s string, i int) int {
	depth := 0
	for _, c := range []byte(s[:i]) {
		switch c {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		}
	}
	return depth
}

// splitCSharpTopLevel splits s on sep wherever no "<>", "()" or "[]" group is open.
func splitCSharpTopLevel(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// balancedCSharpBrackets reports whether every "<", "(" and "[" in s closes in order.
func balancedCSharpBrackets(s string) bool {
	var open []byte
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '<', '(', '[':
			open = append(open, c)
		case '>', ')', ']':
			if len(open) == 0 || !bracketPairs(open[len(open)-1], c) {
				return false
			}
			open = open[:len(open)-1]
		}
	}
	return len(open) == 0
}

// bracketPairs reports whether opener and closer are the two halves of one bracket kind.
func bracketPairs(opener, closer byte) bool {
	return (opener == '<' && closer == '>') || (opener == '(' && closer == ')') || (opener == '[' && closer == ']')
}

// isCSharpIdentifier reports whether c is a C# identifier: first rune "_" or unicode.IsLetter;
// every later rune "_", unicode.IsLetter or unicode.IsDigit. The empty string is not one.
func isCSharpIdentifier(c string) bool {
	if c == "" {
		return false
	}
	for i, r := range c {
		if r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}
