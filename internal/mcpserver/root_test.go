// root_test.go table-tests ResolveRoot and rootErrorMessage, building every fixture with a
// per-package copy of writeScratchTree and never calling t.TempDir(), t.Chdir, or os.Chdir.

package mcpserver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Knatte18/quarry/internal/repopath"
)

// writeScratchTree creates a fresh directory tree under .scratch/mcpserver-tests/<name>/, writes
// each entry in files (key = a forward-slash relative path, value = its contents; parent
// directories are created as needed), registers a t.Cleanup that removes the tree, and returns its
// absolute path.
//
// The helper writes regular files only. A test needing a symlink, an empty directory, or a .git
// entry creates it itself on the path this helper returns.
//
// It never calls t.TempDir(): the system temp directory is banned for this repository's tests, and
// .scratch/ is the sanctioned location instead. This is a deliberate per-package copy of
// internal/repopath/scratchtree_test.go's helper of the same name, because Go test helpers are not
// importable across packages — this package needs a fixture tree in exactly one test file, so a
// standalone helper file would be a copy with a single caller.
func writeScratchTree(t *testing.T, name string, files map[string]string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("writeScratchTree: runtime.Caller(0) failed to resolve this file's path")
	}
	// thisFile is .../internal/mcpserver/root_test.go; the module root is three directories up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	root := filepath.Join(moduleRoot, ".scratch", "mcpserver-tests", name)
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

func TestResolveRoot_FlagAbsentDiscoversFromCwd(t *testing.T) {
	root := writeScratchTree(t, "resolve-discover", map[string]string{
		"a/b/keep.txt": "x",
	})
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	cwd := filepath.Join(root, "a", "b")
	got, err := ResolveRoot("", cwd)
	if err != nil {
		t.Fatalf("ResolveRoot(\"\", %q) = _, %v; want nil error", cwd, err)
	}
	if got != root {
		t.Errorf("ResolveRoot(\"\", %q) = %q; want %q", cwd, got, root)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ResolveRoot(\"\", %q) = %q; want an absolute path", cwd, got)
	}
}

func TestResolveRoot_RelativeFlagRootJoinedAgainstCwd(t *testing.T) {
	root := writeScratchTree(t, "resolve-relative", map[string]string{
		"sub/keep.txt": "x",
	})

	got, err := ResolveRoot("sub", root)
	if err != nil {
		t.Fatalf("ResolveRoot(%q, %q) = _, %v; want nil error", "sub", root, err)
	}
	want := filepath.Join(root, "sub")
	if got != want {
		t.Errorf("ResolveRoot(%q, %q) = %q; want %q", "sub", root, got, want)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ResolveRoot(%q, %q) = %q; want an absolute path", "sub", root, got)
	}
}

func TestResolveRoot_AbsoluteFlagRootTakenAsIs(t *testing.T) {
	root := writeScratchTree(t, "resolve-absolute", map[string]string{
		"keep.txt": "x",
	})
	other := writeScratchTree(t, "resolve-absolute-cwd", map[string]string{
		"keep.txt": "x",
	})

	got, err := ResolveRoot(root, other)
	if err != nil {
		t.Fatalf("ResolveRoot(%q, %q) = _, %v; want nil error", root, other, err)
	}
	if got != filepath.Clean(root) {
		t.Errorf("ResolveRoot(%q, %q) = %q; want %q", root, other, got, root)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ResolveRoot(%q, %q) = %q; want an absolute path", root, other, got)
	}
}

func TestResolveRoot_FlagRootNotADirectory(t *testing.T) {
	root := writeScratchTree(t, "resolve-not-a-directory", map[string]string{
		"afile.txt": "x",
	})

	tests := []struct {
		name     string
		flagRoot string
	}{
		{"names-a-file", filepath.Join(root, "afile.txt")},
		{"names-nothing", filepath.Join(root, "does-not-exist")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveRoot(tt.flagRoot, root)
			if err == nil {
				t.Fatalf("ResolveRoot(%q, _) = _, nil; want error", tt.flagRoot)
			}
			want := "quarry-mcp: --root is not a directory: " + tt.flagRoot
			if err.Error() != want {
				t.Errorf("ResolveRoot(%q, _) error = %q; want %q", tt.flagRoot, err.Error(), want)
			}
		})
	}
}

func TestRootErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		flagRoot string
		cwd      string
		want     string
		wantOK   bool
	}{
		{
			name:     "not-a-directory",
			err:      fmt.Errorf("repopath: root is not a directory: %w", repopath.ErrRootNotDirectory),
			flagRoot: "/given/path",
			cwd:      "/somewhere",
			want:     "quarry-mcp: --root is not a directory: /given/path",
			wantOK:   true,
		},
		{
			name:     "no-repository-root",
			err:      fmt.Errorf("repopath: no repository root found: %w", repopath.ErrNoRepositoryRoot),
			flagRoot: "",
			cwd:      "/somewhere/deep",
			want:     "quarry-mcp: no repository root found above /somewhere/deep; pass --root",
			wantOK:   true,
		},
		{
			name:     "unrelated-error",
			err:      errors.New("some other failure"),
			flagRoot: "/given/path",
			cwd:      "/somewhere",
			want:     "",
			wantOK:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rootErrorMessage(tt.err, tt.flagRoot, tt.cwd)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("rootErrorMessage(%v, %q, %q) = (%q, %v); want (%q, %v)",
					tt.err, tt.flagRoot, tt.cwd, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
