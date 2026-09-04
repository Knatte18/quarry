// repo_test.go covers Open and resolveTarget: absolute, root-escaping, and nonexistent targets
// each matched against their own sentinel via errors.Is; a gitignored target still answered; a
// symlink target answered as a name-only entry inside its parent, never followed; "" and "."
// both answering the repository root; and Open rejecting a relative or non-directory root.

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestOpen_RejectsRelativeRoot asserts Open refuses a relative root.
func TestOpen_RejectsRelativeRoot(t *testing.T) {
	_, err := Open("relative/path")
	if err == nil {
		t.Fatal("Open(\"relative/path\") returned nil error; want an error for a non-absolute root")
	}
}

// TestOpen_RejectsNonDirectoryRoot asserts Open refuses a root that exists but is not a directory.
func TestOpen_RejectsNonDirectoryRoot(t *testing.T) {
	root := writeScratchTree(t, "repo-open-non-directory", map[string]string{"file.txt": "x"})
	filePath := filepath.Join(root, "file.txt")

	_, err := Open(filePath)
	if err == nil {
		t.Fatalf("Open(%q) returned nil error; want an error for a non-directory root", filePath)
	}
}

// TestOpen_AcceptsAbsoluteExistingDirectory asserts Open succeeds for an absolute, existing
// directory.
func TestOpen_AcceptsAbsoluteExistingDirectory(t *testing.T) {
	root := writeScratchTree(t, "repo-open-valid", map[string]string{"file.txt": "x"})
	if _, err := Open(root); err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}
}

// TestRepoTOC_AbsoluteTargetIsOutsideRepo asserts an absolute target is ErrTargetOutsideRepo.
func TestRepoTOC_AbsoluteTargetIsOutsideRepo(t *testing.T) {
	r := openScratchRepo(t, "repo-target-absolute", map[string]string{"foo.go": "package p\n"})
	_, err := r.TOC("/etc/passwd", TOCOptions{})
	if !errors.Is(err, ErrTargetOutsideRepo) {
		t.Errorf("TOC(\"/etc/passwd\", ...) error = %v; want errors.Is(err, ErrTargetOutsideRepo)", err)
	}
}

// TestRepoTOC_EscapingTargetIsOutsideRepo asserts a target that cleans to a path leaving the root
// is ErrTargetOutsideRepo.
func TestRepoTOC_EscapingTargetIsOutsideRepo(t *testing.T) {
	r := openScratchRepo(t, "repo-target-escaping", map[string]string{"foo.go": "package p\n"})
	_, err := r.TOC("../outside", TOCOptions{})
	if !errors.Is(err, ErrTargetOutsideRepo) {
		t.Errorf("TOC(\"../outside\", ...) error = %v; want errors.Is(err, ErrTargetOutsideRepo)", err)
	}
}

// TestRepoTOC_NonexistentTargetIsNotFound asserts a target that does not exist under the root is
// ErrTargetNotFound.
func TestRepoTOC_NonexistentTargetIsNotFound(t *testing.T) {
	r := openScratchRepo(t, "repo-target-nonexistent", map[string]string{"foo.go": "package p\n"})
	_, err := r.TOC("does-not-exist.go", TOCOptions{})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("TOC(\"does-not-exist.go\", ...) error = %v; want errors.Is(err, ErrTargetNotFound)", err)
	}
}

// TestRepoTOC_GitignoredTargetIsAnsweredNotRefused asserts an explicitly named gitignored target is
// still answered — the ignore set exists to keep a listing free of noise, not to make a file
// unaddressable.
func TestRepoTOC_GitignoredTargetIsAnsweredNotRefused(t *testing.T) {
	r := openScratchRepo(t, "repo-target-gitignored", map[string]string{
		".gitignore": "secret.go\n",
		"secret.go":  "package p\n\nfunc F() {}\n",
	})
	got, err := r.TOC("secret.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v; want the gitignored target answered", "secret.go", err)
	}
	if len(got.Files) != 1 || got.Files[0].Name != "secret.go" {
		t.Fatalf("Files = %+v; want the gitignored target listed", got.Files)
	}
}

// TestRepoTOC_SymlinkTargetIsNameOnlyNotFollowed asserts a target that is itself a symlink is
// answered as a name-only entry inside its parent's directory answer, never followed.
func TestRepoTOC_SymlinkTargetIsNameOnlyNotFollowed(t *testing.T) {
	root := writeScratchTree(t, "repo-target-symlink", map[string]string{
		"real.go": "package p\n\nfunc F() {}\n",
	})
	linkPath := filepath.Join(root, "link.go")
	if err := os.Symlink(filepath.Join(root, "real.go"), linkPath); err != nil {
		t.Fatalf("os.Symlink(...) failed: %v", err)
	}
	r := openRepo(t, root)

	got, err := r.TOC("link.go", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", "link.go", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Files) = %d; want 1", len(got.Files))
	}
	entry := got.Files[0]
	if entry.Name != "link.go" {
		t.Errorf("Name = %q; want %q", entry.Name, "link.go")
	}
	if entry.Header != "" || entry.Symbols != nil || entry.Error != "" || entry.Lossy {
		t.Errorf("entry = %+v; want a name-only entry, never opened", entry)
	}
}

// TestRepoTOC_EmptyAndDotBothAnswerRoot asserts "" and "." both answer the repository root with Dir
// equal to ".".
func TestRepoTOC_EmptyAndDotBothAnswerRoot(t *testing.T) {
	r := openScratchRepo(t, "repo-target-root", map[string]string{"foo.go": "package p\n"})
	for _, target := range []string{"", "."} {
		got, err := r.TOC(target, TOCOptions{})
		if err != nil {
			t.Fatalf("TOC(%q, ...) returned error: %v", target, err)
		}
		if got.Dir != "." {
			t.Errorf("TOC(%q, ...).Dir = %q; want %q", target, got.Dir, ".")
		}
	}
}
