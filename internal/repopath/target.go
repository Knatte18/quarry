// target.go declares repoRelPath and repoRelTarget, the pure path arithmetic behind RepoRelPath
// and RepoRelTarget, the exported functions a caller uses to convert a caller-supplied target into
// a clean, forward-slash, repository-relative path before the engine ever sees it.

package repopath

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/Knatte18/quarry/quarry"
)

// repoRelPath converts target, as given by the caller, into a clean, forward-slash,
// repository-relative path. root is the absolute repository root from ResolveRoot; base is the
// directory a relative target is interpreted against — the root itself when --root was given, and
// the caller's own working directory otherwise.
//
// repoRelPath touches no filesystem at all: it is pure path arithmetic. It uses filepath.Abs-style
// joining and filepath.Rel only, never filepath.EvalSymlinks: the engine's resolveTarget uses
// os.Lstat and never os.Stat, so a symlink named directly as the target is answered as a file
// rather than followed, and resolving symlinks here would defeat that rule before the engine ever
// saw the path.
//
// Native separators are accepted on input; the returned path is always forward-slash and
// repository-root relative, which is the form the engine takes and emits. repoRelPath returns the
// cleaned relative form even when it begins with "..": it does not reject a target that leaves the
// root. A caller that wants a target that escapes the root to reach the engine — so the engine's
// own outside-repository rule produces the answer, rather than this package synthesising a second
// copy of it — calls RepoRelPath. It returns an error only when filepath.Rel itself fails, and
// that error is quarry.ErrTargetOutsideRepo.
func repoRelPath(root, base, target string) (string, error) {
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

	// filepath.Rel returns "." when the target is the root itself, which the engine accepts as
	// the root.
	return path.Clean(relSlash), nil
}

// RepoRelPath exports repoRelPath for callers outside this package that want the arithmetic-only
// conversion, without RepoRelTarget's escape rejection — the CLI's resolve verb calls this so a
// target that escapes the root is passed through to the engine, letting the engine's own
// outside-repository rule produce the answer, rather than rejecting the target here.
func RepoRelPath(root, base, target string) (string, error) {
	return repoRelPath(root, base, target)
}

// repoRelTarget converts target the same way repoRelPath does, then rejects a result that leaves
// the repository root. This is the path arithmetic a caller wants when a target that escapes the
// root must be refused here rather than reaching the engine.
func repoRelTarget(root, base, target string) (string, error) {
	rel, err := repoRelPath(root, base, target)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", quarry.ErrTargetOutsideRepo
	}

	return rel, nil
}

// RepoRelTarget exports repoRelTarget for callers outside this package: the CLI's toc verb, and
// the MCP server's own target conversion, both of which must reject an escaping target here
// rather than letting it reach the engine.
func RepoRelTarget(root, base, target string) (string, error) {
	return repoRelTarget(root, base, target)
}
