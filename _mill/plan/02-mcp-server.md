# Batch: mcp-server

```yaml
task: "MCP, thin (T6)"
batch: "mcp-server"
number: 2
cards: 6
verify: go build ./... && go test ./internal/mcpserver/...
depends-on: [1]
```

## Batch Scope

This batch delivers the server itself: the module dependency on the official MCP Go SDK, the
`internal/mcpserver` package (startup root resolution, server construction, the single `toc` tool and
its handler), the thin `cmd/quarry-mcp` binary, and the layering test that mechanically enforces the
facade-only rule for both new packages. It is one batch because none of these compile without the
others, and because the whole surface is ~350 lines over a facade the implementer does not have to
read the engine to understand.

**External interface batch 3 consumes:** `mcpserver.NewServer(repo *quarry.Repo, root string) *mcp.Server`
and `mcpserver.ResolveRoot(flagRoot, cwd string) (string, error)`. Batch 3's tests connect a client to
the value `NewServer` returns over the SDK's in-memory transport; they never construct the tool or the
handler directly.

Batch-local decisions:

- The handler's decision logic lives in an unexported function returning a `*mcp.CallToolResult`,
  separate from the SDK-facing closure, so batch 3 can exercise it through the protocol without the
  logic being tangled with SDK plumbing.
- Every `_test.go` file this batch and batch 3 add to `internal/mcpserver` is an **in-package** test
  (`package mcpserver`, not `package mcpserver_test`), matching how `internal/cli` and
  `internal/engine` test themselves. Two consequences the cards below rely on: the layering test can
  walk this package's own files including its tests, and batch 3 card 15 can call the unexported
  handler function directly for the one assertion that cannot be made through the protocol.
- `NewServer` takes the already-opened `*quarry.Repo` and the already-resolved absolute root rather
  than resolving either itself. Both are resolved exactly once, in `main`, before the transport
  starts (discussion D17): a failure to open is a startup failure, never a per-call error.

## Cards

### Card 5: add the MCP Go SDK dependency

- **Context:**
  - `_mill/discussion.md`
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `github.com/modelcontextprotocol/go-sdk` at exactly `v1.7.0` and
  `github.com/google/jsonschema-go` at exactly `v0.4.3` as direct requirements, with their transitive
  requirements recorded as indirect.
  The SDK version is pinned by discussion D1: V1 used this SDK at this version and plan §9a's probe
  was verified against a server built on it, so protocol compatibility is not part of this task's
  risk surface. The `jsonschema-go` version is **not** pinned by D1, which does not mention that
  module at all — it is pinned to whatever the chosen SDK version's own requirement names, which for
  `go-sdk v1.7.0` is `v0.4.3`. Read that requirement rather than trusting this sentence, and if it
  disagrees, follow the SDK's own file: the constraint is "the version the SDK already resolves to",
  not this number.
  `jsonschema-go` is direct because card 8 constructs the tool's input schema as a
  `jsonschema.Schema` value rather than letting the SDK infer one from a Go type.
  Do this with `go get` at the pinned versions followed by `go mod tidy`, not by hand-editing the
  requirement blocks. `go mod tidy` must leave the tree clean — a second run produces no diff.
  Add no other dependency. The SDK's own indirect requirements are whatever `go mod tidy` records;
  do not promote any of them to direct.
- **Commit:** `build(deps): add modelcontextprotocol/go-sdk v1.7.0 and jsonschema-go v0.4.3`

### Card 6: `internal/mcpserver` package skeleton and startup root resolution

- **Context:**
  - `internal/repopath/root.go`
  - `internal/cli/doc.go`
  - `quarry/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/doc.go`
  - `internal/mcpserver/root.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Write the package doc comment in `doc.go`, following the shape of the existing `internal/cli/doc.go`:
  what the package is, that it binds the `quarry` facade onto MCP tools, that the facade is its only
  route to the engine, and that nothing in it writes to standard output because that stream is
  reserved for the framed MCP transport.
  In `root.go`, declare `ResolveRoot(flagRoot, cwd string) (string, error)`. It delegates to
  `repopath.ResolveRoot` and, on failure, formats this surface's own startup-failure wording from the
  sentinel rather than echoing the `repopath` error text: `errors.Is` against
  `repopath.ErrNoRepositoryRoot` yields `quarry-mcp: no repository root found above <cwd>; pass --root`,
  `errors.Is` against `repopath.ErrRootNotDirectory` yields
  `quarry-mcp: --root is not a directory: <flagRoot as given>`, and anything else is returned wrapped
  behind a `quarry-mcp: ` prefix. On success it returns `repopath.ResolveRoot`'s value unchanged,
  which is always absolute.
  This function exists as a named exported symbol, rather than inline in `main`, for one stated
  reason: it is the `--root` path, which is what rescues a falsified cwd-inheritance assumption
  (discussion D3), the live probe does not exercise it, and `cmd/quarry-mcp` must stay untestable-by-
  design-because-trivial. Say so in its doc comment.
  Take `cwd` as a parameter; do not call `os.Getwd` here.
- **Commit:** `feat(mcpserver): add the package and its startup root resolution`

### Card 7: root-resolution table test

- **Context:**
  - `internal/mcpserver/root.go`
  - `internal/repopath/scratchtree_test.go`
  - `internal/repopath/root.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/root_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Write this file as an in-package test (`package mcpserver`), per this batch's scope note.
  Table-test `ResolveRoot` over four cases: the flag absent, resolving by discovery from the
  given working directory; the flag given as a relative path, joined against the working directory and
  absolutised; the flag given as an absolute path, taken as is; and the flag naming a file or a
  missing path, producing an error whose message carries the `quarry-mcp: --root is not a directory: `
  wording and echoes the path exactly as given.
  Build fixture trees with a per-package copy of the `writeScratchTree` helper, declared in this same
  file rather than as a separate one — this package needs a fixture tree in exactly one test file, so
  a standalone helper file would be a copy with a single caller. Use `.scratch/mcpserver-tests/` as
  its scratch subdirectory. Never call `t.TempDir()`, `t.Chdir` or `os.Chdir`.
  Assert the returned root is absolute in every success case.
- **Commit:** `test(mcpserver): table-test startup root resolution`

### Card 8: the `toc` tool — prose, schema, and handler

- **Context:**
  - `internal/cli/cli.go`
  - `internal/repopath/target.go`
  - `quarry/render.go`
  - `quarry/repo.go`
  - `quarry/quarry.go`
  - `internal/engine/answer.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/toc.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Declare four unexported string constants holding the tool's prose. Reproduce all four **verbatim**,
  ASCII only. They are fixed by discussion D5a and pinned by an exact-string assertion in batch 3;
  do not reword, reflow or extend them.

  Tool description:

  > Table of contents for a directory or file in this repository: its package, its files, each file's header comment, and optionally each file's symbols. Reach for this first to find your way around unfamiliar code, instead of listing directories and grepping for declarations.

  The `target` property's description:

  > Repository-relative path to a directory or a file. Use "" or "." for the repository root.

  The `depth` property's description:

  > How far to recurse into subdirectories. 0, the default, lists this directory's own files and names its subdirectories without descending; N fills N levels; -1 recurses to the bottom of the tree.

  The `symbols` property's description:

  > Populate every file entry's symbols: functions, methods, types, consts and vars. Omit for the per-target default, which is on for a file target and off for a directory target.

  Declare the input struct `tocInput` with three fields and their JSON tags: `Target string`
  (`target`), `Depth int` (`depth`), and `Symbols *bool` (`symbols`). `Symbols` is a pointer because
  the engine's `TOCOptions.Symbols` is a pointer and an absent property must map to `nil`, the
  per-target default, not to `false`. `Depth` is a plain `int` because an absent `depth` means `0`,
  which is a meaningful value on this surface, not a not-set marker.

  Declare `tocInputSchema() *jsonschema.Schema` returning an explicit object schema: `target` typed
  `string` and listed in `Required`; `depth` typed `integer` with `Minimum` set to `-1` via
  `jsonschema.Ptr`; `symbols` typed `boolean`; each property carrying its constant description above,
  and the schema itself carrying no `Required` entry for `depth` or `symbols`. The schema is written
  explicitly rather than inferred from `tocInput` because the SDK's inference has no way to express
  `minimum: -1` or these exact property descriptions.

  Declare `registerTOC(s *mcp.Server, repo *quarry.Repo, root string)`, which calls the SDK's
  generic `mcp.AddTool` instantiated as `mcp.AddTool[tocInput, any]` with a tool value carrying
  `Name: "toc"`, the tool-description constant, and `InputSchema` set to `tocInputSchema()`. The `any`
  output type parameter and a nil output value from the handler are what keep the tool free of an
  `outputSchema` and the result free of `structuredContent` — the SDK emits both for any tool that
  declares an output schema. Leave `OutputSchema` unset. Say this in the function's doc comment,
  because it is the one place the payload-shape decision can be violated silently.

  Declare the decision logic as a separate unexported function taking the repo, the absolute root and
  a `tocInput`, and returning a `*mcp.CallToolResult`. It performs the CLI's pipeline minus flag
  parsing and exit codes, in this fixed order:
  1. Reject a depth below `-1`. The message is exactly
     `--depth must be -1 (whole tree) or a non-negative integer, got <n>` with the offending integer
     substituted. This is the one place this surface's wording deliberately diverges from the CLI's,
     because `-1` is valid here and `"all"` does not exist here.
     Be accurate about what this branch is for, because the discussion's "the schema is advisory to a
     client that ignores it" framing does not hold for this SDK: `mcp.AddTool`'s generated wrapper
     validates the arguments against the input schema *before* the typed handler runs and returns its
     own error result on failure, so a call arriving over the protocol with `depth: -2` is rejected by
     the SDK against the schema's `minimum` and never reaches this branch. The branch is kept anyway,
     as the layer that owns the wording, and batch 3 asserts it directly rather than through the
     protocol. It earns its place: the engine's walk decrements depth with no floor and stops only at
     zero or at the whole-tree sentinel, so if the schema's `minimum` is ever dropped or the handler
     is reached from in-process code, an unvalidated negative depth is an unbounded walk that returns
     a plausible-looking answer instead of an error. Both layers stay; neither is removed to make the
     other reachable.
  2. Relativise the target with `repopath.RepoRelTarget`, passing the root as both the root and the
     base — this surface has no per-call working directory, so targets are repository-relative by
     definition. An `errors.Is` match on `quarry.ErrTargetOutsideRepo` produces
     `target outside repository: <target as given>`; any other error produces the internal-error form.
  3. `os.Lstat` the joined path, never `os.Stat`, so a symlink named as the target is treated as a
     file and not followed, matching the engine's own rule. A not-exist error produces
     `target not found: <repository-relative path>`; any other stat error produces the internal-error
     form.
  4. Call `(*quarry.Repo).TOC` with `quarry.TOCOptions{Depth: ..., Symbols: ...}`. Mirror the CLI's
     error classification: `quarry.ErrTargetNotFound` and `quarry.ErrTargetOutsideRepo` are answers
     and reuse the two messages above; anything else is internal. These branches are race-only in the
     common case — steps 2 and 3 have already excluded both sentinels — but reporting the race as
     success would be a false positive.
  5. Render with `quarry.RenderJSON`. A render failure produces the internal-error form.

  The success result is exactly one text content block whose text is `quarry.RenderJSON`'s bytes
  verbatim, with `IsError` unset, no `StructuredContent`, and no wrapper object, echoed target or
  status field.

  Declare one unexported helper producing the failure result: `IsError` set, exactly one text content
  block whose text is `string(quarry.RenderErrorJSON(msg))`. Every failure path above returns through
  it, so the failure envelope is written once. The internal-error form is that helper called with
  `"internal error: " + err.Error()`, matching the CLI's own prefix.

  Never return a non-nil error from the SDK handler for a query outcome: that channel is for protocol
  faults, and a client surfaces it as a tool malfunction rather than as an answer. A malformed call
  the SDK rejects against the input schema is the SDK's own error and is left alone.

  Do not add a second tool, a `targets` array, a compact or text view, or an output schema. If a
  second tool feels necessary, stop and raise it — that is a ladder cell first, not code.
  This file must not import the engine; every engine identifier it needs comes through the facade.
- **Commit:** `feat(mcpserver): add the toc tool, its schema and its handler`

### Card 9: server construction and the `quarry-mcp` binary

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/mcpserver/root.go`
  - `cmd/quarry/main.go`
  - `quarry/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/mcpserver.go`
  - `cmd/quarry-mcp/main.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `mcpserver.go`, declare an unexported `serverVersion` constant with the value `0.1.0`, and
  `NewServer(repo *quarry.Repo, root string) *mcp.Server`. `NewServer` calls
  `mcp.NewServer(&mcp.Implementation{Name: "quarry", Version: serverVersion}, nil)`, calls
  `registerTOC` on the result, and returns it. It registers exactly one tool.
  State in the doc comments that `quarry` and `toc` are external contracts, not choices: the ladder
  config declares `server: {name: quarry}` and `quarry_tools: [toc]`, and the harness composes
  `mcp__quarry__toc` from them, so any other spelling means the granted cell's tool is never allowed.
  State also that `serverVersion` is deliberately *not* a contract — nothing in the harness reads it;
  it tracks this package's own development rather than the module version.
  `NewServer` returns no error: it validates nothing, because both of its inputs are already resolved
  and validated by the caller.
  In `cmd/quarry-mcp/main.go`, write the whole binary and nothing more: declare a single `--root`
  string flag whose usage text says it overrides discovery from the working directory; parse flags;
  read the working directory; call `mcpserver.ResolveRoot`; call `quarry.Open` on the resolved root
  exactly once; write one line to standard error naming the absolute resolved root; construct the
  server; and run it over `&mcp.StdioTransport{}` with a background context.
  Every failure before the transport starts writes its message to standard error and exits non-zero.
  Nothing in this file writes to standard output — that stream carries framed MCP traffic only, and a
  stray write there is the one way this binary fails catastrophically and silently. Put that
  statement in the file's header comment, alongside the note that everything below it is testable
  in-process through the package precisely because this file holds no logic.
  Note in the header comment that the standard-error line serves interactive and operator use only:
  the ladder harness sets no standard-error sink on the measured process, so during a ladder run the
  line goes nowhere and a misrooting is observed instead from the answers themselves.
  Add no other flag. In particular there is no timeout, no config path, no state directory and no
  output-format override — those were V1's surface and this task deletes them by not rebuilding them.
- **Commit:** `feat(quarry-mcp): add the stdio MCP server binary`

### Card 10: layering test for both MCP packages

- **Context:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/toc.go`
  - `cmd/quarry-mcp/main.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/layering_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a test that walks every `.go` file — production and `_test.go` alike — in this package's own
  directory *and* in the `quarry-mcp` command's directory, parses each file's import block with
  `parser.ParseFile` under `parser.ImportsOnly`, and fails on any import path equal to
  `github.com/Knatte18/quarry/internal/engine` or carrying it as a path prefix.
  Locate both directories from `runtime.Caller(0)` rather than from the working directory, so the
  test does not depend on where it is run from.
  Fail the test when it parsed zero files: a layering check that goes green by finding nothing to
  check is worse than no check at all.
  Explain in the file header why this test exists rather than relying on review: the facade-only rule
  is a constraint the task body states, and the engine's own layering test has no row covering these
  two packages, so without this file the rule would be convention-only while the analogous engine
  rule is mechanical.
  Do not exclude `_test.go` files from the walk — the rule binds the test files too.
- **Commit:** `test(mcpserver): enforce the facade-only layering rule mechanically`

## Batch Tests

`verify: go build ./... && go test ./internal/mcpserver/...` builds every package in the module —
which is what proves `cmd/quarry-mcp` compiles and links the cgo tree-sitter backend, the point of
this batch — and then runs this batch's own two test files, the root-resolution table test and the
layering test. Scoping the test half to `internal/mcpserver` is correct: this batch adds no test
anywhere else and changes no existing package's behaviour. The protocol, golden, defaults, depth and
error-path tests all land in batch 3 and are run by the same command there. The overview's
module-wide `go vet ./...` runs at this batch's boundary as the cross-package check.

`go build ./...` needs `CGO_ENABLED=1` and a C toolchain, which is this repository's standing build
requirement and is already how the CLI and the engine build; no environment change is introduced here.
