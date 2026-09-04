// literal.go sits in a directory literally named "foo_test/", not a "_test" sibling of some "foo/"
// directory that need not exist. resolve_test.go's literal-first cases assert that a glyph naming
// this directory as its unit finds LiteralDeclaration here, rather than one reached by stripping
// the "_test" suffix and looking in a "foo/" directory this repository does not have.

package foo_test

// LiteralDeclaration is the one declaration this fixture exists to be found by unitDirs' literal
// branch.
func LiteralDeclaration() {}
