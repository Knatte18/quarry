// Package quarry is the public Go facade over the extraction engine: the primary surface named by
// docs/rewrite-plan.md §7 item 2. It exists because internal/engine cannot be imported from outside
// this module, and this package is what lets Loomyard's own Go code, or any other importer, reach
// the engine's typed results without a JSON round-trip.
//
// The engine's types, plus this package's own projected and convenience answer types, are what a
// caller reaches through these queries: every type reached through TOC, Resolve, Expand, Delta,
// Name and Enclose is an alias for an engine type, and those queries add no filtering, re-shaping or
// defaulting to the answer itself — which is why the aliases work at all. The queries are methods on Repo,
// except Name, which is a package-level function because the maker performs no I/O and needs no
// repository receiver — "queries" is the word that covers both shapes. TOC, Resolve, Expand and
// Delta delegate to the engine unchanged, and Name keeps that same posture. Enclose also hands its
// answer type back from the engine, but first parses the location spelling, normalises paths and
// applies the four file-free rejection checks, so it is another place this package adds behaviour
// of its own. Glyphs is a method
// for the same reason TOC is — it reads the repository — but it does not delegate to the engine
// unchanged: it is TOC under frozen options followed by a pure projection, GlyphView, which is one
// place this package adds behaviour of its own rather than only re-shaping. The git-backed
// convenience methods are the other: caller-facing conveniences over the git layer, built on top of
// a query rather than being queries themselves, and they exist because putting them only in the
// command line would force the primary Go consumer (Loomyard's own pipeline) to reimplement the one
// thing that layer exists to hold.
//
// The package owns the renderers and the glyphs view's own projection, GlyphView, glyphSymbol and
// glyphsEnvelope. Every JSON success renderer shares one encoder configuration, so its two-space
// indent, one-trailing-newline, no-HTML-escaping byte contract cannot drift between them.
// RenderErrorJSON is deliberately not part of that sharing: it emits a different, compact byte
// contract for the failure envelope. RenderGlyphsText states its own byte contract rather than the
// shared one the other text renderers follow, because an empty glyphs answer renders as the empty
// string, a shape those renderers never produce.
//
// The failure envelope's "ok" key marks that quarry could not answer at all, and never that the
// answer is negative: a negative resolution outcome — not_found, ambiguous, or a resolve result
// carrying a pre-resolution error and reason — is a payload with a status word, rendered by the
// ordinary renderer, not the failure envelope. The maker's rejection is the same kind of negative
// answer without a status word at all: a payload rendered by the ordinary renderer, carrying only an
// error and a reason, and never the failure envelope.
//
// Per docs/rewrite-plan.md §10's phase-1 non-goals, this package holds no cache, no parser pool, and
// no state beyond the repository root it was opened with.
package quarry
