// memo.go holds the per-call record memo every extraction path reads.
// A memo is a local of one exported call and dies with it, which keeps the engine inside its no-cache rule.
// Each file is read and parsed at most once per call.

package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// fileRecord is everything one read and parse of a file yields.
// Go symbols carry the placeholder unit "" and no File, and a language that spells units by declaration mints its own units;
// consumers stamp a copy through stampSymbols, never the record itself.
type fileRecord struct {
	// err is the read, UTF-8 or parse failure text; every other field except lines is then unset.
	err string
	// lines is the file's line count, set on every record whose read succeeded.
	lines int
	// header is the file's final header text.
	header string
	// parsed reports that a registered strategy parsed the file without a treesitter.WithTree error.
	parsed bool
	// lang is the file's language, set only when parsed.
	lang string
	// test and generated are the strategy's flags.
	// lossy is the parse's partial-tree flag, also set when the strategy dropped an id it could not spell.
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

// dirVote is a directory's clause vote.
// pkg is the Go vote, the dominant clause among the Go records, and clauses is the base-name-to-clause map of every Go record that has one;
// the unit derivations, the directory's package documentation and the Go unit collection read these.
// answerPkg and answerLang are the directory answer's package and language, taken from the language with the most files;
// answerLang is empty whenever answerPkg is.
type dirVote struct {
	pkg                   string
	clauses               map[string]string
	answerPkg, answerLang string
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
	// read is the byte source every record build reads through, keyed by repository-relative file path.
	read func(rel string) ([]byte, error)
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
		read: func(rel string) ([]byte, error) {
			dir, base := splitDirBase(rel)
			return os.ReadFile(filepath.Join(r.absDir(dir), base))
		},
	}
}

// lineCount returns the number of lines in src:
// each newline ends one, and a non-empty final line without a newline counts as one more.
// An empty src has zero lines.
func lineCount(src []byte) int {
	count := bytes.Count(src, []byte{'\n'})
	if len(src) > 0 && src[len(src)-1] != '\n' {
		count++
	}
	return count
}

// buildRecord reads and parses base inside dirRel once and returns everything extraction needs from it:
// header, flags, package clause, package doc and, when withSymbols, symbols (Go symbols under the placeholder unit "").
// The bytes come from read, called with the file's repository-relative path.
// A strategy that dropped an id it could not spell makes the record lossy, whether or not symbols were requested.
// A read failure, invalid UTF-8 or a failed parse yields a record holding only err
// (and lines, when the bytes were read);
// a file with no language or strategy yields a record holding only its header and lines.
func (r *Repo) buildRecord(dirRel, base string, withSymbols bool, read func(rel string) ([]byte, error)) *fileRecord {
	rec := &fileRecord{withSymbols: withSymbols}

	src, err := read(joinRel(dirRel, base))
	if err != nil {
		rec.err = err.Error()
		return rec
	}
	rec.lines = lineCount(src)
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
		parsedRec.clause = strategy.Package(root, src)
		parsedRec.packageDoc = strategy.PackageDoc(root, src)
		symbols, dropped := strategy.Symbols("", root, src)
		parsedRec.lossy = partial || len(dropped) > 0
		if withSymbols {
			parsedRec.symbols = symbols
		}
		return nil
	})
	if err != nil {
		rec.err = err.Error()
		return rec
	}

	parsedRec.lines = rec.lines
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
	rec := m.repo.buildRecord(dirRel, base, m.allSymbols || wantSymbols, m.read)
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
//
// The Go vote counts the clauses of Go records only.
// The answer's language is the one with the most files in recs, every file whose extension maps to a language counting, parsed or not,
// and the smaller language name breaks a tie.
// The answer's package is the Go vote when that language is Go, and otherwise the clause most of that language's files declare, the smallest breaking a tie.
func (m *fileMemo) dirVote(dirRel string, recs map[string]*fileRecord, memoise bool) dirVote {
	if memoise {
		if v, ok := m.votes[dirRel]; ok {
			return v
		}
	}

	clauses := make(map[string]string, len(recs))
	otherClauseCounts := make(map[string]map[string]int)
	fileCounts := make(map[string]int)
	for base, rec := range recs {
		if lang, ok := LanguageForExtension(filepath.Ext(base)); ok {
			fileCounts[lang]++
		}
		if !rec.hasClause {
			continue
		}
		if rec.lang == "go" {
			clauses[base] = rec.clause
			continue
		}
		if otherClauseCounts[rec.lang] == nil {
			otherClauseCounts[rec.lang] = make(map[string]int)
		}
		otherClauseCounts[rec.lang][rec.clause]++
	}
	pkg, _ := UnitsForClauseMap(dirRel, clauses)

	majority := ""
	for lang, count := range fileCounts {
		if majority == "" || count > fileCounts[majority] || (count == fileCounts[majority] && lang < majority) {
			majority = lang
		}
	}
	answerPkg := ""
	switch {
	case majority == "go":
		answerPkg = pkg
	case majority != "" && len(otherClauseCounts[majority]) > 0:
		answerPkg = mostCommonClause(otherClauseCounts[majority])
	}
	answerLang := ""
	if answerPkg != "" {
		answerLang = majority
	}

	v := dirVote{pkg: pkg, clauses: clauses, answerPkg: answerPkg, answerLang: answerLang}
	if memoise {
		m.votes[dirRel] = v
	}
	return v
}

// stampSymbols returns a new slice holding a copy of each symbol in syms with its File set to file.
// A Go symbol also has its unit set to unit and its ID recomputed from the glyph:
// the Go strategy uses its unit argument only to build Glyph and ID,
// so stamping after the directory's vote yields symbols identical to extracting with the unit.
// A symbol of any other language keeps the unit and ID its strategy minted.
// The returned slice is never nil, so an empty list still encodes as [].
func stampSymbols(syms []Symbol, unit, file string) []Symbol {
	stamped := make([]Symbol, 0, len(syms))
	for _, sym := range syms {
		if sym.Glyph.Lang == glyph.Go {
			sym.Glyph.Unit = unit
			sym.ID = sym.Glyph.String()
		}
		sym.File = file
		stamped = append(stamped, sym)
	}
	return stamped
}
