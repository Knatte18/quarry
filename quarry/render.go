// render.go declares the two JSON renderers the facade exports: RenderJSON, the successful
// envelope, and RenderErrorJSON, the failure envelope. Both are package-level functions rather than
// methods, per the overview's alias-types-carry-no-methods decision — DirAnswer is an alias for an
// engine type, and Go forbids a method declared here from binding to it.

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

// RenderJSON encodes a into the wire form docs/rewrite-plan.md §4 fixes: two-space indentation, one
// trailing newline, and no other bytes. HTML escaping is disabled because headers and package docs
// are real prose that can legitimately contain '<', '>' and '&' — Go's default encoder would rewrite
// '<' as `<` and make the output both unreadable and unequal to §4's own examples. Key order
// within the object is the struct field declaration order of internal/engine/answer.go, which is
// already §4's order, so no hand-written marshaller is needed here or anywhere this type is encoded.
func RenderJSON(a DirAnswer) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a); err != nil {
		return nil, err
	}
	// Encode already appends exactly one trailing newline; buf carries no other bytes.
	return buf.Bytes(), nil
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
