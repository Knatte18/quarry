// root.go declares DiscoverRoot, the upward .git walk, and ResolveRoot, the --root-or-discovery
// entry point every request pipeline resolves its repository root through.

package repopath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoRepositoryRoot is DiscoverRoot's failure sentinel: the upward walk reached the filesystem
// root without finding a .git entry. Its own text is namespaced to this package and is never
// user-visible through a caller; each caller formats its own user-facing sentence from the
// sentinel and the path DiscoverRoot was called with.
var ErrNoRepositoryRoot = errors.New("repopath: no repository root found; pass --root")

// ErrRootNotDirectory is ResolveRoot's failure sentinel when flagRoot does not resolve to an
// existing directory. Its own text is namespaced to this package and is never user-visible
// through a caller; each caller formats its own user-facing sentence from the sentinel and the
// flagRoot value it was given.
var ErrRootNotDirectory = errors.New("repopath: root is not a directory")

// DiscoverRoot walks upward from startDir looking for a .git entry, file or directory alike —
// the file case is the git-worktree spelling, and rejecting it would make quarry unusable inside
// every mill worktree. It returns the first directory containing one, or ErrNoRepositoryRoot
// wrapped with startDir when the walk reaches the filesystem root without finding one.
//
// DiscoverRoot takes its starting directory as a parameter rather than calling os.Getwd itself,
// so its tests need no process-global state; each caller's own entry point is the one place that
// reads the working directory.
func DiscoverRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w: %s", ErrNoRepositoryRoot, startDir)
		}
		dir = parent
	}
}

// ResolveRoot resolves the repository root for one request: discovery from cwd when flagRoot is
// empty, or flagRoot itself — made absolute against cwd when relative, and verified to be an
// existing directory — when it is not. --root skips the walk entirely; it never falls back to
// discovery.
//
// ResolveRoot takes cwd as a parameter rather than calling os.Getwd itself, so its tests need no
// process-global state; each caller's own entry point is the one place that reads the working
// directory.
func ResolveRoot(flagRoot, cwd string) (string, error) {
	if flagRoot == "" {
		return DiscoverRoot(cwd)
	}

	abs := flagRoot
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(cwd, abs)
	}
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		// Name the path as given, not the resolved one, so the message echoes what the caller
		// typed.
		return "", fmt.Errorf("%w: %s", ErrRootNotDirectory, flagRoot)
	}
	return abs, nil
}
