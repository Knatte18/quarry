// tools_toc.go implements toc_file and toc_dir, the two tree-sitter-backed tools wrapping
// quarry.TOCFile and quarry.TOCDir. Both take plain-string "targets" entries rather than objects,
// both validate "lang" against quarry.TOCLanguages() rather than the servers.yaml registry, and
// neither loads that registry or resolves a state directory — each handler calls
// effectiveTargetDir and tocPreflight directly and never resolveCall, following
// internal/mcpserver/callcontext.go's header comment for why a malformed servers.yaml must not
// fail a toc call.

package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Knatte18/quarry/internal/cli"
	"github.com/Knatte18/quarry/quarry"
)

// tocFileInput is toc_file's call-wide input: a "targets" array of plain path strings, plus the
// per-call overrides toc_file accepts. Unlike lspInput/impactInput, there is no buildTags: toc is
// tree-sitter-backed, never loads the server registry, and registers no --build-tags.
type tocFileInput struct {
	// Targets is the array of plain file paths this call resolves, 1 to 64 per call.
	Targets []string `json:"targets" jsonschema:"the file paths to resolve, 1 to 64 per call"`
	// Lang overrides language detection with this explicit language name, validated against
	// quarry.TOCLanguages() rather than a servers.yaml registry key.
	Lang string `json:"lang,omitempty" jsonschema:"override language detection with this explicit language name, validated against toc's own supported language set rather than a servers.yaml registry key"`
	// DocSentences controls how much of each symbol's docstring reaches "docstring" in the result:
	// 0 omits the key entirely, N keeps the first N sentences, and "all" keeps the docstring
	// unchanged. Resolved per entry against that entry's own resolved file's parent directory.
	DocSentences docSentences `json:"docSentences,omitempty" jsonschema:"number of leading docstring sentences to emit (0 omits the key entirely), or \"all\"; resolved per entry against that entry's own file's parent directory"`
	// TargetDir overrides the server's launch-default project directory for this call, used only
	// to resolve a relative target — toc never detects a project or loads a registry against it.
	TargetDir string `json:"targetDir,omitempty" jsonschema:"base directory to resolve a relative target against, overriding the server's launch default"`
}

// tocDirInput is toc_dir's call-wide input: a "targets" array of plain directory-path strings,
// plus the per-call overrides toc_dir accepts. It carries no docSentences property: --doc-sentences
// is registered on "toc file" only in the CLI, deliberately, because "toc dir" emits headers and
// never docstrings.
type tocDirInput struct {
	// Targets is the array of plain directory paths this call resolves, 1 to 64 per call.
	Targets []string `json:"targets" jsonschema:"the directory paths to resolve, 1 to 64 per call"`
	// Lang restricts the listing to this language's own extensions, validated against
	// quarry.TOCLanguages() rather than a servers.yaml registry key.
	Lang string `json:"lang,omitempty" jsonschema:"restrict the listing to this language's own extensions, validated against toc's own supported language set rather than a servers.yaml registry key"`
	// TargetDir overrides the server's launch-default project directory for this call, used only
	// to resolve a relative target — toc never detects a project or loads a registry against it.
	TargetDir string `json:"targetDir,omitempty" jsonschema:"base directory to resolve a relative target against, overriding the server's launch default"`
}

// tocFileEntry is one target's own result in toc_file's "results" array. It declares no
// "resolution" or "candidates" key: toc consults no language server and "ambiguous" is
// unreachable here.
type tocFileEntry struct {
	// Target echoes this entry's own input path string verbatim.
	Target string `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, or statusError. statusAmbiguous
	// never appears here.
	Status string `json:"status"`
	// Result holds the marshalled quarry.TOCFileResult on a statusFound entry, nested under this
	// key — the same deliberate divergence batch 4 records for impact, applied here so the
	// envelope's "target" stays unambiguously "the input entry, echoed".
	Result map[string]any `json:"result,omitempty"`
	// Error holds a human-readable message on a statusError or statusNotFound entry.
	Error string `json:"error,omitempty"`
}

// tocDirEntry is one target's own result in toc_dir's "results" array. It declares no
// "resolution" or "candidates" key, for the same reason tocFileEntry does not.
type tocDirEntry struct {
	// Target echoes this entry's own input path string verbatim.
	Target string `json:"target"`
	// Status is this entry's outcome: statusFound, statusNotFound, or statusError.
	Status string `json:"status"`
	// Files holds the marshalled per-file entries on a statusFound entry, built through
	// cli.TOCDirEntries so each file's "path" stays caller-relative and round-trips into a
	// following toc_file call.
	Files []any `json:"files,omitempty"`
	// Error holds a human-readable message on a statusError or statusNotFound entry.
	Error string `json:"error,omitempty"`
}

// tocFileOutput is toc_file's output envelope.
type tocFileOutput struct {
	// Results holds one entry per input target, in input order.
	Results []tocFileEntry `json:"results"`
}

// tocDirOutput is toc_dir's output envelope.
type tocDirOutput struct {
	// Results holds one entry per input target, in input order.
	Results []tocDirEntry `json:"results"`
}

// resolveTOCFileEntry resolves one toc_file target: it resolves arg against targetDir, stats it
// with tocStat(abs, false) (a directory is toc_file's own wrong-type outcome), resolves the
// effective DocSentences value against the resolved file's own parent directory — never
// targetDir, because reusing that would pick up a different .quarry.yaml than the CLI does for the
// identical argument — then calls tocFileFn and maps the outcome through classifyTOCError. A
// cli.StructToFields failure is this entry's own statusError carrying the message verbatim:
// rewordMarshalFailure is impact-only and must not be applied here, since the "toc: " prefix is
// correctly attributed for a toc call.
func resolveTOCFileEntry(arg, targetDir, lang, docString string) tocFileEntry {
	abs := cli.ResolveTOCPath(targetDir, arg)

	status, message, err := tocStat(abs, false)
	if err != nil {
		return tocFileEntry{Target: arg, Status: status, Error: message}
	}

	resolved, err := cli.ResolveDocSentences(docString, filepath.Dir(abs))
	if err != nil {
		return tocFileEntry{Target: arg, Status: statusError, Error: err.Error()}
	}

	result, err := tocFileFn(abs, lang, quarry.TOCOptions{DocSentences: resolved})
	if err != nil {
		status, message := classifyTOCError(err)
		return tocFileEntry{Target: arg, Status: status, Error: message}
	}

	fields, err := cli.StructToFields(result)
	if err != nil {
		return tocFileEntry{Target: arg, Status: statusError, Error: err.Error()}
	}
	return tocFileEntry{Target: arg, Status: statusFound, Result: fields}
}

// resolveTOCDirEntry resolves one toc_dir target: it resolves arg against targetDir, stats it with
// tocStat(abs, true) (a file is toc_dir's own wrong-type outcome), then calls tocDirFn and maps the
// outcome through classifyTOCError. A successful call is composed through cli.TOCDirEntries,
// passing arg — the caller-written argument, never abs — so each file's composed "path" stays
// caller-relative and round-trips into a following toc_file call; cli.StructToFields alone is not
// sufficient here, because toc.DirEntry.Name carries json:"-" and the marshalled entries would
// otherwise carry neither "name" nor "path". A cli.TOCDirEntries failure is this entry's own
// statusError carrying the message verbatim, exactly as tocDirOne disposes of the same failure.
func resolveTOCDirEntry(arg, targetDir, lang string) tocDirEntry {
	abs := cli.ResolveTOCPath(targetDir, arg)

	status, message, err := tocStat(abs, true)
	if err != nil {
		return tocDirEntry{Target: arg, Status: status, Error: message}
	}

	result, err := tocDirFn(abs, lang)
	if err != nil {
		status, message := classifyTOCError(err)
		return tocDirEntry{Target: arg, Status: status, Error: message}
	}

	files, err := cli.TOCDirEntries(arg, result)
	if err != nil {
		return tocDirEntry{Target: arg, Status: statusError, Error: err.Error()}
	}
	return tocDirEntry{Target: arg, Status: statusFound, Files: files}
}

// tocFileHandler returns toc_file's handler, closing over cfg. It calls effectiveTargetDir and
// tocPreflight directly, never resolveCall, so a malformed servers.yaml or an unresolvable
// registry/state directory never fails a toc_file call — tocFileFn never consults either.
func tocFileHandler(cfg Config) mcp.ToolHandlerFor[tocFileInput, tocFileOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, in tocFileInput) (*mcp.CallToolResult, tocFileOutput, error) {
		targetDir, err := effectiveTargetDir(cfg, in.TargetDir)
		if err != nil {
			return nil, tocFileOutput{}, err
		}

		docString, err := tocPreflight(in.Lang, in.DocSentences)
		if err != nil {
			return nil, tocFileOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, arg string) tocFileEntry {
			return resolveTOCFileEntry(arg, targetDir, in.Lang, docString)
		})

		return nil, tocFileOutput{Results: results}, nil
	}
}

// tocDirHandler returns toc_dir's handler, closing over cfg. It calls effectiveTargetDir and
// tocPreflight directly, never resolveCall, for the same reason tocFileHandler does.
func tocDirHandler(cfg Config) mcp.ToolHandlerFor[tocDirInput, tocDirOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, in tocDirInput) (*mcp.CallToolResult, tocDirOutput, error) {
		targetDir, err := effectiveTargetDir(cfg, in.TargetDir)
		if err != nil {
			return nil, tocDirOutput{}, err
		}

		// docSentences has no property on tocDirInput, so its zero value never carries a value —
		// tocPreflight's own doc.value() check short-circuits before ever reaching
		// cli.ParseDocSentences.
		if _, err := tocPreflight(in.Lang, docSentences{}); err != nil {
			return nil, tocDirOutput{}, err
		}

		results := runTargets(in.Targets, func(_ int, arg string) tocDirEntry {
			return resolveTOCDirEntry(arg, targetDir, in.Lang)
		})

		return nil, tocDirOutput{Results: results}, nil
	}
}

// registerTOCTools registers toc_file and toc_dir on s, deriving each tool's input schema from
// tocFileInput/tocDirInput and its own output schema, then registering the handler
// tocFileHandler/tocDirHandler builds for cfg — following registerLSPTools' and
// registerImpactTool's own registration shape.
func registerTOCTools(s *mcp.Server, cfg Config) error {
	fileInputSchema, err := inputSchemaFor[tocFileInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register toc_file: %w", err)
	}
	fileOutputSchema, err := outputSchemaFor[tocFileOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register toc_file: %w", err)
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "toc_file",
		Description: "toc_file reports 1-based, inclusive line numbers on every symbol's " +
			"\"start\"/\"sigend\"/\"end\" range. Each entry is a plain file path, not an object. " +
			"The marshalled quarry.TOCFileResult is nested under \"result\" on each found entry. " +
			"lang, when given, is validated against toc's own supported language set, never a " +
			"servers.yaml registry key. Up to 64 entries per call.",
		InputSchema:  fileInputSchema,
		OutputSchema: fileOutputSchema,
	}, tocFileHandler(cfg))

	dirInputSchema, err := inputSchemaFor[tocDirInput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register toc_dir: %w", err)
	}
	dirOutputSchema, err := outputSchemaFor[tocDirOutput]()
	if err != nil {
		return fmt.Errorf("mcpserver: register toc_dir: %w", err)
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "toc_dir",
		Description: "toc_dir reports 1-based, inclusive line numbers on every listed file's own " +
			"toc_file result shape. Each entry is a plain directory path, not an object. Each " +
			"found entry's \"files\" carries a caller-relative \"path\" per file, composed against " +
			"the argument as written, so it round-trips straight into a following toc_file call. " +
			"lang, when given, is validated against toc's own supported language set, never a " +
			"servers.yaml registry key. Up to 64 entries per call.",
		InputSchema:  dirInputSchema,
		OutputSchema: dirOutputSchema,
	}, tocDirHandler(cfg))

	return nil
}
