// delta_answer.go declares the delta query's whole answer shape: the entry the core takes as input,
// the closed disposition and changed-dimension vocabularies, and every type DeltaAnswer is built
// from. This file's JSON tags extend answer.go's closed emitted key set additively — see that
// file's own header comment — and change nothing it already declares.

package engine

// Disposition is the closed per-file vocabulary a DeltaFile's Disposition field is drawn from.
type Disposition string

// The five Disposition values a delta batch ever emits. No other value is valid.
const (
	// DispositionAdded marks a file present only on the after side.
	DispositionAdded Disposition = "added"
	// DispositionRemoved marks a file present only on the before side.
	DispositionRemoved Disposition = "removed"
	// DispositionChanged marks a file present on both sides and extracted on each.
	DispositionChanged Disposition = "changed"
	// DispositionUnsupported marks a file whose extension resolves to no registered strategy. It
	// contributes no symbols on either side and is not an error.
	DispositionUnsupported Disposition = "unsupported"
	// DispositionError marks an entry that was refused before extraction, or that failed extraction.
	// It contributes no symbols to any delta list.
	DispositionError Disposition = "error"
)

// ChangedDimension is the closed vocabulary a ModifiedSymbol's Changed array draws its elements
// from, declared in the exact order a changed array is emitted in.
type ChangedDimension string

// The four ChangedDimension values, in the order a changed array reports them.
const (
	// ChangedBody marks a difference in the body token stream.
	ChangedBody ChangedDimension = "body"
	// ChangedSignature marks a difference in the signature token stream.
	ChangedSignature ChangedDimension = "signature"
	// ChangedDoc marks a difference in the docstring text.
	ChangedDoc ChangedDimension = "doc"
	// ChangedFile marks a difference in the file the symbol was extracted from.
	ChangedFile ChangedDimension = "file"
)

// DeltaEntry is one changed file's before-and-after versions, the pure core's whole input for that
// file. None of its six fields is emitted: the input is not part of the answer.
type DeltaEntry struct {
	// Path is the entry's repository-relative path.
	Path string `json:"-"`
	// Before is the file's source bytes on the before side. nil means the file did not exist on
	// that side; an empty-but-non-nil slice means an existing empty file.
	Before []byte `json:"-"`
	// After is the file's source bytes on the after side, under the same nil-vs-empty rule Before
	// follows.
	After []byte `json:"-"`
	// BeforeUnit is the file's glyph unit on the before side.
	BeforeUnit string `json:"-"`
	// AfterUnit is the file's glyph unit on the after side.
	AfterUnit string `json:"-"`
	// Refusal, when non-empty, means the layer that assembled the batch already decided this entry
	// cannot be extracted: the core skips it entirely and emits DispositionError with Refusal as the
	// message.
	Refusal string `json:"-"`
}

// DeltaFile is one entry's disposition in a DeltaAnswer's Files, one per input DeltaEntry in the
// input's own order.
type DeltaFile struct {
	// Path is the entry's repository-relative path.
	Path string `json:"path"`
	// Disposition is the closed outcome word for this entry.
	Disposition Disposition `json:"disposition"`
	// Error carries the refusal or extraction-failure message, present only for DispositionError.
	Error string `json:"error,omitempty"`
	// LossyBefore is true when the before side's parse reported a syntax error and its surviving
	// symbols may be incomplete.
	LossyBefore bool `json:"lossy_before,omitempty"`
	// LossyAfter mirrors LossyBefore for the after side.
	LossyAfter bool `json:"lossy_after,omitempty"`
}

// SymbolLocation mirrors a Symbol's own File, Start, SigEnd and End fields, so a modified symbol's
// before-side block reads the same way a symbol does without carrying the rest of Symbol's fields.
type SymbolLocation struct {
	// File is the repository-relative path the symbol was extracted from.
	File string `json:"file"`
	// Start mirrors Symbol.Start.
	Start int `json:"start"`
	// SigEnd mirrors Symbol.SigEnd.
	SigEnd int `json:"sigend"`
	// End mirrors Symbol.End.
	End int `json:"end"`
}

// ModifiedSymbol is one symbol-table key whose before-side and after-side occurrences differ. Both
// Before and After are always arrays, so a key held by several declarations — Go's several
// func init() in one package, for instance — needs no second entry shape: the multiplicity is
// visible from the lengths.
type ModifiedSymbol struct {
	// ID is the symbol table key's glyph string.
	ID string `json:"id"`
	// Kind is the symbol table key's kind.
	Kind Kind `json:"kind"`
	// Changed is the union of dimensions that differ across the before/after occurrence multisets,
	// in ChangedDimension's declared order.
	Changed []ChangedDimension `json:"changed"`
	// Before is every before-side occurrence's location, in file-then-line order.
	Before []SymbolLocation `json:"before"`
	// After is every after-side occurrence's full Symbol, in file-then-line order.
	After []Symbol `json:"after"`
}

// RenamedPair is one exact-tier rename: a deleted symbol paired with a created symbol under the
// exact-tier conditions.
//
// An exact pair's constituents are removed from the created and deleted arrays, so the pair is the
// only surviving record of either symbol's location.
type RenamedPair struct {
	// From is the deleted symbol's full before-side Symbol.
	From Symbol `json:"from"`
	// To is the created symbol's full after-side Symbol.
	To Symbol `json:"to"`
}

// RenameSignals carries the evidence-tier's purely mechanical, individually explainable fields for
// one candidate pair. None is omitted.
type RenameSignals struct {
	// SignatureIdenticalModuloName reports whether the two signature token streams are identical
	// modulo the renamed identifier.
	SignatureIdenticalModuloName bool `json:"signature_identical_modulo_name"`
	// BodyTokenSimilarity is the Jaccard coefficient of the two body token streams.
	BodyTokenSimilarity float64 `json:"body_token_similarity"`
	// BodyTokensBefore is the deleted symbol's body token stream length.
	BodyTokensBefore int `json:"body_tokens_before"`
	// BodyTokensAfter is the created symbol's body token stream length.
	BodyTokensAfter int `json:"body_tokens_after"`
	// DocIdentical reports whether the two symbols' docstrings are equal.
	DocIdentical bool `json:"doc_identical"`
}

// RenameCandidate is one created symbol considered as a possible match for a deleted symbol, inside
// that deleted symbol's own RenameCandidateEntry.
type RenameCandidate struct {
	// ID is the created symbol's glyph string.
	ID string `json:"id"`
	// Kind is the created symbol's kind.
	Kind Kind `json:"kind"`
	// File is the created symbol's repository-relative path.
	File string `json:"file"`
	// Signals is the evidence computed for this candidate pair.
	Signals RenameSignals `json:"signals"`
}

// RenameCandidateEntry is one deleted symbol's evidence-tier candidates.
//
// Candidate ordering is deterministic ordering and explicitly not a ranking, not a recommendation
// and not a verdict. A directory with many deleted and many created symbols of one kind and owner
// can produce a large candidate block, which is accepted rather than capped because a cap is a
// threshold under another name.
type RenameCandidateEntry struct {
	// ID is the deleted symbol's glyph string.
	ID string `json:"id"`
	// Kind is the deleted symbol's kind.
	Kind Kind `json:"kind"`
	// Candidates is every created symbol considered for this deleted symbol, sorted by
	// BodyTokenSimilarity descending, then by created ID ascending.
	Candidates []RenameCandidate `json:"candidates"`
}

// DeltaAnswer is the whole answer to a delta query: every input entry's disposition, and the
// symbol-table comparison drawn from the entries extracted as DispositionChanged, DispositionAdded
// or DispositionRemoved.
//
// This answer diverges from the table-of-contents verb's listing rule: a file that is tracked and
// also matched by a gitignore pattern is kept in a delta batch, so this answer can report a symbol
// that verb never lists.
type DeltaAnswer struct {
	// Files is every input entry's disposition, in the input's own order.
	Files []DeltaFile `json:"files"`
	// Created is every symbol present only on the after side, excluding the symbols an exact-tier
	// rename removes.
	Created []Symbol `json:"created"`
	// Deleted is every symbol present only on the before side, excluding the symbols an exact-tier
	// rename removes.
	Deleted []Symbol `json:"deleted"`
	// Modified is every symbol-table key whose occurrences differ between the two sides.
	Modified []ModifiedSymbol `json:"modified"`
	// Renamed is every exact-tier rename pair.
	Renamed []RenamedPair `json:"renamed"`
	// RenameCandidates is every deleted symbol's evidence-tier candidate block.
	RenameCandidates []RenameCandidateEntry `json:"rename_candidates"`
}
