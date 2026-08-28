// tick.go declares its own clock interface, structurally identical to builder's but otherwise
// unrelated, so the implementation-widening spike can measure whether gopls'
// textDocument/implementation crosses package boundaries between structurally identical but
// unrelated interfaces.

package runner

import "time"

// clock is a structural duplicate of builder's clock interface, declared independently.
type clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// realClock is runner's own concrete clock implementation.
type realClock struct{}

// Now implements clock by delegating to time.Now.
func (realClock) Now() time.Time {
	return time.Now()
}

// Sleep implements clock by delegating to time.Sleep.
func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Tick calls c.Now and c.Sleep through the clock interface.
func Tick(c clock) time.Time {
	c.Sleep(time.Millisecond)
	return c.Now()
}

// Run constructs a realClock and drives it through both call shapes.
func Run() time.Time {
	c := realClock{}
	Tick(c)
	return c.Now()
}
