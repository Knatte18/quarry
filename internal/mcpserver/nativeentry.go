// nativeentry.go declares the entry shape shared by impact and assert_no_callers: the flat
// three-form union a quarry-native tool accepts (plain file+line+character, symbol alone, or
// file+symbol) — plain paths, 1-based line and column, no URI form and no ±1 conversion, unlike
// lspentry.go's own 0-based, file://-tolerant union — plus the per-entry within and except fields
// the two tools' matrices assign it.

package mcpserver

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Knatte18/quarry/internal/cli"
	"github.com/Knatte18/quarry/quarry"
)

// nativeEntry is one target of a quarry-native tool call: impact or assert_no_callers. It accepts
// exactly three forms — File plus Line plus Character, Symbol alone, or File plus Symbol —
// enforced by query, not by the schema. Unlike lspEntry, File is a plain path (never a file://
// URI, and never stripped as one) and Line/Character are 1-based, matching quarry.Position's own
// convention directly rather than LSP's 0-based one. raw carries the entry's original JSON bytes,
// following lspEntry's own per-type raw-capture convention (see lspentry.go's header comment for
// why this is not a shared embedded helper type).
type nativeEntry struct {
	raw json.RawMessage

	// File names the file a position or in-file symbol search resolves against, a plain path
	// (absolute, or relative to the call's targetDir). Required together with Line and Character,
	// and optional together with Symbol (its presence there switches the search from project-wide
	// to file-scoped).
	File string `json:"file,omitempty" jsonschema:"the file a position or in-file symbol search resolves against, a plain path (absolute, or relative to the server's target directory); required with line+character, optional with symbol"`
	// Line is the entry's 1-based line number, used together with File and Character. A nil value
	// means the field was omitted.
	Line *int `json:"line,omitempty" jsonschema:"1-based line number, used together with file and character"`
	// Character is the entry's 1-based character offset, used together with File and Line. A nil
	// value means the field was omitted.
	Character *int `json:"character,omitempty" jsonschema:"1-based character offset, used together with file and line"`
	// Symbol is a symbol name, resolved project-wide when File is absent or within File's own file
	// when present. Mutually exclusive with Line/Character.
	Symbol string `json:"symbol,omitempty" jsonschema:"a symbol name to resolve; project-wide when file is absent, or within file's own file when present; mutually exclusive with line+character"`
	// Within restricts this entry's own result to files within the named directory (relative to
	// the call's targetDir, or absolute). Not every quarry-native tool accepts it.
	Within string `json:"within,omitempty" jsonschema:"restrict this entry's own result to files within this directory (relative to the server's target directory, or absolute)"`
	// Except is a set of file paths (relative to the call's targetDir, or absolute) sanctioned to
	// keep referencing this entry's own symbol without being reported as a violation. Only
	// assert_no_callers accepts it — impact drops this property from its published schema even
	// though the Go type is shared.
	Except []string `json:"except,omitempty" jsonschema:"file paths (relative to the server's target directory, or absolute) sanctioned to reference this entry's own symbol without being reported as a violation; assert_no_callers only"`
}

// nativeEntryAlias is a defined type (not a type alias) with nativeEntry's exact underlying type,
// used only inside UnmarshalJSON so decoding into it does not re-invoke nativeEntry.UnmarshalJSON
// recursively — see lspEntry.UnmarshalJSON's doc comment for why a defined type, not an embedded
// helper, is required.
type nativeEntryAlias nativeEntry

// UnmarshalJSON decodes data into e's exported fields via nativeEntryAlias, then records data
// unchanged into e.raw.
func (e *nativeEntry) UnmarshalJSON(data []byte) error {
	var alias nativeEntryAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	alias.raw = append(json.RawMessage(nil), data...)
	*e = nativeEntry(alias)
	return nil
}

// query converts e into the quarry.Query one of the three legal forms describes: File plus Line
// plus Character yields a Query whose Pos.File is cli.AbsOrJoin(targetDir, e.File) with Line and
// Character taken as given — 1-based on this tool, with neither toOneBased nor stripFileURI
// applied, unlike lspEntry.query's own position form; Symbol alone yields Query{Symbol: e.Symbol};
// File plus Symbol yields an InFile query with File resolved the same way. Every other combination
// returns an error naming the three accepted forms.
func (e nativeEntry) query(targetDir string) (quarry.Query, error) {
	const acceptedForms = "file+line+character, symbol alone, or file+symbol"

	hasPosition := e.Line != nil || e.Character != nil
	hasSymbol := e.Symbol != ""

	if hasPosition && hasSymbol {
		return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; got both a position field and symbol", acceptedForms)
	}
	if !hasPosition && !hasSymbol {
		if e.File != "" {
			return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; got file alone", acceptedForms)
		}
		return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; got neither", acceptedForms)
	}

	if hasPosition {
		if e.File == "" {
			return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; a position requires file", acceptedForms)
		}
		if e.Line == nil || e.Character == nil {
			return quarry.Query{}, fmt.Errorf("mcpserver: entry accepts %s; a position requires both line and character", acceptedForms)
		}
		return quarry.Query{Pos: &quarry.Position{
			File:      cli.AbsOrJoin(targetDir, e.File),
			Line:      *e.Line,
			Character: *e.Character,
		}}, nil
	}

	// hasSymbol, no position: either a plain project-wide search (no file) or an in-file search
	// (file present).
	if e.File == "" {
		return quarry.Query{Symbol: e.Symbol}, nil
	}
	return quarry.Query{InFile: &quarry.InFileQuery{
		File: cli.AbsOrJoin(targetDir, e.File),
		Name: e.Symbol,
	}}, nil
}

// exceptSet reproduces assertNoCallersCommand's own inline --except composition
// (internal/cli/cli.go's RunE) exactly: each path in except is resolved with cli.AbsOrJoin
// against targetDir — the effective absolute target directory, never the process working
// directory — then filepath.Cleaned, and the cleaned paths are the returned map's keys.
//
// cli.FilterUnexpectedCallers compares this map against filepath.Clean(r.File) on
// already-absolute quarry.Reference.File values, so resolving a relative except entry against the
// wrong base makes every exemption silently fail to match and turns a sanctioned wrapper into a
// reported violation.
//
// This is reimplemented rather than exported from internal/cli because
// cli.FilterUnexpectedCallers takes the already-built map as a parameter, leaving no shared
// function this package could call instead of rebuilding it.
func exceptSet(targetDir string, except []string) map[string]bool {
	set := make(map[string]bool, len(except))
	for _, e := range except {
		set[filepath.Clean(cli.AbsOrJoin(targetDir, e))] = true
	}
	return set
}
