// units_test.go covers the three exported clause-and-unit helpers units.go declares:
// TestPackageClause tables the four ok-false conditions plus the success case over in-memory bytes;
// TestUnitsForClauseMap tables the vote and the unit derivation over map[base name]clause inputs;
// TestClauseMapForFiles exercises the on-disk caller over a directory built with writeScratchTree.
// None of these three reads a committed fixture tree — every case asserts against the helper's own
// return values only.

package engine

import (
	"reflect"
	"testing"
)

// TestPackageClause tables PackageClause over in-memory bytes: a plain Go file that succeeds, and
// the four conditions that make ok false — an extension with no registered strategy, bytes that
// are not valid UTF-8, a file with no package clause at all, and a file whose content does not
// parse into one.
func TestPackageClause(t *testing.T) {
	tests := []struct {
		name       string
		base       string
		src        []byte
		wantClause string
		wantOK     bool
		noStrategy bool
	}{
		{
			name:       "plain Go file",
			base:       "foo.go",
			src:        []byte("package demo\n"),
			wantClause: "demo",
			wantOK:     true,
		},
		{
			name:       "extension has no registered strategy",
			base:       "foo.go",
			src:        []byte("package demo\n"),
			noStrategy: true,
			wantClause: "",
			wantOK:     false,
		},
		{
			name:       "bytes are not valid UTF-8",
			base:       "foo.go",
			src:        []byte("package demo\n\xff\xfe"),
			wantClause: "",
			wantOK:     false,
		},
		{
			name:       "no package clause at all",
			base:       "foo.go",
			src:        []byte("func F() {}\n"),
			wantClause: "",
			wantOK:     false,
		},
		{
			name:       "content does not parse into a package clause",
			base:       "foo.go",
			src:        []byte("!!! not go source !!!\n"),
			wantClause: "",
			wantOK:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noStrategy {
				// StrategyFor("go") is temporarily emptied so LanguageForExtension still resolves the
				// extension while StrategyFor reports no registration for it — the one condition
				// PackageClause's own doc comment names that no fixture content alone can trigger,
				// since every extension this repository wires already has a strategy registered.
				previous := swapRegistry(make(map[string]Strategy))
				t.Cleanup(func() { swapRegistry(previous) })
			}

			gotClause, gotOK := PackageClause(tt.base, tt.src)
			if gotClause != tt.wantClause || gotOK != tt.wantOK {
				t.Errorf("PackageClause(%q, %q) = (%q, %v); want (%q, %v)", tt.base, tt.src, gotClause, gotOK, tt.wantClause, tt.wantOK)
			}
		})
	}
}

// TestUnitsForClauseMap tables UnitsForClauseMap over map[base name]clause inputs, covering exactly
// the cases the discussion's rejected alternatives name: a plain package; a package with an
// external test file, whose two clauses must produce two distinct units; a package legitimately
// named httptest, asserted not split into a second unit; a tie between two equally common clauses,
// broken lexicographically; a directory in which every clause ends in the test suffix; and the
// repository root, whose unit is the empty string for both branches.
func TestUnitsForClauseMap(t *testing.T) {
	tests := []struct {
		name       string
		dirRel     string
		clauses    map[string]string
		wantDirPkg string
		wantUnits  map[string]string
	}{
		{
			name:   "plain package",
			dirRel: "pkg",
			clauses: map[string]string{
				"a.go": "demo",
				"b.go": "demo",
			},
			wantDirPkg: "demo",
			wantUnits: map[string]string{
				"a.go": "pkg",
				"b.go": "pkg",
			},
		},
		{
			name:   "external test file produces a distinct unit",
			dirRel: "pkg",
			clauses: map[string]string{
				"a.go":      "demo",
				"a_test.go": "demo_test",
			},
			wantDirPkg: "demo",
			wantUnits: map[string]string{
				"a.go":      "pkg",
				"a_test.go": "pkg_test",
			},
		},
		{
			name:   "package legitimately named httptest is never split",
			dirRel: "httptest",
			clauses: map[string]string{
				"a.go": "httptest",
				"b.go": "httptest",
			},
			wantDirPkg: "httptest",
			wantUnits: map[string]string{
				"a.go": "httptest",
				"b.go": "httptest",
			},
		},
		{
			name:   "tie broken lexicographically",
			dirRel: "pkg",
			clauses: map[string]string{
				"a.go": "zed",
				"b.go": "alpha",
			},
			wantDirPkg: "alpha",
			wantUnits: map[string]string{
				"a.go": "pkg",
				"b.go": "pkg",
			},
		},
		{
			name:   "every clause ends in the test suffix",
			dirRel: "pkg",
			clauses: map[string]string{
				"a_test.go": "demo_test",
				"b_test.go": "demo_test",
			},
			wantDirPkg: "demo_test",
			wantUnits: map[string]string{
				// dirPkg is itself "demo_test", so dirPkg+"_test" is "demo_test_test" — neither
				// file's own clause matches it, and both stay in the directory's own unit.
				"a_test.go": "pkg",
				"b_test.go": "pkg",
			},
		},
		{
			name:   "repository root returns the empty unit for both branches",
			dirRel: ".",
			clauses: map[string]string{
				"a.go": "demo",
			},
			wantDirPkg: "demo",
			wantUnits: map[string]string{
				"a.go": "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDirPkg, unitOf := UnitsForClauseMap(tt.dirRel, tt.clauses)
			if gotDirPkg != tt.wantDirPkg {
				t.Errorf("UnitsForClauseMap(%q, %v) dirPkg = %q; want %q", tt.dirRel, tt.clauses, gotDirPkg, tt.wantDirPkg)
			}
			for base, wantUnit := range tt.wantUnits {
				if gotUnit := unitOf(base); gotUnit != wantUnit {
					t.Errorf("unitOf(%q) = %q; want %q", base, gotUnit, wantUnit)
				}
			}
		})
	}
}

// TestUnitsForClauseMap_MissingBaseGetsEmptyClauseUnit asserts a base name absent from clauses maps
// to the unit a file with an empty clause would get, per unitOf's own doc comment.
func TestUnitsForClauseMap_MissingBaseGetsEmptyClauseUnit(t *testing.T) {
	clauses := map[string]string{"a.go": "demo"}
	dirPkg, unitOf := UnitsForClauseMap("pkg", clauses)
	if dirPkg != "demo" {
		t.Fatalf("dirPkg = %q; want %q", dirPkg, "demo")
	}
	want := unitFor("pkg", dirPkg, "")
	if got := unitOf("missing.go"); got != want {
		t.Errorf(`unitOf("missing.go") = %q; want %q (unitFor with an empty clause)`, got, want)
	}
}

// TestClauseMapForFiles builds a directory under writeScratchTree — this package's own scratch-tree
// helper for a fixture that must exist on disk — and asserts: a clause is recorded for each readable
// Go file; a base name naming a file absent from the working tree is skipped with no clause and no
// error; a file whose bytes are not valid UTF-8 is skipped the same way, through the check
// PackageClause performs; and a base name whose extension resolves to no language is skipped
// without contributing a clause.
func TestClauseMapForFiles(t *testing.T) {
	dir := writeScratchTree(t, "units-clause-map-for-files", map[string]string{
		"a.go":      "package demo\n",
		"b.go":      "package demo\n",
		"bad.go":    "package demo\n\xff\xfe",
		"readme.md": "# not a Go file\n",
	})

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", dir, err)
	}

	bases := []string{"a.go", "b.go", "bad.go", "readme.md", "missing.go"}
	got, err := r.ClauseMapForFiles(".", bases)
	if err != nil {
		t.Fatalf("ClauseMapForFiles(\".\", %v) returned error: %v", bases, err)
	}

	want := map[string]string{"a.go": "demo", "b.go": "demo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ClauseMapForFiles(\".\", %v) = %v; want %v", bases, got, want)
	}
}

// TestClauseMapForFiles_UnreadableDirectoryIsAnError asserts the error return is reserved for a
// failure of the call itself, rather than for any per-file condition: a directory that does not
// exist on disk fails the whole call.
func TestClauseMapForFiles_UnreadableDirectoryIsAnError(t *testing.T) {
	dir := writeScratchTree(t, "units-clause-map-for-files-missing-dir", map[string]string{
		"a.go": "package demo\n",
	})

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", dir, err)
	}

	if _, err := r.ClauseMapForFiles("does-not-exist", []string{"a.go"}); err == nil {
		t.Error("ClauseMapForFiles(\"does-not-exist\", ...) returned nil error; want an error for a directory that does not exist")
	}
}
