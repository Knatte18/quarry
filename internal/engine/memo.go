// memo.go holds the per-call record memo every extraction path reads.
// A memo is a local of one exported call and dies with it, which keeps the engine inside its no-cache rule.
// Each file is read and parsed at most once per call.

package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// fileRecord is everything one read and parse of a file yields.
// Symbols carry the placeholder unit "" and no File;
// consumers stamp a copy through stampSymbols, never the record itself.
type fileRecord struct {
	// err is the read, UTF-8 or parse failure text; every other field is then unset.
	err string
	// header is the file's final header text.
	header string
	// parsed reports that a registered strategy parsed the file without a treesitter.WithTree error.
	parsed bool
	// lang is the file's language, set only when parsed.
	lang string
	// test, generated and lossy are the strategy's flags and the parse's partial-tree flag.
	test, generated, lossy bool
	// clause is the file's package clause; hasClause reports it is non-empty.
	clause    string
	hasClause bool
	// packageDoc is the file's package documentation.
	packageDoc string
	// symbols are the file's symbols, extracted only when withSymbols.
	symbols []Symbol
	// withSymbols records whether symbols were requested when the record was built, whatever the
	// outcome: a file that could not yield symbols is never rebuilt for them.
	withSymbols bool
}

// dirVote is a directory's clause vote: its dominant clause, the language of that clause's files,
// and the base-name-to-clause map of every file whose record has a clause.
type dirVote struct {
	pkg, lang string
	clauses   map[string]string
}

// fileMemo memoises one fileRecord per file and one dirVote per directory for the length of one
// exported call.
type fileMemo struct {
	repo       *Repo
	allSymbols bool
	// records is keyed by repository-relative file path.
	records map[string]*fileRecord
	// votes is keyed by repository-relative directory path.
	votes map[string]dirVote
	// builds is the test seam for the parse-once guarantee: it counts record builds per file,
	// incremented before each build (builds started, not builds that succeeded), read by tests and
	// never by production code.
	// Each build performs at most one tree-sitter parse, so a count of at most 1 per file means at
	// most one parse per file.
	builds map[string]int
}

// newFileMemo builds an empty fileMemo for r.
// With allSymbols true every record build extracts symbols;
// otherwise a build extracts them only when the requesting consumer wants that file's symbols.
func newFileMemo(r *Repo, allSymbols bool) *fileMemo {
	return &fileMemo{
		repo:       r,
		allSymbols: allSymbols,
		records:    make(map[string]*fileRecord),
		votes:      make(map[string]dirVote),
		builds:     make(map[string]int),
	}
}

// buildRecord reads and parses base inside dirRel once and returns everything extraction needs from it:
// header, flags, package clause, package doc and, when withSymbols, symbols under the placeholder unit "".
// A read failure, invalid UTF-8 or a failed parse yields a record holding only err;
// a file with no language or strategy yields a record holding only its header.
func (r *Repo) buildRecord(dirRel, base string, withSymbols bool) *fileRecord {
	rec := &fileRecord{withSymbols: withSymbols}

	src, err := os.ReadFile(filepath.Join(r.absDir(dirRel), base))
	if err != nil {
		rec.err = err.Error()
		return rec
	}
	if !utf8.Valid(src) {
		rec.err = fmt.Sprintf("engine: %s: not valid UTF-8", joinRel(dirRel, base))
		return rec
	}

	lang, hasLang := LanguageForExtension(filepath.Ext(base))
	strategy, hasStrategy := StrategyFor(lang)
	if !hasLang || !hasStrategy {
		rec.header = HeaderForFile(base, src)
		return rec
	}

	var parsedRec fileRecord
	err = treesitter.WithTree(lang, src, func(root *ts.Node, partial bool) error {
		parsedRec.header = FirstParagraph(strategy.Header(root, src))
		parsedRec.test = strategy.TestFile(base)
		parsedRec.generated = strategy.Generated(root, src)
		parsedRec.lossy = partial
		parsedRec.clause = strategy.Package(root, src)
		parsedRec.packageDoc = strategy.PackageDoc(root, src)
		if withSymbols {
			parsedRec.symbols = strategy.Symbols("", root, src)
		}
		return nil
	})
	if err != nil {
		rec.err = err.Error()
		return rec
	}

	parsedRec.withSymbols = withSymbols
	parsedRec.parsed = true
	parsedRec.lang = lang
	parsedRec.hasClause = parsedRec.clause != ""
	return &parsedRec
}

// record returns the memoised record for base inside dirRel.
// A record built without symbols is rebuilt, and counted again, when symbols are later wanted:
// an allSymbols memo builds every record with symbols and never rebuilds,
// and a TOC memo requests each file once per call,
// so the rebuild is a counted regression guard rather than an expected path.
func (m *fileMemo) record(dirRel, base string, wantSymbols bool) *fileRecord {
	path := joinRel(dirRel, base)
	if rec, ok := m.records[path]; ok && (!wantSymbols || rec.withSymbols) {
		return rec
	}
	m.builds[path]++
	rec := m.repo.buildRecord(dirRel, base, m.allSymbols || wantSymbols)
	m.records[path] = rec
	return rec
}

// dirRecords returns the record of every base in dirRel keyed by base name,
// requesting symbols for the bases wantSymbols accepts.
func (m *fileMemo) dirRecords(dirRel string, bases []string, wantSymbols func(base string) bool) map[string]*fileRecord {
	recs := make(map[string]*fileRecord, len(bases))
	for _, base := range bases {
		recs[base] = m.record(dirRel, base, wantSymbols(base))
	}
	return recs
}

// dirVote computes dirRel's clause vote from recs, the records of the files that vote.
// The vote reads records and never parses.
// memoise is false only for a vote set extended by an explicitly named, gitignored target,
// which must not leak into the directory's plain vote.
func (m *fileMemo) dirVote(dirRel string, recs map[string]*fileRecord, memoise bool) dirVote {
	if memoise {
		if v, ok := m.votes[dirRel]; ok {
			return v
		}
	}

	clauses := make(map[string]string, len(recs))
	for base, rec := range recs {
		if rec.hasClause {
			clauses[base] = rec.clause
		}
	}
	pkg, _ := UnitsForClauseMap(dirRel, clauses)
	lang := ""
	for base, clause := range clauses {
		if clause == pkg {
			if l, ok := LanguageForExtension(filepath.Ext(base)); ok {
				lang = l
				break
			}
		}
	}

	v := dirVote{pkg: pkg, lang: lang, clauses: clauses}
	if memoise {
		m.votes[dirRel] = v
	}
	return v
}

// stampSymbols returns a new slice holding a copy of each symbol in syms with its unit set to unit,
// its ID recomputed from the glyph and its File set to file.
// The extraction strategy uses its unit argument only to build Glyph and ID,
// so stamping after the directory's vote yields symbols identical to extracting with the unit.
// The returned slice is never nil, so an empty list still encodes as [].
func stampSymbols(syms []Symbol, unit, file string) []Symbol {
	stamped := make([]Symbol, 0, len(syms))
	for _, sym := range syms {
		sym.Glyph.Unit = unit
		sym.ID = sym.Glyph.String()
		sym.File = file
		stamped = append(stamped, sym)
	}
	return stamped
}
