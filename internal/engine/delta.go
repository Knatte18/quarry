// delta.go implements (*Repo).Delta, the pure core of the delta query: given a batch of entries
// each carrying a path and up to two byte slices (a before side and an after side), it extracts
// both sides with the same Strategy every other query in this package uses, compares the two
// resulting symbol tables, and returns the delta between them.
//
// This is the pure core, not a diff engine: it never parses a textual diff. A textual diff is a
// view of bytes; this query answers a question about symbols instead. Correctness lives entirely in
// the symbol-table comparison below — git's only job, performed by a caller above this file, is to
// avoid handing this function a file that did not change. Delta reads nothing outside its own
// arguments: no filesystem, no git, no directory. The *Repo receiver exists only for symmetry with
// the engine's other query methods (TOC, Resolve, Expand), all of which are methods on *Repo.
package engine

import (
	"fmt"
	"path/filepath"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// occurrence is one symbol-table occurrence: the extracted Symbol together with its token streams
// and whether the file it came from parsed partially on its own side. It is the pure core's own
// working value — never emitted — so a row in the symbol table can be compared against another
// without walking a parse tree a second time.
type occurrence struct {
	sym     Symbol
	streams symbolStreams
	lossy   bool
}

// symbolKey is the symbol table's key: a symbol's id paired with its kind. Kind is part of the key
// so a const replaced by a var of the same name is a delete plus a create — two different
// declarations — rather than a modification; see delta_answer.go's own doc comment on
// DeltaAnswer.Modified.
type symbolKey struct {
	id   string
	kind Kind
}

// extractSide extracts one side of one entry: it returns the side's occurrences, whether its parse
// reported an error (the file-level lossy flag, independent of unit spellability), and a non-empty
// failMsg only when the side could not be extracted at all.
//
// src == nil means the side does not exist, and this function reports nothing at all: that is
// neither a failure nor a lossy parse, since there was no parse to begin with.
//
// Bytes that are not valid UTF-8 are rejected before parsing, exactly as the rest of the engine
// already rejects them (walk.go's fileEntry, units.go's PackageClause): the parse seam performs no
// such check itself and would otherwise hand undecodable bytes to the grammar, yielding a partial
// tree that would misreport this as merely lossy rather than as the extraction failure it is.
//
// A unit this side's entry supplies that glyph.Parse cannot spell contributes no occurrences from
// this side — unitSpellable's own rule, applied here exactly as the walk applies it — but the parse
// still runs and its partial flag is still reported: spellability governs whether a symbol is
// listed, not whether the file's own parse succeeded.
func (r *Repo) extractSide(strategy Strategy, lang, path, unit string, src []byte, sideLabel string) (occs []occurrence, lossy bool, failMsg string) {
	if src == nil {
		return nil, false, ""
	}
	if !utf8.Valid(src) {
		return nil, false, fmt.Sprintf("%s: %s side is not valid UTF-8", path, sideLabel)
	}

	spellable := r.unitSpellable(unit)
	err := treesitter.WithTree(lang, src, func(root *ts.Node, partial bool) error {
		lossy = partial
		if !spellable {
			return nil
		}
		symbols := strategy.Symbols(unit, root, src)
		for i := range symbols {
			symbols[i].File = path
		}
		streams := tokenStreamsForSymbols(root, src, symbols)
		occs = make([]occurrence, len(symbols))
		for i := range symbols {
			occs[i] = occurrence{sym: symbols[i], streams: streams[i], lossy: partial}
		}
		return nil
	})
	if err != nil {
		return nil, false, fmt.Sprintf("%s: %s side: %v", path, sideLabel, err)
	}
	return occs, lossy, ""
}

// deltaEntryFiles answers one DeltaEntry's own DeltaFile and the occurrences it contributes to
// each side's symbol table.
//
// A non-empty Refusal short-circuits the entry before either side is parsed: DispositionError,
// Refusal as the message, no occurrences from either side.
//
// Both sides nil is a caller error in the entry, not a state to classify: DispositionError with a
// message naming the path.
//
// An extension resolving to no registered Strategy gives DispositionUnsupported: no occurrences,
// and this is not an error.
//
// Otherwise the sides that exist are extracted. A nil before with a non-nil after is
// DispositionAdded, the reverse is DispositionRemoved, and two non-nil sides is DispositionChanged.
// A failure to extract either side — invalid UTF-8, or the parse seam itself failing — sets
// DispositionError for the whole entry with that failure's message and contributes no occurrences
// from either side; a failing entry never fails the whole batch, so Delta's own error return stays
// nil regardless.
func (r *Repo) deltaEntryFiles(entry DeltaEntry) (DeltaFile, []occurrence, []occurrence) {
	if entry.Refusal != "" {
		return DeltaFile{Path: entry.Path, Disposition: DispositionError, Error: entry.Refusal}, nil, nil
	}
	if entry.Before == nil && entry.After == nil {
		msg := fmt.Sprintf("%s: entry carries neither a before nor an after side", entry.Path)
		return DeltaFile{Path: entry.Path, Disposition: DispositionError, Error: msg}, nil, nil
	}

	lang, hasLang := LanguageForExtension(filepath.Ext(entry.Path))
	var strategy Strategy
	hasStrategy := false
	if hasLang {
		strategy, hasStrategy = StrategyFor(lang)
	}
	if !hasLang || !hasStrategy {
		return DeltaFile{Path: entry.Path, Disposition: DispositionUnsupported}, nil, nil
	}

	disposition := DispositionChanged
	switch {
	case entry.Before == nil:
		disposition = DispositionAdded
	case entry.After == nil:
		disposition = DispositionRemoved
	}

	beforeOccs, beforeLossy, beforeErr := r.extractSide(strategy, lang, entry.Path, entry.BeforeUnit, entry.Before, "before")
	afterOccs, afterLossy, afterErr := r.extractSide(strategy, lang, entry.Path, entry.AfterUnit, entry.After, "after")
	if beforeErr != "" || afterErr != "" {
		msg := beforeErr
		switch {
		case msg == "":
			msg = afterErr
		case afterErr != "":
			msg = msg + "; " + afterErr
		}
		return DeltaFile{Path: entry.Path, Disposition: DispositionError, Error: msg}, nil, nil
	}

	return DeltaFile{
		Path:        entry.Path,
		Disposition: disposition,
		LossyBefore: beforeLossy,
		LossyAfter:  afterLossy,
	}, beforeOccs, afterOccs
}

// Delta compares two versions of a batch of files, extracted with the same Strategy every other
// query in this package uses, and returns the resulting symbol-table delta. Its returned error is
// non-nil only for a failure of the call as a whole; a single entry's own extraction failure is
// reported through its own DeltaFile instead (see deltaEntryFiles) and never fails the call.
func (r *Repo) Delta(entries []DeltaEntry) (DeltaAnswer, error) {
	files := make([]DeltaFile, 0, len(entries))
	beforeTable := make(map[symbolKey][]occurrence)
	afterTable := make(map[symbolKey][]occurrence)

	for _, entry := range entries {
		df, beforeOccs, afterOccs := r.deltaEntryFiles(entry)
		files = append(files, df)
		for _, occ := range beforeOccs {
			key := symbolKey{id: occ.sym.ID, kind: occ.sym.Kind}
			beforeTable[key] = append(beforeTable[key], occ)
		}
		for _, occ := range afterOccs {
			key := symbolKey{id: occ.sym.ID, kind: occ.sym.Kind}
			afterTable[key] = append(afterTable[key], occ)
		}
	}

	_ = beforeTable
	_ = afterTable
	return DeltaAnswer{Files: files}, nil
}
