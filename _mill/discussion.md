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
- A new Tree-sitter parsing backend package, `internal/quarryengine/treesitter`, using the official
  cgo bindings `github.com/tree-sitter/go-tree-sitter` plus one grammar module per language.
- README updates for the new build dependency: `CGO_ENABLED=1` and a C toolchain, plus the
  windows-native cross-compile recipe (see the parsing-backend Decision).
- A new orchestration package, `internal/quarryengine/toc`, holding the per-language extraction
  strategies and the two entry points.
- An extension-to-language map in `internal/quarryengine/registry`, since both new verbs are
  path-scoped and the existing `DetectLanguage` is directory-marker-scoped.
- Per-language extraction strategies **designed for all five languages** (Go, Python, C#,
  TypeScript, Rust) and **implemented and tested for Go, Python, and C#** in this task.
- Facade re-exports `quarry.TOCFile` and `quarry.TOCDir` in `quarry/facade.go`.
- New rows in `internal/quarryengine/layering_test.go`'s `layeringTable` for both new packages.
- **Exact-count and exact-claim invariants that go stale the moment this task lands, and must be
  updated in the same batch as the code that invalidates them.**
  **The invariant being defended, stated first, because the greps below serve it rather than
  define it:** *no prose anywhere in the tree may, after this task lands, still assert any of —*
  (a) *a count or enumeration of the engine's packages;* (b) *a count of quarry's verbs or of the
  facade's re-exports;* (c) *that quarry is LSP-only, or that it does not parse source itself;*
  (d) *that a batch entry is keyed on `symbol`;* (e) *that quarry builds without a C toolchain.*
  Clauses (c), (d), and (e) are not count claims and no grep phrase reliably finds them — verified
  examples the count-greps miss: `README.md:4` ("by speaking the Language Server Protocol to each
  language's own server rather than reimplementing a parser per language"), and
  `internal/cli/cli.go:23-29` (the package-level batch contract, "one JSON entry per symbol under a
  top-level \"results\" array", which toc's `path`-keyed driver contradicts).
  Anything matching that description is in scope whether or not the commands below happen to find
  it — a phrase grep is a search aid, not the specification.

  Two complementary commands, both actually run and their output read:

  ```
  # (a) counts and exact-number claims
  grep -rnE 'five-package|four verbs|29 identifiers|eight blank-identifier|LSP-backed|minPackageDirs|The six internal|import all four|seven re-exported' \
    --include='*.go' --include='*.md' . | grep -v '^\./_mill/' | grep -v '^\./\.scratch/'

  # (b) package-set enumerations, which (a) misses because they use no count word
  grep -rnE 'lsp, registry|registry, daemon|root leaf' \
    --include='*.go' --include='*.md' . | grep -v '^\./_mill/' | grep -v '^\./\.scratch/'
  ```

  Two hits are **explicitly out of scope**: `docs/scout-vs-grep.md:3` and `:130` use
  "LSP-backed" to describe a past measurement of `lyx scout`, not a current claim about what quarry
  is. Historical research documents are not updated when the product changes.
  Everything else both commands return is in scope:
  - `quarry/facade.go:8` — "It re-exports exactly the 29 identifiers this package exported before
    the engine-repackage move: no more, no less." Adding `TOCFile`, `TOCDir`, and the toc result
    type aliases breaks this sentence. Recount and rewrite it; do not leave a stale number.
  - `quarry/facade_test.go:103` — "The eight blank-identifier assignments below reference every
    delegating function in facade.go". That block is a compile-time signature guard, so `TOCFile`
    and `TOCDir` must each get an entry **and** the count in the comment must be updated.
  - `internal/cli/cli.go:7` — the package doc says "exposing four verbs" and enumerates them.
    Must account for the `toc` group.
  - `README.md:3` — "quarry is an LSP-backed code intelligence tool." This stops being true when a
    second, non-LSP parsing backend lands; the sentence must be reworded, not just the verb list
    below it extended. `README.md:8` — "quarry exposes four verbs".
  - `quarry/facade.go:3`, `internal/quarryengine/doc.go:44`, and
    `internal/quarryengine/seam_enforcement_test.go:10` — all three call the engine a
    "five-package DAG". It becomes seven packages (adding `treesitter` and `toc`).
  - `internal/quarryengine/doc.go:66` — "It imports all four packages above", describing `query`.
  - **Package-set enumerations found only by grep (b)**, each of which lists the DAG's members and
    must gain `treesitter` and `toc`:
    `quarry/facade.go:4` ("the root leaf package, lsp, registry, daemon, and query");
    `internal/quarryengine/doc.go:25` ("this package and its lsp, registry, daemon, and query
    subpackages") and `:37` (the seam-enforcement walk list);
    `internal/quarryengine/seam_enforcement_test.go:2-3` and `:101`;
    `internal/quarryengine/layering_test.go:159`.
  - `internal/quarryengine/doc.go:47-71` — the bulleted package-layout list, which describes each
    package and its allowed imports. It needs **two new bullets**, not just a count edit. This is
    the largest single doc change in the task and the one most likely to be skipped.
  - `internal/quarryengine/layering_test.go:20` — "The six internal/quarryengine/... import paths
    this guard reasons about" (becomes eight), and `:53` — "query's production files import all
    four".
  - `internal/quarryengine/layering_test.go:162` and
    `internal/quarryengine/seam_enforcement_test.go:104` — `const minPackageDirs = 6` in both.
    **These are floors (`< minPackageDirs` fails), so adding packages does not break the build** —
    which is exactly the risk: the guard silently loses strength while still passing.
    **The two constants do not mean the same thing, so their comment rewrites differ:**
    `layering_test.go` walks only `internal/quarryengine/` (6 dirs today, 8 after this task), so 8
    is the *exact* count there.
    `seam_enforcement_test.go` walks that tree **plus** `quarry/` (7 today, 9 after), and its
    comment at `:102-103` states the floor is deliberately set one below the real count. Raising it
    to 8 preserves that intentional slack rather than making it an exact-count claim.
    Raise both to 8; write the comments to say what each 8 actually is.
  - `quarry/facade_test.go:117` — "each of the seven re-exported sentinel error values". Becomes
    eight: toc adds `ErrLanguageUnsupported` (see the Emitted-schema Decision).
  - **Non-count claims, found by reading rather than by grep** (invariant clauses (c)–(e)):
    `README.md:4` — "by speaking the Language Server Protocol to each language's own server rather
    than reimplementing a parser per language". After this task quarry parses source itself.
    `internal/cli/cli.go:23-29` — the package doc's batch contract says "one JSON entry per symbol
    under a top-level \"results\" array"; toc's driver keys entries on `path`, so the contract must
    be restated to cover both.
    README's "Building and running" section — must gain the `CGO_ENABLED=1` / C-toolchain
    dependency and the windows cross-compile recipe.
    README's "Testing" section (`README.md:66-71`, two tiers today) — must reflect the cgo build.
- README "Building and running" update for the grammar-subset build tags (see the grammar-set
  Decision).
- README verb-list update.
- A per-language docstring-association survey at **`docs/toc-docstring-association.md`**, in the
  spirit of `docs/scout-multilang.md`. Its content is the per-language node-shape material recorded
  under "Technical context" below, written up as a standalone reference — including the two
  languages this task designs but does not implement.

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
- Caching or a daemon for the parser. See the "No parser daemon" Decision — this was measured at
  both file and directory level, not assumed.

## Decisions

### Parsing backend: official cgo Tree-sitter bindings

- Decision: use `github.com/tree-sitter/go-tree-sitter` (v0.25.0), the canonical cgo bindings for
  the upstream C runtime, with one grammar module per language —
  `tree-sitter-go`, `tree-sitter-python`, `tree-sitter-c-sharp`, `tree-sitter-typescript`,
  `tree-sitter-rust`. Each grammar is an ordinary Go module vendoring its generated C parser, so
  there is no external grammar build step and no build tags: you import the five you want.
  `CGO_ENABLED=1` and a C toolchain become build dependencies.
- **This reverses an earlier decision in this same discussion, on measured evidence.**
  The original choice was the pure-Go runtime `github.com/odvcencio/gotreesitter`, argued on the
  grounds that "the C runtime is ~3.9x faster on a full parse, but the parse cost is noise".
  That argument was wrong twice over, and the correction is recorded here rather than quietly
  dropped:
  1. The 3.9x figure came from the pure-Go project's own README — a vendor self-benchmark, never
     independently checked. Measured here, parsing is near-parity: **7.9 ms (cgo) vs ~10 ms
     (pure Go)** for `internal/cli/cli.go`, i.e. 1.3x, not 3.9x.
  2. Far more importantly, the comparison measured the wrong thing. The dominant cost is not
     parsing, it is **grammar load** — and there the gap is roughly 2500x. The original "113 ms
     including process startup" datum was itself almost entirely pure-Go grammar-load time, i.e. an
     artifact of the option being recommended, used to argue that the alternative's advantage did
     not matter.
- Measured during discussion, both libraries built and run against the same files on this tree:

  | | cgo (chosen) | pure Go (rejected) |
  | --- | --- | --- |
  | Grammar load, first call in process | **0.03 ms** | **70–86 ms** |
  | Each additional grammar, same process | ~0.03 ms | 7–9 ms (C#: 33.5 ms) |
  | Parse `internal/cli/cli.go` (38 KB) | 7.9 ms | ~10 ms |
  | **End to end, per `quarry toc` invocation** | **~0.5 ms** | **~80 ms** |
  | Binary, all five languages | 12.6 MB | 13.1 MB |
  | Build time | 6.8 s | ~2 s |

  All five cgo grammars were verified to load and parse a representative snippet with
  `hasError=false`: go, python, csharp, typescript, rust.
- **Binary size is a wash, and an earlier claim that cgo was "4x smaller" was wrong.** That 3.3 MB
  figure was a build with the Go grammar alone; with all five languages linked it is 12.6 MB against
  pure Go's 12.7–13.1 MB. Size does not favour either option.
- Rationale, now that the numbers are right: the choice came down to ~80 ms of unavoidable
  per-invocation startup against a C toolchain as a build dependency. quarry is built by its author
  for their own machines, so a build dependency is a one-time setup per machine rather than a
  distribution problem — and a documented C toolchain requirement is ordinary for Go projects (any
  project using `sqlite3` carries the same one). Against that, 80 ms is paid on every single call
  forever.
  Maturity reinforces it: the pure-Go library produced **two runtime panics** during this
  discussion — a mistyped `grammar_subset_*` build tag, and a missing external blob — where both
  should have been build-time failures. The cgo bindings have no build-tag surface to mistype.
- **Windows builds.** A C toolchain is needed to *build*, never to *run*. Two supported routes,
  both to be documented in README:
  - build natively on windows with mingw-w64 (MSYS2) or TDM-GCC;
  - cross-compile from linux or WSL2:
    `CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags '-extldflags "-static"' -o quarry.exe ./cmd/quarry`
    (requires `gcc-mingw-w64-x86-64`; `-static` is deliberate, so the `.exe` does not depend on
    `libwinpthread-1.dll` and friends sitting beside it).

  **Unverified:** the cross-compile command above was *not* run during discussion — no mingw-w64
  cross-compiler was installed on the discussion machine. It is the standard incantation, but the
  implementation batch must actually run it and adjust if needed, rather than copy it on trust.
  Note also that building inside WSL2 without `GOOS=windows` produces a *Linux* binary that runs in
  WSL2, not a windows-native `.exe`; both are legitimate targets but they are not the same artifact.
- Rejected: the pure-Go runtime `github.com/odvcencio/gotreesitter` (measured above — no size
  advantage, ~160x slower per invocation end to end, and two runtime panics in one afternoon where
  build-time failures were expected; its only real advantage is needing no C toolchain);
  a wazero/WASM wrapper such as `github.com/malivvan/tree-sitter` (cgo-free, but adds a WASM runtime
  and per-grammar `.wasm` loading — slower than native C for the same dependency burden);
  **shipping both backends behind `//go:build cgo` / `//go:build !cgo`** — see the next Decision,
  which rejects it on correctness grounds, not effort.
- Risk and its containment: cgo is a real constraint on how quarry is built, and it is now a
  documented one. The containment for changing course again is the package split below — the
  backend lives behind an interface in `internal/quarryengine/treesitter`, so swapping runtimes is
  one package, not a rewrite. That containment is what made this reversal cheap, and it is why it
  stays.

### One backend, not two: no cgo/pure-Go dual build

- Decision: exactly one parsing backend ships. quarry does **not** select between the cgo runtime
  and a pure-Go fallback via `//go:build cgo` / `//go:build !cgo`.
- Rationale: two independent implementations of Tree-sitter do not produce identical trees on all
  input. The pure-Go runtime self-reports three "degraded" grammars and has its own error-recovery
  behaviour, so `partial: true` and the set of symbols surviving a syntax error could differ between
  builds. `quarry toc` would then return different answers for the same file depending on how the
  binary was compiled — unacceptable for a tool whose entire value is that an agent can trust the
  result without verifying it. A slow outline is usable; an outline that is correct only for some
  build configurations is worse than none.
  Two further costs make it worse than it looks: the two libraries' node APIs are not
  interchangeable (pure Go takes the language as an argument — `node.Type(lang)`,
  `ChildByFieldName(name, lang)` — while cgo does not: `node.Kind()`, `ChildByFieldName(name)`), so
  a normalising adapter per library would be required around the single code path all extraction
  flows through; and every extraction test would have to run under both backends or one silently
  rots — the same failure mode round 4's review caught with build tags, except the untested artifact
  is an entire parser.
  Finally, `CGO_ENABLED` defaults to 0 when cross-compiling, so the fallback would be selected
  silently rather than deliberately.
- Rejected: dual backend selected by the `cgo` build constraint (silent divergence, doubled test
  matrix, adapter layer);
  dual backend with the active backend named in the JSON envelope (the only honest form of a dual
  build, since it makes divergence visible — but it is complexity for a tool with one user, and it
  pushes the divergence problem onto the consumer instead of solving it).

### No parser daemon, and why the LSP daemon is not a precedent

- Decision: `toc` spawns no daemon, shares no state between calls, and caches nothing.
  Every invocation loads its grammar, parses, and exits.
- Measured during discussion, on this tree:

  | Scope | Cost |
  | --- | --- |
  | First grammar loaded in a process | 70–86 ms (one-time process init) |
  | Each additional grammar in the same process | 7–9 ms (C#: 33.5 ms) |
  | Re-requesting an already-loaded grammar | 0.0 ms |
  | `toc file` on 1.7 KB | 0.08 s wall, end to end |
  | `toc file` on 38 KB (`internal/cli/cli.go`) | 0.10 s wall, end to end |
  | `toc dir internal/quarryengine/daemon` (14 files, 115 KB) | 0.16 s wall; 72 ms of parsing |
  | `toc dir internal/cli` (9 files, 111 KB) | 94 ms of parsing, 10.4 ms/file |

  Per-file parse cost is 5–10 ms and grammar load is paid **once per process**, so a directory's
  cost is `~75 ms + N × ~7 ms`. A 23x larger file costs 20 ms more: the fixed startup dominates,
  not the file size.
- **The LSP daemon is not a precedent for a toc daemon**, despite the infrastructure existing.
  `EnsureServer` exists because gopls indexes an entire module, holds type-checked packages in
  memory, and has a cold start measured in seconds to tens of seconds — persistent state is the
  entire point. Tree-sitter has none of that: no project indexing, no cross-file state, and a
  stateless per-file parse.
  Nor is the machinery reusable. `EnsureServer` spawns a *language server speaking LSP*, wrapped in
  `daemonstate`, `probe`, a spawn-race lock, and toolchain management. A toc daemon would be quarry
  talking to itself over a new protocol, with a new state file, a new lifecycle, and a new lock —
  a second daemon inheriting the first one's problems, including the platform split README already
  documents (the supervised strategy hard-codes a Unix domain socket, so windows falls back).
- What a daemon would actually buy is the ~75 ms fixed init. That is 1–3% of one LLM agent turn,
  bought with shared state, mtime-based cache invalidation, and a platform split — for a verb whose
  entire value is being trustworthy.
- **Batch mode is the cheap version of the same win.** `toc file a.go b.go c.go` pays the 75 ms once
  and ~7 ms per additional file in one process, with no shared state and no invalidation problem.
  The batch-mode Decision was taken for CLI consistency; these numbers are its second, independent
  justification.
- Rejected: a toc daemon reusing `EnsureServer` (wrong shape — that seam launches LSP servers);
  a new toc-specific daemon (new protocol, state file, lock, and platform split to save 75 ms);
  an in-process parse-tree cache keyed on mtime (no second call exists to hit it — each CLI
  invocation is a fresh process).

### Verb shape: a `toc` command group, not two flat verbs

- Decision: `quarry toc file <path>` and `quarry toc dir <path>`, as a cobra parent command with
  `RunE: cli.GroupRunE` and two subcommands. Bare `quarry toc` prints help.
- Rationale: the two verbs return different shapes and must stay distinct, but `toc` / `toc-dir`
  is asymmetric naming for two siblings. The group form is symmetric, gives one help tree, and uses
  machinery that already exists and is already tested — `GroupRunE` is documented at
  `internal/cli/exec.go:222` as "the RunE for parent module group commands (e.g. `quarry lang`)".
- Rejected: `outline <file>` + `manifest <dir>` (two flat verbs matching the existing four, but two
  unrelated names to learn); `toc <file>` + `survey <dir>` (still asymmetric).

### Emitted schema: the closed key set for both verbs

- Decision: the exact keys each verb emits are fixed here, so the plan writer invents none.

  **`toc file <path>`** — envelope fields:
  `header` (string, omitted when absent), `language` (string, one of the five canonical names),
  `symbols` (array), `partial` (bool, omitted when false).

  **Each entry in `symbols`:**
  `kind` (string, closed vocabulary below), `name` (bare identifier),
  `owner` (string, omitted when the symbol has no owner), `signature` (string),
  `docstring` (string, omitted when absent), `start` (int), `end` (int).

  **`toc dir <path>`** — envelope field `files` (array). **Each entry:**
  `path` (string, see the path-form rule below), `language` (string),
  `header` (string, omitted when absent), `partial` (bool, omitted when false),
  `test` (bool, omitted when the language has no reliable rule),
  `generated` (bool, same omission rule), `error` (string, present only on a per-file
  failure, and mutually exclusive with both `header` and `partial`).

  `partial` **is** in the `toc dir` key set: `toc dir` parses each file to extract its header, so a
  file whose parse reports `HasError()` is exactly as lossy here as it is in `toc file`, and the
  header it yielded may be wrong or missing for that reason. The consumer needs the same warning in
  both verbs. `partial` and `error` are mutually exclusive by construction — `partial` means
  "parsed, lossily", `error` means "never parsed at all".

- **Ordering is specified, not incidental.**
  `symbols` is in **source order**, ascending by `start`.
  `files` is in **lexicographic order by filename**, explicitly sorted — not the order the OS
  returns directory entries in, which is not guaranteed stable across filesystems.
  Both orders are part of the contract: the integration test asserts a full JSON envelope, which is
  unwritable against an unspecified order, and a caller diffing two `toc` runs deserves a stable
  result.
- **toc adds exactly one engine sentinel: `ErrLanguageUnsupported`.**
  It is returned by `internal/quarryengine/toc` when a resolved extension maps to no language, or
  to one designed but not yet implemented, and it is re-exported through `quarry/facade.go` like
  every other sentinel. `internal/cli` classifies it with `errors.Is`, matching how the existing
  verbs classify (`internal/cli/cli.go:875-906`).
  Everything else stays CLI-side: path existence and path-type validation happen in `internal/cli`
  before the engine is called at all (they are `os.Stat` concerns the CLI already owns), and read
  failures surface as wrapped `os` errors with no sentinel of their own.
  Consequence: `quarry/facade_test.go:117`'s "seven re-exported sentinel error values" becomes
  eight, and `TestFacadeSentinels_Identity` gains a row. This resolves the "check rather than
  assume" note in the Scope inventory — it is now checked and decided.
- **The `kind` vocabulary is closed and shared across all five languages: `function`, `method`,
  `type`.**
  A Python `class_definition`, a C# `class_declaration` / `interface_declaration` /
  `record_declaration` / `struct_declaration`, and a Go `type_spec` all emit `kind: "type"`.
  A function with an owner (Go method receiver, Python/C# member) emits `kind: "method"`; one
  without emits `kind: "function"`.
- Rationale: three kinds is exactly the set the brief asks for, and it is the set for which
  "signature + docstring tells you what it is for" holds uniformly. A richer vocabulary
  (`class` vs `interface` vs `record` vs `struct`) would be per-language noise that the verbatim
  signature already shows on the very next field — the consumer reads `public interface IFoo` and
  knows, without a redundant `kind: "interface"` costing a token on every entry.
  Fixing the key set here also prevents the plan writer from silently choosing a different one.
- Rejected: a per-language `kind` vocabulary mirroring the grammar node types (breaks the flat,
  uniform shape the output decision committed to, and makes a consumer learn five vocabularies);
  omitting `kind` entirely (the consumer cannot cheaply separate types from functions);
  a `language` field per symbol rather than per file (a file has exactly one language).

### Path handling: seam-relative resolution, round-trippable output

- Decision: **relative positional arguments are joined against the seam cwd**, `CwdFrom(ctx)`
  (`internal/cli/cwdcontext.go:41`), never passed to `os.Stat` raw.
  This mirrors what every existing verb's `RunE` already does when defaulting `--target-dir`, and
  it is what makes `RunCLIIn`/`WithCwd` — the seam `internal/cli`'s own tests depend on — actually
  govern toc. A raw `os.Stat(arg)` would silently use the process cwd and bypass it.
- **`toc dir` emits each file's `path` as the directory argument *as the caller wrote it*, joined
  with the file's name.**
  So `quarry toc dir internal/cli` yields `internal/cli/exec.go`, which the agent can paste
  straight into `quarry toc file internal/cli/exec.go` and have it work from the same cwd.
- **The batch `"path"` key echoes the positional argument verbatim**, exactly as `runBatch` echoes
  `arg` today (`internal/cli/cli.go:918`) — not the absolutized form.
- Rationale: these paths exist to be fed back into another quarry call by an agent. Absolutizing
  them would make every entry longer and pin the output to one machine's layout; emitting bare
  basenames would make them unusable as `toc file` arguments unless the agent reconstructs the
  join itself. Echoing the caller's own form round-trips in the caller's own frame of reference.
- Rejected: absolute paths (verbose, machine-pinned, and not what the caller typed);
  bare basenames (not directly reusable as a `toc file` argument);
  paths relative to the repo root (there is no repo-root concept in toc — it never detects a project
  the way the LSP verbs do).

### Path-type validation

- Decision: `os.Stat` the positional argument first (after the seam-cwd join above).
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
- **`--lang` for toc is validated against toc's own vocabulary, not the server registry.**
  The existing verbs' `--lang` is a *registry key*, looked up inside `registry.DetectLanguage`
  against the registry that `resolveContext` loaded from servers.yaml
  (`internal/cli/cli.go:153`, flag help at `:205`). toc deliberately does not go through
  `resolveContext` — it needs no language server and no state dir — so it cannot and must not reuse
  that validation path.
  toc's `--lang` accepts exactly the canonical language names its own extension map defines
  (`go`, `python`, `csharp`, `typescript`, `rust`). An unrecognised value is an `output.Err` naming
  the valid set. A recognised-but-unimplemented value (`typescript`, `rust`) gets the same
  "not yet supported by toc" error the extension path gives.
- **`--lang` overrides the extension outright; a mismatch is not an error.**
  `quarry toc file --lang go foo.py` parses `foo.py` with the Go grammar and will almost certainly
  come back with `partial: true` and few or no symbols. This is deliberate and matches what
  `--lang` means on every existing verb: an explicit operator override that wins over detection.
  Erroring on mismatch would make the flag useless for its actual purpose — a file with a
  non-standard extension.
  For `toc dir`, `--lang` restricts the listing to that language's extensions instead of listing
  every supported language found.
- Rejected: reusing `DetectLanguage` on the file's parent directory (wrong for mixed-language
  trees); sniffing file content (unnecessary — the extension is definitive for all five languages);
  validating toc's `--lang` against the servers.yaml registry (couples a parser-only verb to the
  language-server config it has no other reason to load);
  erroring on an extension/`--lang` mismatch (defeats the override's purpose).

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

### Signature: exact source text, cut at the body-bearing child

- Decision: the signature is the verbatim source text from the declaration's first byte up to the
  start of that declaration's **body-bearing child node**, trimmed.
  No normalization, no reformatting, no synthesized canonical form, and **never a "first line only"
  truncation**.
  "Body-bearing child" is resolved per kind, and every implemented kind must have an explicit rule:
  - Go function / method: `ChildByFieldName("body")` (a `block`).
  - Go type declaration: the type's own body node — `field_declaration_list` for a `struct_type`,
    the interface body for an `interface_type`. So `type FileLock struct` is the signature, not the
    whole struct with its fields.
  - Go type alias / defined type with no body (`type ID string`): no body-bearing child exists, so
    the signature is the whole `type_spec` text. This is short by construction.
  - Python: `ChildByFieldName("body")` (a `block`), for both `function_definition` and
    `class_definition`.
  - C# block-bodied member: the `declaration_list` (for a type) or the method's `block`.
  - C# expression-bodied member (`=>`): the `arrow_expression_clause`, so `public void Resize(double
    factor)` is the signature and the expression is excluded.
- **The Go type symbol is always the `type_spec`, in both the grouped and ungrouped forms** — one
  rule, no branch on which shape the source used. But the *emitted* signature and range are
  computed from the enclosing `type_declaration` when that declaration holds exactly one spec:
  - Ungrouped `// Doc.` + `type FileLock struct { … }`: one symbol. Its docstring is the
    `type_declaration`'s contiguous `comment` prev-siblings (the ordinary rule — the comment
    precedes the declaration, not the spec). Its `start` is the docstring's first line and its
    signature is `type FileLock struct`, **including the `type` keyword**, because the signature is
    cut from the declaration's first byte, not the spec's. Emitting `FileLock struct` would be
    invalid Go and useless to paste anywhere.
  - Grouped `type ( … )`: one symbol per spec. Each spec's signature is rendered with the `type`
    keyword prepended so the two forms produce identical output for identical types, and each
    spec's docstring is its own preceding comment block inside the group. A comment attached to the
    `type (` line documents the group, not any one spec, and is dropped.
  This resolves what would otherwise be a contradiction between this Decision and the Go notes in
  Technical context: the walked node is the `type_declaration`, the emitted symbol unit is the
  `type_spec`, and the signature always carries the keyword.
- Rationale: it is what the parser already gives, it is guaranteed correct because no transformation
  can introduce drift, and an LLM reads native source syntax at least as well as an invented normal
  form. A normalizer would be five per-language formatters to write, test, and keep correct.
  The per-kind body rule is what makes the "no normalization" rule actually implementable: a Go
  `type_declaration` has no `body` field at all, so a naive `ChildByFieldName("body")` returns nil
  and the signature silently becomes the entire struct body — the exact token blowup this verb
  exists to prevent. The discussion spike papered over this with a first-line-only cut, which is a
  hack that breaks on any multi-line function signature.
- Rejected: a normalized `name + param types + return type` rendering (uniform across languages,
  but five formatters' worth of maintenance and five ways to be subtly wrong);
  cutting at the first `\n` (loses everything after the first line of a multi-line signature; this
  tree happens to contain no multi-line function signature today, so the hack would pass its own
  tests here and break on the first target repo that has one — which is precisely why the rule must
  be structural rather than line-based);
  cutting at the first `{` byte (wrong for a generic constraint or a map literal in a default).

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

- Decision: the file header is the first **non-directive** comment block in the file, and the rule
  tolerates a blank line between that block and the following declaration.
  This is a different rule from docstring association, which requires strict adjacency.
- **Directive-only leading blocks are skipped, and the next block is taken.**
  A leading comment block is classified as a directive block — and skipped — when *every* line in it,
  after delimiter stripping, matches a known directive form:
  - Go: `go:build`, `+build`, `go:generate`, `go:embed`, `nolint`, and the generated-file banner
    `Code generated ... DO NOT EDIT.`.
    The banner is a directive block for header purposes — it is the same class of non-purpose noise
    as a build constraint, and emitting "Code generated by protoc. DO NOT EDIT." as a file's stated
    purpose would be exactly the failure the `//go:build windows` rule exists to prevent.
    The same block is still read by the `generated` detection rule; being skipped as a header and
    being consumed as a marker are independent.
    C#'s `<auto-generated>` block is treated identically.
  - Python: a `#!` shebang on line 1, and a PEP 263 coding line (`coding[:=]`) on line 1 or 2.
  - C#, TypeScript, Rust: preprocessor and attribute forms (`#pragma`, `#nullable`, `#!`) are not
    `comment` nodes in these grammars, so they never reach this rule. Rust's `//!` inner doc comment
    is a *header*, not a directive — see the per-language notes.

  A block mixing directive and prose lines is **not** a directive block and is taken as the header.
  If every leading block is a directive block, the file has no header and the key is omitted.
- Why this is not a hypothetical: verified in this tree. `internal/proc/proc_windows.go:1` is
  `//go:build windows`, then a blank line, then the real header at `:3`. Same shape in
  `internal/quarryengine/query/refs_integration_test.go:1` (`//go:build lsp`),
  `daemon/supervised_lsp_test.go:1`, and `daemon/ensureserver_integration_test.go:1`.
  Without this rule, `quarry toc dir internal/proc` would emit `header: "go:build windows"` — the
  build constraint presented as the file's purpose, which is worse than emitting no header at all.
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
  Each file's header is truncated to its first paragraph, where **"first paragraph" is defined
  after delimiter stripping, not before**: strip the language's comment or string delimiters per the
  docstring-text rule, then cut at the first blank line in the resulting prose.
  This one rule covers every comment form without a per-form special case, because a bare `//` line
  in a Go block, a bare `///` line in a C# block, and a blank line inside a Python module docstring
  all strip to the same empty line. A header with no blank line is emitted whole.
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
- **The same never-fail-the-directory rule extends to I/O, not just to parsing.** A file inside a
  `toc dir` listing that cannot be read (permissions, I/O error) or whose bytes are not valid UTF-8
  is still listed, with an `"error"` key naming the failure and no `header`. It never aborts the
  directory, and it never sets `partial` — `partial` means "parsed, lossily", which is a different
  fact from "never read".
  In `toc file`, where that file *is* the whole request, the same failure is an `output.Err` (exit 1
  single-arg, `status: "error"` rank 3 in batch), because there is no partial answer to give.
- **A `.ts` or `.rs` file inside a `toc dir` listing is listed, with an `error` key.**
  It gets `path`, `language` (`typescript` / `rust`), and
  `error: "language not yet supported by toc"`, and no `header`.
  It is **not** skipped silently, and it **does** count as a supported extension for the
  empty-directory question — a directory containing only `.ts` files returns a non-empty `files`
  array of error entries, not `files: []`.
  This is the same principle as the header-absent rule: a file the agent might need to know about
  is never dropped from a directory overview. Silently skipping would tell the agent the directory
  contains no code, which is false.
  The directory's own `status` stays `found` (rank 0) — an unimplemented language is a known,
  reported limitation, not a failure of the directory listing. The rank-3 `error` row in the batch
  table applies to a `.ts` path passed **directly** to `toc file` or `toc dir`, where the
  unsupported language is the entire request.
- **Empty and no-code-files directories return success, not an error.** `toc dir` on a
  directory containing no file with any of the five languages' extensions returns
  `{"ok":true,"files":[]}` and exit 0. An empty result is a true answer to "what code is in here",
  not a failure.
- **No file-size cap.** Tree-sitter's parse cost is linear and the runtime enforces its own
  work budgets, so no arbitrary byte threshold is imposed. A pathological file surfaces as an
  ordinary slow parse or as `partial: true`, never as a special-cased refusal.
- Rejected: hard-failing the file (loses the directory over one bad file);
  returning recovered symbols with no marker (the consumer cannot know the list is incomplete);
  reusing `partial` for unreadable files (conflates "lossy parse" with "no parse");
  erroring on an empty directory (an empty listing is a valid answer);
  a size cap (invents a threshold with no evidence any real file needs one).

### Batch mode: same shape, own driver, `path` as the per-entry key

- Decision: both subcommands accept 2+ positional arguments and switch to the existing batch shape —
  `{"ok":true,"results":[…]}` with a per-entry `status`, and a process exit code set to the worst
  status across the batch.
  The per-entry identity key is **`"path"`**, not `"symbol"`.
  Because `runBatch` (`internal/cli/cli.go:909`) hard-codes `entry["symbol"] = arg` at line 918,
  toc gets **its own small driver** in `internal/cli` rather than reusing or generalizing `runBatch`.
  It reuses the existing `batchStatus` constants and the `statusRank` map (`internal/cli/cli.go:857`
  and `:867`) unchanged — the shared vocabulary is reused, only the entry-key line differs.
- Explicit outcome → status mapping, covering every failure mode this design defines:

  | toc outcome | `status` | rank |
  | --- | --- | --- |
  | parsed, symbols/files returned | `found` | 0 |
  | parsed with recovery errors (`partial: true` on the entry) | `found` | 0 |
  | `toc dir` on a directory containing no supported files | `found` (empty `files`) | 0 |
  | path does not exist | `not_found` | 1 |
  | wrong path type for the subcommand | `error` | 3 |
  | extension maps to no supported language, or to a designed-but-unimplemented one (`.ts`, `.rs`) | `error` | 3 |
  | file unreadable, or invalid UTF-8 | `error` | 3 |

  `ambiguous` (rank 2) is never produced by either toc subcommand — there is no ambiguity state in a
  path-addressed lookup.
  **The rank column is batch-only.** In the single-argument shape, *every* toc failure — missing
  path, wrong path type, unsupported language, unreadable file — is `output.Err` and **exit 1**,
  with no exit 2 and no exit 3. This keeps toc inside the 0/1/2 single-arg contract
  `internal/cli/cli.go:13-21` already documents for every verb, while rank 3 exists only to make
  "one entry genuinely failed" outrank "one entry was absent" when computing a batch's worst status.
  `partial: true` is a field on the entry and deliberately does **not** degrade the status: a partial
  outline is a usable answer, and ranking it as a failure would poison the exit code of any batch
  containing one mid-edit file.
- Rationale: an agent frequently wants several files in one call, and every existing verb offers
  this. But `"symbol": "internal/foo/bar.go"` would be an actively wrong label for a path, and
  generalizing `runBatch` to take a key name would change the shape all four existing verbs emit —
  a breaking output change to shipped verbs, for a task that is supposed to add verbs, not alter
  them.
- Rejected: reusing `runBatch` as-is (mislabels paths as symbols);
  generalizing `runBatch`'s key (breaks the four existing verbs' output shape);
  exactly-one-argument (simpler, but breaks a pattern all four existing verbs follow);
  ranking `partial` as its own status (invents a fifth vocabulary member and pollutes batch exit
  codes).

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

Both candidate backends were built and run — this design is verified, not assumed.
The extraction spike (docstring walk, signature cut, ranges) was written against the pure-Go
runtime before the backend reversal; its **findings about source structure remain valid**, since
they are facts about Go/Python/C# syntax trees rather than about either library. Only the API names
differ, and the cgo equivalents are listed below.

| Test | Result |
| --- | --- |
| `internal/output/output.go` | 3 symbols, correct signature, docstring, and range spanning docstring + declaration |
| `internal/cli/cli.go` (38.8 KB) | 25 symbols, 10.9 KB of JSON — a 3.5x reduction against reading the file |
| `internal/lock/lock.go` | correctly distinguishes `function`, `method` (receiver), and `type` |
| Deliberately broken Go file | `partial: true` set; symbols before the error preserved; a symbol *after* the error was lost |
| All five cgo grammars | go, python, csharp, typescript, rust each load and parse with `hasError=false` |
| cgo binary, all five languages | 12.6 MB |

Timing measurements live in the "No parser daemon" and "Parsing backend" Decisions, which are where
they drive a choice.

**cgo API surface** (`github.com/tree-sitter/go-tree-sitter` v0.25.0), confirmed by compiling
against it:
`ts.NewLanguage(tsgo.Language())` where the grammar comes from that language's module under
`<module>/bindings/go` — e.g. `tree-sitter-go/bindings/go`, and note
`tstypescript.LanguageTypescript()` rather than a bare `Language()` for TypeScript, since that
module ships both TypeScript and TSX;
`ts.NewParser()`, `(*Parser).SetLanguage(lang) error`, `(*Parser).Parse(src, nil) *Tree`,
`(*Tree).RootNode()`;
`(*Node).Kind()`, `.NamedChildCount()`, `.ChildByFieldName(name)`, `.HasError()`, and the byte/point
accessors the extraction rules need.
Both `*Parser` and `*Tree` own C memory and expose `Close()` — **every parse must release both**,
which the pure-Go library did not require. This is a real difference the implementation must honour;
a leaked tree is a C allocation the Go GC will not reclaim.
`Point{Row, Column}` is 0-based in both libraries, so line numbers are `Row + 1`.

The library also ships helpers for outline/tagging.
**Decision: do not use them.** Owner resolution is done in our own walk, for the same reason it was
under the previous backend: an outline helper does not do docstring association, so the
sibling/in-body walk is ours regardless — and by the time that walk holds the declaration node, the
receiver or enclosing class is one `ChildByFieldName` away.

There is also a runtime `GOTREESITTER_GRAMMAR_SET` environment variable, which restricts *loading*
and has no effect on binary size.
Binary sizes for every option were measured during discussion — see the "Grammar set" Decision for
the table and the choice. Nothing here is left to implementation time.

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

- **cgo is now a build dependency, and this is a deliberate, documented change.** `go.mod` is pure
  Go today (cobra, yaml.v3, flock); after this task, building quarry requires `CGO_ENABLED=1` and a
  C toolchain. README must say so. Running the built binary requires nothing extra.
  This constraint was *reversed* during discussion on measured evidence — see the parsing-backend
  Decision. Do not "restore" pure-Go-ness as a cleanup; it is a decision, not an oversight.
- **Windows remains a supported target and must not regress.** The build route changes (mingw-w64
  natively, or a cross-compile with `-extldflags "-static"`), not the support status.
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
- a Python symbol — assert the range needs no docstring adjustment because the docstring is in-body;
- a multi-line function signature — assert the whole signature is returned, not just its first line
  (this is the regression the spike's first-line hack would have shipped);
- a Go `type X struct { … }` with fields — assert the signature is `type X struct` and the field
  body is **not** in the signature;
- a Go type alias with no body (`type ID string`) — assert the whole spec is the signature;
- a Go grouped `type ( … )` block — assert one symbol per `type_spec`, each with its own range.

**Header extraction (TDD).**
Explicitly cover the two shapes the spike found in this repo:
- a header separated from `package` by a blank line — assert it is still found;
- a file with both a file header and a package doc comment — assert the *first* block wins.
Plus: a file with no header at all — assert the key is absent and the file is still returned.
Directive-block skipping, using real shapes from this tree:
a Go file starting with `//go:build windows`, a blank line, then the header — assert the header is
returned and the build constraint is not (fixture modelled on `internal/proc/proc_windows.go`);
a Go file whose only leading block is a build constraint — assert no `header` key;
a block mixing a `//go:generate` line with prose — assert it is treated as a header, not skipped;
a Python file with a shebang then a module docstring — assert the docstring is the header;
a Go file starting with `// Code generated by X. DO NOT EDIT.` then a real header — assert the
banner is skipped as a header **and** `generated: true` is still set from it.
Ordering: a file with several symbols — assert `symbols` is ascending by `start`; a directory —
assert `files` is lexicographic by filename, not filesystem order.
Truncation, one case per comment form, all asserting the same post-stripping blank-line rule:
a Go `//` block with a bare `//` separator line; a C# `///` block with a bare `///` separator line;
a Python module docstring with a blank line; and a header with no blank line at all — assert it is
returned whole.

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
Thin: assert each of the five grammars loads non-nil and parses a trivial valid file with
`HasError()` false. This is the canary for a grammar-module version bump breaking an API.

Under the cgo bindings there is no build-tag surface, so the round-4 concern about a shipped
configuration going untested does not arise: `go test ./...` exercises exactly the code that ships.
The build-tag double-run this section previously prescribed is obsolete and must not be carried
into the plan.

Two backend-specific things the tests must cover, both new with cgo and neither present under the
previous runtime:
- **Resource release.** `*Parser` and `*Tree` own C memory and expose `Close()`. Assert the
  extraction path releases both on every route, including the error and `partial` routes — a leaked
  tree is a C allocation the Go GC never reclaims, and `toc dir` over a large directory is exactly
  where that compounds.
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
- a batch containing one `partial: true` file — assert the exit code is still 0, i.e. `partial` does
  not degrade batch status;
- a `.ts` or `.rs` file — assert the explicit "not yet supported" error rather than an empty result;
- `--lang` with an unrecognised value — assert the error names the valid set;
- `--lang go` on a `.py` file — assert it parses with the Go grammar and does **not** error on the
  mismatch;
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
This is the end-to-end proof the spike demonstrated by hand.

## Q&A log

- **Q:** Is C faster, and does that matter? **A:** *This answer was wrong and was later reversed — kept here because the reversal is the point.* The 3.9x-on-parse figure was the pure-Go project's self-reported benchmark, never checked; measured, parsing is 1.3x. And parsing was the wrong thing to measure: grammar load is the dominant cost, and there cgo is ~2500x faster (0.03 ms vs 70–86 ms), making it ~160x faster end to end per invocation. See the parsing-backend Decision.
- **Q:** Is a C toolchain acceptable as a build dependency? **A:** Yes — quarry is built by its author for their own machines, and a documented C toolchain is ordinary for Go projects. Running the binary needs nothing extra.
- **Q:** Can we ship two backends, cgo on POSIX and pure Go for windows? **A:** No. **Why:** two independent Tree-sitter implementations do not produce identical trees, so `quarry toc` would give different answers depending on how the binary was built — fatal for a tool whose value is that the agent need not verify the result. See the "One backend, not two" Decision.
- **Q:** Once the binary is built, does it run on windows? **A:** Not a linux-built one — Go binaries are per-OS. A windows `.exe` needs `GOOS=windows` plus a mingw-w64 cross-compiler, or a native windows build. Building in WSL2 without `GOOS=windows` yields a linux binary that runs in WSL2, which is a legitimate but different target.
- **Q:** Does Tree-sitter support all languages regardless of binding? **A:** Yes — grammars are per-language parse tables, and both the cgo bindings and the pure-Go runtime cover all five of quarry's languages. Language coverage does not separate the options.
- **Q:** One `toc <path>` verb dispatching on stat, or two? **A:** Two, because the output shapes differ — but as `toc file` / `toc dir` subcommands rather than the asymmetric `toc` / `toc-dir`.
- **Q:** What if the caller passes the wrong path type? **A:** Stat first, validate against the subcommand's expected type, hard-fail on mismatch with a message naming the correct subcommand.
- **Q:** Should `toc dir` recurse? **A:** No. Explicitly non-recursive.
- **Q:** Should the docstring be part of `toc file`'s output, and should the line range cover it? **A:** Yes to both — this is the core premise, not an option. Signature + docstring is what tells a reader what a method is for.
- **Q:** Should `toc dir` list only the detected language's files, or all languages? **A:** All supported languages, since a directory is not guaranteed single-language. Go first, Python and C# in the same task, TypeScript and Rust deferred.
- **Q:** How can a test file even be flagged, given Go has `_test.go` but C# has nothing? **A:** It cannot, uniformly. Emit the key only where a reliable rule exists; omit it entirely for C# and Rust rather than emitting a false `test: false`.
- **Q:** Is the output format even JSON? Is `start`/`end` too verbose? **A:** JSON, because all four existing verbs use the `output.Ok` envelope and `internal/cli` is the sole JSON site. Flat `"start"` / `"end"` keys rather than a nested range object, to keep it cheap.
- **Q:** [auto] Should the signature rule handle type declarations, which have no `body` field? **A:** Yes — the rule is per-kind "body-bearing child", never a first-line cut, and a grouped `type ( … )` block yields one symbol per `type_spec`. **Why:** a naive `ChildByFieldName("body")` returns nil for Go's `type_declaration` and the signature silently becomes the whole struct — the token blowup the verb exists to prevent.
- **Q:** [auto] Can toc reuse `runBatch`? **A:** No — its own driver, keyed on `"path"`, reusing `batchStatus`/`statusRank`. **Why:** `runBatch` hard-codes `entry["symbol"]`, and generalizing it would change the output shape of all four shipped verbs.
- **Q:** [auto] What does `--lang` mean for toc? **A:** Validated against toc's own five-name vocabulary, not the servers.yaml registry; it overrides the extension outright and a mismatch is not an error. **Why:** toc loads no language server, so it must not be coupled to the registry; erroring on mismatch would defeat the override's only real use.
- **Q:** [auto] Which grammar set ships? **A:** Superseded by the backend reversal. Under cgo each grammar is its own Go module, so quarry imports exactly the five it supports and there is no grammar-set choice, no build tags, and no unused grammars in the binary.
- **Q:** [auto] What does `toc dir` do with a `.ts`/`.rs` file? **A:** Lists it with `error: "language not yet supported by toc"` and no header; it counts as a code file, so such a directory is not "empty". **Why:** silently skipping would tell the agent the directory contains no code, which is false — the same principle as never dropping a header-less file.
- **Q:** [auto] How are paths resolved and emitted? **A:** Relative arguments join against `CwdFrom(ctx)`; `toc dir` entries emit the directory argument as written joined with the filename; the batch key echoes the argument verbatim. **Why:** these paths exist to be pasted back into `quarry toc file`, so they must round-trip in the caller's own frame of reference.
- **Q:** [auto] What is the full emitted key set, and the `kind` vocabulary? **A:** Fixed explicitly in the "Emitted schema" Decision; `kind` is the closed three-member set `function` / `method` / `type` across all five languages. **Why:** a richer vocabulary would be per-language noise the verbatim signature already conveys on the next field.
- **Q:** [auto] Use the library's `Outliner` for owner resolution? **A:** No. **Why:** it does not do docstring association, so our own walk exists regardless, and by then the receiver/enclosing class is one `ChildByFieldName` away — using `Outliner` would mean a per-language `OutlineOwnerRule` table for a value already in hand.
- **Q:** [auto] What happens when the first comment block is a `//go:build` directive? **A:** Directive-only leading blocks are skipped and the next block is taken; a block mixing directives and prose is a header. **Why:** verified in this tree — `internal/proc/proc_windows.go` and three integration tests have exactly that shape, and the naive rule would emit `header: "go:build windows"`.
- **Q:** [auto] Does a `toc dir` entry carry `partial`? **A:** Yes; it is in the closed key set, mutually exclusive with `error`. **Why:** `toc dir` parses each file to get its header, so a lossy parse makes that header suspect in exactly the way `toc file` warns about.
- **Q:** [auto] Are the two `minPackageDirs = 6` constants the same claim? **A:** No — 8 is the exact count in the layering guard and a deliberate one-below floor in the seam guard, so their comments are rewritten differently. **Why:** the seam guard walks `quarry/` too and documents its slack on purpose.
- **Q:** [auto] Does any test exercise the configuration that actually ships? **A:** Under cgo, yes by construction — there is no build-tag surface, so `go test ./...` runs exactly the code that ships. The concern that produced this question is what surfaced the mistyped-tag panic, which in turn contributed to abandoning the pure-Go backend.
- **Q:** [auto] What order are `symbols` and `files` in? **A:** Source order (ascending `start`) and explicit lexicographic filename order. **Why:** the integration test asserts a full envelope, which is unwritable against an unspecified order, and OS directory order is not stable across filesystems.
- **Q:** [auto] Is the Go type symbol the `type_declaration` or the `type_spec`? **A:** The spec is the symbol unit in both grouped and ungrouped forms, but the signature is cut from the declaration and always includes the `type` keyword. **Why:** `FileLock struct` is invalid Go and useless to paste; one rule must produce identical output for both source shapes.
- **Q:** [auto] What exit code does a single-argument toc failure give? **A:** Always 1. Rank 3 is batch-only. **Why:** `internal/cli/cli.go:13-21` documents 0/1/2 as the single-arg contract for every verb; rank 3 exists only to order batch statuses.
- **Q:** [auto] Does toc add engine sentinels? **A:** Exactly one, `ErrLanguageUnsupported`, re-exported through the facade; path existence and type stay CLI-side. **Why:** it is the one failure the engine can detect and the CLI cannot, and it makes the facade's sentinel count a decided number rather than an open question.
- **Q:** [auto] Is a `// Code generated … DO NOT EDIT.` banner a header? **A:** No — it is a directive block for header purposes, while still feeding `generated: true`. **Why:** same non-purpose noise as a build constraint.
- **Q:** Have you actually tested Tree-sitter and got it working? **A:** Not at the time of the recommendation — the recommendation rested on documentation and the platform argument. A spike was then built and run against real quarry source, and it produced four findings that changed the design: the header rule must tolerate a blank line, error recovery is lossy rather than merely incomplete, "top-level" does not generalize beyond Go, and docstring placement is structurally different per language.
