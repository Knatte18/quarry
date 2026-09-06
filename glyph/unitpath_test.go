// unitpath_test.go is the executable form of the path direction's contract: what UnitPath answers
// for a member glyph, a unit-self glyph, a file-self glyph and the zero Glyph, and that it is the
// exact inverse of Self for a Go path.

package glyph

import "testing"

// unitPathCases holds one row per contract case: the Glyph to ask, and the path and ok it must
// answer. The three Go rows are the parsed forms of "internal/x#Foo.Bar", "internal/x#" and
// "internal/x/focus.go#"; the last row is the zero Glyph, whose Lang is not a valid language.
var unitPathCases = []struct {
	name     string
	glyph    Glyph
	wantPath string
	wantOK   bool
}{
	{
		name:     "Go member glyph yields the package directory",
		glyph:    Glyph{Lang: Go, Unit: "internal/x", Owner: []string{"Foo"}, Name: "Bar"},
		wantPath: "internal/x",
		wantOK:   true,
	},
	{
		name:     "Go unit-self glyph yields the package directory",
		glyph:    Glyph{Lang: Go, Unit: "internal/x"},
		wantPath: "internal/x",
		wantOK:   true,
	},
	{
		name:     "Go file-self glyph yields the file",
		glyph:    Glyph{Lang: Go, Unit: "internal/x/focus.go"},
		wantPath: "internal/x/focus.go",
		wantOK:   true,
	},
	{
		name:     "zero Glyph has no known alphabet",
		glyph:    Glyph{},
		wantPath: "",
		wantOK:   false,
	},
}

// TestUnitPath drives unitPathCases, asserting both returns of UnitPath for each row.
func TestUnitPath(t *testing.T) {
	for _, c := range unitPathCases {
		t.Run(c.name, func(t *testing.T) {
			path, ok := c.glyph.UnitPath()
			if path != c.wantPath || ok != c.wantOK {
				t.Errorf("Glyph%+v.UnitPath() = (%q, %t); want (%q, %t)", c.glyph, path, ok, c.wantPath, c.wantOK)
			}
		})
	}
}

// TestUnitPath_SelfRoundTrip asserts the property §3 documents and UnitPath exposes: for every path
// Self accepts, UnitPath on the composed Glyph returns that same path byte-identically. It reuses
// selfComposePaths so the two directions are proven over one set of paths rather than two.
func TestUnitPath_SelfRoundTrip(t *testing.T) {
	for _, p := range selfComposePaths {
		t.Run(p, func(t *testing.T) {
			composed, err := Self(Go, p)
			if err != nil {
				t.Fatalf("Self(Go, %q) error = %v; want nil", p, err)
			}
			path, ok := composed.UnitPath()
			if !ok {
				t.Fatalf("Self(Go, %q).UnitPath() ok = false; want true", p)
			}
			if path != p {
				t.Errorf("Self(Go, %q).UnitPath() path = %q; want %q", p, path, p)
			}
		})
	}
}
