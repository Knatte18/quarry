// tools_assert.go implements assert_no_callers, the quarry-native tool wrapping quarry.Callers: a
// CI-shaped gate reporting whether a symbol has any caller outside its declaration and its own
// sanctioned except list. Unlike the CLI's "assert-no-callers", which has no batch envelope at
// all, this tool always answers through the shared array-parameter-name envelope, and it adds a
// "not_found" status the CLI has no counterpart for.

package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/internal/cli"
)

// assertInput is assert_no_callers' call-wide input: the "targets" array of nativeEntry, the two
// call-wide resolution overrides every language-server-backed tool in this package accepts (lang
// and buildTags), plus NoVerify — a whole-check mode, which is why it is call-wide while within
// and except are per-entry.
type assertInput struct {
	// Targets is the array of entries this call resolves, 1 to 64 per call.
	Targets []nativeEntry `json:"targets" jsonschema:"the entries to resolve, 1 to 64 per call"`
	// Lang overrides language detection with this servers.yaml registry key.
	Lang string `json:"lang,omitempty" jsonschema:"override language detection with this servers.yaml registry key"`
	// BuildTags is a comma-separated Go build tag set scoping this call's language server.
	BuildTags string `json:"buildTags,omitempty" jsonschema:"comma-separated Go build tags scoping this call's language server"`
	// NoVerify skips per-caller definition verification for the whole call: an unverifiable
	// reference is kept as a violation rather than dropped when false (the default, fail-closed
	// behaviour).
	NoVerify bool `json:"noVerify,omitempty" jsonschema:"skip per-caller definition verification for the whole call (fail-closed by default: an unverifiable reference is kept as a violation rather than dropped)"`
}

// assertEntry is one target's own result in assert_no_callers' "results" array. A violation is
// not a status: an entry whose symbol resolved is statusFound whether or not it has violating
// callers, and it carries Violation and Callers alongside. Violation is a pointer so false is
// emitted explicitly on a clean found entry rather than dropped by omitempty.
type assertEntry struct {
	// Target echoes this entry's own input JSON, decoded back into any, never a value derived from
	// the result.
	Target any `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, statusAmbiguous, or statusError.
	Status string `json:"status"`
	// Violation is true when one or more unexpected callers remain on a statusFound entry, and
	// false — never omitted — when the check is clean.
	Violation *bool `json:"violation,omitempty"`
	// Callers holds every unexpected caller on a statusFound entry, always a non-nil, possibly-empty
	// slice so it marshals as "[]" rather than "null" when there are none.
	Callers []referenceField `json:"callers,omitempty"`
	// Candidates holds the ambiguous symbol's candidate positions on a statusAmbiguous entry.
	Candidates []string `json:"candidates,omitempty"`
	// Error holds a human-readable message on a statusError entry.
	Error string `json:"error,omitempty"`
}

// assertOutput is assert_no_callers' output envelope.
type assertOutput struct {
	// Results holds one entry per input target, in input order.
	Results []assertEntry `json:"results"`
}

// resolveAssertEntry resolves one nativeEntry for assert_no_callers: it reports an unknown key on
// entry.raw as this entry's own statusError, then parses entry into a quarry.Query, builds
// callCtx.options(lang, query) with SkipVerification set from the call-wide noVerify, and calls
// callersFn. A non-nil error routes through classifyLSPError, so an ambiguous symbol is
// statusAmbiguous and quarry.ErrSymbolNotFoundSentinel is statusNotFound. On success it applies
// cli.FilterWithin when entry.Within is non-empty, builds the exemption map with
// exceptSet(callCtx.TargetDir, entry.Except), calls cli.FilterUnexpectedCallers, and returns
// statusFound with Violation set to whether any violation remains. No resolution key is ever
// emitted, matching the CLI, and no violation ever produces a whole-call failure.
func resolveAssertEntry(ctx context.Context, callCtx callContext, lang string, noVerify bool, entry nativeEntry) assertEntry {
	target := entryTargetAny(entry.raw)

	if unknown := unknownEntryKeys(entry.raw, "file", "line", "character", "symbol", "within", "except"); len(unknown) > 0 {
		return assertEntry{
			Target: target,
			Status: statusError,
			Error:  fmt.Sprintf("mcpserver: entry has unrecognized propert(y/ies) %v; accepted properties are file, line, character, symbol, within, except", unknown),
		}
	}

	query, err := entry.query(callCtx.TargetDir)
	if err != nil {
		return assertEntry{Target: target, Status: statusError, Error: err.Error()}
	}

	opts := callCtx.options(lang, query)
	opts.SkipVerification = noVerify

	refs, declRefs, err := callersFn(ctx, opts)
	if err != nil {
		status, candidates, message := classifyLSPError(err)
		return assertEntry{Target: target, Status: status, Candidates: candidates, Error: message}
	}

	if entry.Within != "" {
		refs = cli.FilterWithin(refs, entry.Within, callCtx.TargetDir)
	}

	exemptions := exceptSet(callCtx.TargetDir, entry.Except)
	violations := cli.FilterUnexpectedCallers(refs, declRefs, exemptions)

	hasViolation := len(violations) > 0
	return assertEntry{
		Target:    target,
		Status:    statusFound,
		Violation: &hasViolation,
		Callers:   referenceFieldsNative(violations),
	}
}

// assertHandler returns assert_no_callers' handler, closing over cfg so a call's per-entry
// resolution reuses the one callContext resolveCall derives for the whole call.
func assertHandler(cfg Config) mcp.ToolHandlerFor[assertInput, assertOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in assertInput) (*mcp.CallToolResult, assertOutput, error) {
		callCtx, err := resolveCall(cfg, in.BuildTags)
		if err != nil {
			return nil, assertOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, entry nativeEntry) assertEntry {
			return resolveAssertEntry(ctx, callCtx, in.Lang, in.NoVerify, entry)
		})

		return nil, assertOutput{Results: results}, nil
	}
}

// registerAssertTool registers assert_no_callers on s, deriving its input schema from assertInput
// and its output schema from assertOutput, then registering the handler assertHandler builds for
// cfg.
func registerAssertTool(s *mcp.Server, cfg Config) error {
	inputSchema, err := inputSchemaFor[assertInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register assert_no_callers: %w", err)
	}
	outputSchema, err := outputSchemaFor[assertOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register assert_no_callers: %w", err)
	}

	mcp.AddTool(s, &mcp.Tool{
		Name: "assert_no_callers",
		Description: "assert_no_callers reports 1-based line and character on both input and output — " +
			"this tool never applies LSP's 0-based convention or any ±1 conversion. Each entry accepts " +
			"one of three forms: file+line+character, symbol alone, or file+symbol. \"except\" and " +
			"\"within\" are per-entry, because each names paths sanctioned for that one symbol. Up to " +
			"64 entries per call.",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, assertHandler(cfg))

	return nil
}
