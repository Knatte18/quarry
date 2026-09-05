// pack.go generates the kick-start pack a pack cell's card carries: RenderKickstartPack turns one
// batched resolve into the two-line-per-target block a card's sentinel-delimited region holds, and
// the sentinel-delimited read/write protocol below is what lets run's pre-rep-1 gate verify the pack
// sitting in the prompt is the pack the provenance record says it is.
//
// The sentinels mark exactly the substitutable region of a pack cell's card: PackBlockSHA256 is the
// one hash function both the writer here and the run-time gate call, so the two can never drift into
// computing the hash two different ways.

package ladder

import (
	"fmt"
	"strings"

	"github.com/Knatte18/quarry/quarry"
)

// RenderKickstartPack renders results -- one batched (*quarry.Repo).Resolve call's answer, in the
// slice's own order, which is the ladder file's pack_targets order since the facade answers
// positionally -- into the kick-start pack block: two lines per result,
//
//	<target> → <file> <start>-<end>
//	    <signature>
//
// The first line's fields are the result's own Target verbatim, its single symbol's
// repository-relative file, and that symbol's start and end lines joined by a hyphen. The second
// line is indented by exactly four spaces and carries the symbol's signature with its internal
// newlines collapsed: split on newlines, each resulting line trimmed of its surrounding whitespace,
// then joined with a single space. The rendered block ends with no trailing newline.
//
// The docstring is never emitted, and neither is the signature-end line: this is the pack's
// treatment definition, not a formatting preference. Emitting the docstring would turn "the agent
// knows where things are" into "the agent knows where things are and has a slab of prose", which
// would make a measured win uninterpretable. The existing text-view renderer in quarry/text.go emits
// the docstring, which is exactly why this function exists instead of a call to it.
//
// Any result whose status is not StatusFound, and any result carrying a non-empty pre-resolution
// Error, is fatal: RenderKickstartPack returns an error naming the offending target and what came
// back for it, with no partial output -- a pack missing one glyph is a different treatment from the
// one the cards describe. A found result carries exactly one symbol; a found result with no symbols
// is the same class of fatal error, checked explicitly rather than indexing into an empty slice.
func RenderKickstartPack(results []quarry.ResolveResult) (string, error) {
	lines := make([]string, 0, len(results)*2)
	for _, r := range results {
		if r.Error != "" {
			return "", fmt.Errorf("render kickstart pack: target %q: pre-resolution error: %s", r.Target, r.Error)
		}
		if r.Status != quarry.StatusFound {
			return "", fmt.Errorf("render kickstart pack: target %q: status %q, want %q", r.Target, r.Status, quarry.StatusFound)
		}
		if len(r.Symbols) != 1 {
			return "", fmt.Errorf("render kickstart pack: target %q: found result carries %d symbols, want exactly 1", r.Target, len(r.Symbols))
		}
		sym := r.Symbols[0]
		lines = append(lines, fmt.Sprintf("%s → %s %d-%d", r.Target, sym.File, sym.Start, sym.End))
		lines = append(lines, "    "+collapseSignature(sym.Signature))
	}
	return strings.Join(lines, "\n"), nil
}

// collapseSignature splits sig on newlines, trims each resulting line's surrounding whitespace, and
// joins the result with a single space -- the treatment RenderKickstartPack's doc comment describes
// for a multi-line signature.
func collapseSignature(sig string) string {
	parts := strings.Split(sig, "\n")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, " ")
}
