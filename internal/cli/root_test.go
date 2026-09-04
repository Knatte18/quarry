// root_test.go pins discoverRoot and resolveRoot, building every fixture with writeScratchTree
// and never calling t.TempDir(), t.Chdir, or os.Chdir.

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverRoot_GitDirectory(t *testing.T) {
	root := writeScratchTree(t, "discover-git-dir", map[string]string{
		"a/b/c/keep.txt": "x",
	})
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	start := filepath.Join(root, "a", "b", "c")
	got, err := discoverRoot(start)
	if err != nil {
		t.Fatalf("discoverRoot(%q) = _, %v; want nil error", start, err)
	}
	if got != root {
		t.Errorf("discoverRoot(%q) = %q; want %q", start, got, root)
	}
}

func TestDiscoverRoot_GitFile(t *testing.T) {
	root := writeScratchTree(t, "discover-git-file", map[string]string{
		"a/b/c/keep.txt": "x",
		".git":           "gitdir: /somewhere/else\n",
	})

	start := filepath.Join(root, "a", "b", "c")
	got, err := discoverRoot(start)
	if err != nil {
		t.Fatalf("discoverRoot(%q) = _, %v; want nil error", start, err)
	}
	if got != root {
		t.Errorf("discoverRoot(%q) = %q; want %q", start, got, root)
	}
}

func TestDiscoverRoot_NearestOfNested(t *testing.T) {
	root := writeScratchTree(t, "discover-nested", map[string]string{
		"a/b/keep.txt": "x",
	})
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir outer .git: %v", err)
	}
	nested := filepath.Join(root, "a")
	if err := os.Mkdir(filepath.Join(nested, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir nested .git: %v", err)
	}

	start := filepath.Join(root, "a", "b")
	got, err := discoverRoot(start)
	if err != nil {
		t.Fatalf("discoverRoot(%q) = _, %v; want nil error", start, err)
	}
	if got != nested {
		t.Errorf("discoverRoot(%q) = %q; want nearest root %q", start, got, nested)
	}
}

func TestDiscoverRoot_NoneFound(t *testing.T) {
	// A fixture under .scratch/ is inside this repository, so a walk from it always finds this
	// repository's own .git and the no-root-found branch is unreachable from there. Testing it
	// instead by calling discoverRoot on the filesystem root exercises the same terminating branch
	// without needing an out-of-repository directory — which would otherwise require t.TempDir(),
	// banned for this repository's tests.
	fsRoot := "/"
	if runtime.GOOS == "windows" {
		fsRoot = filepath.VolumeName(mustGetwd(t)) + string(filepath.Separator)
	}

	if _, err := os.Lstat(filepath.Join(fsRoot, ".git")); err == nil {
		t.Skip("a .git entry exists at the filesystem root on this machine; not applicable here")
	}

	_, err := discoverRoot(fsRoot)
	if err == nil {
		t.Fatalf("discoverRoot(%q) = _, nil; want error", fsRoot)
	}
	ue, ok := err.(usageError)
	if !ok {
		t.Fatalf("discoverRoot(%q) error type = %T; want usageError", fsRoot, err)
	}
	if !strings.Contains(string(ue), "pass --root") {
		t.Errorf("discoverRoot(%q) error = %q; want it to contain %q", fsRoot, string(ue), "pass --root")
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return wd
}

func TestResolveRoot_FlagRootShortCircuitsDiscovery(t *testing.T) {
	root := writeScratchTree(t, "resolve-short-circuit", map[string]string{
		"keep.txt": "x",
	})
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	other := writeScratchTree(t, "resolve-short-circuit-other", map[string]string{
		"keep.txt": "x",
	})

	got, err := resolveRoot(other, root)
	if err != nil {
		t.Fatalf("resolveRoot(%q, %q) = _, %v; want nil error", other, root, err)
	}
	if got != filepath.Clean(other) {
		t.Errorf("resolveRoot(%q, %q) = %q; want %q", other, root, got, other)
	}
}

func TestResolveRoot_RelativeFlagRoot(t *testing.T) {
	root := writeScratchTree(t, "resolve-relative", map[string]string{
		"sub/keep.txt": "x",
	})

	got, err := resolveRoot("sub", root)
	if err != nil {
		t.Fatalf("resolveRoot(%q, %q) = _, %v; want nil error", "sub", root, err)
	}
	want := filepath.Join(root, "sub")
	if got != want {
		t.Errorf("resolveRoot(%q, %q) = %q; want %q", "sub", root, got, want)
	}
}

func TestResolveRoot_FlagRootErrors(t *testing.T) {
	root := writeScratchTree(t, "resolve-errors", map[string]string{
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
			_, err := resolveRoot(tt.flagRoot, root)
			if err == nil {
				t.Fatalf("resolveRoot(%q, _) = _, nil; want error", tt.flagRoot)
			}
			ue, ok := err.(usageError)
			if !ok {
				t.Fatalf("resolveRoot(%q, _) error type = %T; want usageError", tt.flagRoot, err)
			}
			if !strings.Contains(string(ue), tt.flagRoot) {
				t.Errorf("resolveRoot(%q, _) error = %q; want it to echo the given path", tt.flagRoot, string(ue))
			}
		})
	}
}
