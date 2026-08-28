# Plan: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
slug: "toc-verbs"
approved: false
started: "20260827-182202"
parent: "main"
root: ""
verify: go vet ./...
skip_checks: ["wiki-config-mutation"]
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: treesitter-backend
    file: 01-treesitter-backend.md
    depends-on: []
    verify: go test ./internal/quarryengine ./internal/quarryengine/treesitter ./internal/quarryengine/registry
  - number: 2
    name: toc-scaffolding
    file: 02-toc-scaffolding.md
    depends-on: [1]
    verify: go test ./internal/quarryengine ./internal/quarryengine/toc
  - number: 3
    name: go-strategy
    file: 03-go-strategy.md
    depends-on: [2]
    verify: go test ./internal/quarryengine/toc
  - number: 4
    name: python-csharp-strategies
    file: 04-python-csharp-strategies.md
    depends-on: [3]
    verify: go test ./internal/quarryengine/toc
  - number: 5
    name: toc-entry-points
    file: 05-toc-entry-points.md
    depends-on: [4]
    verify: go test ./internal/quarryengine/toc
  - number: 6
    name: facade-and-cli
    file: 06-facade-and-cli.md
    depends-on: [5]
    verify: go test ./internal/quarryengine/toc ./quarry ./internal/cli
  - number: 7
    name: doc-sentences-config
    file: 07-doc-sentences-config.md
    depends-on: [6]
    verify: go test ./internal/quarryengine/toc ./internal/cli
  - number: 8
    name: docs-and-sweep
    file: 08-docs-and-sweep.md
    depends-on: [7]
    verify: go test ./internal/quarryengine ./quarry ./internal/cli
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: tree-sitter dependency set and pinned versions

- **Decision:** the parsing backend is `github.com/tree-sitter/go-tree-sitter` v0.25.0 plus one grammar
  module per language, at the exact versions already resolved and compiled against in this tree's
  cgo spike: `tree-sitter-go` v0.25.0, `tree-sitter-python` v0.25.0, `tree-sitter-c-sharp` v0.23.5,
  `tree-sitter-typescript` v0.23.2, `tree-sitter-rust` v0.24.2, with `github.com/mattn/go-pointer`
  v0.0.1 as the one indirect dependency the bindings pull in.
  All six modules plus the indirect one are already present in the local module cache, so no network
  fetch is required to build.
  `CGO_ENABLED=1` and a C toolchain become build dependencies of quarry, deliberately, and are never
  runtime dependencies of the built binary.
- **Rationale:** these are the exact versions the discussion's measurement work confirmed compile and
  expose the API the extraction rules use; re-resolving them at `@latest` would silently change the
  node shapes every strategy is written against.
- **Applies to:** all batches

### Decision: the confirmed go-tree-sitter v0.25.0 API surface

- **Decision:** the API every batch codes against, confirmed by reading the module in the local
  module cache and by compiling a dumper against it, rather than assumed:
  `ts.NewLanguage(ptr unsafe.Pointer) *Language`, where `ptr` comes from the grammar module's
  `bindings/go` package — `tsgo.Language()`, `tspython.Language()`, `tscsharp.Language()`,
  `tsrust.Language()`, and — TypeScript only — `tstypescript.LanguageTypescript()` rather than a bare
  `Language()`, because that module ships both TypeScript and TSX;
  `ts.NewParser() *Parser`, `(*Parser).SetLanguage(*Language) error`,
  `(*Parser).Parse(text []byte, oldTree *Tree) *Tree`, `(*Parser).Close()`;
  `(*Tree).RootNode() *Node`, `(*Tree).Close()`;
  `(*Node).Kind() string`, `.Child(i uint) *Node`, `.ChildCount() uint`,
  `.NamedChild(i uint) *Node`, `.NamedChildCount() uint`, `.ChildByFieldName(name string) *Node`,
  `.FieldNameForChild(childIndex uint32) string`,
  `.PrevSibling() *Node`, `.NextSibling() *Node`, `.Parent() *Node`,
  `.StartByte() uint`, `.EndByte() uint`, `.HasError() bool`,
  `.Utf8Text(source []byte) string`,
  and — **`StartPosition() Point` / `EndPosition() Point`, NOT `StartPoint()` / `EndPoint()`** —
  the row/column accessors.
  `ts.Point` has fields `Row uint` and `Column uint`, both **0-based**, so an emitted 1-based line
  number is always `int(node.StartPosition().Row) + 1`.
  Child indices are `uint`, so an index loop is `for i := uint(0); i < n.ChildCount(); i++`.
- **Rationale:** the pre-existing spike at `.scratch/tsspike/` was written against a different
  library (`github.com/odvcencio/gotreesitter`) whose node accessors are named `StartPoint()`/
  `EndPoint()` and whose `Type()` takes a language argument. Copying that spike's method names into
  the real backend would not compile. Nothing under `.scratch/` is a source of API truth for this
  task.
- **Applies to:** all batches

### Decision: every parse releases both C allocations

- **Decision:** `*Parser` and `*Tree` each own C memory and each expose `Close()`. No code outside
  `internal/quarryengine/treesitter` ever constructs either one; the whole engine parses only through
  that package's `WithTree` helper, which closes both in `defer`s on every route — success, extraction
  error, and the `partial` route alike.
- **Rationale:** a leaked tree is a C allocation the Go GC never reclaims, and `toc dir` over a large
  directory is exactly where the leak compounds. Making construction unreachable from outside the
  backend package is what makes "released on every route" a structural property rather than a
  convention every call site must remember.
- **Applies to:** treesitter-backend, toc-scaffolding, go-strategy, python-csharp-strategies,
  toc-entry-points

### Decision: engine returns typed results; the CLI owns every path and envelope concern

- **Decision:** `internal/quarryengine/toc` returns typed Go structs and typed errors only. It never
  imports `internal/output`, cobra, or `internal/cli`, never resolves a cwd, never formats JSON, and
  never decides an exit code.
  Path existence checks, path-type validation, the seam-cwd join, the caller-relative `path` strings
  in `toc dir` output, the `.quarry.yaml` / `$QUARRY_TOC_CONFIG` precedence, and the batch driver all
  live in `internal/cli`.
  The engine's per-directory entry carries the file's **base name** only; `internal/cli` composes the
  emitted `path` as `filepath.Join(<directory argument exactly as the caller wrote it>, <base name>)`.
- **Rationale:** this is the seam `internal/quarryengine/seam_enforcement_test.go` enforces, and the
  emitted paths only round-trip into a follow-up `quarry toc file` call if they are expressed in the
  caller's own frame of reference — a frame the engine deliberately does not know.
- **Applies to:** toc-scaffolding, toc-entry-points, facade-and-cli, doc-sentences-config

### Decision: exactly one new engine sentinel

- **Decision:** `ErrLanguageUnsupported` is added to `internal/quarryengine/errors.go` as a plain
  `errors.New` sentinel (the `ErrNoLanguage` shape, not the data-carrying `Is`-implementing shape),
  re-exported through `quarry/facade.go`, and classified in `internal/cli` with `errors.Is`.
  Every other toc failure mode is either a CLI-side `os.Stat` concern, a CLI-side flag/config
  validation error, or a wrapped `os` error with no sentinel of its own.
- **Rationale:** "the extension maps to no language, or to a language designed but not yet
  implemented" is the one failure the CLI must distinguish structurally; everything else it can report
  verbatim.
- **Applies to:** toc-scaffolding, toc-entry-points, facade-and-cli

### Decision: the emitted key set is closed and is not re-litigated per batch

- **Decision:** `toc file` emits `header` (omitted when absent), `language`, `package` (omitted when
  the file has none), `symbols`, and `partial` (omitted when false). Each `symbols` entry emits
  `kind`, `name`, `owner` (omitted when absent), `signature`, `docstring` (omitted when absent),
  `start`, `sigend` (omitted when the symbol has no body), and `end`.
  `toc dir` emits `files`; each entry emits `path`, `language`, `package` (same omission rule),
  `header` (omitted when absent), `partial` (omitted when false), `test` (omitted when the language
  has no rule), `generated` (same omission rule), and `error` (present only on a per-file failure,
  mutually exclusive with both `header` and `partial`).
  `kind` is the closed three-value vocabulary `function`, `method`, `type`.
  Every omission is implemented with a Go `json:",omitempty"` struct tag on the corresponding field,
  never by hand-building a map. `docstring: ""` is never emitted, including under
  `--doc-sentences 0` — absence is always signalled by omitting the key.
- **Rationale:** an omitted key lets a consumer distinguish "no docstring" from "empty docstring", and
  "cannot tell whether this is a test" from "not a test", without guessing. Fixing the key set once
  here keeps five separate strategies from inventing per-language variants.
- **Applies to:** toc-scaffolding, go-strategy, python-csharp-strategies, toc-entry-points,
  facade-and-cli, doc-sentences-config

### Decision: header truncation is symmetric across both verbs

- **Decision:** a file's `header` is truncated to its first paragraph in **both** `toc file` and
  `toc dir`, by the identical rule: strip the comment or string delimiters first, then cut at the
  first blank line in the resulting prose. A header with no blank line is emitted whole.
  Both entry points apply the truncation themselves, so the rule has one implementation
  (`FirstParagraph`) and one call shape.
- **Rationale:** the truncation rule was originally specified for `toc dir` only, and the asymmetry
  was unintended. A Go `doc.go` holds package documentation and zero symbols by convention, so without
  truncation `toc file` returns the whole file: measured on a 1240-line `doc.go` (1239 of them
  comment) in a real 1059-file repository, the untruncated result was 97.5 KB against 0.6 KB
  truncated — a 164x difference for a file where the agent loses nothing, since the header is by
  definition everything before the first symbol's `start` and a full read is one call away.
- **Applies to:** toc-scaffolding, toc-entry-points, facade-and-cli

### Decision: docstrings are trimmed to their first sentence by default

- **Decision:** `toc file` takes a `--doc-sentences <N|all>` flag controlling how much of each
  docstring reaches `symbols[].docstring`, defaulting to `1`.
  `0` omits the `docstring` key entirely from every symbol; `N` (N ≥ 1) keeps the first N sentences;
  `all` keeps the docstring unchanged. An N larger than the sentence count returns the whole
  docstring and is not an error. A negative number, or any non-numeric value other than `all`, is an
  `output.Err` naming the valid forms.
  The sentence boundary is an explicit rule, never a naive split on `.`: split at `.`, `!`, or `?`
  followed by whitespace or end-of-string, **except** when the terminator belongs to a short, closed
  abbreviation list (`e.g.`, `i.e.`, `cf.`, `vs.`, `etc.`, `resp.`, `approx.`), to a single-letter
  initial, or to an identifier inside backticks.
  The rule and the abbreviation list are documented in the toc package doc comment.
- **Rationale:** the verb exists to show a symbol's *short* docstring, and Go's own convention is
  that the first sentence is the summary — it is what godoc shows in an overview. Measured over 400
  files in a real repository (4.7 MB of source), a full-docstring index was 1.86 MB — 39% of the
  source bytes, a 2.6x reduction; the first-sentence default was 1.39 MB, 29% and 3.4x; and
  `--doc-sentences 0` was 0.75 MB, 16% and 6.3x.
  Those measurements come from a repository written under looser docstring conventions with unusually
  long docstrings, so 39% is a worst case rather than the norm: the default of 1 is chosen because it
  matches the task's own wording and Go convention, **not** because it was tuned on that repository's
  bloat. Do not tune the default further on that data.
  `e.g.` and `i.e.` are common in Go doc comments, so without the abbreviation exception the rule
  splits mid-sentence — which is why the list is explicit rather than a bare regex.
- **Applies to:** toc-scaffolding, toc-entry-points, facade-and-cli, doc-sentences-config

### Decision: per-directory configuration of the doc-sentences default

- **Decision:** the effective `doc-sentences` value is resolved highest-precedence-first:
  1. `--doc-sentences` on the command line — governs that one call;
  2. `$QUARRY_TOC_CONFIG` — an absolute path to a config file, set per shell or per directory;
  3. `.quarry.yaml` in the **target directory itself** — for `toc file`, the file's parent
     directory. `toc dir` is deliberately not part of this chain: it emits headers only, never
     docstrings, so the setting has nothing to affect there and `--doc-sentences` is registered on
     `toc file` alone;
  4. the built-in default, `1`.
  The file is YAML, decoded with `KnownFields(true)` exactly as `LoadRegistry` already does, so a
  misspelled key is a loud error rather than a silent no-op. Its shape is a `toc` mapping with a
  `doc_sentences` key holding an integer ≥ 0 or the string `all`. An absent file is not an error; a
  present but invalid file is.
  **The lookup happens in the target directory and nowhere else.** No walking up the directory tree,
  no repository-root search, no project detection.
  **This file is not `servers.yaml`.** The registry's loader is not reused and `servers.yaml` gains no
  `toc` section.
- **Rationale:** this mirrors the precedence quarry already uses for `servers.yaml`
  (`--config` → `$QUARRY_CONFIG` → the user config directory), with its own file and its own
  environment variable. An upward search would introduce exactly the repository-root concept the
  design deliberately does not have — toc never detects a project the way the LSP verbs do — whereas a
  target-directory-only lookup gives per-directory behaviour without contradicting that. And putting
  toc settings in `servers.yaml` would reintroduce the coupling to the language-server registry that
  toc exists without: toc needs no language server, so it must never load that registry.
- **Applies to:** doc-sentences-config

### Decision: ordering is part of the contract

- **Decision:** `symbols` is in source order, ascending by `start`. `files` is sorted
  lexicographically by base filename with an explicit `sort.Slice`, never left in the order
  `os.ReadDir` happened to return.
- **Rationale:** directory-entry order is not stable across filesystems, and an agent diffing two toc
  runs must not see spurious reordering.
- **Applies to:** go-strategy, python-csharp-strategies, toc-entry-points

### Decision: signatures are verbatim source, cut at the body-bearing child

- **Decision:** a signature is the source text from the declaration's first byte to the start byte of
  that declaration's body-bearing child, `strings.TrimSpace`d — with no normalization, no
  reformatting, and **never** a first-line truncation. When a declaration has no body-bearing child,
  the signature is the whole declaration's text.
  Per-kind body-bearing child resolution is stated in each strategy's own batch.
- **Rationale:** no transformation can introduce drift, and an LLM reads native source syntax at least
  as well as an invented normal form. A first-line cut would silently truncate every multi-line
  signature, which is the shape the verb most needs to show whole.
- **Applies to:** go-strategy, python-csharp-strategies

### Decision: docstrings keep the prose and drop the syntax

- **Decision:** a docstring's emitted text has its language delimiters stripped — Go's `//` prefix per
  line, Python's string-literal quotes (via the grammar's own `string_content` node), C#'s `///`
  prefix plus its XML doc tags, keeping the tags' text content. Each stripped line is then trimmed and
  the lines joined with `\n`, and the whole result is trimmed. Sentence trimming, when requested,
  happens after stripping, never before.
- **Rationale:** this does not contradict the verbatim-signature rule. A signature is code, where
  reformatting introduces semantic drift; a docstring is prose for reading, and some delimiter
  stripping is unavoidable for every language just to obtain "the text" at all.
- **Applies to:** toc-scaffolding, go-strategy, python-csharp-strategies

### Decision: ranges are 1-based, inclusive, and cover the docstring

- **Decision:** `start` is the first line of the docstring when the docstring is a **sibling** of the
  declaration (Go, C#, TypeScript, Rust) and the first line of the declaration otherwise; `end` is
  always the last line of the declaration. Both are 1-based and inclusive.
  Python needs no `start` adjustment: its docstring is the first statement **inside** the definition's
  body, so it already falls within the declaration node's own span.
  Neither range changes with `--doc-sentences`: `start` always marks the whole docstring's first
  line, so a truncated `docstring` field and a full-text read of `start`–`sigend` or `start`–`end`
  stay consistent. That is what makes `--doc-sentences 0` a discovery mode rather than a lossy one.
- **Rationale:** an agent that judges an entry relevant reads exactly `start`–`end` and gets the prose
  and the code together, in one read, with no second guess. Shrinking the range alongside the emitted
  text would defeat that.
- **Applies to:** go-strategy, python-csharp-strategies, toc-entry-points

### Decision: `sigend` — the last line of the signature

- **Decision:** every symbol carries a third line number, `sigend`, alongside `start` and `end`: the
  last line of its signature, 1-based and inclusive like the other two. That gives an agent two
  ranges instead of one — `start`–`sigend` reads the docstring and the signature alone, enough to
  judge relevance; `start`–`end` reads the whole symbol.
  `sigend` is **semantic, not a raw node coordinate**, and its derivation is per language:
  - Go (function, method, and any type with a body): the line the `{` sits on, i.e. the body node's
    start line.
  - C#, block-bodied member or type: the same rule — the line the `{` sits on.
  - Python: the line the `def ...:` or `class ...:` header ends on, i.e. the body `block`'s start
    line minus 1.
  - C#, expression-bodied (`=>`) member: the line before the `arrow_expression_clause` starts.
  The last two subtract a line, so both must be clamped to never fall below the declaration's own
  first line: a single-line `def f(): return 1` or `void F() => 1;` puts the body on the declaration's
  own line, and an unclamped subtraction would emit a `sigend` above `start`. Implement the clamp
  once, in the shared helper, not per strategy.
  `sigend` is **omitted** for a symbol with no body at all — a Go type alias such as `type ID string`,
  or a bodyless C# declaration.
- **Rationale:** `start`–`end` spans the docstring, the signature, and the entire body, so there was
  no range that yielded prose plus signature alone — no way to judge a symbol's relevance without
  reading its implementation.
  Using "where the body begins" directly does not work across languages: in Go the `{` sits on the
  signature's last line, while Python's `block` starts on the line *after* the `def`, so one shared
  rule would leak a line of implementation into every Python signature range. That is why the
  derivation is specified per language rather than left to a single node accessor.
  **Known imprecision, not a defect:** in a single-line Go function (`func f() int { return 1 }`) the
  signature and the body share a line, so `start`–`sigend` includes the body. No line-based range can
  separate them. Document this in the verb's help text; do not introduce column numbers to solve it.
  A fourth field marking where the signature *begins* was considered and rejected: it would have
  yielded "body only" as `sig`–`end`, but measured over 2690 symbols in a real repository the
  docstring is only 13% of a symbol's lines (median 13%), so skipping it saves 13% on a re-read
  against the cost of another field per symbol and a sentinel value in the contract.
- **Applies to:** toc-scaffolding, go-strategy, python-csharp-strategies, toc-entry-points,
  facade-and-cli

### Decision: `package` is one field per file, never per symbol

- **Decision:** `toc file` emits a top-level `package` field, and each `toc dir` entry carries the
  same field. It is resolved per language by a `Package` method on `Strategy`: Go reads the
  `package_clause`'s identifier; C# reads the namespace name from a
  `file_scoped_namespace_declaration` or the outermost `namespace_declaration`; Python has no package
  clause and returns the empty string. An empty result omits the key.
  `Symbol.Name` stays the bare identifier and gains no qualification.
- **Rationale:** without it an agent receives a symbol list with no indication of which package the
  symbols belong to. One field per file rather than per symbol, because the value is identical for
  every symbol in the file and repeating it N times pays for the same string N times.
  Keeping `name` bare remains correct: `refs`, `definition`, and `symbol` take a bare name or a
  `file:line:col` position, so a qualified name pasted into another quarry call would not be
  understood. An agent holding `package`, `owner`, and `name` can compose the qualified form itself
  when it needs one.
- **Applies to:** toc-scaffolding, go-strategy, python-csharp-strategies, toc-entry-points,
  facade-and-cli

### Decision: the two-phase read flow is documented in the verb's help text

- **Decision:** `toc file`'s cobra `Long` text documents the discovery flow the verb exists to enable,
  concretely and with its measured cost:
  1. `quarry toc file --doc-sentences 0 <path>` — the map. Measured on a 1186-line file holding 40
     functions and methods, this was 8.4 KB.
  2. read `start`–`sigend` for the few candidates that look relevant — about 6 lines each — **directly
     from the source file**.
  3. read `start`–`end` for the one that matched — about 16 lines.
  Against reading the whole 1186-line file.
  The decisive point the help text must make: the prose is read **from the source file**, not from
  quarry's rendering of it. The agent never has to trust quarry's `//` and `///` stripping, its C# XML
  tag removal, or its sentence splitting — it sees the actual bytes. `--doc-sentences 0` is therefore
  the recommended *discovery* mode, not a frugality mode.
  The help text also states the single-line-function imprecision from the `sigend` decision above.
- **Rationale:** without this in the help text, no agent finds the cheap path — the verb ships with
  its most valuable usage undiscoverable.
- **Applies to:** doc-sentences-config, docs-and-sweep

  The flow lands in batch 7 card 44, not in batch 6: card 36 writes the rest of `toc file`'s `Long`
  but explicitly defers the two-phase flow, because every step of it names `--doc-sentences`, and that
  flag does not exist until batch 7.

### Decision: container-reachable, never body-reachable

- **Decision:** the symbol walk descends through *container* nodes — Python `class_definition` →
  `block`, C# `namespace_declaration` / `file_scoped_namespace_declaration` → type declaration →
  `declaration_list` — and lists what it finds there. It never descends into a function or method
  body, so a closure or helper declared inside a body is never listed.
- **Rationale:** "top level" is a Go-shaped notion; taken literally it returns zero methods for two of
  the three implemented languages. Containers are namespacing, bodies are implementation.
- **Applies to:** go-strategy, python-csharp-strategies

### Decision: never fail the directory; flag lossiness instead of hiding it

- **Decision:** `partial: true` means "parsed, lossily" and is set whenever the parse tree reports
  `HasError()`. `error` means "never parsed at all" — an unreadable file, invalid UTF-8, or a
  designed-but-unimplemented language. The two are mutually exclusive by construction.
  Inside `toc dir` neither one aborts the listing; every file is still listed, and the directory's own
  outcome stays a success. Inside `toc file`, where that file is the whole request, the same failure
  is an `output.Err` and exit 1.
  A directory containing no file with a supported extension returns `{"ok":true,"files":[]}` and exit
  0.
- **Rationale:** tree-sitter recovery is lossy rather than merely incomplete — a broken declaration can
  swallow later valid ones — so `partial` is the only thing separating "this file has two symbols"
  from "we lost some". And a file the listing cannot read is information the agent needs, not
  something to hide by dropping the row.
- **Applies to:** toc-entry-points, facade-and-cli

### Decision: five languages designed, three implemented

- **Decision:** the extraction-strategy interface, the extension map, and the survey document cover
  Go, Python, C#, TypeScript, and Rust. Only Go, Python, and C# ship concrete strategies in this task.
  A `.ts`, `.tsx`, or `.rs` file resolves to a real language name and then to
  `ErrLanguageUnsupported`, never to a silent empty result.
- **Rationale:** the real per-language work is the docstring-association rule, not the grammar, and
  three structurally different docstring placements are what proves the interface generalizes.
- **Applies to:** all batches

### Decision: guard tests are strengthened, never relaxed

- **Decision:** `internal/quarryengine/layering_test.go` gains one production row and one test row for
  each of the two new packages, and `minPackageDirs` is raised from 6 to 8 in **both**
  `layering_test.go` and `seam_enforcement_test.go`. No other change is made to either guard.
- **Rationale:** both constants are floors (`< minPackageDirs` fails), so leaving them at 6 would keep
  passing while the guard silently loses strength — which is the failure mode worth defending against
  here, not a red build.
- **Applies to:** treesitter-backend, toc-scaffolding

### Decision: comment and documentation conventions

- **Decision:** every new file opens with a file header comment stating that file's purpose, and every
  new package's doc comment lives in exactly one file of that package. Exported identifiers carry
  godoc comments naming the identifier first. The implementer loads the `mill:golang-comments` and
  `mill:code-comments` skills before writing comments, and `mill:markdown` before writing any `.md`
  file.
- **Rationale:** this tree's existing files follow that shape uniformly, and `toc dir`'s own header
  rule assumes it — the new files are among the first things a reader will point `quarry toc dir` at.
- **Applies to:** all batches

## All Files Touched

_Full union of every `Creates:` / `Edits:` / `Moves:` **target** path across every batch, sorted alphabetically (Move **source** paths are excluded — they disappear, like `Deletes:` tokens)._

- `README.md`
- `docs/toc-docstring-association.md`
- `go.mod`
- `go.sum`
- `internal/cli/cli.go`
- `internal/cli/paths.go`
- `internal/cli/toc.go`
- `internal/cli/toc_test.go`
- `internal/cli/tocconfig.go`
- `internal/cli/tocconfig_test.go`
- `internal/quarryengine/cgoguard.go`
- `internal/quarryengine/cgoguard_nocgo.go`
- `internal/quarryengine/cgoguard_test.go`
- `internal/quarryengine/doc.go`
- `internal/quarryengine/errors.go`
- `internal/quarryengine/layering_test.go`
- `internal/quarryengine/registry/extension.go`
- `internal/quarryengine/registry/extension_test.go`
- `internal/quarryengine/seam_enforcement_test.go`
- `internal/quarryengine/toc/classify.go`
- `internal/quarryengine/toc/classify_test.go`
- `internal/quarryengine/toc/comments.go`
- `internal/quarryengine/toc/comments_test.go`
- `internal/quarryengine/toc/csharp.go`
- `internal/quarryengine/toc/csharp_test.go`
- `internal/quarryengine/toc/doc.go`
- `internal/quarryengine/toc/golang.go`
- `internal/quarryengine/toc/golang_test.go`
- `internal/quarryengine/toc/nodes.go`
- `internal/quarryengine/toc/python.go`
- `internal/quarryengine/toc/python_test.go`
- `internal/quarryengine/toc/sentences.go`
- `internal/quarryengine/toc/sentences_test.go`
- `internal/quarryengine/toc/strategy.go`
- `internal/quarryengine/toc/toc.go`
- `internal/quarryengine/toc/toc_integration_test.go`
- `internal/quarryengine/toc/toc_test.go`
- `internal/quarryengine/toc/types.go`
- `internal/quarryengine/treesitter/treesitter.go`
- `internal/quarryengine/treesitter/treesitter_test.go`
- `mill-config.yaml`
- `quarry/facade.go`
- `quarry/facade_test.go`
