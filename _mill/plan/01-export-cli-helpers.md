# Batch: export-cli-helpers

```yaml
task: "Add an MCP wrapper for quarry"
batch: "export-cli-helpers"
number: 1
cards: 5
verify: go test ./internal/cli/... && go test -tags lsp ./internal/cli/... && go test ./internal/quarryengine/query/...
depends-on: []
```

## Batch Scope

This batch exports the thirteen `internal/cli` helpers named in `_mill/discussion.md`'s
`export-inventory` decision, so `internal/mcpserver` can reuse the CLI's exact state-directory
keying, `within` semantics, caller-exclusion, doc-sentence resolution, path resolution, and struct
marshalling instead of forking them. It is one batch because it is a single mechanical operation
over one package with no new behaviour: rename the identifier, update every call site, update every
test, update the prose in three engine doc comments that name one of the renamed helpers.

The external interface batch 2 consumes is exactly these thirteen exported identifiers:
`ResolveConfigPath`, `ResolveStateDir`, `ResolveBuildTags`, `AbsOrJoin`, `FilterWithin`,
`FilterUnexpectedCallers`, `FilterImpactWithin`, `ParseDocSentences`, `ResolveDocSentences`,
`StructToFields`, `ResolveTOCPath`, `ValidateTOCLang`, `TOCDirEntries`.

Batch-local decision beyond `## Shared Decisions`: `isWithinDir`, `loadTOCConfig`,
`resolveTOCConfigPath`, `resolveTOCBaseDir`, `workspaceKey`, and `buildTagsSegment` deliberately
stay unexported. Each is reached only through one of the thirteen above, and `resolveTOCBaseDir`
has no MCP caller at all because target-dir resolution hands every handler an absolute directory.

## Cards

### Card 1: Export the three `internal/cli/paths.go` resolvers

- **Context:**
  - `quarry/facade.go`
- **Edits:**
  - `internal/cli/paths.go`
  - `internal/cli/cli.go`
  - `internal/cli/impact.go`
  - `internal/cli/paths_test.go`
  - `internal/cli/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename `resolveConfigPath` to `ResolveConfigPath`, `resolveStateDir` to
  `ResolveStateDir`, and `resolveBuildTags` to `ResolveBuildTags` in `internal/cli/paths.go`,
  keeping each signature, body, and doc comment semantics unchanged (update the identifier
  spelling inside the doc comments and inside the file's own header comment where they are named).
  Update the prose in `resolveContext`'s doc comment in `internal/cli/cli.go`, which names
  `resolveConfigPath` and `resolveStateDir` in three places.
  Update every call site: `resolveContext` in `internal/cli/cli.go` calls `ResolveConfigPath` and
  `ResolveStateDir`; the `RunE` bodies in `internal/cli/cli.go` and `internal/cli/impact.go` call
  `ResolveBuildTags`. Update the test call sites in `internal/cli/paths_test.go` and
  `internal/cli/resolve_test.go`, including test function names and failure-message strings that
  spell the old identifier. The unexported `workspaceKey` and `buildTagsSegment` are not renamed.
- **Commit:** `refactor(cli): export ResolveConfigPath, ResolveStateDir, ResolveBuildTags`

### Card 2: Export `AbsOrJoin`, `FilterWithin`, `FilterUnexpectedCallers`

- **Context:**
  - `quarry/facade.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/toc.go`
  - `internal/cli/impact.go`
  - `internal/cli/cli_test.go`
  - `internal/cli/assertnocallers_lsp_test.go`
  - `internal/quarryengine/query/callers.go`
  - `internal/quarryengine/query/callers_test.go`
  - `internal/quarryengine/impact/impact.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename `absOrJoin` to `AbsOrJoin`, `filterWithin` to `FilterWithin`, and
  `filterUnexpectedCallers` to `FilterUnexpectedCallers` in `internal/cli/cli.go`, keeping each
  signature and body unchanged. Update **every** call site of all three in
  `internal/cli/cli.go`, not only the ones named here: `AbsOrJoin` is called from
  `definitionCommand`'s `RunE`, `parseQuery`, and `inFileQuery`; `FilterWithin` is called from
  `refsCommand`'s `RunE` (twice), `definitionCommand`'s `RunE` (three times), and
  `assertNoCallersCommand`'s `RunE`; `FilterUnexpectedCallers` is called from
  `assertNoCallersCommand`'s `RunE`. Grep the file for each old spelling and leave none behind —
  a missed site is a compile failure, not a silent divergence. `isWithinDir` stays unexported and is not renamed. Update the prose
  references to the old spellings in the doc comments of `resolveTOCPath` in
  `internal/cli/toc.go`, of `filterImpactWithin` in `internal/cli/impact.go`, and in
  `internal/cli/cli_test.go` (call sites, test names, failure-message strings) and
  `internal/cli/assertnocallers_lsp_test.go` (a comment reference only, no call site). In
  `internal/quarryengine/query/callers.go`, `internal/quarryengine/query/callers_test.go`, and
  `internal/quarryengine/impact/impact.go`, update only the comment prose that names
  `filterUnexpectedCallers` — do not change any Go statement in those three files.
- **Commit:** `refactor(cli): export AbsOrJoin, FilterWithin, FilterUnexpectedCallers`

### Card 3: Export `FilterImpactWithin`

- **Context:**
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/impact.go`
  - `internal/cli/impact_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename `filterImpactWithin` to `FilterImpactWithin` in
  `internal/cli/impact.go`, keeping its signature and body unchanged, including the guarantee that
  the returned `Callers` slice is non-nil so it marshals as `[]`. Update the two call sites in
  `impactCommand`'s `RunE` (the single-argument branch and the `runBatch` closure). Update
  `internal/cli/impact.go`'s own file header comment, which names `filterImpactWithin` in its list
  of the file's functions. Update the call sites, test names, and failure-message strings in
  `internal/cli/impact_test.go`.
- **Commit:** `refactor(cli): export FilterImpactWithin`

### Card 4: Export `ParseDocSentences` and `ResolveDocSentences`

- **Context:**
  - `quarry/facade.go`
- **Edits:**
  - `internal/cli/tocconfig.go`
  - `internal/cli/toc.go`
  - `internal/cli/tocconfig_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename `parseDocSentences` to `ParseDocSentences` and `resolveDocSentences` to
  `ResolveDocSentences` in `internal/cli/tocconfig.go`, keeping both signatures and bodies
  unchanged. `loadTOCConfig` stays unexported and is not renamed. Update the call sites: the
  up-front flag validation in `tocFileCommand`'s `RunE` in `internal/cli/toc.go` calls
  `ParseDocSentences` directly, and `tocFileOne` in the same file calls `ResolveDocSentences`.
  Update the call sites, test names, and failure-message strings in
  `internal/cli/tocconfig_test.go`.
- **Commit:** `refactor(cli): export ParseDocSentences and ResolveDocSentences`

### Card 5: Export `StructToFields`, `ResolveTOCPath`, `ValidateTOCLang`, `TOCDirEntries`

- **Context:**
  - `quarry/facade.go`
- **Edits:**
  - `internal/cli/toc.go`
  - `internal/cli/impact.go`
  - `internal/cli/toc_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rename `structToFields` to `StructToFields`, `resolveTOCPath` to
  `ResolveTOCPath`, `validateTOCLang` to `ValidateTOCLang`, and `tocDirEntries` to `TOCDirEntries`
  in `internal/cli/toc.go`, keeping every signature and body unchanged. `resolveTOCBaseDir` and
  `classifyTOCError` stay unexported and are not renamed. Update the call sites in
  `internal/cli/toc.go`: `tocFileCommand`'s and `tocDirCommand`'s `RunE` call `ValidateTOCLang`;
  `tocFileOne` and `tocDirOne` call `ResolveTOCPath`; `tocFileOne` and `tocDirEntries` call
  `StructToFields`; `tocDirOne` calls `TOCDirEntries`. Update the call sites in
  `internal/cli/impact.go`: `emitImpactResult` and `classifyImpactError` call `StructToFields`, and
  the doc comments of `emitImpactResult`, `classifyImpactError`, and `rewordImpactMarshalFailure`
  name it in prose. Update the prose inside `internal/cli/toc.go` too, where `TOCDirEntries`' own
  doc comment names `structToFields` twice. Update the call sites, test names, and failure-message strings in
  `internal/cli/toc_test.go`.
- **Commit:** `refactor(cli): export StructToFields, ResolveTOCPath, ValidateTOCLang, TOCDirEntries`

## Batch Tests

`verify:` runs the existing `internal/cli` suite twice: once untagged
(`internal/cli/cli_test.go`, `impact_test.go`, `toc_test.go`, `tocconfig_test.go`,
`paths_test.go`, `resolve_test.go`, `cwdcontext_test.go`, `exec_test.go`), and once with
`-tags lsp` so `assertnocallers_lsp_test.go`, `impact_lsp_test.go`, and
`refs_targetdir_lsp_test.go` are compiled. The tagged run `t.Skip`s each of its tests because
`gopls` is not on `$PATH` here, so its value is the compile, which is exactly what a rename batch
needs from it: a rename that missed a call site in a build-tagged file fails to compile rather than
passing silently. No new tests are added — this batch changes no behaviour, so the existing suite
passing unchanged is the assertion. The third chained invocation,
`go test ./internal/quarryengine/query/...`, exists solely to compile
`internal/quarryengine/query/callers_test.go`, whose comment prose card 2 edits: the boundary gate
`go build ./...` does not compile `_test.go` files, so without it nothing in the plan would build
that edit.
