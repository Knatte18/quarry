// scratchtree_test.go provides writeScratchTree, the shared helper for tests that need a fixture
// tree on disk rather than in-memory bytes — chiefly the gitignore tests, whose subject matter is
// files a real .gitignore excludes and which therefore cannot live under testdata/ (see ignore.go's
// package doc for why).

package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeScratchTree creates a fresh directory tree under .scratch/engine-tests/<name>/, writes each
// entry in files (key = a forward-slash relative path, value = its contents; parent directories are
// created as needed), registers a t.Cleanup that removes the tree, and returns its absolute path.
//
// The helper writes regular files only. A test needing a symlink, a directory-only entry, an
// unreadable file, or a specific creation order creates it itself on the path this helper returns —
// that is deliberate, so this one helper does not grow a mode/type/order parameter for the handful
// of cases that need one.
//
// It never calls t.TempDir(): the system temp directory is banned by this task's constraints, and
// .scratch/ — gitignored at the repository root — is the sanctioned location instead.
func writeScratchTree(t *testing.T, name string, files map[string]string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("writeScratchTree: runtime.Caller(0) failed to resolve this file's path")
	}
	// thisFile is .../internal/engine/scratchtree_test.go; the module root is two directories up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	root := filepath.Join(moduleRoot, ".scratch", "engine-tests", name)
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
