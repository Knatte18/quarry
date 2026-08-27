# Batch: facade-and-cli

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "facade-and-cli"
number: 6
cards: 7
verify: go test ./internal/quarryengine/toc ./quarry ./internal/cli
depends-on: [5]
```

## Batch Scope

This batch makes the engine work reachable: the `quarry/` facade gains its re-exports, and
`internal/cli` gains the `toc` command group with both subcommands, the path-type validation, the
`--lang` handling, and a batch driver keyed on `path` rather than `symbol`.

The `--doc-sentences` flag is deliberately **not** in this batch. `toc file` here passes a fixed
`toc.Options{DocSentences: 1}`, the built-in default, so the verb ships complete and correct at its
default setting; batch 7 replaces that one constant with the flag-and-config precedence chain.
Splitting there keeps the command wiring and the configuration precedence reviewable separately.

New CLI code lives in a new file, `internal/cli/toc.go`, rather than in `cli.go` — that file is
already the largest in the tree and toc shares none of its `resolveContext` / `buildOptions`
machinery.

## Cards

### Card 34: facade re-exports

- **Context:**
  - `internal/quarryengine/toc/toc.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/errors.go`
- **Edits:**
  - `quarry/facade.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add the toc surface to the facade, following the file's existing conventions
  exactly — a type alias, a re-exported sentinel var bound to the identical error value, or a one-line
  delegating function, and nothing else.
  Add the type aliases `TOCSymbol = toc.Symbol`, `TOCKind = toc.Kind`, `TOCFileResult = toc.FileTOC`,
  `TOCDirEntry = toc.DirEntry`, `TOCDirResult = toc.DirTOC`, and `TOCOptions = toc.Options`.
  Re-export the three kind constants as `TOCKindFunction`, `TOCKindMethod`, `TOCKindType`, and the
  `AllSentences` sentinel as `TOCAllSentences`, so a facade-only consumer never has to import the
  engine package to name a kind or ask for the whole docstring.
  Add `var ErrLanguageUnsupported = quarryengine.ErrLanguageUnsupported`, alongside the other
  re-exported sentinels.
  Add the two delegating functions:
  `func TOCFile(path string, lang string, opts TOCOptions) (TOCFileResult, error)` and
  `func TOCDir(dir string, lang string) (TOCDirResult, error)`, each a single `return toc....` line.
  Import `github.com/Knatte18/quarry/internal/quarryengine/toc` alongside the existing engine imports.
  Do **not** add behaviour of any kind here — no defaulting of `opts`, no validation, no path
  handling. The facade's whole contract is that it adds nothing.
  The file's own doc comment carries a stale identifier count and a stale package-set enumeration;
  both are rewritten in batch 8 together with every other stale-prose site, so this card changes code
  only.
- **Commit:** `feat(quarry): re-export the toc entry points through the facade`

### Card 35: facade signature and sentinel guards

- **Context:**
  - `quarry/facade.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/toc.go`
- **Edits:**
  - `quarry/facade_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** extend the compile-time guards so the new re-exports are self-checking rather than
  merely present.
  Add one blank-identifier entry per new delegating function to the existing block, each against the
  exact func type its engine counterpart demands:
  `_ func(string, string, TOCOptions) (TOCFileResult, error) = TOCFile` and
  `_ func(string, string) (TOCDirResult, error) = TOCDir`. The comment above that block states a
  count of the assignments it holds; update the count to match what the block now contains.
  Add alias round-trip pairs for the new type aliases in the `aliasCheck...` variable block and the
  matching assignments in `init`, following the existing two-line engine-to-facade-and-back shape, so
  any of them silently becoming a defined type fails the build. The `init` comment states how many
  aliased types it round-trips; update that count too.
  Add `ErrLanguageUnsupported` as a row in `TestFacadeSentinels_Identity`'s table. That test's own doc
  comment states a count of the re-exported sentinels; update it.
  These three counts are the reason this card exists as its own unit: they are prose the compiler
  cannot check, sitting directly above assertions the compiler can, and they are exactly the kind of
  drift the stale-prose sweep in batch 8 is chartered to eliminate.
- **Commit:** `test(quarry): extend the facade guards to the toc re-exports`

### Card 36: the toc command group and `toc file`

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/exec.go`
  - `internal/cli/cwdcontext.go`
  - `internal/output/output.go`
  - `quarry/facade.go`
  - `internal/quarryengine/registry/extension.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/toc.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `internal/cli/toc.go` with a file header comment stating that it holds the
  toc command group and that toc deliberately bypasses `resolveContext` and `buildOptions` — it needs
  no language server and no daemon state directory, so forcing it through the machinery that loads
  the server registry would make it fail on machines where no `servers.yaml` resolves.
  Add `tocCommand() *cobra.Command`, a parent command with `Use: "toc"`, a non-empty `Short`, and
  `RunE: GroupRunE`, so a bare `quarry toc` prints help and an unknown subcommand emits the JSON error
  envelope.
  Add `tocFileCommand() *cobra.Command` as its subcommand: `Use: "file <path>"`,
  `Args: cobra.MinimumNArgs(1)`, a `--lang` string flag, and a `Long` describing the emitted key set
  and the batch shape in the style the existing verbs' `Long` strings use.
  That `Long` must spell out what `start`, `sigend`, and `end` each mean and which pair to read for
  which purpose — `start`–`sigend` to judge relevance from docstring plus signature, `start`–`end` to
  read the whole symbol. The full two-phase discovery flow, which turns on the `--doc-sentences` flag,
  is added to this same `Long` in batch 7, where that flag exists.
  Its `RunE`:
  - resolves the seam cwd with `CwdFrom(ctx)` and, on error, emits `output.Err` and returns nil;
  - validates `--lang` against toc's **own** vocabulary — `registry.ExtensionLanguages()` — and not
    against the server registry. An unrecognised value is an `output.Err` naming the valid set. State
    in a comment why the existing verbs' registry-key validation is not reused: it runs inside
    `resolveContext` against the servers.yaml-loaded registry, which toc never loads;
  - for a single positional argument, joins it against the seam cwd unless it is already absolute,
    `os.Stat`s the result, and emits `output.Err` for a nonexistent path or for a directory — the
    directory message must name the correct subcommand so a mistaken call carries its own fix. Use
    `os.Stat`, not `os.Lstat`, so a symlink's target type is what is validated;
  - calls `quarry.TOCFile(abs, lang, quarry.TOCOptions{DocSentences: 1})` and, on success, emits the
    result through `output.Ok`. Add a `// TODO` free comment naming batch 7 as where that literal `1`
    becomes the resolved flag-and-config value, so the placeholder is not mistaken for the final
    shape;
  - on error, emits `output.Err` with the error's message. Every single-argument toc failure is exit
    1: no exit 2 and no exit 3, keeping toc inside the exit-code contract the package doc already
    documents for every verb.
  Marshal the result by re-marshalling the typed struct into a `map[string]any` via
  `encoding/json`, so `output.Ok`'s `map[string]any` parameter is fed the exact keys the struct tags
  define and the omitempty rules are honoured in one place rather than restated here.
- **Commit:** `feat(cli): add the toc command group and the toc file verb`

### Card 37: `toc dir`

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/cwdcontext.go`
  - `internal/output/output.go`
  - `quarry/facade.go`
  - `internal/quarryengine/registry/extension.go`
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `tocDirCommand() *cobra.Command` as the group's second subcommand:
  `Use: "dir <path>"`, `Args: cobra.MinimumNArgs(1)`, a `--lang` string flag validated the same way,
  and a `Long` describing the emitted key set.
  Its `RunE` mirrors `toc file`'s shape, with three differences:
  - the `os.Stat` type check is inverted — a **file** passed to `toc dir` is the hard error, and its
    message names `toc file` as the fix;
  - it calls `quarry.TOCDir(abs, lang)` with no options argument;
  - each emitted entry's `path` is composed as `filepath.Join(arg, entry.Name)` using the positional
    argument **exactly as the caller wrote it**, never the absolutised form, so
    `quarry toc dir internal/cli` yields entries the caller can paste straight into
    `quarry toc file internal/cli/exec.go` from the same working directory. `DirEntry.Name` is
    `json:"-"` precisely so this composition has to happen here; add the `path` key while building
    each entry's map.
  For `--lang` naming a designed-but-unimplemented language, `toc dir` returns `ok:true` and exit 0
  with every matching file listed carrying its per-file `error` — the flag selects which files to
  list, and an unimplemented language is a reported limitation rather than a failure of the listing.
  That is the opposite of `toc file --lang rust`, where the unsupported language is the whole request
  and there is nothing to list. State both halves in a comment at the branch, because the asymmetry is
  deliberate and reads like a bug otherwise.
- **Commit:** `feat(cli): add the toc dir verb`

### Card 38: the path-keyed batch driver

- **Context:**
  - `internal/cli/cli.go`
  - `internal/output/output.go`
  - `internal/cli/exec.go`
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `runPathBatch(ctx context.Context, out io.Writer, args []string, lookupOne func(path string) (batchStatus, map[string]any))`,
  toc's own batch driver. It is a near-copy of `runBatch` with exactly one line different: the
  per-entry identity key is `"path"` rather than `"symbol"`, and its value is the positional argument
  echoed verbatim, exactly as `runBatch` echoes its own.
  It **reuses** the existing `batchStatus` constants and the `statusRank` map unchanged — the shared
  status vocabulary and ranking are the same, only the key differs. Do not generalize `runBatch`
  itself: that function's `"symbol"` key is part of the output shape all four shipped verbs emit, and
  parameterizing it would change them. Say that in the new function's doc comment, so the duplication
  reads as a deliberate choice rather than an oversight a later cleanup should collapse.
  Wire both subcommands to call `runPathBatch` when `len(args) > 1`, with a per-argument closure that
  performs the same stat-validate-and-call sequence the single-argument path does and maps its outcome
  to a status:
  - parsed successfully, including a `partial: true` entry — `statusFound`;
  - `toc dir` on a directory with no supported files — `statusFound`, with an empty `files`;
  - the path does not exist — `statusNotFound`;
  - the wrong path type for the subcommand — `statusError`;
  - the extension maps to no language, or to a designed-but-unimplemented one — `statusError`;
  - the file is unreadable or not valid UTF-8 — `statusError`.
  `partial: true` is a field on the entry and must **not** degrade the status: ranking it as a failure
  would poison the exit code of any batch containing a single mid-edit file. `ambiguous` is never
  produced, since a path-addressed lookup has no ambiguity state.
- **Commit:** `feat(cli): add the path-keyed toc batch driver`

### Card 39: register the toc group

- **Context:**
  - `internal/cli/toc.go`
  - `internal/cli/exec.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `cmd.AddCommand(tocCommand())` to `Command()`, after the four existing
  `AddCommand` calls, so `quarry toc` appears in the root help listing.
  Change nothing else in `Command()`. The root command's `Short` also needs updating — it enumerates
  quarry's capabilities and would leave `quarry --help` describing a tool without `toc` — but that is
  prose, and it is rewritten in batch 8 alongside every other stale enumeration so the sweep has one
  place to be verified from.
- **Commit:** `feat(cli): register the toc command group on the root command`

### Card 40: CLI toc tests

- **Context:**
  - `internal/cli/toc.go`
  - `internal/cli/cli.go`
  - `internal/cli/cli_test.go`
  - `internal/cli/exec.go`
  - `internal/cli/cwdcontext.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/toc_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** tests driving the new verbs through the `RunCLI` / `RunCLIIn` seam, following the
  offline, spawn-free patterns `cli_test.go` established — no subprocess, no language server, no
  `t.Chdir`. Fixture files are written into a `t.TempDir()` and reached via `RunCLIIn`'s injected seam
  cwd, which is what proves toc honours that seam rather than reading the process working directory.
  Cases:
  - bare `quarry toc` — help is printed and the exit code is 0, the `GroupRunE` contract;
  - `quarry toc badsub` — the JSON error envelope naming the unknown subcommand;
  - `toc file` on a directory — exit 1, and the error message names `toc dir`;
  - `toc dir` on a file — exit 1, and the message names `toc file`;
  - a nonexistent path to either verb — `output.Err` and exit 1;
  - `toc file` on a real Go fixture — `ok:true`, exit 0, and the parsed envelope carrying `language`,
    `package`, `symbols`, and a `header`, with no `partial` key present, and each symbol carrying
    `start`, `sigend`, and `end`;
  - a fixture whose only symbol is a type alias — that symbol's entry has **no** `sigend` key at all,
    asserted against the decoded JSON rather than against the Go struct, since the omission is what
    the consumer sees;
  - the same fixture with a syntax error — `partial: true` present and exit 0;
  - `toc file` on a `.rs` fixture — the explicit "not yet supported" error rather than an empty
    result;
  - `--lang` with an unrecognised value on both verbs — the error names the valid set;
  - `--lang go` on a `.py` fixture — parses with the Go grammar and does **not** error on the
    mismatch;
  - `toc dir --lang rust` on a directory holding `.rs` files — `ok:true`, exit 0, and a per-file
    `error` entry for each, **not** a top-level error;
  - `toc dir` on a directory with no supported files — `files` is `[]` and exit 0;
  - `toc dir` — every entry's `path` is the directory argument as written joined with the filename,
    asserted by passing a relative argument through `RunCLIIn` and checking the emitted prefix is the
    relative form, not the absolute one;
  - an unreadable or invalid-UTF-8 file inside a `toc dir` listing — still listed, with `error`, no
    `header`, no `partial`, and the rest of the directory unaffected;
  - batch mode with two or more arguments on both verbs — the per-entry key is `"path"` and **not**
    `"symbol"`;
  - a batch mixing a found, a not-found and an error argument — the exit code is the worst rank
    present;
  - a batch containing one `partial: true` file — the exit code is still 0.
  Also assert `TestCommand_EveryCommandHasShort` still passes with the new commands present; if the
  group or either subcommand lacks a `Short`, that existing test is what catches it.
- **Commit:** `test(cli): cover the toc verbs, path validation and batch mode`

## Batch Tests

`verify: go test ./internal/quarryengine/toc ./quarry ./internal/cli` covers the three packages this
batch touches. `internal/cli` and `quarry` are where the new code lands; `internal/quarryengine/toc`
re-runs because the facade and CLI are its first external consumers, so a signature that does not
actually compose surfaces here.

New test file: `internal/cli/toc_test.go`. Modified: `quarry/facade_test.go`.

`quarry/facade_test.go` is a compile-time guard as much as a test: most of what card 35 adds fails at
build time rather than at assertion time, which is the intended behaviour.
