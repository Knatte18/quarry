# Discussion: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
task: Add file/dir toc verbs (Tree-sitter-backed)
slug: toc-verbs
status: discussing
parent: main
```

## Problem

quarry is a navigation tool for LLM coding agents: the point is to let an agent find the right file
and the right part of that file without reading whole files for small things.
Today's four verbs — `refs`, `definition`, `symbol`, `assert-no-callers` — all answer "where is X"
once you already know the symbol name.
None of them answers "what is in this file" or "what is in this directory, and is any of it worth
opening at all".
An agent that does not yet know the symbol name has no cheaper option than reading the files.

Two new verbs close that gap.
`toc file <path>` lists every function, method, and type declaration in a file with its signature,
its docstring, and the line range covering the docstring **and** the declaration as one unit — so an
agent that judges an entry relevant knows exactly which lines to read, with no second guess and no
re-scan.
`toc dir <path>` lists every code file in a directory with that file's header comment, so an agent
can rule a file out without opening it.

The premise, stated by the operator during discussion and binding on the whole design: *a method's
signature plus its docstring must be enough to tell a reader what the method is for.*
The docstring is not leading noise to be skipped over — it is half the answer, and the line range
must cover it.

**Why now:** quarry's four existing verbs are 100% LSP-backed, and `textDocument/documentSymbol`
does not reliably give docstring-to-declaration association or a "declaration + docstring" range
uniformly across the five supported languages.
This task introduces Tree-sitter as a second parsing backend alongside the LSP client — not a
replacement for it — which is the piece the toc verbs need and the piece quarry does not have.

## Scope

**In:**

- A new `toc` cobra command group with two subcommands, `toc file <path>` and `toc dir <path>`,
  wired into `internal/cli`'s existing command tree via the already-present `GroupRunE`
  (`internal/cli/exec.go:227`).
- A new Tree-sitter parsing backend package, `internal/quarryengine/treesitter`, using the pure-Go
  runtime `github.com/odvcencio/gotreesitter`.
- A new orchestration package, `internal/quarryengine/toc`, holding the per-language extraction
  strategies and the two entry points.
- An extension-to-language map in `internal/quarryengine/registry`, since both new verbs are
  path-scoped and the existing `DetectLanguage` is directory-marker-scoped.
- Per-language extraction strategies **designed for all five languages** (Go, Python, C#,
  TypeScript, Rust) and **implemented and tested for Go, Python, and C#** in this task.
- Facade re-exports `quarry.TOCFile` and `quarry.TOCDir` in `quarry/facade.go`.
- New rows in `internal/quarryengine/layering_test.go`'s `layeringTable` for both new packages.
- README verb-list update.
- A per-language docstring-association survey doc under `docs/`, in the spirit of
  `docs/scout-multilang.md`.

**Out:**

- TypeScript and Rust *implementation*. Their strategies are designed here and the interface must
  accommodate them, but the concrete strategies and their tests are a separate follow-up task.
  See the "Language scope" Decision for exactly what "designed but not implemented" means.
- Any change to `refs`, `definition`, `symbol`, or `assert-no-callers`. The LSP path is untouched.
- Replacing the LSP backend. Tree-sitter is added alongside it; no existing verb switches backends.
- Recursive directory walking. `toc dir` reads exactly one directory level.
- A non-JSON output format. Both verbs print the existing `output.Ok` / `output.Err` envelope,
  same as the four existing verbs.
- Constants, variables, and struct/class *fields*. Only functions, methods, and type declarations
  are listed. See the "Symbol kinds" Decision.
- Function bodies, and any symbol declared inside a function body (local closures, nested helpers).
- The `impact` verb. This task only records the shared "declaration + docstring is one range" rule
  that the `impact-verb` task must also honour; it does not implement `impact`.
- Caching or a daemon for the parser. Parses are cheap enough to do per call (see "Technical
  context" for the measurement).

## Decisions

### Parsing backend: pure-Go Tree-sitter runtime

- Decision: use `github.com/odvcencio/gotreesitter`, a pure-Go reimplementation of the Tree-sitter
  runtime that reads the same parse-table format as upstream and ships the grammars as embedded
  blobs.
  No cgo, no C toolchain, no external grammar build step.
- Rationale: README pins quarry to linux **and** windows, and `go.mod` is 100% pure Go today
  (cobra, yaml.v3, flock). The official cgo bindings would force `CGO_ENABLED=1`, require MinGW on
  windows, break cross-compilation, and break `go build -o quarry ./cmd/quarry` for any user
  without a C toolchain. The performance argument that would justify that cost does not exist here:
  the C runtime is ~3.9x faster on a full parse, but a full parse of a 38.8 KB file measured at
  113 ms *including process startup* in the spike, against the tens-to-hundreds of milliseconds
  quarry already pays for one LSP round trip. The parse cost is noise.
- Verified, not assumed: a spike was run during discussion (see "Technical context — spike
  results") that resolved the module in a clean `go mod init` tree, built with no C toolchain, and
  extracted correct symbols, signatures, docstrings, and ranges from real quarry source files.
  Version at time of writing: `v0.51.0`.
- Rejected: official cgo bindings `github.com/tree-sitter/go-tree-sitter` (mature and canonical,
  but destroys the pure-Go build story on the windows target quarry supports);
  a wazero/WASM wrapper such as `github.com/malivvan/tree-sitter` (cgo-free, but adds a WASM runtime
  dependency and per-grammar `.wasm` loading for no gain over a native Go runtime).
- Risk and its containment: the library is young and effectively single-maintainer.
  The containment is the package split below — the backend lives behind an interface in
  `internal/quarryengine/treesitter`, so swapping runtimes is one package, not a rewrite.

### Verb shape: a `toc` command group, not two flat verbs

- Decision: `quarry toc file <path>` and `quarry toc dir <path>`, as a cobra parent command with
  `RunE: cli.GroupRunE` and two subcommands. Bare `quarry toc` prints help.
- Rationale: the two verbs return different shapes and must stay distinct, but `toc` / `toc-dir`
  is asymmetric naming for two siblings. The group form is symmetric, gives one help tree, and uses
  machinery that already exists and is already tested — `GroupRunE` is documented at
  `internal/cli/exec.go:222` as "the RunE for parent module group commands (e.g. `quarry lang`)".
- Rejected: `outline <file>` + `manifest <dir>` (two flat verbs matching the existing four, but two
  unrelated names to learn); `toc <file>` + `survey <dir>` (still asymmetric).

### Path-type validation

- Decision: `os.Stat` the positional argument first.
  A non-existent path is an `output.Err` (exit 1).
  A directory passed to `toc file`, or a file passed to `toc dir`, is a hard error whose message
  names the correct subcommand — e.g. `"<path> is a directory; use quarry toc dir"`.
  Symlinks are followed (`os.Stat`, not `os.Lstat`), so the type validated is the target's.
- Rationale: silently switching behaviour on the path type would defeat the whole reason the two
  verbs are separate, and a mismatched call is a caller bug worth surfacing loudly with the fix in
  the message.
- Rejected: auto-dispatching on stat and running whichever verb fits (re-merges two deliberately
  distinct output shapes); ignoring the mismatch (produces a confusing empty result).

### Language detection: by file extension

- Decision: add an extension-to-language map to `internal/quarryengine/registry`
  (`.go` → go, `.py` → python, `.cs` → csharp, `.ts`/`.tsx` → typescript, `.rs` → rust).
  `--lang` still overrides it, matching the existing verbs' flag.
- Rationale: both new verbs are path-scoped, and the existing `registry.DetectLanguage` matches
  marker files (`go.mod`, `Cargo.toml`, …) against a *directory*, with a fixed precedence order.
  Reusing it would resolve a `.ts` file inside a Go module to "go", because `go.mod` wins the
  precedence list. That is correct for the LSP verbs and wrong for a file-scoped one.
- Rejected: reusing `DetectLanguage` on the file's parent directory (wrong for mixed-language
  trees); sniffing file content (unnecessary — the extension is definitive for all five languages).

### Language scope: five designed, three implemented

- Decision: the extraction-strategy interface and the per-language survey cover all five languages.
  Go, Python, and C# strategies are implemented and tested in this task.
  TypeScript and Rust are registered as a follow-up task; until then, `toc` on a `.ts` or `.rs`
  file returns a clear "language not yet supported by toc" error, never a silent empty result.
- Rationale: the real work per language is the docstring-association rule, not the grammar (all 206
  grammars ship with the runtime). Three languages with structurally *different* docstring
  placements — Go sibling comments, Python in-body string literals, C# XML sibling comments — are
  what proves the interface generalizes. Adding TypeScript and Rust afterwards is then filling in a
  proven shape rather than discovering it.
- Rejected: Go only (leaves the "second parsing backend" abstraction exercised by exactly one
  language, which is not an abstraction, it is a guess); all five now (larger batch with no design
  risk left to retire after the third language).

### Symbol kinds: functions, methods, and type declarations

- Decision: `toc file` lists functions, methods, and type/struct/class/interface declarations.
  Constants, variables, and struct/class fields are excluded.
- Rationale: this is exactly what the brief asks for, and it is the set for which "signature +
  docstring tells you what it is for" holds. A const block is not described by a signature.
- Rejected: adding top-level consts/vars (more noise per token than signal); adding fields and
  nested members as a tree (at that point the agent may as well read the file, which is the thing
  the verb exists to avoid).

### "Top-level" means container-reachable, not literally top-level

- Decision: descend through *container* nodes — Python `class_definition` → `block`, C#
  `namespace_declaration` / `file_scoped_namespace_declaration` → `class_declaration` →
  `declaration_list` — and list the declarations found there.
  Never descend into a function/method body.
- Rationale: "top level" is a Go-shaped notion. The spike confirmed that in Python every method is
  nested inside a class block, and in C# every method is nested inside a declaration list, itself
  usually inside a namespace. Taking "top level" literally would return zero methods for Python and
  C#, which is the opposite of the verb's purpose. The container/body distinction is the rule that
  generalizes: containers are namespacing, bodies are implementation.
- Rejected: literal top-level-children-only (returns nothing useful for 2 of the 3 implemented
  languages); unbounded descent (pulls in local closures and helper functions declared inside
  bodies, which are implementation detail).

### Output: flat symbol list, ownership as a field

- Decision: `symbols` is a flat array. Each entry carries `owner` (the receiver type for a Go
  method, the class name for a Python/C# method) when it has one, and `name` stays the bare
  identifier.
  No nested `children` tree.
- Rationale: two reasons, and both were decisive.
  First, `refs`, `definition`, and `symbol` take a bare symbol name or a `file:line:col` position as
  input. If `toc` returned `"FileLock.Release"` as the name, an agent copying that string straight
  into another quarry call would hand the tool something it does not parse. Separate fields keep the
  name reusable across verbs while ownership stays visible.
  Second, a flat list has the same shape for all five languages no matter how differently the source
  nests, and a caller can always rebuild a tree from a flat list — whereas going the other way would
  require the tool to commit to one nesting form per language.
- Rejected: `name: "FileLock.Release"` (breaks cross-verb reuse of the name);
  a nested `children` tree (more tokens, and a different shape per language).

### Line ranges: 1-based, inclusive, spanning docstring and declaration

- Decision: each symbol carries `"start"` and `"end"`: 1-based line numbers, inclusive at both ends,
  where `start` is the first line of the docstring when one is present and the first line of the
  declaration otherwise, and `end` is the last line of the declaration.
- Rationale: this is the whole point of the verb — an agent that judges an entry relevant reads
  exactly `start`–`end` and gets the prose and the code together. 1-based inclusive matches what a
  reader types into a read tool and matches the `file:line:col` convention the existing verbs use.
- Rejected: a nested `"range": {"start":…, "end":…}` object (same information, more tokens);
  adding byte offsets (precise for tooling, useless to the LLM consumer this tool exists for).
- Cross-task rule: the `impact-verb` task's `definition` and `enclosing_range` fields must use the
  same "docstring is part of the range it precedes" rule. Both verbs treat a docstring as part of
  the definition, never as leading noise.

### Signature: exact source text, body excluded

- Decision: the signature is the verbatim source text from the declaration's first byte up to the
  start of its body node, trimmed.
  No normalization, no reformatting, no synthesized canonical form.
- Rationale: it is what the parser already gives, it is guaranteed correct because no transformation
  can introduce drift, and an LLM reads native source syntax at least as well as an invented normal
  form. A normalizer would be five per-language formatters to write, test, and keep correct.
- Rejected: a normalized `name + param types + return type` rendering (uniform across languages,
  but five formatters' worth of maintenance and five ways to be subtly wrong).

### Docstring text: strip comment syntax, keep the prose

- Decision: strip the language's comment/string delimiters and emit the prose.
  Go: strip the `//` prefix per line. Python: strip the string-literal quotes (use the
  `string_content` node the grammar already exposes). C#: strip the `///` prefix **and** the XML
  doc tags (`<summary>`, `<param>`, …), keeping their text content.
- Rationale: this does not contradict the "exact source text" rule for signatures. That rule exists
  because a signature is code, where reformatting can introduce semantic drift. A docstring is prose
  for reading. Some language-specific delimiter stripping is unavoidable for *every* language just to
  obtain "the text" at all — Go's `//`, Python's quotes — so C#'s XML tags are one more syntax detail
  removed in the same step, not a new class of normalization.
  Leaving `<summary>` wrappers on every C# symbol is pure noise in a token budget.
- Rejected: raw source text for docstrings (consistent-looking, but ships comment markers and XML
  tags to a consumer that wants prose).

### Absent docstring and absent header: omit the field

- Decision: a symbol with no docstring omits the `docstring` key entirely, and its range covers the
  declaration alone.
  A file with no header omits the `header` key, and the file is still listed by `toc dir`.
- Rationale: an omitted key lets a consumer distinguish "no docstring" from "empty docstring"
  without guessing, and is the cheapest option in tokens. A file missing its required header is
  information the agent needs — Loomyard's prose-skills conventions make the header mandatory, so its
  absence is a finding, not something to hide by dropping the file from the listing.
- Rejected: always-present `docstring: null` (stable schema, one wasted line per symbol);
  `docstring: ""` (indistinguishable from a genuinely empty docstring);
  omitting header-less files from `toc dir` (silently loses files from a directory overview).

### File header: first block in the file, blank line tolerated

- Decision: the file header is the **first** comment block in the file, and the rule tolerates a
  blank line between that block and the following declaration.
  This is a different rule from docstring association, which requires strict adjacency.
- Rationale: this decision comes directly out of a spike finding. `internal/output/output.go` has a
  blank line between its header block and `package output`, so the strict-adjacency rule that
  docstrings require drops the header entirely. `internal/cli/cli.go` has *two* top-of-file blocks —
  a file header and a `// Package cli …` doc comment — and strict adjacency picked the package doc.
  Loomyard's prose-skills conventions require a header describing *the file's* purpose; the package
  doc describes the package. First-block-in-file is the rule that returns the former.
- Rejected: the block adjacent to the package/namespace declaration (that is Go's doc convention,
  but in this codebase it returns the package doc instead of the file header);
  emitting both `header` and `package_doc` (most complete, but doubles `toc dir`'s cost for
  something the agent rarely needs).

### `toc file` also emits the file's own header

- Decision: `toc file` returns a top-level `header` field alongside `symbols`.
- Rationale: Tree-sitter has already read and parsed the file to get the symbols, so including the
  header costs effectively nothing, while requiring a second call is friction with no gain.
- Rejected: leaving headers exclusively to `toc dir` (sharper separation of concerns, but forces two
  calls for one file's full context).

### `toc dir`: one level, all known code extensions, truncated headers

- Decision: `toc dir` reads exactly one directory level, non-recursively.
  It lists every file whose extension maps to a *supported* language, regardless of which language —
  a mixed directory produces one list with per-file language resolution.
  Each file's header is truncated to its first paragraph.
- Rationale: recursion is what calling `toc dir` on the subdirectory is for, and a recursive dump
  defeats the "is this worth opening" question the verb answers. Multi-language listing is required
  because a directory is not guaranteed single-language and the per-file extension map already gives
  the answer. First-paragraph truncation is enough because headers are written to make sense
  standalone; the first paragraph is the purpose statement.
- Rejected: recursive with a `--depth` flag (risks enormous output);
  single-language-only listing (silently hides files);
  listing every file including non-code (at that point `ls` is cheaper);
  untruncated headers (most complete, most expensive, and the tail is rarely the purpose).

### Test and generated flags: emit only when the language has a reliable rule

- Decision: emit `"test": true` only when the language has a rule that actually determines it, and
  omit the key entirely otherwise. Never emit `"test": false` for a language where test-ness cannot
  be determined from the file.
  Same policy for `"generated"`.
  - Go: `_test.go` suffix — defined by the toolchain, fully reliable.
  - TypeScript: `*.test.ts` / `*.spec.ts` — jest/vitest defaults, strong convention.
  - Python: `test_*.py` / `*_test.py` — pytest's defaults, convention only.
  - C#: **no rule.** Test-ness lives in attributes (`[Fact]`, `[TestMethod]`, `[Test]`) or in a
    `.csproj` referencing `Microsoft.NET.Test.Sdk`. `*Tests.cs` is style, not a rule. Key omitted.
  - Rust: **no rule.** `#[cfg(test)]` modules live inside ordinary files. Key omitted.
  - `generated`: Go requires a `// Code generated ... DO NOT EDIT.` first line; C# uses
    `<auto-generated>` in the leading comment. Python and Rust have no rule — key omitted.
- Rationale: test files are frequently exactly what an agent is looking for, so they must be listed,
  not filtered out by default. But `"test": false` on a C# test file is a lie, and a consumer cannot
  tell a lie from a fact. Omission lets the agent distinguish "not a test" from "cannot tell".
- Rejected: always emitting `test: true|false` on a best-effort basis (stable schema, but ships
  false negatives as facts); dropping the flag entirely (loses it for Go, where it is reliable and
  useful); excluding tests by default behind an `--include-tests` flag (hides files).

### Unparseable input: partial results, explicitly flagged

- Decision: when the parse tree reports `HasError()`, return the symbols that were recovered and set
  `"partial": true` on that file. Never fail the whole call, and never return partial results
  silently.
- Rationale: Tree-sitter is deliberately error-tolerant, and a file mid-edit should still yield a
  usable outline. Failing hard would let one broken file destroy an entire `toc dir` result.
  The flag is not decoration: the spike showed that error recovery is *lossy*, not merely
  incomplete — an unparseable `func Broken(` swallowed the rest of the file and a later, perfectly
  valid `func Later()` vanished from the output. `partial: true` is the only thing distinguishing
  "this file has two symbols" from "we lost some".
- Rejected: hard-failing the file (loses the directory over one bad file);
  returning recovered symbols with no marker (the consumer cannot know the list is incomplete).

### Batch mode, consistent with the existing four verbs

- Decision: both subcommands accept 2+ positional arguments and switch to the existing batch shape —
  `{"ok":true,"results":[…]}` with a per-entry `status`, and a process exit code set to the worst
  status across the batch.
- Rationale: every existing verb does this, an agent frequently wants several files in one call, and
  the machinery (`runBatch`, `batchStatus` in `internal/cli/cli.go`) already exists.
- Rejected: exactly-one-argument (simpler, but breaks a pattern all four existing verbs follow).

### Package layout: backend and orchestration split

- Decision: two new packages.
  `internal/quarryengine/treesitter` is the parsing backend — grammar loading, parser construction,
  and the node-walking primitives. It imports the engine root only.
  `internal/quarryengine/toc` is the orchestration layer holding the per-language extraction
  strategies and the `TOCFile` / `TOCDir` entry points. It imports the root, `registry`, and
  `treesitter`.
  Both get rows in `layeringTable`.
- Rationale: this mirrors the existing `lsp` (backend) / `query` (orchestration) split exactly, and
  it is what keeps the parser backend a separately swappable layer — which was the entire mitigation
  for the young-library risk accepted in the backend decision.
- Rejected: one combined `toc` package (fewer packages, but the backend stops being its own layer);
  putting `toc` inside the existing `query` package (`query` is LSP orchestration; mixing two
  backends in one package is exactly the coupling the split avoids).

### Facade re-export

- Decision: `quarry/facade.go` gains `quarry.TOCFile` and `quarry.TOCDir` as one-line delegating
  functions, plus type aliases for the result types.
- Rationale: `quarry/` is documented as the stable import path for the engine, and the recent
  thin-facade commit exists precisely to keep that contract. A verb that skips the facade makes it
  incomplete.
- Rejected: `internal/cli` calling the engine package directly (smaller surface, breaks the facade
  contract the previous commit established).

## Technical context

### Existing structure mill-plan must work within

- The engine is a five-package DAG under `internal/quarryengine`: the root leaf (`errors.go`,
  `position.go`, `log.go`), `lsp`, `registry`, `daemon`, and `query`, re-exported by the thin
  `quarry/` facade. `internal/quarryengine/doc.go` documents the DAG in full.
- **Two guard tests will fail if the new packages are added carelessly:**
  - `internal/quarryengine/layering_test.go` walks every `.go` file under `internal/quarryengine/`,
    production *and* `_test.go`, and checks each file's intra-engine imports against
    `layeringTable`. Both new packages need rows there — one production row and one test row each,
    following the existing shape.
  - `internal/quarryengine/seam_enforcement_test.go` fails if any non-test file under
    `internal/quarryengine/` or `quarry/` imports `internal/output`, cobra, or any `internal/*cli`
    package. The new engine packages must return typed Go results and typed errors only.
- `internal/cli` is the sole place engine results become JSON, via `output.Ok` / `output.ErrFields`
  (`internal/output/output.go`).
- Every existing verb's `RunE` follows the same shape and the new ones should too:
  `CwdFrom(ctx)` → resolve target dir against the seam cwd → `resolveContext` → `buildOptions` →
  engine call → `SetExit(ctx, output.Ok(...))`.
  Note that `resolveContext` and `buildOptions` exist to serve the LSP verbs (they load the server
  registry and resolve a daemon state dir); `toc` needs neither a language server nor a state dir,
  so it should **not** be forced through them.
- `cli.GroupRunE` (`internal/cli/exec.go:227`) is the ready-made parent-command `RunE`: it prints
  help on a bare invocation and errors on an unknown subcommand.
- README's "Verbs" section lists four verbs and must be updated.

### Spike results (run during discussion, against real quarry source)

A working spike was built and run — this design is verified, not assumed.

| Test | Result |
| --- | --- |
| `go get` in a clean empty module | `v0.51.0`, no cgo, no C toolchain, builds clean |
| `internal/output/output.go` | 3 symbols, correct signature, docstring, and range spanning docstring + declaration |
| `internal/cli/cli.go` (38.8 KB) | 25 symbols, 10.9 KB of JSON, **113 ms** including process startup |
| `internal/lock/lock.go` | correctly distinguishes `function`, `method` (receiver), and `type` |
| Deliberately broken Go file | `partial: true` set; symbols before the error preserved; a symbol *after* the error was lost |
| Python and C# grammars | load and parse cleanly |

Relevant `gotreesitter` API surface, confirmed present:
`grammars.GoLanguage()` / `PythonLanguage()` / `CSharpLanguage()` / `TypescriptLanguage()` /
`RustLanguage()` (note: `Typescript`, not `TypeScript`);
`ts.NewParser(lang)`, `(*Parser).Parse(src) (*Tree, error)`, `(*Tree).RootNode()`;
`(*Node).Type(lang)`, `.ChildCount()`, `.Child(i)`, `.NamedChild(i)`, `.ChildByFieldName(name, lang)`,
`.PrevSibling()`, `.StartByte()`, `.EndByte()`, `.StartPoint()`, `.EndPoint()`, `.HasError()`;
`Point{Row, Column}` is 0-based, so line numbers are `Row + 1`.
The library also ships an `Outliner` (`outline.go`) with an `Owner` field, which is worth evaluating
for the owner-resolution step, but it does not do docstring association — that walk is ours either
way.

Selective grammar inclusion is supported via the `grammar_set_core` build tag and the
`GOTREESITTER_GRAMMAR_SET` environment variable, and external (non-embedded) blobs via the
`grammar_blobs_external` tag. Binary-size impact of the default embedded set was not measured in the
spike and should be checked during implementation.

### Per-language extraction shapes, confirmed by dumping real parse trees

These are the concrete node shapes the three implemented strategies must handle.
This is the survey material for the `docs/` deliverable.

**Go** — declarations are top-level children of `source_file`.
- Kinds: `function_declaration`, `method_declaration`, `type_declaration`
  (the type's name is nested one level down in a `type_spec`, not on the declaration itself).
- Docstring: contiguous `comment` prev-siblings, walked backwards, stopping at the first blank line
  (i.e. when `prev.EndPoint().Row + 1 != cur.StartPoint().Row`).
- Signature: source from the declaration start to `ChildByFieldName("body")`'s start byte.
- Owner: the receiver type on a `method_declaration`.

**Python** — the docstring is *not* a sibling.
- Kinds: `function_definition`, `class_definition`. Methods are `function_definition` nodes inside
  `class_definition` → `block`.
- Docstring: the first `string` node inside the definition's `block`; its text is in the
  `string_content` child, between `string_start` and `string_end` nodes.
- Module header: the first `string` node at file top-level, same shape.
- Consequence for ranges: because the docstring is inside the body, a Python symbol's range is the
  declaration node's own span — the docstring is already inside it. The `start` value therefore needs
  no adjustment for Python, unlike Go and C#.

**C#** — sibling XML doc comments, with extra nesting.
- Kinds: `class_declaration`, `method_declaration`, plus interface/struct/record equivalents.
  Methods live inside `class_declaration` → `declaration_list`.
- Container nesting: `file_scoped_namespace_declaration` (a `namespace X;` statement, one flat level)
  and classic `namespace_declaration` (braced, adding a level) both need descending through.
- Docstring: `comment` prev-siblings whose text starts with `///`, containing XML tags to strip.
- Note the spike file used a `=>` expression-bodied method, which has an
  `arrow_expression_clause` rather than a body block — the signature rule must handle both
  block-bodied and expression-bodied members.

**TypeScript and Rust** (designed, not implemented in this task):
TypeScript uses `/** … */` JSDoc block comments as prev-siblings; Rust uses `///` and `//!` line
doc-comments as prev-siblings, with `//!` being an *inner* doc comment that documents the enclosing
item rather than the following one — which is how Rust file headers are written and is a genuine
trap for the header rule.

### Spike artefacts

The spike lives under `.scratch/tsspike/` (gitignored). It is throwaway reference code, not a
starting point to be promoted — implementation should be written properly against the interface
described above. It is worth reading once for the working docstring-walk and the tree dumper.

## Constraints

There is no `CONSTRAINTS.md` at the hub root. Constraints discovered during discussion:

- **No cgo.** `go.mod` is pure Go today and the README supports linux and windows. Adding a C
  toolchain requirement to the build is out.
- **darwin is already unbuildable** (`internal/proc` has no darwin implementation) and this task must
  not change that either way.
- **The two guard tests are non-negotiable.** New packages must be added to `layeringTable`, and no
  engine package may import `internal/output`, cobra, or `internal/cli`.
- **JSON only.** `output.Ok` / `output.Err` is the envelope for all verbs; `toc` introduces no
  second output format.
- **The facade must stay complete.** Anything the CLI can call, `quarry/` re-exports.

## Testing

TDD candidates are the pure, deterministic units — every extraction rule below is a
source-text-in / struct-out function with no I/O, which makes them the natural place to write the
test first.

**`internal/quarryengine/toc` — per-language extraction (TDD, highest value).**
Table-driven tests over small inline source fixtures, one table per language.
Scenarios that must be covered for each implemented language:
- a symbol with a docstring — assert the range starts at the docstring's first line;
- a symbol with no docstring — assert the `docstring` field is absent and the range starts at the
  declaration;
- two symbols where a blank line separates the first one's trailing comment from the second
  declaration — assert the comment is not misattributed to the second symbol;
- a comment block separated from its declaration by a blank line — assert it is *not* treated as a
  docstring (this is the rule that differs from the header rule);
- a method — assert `owner` is populated and `name` stays bare;
- a container-nested declaration (Python class method, C# namespace→class→method) — assert it is
  found;
- a declaration inside a function body — assert it is *not* found;
- an expression-bodied C# member (`=>`) — assert the signature is correct;
- a Python symbol — assert the range needs no docstring adjustment because the docstring is in-body.

**Header extraction (TDD).**
Explicitly cover the two shapes the spike found in this repo:
- a header separated from `package` by a blank line — assert it is still found;
- a file with both a file header and a package doc comment — assert the *first* block wins.
Plus: a file with no header at all — assert the key is absent and the file is still returned.

**Docstring text stripping (TDD).**
One test per language delimiter form: Go `//`, Python triple-quoted `string_content`, C# `///` plus
XML tag stripping. Include a C# case with `<summary>` and `<param>` to assert tags are removed and
their text kept.

**Test/generated classification (TDD).**
Assert the *omission* behaviour explicitly, not just the positive case: a C# file named `FooTests.cs`
must produce no `test` key at all, and a Go file named `foo_test.go` must produce `test: true`.
This is the rule most likely to rot into a best-effort `false`.

**Error tolerance.**
Use the exact spike scenario as a fixture: a file whose broken declaration swallows a later valid
one. Assert `partial: true` is set and that the surviving symbols are returned. This test documents
that recovery is lossy.

**`internal/quarryengine/treesitter` — backend.**
Thin. Assert each supported grammar loads non-nil and parses a trivial valid file without error.
This is the canary for a grammar-registry change in the upstream library.

**`internal/quarryengine/registry` — extension map.**
Table test over all five extensions plus an unknown one, plus the `--lang` override path.

**`internal/cli` — verb wiring.**
Follow the existing `cli_test.go` patterns.
- `toc file` on a directory and `toc dir` on a file — assert the hard error and that the message
  names the correct subcommand;
- a non-existent path — assert `output.Err` and exit 1;
- bare `quarry toc` — assert help is printed (the `GroupRunE` contract);
- batch mode with 2+ arguments — assert the `results` array shape and the worst-status exit code,
  matching the existing verbs' batch tests;
- a `.ts` or `.rs` file — assert the explicit "not yet supported" error rather than an empty result.

**Guard tests.**
`layering_test.go` and `seam_enforcement_test.go` must pass unmodified except for the new
`layeringTable` rows. Do not relax either guard to make the new packages fit.

**Integration.**
One test running `toc file` against a real file in this repository (`internal/output/output.go` is a
good choice — small, stable, three well-documented symbols) asserting the full JSON envelope.
This is the end-to-end proof the spike demonstrated by hand.

## Q&A log

- **Q:** Is C faster, and does that matter? **A:** ~3.9x on full parse, but 6.77 ms vs 1.7 ms against the tens-to-hundreds of ms quarry already pays per LSP round trip. Speed is not the deciding factor; the pure-Go build story on windows is.
- **Q:** Does Tree-sitter support all languages regardless of binding? **A:** Yes — grammars are per-language parse tables, and both the cgo bindings and the pure-Go runtime cover all five of quarry's languages. Language coverage does not separate the options.
- **Q:** One `toc <path>` verb dispatching on stat, or two? **A:** Two, because the output shapes differ — but as `toc file` / `toc dir` subcommands rather than the asymmetric `toc` / `toc-dir`.
- **Q:** What if the caller passes the wrong path type? **A:** Stat first, validate against the subcommand's expected type, hard-fail on mismatch with a message naming the correct subcommand.
- **Q:** Should `toc dir` recurse? **A:** No. Explicitly non-recursive.
- **Q:** Should the docstring be part of `toc file`'s output, and should the line range cover it? **A:** Yes to both — this is the core premise, not an option. Signature + docstring is what tells a reader what a method is for.
- **Q:** Should `toc dir` list only the detected language's files, or all languages? **A:** All supported languages, since a directory is not guaranteed single-language. Go first, Python and C# in the same task, TypeScript and Rust deferred.
- **Q:** How can a test file even be flagged, given Go has `_test.go` but C# has nothing? **A:** It cannot, uniformly. Emit the key only where a reliable rule exists; omit it entirely for C# and Rust rather than emitting a false `test: false`.
- **Q:** Is the output format even JSON? Is `start`/`end` too verbose? **A:** JSON, because all four existing verbs use the `output.Ok` envelope and `internal/cli` is the sole JSON site. Flat `"start"` / `"end"` keys rather than a nested range object, to keep it cheap.
- **Q:** Have you actually tested Tree-sitter and got it working? **A:** Not at the time of the recommendation — the recommendation rested on documentation and the platform argument. A spike was then built and run against real quarry source, and it produced four findings that changed the design: the header rule must tolerate a blank line, error recovery is lossy rather than merely incomplete, "top-level" does not generalize beyond Go, and docstring placement is structurally different per language.
