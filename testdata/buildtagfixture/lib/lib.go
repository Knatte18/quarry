// lib.go declares the one exported symbol the build-tag visibility live test (batch 6, card 24)
// queries references for: a name unlikely to collide with anything else gopls might match
// elsewhere in the workspace.

package lib

// QuarryBuildTagProbe is the symbol under test: consumer/plain.go calls it unconditionally, and
// consumer/tagged.go calls it only under the "sometag" build constraint, so its reference set
// differs depending on whether that tag is active.
func QuarryBuildTagProbe() int {
	return 42
}
