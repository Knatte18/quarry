// enclose.go implements the enclose query: mapping path:line ranges to the members that enclose them.
// It declares the query's input (Location, RevisionFiles), its answer (EncloseResult) and its closed
// rejection vocabulary.

package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/Knatte18/quarry/glyph"
)

// EncloseResult is the answer to one Location passed to Enclose or EncloseFrom.
// It is either an outcome (Status set) or a rejection (Reason and Error set), never both.
type EncloseResult struct {
	// Target is the caller's location spelling, verbatim. Always present.
	Target string `json:"target"`
	// File is the normalised repository-relative path.
	// Start and End are the line range.
	// All three are filled on a rejection whenever they were determined before it.
	File  string `json:"file,omitempty"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
	// Status is StatusFound when at least one member encloses part of the range
	// and StatusNotFound when none does.
	// It is empty exactly when the location was rejected.
	Status Status `json:"status,omitempty"`
	// Symbols are the innermost members touching the range, in source order, present only for found.
	Symbols []Symbol `json:"symbols,omitempty"`
	// Unit is set only on not_found.
	// Unlike ResolveResult.Unit (a Status), this key holds the file's self glyph string (e.g. a/b.go#),
	// and its presence alone means the range is not inside any member.
	Unit string `json:"unit,omitempty"`
	// Lossy is true when the file parsed only partially, so the answer may be incomplete.
	Lossy bool `json:"lossy,omitempty"`
	// Error is the one-sentence rejection message, set only on a rejection.
	Error string `json:"error,omitempty"`
	// Reason is the rejection's plain-word reason, one of EncloseReasons, set only on a rejection.
	Reason string `json:"reason,omitempty"`
}

// Rejected reports whether r is a rejection rather than an outcome.
// It reads r.Status because Status is empty exactly when the location was rejected.
func (r EncloseResult) Rejected() bool { return r.Status == "" }

// The eight rejection reasons of an enclose location, in the order the checks run.
const (
	// EncloseReasonBadLocation marks a target that is not path:line, path:line-line or path:line:col.
	EncloseReasonBadLocation = "bad_location"
	// EncloseReasonBadRange marks a range whose start is below 1 or after its end.
	EncloseReasonBadRange = "bad_range"
	// EncloseReasonOutsideRoot marks a path that normalises outside the repository root.
	EncloseReasonOutsideRoot = "outside_root"
	// EncloseReasonUnaddressable marks a file whose members no glyph can name:
	// its path or its glyph unit cannot be spelled.
	EncloseReasonUnaddressable = "unaddressable"
	// EncloseReasonUnsupportedLanguage marks a file whose extension has no registered language strategy.
	EncloseReasonUnsupportedLanguage = "unsupported_language"
	// EncloseReasonMissingFile marks a path with no regular file behind it.
	EncloseReasonMissingFile = "missing_file"
	// EncloseReasonUnreadable marks a file that exists but cannot be read as source.
	EncloseReasonUnreadable = "unreadable"
	// EncloseReasonPastEOF marks a range ending after the file's last line.
	EncloseReasonPastEOF = "past_eof"
)

// EncloseReasons lists all eight enclose rejection reasons, in the same order as the constant block above.
// Go cannot reflect over package-level constants, so this slice is the only way a test or a caller can
// enumerate the vocabulary.
// Adding a constant means adding it here in the same edit, exactly as NameReasons does.
var EncloseReasons = []string{
	EncloseReasonBadLocation,
	EncloseReasonBadRange,
	EncloseReasonOutsideRoot,
	EncloseReasonUnaddressable,
	EncloseReasonUnsupportedLanguage,
	EncloseReasonMissingFile,
	EncloseReasonUnreadable,
	EncloseReasonPastEOF,
}

// Location is one input to Enclose: either one normalised location (File, Start and End set)
// or one the caller already rejected (Reason and Error set, plus whichever of File, Start and End
// were determined before the rejection).
// Target is the caller's spelling and is echoed on the answer.
type Location struct {
	Target, File  string
	Start, End    int
	Reason, Error string
}

// RevisionFiles is the engine's git-ignorant view of one revision's files.
type RevisionFiles interface {
	// GoFiles returns the Go files that are immediate children of dirRel at the revision,
	// as repository-relative paths; dirRel "." is the repository root.
	GoFiles(dirRel string) ([]string, error)
	// Read returns one file's bytes at the revision.
	Read(rel string) ([]byte, error)
}

// innermostSymbols returns the members of candidates that enclose the line range [start, end] innermost-first,
// in source order.
//
// A candidate is kept when its [Start, End] span overlaps the range.
// A kept symbol P has as children the other kept symbols whose span lies within P's span and is not identical to it.
// P survives when some line in the overlap of the range and P's span is covered by none of P's children;
// a P with no children always survives.
// Survivors are sorted stably by Start then End, so input order breaks ties, and deduplicated by identity (same ID and Start).
// The result is never nil.
//
// Nesting is decided by span containment only, never by Glyph.Owner, which gives these outcomes:
// a line in an interface method returns the method alone;
// a range over an interface's head lines plus one method returns the type, then the method;
// a receiver method outside its type's span is never that type's child;
// a one-line interface returns the type, then its method, because their identical spans do not nest.
func innermostSymbols(candidates []Symbol, start, end int) []Symbol {
	type identity struct {
		id    string
		start int
	}
	seen := make(map[identity]bool, len(candidates))
	var kept []Symbol
	for _, sym := range candidates {
		key := identity{sym.ID, sym.Start}
		if sym.Start > end || sym.End < start || seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, sym)
	}

	survivors := make([]Symbol, 0, len(kept))
	for _, parent := range kept {
		var children []Symbol
		for _, other := range kept {
			within := other.Start >= parent.Start && other.End <= parent.End
			identical := other.Start == parent.Start && other.End == parent.End
			if within && !identical {
				children = append(children, other)
			}
		}
		if len(children) == 0 || touchesUncoveredLine(parent, children, start, end) {
			survivors = append(survivors, parent)
		}
	}

	sort.SliceStable(survivors, func(i, j int) bool {
		if survivors[i].Start != survivors[j].Start {
			return survivors[i].Start < survivors[j].Start
		}
		return survivors[i].End < survivors[j].End
	})
	return survivors
}

// touchesUncoveredLine reports whether some line in both [start, end] and parent's span lies outside every child's span.
func touchesUncoveredLine(parent Symbol, children []Symbol, start, end int) bool {
	for line := max(start, parent.Start); line <= min(end, parent.End); line++ {
		covered := false
		for _, child := range children {
			if line >= child.Start && line <= child.End {
				covered = true
				break
			}
		}
		if !covered {
			return true
		}
	}
	return false
}

// Enclose answers each location in locs against the working tree, positionally:
// the returned slice has exactly len(locs) elements and element i answers locs[i], with Target echoed.
// A nil or empty locs yields an empty, non-nil slice.
// A location the caller already rejected passes through unchanged;
// every other rejection (unsupported language, missing or unreadable file, unspellable unit, range past
// the end of the file) is that item's answer and never fails the call, which has no whole-call error.
//
// Each file is read and parsed at most once per call, however many locations fall in it,
// and nothing built here outlives the call.
// A found answer carries exactly the glyphs a table-of-contents query mints for the file;
// the working-tree limits of resolving them are documented on the facade's Enclose.
func (r *Repo) Enclose(locs []Location) ([]EncloseResult, error) {
	return r.enclose(locs, newFileMemo(r, true))
}

// enclose is Enclose's worker.
// It takes the memo so a test can read its build counts.
func (r *Repo) enclose(locs []Location, m *fileMemo) ([]EncloseResult, error) {
	results := make([]EncloseResult, len(locs))
	outcomes := make(map[string]fileOutcome)
	for i, loc := range locs {
		results[i] = encloseLocation(loc, outcomes, func(file string) fileOutcome {
			return r.worktreeFileOutcome(file, m)
		})
	}
	verifyEncloseCoverage(locs, results)
	return results, nil
}

// verifyEncloseCoverage guards enclose's positional contract:
// one answer per location in argument order, with Target echoed verbatim on every answer.
// It panics rather than returning an error because the violation is unreachable by construction:
// the workers build results with make(len(locs)) and fill every index in order,
// so a divergence means the engine is broken and every answer in the batch is untrustworthy.
// A whole-call failure (a nil slice with an error) is not a coverage violation and is never verified here.
func verifyEncloseCoverage(locs []Location, results []EncloseResult) {
	if len(results) != len(locs) {
		panic(fmt.Sprintf("engine: enclose returned %d results for %d locations", len(results), len(locs)))
	}
	for i, loc := range locs {
		if results[i].Target != loc.Target {
			panic(fmt.Sprintf("engine: enclose result %d answers %q; want %q", i, results[i].Target, loc.Target))
		}
	}
}

// fileOutcome is what one file contributes to the answers of every location in it:
// either a rejection of the whole file (reason and message set),
// or the file's symbols, line count and lossy flag.
type fileOutcome struct {
	reason, message string
	candidates      []Symbol
	lines           int
	lossy           bool
}

// rejectedFile builds the outcome of a file rejected for reason with the given message.
func rejectedFile(reason, message string) fileOutcome {
	return fileOutcome{reason: reason, message: message}
}

// unreadableFile builds the outcome of a file that exists but cannot be read as source.
func unreadableFile(file, detail string) fileOutcome {
	return rejectedFile(EncloseReasonUnreadable, fmt.Sprintf("file unreadable: %s: %s", file, detail))
}

// unspellableUnitFile builds the outcome of a file whose glyph unit cannot be spelled.
func unspellableUnitFile(file string) fileOutcome {
	return rejectedFile(EncloseReasonUnaddressable, fmt.Sprintf("file's glyph unit cannot be spelled: %s", file))
}

// encloseRejection builds the answer rejecting loc with reason and message,
// carrying whichever of File, Start and End loc holds.
func encloseRejection(loc Location, reason, message string) EncloseResult {
	return EncloseResult{
		Target: loc.Target, File: loc.File, Start: loc.Start, End: loc.End,
		Reason: reason, Error: message,
	}
}

// encloseLocation answers one location.
// A pre-rejected location passes through unchanged.
// An unsupported language is rejected before the file is looked at.
// Otherwise outcomes memoises fileOutcomeOf per file, so each file is examined once per call,
// and the file's outcome is turned into this location's answer.
func encloseLocation(loc Location, outcomes map[string]fileOutcome, fileOutcomeOf func(file string) fileOutcome) EncloseResult {
	if loc.Reason != "" {
		return encloseRejection(loc, loc.Reason, loc.Error)
	}
	lang, hasLang := LanguageForExtension(filepath.Ext(loc.File))
	if _, hasStrategy := StrategyFor(lang); !hasLang || !hasStrategy {
		return encloseRejection(loc, EncloseReasonUnsupportedLanguage, fmt.Sprintf("unsupported language: %s", loc.File))
	}
	outcome, ok := outcomes[loc.File]
	if !ok {
		outcome = fileOutcomeOf(loc.File)
		outcomes[loc.File] = outcome
	}
	return answerFromOutcome(loc, outcome)
}

// answerFromOutcome turns a file's outcome into the answer for one location inside it:
// the file's rejection, past_eof, found with the innermost members, or not_found with the file's self glyph.
func answerFromOutcome(loc Location, outcome fileOutcome) EncloseResult {
	if outcome.reason != "" {
		return encloseRejection(loc, outcome.reason, outcome.message)
	}
	if loc.End > outcome.lines {
		return encloseRejection(loc, EncloseReasonPastEOF,
			fmt.Sprintf("range ends past end of file: end %d, file has %d lines", loc.End, outcome.lines))
	}
	res := EncloseResult{Target: loc.Target, File: loc.File, Start: loc.Start, End: loc.End, Lossy: outcome.lossy}
	if symbols := innermostSymbols(outcome.candidates, loc.Start, loc.End); len(symbols) > 0 {
		res.Status = StatusFound
		res.Symbols = symbols
		return res
	}
	self, err := glyph.Self(glyph.Go, loc.File)
	if err != nil {
		return encloseRejection(loc, EncloseReasonUnaddressable,
			fmt.Sprintf("path cannot be spelled as a glyph unit: %s", loc.File))
	}
	res.Status = StatusNotFound
	res.Unit = self.String()
	return res
}

// worktreeFileOutcome examines file under the repository root:
// the filesystem decides missing and symlink files first,
// then the memo-aware table-of-contents path supplies the file's symbols, unit and lossy flag.
// The unit comes from the DirAnswer that same call returned, never from the memo's votes,
// because an explicitly named gitignored file votes outside the memo.
func (r *Repo) worktreeFileOutcome(file string, m *fileMemo) fileOutcome {
	missing := rejectedFile(EncloseReasonMissingFile, fmt.Sprintf("file not found: %s", file))
	info, err := os.Lstat(filepath.Join(r.root, filepath.FromSlash(file)))
	switch {
	case os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR):
		return missing
	case err != nil:
		return unreadableFile(file, err.Error())
	case info.IsDir():
		return missing
	case info.Mode()&fs.ModeSymlink != 0:
		return unreadableFile(file, "symlink")
	}

	on := true
	answer, err := r.toc(file, TOCOptions{Symbols: &on}, m)
	if errors.Is(err, ErrTargetNotFound) {
		return missing
	}
	if err != nil {
		return unreadableFile(file, err.Error())
	}
	entry := answer.Files[0]
	if entry.Error != "" {
		return unreadableFile(file, entry.Error)
	}

	rec := m.records[file]
	dir, _ := splitDirBase(file)
	if !r.unitSpellable(unitFor(dir, answer.Package, rec.clause)) {
		return unspellableUnitFile(file)
	}
	candidates := make([]Symbol, len(*entry.Symbols))
	for i, sym := range *entry.Symbols {
		sym.File = file
		candidates[i] = sym
	}
	return fileOutcome{candidates: candidates, lines: rec.lines, lossy: entry.Lossy}
}

// EncloseFrom answers each location in locs against the revision files exposes, positionally,
// exactly as Enclose answers against the working tree:
// element i answers locs[i], and a nil or empty locs yields an empty, non-nil slice.
// The revision has no ignore set and no symlinks: every Go file files lists votes on its directory's package.
//
// The only whole-call failures are a GoFiles or Read error, which return a nil slice and that error;
// every per-location problem is that item's rejection.
// Each file is read and parsed at most once per call, and nothing built here outlives the call.
// The asymmetries between a revision answer and a working-tree answer are documented on the facade's EncloseAt.
func (r *Repo) EncloseFrom(files RevisionFiles, locs []Location) ([]EncloseResult, error) {
	return r.encloseFrom(files, locs, newFileMemo(r, true))
}

// revisionDir is one directory at the revision: the base names it lists, their records and their clause vote.
type revisionDir struct {
	listed map[string]bool
	recs   map[string]*fileRecord
	vote   dirVote
}

// encloseFrom is EncloseFrom's worker.
// It takes the memo so a test can read its build counts.
func (r *Repo) encloseFrom(files RevisionFiles, locs []Location, m *fileMemo) ([]EncloseResult, error) {
	// buildRecord turns every read failure into a record error,
	// so the first failure is captured here to reach the caller as a whole-call error.
	var readErr error
	m.read = func(rel string) ([]byte, error) {
		src, err := files.Read(rel)
		if err != nil && readErr == nil {
			readErr = err
		}
		return src, err
	}

	var callErr error
	dirs := make(map[string]revisionDir)
	loadDir := func(dir string) (revisionDir, bool) {
		if d, ok := dirs[dir]; ok {
			return d, true
		}
		paths, err := files.GoFiles(dir)
		if err != nil {
			callErr = err
			return revisionDir{}, false
		}
		bases := make([]string, len(paths))
		listed := make(map[string]bool, len(paths))
		for i, p := range paths {
			bases[i] = path.Base(p)
			listed[bases[i]] = true
		}
		recs := m.dirRecords(dir, bases, func(string) bool { return true })
		if readErr != nil {
			callErr = readErr
			return revisionDir{}, false
		}
		d := revisionDir{listed: listed, recs: recs, vote: m.dirVote(dir, recs, true)}
		dirs[dir] = d
		return d, true
	}

	results := make([]EncloseResult, len(locs))
	outcomes := make(map[string]fileOutcome)
	for i, loc := range locs {
		results[i] = encloseLocation(loc, outcomes, func(file string) fileOutcome {
			dir, base := splitDirBase(file)
			d, ok := loadDir(dir)
			if !ok {
				return fileOutcome{}
			}
			return r.revisionFileOutcome(file, dir, base, d)
		})
		if callErr != nil {
			return nil, callErr
		}
	}
	verifyEncloseCoverage(locs, results)
	return results, nil
}

// revisionFileOutcome examines base inside the revision directory d:
// a base the revision does not list is missing, a record error is unreadable,
// and an unspellable unit is unaddressable before any symbol is stamped.
func (r *Repo) revisionFileOutcome(file, dir, base string, d revisionDir) fileOutcome {
	if !d.listed[base] {
		return rejectedFile(EncloseReasonMissingFile, fmt.Sprintf("file not found: %s", file))
	}
	rec := d.recs[base]
	if rec.err != "" {
		return unreadableFile(file, rec.err)
	}
	unit := unitFor(dir, d.vote.pkg, rec.clause)
	if !r.unitSpellable(unit) {
		return unspellableUnitFile(file)
	}
	return fileOutcome{candidates: stampSymbols(rec.symbols, unit, file), lines: rec.lines, lossy: rec.lossy}
}
