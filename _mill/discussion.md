# Discussion: Add an MCP wrapper for quarry

```yaml
task: Add an MCP wrapper for quarry
slug: quarry-mcp-wrapper
status: discussing
parent: main
```

## Problem

quarry was designed with three exposure layers.
Two exist and are stable: `internal/quarryengine` (the engine) and `internal/cli` (a Cobra wrapper serving humans and LoomYard's own mechanical calls), with `quarry/facade.go` as the stable public Go binding surface between them.
The third — an MCP server letting Claude call quarry's verbs as native, schema-typed tools — was never built.

Why now: an extended benchmark session in `bench/loomyard-eval` measured whether quarry gives real speedup to Claude agents doing exploration, review, and impact-analysis work.
It repeatedly surfaced CLI ergonomics bugs that cost the agent turns, not milliseconds: `toc dir`/`toc file` rejecting `--target-dir`, `refs`/`definition`/`impact` resolving relative positions against the wrong base directory, and `quarry definition` having no separate file/line/char flags (the agent guessed a nonexistent `--file`/`--line`/`--char` syntax before falling back to the documented `file:line:col` positional form).
Each cost 1–3 wasted tool calls.
Turn count — not tool latency — was established as the dominant driver of benchmark wall-clock across the whole session; quarry's own calls run in single-digit milliseconds to ~0.7s even for LSP-backed verbs.
Those specific bugs were fixed in commit `2b6ccc9`, but the underlying failure mode is structural: a CLI requires the model to reconstruct flag syntax from memory or `--help` output, while a tool schema makes parameters typed and validated so they cannot be guessed wrong.

A secondary input is an external claim (https://karanbansal.in/blog/claude-code-lsp/) that Claude Code's native LSP tool integration vastly outperforms grep-based fallback search.
On inspection that is not a claim about the LSP wire protocol being inherently better-understood — it is about having a first-class, schema-typed, native tool rather than a CLI.
That maps directly onto the friction observed and fixed this session.

## Scope

**In:**

- New `internal/mcpserver/` package implementing an MCP server over stdio, exposing seven tools.
- New `cmd/quarry-mcp/` binary — a thin `main.go` that constructs and runs the server.
- A minimal set of newly-exported resolution helpers in `internal/cli` (`ResolveStateDir`, `ResolveConfigPath`, `ResolveBuildTags`, `AbsOrJoin`) so the MCP layer reuses the CLI's exact state-directory keying rather than reimplementing it.
- A translation layer: `file://` URI acceptance on input, 0-based↔1-based line/character conversion for the LSP-mirrored tools only.
- `go.mod` dependency on `github.com/modelcontextprotocol/go-sdk`.
- Three tiers of tests (unit / in-memory transport / real-binary stdio).
- A committed `.mcp.json` at the repo root so a Claude Code session in quarry can connect to the server.

**Out:**

- **No rename or restructure of `internal/cli`.**
  It serves different consumers — human operators, and LoomYard calling quarry mechanically with no LLM in the loop — with different ergonomic needs.
  Forcing CLI/MCP naming consistency would be a net regression for the CLI's own users.
- **No engine changes.**
  Specifically: the `character`-unit asymmetry documented under Technical context is left exactly as it is.
  Fixing it is a separate task, not something folded into a pure-exposure change.
- **No new engine logic.**
  Every handler is a binding onto `quarry/facade.go` — the same functions `internal/cli` already calls.
- **No validation of whether the LSP-mirrored shape actually helps.**
  That is the separate follow-up task `quarry-mcp-vs-cli-bench` (referred to below as #006).
  This task builds the thing to be measured; it does not measure it.
- **No transports other than stdio.**
  No HTTP, no SSE.
- **No MCP resources or prompts.**
  Tools only.
- **No changes to the CLI's own output shapes, flags, or exit codes.**

## Decisions

### mcp-sdk

- Decision: use `github.com/modelcontextprotocol/go-sdk`, currently at `v1.7.0`.
- Rationale: first-party, v1-stable, ships the schema handling and stdio transport this task would otherwise hand-roll.
  `go.mod` has no MCP dependency today, so this is purely additive.
- Rejected: `github.com/mark3labs/mcp-go` (still pre-1.0 at `v1.0.0-beta.1`); hand-rolled JSON-RPC over stdio (no dependency, but re-implements schema negotiation and forces us to track protocol changes ourselves).

### binary-shape

- Decision: a separate `cmd/quarry-mcp` binary, not a subcommand of `cmd/quarry`.
- Rationale: stdio MCP cannot tolerate anything else writing to stdout.
  The existing CLI writes JSON to stdout by design (`internal/output`), so sharing a process with it puts the JSON-RPC stream one stray line away from corruption.
  A dedicated binary removes the hazard structurally rather than by discipline.
- Rejected: `quarry mcp serve` subcommand (one artifact, but shares stdout with a layer built to print to it); both forms (redundant given the above).

### package-placement

- Decision: `internal/mcpserver/` with a thin `cmd/quarry-mcp/main.go`.
- Rationale: the tool shapes are explicitly unvalidated until #006 runs.
  Placing them in a public package would freeze them into the module's public API before they have been measured.
  `internal/` also lets the MCP layer import `internal/cli`'s helpers (see shared-resolution-helpers).
- Rejected: a public `mcp/` package next to `quarry/` (parallels the facade's placement, but freezes unvalidated shapes and cannot reach `internal/cli`).

### shared-resolution-helpers

- Decision: export a minimal set from `internal/cli` — `ResolveStateDir`, `ResolveConfigPath`, `ResolveBuildTags`, `AbsOrJoin` — and have `internal/mcpserver` call them.
  Do not extract a new package; do not duplicate.
- Rationale: MCP and CLI must derive **bit-for-bit identical** state directories.
  The supervised daemon is partitioned by `StateDir` alone (see the contract on `query.Options.StateDir`), and the CLI appends a `tags-<hex>` segment plus a `workspaceKey` basename-and-hash segment.
  If the two layers derive different keys, a second gopls daemon is spawned silently — the MCP server would never reuse a warm daemon the CLI started, and vice versa.
  Duplication guarantees eventual drift on something that must stay identical.
- Rejected: a new `internal/resolve` package both layers import (cleaner layering, but a structural refactor of stable CLI code inside an additive-exposure task); duplicating the logic in `internal/mcpserver` (CLI untouched, two implementations that will diverge).

### character-encoding

- Decision: naive `±1` only.
  On the LSP-mirrored tools, `character` is 0-based on the wire; add 1 to hand to `quarry.Position`, subtract 1 from results.
  No file reads, no UTF-16 conversion in the MCP layer.
- Rationale: this round-trips quarry's own output exactly — a `character` returned by `textDocument_references` can be fed straight back into `textDocument_definition`.
  It matches the CLI's current behaviour, and it is correct for ASCII, which covers Go, Loomyard, and quarry in practice.
  The engine's own column-unit inconsistency (see Technical context) is a real bug, but fixing it is a separate task and must not be smuggled into an exposure-only change.
- Rejected: full LSP-faithful conversion in the MCP layer (correct on non-ASCII, but disagrees with what the CLI prints for the identical query, and requires file reads the handler otherwise does not do); fixing the engine's outbound `+1` first (correct and self-consistent, but changes existing CLI output and is out of scope).

### file-reference-form

- Decision: accept both `file://` URIs and plain paths on input; always emit plain absolute paths on output.
- Rationale: accepting the URI form is the point of the hedge — it is what LSP-shaped input looks like.
  Emitting URIs would force Claude to unwrap a URI before the result is usable in Read/Edit, adding exactly the friction this task exists to remove.
- Rejected: strict LSP `uri` in and out (maximum fidelity, but pushes unwrapping work onto every consumer of a result); plain paths only (abandons the hedge on input, which is where it matters).

### tool-set

- Decision: seven tools, full parity with the CLI verb surface — `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `toc_file`, `toc_dir`, `impact`, `assert_no_callers`.
- Rationale: there is no good reason to assume `assert_no_callers` is used less often through MCP than through the CLI, and it is named explicitly in the task brief.
  Collapsing `toc_file`/`toc_dir` into one tool with a mode parameter would reintroduce exactly the flag-shaped friction this task removes.
- Rejected: six tools dropping `assert_no_callers`; a consolidated `toc` with a mode parameter.

### tool-naming

- Decision: for the three verbs with a direct LSP equivalent, use the literal LSP method names with `/` replaced by `_`: `textDocument_definition`, `textDocument_references`, `workspace_symbol`.
  The four quarry-native tools keep quarry's own vocabulary: `toc_file`, `toc_dir`, `impact`, `assert_no_callers`.
- Rationale: the hypothesis under test in #006 is that Claude has stronger training priors for the LSP shape.
  If so, the name is the strongest single component of that signal.
  A `quarry_` prefix or a bare verb name dilutes precisely the thing being measured.
- Rejected: bare verbs `definition`/`references`/`symbol` (cleaner, but carries the prior only in the parameter shape); `quarry_`-prefixed names (namespaced against other servers' tools, but the prefix is the one token the LSP prior does not have).

### target-dir-resolution

- Decision: `--target-dir` is a server-launch flag supplying the default; every tool additionally accepts an optional per-call `targetDir` override.
- Rationale: mirrors the CLI's own `--target-dir` precedence pattern.
  `.mcp.json` supplies the value once, so Claude fills nothing on the common call, and can still point a query at another project when it needs to.
- Rejected: a required per-call parameter on every tool (explicit and stateless, but adds mandatory boilerplate to every call); server process cwd only (zero params, silently wrong if the client launches the server from elsewhere).

### array-batching

- Decision: **all seven** tools take an array of targets, not a single target.
  This is new functionality relative to today's CLI for the LSP-mirrored three, and matches the existing CLI batch pattern for the rest.
- Rationale: parallel `tool_use` blocks are empirically unavailable to `Agent`-dispatched subagents.
  This was verified earlier in the originating session: 0 of 56 turns batched, across two escalating prompt strategies — most likely an API-level `disable_parallel_tool_use` constraint applied specifically to dispatched subagents.
  Since dispatched subagents are a primary consumer, array input is the **only** mechanism that actually reduces turn count for them: it is one `tool_use` call carrying several targets in its input, not several `tool_use` calls in one turn.
  Turn count is the established dominant cost driver, so this is the single highest-leverage shape decision in the task.
- Rejected: one target per call (would have matched the LSP request shape, but the justification — "Claude can issue several tool calls in one message" — is false for the primary consumer); arrays only on the quarry-native tools (splits the mechanism exactly where it is needed most).
- Note: the "no LSP analogue for a batch shape" cost is real but irrelevant.
  `textDocument/definition` is single-request in every LSP variant, so no option here has a batch shape to mirror — there is no hedge to lose at the batch level.

### entry-shape-lsp-mirrored

- Decision: for the three LSP-mirrored tools, each array element is a union of three forms:
  - explicit position: `{"textDocument": {"uri": "..."}, "position": {"line": 0, "character": 0}}` — verbatim LSP `TextDocumentPositionParams`
  - project-wide symbol name: `{"symbol": "Name"}`
  - file-scoped symbol name: `{"textDocument": {"uri": "..."}, "symbol": "Name"}`
- Rationale: tool-naming already went all-in on literal LSP naming because the name carries the signal.
  The same logic applies to the nesting structure: `{textDocument: {uri}, position: {line, character}}` is an equally recognisable, equally repeated pattern in training data (LSP spec, gopls source, VS Code clients), not merely a set of field names.
  Since #006 will measure this empirically, it must measure a clean maximal version of the LSP shape, not a compromise.
- Rejected: a flat union `{file, line, character}` / `{symbol}` / `{file, symbol}` (keeps LSP names but discards the nesting — a half-hedge that tests neither the pure quarry form nor the pure LSP form, and makes #006's result harder to interpret); string entries like `"pkg/x.go:12:5"` (literally the string-parsing friction this task exists to remove, and reintroduces the position-vs-symbol ambiguity `parsePosition` currently resolves by guessing).

### in-file-is-per-entry

- Decision: file-scoping (the CLI's call-wide `--in-file` flag) becomes a **per-entry** property, expressed as the third entry form above.
  It is not a call-wide parameter.
- Rationale: the point of array batching is covering **heterogeneous** targets in one call.
  A call-wide `--in-file` would force every entry into the same file, leaving the agent two bad options: drop file-scoping entirely (losing precision) or split into one call per file (destroying the turn saving array batching exists to provide) — and it fails worst in exactly the ambiguous same-name case where file-scoping is most needed.
- Rejected: keeping `--in-file` call-wide for CLI parity.

### entry-shape-quarry-native

- Decision: the four quarry-native tools use a **flat** entry shape with quarry's own conventions — plain paths, 1-based line and column, no URI form and no `±1` conversion.
  `impact` and `assert_no_callers` entries are a flat union `{file, line, character}` / `{symbol}` / `{file, symbol}`; `toc_file` and `toc_dir` take arrays of plain paths.
- Rationale: this follows directly from the brief's explicit mandate — mirror LSP only where there is an LSP equivalent, and keep quarry's natural shape where there is none.
  Keeping the two shapes cleanly separated is also what makes #006 interpretable: any measured difference is attributable to shape, not to a muddled hybrid.
  It has the side benefit that quarry-native tool output stays byte-comparable with the CLI's.
- Rejected: applying the LSP nested shape uniformly for internal consistency (would contradict the brief and destroy #006's ability to attribute a result).
- Consequence to handle explicitly: the server exposes two positional conventions at once — 0-based on the LSP-mirrored three, 1-based on the quarry-native ones.
  Every tool's description string must state its convention in its first line so the difference is never inferred.

### result-shape

- Decision: each tool declares a JSON output schema and returns `structuredContent` carrying a batch envelope (`{"results": [{"target": ..., "status": ..., ...}]}`), plus a text content block rendering the same JSON.
- Rationale: schema-typed output rests on the same argument as schema-typed input, and it is cheap to do both.
  The text block costs nothing and keeps clients that ignore `structuredContent` working.
- Rejected: text content only carrying byte-identical CLI JSON (zero new surface, but discards half the "native typed tool" advantage this task is betting on); `structuredContent` only (cleanest, invisible on clients that do not render it).

### error-mapping

- Decision: per-entry `status` (`found` / `not_found` / `ambiguous` / `error`) inside an otherwise-successful result for anything target-specific.
  Tool-level `isError` is reserved for whole-call failures: an unusable `targetDir`, a malformed `servers.yaml`, a daemon spawn failure.
- Rationale: mirrors `classifyLookupError`/`classifySymbolError` exactly, and preserves partial results.
  This matters far more under array batching than it would have for single-target calls: with arrays, `isError` on one bad target would discard every good answer in the same call.
- Rejected: `isError: true` whenever any entry failed (simple, throws away good results); never using `isError` (a config error would look like a normal result until the body is read).
- Note: MCP has no exit code, so the CLI's `statusRank` ordering has no direct home.
  The rank exists only to pick a process exit code; per-entry `status` carries the same information without it.

### param-split

- Decision: per-call parameters are the answer-shaping ones — `lang`, `within`, `buildTags`, `docSentences`, `except`, `noVerify`, plus the optional `targetDir` override (and file-scoping, which is per-entry per in-file-is-per-entry).
  Server-launch-only flags are the environment-shaping ones — `--config`, `--state-dir`, `--timeout`, `--target-dir` (default).
- Rationale: the model should only ever see parameters that change the **answer**, never infrastructure.
- Rejected: everything per-call (full CLI parity, but a wide schema on every tool and more for the model to get wrong); target-only per-call (smallest schema, but `within` and file-scoping are genuinely per-question and would become unreachable).

### test-strategy

- Decision: three tiers, all of them.
  1. Unit tests per handler against the facade, using the existing `testdata/` fixtures.
  2. An in-memory-transport MCP client/server test covering schema validation, array batching, per-entry status, and error mapping.
  3. A `//go:build lsp` subprocess test that execs the real built `quarry-mcp` binary and speaks actual MCP framing over stdio, `t.Skip`ping when gopls is absent.
- Rationale: tier 3 is not optional here.
  It is precisely the test that would catch a stray Cobra or log line corrupting the JSON-RPC stream — the exact risk that motivated the separate-binary decision in binary-shape.
  An in-memory transport can never see that class of failure.
  Tier 3 also matches the convention already established in `internal/cli/refs_targetdir_lsp_test.go`: `//go:build lsp`, tests the real thing, skips without gopls.
- Rejected: tiers 1+2 only (fast and hermetic, never proves the shipped binary's stdio framing works); tiers 1+3 only (every protocol-level assertion would pay a gopls dependency).

### mcp-json-wiring

- Decision: commit `.mcp.json` at quarry's repo root, running `go run ./cmd/quarry-mcp --target-dir .`.
- Rationale: works from a fresh clone with no install step, and dogfoods the server in quarry's own future sessions — including work of this kind — which is the best continuous validation available.
  The compile is cached after the first launch.
- Rejected: invoking an installed `quarry-mcp` from `$PATH` (matches real-world deployment, but is broken until someone runs `go install`); documenting the snippet in `docs/` without committing `.mcp.json` (leaves the brief's own requirement — "a Claude Code session can actually connect to it for testing" — unfulfilled).

## Technical context

**Module.** `github.com/Knatte18/quarry`, Go 1.26.
Existing direct dependencies include `spf13/cobra`, `gofrs/flock`, `gopkg.in/yaml.v3`, and the tree-sitter bindings.
No MCP dependency yet.

**The facade is the binding surface.** `quarry/facade.go` is a behaviour-free re-export of `internal/quarryengine`: every declaration is a type alias, a re-exported sentinel var bound to the identical error value, or a one-line delegating function.
`quarry/facade_test.go` enforces that property mechanically for every re-exported identifier.
**Do not add logic to `quarry/facade.go`** — the MCP layer's translation work belongs in `internal/mcpserver`.

The facade functions the handlers need:

| Tool | Facade call | Returns |
| --- | --- | --- |
| `textDocument_definition` | `quarry.Definition(ctx, Options)` | `[]quarry.Reference` |
| `textDocument_references` | `quarry.References(ctx, Options)` | `[]quarry.Reference` |
| `workspace_symbol` | `quarry.Symbol(ctx, Options)` | `[]quarry.SymbolMatch` |
| `impact` | `quarry.Impact(ctx, Options)` | `quarry.ImpactResult` |
| `assert_no_callers` | `quarry.Callers(ctx, Options)` | `([]Reference, []Reference, error)` — refs and declaration refs |
| `toc_file` | `quarry.TOCFile(path, lang, TOCOptions)` | `quarry.TOCFileResult` |
| `toc_dir` | `quarry.TOCDir(dir, lang)` | `quarry.TOCDirResult` |

Supporting facade calls: `quarry.LoadRegistry(path)`, `quarry.DetectLanguage`, `quarry.NormalizeBuildTags`, `quarry.TOCLanguages()` (the designed language set, what `--lang` is validated against), `quarry.TOCImplemented()` (the subset with a registered Strategy, what an unsupported-language error message is worded from).

**Core types.**

- `quarry.Position{File string; Line int; Character int}` — 1-based line, 1-based **byte** column.
- `quarry.Query{InFile *InFileQuery; Symbol string; Pos *Position}` — exactly one of the three selects the resolution mode.
  This union is what the entry-shape union maps onto.
- `quarry.InFileQuery{File, Name string}` — file-scoped name resolution.
- `quarry.Reference{File, Line, Character int}`, `quarry.SymbolMatch{Name, Kind, File, Line, Character}`.
- `quarry.Options{Registry, TargetDir, StateDir, Lang, Query, Timeout, BuildTags, SkipVerification}`.

**`Options.StateDir` and `Options.BuildTags` are the caller's obligation.**
The engine deliberately does not re-key the state directory on the caller's behalf.
Its own doc comment spells out the failure mode: a caller that varies `BuildTags` while holding `StateDir` fixed has two tag sets served by one daemon and gets the other tag set's answers.
`Options.SkipVerification` is negative-polarity on purpose — the zero value means "verify", and `assert_no_callers`'s `noVerify` parameter is what opts into the noisier unverified behaviour.

**The CLI owns the pre-flight resolution the engine refuses.**
This is the single most important structural fact for this task: an MCP handler is *not* simply "call the facade".
`internal/cli/paths.go` provides:

- `resolveConfigPath(flag)` — `--config` → `$QUARRY_CONFIG` → `<userConfigDir>/quarry/servers.yaml`.
- `resolveBuildTags(flag)` — `--build-tags` → `$QUARRY_BUILD_TAGS` → empty, normalized through `quarry.NormalizeBuildTags`.
- `resolveStateDir(flag, targetDir, buildTags)` — `--state-dir` → `$QUARRY_STATE_DIR` → `<userCacheDir>/quarry/<workspaceKey(targetDir)>`, then appends `buildTagsSegment(buildTags)` when the tag set is non-empty, at **every** tier.
- `workspaceKey(dir)` = `filepath.Base(dir) + "-" + first 12 hex of sha256(filepath.Clean(abs(dir)))`.
- `buildTagsSegment(tags)` = `"tags-" + first 12 hex of sha256(strings.Join(NormalizeBuildTags(tags...), ","))`.

`internal/cli/cli.go:resolveContext(dir, configFlag, stateDirFlag, buildTags)` composes these three into `(Registry, absTargetDir, stateDir, error)`, and `buildOptions(...)` assembles `quarry.Options`.
`internal/mcpserver` should follow the identical sequence.
`userConfigDir`/`userCacheDir` are package-level seams in `paths.go` that tests redirect; the MCP tests can use the same technique only from inside `internal/cli`, so MCP-side tests should pin `--state-dir` explicitly instead.

**Path resolution is the CLI's job, not the engine's.**
`quarry.Query.Pos.File` and `quarry.InFileQuery.File` **must be absolute** — `References` turns them into `file://` URIs directly with no further resolution.
`internal/cli/cli.go:absOrJoin(base, path)` is the one rule: absolute paths are cleaned, relative paths are joined onto `base`.
`base` is the resolved target directory, never the process cwd.
The MCP layer's URI-acceptance step slots in immediately before this: strip a `file://` prefix if present, then `AbsOrJoin` against the effective `targetDir`.

**The cwd seam.** `internal/cli/cwdcontext.go` provides `WithCwd(ctx, absDir)` / `CwdFrom(ctx)`.
`WithCwd` panics on a non-absolute directory.
The MCP server has no meaningful per-call cwd, which is why `targetDir` comes from a launch flag with a per-call override rather than from the process.

**The `character`-unit asymmetry (gotcha, deliberately not fixed here).**
Inbound, `internal/quarryengine/lsp/wire.go:54-63` reads the file and converts quarry's 1-based byte column into LSP's 0-based UTF-16 code-unit character.
Outbound, `internal/quarryengine/query/refs.go:428-429` and `symbol.go:62-63` do a naive `+1` on the LSP character.
So a returned `Character` is UTF-16-derived, not a byte column: the two directions disagree on non-ASCII lines.
Per character-encoding this is left alone and the MCP layer does naive `±1`, which reproduces the CLI's existing behaviour exactly.
A comment in the translation layer should name this so a future reader does not "fix" it locally.

**Existing output envelope, for reference.** `internal/output/output.go` provides `Ok`/`Err`/`ErrFields` writing `{"ok": true|false, ...}`.
Successful lookups carry `"resolution": "complete"` — a machine-readable trust marker telling a caller the language server resolved the query exhaustively, so a redundant grep pass can be skipped.
**Keep this marker in the MCP result envelope**; it is cheap and it is exactly the signal an agent needs.
Batch entries are `{"symbol": <arg>, "status": <batchStatus>, ...fields}` where `batchStatus` is one of `found`/`not_found`/`ambiguous`/`error` (`internal/cli/cli.go:runBatch`, and the path-keyed `runPathBatch` in `toc.go`).
Ambiguity surfaces as `{"candidates": [...]}` from `*quarry.ErrAmbiguousSymbol`; not-found is `quarry.ErrSymbolNotFoundSentinel` via `errors.Is`.
`internal/cli/impact.go:structToFields` is the struct→`map[string]any` helper the impact and toc verbs use; its error messages carry a literal `"toc: "` prefix that `rewordImpactMarshalFailure` rewrites — the MCP layer will want its own equivalent rewording rather than leaking `"toc: "` from an `impact` call.

**`within` filtering is CLI-side, not engine-side.**
`filterWithin`/`isWithinDir` in `cli.go` and `filterImpactWithin` in `impact.go` post-filter results.
The MCP layer needs equivalent filtering; reusing these means exporting them too, or reimplementing the (short) logic in `internal/mcpserver`.
Prefer exporting if the logic is non-trivial, to keep one definition of "within".

**`toc` specifics.** `toc file` resolves an optional per-directory config via `resolveTOCConfigPath(targetDir)` — `$QUARRY_TOC_CONFIG`, else `<targetDir>/.quarry.yaml`, looked up in that directory only with no upward walk.
`--doc-sentences` accepts a number or `"all"` (`quarry.TOCAllSentences`); resolution order is flag → config → default 1, via `resolveDocSentences`.
`toc dir` emits headers only and never consults the config.
`--lang` is validated against `quarry.TOCLanguages()`, but the unsupported-language error message is worded from `quarry.TOCImplemented()`.
`toc dir` composes each listed file's `path` relative to the argument **as the caller wrote it**, so the value round-trips into a follow-up `toc file` call; preserve that.

**Test conventions.** Tests requiring a real gopls are gated by `//go:build lsp` **and** `t.Skip` on `exec.LookPath("gopls")` with the install hint (`internal/cli/refs_targetdir_lsp_test.go`, `impact_lsp_test.go`, `assertnocallers_lsp_test.go`).
Fixtures live under `testdata/` (`impactfixture/`, `buildtagfixture/`, `clockfixture/`).

**Layering rule to respect.** `internal/cli` imports nothing under `internal/quarryengine` directly — every engine identifier reaches it through `quarry/facade.go`.
`internal/quarryengine/layering_test.go` and `seam_enforcement_test.go` police the engine's own package DAG.
`internal/mcpserver` must follow the same rule: facade only, no direct engine imports.

## Constraints

There is no `CONSTRAINTS.md` at the hub root.
Constraints established during this discussion:

- **stdout purity.** Nothing in the `quarry-mcp` process may write to stdout except the MCP transport.
  Diagnostics go to stderr.
  This is load-bearing, not stylistic — see binary-shape and test-strategy tier 3.
- **State-directory keying must be bit-for-bit identical to the CLI's.**
  Any divergence silently spawns a second gopls daemon and forfeits warm-daemon reuse.
- **`quarry/facade.go` stays behaviour-free.**
  `facade_test.go` enforces it mechanically.
- **`internal/mcpserver` imports the facade only**, never `internal/quarryengine/...` directly.
- **No changes to CLI behaviour**, including output shapes, flag names, and exit codes.
  The only permitted edit to `internal/cli` is exporting existing helpers (renaming the identifier, not changing its logic) — plus, if chosen, exporting the `within`-filter helpers.
- **Two positional conventions coexist by design** (0-based LSP-mirrored, 1-based quarry-native).
  Each tool description must state its own convention explicitly.

## Testing

**Tier 1 — handler unit tests (`internal/mcpserver`, no build tag).**
TDD candidates, in this order:

- The **translation layer** is the strongest TDD candidate in the task and should be written test-first: `file://` stripping and passthrough for plain paths, `AbsOrJoin` behaviour against a `targetDir`, the `±1` conversion in both directions, and round-trip stability (a result fed back as an input resolves to the same position).
  Include an explicitly-asserted non-ASCII case documenting the *current* (naive) behaviour, so the deliberate limitation is pinned rather than accidental.
- **Entry-union parsing**: each of the three LSP-mirrored entry forms maps onto the right `quarry.Query` variant; a malformed entry (both `position` and no `textDocument`, neither `symbol` nor `position`, etc.) is rejected with a clear per-entry error rather than a silent guess.
- **Flat entry-union parsing** for the quarry-native tools, including that no `±1` and no URI handling is applied there.
- **Options assembly**: `targetDir` precedence (per-call override beats launch default), state-dir derivation matching `internal/cli`'s for the same inputs — assert the derived path is equal to what the exported CLI helper returns for identical arguments, so drift fails the build.
- **`within` filtering** and `except`/`noVerify` handling for `assert_no_callers`.
- Per-tool handler tests against existing `testdata/` fixtures for the non-LSP-dependent paths (`toc_file`, `toc_dir`).

**Tier 2 — in-memory MCP transport tests.**
Client and server wired over the SDK's in-memory transport, with the facade calls stubbed or pointed at tree-sitter-only fixtures so no gopls is required:

- Tool listing: all seven tools present, names exactly as decided, each with an input and output schema.
- Schema validation rejects a malformed call before any handler runs.
- **Array batching**: a multi-entry call returns one entry per input, in order, with the right per-entry `status`.
- **Error mapping**: a mixed call — one resolvable target, one not-found, one ambiguous — returns `isError: false` with three distinct per-entry statuses and the good result intact.
  This is the regression that matters most under array batching.
- Whole-call failure (unusable `targetDir`, malformed `servers.yaml`) returns `isError: true`.
- `structuredContent` and the text fallback carry the same payload.
- `resolution: "complete"` is present on successful lookup entries.

**Tier 3 — real-binary stdio integration (`//go:build lsp`, `t.Skip` without gopls).**
Builds `cmd/quarry-mcp`, execs it, and speaks real MCP framing over its stdin/stdout:

- Initialize handshake, `tools/list`, and at least one real `tools/call` against a `testdata/` fixture resolving through gopls.
- **Explicitly assert stdout carries only well-formed JSON-RPC frames** — no stray log or Cobra output.
  This assertion is the reason tier 3 exists.
- One multi-entry call, to prove array batching survives real serialization.

`.mcp.json` correctness is verified by dogfooding rather than by a test.

## Q&A log

- **Q:** Which Go MCP library? **A:** Official `modelcontextprotocol/go-sdk` — first-party and stable; no reason to hand-roll JSON-RPC.
- **Q:** Separate binary or subcommand? **A:** Separate `cmd/quarry-mcp` — stdio MCP cannot tolerate anything else writing to stdout, and a separate binary avoids the hazard entirely.
- **Q:** Where does the MCP code live? **A:** `internal/mcpserver/` + thin `main.go` — tool shapes are explicitly unvalidated until #006, so they must not be frozen into a public API yet.
- **Q:** How does the MCP layer reach the CLI's unexported resolution helpers? **A:** Export a minimal set from `internal/cli`. Critical that MCP and CLI use the exact same state-dir key, or a second daemon is spawned silently; duplication guarantees drift on something that must be bit-for-bit identical.
- **Q:** `character` encoding fidelity? **A:** Naive `±1`. Matches the CLI as it is today and is correct for ASCII (which covers Go/Loomyard/quarry in practice). Fixing the engine's column counting is a separate task, not something to sneak into an exposure-only change.
- **Q:** URI or plain path? **A:** Accept both in, always emit plain paths out. Hedge on input (that is the whole point), but do not force Claude to unwrap a URI before it can use the answer in Read/Edit — that would add the friction we are removing.
- **Q:** How many tools? **A:** Seven, full parity. No good reason to assume `assert_no_callers` is used less via MCP than via CLI; and a mode parameter on `toc` is exactly the friction type this task removes.
- **Q:** Tool naming? **A:** Raw LSP method names (`textDocument_definition` etc.). If the hypothesis is "trained on the LSP shape", the name is the strongest part of the signal; a `quarry_` prefix or bare verb dilutes precisely what is being tested.
- **Q:** `targetDir` source? **A:** `--target-dir` at launch plus optional per-call override — matches the CLI's own pattern, zero boilerplate in the common case.
- **Q:** Batch support? **A:** Arrays on **all seven** tools. The single-target recommendation was rejected on empirical grounds: parallel tool calls never fire for `Agent`-dispatched subagents (0/56 turns across two escalating prompt strategies, likely an API-level `disable_parallel_tool_use` on dispatched subagents). Array input is the one mechanism that bypasses it — one `tool_use` call with several targets, not several `tool_use` calls in one turn. The "no LSP counterpart for a batch shape" cost is real but irrelevant, since `textDocument/definition` is single-request in every variant, so no option has anything to mirror at the batch level.
- **Q:** Result shape? **A:** Schema-typed `structuredContent` plus text fallback — same argument as for input; cheap to do both, fallback costs nothing.
- **Q:** Error mapping? **A:** Per-entry status in the payload, `isError` only for whole-call failures. Directly coupled to the array decision: with all seven tools taking arrays, `isError` on one bad target would throw away five good answers with it — more important after the array decision than it was before.
- **Q:** Which parameters are per-call vs launch-only? **A:** Per-call = answer-shaping; launch-only = environment-shaping. The model should only see parameters that change the answer, not infrastructure.
- **Q:** Test strategy? **A:** All three tiers. Tier 3 (real binary, real stdio) is not optional — it is exactly the test that catches the "stray Cobra log line corrupts the JSON-RPC stream" risk that motivated the separate binary, and an in-memory transport can never see it. Matches the `//go:build lsp` + skip-without-gopls pattern already in `refs_targetdir_lsp_test.go`.
- **Q:** `.mcp.json` wiring? **A:** Commit it, running `go run ./cmd/quarry-mcp --target-dir .` — works from a fresh clone and dogfoods the server in quarry's own future sessions, the best continuous validation available. Documenting only would leave the proposal's own requirement ("a Claude Code session must actually be able to connect to it") unfulfilled.
- **Q:** Entry shape for the LSP-mirrored tools, and is file-scoping per-entry or call-wide? **A:** LSP-native nested union, and per-entry. Nesting is as recognisable a training-data pattern as the method names themselves; flattening is a half-hedge that tests neither pure form cleanly and makes #006 harder to interpret. Per-entry file-scoping because the point of batching is heterogeneous targets — a call-wide flag forces dropping precision or splitting into per-file calls, destroying the turn saving, and fails worst in the ambiguous same-name case where scoping is most needed.
- **Q:** Entry shape for the quarry-native tools? **A:** Derived, not asked — flat union with plain paths and 1-based positions, per the brief's explicit mandate to mirror LSP only where an equivalent exists. Keeps #006 interpretable and keeps quarry-native output comparable with the CLI's.
