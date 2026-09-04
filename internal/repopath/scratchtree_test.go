// scratchtree_test.go provides writeScratchTree, the shared helper for tests that need a fixture
// tree on disk rather than pure strings — chiefly root_test.go's .git-entry cases and
// target_test.go's symlink case. This is a deliberate per-package copy of
// internal/cli/scratchtree_test.go's helper of the same name, because Go test helpers are not
// importable across packages; the scratch subdirectory is namespaced to this package
// (.scratch/repopath-tests/) so it does not share a parent directory with a package that tests in
// parallel.

package repopath

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeScratchTree creates a fresh directory tree under .scratch/repopath-tests/<name>/, writes
// each entry in files (key = a forward-slash relative path, value = its contents; parent
// directories are created as needed), registers a t.Cleanup that removes the tree, and returns
// its absolute path.
//
// The helper writes regular files only. A test needing a symlink, an empty directory, or a .git
// entry creates it itself on the path this helper returns.
//
// It never calls t.TempDir(): the system temp directory is banned for this repository's tests,
// and .scratch/ is the sanctioned location instead. This is a deliberate per-package copy of
// internal/engine/scratchtree_test.go's helper of the same name, because Go test helpers are not
// importable across packages. internal/repopath/ sits two directories below the module root — the
// same depth as internal/cli/ — so the runtime.Caller(0) walk here takes the same three
// filepath.Dir steps that file uses; quarry/'s own copy sits one directory down and needs only
// two.
func writeScratchTree(t *testing.T, name string, files map[string]string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("writeScratchTree: runtime.Caller(0) failed to resolve this file's path")
	}
	// thisFile is .../internal/repopath/scratchtree_test.go; the module root is three directories
	// up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	root := filepath.Join(moduleRoot, ".scratch", "repopath-tests", name)
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("writeScratchTree(%q): clearing stale tree: %v", name, err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("writeScratchTree(%q): cleanup: %v", name, err)
		}
	})

	for relPath, contents := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("writeScratchTree(%q): mkdir for %q: %v", name, relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
			t.Fatalf("writeScratchTree(%q): write %q: %v", name, relPath, err)
		}
	}

	return root
}
