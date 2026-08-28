// python.go implements the Python Strategy: function, method, and class extraction, and the Python
// header, generated-file, and test-file rules. It registers itself under the canonical language name
// "python" from this file's own init.

package toc

import (
	ts "github.com/tree-sitter/go-tree-sitter"
)

// pythonStrategy implements Strategy for Python source parsed by the pinned tree-sitter-python
// v0.25.0 grammar.
type pythonStrategy struct{}

func init() {
	Register(pythonStrategy{})
}

// Language implements Strategy by returning "python".
func (pythonStrategy) Language() string {
	return "python"
}

// Symbols implements Strategy. It walks the module root's direct children and, for each
// class_definition found there, descends into its body field (a block) and walks that block's
// children too — the one level of container descent Python needs, since every method is nested
// inside a class block. It never descends into a function_definition's own body, so a nested
// helper declared inside a function is never listed.
func (pythonStrategy) Symbols(root *ts.Node, src []byte) []Symbol {
	symbols := make([]Symbol, 0)
	for i := uint(0); i < root.ChildCount(); i++ {
		symbols = append(symbols, pythonModuleSymbols(root.Child(i), src)...)
	}
	return symbols
}

// pythonModuleSymbols builds the Symbol(s) a single module-level child contributes: zero for
// anything that is not a (possibly decorated) function_definition or class_definition, one for a
// module-level function, and one-plus-its-methods for a class.
func pythonModuleSymbols(child *ts.Node, src []byte) []Symbol {
	decl, outer := unwrapDecorated(child)
	switch decl.Kind() {
	case "function_definition":
		return []Symbol{pythonFunctionSymbol(decl, outer, "", KindFunction, src)}
	case "class_definition":
		return pythonClassSymbols(decl, outer, src)
	default:
		return nil
	}
}

// unwrapDecorated unwraps a decorated_definition so the kind, name, signature, and docstring are
// read from the wrapped definition while the emitted range still covers the decorator line.
//
// When n.Kind() is "decorated_definition", decl is its "definition" field child and outer is n
// itself — decl is what every other extraction rule reads from, and outer is what the range is
// measured from, so a decorator line stays inside the emitted range. When n is not a
// decorated_definition, both return values are n: there is nothing to unwrap.
//
// Skipping this unwrap is the failure mode this helper exists to prevent: without it, every
// decorated function or class in a file is silently dropped, since neither's Kind() is
// "function_definition" or "class_definition" until decorated_definition is peeled off.
func unwrapDecorated(n *ts.Node) (decl *ts.Node, outer *ts.Node) {
	if n.Kind() == "decorated_definition" {
		return n.ChildByFieldName("definition"), n
	}
	return n, n
}

// pythonClassSymbols builds the KindType Symbol for decl (a class_definition) followed by a
// KindMethod Symbol for each function_definition found as a direct child of decl's body block —
// the container descent that reaches Python's methods, since they are nested one block deep.
func pythonClassSymbols(decl, outer *ts.Node, src []byte) []Symbol {
	name := NodeText(decl.ChildByFieldName("name"), src)
	symbols := []Symbol{pythonDeclSymbol(KindType, name, "", decl, outer, src)}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return symbols
	}
	for i := uint(0); i < body.ChildCount(); i++ {
		memberDecl, memberOuter := unwrapDecorated(body.Child(i))
		if memberDecl.Kind() != "function_definition" {
			continue
		}
		symbols = append(symbols, pythonFunctionSymbol(memberDecl, memberOuter, name, KindMethod, src))
	}
	return symbols
}

// pythonFunctionSymbol builds the Symbol for a function_definition (module-level or a class
// method), with kind and owner supplied by the caller since the same node shape serves both.
func pythonFunctionSymbol(decl, outer *ts.Node, owner string, kind Kind, src []byte) Symbol {
	name := NodeText(decl.ChildByFieldName("name"), src)
	return pythonDeclSymbol(kind, name, owner, decl, outer, src)
}

// pythonDeclSymbol builds the Symbol shared by a class_definition and a function_definition: the
// signature and sigend come from the shared node helpers with bodyOnSignatureLine = false, since
// Python's block starts on the line after the "def ...:" or "class ...:" header rather than sharing
// its line the way Go's "{" does.
//
// The range needs no docstring adjustment, unlike Go and C#: Start is outer's first line and End is
// outer's last line, because Python's docstring is the first statement inside the definition's own
// body block, so it already falls within the declaration node's own span. This is the one place
// Python's rule differs from Go's and C#'s sibling-comment adjustment — worth stating here because a
// reader who has just written those two will reach for that adjustment by reflex.
func pythonDeclSymbol(kind Kind, name, owner string, decl, outer *ts.Node, src []byte) Symbol {
	body := decl.ChildByFieldName("body")
	start, _ := Line(outer)
	_, end := Line(outer)
	return Symbol{
		Kind:      kind,
		Name:      name,
		Owner:     owner,
		Signature: SignatureCut(decl, body, src),
		Docstring: pythonDocstring(body, src),
		Start:     start,
		SigEnd:    SigEnd(decl, body, false),
		End:       end,
	}
}

// pythonDocstring returns body's docstring — the delimiter-stripped, normalised text of its first
// statement's string content, when that first statement is an expression_statement whose only named
// child is a string. It returns "" when body is nil (a bodyless definition, which the Python grammar
// never actually produces, but the check keeps this function total) or when the first statement is
// anything else.
//
// Normalisation is StripLineComment(text, ""): with an empty prefix that function performs exactly
// the per-line trim-join-trim rule and nothing else, which is deliberately the same code Go's "//"
// and C#'s "///" stripping runs, just with no prefix to remove. A PEP 257 docstring is indented to
// its enclosing "def" or "class", so a single whole-text TrimSpace would leave every line but the
// first carrying that indentation; the shared "docstrings keep the prose and drop the syntax" rule
// requires Python to come out shaped the same as Go and C#. The deliberate consequence: indentation
// inside a docstring's code example is not preserved, for Python no more and no less than for the
// other two languages.
func pythonDocstring(body *ts.Node, src []byte) string {
	if body == nil || body.NamedChildCount() == 0 {
		return ""
	}
	first := body.NamedChild(0)
	if first.Kind() != "expression_statement" || first.NamedChildCount() != 1 {
		return ""
	}
	str := first.NamedChild(0)
	if str.Kind() != "string" {
		return ""
	}
	content := pythonStringContent(str)
	if content == nil {
		return ""
	}
	return StripLineComment(NodeText(content, src), "")
}

// pythonStringContent returns str's (a "string" node) "string_content" child — the text between its
// string_start and string_end delimiters — or nil when str has none (an empty string literal).
func pythonStringContent(str *ts.Node) *ts.Node {
	for i := uint(0); i < str.NamedChildCount(); i++ {
		c := str.NamedChild(i)
		if c.Kind() == "string_content" {
			return c
		}
	}
	return nil
}

// Package implements Strategy by always returning "". Python has no package clause inside a source
// file — its package identity comes from the directory layout and __init__.py, a filesystem fact
// rather than something the file itself declares. This deliberately does not synthesize a name from
// the filename or the parent directory: this field reports what the file itself declares, and a
// synthesized value would be indistinguishable from a declared one.
func (pythonStrategy) Package(root *ts.Node, src []byte) string {
	return ""
}

// Header implements Strategy by preferring the module docstring — the same node shape as a function
// or class docstring, one level up, read via pythonDocstring(root, src) since the module root's
// direct children serve as its own "body" for this purpose.
//
// When the module has no docstring, Header falls back to the leading comment blocks: it walks
// LeadingBlocks, strips each with StripLineComment(b.Raw, "#"), and returns the first block for
// which IsDirectiveBlock("python", ...) is false. That fallback is what makes the shebang and PEP
// 263 coding-line cases of IsDirectiveBlock reachable, and it means a file with both a shebang and a
// module docstring returns the docstring rather than the shebang.
//
// Header returns the text untruncated; FirstParagraph is applied by the entry points.
func (pythonStrategy) Header(root *ts.Node, src []byte) string {
	if doc := pythonDocstring(root, src); doc != "" {
		return doc
	}
	for _, block := range LeadingBlocks(root, src) {
		stripped := StripLineComment(block.Raw, "#")
		if !IsDirectiveBlock("python", block.StartLine, stripped) {
			return stripped
		}
	}
	return ""
}

// Generated implements Strategy by always reporting known == false. Python has no generated-file
// banner convention, so the caller must omit the "generated" key entirely rather than emit it as
// false.
func (pythonStrategy) Generated(root *ts.Node, src []byte) (generated, known bool) {
	return false, false
}

// TestFile implements Strategy by delegating to TestFileByName("python", base).
func (pythonStrategy) TestFile(base string) (isTest, known bool) {
	return TestFileByName("python", base)
}
