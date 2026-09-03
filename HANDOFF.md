# HANDOFF — the quarry rewrite, state as of 2026-09-03

A fresh session on any machine acts on this file, `docs/rewrite-plan.md` and `docs/glyph.md`.
Nothing else is needed.

## 1. Where things stand

Quarry is being rewritten around one identifier, the glyph (`internal/shedrecipe#Lookup`), and
three queries: `toc`, `resolve`, `members`. Tree-sitter only, Go only.

**T0 is done.** `main` holds:

- `internal/quarryengine` — the cgo guard pair, a short `doc.go`, `ErrLanguageUnsupported`.
- `internal/quarryengine/toc` — the Go strategy, the shared helpers, the extension table (`.go` only).
- `internal/quarryengine/treesitter` — the Go grammar. `go.mod` requires `go-tree-sitter` and
  `tree-sitter-go`, nothing else.
- `docs/rewrite-plan.md` — the plan. §12 is the task table, one mill task per row.
- `docs/glyph.md` — the identifier contract every task builds on.
- `bench/loomyard-eval/ladder/ladder*.yaml` and `bench/loomyard-eval/tasks/` (prompts and
  `*.fasit.json`) — the inputs the new harness (T2) and the first ladder run (T7) need.
- `docs/research/` — the research notes behind the plan.

`CLAUDE.md` is one line: Go repository, no Python. No tracked file may carry a machine path.

## 2. The decisions the plan rests on

- **Go only.** Other languages are added one at a time, when wanted, with extractors written against
  the glyph contract. Nothing in the tree describes a language it does not support: no dormant
  branches, no tests asserting absence.
- **The listing verb is `toc`.** It is a table of contents. `map` was considered and rejected: a
  keyword in Go, a different operation everywhere else.
- **The facade is the primary surface**, for Loomyard's own Go code. The CLI mirrors it with JSON
  and exit codes. **MCP is thin**: one `toc` tool, more only when a ladder cell measures more.
- **The plan validates before dispatch.** A glyph in a plan card is checked syntactically
  (`glyph.Parse`), structurally (`resolve`), and against the plan itself, before any agent runs.
- **Extraction is complete; views filter.** No default view drops anything.

## 3. What was measured, and still holds

The numbers the plan cites. The runs that produced them are on branch `v1-final` under
`bench/loomyard-eval/ladder/results/<root>/conclusion.md`; nothing on `main` reproduces them.

| finding | run |
|---|---|
| A directory table of contents halves exploration on unfamiliar code: turns 8→4, cache_read 127k→83k, recall unchanged. This is the one measured win and the regression gate for the rewrite (T7). | `2026-09-02-toc`, reps 5 |
| Every LSP-shaped tool (definition, references, symbol) sat flat with or below the grep control. Positions are what grep already finds. | `2026-08-30`, 45 runs |
| A lossy compact view cut bytes 4× and cost precision 0.96→0.82; the agent answered from the thinner listing. Views must be complete by default. | `2026-09-02-compact2` |
| Symbol spans are syntactic: docstrings above a declaration p50=3, p90=12, max=52 lines; total span p50=14, p90=52, max=971. A fixed grep window truncates silently. | 1741 symbols, Loomyard |
| Cold tree-sitter extraction: one package 19 ms, the whole repository 318 ms. No daemon is needed. | Loomyard, 439 files |

Two harness rules carry over into T2: do not edit the code under test while a matrix runs (the
binary hash per repetition stays in `provenance.json`), and cost numbers are only comparable within
one results root. Everything else the old harness enforced, `claude -p` now guarantees (plan §9a).

## 4. Next

The mill wiki backlog holds **`glyph-package` (T1)** and **`ladder-harness` (T2)**, wave 1, each
with a proposal, unclaimed. Spawn both with `mill-spawn`; they run in parallel. T2's worktree needs a
gitignored `.scratch/ladder.env` with `LADDER_LOOMYARD_REPO=<path to a Loomyard checkout>`.

After wave 1: T3 engine core, T4 resolve + members, T5 facade + CLI, T6 thin MCP, T7 the ladder run,
in that order (plan §12). For mechanical tasks (T5, T6) use `mill-quick` rather than the full
pipeline; the review rounds were pure overhead on T0.

Millhouse notes: the wiki is a daemon-backed store that renders `Home.md` and the proposal files.
Edit tasks through the `millpy-*` wrappers or `wiki._client`, never the files. `mill-merge` strips
`_mill/` before squashing; a manual squash must `git rm -r _mill` first.

## 5. Open decisions (plan §11)

- The phase-2 type checker: gopls or `go/packages` in-process. Decided when `impact` is built.
- Whether the new harness commits `results/**/raw/`. T2 decides with its first root.
- A C# parameter-list cap, only after a real C# repository is measured — and only when C# is wanted.
