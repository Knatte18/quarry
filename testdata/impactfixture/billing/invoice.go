// invoice.go is an impact fixture: it declares an Invoice type with a documented method and a
// package-level var, so the impact verb's tests have both a resolvable declaration site and a
// file-scope target with no toc.Kind vocabulary of its own.

package billing

// Invoice represents a single customer invoice awaiting payment.
type Invoice struct {
	Amount float64
}

// DefaultRate is the discount rate applied when a caller does not specify one.
// It is a package-level var, not a function or type, so it is the file-scope target that has no
// toc.Kind vocabulary of its own.
var DefaultRate = 0.05

// ApplyDiscount reduces the invoice's amount by rate, a fraction between 0 and 1.
// It mutates the invoice in place and is the impact verb's canonical resolution target across
// this fixture tree.
func (inv *Invoice) ApplyDiscount(rate float64) {
	inv.Amount -= inv.Amount * rate
}
