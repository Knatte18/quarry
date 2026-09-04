// tools_test.go calls tools/list over the connected client session and pins the toc tool's shape:
// exactly one tool, no output schema, the input schema's required/optional split and each
// property's type (and depth's minimum), and all four fixed prose strings as exact literals.
//
// The schema arrives on the client side as generic decoded JSON — a map[string]any the client's own
// JSON decoder built from the wire bytes, not the server's own *jsonschema.Schema value — so every
// assertion below reads that decoded shape. Comparing tocInputSchema against itself would pin
// nothing.

package mcpserver

import (
	"context"
	"testing"
)

func TestToolsList_OneToolNoOutputSchema(t *testing.T) {
	ctx := context.Background()
	client := connectedClient(t, fixtureRepoRoot(t))

	result, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("ListTools returned %d tools; want exactly 1", len(result.Tools))
	}

	tool := result.Tools[0]
	if tool.Name != "toc" {
		t.Errorf("tool.Name = %q; want %q", tool.Name, "toc")
	}
	if tool.OutputSchema != nil {
		t.Errorf("tool.OutputSchema = %v; want nil (no outputSchema)", tool.OutputSchema)
	}

	wantDescription := `Table of contents for a directory or file in this repository: its package, its files, each file's header comment, and optionally each file's symbols. Reach for this first to find your way around unfamiliar code, instead of listing directories and grepping for declarations.`
	if tool.Description != wantDescription {
		t.Errorf("tool.Description = %q; want %q", tool.Description, wantDescription)
	}

	schema, ok := tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("tool.InputSchema is a %T; want map[string]any", tool.InputSchema)
	}
	if schema["type"] != "object" {
		t.Errorf(`schema["type"] = %v; want "object"`, schema["type"])
	}

	required, ok := schema["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "target" {
		t.Errorf(`schema["required"] = %v; want ["target"]`, schema["required"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema[\"properties\"] is a %T; want map[string]any", schema["properties"])
	}

	target, ok := properties["target"].(map[string]any)
	if !ok {
		t.Fatalf("properties[\"target\"] is a %T; want map[string]any", properties["target"])
	}
	if target["type"] != "string" {
		t.Errorf(`target["type"] = %v; want "string"`, target["type"])
	}
	wantTargetDescription := `Repository-relative path to a directory or a file. Use "" or "." for the repository root.`
	if target["description"] != wantTargetDescription {
		t.Errorf("target description = %q; want %q", target["description"], wantTargetDescription)
	}

	depth, ok := properties["depth"].(map[string]any)
	if !ok {
		t.Fatalf("properties[\"depth\"] is a %T; want map[string]any", properties["depth"])
	}
	if depth["type"] != "integer" {
		t.Errorf(`depth["type"] = %v; want "integer"`, depth["type"])
	}
	if depth["minimum"] != -1.0 {
		t.Errorf(`depth["minimum"] = %v; want -1`, depth["minimum"])
	}
	wantDepthDescription := `How far to recurse into subdirectories. 0, the default, lists this directory's own files and names its subdirectories without descending; N fills N levels; -1 recurses to the bottom of the tree.`
	if depth["description"] != wantDepthDescription {
		t.Errorf("depth description = %q; want %q", depth["description"], wantDepthDescription)
	}

	symbols, ok := properties["symbols"].(map[string]any)
	if !ok {
		t.Fatalf("properties[\"symbols\"] is a %T; want map[string]any", properties["symbols"])
	}
	if symbols["type"] != "boolean" {
		t.Errorf(`symbols["type"] = %v; want "boolean"`, symbols["type"])
	}
	wantSymbolsDescription := `Populate every file entry's symbols: functions, methods, types, consts and vars. Omit for the per-target default, which is on for a file target and off for a directory target.`
	if symbols["description"] != wantSymbolsDescription {
		t.Errorf("symbols description = %q; want %q", symbols["description"], wantSymbolsDescription)
	}
}
