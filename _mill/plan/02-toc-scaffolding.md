# Batch: toc-scaffolding

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "toc-scaffolding"
number: 2
cards: 9
verify: go test ./internal/quarryengine ./internal/quarryengine/toc
depends-on: [1]
```

## Batch Scope

This batch creates the `internal/quarryengine/toc` package's language-independent half: the result
types with their exact JSON tags, the `Strategy` interface every language implements, the strategy
lookup, and the five pure text rules the strategies and entry points share — comment-delimiter
stripping, first-paragraph truncation, sentence trimming, directive-block detection, and
filename-based test/generated classification. It also adds the one new engine sentinel and finishes
the guard-test work batch 1 deliberately left half-done.

Every function in this batch is text-in / value-out with no I/O and no tree-sitter node in its
signature except through the `Strategy` interface, which is why the tests here are written first.

The external interface batches 3–5 consume: the `Symbol`, `FileTOC`, `DirEntry`, and `DirTOC` types;
the `Strategy` interface with `Register` / `StrategyFor` / `Implemented`; `StripComment`,
`StripLineComment`, `StripXMLDocTags`, `FirstParagraph`, `FirstSentences`, `IsDirectiveBlock`,
`TestFileByName`, and `GeneratedByBanner`.

## Cards

### Card 8: the ErrLanguageUnsupported sentinel

- **Context:**
  - `internal/quarryengine/doc.go`
- **Edits:**
  - `internal/quarryengine/errors.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `ErrLanguageUnsupported` as a package-level `errors.New` sentinel, following
  the `ErrNoLanguage` shape at the top of the file rather than the data-carrying types below it — toc
  has no per-failure fields worth carrying. Its message is
  `"quarry: language not yet supported by toc"`, which is the exact string the CLI surfaces to the
  user for both a designed-but-unimplemented language and an extension with no language at all.
  Its doc comment must state that it is returned when a resolved extension maps to no language, or to
  a language the toc strategies do not yet implement, and that callers wrap it with
  `fmt.Errorf("...: %w", ErrLanguageUnsupported)` so `errors.Is` still matches after wrapping.
  Do not add a matching `...Sentinel` variable — that suffix exists only for the data-carrying types
  whose `Is` methods compare against it, and `ErrLanguageUnsupported` has no such type.
- **Commit:** `feat(quarryengine): add the ErrLanguageUnsupported sentinel`

### Card 9: toc package doc and result types

- **Context:**
  - `internal/quarryengine/doc.go`
  - `internal/quarryengine/query/refs.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/doc.go`
  - `internal/quarryengine/toc/types.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package toc` with its package doc comment in `doc.go` only. The doc must
  state that this package is the toc orchestration layer — the per-language extraction strategies and
  the `TOCFile` / `TOCDir` entry points — that it imports the engine root, `registry`, and
  `treesitter`, and that it returns typed results and typed errors with no JSON, no exit codes, and
  no cwd resolution of its own. It must also document the sentence-boundary rule and its abbreviation
  list once, here, since that rule is the least obvious thing in the package.
  In `types.go` declare, with the exact JSON tags the Shared Decision "the emitted key set is closed
  and is not re-litigated per batch" fixes:
  - `type Kind string` with the closed constants `KindFunction Kind = "function"`,
    `KindMethod Kind = "method"`, and `KindType Kind = "type"`.
  - `type Symbol struct` with fields `Kind Kind` (`json:"kind"`), `Name string` (`json:"name"`),
    `Owner string` (`json:"owner,omitempty"`), `Signature string` (`json:"signature"`),
    `Docstring string` (`json:"docstring,omitempty"`), `Start int` (`json:"start"`),
    `SigEnd int` (`json:"sigend,omitempty"`), and `End int` (`json:"end"`).
    `SigEnd` is the last line of the signature and is zero — hence omitted — for a symbol with no
    body, such as a Go type alias. Since every real line number is 1-based, zero is unambiguous as
    the absent marker and `omitempty` is the whole mechanism; document that.
  - `type FileTOC struct` with `Header string` (`json:"header,omitempty"`),
    `Language string` (`json:"language"`), `Package string` (`json:"package,omitempty"`),
    `Symbols []Symbol` (`json:"symbols"`), and `Partial bool` (`json:"partial,omitempty"`).
  - `type DirEntry struct` with `Name string` (`json:"-"`), `Language string` (`json:"language"`),
    `Package string` (`json:"package,omitempty"`), `Header string` (`json:"header,omitempty"`),
    `Partial bool` (`json:"partial,omitempty"`),
    `Test *bool` (`json:"test,omitempty"`), `Generated *bool` (`json:"generated,omitempty"`), and
    `Error string` (`json:"error,omitempty"`).
    `Package` is one field per file rather than per symbol: the value is identical for every symbol
    in the file, and repeating it would pay for the same string once per symbol. Document that, and
    document that `Symbol.Name` deliberately stays bare — an agent composes the qualified form from
    `Package`, `Owner`, and `Name` when it needs one, because the other verbs accept only bare names
    and positions.
  - `type DirTOC struct` with `Files []DirEntry` (`json:"files"`).
  - `type Options struct` with a single field `DocSentences int`, documented as: `0` omits the
    `docstring` key from every symbol, a positive N keeps the first N sentences, and the sentinel
    value `AllSentences` keeps the whole docstring. Declare
    `const AllSentences = -1` beside it. The zero value of `Options` therefore means "omit every
    docstring", so every caller must set the field explicitly — say that in the doc comment, since a
    forgotten `Options{}` would otherwise silently drop every docstring.
  `Test` and `Generated` are pointers, not bools, precisely because the contract distinguishes
  "false" from "the language has no rule": a nil pointer omits the key, and a pointer to `false`
  emits `false`. Say that in their doc comments — a later reader will otherwise "simplify" them to
  plain bools and silently turn "cannot tell" into "no".
  `DirEntry.Name` carries the file's base name and is `json:"-"` because `internal/cli` composes the
  emitted `path` from the caller's own spelling of the directory argument; document that too.
  `FileTOC.Symbols` must be non-nil (an empty slice, never nil) whenever the parse succeeded, so the
  emitted `symbols` key is `[]` rather than `null`; same for `DirTOC.Files`.
- **Commit:** `feat(toc): add the toc package result and option types`

### Card 10: the Strategy interface and lookup

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/strategy.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** declare the per-language extraction contract, designed to accommodate all five
  languages even though only three implement it in this task:

  ```go
  type Strategy interface {
  	Language() string
  	Symbols(root *ts.Node, src []byte) []Symbol
  	Header(root *ts.Node, src []byte) string
  	Package(root *ts.Node, src []byte) string
  	Generated(root *ts.Node, src []byte) (generated bool, known bool)
  	TestFile(base string) (isTest bool, known bool)
  }
  ```

  Each method's doc comment states its contract: `Symbols` returns entries in source order ascending
  by `Start`, never descends into a function or method body, returns each symbol's **full** docstring
  (sentence trimming is the entry point's job, not the strategy's), sets `SigEnd` per the
  language-specific derivation or leaves it zero for a bodyless symbol, and returns an empty
  (non-nil) slice when the file has no listable declaration;
  `Package` returns the file's package or namespace name, or `""` for a language with no such concept
  and for a file that has none — never a name derived from the filename or the directory, since the
  field reports what the file itself declares;
  `Header` returns the **untruncated** stripped
  prose of the file's first non-directive comment block, or `""` when absent — first-paragraph
  truncation is likewise applied by the entry points, so both verbs share one truncation call site;
  `Generated` and `TestFile` return `known == false` for a language with no reliable rule, and the
  caller must then omit the key entirely rather than emitting `false`.
  Add `Register(s Strategy)` and `StrategyFor(lang string) (Strategy, bool)` backed by one unexported
  package-level map keyed on the canonical language name. `Register` panics on a duplicate
  registration, which can only be a programming error at package-init time.
  Add `Implemented() []string` returning the sorted registered language names, so the entry points can
  tell "designed but not implemented" from "unknown extension" without a second list to keep in sync.
  Concrete strategies register themselves from their own file's `init`, so the set is derived from
  what actually compiled in rather than from a hand-maintained slice.
  Import the runtime as `ts "github.com/tree-sitter/go-tree-sitter"`.
- **Commit:** `feat(toc): add the per-language extraction Strategy interface`

### Card 11: comment-delimiter stripping and first-paragraph truncation

- **Context:**
  - `internal/quarryengine/toc/types.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/comments.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add the shared text rules, all pure functions over strings:
  - `StripLineComment(text, prefix string) string` — splits `text` on `\n`, trims each line's leading
    whitespace, removes one leading `prefix` (`//` for Go, `///` for C#) when present, trims the
    result, rejoins with `\n`, and trims the whole. A line that is exactly the bare prefix becomes an
    empty line, which is what makes the truncation rule below work uniformly across comment forms.
    An **empty** `prefix` is a supported call: the function then performs the per-line trim, join and
    whole trim with no prefix removal. That is the exact normalisation a Python docstring needs — it
    has no line delimiter to strip but is indented to its `def` — so the Python strategy calls it that
    way rather than reimplementing the rule. Document the empty-prefix case in the doc comment, so it
    is a stated contract rather than an accident of the implementation.
  - `StripComment(text, prefix string) string` — the delimiter-stripping entry point the Go and C#
    strategies call, dispatching on the comment **form**: when `text`'s first non-whitespace
    characters are `/*`, it removes the opening `/*` (and a `/**` variant's extra `*`), the closing
    `*/`, and one leading `*` from each intermediate line, then applies the same per-line trim, join
    and whole trim `StripLineComment` applies; otherwise it delegates to `StripLineComment(text,
    prefix)` unchanged.
    This exists because tree-sitter emits `/* ... */` as a `comment` node exactly like `//`, so a
    block-form Go file header or C# doc comment reaches the strip path with its delimiters intact. A
    prefix-only rule leaves `/*` and `*/` sitting in the emitted `docstring` and `header`. Say that in
    the doc comment.
    Scope limit, also to be stated there: only Go and C# have a block comment form among the
    implemented languages, and only the delimiters are removed — no reflowing, no de-indentation
    beyond the shared per-line trim.
  - `StripXMLDocTags(text string) string` — removes XML doc-comment tags such as `<summary>`,
    `</summary>`, and `<param name="x">` while keeping their text content, then collapses the runs of
    blank lines the removal leaves behind and trims the result. Implement the tag match with a
    package-level compiled `regexp.MustCompile` over `<[^>]*>`, compiled once at package scope rather
    than per call.
  - `FirstParagraph(text string) string` — returns everything before the first empty line in `text`,
    or the whole of `text` when it contains no empty line, trimmed. The doc comment must state that
    this is applied to text that has **already** been delimiter-stripped, never to raw comment source;
    that ordering is the whole reason one rule covers a Go `//` block, a C# `///` block, and a Python
    module docstring without a per-form special case. It must also state that both `TOCFile` and
    `TOCDir` call it on the file header — the truncation is symmetric across the two verbs, and an
    "optimization" that skips it for one of them is a regression, not a simplification.
  This file imports only `regexp` and `strings`.
- **Commit:** `feat(toc): add comment stripping and first-paragraph truncation`

### Card 12: sentence-boundary trimming

- **Context:**
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/types.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/sentences.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `FirstSentences(text string, n int) string`, returning the first `n` sentences
  of an already-delimiter-stripped docstring.
  - `n == AllSentences` returns `text` unchanged.
  - `n <= 0` returns `""` — the caller is responsible for then omitting the key rather than emitting
    an empty string.
  - `n` greater than the number of sentences returns the whole text. This is not an error.
  A sentence ends at a `.`, `!`, or `?` that is followed by whitespace or by end-of-string, **except**
  when that terminator belongs to one of exactly three excluded shapes:
  - a known abbreviation, from this closed list and no other: `e.g.`, `i.e.`, `cf.`, `vs.`, `etc.`,
    `resp.`, `approx.` — matched case-insensitively against the token ending at the candidate
    terminator;
  - a single-letter initial, i.e. a `.` whose preceding token is one letter (`A.`, `B.`);
  - a terminator inside a backtick-quoted span, so an identifier such as a dotted package-qualified
    name written between backticks never ends a sentence. Track backtick state by counting backticks
    while scanning; an unpaired trailing backtick leaves the remainder of the text inside the span,
    which is the safe direction (it under-splits rather than splitting mid-identifier).
  Keep the abbreviation list short and explicit, in one package-level slice, and reference the package
  doc comment for the rule's rationale. `e.g.` and `i.e.` are common in Go doc comments, so without
  the exception the rule splits mid-sentence.
  Preserve the original inter-sentence spacing exactly: return the input's own prefix up to and
  including the nth terminator, then `strings.TrimSpace` the result — never re-join split pieces with
  a synthesized separator, which would rewrite a docstring's newlines into spaces.
  This file imports only `strings` and `unicode`.
- **Commit:** `feat(toc): add sentence-boundary docstring trimming`

### Card 13: directive-block detection

- **Context:**
  - `internal/quarryengine/toc/comments.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/classify.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `IsDirectiveBlock(lang string, startLine int, stripped string) bool`,
  reporting whether an already-delimiter-stripped leading comment block is a directive block and must
  therefore be skipped when looking for the file header. It returns true only when **every** non-empty
  line in `stripped` matches a known directive form for `lang`:
  - `go`: a line beginning with `go:build`, `+build`, `go:generate`, `go:embed`, or `nolint`, or a
    line matching the generated-file banner — the `Code generated ` prefix together with the
    `DO NOT EDIT.` suffix, per the toolchain's own convention.
  - `csharp`: a line containing `<auto-generated`, treated identically to Go's banner.
  - `python`: a shebang, or a PEP 263 coding line matching `coding[:=]`. **The match is stated on the
    already-stripped text, not on the raw source.** Python callers strip with
    `StripLineComment(raw, "#")`, which removes the leading `#`, so a raw `#!/usr/bin/env python`
    arrives here as `!/usr/bin/env python` — the shebang form this function matches is therefore a
    line beginning with `!`, never one beginning with `#!`. Say this explicitly in the doc comment:
    matching `#!` here would make the rule permanently unreachable, because no caller ever passes raw
    text. Both forms are directives only on their original first-or-second physical line, which is
    what `startLine` (the block's 1-based starting line) is for — a shebang-shaped line in the middle
    of a file is prose, not a shebang.
  - any other language: always false. Preprocessor and attribute forms (`#pragma`, `#nullable`, `#!`)
    are not `comment` nodes in the C#, TypeScript, and Rust grammars, so they never reach this rule at
    all; say so in the doc comment so nobody adds dead cases for them.
  A block mixing a directive line with a prose line is **not** a directive block, and the function
  must return false for it. Empty lines inside a block are ignored when deciding.
- **Commit:** `feat(toc): add leading-directive-block detection`

### Card 14: filename and banner classification

- **Context:**
  - `internal/quarryengine/toc/comments.go`
- **Edits:**
  - `internal/quarryengine/toc/classify.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add the two classification helpers the `Generated` and `TestFile` strategy methods
  delegate to, each returning a `known` flag alongside its answer:
  - `TestFileByName(lang, base string) (isTest, known bool)` — `go`: `known` is true and `isTest` is
    the `_test.go` suffix, which the toolchain defines rather than merely conventions.
    `typescript`: `known` is true for a `.test.ts`, `.test.tsx`, `.spec.ts`, or `.spec.tsx` suffix
    check, the jest/vitest defaults.
    `python`: `known` is true for a `test_` prefix or a `_test.py` suffix, pytest's defaults.
    `csharp` and `rust`: `known` is false and `isTest` is false — test-ness lives in attributes or in
    a project file, and a `Tests.cs`-shaped name is style, not a rule.
  - `GeneratedByBanner(lang, leadingComment string) (generated, known bool)` — `go`: `known` is true,
    and `generated` is whether `leadingComment` matches the `Code generated ... DO NOT EDIT.` banner.
    `csharp`: `known` is true, and `generated` is whether `leadingComment` contains `<auto-generated`.
    `python` and `rust`: `known` is false.
  The doc comment on each must state the omission policy explicitly: a caller must never emit `false`
  for a language whose `known` is false, because a `false` test flag on a C# test file is a lie a
  consumer cannot distinguish from a fact. That is the rule most likely to rot into a best-effort
  `false`.
  Note in `GeneratedByBanner`'s doc comment that being skipped as a header (card 13) and being
  consumed as a generated-file marker here are independent readings of the same block — the banner is
  both.
- **Commit:** `feat(toc): add test-file and generated-file classification`

### Card 15: tests for the shared text rules

- **Context:**
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/sentences.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/comments_test.go`
  - `internal/quarryengine/toc/sentences_test.go`
  - `internal/quarryengine/toc/classify_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** table-driven tests, written before the implementations they cover are considered
  done, over every rule in cards 11–14.
  `comments_test.go` covers: `StripLineComment` for a Go `//` block and a C# `///` block, including a
  bare-prefix line collapsing to an empty line, plus the **empty-prefix** call over an indented
  multi-line Python-docstring-shaped string, asserting every line loses its indentation and no `#` or
  `/` is removed.
  `StripComment` for: a `//` block, asserting it matches `StripLineComment`'s output exactly; a Go
  `/* ... */` block whose intermediate lines carry no leading `*`; a `/**` … `*/` block whose
  intermediate lines each start with `*`, asserting both the delimiters and the per-line `*` are
  removed; and a single-line `/* one liner */`. The block-form cases are the ones that fail if the
  dispatch is dropped, and they are why the function exists rather than the strategies calling
  `StripLineComment` directly.
  Also `StripXMLDocTags` for a `<summary>` block and for a
  `<param name="x">` element, asserting the tags are removed and their text kept;
  `FirstParagraph` for a Go `//` block with a bare `//` separator line, a C# `///` block with a bare
  `///` separator line, a Python module docstring with a blank line, and a header with no blank line
  at all — asserting that last one is returned whole. All four cases must assert the same
  post-stripping blank-line rule, which is the point of having one function rather than four.
  `sentences_test.go` covers `FirstSentences` for: the default of one sentence; `AllSentences`; an `n`
  larger than the sentence count, asserting the whole text and no error; `n <= 0` returning `""`; a
  docstring whose first sentence contains `e.g.`, asserting it is **not** split there; the same for
  `i.e.`; a single-letter initial; a backtick-quoted dotted identifier, asserting the `.` inside it
  does not end the sentence; and a multi-line docstring, asserting the original newline between two
  kept sentences survives rather than being collapsed to a space.
  `classify_test.go` covers `IsDirectiveBlock` for: a `go:build` block; a `Code generated` banner; a
  block mixing a `go:generate` line with prose, asserting it is **not** a directive block; a Python
  shebang at line 1, passed in the post-strip form the real caller produces — `!/usr/bin/env python`,
  with no leading `#` — asserting it **is** a directive block, plus the raw `#!/usr/bin/env python`
  form asserting it is **not** matched, which is what pins the stripped-text contract; a
  `!`-prefixed line starting at line 12, asserting it is not treated as a shebang;
  and a C# `<auto-generated` block. It also covers `TestFileByName` and `GeneratedByBanner` across all
  five languages, asserting the *omission* behaviour explicitly — a `Tests.cs`-shaped C# name must
  return `known == false`, and a `_test.go`-suffixed name must return `isTest == true, known == true`.
  Also assert `Register` panics on a duplicate language and that `StrategyFor` reports `false` for an
  unregistered name.
- **Commit:** `test(toc): cover the shared comment, sentence and classification rules`

### Card 16: toc layering rows and the guard-floor raise

- **Context:**
  - `internal/quarryengine/toc/doc.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:**
  - `internal/quarryengine/layering_test.go`
  - `internal/quarryengine/seam_enforcement_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `tocPkg = rootPkg + "/toc"` to `layering_test.go`'s import-path constant block
  and two `layeringTable` rows for `pkgDir: "toc"` — production and test — each allowing
  `pathSet(rootPkg, registryPkg, treesitterPkg)`, matching the package-layout decision that toc
  imports the root, `registry`, and `treesitter` and nothing else in the DAG.
  Raise `minPackageDirs` from 6 to 8 in both `layering_test.go` and `seam_enforcement_test.go`. The
  two constants do **not** mean the same thing, so write each comment to say what its own 8 is:
  `layering_test.go` walks only the engine tree, where 8 is now the *exact* directory count (root,
  lsp, registry, daemon, daemon/daemontest, query, treesitter, toc);
  `seam_enforcement_test.go` walks that same tree plus the facade directory, where the real count is
  now 9 and its existing comment already states the floor is deliberately one below the real count —
  raising it to 8 preserves that intentional slack rather than removing it.
  Both constants are floors (`< minPackageDirs` fails), so leaving them at 6 would keep the build
  green while the guard silently lost strength. That is the reason this card exists at all, and it
  belongs in the comment.
  Update `layering_test.go`'s `layeringTable` doc comment, which enumerates the allowed directions in
  prose, to include the two new packages.
  Update the import-path constant block's own comment as well. Batch 1 card 7 raised its stated count
  to seven when it added `treesitterPkg`; `tocPkg` makes it eight, and this card owns that second
  bump. Both comment updates belong here for the same reason the `minPackageDirs` bump does: this is
  the card that adds the eighth path, so leaving either count to batch 8 would mean shipping a batch
  whose own guard file contradicts itself.
  Make no other change to either guard: do not widen a banned list, do not add an exemption, and do
  not relax a walk.
- **Commit:** `test(quarryengine): add toc layering rows and raise the package-dir floor`

## Batch Tests

`verify: go test ./internal/quarryengine ./internal/quarryengine/toc` covers the engine root — where
the new sentinel and both strengthened guard tests live — and the new `toc` package. `registry` and
`treesitter` are unchanged by this batch, so they are not re-run here; batch 1's verify already
covered them and the module-wide `go vet ./...` catches any cross-package break.

New test files: `internal/quarryengine/toc/comments_test.go`,
`internal/quarryengine/toc/sentences_test.go`, `internal/quarryengine/toc/classify_test.go`.
Modified: `internal/quarryengine/layering_test.go`,
`internal/quarryengine/seam_enforcement_test.go`.

`TestLayeringInvariant_ImportDirections` and `TestEngineSeamInvariant_BannedImports` are both
load-bearing here rather than incidental: this is the batch where their directory floors go from
"passes because nothing changed" to "passes because both new packages exist and are correctly
placed".
