// blank.go carries the three blank-identifier declaration shapes glyph_test.go asserts are never
// listed: a blank var initializer, a blank interface-assertion var, and a blank type declaration.

package glyphs

// BlankIface is asserted against by the blank var below.
type BlankIface interface {
	M()
}

// BlankImpl implements BlankIface, asserted by the blank var below.
type BlankImpl struct{}

// M implements BlankIface.
func (BlankImpl) M() {}

var _ = 1

var _ BlankIface = (*BlankImpl)(nil)

type _ struct{}
