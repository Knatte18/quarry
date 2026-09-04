// iface.go carries the interface shapes glyph_test.go's interface-walk coverage exercises: a named
// interface with two documented methods, an interface embedded by name, and three positions where
// an anonymous interface_type must never contribute a method symbol — a struct field, a function
// parameter, and a generic constraint.

package glyphs

// Iface is a named interface with two methods and one embedded interface.
type Iface interface {
	// M1 does the first thing.
	M1(x int) error
	M2()
	Embedded
}

// Embedded is embedded by Iface. Its own method is never treated as a member of Iface.
type Embedded interface {
	E()
}

// HasAnonField holds an anonymous interface field, which contributes no method symbol: struct
// fields are excluded from the identifier contract outright, and the field has no type name to own
// a method anyway.
type HasAnonField struct {
	Field interface{ AnonField() }
}

// AnonParam takes an anonymous interface parameter, which contributes no method symbol.
func AnonParam(p interface{ AnonParam() }) {}

// GenericConstraint takes a type parameter constrained by an anonymous interface, which
// contributes no method symbol.
func GenericConstraint[T interface{ AnonConstraint() }]() {}
