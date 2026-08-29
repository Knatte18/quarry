// tools_impact.go implements impact, the quarry-native tool wrapping quarry.Impact: every caller
// of a symbol, each paired with its own enclosing declaration. impact shares nativeEntry with
// assert_no_callers but accepts a per-entry "within" alone — its published schema drops "except"
// even though the Go type carries it, since per-entry-vs-call-wide's matrix gives impact no
// call-wide or per-entry except at all.

package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/internal/cli"
)

// impactInput is impact's call-wide input: the "targets" array of nativeEntry plus the three
// call-wide resolution overrides every language-server-backed tool in this package accepts.
type impactInput struct {
	// Targets is the array of entries this call resolves, 1 to 64 per call.
	Targets []nativeEntry `json:"targets" jsonschema:"the entries to resolve, 1 to 64 per call"`
	// Lang overrides language detection with this servers.yaml registry key.
	Lang string `json:"lang,omitempty" jsonschema:"override language detection with this servers.yaml registry key"`
	// BuildTags is a comma-separated Go build tag set scoping this call's language server.
	BuildTags string `json:"buildTags,omitempty" jsonschema:"comma-separated Go build tags scoping this call's language server"`
	// TargetDir overrides the server's launch-default project directory for this call.
	TargetDir string `json:"targetDir,omitempty" jsonschema:"project directory to detect the language in and root this call at, overriding the server's launch default"`
}

// impactEntry is one target's own result in impact's "results" array. Result nests the marshalled
// quarry.ImpactResult under its own key rather than flattening it: quarry.ImpactResult.Target
// marshals to a top-level "target", and flattening it here the way runBatch does in
// internal/cli/cli.go would have it overwrite this entry's own "target" — the echoed input, which
// always wins and always means "the input entry, echoed".
type impactEntry struct {
	// Target echoes this entry's own input JSON, decoded back into any, never a value derived from
	// the result.
	Target any `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, statusAmbiguous, or statusError.
	Status string `json:"status"`
	// Resolution is resolutionComplete on a statusFound entry, and absent otherwise.
	Resolution string `json:"resolution,omitempty"`
	// Result holds the marshalled quarry.ImpactResult on a statusFound entry, nested under this key
	// so its own "target" field never collides with this entry's own echoed "target" above.
	Result map[string]any `json:"result,omitempty"`
	// Candidates holds the ambiguous symbol's candidate positions on a statusAmbiguous entry.
	Candidates []string `json:"candidates,omitempty"`
	// Error holds a human-readable message on a statusError entry.
	Error string `json:"error,omitempty"`
}

// impactOutput is impact's output envelope.
type impactOutput struct {
	// Results holds one entry per input target, in input order.
	Results []impactEntry `json:"results"`
}

// resolveImpactEntry resolves one nativeEntry for impact: it reports an unknown key on entry.raw
// (except is not accepted by this tool, even though nativeEntry carries the field for
// assert_no_callers) as this entry's own statusError, then parses entry into a quarry.Query, calls
// impactFn with callCtx.options(lang, query), applies cli.FilterImpactWithin when entry.Within is
// non-empty and the call succeeded, marshals the result with cli.StructToFields, and maps the
// outcome through classifyLSPError. Positions inside the marshalled result are left exactly as the
// engine produced them — 1-based, unconverted.
func resolveImpactEntry(ctx context.Context, callCtx callContext, lang string, entry nativeEntry) impactEntry {
	target := entryTargetAny(entry.raw)

	if unknown := unknownEntryKeys(entry.raw, "file", "line", "character", "symbol", "within"); len(unknown) > 0 {
		return impactEntry{
			Target: target,
			Status: statusError,
			Error:  fmt.Sprintf("mcpserver: entry has unrecognized propert(y/ies) %v; accepted properties are file, line, character, symbol, within", unknown),
		}
	}

	query, err := entry.query(callCtx.TargetDir)
	if err != nil {
		return impactEntry{Target: target, Status: statusError, Error: err.Error()}
	}

	result, err := impactFn(ctx, callCtx.options(lang, query))
	if err == nil && entry.Within != "" {
		result = cli.FilterImpactWithin(result, entry.Within, callCtx.TargetDir)
	}

	status, candidates, message := classifyLSPError(err)
	if status != statusFound {
		return impactEntry{Target: target, Status: status, Candidates: candidates, Error: message}
	}

	fields, marshalErr := cli.StructToFields(result)
	if marshalErr != nil {
		return impactEntry{Target: target, Status: statusError, Error: rewordMarshalFailure(marshalErr)}
	}
	return impactEntry{Target: target, Status: statusFound, Resolution: resolutionComplete, Result: fields}
}

// impactHandler returns impact's handler, closing over cfg so a call's per-entry resolution reuses
// the one callContext resolveCall derives for the whole call.
func impactHandler(cfg Config) mcp.ToolHandlerFor[impactInput, impactOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in impactInput) (*mcp.CallToolResult, impactOutput, error) {
		callCtx, err := resolveCall(cfg, in.TargetDir, in.BuildTags)
		if err != nil {
			return nil, impactOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, entry nativeEntry) impactEntry {
			return resolveImpactEntry(ctx, callCtx, in.Lang, entry)
		})

		return nil, impactOutput{Results: results}, nil
	}
}

// registerImpactTool registers impact on s, deriving its input schema from impactInput and its
// output schema from impactOutput, then registering the handler impactHandler builds for cfg.
//
// dropEntryProperty removes "except" from the derived input schema's targets item schema: nativeEntry
// is shared with assert_no_callers and therefore carries an Except field, but impact does not accept
// except — per-entry-vs-call-wide's matrix gives impact a per-entry set of within alone, and a tool
// exposes exactly its matrix row and nothing else. Pruning the published property is what makes the
// schema agree with resolveImpactEntry's own unknownEntryKeys rejection; the shared Go type is kept
// so both tools parse through one nativeEntry.query.
func registerImpactTool(s *mcp.Server, cfg Config) error {
	inputSchema, err := inputSchemaFor[impactInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register impact: %w", err)
	}
	dropEntryProperty(inputSchema, "except")

	outputSchema, err := outputSchemaFor[impactOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register impact: %w", err)
	}

	mcp.AddTool(s, &mcp.Tool{
		Name: "impact",
		Description: "impact reports 1-based line and character on both input and output — this tool " +
			"never applies LSP's 0-based convention or any ±1 conversion. Each entry accepts one of " +
			"three forms: file+line+character, symbol alone, or file+symbol. The marshalled result is " +
			"nested under \"result\" on each found entry so it cannot collide with the entry's own " +
			"echoed \"target\". Up to 64 entries per call.",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, impactHandler(cfg))

	return nil
}
