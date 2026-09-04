// resolve.go implements the internal span lookup: the inverse of the walk, from a glyph back to
// the declarations it names. Repo.unitDirs maps a glyph unit back to the directory or directories
// that hold it, Repo.symbolsOfUnit parses those directories' files once and returns every symbol
// they declare, and Repo.SpansOf is the public, per-glyph wrapper the rest of this task's later
// verbs (resolve, expand) are built on. This file deliberately stops short of a status vocabulary:
// zero matches is an empty slice, not a "not_found" value, and the one fact the lookup cannot
// express without inventing one — a unit that names two different directories — is carried on
// unitDirs' own collision return instead, for the later resolve verb to promote into a status when
// it defines one.

package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// unitDirs maps a glyph unit back to the directory or directories that hold it, resolved
// literal-first:
//
//  1. If the directory named exactly by unit exists under the root, it is a hit, and the search in
//     it is restricted to files belonging to unit itself — excluding any file whose clause is that
//     directory package's own "_test" sibling, which belongs to unit+"_test" instead.
//  2. If unit ends in "_test" and the directory named by unit with that suffix trimmed exists, it is
//     a second hit, and the search in it is restricted to files whose clause is exactly that
//     directory package's "_test" sibling.
//  3. Neither existing means no directory at all: dirs is nil.
//
// When both exist, both are returned and collision is true. SpansOf ignores collision and returns
// the union of both directories' symbols; the later resolve verb promotes collision into an
// "ambiguous" status when it builds the status vocabulary. The flag lives on this unexported helper
// rather than on a public return so this task records the fact without inventing a status type that
// is not its to design.
//
// The literal-first order is the point of this function, not an implementation detail: a directory
// literally named "foo_test/" is legal Go, and the walk gives its own declarations the unit
// ".../foo_test" via unitFor, exactly as it would for the external-test sibling of a directory
// literally named "foo/". An unconditional strip of the "_test" suffix would therefore send the
// lookup into a "foo/" directory that need not exist, and one glyph string would silently name two
// different units depending on which branch ran first. docs/glyph.md §2 gives the external test
// unit its pseudo-path without saying what happens when a real directory spells the same string —
// that is a gap in the identifier contract, not one this engine closes; quarry's chosen behaviour is
// to check both and report the collision rather than pick one silently. Neither repository this
// engine's tests run against has such a directory, which is exactly why this rule must be right by
// construction rather than by test.
func (r *Repo) unitDirs(unit string) (dirs []string, collision bool) {
	if r.dirExists(unit) {
		dirs = append(dirs, unit)
	}
	if base, ok := strings.CutSuffix(unit, "_test"); ok && r.dirExists(base) {
		dirs = append(dirs, base)
	}
	return dirs, len(dirs) == 2
}

// dirExists reports whether dirRel names an existing directory under the repository root. It uses
// os.Lstat, never os.Stat, so a path whose final component is a symlink to a directory is not
// treated as a hit: the walk never descends a symlink (see walkDir's own doc comment), so no file's
// glyph unit can ever equal a path that resolves through one, and unitDirs must agree with the walk
// it inverts.
func (r *Repo) dirExists(dirRel string) bool {
	info, err := os.Lstat(r.absDir(dirRel))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// dirChainBelowRoot splits the repository-relative directory dirRel into the chain of directories
// from the first one below the root down to dirRel itself, inclusive: "a/b/c" becomes
// ["a", "a/b", "a/b/c"]. It never includes "." — the root is extended once by SpansOf's caller and
// is never extended a second time.
func dirChainBelowRoot(dirRel string) []string {
	if dirRel == "." {
		return nil
	}
	segments := strings.Split(dirRel, "/")
	chain := make([]string, 0, len(segments))
	cur := ""
	for _, seg := range segments {
		if cur == "" {
			cur = seg
		} else {
			cur = cur + "/" + seg
		}
		chain = append(chain, cur)
	}
	return chain
}

// symbolsOfUnit is the unit-level extraction primitive: it parses each of unit's .go files exactly
// once and returns every symbol in all of them, each with File set to the declaration's
// repository-relative, forward-slash path, ordered by file then start line. It is the primitive and
// SpansOf the thin wrapper, rather than the other way round, because a per-glyph lookup re-parses
// the whole unit directory and nothing is cached: grouping by unit is what keeps a whole-repository
// check in seconds rather than minutes, and the later resolve verb needs the same grouping anyway
// for the many glyphs one card can name.
//
// symbolsOfUnit resolves its directories through unitDirs and, when both exist, returns the union.
// In every directory the .go files are filtered through ig — the same ignore set the walk uses —
// before being parsed: without that filter a gitignored .go file beside listed ones would
// contribute spans the walk never listed.
//
// ig must already carry the repository root's own patterns and nothing below them — the caller's
// newIgnoreSet(root) followed by one extend("."). For each directory unitDirs returns, symbolsOfUnit
// extends ig down dirChainBelowRoot — from the first directory below the root to that directory
// inclusive, one extend per intervening directory — and trims them all back in reverse before
// moving to the next directory. The target directory's own .gitignore is included in that chain,
// unlike the walk, where walkDir owns that last step on descent; nothing descends here, so the last
// step is symbolsOfUnit's own. Doing this per directory rather than once is what makes the
// two-directory collision case correct: the literal directory and the suffix-stripped one sit under
// different chains, and one set built for either would filter the other wrongly.
//
// A file that cannot be read, is not valid UTF-8, or belongs to a directory whose unit is
// unspellable contributes nothing — exactly as it contributes no symbols to a walk answer, which is
// what keeps the two readings equal. A parse the grammar reports an error on still contributes its
// surviving symbols, for the same reason.
func (r *Repo) symbolsOfUnit(unit string, ig *ignoreSet) ([]Symbol, error) {
	dirs, _ := r.unitDirs(unit)

	symbols := make([]Symbol, 0)
	for _, dirRel := range dirs {
		chain := dirChainBelowRoot(dirRel)
		counts := make([]int, 0, len(chain))
		var extendErr error
		for _, seg := range chain {
			n, err := ig.extend(seg)
			if err != nil {
				extendErr = fmt.Errorf("engine: read .gitignore for %q: %w", seg, err)
				break
			}
			counts = append(counts, n)
		}

		var dirSymbols []Symbol
		var dirErr error
		if extendErr == nil {
			dirSymbols, dirErr = r.symbolsOfDir(unit, dirRel, ig)
		}

		// Trim exactly what was extended, in reverse, regardless of whether extending or reading this
		// directory failed partway through — an early return here would leave ig holding this
		// directory's patterns for every later call, corrupting every subsequent lookup in this
		// process.
		for i := len(counts) - 1; i >= 0; i-- {
			ig.trim(counts[i])
		}

		if extendErr != nil {
			return nil, extendErr
		}
		if dirErr != nil {
			return nil, dirErr
		}
		symbols = append(symbols, dirSymbols...)
	}

	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].File != symbols[j].File {
			return symbols[i].File < symbols[j].File
		}
		return symbols[i].Start < symbols[j].Start
	})
	return symbols, nil
}

// symbolsOfDir returns every symbol in dirRel whose glyph unit (per unitFor) is exactly unit. It
// reads dirRel once with os.ReadDir, drops every entry ig.match excludes and every directory and
// symlink entry, then reuses dirPackage — the same per-directory clause vote the walk runs — to
// learn each file's own package clause before deciding, file by file, whether that file's unitFor
// result is the unit being searched for.
func (r *Repo) symbolsOfDir(unit, dirRel string, ig *ignoreSet) ([]Symbol, error) {
	osEntries, err := os.ReadDir(r.absDir(dirRel))
	if err != nil {
		return nil, fmt.Errorf("engine: read dir %q: %w", dirRel, err)
	}

	var fileEntries []os.DirEntry
	for _, entry := range osEntries {
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		childRel := joinRel(dirRel, entry.Name())
		if ig.match(childRel, false) {
			continue
		}
		fileEntries = append(fileEntries, entry)
	}

	dirPkg, clauses := r.dirPackage(dirRel, fileEntries)

	var symbols []Symbol
	for _, entry := range fileEntries {
		base := entry.Name()
		clause, ok := clauses[base]
		if !ok {
			// dirPackage never recorded a clause for this file: an unknown extension, an unreadable
			// file, invalid UTF-8, an unparseable file, or an empty package clause. None of those
			// contribute a vote to dirPackage's tie-break, and none of them contribute a symbol here
			// either — the same rule the walk applies.
			continue
		}
		if unitFor(dirRel, dirPkg, clause) != unit {
			continue
		}

		lang, _ := LanguageForExtension(filepath.Ext(base))
		strategy, _ := StrategyFor(lang)
		src, err := os.ReadFile(filepath.Join(r.absDir(dirRel), base))
		if err != nil || !utf8.Valid(src) {
			continue
		}

		fileRel := joinRel(dirRel, base)
		_ = treesitter.WithTree(lang, src, func(root *ts.Node, _ bool) error {
			for _, sym := range strategy.Symbols(unit, root, src) {
				sym.File = fileRel
				symbols = append(symbols, sym)
			}
			return nil
		})
	}
	return symbols, nil
}

// sameOwner reports whether a and b name the same owner chain: both nil (a package-level name), or
// equal element by element. Owner is always either nil or a non-nil one-element slice for Go, per
// docs/glyph.md §3, so this simple comparison never needs to treat a nil slice and an empty one as
// different.
func sameOwner(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SpansOf returns every declaration g names: an empty slice, never nil, when nothing matches, and a
// nil error alongside it — there is no status vocabulary here, unlike the later resolve verb's
// found/multipart/ambiguous/not_found. It returns ErrLanguageUnsupported, wrapped, when g.Lang is
// not glyph.Go, so the error names the real cause rather than surfacing as a directory that happens
// to not exist.
//
// SpansOf then validates g by round-tripping it through glyph.Parse(g.Lang, g.String()), returning
// the resulting parse error wrapped on failure. A Glyph is a plain struct any caller can build by
// hand, so this one check — calling the one implementation of the grammar rather than restating its
// rules here — covers the empty unit, a dot segment, a member that is too deep, and every other
// alphabet violation alike. It is also what makes the walk's claim that no root span is ever
// produced true rather than aspirational: a hand-built Glyph naming the empty root unit is rejected
// here before any directory is ever read.
//
// It then builds a fresh ignoreSet for the repository root carrying the root's own patterns only —
// newIgnoreSet(r.root) followed by one extend(".") — calls symbolsOfUnit, and filters the result by
// owner chain and name.
func (r *Repo) SpansOf(g glyph.Glyph) ([]Symbol, error) {
	if g.Lang != glyph.Go {
		return nil, fmt.Errorf("engine: spans of %q: %w", g.String(), ErrLanguageUnsupported)
	}
	if _, err := glyph.Parse(g.Lang, g.String()); err != nil {
		return nil, fmt.Errorf("engine: spans of %q: %w", g.String(), err)
	}

	ig := newIgnoreSet(r.root)
	if _, err := ig.extend("."); err != nil {
		return nil, fmt.Errorf("engine: read .gitignore for %q: %w", ".", err)
	}

	symbols, err := r.symbolsOfUnit(g.Unit, ig)
	if err != nil {
		return nil, err
	}

	matches := make([]Symbol, 0, len(symbols))
	for _, sym := range symbols {
		if sameOwner(sym.Glyph.Owner, g.Owner) && sym.Glyph.Name == g.Name {
			matches = append(matches, sym)
		}
	}
	return matches, nil
}
