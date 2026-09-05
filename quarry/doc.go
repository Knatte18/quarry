// Package quarry is the public Go facade over the extraction engine: the primary surface named by
// docs/rewrite-plan.md §7 item 2. It exists because internal/engine cannot be imported from outside
// this module, and this package is what lets Loomyard's own Go code, or any other importer, reach
// the engine's typed results without a JSON round-trip.
//
// The package adds no behaviour of its own. The answer types are aliases for the engine's own
// types, and the package exposes four queries, not one: TOC, Resolve, Expand and Name. TOC, Resolve
// and Expand are methods; Name is a package-level function, not a method, because the maker performs
// no I/O and needs no repository receiver — "queries" is the word that covers both shapes. The three
// method queries delegate to the engine unchanged, exactly as TOC does — no filtering, no re-shaping,
// no defaulting — and Name keeps that same posture.
//
// The package owns nine renderers, the only code it owns beyond the queries themselves: the two
// existing JSON renderers, RenderJSON and RenderErrorJSON; the three new JSON renderers,
// RenderResolveJSON, RenderExpandJSON and RenderNameJSON; and the four text renderers, RenderText,
// RenderResolveText, RenderExpandText and RenderNameText. The four JSON success renderers —
// RenderJSON, RenderResolveJSON, RenderExpandJSON and RenderNameJSON — share one encoder
// configuration, so their two-space indent, one-trailing-newline, no-HTML-escaping byte contract
// cannot drift between them. RenderErrorJSON is deliberately not part of that sharing: it emits a
// different, compact byte contract for the failure envelope.
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
