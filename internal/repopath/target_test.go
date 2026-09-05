// target_test.go pins RepoRelTarget's conversion, rebasing, and escape-rejection rules. Only the
// symlink case needs a fixture — built with writeScratchTree — everything else is pure string
// inputs. Never calls t.TempDir().

package repopath

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

	relGot, err := RepoRelTarget(root, base, "file.go")
	if err != nil {
		t.Fatalf("RepoRelTarget(cwd-relative) = _, %v; want nil error", err)
	}

	absGot, err := RepoRelTarget(root, base, "/repo/internal/logger/file.go")
	if err != nil {
		t.Fatalf("RepoRelTarget(absolute) = _, %v; want nil error", err)
	}

	if relGot != absGot {
		t.Errorf("repoRelTarget: relative form = %q, absolute form = %q; want equal", relGot, absGot)
	}
	want := "internal/logger/file.go"
	if relGot != want {
		t.Errorf("RepoRelTarget(cwd-relative) = %q; want %q", relGot, want)
	}
}

func TestRepoRelTarget_RootItself(t *testing.T) {
	got, err := RepoRelTarget("/repo", "/repo", ".")
	if err != nil {
		t.Fatalf("RepoRelTarget(root itself) = _, %v; want nil error", err)
	}
	if got != "." {
		t.Errorf("RepoRelTarget(root itself) = %q; want %q", got, ".")
	}
}

func TestRepoRelTarget_NestedNoLeadingDotSlash(t *testing.T) {
	got, err := RepoRelTarget("/repo", "/repo", "a/b/c.go")
	if err != nil {
		t.Fatalf("RepoRelTarget(nested) = _, %v; want nil error", err)
	}
	if got != "a/b/c.go" {
		t.Errorf("RepoRelTarget(nested) = %q; want %q", got, "a/b/c.go")
	}
}

// TestRepoRelTarget_Rebasing pins the rebasing rule the discussion's round-2 review caught, and
// which golden-tests-run-the-cli-in-process depends on: with base set to the root (the --root
// path), a relative target resolves under the root; with base set to a subdirectory (the
// no-root cwd path), the same relative target resolves under that subdirectory.
func TestRepoRelTarget_Rebasing(t *testing.T) {
	root := "/repo"

	gotAtRoot, err := RepoRelTarget(root, root, "sub/file.go")
	if err != nil {
		t.Fatalf("RepoRelTarget(base=root) = _, %v; want nil error", err)
	}
	if want := "sub/file.go"; gotAtRoot != want {
		t.Errorf("RepoRelTarget(base=root) = %q; want %q", gotAtRoot, want)
	}

	gotAtSubdir, err := RepoRelTarget(root, "/repo/sub", "file.go")
	if err != nil {
		t.Fatalf("RepoRelTarget(base=subdir) = _, %v; want nil error", err)
	}
	if want := "sub/file.go"; gotAtSubdir != want {
		t.Errorf("RepoRelTarget(base=subdir) = %q; want %q", gotAtSubdir, want)
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
			_, err := RepoRelTarget(tt.root, tt.base, tt.target)
			if !errors.Is(err, quarry.ErrTargetOutsideRepo) {
				t.Errorf("RepoRelTarget(%q) error = %v; want errors.Is(_, quarry.ErrTargetOutsideRepo)", tt.target, err)
			}
		})
	}
}

// TestRepoRelTarget_RejectsSeparator pins the separator reject repoRelTarget adds after the
// existing escape check: a target whose cleaned relative form carries a "#" in its first segment,
// in a middle segment, or in its basename is rejected with quarry.ErrTargetHasSeparator.
func TestRepoRelTarget_RejectsSeparator(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"first-segment", "a#b/c.go"},
		{"middle-segment", "a/b#c/d.go"},
		{"basename", "a/b/c#.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RepoRelTarget("/repo", "/repo", tt.target)
			if !errors.Is(err, quarry.ErrTargetHasSeparator) {
				t.Errorf("RepoRelTarget(%q) error = %v; want errors.Is(_, quarry.ErrTargetHasSeparator)", tt.target, err)
			}
		})
	}
}

// TestRepoRelTarget_EscapeWinsOverSeparator pins the order between the two rejections: a target
// that both escapes the root and carries a "#" still reports the escape, since the escape check
// runs first.
func TestRepoRelTarget_EscapeWinsOverSeparator(t *testing.T) {
	_, err := RepoRelTarget("/repo", "/repo", "../out#side")
	if !errors.Is(err, quarry.ErrTargetOutsideRepo) {
		t.Errorf("RepoRelTarget(escaping and separator) error = %v; want errors.Is(_, quarry.ErrTargetOutsideRepo)", err)
	}
	if errors.Is(err, quarry.ErrTargetHasSeparator) {
		t.Errorf("RepoRelTarget(escaping and separator) error = %v; want not errors.Is(_, quarry.ErrTargetHasSeparator)", err)
	}
}

// TestRepoRelTarget_CleanTargetUnaffected pins that a target with no "#" anywhere in its cleaned
// relative form is unaffected by the separator reject.
func TestRepoRelTarget_CleanTargetUnaffected(t *testing.T) {
	got, err := RepoRelTarget("/repo", "/repo", "a/b/c.go")
	if err != nil {
		t.Fatalf("RepoRelTarget(clean) = _, %v; want nil error", err)
	}
	if want := "a/b/c.go"; got != want {
		t.Errorf("RepoRelTarget(clean) = %q; want %q", got, want)
	}
}

func TestRepoRelTarget_NativeSeparatorsToForwardSlash(t *testing.T) {
	root := filepath.Join(string(filepath.Separator) + "repo")
	base := root
	target := filepath.Join("a", "b", "c.go")

	got, err := RepoRelTarget(root, base, target)
	if err != nil {
		t.Fatalf("RepoRelTarget(native separators) = _, %v; want nil error", err)
	}
	if want := "a/b/c.go"; got != want {
		t.Errorf("RepoRelTarget(native separators) = %q; want %q", got, want)
	}
}

// TestRepoRelPathArithmetic_LeadingDotDotNotRejected pins that repoRelPath is arithmetic only: a
// target that leaves the root comes back as a leading-".." relative path rather than an error, for
// a sibling directory of the root and for a parent of the root.
func TestRepoRelPathArithmetic_LeadingDotDotNotRejected(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		base   string
		target string
		want   string
	}{
		{"sibling-directory", "/repo", "/repo", "../sibling/file.go", "../sibling/file.go"},
		{"parent-of-root", "/repo/sub", "/repo/sub", "..", ".."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repoRelPath(tt.root, tt.base, tt.target)
			if err != nil {
				t.Fatalf("repoRelPath(%q) = _, %v; want nil error", tt.target, err)
			}
			if got != tt.want {
				t.Errorf("repoRelPath(%q) = %q; want %q", tt.target, got, tt.want)
			}
		})
	}
}

// TestRepoRelPathArithmetic_AgreesWithRepoRelTarget pins that repoRelPath and repoRelTarget agree
// on every input that neither escapes the root nor contains a "#", including the root itself and
// a nested path. The separator divergence is asserted as its own row, since it is where the two
// functions now disagree: repoRelPath returns the cleaned relative path with no error, while
// repoRelTarget rejects it with quarry.ErrTargetHasSeparator.
func TestRepoRelPathArithmetic_AgreesWithRepoRelTarget(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		base   string
		target string
	}{
		{"root-itself", "/repo", "/repo", "."},
		{"nested-path", "/repo", "/repo", "a/b/c.go"},
		{"cwd-relative", "/repo", "/repo/internal/logger", "file.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathGot, pathErr := repoRelPath(tt.root, tt.base, tt.target)
			targetGot, targetErr := repoRelTarget(tt.root, tt.base, tt.target)
			if pathErr != nil || targetErr != nil {
				t.Fatalf("repoRelPath/repoRelTarget(%q) errors = %v, %v; want both nil", tt.target, pathErr, targetErr)
			}
			if pathGot != targetGot {
				t.Errorf("repoRelPath(%q) = %q; repoRelTarget(%q) = %q; want equal", tt.target, pathGot, tt.target, targetGot)
			}
		})
	}
}

// TestRepoRelPathArithmetic_DivergesOnSeparator pins the one input class where repoRelPath and
// repoRelTarget now disagree: a target whose cleaned relative form carries a "#". repoRelPath
// still returns it as ordinary arithmetic; repoRelTarget rejects it.
func TestRepoRelPathArithmetic_DivergesOnSeparator(t *testing.T) {
	root, base, target := "/repo", "/repo", "a#b/c.go"

	pathGot, pathErr := repoRelPath(root, base, target)
	if pathErr != nil {
		t.Fatalf("repoRelPath(%q) = _, %v; want nil error", target, pathErr)
	}
	if want := "a#b/c.go"; pathGot != want {
		t.Errorf("repoRelPath(%q) = %q; want %q", target, pathGot, want)
	}

	_, targetErr := repoRelTarget(root, base, target)
	if !errors.Is(targetErr, quarry.ErrTargetHasSeparator) {
		t.Errorf("repoRelTarget(%q) error = %v; want errors.Is(_, quarry.ErrTargetHasSeparator)", target, targetErr)
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

	got, err := RepoRelTarget(root, root, "link")
	if err != nil {
		t.Fatalf("RepoRelTarget(symlink) = _, %v; want nil error", err)
	}
	if got != "link" {
		t.Errorf("RepoRelTarget(symlink) = %q; want %q (the symlink's own path, not the directory it points at)", got, "link")
	}
}
