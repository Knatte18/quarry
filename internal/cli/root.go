// root.go declares discoverRoot, the upward .git walk, and resolveRoot, the --root-or-discovery
// entry point every request pipeline resolves its repository root through.

package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// discoverRoot walks upward from startDir looking for a .git entry, file or directory alike —
// the file case is the git-worktree spelling, and rejecting it would make quarry unusable inside
// every mill worktree. It returns the first directory containing one, or a usageError naming
// --root as the fix when the walk reaches the filesystem root without finding one.
//
// discoverRoot takes its starting directory as a parameter rather than calling os.Getwd itself,
// so its tests need no process-global state; Run (batch 4) is the one place that reads the
// working directory.
func discoverRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", usageError(fmt.Sprintf("no repository root found above %s; pass --root", startDir))
		}
		dir = parent
	}
}

// resolveRoot resolves the repository root for one request: discovery from cwd when flagRoot is
// empty, or flagRoot itself — made absolute against cwd when relative, and verified to be an
// existing directory — when it is not. --root skips the walk entirely; it never falls back to
// discovery.
//
// resolveRoot takes cwd as a parameter rather than calling os.Getwd itself, so its tests need no
// process-global state; Run (batch 4) is the one place that reads the working directory.
func resolveRoot(flagRoot, cwd string) (string, error) {
	if flagRoot == "" {
		return discoverRoot(cwd)
	}

	abs := flagRoot
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(cwd, abs)
	}
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		// Name the path as given, not the resolved one, so the message echoes what the caller typed.
		return "", usageError(fmt.Sprintf("--root is not a directory: %s", flagRoot))
	}
	return abs, nil
}
