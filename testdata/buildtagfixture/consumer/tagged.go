//go:build sometag

// tagged.go is excluded from compilation unless the "sometag" build tag is active, so its call
// site is the reference that must be invisible without --build-tags and visible with it.

package consumer

import "buildtagfixture/lib"

// CallTagged calls lib.QuarryBuildTagProbe only when the "sometag" build tag is active.
func CallTagged() int {
	return lib.QuarryBuildTagProbe()
}
