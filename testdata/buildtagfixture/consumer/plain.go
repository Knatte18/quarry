// plain.go carries no build constraint, so it is always compiled — its call site is the reference
// that must be visible with or without the fixture's "sometag" build tag active.

package consumer

import "buildtagfixture/lib"

// CallPlain calls lib.QuarryBuildTagProbe unconditionally.
func CallPlain() int {
	return lib.QuarryBuildTagProbe()
}
