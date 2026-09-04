// repo_test.go covers Open's error and success paths, TOC's delegation to the engine on a
// directory and a file target, the sentinel transitivity that is this batch's whole point, and the
// compile-time proof that the facade's answer types are aliases, not defined types.

package quarry

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/internal/engine"
)

// var _ DirAnswer = engine.DirAnswer{} and var _ engine.TOCOptions = TOCOptions{} are compile-time
// proof that the facade's aliases are aliases, not defined types: a value of the engine's type is
// directly assignable to the facade's spelling and back, with no conversion. This is what fails
// loudly, as a build error rather than a test failure, if a later edit turns an alias into a
// defined type.
var (
	_ DirAnswer         = engine.DirAnswer{}
	_ engine.TOCOptions = TOCOptions{}
)

// TestOpen covers Open's error and success paths over an absolute root, a relative root, a
// non-existent root, and a root naming a file rather than a directory.
func TestOpen(t *testing.T) {
	root := writeScratchTree(t, "open", map[string]string{"file.txt": "x"})
	filePath := filepath.Join(root, "file.txt")
	missingPath := filepath.Join(root, "does-not-exist")

	tests := []struct {
		name    string
		root    string
		wantErr bool
	}{
		{"RelativeRoot", "relative/path", true},
		{"NonExistentRoot", missingPath, true},
		{"FileRoot", filePath, true},
		{"AbsoluteExistingDirectory", root, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Open(tt.root)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Open(%q) returned nil error; want an error", tt.root)
				}
				const wantPrefix = "quarry: open"
				if len(err.Error()) < len(wantPrefix) || err.Error()[:len(wantPrefix)] != wantPrefix {
					t.Errorf("Open(%q) error = %q; want it to begin with %q", tt.root, err.Error(), wantPrefix)
				}
				return
			}
			if err != nil {
				t.Fatalf("Open(%q) returned error: %v", tt.root, err)
			}
			if r == nil {
				t.Fatalf("Open(%q) returned nil *Repo with a nil error", tt.root)
			}
		})
	}
}

// TestRepoTOC_DirectoryTarget asserts a directory target answers with the directory's own Dir and
// its files.
func TestRepoTOC_DirectoryTarget(t *testing.T) {
	root := writeScratchTree(t, "toc-directory", map[string]string{
		"sub/a.go": "package sub\n",
		"sub/b.go": "package sub\n",
	})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	got, err := r.TOC("sub", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", "sub", err)
	}
	if got.Dir != "sub" {
		t.Errorf("Dir = %q; want %q", got.Dir, "sub")
	}
	if len(got.Files) != 2 {
		t.Errorf("len(Files) = %d; want 2", len(got.Files))
	}
}

// TestRepoTOC_FileTarget asserts a file target answers with exactly one entry in Files.
func TestRepoTOC_FileTarget(t *testing.T) {
	root := writeScratchTree(t, "toc-file", map[string]string{
		"sub/a.go": "package sub\n",
	})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	got, err := r.TOC("sub/a.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", "sub/a.go", err)
	}
	if len(got.Files) != 1 || got.Files[0].Name != "a.go" {
		t.Errorf("Files = %+v; want a single entry named %q", got.Files, "a.go")
	}
}

// TestRepoTOC_MissingTargetIsNotFound asserts a missing target's error satisfies
// errors.Is(err, ErrTargetNotFound) against the facade's own sentinel, never the engine's, which is
// the sentinel transitivity this batch exists to prove.
func TestRepoTOC_MissingTargetIsNotFound(t *testing.T) {
	root := writeScratchTree(t, "toc-missing", map[string]string{"a.go": "package p\n"})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	_, err = r.TOC("does-not-exist.go", TOCOptions{})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("TOC(%q, ...) error = %v; want errors.Is(err, ErrTargetNotFound)", "does-not-exist.go", err)
	}
}

// TestRepoTOC_AbsoluteTargetIsOutsideRepo asserts an absolute target's error satisfies
// errors.Is(err, ErrTargetOutsideRepo) against the facade's own sentinel.
func TestRepoTOC_AbsoluteTargetIsOutsideRepo(t *testing.T) {
	root := writeScratchTree(t, "toc-absolute", map[string]string{"a.go": "package p\n"})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	_, err = r.TOC("/etc/passwd", TOCOptions{})
	if !errors.Is(err, ErrTargetOutsideRepo) {
		t.Errorf("TOC(\"/etc/passwd\", ...) error = %v; want errors.Is(err, ErrTargetOutsideRepo)", err)
	}
}
