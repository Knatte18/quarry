# Batch: toc-tools

```yaml
task: "Add an MCP wrapper for quarry"
batch: "toc-tools"
number: 5
cards: 3
verify: go test ./internal/mcpserver/...
depends-on: [4]
```

## Batch Scope

This batch delivers the two tree-sitter-backed tools, `toc_file` and `toc_dir`. It is one batch
because both take plain-string entries rather than objects, both validate `lang` against
`quarry.TOCLanguages()` rather than the servers.yaml registry, and neither ever loads that registry
or resolves a state directory — a whole-call split that is easiest to get right when both tools are
written together.

The external interface batch 6 consumes: the two `register*` functions now called from
`NewServer`, and the whole-call validation order this batch establishes for the toc tools.

Batch-local decisions beyond `## Shared Decisions`:

- The toc tools' whole-call failure set is not the LSP tools'. Schema validation and `targetDir`
  absolutisation fail a toc call, and so do `cli.ValidateTOCLang` and `cli.ParseDocSentences` on a
  non-empty `docSentences` — both of which the CLI performs once, up front, before any argument is
  processed. Config-path resolution, `quarry.LoadRegistry`, and state-directory resolution do
  **not** fail a toc call, because `tocFileCommand` and `tocDirCommand` never call them; failing
  there would diverge from a CLI invocation that succeeds.
- The toc status rule is toc's own, not the LSP predicates: `os.IsNotExist` on the stat is
  `not_found`; a directory passed to `toc_file` or a file passed to `toc_dir` is `error`;
  `quarry.ErrLanguageUnsupported` is `error` worded from `quarry.TOCImplemented()`; any other error
  is `error`. `ambiguous` is unreachable here — toc uses no language server and no symbol
  resolution.
- `toc_dir`'s per-file `path` stays caller-relative, composed by `cli.TOCDirEntries` against the
  argument as the caller wrote it. This is the one documented exception to the always-absolute
  output rule, and it exists so the value round-trips straight into a following `toc_file` call.

## Cards

### Card 21: Add toc entry handling and whole-call validation

- **Context:**
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/result.go`
  - `internal/cli/toc.go`
  - `internal/cli/tocconfig.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/tocentry.go`
  - `internal/mcpserver/tocentry_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tocentry.go` declaring
  `func tocPreflight(lang string, doc docSentences) (string, error)` which runs the toc tools'
  whole-call validations in the CLI's own order: `cli.ValidateTOCLang(lang)` first, then, when
  `doc` carries a value, `cli.ParseDocSentences` on that value's string form, discarding the parsed
  int and keeping the string so each entry can re-resolve it against its own directory. It returns
  the string form of `docSentences` (empty when unset) and any validation error, which the caller
  turns into a whole-call failure. Declare
  `func tocStat(abs string, wantDir bool) (string, string, error)` returning the per-entry status
  and message for the stat step: `os.IsNotExist` yields `statusNotFound` carrying the stat error's
  own message, a directory when `wantDir` is false yields `statusError` with the CLI's own wording
  from `tocFileOne`, a non-directory when `wantDir` is true yields `statusError` with the CLI's own
  wording from `tocDirOne`, and any other stat error yields `statusError` carrying its message.
  Document that applying the LSP predicates here instead would report `error` where the CLI reports
  `not_found` for a missing file, which is a divergence rather than a mapping. Create
  `internal/mcpserver/tocentry_test.go` asserting that an invalid `lang` and an invalid
  `docSentences` each produce a `tocPreflight` error, that `docSentences` given as `3` and as
  `"all"` both pass and both reach `cli.ParseDocSentences` as a string, that an unset
  `docSentences` skips the parse entirely, and that `tocStat` returns `not_found` for a missing
  path and `error` for the wrong file type in each direction.
- **Commit:** `feat(mcpserver): add toc pre-flight validation and stat classification`

### Card 22: Add the `toc_file` and `toc_dir` tools

- **Context:**
  - `internal/mcpserver/tocentry.go`
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/result.go`
  - `internal/mcpserver/schema.go`
  - `internal/mcpserver/facade.go`
  - `internal/cli/toc.go`
  - `internal/cli/tocconfig.go`
  - `internal/cli/paths.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/mcpserver/mcpserver.go`
- **Creates:**
  - `internal/mcpserver/tools_toc.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_toc.go` declaring
  `type tocFileInput struct { Targets []string; Lang string; DocSentences docSentences; TargetDir string }`
  with json tags `targets`, `lang,omitempty`, `docSentences,omitempty`, `targetDir,omitempty`, and
  `type tocDirInput struct { Targets []string; Lang string; TargetDir string }` with no
  `docSentences` property — the CLI registers `--doc-sentences` on `toc file` only, deliberately,
  because `toc dir` emits headers and never docstrings. Neither input type declares `buildTags`:
  toc is tree-sitter-backed, never loads the server registry, and registers no `--build-tags`. Each
  `lang` property's `jsonschema` description states that it is validated against
  `quarry.TOCLanguages()` rather than a servers.yaml registry key, that on `toc_file` it overrides
  detection, and that on `toc_dir` it restricts which extensions are listed. Declare
  `type tocFileEntry struct { Target string; Status string; Result map[string]any; Error string }`,
  `type tocDirEntry struct { Target string; Status string; Files []any; Error string }`, and their
  `Results` wrappers; neither declares a `resolution` or `candidates` key, because toc consults no
  language server and `ambiguous` is unreachable there. Declare
  `registerTOCTools(s *mcp.Server, cfg Config) error` following card 14's registration shape. Each
  handler calls `effectiveTargetDir(cfg, in.TargetDir)` and then `tocPreflight`, returning either
  error unchanged as a whole-call failure, and never calls `resolveCall`. It then runs `runTargets`
  over `in.Targets`. `toc_file` per entry: `abs := cli.ResolveTOCPath(targetDir, arg)`, then
  `tocStat(abs, false)`, then `cli.ResolveDocSentences(docString, filepath.Dir(abs))` — the config
  base is each resolved file's own parent directory, exactly as `tocFileOne` sets it, and never the
  call's `targetDir`, because reusing that would pick up a different `.quarry.yaml` than the CLI
  does for the identical argument — then `tocFileFn(abs, in.Lang, quarry.TOCOptions{DocSentences: resolved})`,
  mapping an error through `classifyTOCError` and a success through `cli.StructToFields` under the
  entry's `Result` key. `toc_dir` per entry: `abs := cli.ResolveTOCPath(targetDir, arg)`, then
  `tocStat(abs, true)`, then `tocDirFn(abs, in.Lang)`, mapping an error through `classifyTOCError`
  and a success through `cli.TOCDirEntries(arg, result)` under the entry's `Files` key — passing
  the argument as the caller wrote it, never `abs`, so each file's composed `path` stays
  caller-relative and round-trips into a following `toc_file` call. A `cli.StructToFields` or
  `cli.TOCDirEntries` failure is that entry's `status: statusError` carrying the error's own
  message verbatim, exactly as `tocFileOne` and `tocDirOne` dispose of the same failure;
  `rewordMarshalFailure` is `impact`-only and must not be applied here, because the `toc: `
  prefix is correctly attributed for a toc call. `cli.TOCDirEntries` is
  mandatory here and `cli.StructToFields` alone is not sufficient, because `toc.DirEntry.Name`
  carries `json:"-"` and the marshalled entries would otherwise carry neither `name` nor `path`.
  Each entry's `Target` echoes the input path string verbatim. Each tool's `Description` opens with
  a sentence stating that line numbers in its output are 1-based, then states that entries are
  plain paths. Edit `NewServer` in `internal/mcpserver/mcpserver.go` to call
  `registerTOCTools(s, cfg)` and return its error.
- **Commit:** `feat(mcpserver): add the toc_file and toc_dir tools`

### Card 23: Handler unit tests for the toc tools

- **Context:**
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/tocentry.go`
  - `internal/mcpserver/facade.go`
  - `internal/mcpserver/result.go`
  - `internal/cli/toc.go`
  - `internal/cli/toc_test.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/mcpserver/tools_toc_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/mcpserver/tools_toc_test.go` driving both handlers directly.
  The stat, path-composition, and config-base assertions run against real fixture files written
  into a `t.TempDir()`; the facade-outcome assertions replace `tocFileFn` and `tocDirFn` with
  stubs restored by `t.Cleanup`. Assert: a missing path is `status: "not_found"`; a directory
  passed to `toc_file` and a file passed to `toc_dir` are each `status: "error"`; a stub returning
  a wrapped `quarry.ErrLanguageUnsupported` is `status: "error"` with a message worded from
  `quarry.TOCImplemented()` and never `not_found`; `toc_dir` entries carry a `path`, which proves
  `cli.TOCDirEntries` was applied rather than `cli.StructToFields` alone; that `path` is
  caller-relative and composed against the argument as written, so it round-trips into a
  `toc_file` call; `toc_file` resolves its `.quarry.yaml` against the target file's own parent
  directory and not against the call's `targetDir`, verified with two fixture directories carrying
  different `doc_sentences` values; `docSentences` sent as `3` and as `"all"` both succeed;
  `toc_file`'s `result` wrapper key carries the marshalled `quarry.TOCFileResult`; a marshal
  failure is a per-entry `error` carrying the message verbatim, with no `impact: ` rewording; an invalid
  `lang` and an invalid `docSentences` each fail the whole call; and a `Config.ConfigPath` pointing
  at a malformed `servers.yaml` leaves both toc tools succeeding, because neither loads the
  registry.
- **Commit:** `test(mcpserver): cover the toc_file and toc_dir handlers`

## Batch Tests

`verify: go test ./internal/mcpserver/...` runs `tocentry_test.go` and `tools_toc_test.go`
alongside every earlier test in the package, which must keep passing because card 22 edits
`NewServer`. Scope is the one package these cards touch. The malformed-`servers.yaml` assertion and
the per-entry config-base assertion are the two behavioural divergences from the CLI that nothing
else in the plan would catch: the first would make a toc call fail where the CLI succeeds, and the
second would silently read a different `.quarry.yaml` than the CLI reads for the identical
argument.
