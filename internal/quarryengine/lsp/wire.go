// wire.go implements the LSP wire shapes for text positions, ranges, and locations, plus the
// conversion to/from the caller-facing quarryengine.Position.
// It is ported from the recovered tools/scout-poc/gopls.go harness (git show 3b4dcf86).

package lsp

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf16"

	"github.com/Knatte18/quarry/internal/quarryengine"
)

// Position is the LSP wire shape for a text position: a zero-based line
// and a UTF-16 code-unit offset into that line (not a byte or rune offset —
// LSP mandates UTF-16 regardless of the server's internal encoding).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is the LSP wire shape for a half-open [Start, End) text range.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is the LSP wire shape for one reference result: a document URI
// plus the range within it.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// ToPosition converts a caller-supplied quarryengine.Position (1-based line, 1-based
// byte column) to the LSP wire position (0-based line, 0-based UTF-16
// code-unit offset). The conversion re-reads p.File because the byte column
// and LSP's UTF-16 offset only coincide for pure-ASCII lines; any
// multi-byte rune before the target column would otherwise misalign the
// position the server is asked to query.
func ToPosition(p quarryengine.Position) (Position, error) {
	data, err := os.ReadFile(p.File)
	if err != nil {
		return Position{}, fmt.Errorf("read %s: %w", p.File, err)
	}

	lines := strings.Split(string(data), "\n")
	if p.Line < 1 || p.Line > len(lines) {
		return Position{}, fmt.Errorf("line %d out of range in %s (%d lines)", p.Line, p.File, len(lines))
	}

	line := lines[p.Line-1]
	byteCol := p.Character - 1
	if byteCol < 0 {
		byteCol = 0
	}
	if byteCol > len(line) {
		byteCol = len(line)
	}

	return Position{Line: p.Line - 1, Character: utf16Length(line[:byteCol])}, nil
}

// utf16Length returns the number of UTF-16 code units s would occupy on the
// wire, the unit LSP positions are always expressed in regardless of the
// server's internal string representation.
func utf16Length(s string) int {
	n := 0
	for _, r := range s {
		if units := utf16.RuneLen(r); units > 0 {
			n += units
		} else {
			// RuneLen reports -1 for a rune it cannot encode (e.g. an
			// unpaired surrogate); count it as one unit rather than drop
			// it, so the running offset never falls behind the true
			// position.
			n++
		}
	}
	return n
}

// FormatLocation renders loc as a 1-based "file:line:col" string for
// display: the file:// URI scheme is stripped back to a plain filesystem
// path, and the LSP's 0-based line/UTF-16-character position is converted
// to a 1-based line/column, the inverse direction of ToPosition.
func FormatLocation(loc Location) string {
	path := strings.TrimPrefix(loc.URI, "file://")
	return fmt.Sprintf("%s:%d:%d", path, loc.Range.Start.Line+1, loc.Range.Start.Character+1)
}
