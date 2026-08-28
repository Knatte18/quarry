// wait.go declares its own clock interface, structurally identical to builder's and runner's but
// otherwise unrelated, completing the three-package structural-duplicate fixture the
// implementation-widening spike measures against.

package sched

import "time"

// clock is a structural duplicate of builder's and runner's clock interfaces, declared
// independently.
type clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// realClock is sched's own concrete clock implementation.
type realClock struct{}

// Now implements clock by delegating to time.Now.
func (realClock) Now() time.Time {
	return time.Now()
}

// Sleep implements clock by delegating to time.Sleep.
func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Wait calls c.Now and c.Sleep through the clock interface.
func Wait(c clock) time.Time {
	c.Sleep(time.Millisecond)
	return c.Now()
}

// Run constructs a realClock and drives it through both call shapes.
func Run() time.Time {
	c := realClock{}
	Wait(c)
	return c.Now()
}
