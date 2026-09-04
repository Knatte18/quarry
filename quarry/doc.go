// Package quarry is the public Go facade over the extraction engine: the primary surface named by
// docs/rewrite-plan.md §7 item 2. It exists because internal/engine cannot be imported from outside
// this module, and this package is what lets Loomyard's own Go code, or any other importer, reach
// the engine's typed results without a JSON round-trip.
//
// The package adds no behaviour of its own. The answer types are aliases for the engine's own
// types, and the query methods delegate to the engine unchanged. The only code this package owns is
// the two renderers, RenderJSON and RenderErrorJSON, and the text renderer RenderText — added in
// batch 2 — which turn a typed answer into the CLI's output bytes.
//
// Per docs/rewrite-plan.md §10's phase-1 non-goals, this package holds no cache, no parser pool, and
// no state beyond the repository root it was opened with.
package quarry
