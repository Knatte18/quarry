// answer.go declares the engine package's answer shape: the closed Kind vocabulary, Symbol, the
// recursive DirAnswer, FileEntry, TOCOptions, the closed Status vocabulary, ResolveResult, and
// ExpandAnswer. Every JSON tag here is the exact emitted key set the plan's "the emitted key set is
// plan §4's and is closed" Shared Decision fixes — no field is added or renamed without a
// corresponding Shared Decision change. The C1 task ("Glyph self-form and the resolve contract") is
// exactly such a change: it renames ResolveResult.Dir to ResolveResult.Listing and its "dir" JSON
// key to "listing", because the field now carries a self glyph's answer as well as a path's.

package engine

import "github.com/Knatte18/quarry/glyph"

// Kind is the closed vocabulary a Symbol's Kind field is drawn from.
type Kind string

// The five Kind values toc file ever emits. No other value is valid.
const (
	// KindFunction marks a free function: a func with no receiver.
	KindFunction Kind = "function"
	// KindMethod marks a function bound to a receiver, including an interface method.
	KindMethod Kind = "method"
	// KindType marks a type-level declaration.
	KindType Kind = "type"
	// KindConst marks a package-level const declaration.
	KindConst Kind = "const"
	// KindVar marks a package-level var declaration.
	KindVar Kind = "var"
)

// Status is the closed per-entry vocabulary of docs/glyph.md §5, shared by ResolveResult's Status
// and Unit keys and by ExpandAnswer's. The Unit key of both result types draws from this same type
// but only ever carries StatusFound or StatusNotFound, so the package holds one vocabulary rather
// than two overlapping ones.
type Status string

// The four Status values docs/glyph.md §5 defines. No other value is valid.
const (
	// StatusFound marks exactly one matching declaration.
	StatusFound Status = "found"
	// StatusNotFound marks no matching declaration.
	StatusNotFound Status = "not_found"
	// StatusAmbiguous marks several different declarations matching, with nothing chosen between
	// them.
	StatusAmbiguous Status = "ambiguous"
	// StatusMultipart marks one symbol the language lets be declared in several places, with every
	// part returned.
	StatusMultipart Status = "multipart"
)

// Symbol is one listable declaration extracted from a file: a function, method, type, const, or
// var, in source order.
//
// Symbol carries its own Glyph rather than the bare Name/Owner pair V1 used, because a caller
// reading Glyph.Name and Glyph.Owner and a caller reading the emitted ID string were two parallel
// spellings of one identity — exactly the drift docs/glyph.md's one-implementation-of-the-grammar
// rule exists to prevent. ID is computed and stored once, at build time, rather than derived on
// demand by a custom MarshalJSON: the emitted key set stays a plain struct with plain tags, and
// there is no second place that could compute it differently.
type Symbol struct {
	// Glyph is this symbol's parsed identity: unit, owner chain, and name. It is never emitted —
	// ID, computed from it once at build time, is the wire form.
	Glyph glyph.Glyph `json:"-"`
	// ID is Glyph.String(), stored rather than recomputed so the JSON encoding stays a plain
	// struct-tag marshal with no custom MarshalJSON.
	ID string `json:"id"`
	// Kind is the symbol's declaration kind.
	Kind Kind `json:"kind"`
	// File is the repository-relative path of the file this symbol was extracted from, empty and
	// therefore omitted inside a toc answer, where the symbol already sits in its file's own entry.
	// symbolsOfUnit fills File, and Resolve and Expand emit it, because their entries span
	// files.
	File string `json:"file,omitempty"`
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
	// Signature is the verbatim source text from the declaration's first byte to the start of its
	// body-bearing child, trimmed — never reformatted, never truncated.
	Signature string `json:"signature"`
	// Doc is the symbol's complete, untrimmed, delimiter-stripped docstring. An empty docstring is
	// never emitted as "" — its absence is always signalled by omitting this key.
	Doc string `json:"doc,omitempty"`
	// HeadStart and HeadEnd are populated only for KindType, and are JSON-hidden: they are consumed
	// by the expand verb, never emitted directly. For Go they equal this same symbol's own
	// Start and End — doc block included when one is attached, never the bare declaration node's
	// line range — for every Go type, interfaces included. Subtracting a type's member spans from
	// its head range to render just the head is the consumer's job, not the extractor's, which is
	// why one span pair suffices here and no discontiguous span type is needed.
	HeadStart int `json:"-"`
	HeadEnd   int `json:"-"`
	// DeclStart, BodyStart and DeclEnd are byte offsets into the file's own source, not line
	// numbers. They are JSON-hidden for the same reason HeadStart and HeadEnd are: they exist for
	// one consumer rather than for the wire. DeclStart and DeclEnd are the declaration node's start
	// and end bytes. BodyStart is the body-bearing child's start byte when the declaration has one,
	// and equals DeclEnd when it does not, so BodyStart == DeclEnd is the marker for a declaration
	// with no body at all and the body byte range is empty rather than absent. The span
	// [DeclStart, BodyStart) is the same span SignatureCut is given, so a text comparison over the
	// signature and a token comparison over it are over the same bytes by construction.
	//
	// The emitted key set is unchanged by this addition, which is what keeps this file's own header
	// rule — every JSON tag here is the closed emitted key set — satisfied without a Shared Decision
	// change.
	DeclStart int `json:"-"`
	BodyStart int `json:"-"`
	DeclEnd   int `json:"-"`
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

// ResolveResult is the answer to one target passed to Resolve: a glyph, member or self alike —
// every target is a glyph now that glyph.Parse is the resolve contract's only classifier. It
// expresses either a resolution outcome (Status set) or a pre-resolution rejection of the target
// string itself (Error set), and never an engine failure — an engine failure fails the whole
// Resolve call instead. Status and Error are never both set.
type ResolveResult struct {
	// Target is the caller's argument, verbatim. Always present.
	Target string `json:"target"`
	// ID is the parsed glyph's Glyph.String() form, the same wire form Symbol.ID carries. Present on
	// every non-rejection result, since every target is a glyph now. For Go it is byte-identical to
	// Target on every non-error result, because the Go alphabet normalises nothing — the key earns
	// its place as parity with Symbol.ID, not as canonicalisation.
	ID string `json:"id,omitempty"`
	// Status is the resolution outcome: found, not_found, ambiguous, or multipart. It is absent only
	// when the target never reached resolution — a pre-resolution rejection, carried by Error and
	// Reason instead.
	Status Status `json:"status,omitempty"`
	// Unit is set only on a not_found, and carries only StatusFound or StatusNotFound: found when the
	// directory, module or namespace is there and only the member is missing, not_found otherwise.
	// This applies to a self glyph exactly as it does to a member glyph — a self glyph belongs to a
	// unit too — so Unit is never suppressed for one.
	//
	// Contract gap: docs/glyph.md §2's external test unit and a real directory spelling the same
	// string can both exist; unitDirs reports the collision as a bare bool with no notion of "unit
	// directories", so which of the two this key promotes from is decided where the flag is read, not
	// here.
	Unit Status `json:"unit,omitempty"`
	// Symbols carries the matches for found (exactly one) and for multipart (every declaration).
	Symbols []Symbol `json:"symbols,omitempty"`
	// Candidates carries the matches for ambiguous. The separate key is the signal that nothing was
	// chosen.
	//
	// Contract gap: docs/glyph.md §5 says candidates in a multi-language repository are marked by
	// language, while Symbol's key set has no language key. With Go the only alphabet the case is
	// unreachable, so no key is added here — a second language's task adds the marker against a real
	// case.
	Candidates []Symbol `json:"candidates,omitempty"`
	// Listing is the directory-or-file listing a found self glyph resolved to: a directory's own
	// DirAnswer when the self glyph names a directory, and the enclosing directory's DirAnswer
	// holding exactly that one file's entry when it names a file. The old name, "Dir", claimed
	// "directory" while also carrying single-file answers, and it repeated the inner DirAnswer.Dir
	// key's own word one level up; "listing" names what the block actually is without either
	// problem. It is a pointer so an absent answer is dropped by omitempty; a non-pointer struct
	// would always marshal.
	Listing *DirAnswer `json:"listing,omitempty"`
	// Error carries a pre-resolution rejection of the target string itself — a glyph.Parse rejection
	// — and never an engine read failure. Status is empty whenever Error is set.
	Error string `json:"error,omitempty"`
	// Reason is the plain-word form of the rejection Error names, for a grammar rejection. It is
	// deliberately a plain string, not glyph.Reason: the emitted JSON is a plain word and answer.go
	// needs no exported alias for the grammar's own type.
	Reason string `json:"reason,omitempty"`
}

// ExpandAnswer is the answer to one glyph passed to Expand: the target type's head plus every
// member whose owner chain begins with it. Status is found, not_found or ambiguous, and never
// multipart, because the kind gate sends every match set holding no type to NotATypeError — where a
// several-declaration init glyph lands, however many declarations it has — and docs/rewrite-plan.md's
// section 5, The queries, rule that a Go type never splits, only init does, closes the remaining
// type-only cases. A language with partial types adds its row then, not now.
type ExpandAnswer struct {
	// ID is the parsed glyph's Glyph.String() form, the same wire form Symbol.ID carries.
	ID string `json:"id"`
	// Status is the resolution outcome: found, not_found, or ambiguous.
	Status Status `json:"status"`
	// Unit is set only on a not_found, and carries only StatusFound or StatusNotFound, exactly as
	// ResolveResult.Unit does.
	Unit Status `json:"unit,omitempty"`
	// Head is the type's own symbol entry, with Start and End substituted from HeadStart and
	// HeadEnd, present only for found. It is a pointer so an absent answer is dropped by omitempty; a
	// non-pointer struct would always marshal.
	//
	// Contract gap: substituting HeadStart/HeadEnd for Start/End means the type's full declaration
	// span is not recoverable from an ExpandAnswer for a language whose head is a strict subset of
	// its declaration. With Go the only alphabet the case is unreachable today.
	Head *Symbol `json:"head,omitempty"`
	// Members is every symbol whose owner chain begins with the target type, sorted by file and
	// line, present only for found.
	Members []Symbol `json:"members,omitempty"`
	// Candidates carries the matches for ambiguous, exactly as ResolveResult.Candidates does.
	Candidates []Symbol `json:"candidates,omitempty"`
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
