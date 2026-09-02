# The MCP surface — what gopls's own MCP server changes, and what quarry emits

Investigated 2026-09-02, against gopls v0.23.0 and quarry `d464334`, both run against the Loomyard
checkout at 72c23d9. Every claim here is from a live run; the captured outputs are in
`output-formats/` next to this file.

This is a companion to HANDOFF §6 (the three-way split). §6 says *which question* quarry wins;
this says *what shape* to answer it in, and what the language server now ships on its own.

## gopls ships its own MCP server, in the same binary quarry already spawns

`gopls mcp` (v0.23.0) starts an MCP server over stdio. It is not a separate product or install —
it is a subcommand of the language server quarry already depends on. It exposes eight tools:

| tool | LSP method behind it | translated? |
|---|---|---|
| `go_workspace` | none | invented |
| `go_search` | `workspace/symbol` | renamed |
| `go_file_context` | none — composed | invented |
| `go_package_api` | none — composed | invented |
| `go_symbol_references` | `textDocument/references` | re-addressed |
| `go_rename_symbol` | `textDocument/rename` | re-addressed |
| `go_diagnostics` | diagnostics | pull-shaped, takes files |
| `go_vulncheck` | none | invented |

Four of the eight have no LSP method at all. They are composed for the task rather than forwarded.

**Not one tool takes a position.** There is no `line`/`character` anywhere in the surface. Every
symbol-addressing tool takes `file` + `symbol`, and `go_symbol_references` documents its symbol
argument as "(possibly qualified)" — `Server.Run`. The gopls authors, who own the protocol server,
concluded independently that an LLM-facing surface should be addressed by qualified name, never by
cursor position.

That is direct evidence against HANDOFF §6's old item "…lenient input (`Type.Method` qualifiers
stripped)". The receiver is half the key of a Go method. §6 is corrected accordingly.

`go_rename_symbol` is the tool HANDOFF §6 used to rank first as `rename_impact`. It is shipped
upstream. Building it in quarry would be duplication.

## What gopls MCP does *not* do: line numbers

`go_package_api` and `go_file_context` return **prose** — Go source text with doc comments, in
markdown fences. Grepping their output for line references returns zero. They are documents to read,
not maps to index into.

```
gopls  go_package_api  (internal/logger)  15 754 bytes   no line numbers
gopls  go_file_context (logger.go)        20 496 bytes   no line numbers
quarry toc dir         (internal/logger)   5 463 bytes   start/sigend/end
quarry toc dir --compact                   1 716 bytes   start/sigend/end
quarry toc file        (logger.go)         7 694 bytes   start/sigend/end
quarry toc file --compact                  3 863 bytes   start/sigend/end
```

So HANDOFF §6's row 2 — *where does this symbol begin and end* — is not answered by gopls MCP at any
size. That capability remains quarry's alone, and quarry's rendering of it is 3–9× smaller.

## Why the thin-wrapper layer never measured well

The design split visible in the code maps exactly onto the measurement split:

| layer | shape | measured |
|---|---|---|
| `workspace_symbol`, `textDocument_definition`, `textDocument_references` | thin LSP forwarding | never separated from control, any run since August |
| `toc_*`, `impact` | composed and decoded | `toc_dir` is the one win; `impact` is the best-shaped envelope |

`output-formats/symbol.txt` shows why, in one response:

- `"kind": 12`, `23`, `8`, `6` — undecoded LSP `SymbolKind` integers. Every other quarry command emits
  named kinds.
- three naming conventions in one array: a full import path, a bare name, and `Type.member`.
- fuzzy-match noise: querying `dualHandler` returns `TestExecute_FailHandlerReturnsOne` from an
  unrelated package. `workspace/symbol` is fuzzy by protocol definition; the envelope does not say so.

Each of those is *faithful* to LSP. That is the problem. **LSP assumes the client is an editor**: the
`SymbolKind` table lives in the client, the candidate list is something a human clicks, and the
position comes from a cursor that is already there. An LLM has none of the three. Every awkward thing
in that output is an editor affordance with the editor removed.

This is a better explanation of the null results than "LSP tools do not help": they do not help *in
this shape*, and the shape is inherited from a client that is not present.

Note the distinction that survives: `"kind": 12` is not more faithful than `"kind": "function"` — it
is only less decoded. Decoding an enum, or labelling a query as fuzzy, documents the contract rather
than changing it. A wrapper should add no semantics; it need not forward the protocol's internal
representation.

## What quarry emits on the wire

Measured on `toc_dir internal/logger` through `cmd/quarry-mcp`:

```
wire line          11 356 bytes
content[0].text     5 501 bytes   the whole JSON document, as a STRING
structuredContent   5 491 bytes   the same JSON document, as an OBJECT
```

The payload is sent twice. The MCP go-sdk emits `structuredContent` when a tool declares an
`outputSchema` — all seven quarry tools do — and *also* serializes it into a text block for clients
that predate or ignore that field. This is the spec's own backwards-compatibility recommendation, not
an SDK defect.

**It does not cost context.** Verified against a ladder transcript: the agent's `tool_result` for a
`toc_dir` call is a single string block of 48 607 characters — one copy. Claude Code renders
`content[].text` and discards `structuredContent`. The duplication is transport-only, over a local
stdio pipe.

**JSON is not the MCP payload standard.** The envelope is JSON-RPC 2.0 and every message on the wire
is JSON, but a tool result's `content[].text` is an opaque string — markdown, plain text, anything.
gopls puts markdown there. quarry puts JSON. That is a free choice, not a constraint. What it does not
do is rescue the compact form: the model reads `content`, so what goes in that block is exactly the
trade-off the compact ladder measured and rejected (HANDOFF §3).

## Two inconsistencies in the current CLI/MCP surface

- **`definition` returns `"ok": true` while exiting 2.** On an ambiguous name it correctly refuses and
  lists candidates — good behaviour — but a consumer reading `ok` sees success and a consumer reading
  the exit code sees failure. `output-formats/definition-ambiguous.txt`.
- **Addressing differs per command.** `refs`, `definition` and `impact` accept `--in-file <path>
  <name>`; `assert-no-callers` rejects `--in-file` as an unknown flag and takes only a bare name or
  `file:line:col`. Same engine, same resolution. `assert-no-callers` is the one command explicitly
  built for a plan's `verify:` step, and it is the one that cannot take symbol-plus-file.
- Option surfaces are uneven: `noVerify` on `assert_no_callers` only, `docSentences` on `toc_file`
  only, `buildTags` on five of seven.
- The MCP tool names `textDocument_definition`, `textDocument_references` and `workspace_symbol` are
  LSP *method* names in a tool namespace. gopls named its equivalents `go_search` and
  `go_symbol_references`. If the wrapper/composed distinction is worth signalling in the names — it
  is — the wrapper half still needs tool names rather than protocol methods.

`targets` as a uniform batching key across all seven tools is the surface's best feature and is more
consistent than gopls's.

## What gopls MCP does not threaten

The overlap is narrower than it first looks. gopls MCP competes with **one of quarry's three
surfaces**, for **one of its three focus languages**.

- **The Cobra CLI has no MCP equivalent.** `assert-no-callers` is an exit-code contract — "Exit 0: no
  unexpected callers remain. Exit 1: …" — built for a plan card's `verify:` step. An MCP server has no
  exit code to give a CI job. The ladder's own annex generator also shells out to this CLI.
- **The `quarry/` Go facade has no MCP equivalent**, and it is the entire basis of the HANDOFF §7
  direction (called from Loomyard's own code, never by the LLM).
- **gopls MCP is Go-only.** Quarry's three focus languages (Go, Python, C#) under one tool namespace
  is one server, not three — and tool-surface size is a measured cost, not an aesthetic one (a b4 rep
  lost recall to tool-surface friction, HANDOFF §1).

## Recommendation

**Seven MCP tools become four.** This halves the tool surface and drops exactly the layer that has
never separated from control:

| tool | verdict |
|---|---|
| `toc_file`, `toc_dir` | keep — the one measured win, and the line spans nothing else provides |
| `impact` | keep — the only tool answering something grep structurally cannot |
| `assert_no_callers` | keep — the gate, and the only exit-code contract |
| `textDocument_definition` | drop — returns a position; grep finds the same line in 5 ms |
| `textDocument_references` | drop — bare positions; `impact` returns those plus the enclosing declaration |
| `workspace_symbol` | drop — fuzzy, undecoded kinds, cross-package noise |

The CLI keeps all seven verbs: it is a different surface with different consumers, and its JSON
envelope is already the right contract for mechanical callers.
