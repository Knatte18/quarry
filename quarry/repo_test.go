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

// resolveExpandFixture is the source used by every Resolve and Expand test in this file. Its one Go
// file sits under the nested "sub" package directory, not the fixture root: a file at the fixture
// root has the empty unit, which no glyph can spell, so a glyph case built against the root could
// never resolve. Foo is a free function; Thing is a type with one member method.
const resolveExpandFixture = "package sub\n\nfunc Foo() {}\n\ntype Thing struct{}\n\nfunc (t Thing) Method() {}\n"

// TestRepoResolve_GlyphFound asserts a glyph naming a free function resolves to StatusFound with
// that one declaration in Symbols.
func TestRepoResolve_GlyphFound(t *testing.T) {
	root := writeScratchTree(t, "resolve-glyph-found", map[string]string{"sub/a.go": resolveExpandFixture})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	results, err := r.Resolve([]string{"sub#Foo"})
	if err != nil {
		t.Fatalf("Resolve([%q]) returned error: %v", "sub#Foo", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(Resolve([%q])) = %d; want 1", "sub#Foo", len(results))
	}
	got := results[0]
	if got.Status != StatusFound {
		t.Errorf("Status = %q; want %q", got.Status, StatusFound)
	}
	if len(got.Symbols) != 1 {
		t.Fatalf("len(Symbols) = %d; want 1", len(got.Symbols))
	}
	if got.Symbols[0].ID != "sub#Foo" {
		t.Errorf("Symbols[0].ID = %q; want %q", got.Symbols[0].ID, "sub#Foo")
	}
}

// TestRepoResolve_GlyphMemberMissingIsNotFound asserts a glyph whose unit exists but whose member
// does not resolves to StatusNotFound with Unit == StatusFound.
func TestRepoResolve_GlyphMemberMissingIsNotFound(t *testing.T) {
	root := writeScratchTree(t, "resolve-glyph-missing-member", map[string]string{"sub/a.go": resolveExpandFixture})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	results, err := r.Resolve([]string{"sub#DoesNotExist"})
	if err != nil {
		t.Fatalf("Resolve([%q]) returned error: %v", "sub#DoesNotExist", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(Resolve([%q])) = %d; want 1", "sub#DoesNotExist", len(results))
	}
	got := results[0]
	if got.Status != StatusNotFound {
		t.Errorf("Status = %q; want %q", got.Status, StatusNotFound)
	}
	if got.Unit != StatusFound {
		t.Errorf("Unit = %q; want %q", got.Unit, StatusFound)
	}
}

// TestRepoResolve_PathTarget asserts a repository-relative path target resolves to StatusFound with
// a non-nil directory answer.
func TestRepoResolve_PathTarget(t *testing.T) {
	root := writeScratchTree(t, "resolve-path", map[string]string{"sub/a.go": resolveExpandFixture})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	results, err := r.Resolve([]string{"sub"})
	if err != nil {
		t.Fatalf("Resolve([%q]) returned error: %v", "sub", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(Resolve([%q])) = %d; want 1", "sub", len(results))
	}
	got := results[0]
	if got.Status != StatusFound {
		t.Errorf("Status = %q; want %q", got.Status, StatusFound)
	}
	if got.Dir == nil {
		t.Fatalf("Dir = nil; want a non-nil directory answer")
	}
}

// TestRepoResolve_TwoTargetsPositional asserts a two-element target slice returns exactly two
// results, positionally.
func TestRepoResolve_TwoTargetsPositional(t *testing.T) {
	root := writeScratchTree(t, "resolve-two-targets", map[string]string{"sub/a.go": resolveExpandFixture})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	results, err := r.Resolve([]string{"sub#Foo", "sub#Thing"})
	if err != nil {
		t.Fatalf("Resolve(...) returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(Resolve(...)) = %d; want 2", len(results))
	}
	if results[0].Target != "sub#Foo" {
		t.Errorf("results[0].Target = %q; want %q", results[0].Target, "sub#Foo")
	}
	if results[1].Target != "sub#Thing" {
		t.Errorf("results[1].Target = %q; want %q", results[1].Target, "sub#Thing")
	}
}

// TestRepoExpand_TypeFound asserts a glyph naming a type expands to StatusFound with a non-nil head.
func TestRepoExpand_TypeFound(t *testing.T) {
	root := writeScratchTree(t, "expand-type-found", map[string]string{"sub/a.go": resolveExpandFixture})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	got, err := r.Expand("sub#Thing")
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", "sub#Thing", err)
	}
	if got.Status != StatusFound {
		t.Errorf("Status = %q; want %q", got.Status, StatusFound)
	}
	if got.Head == nil {
		t.Fatalf("Head = nil; want a non-nil head")
	}
}

// TestRepoExpand_FunctionIsNotAType is the transitivity test, the analogue of the sentinel tests
// above: it asserts a glyph naming a free function returns a non-nil error for which
// errors.As(err, &notType) against *NotATypeError succeeds, with ID and Kind readable, for a caller
// that never imports the engine.
func TestRepoExpand_FunctionIsNotAType(t *testing.T) {
	root := writeScratchTree(t, "expand-function-not-a-type", map[string]string{"sub/a.go": resolveExpandFixture})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	_, err = r.Expand("sub#Foo")
	if err == nil {
		t.Fatalf("Expand(%q) returned nil error; want a non-nil error", "sub#Foo")
	}
	var notType *NotATypeError
	if !errors.As(err, &notType) {
		t.Fatalf("Expand(%q) error = %v; want errors.As(err, &notType) against *NotATypeError to succeed", "sub#Foo", err)
	}
	if notType.ID != "sub#Foo" {
		t.Errorf("notType.ID = %q; want %q", notType.ID, "sub#Foo")
	}
	if notType.Kind != KindFunction {
		t.Errorf("notType.Kind = %q; want %q", notType.Kind, KindFunction)
	}
}
