// tools_lsp.go implements textDocument_definition and textDocument_references, the two
// LSP-mirrored tools whose per-entry shape is a definition or reference list: both share lspInput's
// call-wide "targets" array plus lang/buildTags/targetDir overrides, and both resolve each entry
// through resolveLSPEntry, the one function that runs the unknown-key check, the query parse, the
// facade call, the per-entry --within filter, and the error classification every LSP-mirrored
// lookup needs. Only the wrapping into definitionEntry vs referencesEntry, and which facade seam
// variable (definitionFn vs referencesFn) is called, differs between the two tools.

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/internal/cli"
	"github.com/Knatte18/quarry/quarry"
)

// lspInput is the call-wide input every LSP-mirrored tool (textDocument_definition,
// textDocument_references, workspace_symbol) shares: the "targets" array plus the three call-wide
// resolution overrides every language-server-backed tool in this package accepts.
type lspInput struct {
	// Targets is the array of entries this call resolves, 1 to 64 per call.
	Targets []lspEntry `json:"targets" jsonschema:"the entries to resolve, 1 to 64 per call"`
	// Lang overrides language detection with this servers.yaml registry key, validated through
	// quarry.LoadRegistry and quarry.DetectLanguage.
	Lang string `json:"lang,omitempty" jsonschema:"override language detection with this servers.yaml registry key"`
	// BuildTags is a comma-separated Go build tag set scoping this call's language server.
	BuildTags string `json:"buildTags,omitempty" jsonschema:"comma-separated Go build tags scoping this call's language server"`
	// TargetDir overrides the server's launch-default project directory for this call.
	TargetDir string `json:"targetDir,omitempty" jsonschema:"project directory to detect the language in and root this call at, overriding the server's launch default"`
}

// definitionEntry is one target's own result in textDocument_definition's "results" array. Target
// and Status are the only fields every entry carries; the rest are populated only for the outcome
// they describe.
type definitionEntry struct {
	// Target echoes this entry's own input JSON, decoded back into any, never a value derived from
	// the result.
	Target any `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, statusAmbiguous, or statusError.
	Status string `json:"status"`
	// Resolution is resolutionComplete on a statusFound entry, and absent otherwise.
	Resolution string `json:"resolution,omitempty"`
	// Definitions holds the resolved definition positions on a statusFound entry.
	Definitions []referenceField `json:"definitions,omitempty"`
	// Candidates holds the ambiguous symbol's candidate positions on a statusAmbiguous entry.
	Candidates []string `json:"candidates,omitempty"`
	// Error holds a human-readable message on a statusError entry.
	Error string `json:"error,omitempty"`
}

// referencesEntry is one target's own result in textDocument_references's "results" array,
// identical to definitionEntry except its results key is "references" rather than "definitions".
type referencesEntry struct {
	// Target echoes this entry's own input JSON, decoded back into any, never a value derived from
	// the result.
	Target any `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, statusAmbiguous, or statusError.
	Status string `json:"status"`
	// Resolution is resolutionComplete on a statusFound entry, and absent otherwise.
	Resolution string `json:"resolution,omitempty"`
	// References holds the resolved reference positions on a statusFound entry.
	References []referenceField `json:"references,omitempty"`
	// Candidates holds the ambiguous symbol's candidate positions on a statusAmbiguous entry.
	Candidates []string `json:"candidates,omitempty"`
	// Error holds a human-readable message on a statusError entry.
	Error string `json:"error,omitempty"`
}

// definitionOutput is textDocument_definition's output envelope.
type definitionOutput struct {
	// Results holds one entry per input target, in input order.
	Results []definitionEntry `json:"results"`
}

// referencesOutput is textDocument_references's output envelope.
type referencesOutput struct {
	// Results holds one entry per input target, in input order.
	Results []referencesEntry `json:"results"`
}

// lspEntryResult is one entry's resolved outcome, before it is wrapped into definitionEntry's or
// referencesEntry's own field names by the caller.
type lspEntryResult struct {
	Status     string
	Resolution string
	Refs       []referenceField
	Candidates []string
	Error      string
}

// entryTargetAny decodes raw — an entry's own captured input JSON — back into any, for the "target"
// key every result entry echoes. A decode failure (which resolveLSPEntry itself never allows,
// since raw is always valid JSON captured by lspEntry.UnmarshalJSON) yields nil.
func entryTargetAny(raw json.RawMessage) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}

// resolveLSPEntry resolves one lspEntry shared by textDocument_definition and
// textDocument_references: it reports an unknown key on entry.raw as statusError first, then parses
// entry into a quarry.Query, then calls call with callCtx.options(lang, query), applies
// cli.FilterWithin when entry.Within is non-empty and the call succeeded, and maps the outcome
// through classifyLSPError. call is definitionFn or referencesFn, passed by the caller so this
// function stays agnostic to which of the two tools is resolving the entry.
func resolveLSPEntry(ctx context.Context, callCtx callContext, lang string, entry lspEntry, call func(context.Context, quarry.Options) ([]quarry.Reference, error)) lspEntryResult {
	if unknown := unknownEntryKeys(entry.raw, "textDocument", "position", "symbol", "within"); len(unknown) > 0 {
		return lspEntryResult{
			Status: statusError,
			Error:  fmt.Sprintf("mcpserver: entry has unrecognized propert(y/ies) %v; accepted properties are textDocument, position, symbol, within", unknown),
		}
	}

	query, err := entry.query(callCtx.TargetDir)
	if err != nil {
		return lspEntryResult{Status: statusError, Error: err.Error()}
	}

	refs, err := call(ctx, callCtx.options(lang, query))
	if err == nil && entry.Within != "" {
		refs = cli.FilterWithin(refs, entry.Within, callCtx.TargetDir)
	}

	status, candidates, message := classifyLSPError(err)
	if status != statusFound {
		return lspEntryResult{Status: status, Candidates: candidates, Error: message}
	}
	return lspEntryResult{Status: statusFound, Resolution: resolutionComplete, Refs: referenceFieldsWire(refs)}
}

// definitionHandler returns textDocument_definition's handler, closing over cfg so a call's
// per-entry resolution reuses the one callContext resolveCall derives for the whole call.
func definitionHandler(cfg Config) mcp.ToolHandlerFor[lspInput, definitionOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in lspInput) (*mcp.CallToolResult, definitionOutput, error) {
		callCtx, err := resolveCall(cfg, in.TargetDir, in.BuildTags)
		if err != nil {
			return nil, definitionOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, entry lspEntry) definitionEntry {
			r := resolveLSPEntry(ctx, callCtx, in.Lang, entry, definitionFn)
			return definitionEntry{
				Target:      entryTargetAny(entry.raw),
				Status:      r.Status,
				Resolution:  r.Resolution,
				Definitions: r.Refs,
				Candidates:  r.Candidates,
				Error:       r.Error,
			}
		})

		return nil, definitionOutput{Results: results}, nil
	}
}

// referencesHandler returns textDocument_references's handler, closing over cfg so a call's
// per-entry resolution reuses the one callContext resolveCall derives for the whole call.
func referencesHandler(cfg Config) mcp.ToolHandlerFor[lspInput, referencesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in lspInput) (*mcp.CallToolResult, referencesOutput, error) {
		callCtx, err := resolveCall(cfg, in.TargetDir, in.BuildTags)
		if err != nil {
			return nil, referencesOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, entry lspEntry) referencesEntry {
			r := resolveLSPEntry(ctx, callCtx, in.Lang, entry, referencesFn)
			return referencesEntry{
				Target:     entryTargetAny(entry.raw),
				Status:     r.Status,
				Resolution: r.Resolution,
				References: r.Refs,
				Candidates: r.Candidates,
				Error:      r.Error,
			}
		})

		return nil, referencesOutput{Results: results}, nil
	}
}

// registerLSPTools registers textDocument_definition and textDocument_references on s, deriving
// each tool's input schema from lspInput and its own output schema, then registering the handler
// definitionHandler/referencesHandler builds for cfg.
func registerLSPTools(s *mcp.Server, cfg Config) error {
	definitionInputSchema, err := inputSchemaFor[lspInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register textDocument_definition: %w", err)
	}
	definitionOutSchema, err := outputSchemaFor[definitionOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register textDocument_definition: %w", err)
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "textDocument_definition",
		Description: "textDocument_definition reports 0-based line and character on both input and " +
			"output. Each entry accepts one of three forms: textDocument+position, symbol alone, or " +
			"textDocument+symbol. Up to 64 entries per call.",
		InputSchema:  definitionInputSchema,
		OutputSchema: definitionOutSchema,
	}, definitionHandler(cfg))

	referencesInputSchema, err := inputSchemaFor[lspInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register textDocument_references: %w", err)
	}
	referencesOutSchema, err := outputSchemaFor[referencesOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register textDocument_references: %w", err)
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "textDocument_references",
		Description: "textDocument_references reports 0-based line and character on both input and " +
			"output. Each entry accepts one of three forms: textDocument+position, symbol alone, or " +
			"textDocument+symbol. Up to 64 entries per call.",
		InputSchema:  referencesInputSchema,
		OutputSchema: referencesOutSchema,
	}, referencesHandler(cfg))

	return nil
}
