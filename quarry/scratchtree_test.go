// scratchtree_test.go provides writeScratchTree, this package's own copy of
// internal/engine/scratchtree_test.go's helper of the same name. Go test helpers are not importable
// across packages, so each package that needs an on-disk fixture tree declares its own.

package quarry

import (
	"os"
	"path/filepath"
	"testing"
)

// packageDir returns the directory of the package under test, which go test sets as the working
// directory. Unlike runtime.Caller, it stays absolute when the test binary is built with -trimpath.
func packageDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("packageDir: %v", err)
	}
	return dir
}

// writeScratchTree creates a fresh directory tree under .scratch/quarry-tests/<name>/, writes each
// entry in files (key = a forward-slash relative path, value = its contents; parent directories are
// created as needed), registers a t.Cleanup that removes the tree, and returns its absolute path.
//
// The helper writes regular files only. A test needing a symlink, a directory-only entry, an
// unreadable file, or a specific creation order creates it itself on the path this helper returns.
//
// It never calls t.TempDir(): the system temp directory is banned for this repository's tests, and
// .scratch/ — gitignored at the repository root — is the sanctioned location instead. This is a
// deliberate per-package copy of internal/engine/scratchtree_test.go's helper, because Go test
// helpers are not importable across packages.
func writeScratchTree(t *testing.T, name string, files map[string]string) string {
	t.Helper()

	// quarry/ sits one directory below the module root.
	moduleRoot := filepath.Dir(packageDir(t))

	root := filepath.Join(moduleRoot, ".scratch", "quarry-tests", name)
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
