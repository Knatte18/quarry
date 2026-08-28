// types.go declares the impact package's result types: Range, Target, Definition, Caller, and
// Result. Every JSON tag here is the exact emitted key set the plan's json-key-disposition Shared
// Decision fixes — no field is added or renamed without a corresponding Shared Decision change.

package impact

import "github.com/Knatte18/quarry/internal/quarryengine/toc"

// Range is a declaration's line span, in the same 1-based, inclusive convention toc.Symbol uses.
type Range struct {
	// StartLine is the declaration's first line — the docstring's first line when the declaration
	// carries a sibling docstring, mirroring toc.Symbol.Start.
	StartLine int `json:"start_line"`
	// SigEndLine is the last line of the declaration's signature, mirroring toc.Symbol.SigEnd. Zero
	// is toc.Symbol.SigEnd's documented absent marker, which is why only this field is omitempty.
	SigEndLine int `json:"sigend_line,omitempty"`
	// EndLine is the declaration's last line, mirroring toc.Symbol.End.
	EndLine int `json:"end_line"`
}

// Target identifies the resolved declaration an Impact query was issued against. Its provenance is
// always the single toc.Symbol the query position's enclosing lookup found — never the query
// string itself and never an LSP candidate.
type Target struct {
	// Kind is the target's toc.Kind, omitted when the target's own declaration line yielded no
	// enclosing symbol (a file-scope target).
	Kind toc.Kind `json:"kind,omitempty"`
	// Name is the target symbol's bare name.
	Name string `json:"name,omitempty"`
	// Owner is the enclosing type's bare name for a method target, empty otherwise.
	Owner string `json:"owner,omitempty"`
	// Package is the declaring file's declared package or namespace.
	Package string `json:"package,omitempty"`
	// Signature is the target symbol's verbatim, trimmed signature text.
	Signature string `json:"signature,omitempty"`
}

// Definition is the resolved declaration site's location and, when available, its enclosing
// declaration's range. Definition and Target are omitted together when no declaration was found
// for the query at all — see Result's doc comment for that joint-absence rule.
type Definition struct {
	// File is the declaration site's absolute file path.
	File string `json:"file"`
	// Line is the declaration site's 1-based line.
	Line int `json:"line"`
	// StartLine, SigEndLine, and EndLine are the enclosing declaration's range, present only when
	// the enclosing lookup found a match (outcome 1 of the three-outcome-degradation-rule). They are
	// jointly omitted, not individually, on outcomes 2 and 3.
	StartLine  int `json:"start_line,omitempty"`
	SigEndLine int `json:"sigend_line,omitempty"`
	EndLine    int `json:"end_line,omitempty"`
	// Error is set only on outcome 3 of the three-outcome-degradation-rule — the declaring file
	// could not be parsed at all. Its absence on outcome 2 (a file that parsed but has no listable
	// declaration covering the line) is deliberate: that outcome is a correct file-scope answer, not
	// a failure.
	Error string `json:"error,omitempty"`
}

// Caller is one call site referencing the resolved target, together with its enclosing
// declaration's identity and range when one was found.
type Caller struct {
	// File is the call site's absolute file path.
	File string `json:"file"`
	// CallSiteLine is the call site's 1-based line.
	CallSiteLine int `json:"call_site_line"`
	// CallSiteCharacter is the call site's 1-based character offset.
	CallSiteCharacter int `json:"call_site_character,omitempty"`
	// Kind, Name, Owner, Package, and Signature identify the caller's enclosing declaration, with
	// the same omitempty discipline Target carries and for the same reason: a file-scope caller (no
	// enclosing declaration) omits all five.
	Kind      toc.Kind `json:"kind,omitempty"`
	Name      string   `json:"name,omitempty"`
	Owner     string   `json:"owner,omitempty"`
	Package   string   `json:"package,omitempty"`
	Signature string   `json:"signature,omitempty"`
	// EnclosingRange is a pointer specifically so the whole object is omitted rather than emitted as
	// a zero-valued triple: its absence on a caller entry is the documented, meaningful signal
	// "file-scope reference — no enclosing declaration", not "lookup failed". A resolver failure is
	// reported through Error instead, below.
	EnclosingRange *Range `json:"enclosing_range,omitempty"`
	// Error is set only when the caller's own file could not be parsed at all (outcome 3), never for
	// a file-scope reference (outcome 2), matching Definition.Error's discipline.
	Error string `json:"error,omitempty"`
}

// Result is Impact's return value: the resolved target's identity and declaration site, plus every
// verified caller found.
type Result struct {
	// Target is the resolved declaration's identity, omitted together with Definition when the
	// query resolved to no declaration at all — a case distinct from a resolved declaration with no
	// enclosing symbol, which instead sets Definition with Target left nil.
	Target *Target `json:"target,omitempty"`
	// Definition is the resolved declaration's location, omitted together with Target under the same
	// rule stated above.
	Definition *Definition `json:"definition,omitempty"`
	// Callers is every surviving caller entry, always a non-nil, possibly-empty slice so the emitted
	// key is "[]" rather than "null" when there are none.
	Callers []Caller `json:"callers"`
}
