// usage.go declares the CLI's usage text, the one message printed for --help, -h, and every usage
// error's accompanying help.

package cli

// usageText is the CLI's help text, printed on --help, -h, and alongside a usage error. It is
// ASCII only — no em dash, no typographic quotes — so it is stable across terminals and
// byte-comparable in tests. There is no --json flag: JSON is the default, and naming it would
// imply a third format exists.
//
// The flags list stays one combined list rather than one section per verb, with the
// per-verb flags marked as such in their own descriptions, so a flag valid for more than one verb
// is not repeated once per verb. The usage block above it carries the per-verb bracketed lists
// instead, so a reader sees each verb's real shape there; between the two blocks, validity is
// stated once from each direction, which is what makes a per-verb rejection message ("--depth is
// not valid for resolve") legible from this text alone. The block after the flags list, spelling
// the glyphs preset's expansion literally, is neither a verb shape nor a flag: it is the one thing
// a preset verb needs stated that neither of the other two blocks can carry, so it gets a block of
// its own rather than looking like an exception to the two-block rule.
const usageText = `quarry - a table of contents for a source repository

usage:
  quarry toc <target> [--view <name>] [--depth N|all] [--symbols|--no-symbols] [--text] [--root <path>]
  quarry glyphs <target> [--text] [--root <path>]
  quarry resolve <glyph> [--text] [--root <path>]
  quarry expand <glyph> [--text] [--root <path>]
  quarry delta <target> --from <rev> [--to <rev>] [--text] [--root <path>]
  quarry name <declaration> --unit <unit> [--text]

flags:
  --view <name>     toc only: "full" (default), the complete answer, or "glyphs", the flat
                    one-line-per-symbol projection
  --depth <N|all>   toc only: how far to recurse into subdirectories (default 0)
  --symbols         toc only: populate every file entry's symbols
  --no-symbols      toc only: leave every file entry's symbols unpopulated
  --from <rev>      delta only: the before-side revision (required)
  --to <rev>        delta only: the after-side revision (default: the working tree)
  --text            emit the lossless text view instead of JSON
  --root <path>     use <path> as the repository root instead of discovering one, not valid for name
  --unit <unit>     name only: the glyph unit the declaration will belong to
  -h, --help        print this text and exit 0

  quarry glyphs <target> is quarry toc --view glyphs --depth all --symbols <target>. The expansion
  is frozen, so the three query flags (--depth, --symbols/--no-symbols, --view) are not accepted
  on glyphs.

exit codes:
  0  answered
  1  negative answer: not found, outside the repository, ambiguous, not a type,
     not a well-formed glyph, or a declaration that names no single symbol
  2  usage error
  3  internal error
`
