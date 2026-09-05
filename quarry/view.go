// view.go holds the glyphs view: the flat, non-recursive projection of a complete
// table-of-contents answer, plus that view's own two renderers. The projection is applied after
// the query returns, so extraction underneath — internal/engine — stays complete and is never
// reached by a view concept; GlyphView reads a finished DirAnswer and reshapes it, nothing more.

package quarry

import "sort"

// GlyphsAnswer is the glyphs view's projected answer: a flat index of every symbol under a
// target's whole tree, plus the paths that could not be fully extracted. It carries no JSON
// struct tags on any field: the wire shape is glyphsEnvelope, the unexported shadow struct
// RenderGlyphsJSON builds from it, so there is exactly one description of the emitted key set. An
// untagged struct here makes a caller's direct json.Marshal of this value visibly a different
// document from the rendered one, rather than a plausible-looking one that happens to differ.
type GlyphsAnswer struct {
	// Target is the query's own target, echoed verbatim — see GlyphView's doc comment for why no
	// normalisation happens here.
	Target string
	// Symbols is every symbol in the target's whole tree, in depth-first order.
	Symbols []Symbol
	// Incomplete is the path of every file that could not be fully extracted, sorted, nil when
	// there are none. See GlyphView's doc comment for this field's exact scope.
	Incomplete []string
}

// GlyphView projects a, a complete table-of-contents answer for target, into the glyphs view: the
// flat, depth-first list of every symbol in a's whole tree, plus the paths that were only
// partially extracted. GlyphView is pure over a: it reads a and returns a new value, and never
// mutates a or shares a backing array with any slice inside it.
//
// Target is target verbatim, echoed with no normalisation of any kind. The callers normalise — the
// CLI passes the already-repository-relative form, and Glyphs maps an empty target to "." before
// both its query and its echo — so one query never has two spellings in its own answer.
//
// Symbols is every symbol in a's whole tree in depth-first order, matching dirBlocks's own order
// in quarry/text.go: this directory's Files in slice order, each file's *Symbols in slice order,
// then each entry of Dirs in slice order, recursively. A FileEntry whose Symbols pointer is nil
// contributes nothing and is not an error; a non-nil pointer to an empty slice likewise
// contributes nothing. Each contributed symbol is a copy with Doc, Signature and SigEnd set to
// their zero values and File set to joinRel(the enclosing DirAnswer's Dir, the enclosing
// FileEntry's Name), reusing the existing unexported joinRel in quarry/text.go rather than
// re-deriving the rule. Every other field, Glyph included, is carried across untouched.
//
// Incomplete is the joinRel-joined path of every FileEntry in the whole tree whose Error is
// non-empty or whose Lossy is true, sorted, and left nil when there are none. "An absent symbol
// line means the symbol is not in the target" holds for the frozen glyphs preset, which is --depth
// all, and there only — a depth-cut answer is truncated by construction and contributes nothing to
// Incomplete, because a depth-cut DirAnswer with no Files and no Dirs is indistinguishable from a
// genuinely empty leaf directory and this function has nothing to detect.
func GlyphView(target string, a DirAnswer) GlyphsAnswer {
	var symbols []Symbol
	var incomplete []string
	collectGlyphs(a, &symbols, &incomplete)
	sort.Strings(incomplete)
	return GlyphsAnswer{Target: target, Symbols: symbols, Incomplete: incomplete}
}

// collectGlyphs appends a's own symbols and incomplete-file paths onto *symbols and *incomplete,
// in depth-first order — a's own Files, then each entry of a.Dirs in slice order, recursively —
// and is the shared depth-first walk GlyphView reduces the whole tree with.
func collectGlyphs(a DirAnswer, symbols *[]Symbol, incomplete *[]string) {
	for _, fe := range a.Files {
		if fe.Error != "" || fe.Lossy {
			*incomplete = append(*incomplete, joinRel(a.Dir, fe.Name))
		}
		if fe.Symbols == nil {
			continue
		}
		file := joinRel(a.Dir, fe.Name)
		for _, sym := range *fe.Symbols {
			sym.Doc = ""
			sym.Signature = ""
			sym.SigEnd = 0
			sym.File = file
			*symbols = append(*symbols, sym)
		}
	}
	for _, child := range a.Dirs {
		collectGlyphs(child, symbols, incomplete)
	}
}
