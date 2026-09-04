// widget.go declares the type whose members the expand verb's cross-file member test collects. It
// sorts after aardvark.go deliberately: the members declared there must still come back first.

package methods

// Widget is a fixture type with three methods, two of them in a sibling file.
type Widget struct {
	// Name is a fixture field. Struct fields are not part of the identifier contract and
	// contribute no symbol.
	Name string
}

// Zeta is Widget's method declared in the same file as the type itself.
func (w *Widget) Zeta() string {
	return w.Name
}
