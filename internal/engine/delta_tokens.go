// delta_tokens.go holds the token stream machinery every delta classification tier is built on: the
// per-symbol signature and body token streams extracted by byte range from one walk of a file's
// parse tree, the identical-modulo-the-renamed-identifier predicate over two streams, and the body
// token similarity signal. Nothing here compares two symbol tables — that is delta.go's job — so
// this file has no notion of created, deleted, modified or renamed.

package engine

import (
	ts "github.com/tree-sitter/go-tree-sitter"
)

// token is one leaf node's kind and text, both copied into plain strings so the value owns its own
// memory. treesitter.WithTree's own doc comment forbids retaining a node past its callback's
// return, so a stream holding *ts.Node pointers would be reading freed memory by the time it is
// compared; copying kind and text into a token is what makes a stream safe to hold afterwards.
type token struct {
	kind string
	text string
}

// tokenStream is an ordered sequence of tokens, in source order.
type tokenStream []token

// symbolStreams is one symbol's signature and body token streams.
type symbolStreams struct {
	signature tokenStream
	body      tokenStream
}

// tokenStreamsForSymbols walks root's leaves exactly once, in source order, and returns, per
// symbol, its signature stream and its body stream — indexed the same way symbols is, so
// streams[i] belongs to symbols[i].
//
// Each leaf is assigned to every symbol range that contains it, never to just one: symbol spans
// nest and overlap by design, so an interface method element's leaves belong to the interface
// type's body stream and to that method symbol's own signature stream at the same time, and the
// several symbols declared by one const spec share one span verbatim and therefore share one
// identical stream. A leaf belongs to a symbol's signature stream when its start byte lies in the
// half-open range [DeclStart, BodyStart), and to that symbol's body stream when its start byte lies
// in the half-open range [BodyStart, DeclEnd). A symbol whose BodyStart equals its DeclEnd
// therefore has an empty body stream, which is the intended outcome for every const, var, type
// alias and interface method element.
//
// A leaf is a node with no children. Anonymous leaves are included, not only named ones —
// operators, keywords and punctuation are anonymous in the Go grammar, and a stream restricted to
// named leaves would omit every one of them, which would make two bodies differing only in an
// operator compare equal. Whitespace and line numbers contribute nothing, and neither stream ever
// includes a doc block, since the declaration node does not span it.
//
// This function walks the tree once and assigns leaves to every containing symbol range in that
// same walk; it never walks the tree a second time and never looks a node up per symbol.
func tokenStreamsForSymbols(root *ts.Node, src []byte, symbols []Symbol) []symbolStreams {
	streams := make([]symbolStreams, len(symbols))
	walkLeaves(root, func(leaf *ts.Node) {
		start := int(leaf.StartByte())
		tok := token{kind: leaf.Kind(), text: NodeText(leaf, src)}
		for i := range symbols {
			sym := &symbols[i]
			if start >= sym.DeclStart && start < sym.BodyStart {
				streams[i].signature = append(streams[i].signature, tok)
			}
			if start >= sym.BodyStart && start < sym.DeclEnd {
				streams[i].body = append(streams[i].body, tok)
			}
		}
	})
	return streams
}

// walkLeaves visits every leaf (a node with no children) under n, in source order, including
// anonymous nodes — n.Child(i) walks every child regardless of named status, unlike
// n.NamedChild(i).
func walkLeaves(n *ts.Node, visit func(*ts.Node)) {
	if n.ChildCount() == 0 {
		visit(n)
		return
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		walkLeaves(n.Child(i), visit)
	}
}

// identifierKind is the tree-sitter-go node kind an identifier leaf carries. It is the only kind
// identicalModuloName's substitution rule ever admits.
const identifierKind = "identifier"

// identicalModuloName reports whether a and b are identical modulo the renamed identifier: they
// have the same length and, at every position, either the two tokens are equal in both kind and
// text, or both tokens are identifier nodes whose texts are respectively deletedName and
// createdName.
//
// This is an exact structural test: there is no threshold, no tuning knob and no partial credit.
// The substitution is restricted to identifier nodes carrying exactly those two names, which is
// what keeps it exact — it admits the recursive self-call a renamed function makes and nothing
// else. An anonymous node can never be substituted, since it is not an identifier node, so
// including anonymous leaves in the stream (tokenStreamsForSymbols) widens what is compared
// without widening what may be substituted; that is the property that makes this predicate safe
// against a stream that includes every operator, keyword and punctuation token.
//
// The predicate is used for both the body streams and the signature streams under the same rule,
// and never as a textual substitution over a signature string: replacing the text "Run" with
// "Execute" inside a method head would also hit the "Runner" in its receiver, while a stream has
// real identifier nodes to key on.
// renamedIdentifierPlaceholder replaces an identifier token bearing a symbol's own name before the
// two body streams are turned into multisets for bodyTokenSimilarity. It is chosen so it can never
// collide with a real Go identifier's text: a NUL byte is not a legal identifier rune, and no source
// text tokenStreamsForSymbols hands back can contain one.
const renamedIdentifierPlaceholder = "\x00renamed\x00"

// bodyTokenSimilarity returns the body token similarity signal for two body streams and the two
// symbol names: the Jaccard coefficient of the two streams treated as multisets of (kind, text)
// pairs, with the identifier bearing the symbol's own name normalised to
// renamedIdentifierPlaceholder on both sides before the multisets are built. Two empty streams
// give exactly 1.0. The result is always in the closed interval [0, 1].
//
// This value is a reported signal on a candidate quarry has explicitly declined to resolve — no
// asserted outcome anywhere in the delta reads it. It is deliberately a cheap linear-time
// coefficient rather than an order-sensitive metric, because precision here would buy nothing
// quarry is allowed to spend it on.
func bodyTokenSimilarity(before, after tokenStream, beforeName, afterName string) float64 {
	beforeCounts := multisetCounts(before, beforeName)
	afterCounts := multisetCounts(after, afterName)
	if len(beforeCounts) == 0 && len(afterCounts) == 0 {
		return 1.0
	}

	var intersection, union int
	seen := make(map[token]bool, len(beforeCounts)+len(afterCounts))
	for tok, count := range beforeCounts {
		other := afterCounts[tok]
		intersection += min(count, other)
		union += max(count, other)
		seen[tok] = true
	}
	for tok, count := range afterCounts {
		if seen[tok] {
			continue
		}
		union += count
	}
	return float64(intersection) / float64(union)
}

// multisetCounts counts stream's tokens by (kind, text), after normalising any identifier token
// bearing name to renamedIdentifierPlaceholder.
func multisetCounts(stream tokenStream, name string) map[token]int {
	counts := make(map[token]int, len(stream))
	for _, tok := range stream {
		if tok.kind == identifierKind && tok.text == name {
			tok = token{kind: identifierKind, text: renamedIdentifierPlaceholder}
		}
		counts[tok]++
	}
	return counts
}

func identicalModuloName(a, b tokenStream, deletedName, createdName string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].kind == b[i].kind && a[i].text == b[i].text {
			continue
		}
		if a[i].kind == identifierKind && b[i].kind == identifierKind &&
			a[i].text == deletedName && b[i].text == createdName {
			continue
		}
		return false
	}
	return true
}
