// csharp_type.go holds CanonicalCSharpType, the one canonical spelling of a C# type or parameter.
// The C# alphabet's parameter lists, conversion-operator targets and interface qualifiers use it.

package glyph

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// csharpTypeToken is one lexical unit of a C# type spelling.
// It is an identifier-like word or one punctuation unit ("::" is one unit).
type csharpTypeToken struct {
	text string
	word bool
}

// csharpBuiltinAliases maps a framework type name to the keyword C# spells it with.
var csharpBuiltinAliases = map[string]string{
	"Boolean": "bool",
	"Byte":    "byte",
	"SByte":   "sbyte",
	"Char":    "char",
	"Decimal": "decimal",
	"Double":  "double",
	"Single":  "float",
	"Int16":   "short",
	"UInt16":  "ushort",
	"Int32":   "int",
	"UInt32":  "uint",
	"Int64":   "long",
	"UInt64":  "ulong",
	"Object":  "object",
	"String":  "string",
	"IntPtr":  "nint",
	"UIntPtr": "nuint",
}

// CanonicalCSharpType returns the canonical spelling of a C# parameter or type spelling.
// Whitespace is removed except one space between two adjacent words.
// Every modifier is kept, and the "global::" qualifier is removed.
// A built-in alias, bare or "System."-qualified at any nesting depth, is replaced by its keyword.
// Tuple element names are dropped, and everything else is kept as written.
// The function is total and idempotent: canonicalising a canonical spelling returns it unchanged.
func CanonicalCSharpType(s string) string {
	tokens := dropGlobalQualifiers(tokenizeCSharpType(s))
	var out strings.Builder
	var open []string
	var prev csharpTypeToken
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		if !tok.word {
			switch tok.text {
			case "(", "<", "[":
				open = append(open, tok.text)
			case ")", ">", "]":
				if len(open) > 0 {
					open = open[:len(open)-1]
				}
			}
			out.WriteString(tok.text)
			prev = tok
			i++
			continue
		}
		end := dottedChainEnd(tokens, i)
		var chain strings.Builder
		for _, part := range tokens[i:end] {
			chain.WriteString(part.text)
		}
		i = end
		if isTupleElementName(prev, open) {
			continue
		}
		if prev.word {
			out.WriteByte(' ')
		}
		out.WriteString(builtinKeyword(chain.String()))
		prev = csharpTypeToken{text: chain.String(), word: true}
	}
	return out.String()
}

// isTupleElementName reports whether a word following prev is a tuple element's name.
// It is one when the innermost open bracket is a parenthesis and prev ends a type.
func isTupleElementName(prev csharpTypeToken, open []string) bool {
	if len(open) == 0 || open[len(open)-1] != "(" {
		return false
	}
	if prev.word {
		return true
	}
	switch prev.text {
	case ">", "]", "?", ")", "*":
		return true
	}
	return false
}

// builtinKeyword replaces a whole dotted name that is a built-in alias with its keyword.
// The alias may be bare or qualified by "System."; any other name is returned unchanged.
func builtinKeyword(name string) string {
	bare := strings.TrimPrefix(name, "System.")
	if keyword, ok := csharpBuiltinAliases[bare]; ok {
		return keyword
	}
	return name
}

// dottedChainEnd returns the index just past the dotted name that starts at tokens[start].
func dottedChainEnd(tokens []csharpTypeToken, start int) int {
	end := start + 1
	for end+1 < len(tokens) && tokens[end].text == "." && tokens[end+1].word {
		end += 2
	}
	return end
}

// dropGlobalQualifiers removes every "global ::" pair.
func dropGlobalQualifiers(tokens []csharpTypeToken) []csharpTypeToken {
	kept := make([]csharpTypeToken, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		if tokens[i].word && tokens[i].text == "global" && i+1 < len(tokens) && tokens[i+1].text == "::" {
			i++
			continue
		}
		kept = append(kept, tokens[i])
	}
	return kept
}

// tokenizeCSharpType splits s into words and punctuation units, discarding whitespace.
func tokenizeCSharpType(s string) []csharpTypeToken {
	var tokens []csharpTypeToken
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case isCSharpTypeWordRune(r):
			start := i
			for i < len(s) {
				r, size = utf8.DecodeRuneInString(s[i:])
				if !isCSharpTypeWordRune(r) {
					break
				}
				i += size
			}
			tokens = append(tokens, csharpTypeToken{text: s[start:i], word: true})
		case strings.HasPrefix(s[i:], "::"):
			tokens = append(tokens, csharpTypeToken{text: "::"})
			i += 2
		default:
			tokens = append(tokens, csharpTypeToken{text: s[i : i+size]})
			i += size
		}
	}
	return tokens
}

// isCSharpTypeWordRune reports whether r can belong to an identifier-like word.
func isCSharpTypeWordRune(r rune) bool {
	return r == '_' || r == '@' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}
