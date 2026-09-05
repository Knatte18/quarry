// usage.go declares the CLI's usage text, the one message printed for --help, -h, and every usage
// error's accompanying help.

package cli

// usageText is the CLI's help text, printed on --help, -h, and alongside a usage error. It is
// ASCII only — no em dash, no typographic quotes — so it is stable across terminals and
// byte-comparable in tests. There is no --json flag: JSON is the default, and naming it would
// imply a third format exists.
//
// The flags list stays one combined list rather than four per-verb sections, with the
// per-verb flags marked as such in their own descriptions, so a flag valid for more than one verb
// is not repeated once per verb. The usage block above it carries the per-verb bracketed lists
// instead, so a reader sees each verb's real shape there; between the two blocks, validity is
// stated once from each direction, which is what makes a per-verb rejection message ("--depth is
// not valid for resolve") legible from this text alone.
const usageText = `quarry - a table of contents for a source repository

usage:
  quarry toc <target> [--depth N|all] [--symbols|--no-symbols] [--text] [--root <path>]
  quarry resolve <glyph> [--text] [--root <path>]
  quarry expand <glyph> [--text] [--root <path>]
  quarry name <declaration> --unit <unit> [--text]

flags:
  --depth <N|all>   toc only: how far to recurse into subdirectories (default 0)
  --symbols         toc only: populate every file entry's symbols
  --no-symbols      toc only: leave every file entry's symbols unpopulated
  --text            emit the lossless text view instead of JSON
  --root <path>     use <path> as the repository root instead of discovering one, not valid for name
  --unit <unit>     name only: the glyph unit the declaration will belong to
  -h, --help        print this text and exit 0

exit codes:
  0  answered
  1  negative answer: not found, outside the repository, ambiguous, not a type,
     not a well-formed glyph, or a declaration that names no single symbol
  2  usage error
  3  internal error
`
