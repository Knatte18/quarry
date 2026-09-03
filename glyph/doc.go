// Package glyph names one symbol in a repository: the form is unit "#" member, one name for one
// source symbol, as docs/glyph.md defines it. This package is pure Go, standard library only, no
// cgo and no dependencies, so any program can import it without the engine.
//
// Parse is the syntactic check: it reads no source and accepts exactly the alphabet of the
// language it is given. String is a total pure printer: it never validates the Glyph it is given
// and never returns an error. Go is the only Language this package implements today.
package glyph
