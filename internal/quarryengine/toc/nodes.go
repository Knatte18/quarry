// nodes.go holds the language-independent tree-sitter node helpers every strategy shares: verbatim
// text extraction, 1-based line conversion, signature cutting, and the two comment-block walks a
// docstring rule and a header rule each need. Nothing here is specific to one language; the
// per-language shape decisions live in each strategy's own file.

package toc

import (
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// NodeText returns the verbatim source span n covers, untrimmed. Every helper in this file that
// needs source text goes through this one function so the byte-range slicing has exactly one
// implementation.
func NodeText(n *ts.Node, src []byte) string {
	return string(src[n.StartByte():n.EndByte()])
}

// Line returns n's 1-based, inclusive start and end line numbers. tree-sitter's own Point.Row is
// 0-based, so every 1-based line number this package emits goes through this one function — the
// 0-based-to-1-based conversion has exactly one implementation.
func Line(n *ts.Node) (start, end int) {
	return int(n.StartPosition().Row) + 1, int(n.EndPosition().Row) + 1
}

// SignatureCut returns decl's signature: the trimmed source text from decl's first byte to body's
// start byte, when body is non-nil, or the trimmed whole of decl's own text when body is nil.
//
// The cut is a byte range, never a line truncation: a multi-line parameter list is part of the
// signature and is returned whole, exactly as the plan's verbatim-signature decision requires.
func SignatureCut(decl *ts.Node, body *ts.Node, src []byte) string {
	if body != nil {
		return strings.TrimSpace(string(src[decl.StartByte():body.StartByte()]))
	}
	return strings.TrimSpace(NodeText(decl, src))
}

// SigEnd returns the last line of decl's signature: the single implementation of the sigend
// derivation every strategy calls rather than reimplementing.
//
// body == nil means decl has no body-bearing child at all — a type alias, for instance — so there
// is no separate "signature end" to report, and SigEnd returns 0, the absent marker Symbol.SigEnd
// relies on to omit the emitted key.
//
// Otherwise SigEnd is body's own start line, since Go's body-opening "{" sits on the signature's
// last line.
//
// Known imprecision, not a defect: a single-line Go function shares one line between its signature
// and its body, so start-sigend includes the body there. No line-based range can separate them; the
// fix is help text, not columns.
func SigEnd(decl *ts.Node, body *ts.Node) int {
	if body == nil {
		return 0
	}
	return int(body.StartPosition().Row) + 1
}

// CommentBlockAbove walks n's PrevSibling chain backwards over contiguous "comment" nodes, stopping
// at the first non-comment sibling or at the first blank line — detected as
// prev.EndPosition().Row+1 != cur.StartPosition().Row, i.e. prev's last line is not immediately
// followed by cur's first line. It returns the topmost comment node of the resulting block and the
// raw source of its lines joined with "\n", or (nil, "") when n has no adjacent comment at all.
//
// The walk starts at n itself, so the very first adjacency check is between n and its immediate
// PrevSibling: a comment separated from n by a blank line never enters the block, which is what
// keeps a trailing comment left over from the previous declaration from being misattributed to this
// one. The same loop then keeps walking comment-to-comment, so it doubles as the grouped
// type-declaration walk one level down: a type_spec's PrevSibling chain over interleaved comment and
// type_spec children works unchanged.
//
// The header rule (LeadingBlocks) deliberately differs from this rule by tolerating one blank line —
// say so at each call site that relies on the distinction.
func CommentBlockAbove(n *ts.Node, src []byte) (first *ts.Node, raw string) {
	var comments []*ts.Node
	cur := n
	for {
		prev := cur.PrevSibling()
		if prev == nil || prev.Kind() != "comment" {
			break
		}
		if prev.EndPosition().Row+1 != cur.StartPosition().Row {
			break
		}
		comments = append(comments, prev)
		cur = prev
	}
	if len(comments) == 0 {
		return nil, ""
	}
	// comments was collected nearest-to-n first; reverse it into source order.
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}
	lines := make([]string, len(comments))
	for i, c := range comments {
		lines[i] = NodeText(c, src)
	}
	return comments[0], strings.Join(lines, "\n")
}

// CommentBlock is one group of contiguous leading comment children of a root node, as grouped by
// LeadingBlocks.
type CommentBlock struct {
	// First is the block's topmost comment node.
	First *ts.Node
	// Raw is the raw source of every comment line in the block, joined with "\n" — joined exactly
	// the way CommentBlockAbove's raw is, so one StripComment call behaves identically on both.
	Raw string
	// StartLine is the block's 1-based starting line: First.StartPosition().Row + 1, returned rather
	// than recomputed at each call site so the 0-based-to-1-based conversion lives in one place.
	StartLine int
}

// LeadingBlocks returns root's leading "comment" children, grouped into blocks by the same
// blank-line rule CommentBlockAbove uses, in source order, stopping at the first non-comment child
// of root.
//
// This is what every strategy's header rule iterates to skip directive blocks (a build constraint,
// a generate directive, a shebang) and find the file's real header. The block text and its start line are
// part of this helper's contract, not the callers': CommentBlockAbove only walks upward from a
// declaration, so a caller handed bare comment nodes would have to reinvent this downward sibling
// walk once per strategy — exactly the per-strategy helper duplication the plan's shared-helper rule
// exists to prevent.
func LeadingBlocks(root *ts.Node, src []byte) []CommentBlock {
	var blocks []CommentBlock
	var current []*ts.Node
	for i := uint(0); i < root.ChildCount(); i++ {
		c := root.Child(i)
		if c.Kind() != "comment" {
			break
		}
		if len(current) > 0 {
			prev := current[len(current)-1]
			if prev.EndPosition().Row+1 != c.StartPosition().Row {
				blocks = append(blocks, newCommentBlock(current, src))
				current = nil
			}
		}
		current = append(current, c)
	}
	if len(current) > 0 {
		blocks = append(blocks, newCommentBlock(current, src))
	}
	return blocks
}

// newCommentBlock builds the CommentBlock for one contiguous run of comment nodes, already known
// non-empty.
func newCommentBlock(nodes []*ts.Node, src []byte) CommentBlock {
	lines := make([]string, len(nodes))
	for i, n := range nodes {
		lines[i] = NodeText(n, src)
	}
	return CommentBlock{
		First:     nodes[0],
		Raw:       strings.Join(lines, "\n"),
		StartLine: int(nodes[0].StartPosition().Row) + 1,
	}
}
