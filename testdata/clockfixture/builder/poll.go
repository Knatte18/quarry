// poll.go is a query fixture: it declares a clock interface with an interface-dispatch call
// site and a concrete call site on its own implementer, so the implementation-widening spike
// (internal/quarryengine/query/implementation_spike_lsp_test.go) has both call shapes to measure
// against.

package builder

import "time"

// clock abstracts time.Now and time.Sleep so Poll can be driven by a fake in a real test suite.
// It is unexported: this fixture exists only to be queried by gopls, never imported.
type clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// realClock is the production clock implementation, delegating to the time package.
type realClock struct{}

// Now implements clock by delegating to time.Now.
func (realClock) Now() time.Time {
	return time.Now()
}

// Sleep implements clock by delegating to time.Sleep.
func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Poll calls c.Now and c.Sleep through the clock interface — the interface-dispatch call site.
func Poll(c clock) time.Time {
	c.Sleep(time.Millisecond)
	return c.Now()
}

// Run constructs a realClock, passes it to Poll (interface dispatch), and also calls Now
// directly on the concrete realClock value — the concrete call site.
func Run() time.Time {
	c := realClock{}
	Poll(c)
	return c.Now()
}
