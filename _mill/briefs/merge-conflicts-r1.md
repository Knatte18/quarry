# Conflict Resolution Brief

Your sole job is to resolve git conflict markers in the listed files, stage each resolved file, and report success.
Do NOT commit.
Do NOT run `git merge --continue` — the SKILL does that after receiving `{"status":"success"}`.

## Task intent

These excerpts describe what THIS branch is trying to accomplish.
When the merge introduces a parent-side change that conflicts with this branch's intent, the resolution preserves THIS branch's intent.
In particular: if a file appears under a batch's `Deletes:` list and the merge introduces a modified version of that file from the parent, the resolution is to delete the file (your branch's intent overrides).
Stage the deletion with `git -C /home/knatte/Code/quarry/wts/toc-verbs rm <file>`.

### From discussion.md

# Discussion: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
task: Add file/dir toc verbs (Tree-sitter-backed)
slug: toc-verbs
status: discussing
parent: main
```

> This file is the specification: what is decided, and what the plan must deliver.
> The rationale, the measurements behind each choice, and every rejected alternative live in
> `_mill/discussion-meta.md`. Nothing there is needed to write the plan — read it only when you want
> to know why something is the way it is, or before reopening a settled question.

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

The premise, binding on the whole design: *a method's signature plus its docstring must be enough to
tell a reader what the method is for.*
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
- A new Tree-sitter parsing backend package, `internal/quarryengine/treesitter`, using the official
  cgo bindings `github.com/tree-sitter/go-tree-sitter` plus one grammar module per language.
- A new orchestration package, `internal/quarryengine/toc`, holding the per-language extraction
  strategies and the two entry points.
- An extension-to-language map in `internal/quarryengine/registry`, since both new verbs are
  path-scoped and the existing `DetectLanguage` is directory-marker-scoped.
- Per-language extraction strategies **designed for all five languages** (Go, Python, C#,
  TypeScript, Rust) and **implemented and tested for Go, Python, and C#** in this task.
- Facade re-exports `quarry.TOCFile` and `quarry.TOCDir` in `quarry/facade.go`.
- New rows in `internal/quarryengine/layering_test.go`'s `layeringTable` for both new packages.
- README updates: the new build dependency (`CGO_ENABLED=1` and a C toolchain) with the
  windows cross-compile recipe, the verb list, and the testing section.
- A per-language docstring-association survey at **`docs/toc-docstring-association.md`**, in the
  spirit of `docs/scout-multilang.md`. Its content is the per-language node-shape material recorded
  under "Technical context" below, written up as a standalone reference — including the two
  languages this task designs but does not implement.
- **Stale-prose sweep.** See the dedicated subsection below; it is a deliverable, not a courtesy.

**Out:**

- TypeScript and Rust *implementation*. Their strategies are designed here and the interface must
  accommodate them, but the concrete strategies and their tests are a separate follow-up task.
- Any change to `refs`, `definition`, `symbol`, or `assert-no-callers`. The LSP path is untouched.
- Replacing the LSP backend. Tree-sitter is added alongside it; no existing verb switches backends.
- Recursive directory walking. `toc dir` reads exactly one directory level.
- A non-JSON output format. Both verbs print the existing `output.Ok` / `output.Err` envelope,
  same as the four existing verbs.
- Constants, variables, and struct/class *fields*. Only functions, methods, and type declarations
  are listed.
- Function bodies, and any symbol declared inside a function body (local closures, nested helpers).
- The `impact` verb. This task only records the shared "declaration + docstring is one range" rule
  that the `impact-verb` task must also honour; it does not implement `impact`.
- Caching or a daemon for the parser. Decided against on measured evidence
  (`discussion-meta.md` §2).
- Shipping two parsing backends selected by build tag. Decided against on correctness grounds
  (`discussion-meta.md` §1.5).

### Stale-prose sweep (in scope)

**The invariant:** *no prose anywhere in the tree may, after this task lands, still assert any of —*
(a) *a count or enumeration of the engine's packages;*
(b) *a count or enumeration of quarry's verbs or capabilities — including prose that lists what
quarry does without counting it, such as the root cobra command's `Short`;*
(c) *a count of the facade's re-exports;*
(d) *that quarry is LSP-only, or that it does not parse source itself;*
(e) *that a batch entry is keyed on `symbol`;*
(f) *that quarry builds without a C toolchain.*

Anything matching that description is in scope whether or not the commands below find it — a phrase
grep is a search aid, not the specification. Clauses (d)–(f) are not count-shaped and no grep
reliably reaches them.

Two search aids, both run and their output read:

```
# (a) counts and exact-number claims
grep -rnE 'five-package|four verbs|29 identifiers|eight blank-identifier|LSP-backed|minPackageDirs|The six internal|import all four|seven re-exported' \
  --include='*.go' --include='*.md' . | grep -v '^\./_mill/' | grep -v '^\./\.scratch/'

# (b) package-set enumerations, which (a) misses because they use no count word
grep -rnE 'lsp, registry|registry, daemon|root leaf' \
  --include='*.go' --include='*.md' . | grep -v '^\./_mill/' | grep -v '^\./\.scratch/'
```

**Out of scope:** `docs/scout-vs-grep.md:3` and `:130` use "LSP-backed" to describe a past
measurement of `lyx scout`, not a current claim about quarry. Historical research documents are not
updated when the product changes.

Everything else currently in scope:

- `quarry/facade.go:8` — "It re-exports exactly the 29 identifiers this package exported before
  the engine-repackage move: no more, no less." Adding `TOCFile`, `TOCDir`, and the toc result
  type aliases breaks this sentence. Recount and rewrite; do not leave a stale number.
- `quarry/facade_test.go:103` — "The eight blank-identifier assignments below reference every
  delegating function in facade.go". That block is a compile-time signature guard, so `TOCFile`
  and `TOCDir` must each get an entry **and** the count in the comment must be updated.
- `quarry/facade_test.go:117` — "each of the seven re-exported sentinel error values". Becomes
  eight: toc adds `ErrLanguageUnsupported`.
- `internal/cli/cli.go:7` — the package doc says "exposing four verbs" and enumerates them.
- `internal/cli/cli.go:23-29` — the package doc's batch contract says "one JSON entry per symbol
  under a top-level \"results\" array"; toc's driver keys entries on `path`, so the contract must be
  restated to cover both.
- `internal/cli/cli.go:51` — the root cobra command's `Short`: "code intelligence lookups
  (references, definitions, symbol search) across supported languages". A capability enumeration
  that leaves `quarry --help` describing a tool without `toc`.
- `README.md:3` — "quarry is an LSP-backed code intelligence tool."
  `README.md:4` — "by speaking the Language Server Protocol to each language's own server rather
  than reimplementing a parser per language". After this task quarry parses source itself.
  `README.md:8` — "quarry exposes four verbs".
  README's "Building and running" and "Testing" (`README.md:66-71`) sections.
- `quarry/facade.go:3`, `internal/quarryengine/doc.go:44`, and
  `internal/quarryengine/seam_enforcement_test.go:10` — all three call the engine a
  "five-package DAG". It becomes seven packages (adding `treesitter` and `toc`).
- `internal/quarryengine/doc.go:66` — "It imports all four packages above", describing `query`.
- Package-set enumerations, each of which lists the DAG's members and must gain the two new
  packages: `quarry/facade.go:4`; `internal/quarryengine/doc.go:25` and `:37`;
  `internal/quarryengine/seam_enforcement_test.go:2-3` and `:101`;
  `internal/quarryengine/layering_test.go:159`.
- `internal/quarryengine/doc.go:47-71` — the bulleted package-layout list describing each package
  and its allowed imports. Needs **two new bullets**, not just a count edit. Largest single doc
  change in the task, and the one most likely to be skipped.
- `internal/quarryengine/layering_test.go:20` — "The six internal/quarryengine/... import paths
  this guard reasons about" (becomes eight), and `:53` — "query's production files import all four".
- `internal/quarryengine/layering_test.go:162` and
  `internal/quarryengine/seam_enforcement_test.go:104` — `const minPackageDirs = 6` in both.
  **These are floors (`< minPackageDirs` fails), so adding packages does not break the build** —
  which is the risk: the guard silently loses strength while still passing.
  **The two constants do not mean the same thing, so their comment rewrites differ:**
  `layering_test.go` walks only `internal/quarryengine/` (6 dirs today, 8 after), so 8 is the
  *exact* count there. `seam_enforcement_test.go` walks that tree **plus** `quarry/` (7 today, 9
  after), and its comment at `:102-103` states the floor is deliberately one below the real count;
  raising it to 8 preserves that intentional slack.
  Raise both to 8; write each comment to say what its 8 actually is.

## Decisions

### Parsing backend: official cgo Tree-sitter bindings

- Use `github.com/tree-sitter/go-tree-sitter` (v0.25.0), the canonical cgo bindings for the upstream
  C runtime, with one grammar module per language — `tree-sitter-go`, `tree-sitter-python`,
  `tree-sitter-c-sharp`, `tree-sitter-typescript`, `tree-sitter-rust`.
  Each grammar is an ordinary Go module vendoring its generated C parser: no external grammar build
  step, no build tags, no grammar-set choice. Import the five you want.
- `CGO_ENABLED=1` and a C toolchain are build dependencies. Running the built binary requires
  nothing extra.
- Rationale: end-to-end cost is ~0.5 ms per invocation against ~80 ms for the pure-Go alternative,
  dominated by grammar load rather than parsing; binary size is equivalent either way. A C toolchain
  is a one-time per-machine setup for a tool built by its author, and is ordinary for Go projects.
  Full measurements and the alternatives weighed: `discussion-meta.md` §1.
- **Windows builds.** A C toolchain is needed to *build*, never to *run*. Two supported routes, both
  to be documented in README:
  - natively on windows with mingw-w64 (MSYS2) or TDM-GCC;
  - cross-compiled from linux or WSL2:
    `CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags '-extldflags "-static"' -o quarry.exe ./cmd/quarry`
    (requires `gcc-mingw-w64-x86-64`; `-static` is deliberate, so the `.exe` does not depend on
    `libwinpthread-1.dll` and friends sitting beside it).

  **This cross-compile command is unverified — it was not run.** The implementation batch must run
  it and adjust if needed rather than copy it on trust.
  **If no mingw-w64 toolchain is available to the batch, this does not block the task:** document
  the recipe in README marked explicitly as unverified, and file a follow-up to verify it on a
  machine that has the toolchain. Windows support is a property of the *source* — nothing in this
  task's code is platform-specific and `internal/proc`'s windows implementation is untouched — so an
  unverified build recipe is a documentation gap, not a regression.

### One parsing backend, not two

- Exactly one backend ships. quarry does **not** select between the cgo runtime and a pure-Go
  fallback via `//go:build cgo` / `//go:build !cgo`.
- Rationale: two independent Tree-sitter implementations do not produce identical trees, so
  `partial: true` and the symbols surviving a syntax error could differ between builds — `quarry toc`
  would return different answers for the same file depending on how the binary was compiled. Full
  argument: `discussion-meta.md` §1.5.

### No parser daemon

- `toc` spawns no daemon, shares no state between calls, and caches nothing. Every invocation loads
  its grammar, parses, and exits.
- Rationale: Tree-sitter has no project index and no cross-file state, so there is nothing for a
  daemon to keep warm; the LSP daemon exists for gopls's seconds-long cold start and is LSP-shaped
  rather than reusable. Batch mode already captures the one-process win. Measurements and the full
  argument: `discussion-meta.md` §2.

### Verb shape: a `toc` command group

- `quarry toc file <path>` and `quarry toc dir <path>`, as a cobra parent command with
  `RunE: cli.GroupRunE` and two subcommands. Bare `quarry toc` prints help.
- Rationale: the two verbs return different shapes and must stay distinct, but the group form is
  symmetric, gives one help tree, and uses machinery that already exists and is already tested —
  `GroupRunE` is documented at `internal/cli/exec.go:222` as "the RunE for parent module group
  commands (e.g. `quarry lang`)".

### Emitted schema: the closed key set for both verbs

The exact keys each verb emits are fixed here, so the plan writer invents none.

**`toc file <path>`** — envelope fields:
`header` (string, omitted when absent), `language` (string, one of the five canonical names),
`symbols` (array), `partial` (bool, omitted when false).

**Each entry in `symbols`:**
`kind` (string, closed vocabulary below), `name` (bare identifier),
`owner` (string, omitted when the symbol has no owner), `signature` (string),
`docstring` (string, omitted when absent), `start` (int), `end` (int).

**`toc dir <path>`** — envelope field `files` (array). **Each entry:**
`path` (string, see the path-handling Decision), `language` (string),
`header` (string, omitted when absent), `partial` (bool, omitted when false),
`test` (bool, omitted when the language has no reliable rule),
`generated` (bool, same omission rule), `error` (string, present only on a per-file failure, and
mutually exclusive with both `header` and `partial`).

`partial` **is** in the `toc dir` key set: `toc dir` parses each file to extract its header, so a
file whose parse reports `HasError()` is exactly as lossy here as in `toc file`, and the header it
yielded may be wrong or missing for that reason. `partial` and `error` are mutually exclusive by
construction — `partial` means "parsed, lossily", `error` means "never parsed at all".

- **The `kind` vocabulary is closed and shared across all five languages: `function`, `method`,
  `type`.**
  A Python `class_definition`, a C# `class_declaration` / `interface_declaration` /
  `record_declaration` / `struct_declaration`, and a Go `type_spec` all emit `kind: "type"`.
  A function with an owner (Go method receiver, Python/C# member) emits `kind: "method"`; one
  without emits `kind: "function"`.
  Rationale: three kinds is the set for which "signature + docstring tells you what it is for" holds
  uniformly, and a richer vocabulary would be per-language noise the verbatim signature already
  shows on the very next field.
- **Ordering is part of the contract.**
  `symbols` is in **source order**, ascending by `start`.
  `files` is in **lexicographic order by filename**, explicitly sorted — not the order the OS
  returns directory entries in, which is not stable across filesystems.
- **toc adds exactly one engine sentinel: `ErrLanguageUnsupported`.**
  Returned by `internal/quarryengine/toc` when a resolved extension maps to no language, or to one
  designed but not yet implemented; re-exported through `quarry/facade.go` like every other
  sentinel, and classified by `internal/cli` with `errors.Is`, matching the existing verbs
  (`internal/cli/cli.go:875-906`).
  Everything else stays CLI-side: path existence and path-type validation happen in `internal/cli`
  before the engine is called (they are `os.Stat` concerns the CLI already owns), and read failures
  surface as wrapped `os` errors with no sentinel of their own.

### Path handling: seam-relative resolution, round-trippable output

- **Relative positional arguments are joined against the seam cwd**, `CwdFrom(ctx)`
  (`internal/cli/cwdcontext.go:41`), never passed to `os.Stat` raw. This mirrors what every existing
  verb's `RunE` does when defaulting `--target-dir`, and it is what makes `RunCLIIn`/`WithCwd` — the
  seam `internal/cli`'s own tests depend on — actually govern toc.
- **`toc dir` emits each file's `path` as the directory argument *as the caller wrote it*, joined
  with the file's name.** So `quarry toc dir internal/cli` yields `internal/cli/exec.go`, which the
  agent can paste straight into `quarry toc file internal/cli/exec.go` from the same cwd.
- **The batch `"path"` key echoes the positional argument verbatim**, exactly as `runBatch` echoes
  `arg` today (`internal/cli/cli.go:918`) — not the absolutized form.
- Rationale: these paths exist to be fed back into another quarry call by an agent, so they must
  round-trip in the caller's own frame of reference.

### Path-type validation

- `os.Stat` the positional argument (after the seam-cwd join above).
  A non-existent path is an `output.Err` (exit 1).
  A directory passed to `toc file`, or a file passed to `toc dir`, is a hard error whose message
  names the correct subcommand — e.g. `"<path> is a directory; use quarry toc dir"`.
  Symlinks are followed (`os.Stat`, not `os.Lstat`), so the type validated is the target's.
- Rationale: silently switching behaviour on path type would defeat the reason the two verbs are
  separate, and a mismatched call is a caller bug worth surfacing with the fix in the message.

### Language detection: by file extension

- Add an extension-to-language map to `internal/quarryengine/registry`
  (`.go` → go, `.py` → python, `.cs` → csharp, `.ts`/`.tsx` → typescript, `.rs` → rust).
- Rationale: both new verbs are path-scoped, while `registry.DetectLanguage` matches marker files
  (`go.mod`, `Cargo.toml`, …) against a *directory* with a fixed precedence order. Reusing it would
  resolve a `.ts` file inside a Go module to "go" — correct for the LSP verbs, wrong for a
  file-scoped one.
- **`--lang` for toc is validated against toc's own vocabulary, not the server registry.**
  The existing verbs' `--lang` is a registry key looked up inside `registry.DetectLanguage` against
  the registry `resolveContext` loaded from servers.yaml (`internal/cli/cli.go:153`, flag help at
  `:205`). toc does not go through `resolveContext` — it needs no language server and no state dir —
  so it must not reuse that validation path.
  toc's `--lang` accepts exactly the canonical names its own extension map defines (`go`, `python`,
  `csharp`, `typescript`, `rust`). An unrecognised value is an `output.Err` naming the valid set.
- **`--lang` overrides the extension outright; a mismatch is not an error.**
  `quarry toc file --lang go foo.py` parses `foo.py` with the Go grammar and will almost certainly
  return `partial: true` with few or no symbols. This matches what `--lang` means on every existing
  verb: an explicit operator override that wins over detection. Erroring on mismatch would make the
  flag useless for its actual purpose — a file with a non-standard extension.
- **For `toc dir`, `--lang` restricts the listing** to that language's extensions instead of listing
  every supported language found.
- **`toc dir --lang <designed-but-unimplemented>` lists, it does not error.**
  `quarry toc dir --lang rust <dir>` returns `ok:true`, exit 0, with every `.rs` file listed carrying
  `error: "language not yet supported by toc"` — identical to the extension path's behaviour for
  `.rs` files in a mixed directory. `--lang` here selects *which files to list*.
  `toc file --lang rust <path>` is `output.Err` exit 1: there the unsupported language is the entire
  request and there is nothing to list.

### Language scope: five designed, three implemented

- The extraction-strategy interface and the per-language survey cover all five languages.
  Go, Python, and C# strategies are implemented and tested in this task.
  TypeScript and Rust are a follow-up task; until then, `toc` on a `.ts` or `.rs` file returns a
  clear "language not yet supported by toc" error, never a silent empty result.
- Rationale: the real work per language is the docstring-association rule, not the grammar. Three
  languages with structurally *different* docstring placements — Go sibling comments, Python in-body
  string literals, C# XML sibling comments — are what proves the interface generalizes.

### Symbol kinds: functions, methods, and type declarations

- `toc file` lists functions, methods, and type/struct/class/interface declarations.
  Constants, variables, and struct/class fields are excluded.
- Rationale: this is the set for which "signature + docstring tells you what it is for" holds. A
  const block is not described by a signature.

### "Top-level" means container-reachable, not literally top-level

- Descend through *container* nodes — Python `class_definition` → `block`, C#
  `namespace_declaration` / `file_scoped_namespace_declaration` → `class_declaration` →
  `declaration_list` — and list the declarations found there.
  Never descend into a function/method body.
- Rationale: "top level" is a Go-shaped notion. In Python every method is nested inside a class
  block; in C# inside a declaration list, itself usually inside a namespace. Taken literally the rule
  returns zero methods for two of the three implemented languages. Containers are namespacing;
  bodies are implementation.

### Output: flat symbol list, ownership as a field

- `symbols` is a flat array. Each entry carries `owner` (the receiver type for a Go method, the
  class name for a Python/C# method) when it has one, and `name` stays the bare identifier.
  No nested `children` tree.
- Rationale: `refs`, `definition`, and `symbol` take a bare symbol name or a `file:line:col` position
  as input, so a qualified `"FileLock.Release"` in `name` would hand another quarry call something it
  does not parse. Separate fields keep the name reusable across verbs while ownership stays visible.
  A flat list also has the same shape for all five languages regardless of how the source nests.

### Line ranges: 1-based, inclusive, spanning docstring and declaration

- Each symbol carries `"start"` and `"end"`: 1-based line numbers, inclusive at both ends, where
  `start` is the first line of the docstring when one is present and the first line of the
  declaration otherwise, and `end` is the last line of the declaration.
- Rationale: this is the whole point of the verb — an agent that judges an entry relevant reads
  exactly `start`–`end` and gets the prose and the code together. 1-based inclusive matches both what
  a reader types into a read tool and the `file:line:col` convention the existing verbs use.
- **Cross-task rule:** the `impact-verb` task's `definition` and `enclosing_range` fields must use
  the same "docstring is part of the range it precedes" rule. Both verbs treat a docstring as part of
  the definition, never as leading noise.

### Signature: exact source text, cut at the body-bearing child

- The signature is the verbatim source text from the declaration's first byte up to the start of that
  declaration's **body-bearing child node**, trimmed.
  No normalization, no reformatting, no synthesized canonical form, and **never a "first line only"
  truncation**.
- "Body-bearing child" is resolved per kind, and every implemented kind has an explicit rule:
  - Go function / method: `ChildByFieldName("body")` (a `block`).
  - Go type declaration: the type's own body node — `field_declaration_list` for a `struct_type`,
    the interface body for an `interface_type`. So `type FileLock struct` is the signature, not the
    whole struct with its fields.
  - Go type alias / defined type with no body (`type ID string`): no body-bearing child exists, so
    the signature is the whole `type_spec` text. This is short by construction.
  - Python: `ChildByFieldName("body")` (a `block`), for both `function_definition` and
    `class_definition`.
  - C# block-bodied member: the `declaration_list` (for a type) or the method's `block`.
  - C# expression-bodied member (`=>`): the `arrow_expression_clause`, so
    `public void Resize(double factor)` is the signature and the expression is excluded.
- **The Go type symbol unit is always the `type_spec`, in both the grouped and ungrouped forms** —
  one rule, no branch on which shape the source used. The *emitted* signature and range are computed
  from the enclosing `type_declaration` when that declaration holds exactly one spec:
  - Ungrouped `// Doc.` + `type FileLock struct { … }`: one symbol. Its docstring is the
    `type_declaration`'s contiguous `comment` prev-siblings (the ordinary rule — the comment precedes
    the declaration, not the spec). Its `start` is the docstring's first line and its signature is
    `type FileLock struct`, **including the `type` keyword**, because the signature is cut from the
    declaration's first byte, not the spec's. `FileLock struct` would be invalid Go and useless to
    paste anywhere.
  - Grouped `type ( … )`: one symbol per spec. Each spec's signature is rendered with the `type`
    keyword prepended so both forms produce identical output for identical types, and each spec's
    docstring is its own preceding comment block inside the group. A comment attached to the
    `type (` line documents the group, not any one spec, and is dropped.
- Rationale: it is what the parser already gives, no transformation can introduce drift, and an LLM
  reads native source syntax at least as well as an invented normal form. The per-kind body rule is
  what makes "no normalization" implementable at all: a Go `type_declaration` has no `body` field, so
  a naive `ChildByFieldName("body")` returns nil and the signature silently becomes the entire struct
  body — the exact token blowup this verb exists to prevent.

### Docstring text: strip comment syntax, keep the prose

- Strip the language's comment/string delimiters and emit the prose.
  Go: strip the `//` prefix per line. Python: strip the string-literal quotes (use the
  `string_content` node the grammar already exposes). C#: strip the `///` prefix **and** the XML doc
  tags (`<summary>`, `<param>`, …), keeping their text content.
- Rationale: this does not contradict the "exact source text" rule for signatures. That rule exists
  because a signature is code, where reformatting can introduce semantic drift; a docstring is prose
  for reading. Some language-specific delimiter stripping is unavoidable for *every* language just to
  obtain "the text" — Go's `//`, Python's quotes — so C#'s XML tags are one more syntax detail
  removed in the same step, not a new class of normalization.

### Absent docstring and absent header: omit the field

- A symbol with no docstring omits the `docstring` key entirely, and its range covers the
  declaration alone.
  A file with no header omits the `header` key, and the file is still listed by `toc dir`.
- Rationale: an omitted key lets a consumer distinguish "no docstring" from "empty docstring" without
  guessing, and is cheapest in tokens. A file missing its required header is information the agent
  needs — Loomyard's prose-skills conventions make the header mandatory, so its absence is a finding,
  not something to hide by dropping the file.

### File header: first non-directive block, blank line tolerated

- The file header is the first **non-directive** comment block in the file, and the rule tolerates a
  blank line between that block and the following declaration.
  This is a different rule from docstring association, which requires strict adjacency.
- **Directive-only leading blocks are skipped, and the next block is taken.**
  A leading comment block is a directive block — and skipped — when *every* line in it, after
  delimiter stripping, matches a known directive form:
  - Go: `go:build`, `+build`, `go:generate`, `go:embed`, `nolint`, and the generated-file banner
    `Code generated ... DO NOT EDIT.`.
    The banner is a directive block for header purposes — the same class of non-purpose noise as a
    build constraint. The same block is still read by the `generated` detection rule; being skipped
    as a header and being consumed as a marker are independent.
    C#'s `<auto-generated>` block is treated identically.
  - Python: a `#!` shebang on line 1, and a PEP 263 coding line (`coding[:=]`) on line 1 or 2.
  - C#, TypeScript, Rust: preprocessor and attribute forms (`#pragma`, `#nullable`, `#!`) are not
    `comment` nodes in these grammars, so they never reach this rule. Rust's `//!` inner doc comment
    is a *header*, not a directive — see the per-language notes.

  A block mixing directive and prose lines is **not** a directive block and is taken as the header.
  If every leading block is a directive block, the file has no header and the key is omitted.
- This is not hypothetical. `internal/proc/proc_windows.go:1` is `//go:build windows`, then a blank
  line, then the real header at `:3`; same shape in
  `internal/quarryengine/query/refs_integration_test.go:1` (`//go:build lsp`),
  `daemon/supervised_lsp_test.go:1`, and `daemon/ensureserver_integration_test.go:1`.
  Without the rule, `quarry toc dir internal/proc` emits `header: "go:build windows"` — the build
  constraint presented as the file's purpose, worse than emitting no header at all.
- The first-block rule (rather than the block adjacent to `package`) is also load-bearing:
  `internal/output/output.go` has a blank line between its header and `package output`, and
  `internal/cli/cli.go` has *two* top-of-file blocks — a file header and a `// Package cli …` doc.
  Loomyard's prose-skills conventions require a header describing *the file's* purpose; the package
  doc describes the package.

### `toc file` also emits the file's own header

- `toc file` returns a top-level `header` field alongside `symbols`.
- Rationale: the file is already parsed to get the symbols, so including the header costs
  effectively nothing, while requiring a second call is friction with no gain.

### `toc dir`: one level, all known code extensions, truncated headers

- `toc dir` reads exactly one directory level, non-recursively.
- It lists every file whose extension maps to a supported language, regardless of which language — a
  mixed directory produces one list with per-file language resolution.
- Each file's header is truncated to its first paragraph, where **"first paragraph" is defined after
  delimiter stripping, not before**: strip the comment or string delimiters per the docstring-text
  rule, then cut at the first blank line in the resulting prose.
  One rule covers every comment form without a per-form special case, because a bare `//` line in a
  Go block, a bare `///` line in a C# block, and a blank line inside a Python module docstring all
  strip to the same empty line. A header with no blank line is emitted whole.
- Rationale: recursion is what calling `toc dir` on the subdirectory is for, and a recursive dump
  defeats the "is this worth opening" question the verb answers. Multi-language listing is required
  because a directory is not guaranteed single-language. First-paragraph truncation suffices because
  headers are written to make sense standalone; the first paragraph is the purpose statement.

### Test and generated flags: emit only when the language has a reliable rule

- Emit `"test": true` only when the language has a rule that actually determines it, and omit the key
  entirely otherwise. Never emit `"test": false` for a language where test-ness cannot be determined
  from the file. Same policy for `"generated"`.
  - Go: `_test.go` suffix — defined by the toolchain, fully reliable.
  - TypeScript: `*.test.ts` / `*.spec.ts` — jest/vitest defaults, strong convention.
  - Python: `test_*.py` / `*_test.py` — pytest's defaults, convention only.
  - C#: **no rule.** Test-ness lives in attributes (`[Fact]`, `[TestMethod]`, `[Test]`) or in a
    `.csproj` referencing `Microsoft.NET.Test.Sdk`. `*Tests.cs` is style, not a rule. Key omitted.
  - Rust: **no rule.** `#[cfg(test)]` modules live inside ordinary files. Key omitted.
  - `generated`: Go requires a `// Code generated ... DO NOT EDIT.` first line; C# uses
    `<auto-generated>` in the leading comment. Python and Rust have no rule — key omitted.
- Rationale: test files are frequently exactly what an agent is looking for, so they must be listed,
  not filtered out. But `"test": false` on a C# test file is a lie a consumer cannot distinguish from
  a fact. Omission lets the agent tell "not a test" from "cannot tell".

### Unparseable and unreadable input: partial results, explicitly flagged

- When the parse tree reports `HasError()`, return the symbols that were recovered and set
  `"partial": true` on that file. Never fail the whole call, and never return partial results
  silently.
  Tree-sitter is deliberately error-tolerant, and a file mid-edit should still yield a usable
  outline. The flag is load-bearing, not decorative: recovery is *lossy*, not merely incomplete — an
  unparseable declaration can swallow later valid ones, so `partial: true` is the only thing
  distinguishing "this file has two symbols" from "we lost some".
- **The never-fail-the-directory rule extends to I/O, not just parsing.** A file inside a `toc dir`
  listing that cannot be read (permissions, I/O error) or whose bytes are not valid UTF-8 is still
  listed, with an `"error"` key naming the failure and no `header`. It never aborts the directory,
  and it never sets `partial` — `partial` means "parsed, lossily", a different fact from "never
  read".
  In `toc file`, where that file *is* the whole request, the same failure is an `output.Err`.
- **A `.ts` or `.rs` file inside a `toc dir` listing is listed, with an `error` key.**
  It gets `path`, `language`, and `error: "language not yet supported by toc"`, and no `header`.
  It is **not** skipped silently, and it **does** count as a code file for the empty-directory
  question — a directory containing only `.ts` files returns a non-empty `files` array of error
  entries, not `files: []`.
  The directory's own `status` stays `found` (rank 0): an unimplemented language is a known, reported
  limitation, not a failure of the listing.
- **Empty and no-code-files directories return success.** `toc dir` on a directory containing no file
  with any of the five languages' extensions returns `{"ok":true,"files":[]}` and exit 0. An empty
  result is a true answer to "what code is in here".
- **No file-size cap.** Parse cost is linear and the runtime enforces its own work budgets, so no
  arbitrary byte threshold is imposed. A pathological file surfaces as an ordinary slow parse or as
  `partial: true`, never as a special-cased refusal.

### Batch mode: same shape, own driver, `path` as the per-entry key

- Both subcommands accept 2+ positional arguments and switch to the existing batch shape —
  `{"ok":true,"results":[…]}` with a per-entry `status`, and a process exit code set to the worst
  status across the batch.
- The per-entry identity key is **`"path"`**, not `"symbol"`.
  Because `runBatch` (`internal/cli/cli.go:909`) hard-codes `entry["symbol"] = arg` at line 918, toc
  gets **its own small driver** in `internal/cli` rather than reusing or generalizing `runBatch`.
  It reuses the existing `batchStatus` constants and the `statusRank` map (`internal/cli/cli.go:857`
  and `:867`) unchanged — the shared vocabulary is reused, only the entry-key line differs.
  Generalizing `runBatch` would change the output shape all four shipped verbs emit.
- Outcome → status mapping, covering every failure mode this design defines:

  | toc outcome | `status` | rank |
  | --- | --- | --- |
  | parsed, symbols/files returned | `found` | 0 |
  | parsed with recovery errors (`partial: true` on the entry) | `found` | 0 |
  | `toc dir` on a directory containing no supported files | `found` (empty `files`) | 0 |
  | path does not exist | `not_found` | 1 |
  | wrong path type for the subcommand | `error` | 3 |
  | extension maps to no supported language, or to a designed-but-unimplemented one (`.ts`, `.rs`) | `error` | 3 |
  | file unreadable, or invalid UTF-8 | `error` | 3 |

  `ambiguous` (rank 2) is never produced — there is no ambiguity state in a path-addressed lookup.
  **The rank column is batch-only.** In the single-argument shape, *every* toc failure — missing path,
  wrong path type, unsupported language, unreadable file — is `output.Err` and **exit 1**, with no
  exit 2 and no exit 3. This keeps toc inside the 0/1/2 single-arg contract
  `internal/cli/cli.go:13-21` already documents for every verb; rank 3 exists only to make "one entry
  genuinely failed" outrank "one entry was absent" when computing a batch's worst status.
  `partial: true` is a field on the entry and deliberately does **not** degrade the status: ranking it
  as a failure would poison the exit code of any batch containing one mid-edit file.

### Package layout: backend and orchestration split

- Two new packages.
  `internal/quarryengine/treesitter` is the parsing backend — grammar loading, parser construction,
  and the node-walking primitives. It imports the engine root only.
  `internal/quarryengine/toc` is the orchestration layer holding the per-language extraction
  strategies and the `TOCFile` / `TOCDir` entry points. It imports the root, `registry`, and
  `treesitter`.
  Both get rows in `layeringTable`.
- Rationale: this mirrors the existing `lsp` (backend) / `query` (orchestration) split exactly, and
  keeps the parser backend a separately swappable layer.

### Facade re-export

- `quarry/facade.go` gains `quarry.TOCFile` and `quarry.TOCDir` as one-line delegating functions,
  plus type aliases for the result types, plus the `ErrLanguageUnsupported` sentinel re-export.
- Rationale: `quarry/` is documented as the stable import path for the engine. A verb that skips the
  facade makes it incomplete.

## Technical context

### Existing structure mill-plan must work within

- The engine is a five-package DAG under `internal/quarryengine`: the root leaf (`errors.go`,
  `position.go`, `log.go`), `lsp`, `registry`, `daemon`, and `query`, re-exported by the thin
  `quarry/` facade. `internal/quarryengine/doc.go` documents the DAG in full.
- **Two guard tests will fail if the new packages are added carelessly:**
  - `internal/quarryengine/layering_test.go` walks every `.go` file under `internal/quarryengine/`,
    production *and* `_test.go`, and checks each file's intra-engine imports against `layeringTable`.
    Both new packages need rows there — one production row and one test row each, following the
    existing shape.
  - `internal/quarryengine/seam_enforcement_test.go` fails if any non-test file under
    `internal/quarryengine/` or `quarry/` imports `internal/output`, cobra, or any `internal/*cli`
    package. The new engine packages must return typed Go results and typed errors only.
- `internal/cli` is the sole place engine results become JSON, via `output.Ok` / `output.ErrFields`
  (`internal/output/output.go`).
- Every existing verb's `RunE` follows the same shape and the new ones should too:
  `CwdFrom(ctx)` → resolve target dir against the seam cwd → `resolveContext` → `buildOptions` →
  engine call → `SetExit(ctx, output.Ok(...))`.
  Note that `resolveContext` and `buildOptions` exist to serve the LSP verbs (they load the server
  registry and resolve a daemon state dir); `toc` needs neither a language server nor a state dir, so
  it should **not** be forced through them.
- `cli.GroupRunE` (`internal/cli/exec.go:227`) is the ready-made parent-command `RunE`: it prints
  help on a bare invocation and errors on an unknown subcommand.

### cgo API surface

`github.com/tree-sitter/go-tree-sitter` v0.25.0, confirmed by compiling against it:

- `ts.NewLanguage(tsgo.Language())`, where the grammar comes from that language's module under
  `<module>/bindings/go` — e.g. `tree-sitter-go/bindings/go`. Note
  `tstypescript.LanguageTypescript()` rather than a bare `Language()` for TypeScript, since that
  module ships both TypeScript and TSX.
- `ts.NewParser()`, `(*Parser).SetLanguage(lang) error`, `(*Parser).Parse(src, nil) *Tree`,
  `(*Tree).RootNode()`.
- `(*Node).Kind()`, `.NamedChildCount()`, `.ChildByFieldName(name)`, `.HasError()`, and the
  byte/point accessors the extraction rules need.
- **Both `*Parser` and `*Tree` own C memory and expose `Close()` — every parse must release both.**
  A leaked tree is a C allocation the Go GC will not reclaim.
- `Point{Row, Column}` is 0-based, so line numbers are `Row + 1`.
- The library ships outline/tagging helpers with an `Owner` field. **Do not use them** — owner
  resolution happens in our own walk, which exists anyway for docstring association.

There is no grammar-set choice and no build tags: each grammar is its own Go module, so quarry
imports exactly the five languages it supports and links nothing else.

### Per-language extraction shapes, confirmed by dumping real parse trees

These are the concrete node shapes the three implemented strategies must handle, and the survey
material for the `docs/toc-docstring-association.md` deliverable.

**Go** — declarations are top-level children of `source_file`.
- Kinds: `function_declaration`, `method_declaration`, `type_declaration`
  (the type's name is nested one level down in a `type_spec`, not on the declaration itself).
- Docstring: contiguous `comment` prev-siblings, walked backwards, stopping at the first blank line
  (i.e. when `prev.EndPoint().Row + 1 != cur.StartPoint().Row`).
- Signature: source from the declaration start to the body-bearing child's start byte.
- Owner: the receiver type on a `method_declaration`.

**Python** — the docstring is *not* a sibling.
- Kinds: `function_definition`, `class_definition`. Methods are `function_definition` nodes inside
  `class_definition` → `block`.
- Docstring: the first `string` node inside the definition's `block`; its text is in the
  `string_content` child, between `string_start` and `string_end` nodes.
- Module header: the first `string` node at file top-level, same shape.
- Consequence for ranges: because the docstring is inside the body, a Python symbol's range is the
  declaration node's own span — the docstring is already inside it. The `start` value needs no
  adjustment for Python, unlike Go and C#.

**C#** — sibling XML doc comments, with extra nesting.
- Kinds: `class_declaration`, `method_declaration`, plus interface/struct/record equivalents.
  Methods live inside `class_declaration` → `declaration_list`.
- Container nesting: `file_scoped_namespace_declaration` (a `namespace X;` statement, one flat level)
  and classic `namespace_declaration` (braced, adding a level) both need descending through.
- Docstring: `comment` prev-siblings whose text starts with `///`, containing XML tags to strip.
- Expression-bodied members (`=>`) have an `arrow_expression_clause` rather than a body block — the
  signature rule must handle both block-bodied and expression-bodied members.

**TypeScript and Rust** (designed, not implemented in this task):
TypeScript uses `/** … */` JSDoc block comments as prev-siblings; Rust uses `///` and `//!` line
doc-comments as prev-siblings, with `//!` being an *inner* doc comment that documents the enclosing
item rather than the following one — which is how Rust file headers are written, and a genuine trap
for the header rule.

## Constraints

There is no `CONSTRAINTS.md` at the hub root. Constraints established during discussion:

- **cgo is a build dependency, and it is deliberate.** After this task, building quarry requires
  `CGO_ENABLED=1` and a C toolchain; README must say so. Running the built binary requires nothing
  extra. **Do not remove cgo as a cleanup** — it is a decision, not an oversight
  (`discussion-meta.md` §1).
- **Windows remains a supported target and must not regress.** The build route changes (mingw-w64
  natively, or a cross-compile with `-extldflags "-static"`), not the support status.
- **darwin is already unbuildable** (`internal/proc` has no darwin implementation) and this task must
  not change that either way.
- **The two guard tests are non-negotiable.** New packages must be added to `layeringTable`, and no
  engine package may import `internal/output`, cobra, or `internal/cli`.
- **JSON only.** `output.Ok` / `output.Err` is the envelope for all verbs; `toc` introduces no second
  output format.
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
- a Python symbol — assert the range needs no docstring adjustment because the docstring is in-body;
- a multi-line function signature — assert the whole signature is returned, not just its first line;
- a Go `type X struct { … }` with fields — assert the signature is `type X struct` and the field body
  is **not** in the signature;
- a Go type alias with no body (`type ID string`) — assert the whole spec is the signature;
- a Go grouped `type ( … )` block — assert one symbol per `type_spec`, each with its own range and
  each signature carrying the `type` keyword.

**Header extraction (TDD).**
- a header separated from `package` by a blank line — assert it is still found;
- a file with both a file header and a package doc comment — assert the *first* block wins;
- a file with no header at all — assert the key is absent and the file is still returned.

Directive-block skipping, using real shapes from this tree:
- a Go file starting with `//go:build windows`, a blank line, then the header — assert the header is
  returned and the build constraint is not (fixture modelled on `internal/proc/proc_windows.go`);
- a Go file whose only leading block is a build constraint — assert no `header` key;
- a block mixing a `//go:generate` line with prose — assert it is treated as a header, not skipped;
- a Python file with a shebang then a module docstring — assert the docstring is the header;
- a Go file starting with `// Code generated by X. DO NOT EDIT.` then a real header — assert the
  banner is skipped as a header **and** `generated: true` is still set from it.

Truncation, one case per comment form, all asserting the same post-stripping blank-line rule:
a Go `//` block with a bare `//` separator line; a C# `///` block with a bare `///` separator line;
a Python module docstring with a blank line; and a header with no blank line at all — assert it is
returned whole.

**Ordering.** A file with several symbols — assert `symbols` is ascending by `start`.
A directory — assert `files` is lexicographic by filename, not filesystem order.

**Docstring text stripping (TDD).**
One test per language delimiter form: Go `//`, Python triple-quoted `string_content`, C# `///` plus
XML tag stripping. Include a C# case with `<summary>` and `<param>` to assert tags are removed and
their text kept.

**Test/generated classification (TDD).**
Assert the *omission* behaviour explicitly, not just the positive case: a C# file named
`FooTests.cs` must produce no `test` key at all, and a Go file named `foo_test.go` must produce
`test: true`. This is the rule most likely to rot into a best-effort `false`.

**Error tolerance.**
A file whose broken declaration swallows a later valid one. Assert `partial: true` is set and that
the surviving symbols are returned. This test documents that recovery is lossy.

**`internal/quarryengine/treesitter` — backend.**
Assert each of the five grammars loads non-nil and parses a trivial valid file with `HasError()`
false. This is the canary for a grammar-module version bump breaking an API.

Two backend-specific things the tests must cover:

- **Resource release.** `*Parser` and `*Tree` own C memory and expose `Close()`. Assert the
  extraction path releases both on every route, including the error and `partial` routes — a leaked
  tree is a C allocation the Go GC never reclaims, and `toc dir` over a large directory is where that
  compounds.
- **A build that actually links C.** The suite must fail loudly, not skip, if `CGO_ENABLED=0`;
  otherwise a cross-compile misconfiguration produces a green run against a binary that cannot be
  built at all.

**`internal/quarryengine/registry` — extension map.**
Table test over all five extensions plus an unknown one, plus the `--lang` override path.

**`internal/cli` — verb wiring.**
Follow the existing `cli_test.go` patterns.

- `toc file` on a directory and `toc dir` on a file — assert the hard error and that the message
  names the correct subcommand;
- a non-existent path — assert `output.Err` and exit 1;
- bare `quarry toc` — assert help is printed (the `GroupRunE` contract);
- batch mode with 2+ arguments — assert the per-entry key is `"path"` (not `"symbol"`), and assert
  the worst-status exit code across a batch mixing `found`, `not_found`, and `error`;
- a batch containing one `partial: true` file — assert the exit code is still 0;
- a `.ts` or `.rs` file passed to `toc file` — assert the explicit "not yet supported" error rather
  than an empty result;
- `--lang` with an unrecognised value — assert the error names the valid set;
- `--lang go` on a `.py` file — assert it parses with the Go grammar and does **not** error on the
  mismatch;
- `toc dir --lang rust` on a directory containing `.rs` files — assert `ok:true`, exit 0, and error
  entries per file, **not** a top-level error;
- `toc dir` on a directory with no supported files — assert `{"ok":true,"files":[]}` and exit 0;
- an unreadable or invalid-UTF-8 file inside `toc dir` — assert the file is still listed with an
  `error` key, no `header`, no `partial`, and that the rest of the directory is unaffected.

**Regression guard for the stale doc invariants.**
`quarry/facade_test.go`'s blank-identifier block is already a compile-time signature guard; adding
`TOCFile` and `TOCDir` entries there is what makes the facade re-export self-checking. Verify the
package builds after the `facade.go:8` count is rewritten — a stale count is a doc bug the compiler
cannot catch, so it needs a reviewer's eye rather than a test.

**Guard tests.**
`layering_test.go` and `seam_enforcement_test.go` must pass with only two kinds of change: the new
`layeringTable` rows, and raising `minPackageDirs` from 6 to 8 in both files (a *strengthening*, not
a relaxation — the constant is a floor, so leaving it at 6 would pass while quietly weakening the
guard). Do not relax either guard in any other way to make the new packages fit.

**Integration.**
One test running `toc file` against a real file in this repository (`internal/output/output.go` is a
good choice — small, stable, three well-documented symbols) asserting the full JSON envelope.


### From _mill/plan/00-overview.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
slug: "toc-verbs"
approved: true
started: "20260827-182202"
parent: "main"
root: ""
verify: go vet ./...
skip_checks: ["wiki-config-mutation"]
```

### From _mill/plan/01-treesitter-backend.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "treesitter-backend"
number: 1
cards: 7
verify: go test ./internal/quarryengine ./internal/quarryengine/treesitter ./internal/quarryengine/registry
depends-on: []
```



- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/cgoguard.go`
  - `internal/quarryengine/cgoguard_nocgo.go`
- **Deletes:** none
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:**
  - `internal/quarryengine/treesitter/treesitter.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/treesitter/treesitter_test.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/registry/extension.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/registry/extension_test.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/layering_test.go`
- **Creates:** none
- **Deletes:** none

### From _mill/plan/02-toc-scaffolding.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "toc-scaffolding"
number: 2
cards: 9
verify: go test ./internal/quarryengine ./internal/quarryengine/toc
depends-on: [1]
```



- **Edits:**
  - `internal/quarryengine/errors.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/doc.go`
  - `internal/quarryengine/toc/types.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/strategy.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/comments.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/sentences.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/classify.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/classify.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/comments_test.go`
  - `internal/quarryengine/toc/sentences_test.go`
  - `internal/quarryengine/toc/classify_test.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/layering_test.go`
  - `internal/quarryengine/seam_enforcement_test.go`
- **Creates:** none
- **Deletes:** none

### From _mill/plan/03-go-strategy.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "go-strategy"
number: 3
cards: 5
verify: go test ./internal/quarryengine/toc
depends-on: [2]
```



- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/nodes.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/golang.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/golang.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/golang.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/golang_test.go`
- **Deletes:** none

### From _mill/plan/04-python-csharp-strategies.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "python-csharp-strategies"
number: 4
cards: 6
verify: go test ./internal/quarryengine/toc
depends-on: [3]
```



- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/python.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/python.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/python_test.go`
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/csharp.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/csharp.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/csharp_test.go`
- **Deletes:** none

### From _mill/plan/05-toc-entry-points.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "toc-entry-points"
number: 5
cards: 6
verify: go test ./internal/quarryengine/toc
depends-on: [4]
```



- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/toc.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/toc.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/toc_test.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/toc_test.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/toc_integration_test.go`
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/toc/toc_test.go`
- **Creates:** none
- **Deletes:** none

### From _mill/plan/06-facade-and-cli.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "facade-and-cli"
number: 6
cards: 7
verify: go test ./internal/quarryengine/toc ./quarry ./internal/cli
depends-on: [5]
```



- **Edits:**
  - `quarry/facade.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `quarry/facade_test.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/cli/toc.go`
- **Deletes:** none
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/cli/toc_test.go`
- **Deletes:** none

### From _mill/plan/07-doc-sentences-config.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "doc-sentences-config"
number: 7
cards: 6
verify: go test ./internal/quarryengine/toc ./internal/cli
depends-on: [6]
```



- **Edits:**
  - `internal/cli/paths.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/cli/tocconfig.go`
- **Deletes:** none
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Edits:** none
- **Creates:**
  - `internal/cli/tocconfig_test.go`
- **Deletes:** none
- **Edits:**
  - `internal/cli/toc_test.go`
- **Creates:** none
- **Deletes:** none

### From _mill/plan/08-docs-and-sweep.md


```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "docs-and-sweep"
number: 8
cards: 8
verify: go test ./internal/quarryengine ./quarry ./internal/cli
depends-on: [7]
```



- **Edits:** none
- **Creates:**
  - `docs/toc-docstring-association.md`
- **Deletes:** none
- **Edits:**
  - `README.md`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/doc.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `quarry/facade.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `internal/quarryengine/seam_enforcement_test.go`
  - `internal/quarryengine/layering_test.go`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `mill-config.yaml`
- **Creates:** none
- **Deletes:** none
- **Edits:**
  - `quarry/facade_test.go`
  - `internal/quarryengine/layering_test.go`
  - `internal/quarryengine/seam_enforcement_test.go`
- **Creates:** none
- **Deletes:** none

## Conflicting files

- `README.md`
- `internal/quarryengine/doc.go`
- `quarry/facade.go`
- `quarry/facade_test.go`

## Instructions

For each file listed above:

1. Read the file and locate every conflict block (`<<<<<<<`, `=======`, `>>>>>>>`).
2. Understand both sides of the conflict — what each branch intended.
3. Write a resolution that preserves the intent of both sides.
   When both sides modify **different, non-overlapping parts** of the same conflict region — for example, different columns of one table row, different keys of one object, or disjoint lines of a prose block — **combine both edits** into a single resolved structure.
   Do NOT pick one side wholesale just because the region overlaps syntactically;
   picking one side wholesale is correct only when the two changes are genuinely mutually exclusive (e.g. the same key is renamed to two different values).
   Worked example: if `ours` changes column A and `theirs` changes column B of the same table row, the resolution keeps both column changes in a single row — it does not discard either.
4. Before keeping content from either side inside a conflict hunk, search the rest of the file (outside the hunk) for that same content.
   This judgment call is scoped narrowly — it applies only when a hunk's content might be a moved duplicate of content living elsewhere in the file;
   it does NOT apply to every ordinary step-3 disjoint-region combine (e.g. the column-A/column-B worked example above), which remains today's silent, high-confidence success path.
   Two branches:
   - **Confident case:** if the content clearly already exists elsewhere and the surrounding context makes it unambiguous that this is the same item having been moved (not two independent, separately-intended copies) — do not re-add it in the hunk;
     keep only the other side's unrelated edit.
     Worked example: one side moves a roadmap item from `## Planned` to `## Done`, while the other side makes an unrelated edit elsewhere in the file.
     The resolution keeps the item only under `## Done`;
     it is not re-added under `## Planned`.
   - **Ambiguous case:** if you cannot confidently tell whether this is the same moved content or a legitimate independent duplication — fall back to step 3's default (keep both) rather than guessing, and report the ambiguity via the `discarded` field (see Report section) with the description `"kept both sides of a conflict, ambiguous move-vs-duplicate"`.
     Worked example: a similarly-worded item appears in two different sections and you cannot tell whether it is the same item moved or a legitimate second, independently-added item.
     The resolution keeps both occurrences and reports the ambiguity via `discarded`.
5. Run `git -C /home/knatte/Code/quarry/wts/toc-verbs add <file>` to stage the resolved file.
6. For modify/delete (DU) conflicts: if Task intent above lists this file under a batch's `Deletes:`, run `git -C /home/knatte/Code/quarry/wts/toc-verbs rm <file>` instead of editing;
   that stages the intentional deletion.
7. For UD conflicts — files this branch **modified** that the parent branch **deleted**: do not silently keep the modification.
   Instead: a. Run `git log --diff-filter=D --oneline MERGE_HEAD -- <file>` to find the deletion commit on the parent. b. Run `git show <deletion-commit>` to inspect context. c. If the deletion commit message mentions a replacement file (e.g. "replaced by", "moved to", "consolidated into"),
   or the commit also adds a file in the same directory with overlapping content: stage the deletion — `git -C /home/knatte/Code/quarry/wts/toc-verbs rm <file>`. d. If detection is inconclusive: report `{"status":"stuck","stuck_type":"logic","reason":"modify/delete conflict on <file>: cannot determine if parent deletion is a replacement -- operator must decide"}` and halt.
   Do NOT silently keep the modification.
8. Before reporting `{"status":"success"}` (with or without `discarded`), re-read each file listed in Conflicting files in full and explicitly verify no contradictory losing-side claims survive the resolution — e.g. a stale value from one side of the conflict left alongside the correct value from the other side, or a claim that only made sense before the other side's edit was applied.
   If you find a contradiction you missed, fix it before reporting.
   If you find a contradiction you cannot confidently resolve, report `{"status":"stuck","stuck_type":"logic","reason":"self-verification found an unresolved contradiction in <file>: <description>"}` instead of `{"status":"success"}`.

Never use `git checkout --ours` or `git checkout --theirs` — they silently discard one side of the conflict.

## Report

Your last output line MUST be a bare JSON object (no code fence, no backticks):

On success (nothing discarded):

{"status":"success"}

On success with discarded content — if you had to drop content from one side (e.g. two sides made mutually exclusive changes and only one could survive), list each dropped item:

{"status":"success","discarded":["<short description of what was dropped from which side>"]}

An empty or absent `discarded` field means nothing was lost.
If anything was discarded, you MUST list it;
an empty list when content was actually dropped is a protocol violation. `discarded` also carries the step 4 ambiguous-case entry `"kept both sides of a conflict, ambiguous move-vs-duplicate"` — even though nothing was technically dropped in that case, the field's purpose is to surface anything the operator should double-check before `git merge --continue`, which covers both a genuine drop and a kept-both ambiguity.
The `mill-merge-in` frontend reads this field and surfaces any losses (or ambiguities) to the operator before continuing, rather than silently running `git merge --continue`.

If you cannot resolve one or more conflicts:

{"status":"stuck","stuck_type":"logic","reason":"<one-line description of what you could not resolve>"}

Anything other than this JSON object on the last line is a protocol violation;
the merge-in dispatcher treats that as stuck_type: logic with reason "no structured report" — your work is lost.
Do not wrap the JSON in a code fence;
do not add commentary after it.

## Tools

Available: Read, Edit, Write, Bash, Grep, Glob.
Use `git -C /home/knatte/Code/quarry/wts/toc-verbs` for any git commands;
do not `cd`.
Worktree cwd is `/home/knatte/Code/quarry/wts/toc-verbs`.
