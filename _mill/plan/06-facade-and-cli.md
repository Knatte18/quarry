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
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/registry/extension.go`
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
  Add the four delegating functions:
  `func TOCFile(path string, lang string, opts TOCOptions) (TOCFileResult, error)`,
  `func TOCDir(dir string, lang string) (TOCDirResult, error)`, each a single `return toc....` line,
  `func TOCLanguages() []string`, a single `return registry.ExtensionLanguages()` line, and
  `func TOCImplemented() []string`, a single `return toc.Implemented()` line.
  `TOCLanguages` exists so the CLI can validate `--lang` against toc's own vocabulary without
  importing an engine subpackage directly: `internal/cli` today imports **nothing** under
  `internal/quarryengine` — every engine identifier reaches it through this file — and cards 36 and 37
  must not be the first exception. Say that in its doc comment.
  `TOCImplemented` exists for the same reason and covers the other set: `TOCLanguages` returns the five
  **designed** names the extension map knows, `TOCImplemented` the three that have a registered
  strategy. Both are needed and they are not interchangeable — `--lang` validates against the designed
  set, so `toc dir --lang rust` stays a legal request that lists `.rs` files with per-file errors,
  while the unsupported-language message names the implemented set, so it tells the caller what quarry
  can actually read. Say exactly that in both doc comments, since re-exporting two nearly identical
  string slices invites a later cleanup that collapses them into one.
  Add the `github.com/Knatte18/quarry/internal/quarryengine/toc` import. The `registry` import
  `TOCLanguages` needs is already present in this file — do not add it a second time.
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
  `_ func(string, string, TOCOptions) (TOCFileResult, error) = TOCFile`,
  `_ func(string, string) (TOCDirResult, error) = TOCDir`, and
  `_ func() []string = TOCLanguages`, and `_ func() []string = TOCImplemented`. The comment above that
  block states a count of the assignments it holds; update the count to match what the block now contains.
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
  - resolves the seam cwd with `CwdFrom(ctx)` and, on error, emits the error and returns nil;
  **Every error path in this file uses the shape the existing verbs use: `SetExit(ctx, output.Err(out, msg))`,
  then `return nil`.** `output.Err` writes the envelope and *returns* the exit code; calling it alone
  discards that code and leaves the process exiting 0 with an `ok:false` body. Stating "emits
  `output.Err` and exit 1" without the `SetExit` wrapper is exactly how that bug gets written, so the
  wrapper is named here once and applies to every error return below;
  - validates `--lang` against toc's **own** vocabulary — `quarry.TOCLanguages()`, the facade
    re-export card 34 adds — and not against the server registry. Call it through the facade, never
    `registry.ExtensionLanguages()` directly: `internal/cli` imports nothing under
    `internal/quarryengine`, and this file must not become the first exception. An unrecognised value
    is an `output.Err` naming the valid set. State in a comment why the existing verbs' registry-key
    validation is not reused: it runs inside `resolveContext` against the servers.yaml-loaded
    registry, which toc never loads;
  - for a single positional argument, joins it against the seam cwd unless it is already absolute,
    `os.Stat`s the result, and emits `output.Err` for a nonexistent path or for a directory — the
    directory message must name the correct subcommand so a mistaken call carries its own fix. Use
    `os.Stat`, not `os.Lstat`, so a symlink's target type is what is validated;
  - calls `quarry.TOCFile(abs, lang, quarry.TOCOptions{DocSentences: 1})` and, on success, emits the
    result through `output.Ok`. Add a `// TODO` free comment naming batch 7 as where that literal `1`
    becomes the resolved flag-and-config value, so the placeholder is not mistaken for the final
    shape;
  - on error, emits `output.Err` with the message `classifyTOCError` returns. Every single-argument toc
    failure is exit 1: no exit 2 and no exit 3, keeping toc inside the exit-code contract the package
    doc already documents for every verb.
  Add `classifyTOCError(err error) (batchStatus, string)` in this same file — the one place an engine
  toc error becomes a CLI outcome, shared by this command, `toc dir`, and the batch driver in card 38:
  - `errors.Is(err, quarry.ErrLanguageUnsupported)` — returns `statusError` and a **distinct, stable
    message** naming the situation and the implemented language set, built from
    `quarry.TOCImplemented()` rather than a hard-coded list, so the message cannot drift from what
    actually has a strategy.
    It is `TOCImplemented()`, not `TOCLanguages()`: this message answers "what can quarry read", and
    naming the designed set would list `rust` and `typescript` as available in the very error saying
    they are not. This branch is the whole reason the sentinel exists: it is what lets the CLI distinguish
    "quarry cannot read this language" from "quarry failed to read this file", which the wrapped
    engine text alone does not make reliably machine-checkable.
  - anything else — `statusError` and `err.Error()` unchanged.
  The status is `statusError` in both branches on purpose. `found`/`not_found`/`ambiguous`/`error` is
  the closed vocabulary the four existing verbs share, and an unsupported language is an error rather
  than a fifth outcome; the distinction lives in the message. Say that in the helper's doc comment,
  together with the reason the test uses `errors.Is` rather than comparing the message: the message is
  for humans, the sentinel is the contract.
  Marshal the result by re-marshalling the typed struct into a `map[string]any` via
  `encoding/json`, so `output.Ok`'s `map[string]any` parameter is fed the exact keys the struct tags
  define and the omitempty rules are honoured in one place rather than restated here.
- **Commit:** `feat(cli): add the toc command group and the toc file verb`

### Card 37: `toc dir`

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/cwdcontext.go`
  - `internal/cli/exec.go`
  - `internal/output/output.go`
  - `internal/quarryengine/toc/types.go`
  - `quarry/facade.go`
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
    Put the composition in **one shared helper**, `tocDirEntries(arg string, result quarry.TOCDirResult) ([]any, error)`,
    that both the single-argument `RunE` and the batch closure card 38 wires call. The batch path must
    not re-derive it: `runPathBatch` supplies the entry-level `"path"` key, which is the *argument*,
    while every element of `files[]` needs its own per-file `path` — two different keys at two
    different levels, and nothing in `runPathBatch` composes the inner one. A multi-argument
    `quarry toc dir a b` that skipped this helper would emit `files[]` entries with no `path` at all,
    silently and only in batch mode.
    Spell out the correlation step the helper performs, because the re-marshal card 36 mandates
    destroys the base names on the way through: `Name` is `json:"-"`, so after `encoding/json` has turned the result into a
    `map[string]any` the decoded `files` array no longer carries it. Zip that array back to
    `result.Files` **by index** — the marshal preserves slice order, so element `i` of the decoded
    array is `result.Files[i]` — and inject `path` from the typed entry while walking the pair. Assert
    the two lengths match before the loop and fail loudly if they do not, rather than indexing into a
    shorter slice.
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
  to a status.
  `toc dir`'s closure builds its entry's `files` array through card 37's `tocDirEntries` helper,
  passing that argument's own spelling — the same helper the single-argument path uses. This is the
  one thing the batch path cannot inherit from `runPathBatch`: that driver writes the per-entry
  `"path"` key (the argument), not the per-file `path` inside `files[]`, and omitting the helper here
  produces a `files` array with no `path` key in batch mode only. Say that at the call site.
  Statuses:
  - parsed successfully, including a `partial: true` entry — `statusFound`;
  - `toc dir` on a directory with no supported files — `statusFound`, with an empty `files`;
  - the path does not exist — `statusNotFound`;
  - the wrong path type for the subcommand — `statusError`;
  - any error back from the engine — `statusError`, with both the status and the entry's `error`
    message taken from `classifyTOCError` (card 36) rather than re-derived here. That is what routes
    the designed-but-unimplemented and unknown-extension cases through the `errors.Is` branch, so a
    batch entry for a `.rs` file carries the same distinct message the single-argument path emits,
    and the sentinel classification is not silently bypassed in batch mode;
  - the file is unreadable or not valid UTF-8 — `statusError` through the same helper.
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
    result, carrying the distinct message `classifyTOCError`'s `errors.Is` branch produces rather than
    the engine's wrapped text;
  - the same `.rs` argument inside a multi-argument batch — the entry's `error` message is identical
    to the single-argument one, which is what proves the batch driver routes through
    `classifyTOCError` instead of re-deriving the message;
  - a unit test of `classifyTOCError` itself over `fmt.Errorf("...: %w", quarry.ErrLanguageUnsupported)`
    — asserting the sentinel branch is reached **through a wrap**, since every real caller wraps, and
    a `==` comparison rather than `errors.Is` would pass the unwrapped case and fail in production;
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
  - batch mode on `toc dir` with two directory arguments — **every element of each entry's `files`
    array carries its own `path`**, composed from that entry's own argument as written. This is the
    assertion that fails if the batch closure skips `tocDirEntries`, and nothing else in this file
    covers it: the entry-level `"path"` key is present either way;
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
