// usage.go declares the CLI's usage text, the one message printed for --help, -h, and every usage
// error's accompanying help.

package cli

// usageText is the CLI's help text, printed on --help, -h, and alongside a usage error. It is
// ASCII only — no em dash, no typographic quotes — so it is stable across terminals and
// byte-comparable in tests. There is no --json flag: JSON is the default, and naming it would
// imply a third format exists.
const usageText = `quarry - a table of contents for a source repository

usage:
  quarry toc <target> [flags]

flags:
  --depth <N|all>   how far to recurse into subdirectories (default 0)
  --symbols         populate every file entry's symbols
  --no-symbols      leave every file entry's symbols unpopulated
  --text            emit the lossless text view instead of JSON
  --root <path>     use <path> as the repository root instead of discovering one
  -h, --help        print this text and exit 0

exit codes:
  0  answered
  1  negative answer: target not found, or target outside the repository
  2  usage error
  3  internal error
`
