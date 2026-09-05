// render.go declares the JSON renderers the facade exports: RenderJSON, RenderResolveJSON,
// RenderExpandJSON and RenderDeltaJSON, the four successful envelopes, sharing one unexported
// encoder configuration in renderJSON, and RenderErrorJSON, the failure envelope. All are
// package-level functions rather than methods, per the overview's alias-types-carry-no-methods
// decision — DirAnswer, ResolveResult, ExpandAnswer and GitDeltaAnswer are aliases for engine types
// or, for GitDeltaAnswer, a facade type embedding one, and Go forbids a method declared here from
// binding to any of them.

package quarry

import (
	"bytes"
	"encoding/json"
)

// errorEnvelope is the failure envelope's wire shape: OK and Error, and nothing else — no kind, no
// status, because the process exit code already discriminates which failure this is.
type errorEnvelope struct {
	// OK is never set, so it always marshals as false. The field exists so the key is present at
	// all on the failure path; RenderJSON's DirAnswer has no such field, so the key is absent
	// there, which is what lets a caller tell success from failure by this key's presence alone.
	OK bool `json:"ok"`
	// Error is the human-readable failure message this envelope carries.
	Error string `json:"error"`
}

// renderJSON encodes v into the wire form docs/rewrite-plan.md §4 fixes: two-space indentation, one
// trailing newline, and no other bytes. HTML escaping is disabled because headers, package docs and
// signatures are real prose that can legitimately contain '<', '>' and '&' — Go's default encoder
// would rewrite '<' as `<` and make the output both unreadable and unequal to §4's own examples.
// Key order within the object is the struct field declaration order of the type being encoded, so no
// hand-written marshaller is needed here or anywhere this helper is used. renderJSON is the one
// place this encoder configuration is built; every exported success renderer in this file delegates
// to it so the byte contract cannot drift between them.
func renderJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encode already appends exactly one trailing newline; buf carries no other bytes.
	return buf.Bytes(), nil
}

// RenderJSON encodes a, a table-of-contents answer, as a successful JSON envelope. See renderJSON
// for the byte contract this and every other success renderer in this file share.
func RenderJSON(a DirAnswer) ([]byte, error) {
	return renderJSON(a)
}

// RenderResolveJSON encodes r, a resolve answer, as a successful JSON envelope. It emits the same
// byte contract as RenderJSON — two-space indent, one trailing newline, no HTML escaping — and its
// key order within the object is ResolveResult's own field declaration order, so no hand-written
// marshaller is needed.
func RenderResolveJSON(r ResolveResult) ([]byte, error) {
	return renderJSON(r)
}

// RenderExpandJSON encodes a, an expand answer, as a successful JSON envelope. It emits the same
// byte contract as RenderJSON — two-space indent, one trailing newline, no HTML escaping — and its
// key order within the object is ExpandAnswer's own field declaration order, so no hand-written
// marshaller is needed.
func RenderExpandJSON(a ExpandAnswer) ([]byte, error) {
	return renderJSON(a)
}

// RenderDeltaJSON encodes a, the git-wrapped delta answer, as a successful JSON envelope. It emits
// the same byte contract as RenderJSON — two-space indent, one trailing newline, no HTML escaping —
// and its key order within the object is GitDeltaAnswer's own field declaration order (from, to, then
// the embedded DeltaAnswer's own fields), so no hand-written marshaller is needed. RenderDeltaJSON
// never emits the failure envelope's "ok" marker key: that key is present only on the failure path,
// and an empty delta is a successful answer rather than a negative one.
func RenderDeltaJSON(a GitDeltaAnswer) ([]byte, error) {
	return renderJSON(a)
}

// RenderErrorJSON encodes msg as the compact failure envelope {"ok":false,"error":"<msg>"} followed
// by one newline, with no space after either colon — see the overview's
// json-encoder-spacing-is-the-byte-contract. ok is present only on this path, so it can never
// disagree with the exit code that always accompanies it; RenderErrorJSON reports no error itself,
// because a struct of one bool and one string cannot fail to marshal and a bytes.Buffer write cannot
// fail. The unreachable branch below exists only so stdout always carries a parseable object rather
// than nothing, if that assumption is ever wrong.
func RenderErrorJSON(msg string) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(errorEnvelope{Error: msg}); err != nil {
		// Unreachable: see the doc comment above.
		return []byte(`{"ok":false,"error":"internal error: failed to render error envelope"}` + "\n")
	}
	return buf.Bytes()
}
