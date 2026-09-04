# Batch: mcp-server-tests

```yaml
task: "MCP, thin (T6)"
batch: "mcp-server-tests"
number: 3
cards: 6
verify: go test ./internal/mcpserver/...
depends-on: [2]
```

## Batch Scope

This batch is the automated half of discussion D13's §9a gate: an in-process client talks to the
server over the SDK's in-memory transport and asserts everything that would otherwise break T7
silently — the tool count and its schema, the four fixed prose strings, the absence of an
`outputSchema`, the exact payload bytes, the pointer-vs-bool default semantics, the depth validation,
and the failure wording. It is one batch because every card shares the same committed fixture
repository and the same client-session helper, and because none of it changes production code.

Every test here goes through the protocol rather than calling `tocResult` — batch 2 card 8's handler
decision function — with one stated exception, card 15's negative-depth wording. The properties being
pinned are wire properties, and a test that called `tocResult` in-process would not catch an
`outputSchema` the SDK derived or a `structuredContent` it attached. The exception exists because the
SDK validates arguments against the input schema before the handler runs, so `tocResult`'s own
negative-depth rejection is unreachable over the protocol while the schema carries its `minimum` —
the assertion that pins that message therefore has to call the function. Every file here is an
in-package test (`package mcpserver`), per batch 2's scope note, so that call needs no export.

Batch-local decision: the fixture is a committed tree under this package's own `testdata/`, following
`internal/engine/testdata/`'s precedent rather than `internal/cli`'s programmatic `writeScratchTree`
trees. A committed tree makes golden bytes obviously stable, which is the property these tests exist
for, and Go's toolchain ignores `testdata/` when building, so the committed `.go` fixture files are
inert.

## Cards

### Card 11: committed fixture repository and the session helper

- **Context:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/engine/testdata/tree/pkg/alpha.go`
  - `internal/engine/testdata/tree/sub/doc.go`
  - `quarry/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/testdata/repo/alpha/doc.go`
  - `internal/mcpserver/testdata/repo/alpha/alpha.go`
  - `internal/mcpserver/testdata/repo/alpha/sub/leaf.go`
  - `internal/mcpserver/fixture_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Write a small committed fixture repository under this package's `testdata/`: a package directory
  with a doc file carrying a package comment, a second file declaring one exported type, one method
  on it, one exported function, one package-level const and one package-level var, and a nested
  subdirectory holding a single file in its own package. Keep every symbol and every header comment
  short and ASCII — these bytes end up inside every golden in card 13, so a long comment makes the
  goldens harder to review for no gain. Keep the fixture stable: no build tags, no generated code, no
  imports beyond the standard library, and preferably no imports at all.
  Write `fixture_test.go` as an in-package test (`package mcpserver`), per batch 2's scope note, and
  declare two helpers in it. The first returns the fixture repository's absolute path, resolved from
  `runtime.Caller(0)` so it does not depend on the working directory, and fails the test if the
  directory is missing. The second takes a repository root as a parameter, opens it with
  `quarry.Open`, constructs a server with `NewServer`, wires it to a client over
  `mcp.NewInMemoryTransports`, connects both sides, registers `t.Cleanup` closing both sessions, and
  returns the connected client session. It takes the root as a parameter rather than hard-coding the
  fixture because card 16 runs one case against a scratch tree the committed fixture cannot hold.
  Every later card calls the second helper; none of them repeats the wiring.
  State in the file header why the fixture is committed rather than built at test time, naming the
  engine's `testdata/` precedent and the golden-stability reason.
- **Commit:** `test(mcpserver): add the committed fixture repository and the session helper`

### Card 12: `tools/list` — one tool, its schema, its prose, no output schema

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/mcpserver/fixture_test.go`
  - `internal/mcpserver/mcpserver.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/tools_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Call `ListTools` on the connected client session and assert, in one test file:
  the result carries exactly one tool, and its name is `toc` — an accidental second tool is one of the
  two drifts that would silently invalidate T7, and this is the assertion that catches it;
  the tool carries **no** output schema, which is what enforces the no-`structuredContent` decision at
  its actual point of violation, since the SDK emits structured content for any tool that declares
  one;
  the input schema is an object with `target` required and `depth` and `symbols` not required;
  `depth` is typed integer and carries a minimum of `-1`;
  `target` is typed string and `symbols` is typed boolean.
  Assert the tool description and all three property descriptions as **exact strings**, spelled out
  literally in the test rather than referenced from the production constants — a test that compares a
  constant against itself pins nothing. These four strings are the granted ladder cell's entire
  prompt cost and the thing that decides whether the agent calls the tool, so a reword must be a
  deliberate, reviewed change rather than drift.
  Note in the file header that the schema arrives on the client side as generic decoded JSON, not as
  the server's own schema value, so the assertions read it as decoded JSON.
- **Commit:** `test(mcpserver): pin the toc tool's shape, schema and prose`

### Card 13: golden payload bytes, and the CLI-mirror assertion

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/mcpserver/fixture_test.go`
  - `internal/engine/golden_test.go`
  - `internal/engine/loomyard_test.go`
  - `quarry/render.go`
  - `quarry/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/toc_golden_test.go`
  - `internal/mcpserver/testdata/golden/toc-dir.json`
  - `internal/mcpserver/testdata/golden/toc-file.json`
  - `internal/mcpserver/testdata/golden/toc-dir-depth1.json`
  - `internal/mcpserver/testdata/golden/toc-dir-depth-all.json`
  - `internal/mcpserver/testdata/golden/toc-dir-symbols-true.json`
  - `internal/mcpserver/testdata/golden/toc-file-symbols-false.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Call the tool through the client session for six cases against the fixture repository and compare
  the first content block's text byte-for-byte against its own committed golden: the package
  directory with default options; that directory's second source file as a file target with default
  options; the repository root at depth 1; the repository root at depth -1; the package directory with
  symbols requested; and the file target with symbols suppressed. One golden per case, named for the
  case.
  Assert in every case that the result carries exactly one content block, that it is a text block,
  and that the error flag is unset.
  Assert in at least one case that the result carries **no** structured content. That is the wire-side
  half of the decision card 12's output-schema assertion covers on the declaration side.
  Add one assertion that goes further than the goldens: for the same target and the same options, the
  text the tool returned and `quarry.RenderJSON` of the facade's own answer are identical bytes. That
  single assertion is what makes "a mirror of the CLI" testable rather than aspirational, and unlike
  the goldens it cannot rot into agreeing with a wrong implementation.
  Follow the engine's golden convention: declare a `-update` flag and, when it is set, rewrite each
  case's own committed golden from the current run instead of comparing against it. Generate the six
  committed goldens with that flag rather than hand-writing them — a hand-written golden pins the
  wrong bytes and then passes forever. Say so in the file header, aimed at the next `-update` run.
- **Commit:** `test(mcpserver): golden the toc payload bytes and assert the CLI mirror`

### Card 14: absent-property semantics

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/mcpserver/fixture_test.go`
  - `internal/engine/answer.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/toc_defaults_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Assert that a call omitting the symbols property produces the engine's per-target default and not a
  suppressed-symbols answer: for a file target the file entry carries its symbols, and for a directory
  target it does not. This is the pointer-versus-bool trap in the engine's options struct — an
  implementation that maps an absent property to false rather than to nil passes every other test in
  this batch — so it gets its own named test rather than riding along in a golden.
  Assert separately that a call omitting the depth property behaves as depth 0: the target's own files
  are listed and its subdirectories are named without being descended into. Separate failure mode,
  separate assertion.
  Read the assertions off the decoded payload rather than off a golden, so the test states the
  property it is pinning instead of restating a byte string.
- **Commit:** `test(mcpserver): pin the absent-symbols and absent-depth defaults`

### Card 15: depth validation

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/mcpserver/fixture_test.go`
  - `internal/engine/walk.go`
  - `quarry/render.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/toc_depth_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Assert through the client session that depth -1 is accepted and recurses to the bottom of the
  fixture tree — the nested subdirectory's file appears in the answer.
  Assert through the client session that depth -2 and depth -7 are each rejected: the error flag is
  set and the result carries exactly one text block whose text is not a table-of-contents answer.
  Do not assert the rejection wording here. This rejection comes from the SDK's own schema
  validation, which runs before the handler, so the text is the SDK's arguments-validation message
  and not this server's — pinning it would pin a dependency's private wording, and it is not what
  the surface promises.
  Assert the wording instead by calling `tocResult` directly, in this same file, for depth -2 and
  depth -7: the returned result has the error flag set, carries exactly one text
  block, and its text equals the facade's failure envelope built from the message
  `--depth must be -1 (whole tree) or a non-negative integer, got <n>` with the offending integer
  substituted. This is the batch's one deliberate departure from protocol-only testing, for the
  reason stated in Batch Scope; keep it to this one assertion.
  State in the file header why this is the load-bearing one of this batch's schema tests, and be
  accurate about the two layers: the schema's minimum is what rejects a protocol call, the handler's
  own check is what owns the wording and what would still reject if the minimum were ever dropped,
  and the reason either is needed at all is that the engine's directory walk decrements depth without
  a floor and stops only at zero or at the whole-tree sentinel — so an unvalidated negative depth is
  an unbounded walk that returns a plausible-looking answer rather than an error, a defect that would
  reach T7 as a cost measurement rather than as a failure.
- **Commit:** `test(mcpserver): pin depth -1 and the negative-depth rejection`

### Card 16: error paths, their wording, and the never-follow rule

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/mcpserver/fixture_test.go`
  - `internal/mcpserver/root_test.go`
  - `internal/cli/cli.go`
  - `internal/engine/repo.go`
  - `internal/engine/toc.go`
  - `quarry/render.go`
  - `quarry/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/toc_errors_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Assert two failure cases against the committed fixture repository: a target that does not exist, and
  a target that escapes the repository.
  For each, assert the error flag is set, the result carries exactly one text block, and its text
  equals the facade's failure envelope built from the CLI's own wording for that condition — the
  not-found sentence naming the repository-relative path, and the outside-repository sentence naming
  the target exactly as the caller gave it. Assert the wording, not just the flag: the whole claim
  being tested is that the two surfaces say the same thing.
  Then assert a third case which is deliberately **not** a failure: a target that is a broken symbolic
  link. It succeeds. Both the handler and the engine stat with `os.Lstat` and never `os.Stat`, so a
  symbolic link named directly as the target is answered as a file rather than followed, and the
  engine emits it as a name-only file entry with no error. Assert that shape: the error flag is unset,
  and the answer carries the link as an entry under its own name. Do not assert a failure envelope
  here — an implementation that returned one would be following the link, which is the rule this case
  exists to pin.
  A symbolic link is something the committed fixture cannot carry portably, so build this case's tree
  under the repository's gitignored scratch directory with the same per-package `writeScratchTree`
  approach card 7 uses, open it with `quarry.Open` as its own repository, and construct a server over
  it with the session helper. Never call `t.TempDir()`.
  Skip the symbolic-link case on a platform that cannot create one, rather than failing.
- **Commit:** `test(mcpserver): pin the two failure paths and the never-follow symlink rule`

## Batch Tests

`verify: go test ./internal/mcpserver/...` runs this batch's five new test files together with
batch 2's two. Scoping to this one package is correct and deliberate: every file this batch creates
lives in it, no production code changes, and no other package imports it. The overview's module-wide
`go vet ./...` runs at this batch's boundary as the cross-package check.

The six goldens under `testdata/golden/` are produced by running this package's tests once with the
`-update` flag and are then committed; a later run without the flag is what turns them into a gate.
The live `claude -p` probe is deliberately not here — see the overview's Shared Decision on it.
