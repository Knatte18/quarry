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

// PackSentinelBegin marks the start of a pack cell's card's substitutable region. It is spelled as a
// markdown comment line so it renders invisibly in a rendered card while remaining a literal,
// greppable line in the prompt text handed to the model. It must appear on its own line, exactly
// once, before PackSentinelEnd.
const PackSentinelBegin = "<!-- KICKSTART-PACK:BEGIN -->"

// PackSentinelEnd marks the end of a pack cell's card's substitutable region, for the same reason and
// under the same one-line, exactly-once, after-PackSentinelBegin rule PackSentinelBegin's doc comment
// states.
const PackSentinelEnd = "<!-- KICKSTART-PACK:END -->"

// ExtractPackBlock returns the text strictly between cardText's two sentinel lines -- neither
// sentinel line included -- with leading and trailing blank lines removed and no trailing newline. A
// card missing either sentinel, carrying either more than once, or carrying them in the wrong order,
// is an error naming which condition failed. A card whose block is empty is not an error: that is the
// state an authored card starts in, before the pack is generated into it.
func ExtractPackBlock(cardText string) (string, error) {
	lines := strings.Split(cardText, "\n")
	beginIdx, endIdx, err := findPackSentinels(lines)
	if err != nil {
		return "", fmt.Errorf("extract pack block: %w", err)
	}
	return trimBlankLines(lines[beginIdx+1 : endIdx]), nil
}

// WritePackIntoCard returns cardText with the region between its two sentinels replaced by pack. It
// preserves everything outside the sentinels byte for byte, and it is idempotent: writing the same
// pack twice yields the same text. A card missing its sentinels is an error, never a silent append --
// appending would produce a card whose prompt text disagrees with the hash provenance records, which
// is the one failure this whole mechanism exists to make impossible.
func WritePackIntoCard(cardText, pack string) (string, error) {
	lines := strings.Split(cardText, "\n")
	beginIdx, endIdx, err := findPackSentinels(lines)
	if err != nil {
		return "", fmt.Errorf("write pack into card: %w", err)
	}
	newLines := make([]string, 0, beginIdx+1+1+len(lines)-endIdx)
	newLines = append(newLines, lines[:beginIdx+1]...)
	newLines = append(newLines, pack)
	newLines = append(newLines, lines[endIdx:]...)
	return strings.Join(newLines, "\n"), nil
}

// PackBlockSHA256 returns the hex sha256 of block's bytes, computed by calling the package's existing
// sha256Hex helper rather than by spelling the same computation a second time. Both WritePackIntoCard's
// caller and the run-time gate call this one function, so the two cannot drift.
func PackBlockSHA256(block string) string {
	return sha256Hex(block)
}

// findPackSentinels scans lines for exactly one PackSentinelBegin line and exactly one
// PackSentinelEnd line, in that order, and returns their indices. It returns an error naming which
// condition failed -- missing, duplicated, or out of order -- otherwise.
func findPackSentinels(lines []string) (beginIdx, endIdx int, err error) {
	beginIdx, endIdx = -1, -1
	beginCount, endCount := 0, 0
	for i, line := range lines {
		switch line {
		case PackSentinelBegin:
			beginCount++
			beginIdx = i
		case PackSentinelEnd:
			endCount++
			endIdx = i
		}
	}
	switch {
	case beginCount == 0:
		return 0, 0, fmt.Errorf("missing sentinel %q", PackSentinelBegin)
	case endCount == 0:
		return 0, 0, fmt.Errorf("missing sentinel %q", PackSentinelEnd)
	case beginCount > 1:
		return 0, 0, fmt.Errorf("sentinel %q appears more than once", PackSentinelBegin)
	case endCount > 1:
		return 0, 0, fmt.Errorf("sentinel %q appears more than once", PackSentinelEnd)
	case beginIdx > endIdx:
		return 0, 0, fmt.Errorf("sentinel %q appears after %q", PackSentinelBegin, PackSentinelEnd)
	}
	return beginIdx, endIdx, nil
}

// trimBlankLines joins lines with "\n" after dropping leading and trailing lines that are empty or
// hold only whitespace.
func trimBlankLines(lines []string) string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}
