# Batch: mcpserver-targetdir-removal

```yaml
task: "Rethink quarry-mcp's per-call targetDir ergonomics"
batch: "mcpserver-targetdir-removal"
number: 1
cards: 5
verify: go test ./internal/mcpserver/...
depends-on: []
```

## Batch Scope

This batch delivers the whole Go-side change: the call-wide `targetDir` property disappears from all
seven MCP tools' published input schemas, `effectiveTargetDir` is deleted, `resolveCall` loses its
override parameter, and every doc comment and jsonschema description string that described the
override as existing is corrected. It is one batch because the six input structs, `callcontext.go`,
and the package's own tests form a single compilation unit — deleting a struct field and the helper
it fed cannot be split across batches without leaving a batch boundary at a non-compiling tree.

The cards are ordered TDD-first: card 1 extends the transport-level schema matrix to assert the
property is absent, which fails against the current tree; card 2 removes the fields and the helper
and makes it pass. Cards 3-5 are the comment, description-string, and toc-test work that has no
compile dependency on that order.

No batch-local decision differs from `## Shared Decisions` in the overview.

The external interface batch 2 consumes is the finished behaviour this batch establishes: the server
is scoped once at launch, `toc_file`/`toc_dir` retain a partial absolute-path escape the five
language-server-backed tools do not, and no tool accepts a per-call target directory. Batch 2
documents exactly that and aligns the bench ladder's prompt to it.

## Cards

### Card 1: Assert targetDir is absent from every published schema, and that sending it fails the call

- **Context:**
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/tools_lsp_test.go`
  - `internal/mcpserver/schema_test.go`
- **Edits:**
  - `internal/mcpserver/transport_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Extend `TestToolsList_PerToolParameterMatrix` in
  `internal/mcpserver/transport_test.go` with a loop over the existing `wantToolNames` slice
  asserting that `schemaProperties(t, toolByName(t, res.Tools, name).InputSchema)` has no
  `"targetDir"` key, for all seven tools. Model it directly on the existing `buildTags`
  call-wide-absence loop in the same function, which already loops over a name list and checks
  `props["buildTags"]`. Use the call-wide `schemaProperties` helper, never `entryProperties` —
  `targetDir` was a call-wide property, not a per-entry one. Also extend the same function's own doc
  comment to name the new assertion alongside the ones it already lists.
  Then add a new test function `TestCallTool_TargetDirIsRejectedAsWholeCallError` to the same file,
  built on `newConnectedPair` and `newTransportTestConfig` exactly as
  `TestCallTool_MalformedCall_RejectedBeforeHandlerRuns` is: stub `definitionFn` with
  `withStubbedFacade(t, &definitionFn, failIfCalledLookupFn(t))`, call `textDocument_definition` with
  arguments `{"targets":[{"symbol":"S"}],"targetDir":"/somewhere/else"}`, and assert `CallTool`
  returns no protocol-level error but a result whose `IsError` is true and whose
  `StructuredContent` carries no decodable `results` array. Do not assert the SDK validator's exact
  error string — the observable contract is that the call failed as a whole and no per-entry
  `status: "error"` array came back. Do not edit `internal/mcpserver/schema_test.go` in this card or
  any other card of this batch: `TestInputSchemaFor_CallWidePropertySurvives` pins the call-wide
  `additionalProperties` behaviour the whole-call rejection depends on, and it is a fixture-level
  test, not the regression guard being added here.
  This card is expected to fail `go test ./internal/mcpserver/...` on its own; card 2 makes it pass.
- **Commit:** `test(mcpserver): assert no tool publishes a call-wide targetDir property`

### Card 2: Remove the TargetDir input fields, delete effectiveTargetDir, drop resolveCall's override

- **Context:**
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/mcpserver.go`
  - `internal/cli/toc.go`
- **Edits:**
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/callcontext_test.go`
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/tools_toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** This is one atomic change: the field removal, the helper deletion, the signature
  change, and the in-package test updates must land in a single commit, because the package does not
  compile between any two of them.
  Delete the `TargetDir string` field, its doc comment, and its `jsonschema` tag from all six input
  structs: `lspInput` (`internal/mcpserver/tools_lsp.go`), `symbolInput`
  (`internal/mcpserver/tools_symbol.go`), `impactInput` (`internal/mcpserver/tools_impact.go`),
  `assertInput` (`internal/mcpserver/tools_assert.go`), and both `tocFileInput` and `tocDirInput`
  (`internal/mcpserver/tools_toc.go`). Do not touch the `Lang`, `BuildTags`, `NoVerify`, or
  `DocSentences` fields on any of them.
  Delete the function `effectiveTargetDir` from `internal/mcpserver/callcontext.go` entirely,
  together with the now-unused `fmt` and `path/filepath` imports if nothing else in that file still
  needs them.
  Change `resolveCall`'s signature from `resolveCall(cfg Config, targetDirOverride, buildTags string)`
  to `resolveCall(cfg Config, buildTags string)`, and replace its first statement — the
  `effectiveTargetDir` call and its error check — with direct use of `cfg.TargetDir`. Keep every
  later step and its order exactly as it is: `cli.ResolveBuildTags`, then `cli.ResolveConfigPath`,
  then `quarry.LoadRegistry`, then `cli.ResolveStateDir`, with `cfg.TargetDir` passed where
  `absTargetDir` was passed before. `callContext.TargetDir` is now always `cfg.TargetDir`.
  Update the five `resolveCall` call sites to drop the override argument: `definitionHandler` and
  `referencesHandler` in `internal/mcpserver/tools_lsp.go`, `symbolHandler` in
  `internal/mcpserver/tools_symbol.go`, `impactHandler` in `internal/mcpserver/tools_impact.go`, and
  `assertHandler` in `internal/mcpserver/tools_assert.go` — each becomes
  `resolveCall(cfg, in.BuildTags)`.
  Update the two toc handlers in `internal/mcpserver/tools_toc.go`: `tocFileHandler` and
  `tocDirHandler` each currently open with an `effectiveTargetDir(cfg, in.TargetDir)` call and its
  error branch. Replace both with a plain `targetDir := cfg.TargetDir` and delete the error branch —
  `cfg.TargetDir` is already absolute, guaranteed by `ResolveLaunchTargetDir` and re-checked by
  `NewServer`'s `filepath.IsAbs` guard, so the branch would be unreachable and untestable. Both
  handlers must keep calling `tocPreflight` directly and must not be routed through `resolveCall`.
  In `internal/mcpserver/callcontext_test.go`, delete `TestEffectiveTargetDir_OverrideAbsolutised`
  and `TestEffectiveTargetDir_EmptyOverrideUsesLaunchDefault` — both test the deleted function, and
  the first also loses its subject entirely. Update `TestResolveCall_StateDirMatchesCLIResolution`
  and `TestResolveCall_BuildTagsSegment` to the new two-argument call shape. Add a new test
  `TestResolveCall_TargetDirIsAlwaysConfigTargetDir` asserting that `resolveCall(cfg, "")` returns a
  `callContext` whose `TargetDir` equals `cfg.TargetDir` and whose `StateDir` equals
  `cli.ResolveStateDir(cfg.StateDir, cfg.TargetDir, cli.ResolveBuildTags(""))` — that pairing is the
  invariant the whole task now rests on. The `StateDir` half deliberately restates the assertion
  `TestResolveCall_StateDirMatchesCLIResolution` already makes and this card retains; keep both.
  Asserting them as a pair is the point: `TargetDir == cfg.TargetDir` alone does not show that the
  state directory was derived from that same value, and the existing test alone does not show which
  target directory it was derived from. Neither test's assertion is redundant with the other once
  the override that could make them disagree is gone. Give the new test a doc comment stating that. Remove the `path/filepath` import from this test file if the
  deletions leave it unused.
- **Commit:** `feat(mcpserver)!: drop the per-call targetDir override from all seven tools`

### Card 3: Reword the jsonschema description strings that name targetDir as a parameter

- **Context:**
  - `internal/mcpserver/schema.go`
- **Edits:**
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/lspentry.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Edit only the text inside `jsonschema:"..."` struct tags in this card; Go `//`
  comments in the same two files are card 4's work. Five strings reference `targetDir` as if it were
  a parameter the caller can set, and each must instead name the server's target directory.
  In `internal/mcpserver/nativeentry.go`: `nativeEntry.File`'s tag, whose text ends
  `a plain path (absolute, or relative to targetDir); required with line+character, optional with symbol`;
  `nativeEntry.Within`'s tag, whose text ends `(relative to targetDir, or absolute)`; and
  `nativeEntry.Except`'s tag, whose text contains `file paths (relative to targetDir, or absolute)`.
  In `internal/mcpserver/lspentry.go`: `textDocumentIdentifier.URI`'s tag, whose text ends
  `as a file:// URI or a plain path (absolute, or relative to targetDir)`; and `lspEntry.Within`'s
  tag, whose text ends `(relative to targetDir, or absolute)`.
  In every case replace the bare token `targetDir` with the phrase `the server's target directory`,
  adjusting the surrounding wording so each string still reads as a grammatical sentence — for
  example `(absolute, or relative to the server's target directory)`. Change nothing else about the
  strings: the accepted-form guidance, the per-tool qualifiers, and the `assert_no_callers only`
  suffix on `Except` all stay as written. These strings are the only documentation the model reads
  per call, so a dangling reference to a removed parameter is a live documentation bug.
- **Commit:** `docs(mcpserver): name the server's target directory in tool schema descriptions`

### Card 4: Reword every doc comment that describes a per-call target-directory override

- **Context:**
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/schema.go`
  - `internal/cli/toc.go`
- **Edits:**
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/translate.go`
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/lspentry.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Edit only Go `//` comments in this card; `jsonschema:"..."` tag text is card 3's
  work. Three distinct classes of staleness must all be dispositioned, and a token grep finds only
  the first.
  Class one — comments naming `targetDir`, `TargetDir`, or a per-call override.
  In `internal/mcpserver/mcpserver.go`: the `Config` type's doc comment, whose closing clause
  `a handler never sees these raw values, only what resolveCall derives from them per call` becomes
  only half true now that the toc handlers read `cfg.TargetDir` directly — reword it rather than
  delete it; and `Config.TargetDir`'s own doc comment, whose clause
  `used whenever a call omits its own targetDir override` must go, leaving it describing the single
  project directory the server is rooted at.
  In `internal/mcpserver/callcontext.go`: the file header paragraph stating that the toc tools call
  `effectiveTargetDir` directly, which must instead say they read `Config.TargetDir` directly while
  keeping its existing reason (a malformed `servers.yaml` must not fail a toc call);
  `callContext.TargetDir`'s doc comment, whose `either the launch default or an absolutised per-call
  override` becomes `carried straight from Config.TargetDir` or equivalent; and `resolveCall`'s own
  doc comment, whose parenthetical crediting `effectiveTargetDir` with the first resolution step must
  name `Config.TargetDir` instead while leaving the rest of the documented ordering intact.
  In `internal/mcpserver/translate.go`: `resolveEntryFile`'s doc comment, whose clause
  `this package's own callers only ever pass ResolveLaunchTargetDir's or effectiveTargetDir's result`
  must drop the two-source framing and name `Config.TargetDir` as the single source.
  In `internal/mcpserver/tools_lsp.go`: the file header's `plus lang/buildTags/targetDir overrides`
  becomes `plus lang/buildTags overrides`.
  In `internal/mcpserver/tools_toc.go`: the file header's statement that each handler calls
  `effectiveTargetDir` and `tocPreflight` directly, and the `tocFileHandler` and `tocDirHandler` doc
  comments making the same claim — all three must say the handler reads `cfg.TargetDir` and calls
  `tocPreflight` directly, never `resolveCall`, preserving the existing reason each already gives.
  In `internal/mcpserver/nativeentry.go`: `nativeEntry.File`'s, `nativeEntry.Within`'s, and
  `nativeEntry.Except`'s doc comments, which each say `the call's targetDir`; and `exceptSet`'s doc
  comment, whose phrase `the effective absolute target directory` paraphrases the deleted helper
  without naming it and must become `the server's target directory`.
  In `internal/mcpserver/lspentry.go`: `textDocumentIdentifier.URI`'s and `lspEntry.Within`'s doc
  comments, which each say `the call's targetDir`.
  Class two — comments stating the override's existence purely by count, which contain neither
  spelling of the token. `lspInput`'s doc comment in `internal/mcpserver/tools_lsp.go`,
  `impactInput`'s in `internal/mcpserver/tools_impact.go`, and `assertInput`'s in
  `internal/mcpserver/tools_assert.go` each say
  `the three call-wide resolution overrides every language-server-backed tool in this package accepts`.
  That number is now two, `lang` and `buildTags`. Correct all three.
  Class three — back-references that must be re-read and confirmed, not blanket-edited. Re-read
  `symbolInput`'s doc comment in `internal/mcpserver/tools_symbol.go`, which says
  `the same call-wide resolution overrides lspInput carries`, and confirm it still describes the
  struct accurately after class two's correction; it is expected to need no edit, which is why
  `internal/mcpserver/tools_symbol.go` is read-only for this card. Re-read `tocFileInput`'s and
  `tocDirInput`'s doc comments in `internal/mcpserver/tools_toc.go`, which both say
  `plus the per-call overrides` the tool accepts — that phrasing is itself the stale wording this
  task removes, and both must be reworded to describe only the properties that remain.
  Leave every Go doc comment that names `targetDir` as a live function parameter unchanged:
  `nativeEntry.query`'s and `lspEntry.query`'s `targetDir` parameter, `resolveEntryFile`'s
  `targetDir` parameter, `exceptSet`'s `targetDir` parameter, `resolveTOCFileEntry`'s and
  `resolveTOCDirEntry`'s `targetDir` parameters in `internal/mcpserver/tools_toc.go` — both survive
  the change untouched, and both their signatures and their doc comments' references to that
  parameter stay exactly as they are — and the local `targetDir` variables the toc handlers now
  assign from `cfg.TargetDir` and pass into those two functions. Those all name real identifiers, not
  a removed tool property.
- **Commit:** `docs(mcpserver): correct every comment describing a per-call target directory`

### Card 5: Pin toc's surviving absolute-path escape hatch with a test

- **Context:**
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/tools_lsp_test.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/result.go`
  - `internal/cli/toc.go`
- **Edits:**
  - `internal/mcpserver/tools_toc_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a new test function `TestTOCFileHandler_AbsoluteTargetResolvesOutsideLaunchRoot`
  to `internal/mcpserver/tools_toc_test.go`. There is no such case today — every existing toc test
  roots its fixture under `cfg.TargetDir`. Build `cfg` with `newTestConfig(t)`, write a fixture file
  into a second, independent `t.TempDir()` that is not under `cfg.TargetDir` using the file's own
  `writeTOCTestFile` helper, stub `tocFileFn` with `withStubbedFacade` and `stubTOCFileFn` keyed by
  that fixture's absolute path, and call `tocFileHandler(cfg)` with a `tocFileInput` whose single
  target is that absolute path. Assert the entry's `Status` is `statusFound`, which proves
  `cli.ResolveTOCPath` ignored `targetDir` for an absolute argument. Give the function a doc comment
  naming this as the partial escape hatch that survives the per-call override's removal, and state
  that the five language-server-backed tools have no equivalent because their queries are served by
  the gopls rooted at the server's target directory.
  Also correct two stale comments in the same file. The doc comment on
  `TestTOCFileHandler_ResolvesQuarryYAMLAgainstFilesOwnDirectory` says the effective `DocSentences`
  value is resolved against the file's own parent directory `never the call's targetDir` — reword
  that clause to name the server's target directory. The doc comment on
  `TestTOCTools_MalformedServersYAMLStillSucceeds` says the handlers call `effectiveTargetDir` and
  `tocPreflight` directly — reword it to say they read `cfg.TargetDir` and call `tocPreflight`
  directly, matching card 4's wording for the same claim in the production file.
  Do not convert or delete any existing test in this file; no test here sets an input struct's
  `TargetDir`, so there is no override-case removal work in this card.
- **Commit:** `test(mcpserver): pin toc's absolute-path escape hatch outside the launch root`

## Batch Tests

`verify: go test ./internal/mcpserver/...` — the whole `internal/mcpserver` package suite. The scope
is the package this batch edits, and every file it touches is a member of it, so a broader command
would add cost without adding coverage. `go build ./...` runs as the overview's module-wide check at
the batch boundary and is what catches a break in `cmd/quarry-mcp`, the only consumer of this
package.

The batch's own new assertions are:

- `TestToolsList_PerToolParameterMatrix` (card 1, extended) — `targetDir` is absent from the
  call-wide properties of all seven registered tools, asserted through a real client/server pair over
  `ListTools`, so a tool that reintroduced the property in its registration path would be caught.
- `TestCallTool_TargetDirIsRejectedAsWholeCallError` (card 1, new) — a call carrying `targetDir`
  fails as a whole-call error with no `results` array, never as a per-entry `status: "error"`. This
  is the only place the hard-removal behaviour is observable, since it is the SDK's own validator
  that rejects the call.
- `TestResolveCall_TargetDirIsAlwaysConfigTargetDir` (card 2, new) — `callContext.TargetDir` equals
  `cfg.TargetDir` and the derived `StateDir` matches `cli.ResolveStateDir`'s own result for the same
  inputs. This is the invariant the removal rests on.
- `TestTOCFileHandler_AbsoluteTargetResolvesOutsideLaunchRoot` (card 5, new) — an absolute `target`
  still resolves outside the launch root for `toc_file`, pinning the partial escape hatch batch 2
  documents.

Card 1 is expected to fail this batch's `verify:` on its own; it is written first deliberately, and
card 2 is what turns it green. The batch as a whole must end green.

`TestInputSchemaFor_CallWidePropertySurvives` in `internal/mcpserver/schema_test.go` is deliberately
left untouched by every card in this batch — it pins the call-wide `additionalProperties` strictness
the whole-call rejection depends on, and weakening or moving it would remove the foundation card 1's
new transport case stands on.
