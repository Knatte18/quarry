// answer.go declares the engine package's answer shape: the closed Kind vocabulary, Symbol, the
// recursive DirAnswer, FileEntry, and TOCOptions. Every JSON tag here is the exact emitted key set
// the plan's "the emitted key set is plan §4's and is closed" Shared Decision fixes — no field is
// added or renamed without a corresponding Shared Decision change.

package engine

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
	// Docstring is the symbol's complete, untrimmed, delimiter-stripped docstring. An empty
	// docstring is never emitted as "" — its absence is always signalled by omitting this key.
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

// DirAnswer is the recursive answer to a table-of-contents query, per the rewrite plan's §4: one
// directory's identity and package facts, its files, and — when the query's depth reaches this
// far — its subdirectories as nested DirAnswer values. A file target is answered as a one-entry
// DirAnswer carrying its enclosing directory's facts; see Repo.TOC's doc comment for that shape.
type DirAnswer struct {
	// Dir is the repository-relative path with forward slashes, "." for the repository root.
	Dir string `json:"dir"`
	// Package is the directory's package name, present only when the directory has one — a
	// directory with no .go file, or one whose files agree on no single clause under the tie-break
	// rule, has none.
	Package string `json:"package,omitempty"`
	// Language is the language of the directory's package, present only when Package is present.
	Language string `json:"language,omitempty"`
	// Doc is the directory's package documentation, selected from the files sharing Package. Empty
	// when no such file carries one.
	Doc string `json:"doc,omitempty"`
	// Files is every listed file's entry, sorted lexicographically by Name. Omitted when a depth
	// cut leaves this DirAnswer identity-plus-doc only.
	Files []FileEntry `json:"files,omitempty"`
	// Dirs is this directory's subdirectories, sorted lexicographically by Dir. Omitted at the
	// bottom of a depth-limited query, where a subdirectory contributes nothing further.
	Dirs []DirAnswer `json:"dirs,omitempty"`
}

// FileEntry is one file's summary inside a DirAnswer's Files.
type FileEntry struct {
	// Name is the file's base name.
	Name string `json:"name"`
	// Header is the file's first-paragraph-truncated, delimiter-stripped leading comment, or absent
	// when the file has none.
	Header string `json:"header,omitempty"`
	// Test is true when this file is a test file by its language's toolchain convention. It is a
	// plain bool, not a pointer: the pointer in V1 existed to say "this language has no rule", a
	// state that cannot arise while Go is the only language, and the plan forbids emitting
	// "test: false" either way, so both the false value and the no-rule value collapse to the same
	// omitted key.
	Test bool `json:"test,omitempty"`
	// Generated mirrors Test's plain-bool discipline for the same reason: true only when the file
	// matches its language's generated-file banner convention, omitted otherwise.
	Generated bool `json:"generated,omitempty"`
	// Package is emitted only when this file's own package clause differs from the directory's
	// package — the deviation case, such as an external test file's "_test" suffixed package.
	Package string `json:"package,omitempty"`
	// Language is emitted only when this file's language differs from the directory's language.
	Language string `json:"language,omitempty"`
	// Lossy is true when the parse tree reported a syntax error and the result may be incomplete.
	// It carries what V1's Partial field carried; the rename frees the word "partial" for C#'s own
	// meaning. Lossy and Error are never both set.
	Lossy bool `json:"lossy,omitempty"`
	// Error is set only when this file could not be read or parsed at all — an unreadable file or
	// invalid UTF-8 — and is mutually exclusive with Lossy.
	Error string `json:"error,omitempty"`
	// Symbols is the one pointer field on this type, and deliberately so: a nil Symbols means
	// symbols were not requested for this file, and a present, possibly-empty *[]Symbol means they
	// were requested and the file declares none. encoding/json's omitempty drops a nil pointer and
	// an empty slice alike, so only the pointer-vs-non-pointer distinction — not the slice's own
	// length — can tell "not requested" apart from "requested, found none". A plain []Symbol field
	// cannot make that distinction, which is why this field is the exception to every other slice
	// in this file staying a plain, non-pointer type.
	Symbols *[]Symbol `json:"symbols,omitempty"`
}

// DepthAll requests a TOC query recurse to the bottom of the tree, rather than stopping after a
// fixed number of levels.
const DepthAll = -1

// TOCOptions carries the two knobs Repo.TOC accepts.
type TOCOptions struct {
	// Depth controls how far a directory query recurses. 0 fills the target directory's own Files
	// and lists its direct subdirectories as identity-plus-doc DirAnswer values with no Files or
	// Dirs of their own. N fills the Files of every subdirectory N levels down, with that level's
	// own leaf Dirs again identity-plus-doc. DepthAll recurses to the bottom of the tree. Depth is
	// ignored for a file target, since there is nothing below a file to fill.
	Depth int
	// Symbols selects whether each FileEntry's Symbols field is populated. nil selects the
	// per-target default: true for a file target, false for a directory target. A non-nil value
	// wins for every file entry at every depth of the query, overriding that default.
	Symbols *bool
}
