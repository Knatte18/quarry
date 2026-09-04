// resolve.go implements the internal span lookup and the exported resolve verb built on it.
// Repo.unitDirs maps a glyph unit back to the directory or directories that hold it,
// Repo.symbolsOfUnit parses those directories' files once and returns every symbol they declare,
// and Repo.SpansOf is the single-glyph convenience the round trip is written against — Resolve and
// Expand are built on symbolsOfUnit through the per-call unitMemo instead, so each unit is parsed
// once per call rather than once per glyph. SpansOf still returns an empty slice with no status
// when nothing matches; Repo.Resolve, above it in this file, holds the status vocabulary and
// promotes unitDirs' collision flag into it.

package engine

import (
	"errors"
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
// the union of both directories' symbols; Resolve, in this same file, reads the flag through
// unitMemo.dirsOf and promotes it into an "ambiguous" status. The status type it promotes into
// lives in internal/engine/answer.go, not here.
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

// unitDirsResult is one memoised unitDirs answer. It carries no error because unitDirs itself
// returns none — unitDirs is two dirExists calls, each an os.Lstat whose failure is reported as
// "not a directory", never as a read error the caller must handle.
type unitDirsResult struct {
	dirs      []string
	collision bool
}

// unitMemo is the per-call memo Resolve and Expand build once and discard when their call returns:
// it turns repeated lookups of the same unit across many targets into one parse and one unitDirs
// call per unit, rather than one per target. unitMemo is a local of the exported entry point and
// dies with it — nothing is stored on Repo, because a memo that cannot outlive the call it was
// built in cannot go stale, which is what keeps this inside the engine's no-cache rule while still
// turning twenty lookups over five units into five parses instead of twenty. symbolsOf is the only
// site in either verb that calls symbolsOfUnit; SpansOf is deliberately not used by either verb,
// because it is the per-glyph wrapper and calling it per target would re-parse each unit once per
// target.
type unitMemo struct {
	repo *Repo
	ig   *ignoreSet
	// symbols memoises symbolsOfUnit's result per unit.
	symbols map[string][]Symbol
	// dirs memoises unitDirs' result per unit.
	dirs map[string]unitDirsResult
	// parses counts the symbolsOfUnit calls this memo has made. It is the seam that makes the
	// "each unit is parsed once" grouping guarantee observable: read by tests and never by
	// production code, since a map's entry count is true by construction and wall-clock is the
	// very thing the guarantee's test must be independent of.
	parses int
}

// newUnitMemo builds an empty unitMemo for r, with an ignore set carrying the repository root's own
// patterns and nothing below them — newIgnoreSet(r.root) followed by one extend(".") — exactly as
// SpansOf builds its own. That is precisely what symbolsOfUnit's doc comment requires of its
// caller; symbolsOfUnit extends and trims per directory itself.
func newUnitMemo(r *Repo) (*unitMemo, error) {
	ig := newIgnoreSet(r.root)
	if _, err := ig.extend("."); err != nil {
		return nil, fmt.Errorf("engine: read .gitignore for %q: %w", ".", err)
	}
	return &unitMemo{
		repo:    r,
		ig:      ig,
		symbols: make(map[string][]Symbol),
		dirs:    make(map[string]unitDirsResult),
	}, nil
}

// symbolsOf returns unit's symbols, parsing it on the first call and returning the memoised slice
// on every later one. parses is incremented before the underlying call, not after a successful
// return, so the counter means what its doc comment says: calls made, not calls that succeeded.
func (m *unitMemo) symbolsOf(unit string) ([]Symbol, error) {
	if symbols, ok := m.symbols[unit]; ok {
		return symbols, nil
	}
	m.parses++
	symbols, err := m.repo.symbolsOfUnit(unit, m.ig)
	if err != nil {
		return nil, err
	}
	m.symbols[unit] = symbols
	return symbols, nil
}

// dirsOf returns unit's directory list and collision flag, memoising m.repo.unitDirs(unit). It
// returns no error, for the reason recorded on unitDirsResult.
func (m *unitMemo) dirsOf(unit string) unitDirsResult {
	if res, ok := m.dirs[unit]; ok {
		return res
	}
	dirs, collision := m.repo.unitDirs(unit)
	res := unitDirsResult{dirs: dirs, collision: collision}
	m.dirs[unit] = res
	return res
}

// isGlyphTarget reports whether target is a glyph rather than a repository-relative path, by
// strings.Contains(target, "#") and nothing else. A target containing "#" is a glyph and is handed
// to glyph.Parse, which decides whether it is a well-formed one; everything else is a
// repository-relative path. This split never pre-empts any of the grammar's own rejections — "#x",
// with an empty unit, is a glyph target that glyph.Parse then rejects, not a path.
func isGlyphTarget(target string) bool {
	return strings.Contains(target, "#")
}

// statusForMatches is the whole decision Resolve and Expand turn on, expressed once so the two
// verbs can never disagree about what a glyph resolves to. Its rows, checked in this exact order:
//
//  1. Zero matches is StatusNotFound. Checked first, so a unit collision with no match in either
//     directory is not_found rather than an ambiguous with nothing to be ambiguous between — and
//     that not_found carries unit: found, which is what a plan card creating the first declaration
//     of an existing unit needs.
//  2. A collision is StatusAmbiguous. Checked before every row below, so a single match under a
//     collision is ambiguous rather than found: a found whose glyph string names two different
//     units is exactly the failure the literal-first unit lookup exists to prevent. Several init
//     under a collision is ambiguous too, not multipart — the glyph does not unambiguously name one
//     unit, and multipart would assert that it does.
//  3. Exactly one match is StatusFound.
//  4. An owner-less "init" name is StatusMultipart. This is the only Go glyph that names one symbol
//     the language lets be declared in several places, and the discriminator is the glyph's own
//     name rather than any property of the declarations — decidable without reading build tags,
//     which the engine has no business interpreting. A build-tagged pair of init functions is still
//     init and still multipart.
//  5. Everything else is StatusAmbiguous.
//
// statusForMatches does not evaluate build constraints: doing so would make the answer depend on a
// GOOS and GOARCH the engine does not know and the caller did not state, and reporting both
// candidates with their files is the honest answer where guessing one is a silent pick. The caller
// places the matches — Symbols for found and multipart, Candidates for ambiguous, neither for
// not_found — so the status and the placement have one source.
func statusForMatches(g glyph.Glyph, matches []Symbol, collision bool) Status {
	switch {
	case len(matches) == 0:
		return StatusNotFound
	case collision:
		return StatusAmbiguous
	case len(matches) == 1:
		return StatusFound
	case len(g.Owner) == 0 && g.Name == "init":
		return StatusMultipart
	default:
		return StatusAmbiguous
	}
}

// matchesFor returns every symbol in symbols whose owner chain and name equal g's, reusing
// sameOwner and preserving symbols' own order — and therefore the file-then-line order
// symbolsOfUnit established. It is the filter SpansOf performs inline, written once here so
// Resolve and Expand share one copy.
//
// SpansOf keeps its own inline copy of that filter rather than calling matchesFor, and this task
// changes no line of SpansOf's code — that duplication is deliberate rather than overlooked, for
// two reasons. First, this task's scope permits exactly two edits to code T3 wrote for its own
// purposes, both named and both elsewhere in this plan; a third, however behaviour-preserving,
// would be a scope change this plan does not make. Second, SpansOf is what T3's round trip is
// written against, so refactoring it would put a test this task must keep passing onto code this
// task rewrote. Whether the two filters collapse into one is a question for whichever later task
// next edits SpansOf for its own reasons.
func matchesFor(symbols []Symbol, g glyph.Glyph) []Symbol {
	matches := make([]Symbol, 0, len(symbols))
	for _, sym := range symbols {
		if sameOwner(sym.Glyph.Owner, g.Owner) && sym.Glyph.Name == g.Name {
			matches = append(matches, sym)
		}
	}
	return matches
}

// resolveGlyphTarget answers one glyph target.
//
// It parses target with glyph.Parse(glyph.Go, target) — the alphabet is hardcoded; a multi-alphabet
// dispatch is where a second language would enter, and nothing about the answer's key set depends
// on it. On a parse failure it extracts the *glyph.ParseError with errors.As and returns a
// ResolveResult whose Target is the argument, whose Error is the error's Error() text, whose Reason
// is string(parseErr.Reason), and whose Status is left empty — with a nil error, so the call
// continues and the other targets are still answered. The four statuses are resolution outcomes,
// and a string that is not a glyph was never searched for; answering not_found would tell a
// validator the name is free when the truth is that it is unspellable. Should errors.As ever fail —
// glyph.Parse returns nothing else, so it cannot — Reason is left empty rather than panicking on a
// nil pointer.
//
// Otherwise it sets Target to the argument and ID to the parsed glyph's String(). It reads
// m.dirsOf(g.Unit) for the collision flag and the directory list, then m.symbolsOf(g.Unit). An
// error from symbolsOf is returned as this function's error and fails the whole call: an engine
// read failure is not an answer about a glyph, and a unit that failed to read would otherwise
// answer not_found for every glyph in it, which a done-check reads as success. A unit whose
// directory does not exist is not that case and never reaches it — unitDirs returns an empty slice
// and symbolsOfUnit over zero directories returns an empty slice and a nil error.
//
// It filters with matchesFor, switches on statusForMatches(g, matches, res.collision), and
// populates: StatusFound and StatusMultipart set Symbols to the matches; StatusAmbiguous sets
// Candidates to the matches; StatusNotFound sets neither and sets Unit to StatusFound when the
// memoised directory list is non-empty and to StatusNotFound when it is empty. Unit is set on
// not_found and on nothing else — docs/glyph.md §5 attaches it to the miss, and a found that also
// said so would be clutter.
//
// unit: found is directory existence and nothing more, so it is unitDirs' answer restated
// nowhere: deriving it any other way would be a second unit-to-directory implementation in one
// package, drifting on the corner case the literal-first rule was written for. The consequence: a
// "_test" unit whose stripped directory exists but which holds no external test package at all
// reports unit: found, because the directory is there, and a card creating that package's first
// file is exactly the case that wants it. The third of this task's contract gaps, closed by
// nobody: dirExists uses os.Lstat, which refuses a symlink only in the path's final component, so a
// unit reached through an intermediate symlinked directory resolves here while the walk never
// descends that directory and so never lists those declarations. The behaviour is inherited
// unchanged and deliberately not narrowed — changing dirExists would be a change to the walk's
// inverse, and whether a unit may be reached through a link at all is a statement docs/glyph.md does
// not make.
//
// The identifier-contract gap over the external test unit versus a same-named real directory is
// recorded on unitDirs' own doc comment, almost verbatim, and is not restated here. This function
// adds only what is new: this verb promotes the reported collision to ambiguous, which is the half
// of the gap unitDirs could not state because the status type did not exist when it was written.
func (r *Repo) resolveGlyphTarget(target string, m *unitMemo) (ResolveResult, error) {
	g, err := glyph.Parse(glyph.Go, target)
	if err != nil {
		reason := ""
		var parseErr *glyph.ParseError
		if errors.As(err, &parseErr) {
			reason = string(parseErr.Reason)
		}
		return ResolveResult{Target: target, Error: err.Error(), Reason: reason}, nil
	}

	res := ResolveResult{Target: target, ID: g.String()}

	dirsRes := m.dirsOf(g.Unit)
	symbols, err := m.symbolsOf(g.Unit)
	if err != nil {
		return ResolveResult{}, err
	}

	matches := matchesFor(symbols, g)
	status := statusForMatches(g, matches, dirsRes.collision)
	res.Status = status
	switch status {
	case StatusFound, StatusMultipart:
		res.Symbols = matches
	case StatusAmbiguous:
		res.Candidates = matches
	case StatusNotFound:
		if len(dirsRes.dirs) > 0 {
			res.Unit = StatusFound
		} else {
			res.Unit = StatusNotFound
		}
	}
	return res, nil
}

// resolvePathTarget answers a target with no "#" as a repository-relative path, by calling r.TOC
// with TOCOptions{Depth: 0, Symbols: &symbolsOff} where symbolsOff is a local false. Reusing the
// directory answer rather than restating the rule is what makes the explicitly-named-gitignored-
// target rule, the never-follow-a-symlink rule and the empty-string-and-dot-mean-the-root rule hold
// here for free and keeps them from drifting.
//
// Disposition, in this order: a nil error sets Status to StatusFound and Dir to the address of the
// returned DirAnswer; errors.Is(err, ErrTargetNotFound) sets Status to StatusNotFound with Dir
// absent; errors.Is(err, ErrTargetOutsideRepo) returns an entry with Status empty and Error set to
// the error's text and Reason empty, the second and last member of the per-entry error domain; any
// other error is returned as this function's error and fails the whole call. TOC's two sentinels
// are the only errors this verb converts into an answer. Target is always set to the argument; ID
// and Unit are never set on a path result, because a path has no glyph and belongs to no unit.
//
// Symbols are switched off explicitly rather than left to the per-target default, which would turn
// them on for a file target: this verb answers where a thing is and whether it exists, and what is
// inside it is the table-of-contents question. A plan card whose target is a Markdown page has no
// symbols to want, and paying a tree-sitter parse per Go path target inside a call measured against
// a 150 ms budget would be a cost with no consumer. A file target's answer is its enclosing
// directory's answer holding exactly that one file entry — the shape TOC already produces — so the
// caller can read the package and language a bare file entry would not carry.
func (r *Repo) resolvePathTarget(target string) (ResolveResult, error) {
	symbolsOff := false
	dir, err := r.TOC(target, TOCOptions{Depth: 0, Symbols: &symbolsOff})
	switch {
	case err == nil:
		return ResolveResult{Target: target, Status: StatusFound, Dir: &dir}, nil
	case errors.Is(err, ErrTargetNotFound):
		return ResolveResult{Target: target, Status: StatusNotFound}, nil
	case errors.Is(err, ErrTargetOutsideRepo):
		return ResolveResult{Target: target, Error: err.Error()}, nil
	default:
		return ResolveResult{}, err
	}
}

// Resolve answers every target in targets, positionally: the returned slice has exactly
// len(targets) elements, and element i answers targets[i]. A target repeated twice is answered
// twice. That positional 1:1 mapping is what a caller resolving every glyph of a draft in one call
// needs to map each answer back to the card that asked for it, and it is the only shape where a
// duplicate or a malformed target cannot silently vanish.
//
// A ResolveResult expresses either a resolution outcome or a pre-resolution rejection of the target
// string, never an engine failure: every engine error other than the two TOC sentinels the path
// branch converts fails the whole call, and losing the other answers is right precisely because an
// engine failure makes the whole answer untrustworthy, unlike a malformed target, which taints only
// itself.
//
// "Grouped by unit" is an execution property, not the output shape — the answer is flat, and each
// distinct unit is parsed exactly once per call by the memo. The output is not grouped because a
// per-unit group's natural key is "unit" holding a path, and docs/glyph.md §5 already spells "unit"
// as a key holding "found" or "not_found".
//
// Ordering: results come back in argument order, and within a result Symbols and Candidates are
// ordered by file then by start line, file comparison being the raw repository-relative
// forward-slash string under sort.Strings semantics with no case folding and no locale. No caller
// sorts.
//
// Resolve builds one unitMemo with newUnitMemo, returning that error if it fails, and delegates to
// the unexported resolve.
func (r *Repo) Resolve(targets []string) ([]ResolveResult, error) {
	m, err := newUnitMemo(r)
	if err != nil {
		return nil, err
	}
	return r.resolve(targets, m)
}

// resolve is Resolve's unexported worker. It takes the memo, rather than building its own, so a
// test can construct one, pass it in, and read parses afterwards; Resolve itself never exposes it.
//
// resolve allocates a result slice of exactly len(targets), and for each target in order calls
// resolveGlyphTarget when isGlyphTarget reports true and resolvePathTarget otherwise. Any error from
// either fails the whole call: it returns a nil slice and that error, unwrapped further only if it
// does not already name the target or unit it was reading. A nil targets slice yields an empty,
// non-nil result slice and a nil error.
func (r *Repo) resolve(targets []string, m *unitMemo) ([]ResolveResult, error) {
	results := make([]ResolveResult, len(targets))
	for i, target := range targets {
		var res ResolveResult
		var err error
		if isGlyphTarget(target) {
			res, err = r.resolveGlyphTarget(target, m)
		} else {
			res, err = r.resolvePathTarget(target)
		}
		if err != nil {
			return nil, err
		}
		results[i] = res
	}
	return results, nil
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
// check in seconds rather than minutes, and Resolve needs the same grouping for the many glyphs
// one call can name — the grouping is realised in unitMemo, below.
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
// nil error alongside it — there is no status vocabulary here, unlike Resolve's
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
