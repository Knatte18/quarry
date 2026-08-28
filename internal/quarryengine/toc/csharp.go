// csharp.go implements the C# Strategy: type and member extraction, and the C# header,
// generated-file, and test-file rules. It registers itself under the canonical language name
// "csharp" from this file's own init.

package toc

import (
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// csharpStrategy implements Strategy for C# source parsed by the pinned tree-sitter-c-sharp v0.23.5
// grammar.
type csharpStrategy struct{}

func init() {
	Register(csharpStrategy{})
}

// Language implements Strategy by returning "csharp".
func (csharpStrategy) Language() string {
	return "csharp"
}

// Symbols implements Strategy. It descends through C#'s container nodes — a file-scoped namespace's
// following siblings, a braced namespace's declaration_list body, and a type declaration's own
// declaration_list body — and emits a Symbol for each type declaration and each named callable
// member it finds there.
//
// A file_scoped_namespace_declaration ("namespace X;") is matched but never descended into: the
// declarations it governs are its siblings, not its children, so the walk simply continues over the
// same child list rather than recursing.
func (csharpStrategy) Symbols(root *ts.Node, src []byte) []Symbol {
	return csharpWalkContainer(root, "", src)
}

// csharpWalkContainer walks container's direct children — the compilation_unit root, a namespace's
// declaration_list body, or a type's declaration_list body — and returns the Symbols found there.
// owner is "" at the root and namespace levels, and the enclosing type's bare name inside a type's
// body.
//
// The emitted member set is closed and deliberately excludes everything but
// method_declaration, constructor_declaration, and destructor_declaration:
//   - property_declaration, indexer_declaration, event_declaration, event_field_declaration, and
//     field_declaration are state, not a callable, and the accessor-bearing forms would additionally
//     need a SigEnd rule for accessor_list that nothing else in this design requires;
//   - delegate_declaration and enum_member_declaration are neither state nor a callable this
//     package's Kind vocabulary has room for;
//   - operator_declaration and conversion_operator_declaration are callables, but neither carries a
//     "name" field — an operator's identity lives in an "operator:" field holding the symbol, and a
//     conversion operator has no name node at all, only a "type:" field. Emitting them would need a
//     bespoke name-synthesis rule the function/method/type vocabulary has no room for, so they are
//     omitted rather than half-named.
//
// A nested type declaration inside a declaration_list is not a member: it is matched by the type
// branch below and emitted as KindType, then descended into like any other type. A declaration
// inside a member's own body is never reached, because this walk only ever descends into a
// declaration_list or a namespace body, never into a block.
func csharpWalkContainer(container *ts.Node, owner string, src []byte) []Symbol {
	symbols := make([]Symbol, 0)
	for i := uint(0); i < container.ChildCount(); i++ {
		child := container.Child(i)
		switch child.Kind() {
		case "file_scoped_namespace_declaration":
			// Matched, deliberately, so a reader sees this shape was considered — but there is
			// nothing to descend into: the declarations it governs are child's siblings in this same
			// container, which the loop already visits on its own.
		case "namespace_declaration":
			body := child.ChildByFieldName("body")
			if body != nil {
				symbols = append(symbols, csharpWalkContainer(body, "", src)...)
			}
		case "class_declaration", "interface_declaration", "record_declaration", "struct_declaration":
			name := NodeText(child.ChildByFieldName("name"), src)
			symbols = append(symbols, csharpTypeSymbol(child, name, src))
			body := child.ChildByFieldName("body")
			if body != nil {
				symbols = append(symbols, csharpWalkContainer(body, name, src)...)
			}
		case "method_declaration", "constructor_declaration", "destructor_declaration":
			symbols = append(symbols, csharpMemberSymbol(child, owner, src))
		}
	}
	return symbols
}

// csharpTypeSymbol builds the KindType Symbol for decl (a class_declaration, interface_declaration,
// record_declaration, or struct_declaration).
func csharpTypeSymbol(decl *ts.Node, name string, src []byte) Symbol {
	body := decl.ChildByFieldName("body")
	return csharpDeclSymbol(KindType, name, "", decl, body, src)
}

// csharpMemberSymbol builds the KindMethod Symbol for decl (a method_declaration,
// constructor_declaration, or destructor_declaration), all three of which carry a "name" field in
// the pinned grammar.
func csharpMemberSymbol(decl *ts.Node, owner string, src []byte) Symbol {
	name := NodeText(decl.ChildByFieldName("name"), src)
	body := decl.ChildByFieldName("body")
	return csharpDeclSymbol(KindMethod, name, owner, decl, body, src)
}

// csharpDeclSymbol builds the Symbol shared by a type declaration and a member declaration.
// Docstring and range come from CommentBlockAbove: Start is the comment block's first line when one
// was found, and decl's own first line otherwise; End is always decl's last line.
func csharpDeclSymbol(kind Kind, name, owner string, decl, body *ts.Node, src []byte) Symbol {
	docComment, raw := CommentBlockAbove(decl, src)
	start, _ := Line(decl)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(decl)
	return Symbol{
		Kind:      kind,
		Name:      name,
		Owner:     owner,
		Signature: csharpSignature(decl, body, src),
		Docstring: StripXMLDocTags(StripComment(raw, "///")),
		Start:     start,
		SigEnd:    SigEnd(decl, body, csharpBodyOnSignatureLine(body)),
		End:       end,
	}
}

// csharpSignature returns decl's signature, cut at body per SignatureCut, with one C#-specific
// addition: when body is nil — a positional record declaration, or an interface method with no
// implementation — the declaration's whole text ends in a trailing ";" that SignatureCut has no
// reason to know about, so it is trimmed here. Without this trim every bodyless C# signature would
// end in a stray semicolon.
func csharpSignature(decl, body *ts.Node, src []byte) string {
	sig := SignatureCut(decl, body, src)
	if body == nil {
		sig = strings.TrimSpace(strings.TrimSuffix(sig, ";"))
	}
	return sig
}

// csharpBodyOnSignatureLine reports whether body's own kind puts its opening token on the
// signature's last line, the flag SigEnd needs. It is read off body's Kind() rather than off decl's,
// so the rule is stated once for every C# kind rather than per call site:
//   - "block" and "declaration_list" hold the body-opening "{" on the signature's last line;
//   - "arrow_expression_clause" starts on the line after the signature ends, before the "=>";
//   - nil has no body-opening token at all; the return value is unused in that case, since SigEnd
//     short-circuits to 0 for a nil body before ever consulting this flag.
func csharpBodyOnSignatureLine(body *ts.Node) bool {
	if body == nil {
		return false
	}
	switch body.Kind() {
	case "block", "declaration_list":
		return true
	default:
		// "arrow_expression_clause" is the only other body shape this grammar produces.
		return false
	}
}
