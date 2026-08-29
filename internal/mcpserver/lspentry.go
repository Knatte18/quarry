// lspentry.go declares the entry shape shared by textDocument_definition, textDocument_references,
// and workspace_symbol: the raw-capture convention every entry type in this package follows, the
// three-form union an LSP-mirrored entry accepts (textDocument+position, symbol alone, or
// textDocument+symbol), and runTargets, the sequential batching loop every tool's handler drives its
// "targets" array through.

package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/Knatte18/quarry/quarry"
)

// textDocumentIdentifier names the file an entry's position or in-file symbol search resolves
// against, mirroring LSP's own TextDocumentIdentifier.
type textDocumentIdentifier struct {
	// URI is a file:// URI or a plain path, resolved against the call's targetDir by
	// resolveEntryFile. It must not be empty when TextDocument is present.
	URI string `json:"uri,omitempty" jsonschema:"the file this entry's position or symbol search resolves against, as a file:// URI or a plain path (absolute, or relative to targetDir)"`
}

// lspPosition is a 0-based line/character pair, mirroring LSP's own Position. Line and Character are
// pointers so a field the caller omitted is distinguishable from an explicit 0 — neither axis is
// inferred as required by the schema, since the handler validates the legal combinations itself.
type lspPosition struct {
	// Line is the entry's 0-based line number. A nil value means the field was omitted.
	Line *int `json:"line,omitempty" jsonschema:"0-based line number"`
	// Character is the entry's 0-based character offset. A nil value means the field was omitted.
	Character *int `json:"character,omitempty" jsonschema:"0-based character offset"`
}

// lspEntry is one target of an LSP-mirrored tool call: textDocument_definition,
// textDocument_references, or workspace_symbol. It accepts exactly three forms — TextDocument plus
// Position, Symbol alone, or TextDocument plus Symbol — enforced by query, not by the schema. raw
// carries the entry's original JSON bytes so a handler can echo the input verbatim under "target"
// and detect a key the tool does not declare via unknownEntryKeys; it is unexported and therefore
// invisible to both encoding/json and schema inference.
type lspEntry struct {
	raw json.RawMessage

	// TextDocument names the file a position or in-file symbol search resolves against. Required
	// together with Position, and optional together with Symbol (its presence there switches the
	// search from project-wide to file-scoped).
	TextDocument *textDocumentIdentifier `json:"textDocument,omitempty" jsonschema:"the file a position or in-file symbol search resolves against; required with position, optional with symbol"`
	// Position is an explicit 0-based source position, used together with TextDocument. Mutually
	// exclusive with Symbol.
	Position *lspPosition `json:"position,omitempty" jsonschema:"an explicit 0-based source position, used together with textDocument; mutually exclusive with symbol"`
	// Symbol is a symbol name, resolved project-wide when TextDocument is absent or within
	// TextDocument's file when present. Mutually exclusive with Position.
	Symbol string `json:"symbol,omitempty" jsonschema:"a symbol name to resolve; project-wide when textDocument is absent, or within textDocument's file when present; mutually exclusive with position"`
	// Within restricts this entry's own reference results to files within the named directory
	// (relative to the call's targetDir, or absolute). Not every LSP-mirrored tool accepts it.
	Within string `json:"within,omitempty" jsonschema:"restrict this entry's own reference results to files within this directory (relative to targetDir, or absolute)"`
}

// lspEntryAlias is a defined type (not a type alias) with lspEntry's exact underlying type, used
// only inside UnmarshalJSON. A defined type carries no methods of the type it is defined from, so
// decoding into it does not re-invoke lspEntry.UnmarshalJSON recursively, while every exported field
// lspEntry declares is still populated.
type lspEntryAlias lspEntry

// UnmarshalJSON decodes data into e's exported fields via lspEntryAlias, then records data unchanged
// into e.raw. Decoding through the alias — rather than embedding a shared raw-capture helper type —
// is required: Go promotes a method from an embedded field to the outer type, so an embedded
// raw-capture type's UnmarshalJSON would hijack lspEntry's own decode and leave every declared field
// at its zero value.
func (e *lspEntry) UnmarshalJSON(data []byte) error {
	var alias lspEntryAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	alias.raw = append(json.RawMessage(nil), data...)
	*e = lspEntry(alias)
	return nil
}

// query converts e into the quarry.Query one of the three legal forms describes, resolving any
// TextDocument.URI against targetDir via resolveEntryFile and converting a Position's 0-based Line
// and Character to the 1-based values quarry.Position expects. Every combination outside the three
// legal forms — a position with no textDocument, neither symbol nor position, both symbol and
// position, a position missing line or character, or an empty uri — returns an error naming the
// three accepted forms.
func (e lspEntry) query(targetDir string) (quarry.Query, error) {
	const acceptedForms = "textDocument+position, symbol alone, or textDocument+symbol"

	hasPosition := e.Position != nil
	hasSymbol := e.Symbol != ""

	if hasPosition && hasSymbol {
		return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; got both position and symbol", acceptedForms)
	}
	if !hasPosition && !hasSymbol {
		return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; got neither", acceptedForms)
	}

	if hasPosition {
		if e.TextDocument == nil {
			return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; position requires textDocument", acceptedForms)
		}
		if e.TextDocument.URI == "" {
			return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; textDocument.uri must not be empty", acceptedForms)
		}
		if e.Position.Line == nil || e.Position.Character == nil {
			return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; position requires both line and character", acceptedForms)
		}
		return quarry.Query{Pos: &quarry.Position{
			File:      resolveEntryFile(targetDir, e.TextDocument.URI),
			Line:      toOneBased(*e.Position.Line),
			Character: toOneBased(*e.Position.Character),
		}}, nil
	}

	// hasSymbol, no position: either a plain project-wide search (no textDocument) or an in-file
	// search (textDocument present).
	if e.TextDocument == nil {
		return quarry.Query{Symbol: e.Symbol}, nil
	}
	if e.TextDocument.URI == "" {
		return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; textDocument.uri must not be empty", acceptedForms)
	}
	return quarry.Query{InFile: &quarry.InFileQuery{
		File: resolveEntryFile(targetDir, e.TextDocument.URI),
		Name: e.Symbol,
	}}, nil
}

// runTargets executes one for each of targets strictly sequentially in input order, returning
// exactly one result per input, always. Entries are never run concurrently: every facade call
// acquires its own connection, and running a 64-entry array concurrently would mean 64 simultaneous
// dials against the supervised daemon for no gain. Sequential execution is also what makes the
// one-result-per-input-in-input-order contract trivially true.
func runTargets[E any, R any](targets []E, one func(int, E) R) []R {
	results := make([]R, len(targets))
	for i, target := range targets {
		results[i] = one(i, target)
	}
	return results
}
