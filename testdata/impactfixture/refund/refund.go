// refund.go is an impact fixture: it declares two exported functions that call
// billing.Invoice.ApplyDiscount from three distinct call sites across two enclosing
// declarations, so the impact verb's tests have a caller set spanning more than one file and more
// than one enclosing symbol.

package refund

import "impactfixture/billing"

// ProcessRefund applies the default discount to inv twice, on two distinct lines, so the impact
// verb's tests can prove one caller entry is emitted per call site rather than per enclosing
// declaration.
func ProcessRefund(inv *billing.Invoice) {
	inv.ApplyDiscount(billing.DefaultRate)
	inv.ApplyDiscount(billing.DefaultRate)
}

// Reconcile applies the default discount to inv once, so the caller set spans two enclosing
// declarations within this one file.
func Reconcile(inv *billing.Invoice) {
	inv.ApplyDiscount(billing.DefaultRate)
}
