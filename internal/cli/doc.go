// Package cli is the whole of the quarry command below os.Exit: flag parsing, calling
// internal/repopath to resolve a repository root and a cwd-relative target, the exit-code
// mapping, and the choice of renderer. cmd/quarry holds one line and nothing else, and this split
// exists so the golden tests can capture exactly the bytes the binary emits without building or
// exec'ing anything.
//
// This package is the only layer with a working directory — internal/engine deliberately
// performs no git discovery and no cwd resolution — and the two path frames never mix: input is
// interpreted where the user is, except for the paths in enclose locations, which are repository-root
// relative; output is always repository-root relative with forward slashes.
//
// Every verb has a pipeline of its own except "glyphs". "toc" takes a repository-relative or
// cwd-relative path — a directory or a file. "glyphs" takes the same repository-relative or
// cwd-relative path "toc" takes, but it has no pipeline of its own: parseArgs rewrites it to a frozen
// "toc" expansion (--view glyphs --depth all --symbols), so it reaches exactly the same parsing,
// the same dispatch and the same renderers, and nothing below the parser knows the verb "glyphs"
// ever existed. "resolve" takes a glyph only, including a self glyph naming a whole unit. "expand"
// takes a glyph only. "delta" takes a path target and two revisions, and reports the symbol-table
// difference between the two versions of that target the two revisions name. "name" takes a
// declaration head, which is neither a path nor a glyph. "enclose" takes path:line locations and
// reports the members enclosing each, at a revision or in the working tree.
//
// A target is handed to the facade verbatim, with no path arithmetic and no stat ever applied to
// it, whenever the verb does not take a path. This is because a glyph's unit is repository-relative
// by the grammar's own definition — cwd arithmetic on it would corrupt it, the same way rebasing a
// remote URL against a local directory would — and because a declaration head is neither a path nor
// a glyph, so no path arithmetic applies to it either. The verbs that take a single path target
// are the ones this package converts with internal/repopath before the engine sees the target,
// including "glyphs", which is rewritten to "toc" before parseArgs ever returns. "enclose" takes
// locations whose paths the facade resolves against the repository root, so this package applies
// no cwd-relative conversion to them.
//
// The failure envelope's "ok" key, present only on the failure path and always false there, marks
// that quarry could not answer the query at all — a usage error, an internal error, or a
// grammar-rejected target passed to "expand". It never marks that the answer is negative: a
// negative resolution outcome — not_found, ambiguous, or "resolve"'s own pre-resolution rejection
// of a target string — is a payload carrying a status word (or, for the pre-resolution case, an
// error field of its own, and for "name"'s own rejection, an error and a reason field carrying no
// status word at all), rendered and written to stdout exactly as a positive answer is. Reading the
// "ok" key's presence, not the exit code alone, is what tells a caller which of the two shapes it
// is holding.
//
// Classification happens exactly once, and it is glyph.Parse doing it: no surface in this
// package, or in internal/repopath, or in the engine, tests a target for "#" containment to decide
// whether it is a path or a glyph. "resolve" and "expand" hand their target to the facade
// unclassified, and glyph.Parse either accepts it as a glyph or rejects it with ReasonNoSeparator,
// exactly as it would any other malformed input. "toc" and "delta" take a path only, and a "#" in
// any segment of that path is an explicit error for both — internal/repopath.RepoRelTarget returns
// quarry.ErrTargetHasSeparator rather than reclassifying the target as a glyph. The grammar's
// separator rule holds everywhere a target is taken: at the verbs that read one as a glyph, and at
// the verbs that take a path instead. "enclose" answers a "#" in a path segment as an
// "unaddressable" item, not as a usage error.
package cli
