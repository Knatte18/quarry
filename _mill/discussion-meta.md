# Discussion record: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
task: Add file/dir toc verbs (Tree-sitter-backed)
slug: toc-verbs
companion: _mill/discussion.md
```

This is the **rationale and evidence record** for the toc-verbs discussion.
`_mill/discussion.md` is the specification — it states what is decided.
This file states *why*, records the measurements that drove each choice, lists every rejected
alternative, and preserves the discussion's own history including one reversed decision.

Nothing here is required to write the implementation plan.
Read it when you want to know why something is the way it is, or before reopening a settled
question.

---

## 1. Parsing backend: the measurements, and one reversal

### 1.1 What was decided first, and why it was wrong

The backend was first chosen as the pure-Go runtime `github.com/odvcencio/gotreesitter`, on the
argument that "the C runtime is ~3.9x faster on a full parse, but the parse cost is noise".

That argument failed twice:

1. **The 3.9x figure was a vendor self-benchmark.** It came from the pure-Go project's own README —
   the project's claim about its own disadvantage — and was never independently checked. Measured
   here, parsing is near-parity: **7.9 ms (cgo) vs ~10 ms (pure Go)** for `internal/cli/cli.go`,
   i.e. 1.3x, not 3.9x.
2. **It measured the wrong thing.** The dominant cost is not parsing, it is **grammar load**, and
   there the gap is roughly 2500x. Worse, the supporting datum — "113 ms including process startup"
   — was itself almost entirely pure-Go grammar-load time. A cost that exists *only in the option
   being recommended* was used as evidence that the alternative's advantage did not matter.

The decision was reversed to the official cgo bindings once both libraries were built and measured.

### 1.2 Measured comparison

Both libraries compiled and run against the same files on this tree.

| | cgo (chosen) | pure Go (rejected) |
| --- | --- | --- |
| Grammar load, first call in process | **0.03 ms** | **70–86 ms** |
| Each additional grammar, same process | ~0.03 ms | 7–9 ms (C#: 33.5 ms) |
| Parse `internal/cli/cli.go` (38 KB) | 7.9 ms | ~10 ms |
| **End to end, per `quarry toc` invocation** | **~0.5 ms** | **~80 ms** |
| Binary, all five languages | 12.6 MB | 13.1 MB |
| Build time | 6.8 s | ~2 s |

All five cgo grammars verified to load and parse a representative snippet with `hasError=false`:
go, python, csharp, typescript, rust.

**A binary-size claim made during discussion was also wrong.** cgo was described as "4x smaller"
(3.3 MB); that was a build with the Go grammar alone. With all five languages linked it is 12.6 MB
against pure Go's 12.7–13.1 MB. **Size does not favour either option** and formed no part of the
final decision.

### 1.3 What actually decided it

The choice reduced to ~80 ms of unavoidable per-invocation startup against a C toolchain as a build
dependency. quarry is built by its author for their own machines, so a build dependency is a
one-time setup per machine rather than a distribution problem, and a documented C toolchain
requirement is ordinary for Go projects — anything using `sqlite3` carries the same one. Against
that, 80 ms is paid on every call forever.

Maturity reinforced it. The pure-Go library produced **two runtime panics** during this discussion —
a mistyped `grammar_subset_*` build tag, and a missing external blob under
`grammar_blobs_external` — where both should have been build-time failures. The cgo bindings have no
build-tag surface to mistype.

### 1.4 Rejected backend alternatives

- **Pure-Go runtime `github.com/odvcencio/gotreesitter`** — no size advantage, ~160x slower per
  invocation end to end, and two runtime panics in one afternoon where build-time failures were
  expected. Its only real advantage is needing no C toolchain.
- **wazero/WASM wrapper (`github.com/malivvan/tree-sitter`)** — cgo-free, but adds a WASM runtime
  and per-grammar `.wasm` loading; slower than native C for the same dependency burden.
- **`grammar_blobs_external`** (pure-Go mode putting grammars in separate files) — considered as a
  way to buy back the startup cost while staying pure Go. Verified to break single-binary
  distribution: it panics at runtime with
  `open grammars/grammar_blobs/go.bin: no such file or directory` unless the blobs ship alongside
  the binary.
- **`grammar_set_core`** and the per-language `grammar_subset` tag family — the pure-Go library's
  binary-size controls, moot once cgo was chosen. Sizes measured while that backend was still the
  candidate: untagged (all 206 grammars) 30.2 MB; `grammar_set_core` 23.8 MB;
  `grammar_blobs_external` 14.9 MB; per-language `grammar_subset` 12.7 MB; baseline quarry with no
  Tree-sitter at all 5.9 MB.
  Note the tag for C# is `grammar_subset_c_sharp`, not `grammar_subset_csharp`, and TypeScript also
  requires `grammar_subset_javascript` — the mistyped-tag panic came from exactly this.
- **The runtime `GOTREESITTER_GRAMMAR_SET` environment variable** — restricts *loading*, not binary
  size; it never addressed the problem it appeared to.

### 1.5 Dual backend: why not

Considered: ship both, selected automatically by Go's built-in `cgo` build constraint —
`//go:build cgo` for the C runtime, `//go:build !cgo` for a pure-Go fallback. On a machine with a C
toolchain you get the fast one; on Windows without one you get the portable one, with no flag for
the user.

Rejected on correctness, not effort:

- **Two independent Tree-sitter implementations do not produce identical trees.** The pure-Go
  runtime self-reports three "degraded" grammars and has its own error-recovery behaviour, so
  `partial: true` and the set of symbols surviving a syntax error could differ between builds.
  `quarry toc` would return different answers for the same file depending on how the binary was
  compiled — unacceptable for a tool whose entire value is that an agent can trust the result
  without verifying it. A slow outline is usable; an outline that is correct only for some build
  configurations is worse than none.
- **The node APIs are not interchangeable.** Pure Go takes the language as an argument —
  `node.Type(lang)`, `ChildByFieldName(name, lang)` — while cgo does not: `node.Kind()`,
  `ChildByFieldName(name)`. A normalising adapter per library would wrap the single code path all
  extraction flows through.
- **The test matrix doubles or one backend rots silently.** Every extraction test would have to run
  under both. This is the same failure mode the round-4 review caught with build tags, except the
  untested artifact is an entire parser.
- **`CGO_ENABLED` defaults to 0 when cross-compiling**, so the fallback would be selected silently
  rather than deliberately.

Also rejected: a dual backend that names the active backend in the JSON envelope. That is the only
honest form of a dual build, since it makes divergence visible — but it is complexity for a tool
with one user, and it pushes the divergence problem onto the consumer instead of solving it.

---

## 2. No parser daemon: the measurements

### 2.1 Cost breakdown

| Scope | Cost |
| --- | --- |
| First grammar loaded in a process (pure-Go runtime) | 70–86 ms, one-time process init |
| Each additional grammar in the same process | 7–9 ms (C#: 33.5 ms) |
| Re-requesting an already-loaded grammar | 0.0 ms |
| `toc file` on 1.7 KB | 0.08 s wall, end to end |
| `toc file` on 38 KB (`internal/cli/cli.go`) | 0.10 s wall, end to end |
| `toc dir internal/output` (2 files, 8.3 KB) | 13 ms parsing, 6.6 ms/file |
| `toc dir internal/lock` (2 files, 6.1 KB) | 11 ms parsing, 5.6 ms/file |
| `toc dir internal/quarryengine/query` (7 files, 61 KB) | 47 ms parsing, 6.8 ms/file |
| `toc dir internal/cli` (9 files, 111 KB) | 94 ms parsing, 10.4 ms/file |
| `toc dir internal/quarryengine/daemon` (14 files, 115 KB) | 0.16 s wall; 72 ms parsing, 5.1 ms/file |

Per-file parse cost is 5–10 ms and grammar load is paid once per process, so a directory's cost is
`fixed init + N × ~7 ms`. A 23x larger file costs 20 ms more: fixed startup dominates, not file size.
These figures were taken with the pure-Go runtime; under cgo the fixed-init term collapses to
~0.03 ms, which strengthens the conclusion rather than changing it.

### 2.2 Why the LSP daemon is not a precedent

`EnsureServer` exists because gopls indexes an entire module, holds type-checked packages in memory,
and has a cold start measured in seconds to tens of seconds — persistent state is the entire point.
Tree-sitter has none of that: no project indexing, no cross-file state, a stateless per-file parse.

Nor is the machinery reusable. `EnsureServer` spawns a *language server speaking LSP*, wrapped in
`daemonstate`, `probe`, a spawn-race lock, and toolchain management. A toc daemon would be quarry
talking to itself over a new protocol, with a new state file, a new lifecycle, and a new lock — a
second daemon inheriting the first one's problems, including the platform split README already
documents (the supervised strategy hard-codes a Unix domain socket, so windows falls back).

What a daemon would buy is the fixed init: 1–3% of one LLM agent turn, bought with shared state,
mtime-based cache invalidation, and a platform split — for a verb whose entire value is being
trustworthy.

**Batch mode is the cheap version of the same win.** `toc file a.go b.go c.go` pays the fixed init
once and ~7 ms per additional file in one process, with no shared state and no invalidation problem.
The batch-mode decision was taken for CLI consistency; these numbers are its second, independent
justification.

Rejected: a toc daemon reusing `EnsureServer` (wrong shape — that seam launches LSP servers);
a new toc-specific daemon (new protocol, state file, lock, and platform split to save the init);
an in-process parse-tree cache keyed on mtime (no second call exists to hit it — each CLI invocation
is a fresh process).

---

## 3. Windows: what was and was not verified

A C toolchain is needed to **build**, never to **run**.

- Native windows builds use mingw-w64 (MSYS2) or TDM-GCC.
- Cross-compiling from linux or WSL2 needs `gcc-mingw-w64-x86-64` and the `-static` external
  linker flag, so the `.exe` does not depend on `libwinpthread-1.dll` and friends sitting beside it.
- **A linux-built binary does not run on windows.** Go binaries are per-OS; a windows `.exe` requires
  `GOOS=windows`.
- **Building inside WSL2 without `GOOS=windows` produces a *Linux* binary** that runs in WSL2, not a
  windows-native `.exe`. Both are legitimate targets; they are not the same artifact. If quarry only
  ever runs inside WSL2, the windows-native question does not arise at all.

**Not verified during discussion:** the cross-compile command. No mingw-w64 cross-compiler was
installed on the discussion machine (`x86_64-w64-mingw32-gcc` absent, `zig` absent). It is the
standard incantation, but it was not run. `_mill/discussion.md` carries the recipe, the instruction
to actually run it, and the fallback if no toolchain is available to the implementation batch.

---

## 4. Rejected alternatives, by decision

Every alternative weighed during discussion, kept so a future reader does not re-litigate a settled
question without knowing what was already considered.

**Verb shape.** Rejected: `outline <file>` + `manifest <dir>` (two flat verbs matching the existing
four, but two unrelated names to learn); `toc <file>` + `survey <dir>` (still asymmetric);
`toc` + `toc-dir` (asymmetric naming for two siblings);
a single `toc <path>` dispatching on `stat` (re-merges two deliberately distinct output shapes).

**Path-type validation.** Rejected: auto-dispatching on stat and running whichever verb fits;
ignoring the mismatch (produces a confusing empty result).

**Path handling.** Rejected: absolute paths (verbose, machine-pinned, not what the caller typed);
bare basenames (not directly reusable as a `toc file` argument);
paths relative to the repo root (there is no repo-root concept in toc — it never detects a project
the way the LSP verbs do).

**Language detection.** Rejected: reusing `DetectLanguage` on the file's parent directory (wrong for
mixed-language trees — a `.ts` file inside a Go module resolves to "go" because `go.mod` wins the
precedence list); sniffing file content (unnecessary — the extension is definitive for all five);
validating toc's `--lang` against the servers.yaml registry (couples a parser-only verb to the
language-server config it has no other reason to load);
erroring on an extension/`--lang` mismatch (defeats the override's only real purpose — a file with a
non-standard extension).

**Language scope.** Rejected: Go only (leaves the "second parsing backend" abstraction exercised by
exactly one language, which is not an abstraction but a guess); all five now (larger batch with no
design risk left to retire after the third language).

**Symbol kinds.** Rejected: adding top-level consts/vars (more noise per token than signal);
adding fields and nested members as a tree (at that point the agent may as well read the file, which
is the thing the verb exists to avoid).

**"Top-level".** Rejected: literal top-level-children-only (returns nothing useful for 2 of the 3
implemented languages — Python methods live in a class block, C# methods in a declaration list);
unbounded descent (pulls in local closures and helper functions declared inside bodies).

**Output shape.** Rejected: `name: "FileLock.Release"` (breaks cross-verb reuse — `refs`,
`definition`, and `symbol` take a bare name or a `file:line:col` position, so a qualified name pasted
into another quarry call would not parse);
a nested `children` tree (more tokens, and a different shape per language; a flat list can always be
rebuilt into a tree, but not the reverse without committing to one nesting form per language).

**`kind` vocabulary.** Rejected: a per-language vocabulary mirroring grammar node types (breaks the
flat uniform shape and makes a consumer learn five vocabularies);
omitting `kind` entirely (the consumer cannot cheaply separate types from functions);
a `language` field per symbol rather than per file (a file has exactly one language).

**Line ranges.** Rejected: a nested `"range": {"start":…, "end":…}` object (same information, more
tokens); adding byte offsets (precise for tooling, useless to the LLM consumer this tool exists for).

**Signature.** Rejected: a normalized `name + param types + return type` rendering (uniform across
languages, but five formatters' worth of maintenance and five ways to be subtly wrong);
cutting at the first `\n` (loses everything after the first line of a multi-line signature — and
this tree contains no multi-line function signature today, so the hack would pass its own tests here
and break on the first target repo that has one, which is precisely why the rule must be structural
rather than line-based);
cutting at the first `{` byte (wrong for a generic constraint or a map literal in a default).

**Docstring text.** Rejected: raw source text for docstrings (consistent-looking, but ships comment
markers and XML tags to a consumer that wants prose).

**Absent fields.** Rejected: always-present `docstring: null` (stable schema, one wasted line per
symbol); `docstring: ""` (indistinguishable from a genuinely empty docstring);
omitting header-less files from `toc dir` (silently loses files from a directory overview).

**File header.** Rejected: the block adjacent to the package/namespace declaration (that is Go's doc
convention, but in this codebase it returns the package doc instead of the file header);
emitting both `header` and `package_doc` (most complete, but doubles `toc dir`'s cost for something
the agent rarely needs).

**`toc file` header.** Rejected: leaving headers exclusively to `toc dir` (sharper separation of
concerns, but forces two calls for one file's full context).

**`toc dir` shape.** Rejected: recursive with a `--depth` flag (risks enormous output);
single-language-only listing (silently hides files);
listing every file including non-code (at that point `ls` is cheaper);
untruncated headers (most complete, most expensive, and the tail is rarely the purpose).

**Test/generated flags.** Rejected: always emitting `test: true|false` on a best-effort basis
(stable schema, but ships false negatives as facts — `test: false` on a C# test file is a lie a
consumer cannot distinguish from a fact);
dropping the flag entirely (loses it for Go, where it is reliable and useful);
excluding tests by default behind an `--include-tests` flag (hides files).

**Unparseable input.** Rejected: hard-failing the file (loses the directory over one bad file);
returning recovered symbols with no marker (the consumer cannot know the list is incomplete);
reusing `partial` for unreadable files (conflates "lossy parse" with "no parse");
erroring on an empty directory (an empty listing is a valid answer);
a file-size cap (invents a threshold with no evidence any real file needs one).

**Batch mode.** Rejected: reusing `runBatch` as-is (mislabels paths as symbols — it hard-codes
`entry["symbol"] = arg`);
generalizing `runBatch`'s key (breaks the four existing verbs' output shape — a breaking output
change to shipped verbs, for a task meant to add verbs, not alter them);
exactly-one-argument (simpler, but breaks a pattern all four existing verbs follow);
ranking `partial` as its own status (invents a fifth vocabulary member and pollutes batch exit
codes).

**Package layout.** Rejected: one combined `toc` package (fewer packages, but the backend stops
being its own layer — and that layer is what made the backend reversal a one-package change);
putting `toc` inside the existing `query` package (`query` is LSP orchestration; mixing two backends
in one package is exactly the coupling the split avoids).

**Facade.** Rejected: `internal/cli` calling the engine package directly (smaller surface, breaks
the facade contract the thin-facade commit established).

**Library outline helpers.** Rejected for owner resolution: the library ships outline/tagging
helpers with an `Owner` field, but they do not do docstring association, so the sibling/in-body walk
is ours regardless — and by the time that walk holds the declaration node, the receiver or enclosing
class is one `ChildByFieldName` away. Using them would mean maintaining a declarative owner-rule
table per language for a value already in hand, and coupling to a second, larger piece of the
library's API.

---

## 5. Spike results

A working extraction spike was built and run against real quarry source. The extraction logic
(docstring walk, signature cut, ranges) was written against the pure-Go runtime before the backend
reversal; its **findings about source structure remain valid**, being facts about Go/Python/C#
syntax trees rather than about either library.

| Test | Result |
| --- | --- |
| `internal/output/output.go` | 3 symbols, correct signature, docstring, and range spanning docstring + declaration |
| `internal/cli/cli.go` (38.8 KB) | 25 symbols, 10.9 KB of JSON — a 3.5x reduction against reading the file |
| `internal/lock/lock.go` | correctly distinguishes `function`, `method` (receiver), and `type` |
| Deliberately broken Go file | `partial: true` set; symbols before the error preserved; a symbol *after* the error was lost |
| All five cgo grammars | go, python, csharp, typescript, rust each load and parse with `hasError=false` |
| cgo binary, all five languages | 12.6 MB |

**Four spike findings changed the design**, each of which would otherwise have shipped as a bug:

1. **The header rule must tolerate a blank line, and must skip directive blocks.**
   `internal/output/output.go` has a blank line between its header and `package output`, so a strict
   adjacency rule drops the header entirely. `internal/cli/cli.go` has *two* top-of-file blocks — a
   file header and a `// Package cli …` doc — and strict adjacency picks the package doc.
   Separately, `internal/proc/proc_windows.go:1` is `//go:build windows` followed by a blank line
   and then the real header; the same shape appears in
   `internal/quarryengine/query/refs_integration_test.go:1`,
   `daemon/supervised_lsp_test.go:1`, and `daemon/ensureserver_integration_test.go:1`. Without the
   directive-skipping rule, `quarry toc dir internal/proc` would emit `header: "go:build windows"`.
2. **Error recovery is lossy, not merely incomplete.** In the broken-file fixture, an unparseable
   `func Broken(` swallowed the rest of the file and a later, perfectly valid `func Later()`
   vanished from the output. This is why `partial: true` is load-bearing rather than decorative.
3. **"Top-level" is a Go-shaped notion that does not generalize.** Python methods live inside
   `class_definition` → `block`; C# methods inside `class_declaration` → `declaration_list`, itself
   usually inside a namespace. Taken literally, "top level" returns zero methods for two of the
   three implemented languages.
4. **Docstring placement is structurally different per language.** Go and C# put it in `comment`
   prev-siblings; Python puts it as the first `string` node *inside* the body. One shared code path
   is impossible; it must be a per-language strategy behind one interface.

A fifth finding came from the signature rule: the spike used a first-line-only cut
(`SplitN(sig, "\n", 2)[0]`) to avoid emitting whole struct bodies, which happens to work in this
tree only because it contains no multi-line function signature.

### Spike artefacts

The spike lives under `.scratch/tsspike/` and `.scratch/cgobench/` (gitignored). It is throwaway
reference code, not a starting point to be promoted — implementation should be written properly
against the interface `_mill/discussion.md` describes. It is worth reading once for the working
docstring-walk, the tree dumper, and the two benchmark harnesses.

---

## 6. Q&A log

The running record of questions answered during discussion. Distilled decisions live in
`_mill/discussion.md`; this captures the edge cases, tie-breakers, and corrections that would
otherwise be lost. Entries marked `[auto]` were resolved by the assistant during review rounds
rather than by the operator.

- **Q:** Is C faster, and does that matter? **A:** *This answer was given as "3.9x on parse, but
  speed is not the deciding factor" and was later reversed.* The 3.9x figure was the pure-Go
  project's self-reported benchmark, never checked; measured, parsing is 1.3x. And parsing was the
  wrong thing to measure: grammar load is the dominant cost, and there cgo is ~2500x faster
  (0.03 ms vs 70–86 ms), making it ~160x faster end to end per invocation. See §1.
- **Q:** Does Tree-sitter support all languages regardless of binding? **A:** Yes — grammars are
  per-language parse tables, and both the cgo bindings and the pure-Go runtime cover all five of
  quarry's languages. Language coverage does not separate the options.
- **Q:** What *is* a grammar, that has to be loaded? **A:** A formal description of one language's
  syntax, compiled to a parse table — for each parser state and next token, shift/reduce/error.
  Typically hundreds of KB to a few MB per language. Under cgo it is generated C compiled into the
  binary as static data, so "loading" is taking a variable's address; under the pure-Go runtime it
  is a compressed blob decompressed and rebuilt in memory on first use. That difference *is* the
  70–86 ms.
- **Q:** Is a C toolchain acceptable as a build dependency? **A:** Yes — quarry is built by its
  author for their own machines, and a documented C toolchain is ordinary for Go projects. Running
  the binary needs nothing extra.
- **Q:** Can we ship two backends, cgo on POSIX and pure Go for windows? **A:** No. Two independent
  Tree-sitter implementations do not produce identical trees, so `quarry toc` would give different
  answers depending on how the binary was built — fatal for a tool whose value is that the agent
  need not verify the result. See §1.5.
- **Q:** Once the binary is built, does it run on windows? **A:** Not a linux-built one — Go
  binaries are per-OS. A windows `.exe` needs `GOOS=windows` plus a mingw-w64 cross-compiler, or a
  native windows build. See §3.
- **Q:** Can I just build in WSL2 on Win11? **A:** Yes, but that yields a *Linux* binary that runs
  in WSL2. Fine if quarry only runs there; not a windows-native `.exe`. WSL2 can produce the `.exe`
  too, with one apt package. See §3.
- **Q:** Would a daemon be worth it, since we have one for LSP anyway? **A:** No. See §2 — the LSP
  daemon exists for gopls's seconds-long cold start and persistent index; Tree-sitter has neither,
  and the machinery is LSP-shaped rather than reusable.
- **Q:** One `toc <path>` verb dispatching on stat, or two? **A:** Two, because the output shapes
  differ — but as `toc file` / `toc dir` subcommands rather than the asymmetric `toc` / `toc-dir`.
- **Q:** What if the caller passes the wrong path type? **A:** Stat first, validate against the
  subcommand's expected type, hard-fail on mismatch with a message naming the correct subcommand.
- **Q:** Should `toc dir` recurse? **A:** No. Explicitly non-recursive.
- **Q:** Should the docstring be part of `toc file`'s output, and should the line range cover it?
  **A:** Yes to both — this is the core premise, not an option. Signature + docstring is what tells
  a reader what a method is for.
- **Q:** Should `toc dir` list only the detected language's files, or all languages? **A:** All
  supported languages, since a directory is not guaranteed single-language. Go first, Python and C#
  in the same task, TypeScript and Rust deferred.
- **Q:** How can a test file even be flagged, given Go has `_test.go` but C# has nothing? **A:** It
  cannot, uniformly. Emit the key only where a reliable rule exists; omit it entirely for C# and
  Rust rather than emitting a false `test: false`.
- **Q:** Is the output format even JSON? Is `start`/`end` too verbose? **A:** JSON, because all four
  existing verbs use the `output.Ok` envelope and `internal/cli` is the sole JSON site. Flat
  `"start"` / `"end"` keys rather than a nested range object, to keep it cheap.
- **Q:** Have you actually tested Tree-sitter and got it working? **A:** Not at the time of the
  original recommendation — it rested on documentation and the platform argument. A spike was then
  built and run against real quarry source, producing the four design-changing findings in §5.
- **Q:** [auto] Should the signature rule handle type declarations, which have no `body` field?
  **A:** Yes — the rule is per-kind "body-bearing child", never a first-line cut, and a grouped
  `type ( … )` block yields one symbol per `type_spec`. A naive `ChildByFieldName("body")` returns
  nil for Go's `type_declaration` and the signature silently becomes the whole struct.
- **Q:** [auto] Is the Go type symbol the `type_declaration` or the `type_spec`? **A:** The spec is
  the symbol unit in both grouped and ungrouped forms, but the signature is cut from the declaration
  and always includes the `type` keyword. `FileLock struct` is invalid Go and useless to paste.
- **Q:** [auto] Can toc reuse `runBatch`? **A:** No — its own driver, keyed on `"path"`, reusing
  `batchStatus`/`statusRank`. `runBatch` hard-codes `entry["symbol"]`, and generalizing it would
  change the output shape of all four shipped verbs.
- **Q:** [auto] What exit code does a single-argument toc failure give? **A:** Always 1. Rank 3 is
  batch-only. `internal/cli/cli.go:13-21` documents 0/1/2 as the single-arg contract for every verb.
- **Q:** [auto] What does `--lang` mean for toc? **A:** Validated against toc's own five-name
  vocabulary, not the servers.yaml registry; it overrides the extension outright and a mismatch is
  not an error.
- **Q:** [auto] What does `toc dir` do with a `.ts`/`.rs` file? **A:** Lists it with
  `error: "language not yet supported by toc"` and no header; it counts as a code file, so such a
  directory is not "empty". Silently skipping would tell the agent the directory contains no code.
- **Q:** [auto] And `toc dir --lang rust`? **A:** Lists, does not error — `ok:true`, exit 0, every
  `.rs` file carrying the error key. `--lang` selects which files to list; the listing rule for an
  unimplemented language is already decided. `toc file --lang rust` *does* error, because there the
  unsupported language is the whole request.
- **Q:** [auto] How are paths resolved and emitted? **A:** Relative arguments join against
  `CwdFrom(ctx)`; `toc dir` entries emit the directory argument as written joined with the filename;
  the batch key echoes the argument verbatim. These paths exist to be pasted back into
  `quarry toc file`.
- **Q:** [auto] What is the full emitted key set, and the `kind` vocabulary? **A:** Fixed explicitly
  in the Emitted-schema Decision; `kind` is the closed three-member set `function` / `method` /
  `type` across all five languages.
- **Q:** [auto] What order are `symbols` and `files` in? **A:** Source order (ascending `start`) and
  explicit lexicographic filename order. OS directory order is not stable across filesystems.
- **Q:** [auto] Does toc add engine sentinels? **A:** Exactly one, `ErrLanguageUnsupported`,
  re-exported through the facade; path existence and type stay CLI-side.
- **Q:** [auto] Use the library's outline helpers for owner resolution? **A:** No — see §4.
- **Q:** [auto] What happens when the first comment block is a `//go:build` directive? **A:**
  Directive-only leading blocks are skipped and the next block is taken; a block mixing directives
  and prose is a header. Verified against four files in this tree.
- **Q:** [auto] Is a `// Code generated … DO NOT EDIT.` banner a header? **A:** No — it is a
  directive block for header purposes, while still feeding `generated: true`.
- **Q:** [auto] Does a `toc dir` entry carry `partial`? **A:** Yes; it is in the closed key set,
  mutually exclusive with `error`. `toc dir` parses each file to get its header, so a lossy parse
  makes that header suspect.
- **Q:** [auto] Are the two `minPackageDirs = 6` constants the same claim? **A:** No — 8 is the
  exact count in the layering guard and a deliberate one-below floor in the seam guard, so their
  comments are rewritten differently.
- **Q:** [auto] Which grammar set ships? **A:** Superseded by the backend reversal. Under cgo each
  grammar is its own Go module, so quarry imports exactly the five it supports — no grammar-set
  choice, no build tags, no unused grammars.
- **Q:** [auto] Does any test exercise the configuration that actually ships? **A:** Under cgo, yes
  by construction — there is no build-tag surface, so `go test ./...` runs exactly the code that
  ships. The concern that produced this question is what surfaced the mistyped-tag panic, which in
  turn contributed to abandoning the pure-Go backend.

---

## 7. Review history

Six holistic discussion-review rounds ran (`opushigh`, effort high), all findings fixed.
Reports under `_mill/reviews/`. What each round caught, as a record of where this design was weak:

| Round | Findings | Most consequential catch |
| --- | --- | --- |
| 1 | 2 BLOCKING, 5 NIT | The signature rule was unimplementable: Go's `type_declaration` has no `body` field, so the stated rule emits whole struct bodies |
| 2 | 3 BLOCKING, 4 NIT | The emitted schema had no `kind` field and no defined path form; a `buildOptions` citation was factually false |
| 3 | 2 BLOCKING, 3 NIT | The header rule picked up `//go:build` directive blocks — verified in four files in this tree |
| 4 | 2 BLOCKING, 3 NIT | The shipped build-tag configuration was exercised by no test; `go test ./...` ran untagged and would stay green against a mistyped tag |
| 5 | 1 BLOCKING, 4 NIT | The Go type symbol unit was self-contradictory between two sections; the staleness invariant covered only count-shaped prose |
| 6 | 1 BLOCKING, 3 NIT | `toc dir --lang <unimplemented>` had two defensible answers; superseded grammar-set deliverables survived the backend reversal |

Round 4's finding is worth singling out: acting on it is what led to running the tagged build, which
panicked on a mistyped `grammar_subset_csharp`, which was one of the two panics that decided the
backend reversal. The review caught a testing gap; the testing gap exposed a library weakness.
