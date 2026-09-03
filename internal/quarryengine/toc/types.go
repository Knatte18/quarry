// types.go declares the toc package's result and option types: the closed Kind vocabulary, Symbol,
// FileTOC, DirEntry, DirTOC, and Options. Every JSON tag here is the exact emitted key set the
// plan's "the emitted key set is closed and is not re-litigated per batch" Shared Decision fixes —
// no field is added or renamed without a corresponding Shared Decision change.

package toc

// Kind is the closed vocabulary a Symbol's Kind field is drawn from.
type Kind string

// The three Kind values toc file ever emits. No other value is valid.
const (
	// KindFunction marks a free function: a func with no receiver.
	KindFunction Kind = "function"
	// KindMethod marks a function bound to a receiver (see Symbol.Owner).
	KindMethod Kind = "method"
	// KindType marks a type-level declaration.
	KindType Kind = "type"
)

// Symbol is one listable declaration extracted from a file: a function, method, or type, in source
// order.
type Symbol struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
	// Owner is the enclosing type's bare name for a method, and empty for a free function or a
	// type-level declaration. Name deliberately stays bare — it gains no Owner or Package
	// qualification — because refs, definition, and symbol accept only a bare name or a
	// file:line:col position; a caller composes the qualified form itself from Package, Owner, and
	// Name when it needs one for display.
	Owner string `json:"owner,omitempty"`
	// Signature is the verbatim source text from the declaration's first byte to the start of its
	// body-bearing child, trimmed — never reformatted, never truncated.
	Signature string `json:"signature"`
	// Docstring is the symbol's full, delimiter-stripped docstring, optionally trimmed to its first
	// N sentences by the entry point per the caller's --doc-sentences option. An empty docstring is
	// never emitted as "" — its absence is always signalled by omitting this key.
	Docstring string `json:"docstring,omitempty"`
	// Start is the first line of the docstring when the docstring is a sibling of the declaration,
	// and the first line of the declaration otherwise. 1-based, inclusive.
	Start int `json:"start"`
	// SigEnd is the last line of the signature, 1-based and inclusive like Start and End. It is the
	// semantic "where the body begins" line, derived per language, and is zero — hence omitted by
	// omitempty — for a symbol with no body at all, such as a Go type alias. Every real line number
	// is 1-based, so zero is unambiguous as the absent marker.
	SigEnd int `json:"sigend,omitempty"`
	// End is the last line of the declaration. 1-based, inclusive.
	End int `json:"end"`
}

// FileTOC is the result of a single-file toc extraction.
type FileTOC struct {
	// Header is the file's first-paragraph-truncated, delimiter-stripped leading comment, or empty
	// when the file has none.
	Header string `json:"header,omitempty"`
	// Language is the canonical language name toc resolved the file to.
	Language string `json:"language"`
	// Package is the file's declared package or namespace name, or empty for a language with no
	// such concept, or a file that declares none. This is one field per file rather than per
	// symbol: the value is identical for every symbol in the file, so repeating it per symbol would
	// pay for the same string once per symbol for no benefit.
	Package string `json:"package,omitempty"`
	// Symbols is every listable declaration in source order ascending by Start. It is always a
	// non-nil, possibly-empty slice when the parse succeeded, so the emitted key is "[]" rather than
	// "null".
	Symbols []Symbol `json:"symbols"`
	// Partial is true when the parse tree reported a syntax error and the result may be lossy. It
	// is omitted when false.
	Partial bool `json:"partial,omitempty"`
}

// DirEntry is one file's toc summary inside a directory listing.
type DirEntry struct {
	// Name is the file's base name. It carries json:"-" because internal/cli composes the emitted
	// "path" key as filepath.Join(<the caller's own directory argument>, Name) — this package
	// deliberately does not know the caller's frame of reference.
	Name string `json:"-"`
	// Language is the canonical language name toc resolved the file to.
	Language string `json:"language"`
	// Package mirrors FileTOC.Package: one field per file, omitted when the file declares none.
	Package string `json:"package,omitempty"`
	// Header mirrors FileTOC.Header: the first-paragraph-truncated, delimiter-stripped leading
	// comment, omitted when absent.
	Header string `json:"header,omitempty"`
	// Partial mirrors FileTOC.Partial, omitted when false.
	Partial bool `json:"partial,omitempty"`
	// Test is a pointer, not a bool, because the contract distinguishes "false" from "the language
	// has no rule": a nil pointer omits the "test" key entirely, and a pointer to false emits
	// false. Do not "simplify" this to a plain bool — that would silently turn "cannot tell" into
	// "no" for every language whose Strategy reports known == false.
	Test *bool `json:"test,omitempty"`
	// Generated mirrors Test's pointer discipline for the same reason: nil omits the "generated"
	// key, a pointer to false emits false, and neither may be invented for a language with no
	// reliable rule.
	Generated *bool `json:"generated,omitempty"`
	// Error is set only when this file could not be parsed at all — an unreadable file, invalid
	// UTF-8, or an unsupported language — and is mutually exclusive with both Header and Partial.
	Error string `json:"error,omitempty"`
}

// DirTOC is the result of a directory toc listing.
type DirTOC struct {
	// Files is every listed file's entry. Ordering is the caller's (internal/cli's) responsibility,
	// not this package's.
	Files []DirEntry `json:"files"`
	// Dirs is every direct subdirectory's base name, sorted lexicographically, with no other detail
	// per entry: TOCDir never descends into a subdirectory, so nothing beyond its name is ever known
	// here. Like Files, it is always a non-nil, possibly-empty slice, so the emitted key is "[]"
	// rather than "null" or an omitted key.
	Dirs []string `json:"dirs"`
}

// AllSentences is the Options.DocSentences sentinel meaning "keep the whole docstring, unsplit".
const AllSentences = -1

// Options configures how much of each symbol's docstring toc emits.
type Options struct {
	// DocSentences controls how many leading sentences of each symbol's docstring reach the
	// emitted "docstring" key: 0 omits the key from every symbol, a positive N keeps the first N
	// sentences (an N larger than the sentence count keeps the whole docstring and is not an
	// error), and AllSentences keeps the docstring unsplit.
	//
	// The zero value of Options therefore means "omit every docstring" — every caller must set this
	// field explicitly. A forgotten Options{} silently drops every docstring rather than defaulting
	// to some other behavior.
	DocSentences int
}
