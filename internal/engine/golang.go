// golang.go implements the Go Strategy: function, method, and type extraction, and the Go header,
// generated-file, and test-file rules. It registers itself under the canonical language name "go"
// from this file's own init.

package engine

import (
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// goStrategy implements Strategy for Go source parsed by the pinned tree-sitter-go v0.25.0 grammar.
type goStrategy struct{}

func init() {
	Register(goStrategy{})
}

// Language implements Strategy by returning "go".
func (goStrategy) Language() string {
	return "go"
}

// Symbols implements Strategy. It iterates the direct children of the source_file root in source
// order and handles exactly three child kinds — function_declaration, method_declaration, and
// type_declaration — ignoring every other child. It never descends into a "block", so a
// type_declaration or func literal nested inside a function body is never listed.
func (goStrategy) Symbols(root *ts.Node, src []byte) []Symbol {
	symbols := make([]Symbol, 0)
	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		switch child.Kind() {
		case "function_declaration":
			symbols = append(symbols, goFunctionSymbol(child, src))
		case "method_declaration":
			symbols = append(symbols, goMethodSymbol(child, src))
		case "type_declaration":
			symbols = append(symbols, goTypeSymbols(child, src)...)
		}
	}
	return symbols
}

// goFunctionSymbol builds the KindFunction Symbol for a function_declaration node.
func goFunctionSymbol(decl *ts.Node, src []byte) Symbol {
	body := decl.ChildByFieldName("body")
	name := NodeText(decl.ChildByFieldName("name"), src)
	return goDeclSymbol(KindFunction, name, "", decl, body, src)
}

// goMethodSymbol builds the KindMethod Symbol for a method_declaration node. Owner is the receiver's
// type name, read from the receiver field's parameter_declaration's type field, with a leading "*"
// stripped when that type is a pointer_type — so a "*T" receiver yields "T".
func goMethodSymbol(decl *ts.Node, src []byte) Symbol {
	body := decl.ChildByFieldName("body")
	name := NodeText(decl.ChildByFieldName("name"), src)
	owner := goReceiverTypeName(decl.ChildByFieldName("receiver"), src)
	return goDeclSymbol(KindMethod, name, owner, decl, body, src)
}

// goReceiverTypeName extracts the bare receiver type name from a method's receiver field
// (a parameter_list holding exactly one parameter_declaration), stripping a leading "*" when the
// declared type is a pointer_type.
func goReceiverTypeName(receiver *ts.Node, src []byte) string {
	for i := uint(0); i < receiver.NamedChildCount(); i++ {
		param := receiver.NamedChild(i)
		if param.Kind() != "parameter_declaration" {
			continue
		}
		typeNode := param.ChildByFieldName("type")
		if typeNode.Kind() == "pointer_type" {
			// A pointer_type's named child is the pointee type; NodeText on the pointer_type itself
			// would keep the leading "*", which the Owner field must not carry.
			return NodeText(typeNode.NamedChild(0), src)
		}
		return NodeText(typeNode, src)
	}
	return ""
}

// goDeclSymbol builds the Symbol shared by function_declaration and method_declaration extraction:
// docstring and range come from CommentBlockAbove, and the signature and sigend come from the
// shared node helpers with bodyOnSignatureLine = true, since Go always puts the body's opening "{"
// on the signature's own last line.
func goDeclSymbol(kind Kind, name, owner string, decl, body *ts.Node, src []byte) Symbol {
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
		Signature: SignatureCut(decl, body, src),
		Docstring: StripComment(raw, "//"),
		Start:     start,
		SigEnd:    SigEnd(decl, body),
		End:       end,
	}
}

// goTypeSymbols builds the KindType Symbol(s) for one type_declaration node, covering both the
// ungrouped ("type X int") and grouped ("type ( ... )") source shapes with one rule rather than a
// branch per shape.
//
// The two shapes are distinguished by the presence of a literal "(" child of decl, never by
// counting spec children: "type ( X int )" is legal Go with a single spec, and a spec-count test
// would route it down the ungrouped path, cutting Signature from the type_declaration and emitting
// the whole "type (\n\tX int\n)" block with the group's range instead of the spec's. Verified against
// the pinned tree-sitter-go v0.25.0: a grouped declaration always has a literal "(" child and an
// ungrouped one never does, whatever the spec count.
func goTypeSymbols(decl *ts.Node, src []byte) []Symbol {
	if !goDeclIsGrouped(decl) {
		spec := goSpecChild(decl)
		return []Symbol{goUngroupedTypeSymbol(decl, spec, src)}
	}
	symbols := make([]Symbol, 0, decl.NamedChildCount())
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		spec := decl.NamedChild(i)
		if spec.Kind() != "type_spec" && spec.Kind() != "type_alias" {
			continue
		}
		symbols = append(symbols, goGroupedTypeSymbol(spec, src))
	}
	return symbols
}

// goDeclIsGrouped reports whether decl (a type_declaration) is the grouped "type ( ... )" form, by
// the presence of a literal "(" child — the one shape marker that survives a single-spec group,
// where a spec-count test would not.
func goDeclIsGrouped(decl *ts.Node) bool {
	for i := uint(0); i < decl.ChildCount(); i++ {
		if decl.Child(i).Kind() == "(" {
			return true
		}
	}
	return false
}

// goSpecChild returns decl's single spec child — a type_spec, or a type_alias for "type X = Y",
// which the pinned grammar produces instead of a type_spec. It is called only on an ungrouped
// type_declaration, which the grammar always gives exactly one spec child.
func goSpecChild(decl *ts.Node) *ts.Node {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if c.Kind() == "type_spec" || c.Kind() == "type_alias" {
			return c
		}
	}
	return nil
}

// goUngroupedTypeSymbol builds the Symbol for an ungrouped type_declaration's single spec. Its
// Signature and range are computed from decl, not from spec: Signature is cut from decl's own first
// byte so the emitted text carries the "type" keyword, and the docstring walk starts at decl, since
// decl — not spec — is the node with source_file siblings.
func goUngroupedTypeSymbol(decl, spec *ts.Node, src []byte) Symbol {
	body := goTypeBody(spec)
	docComment, raw := CommentBlockAbove(decl, src)
	start, _ := Line(decl)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(decl)
	return Symbol{
		Kind:      KindType,
		Name:      NodeText(spec.ChildByFieldName("name"), src),
		Signature: SignatureCut(decl, body, src),
		Docstring: StripComment(raw, "//"),
		Start:     start,
		SigEnd:    SigEnd(decl, body),
		End:       end,
	}
}

// goGroupedTypeSymbol builds the Symbol for one spec inside a grouped "type ( ... )" declaration.
// Its Signature and range are computed from spec itself, not from the enclosing type_declaration:
// the grammar makes each spec's own doc comment a comment child of the type_declaration, interleaved
// with the spec siblings, so CommentBlockAbove's prev-sibling walk works unchanged one level down
// from the ungrouped case. The rendered signature is "type " prepended to the spec's own signature
// text, so the grouped and ungrouped forms produce identical output for identical types. A comment
// on the "type (" line itself is a prev-sibling of decl, not of any spec, so it is never picked up
// here — that is this rule falling out naturally, not a special case.
func goGroupedTypeSymbol(spec *ts.Node, src []byte) Symbol {
	body := goTypeBody(spec)
	docComment, raw := CommentBlockAbove(spec, src)
	start, _ := Line(spec)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(spec)
	return Symbol{
		Kind:      KindType,
		Name:      NodeText(spec.ChildByFieldName("name"), src),
		Signature: "type " + SignatureCut(spec, body, src),
		Docstring: StripComment(raw, "//"),
		Start:     start,
		SigEnd:    SigEnd(spec, body),
		End:       end,
	}
}

// goTypeBody resolves spec's body-bearing child from its "type" field child: the first direct child
// of that type node whose kind is "field_declaration_list" (a struct_type's body) or "{" (an
// interface_type's body, which the grammar exposes with no named body node and no "body" field at
// all). It returns nil when spec's type has neither — "type ID string", "type Alias = T" — in which
// case the whole spec text is the signature, short by construction.
//
// A naive spec.ChildByFieldName("body") is wrong here: a Go type_declaration's spec has no "body"
// field, so the naive call returns nil unconditionally and the signature would silently become the
// entire struct body — the exact token blowup this verb exists to prevent.
func goTypeBody(spec *ts.Node) *ts.Node {
	typeNode := spec.ChildByFieldName("type")
	if typeNode == nil {
		return nil
	}
	for i := uint(0); i < typeNode.ChildCount(); i++ {
		c := typeNode.Child(i)
		if c.Kind() == "field_declaration_list" || c.Kind() == "{" {
			return c
		}
	}
	return nil
}

// Package implements Strategy by returning the package_clause child's package_identifier text, or
// "" when root has no package_clause — the shape a file broken badly enough to lose its package
// clause under a partial parse takes.
func (goStrategy) Package(root *ts.Node, src []byte) string {
	for i := uint(0); i < root.ChildCount(); i++ {
		clause := root.Child(i)
		if clause.Kind() != "package_clause" {
			continue
		}
		for j := uint(0); j < clause.ChildCount(); j++ {
			ident := clause.Child(j)
			if ident.Kind() == "package_identifier" {
				return NodeText(ident, src)
			}
		}
	}
	return ""
}

// goPackageClauseNode returns root's package_clause child, or nil when root has none — the shape a
// file broken badly enough to lose its package clause under a partial parse takes.
func goPackageClauseNode(root *ts.Node) *ts.Node {
	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child.Kind() == "package_clause" {
			return child
		}
	}
	return nil
}

// PackageDoc implements Strategy by taking the comment block immediately above the package_clause
// — via CommentBlockAbove, the same prev-sibling walk with the blank-line boundary Header's own
// directive-skipping loop does not use — and returning FirstParagraph of it only when the
// stripped block's first line begins with "Package " followed by the package name this file
// declares. Otherwise it returns "".
//
// The prefix test is what this method exists for: a file can carry both a file header and a
// separate package doc comment as two distinct leading blocks, and only the "Package <name>"
// prefix distinguishes the latter from the former. Without it, an adjacent file header sitting
// immediately above the package clause — with no package doc comment of its own — would be
// misread as the package documentation.
func (goStrategy) PackageDoc(root *ts.Node, src []byte) string {
	clause := goPackageClauseNode(root)
	if clause == nil {
		return ""
	}
	_, raw := CommentBlockAbove(clause, src)
	if raw == "" {
		return ""
	}
	stripped := StripComment(raw, "//")

	pkgName := goStrategy{}.Package(root, src)
	firstLine := stripped
	if idx := strings.IndexByte(stripped, '\n'); idx >= 0 {
		firstLine = stripped[:idx]
	}
	prefix := "Package " + pkgName
	rest := strings.TrimPrefix(firstLine, prefix)
	if rest == firstLine || (rest != "" && !strings.HasPrefix(rest, " ")) {
		// Either the prefix was absent, or it was present but immediately followed by more of the
		// same identifier (e.g. package "p" against a header reading "Package pkg ...") rather than
		// a word boundary — neither is the "Package <name>" convention this rule tests for.
		return ""
	}
	return FirstParagraph(stripped)
}

// Header implements Strategy by walking LeadingBlocks in order, stripping each block with
// StripComment(b.Raw, "//"), and returning the first block for which IsDirectiveBlock("go", ...) is
// false. It returns "" when every leading block is a directive block, or the file has no leading
// comment at all.
//
// This deliberately differs from docstring association in two ways. First, it takes the first
// non-directive block rather than the block adjacent to the package clause: a file can carry a build
// constraint, a blank line, then its real header, and package itself carries no comment of its own
// in that shape. Second, it tolerates the blank line CommentBlockAbove would treat as a block
// boundary against a declaration: a file can carry both a file header and a separate package doc
// comment immediately above package, and it is the earlier, file-header block that this method must
// return.
func (goStrategy) Header(root *ts.Node, src []byte) string {
	for _, block := range LeadingBlocks(root, src) {
		stripped := StripComment(block.Raw, "//")
		if !IsDirectiveBlock("go", block.StartLine, stripped) {
			return stripped
		}
	}
	return ""
}

// Generated implements Strategy by reading the first leading block — directive or not, since a
// generated-file banner is a directive block for Header's purposes and a marker here; the two
// readings are independent — and delegating to GeneratedByBanner("go", ...) on that block's
// delimiter-stripped text, which is the form GeneratedByBanner's own contract requires. Go's own
// rule is always known, so the known return GeneratedByBanner reports is discarded here — see the
// Strategy interface's doc comment for why Generated itself carries no known return any more.
func (goStrategy) Generated(root *ts.Node, src []byte) bool {
	blocks := LeadingBlocks(root, src)
	if len(blocks) == 0 {
		generated, _ := GeneratedByBanner("go", "")
		return generated
	}
	stripped := StripComment(blocks[0].Raw, "//")
	generated, _ := GeneratedByBanner("go", stripped)
	return generated
}

// TestFile implements Strategy by delegating to TestFileByName("go", base) and discarding its
// known return, for the same reason Generated discards GeneratedByBanner's.
func (goStrategy) TestFile(base string) bool {
	isTest, _ := TestFileByName("go", base)
	return isTest
}
