// golang.go implements the Go Strategy: function, method, type, const, and var extraction, and the
// Go header, generated-file, and test-file rules. It registers itself under the canonical language
// name "go" from this file's own init.

package engine

import (
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
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

// goBlank reports whether name is the blank identifier "_". A blank-named declaration names
// nothing a plan could ever target, and giving every blank declaration in a package the one glyph
// its bare name would produce would collapse them the way func init's shared id does, but without
// init's defined multipart meaning. docs/glyph.md's identifier contract is silent on this point;
// excluding the blank identifier from every kind this strategy extracts is quarry's own choice,
// made so that every emitted id stays addressable. Every builder in this file that can see a
// blank-named declaration — a function, a method, a type, a const, or a var — checks this before
// building a Symbol for it.
func goBlank(name string) bool {
	return name == "_"
}

// Symbols implements Strategy. It iterates the direct children of the source_file root in source
// order and handles five child kinds — function_declaration, method_declaration, type_declaration,
// const_declaration, and var_declaration — ignoring every other child. It never descends into a
// "block", so a type_declaration or func literal nested inside a function body is never listed.
func (goStrategy) Symbols(unit string, root *ts.Node, src []byte) []Symbol {
	symbols := make([]Symbol, 0)
	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		switch child.Kind() {
		case "function_declaration":
			if sym, ok := goFunctionSymbol(unit, child, src); ok {
				symbols = append(symbols, sym)
			}
		case "method_declaration":
			if sym, ok := goMethodSymbol(unit, child, src); ok {
				symbols = append(symbols, sym)
			}
		case "type_declaration":
			symbols = append(symbols, goTypeSymbols(unit, child, src)...)
		case "const_declaration":
			symbols = append(symbols, goConstSymbols(unit, child, src)...)
		case "var_declaration":
			symbols = append(symbols, goVarSymbols(unit, child, src)...)
		}
	}
	return symbols
}

// goFunctionSymbol builds the KindFunction Symbol for a function_declaration node, or reports ok
// == false for a blank-named one — see goBlank. Several func init() in one package all carry the
// one id "<unit>#init" and are listed separately with their own spans: that falls out of building
// the glyph from the name alone, with no dedup and no special case for "init".
func goFunctionSymbol(unit string, decl *ts.Node, src []byte) (sym Symbol, ok bool) {
	name := NodeText(decl.ChildByFieldName("name"), src)
	if goBlank(name) {
		return Symbol{}, false
	}
	body := decl.ChildByFieldName("body")
	return goDeclSymbol(KindFunction, unit, nil, name, decl, body, src), true
}

// goMethodSymbol builds the KindMethod Symbol for a method_declaration node, or reports ok == false
// for a blank-named one — see goBlank. Owner is the receiver's bare type name, read from the
// receiver field's parameter_declaration's type field via goReceiverTypeName.
func goMethodSymbol(unit string, decl *ts.Node, src []byte) (sym Symbol, ok bool) {
	name := NodeText(decl.ChildByFieldName("name"), src)
	if goBlank(name) {
		return Symbol{}, false
	}
	body := decl.ChildByFieldName("body")
	owner := goReceiverTypeName(decl.ChildByFieldName("receiver"), src)
	return goDeclSymbol(KindMethod, unit, []string{owner}, name, decl, body, src), true
}

// goReceiverTypeName extracts the bare receiver type name from a method's receiver field
// (a parameter_list holding exactly one parameter_declaration), stripping a leading "*" when the
// declared type is a pointer_type, and — after that unwrap — stripping a generic receiver's type
// parameters when the declared type is a generic_type.
//
// Type parameters are not part of a glyph (docs/glyph.md §3: "Box[T]" is "Box", since Go does not
// allow the two to coexist), so "func (b *Box[T]) M()" must yield the owner "Box", never "Box[T]" —
// the latter's "[" and "]" would make glyph.Parse reject the resulting id with a type-parameters
// reason. A generic_type's own "type" field is exactly that bare identifier, so this function takes
// it directly rather than the generic_type's whole text. This is the one place in the Go strategy
// this stripping is applied explicitly: a type's own declared *name* (goSpecChild's target, for
// instance) is already bare wherever it appears, so only a receiver's reference to that name — which
// spells out the type parameters again — needs unwrapping.
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
			typeNode = typeNode.NamedChild(0)
		}
		if typeNode.Kind() == "generic_type" {
			return NodeText(typeNode.ChildByFieldName("type"), src)
		}
		return NodeText(typeNode, src)
	}
	return ""
}

// goDeclOffsets computes the DeclStart, BodyStart and DeclEnd triple for a declaration node decl and
// its possibly-nil body-bearing child body: DeclStart and DeclEnd are decl's own start and end bytes,
// and BodyStart is body's start byte when body is non-nil and DeclEnd otherwise. Every builder in
// this file that fills these three Symbol fields calls this one helper, so the nil-body rule has
// exactly one implementation rather than six.
func goDeclOffsets(decl, body *ts.Node) (declStart, bodyStart, declEnd int) {
	declStart = int(decl.StartByte())
	declEnd = int(decl.EndByte())
	if body != nil {
		bodyStart = int(body.StartByte())
	} else {
		bodyStart = declEnd
	}
	return declStart, bodyStart, declEnd
}

// goDeclSymbol builds the Symbol shared by function_declaration and method_declaration extraction:
// docstring and range come from CommentBlockAbove, and the signature and sigend come from the
// shared node helpers. owner is nil for a package-level name and a one-element slice naming the
// receiver's bare type for a method — the same rule docs/glyph.md §3 states for the Go member form.
func goDeclSymbol(kind Kind, unit string, owner []string, name string, decl, body *ts.Node, src []byte) Symbol {
	docComment, raw := CommentBlockAbove(decl, src)
	start, _ := Line(decl)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(decl)
	declStart, bodyStart, declEnd := goDeclOffsets(decl, body)
	g := glyph.Glyph{Lang: glyph.Go, Unit: unit, Owner: owner, Name: name}
	return Symbol{
		Glyph:     g,
		ID:        g.String(),
		Kind:      kind,
		Signature: SignatureCut(decl, body, src),
		Doc:       StripComment(raw, "//"),
		Start:     start,
		SigEnd:    SigEnd(decl, body),
		End:       end,
		DeclStart: declStart,
		BodyStart: bodyStart,
		DeclEnd:   declEnd,
	}
}

// goTypeSymbols builds the KindType Symbol(s) for one type_declaration node, covering both the
// ungrouped ("type X int") and grouped ("type ( ... )") source shapes with one rule rather than a
// branch per shape, and — for each type spec whose declared type is a named interface — the
// KindMethod Symbol(s) for that interface's own method_elem members, via
// goInterfaceMethodSymbols.
//
// The two shapes are distinguished by the presence of a literal "(" child of decl, never by
// counting spec children: "type ( X int )" is legal Go with a single spec, and a spec-count test
// would route it down the ungrouped path, cutting Signature from the type_declaration and emitting
// the whole "type (\n\tX int\n)" block with the group's range instead of the spec's. Verified against
// the pinned tree-sitter-go v0.25.0: a grouped declaration always has a literal "(" child and an
// ungrouped one never does, whatever the spec count.
func goTypeSymbols(unit string, decl *ts.Node, src []byte) []Symbol {
	if !goDeclIsGrouped(decl) {
		spec := goSpecChild(decl)
		sym, ok := goUngroupedTypeSymbol(unit, decl, spec, src)
		if !ok {
			return nil
		}
		symbols := []Symbol{sym}
		return append(symbols, goInterfaceMethodSymbols(unit, sym.Glyph.Name, spec, src)...)
	}
	symbols := make([]Symbol, 0, decl.NamedChildCount())
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		spec := decl.NamedChild(i)
		if spec.Kind() != "type_spec" && spec.Kind() != "type_alias" {
			continue
		}
		sym, ok := goGroupedTypeSymbol(unit, spec, src)
		if !ok {
			continue
		}
		symbols = append(symbols, sym)
		symbols = append(symbols, goInterfaceMethodSymbols(unit, sym.Glyph.Name, spec, src)...)
	}
	return symbols
}

// goInterfaceMethodSymbols returns one KindMethod Symbol per method_elem directly inside typeSpec's
// declared type, owned by ownerName — the enclosing interface's own name — but only when typeSpec's
// "type" field is itself an interface_type; every other typeSpec (a struct, an alias, a defined
// scalar) contributes nothing, so every caller can call this unconditionally once per type symbol.
//
// Only the interface named by a file-scope type_spec or type_alias is ever descended into this way:
// this function's callers, goTypeSymbols' two branches, only ever hand it the interface_type that is
// such a spec's own "type" field child, never one reached by descending into a struct field, a
// parameter, a return type, a var, or a generic constraint. An anonymous interface_type in any of
// those positions is therefore never reached from here and never contributes a symbol — it has no
// type name to own its methods, and docs/glyph.md's identifier contract excludes struct fields and
// local declarations outright. Without this bound the walk would emit owner-less or wrongly-owned
// glyphs for a method that was never actually the named interface's own, and the round trip — two
// readings of one walk — cannot catch that kind of mistake.
//
// An embedded interface (Embedded, or an inline anonymous one) is a type_elem, not a method_elem, so
// it is excluded by the child-kind switch below falling out naturally rather than by a special case:
// the embedded name is not itself a member of the embedder.
func goInterfaceMethodSymbols(unit, ownerName string, typeSpec *ts.Node, src []byte) []Symbol {
	typeNode := typeSpec.ChildByFieldName("type")
	if typeNode == nil || typeNode.Kind() != "interface_type" {
		return nil
	}
	var symbols []Symbol
	for i := uint(0); i < typeNode.ChildCount(); i++ {
		elem := typeNode.Child(i)
		if elem.Kind() != "method_elem" {
			continue
		}
		name := NodeText(elem.ChildByFieldName("name"), src)
		if goBlank(name) {
			continue
		}
		docComment, raw := CommentBlockAbove(elem, src)
		start, _ := Line(elem)
		if docComment != nil {
			start, _ = Line(docComment)
		}
		_, end := Line(elem)
		declStart, bodyStart, declEnd := goDeclOffsets(elem, nil)
		g := glyph.Glyph{Lang: glyph.Go, Unit: unit, Owner: []string{ownerName}, Name: name}
		symbols = append(symbols, Symbol{
			Glyph:     g,
			ID:        g.String(),
			Kind:      KindMethod,
			Signature: SignatureCut(elem, nil, src),
			Doc:       StripComment(raw, "//"),
			Start:     start,
			End:       end,
			DeclStart: declStart,
			BodyStart: bodyStart,
			DeclEnd:   declEnd,
		})
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

// goUngroupedTypeSymbol builds the Symbol for an ungrouped type_declaration's single spec, or
// reports ok == false for a blank-named one — see goBlank. Its Signature and range are computed
// from decl, not from spec: Signature is cut from decl's own first byte so the emitted text carries
// the "type" keyword, and the docstring walk starts at decl, since decl — not spec — is the node
// with source_file siblings.
//
// HeadStart and HeadEnd are set to this same Start and End, doc block included, for every Go type
// alike — a struct, an interface, a defined scalar — because the expand verb emits a type's head as
// this one contiguous span alongside its member symbols' own spans, and subtracting those member
// spans from the head range is that consumer's job, not this extractor's. An interface is the one
// Go type whose declaration contains its own members (goInterfaceMethodSymbols' method symbols), so
// for an interface specifically the member spans genuinely do lie inside this head range; for every
// other type they simply have no member spans to subtract.
func goUngroupedTypeSymbol(unit string, decl, spec *ts.Node, src []byte) (sym Symbol, ok bool) {
	name := NodeText(spec.ChildByFieldName("name"), src)
	if goBlank(name) {
		return Symbol{}, false
	}
	body := goTypeBody(spec)
	docComment, raw := CommentBlockAbove(decl, src)
	start, _ := Line(decl)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(decl)
	declStart, bodyStart, declEnd := goDeclOffsets(decl, body)
	g := glyph.Glyph{Lang: glyph.Go, Unit: unit, Name: name}
	return Symbol{
		Glyph:     g,
		ID:        g.String(),
		Kind:      KindType,
		Signature: SignatureCut(decl, body, src),
		Doc:       StripComment(raw, "//"),
		Start:     start,
		SigEnd:    SigEnd(decl, body),
		End:       end,
		HeadStart: start,
		HeadEnd:   end,
		DeclStart: declStart,
		BodyStart: bodyStart,
		DeclEnd:   declEnd,
	}, true
}

// goGroupedTypeSymbol builds the Symbol for one spec inside a grouped "type ( ... )" declaration,
// or reports ok == false for a blank-named one — see goBlank. Its Signature and range are computed
// from spec itself, not from the enclosing type_declaration: the grammar makes each spec's own doc
// comment a comment child of the type_declaration, interleaved with the spec siblings, so
// CommentBlockAbove's prev-sibling walk works unchanged one level down from the ungrouped case. The
// rendered signature is "type " prepended to the spec's own signature text, so the grouped and
// ungrouped forms produce identical output for identical types. A comment on the "type (" line
// itself is a prev-sibling of decl, not of any spec, so it is never picked up here — that is this
// rule falling out naturally, not a special case.
//
// HeadStart and HeadEnd are set exactly as goUngroupedTypeSymbol's doc comment describes.
func goGroupedTypeSymbol(unit string, spec *ts.Node, src []byte) (sym Symbol, ok bool) {
	name := NodeText(spec.ChildByFieldName("name"), src)
	if goBlank(name) {
		return Symbol{}, false
	}
	body := goTypeBody(spec)
	docComment, raw := CommentBlockAbove(spec, src)
	start, _ := Line(spec)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(spec)
	declStart, bodyStart, declEnd := goDeclOffsets(spec, body)
	g := glyph.Glyph{Lang: glyph.Go, Unit: unit, Name: name}
	return Symbol{
		Glyph:     g,
		ID:        g.String(),
		Kind:      KindType,
		Signature: "type " + SignatureCut(spec, body, src),
		Doc:       StripComment(raw, "//"),
		Start:     start,
		SigEnd:    SigEnd(spec, body),
		End:       end,
		HeadStart: start,
		HeadEnd:   end,
		DeclStart: declStart,
		BodyStart: bodyStart,
		DeclEnd:   declEnd,
	}, true
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

// goConstSymbols builds the KindConst Symbol(s) for one const_declaration node, covering the
// ungrouped ("const X = 1") and grouped ("const ( ... )") shapes. Confirmed against the pinned
// tree-sitter-go v0.25.0: a const_declaration's spec children — const_spec — sit directly among its
// own children in both shapes, exactly like a type_declaration's, so goDeclIsGrouped's literal-"("-
// child test tells them apart here for the same reason it does there: a single-spec group,
// "const ( X = 1 )", is legal Go, and a spec-count test would misroute it to the ungrouped path.
func goConstSymbols(unit string, decl *ts.Node, src []byte) []Symbol {
	if !goDeclIsGrouped(decl) {
		return goUngroupedConstOrVarSymbols(unit, KindConst, decl, goConstOrVarSpecChild(decl, "const_spec"), src)
	}
	var symbols []Symbol
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		spec := decl.NamedChild(i)
		if spec.Kind() != "const_spec" {
			continue
		}
		symbols = append(symbols, goGroupedConstOrVarSymbols(unit, KindConst, spec, src)...)
	}
	return symbols
}

// goVarSymbols builds the KindVar Symbol(s) for one var_declaration node, covering the ungrouped
// ("var x int") and grouped ("var ( ... )") shapes.
//
// Unlike const_declaration, a var_declaration's grouped specs are not direct children of decl:
// confirmed against the pinned tree-sitter-go v0.25.0, the grammar wraps them in an intermediate
// var_spec_list child instead, whose own children carry the "(", the var_spec siblings, and the
// ")". goDeclIsGrouped's "("-scan therefore cannot run on decl itself here — it would never find
// the "(", which sits one level down — so grouping is told apart by that wrapper node's presence
// among decl's named children instead. A var_spec_list only ever appears for the grouped form,
// so this is the same single-spec-safe test as goDeclIsGrouped's, keyed on a different node the
// grammar itself provides for exactly this purpose.
func goVarSymbols(unit string, decl *ts.Node, src []byte) []Symbol {
	if list := goVarSpecList(decl); list != nil {
		var symbols []Symbol
		for i := uint(0); i < list.NamedChildCount(); i++ {
			spec := list.NamedChild(i)
			if spec.Kind() != "var_spec" {
				continue
			}
			symbols = append(symbols, goGroupedConstOrVarSymbols(unit, KindVar, spec, src)...)
		}
		return symbols
	}
	return goUngroupedConstOrVarSymbols(unit, KindVar, decl, goConstOrVarSpecChild(decl, "var_spec"), src)
}

// goVarSpecList returns decl's var_spec_list child — present only for the grouped "var ( ... )"
// form — or nil for the ungrouped form, whose single spec is a var_spec child of decl directly.
func goVarSpecList(decl *ts.Node) *ts.Node {
	return goConstOrVarSpecChild(decl, "var_spec_list")
}

// goConstOrVarSpecChild returns decl's single named child of kind kind, or nil when it has none.
// It is called with "const_spec" for an ungrouped const_declaration, "var_spec" for an ungrouped
// var_declaration, and "var_spec_list" to detect the grouped var_declaration shape.
func goConstOrVarSpecChild(decl *ts.Node, kind string) *ts.Node {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		if c := decl.NamedChild(i); c.Kind() == kind {
			return c
		}
	}
	return nil
}

// goSpecNames returns every identifier under spec's "name" field, in source order: one name for
// "const x = 1" or "var x int", several for "const a, b = 1, 2" — the grammar's "name" field holds
// multiple identifier children for exactly this shape, on both const_spec and var_spec. The field
// also carries the "," tokens separating those identifiers, tagged with the same field name, so
// every non-identifier (unnamed) child is skipped here; without that filter each "," would surface
// as a spurious, blank-adjacent extra name.
func goSpecNames(spec *ts.Node, src []byte) []string {
	nodes := spec.ChildrenByFieldName("name", spec.Walk())
	names := make([]string, 0, len(nodes))
	for i := range nodes {
		if !nodes[i].IsNamed() {
			continue
		}
		names = append(names, NodeText(&nodes[i], src))
	}
	return names
}

// goUngroupedConstOrVarSymbols builds one Symbol per non-blank name in spec — decl's single,
// ungrouped const_spec or var_spec — sharing decl's own span, doc, and signature: Start/End and the
// docstring come from CommentBlockAbove on decl, since decl, not spec, is the node with
// source_file siblings, and Signature is decl's own text trimmed whole, which carries the "const"
// or "var" keyword because decl's span already starts there. SigEnd and HeadStart/HeadEnd stay
// zero: none of these shapes has a body-bearing child, and head spans are populated for KindType
// only. Several names sharing one spec produce distinct glyphs over identical spans, which needs no
// special case.
func goUngroupedConstOrVarSymbols(unit string, kind Kind, decl, spec *ts.Node, src []byte) []Symbol {
	docComment, raw := CommentBlockAbove(decl, src)
	start, _ := Line(decl)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(decl)
	signature := SignatureCut(decl, nil, src)
	doc := StripComment(raw, "//")
	declStart, bodyStart, declEnd := goDeclOffsets(decl, nil)

	var symbols []Symbol
	for _, name := range goSpecNames(spec, src) {
		if goBlank(name) {
			continue
		}
		g := glyph.Glyph{Lang: glyph.Go, Unit: unit, Name: name}
		symbols = append(symbols, Symbol{
			Glyph:     g,
			ID:        g.String(),
			Kind:      kind,
			Signature: signature,
			Doc:       doc,
			Start:     start,
			End:       end,
			DeclStart: declStart,
			BodyStart: bodyStart,
			DeclEnd:   declEnd,
		})
	}
	return symbols
}

// goGroupedConstOrVarSymbols builds one Symbol per non-blank name in spec — one const_spec or
// var_spec inside a grouped "const ( ... )" or "var ( ... )" declaration — sharing spec's own span
// and doc, mirroring goGroupedTypeSymbol's rule: Signature is the declaration's keyword prepended to
// spec's own signature text, verbatim, never synthesised from a preceding spec even for a bare
// `iota` spec with no type and no value of its own. SigEnd and HeadStart/HeadEnd stay zero, for the
// same reason goUngroupedConstOrVarSymbols' do.
func goGroupedConstOrVarSymbols(unit string, kind Kind, spec *ts.Node, src []byte) []Symbol {
	docComment, raw := CommentBlockAbove(spec, src)
	start, _ := Line(spec)
	if docComment != nil {
		start, _ = Line(docComment)
	}
	_, end := Line(spec)
	keyword := "const "
	if kind == KindVar {
		keyword = "var "
	}
	signature := keyword + SignatureCut(spec, nil, src)
	doc := StripComment(raw, "//")
	declStart, bodyStart, declEnd := goDeclOffsets(spec, nil)

	var symbols []Symbol
	for _, name := range goSpecNames(spec, src) {
		if goBlank(name) {
			continue
		}
		g := glyph.Glyph{Lang: glyph.Go, Unit: unit, Name: name}
		symbols = append(symbols, Symbol{
			Glyph:     g,
			ID:        g.String(),
			Kind:      kind,
			Signature: signature,
			Doc:       doc,
			Start:     start,
			End:       end,
			DeclStart: declStart,
			BodyStart: bodyStart,
			DeclEnd:   declEnd,
		})
	}
	return symbols
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
