// units.go declares the exported clause-and-unit seam package quarry needs to derive a delta
// batch's units without reaching into this package's unexported dirPackage and unitFor. Every rule
// declared here is the same rule walk.go's dirPackage and unitFor already apply, extracted rather
// than copied, so both sides of a comparison — a file read from disk and one read from a git
// revision — call one implementation and never a second. A glyph unit is a directory-level fact no
// single file's own content can establish: it depends on every file in the directory's clause, not
// just one file's, which is why the per-file primitive below returns a clause and the
// directory-level primitive built on it is the one that turns a whole directory's clauses into a
// unit.

package engine

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// UnitsForClauseMap runs the clause vote dirPackage's own per-directory rule applies over clauses —
// the base-name-to-package-clause map for one directory dirRel — and returns the directory's
// dominant clause dirPkg together with unitOf, a function mapping a file's base name to the glyph
// unit unitFor would derive for it from dirRel, dirPkg and that file's own clause.
//
// The vote: dirPkg is the most common clause among files whose clause does not end in "_test",
// falling back to the most common clause overall when every clause ends in "_test", with the
// lexicographically smallest clause breaking a tie. mostCommonClause's tie-break is preserved
// exactly, which is what keeps a directory's answer independent of directory-read order.
//
// A base name absent from clauses maps to the unit a file with an empty clause would get: unitOf
// reads clauses[base], and a missing key returns "" from a Go map exactly as an explicit empty
// clause would, so the two inputs are indistinguishable to unitOf by construction.
//
// The discriminator unitOf applies is the clause, never the filename — a file belongs to the
// external-test unit only when its clause is exactly dirPkg plus the "_test" suffix — and the
// repository root stays the documented exception returning the empty string for both branches; see
// unitFor's own doc comment for both rules in full.
func UnitsForClauseMap(dirRel string, clauses map[string]string) (dirPkg string, unitOf func(base string) string) {
	counts := make(map[string]int, len(clauses))
	for _, clause := range clauses {
		counts[clause]++
	}

	nonTest := make(map[string]int, len(counts))
	for clause, n := range counts {
		if !strings.HasSuffix(clause, "_test") {
			nonTest[clause] = n
		}
	}
	switch {
	case len(nonTest) > 0:
		dirPkg = mostCommonClause(nonTest)
	case len(counts) > 0:
		dirPkg = mostCommonClause(counts)
	}

	unitOf = func(base string) string {
		return unitFor(dirRel, dirPkg, clauses[base])
	}
	return dirPkg, unitOf
}

// PackageClause returns the package clause a file named base, holding src, declares — the same
// clause dirPackage's own per-file vote would compute for it, and the same clause a caller reading
// this file's bytes from a git revision rather than from disk must compute identically.
//
// ok is false under exactly the four conditions under which dirPackage records no clause for a
// file today: base's extension names no language or no registered strategy (LanguageForExtension
// and StrategyFor), src is not valid UTF-8, a treesitter.WithTree parse of src returns an error, or
// the strategy's own Package result is the empty string.
//
// The UTF-8 check lives here, inside PackageClause, rather than in each caller: this is the one
// place both sides of a directory's clause vote pass through, so a caller-side check would hold
// only for the caller that reads from disk and leave bytes fetched from a revision unchecked,
// letting one file vote on one side of a comparison only.
func PackageClause(base string, src []byte) (clause string, ok bool) {
	lang, hasLang := LanguageForExtension(filepath.Ext(base))
	if !hasLang {
		return "", false
	}
	strategy, hasStrategy := StrategyFor(lang)
	if !hasStrategy {
		return "", false
	}
	if !utf8.Valid(src) {
		return "", false
	}
	if err := treesitter.WithTree(lang, src, func(root *ts.Node, _ bool) error {
		clause = strategy.Package(root, src)
		return nil
	}); err != nil {
		return "", false
	}
	if clause == "" {
		return "", false
	}
	return clause, true
}
