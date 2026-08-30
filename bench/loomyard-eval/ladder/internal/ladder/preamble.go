// preamble.go builds the full prompt string for one run of a config: the committed Agent B preamble
// for the "none" control, or a freshly generated MCP-shaped preamble naming a rung config's allowed
// tools, byte-for-byte matched against the committed prompt text this suite measures against.

package ladder

import (
	"fmt"
	"strings"
)

// PARALLEL_OPENING is copied byte-for-byte from bench/loomyard-eval/README.md's committed preambles,
// which every rung's prompt reuses verbatim.
const PARALLEL_OPENING = `USE PARALLEL TOOL CALLS. Whenever you have more than one independent thing to
read or check, issue ALL of those tool calls together in the SAME turn --
never one at a time across separate turns. This is not optional.`

// PARALLEL_BLOCK is copied byte-for-byte from bench/loomyard-eval/README.md's committed preambles,
// which every rung's prompt reuses verbatim.
const PARALLEL_BLOCK = `<use_parallel_tool_calls>
For maximum efficiency, whenever you need to perform multiple independent
operations, invoke all relevant tools simultaneously rather than
sequentially. Prioritize calling tools in parallel whenever possible. For
example, once you know several independent locations to read or check (e.g.
a list of caller locations from a single lookup), issue all of those Read or
Bash calls together in one turn rather than one at a time across separate
turns -- each turn costs a full round of model latency regardless of how
fast the underlying tool executes, so batching directly cuts wall-clock and
token cost. Err on the side of maximizing parallel tool calls rather than
running too many tools sequentially. Only batch tool calls that are
independent of each other -- two Read calls at two locations you already
know about are never dependent on each other.
</use_parallel_tool_calls>`

// B_PREAMBLE_BODY is the committed Agent B preamble's own body, between PARALLEL_OPENING and
// PARALLEL_BLOCK. It ends at the <TARGET_DIR> paragraph and excludes the committed template's own
// <TASK TEXT> placeholder line -- PreambleFor appends the task text itself, so carrying the
// placeholder through would emit both the placeholder and the text into every control run's prompt.
const B_PREAMBLE_BODY = `You are working on a code task in the codebase at <TARGET_DIR>. You have
standard tools: Read, Grep, Bash, Glob. Explore as needed to answer
thoroughly and correctly.`

// closingSentence is copied byte-for-byte from bench/loomyard-eval/README.md's committed preambles.
const closingSentence = `When you are completely done, end your final message with ONLY a fenced json
code block matching the schema below -- no other trailing prose after it.`

// toolDescriptions holds a one-line description of each canonical tool's job, for the generated
// MCP-shaped preamble body. Written without any binary path, shell verb syntax, or --prefixed flag --
// every rung's tool is a call, not a CLI verb.
var toolDescriptions = map[string]string{
	"toc_dir":                 "lists every source file directly in a directory (not recursive) with its package, header comment, and test/generated flags",
	"toc_file":                "returns the table of contents for one file: every function, method, and type with its signature, docstring, and precise line range",
	"textDocument_definition": "jumps to a symbol's definition, LSP-resolved",
	"textDocument_references": "finds every reference to a symbol, LSP-resolved -- including interface-dispatched calls a text search cannot see",
	"workspace_symbol":        "searches for a symbol by name across the whole target codebase",
	"impact":                  "resolves a symbol, finds its callers, and reports each caller's full enclosing-function line range",
	"assert_no_callers":       "fails if the symbol has callers outside its declaration",
}

// mcpPreambleBodyTemplate is the generated MCP-shaped preamble body, byte-for-byte matched against the
// Python module's _mcp_preamble_body f-string. %s placeholders are the target directory and the
// tool-lines block, in that order.
const mcpPreambleBodyTemplate = `You are working on a code task in the codebase at %s. You have
access to the following code-navigation tools, each a call taking a
call-wide input with a ` + "`targets`" + ` array:

%s

Never set buildTags on any of these calls -- the default build-tag set is
the one this run is scoped to.

Use these tools as your PRIMARY tool for anything they cover: symbol
lookups, "who calls this / where is this defined", file/directory surveys,
and caller-impact analysis. Do NOT reach for grep/ripgrep as a reflex, and
do NOT use it to "double-check" a question one of these tools has already
answered -- that defeats the point of having it and just spends tokens
re-deriving what you already know.

If you already know a symbol's declaration line -- e.g. from an earlier
table-of-contents call, which gives you every symbol's exact line range up
front -- call the matching tool with that ` + "`file:line:character`" + ` position
directly instead of the bare symbol name. A bare name triggers a
project-wide symbol search that is often genuinely ambiguous, and costs you
a second round trip to disambiguate with the position you already had.`

// mcpPreambleBody builds the body of a freshly generated MCP-shaped preamble for a quarry rung: names
// config's allowed tools by their client-side name and nothing else, then carries over the three
// exposure-independent instructions from the committed Agent A template.
func mcpPreambleBody(l *Ladder, config LadderConfig, targetDir string) string {
	lines := make([]string, 0, len(config.Allowed))
	for _, tool := range l.QuarryTools {
		if stringSliceContains(config.Allowed, tool) {
			lines = append(lines, fmt.Sprintf("- %s -- %s", MCPName(tool), toolDescriptions[tool]))
		}
	}
	return fmt.Sprintf(mcpPreambleBodyTemplate, targetDir, strings.Join(lines, "\n"))
}

// PreambleFor returns the full prompt string for one run of config against targetDir.
//
// When config.Allowed is empty, it reproduces the committed Agent B preamble exactly. Otherwise it
// generates a freshly written MCP-shaped preamble naming config's allowed tools by their
// mcp__quarry__* client-side names. Both shapes share PARALLEL_OPENING, PARALLEL_BLOCK, the closing
// schema-only-output sentence, and schemaJSON.
func PreambleFor(l *Ladder, config LadderConfig, targetDir, taskText, schemaJSON string) string {
	var body string
	if len(config.Allowed) > 0 {
		body = mcpPreambleBody(l, config, targetDir)
	} else {
		body = strings.ReplaceAll(B_PREAMBLE_BODY, "<TARGET_DIR>", targetDir)
	}

	return strings.Join([]string{PARALLEL_OPENING, body, taskText, PARALLEL_BLOCK, closingSentence, schemaJSON}, "\n\n")
}
