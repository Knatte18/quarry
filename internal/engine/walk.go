// walk.go holds the per-directory work Repo.TOC drives, as unexported methods on *Repo:
// dirDoc, fileEntry, unitSpellable, and the recursion itself, walkDir; and the
// unexported free function unitFor. The per-file clause read, the directory's clause vote, and the
// unit derivation live in the exported PackageClause and UnitsForClauseMap, declared in units.go
// for the caller outside this package that needs them: UnitsForClauseMap's own unitOf
// delegates to unitFor for the per-file derivation, so each rule still exists exactly once.
//
// How many times a file is parsed, and why. Each file is read and parsed once per call, by
// buildRecord in memo.go, which extracts the clause, header, flags, package doc and symbols from
// one parse with a placeholder unit. The unit is a directory-level fact, so it is stamped onto a
// copy of the symbols after the directory's clause vote (fileMemo.dirVote) fixes it.
//
// The cost is priced, not assumed: one parse of every file in this repository is measured at
// 616 ms for 469 files, so a whole-tree walk with symbols is under a second and the round trip's
// own lookup pass brings it to under 2 s — two orders of magnitude inside go test's default
// timeout, with no cache and no concurrency.

package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Knatte18/quarry/glyph"
)

// absDir returns the filesystem path for the repository-relative directory dirRel ("." for the
// repository root itself).
func (r *Repo) absDir(dirRel string) string {
	if dirRel == "." {
		return r.root
	}
	return filepath.Join(r.root, filepath.FromSlash(dirRel))
}

// joinRel joins the repository-relative directory parent with child's base name, using forward
// slashes and treating "." as the repository root.
func joinRel(parent, child string) string {
	if parent == "." {
		return child
	}
	return parent + "/" + child
}

// unitFor derives the glyph unit a file belongs to, from the directory's own repository-relative
// path dirRel, the directory's dominant package clause dirPkg, and this file's own package clause
// fileClause.
//
// A file whose clause is exactly dirPkg+"_test" belongs to the external-test unit dirRel+"_test";
// every other file belongs to dirRel itself. The discriminator is the clause, never the filename:
// keying on the literal "_test" suffix instead would split a package legitimately named mytest or
// httptest into two units it never actually has, and a same-package "_test.go" file (one whose
// clause is dirPkg, not dirPkg+"_test") belongs to the package's own unit, not a second one, even
// though its filename carries "_test" too.
//
// dirRel == "." — the repository root — is an exception to the suffix rule: unitFor returns "" for
// both branches there, never "._test". Left unhandled, a root-level external test package would
// become the unit "._test", which glyph.Parse accepts as a legal segment; root-level "_test" files
// would then be emitted under a spellable unit while their non-test siblings in the same directory
// are excluded as unspellable (the root's own unit is ""), an inconsistency with no upside. The
// empty string keeps the whole root out uniformly instead, which is the same open gap
// unitSpellable's doc comment records, not a second rule.
func unitFor(dirRel, dirPkg, fileClause string) string {
	if dirRel == "." {
		return ""
	}
	if fileClause == dirPkg+"_test" {
		return dirRel + "_test"
	}
	return dirRel
}

// unitSpellable reports whether glyph.Parse accepts unit as a Go unit, by attempting to parse
// unit+"#x" — an arbitrary well-formed member appended solely so Parse has a complete glyph to
// check; only the unit half of the result matters here. Called once per distinct unit encountered
// in a directory, before any extraction from that directory, so its cost is at most a couple of
// Parse calls per directory rather than one per file.
//
// A directory whose unit the Go alphabet cannot spell is still listed — its files still get Name,
// Header, Test and Generated like any other file — but no file entry in it carries Symbols. This
// one rule covers every rejection the alphabet makes: the repository root's empty unit, a "." or
// ".." segment, and a segment holding a space, a backslash, or an ASCII control rune. Emitting
// nothing is the honest answer to a name the contract cannot spell; the alternative would be
// minting a unit spelling glyph.Parse itself rejects. What the repository root's own unit should
// spell is an open gap in the identifier contract (docs/glyph.md §2), not something this engine
// decides on the contract's behalf.
//
// For a unit whose own name carries a "#", the probe string unit+"#x" now carries two of them and
// is rejected as ReasonMultipleSeparators where, before the resolve contract closed over the
// grammar, it was rejected as ReasonMemberBadRune instead — the probe's reason changed, and the
// boolean this function returns from it did not, so every answer this function emits is
// byte-identical to what it emitted before, and the round trip stays true by construction. That
// change is a deliberate asymmetry, not a bug: naming such a directory as a resolve or expand target
// is now an error under the closed grammar — a "#" in a unit component can never be part of a
// well-formed glyph — while encountering one below a listed target during a walk is still a silent
// listing with no symbols, exactly as before. The contract governs what a caller may name; this
// function governs what the walk may mint, and the two are allowed to disagree on a directory
// neither of them expects to see.
func (r *Repo) unitSpellable(unit string) bool {
	_, err := glyph.Parse(glyph.Go, unit+"#x")
	return err == nil
}

// mostCommonClause returns the clause with the highest count in counts, the lexicographically
// smallest clause breaking a tie.
func mostCommonClause(counts map[string]int) string {
	best := ""
	bestCount := -1
	for clause, n := range counts {
		if n > bestCount || (n == bestCount && clause < best) {
			best = clause
			bestCount = n
		}
	}
	return best
}

// dirDoc selects the directory's package documentation from the package docs the file records
// already hold, keyed by base name in docs. Candidates are the files whose clause
// (from clauses) equals pkg, in sorted order with "doc.go" tried first; the first non-empty result
// wins. dirDoc opens no file and parses nothing — it only selects among strings already computed.
// No match returns "", which omitempty turns into an absent key rather than an empty one.
func (r *Repo) dirDoc(clauses map[string]string, docs map[string]string, pkg string) string {
	var candidates []string
	for base, clause := range clauses {
		if clause == pkg {
			candidates = append(candidates, base)
		}
	}
	sort.Strings(candidates)

	ordered := make([]string, 0, len(candidates))
	for _, base := range candidates {
		if base == "doc.go" {
			ordered = append(ordered, base)
		}
	}
	for _, base := range candidates {
		if base != "doc.go" {
			ordered = append(ordered, base)
		}
	}

	for _, base := range ordered {
		if doc := docs[base]; doc != "" {
			return doc
		}
	}
	return ""
}

// fileEntry turns the file record rec of base inside directory dirRel into one FileEntry. It
// reads and parses nothing: the record already holds everything.
//
// A parsed record gives Header, Test, Generated and Lossy as recorded; Package is emitted only
// when the file's own clause differs from the directory vote's v.pkg, Language only when the
// file's language differs from v.lang, and Symbols is set to a non-nil pointer — to a possibly-empty
// slice stamped with the file's unit — only when wantSymbols.
//
// A file with no language gets its recorded Header and never gets Symbols, whatever wantSymbols
// says. A record carrying an error sets Error and leaves Header, Lossy and Symbols unset; the file
// is still listed, never skipped. A parse that reports an error sets Lossy. Error and Lossy are
// never both set.
//
// spellable is the caller's per-directory cache of unit -> unitSpellable(unit), populated lazily
// here on first use of a given unit and reused for every later file sharing it — at most two
// distinct units exist per directory (the directory's own, and its "_test" external-test
// counterpart), so this keeps unitSpellable's glyph.Parse call to at most two per directory rather
// than one per file. When wantSymbols is true but this file's own unit (derived by unitFor from
// dirRel, v.pkg, and the file's clause) is unspellable, Symbols is left nil regardless — the
// directory is still listed, just without symbols, per unitSpellable's own doc comment.
func (r *Repo) fileEntry(dirRel, base string, rec *fileRecord, v dirVote, wantSymbols bool, spellable map[string]bool) FileEntry {
	entry := FileEntry{Name: base}
	if rec.err != "" {
		entry.Error = rec.err
		return entry
	}
	entry.Header = rec.header
	if !rec.parsed {
		return entry
	}

	entry.Test = rec.test
	entry.Generated = rec.generated
	entry.Lossy = rec.lossy
	if rec.clause != v.pkg {
		entry.Package = rec.clause
	}
	if rec.lang != v.lang {
		entry.Language = rec.lang
	}
	if wantSymbols {
		unit := unitFor(dirRel, v.pkg, rec.clause)
		ok, cached := spellable[unit]
		if !cached {
			ok = r.unitSpellable(unit)
			spellable[unit] = ok
		}
		if ok {
			symbols := stampSymbols(rec.symbols, unit, "")
			entry.Symbols = &symbols
		}
	}
	return entry
}

// walkDir answers the directory at dirRel, recursively.
//
// On entry it calls ig.extend(dirRel) and holds the count it returns, and on exit calls ig.trim
// with that same count, so a directory's own patterns are in force for its subtree and are
// dropped again on the way out. walkDir owns the extend/trim for its own directory, at every
// level including the first: the caller hands it a set already carrying the chain from the root
// down to the target's parent and never the target's own, so no directory's patterns are appended
// twice and the trim accounting stays exact.
//
// It reads the directory once with os.ReadDir, drops every entry ig.match excludes, and never
// descends into an excluded directory. An entry whose DirEntry.Type() has fs.ModeSymlink set
// becomes a FileEntry carrying Name alone — never descended into, never opened, no Header, no
// Symbols — whatever its target is; detection is on Type() and never on IsDir(), because IsDir()
// is false for a symlink to a directory, so keying on it would emit a directory as a file entry
// and read a "header" through the link. Not following also means the walk is finite by
// construction and needs no visited set.
//
// identityOnly fills only Dir, Package and Doc — the shape a subdirectory takes at the depth cut.
// Files is sorted lexicographically by Name and Dirs by Dir, both with sort.Slice over the raw
// string, no case folding and no locale.
//
// The answer's Dir is the repository-relative path with forward slashes, "." for the root.
// Language on the directory answer is the language of its package, present only when there is
// one — and, per the identityOnly rule above, never on an identity-only answer: the plan's own
// example shows a depth-zero subdirectory carrying Dir, Package and Doc alone, so Language is set
// only on the non-identityOnly path, after the early return below.
func (r *Repo) walkDir(dirRel string, ig *ignoreSet, depth int, wantSymbols bool, identityOnly bool, m *fileMemo) (DirAnswer, error) {
	n, err := ig.extend(dirRel)
	if err != nil {
		return DirAnswer{}, fmt.Errorf("engine: read .gitignore for %q: %w", dirRel, err)
	}
	defer ig.trim(n)

	osEntries, err := os.ReadDir(r.absDir(dirRel))
	if err != nil {
		return DirAnswer{}, fmt.Errorf("engine: read dir %q: %w", dirRel, err)
	}

	var dirEntries, fileEntries, symlinkEntries []os.DirEntry
	for _, entry := range osEntries {
		childRel := joinRel(dirRel, entry.Name())
		if ig.match(childRel, entry.IsDir()) {
			continue
		}
		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			symlinkEntries = append(symlinkEntries, entry)
		case entry.IsDir():
			dirEntries = append(dirEntries, entry)
		default:
			fileEntries = append(fileEntries, entry)
		}
	}

	// An identity-only answer carries no files, so its records need no symbols.
	want := wantSymbols && !identityOnly
	bases := make([]string, 0, len(fileEntries))
	for _, entry := range fileEntries {
		bases = append(bases, entry.Name())
	}
	recs := m.dirRecords(dirRel, bases, func(string) bool { return want })
	v := m.dirVote(dirRel, recs, true)

	answer := DirAnswer{Dir: dirRel}
	if v.pkg != "" {
		answer.Package = v.pkg
	}

	docs := make(map[string]string)
	spellable := make(map[string]bool)
	files := make([]FileEntry, 0, len(fileEntries)+len(symlinkEntries))
	for _, base := range bases {
		rec := recs[base]
		if rec.packageDoc != "" {
			docs[base] = rec.packageDoc
		}
		files = append(files, r.fileEntry(dirRel, base, rec, v, want, spellable))
	}
	for _, entry := range symlinkEntries {
		files = append(files, FileEntry{Name: entry.Name()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	answer.Doc = r.dirDoc(v.clauses, docs, v.pkg)

	if identityOnly {
		return answer, nil
	}
	if v.pkg != "" {
		answer.Language = v.lang
	}
	answer.Files = files

	var dirs []DirAnswer
	for _, entry := range dirEntries {
		childRel := joinRel(dirRel, entry.Name())
		childIdentityOnly := depth == 0
		childDepth := depth
		if !childIdentityOnly && depth != DepthAll {
			childDepth = depth - 1
		}
		childAnswer, err := r.walkDir(childRel, ig, childDepth, wantSymbols, childIdentityOnly, m)
		if err != nil {
			return DirAnswer{}, err
		}
		dirs = append(dirs, childAnswer)
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Dir < dirs[j].Dir })
	answer.Dirs = dirs

	return answer, nil
}
