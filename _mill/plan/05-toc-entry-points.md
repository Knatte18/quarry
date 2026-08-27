# Batch: toc-entry-points

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "toc-entry-points"
number: 5
cards: 6
verify: go test ./internal/quarryengine/toc
depends-on: [4]
```

## Batch Scope

This batch closes the engine half of the task: the two entry points `TOCFile` and `TOCDir`, which
resolve a language, read bytes, drive `treesitter.WithTree`, dispatch to a strategy, and apply the
two post-processing rules that belong to the entry point rather than to any strategy — first-paragraph
header truncation and sentence trimming of every docstring.

Putting both post-processing rules here rather than in the strategies is deliberate: it gives each
rule exactly one call site, keeps `--doc-sentences` from having to be threaded through five
strategies, and makes the header truncation symmetric across the two verbs by construction rather
than by discipline.

The external interface batch 6 consumes: `TOCFile(path string, langOverride string, opts Options) (FileTOC, error)`
and `TOCDir(dir string, langOverride string) (DirTOC, error)`.

## Cards

### Card 28: TOCFile

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/sentences.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/treesitter/treesitter.go`
  - `internal/quarryengine/registry/extension.go`
  - `internal/quarryengine/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/toc.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `TOCFile(path string, langOverride string, opts Options) (FileTOC, error)`.
  Language resolution, in this order: when `langOverride` is non-empty it wins outright and the
  extension is ignored — a mismatch is not an error, matching what `--lang` means on every existing
  verb; otherwise `registry.LanguageForExtension(filepath.Ext(path))`. An extension that maps to no
  language returns a wrapped `quarryengine.ErrLanguageUnsupported`.
  A resolved language with no registered strategy — a designed-but-unimplemented one — also returns a
  wrapped `ErrLanguageUnsupported`, using `StrategyFor` rather than a second hard-coded list.
  Validating `langOverride` against toc's own vocabulary is the CLI's job, not this function's;
  document that, so nobody adds a second validation here that would drift from the flag's error
  message.
  Read the file with `os.ReadFile` and return the wrapped `os` error unchanged on failure — no
  sentinel of its own. Reject content that is not valid UTF-8 with a plain error naming the path; this
  is the "never parsed at all" case, distinct from a lossy parse.
  Parse through `treesitter.WithTree`, and inside the callback build the result:
  `Language` is the resolved language; `Partial` is the callback's `partial` argument;
  `Symbols` is the strategy's `Symbols` output, guaranteed non-nil;
  `Package` is `strategy.Package(root, src)`, left empty when the strategy reports none so the key is
  omitted;
  `Header` is `FirstParagraph(strategy.Header(root, src))` — the same truncation `TOCDir` applies, so
  a package-documentation file with zero symbols does not return its whole contents.
  Then apply the docstring policy to every symbol: when `opts.DocSentences` is `0`, clear `Docstring`
  so the key is omitted; when it is `AllSentences`, leave it; otherwise replace it with
  `FirstSentences(sym.Docstring, opts.DocSentences)`. Never write an empty-but-present docstring —
  clearing the field is what omits the key, and that is the only mechanism used.
  Leave `Start`, `SigEnd` and `End` untouched by the docstring policy: the ranges always cover the
  whole docstring, so a truncated `docstring` field and a read of `start`–`sigend` or `start`–`end`
  stay consistent. That is what makes `DocSentences: 0` a discovery mode rather than a lossy one, and
  it is worth stating in the function's doc comment.
  Copy the result out of the callback before returning, so no `*ts.Node` outlives the tree.
- **Commit:** `feat(toc): add the TOCFile entry point`

### Card 29: TOCDir

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/treesitter/treesitter.go`
  - `internal/quarryengine/registry/extension.go`
  - `internal/quarryengine/errors.go`
- **Edits:**
  - `internal/quarryengine/toc/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `TOCDir(dir string, langOverride string) (DirTOC, error)`.
  Read exactly one directory level with `os.ReadDir` — no recursion, ever. Skip subdirectories.
  For each remaining entry, resolve its language from its extension; skip the file entirely when the
  extension maps to no language, so a Markdown or YAML file never appears. When `langOverride` is
  non-empty,
  restrict the listing to that language's extensions via `registry.ExtensionsForLanguage`, listing
  only matching files rather than reinterpreting the others.
  Sort the surviving entries lexicographically by base filename with an explicit `sort.Slice` before
  emitting, never relying on `os.ReadDir`'s order.
  For each listed file, build a `DirEntry` with `Name` set to the base filename and `Language` set to
  the resolved language, then:
  - a language with no registered strategy sets `Error` to `quarryengine.ErrLanguageUnsupported`'s
    message and leaves `Header` and `Partial` unset. The file is **listed**, not skipped, and it
    counts as a code file for the empty-directory question — a directory holding only unimplemented
    languages returns a non-empty `Files`, not an empty one;
  - a read failure or invalid UTF-8 sets `Error` to that failure's message and leaves `Header` and
    `Partial` unset. It never aborts the directory;
  - otherwise parse through `treesitter.WithTree`, set `Partial` from the callback, set `Package` to
    `strategy.Package(root, src)`, set `Header` to
    `FirstParagraph(strategy.Header(root, src))`, and set `Test` and `Generated` from the strategy's
    `TestFile` and `Generated` methods — assigning a pointer only when the method reports
    `known == true`, and leaving the pointer nil otherwise so the key is omitted.
  `Error` and `Partial` are mutually exclusive by construction: `Partial` is only ever set on the
  route that actually parsed. State that invariant in the function's doc comment.
  A directory containing no file with a supported extension returns a `DirTOC` whose `Files` is an
  empty non-nil slice and a nil error. That is a true answer to "what code is in here", not a failure.
  Impose no file-size cap: parse cost is linear and the runtime enforces its own work budgets, so a
  pathological file surfaces as a slow parse or as `Partial`, never as a special-cased refusal.
  `TOCDir` takes no `Options`: it emits headers, never docstrings, so the doc-sentences policy does
  not apply to it. Say so in the doc comment, since the asymmetry with `TOCFile` is otherwise
  surprising.
- **Commit:** `feat(toc): add the TOCDir entry point`

### Card 30: TOCFile tests

- **Context:**
  - `internal/quarryengine/toc/toc.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/sentences.go`
  - `internal/quarryengine/toc/golang_test.go`
  - `internal/quarryengine/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/toc_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** tests for `TOCFile` over files written into a `t.TempDir()`, since this is the
  first code in the package that touches disk.
  Cases:
  - a Go file with symbols — the full `FileTOC` value, asserting `Language`, `Symbols` ascending by
    `Start`, and `Partial` false;
  - a file whose only content is a multi-line header comment and zero declarations — `Header` is the
    **first paragraph only** and `Symbols` is empty. This is the header-truncation case for
    `toc file`, and the one a package-documentation file exercises in practice;
  - a header with no blank line — returned whole;
  - default options (`DocSentences: 1`) — each symbol's `Docstring` is exactly one sentence;
  - `DocSentences: AllSentences` — the whole docstring;
  - `DocSentences: 0` — `Docstring` is empty on every symbol, so the key would be omitted; assert
    emptiness rather than any placeholder;
  - `DocSentences` larger than the sentence count — the whole docstring and no error;
  - a docstring containing `e.g.` in its first sentence, under the default — not split at `e.g.`;
  - a docstring containing a backtick-quoted dotted identifier — not split inside the backticks;
  - `Start`, `SigEnd` and `End` are identical across `DocSentences` values 0, 1, and `AllSentences`
    for the same fixture — no range shrinks with the emitted text;
  - `Package` matches the fixture's package clause, and is empty for a fixture whose package clause is
    lost under a `Partial` parse;
  - a symbol with a body carries a `SigEnd` satisfying `Start <= SigEnd <= End`, and a bodyless symbol
    carries `SigEnd == 0` — assert the ordering invariant across every symbol of a multi-symbol
    fixture rather than only on one, since it is the property a consumer relies on;
  - a `.md` file — a wrapped `ErrLanguageUnsupported`, matched with `errors.Is`;
  - a `.rs` file — the same wrapped `ErrLanguageUnsupported`, proving a designed-but-unimplemented
    language is not a silent empty result;
  - a `langOverride` of `go` on a `.py` file — parses with the Go grammar and does **not** error on
    the mismatch;
  - a nonexistent path — a wrapped `os` error, matched with `errors.Is(err, os.ErrNotExist)`;
  - a file whose bytes are not valid UTF-8 — an error naming the path, and **not** a `Partial` result;
  - a file with a syntax error that swallows a later declaration — `Partial` true with the surviving
    symbols returned.
- **Commit:** `test(toc): cover TOCFile language resolution and docstring policy`

### Card 31: TOCDir tests

- **Context:**
  - `internal/quarryengine/toc/toc.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/golang_test.go`
  - `internal/quarryengine/errors.go`
- **Edits:**
  - `internal/quarryengine/toc/toc_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** tests for `TOCDir` over directories built in a `t.TempDir()`.
  Cases:
  - a mixed directory holding Go, Python and C# files — one list with per-file language resolution,
    and `Files` in lexicographic order by base filename regardless of creation order;
  - a subdirectory present in the directory — not listed and not recursed into;
  - a non-code file — not listed at all;
  - a directory with no supported file — `Files` is empty and non-nil, and the error is nil;
  - a directory holding only unimplemented-language files — `Files` is **non-empty**, each entry
    carrying `Error` and no `Header`, and the error is nil;
  - a file with a header — `Header` is its first paragraph only;
  - a Go file and a C# file in the same directory — each entry's `Package` is its own package or
    namespace name; a Python file in the same listing — `Package` is empty and the key is omitted;
  - a Go test-suffixed file — `Test` is a pointer to true; a C# file — `Test` is nil, which is the
    omission case and must be asserted as nil rather than as false;
  - a generated Go file — `Generated` is a pointer to true and `Header` is the block **after** the
    banner;
  - an unreadable file (created and then chmod-ed to be unreadable, skipped when the test runs as a
    user for whom that has no effect) — the entry is listed with `Error` set, `Header` and `Partial`
    unset, and every other file in the directory is unaffected;
  - a file whose bytes are not valid UTF-8 — same shape: listed with `Error`, no `Partial`;
  - a file with a syntax error — listed with `Partial` true and `Error` empty, asserting the two are
    never both set;
  - a `langOverride` of `python` on a mixed directory — only the Python files are listed;
  - a `langOverride` of `rust` on a directory holding Rust files — those files are listed with `Error`
    set and the call itself returns no error.
- **Commit:** `test(toc): cover TOCDir listing, ordering and per-file failures`

### Card 32: repository integration test

- **Context:**
  - `internal/quarryengine/toc/toc.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/output/output.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/toc_integration_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** one test running `TOCFile` against a real file in this repository rather than a
  synthetic fixture — `internal/output/output.go`, chosen because it is small, stable, and carries a
  file header plus three well-documented functions.
  Locate the file relative to the test's own source location via `runtime.Caller(0)` and
  `filepath.Join`, exactly as `layering_test.go` and `seam_enforcement_test.go` already do, rather
  than assuming a working directory.
  Assert the whole `FileTOC` shape: `Language` is `go`, `Package` is `output`, `Partial` is false,
  `Header` is the file's first header paragraph, and the symbol list contains the file's three
  exported functions with `KindFunction`, non-empty signatures, non-empty docstrings, `Start` values
  that are ascending and that each point at the symbol's own doc-comment line, and a `SigEnd` on each
  satisfying `Start <= SigEnd <= End`.
  Assert loosely enough to survive an ordinary edit to that file — check the symbol names, kinds, and
  range ordering rather than pinning exact line numbers or exact prose. A test that has to be updated
  every time an unrelated comment is reflowed is a test that gets deleted.
  Do not tag this test; it is hermetic, spawns nothing, and needs no language server, so it belongs in
  the default tier alongside the rest of the package.
- **Commit:** `test(toc): add a repository-file integration test for TOCFile`

### Card 33: strategy coverage guard

- **Context:**
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/toc.go`
  - `internal/quarryengine/registry/extension.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:**
  - `internal/quarryengine/toc/toc_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `TestImplemented_MatchesRegisteredStrategies`, asserting `Implemented()`
  returns exactly `csharp`, `go`, `python` in sorted order, and
  `TestExtensionLanguages_AllHaveGrammars`, asserting every name `registry.ExtensionLanguages()`
  returns is one `treesitter.Supported` reports true for.
  The second assertion is the one that matters: it fails the moment the extension map and the grammar
  set disagree, which is how a sixth language would otherwise get half-added — an extension that
  resolves to a language name the backend cannot parse, surfacing as a confusing runtime error rather
  than a build-time one.
  Also assert that every name in `Implemented()` appears in `registry.ExtensionLanguages()`, so a
  strategy can never be registered under a name no extension resolves to.
- **Commit:** `test(toc): guard the extension, grammar and strategy sets against drift`

## Batch Tests

`verify: go test ./internal/quarryengine/toc` runs the whole toc package, which is where every change
in this batch lands. The three earlier batches' tests re-run alongside, which is the point: the entry
points are the first code to compose the strategies with the shared text rules, so a mismatch between
them surfaces here.

New test files: `internal/quarryengine/toc/toc_test.go`,
`internal/quarryengine/toc/toc_integration_test.go`.

Unlike batches 2–4, this batch's tests touch disk. They do so only through `t.TempDir()`, except the
integration test, which reads one file from this repository and writes nothing.
