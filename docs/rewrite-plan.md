# Quarry rewrite — what goes, what stays, what gets built

Decided 2026-09-03. Quarry V1 is frozen on the `v1-final` branch (worktree `wts/v1-final`), kept as a
reference. `main` is cleaned out and the rewrite is built there under the same module path,
`github.com/Knatte18/quarry`. There is no "V2" in any name: when this plan is done, what is on `main`
is quarry.

The benchmark harness (`bench/loomyard-eval/ladder/`), the research notes (`docs/research/`), the
results roots and `HANDOFF.md` stay where they are. They are the measurement record that motivates
this document and the instrument that will measure the result.

## 1. Why a rewrite, in four measured lessons

Every claim here has a results root or a captured output behind it; see `HANDOFF.md` §1, §6, §8 and
`docs/research/mcp-surface.md`.

1. **Only the tree-sitter half ever paid.** Across 45 + 30 + 6 measured runs, `toc_dir` is the one
   tool that separated from the control (turns 8→4, cache_read 127k→83k, correctness unchanged).
   Every LSP-shaped tool — `definition`, `references`, `workspace_symbol` — was flat or worse in
   every run since August. `impact` has the best-shaped envelope but sits on the LSP layer.
2. **The LSP layer's shape is an editor's shape.** Positions as addresses, undecoded `SymbolKind`
   integers, fuzzy `workspace/symbol` behind every bare-name call without showing candidates, three
   naming conventions in one array. Faithful to the protocol, useless to a caller that has no cursor
   and no client-side enum table. gopls's own MCP server, shipped in the same binary, takes no
   position anywhere and addresses everything by file + qualified name.
3. **Identity and location were one field, and they are two things.** A plan needs a stable name for
   a symbol. An implementer needs where that symbol is *right now*, and that moves with every edit.
   V1 answered both with `{file, line, character}`.
4. **Extraction is complete; a view filters; no view is ever forced.** The compact-toc ladder
   (`results/2026-09-02-compact2`) forced a one-sentence-per-file view and lost precision 0.96→0.82
   while cutting bytes 4×: the agent answered from the thinner map instead of noticing it was thin.

Three design generations were visible in one CLI (`docs/research/output-formats/INDEX.md`). The
rewrite has one.

## 2. What is deleted, what is kept

Non-test lines at `2565ef5`:

| part | lines | disposition |
|---|---|---|
| `internal/quarryengine/toc`, `treesitter`, the extension→language table in `registry` | ~2 100 (+2 600 test) | **kept** — the extractors for Go, Python, C#, comment association, sentence splitter, tree-sitter wrapper |
| `internal/quarryengine/{daemon,lsp,query,impact,registry (rest)}`, `internal/{lock,proc}` | ~3 400 | **deleted** — the LSP layer. `impact` returns in phase 2 behind the new contract, on whichever type checker phase 2 picks |
| `internal/cli`, `internal/mcpserver`, `internal/output`, `cmd/*`, `quarry/` facade | ~4 500 | **deleted and rebuilt** from the contract, much smaller: three verbs, not seven |
| `go.mod`: tree-sitter Rust and TypeScript grammars, LSP/JSON-RPC deps, `gopls` requirement | | **removed** — three focus languages, no daemon |
| `testdata/{impactfixture,clockfixture,buildtagfixture}` | | deleted with the layer that used them |
| `bench/`, `docs/`, `HANDOFF.md`, results | 17 000 (harness) | untouched |

The first commit on `main` after this document is the deletion, so nothing half-V1 can survive into
the new code. Until the new `cmd/quarry-mcp` exists the harness cannot run; that is expected and
short.

## 3. The symbol identifier

```
<package>#<Owner.>Name
```

- `package` is the **basename of the package directory**: `reedengine`, `render`, `logger`, `lyx`.
  Not the package clause (`main` is eight directories in Loomyard), not the import path (`internal/`
  would be written thousands of times in plan files).
- `Owner` is the enclosing type for a method or member, absent for a package-level symbol. Nested
  owners chain with dots: `Beta.Inner.handle`.
- `Name` is the declared name.

Examples: `logger#stderrHandlerSnapshot`, `logger#dualHandler.stderr`, `render#Renderer.Draw`,
`lyx#run`.

**One spelling.** No suffix matching, no aliases, no import-path form. An id is a name, not a path.

**Uniqueness is an invariant quarry checks, not assumes.** Package basenames must be unique across
the repository. Loomyard today: 83 packages, 0 duplicate basenames (measured 2026-09-03). If two
directories share a basename, `resolve` refuses both with both paths in the error until one is
renamed. The convention that makes ids short is enforced the first time someone breaks it.

**What the id abstracts away, and nothing more:** the file within the package (a Go method moves
between files freely) and every line number. Moving a package or renaming a symbol changes the id,
deliberately: the id *is* the name, and a Rename card's `old -> new` pair is exactly a pair of ids.

**Two outcomes that must never be confused:**

| outcome | meaning | result |
|---|---|---|
| ambiguous | the id matches several *different* symbols (Go build-tag duplicates, C# overloads, a Python name defined twice in one module) | error, candidates listed, never a silent pick |
| multipart | the id names *one* symbol the language allows in several places (C# `partial class`, C# partial methods) | success, every part returned |

Go never produces multipart: a type is declared once and each method once, however many files they
span. That is a `members` question (§5), not a resolution question.

**Alignment with Loomyard's plan format.** `manifest/designs/plan-card-format.md` already says
"package-qualified short names (`shedrecipe.Lookup`), never file:line, never full import paths". That
is this id with `.` instead of `#`. Loomyard can keep `.` and map the first separator to `#`, or adopt
`#`. Recommendation: adopt `#`, because `pkg.Type.Method` and `pkg.Name` are then distinguished by
the separator alone and no parser needs the package list to split them. This is Loomyard's decision.

## 4. One envelope

Every command, every surface, the same shape. Deviations are what made V1 three generations.

- **`ok` agrees with the exit code.** V1's `definition` returned `ok: true` and exited 2.
- **`status` per entry:** `found`, `not_found`, `ambiguous` (with `candidates`), `multipart`.
- **One location shape everywhere:** `file` (relative to the repository root, never absolute),
  `start`, `sigend`, `end` (1-based, inclusive), `kind` (a word, never an LSP integer), `id`,
  `signature` (verbatim source), `doc` (complete, never truncated by extraction).
- **Ids as keys in every output.** What `map` lists is what `resolve` takes.
- **`verified` per entry** on anything that claims a reference (phase 1 textual refs: always
  `false`; phase 2 impact: `true` when the type checker confirmed it). A consumer that wants only
  verified entries filters; a gate that reads unverified entries as proof is a defect.
- **JSON is the contract** for the CLI and the facade. The MCP `content[].text` block carries the
  same JSON. A compact text view may exist as an explicit option; it is never the default and never
  the only form.

## 5. The queries

All phase-1 queries are the same tree-sitter parse with a different entry point. None needs a type
checker, a daemon, or an index.

**`resolve <id>...`** — where is this, right now. Per id: location(s) and status. Called by an
implementer immediately before every read and every edit, because lines have moved since the last
time. Many ids in one call are grouped by package and each package is parsed once.

**`members <type-id>`** — what does this type consist of, across files. The type's *head* (its own
lines: declaration, doc, fields, class-level attributes — in Go the `type` block; in Python and C#
the class span minus its member spans) and every member with its location, sorted by file and line.
One line per member; the caller chooses what to read. This is how a large class is handled: the plan
names members, `members` gives the map, the implementer reads head + the named member + siblings it
picks. The whole class is `start`–`end` of the type symbol and is available, never the default.

**`map <package|file>...`** — what is here. V1's `toc dir`/`toc file`, the measured win, with ids as
keys. The first call on unfamiliar code. Headers and docs complete (lesson 4).

**`refs <id>` (phase 1, textual)** — every occurrence of the name in the repository with its
enclosing symbol. A syntactic over-approximation: exact for rare names, noisy for `Run`/`Handle`/
`New`, blind to reflection and string dispatch like everything else. Every entry `verified: false`.
It exists as a candidate list and as one gate direction: **zero textual hits proves zero callers;
one or more hits proves nothing.**

**`impact <id>` (phase 2)** — who calls this, type-resolved. Tree-sitter cannot do this: at a call
site `x.Run()` the receiver type is not in the syntax, and guessing it is the silent pick §3 forbids.
Needs a type checker. The open decision is which: gopls as in V1, or `go/packages` in-process for Go
(exact, no daemon, no protocol, Go-only; Python and C# would need a language server when they need
this). Decided when phase 2 starts, not now. `impact` returns each caller as an id plus location plus
`verified: true`.

**`assert-no-callers <id>` (phase 2 as a gate; phase 1 only in its zero-hit direction)** — the
exit-code contract for a Delete card's verify step.

Performance, measured 2026-09-03 in-process on a 4-core WSL2 host, Loomyard, every file read fresh
from disk, no cache:

| operation | time |
|---|---|
| walk repository, basename→directory map (83 packages) | 2 ms |
| resolve one id, 5-file package (32 KB) | 5–8 ms |
| resolve one id, 35-file package (317 KB), serial / name-prefiltered / parallel | 65 / 24 / 26 ms |
| 20 ids across 5 packages, grouped, serial / packages in parallel | 109 / 76 ms |
| every symbol in the repository, 469 files, serial / parallel | 616 / 237 ms |

Nearly all of it is cgo into tree-sitter (parse 47 of the 65 ms; node access and allocation
callbacks the rest). Nothing on quarry's side is worth optimising yet. No cache in phase 1: every
answer reads the source as it is at that moment, which is the point. If one is ever needed it is
keyed on (path, mtime, size) per file and parses only what changed.

## 6. Languages

Go, Python, C#. Not five. Each language's extractor must deliver, per symbol: kind, name, owner
chain, signature, doc, start/sigend/end, and the package basename. Known gaps in the kept
extractors, all phase 1:

| language | gap today | needed for |
|---|---|---|
| Python | nested classes dropped (`Beta.Inner` and its methods absent; `Owner` is one level) | id uniqueness, `members` |
| Python | class-level attributes are not symbols | the type head in `members` |
| C# | `partial` not recognised (the word `partial` in toc today means "lossy parse" — rename that field) | multipart resolution |
| C# | fields and properties not emitted as members | `members` |
| C# | overloads share an id | ambiguity reporting now; a signature suffix later if a real plan needs to name one overload |
| Go | none known; build-tag duplicates must report as ambiguous | |

## 7. Surfaces, in order of importance

1. **The Go facade** (`quarry/` package). The primary consumer is Loomyard's own Go code, never the
   LLM. Typed results, no JSON round-trip, grammars loaded once per process.
2. **The CLI.** Same queries, JSON out, exit codes for gates. The scripting contract.
3. **MCP.** A mirror of the CLI for an LLM that has the tools granted. At most four tools (`map`,
   `resolve`, `members`, and `impact` when it exists). Tool names are verbs, never protocol methods.

## 8. How Loomyard uses it

### 8.1 Plan cards (`manifest/designs/plan-card-format.md`), by card type

| card | before dispatch (mechanical, plan-time) | during implementation (the agent) | after (mechanical, done-check) |
|---|---|---|---|
| Create | `resolve` target → must be `not_found`; `map`/`members` on the package for "nothing equivalent exists" | `map` the package it goes into | `resolve` → `found` |
| Edit | `resolve` target → `found`; phase 2 `impact` → `ImpactSummary` and tier-1 test package set | `resolve` before each read/edit; `members` when the target is a type | `resolve` → still `found`, in the expected package |
| Delete | `refs` textual: zero hits proves it; hits → phase 2 `assert-no-callers` or a human | — | `resolve` → `not_found`; `refs` old name → zero |
| Rename | `resolve old` → found, `resolve new` → not_found | the rename mechanic is out of quarry's scope (go/types script) | `resolve old` → not_found, `resolve new` → found, `refs old` → zero |
| Move | `resolve` → current file | — | `resolve` → new file |
| any | every symbol in `Uses` and every target must resolve unambiguously, or the plan is invalid before an agent is spawned | `Uses` resolved into a pack: id, file, span, signature, doc, for a one-shot read | |

The dependency graph the format derives (`Uses` ∩ other cards' targets) is a graph over ids; `resolve`
makes every node in it checkable at plan time. The tier-1 verify scope ("packages holding the target
symbols plus packages holding callers") is the id's package in phase 1 and adds the callers' packages
in phase 2.

### 8.2 Mechanical use from Loomyard's Go code, without any LLM

- **Plan validation gate.** Every id in a plan resolves, unambiguously, before dispatch. Catches
  typos, stale names and ambiguity when they are cheap.
- **Plan pack generation.** The annex the ladder harness already builds by hand: resolved spans for
  `Uses` and the targets, injected into the implementer's prompt at dispatch. Re-resolved, never
  cached, because the previous card may have moved things.
- **Done-checks per card**, table above. A Create card whose symbol does not resolve is not done; a
  Delete card whose symbol still resolves is not done. No judgment involved.
- **Diff to symbols.** Given a diff's changed line ranges, `map` on the touched files gives the set of
  symbols the diff touched, by id. That is review scope, changelog input, and the check that a card
  changed only its declared targets.
- **Test targeting.** Textual `refs` restricted to `_test.go` files is exact enough to say which test
  files mention a changed symbol, without types.
- **Documentation drift.** `map` over a package against the symbol names a doc or a codeguide page
  claims; missing or extra names are mechanical findings.
- **Repository invariants.** Unique package basenames; every exported symbol has a doc (lesson: the
  extractor already knows); a Rename left no old-name occurrence.
- **Phase 2:** `impact` before/after sets on Edit and Rename, `assert-no-callers` on Delete, and the
  31-false-positive class of incident (`docs/research/scout-agent-usage-findings.md`) prevented by
  `verified` being read, not assumed.

### 8.3 LLM use

Only through the same surfaces. What the ladders showed: `map` on unfamiliar code is worth a tool
grant; nothing else was, in V1's shapes. Whether `resolve` and `members` earn a grant for an
implementer that already has a pack is a ladder question for after they exist.

## 9. Build order on `main`

Each step is one commit that builds and tests green on its own.

1. **Delete** (§2). `go build ./... && go test ./...` green on what remains.
2. **Types and the id.** `Symbol` gains `ID`, the owner chain, the head span; the id grammar with
   parser, printer and tests; the package-basename walk with the uniqueness check.
3. **`resolve`** in the engine, with the ambiguity/multipart distinction and tests on fixtures for
   all three languages.
4. **`map`** — the kept toc, re-keyed by id, headers and docs complete.
5. **`members`** — head computation for Python and C#; the Go case falls out of `resolve`.
6. **Extractor gaps** (§6): Python nested classes and attributes, C# partial, fields, properties.
7. **Facade**, then **CLI** (three verbs, one envelope, exit codes), then **MCP** as its mirror.
8. **Textual `refs`** with `verified: false`, and the zero-hit direction of `assert-no-callers`.
9. **Ladder.** `run-toc.sh` against the new `cmd/quarry-mcp`: `a2-toc-dir` (now `map`) must
   reproduce its separation from control. That is the regression gate for the rewrite.
10. **Phase 2 decision:** the type checker. Then `impact` and the full `assert-no-callers`.

## 10. Non-goals

- Positions as input. A `file:line:col` form does not exist on any surface.
- Fuzzy matching of any kind. Unknown is `not_found`; several is `ambiguous`.
- A daemon, an index, or a cache in phase 1.
- Languages beyond Go, Python, C#.
- Renaming or editing source. Quarry reads.
- Compact-by-default. Views are options; extraction is complete.

## 11. Open decisions

- Type checker for phase 2 (gopls vs `go/packages` in-process; what Python and C# use).
- C# overload naming (a signature suffix on the id, or ambiguity only).
- Whether Loomyard adopts `#` or keeps `.` with the mapping (§3).
- Whether `results/**/raw/` is un-ignored (carried over from `HANDOFF.md` §4).
