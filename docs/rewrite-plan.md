# Quarry — the design and its decisions

Decided 2026-09-03; built as T0–T7 and merged on `main` by 2026-09-04. V1 is frozen on
`v1-final` (worktree `wts/v1-final`) together with its results roots and the harness that
produced them. `docs/glyph.md` is the identifier contract. `HANDOFF.md` is current state.
`docs/roadmap.md` is what happens next. This document is the distilled record of what quarry
is and why; the full plan text, the deletion inventory and the wave-by-wave task table are in
this file's git history.

## 1. What quarry is

One identifier — the glyph, `unit#member` — and three queries over one tree-sitter parse:
`toc` (what is here), `resolve` (where is this, right now), `expand` (what does this type
consist of). Go only. No daemon, no index, no cache: every answer reads the source as it is at
that moment. Quarry reads; it never edits.

Surfaces, in order of importance: the `glyph` package (importable without the engine, Loomyard's
plan parser first); the `quarry/` Go facade (the primary consumer is Go code, never the LLM);
the CLI (same queries, JSON out, exit codes for gates); MCP (`cmd/quarry-mcp`, a thin mirror —
only the tools a ladder cell measures, `toc` first).

## 2. The measured record

Every claim has a results root behind it (`v1-final` roots, and `results/2026-09-04-toc` on
`main`).

1. **`toc` on a directory was V1's one separated tool** (August: turns 8→4, cache_read
   127k→83k, correctness unchanged) — **and T7's clean rerun came back flat-to-reversed**
   (n=5, no cost metric separated, tool used twice in every rep, correctness unchanged).
   The two roots are cited together, never the August win alone; which of host, harness, CLI
   version, model version or small-n noise moved is undetermined.
2. **Every LSP-shaped tool** (definition, references, symbol) sat flat with or below the grep
   control, in every run since August.
3. **A lossy compact view cut bytes 4× and cost precision 0.96→0.82.** Extraction is complete;
   views filter; no view is ever forced.
4. **Identity and location are two things.** A plan needs a stable name; an implementer needs
   where that is right now, which moves with every edit. V1 answered both with
   `{file, line, character}`, an editor's shape, useless to a caller with no cursor.

Three harness rules carry into every matrix: never edit the code under test mid-matrix (binary
hash per rep in `provenance.json`); cost numbers compare only within one results root; and a
harness change is proven only by a non-control cell completing end to end — a control cell's
empty MCP config never exercises the rung path (learned 2026-09-04, when the harness's first
real MCP cell failed structurally).

## 3. The glyph

The contract is `docs/glyph.md`. One form, `unit#member`, spelled in each language's own
compiler-guaranteed alphabet (`internal/logger#dualHandler.stderr`;
`loomyard.engine.layout#Beta.Inner.handle`; `Loomyard.Engine.Layout#Renderer.Draw(int)`).
A glyph carries no file and no line: those are what quarry returns for it, and they move while
it does not. One implementation: package `glyph`, pure Go, no cgo.

Two outcomes that must never be confused:

| outcome | meaning | result |
|---|---|---|
| ambiguous | the glyph matches several *different* symbols | error, candidates listed, never a silent pick |
| multipart | the glyph names *one* symbol the language allows in several places (C# `partial`) | success, every part returned |

## 4. The envelope

Every command, every surface, the same shape:

- **`ok` agrees with the exit code.** `status` per entry: `found`, `not_found`, `ambiguous`
  (with `candidates`), `multipart`.
- **One symbol entry everywhere:** `id` (the glyph), `kind` (a word, never an LSP integer),
  `start`/`sigend`/`end` (1-based, inclusive), `signature` (verbatim), `doc` (complete, never
  truncated), `file` where entries can span files.
- **One file entry everywhere:** `name`, `header`, deviations only when present (`test`,
  `generated`, `package`, `language`), `symbols` when asked. Shared facts (`dir`, `package`,
  `language`, the package `doc`) are stated once on the directory answer; defaults are never
  written.
- **The answer is one recursive type:** a directory answer holds its files and its `dirs`, each
  of which is itself a directory answer. Two knobs: `depth` (how far down `dirs` are filled;
  default `0`) and `symbols` (whether file entries carry symbols; default `true` for a file
  argument, `false` for a directory).
- **`toc` lists every non-gitignored file, not only source.** A file without a language gets
  `name` and a `header` from wherever its format keeps one; never `symbols`. The alphabet is
  chosen per file, never per repository. `toc` takes a path; `resolve` takes a glyph, and a
  trailing `#` addresses a unit or a file as itself.
- **Glyphs as keys in every output**, under `id`. `toc`'s file and directory entries carry no
  identifier of their own — only a symbol entry does — so once `resolve` takes glyphs only, a
  consumer holding a listing cannot feed a file entry back to `resolve` without concatenating the
  directory, a slash, the name and a `#` by hand, which is the printing the one-implementation
  rule forbids outside package `glyph`. The round trip is made two ways instead: a symbol entry's
  `id` is already a glyph, and a file or directory entry's self glyph is built from its path with
  the `Self` constructor.
- **JSON is the contract** for CLI and facade; the MCP `content[].text` block is a lossless text
  view of the same data. MCP spells whole-tree depth `-1` where the CLI says `--depth all`.

The byte-level record of every output shape is `docs/research/output-formats/after/`, pinned by
`internal/cli`'s golden tests against Loomyard `72c23d9`.

## 5. The queries

**`resolve <glyph>`** — location(s) and status, one target per CLI call (one invocation,
one answer, one exit code); the facade keeps the engine's multi-target `Resolve(targets
[]string)` so the validator (§7) batches many glyphs, grouped by unit, each unit parsed once.
A bare path is rejected pre-resolution with a message naming the fix — append the trailing `#`;
that trailing `#` is how a whole file is a plan target. A `not_found` says whether the *unit*
exists (`unit: found|not_found`) — a Create card needs the first, a typo produces the second.
Negative answers render the payload (`unit`, `candidates`, `reason`) with exit 1; the error
envelope is only for usage/internal errors with no payload.

**`expand <glyph>`** — the type's head plus every member with location, sorted by file and
line. `toc`'s question asked of an owner instead of a container; it stays a separate verb, and
`toc` never takes a glyph — a Go type's methods live in other files and belong to no file's
toc.

**`toc <dir|file> [--depth N|all] [--symbols]`** — a table of contents over the recursive
answer of §4. One target per call. A `#` in any path segment is rejected as an explicit error,
never reclassified as a glyph.

**No reference query in phase 1.** A textual search on a shared name never returns zero and
its hits mean nothing; on a unique name grep is as good. Anything about callers waits for the
type checker: at a call site `x.Run()` the receiver type is not in the syntax, and guessing is
the silent pick §3 forbids. **Phase 2** (parked — `docs/roadmap.md`): `impact <glyph>` (each
caller as glyph + location + `verified: true`) and `assert-no-callers <glyph>` (the exit-code
contract for a Delete card).

Performance (2026-09-03, 4-core WSL2, Loomyard, no cache): resolve one glyph in a 35-file
package 24–65 ms; twenty glyphs across five packages 76–109 ms; every symbol in the repository
237–616 ms. Nearly all of it is cgo into tree-sitter; nothing on quarry's side is worth
optimising yet.

## 6. Languages

**Go. Only Go.** Python and C# are specified in `docs/glyph.md` so the form is known to hold;
their extractors are written fresh against the contract when a language is wanted (the V1 ones
on `v1-final` are reference, not a starting point). Each extractor delivers, per symbol: kind,
name, owner chain, signature, doc, start/sigend/end, and the unit. Known gaps for whoever picks
them up:

| language | gap | needed for |
|---|---|---|
| Python | nested classes dropped; class-level attributes not symbols | glyph uniqueness, `expand` |
| C# | `partial` not recognised; fields/properties not members; overloads share a glyph | multipart, `expand`, ambiguity |
| all (V1) | package doc not extracted | `doc` on the directory answer |

## 7. How Loomyard uses it

**Plan cards.** Every glyph in a plan resolves, unambiguously, before an agent is spawned.
The planner never composes a glyph for an existing symbol — every symbol it knows came from
`toc`, whose lines carry the glyph verbatim; only Create composes one, and that is what
validation catches. The validator (exposed to the planning agent as a tool, and the same
function the mechanical gate runs) has three layers: syntactic (`glyph.Parse`, no source
read), structural (`resolve` — Edit/Delete/Rename-from/Move/`Uses` must be `found`; Create and
Rename-to `not_found` with `unit: found` or created by the plan; `ambiguous` always rejects),
and plan-internal (colliding Creates rejected without asking quarry). All glyphs of a draft go
in one facade `Resolve` call.

**Mechanical use, no LLM:** the execution DAG (`Uses` ∩ targets, every edge checkable);
done-checks per card (a Create that does not resolve is not done; a Delete that still resolves
is not done); plan-pack generation (resolved spans injected at dispatch, re-resolved, never
cached); diff-to-symbols; documentation drift; repository invariants (every package a doc,
every exported symbol a doc).

**LLM use:** only through the same surfaces, and only tools a ladder cell measures. The
planner's settled set is `toc` plus Loomyard's validator; everything else is a ladder question.

Loomyard's own adoption (planparser imports `glyph`, the format's spelling changes) is work in
Loomyard's repository — see `docs/roadmap.md`.

## 8. The harness

`bench/loomyard-eval/ladder/`: one Go program around headless `claude -p` — exact prompt,
`--mcp-config`/`--strict-mcp-config`, `--allowedTools`, `--max-turns`, stream-json transcript,
`--no-session-persistence`, `--setting-sources ""` — with worktree pinning, metrics from the
stream, an Opus scorer against the fasit, `summary.json`/`provenance.json`/table, and resume.
Two gates survive because the CLI cannot enforce them: the cell used the tool it was granted,
and the control cells' blinding check. The three harness rules are in §2. Raw run data
(`results/**/raw/`) is never committed (settled by T7: resolved memory paths are machine
paths, and the committed artifacts fully summarise the tree).

## 9. Non-goals

- Positions as input: no `file:line:col` form exists on any surface.
- Fuzzy matching of any kind. Unknown is `not_found`; several is `ambiguous`.
- A daemon, an index, or a cache in phase 1.
- Languages beyond Go until a language is wanted.
- Renaming or editing source. Quarry reads.
- Compact-by-default. Views are options; extraction is complete.

## 10. Open decisions

- Type checker for phase 2 (gopls vs `go/packages` in-process) — decided when T8 unparks
  (`docs/roadmap.md`), not before.
- C# long parameter lists: whether to cap a method glyph, decided only after measuring a real
  C# repository.
