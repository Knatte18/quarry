// target_test.go pins repoRelTarget's conversion, rebasing, and escape-rejection rules. Only the
// symlink case needs a fixture — built with writeScratchTree — everything else is pure string
// inputs. Never calls t.TempDir().

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

func TestRepoRelTarget_CwdRelativeAndAbsoluteAgree(t *testing.T) {
	root := "/repo"
	base := "/repo/internal/logger"

	relGot, err := repoRelTarget(root, base, "file.go")
	if err != nil {
		t.Fatalf("repoRelTarget(cwd-relative) = _, %v; want nil error", err)
	}

	absGot, err := repoRelTarget(root, base, "/repo/internal/logger/file.go")
	if err != nil {
		t.Fatalf("repoRelTarget(absolute) = _, %v; want nil error", err)
	}

	if relGot != absGot {
		t.Errorf("repoRelTarget: relative form = %q, absolute form = %q; want equal", relGot, absGot)
	}
	want := "internal/logger/file.go"
	if relGot != want {
		t.Errorf("repoRelTarget(cwd-relative) = %q; want %q", relGot, want)
	}
}

func TestRepoRelTarget_RootItself(t *testing.T) {
	got, err := repoRelTarget("/repo", "/repo", ".")
	if err != nil {
		t.Fatalf("repoRelTarget(root itself) = _, %v; want nil error", err)
	}
	if got != "." {
		t.Errorf("repoRelTarget(root itself) = %q; want %q", got, ".")
	}
}

func TestRepoRelTarget_NestedNoLeadingDotSlash(t *testing.T) {
	got, err := repoRelTarget("/repo", "/repo", "a/b/c.go")
	if err != nil {
		t.Fatalf("repoRelTarget(nested) = _, %v; want nil error", err)
	}
	if got != "a/b/c.go" {
		t.Errorf("repoRelTarget(nested) = %q; want %q", got, "a/b/c.go")
	}
}

// TestRepoRelTarget_Rebasing pins the rebasing rule the discussion's round-2 review caught, and
// which golden-tests-run-the-cli-in-process depends on: with base set to the root (the --root
// path), a relative target resolves under the root; with base set to a subdirectory (the
// no-root cwd path), the same relative target resolves under that subdirectory.
func TestRepoRelTarget_Rebasing(t *testing.T) {
	root := "/repo"

	gotAtRoot, err := repoRelTarget(root, root, "sub/file.go")
	if err != nil {
		t.Fatalf("repoRelTarget(base=root) = _, %v; want nil error", err)
	}
	if want := "sub/file.go"; gotAtRoot != want {
		t.Errorf("repoRelTarget(base=root) = %q; want %q", gotAtRoot, want)
	}

	gotAtSubdir, err := repoRelTarget(root, "/repo/sub", "file.go")
	if err != nil {
		t.Fatalf("repoRelTarget(base=subdir) = _, %v; want nil error", err)
	}
	if want := "sub/file.go"; gotAtSubdir != want {
		t.Errorf("repoRelTarget(base=subdir) = %q; want %q", gotAtSubdir, want)
	}
}

func TestRepoRelTarget_EscapesRoot(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		base   string
		target string
	}{
		{"dot-dot", "/repo", "/repo", "../outside"},
		{"absolute-outside", "/repo", "/repo", "/elsewhere/file.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repoRelTarget(tt.root, tt.base, tt.target)
			if !errors.Is(err, quarry.ErrTargetOutsideRepo) {
				t.Errorf("repoRelTarget(%q) error = %v; want errors.Is(_, quarry.ErrTargetOutsideRepo)", tt.target, err)
			}
		})
	}
}

func TestRepoRelTarget_NativeSeparatorsToForwardSlash(t *testing.T) {
	root := filepath.Join(string(filepath.Separator) + "repo")
	base := root
	target := filepath.Join("a", "b", "c.go")

	got, err := repoRelTarget(root, base, target)
	if err != nil {
		t.Fatalf("repoRelTarget(native separators) = _, %v; want nil error", err)
	}
	if want := "a/b/c.go"; got != want {
		t.Errorf("repoRelTarget(native separators) = %q; want %q", got, want)
	}
}

func TestRepoRelTarget_SymlinkNotResolvedThrough(t *testing.T) {
	root := writeScratchTree(t, "target-symlink", map[string]string{
		"realdir/file.go": "package realdir\n",
	})

	link := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "realdir"), link); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	got, err := repoRelTarget(root, root, "link")
	if err != nil {
		t.Fatalf("repoRelTarget(symlink) = _, %v; want nil error", err)
	}
	if got != "link" {
		t.Errorf("repoRelTarget(symlink) = %q; want %q (the symlink's own path, not the directory it points at)", got, "link")
	}
}
