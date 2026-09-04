// ignore.go implements the subset of .gitignore matching the walk needs: pattern parsing, ordered
// last-match-wins evaluation, and the extend/trim frame discipline that lets the walk push a
// directory's patterns on descent and pop them on the way back out.
//
// Directory pruning is part of this rule, not the caller's: a directory that match reports excluded
// is never descended into, so nothing beneath it can be re-included by a later pattern unless the
// directory itself is re-included first — this is git's own rule, and the one this repository's own
// "/quarry" plus "!/quarry/" pair turns on.
//
// Deliberately NOT supported: core.excludesFile, .git/info/exclude, and any .gitattributes
// interaction. Only a repository-tracked .gitignore file at each directory is read.

package engine

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ignorePattern is the parsed form of one .gitignore line.
type ignorePattern struct {
	// glob is the pattern with its "!" negation and anchoring "/" already stripped, and its
	// directory-only trailing "/" already stripped.
	glob string
	// negate reports whether this pattern re-includes a path a previous pattern excluded.
	negate bool
	// dirOnly reports whether this pattern matches directories only.
	dirOnly bool
	// anchored reports whether glob must match starting at dirRel rather than at any depth below it.
	anchored bool
	// dirRel is the repository-relative directory of the .gitignore file this pattern came from,
	// using forward slashes, or "" for the repository root.
	dirRel string
}

// ignoreSet is an ordered collection of gitignore patterns collected while a walk descends a
// repository. Patterns are appended per directory via extend and removed per directory via trim, so
// the set always reflects exactly the .gitignore files on the path from the repository root to the
// walk's current directory — nothing is cached across calls to TOC or SpansOf.
type ignoreSet struct {
	root     string
	patterns []ignorePattern
}

// newIgnoreSet returns an empty pattern set for a repository rooted at root.
func newIgnoreSet(root string) *ignoreSet {
	return &ignoreSet{root: root}
}

// extend reads the .gitignore file in the repository-relative directory dirRel, when one exists,
// and appends its patterns to the set. It returns how many patterns were appended so the caller can
// later pass that count to trim; a missing .gitignore file is not an error and returns (0, nil).
func (s *ignoreSet) extend(dirRel string) (n int, err error) {
	dirRel = path.Clean(filepath.ToSlash(dirRel))
	if dirRel == "." {
		dirRel = ""
	}

	gitignorePath := filepath.Join(s.root, filepath.FromSlash(dirRel), ".gitignore")
	f, err := os.Open(gitignorePath)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if p, ok := parseIgnoreLine(line, dirRel); ok {
			s.patterns = append(s.patterns, p)
			n++
		}
	}
	if err := scanner.Err(); err != nil {
		return n, err
	}
	return n, nil
}

// trim drops the last n appended patterns, so the walk can leave a directory and lose that
// directory's own patterns again. n is always the value the matching extend returned: the count is
// the caller's to hold, not the set's, so the set keeps no frame stack and a caller that extends
// twice before trimming is expressing exactly what it means.
func (s *ignoreSet) trim(n int) {
	s.patterns = s.patterns[:len(s.patterns)-n]
}

// match reports whether pathRel, a repository-relative, forward-slash path, is excluded. .git is
// always excluded as a directory, unconditionally and before any pattern is consulted. Otherwise the
// set is evaluated in order and the last matching pattern's polarity wins.
func (s *ignoreSet) match(pathRel string, isDir bool) bool {
	pathRel = path.Clean(filepath.ToSlash(pathRel))
	if pathRel == "." || pathRel == "" {
		return false
	}
	if isDir && pathRel == ".git" {
		return true
	}

	excluded := false
	for _, p := range s.patterns {
		if p.dirOnly && !isDir {
			continue
		}
		if matchIgnorePattern(p, pathRel) {
			excluded = !p.negate
		}
	}
	return excluded
}

// parseIgnoreLine parses one line of a .gitignore file found in repository-relative directory
// dirRel. It reports (pattern, false) for a blank line, a comment line (first non-whitespace
// character "#"), or (after leading-whitespace trim) an empty line.
func parseIgnoreLine(line, dirRel string) (ignorePattern, bool) {
	// gitignore does not trim trailing whitespace unless it is escaped, but this engine only needs
	// to read quarry's own patterns and the common forms its tests exercise, so trailing whitespace
	// is trimmed unconditionally rather than honoring backslash escapes.
	line = strings.TrimRight(line, " \t\r")
	if line == "" || strings.HasPrefix(line, "#") {
		return ignorePattern{}, false
	}

	p := ignorePattern{dirRel: dirRel}

	if strings.HasPrefix(line, "!") {
		p.negate = true
		line = line[1:]
	}

	if strings.HasSuffix(line, "/") {
		p.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	if strings.HasPrefix(line, "/") {
		p.anchored = true
		line = strings.TrimPrefix(line, "/")
	} else if strings.Contains(line, "/") {
		// A pattern with an interior slash (anywhere other than the trailing position already
		// stripped above) is anchored to its own .gitignore's directory, exactly as a leading "/"
		// pattern is.
		p.anchored = true
	}

	p.glob = line
	return p, true
}

// matchIgnorePattern reports whether p matches the repository-relative path pathRel.
func matchIgnorePattern(p ignorePattern, pathRel string) bool {
	// relToGitignoreDir is pathRel expressed relative to the directory the pattern's .gitignore
	// lives in; a pattern never matches a path outside that subtree.
	relToGitignoreDir := pathRel
	if p.dirRel != "" {
		prefix := p.dirRel + "/"
		if pathRel == p.dirRel {
			return false
		}
		if !strings.HasPrefix(pathRel, prefix) {
			return false
		}
		relToGitignoreDir = strings.TrimPrefix(pathRel, prefix)
	}

	if p.anchored {
		return globMatch(p.glob, relToGitignoreDir)
	}

	// An unanchored pattern matches at any depth below the .gitignore's directory: try the glob
	// against the full relative path and against every suffix starting at a segment boundary.
	segments := strings.Split(relToGitignoreDir, "/")
	for i := range segments {
		if globMatch(p.glob, strings.Join(segments[i:], "/")) {
			return true
		}
	}
	return false
}

// globMatch reports whether glob matches candidate, where "*" and "?" match within one path
// segment, and "**" matches across segments.
func globMatch(glob, candidate string) bool {
	if !strings.Contains(glob, "/") {
		// A slash-free glob only ever matches a single path segment: path.Match itself already
		// refuses to let "*" or "?" consume a "/", so a multi-segment candidate correctly fails here
		// without any extra segment-counting. The caller supplies the right single-segment candidate
		// for the unanchored, matches-at-any-depth case by enumerating suffixes before calling in.
		return segmentGlobMatch(glob, candidate)
	}
	return doubleStarMatch(strings.Split(glob, "/"), strings.Split(candidate, "/"))
}

// doubleStarMatch matches a slash-split glob (which may contain "**" segments) against a
// slash-split candidate path.
func doubleStarMatch(globSegs, candSegs []string) bool {
	if len(globSegs) == 0 {
		return len(candSegs) == 0
	}
	if globSegs[0] == "**" {
		if len(globSegs) == 1 {
			return true
		}
		for i := 0; i <= len(candSegs); i++ {
			if doubleStarMatch(globSegs[1:], candSegs[i:]) {
				return true
			}
		}
		return false
	}
	if len(candSegs) == 0 {
		return false
	}
	if !segmentGlobMatch(globSegs[0], candSegs[0]) {
		return false
	}
	return doubleStarMatch(globSegs[1:], candSegs[1:])
}

// segmentGlobMatch reports whether pattern matches name within one path segment, where "*" matches
// any run of non-"/" characters and "?" matches exactly one non-"/" character. Since neither pattern
// nor name can contain "/" at this point, path.Match's segment semantics apply directly.
func segmentGlobMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}
