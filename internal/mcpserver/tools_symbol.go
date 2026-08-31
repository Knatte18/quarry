// tools_symbol.go implements workspace_symbol, the one LSP-mirrored tool whose entry carries no
// position or textDocument at all: a bare name search against the language server's workspace/symbol
// request. Its entry type, input type, and output shapes are declared separately from card 14's
// lspEntry/lspInput/definitionOutput/referencesOutput family because a symbolEntry accepts exactly
// one property, and reusing lspEntry here would advertise textDocument/position/within properties
// this tool cannot accept.

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/quarry"
)

// symbolEntry is one target of a workspace_symbol call: a bare symbol name to search project-wide.
// raw carries the entry's original JSON bytes, following lspEntry's own per-type raw-capture
// convention (see lspentry.go's header comment for why this is not a shared embedded helper type).
type symbolEntry struct {
	raw json.RawMessage

	// Query is the symbol name to search for, using LSP's own WorkspaceSymbolParams field name.
	Query string `json:"query,omitempty" jsonschema:"the symbol name to search for, project-wide"`
}

// symbolEntryAlias is a defined type (not a type alias) with symbolEntry's exact underlying type,
// used only inside UnmarshalJSON so decoding into it does not re-invoke symbolEntry.UnmarshalJSON
// recursively — see lspEntry.UnmarshalJSON's doc comment for why a defined type, not an embedded
// helper, is required.
type symbolEntryAlias symbolEntry

// UnmarshalJSON decodes data into e's exported fields via symbolEntryAlias, then records data
// unchanged into e.raw.
func (e *symbolEntry) UnmarshalJSON(data []byte) error {
	var alias symbolEntryAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	alias.raw = append(json.RawMessage(nil), data...)
	*e = symbolEntry(alias)
	return nil
}

// symbolInput is workspace_symbol's call-wide input: the same call-wide resolution overrides
// lspInput carries, but with no "within" property — the CLI registers none for "symbol", and
// query.Symbol has nothing to filter per-file against.
type symbolInput struct {
	// Targets is the array of entries this call resolves, 1 to 64 per call.
	Targets []symbolEntry `json:"targets" jsonschema:"the entries to resolve, 1 to 64 per call"`
	// Lang overrides language detection with this servers.yaml registry key.
	Lang string `json:"lang,omitempty" jsonschema:"override language detection with this servers.yaml registry key"`
	// BuildTags is a comma-separated Go build tag set scoping this call's language server.
	BuildTags string `json:"buildTags,omitempty" jsonschema:"comma-separated Go build tags scoping this call's language server"`
}

// symbolMatchEntry is one target's own result in workspace_symbol's "results" array. It carries no
// "resolution" key — classifySymbolError never sets that marker, and adding it here would claim an
// exhaustive language-server resolution the CLI does not claim — and no "candidates" key either,
// since this tool never emits an "ambiguous" status.
type symbolMatchEntry struct {
	// Target echoes this entry's own input JSON, decoded back into any, never a value derived from
	// the result.
	Target any `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, or statusError.
	Status string `json:"status"`
	// Symbols holds the resolved workspace/symbol matches on a statusFound entry. omitzero, not
	// omitempty, matching definitionEntry.Definitions' rule: symbolFieldsWire always returns a
	// non-nil slice, and a found entry must emit its "symbols" key even when the slice is empty.
	Symbols []symbolField `json:"symbols,omitzero"`
	// Error holds a human-readable message on a statusError entry.
	Error string `json:"error,omitempty"`
}

// symbolOutput is workspace_symbol's output envelope.
type symbolOutput struct {
	// Results holds one entry per input target, in input order.
	Results []symbolMatchEntry `json:"results"`
}

// resolveSymbolEntry resolves one symbolEntry: it reports an unknown key on entry.raw, or an empty
// Query, as this entry's own statusError; otherwise it calls symbolFn with
// callCtx.options(lang, quarry.Query{Symbol: entry.Query}) and maps the outcome with
// classifySymbolError — not classifyLSPError, which would add an "ambiguous" branch quarry.Symbol
// never reaches.
func resolveSymbolEntry(ctx context.Context, callCtx callContext, lang string, entry symbolEntry) symbolMatchEntry {
	target := entryTargetAny(entry.raw)

	if unknown := unknownEntryKeys(entry.raw, "query"); len(unknown) > 0 {
		return symbolMatchEntry{
			Target: target,
			Status: statusError,
			Error:  fmt.Sprintf("mcpserver: entry has unrecognized propert(y/ies) %v; the only accepted property is query", unknown),
		}
	}
	if entry.Query == "" {
		return symbolMatchEntry{
			Target: target,
			Status: statusError,
			Error:  "mcpserver: entry's query must not be empty; query is the only accepted property",
		}
	}

	matches, err := symbolFn(ctx, callCtx.options(lang, quarry.Query{Symbol: entry.Query}))
	status, message := classifySymbolError(err)
	if status != statusFound {
		return symbolMatchEntry{Target: target, Status: status, Error: message}
	}
	return symbolMatchEntry{Target: target, Status: statusFound, Symbols: symbolFieldsWire(matches)}
}

// symbolHandler returns workspace_symbol's handler, closing over cfg so a call's per-entry
// resolution reuses the one callContext resolveCall derives for the whole call.
func symbolHandler(cfg Config) mcp.ToolHandlerFor[symbolInput, symbolOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in symbolInput) (*mcp.CallToolResult, symbolOutput, error) {
		callCtx, err := resolveCall(cfg, in.BuildTags)
		if err != nil {
			return nil, symbolOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, entry symbolEntry) symbolMatchEntry {
			return resolveSymbolEntry(ctx, callCtx, in.Lang, entry)
		})

		return nil, symbolOutput{Results: results}, nil
	}
}

// registerSymbolTool registers workspace_symbol on s, deriving its input schema from symbolInput and
// its output schema from symbolOutput, then registering the handler symbolHandler builds for cfg.
func registerSymbolTool(s *mcp.Server, cfg Config) error {
	inputSchema, err := inputSchemaFor[symbolInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register workspace_symbol: %w", err)
	}
	outputSchema, err := outputSchemaFor[symbolOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register workspace_symbol: %w", err)
	}

	mcp.AddTool(s, &mcp.Tool{
		Name: "workspace_symbol",
		Description: "workspace_symbol reports 0-based line and character in its results. " +
			"\"query\" is the only accepted entry property. Matching is the language server's own " +
			"fuzzy search: a short query like \"Run\" returns every loosely-matching symbol up to " +
			"the server's result cap, including symbols from dependency modules outside the " +
			"project. A discovery tool for when a symbol's file is not yet known — once it is, " +
			"prefer textDocument_definition/textDocument_references, which resolve one symbol " +
			"exactly. Result names may be qualified as Type.Method; strip to the bare method name " +
			"before reusing one as another tool's symbol input. Up to 64 entries per call.",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, symbolHandler(cfg))

	return nil
}
