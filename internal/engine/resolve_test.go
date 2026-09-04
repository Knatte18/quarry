// resolve_test.go covers Repo.unitDirs, Repo.symbolsOfUnit's ignore filtering, and the public
// Repo.SpansOf: the literal-first unit lookup, the external-test unit split, the collision case,
// argument validation through glyph.Parse's own round trip, and the two different dispositions an
// unspellable unit can take — "listed but no symbols" on the walk's side, and "rejected outright"
// here.
//
// Fixtures split the same way the rest of this package's tests do: cases that do not exercise
// .gitignore behaviour read the committed testdata/tree/ and testdata/foo_test/ fixtures against
// the quarry module root; the collision and ignore-filter cases build a run-time tree under
// .scratch/engine-tests/ via writeScratchTree, since a committed tree cannot hold a file its own
// .gitignore excludes.

package engine

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// openQuarryRoot opens this module's own root as a Repo, so a test can query the committed
// testdata/ fixtures by their real repository-relative paths.
func openQuarryRoot(t *testing.T) *Repo {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("openQuarryRoot: runtime.Caller(0) failed to resolve this file's path")
	}
	// thisFile is .../internal/engine/resolve_test.go; the module root is two directories up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return openRepo(t, moduleRoot)
}

// TestSpansOf_Hit asserts a glyph naming a real declaration in the committed testdata/tree/pkg
// fixture returns exactly its own span, with File set to the repository-relative path.
func TestSpansOf_Hit(t *testing.T) {
	r := openQuarryRoot(t)
	g := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/tree/pkg", Name: "Alpha"}

	got, err := r.SpansOf(g)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", g, err)
	}
	if len(got) != 1 {
		t.Fatalf("SpansOf(%v) = %d symbols; want 1", g, len(got))
	}
	sym := got[0]
	if sym.ID != "internal/engine/testdata/tree/pkg#Alpha" {
		t.Errorf("ID = %q; want %q", sym.ID, "internal/engine/testdata/tree/pkg#Alpha")
	}
	if sym.File != "internal/engine/testdata/tree/pkg/alpha.go" {
		t.Errorf("File = %q; want %q", sym.File, "internal/engine/testdata/tree/pkg/alpha.go")
	}
	if sym.Kind != KindFunction {
		t.Errorf("Kind = %q; want %q", sym.Kind, KindFunction)
	}
}

// TestSpansOf_Miss asserts a glyph whose unit exists but whose name does not returns an empty
// slice and a nil error, never a status value.
func TestSpansOf_Miss(t *testing.T) {
	r := openQuarryRoot(t)
	g := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/tree/pkg", Name: "NoSuchDeclaration"}

	got, err := r.SpansOf(g)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", g, err)
	}
	if len(got) != 0 {
		t.Fatalf("SpansOf(%v) = %d symbols; want 0", g, len(got))
	}
}

// TestSpansOf_UnitDirectoryMissing asserts a glyph whose unit directory does not exist returns an
// empty slice and a nil error, the same disposition as a name miss inside an existing unit.
func TestSpansOf_UnitDirectoryMissing(t *testing.T) {
	r := openQuarryRoot(t)
	g := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/does-not-exist", Name: "X"}

	got, err := r.SpansOf(g)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", g, err)
	}
	if len(got) != 0 {
		t.Fatalf("SpansOf(%v) = %d symbols; want 0", g, len(got))
	}
}

// TestSpansOf_ExternalTestUnit asserts the external-test unit split works in both directions:
// testdata/tree/pkg's own unit finds only its own declarations, and its "_test"-suffixed sibling
// unit finds only export_test.go's external test declaration.
func TestSpansOf_ExternalTestUnit(t *testing.T) {
	r := openQuarryRoot(t)

	ownGlyph := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/tree/pkg", Name: "Alpha"}
	own, err := r.SpansOf(ownGlyph)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", ownGlyph, err)
	}
	if len(own) != 1 || own[0].File != "internal/engine/testdata/tree/pkg/alpha.go" {
		t.Fatalf("SpansOf(%v) = %+v; want exactly Alpha from alpha.go", ownGlyph, own)
	}

	externalGlyph := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/tree/pkg_test", Name: "TestExported"}
	external, err := r.SpansOf(externalGlyph)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", externalGlyph, err)
	}
	if len(external) != 1 {
		t.Fatalf("SpansOf(%v) = %d symbols; want 1", externalGlyph, len(external))
	}
	if external[0].File != "internal/engine/testdata/tree/pkg/export_test.go" {
		t.Errorf("File = %q; want %q", external[0].File, "internal/engine/testdata/tree/pkg/export_test.go")
	}

	// The package's own unit never sees the external test declaration, and vice versa.
	ownForExternalName := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/tree/pkg", Name: "TestExported"}
	got, err := r.SpansOf(ownForExternalName)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", ownForExternalName, err)
	}
	if len(got) != 0 {
		t.Errorf("SpansOf(%v) = %d symbols; want 0 — TestExported belongs to the _test unit", ownForExternalName, len(got))
	}
}

// TestSpansOf_LiteralFirst asserts a glyph naming the committed testdata/foo_test/ directory finds
// that directory's own declaration, not one reached by stripping the "_test" suffix and looking in
// a "foo/" directory this repository does not have.
func TestSpansOf_LiteralFirst(t *testing.T) {
	r := openQuarryRoot(t)
	g := glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/foo_test", Name: "LiteralDeclaration"}

	got, err := r.SpansOf(g)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", g, err)
	}
	if len(got) != 1 {
		t.Fatalf("SpansOf(%v) = %d symbols; want 1", g, len(got))
	}
	if got[0].File != "internal/engine/testdata/foo_test/literal.go" {
		t.Errorf("File = %q; want %q", got[0].File, "internal/engine/testdata/foo_test/literal.go")
	}

	dirs, collision := r.unitDirs("internal/engine/testdata/foo_test")
	if collision {
		t.Errorf("collision = true; want false — this fixture has no sibling %q directory", "internal/engine/testdata/foo")
	}
	if len(dirs) != 1 || dirs[0] != "internal/engine/testdata/foo_test" {
		t.Errorf("unitDirs = %v; want exactly [%q]", dirs, "internal/engine/testdata/foo_test")
	}
}

// TestUnitDirs_Collision builds a .scratch/ tree holding both a "foo/" directory with an external
// test package and a literal "foo_test/" directory, and asserts unitDirs finds both with collision
// true while SpansOf returns each directory's own declaration — the union the caller sees.
func TestUnitDirs_Collision(t *testing.T) {
	r := openScratchRepo(t, "resolve-unit-collision", map[string]string{
		"foo/own.go":      "package foo\n\nfunc FooOwn() {}\n",
		"foo/own_test.go": "package foo_test\n\nfunc FooExternal() {}\n",
		"foo_test/lit.go": "package foo_test\n\nfunc FooLiteral() {}\n",
	})

	dirs, collision := r.unitDirs("foo_test")
	if !collision {
		t.Fatalf("unitDirs(%q) collision = false; want true", "foo_test")
	}
	wantDirs := map[string]bool{"foo_test": true, "foo": true}
	if len(dirs) != 2 {
		t.Fatalf("unitDirs(%q) = %v; want exactly 2 directories", "foo_test", dirs)
	}
	for _, d := range dirs {
		if !wantDirs[d] {
			t.Errorf("unitDirs(%q) returned unexpected directory %q", "foo_test", d)
		}
	}

	externalGlyph := glyph.Glyph{Lang: glyph.Go, Unit: "foo_test", Name: "FooExternal"}
	external, err := r.SpansOf(externalGlyph)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", externalGlyph, err)
	}
	if len(external) != 1 || external[0].File != "foo/own_test.go" {
		t.Fatalf("SpansOf(%v) = %+v; want exactly FooExternal from foo/own_test.go", externalGlyph, external)
	}

	literalGlyph := glyph.Glyph{Lang: glyph.Go, Unit: "foo_test", Name: "FooLiteral"}
	literal, err := r.SpansOf(literalGlyph)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", literalGlyph, err)
	}
	if len(literal) != 1 || literal[0].File != "foo_test/lit.go" {
		t.Fatalf("SpansOf(%v) = %+v; want exactly FooLiteral from foo_test/lit.go", literalGlyph, literal)
	}
}

// TestSpansOf_IgnoreFilter builds a .scratch/ tree whose .gitignore excludes one .go file beside
// two listed ones, and asserts no span from the excluded file is ever returned, while the two
// listed files' own declarations still are.
func TestSpansOf_IgnoreFilter(t *testing.T) {
	r := openScratchRepo(t, "resolve-ignore-filter", map[string]string{
		"pkg/.gitignore":  "excluded.go\n",
		"pkg/a.go":        "package pkg\n\nfunc A() {}\n",
		"pkg/b.go":        "package pkg\n\nfunc B() {}\n",
		"pkg/excluded.go": "package pkg\n\nfunc Excluded() {}\n",
	})

	excludedGlyph := glyph.Glyph{Lang: glyph.Go, Unit: "pkg", Name: "Excluded"}
	got, err := r.SpansOf(excludedGlyph)
	if err != nil {
		t.Fatalf("SpansOf(%v) returned error: %v", excludedGlyph, err)
	}
	if len(got) != 0 {
		t.Fatalf("SpansOf(%v) = %+v; want 0 — Excluded lives in a gitignored file", excludedGlyph, got)
	}

	for _, name := range []string{"A", "B"} {
		g := glyph.Glyph{Lang: glyph.Go, Unit: "pkg", Name: name}
		got, err := r.SpansOf(g)
		if err != nil {
			t.Fatalf("SpansOf(%v) returned error: %v", g, err)
		}
		if len(got) != 1 {
			t.Errorf("SpansOf(%v) = %d symbols; want 1", g, len(got))
		}
	}
}

// TestSpansOf_LanguageUnsupported asserts a non-Go Lang is rejected with ErrLanguageUnsupported,
// wrapped so errors.Is still succeeds.
func TestSpansOf_LanguageUnsupported(t *testing.T) {
	r := openQuarryRoot(t)
	g := glyph.Glyph{Lang: glyph.Language("python"), Unit: "internal/engine/testdata/tree/pkg", Name: "Alpha"}

	_, err := r.SpansOf(g)
	if !errors.Is(err, ErrLanguageUnsupported) {
		t.Fatalf("SpansOf(%v) error = %v; want errors.Is(err, ErrLanguageUnsupported)", g, err)
	}
}

// TestSpansOf_InvalidGlyphSurfacesParseError covers argument validation and unspellable units
// together, because they are the same rule surfacing through the same code path: SpansOf validates
// a hand-built Glyph by round-tripping it through glyph.Parse before ever reading a directory, so
// every alphabet violation — an empty unit, a "." segment, a member with too many components, or a
// unit the alphabet's rune rule rejects — is rejected outright with the grammar's own *glyph.
// ParseError, surfaced via errors.As. This is a different disposition from the walk's "listed but
// no symbols" for the same kind of unit (see unitFor's and unitSpellable's own doc comments): the
// walk lists the directory and simply omits Symbols, while the lookup here refuses to read the
// directory at all. The tests assert the two dispositions separately, and this table exists only to
// assert this one.
func TestSpansOf_InvalidGlyphSurfacesParseError(t *testing.T) {
	tests := []struct {
		name       string
		g          glyph.Glyph
		wantReason glyph.Reason
	}{
		{
			name:       "EmptyUnit",
			g:          glyph.Glyph{Lang: glyph.Go, Unit: "", Name: "X"},
			wantReason: glyph.ReasonUnitEmpty,
		},
		{
			name:       "DotSegment",
			g:          glyph.Glyph{Lang: glyph.Go, Unit: "a/./b", Name: "X"},
			wantReason: glyph.ReasonUnitDotSegment,
		},
		{
			name:       "MemberTooDeep",
			g:          glyph.Glyph{Lang: glyph.Go, Unit: "a/b", Owner: []string{"A", "B"}, Name: "C"},
			wantReason: glyph.ReasonMemberTooDeep,
		},
		{
			// The same rule as EmptyUnit above, exercised against a real fixture directory whose
			// path the Go alphabet's unit rune rule rejects, rather than a hand-built string: the
			// space-bearing testdata/units/ fixture directory glyph_test.go also reads.
			name:       "UnspellableUnitSpaceRune",
			g:          glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/units/test data/pkg", Name: "Spaced"},
			wantReason: glyph.ReasonUnitBadRune,
		},
	}

	r := openQuarryRoot(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.SpansOf(tt.g)
			var parseErr *glyph.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("SpansOf(%v) error = %v; want errors.As to *glyph.ParseError", tt.g, err)
			}
			if parseErr.Reason != tt.wantReason {
				t.Errorf("ParseError.Reason = %q; want %q", parseErr.Reason, tt.wantReason)
			}
		})
	}
}
