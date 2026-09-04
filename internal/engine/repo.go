// repo.go declares Repo, the engine package's entry point, Open, which constructs one, and the one
// target-validation path, resolveTarget, that every query runs a caller-supplied target through
// before answering it.

package engine

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Repo is an opened repository root. It holds nothing but the absolute repository root: no
// gitignore pattern set, no parsed tree, no cache of any kind. The gitignore pattern set a query
// needs is collected fresh on every call and discarded when the call returns — see ignore.go's
// package doc. A long-lived process holding a Repo across many calls would otherwise serve a stale
// file list after a .gitignore edit on disk, a result indistinguishable from a bug in the walk.
type Repo struct {
	root string
}

// Open returns a Repo rooted at root. root must be an absolute path naming an existing directory;
// anything else is an error. Open performs no git discovery — root need not be a git repository
// root, or even inside one — and no cwd resolution: a relative root is rejected rather than
// resolved against the process's working directory.
func Open(root string) (*Repo, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("engine: open %q: not an absolute path", root)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("engine: open %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("engine: open %q: not a directory", root)
	}
	return &Repo{root: root}, nil
}

// ErrTargetOutsideRepo is returned by resolveTarget when a target is absolute, or cleans to a path
// that leaves the repository root. Callers wrap it with fmt.Errorf("...: %w", ErrTargetOutsideRepo)
// so errors.Is(err, ErrTargetOutsideRepo) still succeeds after wrapping.
var ErrTargetOutsideRepo = errors.New("engine: target outside repository")

// ErrTargetNotFound is returned by resolveTarget when a target does not exist under the repository
// root. Callers wrap it the same way ErrTargetOutsideRepo is wrapped.
var ErrTargetNotFound = errors.New("engine: target not found")

// resolveTarget is the single validation path every query runs target through before answering
// it. It performs, in order:
//
//  1. A target that is absolute, or that leaves the root once cleaned (any leading ".."), returns
//     ErrTargetOutsideRepo.
//  2. A target that does not exist under the root returns ErrTargetNotFound.
//  3. Otherwise it returns the cleaned repository-relative, forward-slash path and the stat
//     result.
//
// "" and "." both mean the repository root and are valid; the root's own rel is ".".
//
// The stat is os.Lstat, never os.Stat: os.Stat follows a symlink and would silently descend into
// its target, which contradicts the never-follow rule for the one path that rule does not
// otherwise cover — a symlink named directly as the query's own target, rather than encountered
// while walking a directory.
//
// Validation deliberately does not consult the ignore set: the filter exists so a listing is not
// noise, not to make a file unaddressable. An explicitly named gitignored target is answered, not
// refused.
func (r *Repo) resolveTarget(target string) (rel string, info os.FileInfo, err error) {
	if target == "" || target == "." {
		info, err := os.Lstat(r.root)
		if err != nil {
			return "", nil, fmt.Errorf("engine: resolve target %q: %w", target, err)
		}
		return ".", info, nil
	}

	if filepath.IsAbs(target) {
		return "", nil, fmt.Errorf("engine: resolve target %q: %w", target, ErrTargetOutsideRepo)
	}

	cleaned := path.Clean(filepath.ToSlash(target))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", nil, fmt.Errorf("engine: resolve target %q: %w", target, ErrTargetOutsideRepo)
	}
	if cleaned == "." {
		info, err := os.Lstat(r.root)
		if err != nil {
			return "", nil, fmt.Errorf("engine: resolve target %q: %w", target, err)
		}
		return ".", info, nil
	}

	full := filepath.Join(r.root, filepath.FromSlash(cleaned))
	info, err = os.Lstat(full)
	if os.IsNotExist(err) {
		return "", nil, fmt.Errorf("engine: resolve target %q: %w", target, ErrTargetNotFound)
	}
	if err != nil {
		return "", nil, fmt.Errorf("engine: resolve target %q: %w", target, err)
	}
	return cleaned, info, nil
}
