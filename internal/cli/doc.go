// Package cli is the whole of the quarry command below os.Exit: flag parsing, repository-root
// discovery, cwd-relative target interpretation, the exit-code mapping, and the choice of
// renderer. cmd/quarry holds one line and nothing else, and this split exists so the golden
// tests can capture exactly the bytes the binary emits without building or exec'ing anything.
//
// This package is the only layer with a working directory — internal/engine deliberately
// performs no git discovery and no cwd resolution — and the two path frames never mix: input is
// interpreted where the user is, output is always repository-root relative with forward slashes.
//
// The command has three verbs. "toc" takes a repository-relative or cwd-relative path — a
// directory or a file. "resolve" takes either a path, by that same rule, or a glyph. "expand"
// takes a glyph only.
//
// A target containing a "#" is classified as a glyph by this package, on sight, by
// strings.Contains and nothing else, and is handed to the facade verbatim: no path arithmetic and
// no stat is ever applied to it. This is because a glyph's unit is repository-relative by the
// grammar's own definition — cwd arithmetic on it would corrupt it, the same way rebasing a
// remote URL against a local directory would.
//
// The failure envelope's "ok" key, present only on the failure path and always false there, marks
// that quarry could not answer the query at all — a usage error, an internal error, or a
// grammar-rejected target passed to "expand". It never marks that the answer is negative: a
// negative resolution outcome — not_found, ambiguous, or "resolve"'s own pre-resolution rejection
// of a target string — is a payload carrying a status word (or, for the pre-resolution case, an
// error field of its own), rendered and written to stdout exactly as a positive answer is. Reading the
// "ok" key's presence, not the exit code alone, is what tells a caller which of the two shapes it
// is holding.
//
// Known contract gap, recorded in the same style the engine records its own: this package
// classifies the argument as given — by "#" containment, before any path arithmetic runs — but
// hands the engine the repository-relative form of a path target, and the engine classifies that
// string again, by the same "#" containment rule, once it reaches resolveTarget. The two
// classifications disagree in exactly one case: a path target whose repository-relative form
// acquires a "#" from a directory name somewhere in its chain. There the caller asked for a path
// answer and gets a glyph grammar rejection instead, naming the rejection's reason rather than the
// path it named. Closing this needs the target's class to travel with it from this package down
// into the engine, which means a new engine signature, and that is out of this task's scope. A "#"
// in a directory name is legal on every filesystem this command runs on and pathological in
// practice; no repository quarry is measured against contains one. The gap is deliberately not
// pinned by a test: a test would pin behaviour this task considers wrong but unfixable at this
// layer, and the fix belongs to whichever task next touches the engine's target-resolution
// signature.
package cli
