package pkg_test

import "testing"

// TestExported is a fixture external test, deliberately declaring the "_test" suffixed package
// deviation this directory's tie-break and per-file Package emission must handle.
func TestExported(t *testing.T) {
}
