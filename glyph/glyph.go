// glyph.go declares Language, Glyph, and the total String printer that turns a Glyph back into its
// canonical string form.

package glyph

import "strings"

// Language is the closed set of languages a Glyph can name a symbol in. The zero value is
// deliberately not a valid language, so a forgotten argument is an error at the first call rather
// than a silent Go.
type Language string

// Go is the only Language this package defines today.
const Go Language = "go"

// Glyph is the parsed form of a glyph string: a symbol's unit and member, split into their
// components. A Glyph built by hand rather than by Parse is the builder's responsibility — this
// package exports no constructor and no Validate method, and String does not check the value it is
// given.
type Glyph struct {
	// Lang is the language whose alphabet this Glyph's Unit and member were parsed against.
	Lang Language
	// Unit is the unit half of the glyph: the Go package directory, the Python module, or the C#
	// namespace, per docs/glyph.md §2.
	Unit string
	// Owner is the enclosing type chain, outermost first, and is nil for a package-level Go name.
	Owner []string
	// Name is the symbol's own name.
	Name string
	// Params is nil for Go always. Its nil-versus-non-nil state, not its length, is what decides
	// whether String prints parentheses: a non-nil empty slice prints "()".
	Params []string
}

// IsSelf reports whether g is the self form: a glyph naming its unit's whole directory or file
// rather than a member within it. All three of Owner, Name and Params are tested, not merely Owner
// and Name, because String prints "()" for a non-nil empty Params: its nil-versus-non-nil state,
// not its length, decides the parentheses, so a hand-built Glyph{Unit: "a", Params: []string{}}
// would report true here while printing "a#()", breaking the property that removing the trailing
// "#" yields the plain path.
func (g Glyph) IsSelf() bool {
	return len(g.Owner) == 0 && g.Name == "" && g.Params == nil
}

// String is a total pure printer: it never returns an error, never panics, and never validates. It
// prints Unit, "#", the Owner chain and Name joined by ".", and Params in parentheses when Params
// is non-nil — including when Params is a non-nil empty slice, which prints "()".
func (g Glyph) String() string {
	// Build parts with a fresh allocation rather than append(g.Owner, g.Name): the latter can write
	// into the caller's backing array and corrupt a Glyph the caller still holds.
	parts := make([]string, 0, len(g.Owner)+1)
	parts = append(parts, g.Owner...)
	parts = append(parts, g.Name)

	params := ""
	if g.Params != nil {
		params = "(" + strings.Join(g.Params, ",") + ")"
	}

	return g.Unit + "#" + strings.Join(parts, ".") + params
}
