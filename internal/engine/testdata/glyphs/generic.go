// generic.go carries a generic type with both a value-receiver and a pointer-receiver method, for
// glyph_test.go's assertion that a generic receiver's owner is the bare type name, never the
// parameterised one.

package glyphs

// Box is a generic container.
type Box[T any] struct {
	v T
}

// Get returns the boxed value.
func (b Box[T]) Get() T { return b.v }

// Set stores v in the box.
func (b *Box[T]) Set(v T) { b.v = v }
