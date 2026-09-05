// Package quarry is the public Go facade over the extraction engine: the primary surface named by
// docs/rewrite-plan.md §7 item 2. It exists because internal/engine cannot be imported from outside
// this module, and this package is what lets Loomyard's own Go code, or any other importer, reach
// the engine's typed results without a JSON round-trip.
//
// The engine's types, plus this package's one projected answer type, are what a caller reaches
// through these queries: every type reached through TOC, Resolve, Expand and Name is an alias for
// an engine type, and those four queries add no filtering, re-shaping or defaulting of their own —
// which is why the aliases work at all. The package exposes five queries, not four: TOC, Resolve,
// Expand, Name and Glyphs. TOC, Resolve, Expand and Glyphs are methods; Name is a package-level
// function, not a method, because the maker performs no I/O and needs no repository receiver —
// "queries" is the word that covers both shapes. Glyphs is a method for the same reason TOC is —
// it reads the repository — but unlike the other three method queries it does not delegate to the
// engine unchanged: it is TOC under frozen options followed by a pure projection, GlyphView, which
// is the one place this package adds behaviour of its own rather than only re-shaping.
//
// The package owns eleven renderers and the glyphs view's own projection, GlyphView, glyphSymbol
// and glyphsEnvelope: the two existing JSON renderers, RenderJSON and RenderErrorJSON; the four new
// JSON renderers, RenderResolveJSON, RenderExpandJSON, RenderNameJSON and RenderGlyphsJSON; and the
// five text renderers, RenderText, RenderResolveText, RenderExpandText, RenderNameText and
// RenderGlyphsText. The five JSON success renderers — RenderJSON, RenderResolveJSON,
// RenderExpandJSON, RenderNameJSON and RenderGlyphsJSON — share one encoder configuration, so their
// two-space indent, one-trailing-newline, no-HTML-escaping byte contract cannot drift between them.
// RenderErrorJSON is deliberately not part of that sharing: it emits a different, compact byte
// contract for the failure envelope. RenderGlyphsText states its own byte contract rather than the
// shared one RenderText, RenderResolveText, RenderExpandText and RenderNameText follow, because an
// empty glyphs answer renders as the empty string, a shape those four renderers never produce.
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
