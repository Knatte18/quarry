// text.go declares RenderText, the lossless text view the MCP transport (T6) will put inside a
// content[].text block: the same information as RenderJSON, with no keys, no defaults, and prose
// left intact. The grammar this file implements is fixed in full by _mill/discussion.md's
// text-view-grammar decision, which this file follows to the character.

package quarry

import (
	"strconv"
	"strings"
)

// RenderText renders a as the lossless text view. targetIsFile selects the file form (one block,
// the target file's own line and symbols) over the directory form (one block per DirAnswer,
// depth-first, blocks separated by exactly one blank line). targetIsFile is authoritative and is
// never inferred from a's shape, because a directory holding exactly one file and no subdirectories
// is indistinguishable from a file target by shape alone — the caller, which knows what it asked
// for, must say. RenderText cannot fail and returns no error; the returned string has no trailing
// whitespace on any line and ends with exactly one "\n".
func RenderText(a DirAnswer, targetIsFile bool) string {
	if targetIsFile {
		var b strings.Builder
		writeFileForm(&b, a)
		return b.String()
	}
	return strings.Join(dirBlocks(a), "\n")
}

// dirBlocks returns one string per DirAnswer in a's tree, in depth-first order — a itself, then
// each entry of a.Dirs in order, recursively. Each returned block already ends with its own single
// trailing "\n"; joining the slice with "\n" (an extra blank line between elements) reproduces the
// grammar's "blocks separated by exactly one blank line, none before the first or after the last".
func dirBlocks(a DirAnswer) []string {
	var b strings.Builder
	writeDirLine1(&b, a)
	if a.Doc != "" {
		b.WriteString(normalizeProse(a.Doc))
		b.WriteString("\n")
	}
	for _, fe := range a.Files {
		writeFileLine(&b, fe)
		writeSymbolLines(&b, fe)
	}
	blocks := []string{b.String()}
	for _, child := range a.Dirs {
		blocks = append(blocks, dirBlocks(child)...)
	}
	return blocks
}

// normalizeProse collapses s's internal newlines and runs of whitespace to single spaces. It is
// applied to every Doc, Header, Signature and FileEntry.Error value before printing. Error is in
// this list because it is an arbitrary os or UTF-8 message quarry does not author, emitted inside a
// bracketed tag, so a multi-line value would break the one-record-per-line property the whole format
// rests on. "Prose intact" means nothing is truncated or dropped, not that source line breaks
// survive.
func normalizeProse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// joinRel joins the repository-relative directory parent with child's base name, using forward
// slashes and treating "." as the repository root. This is the same rule as
// internal/engine/walk.go's own unexported joinRel, re-declared here because that one is unexported
// and this task does not modify the engine.
func joinRel(parent, child string) string {
	if parent == "." {
		return child
	}
	return parent + "/" + child
}

// writeDirLine1 writes a directory block's line 1: a.Dir, then " (package <Package>, <Language>)"
// only when Package is non-empty, with ", <Language>" inside those parentheses only when Language is
// also non-empty, then ", <N> files" only when a.Files is non-empty, using the singular "1 file"
// when there is exactly one.
func writeDirLine1(b *strings.Builder, a DirAnswer) {
	b.WriteString(a.Dir)
	writePackageParen(b, a.Package, a.Language)
	if n := len(a.Files); n > 0 {
		b.WriteString(", ")
		b.WriteString(strconv.Itoa(n))
		if n == 1 {
			b.WriteString(" file")
		} else {
			b.WriteString(" files")
		}
	}
	b.WriteString("\n")
}

// writePackageParen writes " (package <pkg>, <lang>)" when pkg is non-empty, with ", <lang>" inside
// the parentheses only when lang is also non-empty; it writes nothing when pkg is empty. This is the
// one package-facts clause both the directory form's line 1 and the file form's line share.
func writePackageParen(b *strings.Builder, pkg, lang string) {
	if pkg == "" {
		return
	}
	b.WriteString(" (package ")
	b.WriteString(pkg)
	if lang != "" {
		b.WriteString(", ")
		b.WriteString(lang)
	}
	b.WriteString(")")
}

// writeFileLine writes one file entry's line in the directory form: the bare Name, then fileTags,
// then ": " + normalizeProse(Header) when Header is non-empty.
func writeFileLine(b *strings.Builder, fe FileEntry) {
	b.WriteString(fe.Name)
	b.WriteString(fileTags(fe))
	if fe.Header != "" {
		b.WriteString(": ")
		b.WriteString(normalizeProse(fe.Header))
	}
	b.WriteString("\n")
}

// fileTags returns a space-separated run of bracketed markers for fe, each emitted only when its
// underlying field is present, in this fixed order: [test], [generated], [package <Package>],
// [language <Language>], [lossy], [error <normalizeProse(Error)>]. When at least one tag is emitted
// the run is preceded by a single space; when none is, fileTags returns "".
func fileTags(fe FileEntry) string {
	var tags []string
	if fe.Test {
		tags = append(tags, "[test]")
	}
	if fe.Generated {
		tags = append(tags, "[generated]")
	}
	if fe.Package != "" {
		tags = append(tags, "[package "+fe.Package+"]")
	}
	if fe.Language != "" {
		tags = append(tags, "[language "+fe.Language+"]")
	}
	if fe.Lossy {
		tags = append(tags, "[lossy]")
	}
	if fe.Error != "" {
		tags = append(tags, "[error "+normalizeProse(fe.Error)+"]")
	}
	if len(tags) == 0 {
		return ""
	}
	return " " + strings.Join(tags, " ")
}

// writeSymbolLines writes fe's symbol lines, one per element of *fe.Symbols in order, immediately
// after fe's own line and before the next file's line. A nil Symbols writes nothing.
func writeSymbolLines(b *strings.Builder, fe FileEntry) {
	if fe.Symbols == nil {
		return
	}
	for _, sym := range *fe.Symbols {
		writeSymbolLine(b, sym)
	}
}

// writeSymbolLine writes one symbol's line, identical in both the directory and file forms:
// "<Start>-<End>", then " (sig <Start>-<SigEnd>)" only when SigEnd != 0 — the engine's documented
// marker for a symbol with no body, such as a Go type alias, never line zero, since every real line
// number is 1-based — then " <ID>: " + normalizeProse(Signature). When Doc is non-empty, a following
// line of exactly four spaces then normalizeProse(Doc); when it is empty, no line at all.
func writeSymbolLine(b *strings.Builder, sym Symbol) {
	b.WriteString(strconv.Itoa(sym.Start))
	b.WriteString("-")
	b.WriteString(strconv.Itoa(sym.End))
	if sym.SigEnd != 0 {
		b.WriteString(" (sig ")
		b.WriteString(strconv.Itoa(sym.Start))
		b.WriteString("-")
		b.WriteString(strconv.Itoa(sym.SigEnd))
		b.WriteString(")")
	}
	b.WriteString(" ")
	b.WriteString(sym.ID)
	b.WriteString(": ")
	b.WriteString(normalizeProse(sym.Signature))
	b.WriteString("\n")
	if sym.Doc != "" {
		b.WriteString("    ")
		b.WriteString(normalizeProse(sym.Doc))
		b.WriteString("\n")
	}
}

// writeFileForm writes the file-form block: joinRel(a.Dir, fe.Name), then " (package <Package>,
// <Language>)" under the same presence rules writePackageParen applies — these are the enclosing
// directory's own facts, taken from a, not from the entry — then the entry's tags, then ": " +
// normalizeProse(fe.Header) when the header is non-empty, then the entry's symbol lines. fe is
// a.Files[0]. When a.Files is empty — which the engine never produces for a file target —
// writeFileForm emits the directory form's line 1 alone rather than panicking.
func writeFileForm(b *strings.Builder, a DirAnswer) {
	if len(a.Files) == 0 {
		writeDirLine1(b, a)
		return
	}
	fe := a.Files[0]
	b.WriteString(joinRel(a.Dir, fe.Name))
	writePackageParen(b, a.Package, a.Language)
	b.WriteString(fileTags(fe))
	if fe.Header != "" {
		b.WriteString(": ")
		b.WriteString(normalizeProse(fe.Header))
	}
	b.WriteString("\n")
	writeSymbolLines(b, fe)
}
