// spaced.go sits under a directory whose name contains a space on purpose: glyph_test.go asserts
// this file is listed with its header and carries no symbols when queried from the quarry root,
// since a space is a rune the Go alphabet's unit rule rejects.

package pkg

// Spaced is listed with a header but never with symbols, since its directory's unit contains a
// disallowed space rune.
func Spaced() {}
