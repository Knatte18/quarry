// target.go declares RepoRelTarget, a caller's own path arithmetic that converts a
// caller-supplied target into a clean, forward-slash, repository-relative path before the engine
// ever sees it.

package repopath

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/Knatte18/quarry/quarry"
)

// RepoRelTarget converts target, as given by the caller, into a clean, forward-slash,
// repository-relative path. root is the absolute repository root from ResolveRoot; base is the
// directory a relative target is interpreted against — the root itself when --root was given, and
// the caller's own working directory otherwise.
//
// RepoRelTarget touches no filesystem at all: it is pure path arithmetic, so a target that
// escapes the root is rejected before any stat happens, which is step 3 of
// target-kind-and-the-cli-stat's fixed order. It uses filepath.Abs-style joining and filepath.Rel
// only, never filepath.EvalSymlinks: the engine's resolveTarget uses os.Lstat and never os.Stat,
// so a symlink named directly as the target is answered as a file rather than followed, and
// resolving symlinks here would defeat that rule before the engine ever saw the path.
//
// Native separators are accepted on input; the returned path is always forward-slash and
// repository-root relative, which is the form the engine takes and emits.
func RepoRelTarget(root, base, target string) (string, error) {
	var abs string
	if filepath.IsAbs(target) {
		abs = filepath.Clean(target)
	} else {
		abs = filepath.Join(base, target)
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", quarry.ErrTargetOutsideRepo
	}

	relSlash := filepath.ToSlash(rel)
	if relSlash == ".." || strings.HasPrefix(relSlash, "../") {
		return "", quarry.ErrTargetOutsideRepo
	}

	// filepath.Rel returns "." when the target is the root itself, which the engine accepts as the
	// root.
	return path.Clean(relSlash), nil
}
