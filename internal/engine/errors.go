// errors.go holds the one error sentinel the engine's subpackages share.

package engine

import "errors"

// ErrLanguageUnsupported is returned by SpansOf when called with a glyph.Glyph whose Lang is not
// glyph.Go. Neither of this sentinel's original triggers survives in the rewrite: toc now lists
// every file regardless of its extension, so an unmapped extension is no longer an error, and there
// is no per-query language override any more, so a language can no longer be requested and refused.
// SpansOf's caller is the one place this can still fire, because a glyph.Glyph is a plain struct
// any caller can build by hand with any Language value, and the engine needs a defined answer for a
// language it has no extractor for. Callers wrap it with fmt.Errorf("...: %w",
// ErrLanguageUnsupported) so errors.Is(err, ErrLanguageUnsupported) still succeeds after wrapping.
var ErrLanguageUnsupported = errors.New("quarry: language not supported")
