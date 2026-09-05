// Package quarry is the public Go facade over the extraction engine: the primary surface named by
// docs/rewrite-plan.md §7 item 2. It exists because internal/engine cannot be imported from outside
// this module, and this package is what lets Loomyard's own Go code, or any other importer, reach
// the engine's typed results without a JSON round-trip.
//
// The package adds no behaviour of its own, with one exception: DeltaGit. The answer types are
// aliases for the engine's own types, and the package exposes four query methods, not one: TOC,
// Resolve, Expand and Delta. Delta delegates to the engine unchanged, exactly as TOC does — no
// filtering, no re-shaping, no defaulting. DeltaGit is the exception: it is a caller-facing
// convenience over the git layer, not query behaviour, and it exists because putting it only in the
// command line would force the primary Go consumer (Loomyard's own pipeline) to reimplement the one
// thing that layer exists to hold.
//
// The package owns nine renderers, the only code it owns beyond the query methods themselves: the
// two existing JSON renderers, RenderJSON and RenderErrorJSON; the two JSON renderers RenderResolveJSON
// and RenderExpandJSON; the two new JSON and text renderers RenderDeltaJSON and RenderDeltaText,
// arriving with the delta verb; and the three text renderers, RenderText, RenderResolveText and
// RenderExpandText. The four JSON success renderers — RenderJSON, RenderResolveJSON,
// RenderExpandJSON and RenderDeltaJSON — share one encoder configuration, so their two-space indent,
// one-trailing-newline, no-HTML-escaping byte contract cannot drift between them. RenderErrorJSON is
// deliberately not part of that sharing: it emits a different, compact byte contract for the failure
// envelope.
//
// The failure envelope's "ok" key marks that quarry could not answer at all, and never that the
// answer is negative: a negative resolution outcome — not_found, ambiguous, or a resolve result
// carrying a pre-resolution error and reason — is a payload with a status word, rendered by the
// ordinary renderer, not the failure envelope.
//
// Per docs/rewrite-plan.md §10's phase-1 non-goals, this package holds no cache, no parser pool, and
// no state beyond the repository root it was opened with.
package quarry
