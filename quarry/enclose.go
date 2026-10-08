// enclose.go declares the facade's enclose methods, Enclose and EncloseAt, the location-spelling
// parser behind them, and the git-backed adapter that lets the engine read a revision's files.

package quarry

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine"
	"github.com/Knatte18/quarry/internal/gitsrc"
	"github.com/Knatte18/quarry/internal/repopath"
)

// locationPattern matches path:line, path:line-line and path:line:col. Its lazy path group takes
// the shortest path that leaves a valid suffix, so a:12:5 is path a, line 12, column 5.
var locationPattern = regexp.MustCompile("^(.+?):(\\d+)(?:-(\\d+)|:(\\d+))?$")

// parseLocation splits target into its path and line range.
// path:N gives start = end = N, path:N-M gives start N and end M,
// and path:N:C gives start = end = N with the column checked for digits but never converted.
// ok is false when target has no such spelling or a line number overflows an int.
func parseLocation(target string) (path string, start, end int, ok bool) {
	m := locationPattern.FindStringSubmatch(target)
	if m == nil {
		return "", 0, 0, false
	}
	start, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, 0, false
	}
	end = start
	if m[3] != "" {
		end, err = strconv.Atoi(m[3])
		if err != nil {
			return "", 0, 0, false
		}
	}
	return m[1], start, end, true
}

// locationFor turns one target spelling into an engine.Location, applying the four checks that need
// no file access in order: bad_location, bad_range, outside_root and the path half of unaddressable.
// A target that fails one carries the rejection and whatever was determined before it.
func (r *Repo) locationFor(target string) engine.Location {
	path, start, end, ok := parseLocation(target)
	if !ok {
		return engine.Location{
			Target: target,
			Reason: EncloseReasonBadLocation,
			Error:  "not a location: want path:line, path:line-line or path:line:col: " + target,
		}
	}
	if start < 1 || start > end {
		return engine.Location{
			Target: target,
			Start:  start,
			End:    end,
			Reason: EncloseReasonBadRange,
			Error:  fmt.Sprintf("bad range: start %d must be at least 1 and not after end %d", start, end),
		}
	}

	rel, err := repopath.RepoRelTarget(r.root, r.root, path)
	if errors.Is(err, ErrTargetHasSeparator) {
		return engine.Location{
			Target: target,
			Start:  start,
			End:    end,
			Reason: EncloseReasonUnaddressable,
			Error:  "path cannot be spelled as a glyph unit: " + path,
		}
	}
	if err != nil {
		return engine.Location{
			Target: target,
			Start:  start,
			End:    end,
			Reason: EncloseReasonOutsideRoot,
			Error:  "path outside repository: " + path,
		}
	}
	if _, err := glyph.Self(glyph.Go, rel); err != nil {
		return engine.Location{
			Target: target,
			File:   rel,
			Start:  start,
			End:    end,
			Reason: EncloseReasonUnaddressable,
			Error:  "path cannot be spelled as a glyph unit: " + rel,
		}
	}
	return engine.Location{Target: target, File: rel, Start: start, End: end}
}

// Enclose maps each target, spelled path:line, path:line-line or path:line:col, to the innermost
// members of the working tree that enclose it, and returns exactly one answer per target in input order.
// A relative path is repository-root-relative, never cwd-relative; an absolute path inside the root is accepted.
// A malformed or unanswerable target is that item's rejection and never fails the call.
//
// Enclose mints the glyphs a table-of-contents query mints for the file,
// and Resolve does not find three of those cases, so a found answer there will not Resolve:
// an explicitly named gitignored file, a file with no package clause,
// and a path through a symlinked directory.
func (r *Repo) Enclose(targets []string) ([]EncloseResult, error) {
	return r.EncloseAt("", targets)
}

// EncloseAt answers each target exactly as Enclose does, but against the files committed at rev
// when rev is non-empty; an empty rev answers against the working tree with no git involved.
// All targets of one call are answered against the same single revision.
//
// A revision that does not exist, a root that is not a git repository or not its top level,
// and a failure reading a file from git fail the whole call, with the git layer's error returned
// unchanged so ErrUnknownRevision, ErrNotARepository and ErrRootNotTopLevel stay matchable.
//
// A revision answer differs from a working-tree answer in three rare ways:
// no ignore set applies at a revision, so a tracked and gitignored Go file votes on its directory's package
// and can change the unit minted there;
// a symlink committed under a Go name yields its link text, which answers as any other bytes
// where the working tree says unreadable;
// and a directory named like a Go file is read as its tree listing where the working tree says missing_file,
// while a submodule named like one fails the whole call.
func (r *Repo) EncloseAt(rev string, targets []string) ([]EncloseResult, error) {
	locs := make([]engine.Location, len(targets))
	for i, target := range targets {
		locs[i] = r.locationFor(target)
	}
	if rev == "" {
		return r.engine.Enclose(locs)
	}

	gr, err := gitsrc.Open(r.root)
	if err != nil {
		return nil, err
	}
	if err := gr.VerifyRevision(rev); err != nil {
		return nil, err
	}
	return r.engine.EncloseFrom(gitRevisionFiles{gr: gr, rev: rev}, locs)
}

// gitRevisionFiles adapts the git layer to engine.RevisionFiles for one revision.
type gitRevisionFiles struct {
	gr  *gitsrc.Repo
	rev string
}

// GoFiles returns the Go files directly under dirRel at the revision.
func (g gitRevisionFiles) GoFiles(dirRel string) ([]string, error) {
	return g.gr.DirFilesAtRevision(g.rev, dirRel)
}

// Read returns the file's bytes at the revision.
func (g gitRevisionFiles) Read(rel string) ([]byte, error) {
	return g.gr.ReadBlob(g.rev, rel)
}
