// errors.go holds the one error sentinel the engine's subpackages share.

package quarryengine

import "errors"

// ErrLanguageUnsupported is returned when a file's extension maps to no language, or when a
// requested language has no toc strategy. Callers wrap it with fmt.Errorf("...: %w",
// ErrLanguageUnsupported) so errors.Is(err, ErrLanguageUnsupported) still succeeds after wrapping.
var ErrLanguageUnsupported = errors.New("quarry: language not supported")
