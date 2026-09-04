// resolve.go implements the internal span lookup: the inverse of the walk, from a glyph back to
// the declarations it names. Repo.unitDirs maps a glyph unit back to the directory or directories
// that hold it; Repo.symbolsOfUnit and Repo.SpansOf, this file's later additions, are built on top
// of it. This file deliberately stops short of a status vocabulary: zero matches is an empty slice,
// not a "not_found" value, and the one fact the lookup cannot express without inventing one — a
// unit that names two different directories — is carried on unitDirs' own collision return instead,
// for the later resolve verb to promote into a status when it defines one.

package engine

import (
	"os"
	"strings"
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
