// enclose.go implements the enclose query: mapping path:line ranges to the members that enclose them.
// It declares the query's input (Location, RevisionFiles), its answer (EncloseResult) and its closed
// rejection vocabulary.

package engine

// EncloseResult is the answer to one Location passed to Enclose or EncloseFrom.
// It is either an outcome (Status set) or a rejection (Reason and Error set), never both.
type EncloseResult struct {
	// Target is the caller's location spelling, verbatim. Always present.
	Target string `json:"target"`
	// File is the normalised repository-relative path.
	// Start and End are the line range.
	// All three are filled on a rejection whenever they were determined before it.
	File  string `json:"file,omitempty"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
	// Status is StatusFound when at least one member encloses part of the range
	// and StatusNotFound when none does.
	// It is empty exactly when the location was rejected.
	Status Status `json:"status,omitempty"`
	// Symbols are the innermost members touching the range, in source order, present only for found.
	Symbols []Symbol `json:"symbols,omitempty"`
	// Unit is set only on not_found.
	// Unlike ResolveResult.Unit (a Status), this key holds the file's self glyph string (e.g. a/b.go#),
	// and its presence alone means the range is not inside any member.
	Unit string `json:"unit,omitempty"`
	// Lossy is true when the file parsed only partially, so the answer may be incomplete.
	Lossy bool `json:"lossy,omitempty"`
	// Error is the one-sentence rejection message, set only on a rejection.
	Error string `json:"error,omitempty"`
	// Reason is the rejection's plain-word reason, one of EncloseReasons, set only on a rejection.
	Reason string `json:"reason,omitempty"`
}

// Rejected reports whether r is a rejection rather than an outcome.
// It reads r.Status because Status is empty exactly when the location was rejected.
func (r EncloseResult) Rejected() bool { return r.Status == "" }

// The eight rejection reasons of an enclose location, in the order the checks run.
const (
	// EncloseReasonBadLocation marks a target that is not path:line, path:line-line or path:line:col.
	EncloseReasonBadLocation = "bad_location"
	// EncloseReasonBadRange marks a range whose start is below 1 or after its end.
	EncloseReasonBadRange = "bad_range"
	// EncloseReasonOutsideRoot marks a path that normalises outside the repository root.
	EncloseReasonOutsideRoot = "outside_root"
	// EncloseReasonUnaddressable marks a file whose members no glyph can name:
	// its path or its glyph unit cannot be spelled.
	EncloseReasonUnaddressable = "unaddressable"
	// EncloseReasonUnsupportedLanguage marks a file whose extension has no registered language strategy.
	EncloseReasonUnsupportedLanguage = "unsupported_language"
	// EncloseReasonMissingFile marks a path with no regular file behind it.
	EncloseReasonMissingFile = "missing_file"
	// EncloseReasonUnreadable marks a file that exists but cannot be read as source.
	EncloseReasonUnreadable = "unreadable"
	// EncloseReasonPastEOF marks a range ending after the file's last line.
	EncloseReasonPastEOF = "past_eof"
)

// EncloseReasons lists all eight enclose rejection reasons, in the same order as the constant block above.
// Go cannot reflect over package-level constants, so this slice is the only way a test or a caller can
// enumerate the vocabulary.
// Adding a constant means adding it here in the same edit, exactly as NameReasons does.
var EncloseReasons = []string{
	EncloseReasonBadLocation,
	EncloseReasonBadRange,
	EncloseReasonOutsideRoot,
	EncloseReasonUnaddressable,
	EncloseReasonUnsupportedLanguage,
	EncloseReasonMissingFile,
	EncloseReasonUnreadable,
	EncloseReasonPastEOF,
}

// Location is one input to Enclose: either one normalised location (File, Start and End set)
// or one the caller already rejected (Reason and Error set, plus whichever of File, Start and End
// were determined before the rejection).
// Target is the caller's spelling and is echoed on the answer.
type Location struct {
	Target, File  string
	Start, End    int
	Reason, Error string
}

// RevisionFiles is the engine's git-ignorant view of one revision's files.
type RevisionFiles interface {
	// GoFiles returns the Go files that are immediate children of dirRel at the revision,
	// as repository-relative paths; dirRel "." is the repository root.
	GoFiles(dirRel string) ([]string, error)
	// Read returns one file's bytes at the revision.
	Read(rel string) ([]byte, error)
}
