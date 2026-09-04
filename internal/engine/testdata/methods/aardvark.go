// aardvark.go declares two of Widget's methods, in the file that sorts first in this package.

package methods

// Alpha is Widget's value-receiver method.
func (w Widget) Alpha() int {
	return len(w.Name)
}

// Beta is Widget's pointer-receiver method, declared after Alpha in this same file so the
// within-file start-line order is asserted alongside the across-file order.
func (w *Widget) Beta() {
	w.Name = ""
}
