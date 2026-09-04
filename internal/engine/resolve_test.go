// resolve_test.go covers Repo.unitDirs, Repo.symbolsOfUnit's ignore filtering, the public
// Repo.SpansOf, and Repo.Resolve with its closed status vocabulary: the literal-first unit lookup,
// the external-test unit split, the collision case, argument validation through glyph.Parse's own
// round trip, the two different dispositions an unspellable unit can take — "listed but no
// symbols" on the walk's side, and "rejected outright" here — the target split between a glyph and
// a path, the found/multipart/ambiguous/not_found decision, the parse-once memo guarantee, the
// argument-order and arity contract, and the per-entry error boundary.
//
// Fixtures split the same way the rest of this package's tests do: cases that do not exercise
// .gitignore behaviour read the committed testdata/tree/, testdata/foo_test/, testdata/glyphs/,
// testdata/methods/ and testdata/tags/ fixtures against the quarry module root; the two unit
// collision trees, the ignore-filter cases and the permissions case build run-time trees under
// .scratch/engine-tests/ via writeScratchTree (through openScratchRepo), since a committed tree
// cannot hold a file its own .gitignore excludes, cannot express a directory collision without
// breaking TestSpansOf_LiteralFirst, and cannot hold an unreadable directory in version control.

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

// TestStatusForMatches tables statusForMatches over its full row order. Neither this test nor the
// function it exercises reads the filesystem: matches is read only for its length, and g only for
// Owner and Name, so every case builds its values by hand rather than through a fixture. The two
// rows that pin the check order are zero-matches-under-collision, which must be not_found and not
// ambiguous, and several-init-under-collision, which must be ambiguous and not multipart.
func TestStatusForMatches(t *testing.T) {
	initGlyph := glyph.Glyph{Lang: glyph.Go, Name: "init"}
	fooGlyph := glyph.Glyph{Lang: glyph.Go, Name: "Foo"}
	methodGlyph := glyph.Glyph{Lang: glyph.Go, Owner: []string{"Widget"}, Name: "Method"}

	tests := []struct {
		name      string
		g         glyph.Glyph
		matches   []Symbol
		collision bool
		want      Status
	}{
		{"ZeroMatchesNoCollision", fooGlyph, nil, false, StatusNotFound},
		{"ZeroMatchesCollision", fooGlyph, nil, true, StatusNotFound},
		{"OneMatchNoCollision", fooGlyph, make([]Symbol, 1), false, StatusFound},
		{"OneMatchCollision", fooGlyph, make([]Symbol, 1), true, StatusAmbiguous},
		{"ThreeInitNoCollision", initGlyph, make([]Symbol, 3), false, StatusMultipart},
		{"ThreeInitCollision", initGlyph, make([]Symbol, 3), true, StatusAmbiguous},
		{"TwoNonInitPackageLevel", fooGlyph, make([]Symbol, 2), false, StatusAmbiguous},
		{"TwoMatchesCollision", fooGlyph, make([]Symbol, 2), true, StatusAmbiguous},
		{"TwoMatchesOwnedName", methodGlyph, make([]Symbol, 2), false, StatusAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusForMatches(tt.g, tt.matches, tt.collision)
			if got != tt.want {
				t.Errorf("statusForMatches(%v, %d matches, collision=%v) = %q; want %q", tt.g, len(tt.matches), tt.collision, got, tt.want)
			}
		})
	}
}

// TestIsGlyphTarget tables isGlyphTarget over every target containing "#" and every target that
// does not. "#x" is a glyph target the grammar then rejects (an empty unit), not a path — the
// split does not pre-empt the alphabet's own rules, and this test asserts only the split itself.
func TestIsGlyphTarget(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{"a/b#C", true},
		{"a/b", false},
		{"#x", true},
		{"a#b#c", true},
		{"Makefile", false},
		{"notes.rst", false},
		{"", false},
		{".", false},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := isGlyphTarget(tt.target)
			if got != tt.want {
				t.Errorf("isGlyphTarget(%q) = %v; want %v", tt.target, got, tt.want)
			}
		})
	}
}

// symbolFromTOC returns the symbol named id from fileRel's own TOC file-target answer — the walk's
// independent computation of a symbol's fields, read through a different code path than Resolve's
// own, so a test comparing the two is a genuine parity check rather than a tautology.
func symbolFromTOC(t *testing.T, r *Repo, fileRel, id string) Symbol {
	t.Helper()
	dir, err := r.TOC(fileRel, TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC(%q) returned error: %v", fileRel, err)
	}
	if len(dir.Files) != 1 || dir.Files[0].Symbols == nil {
		t.Fatalf("TOC(%q) = %+v; want exactly one file entry carrying symbols", fileRel, dir)
	}
	for _, sym := range *dir.Files[0].Symbols {
		if sym.ID == id {
			return sym
		}
	}
	t.Fatalf("TOC(%q) symbols = %+v; want one named %q", fileRel, *dir.Files[0].Symbols, id)
	return Symbol{}
}

// TestResolve_Found asserts a package-level function and a method each resolve to exactly one
// match, with every listable field of the returned symbol matching what an independent TOC
// file-target walk reports for the same declaration.
func TestResolve_Found(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		fileRel string
	}{
		{"PackageLevelFunction", "internal/engine/testdata/tree/pkg#Alpha", "internal/engine/testdata/tree/pkg/alpha.go"},
		{"Method", "internal/engine/testdata/methods#Widget.Alpha", "internal/engine/testdata/methods/aardvark.go"},
	}

	r := openQuarryRoot(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := r.Resolve([]string{tt.target})
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", tt.target, err)
			}
			if len(results) != 1 {
				t.Fatalf("Resolve(%q) = %d results; want 1", tt.target, len(results))
			}
			res := results[0]
			if res.Status != StatusFound {
				t.Fatalf("Status = %q; want %q", res.Status, StatusFound)
			}
			if len(res.Symbols) != 1 {
				t.Fatalf("Symbols = %d entries; want 1", len(res.Symbols))
			}
			if res.Candidates != nil {
				t.Errorf("Candidates = %v; want absent", res.Candidates)
			}
			if res.Unit != "" {
				t.Errorf("Unit = %q; want empty", res.Unit)
			}
			if res.ID != res.Target {
				t.Errorf("ID = %q; want equal to Target %q — the Go alphabet normalises nothing", res.ID, res.Target)
			}

			got := res.Symbols[0]
			if got.File != tt.fileRel {
				t.Errorf("File = %q; want %q", got.File, tt.fileRel)
			}
			// symbolFromTOC's own Symbol carries no File — inside a toc answer the symbol already
			// sits in its file's own entry — so only the fields both readings share are compared.
			want := symbolFromTOC(t, r, tt.fileRel, got.ID)
			if got.Start != want.Start || got.SigEnd != want.SigEnd || got.End != want.End {
				t.Errorf("Start/SigEnd/End = %d/%d/%d; want %d/%d/%d", got.Start, got.SigEnd, got.End, want.Start, want.SigEnd, want.End)
			}
			if got.Signature != want.Signature {
				t.Errorf("Signature = %q; want %q", got.Signature, want.Signature)
			}
			if got.Doc != want.Doc {
				t.Errorf("Doc = %q; want %q", got.Doc, want.Doc)
			}
		})
	}
}

// TestResolve_Multipart resolves the bare init glyph of the glyphs fixture package, asserting all
// three func init() declarations come back as one multipart result in file-then-line order.
func TestResolve_Multipart(t *testing.T) {
	r := openQuarryRoot(t)
	target := "internal/engine/testdata/glyphs#init"

	results, err := r.Resolve([]string{target})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", target, err)
	}
	if len(results) != 1 {
		t.Fatalf("Resolve(%q) = %d results; want 1", target, len(results))
	}
	res := results[0]
	if res.Status != StatusMultipart {
		t.Fatalf("Status = %q; want %q", res.Status, StatusMultipart)
	}
	if len(res.Symbols) != 3 {
		t.Fatalf("Symbols = %d entries; want 3", len(res.Symbols))
	}
	if res.Candidates != nil {
		t.Errorf("Candidates = %v; want absent", res.Candidates)
	}
	for i := 1; i < len(res.Symbols); i++ {
		prev, cur := res.Symbols[i-1], res.Symbols[i]
		if prev.File > cur.File || (prev.File == cur.File && prev.Start > cur.Start) {
			t.Errorf("Symbols not in file-then-line order: %+v then %+v", prev, cur)
		}
	}
}

// TestResolve_AmbiguousBuildTags resolves the duplicated function and type glyphs of the tags
// fixture package. Nothing under testdata/ is ever built, so both build-tagged declarations are
// read regardless of GOOS, and the engine reports both rather than guessing which the toolchain
// would pick.
func TestResolve_AmbiguousBuildTags(t *testing.T) {
	r := openQuarryRoot(t)
	targets := []string{
		"internal/engine/testdata/tags#Dup",
		"internal/engine/testdata/tags#DupType",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			results, err := r.Resolve([]string{target})
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", target, err)
			}
			res := results[0]
			if res.Status != StatusAmbiguous {
				t.Fatalf("Status = %q; want %q", res.Status, StatusAmbiguous)
			}
			if len(res.Candidates) != 2 {
				t.Fatalf("Candidates = %d entries; want 2", len(res.Candidates))
			}
			if res.Candidates[0].File == res.Candidates[1].File {
				t.Errorf("both candidates from %q; want two distinct files", res.Candidates[0].File)
			}
			if res.Symbols != nil {
				t.Errorf("Symbols = %v; want absent", res.Symbols)
			}
		})
	}
}

// TestResolve_NotFoundBothWays asserts the two dispositions a not_found result can carry — unit:
// found when the directory exists and only the member is missing, unit: not_found when the unit
// directory does not exist at all — and marshals one of each to pin the emitted JSON spelling, plus
// a found and an ambiguous result so every one of ResolveResult's keys is observed both present and
// absent across this test and card 13's.
func TestResolve_NotFoundBothWays(t *testing.T) {
	r := openQuarryRoot(t)

	missingName := "internal/engine/testdata/tree/pkg#NoSuchDeclaration"
	missingUnit := "internal/engine/testdata/does-not-exist#X"

	results, err := r.Resolve([]string{missingName, missingUnit})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Resolve = %d results; want 2", len(results))
	}

	nameRes, unitRes := results[0], results[1]
	if nameRes.Status != StatusNotFound || nameRes.Unit != StatusFound {
		t.Errorf("Resolve(%q) = %+v; want not_found with unit: found", missingName, nameRes)
	}
	if unitRes.Status != StatusNotFound || unitRes.Unit != StatusNotFound {
		t.Errorf("Resolve(%q) = %+v; want not_found with unit: not_found", missingUnit, unitRes)
	}

	for _, res := range []ResolveResult{nameRes, unitRes} {
		m := marshalToMap(t, res)
		for _, key := range []string{"target", "id", "status", "unit"} {
			if _, ok := m[key]; !ok {
				t.Errorf("marshalled %s: missing key %q in %v", res.Target, key, m)
			}
		}
		if got := m["unit"]; got != string(res.Unit) {
			t.Errorf("marshalled %s: unit = %v; want %q", res.Target, got, res.Unit)
		}
		for _, key := range []string{"symbols", "candidates", "dir", "error", "reason"} {
			if _, ok := m[key]; ok {
				t.Errorf("marshalled %s: unexpected key %q present in %v", res.Target, key, m)
			}
		}
	}

	foundTarget := "internal/engine/testdata/tree/pkg#Alpha"
	foundResults, err := r.Resolve([]string{foundTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", foundTarget, err)
	}
	foundMap := marshalToMap(t, foundResults[0])
	if symbols, ok := foundMap["symbols"].([]any); !ok || len(symbols) != 1 {
		t.Errorf("marshalled found result: symbols = %v; want one entry present under \"symbols\"", foundMap["symbols"])
	}
	if _, ok := foundMap["candidates"]; ok {
		t.Errorf("marshalled found result: candidates present; want absent")
	}

	ambiguousTarget := "internal/engine/testdata/tags#Dup"
	ambiguousResults, err := r.Resolve([]string{ambiguousTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", ambiguousTarget, err)
	}
	ambiguousMap := marshalToMap(t, ambiguousResults[0])
	if candidates, ok := ambiguousMap["candidates"].([]any); !ok || len(candidates) != 2 {
		t.Errorf("marshalled ambiguous result: candidates = %v; want two entries present under \"candidates\"", ambiguousMap["candidates"])
	}
	if _, ok := ambiguousMap["symbols"]; ok {
		t.Errorf("marshalled ambiguous result: symbols present; want absent")
	}
}

// resolveCollisionTree builds the run-time collision tree TestResolve_AmbiguousCollision and
// TestResolve_CandidatesOrdered assert over, named distinctly from TestUnitDirs_Collision's own
// tree so the two cannot collide on disk: a "foo" directory holding a package-clause "foo" file
// and an external-test-clause sibling, plus a literal "foo_test" directory. The unit "foo_test"
// then resolves to both. The external test file under foo and the file under foo_test each declare
// a Dup function and a Thing type of the same name; the foo_test directory's own file additionally
// declares a Second type, declared nowhere else, so a single match under the collision can be
// asserted too. Their repository-relative paths sort with the foo directory's file first even
// though unitDirs returns the literal foo_test directory first.
func resolveCollisionTree(t *testing.T, name string) *Repo {
	t.Helper()
	return openScratchRepo(t, name, map[string]string{
		"foo/own.go":      "package foo\n\nfunc FooOwn() {}\n",
		"foo/own_test.go": "package foo_test\n\nfunc Dup() {}\n\ntype Thing struct{}\n",
		"foo_test/lit.go": "package foo_test\n\nfunc Dup() {}\n\ntype Thing struct{}\n\ntype Second struct{}\n",
	})
}

// TestResolve_AmbiguousCollision asserts three dispositions over the shared collision tree: the
// doubly-declared function glyph is ambiguous with both declarations in Candidates; the
// singly-declared type glyph, matching exactly once, is still ambiguous rather than found, because
// a found whose glyph string names two different units is exactly the failure literal-first exists
// to prevent; and a name declared in neither directory is not_found with unit: found, since both
// directories exist and only the member is missing.
func TestResolve_AmbiguousCollision(t *testing.T) {
	r := resolveCollisionTree(t, "resolve-verb-collision")

	dupResults, err := r.Resolve([]string{"foo_test#Dup"})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", "foo_test#Dup", err)
	}
	dupRes := dupResults[0]
	if dupRes.Status != StatusAmbiguous {
		t.Errorf("Dup Status = %q; want %q", dupRes.Status, StatusAmbiguous)
	}
	if len(dupRes.Candidates) != 2 {
		t.Errorf("Dup Candidates = %d entries; want 2", len(dupRes.Candidates))
	}

	secondResults, err := r.Resolve([]string{"foo_test#Second"})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", "foo_test#Second", err)
	}
	secondRes := secondResults[0]
	if secondRes.Status != StatusAmbiguous {
		t.Errorf("Second Status = %q; want %q — a single match under a unit collision is still ambiguous", secondRes.Status, StatusAmbiguous)
	}
	if len(secondRes.Candidates) != 1 {
		t.Errorf("Second Candidates = %d entries; want 1", len(secondRes.Candidates))
	}

	missingResults, err := r.Resolve([]string{"foo_test#NoSuchName"})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", "foo_test#NoSuchName", err)
	}
	missingRes := missingResults[0]
	if missingRes.Status != StatusNotFound || missingRes.Unit != StatusFound {
		t.Errorf("NoSuchName result = %+v; want not_found with unit: found", missingRes)
	}
}

// TestResolve_CandidatesOrdered asserts the doubly-declared glyph's Candidates come back ordered by
// file then start line — the foo directory's file first — even though symbolsOfUnit appended the
// literal foo_test directory's symbols first. os.ReadDir already returns entries sorted by
// filename, so a single directory's read order proves nothing; this collision tree is the one place
// the engine's own sort is genuinely load-bearing.
func TestResolve_CandidatesOrdered(t *testing.T) {
	r := resolveCollisionTree(t, "resolve-verb-collision-ordered")

	results, err := r.Resolve([]string{"foo_test#Dup"})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", "foo_test#Dup", err)
	}
	candidates := results[0].Candidates
	if len(candidates) != 2 {
		t.Fatalf("Candidates = %d entries; want 2", len(candidates))
	}
	if candidates[0].File != "foo/own_test.go" {
		t.Errorf("Candidates[0].File = %q; want %q", candidates[0].File, "foo/own_test.go")
	}
	if candidates[1].File != "foo_test/lit.go" {
		t.Errorf("Candidates[1].File = %q; want %q", candidates[1].File, "foo_test/lit.go")
	}
}

// TestResolve_ParsesEachUnitOnce constructs a unitMemo directly and calls r.resolve with eight
// targets spread over three distinct units, one of them named four times, asserting parses equals
// the number of distinct units rather than the number of targets. Asserting the memo map's length
// instead would be true by construction and prove nothing.
func TestResolve_ParsesEachUnitOnce(t *testing.T) {
	r := openQuarryRoot(t)
	m, err := newUnitMemo(r)
	if err != nil {
		t.Fatalf("newUnitMemo returned error: %v", err)
	}

	targets := []string{
		"internal/engine/testdata/tree/pkg#Alpha",
		"internal/engine/testdata/tree/pkg#Beta",
		"internal/engine/testdata/tree/pkg#Alpha",
		"internal/engine/testdata/tree/pkg#NoSuchName",
		"internal/engine/testdata/methods#Widget",
		"internal/engine/testdata/methods#Widget.Alpha",
		"internal/engine/testdata/tags#Dup",
		"internal/engine/testdata/tags#DupType",
	}
	if _, err := r.resolve(targets, m); err != nil {
		t.Fatalf("resolve(%v) returned error: %v", targets, err)
	}
	if m.parses != 3 {
		t.Errorf("parses = %d; want 3 — three distinct units across %d targets", m.parses, len(targets))
	}
}

// TestResolve_UnitDirectoryMissingIsNotAnError asserts a glyph whose unit directory does not exist
// returns a result rather than failing the call — the assertion that stops a future change turning
// a missing directory into a call failure and making the create-a-new-unit case unanswerable.
func TestResolve_UnitDirectoryMissingIsNotAnError(t *testing.T) {
	r := openQuarryRoot(t)
	target := "internal/engine/testdata/does-not-exist#X"

	results, err := r.Resolve([]string{target})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v; want a not_found result, not a call failure", target, err)
	}
	if results == nil || len(results) != 1 {
		t.Fatalf("Resolve(%q) = %v; want a non-nil slice of length 1", target, results)
	}
	res := results[0]
	if res.Status != StatusNotFound {
		t.Errorf("Status = %q; want %q", res.Status, StatusNotFound)
	}
	if res.Unit != StatusNotFound {
		t.Errorf("Unit = %q; want %q", res.Unit, StatusNotFound)
	}
}

// TestResolve_ReadFailureFailsTheCall makes a fixture unit's directory unreadable mid-test and
// asserts Resolve fails the whole call with an error that is neither ErrTargetNotFound nor
// ErrTargetOutsideRepo, rather than reporting a not_found entry — an engine read failure is not an
// answer about a glyph. Skips when the host cannot revoke read permission (running as root, or a
// filesystem without the bit), detected by re-reading the directory after the chmod.
func TestResolve_ReadFailureFailsTheCall(t *testing.T) {
	r := openScratchRepo(t, "resolve-read-failure", map[string]string{
		"pkg/a.go": "package pkg\n\nfunc A() {}\n",
	})

	dirPath := filepath.Join(r.root, "pkg")
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("stat %q: %v", dirPath, err)
	}
	origMode := info.Mode()

	if err := os.Chmod(dirPath, 0o000); err != nil {
		t.Fatalf("chmod %q: %v", dirPath, err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dirPath, origMode); err != nil {
			t.Errorf("restore chmod %q: %v", dirPath, err)
		}
	})

	if entries, readErr := os.ReadDir(dirPath); readErr == nil {
		t.Skipf("host does not enforce directory read permissions (read %d entries); skipping", len(entries))
	}

	target := "pkg#A"
	results, err := r.Resolve([]string{target})
	if err == nil {
		t.Fatalf("Resolve(%q) = %v, nil error; want a non-nil error", target, results)
	}
	if errors.Is(err, ErrTargetNotFound) || errors.Is(err, ErrTargetOutsideRepo) {
		t.Errorf("Resolve error = %v; want neither ErrTargetNotFound nor ErrTargetOutsideRepo", err)
	}
	if results != nil {
		t.Errorf("Resolve(%q) = %v; want nil slice on failure", target, results)
	}
}

// TestResolve_ArgumentOrderAndArity asserts Resolve's positional 1:1 contract: exactly len(targets)
// results, each answering its own argument at the same index, with a repeated target answered
// twice and identically. A nil targets slice returns an empty, non-nil slice and a nil error.
func TestResolve_ArgumentOrderAndArity(t *testing.T) {
	r := openQuarryRoot(t)
	glyphTarget := "internal/engine/testdata/tree/pkg#Alpha"
	pathTarget := "internal/engine/testdata/tree/pkg/alpha.go"
	malformed := "a/./b#X"

	targets := []string{glyphTarget, pathTarget, malformed, glyphTarget}
	results, err := r.Resolve(targets)
	if err != nil {
		t.Fatalf("Resolve(%v) returned error: %v", targets, err)
	}
	if len(results) != len(targets) {
		t.Fatalf("Resolve(%v) = %d results; want %d", targets, len(results), len(targets))
	}
	for i, res := range results {
		if res.Target != targets[i] {
			t.Errorf("results[%d].Target = %q; want %q", i, res.Target, targets[i])
		}
	}
	if !reflect.DeepEqual(results[0], results[3]) {
		t.Errorf("repeated target answered differently: results[0] = %+v, results[3] = %+v", results[0], results[3])
	}

	empty, err := r.Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve(nil) returned error: %v", err)
	}
	if empty == nil {
		t.Fatalf("Resolve(nil) = nil slice; want an empty, non-nil slice")
	}
	if len(empty) != 0 {
		t.Fatalf("Resolve(nil) = %d results; want 0", len(empty))
	}
}

// TestResolve_PathTargets asserts every disposition a path target can take: an existing file
// answers found with Dir carrying the enclosing directory's own facts and exactly one Files entry
// with Symbols nil; an existing directory answers found with its own Files populated and every
// entry's Symbols nil; a missing path answers not_found with Dir absent and Unit empty, since a
// path belongs to no unit; an absolute path and a ".."-escaping path each answer with Status empty,
// Error non-empty and Reason empty; and an explicitly named gitignored file is still answered,
// because the ignore filter exists so a listing is not noise, not to make a file unaddressable.
func TestResolve_PathTargets(t *testing.T) {
	r := openQuarryRoot(t)

	fileTarget := "internal/engine/testdata/tree/pkg/alpha.go"
	fileResults, err := r.Resolve([]string{fileTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", fileTarget, err)
	}
	fileRes := fileResults[0]
	if fileRes.Status != StatusFound {
		t.Fatalf("Status = %q; want %q", fileRes.Status, StatusFound)
	}
	if fileRes.Dir == nil {
		t.Fatalf("Dir is nil; want the enclosing directory's answer")
	}
	if fileRes.Dir.Dir != "internal/engine/testdata/tree/pkg" {
		t.Errorf("Dir.Dir = %q; want %q", fileRes.Dir.Dir, "internal/engine/testdata/tree/pkg")
	}
	if fileRes.Dir.Package == "" || fileRes.Dir.Language == "" {
		t.Errorf("Dir.Package/Language = %q/%q; want both non-empty", fileRes.Dir.Package, fileRes.Dir.Language)
	}
	if len(fileRes.Dir.Files) != 1 {
		t.Fatalf("Dir.Files = %d entries; want 1", len(fileRes.Dir.Files))
	}
	if fileRes.Dir.Files[0].Symbols != nil {
		t.Errorf("Dir.Files[0].Symbols = %v; want nil — symbols are switched off", fileRes.Dir.Files[0].Symbols)
	}

	dirTarget := "internal/engine/testdata/tree/pkg"
	dirResults, err := r.Resolve([]string{dirTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", dirTarget, err)
	}
	dirRes := dirResults[0]
	if dirRes.Status != StatusFound {
		t.Fatalf("Status = %q; want %q", dirRes.Status, StatusFound)
	}
	if dirRes.Dir == nil || len(dirRes.Dir.Files) == 0 {
		t.Fatalf("Dir = %+v; want a populated Files list", dirRes.Dir)
	}
	for _, fe := range dirRes.Dir.Files {
		if fe.Symbols != nil {
			t.Errorf("Files[%q].Symbols = %v; want nil", fe.Name, fe.Symbols)
		}
	}

	missingTarget := "internal/engine/testdata/does-not-exist-path"
	missingResults, err := r.Resolve([]string{missingTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", missingTarget, err)
	}
	missingRes := missingResults[0]
	if missingRes.Status != StatusNotFound {
		t.Errorf("Status = %q; want %q", missingRes.Status, StatusNotFound)
	}
	if missingRes.Dir != nil {
		t.Errorf("Dir = %+v; want absent", missingRes.Dir)
	}
	if missingRes.Unit != "" {
		t.Errorf("Unit = %q; want empty — a path belongs to no unit", missingRes.Unit)
	}

	rejected := []string{"/etc/passwd", "../outside"}
	rejectedResults, err := r.Resolve(rejected)
	if err != nil {
		t.Fatalf("Resolve(%v) returned error: %v", rejected, err)
	}
	for i, res := range rejectedResults {
		if res.Status != "" {
			t.Errorf("Resolve(%q) Status = %q; want empty", rejected[i], res.Status)
		}
		if res.Error == "" {
			t.Errorf("Resolve(%q) Error is empty; want non-empty", rejected[i])
		}
		if res.Reason != "" {
			t.Errorf("Resolve(%q) Reason = %q; want empty", rejected[i], res.Reason)
		}
	}

	scratch := openScratchRepo(t, "resolve-path-ignore-filter", map[string]string{
		".gitignore":   "excluded.txt\n",
		"excluded.txt": "hidden\n",
	})
	excludedTarget := "excluded.txt"
	excludedResults, err := scratch.Resolve([]string{excludedTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", excludedTarget, err)
	}
	if excludedResults[0].Status != StatusFound {
		t.Errorf("Resolve(%q) Status = %q; want %q — an explicitly named gitignored file is still answered", excludedTarget, excludedResults[0].Status, StatusFound)
	}

	// Marshal the existing-file result: dir present under exactly that spelling, carrying the
	// directory answer's own keys, with id and unit absent — the path branch is the reason both
	// carry omitempty, and this is the only marshal that observes dir present.
	m := marshalToMap(t, fileRes)
	dirVal, ok := m["dir"].(map[string]any)
	if !ok {
		t.Fatalf("marshalled path result: dir missing or wrong shape in %v", m)
	}
	for _, key := range []string{"dir", "package", "language", "files"} {
		if _, ok := dirVal[key]; !ok {
			t.Errorf("marshalled dir: missing key %q in %v", key, dirVal)
		}
	}
	if _, ok := m["id"]; ok {
		t.Errorf("marshalled path result: id present; want absent")
	}
	if _, ok := m["unit"]; ok {
		t.Errorf("marshalled path result: unit present; want absent")
	}
}

// TestResolve_MalformedGlyphEntries asserts one entry per distinct grammar rejection the engine can
// reach: each answers with Status empty, Error non-empty and Reason equal to the grammar's own word
// for that rejection. It marshals one entry to pin that a rejected target emits no status word at
// all, and asserts a call mixing one malformed target with two valid ones still answers the two
// valid ones normally with a nil call error — the whole reason the rejection is carried per entry
// rather than raised as the call's error.
func TestResolve_MalformedGlyphEntries(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantReason glyph.Reason
	}{
		{"MemberTooDeep", "internal/engine/testdata/tree/pkg#A.B.C", glyph.ReasonMemberTooDeep},
		{"UnitBadRune", "internal/engine/testdata/tree pkg#X", glyph.ReasonUnitBadRune},
		{"UnitDotSegment", "a/./b#X", glyph.ReasonUnitDotSegment},
		{"MemberKeyword", "internal/engine/testdata/tree/pkg#func", glyph.ReasonMemberKeyword},
	}

	r := openQuarryRoot(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := r.Resolve([]string{tt.target})
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", tt.target, err)
			}
			res := results[0]
			if res.Status != "" {
				t.Errorf("Status = %q; want empty", res.Status)
			}
			if res.Error == "" {
				t.Errorf("Error is empty; want non-empty")
			}
			if res.Reason != string(tt.wantReason) {
				t.Errorf("Reason = %q; want %q", res.Reason, tt.wantReason)
			}
		})
	}

	results, err := r.Resolve([]string{tests[0].target})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", tests[0].target, err)
	}
	m := marshalToMap(t, results[0])
	for _, key := range []string{"error", "reason"} {
		if _, ok := m[key]; !ok {
			t.Errorf("marshalled malformed result: missing key %q in %v", key, m)
		}
	}
	if _, ok := m["status"]; ok {
		t.Errorf("marshalled malformed result: status present; want absent")
	}

	mixed := []string{
		"internal/engine/testdata/tree/pkg#Alpha",
		tests[0].target,
		"internal/engine/testdata/tree/pkg#Beta",
	}
	mixedResults, err := r.Resolve(mixed)
	if err != nil {
		t.Fatalf("Resolve(%v) returned error: %v", mixed, err)
	}
	if mixedResults[0].Status != StatusFound || mixedResults[2].Status != StatusFound {
		t.Errorf("valid entries in a mixed call = %+v, %+v; want both found", mixedResults[0], mixedResults[2])
	}
	if mixedResults[1].Status != "" || mixedResults[1].Error == "" {
		t.Errorf("malformed entry in a mixed call = %+v; want empty status and non-empty error", mixedResults[1])
	}
}
