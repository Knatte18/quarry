# Quarry rewrite — what goes, what stays, what gets built

Decided 2026-09-03. Quarry V1 is frozen on the `v1-final` branch (worktree `wts/v1-final`), kept as a
reference. `main` is cleaned out and the rewrite is built there under the same module path,
`github.com/Knatte18/quarry`. There is no "V2" in any name: when this plan is done, what is on `main`
is quarry.

`HANDOFF.md` is the state of the rewrite and the short list of measurements this document rests
on; the research notes (`docs/research/`) are the long form. The V1 harness code is replaced (§9a),
and the results roots it produced are archived with it on `v1-final`; a `conclusion.md` is read from
that branch when a number is needed.

## 1. Why a rewrite, in four measured lessons

Every claim here has a results root or a captured output behind it; see `HANDOFF.md` §3 and
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
| `internal/quarryengine/toc`, `treesitter`, the extension→language table in `registry` | ~2 100 (+2 600 test) | **kept, Go only** — the Go extractor, comment association, sentence splitter, tree-sitter wrapper. The Python and C# extractors, their grammars and their tests are **deleted** with the rest: they stay on `v1-final`, and when those languages are added their extractors are written against the glyph contract, not extended from V1 |
| `internal/quarryengine/{daemon,lsp,query,impact,registry (rest)}`, `internal/{lock,proc}` | ~3 400 | **deleted** — the LSP layer. `impact` returns in phase 2 behind the new contract, on whichever type checker phase 2 picks |
| `internal/cli`, `internal/mcpserver`, `internal/output`, `cmd/*`, `quarry/` facade | ~4 500 | **deleted and rebuilt** from the contract, much smaller: three verbs, not seven |
| `go.mod`: every tree-sitter grammar but Go, LSP/JSON-RPC deps, `gopls` requirement | | **removed** — one language, no daemon |
| `testdata/{impactfixture,clockfixture,buildtagfixture}` | | deleted with the layer that used them |
| `bench/loomyard-eval/ladder/{cmd,internal,tools}`, `run*.sh`, `launch-session.sh`, `.claude/skills/ladder-run` | 9 000 (+8 300 test) | **deleted and rebuilt** around headless `claude -p` (§9a), ~1 000–1 500 lines |
| `ladder*.yaml`, `bench/loomyard-eval/tasks`, `docs/research/**`, `docs/toc-docstring-association.md`, `HANDOFF.md` | | kept — the fasit, the task prompts, the research record; `docs/research/output-formats/` is the "before" side T5 adds `after/` to |
| `bench/loomyard-eval/results/**`, `bench/loomyard-eval/ladder/results/**`, `bench/loomyard-eval/scripts/gen_compact_toc.py` | | **deleted** — the V1 results roots are archived on `v1-final` with the harness that produced them; `HANDOFF.md` is their summary. The Python script goes with them: no Python in this repository, no exceptions |
| `docs/mcp-setup.md`, `docs/when-to-use-quarry.md`, `docs/servers.yaml.example`, `.mcp.json` | | **deleted** — they describe the seven verbs, `cmd/quarry-mcp` and the daemon, all of which are on `v1-final`. Nothing is marked "superseded" on `main`: what is archived is deleted. T5 and T6 write the new documents and the new `.mcp.json` when the surface they describe exists |
| `README.md` | | replaced by a short stub: what the rewrite is, `docs/rewrite-plan.md`, `docs/glyph.md`, and that V1 is on `v1-final`. T5/T6 write the real one |

The first commit on `main` after this document is the deletion, so nothing half-V1 can survive into
the new code. Until the new `cmd/quarry-mcp` exists the harness cannot run; that is expected and
short.

## 3. The glyph

The identifier is specified in `docs/glyph.md`, which is the contract Loomyard's plan format adopts.
In one line: one form, `unit#member`, with the unit and the member spelled in each language's own
compiler-guaranteed alphabet.

```
internal/logger#dualHandler.stderr          Go: directory path, Type.Name
loomyard.engine.layout#Beta.Inner.handle    Python: dotted module path, Class.Inner.name
Loomyard.Engine.Layout#Renderer.Draw(int)   C#: namespace, Type.Name(param types), always
```

- Unique by construction in every language, in any repository; quarry enforces nothing and no
  naming discipline is assumed. The basename-plus-invariant, shortest-suffix, numbered and
  config-root schemes were all rejected for drifting (§2 of the spec says why).
- A glyph carries no file and no line: those are what quarry returns for it, and they move while it
  does not. Rename or move-between-units is a new glyph, deliberately — the glyph is the name.
- C# method glyphs always carry parameter types, so adding an overload never changes an existing
  glyph. Go and Python have no overloads and never pay for this.
- One implementation: package `glyph`, pure Go, no cgo, imported by Loomyard's plan parser.

**Two outcomes that must never be confused:**

| outcome | meaning | result |
|---|---|---|
| ambiguous | the glyph matches several *different* symbols (Go build-tag duplicates, a Python name defined twice in one module) | error, candidates listed, never a silent pick |
| multipart | the glyph names *one* symbol the language allows in several places (C# `partial class`, C# partial methods) | success, every part returned |

Go never produces multipart: a type is declared once and each method once, however many files they
span. That is a `members` question (§5), not a resolution question.

**Loomyard's plan format changes to match.** `manifest/designs/plan-card-format.md` today says
"package-qualified short names (`shedrecipe.Lookup`), never file:line, never full import paths".
As a glyph that is `internal/shedrecipe#Lookup`: the principle stays, the spelling changes in two places.
The unit becomes the repository-relative directory instead of the bare package name, which is the price of uniqueness in a large repository,
and the separator becomes `#`, so the parser splits unit from member without a package list.
Card types, `Uses` and the derived dependency graph do not change.
That is Loomyard work, done in Loomyard's repository once T5 (§12) has merged, and not a task here:
`planparser` imports `glyph` and drops its own name handling, the validator calls `resolve`, the planner prompt and the format document get the new spelling.

## 4. One envelope, one entry type at every depth

Every command, every surface, the same shape. Deviations are what made V1 three generations.

- **`ok` agrees with the exit code.** V1's `definition` returned `ok: true` and exited 2.
- **`status` per entry:** `found`, `not_found`, `ambiguous` (with `candidates`), `multipart`.
- **One symbol entry everywhere:** `id` (the glyph), `kind` (a word, never an LSP integer), `start`, `sigend`,
  `end` (1-based, inclusive), `signature` (verbatim source), `doc` (complete, never truncated by
  extraction), and `file` (relative to the repository root, never absolute) wherever entries can span
  files. `resolve`, `members` and `toc` all return this entry and nothing else for a symbol.
- **One file entry everywhere, carrying only what is the file's own:** `name`, `header`, the
  deviations only when present (`test`, `generated`, `package` when it differs from the
  directory's, as `logger_test` does, `language` when it differs from the directory's), and
  `symbols` — filled for a file query, absent for a directory query, filled for a directory only on
  explicit request. Everything the files share — `dir`, `package`, `language` — lives on the
  directory answer that holds them, once. A file query is a directory answer with one file entry;
  the file never repeats its parent's facts.
  Files that are not code are file entries too (below): `name`, a `header` where the format has a
  place for one, never `symbols`.
- **Shared facts once, defaults never.** Package, directory and language are stated once at the top.
  `test: false`, `generated: false`, `ok: true` inside data, empty `dirs: []`, and a directory prefix
  repeated on every path are V1 clutter: 25 files carried 100 fields that said nothing.
- **Glyphs as keys in every output**, under the JSON key `id`. What `toc` lists is what `resolve` takes.
- **`verified` per entry** on anything that claims a reference (phase 2 `impact`: `true` when the
  type checker confirmed it, `false` when it could not). A consumer that wants only verified entries
  filters; a gate that reads unverified entries as proof is a defect.
- **JSON is the contract** for the CLI and the facade. The MCP `content[].text` block carries a
  lossless text view of the same data — no keys, no defaults, prose intact. Whether that view beats
  JSON for an agent is a ladder cell, not an assumption: `results/2026-09-02-compact2` cut the
  envelope and the prose at the same time, so the envelope's own cost was never measured.

### Depth

The answer is one recursive type. A **directory answer** holds `dir`, `package`, `language`, `doc`,
its `files` and its `dirs`, and each entry in `dirs` is itself a directory answer.
`language` is the language of the `package`, present only when there is one;
it is what tells a consumer which glyph alphabet the `id`s below are spelled in.

`doc` is the **package documentation**: what this directory is for, in the language's own place for
saying so — Go's `// Package render ...` comment above the package clause, Python's `__init__.py`
module docstring, and for C# the first paragraph of a `README.md` in the directory, since the
language has no convention. It is carried whole (one paragraph) and it is the one thing a
subdirectory entry carries besides its identity, so `toc` on a parent is a list of its
subdirectories each saying what it holds. Loomyard today: 54 of 83 packages have one; the 29
without are mostly `*cli` packages. A missing `doc` is absent, not empty, and is a repository
invariant `toc` can report (§8.2). Two independent
knobs say how much of the tree is filled in:

| knob | values | default |
|---|---|---|
| `depth` | how far down the directory tree `dirs` are filled: `0` lists subdirectories by `dir`, `package` and `doc` only; `1` fills their files; `N`; `all` | `0` |
| `symbols` | whether file entries carry `symbols` | `true` for a file argument, `false` for a directory argument |

They are separate because the two big shapes need them separately: a whole tree with headers only
(orientation) and one package with every symbol (mechanical consumers). Loomyard's `internal/` is
72 directories and 394 KB at headers only, so even the header map of a whole tree is not something
an agent gets without asking.

### Files that are not code, and directories with more than one language

`toc` lists every file in the directory that is not gitignored, not only source files.
The question `toc` answers is whether a file is worth opening,
and that question is asked of `README.md`, `viewer.html` and `config.yaml` as much as of `layout.go`.
A file without a language gets `name` and a `header` taken from wherever its format keeps one:
Markdown gives its first heading and first paragraph;
HTML, CSS, JavaScript, YAML and shell give a leading comment when there is one;
anything else gives the name alone.
Such entries never carry `symbols`, and `--symbols` does not change them.

The alphabet is chosen per file, never per repository.
A repository is not assumed to hold one language:
a Python data parser inside a C# tree gets a Python glyph (`tools.parse_data#load`) and the C# code around it gets C# glyphs, in the same `toc` answer.
A file whose language differs from its directory's `package` language says so with `language` on the file entry;
a directory with no `package` says nothing and its files each carry their own.

A file is itself a target.
What `toc` lists, `resolve` takes, and `toc` lists file names,
so a plan card whose target is a whole file — the HTML viewer an agent is to create, a Markdown page — names it by its repository-relative path and `resolve` answers `found` or `not_found` with the file entry (§5).
A target without `#` is a path; a target with `#` is a glyph.
Nothing else is needed for non-code deliverables, and `plan-card-format.md`'s rule "never file:line" stands: a path is not a line.

### Examples, real data (Loomyard 72c23d9)

`toc internal/reedengine/render` — a directory: its files with headers, nothing below:

```json
{ "dir": "internal/reedengine/render", "package": "render", "language": "go",
  "doc": "Package render owns the closed display vocabulary and the deterministic Rules(strands, box, params) -> (layout, focus) function that turns a set of strands into a tmux window_layout string. It is a pure leaf: no I/O, no tmux, no engine import.",
  "files": [
    { "name": "checksum.go", "header": "checksum.go computes tmux's window_layout checksum: a 16-bit rotate-right-1 accumulator over the layout body ..." },
    { "name": "checksum_test.go", "test": true, "header": "checksum_test.go pins a known-good layoutChecksum fixture from live psmux testing, ..." },
    { "name": "focus.go", "header": "focus.go resolves which pane receives tmux input focus, and detects whether a strand has a descendant present ..." },
    { "name": "layout.go", "header": "layout.go is the layout mechanics layer: it turns a resolved, ordered list of pane placements within a Box into a tmux/psmux window_layout body and its checksum-prefixed full string. It is region-relative — ..." },
    ... 12 files ] }
```

`toc internal/reedengine` — a directory with a subdirectory. At `depth: 0` the subdirectory is an
entry with its identity and its package doc, nothing else:

```json
{ "dir": "internal/reedengine", "package": "reedengine", "language": "go",
  "doc": "Package reedengine is the domain kernel for lyx's tmux window manager: the tmux subprocess overlay, strand bookkeeping, persisted state, config, and (in the operations layer) the lifecycle verbs that compose them. ...",
  "files": [ ... 67 files ... ],
  "dirs": [
    { "dir": "internal/reedengine/render", "package": "render",
      "doc": "Package render owns the closed display vocabulary and the deterministic Rules(strands, box, params) -> (layout, focus) function that turns a set of strands into a tmux window_layout string. It is a pure leaf: no I/O, no tmux, no engine import." } ] }
```

`toc internal` — the orientation map: no files of its own, every package under it with its doc.
The 29 packages without one show as `dir` and `package` alone, which is the finding.

`toc internal/reedengine --depth 1` — the same, with the subdirectory filled the way the first
example is; `--depth all` keeps going down.

`toc internal/reedengine/render/layout.go` — a file: a directory answer with one file entry, and
that entry carries `symbols` because the argument was a file:

```json
{ "dir": "internal/reedengine/render", "package": "render", "language": "go", "doc": "Package render owns ...",
  "files": [
    { "name": "layout.go",
      "header": "layout.go is the layout mechanics layer: it turns a resolved, ordered list of pane placements\nwithin a Box into a tmux/psmux window_layout body and its checksum-prefixed full string.\nIt is region-relative — offsets are anchored to box.X/box.Y rather than the whole window — so the\nstack region can be rendered independently of the Box it is placed within.\nThis file makes no placement or height decisions;\nthose live in policy.go and height.go.\nIt only renders the string from the placements it is given.",
      "symbols": [
        { "id": "internal/reedengine/render#placement", "kind": "type", "start": 16, "sigend": 20, "end": 29,
          "signature": "type placement struct",
          "doc": "placement is one resolved pane: its tmux pane id and the row height it\nhas been assigned." },
        { "id": "internal/reedengine/render#buildStackBody", "kind": "function", "start": 31, "sigend": 34, "end": 50,
          "signature": "func buildStackBody(box Box, panes []placement) string",
          "doc": "buildStackBody renders panes into a tmux window_layout body positioned\nwithin box: \"<box.W>x<box.H>,<box.X>,<box.Y>[<w>x<h>,<x>,<y>,<paneNum>,...]}\"." },
        { "id": "internal/reedengine/render#wrapLayout", "kind": "function", "start": 52, "sigend": 54, "end": 56,
          "signature": "func wrapLayout(body string) string",
          "doc": "wrapLayout prefixes body with its tmux layout checksum, producing the full\nwindow_layout string tmux's select-layout accepts." },
        { "id": "internal/reedengine/render#bandHeader", "kind": "function", "start": 58, "sigend": 63, "end": 76,
          "signature": "func bandHeader(fullBox Box, headerPaneID string, headerHeight int, ...) ...",
          "doc": "..." } ] } ] }
```

`toc internal/reedengine/render --symbols` — every one of the 12 files shaped like `layout.go`
above. `toc internal --depth all --symbols` is the whole repository, every symbol: the shape a
diff-to-symbols or documentation-drift check wants in one call.

The same directory answer as lossless text, for the MCP block — no keys, no defaults, prose intact:

```
internal/reedengine/render (package render, go), 12 files
Package render owns the closed display vocabulary and the deterministic Rules(strands, box, params) -> (layout, focus) function that turns a set of strands into a tmux window_layout string. It is a pure leaf: no I/O, no tmux, no engine import.
checksum.go: checksum.go computes tmux's window_layout checksum: a 16-bit rotate-right-1 accumulator ...
checksum_test.go [test]: checksum_test.go pins a known-good layoutChecksum fixture from live psmux testing, ...
focus.go: focus.go resolves which pane receives tmux input focus, and detects whether a strand ...
...
```

and a file with symbols:

```
internal/reedengine/render/layout.go (package render, go): layout.go is the layout mechanics layer: ...
16-29 (sig 16-20) internal/reedengine/render#placement: type placement struct
    placement is one resolved pane: its tmux pane id and the row height it has been assigned.
31-50 (sig 31-34) internal/reedengine/render#buildStackBody: func buildStackBody(box Box, panes []placement) string
    buildStackBody renders panes into a tmux window_layout body positioned within box: ...
```

## 5. The queries

Phase 1 is `resolve`, `members` and `toc`: the same tree-sitter parse with three entry points. None
needs a type checker, a daemon, or an index. Phase 1 says nothing about callers.

**`resolve <glyph|path>...`** — where is this, right now. Per glyph: location(s) and status. Called by an
implementer immediately before every read and every edit, because lines have moved since the last
time. Many glyphs in one call are grouped by unit and each unit is parsed once.
Three details of the answer exist for the validator (§8.1):

- A `not_found` says whether the *unit* exists: `unit: found` when the directory, module or namespace is there and only the member is missing, `unit: not_found` otherwise.
  A Create card needs the first and a typo in the unit produces the second, and the two are not the same finding.
- An argument without `#` is a repository-relative path, file or directory, and resolves to its file entry or directory answer with `found` or `not_found`.
  This is what makes a non-code deliverable (§4) a checkable target.
- In a repository with more than one language, a glyph is tried against each alphabet present.
  Go units contain `/` and collide with nothing; a Python module path and a C# namespace can spell the same string, and when both match the answer is `ambiguous` with the candidates marked by language.
  There is no language prefix in the glyph; the case is rare and the status is honest.

**`members <glyph>`** — what does this type consist of, across files. The glyph must name a type;
on any other kind the answer is `ok: false` naming the kind. The type's *head* (its own
lines: declaration, doc, fields, class-level attributes — in Go the `type` block; in Python and C#
the class span minus its member spans) and every member with its location, sorted by file and line.
One line per member; the caller chooses what to read. This is how a large class is handled: the plan
names members, `members` gives the map, the implementer reads head + the named member + siblings it
picks. The whole class is `start`–`end` of the type symbol and is available, never the default.

**`toc <dir|file>... [--depth N|all] [--symbols]`** — what is here: a table of contents. V1's
`toc dir` and `toc file` as one verb over one recursive answer (§4 Depth): a directory answers with its files and its
subdirectories' identities, a file with its entry and symbols, and the two knobs fill in more of the
tree or more of each file. The measured win. Headers and docs complete (lesson 4).

**No reference query in phase 1.** A textual search for a name was considered and dropped: on a
name shared by several methods (`Apply`, `Run`, `Handle`) it never returns zero and its hits mean
nothing, and on a unique name grep is as good. Anything about callers waits for the type checker.

**`impact <glyph>` (phase 2)** — who calls this, type-resolved. Tree-sitter cannot do this: at a call
site `x.Run()` the receiver type is not in the syntax, and guessing it is the silent pick §3 forbids.
Needs a type checker. The open decision is which: gopls as in V1, or `go/packages` in-process for Go
(exact, no daemon, no protocol, Go-only; Python and C# would need a language server when they need
this). Decided when phase 2 starts, not now. `impact` returns each caller as a glyph plus location plus
`verified: true`.

**`assert-no-callers <glyph>` (phase 2)** — the exit-code contract for a Delete card's verify step.

Performance, measured 2026-09-03 in-process on a 4-core WSL2 host, Loomyard, every file read fresh
from disk, no cache:

| operation | time |
|---|---|
| walk repository, list of package directories (83 packages) | 2 ms |
| resolve one glyph, 5-file package (32 KB) | 5–8 ms |
| resolve one glyph, 35-file package (317 KB), serial / name-prefiltered / parallel | 65 / 24 / 26 ms |
| 20 glyphs across 5 packages, grouped, serial / packages in parallel | 109 / 76 ms |
| every symbol in the repository, 469 files, serial / parallel | 616 / 237 ms |

Nearly all of it is cgo into tree-sitter (parse 47 of the 65 ms; node access and allocation
callbacks the rest). Nothing on quarry's side is worth optimising yet. No cache in phase 1: every
answer reads the source as it is at that moment, which is the point. If one is ever needed it is
keyed on (path, mtime, size) per file and parses only what changed.

## 6. Languages

**Go. Only Go.** Other languages are added one at a time, when wanted, after the Go surface is
measured (§9). Python and C# are specified in `docs/glyph.md` so the form is known to hold for
them; their V1 extractors are deleted with the rest (§2) and are written fresh against the glyph
contract when their turn comes. Each language's extractor must deliver, per symbol: kind, name,
owner chain, signature, doc, start/sigend/end, and the unit. Known gaps, in the Go extractor that
is kept and in the V1 Python and C# extractors on `v1-final`, for whoever picks them up:

| language | gap today | needed for |
|---|---|---|
| Go | external test package (`logger_test`) not distinguished from the package; several `init` must resolve `multipart` | unit identity, `resolve` |
| Python | nested classes dropped (`Beta.Inner` and its methods absent; `Owner` is one level) | glyph uniqueness, `members` |
| Python | class-level attributes are not symbols | the type head in `members` |
| C# | `partial` not recognised (the word `partial` in toc today means "lossy parse" — rename that field) | multipart resolution |
| C# | fields and properties not emitted as members | `members` |
| C# | overloads share a glyph | ambiguity reporting now; a signature suffix later if a real plan needs to name one overload |
| Go | none known; build-tag duplicates must report as ambiguous | |
| all | package doc is not extracted (Go `// Package x` block, Python `__init__.py` docstring, C# `README.md` first paragraph) | `doc` on the directory answer |

## 7. Surfaces, in order of importance

1. **The `glyph` package.** The one implementation of the name. Importable by anything, Loomyard's
   plan parser first, without the engine.
2. **The Go facade** (`quarry/` package). The primary consumer is Loomyard's own Go code, never the
   LLM. Typed results, no JSON round-trip, grammars loaded once per process.
3. **The CLI.** Same queries, JSON out, exit codes for gates. The scripting contract.
4. **MCP.** A mirror of the CLI for an LLM that has the tools granted, and kept thin:
   only the tools a ladder cell measures, `toc` first. Tool names are verbs, never protocol methods.

## 8. How Loomyard uses it

### 8.1 Plan cards (`manifest/designs/plan-card-format.md`), by card type

| card | before dispatch (mechanical, plan-time) | during implementation (the agent) | after (mechanical, done-check) |
|---|---|---|---|
| Create | `glyph.Parse` for the form; `resolve` target → must be `not_found` with `unit: found`, or the unit is itself a Create in the plan; for `Type.Name` the owner `unit#Type` → `found` or a Create in the plan; `toc`/`members` on the package for "nothing equivalent exists" | `toc` the package it goes into | `resolve` → `found` |
| Edit | `resolve` target → `found`; phase 2 `impact` → `ImpactSummary` and tier-1 test package set | `resolve` before each read/edit; `members` when the target is a type | `resolve` → still `found`, in the expected package |
| Delete | `resolve` target → `found`; phase 2 `assert-no-callers`; until then Loomyard's degraded mode (scoped grep, a human) | — | `resolve` → `not_found` |
| Rename | `resolve old` → found, `resolve new` → not_found | the rename mechanic is out of quarry's scope (go/types script) | `resolve old` → not_found, `resolve new` → found |
| Move | `resolve` → current file | — | `resolve` → new file |
| any | every glyph in `Uses` and every target must resolve unambiguously, or the plan is invalid before an agent is spawned; a target without `#` is a path (an HTML viewer, a Markdown page) and follows the same rows with the file entry in place of the symbol | `Uses` resolved into a pack: glyph, file, span, signature, doc, for a one-shot read | |

**How the planner gets the glyphs right.** It never composes a glyph for an existing symbol: every
symbol it knows about came from `toc`, whose lines carry the glyph verbatim, so Edit, Delete,
Rename-from, Move and `Uses` are copied, not spelled. Only Create composes one — an existing unit
from `toc` plus a new name — and that is what validation catches. **The planner validates before
it hands off, with the same code the mechanical gate runs.** Loomyard exposes its own plan
validator to the planning agent as a tool; it calls quarry's facade underneath, and the final
mechanical gate is the same function. Passing one is passing the other, so a plan that reaches
the gate never fails it on a glyph, and no fresh agent is spawned to fix what the running one
could have. The validator's contract per glyph has three layers, and only the last needs the symbol to exist:

1. **Syntactic, no source read.** `glyph.Parse(lang, s)` checks the form against the alphabet — one `#`, a well-formed unit for the language, a well-formed member — and returns the canonical spelling, so `Draw (int)` comes back as `Draw(int)`.
   The language is the card's: for an existing unit it is what `toc` reported, for a unit the plan creates the card says it, since there is no directory to look at.
2. **Structural, against what exists around the target.** `resolve` — Edit, Delete, Rename-from, Move and `Uses` must be `found`; Create and Rename-to must be `not_found` with `unit: found`, or in a unit that is itself a Create in the plan; a Create of `Type.Name` needs its owner `unit#Type` found or created in the plan; `ambiguous` is always a rejection with its candidates.
3. **Plan-internal.** Two Create cards with one glyph, or a Create that collides with another card's Rename-to, are rejected without asking quarry.

All glyphs of a draft go in one `resolve` call. The stencil's rule is one line: a plan is handed off with the
validator's last answer green; the gate rejects everything else.

The dependency graph the format derives (`Uses` ∩ other cards' targets) is a graph over glyphs;
`resolve` makes every node in it checkable at plan time. The tier-1 verify scope ("packages holding
the target symbols plus packages holding callers") is the glyph's unit in phase 1 and adds the callers' packages
in phase 2.

### 8.2 Mechanical use from Loomyard's Go code, without any LLM

- **The execution DAG.** The plan format already derives its edges (`Uses` ∩ other cards'
  targets). With glyphs as nodes every edge is checkable by `resolve` before anything runs, so
  which cards are safe to execute in parallel — disjoint symbol sets, each in its own worktree — is
  a mechanical answer, not a judgment. Phase 2 tightens it with `impact`: two cards that touch
  disjoint symbols but share a caller are still not independent.
- **Plan validation gate.** Every glyph in a plan resolves, unambiguously, before dispatch. Catches
  typos, stale names and ambiguity when they are cheap.
- **Plan pack generation.** The annex the ladder harness already builds by hand: resolved spans for
  `Uses` and the targets, injected into the implementer's prompt at dispatch. Re-resolved, never
  cached, because the previous card may have moved things.
- **Done-checks per card**, table above. A Create card whose symbol does not resolve is not done; a
  Delete card whose symbol still resolves is not done. No judgment involved.
- **Diff to symbols.** Given a diff's changed line ranges, `toc` on the touched files gives the set of
  symbols the diff touched, by glyph. That is review scope, changelog input, and the check that a card
  changed only its declared targets.
- **Documentation drift.** `toc` over a package against the symbol names a doc or a codeguide page
  claims; missing or extra names are mechanical findings.
- **Repository invariants.** Every package has a package doc (29 of 83 do not, today); every
  exported symbol has a doc. The extractor already knows both.
- **Phase 2:** `impact` before/after sets on Edit and Rename, `assert-no-callers` on Delete, test
  targeting by caller package, and the
  31-false-positive class of incident (`docs/research/scout-agent-usage-findings.md`) prevented by
  `verified` being read, not assumed.

### 8.3 LLM use

Only through the same surfaces. What the ladders showed: `toc` on unfamiliar code is worth a tool
grant; nothing else was, in V1's shapes. Whether `resolve` and `members` earn a grant for an
implementer that already has a pack is a ladder question for after they exist. The planner is the
one agent with a settled tool set: `toc` to see what exists, and Loomyard's validator (over
`resolve`) to check every glyph it wrote, in one call, before handing off (§8.1).

## 9. Build order on `main`

Each step is one commit that builds and tests green on its own. The harness rewrite (§9a) lands
before step 8.

1. **Delete** (§2). `go build ./... && go test ./...` green on what remains.
2. **The `glyph` package** — pure Go, no cgo, no dependencies: parser, printer, the Go alphabet,
   tests, per `docs/glyph.md`; the structural split at `#` already accepts the other two. Then `Symbol` gains its glyph, the owner chain, the head
   span, and the unit walk (directory → Go unit; source root → Python module; namespace → C#).
3. **`resolve`** in the engine, with the ambiguity/multipart distinction and tests on Go fixtures.
4. **`toc`** — the kept extractor, re-keyed by glyph, headers and docs complete.
5. **`members`** — the Go head falls out of `resolve`.
6. **Go extractor gaps** (§6): external test packages, `init` as multipart, package doc.
7. **Facade**, then **CLI** (three verbs, one envelope, exit codes), then a thin **MCP**: only
   the tools the ladder measures, `toc` first.
8. **Ladder.** `run-toc.sh` against the new `cmd/quarry-mcp`: `a2-toc-dir` must
   reproduce its separation from control. That is the regression gate for the rewrite.
9. **The next language, when wanted:** its glyph alphabet in `glyph`, an extractor written against
   the contract, its `members` head. Python and C# are designed now, coded after 8, one at a time.
10. **Phase 2 decision:** the type checker. Then `impact` and the full `assert-no-callers`.

## 9a. The harness is rewritten too

`bench/loomyard-eval/ladder/` is 17 000 lines (9 000 non-test) and is coupled to V1 in three
places — it requires the seven V1 tool names in its yaml, warms the daemon through
`workspace_symbol`, and builds `cmd/quarry-mcp` with V1's flags — so it cannot run against the
rewrite unchanged. Its size is architectural, not incidental: it drives an *interactive* Claude Code
session in tmux, which runs a skill, which dispatches a subagent, and then reconstructs what
happened by locating the subagent's transcript under `~/.claude/projects`. Every gate and every
rule it carried exists because the harness does not control the run: the prompt gate, the
outcome marker, the one-tmux-session rule, the transcript hunt, the static per-cell agent
definitions, the post-hoc turn ceiling, the unusable `output_tokens`.

**Probe, 2026-09-03, Claude Code 2.1.259, this host:** headless `claude -p` gives the harness
direct control of everything the old one inferred.

| need | `claude -p` | verified |
|---|---|---|
| the exact prompt | positional argument | yes |
| the MCP server, and only it | `--mcp-config <file> --strict-mcp-config` | connected, listed in the `system` record |
| the tool allowlist | `--allowedTools mcp__quarry__toc_dir --tools ""` | `toc_dir` ran; `toc_file` was denied and recorded in `permission_denials` |
| the turn ceiling | `--max-turns N` (accepted, not in `--help`) | `terminal_reason: max_turns` |
| the transcript | `--output-format stream-json --verbose` on stdout | every assistant and tool record, with usage |
| usage, including `output_tokens` | per-message `usage` and a final `result` with `num_turns`, `duration_ms`, `total_cost_usd` (a list-price estimate, `costBasis: list`), `modelUsage` | yes — `output_tokens` was unusable in the old transcripts |
| model and effort | `--model claude-sonnet-5 --effort medium` | both accepted; effort is not echoed in the result, so the harness records the flags it passed |
| no session residue | `--no-session-persistence` | yes |
| stdin | must be `< /dev/null`, or a warning lands in the stream | yes |

`modelUsage` in the final record also shows a little Haiku usage — Claude Code's own overhead, not
the run — so metrics come from the assistant records, which carry the model, never from that total.

The V1 README retired an earlier `claude -p` port for being "a headless subprocess the operator
cannot see inside". That is `tee` to a log file and `tail -f`. No other reason is recorded.

**The new harness** is one Go program of roughly 1 000–1 500 lines with tests, replacing
`cmd/ladderbench`, `internal/ladder`, `tools/runmatrix`, `run*.sh`, `launch-session.sh` and
`.claude/skills/ladder-run`. Per cell and repetition it: restores the task worktree at the pinned
commit; writes the MCP config for the cell; runs `claude -p` with the task prompt and the flags
above, tee-ing the stream to `results/<root>/<cell>/<rep>/transcript.jsonl`; computes the metrics
from the stream (turns, tool calls and bytes, Read bytes, greps, cache and output tokens); runs the
scorer the same way, Opus against the fasit, JSON out; writes `summary.json`, `provenance.json`
(quarry commit, dirty flag, server binary hash, `claude --version`, host) and the table. Resumable
by the existence of a rep's result file. Two gates survive because the CLI cannot enforce them:
the cell used the tool it was granted (the `none` arm called nothing), and the control cells' blinding
check. Everything else the CLI now guarantees.

**Kept:** the yaml shape (cells, tasks, pins, fasit, reps, models) and the scorer prompt and
schemas. The V1 results roots are on `v1-final`. Two of the old harness rules still apply
(`HANDOFF.md` §3): do not edit source mid-matrix (the binary hash per rep stays), and cost numbers
are comparable only within one root. The other rules are retired with the architecture that needed
them.

**Order:** the harness lands before §9 step 8, since step 8 is its first run.

## 10. Non-goals

- Positions as input. A `file:line:col` form does not exist on any surface.
- Fuzzy matching of any kind. Unknown is `not_found`; several is `ambiguous`.
- A daemon, an index, or a cache in phase 1.
- Languages beyond Go, until a language is wanted. Python and C# are designed (`docs/glyph.md`) and
  are the first two candidates; nothing else is designed.
- Renaming or editing source. Quarry reads.
- Compact-by-default. Views are options; extraction is complete.

## 11. Open decisions

- Type checker for phase 2 (gopls vs `go/packages` in-process; what Python and C# use).
- C# long parameter lists: whether to cap a method glyph at N types plus a hash, decided only
  after measuring a real C# repository (`docs/glyph.md` §3).
- Whether the new harness's `results/**/raw/` is committed or ignored; T2 decides when it writes
  its first root.

## 12. Work breakdown — one mill task per row

Each task gets its own wiki entry, worktree and agent. `after` is the hard dependency; tasks in the
same wave have none between them and run in parallel. Every task ends with `go build ./... && go
test ./...` green in its worktree and one merge to `main`.

| wave | task | scope | after | done when |
|---|---|---|---|---|
| 0 | **T0 delete V1** | remove the LSP layer, the seven-verb CLI and MCP, the facade, the fixtures, the V1 harness code and skill, every grammar but Go and the LSP deps from `go.mod`, the Python and C# extractors and their tests, the V1 docs and `.mcp.json` of §2, the V1 results roots and the Python script; README stub; keep `toc`, `treesitter` and the extension table reduced to Go, yaml, tasks, `docs/research`, the two rewrite documents | — | tree builds and tests green with only the Go extractor; nothing under `internal/` references a daemon or a language other than Go |
| 1 | **T1 glyph package** | `glyph/`: pure Go, no deps; structural split at `#`; the Go alphabet (unit path, `_test` unit, `Type.Name`, `init`); `Parse(lang, s)`, `String`, canonical form; table tests from `docs/glyph.md` §1–§3 including rejects | T0 | every example and corner case in the spec is a test; `go list -deps` shows no cgo |
| 1 | **T2 harness** | `bench/loomyard-eval/ladder/`: one Go program around `claude -p` per §9a; yaml loader for the kept shape; worktree pin; MCP config; stream-json capture and metrics; scorer; `summary.json`, `provenance.json`, table; resume; the two surviving gates. Integration test against a stub MCP server, not quarry | T0 | a `reps: 1` run of `ladder-toc.yaml` cell `a0-none` completes end to end on this host, and the metrics match the transcript by hand |
| 2 | **T3 engine core** | `Symbol` gains glyph, owner chain, head span; the Go unit walk (directory → unit, external test package, several `init`); package-doc extraction; the recursive directory answer of §4 with `depth`/`symbols`, non-code files with headers; `toc` in the engine re-keyed by glyph; T3 also settles the engine package layout, since `resolve` and `members` join `toc` next: one package, files per concern, never a package per verb | T1 | `toc` on `internal/reedengine/render` and on `layout.go` reproduce the §4 examples byte for byte, apart from prose; **round trip over all of Loomyard:** every declaration `toc` lists has a glyph, `resolve` of each returns exactly that span, zero misses, zero extras |
| 3 | **T4 resolve + members** | `resolve` with `found`/`not_found`/`ambiguous`/`multipart` (build tags, `init`), `unit: found|not_found` on a miss, paths without `#` as targets; grouping by unit; `members` with the Go head; ordering guarantees; timing test against Loomyard kept as a benchmark | T3 | glyph.md §5 statuses each have a fixture; `resolve` of twenty glyphs across five units under 150 ms on this host |
| 4 | **T5 facade + CLI** | `quarry/` facade with typed results; CLI verbs `toc`, `resolve`, `members` over one envelope; `ok` = exit code; relative paths; JSON and the lossless text view; golden tests on the Loomyard commands from `docs/research/output-formats/` | T4 | `docs/research/output-formats/` gets an `after/` directory with the new outputs for the same commands |
| 5 | **T6 MCP, thin** | `cmd/quarry-mcp` with only what the ladder measures: one `toc` tool over the facade, JSON in `content[].text`; no text view, no `resolve` or `members` until a ladder cell measures them | T5 | the harness probe of §9a runs against it: connect, `toc` call, allowlist denial |
| 6 | **T7 ladder** | run `ladder-toc.yaml` (`a0-none`, `a2-toc-dir`) with T2 against T6, reps 5; write `results/<date>-toc/conclusion.md` | T2, T6 | `a2` separates from control on turns and cache_read as in `v1-final:bench/loomyard-eval/ladder/results/2026-09-02-toc/conclusion.md`, or the conclusion says why not |
| 7 | **T8 type checker** | decide gopls vs `go/packages`; `impact`, `assert-no-callers`, `verified` per entry; the DAG tightening of §8.2 | T7 | a Delete card's gate refuses on a caller found through an interface |

Wave 1 is the parallel one. T2 is the long pole of wave 1 and overlaps waves 2–5; it
needs no quarry code until T7.

**Not tasks yet.** Go only, until T7 has run.
A second language becomes a task when it is wanted, one language per task:
its alphabet in `glyph/`, an extractor written against the contract (the V1 one on `v1-final` is reference, not a starting point), its `members` head, its package doc source.
The done criterion is the T3 round trip over a real repository in that language at 100 %;
for C#, built-in aliases (`Int32` → `int`) canonicalised by a fixed table.
Loomyard's adoption of glyphs (§8) is work in Loomyard's repository after T5 merges, never a task in this one.
