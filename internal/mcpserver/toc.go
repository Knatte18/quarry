// toc.go declares the toc tool: its fixed prose, its explicit input schema, the SDK-facing
// registration closure, and tocResult, the decision logic that closure delegates to.

package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Knatte18/quarry/internal/repopath"
	"github.com/Knatte18/quarry/quarry"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The toc tool's fixed prose: its own description and the description of each of its three input
// properties. These strings are pinned verbatim by discussion D5a and by an exact-string assertion
// in batch 3's tools/list test — they are not reworded, reflowed, or extended here.
const (
	// tocDescription is the toc tool's own description.
	tocDescription = `Table of contents for a directory or file in this repository: its package, its files, each file's header comment, and optionally each file's symbols. Reach for this first to find your way around unfamiliar code, instead of listing directories and grepping for declarations.`
	// tocTargetDescription is the target property's description.
	tocTargetDescription = `Repository-relative path to a directory or a file. Use "" or "." for the repository root.`
	// tocDepthDescription is the depth property's description.
	tocDepthDescription = `How far to recurse into subdirectories. 0, the default, lists this directory's own files and names its subdirectories without descending; N fills N levels; -1 recurses to the bottom of the tree.`
	// tocSymbolsDescription is the symbols property's description.
	tocSymbolsDescription = `Populate every file entry's symbols: functions, methods, types, consts and vars. Omit for the per-target default, which is on for a file target and off for a directory target.`
)

// tocInput is the toc tool's input, as unmarshaled from the call's arguments by the SDK's
// generated wrapper.
type tocInput struct {
	// Target is the repository-relative path argument.
	Target string `json:"target"`
	// Depth is the recursion-depth argument. It is a plain int, not a pointer, because an absent
	// depth means 0, which is a meaningful value on this surface — not a not-set marker.
	Depth int `json:"depth"`
	// Symbols is the symbols-population argument. It is a pointer because the engine's
	// TOCOptions.Symbols is a pointer and an absent property must map to nil, the per-target
	// default, rather than to false.
	Symbols *bool `json:"symbols"`
}

// tocInputSchema returns the toc tool's input schema, written explicitly rather than inferred from
// tocInput, because the SDK's inference has no way to express depth's minimum of -1 or these
// properties' exact descriptions.
func tocInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"target": {
				Type:        "string",
				Description: tocTargetDescription,
			},
			"depth": {
				Type:        "integer",
				Description: tocDepthDescription,
				Minimum:     jsonschema.Ptr(-1.0),
			},
			"symbols": {
				Type:        "boolean",
				Description: tocSymbolsDescription,
			},
		},
		Required: []string{"target"},
	}
}

// registerTOC registers the toc tool on s, bound to repo and root. It calls the SDK's generic
// mcp.AddTool instantiated as mcp.AddTool[tocInput, any]: the any output type parameter and a nil
// output value from the handler are what keep the tool free of an outputSchema and the result free
// of structuredContent, since the SDK emits both for any tool that declares an output schema. This
// is the one place that payload-shape decision can be violated silently, so it is stated here
// rather than only in the Shared Decision it implements.
func registerTOC(s *mcp.Server, repo *quarry.Repo, root string) {
	tool := &mcp.Tool{
		Name:        "toc",
		Description: tocDescription,
		InputSchema: tocInputSchema(),
	}
	mcp.AddTool(s, tool, func(_ context.Context, _ *mcp.CallToolRequest, in tocInput) (*mcp.CallToolResult, any, error) {
		return tocResult(repo, root, in), nil, nil
	})
}

// tocResult performs the CLI's pipeline minus flag parsing and exit codes, translating one tocInput
// into a *mcp.CallToolResult. It never returns a non-nil error: that channel is for protocol
// faults, and a client surfaces it as a tool malfunction rather than as an answer.
func tocResult(repo *quarry.Repo, root string, in tocInput) *mcp.CallToolResult {
	// Reject a depth below -1. mcp.AddTool's generated wrapper already validates the arguments
	// against the input schema's minimum before this handler runs, so a call arriving over the
	// protocol with depth: -2 never reaches this branch — it is rejected by the SDK first. The
	// branch is kept anyway, as the layer that owns the wording, and because the engine's walk
	// decrements depth with no floor: if the schema's minimum is ever dropped, or this function is
	// reached from in-process code that bypasses the SDK wrapper, an unvalidated negative depth is
	// an unbounded walk that returns a plausible-looking answer instead of an error.
	if in.Depth < -1 {
		return errorResult(fmt.Sprintf("--depth must be -1 (whole tree) or a non-negative integer, got %d", in.Depth))
	}

	// Relativise the target against root as both root and base: this surface has no per-call
	// working directory, so targets are repository-relative by definition.
	rel, err := repopath.RepoRelTarget(root, root, in.Target)
	if err != nil {
		if errors.Is(err, quarry.ErrTargetOutsideRepo) {
			return errorResult("target outside repository: " + in.Target)
		}
		if errors.Is(err, quarry.ErrTargetHasSeparator) {
			return errorResult("target contains the glyph separator \"#\": " + in.Target)
		}
		return errorResult("internal error: " + err.Error())
	}

	// Lstat, never Stat, so a symlink named as the target is treated as a file and not followed,
	// matching the engine's own rule.
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		if os.IsNotExist(err) {
			return errorResult("target not found: " + rel)
		}
		return errorResult("internal error: " + err.Error())
	}

	// The two sentinel branches below are race-only in the common case: the relativisation and stat
	// above have already excluded both errors, so they fire only when the target is removed between
	// the stat and the engine's own walk. Reporting that race as success would be a false positive.
	answer, err := repo.TOC(rel, quarry.TOCOptions{Depth: in.Depth, Symbols: in.Symbols})
	if err != nil {
		if errors.Is(err, quarry.ErrTargetNotFound) {
			return errorResult("target not found: " + rel)
		}
		if errors.Is(err, quarry.ErrTargetOutsideRepo) {
			return errorResult("target outside repository: " + in.Target)
		}
		return errorResult("internal error: " + err.Error())
	}

	out, err := quarry.RenderJSON(answer)
	if err != nil {
		return errorResult("internal error: " + err.Error())
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(out)}},
	}
}

// errorResult builds the failure envelope every tocResult failure path returns through, so the
// failure envelope is written once: IsError set, exactly one text content block whose text is
// quarry.RenderErrorJSON(msg) verbatim.
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: string(quarry.RenderErrorJSON(msg))}},
	}
}
