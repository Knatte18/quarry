// errors.go holds the one error sentinel the engine's subpackages share.

package engine

import "errors"

// ErrLanguageUnsupported is returned by SpansOf when called with a glyph.Glyph whose Lang is neither glyph.Go nor glyph.CSharp.
// A glyph.Glyph is a plain struct any caller can build by hand with any Language value, and the engine needs a defined answer for a language it has no extractor for.
// Callers wrap it with fmt.Errorf("...: %w", ErrLanguageUnsupported), so errors.Is(err, ErrLanguageUnsupported) still succeeds after wrapping.
var ErrLanguageUnsupported = errors.New("quarry: language not supported")
