// translate.go implements the URI-stripping and 0-based/1-based conversions every handler needs
// before it can call the facade or return a wire-shaped result: quarry.Query.Pos.File and
// quarry.InFileQuery.File must be absolute paths, never file:// URIs, and an LSP-mirrored tool's
// position must round-trip through the 0-based line/character convention LSP itself uses, even
// though quarry.Position is 1-based on both axes.

package mcpserver

import (
	"strings"

	"github.com/Knatte18/quarry/internal/cli"
)

// stripFileURI removes a leading "file://" prefix from s, leaving everything else unchanged.
// A "file:///abs/path" input yields "/abs/path"; a plain path passes through untouched.
func stripFileURI(s string) string {
	return strings.TrimPrefix(s, "file://")
}

// resolveEntryFile resolves raw (a tool entry's file field, possibly a file:// URI) against
// targetDir, which is always absolute by the time this is called — cli.AbsOrJoin's contract, and
// this package's own callers only ever pass ResolveLaunchTargetDir's or effectiveTargetDir's
// result.
//
// The result must itself be absolute: quarry.Query.Pos.File and quarry.InFileQuery.File are turned
// into file:// URIs by query.References with no further resolution, so a relative path here would
// produce a malformed URI rather than fail loudly.
func resolveEntryFile(targetDir, raw string) string {
	return cli.AbsOrJoin(targetDir, stripFileURI(raw))
}

// toOneBased converts an LSP-mirrored tool's incoming 0-based line or character value to the
// 1-based value quarry.Position expects.
//
// The engine's character unit is asymmetric across this boundary:
// internal/quarryengine/lsp/wire.go's ToPosition converts a 1-based byte column into a 0-based
// UTF-16 character inbound (re-reading the file to do it correctly), while
// internal/quarryengine/query/refs.go's toSortedReferences does a naive "+1" outbound with no
// UTF-16 accounting at all. This layer deliberately reproduces the CLI's existing naive behaviour
// rather than fixing it locally: a returned character must round-trip straight back into a
// following call the same naive way, or every non-ASCII position this server hands back would stop
// working as a follow-up query.
func toOneBased(v int) int {
	return v + 1
}

// toZeroBased converts a quarry.Position (or Reference/SymbolMatch) 1-based line or character
// value to the 0-based value an LSP-mirrored tool reports. See toOneBased's doc comment for the
// asymmetric character-unit behaviour this deliberately reproduces rather than fixes.
func toZeroBased(v int) int {
	return v - 1
}
