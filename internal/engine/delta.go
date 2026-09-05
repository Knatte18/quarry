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
	"sort"
	"strings"
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

// comparisonTuple is the four-dimension comparison tuple every symbol-table occurrence reduces to:
// a hash of its body token stream, its signature text, its doc text, and its file. These are the
// same four dimensions ChangedDimension's closed vocabulary draws from, so the comparison here and
// the changed array a caller reads can never cover different ground.
type comparisonTuple struct {
	bodyHash  string
	signature string
	doc       string
	file      string
}

// hashBodyStream returns a string built deterministically from body's tokens, kind and text alike,
// so two occurrences with the same body compare equal and two with a different body compare
// unequal — the same guarantee a real hash would give, without a collision ever being possible: the
// two separator bytes used here can never appear inside a tree-sitter node's own kind or text.
func hashBodyStream(body tokenStream) string {
	var sb strings.Builder
	for _, tok := range body {
		sb.WriteString(tok.kind)
		sb.WriteByte(0)
		sb.WriteString(tok.text)
		sb.WriteByte(1)
	}
	return sb.String()
}

// tupleFor reduces one occurrence to its comparison tuple.
func tupleFor(occ occurrence) comparisonTuple {
	return comparisonTuple{
		bodyHash:  hashBodyStream(occ.streams.body),
		signature: occ.sym.Signature,
		doc:       occ.sym.Doc,
		file:      occ.sym.File,
	}
}

// tupleMultisetEqual reports whether a and b, read as multisets of comparisonTuple, are equal.
func tupleMultisetEqual(a, b []comparisonTuple) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[comparisonTuple]int, len(a))
	for _, t := range a {
		counts[t]++
	}
	for _, t := range b {
		counts[t]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}

// stringMultisetEqual reports whether a and b, read as multisets of string, are equal. It backs
// changedDimensions' per-dimension comparison.
func stringMultisetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}

// changedDimensions returns the union of dimensions that differ between beforeOccs and afterOccs,
// in ChangedDimension's own declared order: body, signature, doc, file. Each dimension is compared
// as its own multiset across every occurrence under the key, never occurrence-by-occurrence,
// because a multi-occurrence key — several func init() in one package, for instance — has no
// per-occurrence identity to line the two sides up by; a doc-only change to one of two such
// occurrences is reported as changed:["doc"] under this rule, never as a vanished difference.
func changedDimensions(beforeOccs, afterOccs []occurrence) []ChangedDimension {
	project := func(occs []occurrence, f func(occurrence) string) []string {
		out := make([]string, len(occs))
		for i, occ := range occs {
			out[i] = f(occ)
		}
		return out
	}
	bodyOf := func(occ occurrence) string { return hashBodyStream(occ.streams.body) }
	sigOf := func(occ occurrence) string { return occ.sym.Signature }
	docOf := func(occ occurrence) string { return occ.sym.Doc }
	fileOf := func(occ occurrence) string { return occ.sym.File }

	var changed []ChangedDimension
	if !stringMultisetEqual(project(beforeOccs, bodyOf), project(afterOccs, bodyOf)) {
		changed = append(changed, ChangedBody)
	}
	if !stringMultisetEqual(project(beforeOccs, sigOf), project(afterOccs, sigOf)) {
		changed = append(changed, ChangedSignature)
	}
	if !stringMultisetEqual(project(beforeOccs, docOf), project(afterOccs, docOf)) {
		changed = append(changed, ChangedDoc)
	}
	if !stringMultisetEqual(project(beforeOccs, fileOf), project(afterOccs, fileOf)) {
		changed = append(changed, ChangedFile)
	}
	return changed
}

// locationsFor returns one SymbolLocation per occurrence, unsorted — the ordering pass (card 21)
// sorts this array independently of the After array it sits beside in a ModifiedSymbol.
func locationsFor(occs []occurrence) []SymbolLocation {
	locs := make([]SymbolLocation, len(occs))
	for i, occ := range occs {
		locs[i] = SymbolLocation{File: occ.sym.File, Start: occ.sym.Start, SigEnd: occ.sym.SigEnd, End: occ.sym.End}
	}
	return locs
}

// symbolsFor returns one Symbol per occurrence, unsorted, in occs' own order.
func symbolsFor(occs []occurrence) []Symbol {
	syms := make([]Symbol, len(occs))
	for i, occ := range occs {
		syms[i] = occ.sym
	}
	return syms
}

// compareTables reduces beforeTable and afterTable to created, deleted and modified: a key present
// only on the after side contributes every one of its symbols to created; a key present only on the
// before side contributes every one of its symbols to deleted; a key present on both sides is
// compared as a multiset of comparison tuples — never of body hashes alone — and any difference
// produces exactly one modified entry for that key. Equal multisets means unchanged, and such a
// symbol appears in none of the three returned slices.
func compareTables(beforeTable, afterTable map[symbolKey][]occurrence) (created, deleted []Symbol, modified []ModifiedSymbol) {
	keys := make(map[symbolKey]bool, len(beforeTable)+len(afterTable))
	for k := range beforeTable {
		keys[k] = true
	}
	for k := range afterTable {
		keys[k] = true
	}

	for key := range keys {
		beforeOccs := beforeTable[key]
		afterOccs := afterTable[key]
		switch {
		case len(beforeOccs) == 0:
			created = append(created, symbolsFor(afterOccs)...)
		case len(afterOccs) == 0:
			deleted = append(deleted, symbolsFor(beforeOccs)...)
		default:
			beforeTuples := make([]comparisonTuple, len(beforeOccs))
			for i, occ := range beforeOccs {
				beforeTuples[i] = tupleFor(occ)
			}
			afterTuples := make([]comparisonTuple, len(afterOccs))
			for i, occ := range afterOccs {
				afterTuples[i] = tupleFor(occ)
			}
			if tupleMultisetEqual(beforeTuples, afterTuples) {
				continue
			}
			modified = append(modified, ModifiedSymbol{
				ID:      key.id,
				Kind:    key.kind,
				Changed: changedDimensions(beforeOccs, afterOccs),
				Before:  locationsFor(beforeOccs),
				After:   symbolsFor(afterOccs),
			})
		}
	}
	return created, deleted, modified
}

// sideSymbol is one single-occurrence deleted or created symbol considered as a rename endpoint. A
// key held by several declarations on its own side never becomes a sideSymbol at all — see
// singleOccurrenceOnly — because such a key is never a rename candidate on either tier.
type sideSymbol struct {
	occ occurrence
}

// symbolKeyOf returns s's symbol-table key.
func symbolKeyOf(s sideSymbol) symbolKey {
	return symbolKey{id: s.occ.sym.ID, kind: s.occ.sym.Kind}
}

// singleOccurrenceOnly returns every key in table that is absent from other and holds exactly one
// occurrence, as sideSymbol values: the deleted-only or created-only, single-declaration subset
// either rename tier ever considers.
func singleOccurrenceOnly(table, other map[symbolKey][]occurrence) []sideSymbol {
	var out []sideSymbol
	for key, occs := range table {
		if _, exists := other[key]; exists {
			continue
		}
		if len(occs) != 1 {
			continue
		}
		out = append(out, sideSymbol{occ: occs[0]})
	}
	return out
}

// renameStructural13 reports whether d and c satisfy the first three conditions shared by both
// rename tiers: the same glyph unit (the before side's unit for d, the after side's for c), the
// same owner chain and kind, and a different name.
func renameStructural13(d, c sideSymbol) bool {
	dg, cg := d.occ.sym.Glyph, c.occ.sym.Glyph
	return dg.Unit == cg.Unit &&
		sameOwner(dg.Owner, cg.Owner) &&
		d.occ.sym.Kind == c.occ.sym.Kind &&
		dg.Name != cg.Name
}

// renameExactBodyAndSignature reports whether d and c additionally satisfy exact-tier conditions 4
// and 5: their body token streams are identical modulo the renamed identifier, and so are their
// signature token streams, under the same node-based substitution rule — never a textual
// substitution over either's verbatim text.
func renameExactBodyAndSignature(d, c sideSymbol) bool {
	dName, cName := d.occ.sym.Glyph.Name, c.occ.sym.Glyph.Name
	return identicalModuloName(d.occ.streams.body, c.occ.streams.body, dName, cName) &&
		identicalModuloName(d.occ.streams.signature, c.occ.streams.signature, dName, cName)
}

// renameExactBodyCondition reports whether d and c satisfy exact-tier condition 7: both have a
// non-empty body stream, and neither side's file had a partial parse. This is what keeps every
// const, var, type alias and interface method element — all of which have an empty body stream on
// both sides — out of the exact tier: without it condition 4 would be vacuously satisfied by any
// two of them, and the lossy half guards against a truncated table manufacturing a spurious delete.
func renameExactBodyCondition(d, c sideSymbol) bool {
	return len(d.occ.streams.body) > 0 && len(c.occ.streams.body) > 0 && !d.occ.lossy && !c.occ.lossy
}

// renamePair is one candidate pairing found while scanning the single-occurrence deleted and
// created lists for a structural-plus-token match.
type renamePair struct {
	d sideSymbol
	c sideSymbol
}

// classifyExactTier runs the exact tier over singleDeleted and singleCreated — the subset of
// deleted and created symbols whose key holds exactly one declaration on its own side.
//
// A pair is asserted only when it is the unique match on both sides among every pair satisfying
// conditions 1-5: several matches for the same deleted or created symbol means nothing was chosen,
// so all of them fall through to the evidence tier instead (classifyEvidenceTier), whether or not
// they were ever this function's own candidate. Condition 7 is checked last, after uniqueness, so a
// pair demoted for an empty body stream or a lossy side is demoted rather than silently dropped.
//
// It returns every asserted RenamedPair, and the symbol-table keys of the deleted and created
// symbols an asserted pair removes from their respective arrays.
func classifyExactTier(singleDeleted, singleCreated []sideSymbol) (renamed []RenamedPair, assertedDeleted, assertedCreated map[symbolKey]bool) {
	assertedDeleted = make(map[symbolKey]bool)
	assertedCreated = make(map[symbolKey]bool)

	var examPairs []renamePair
	for _, d := range singleDeleted {
		for _, c := range singleCreated {
			if renameStructural13(d, c) && renameExactBodyAndSignature(d, c) {
				examPairs = append(examPairs, renamePair{d: d, c: c})
			}
		}
	}

	dDegree := make(map[symbolKey]int, len(examPairs))
	cDegree := make(map[symbolKey]int, len(examPairs))
	for _, p := range examPairs {
		dDegree[symbolKeyOf(p.d)]++
		cDegree[symbolKeyOf(p.c)]++
	}

	for _, p := range examPairs {
		dKey, cKey := symbolKeyOf(p.d), symbolKeyOf(p.c)
		if dDegree[dKey] != 1 || cDegree[cKey] != 1 {
			continue
		}
		if !renameExactBodyCondition(p.d, p.c) {
			continue
		}
		renamed = append(renamed, RenamedPair{From: p.d.occ.sym, To: p.c.occ.sym})
		assertedDeleted[dKey] = true
		assertedCreated[cKey] = true
	}

	return renamed, assertedDeleted, assertedCreated
}

// renameCandidateFor builds one evidence-tier RenameCandidate for the created side of a (d, c)
// structural match. Every signal is filled mechanically, over the token streams under the same
// node-based rule the exact tier uses — never over the verbatim Signature or Doc text — and no
// composite score is computed anywhere.
func renameCandidateFor(d, c sideSymbol) RenameCandidate {
	dName, cName := d.occ.sym.Glyph.Name, c.occ.sym.Glyph.Name
	return RenameCandidate{
		ID:   c.occ.sym.ID,
		Kind: c.occ.sym.Kind,
		File: c.occ.sym.File,
		Signals: RenameSignals{
			SignatureIdenticalModuloName: identicalModuloName(d.occ.streams.signature, c.occ.streams.signature, dName, cName),
			BodyTokenSimilarity:          bodyTokenSimilarity(d.occ.streams.body, c.occ.streams.body, dName, cName),
			BodyTokensBefore:             len(d.occ.streams.body),
			BodyTokensAfter:              len(c.occ.streams.body),
			DocIdentical:                 d.occ.sym.Doc == c.occ.sym.Doc,
		},
	}
}

// classifyEvidenceTier reports every (deleted, created) pair satisfying the first three exact-tier
// conditions — same unit, same owner chain, same kind, different name — that is not part of an
// asserted exact pair (assertedDeleted/assertedCreated, as classifyExactTier returns them).
//
// There is no similarity threshold anywhere: structural facts alone gate the candidate set. Neither
// the deleted symbol nor any candidate created symbol is removed from its array — the candidate
// block only cross-references them and attaches signals, since suppressing either for a candidate
// quarry has not resolved would be a silent pick in disguise.
//
// Candidates within one entry are sorted by BodyTokenSimilarity descending, then by created ID
// ascending — a deterministic ordering, not a ranking. The returned entries are otherwise unordered
// among themselves; the outer ordering pass (card 21) sorts them.
func classifyEvidenceTier(singleDeleted, singleCreated []sideSymbol, assertedDeleted, assertedCreated map[symbolKey]bool) []RenameCandidateEntry {
	var order []symbolKey
	byDeleted := make(map[symbolKey][]RenameCandidate)
	deletedByKey := make(map[symbolKey]sideSymbol)

	for _, d := range singleDeleted {
		dKey := symbolKeyOf(d)
		if assertedDeleted[dKey] {
			continue
		}
		for _, c := range singleCreated {
			cKey := symbolKeyOf(c)
			if assertedCreated[cKey] {
				continue
			}
			if !renameStructural13(d, c) {
				continue
			}
			if _, seen := deletedByKey[dKey]; !seen {
				order = append(order, dKey)
				deletedByKey[dKey] = d
			}
			byDeleted[dKey] = append(byDeleted[dKey], renameCandidateFor(d, c))
		}
	}

	entries := make([]RenameCandidateEntry, 0, len(order))
	for _, dKey := range order {
		cands := byDeleted[dKey]
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].Signals.BodyTokenSimilarity != cands[j].Signals.BodyTokenSimilarity {
				return cands[i].Signals.BodyTokenSimilarity > cands[j].Signals.BodyTokenSimilarity
			}
			return cands[i].ID < cands[j].ID
		})
		d := deletedByKey[dKey]
		entries = append(entries, RenameCandidateEntry{
			ID:         d.occ.sym.ID,
			Kind:       d.occ.sym.Kind,
			Candidates: cands,
		})
	}
	return entries
}

// removeKeys returns syms with every symbol whose (ID, Kind) key is in remove excluded, preserving
// the order of the symbols that remain.
func removeKeys(syms []Symbol, remove map[symbolKey]bool) []Symbol {
	out := make([]Symbol, 0, len(syms))
	for _, s := range syms {
		if remove[symbolKey{id: s.ID, kind: s.Kind}] {
			continue
		}
		out = append(out, s)
	}
	return out
}

// sortSymbols sorts syms in place by file ascending, then Start ascending, then id ascending, then
// kind ascending — symbolsOfUnit's existing file-then-Start rule with a total tie-break appended.
// The tie-break is load-bearing, not decoration: the const and var builders give every name
// declared in one spec the same Start and End, so file-then-Start alone cannot separate them, and
// without it a stable sort would preserve whatever order the table's map iteration happened to
// produce — randomised per run, since Go deliberately randomises map iteration order. The pair of
// id and kind is unique because it is the symbol table's own key. This is ordering, never ranking.
func sortSymbols(syms []Symbol) {
	sort.Slice(syms, func(i, j int) bool {
		a, b := syms[i], syms[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Kind < b.Kind
	})
}

// sortLocations sorts locs in place by file ascending then Start ascending. A ModifiedSymbol's
// Before array is independently sorted this way and is aligned with its After array by nothing: a
// key held by several declarations has no per-occurrence identity to align by.
func sortLocations(locs []SymbolLocation) {
	sort.Slice(locs, func(i, j int) bool {
		if locs[i].File != locs[j].File {
			return locs[i].File < locs[j].File
		}
		return locs[i].Start < locs[j].Start
	})
}

// sortDeltaAnswer puts every array of a DeltaAnswer into its stated total order, in place. created
// and deleted, and each ModifiedSymbol's After array, sort by sortSymbols' rule; each
// ModifiedSymbol's Before array sorts independently by sortLocations' rule. modified itself sorts
// by id ascending then kind ascending — the table key itself, so the order never depends on which
// occurrence was seen first. renamed sorts by the From symbol's id ascending then the To symbol's
// id ascending. candidates sorts by the deleted symbol's id ascending then its kind ascending, with
// the candidates inside each entry left in the order classifyEvidenceTier already gave them. files
// is never sorted: it keeps the input batch's own order, unchanged, so a caller can index it
// against what it submitted. Every rule here is ordering, never ranking.
func sortDeltaAnswer(created, deleted []Symbol, modified []ModifiedSymbol, renamed []RenamedPair, candidates []RenameCandidateEntry) {
	sortSymbols(created)
	sortSymbols(deleted)

	sort.Slice(modified, func(i, j int) bool {
		a, b := modified[i], modified[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Kind < b.Kind
	})
	for i := range modified {
		sortSymbols(modified[i].After)
		sortLocations(modified[i].Before)
	}

	sort.Slice(renamed, func(i, j int) bool {
		a, b := renamed[i], renamed[j]
		if a.From.ID != b.From.ID {
			return a.From.ID < b.From.ID
		}
		return a.To.ID < b.To.ID
	})

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Kind < b.Kind
	})
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

	created, deleted, modified := compareTables(beforeTable, afterTable)

	singleDeleted := singleOccurrenceOnly(beforeTable, afterTable)
	singleCreated := singleOccurrenceOnly(afterTable, beforeTable)
	renamed, assertedDeleted, assertedCreated := classifyExactTier(singleDeleted, singleCreated)
	candidates := classifyEvidenceTier(singleDeleted, singleCreated, assertedDeleted, assertedCreated)

	created = removeKeys(created, assertedCreated)
	deleted = removeKeys(deleted, assertedDeleted)

	sortDeltaAnswer(created, deleted, modified, renamed, candidates)

	return DeltaAnswer{
		Files:            files,
		Created:          created,
		Deleted:          deleted,
		Modified:         modified,
		Renamed:          renamed,
		RenameCandidates: candidates,
	}, nil
}
