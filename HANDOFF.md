> Next agent: call the Skill tool with `mill:conversation` before reading the rest of this document.

# HANDOFF — the quarry rewrite, state as of 2026-09-03 (evening)

A fresh session on any machine acts on this file, `docs/rewrite-plan.md` and `docs/glyph.md`.
Nothing else is needed. The plan (§12) is the task table; the glyph document is the identifier
contract. Neither is summarised here — read them.

## 1. Where things stand

Quarry is being rewritten around one identifier, the glyph, and three queries: `toc`, `resolve`,
`expand`. Tree-sitter only, Go only. **T0 (the V1 deletion) is merged; wave 1 (T1, T2) is spawned
and mid-discussion** (§4). What survives on `main` after T0 is listed in plan §2's "kept" rows:
the Go extractor under `internal/quarryengine/`, the ladder yaml and tasks under
`bench/loomyard-eval/`, the two rewrite documents, and `docs/research/`.

`CLAUDE.md` is one line: Go repository, no Python. No tracked file may carry a machine path.

**Uncommitted on `main` right now:** this file, `docs/rewrite-plan.md` and `docs/glyph.md` — the
`members`→`expand` rename and its rationale (this session). Commit before switching machines.

## 2. The decisions the plan rests on

- **Go only.** Other languages are added one at a time, when wanted, with extractors written
  against the glyph contract. Nothing in the tree describes a language it does not support.
- **The listing verb is `toc`.** `map` was considered and rejected: a keyword in Go, a different
  operation everywhere else.
- **The type verb is `expand`** (renamed from `members`, 2026-09-03, in all three documents).
  "Member" stays the grammar's word for what follows `#`; as a verb it was one language's
  vocabulary and said nothing about the operation. `expand` unfolds one type into its head and
  everything its owner chain holds, across files. Type glyphs only — paths belong to `toc`, a leaf
  has nothing to unfold, and `toc` never takes a glyph (full rationale in plan §5).
- **The facade is the primary surface**, for Loomyard's own Go code. The CLI mirrors it with JSON
  and exit codes. **MCP is thin**: one `toc` tool, more only when a ladder cell measures more.
- **The plan validates before dispatch**: `glyph.Parse`, then `resolve`, then plan-internal checks,
  before any agent runs (plan §8.1).
- **Extraction is complete; views filter.** No default view drops anything.

## 3. What was measured, and still holds

The numbers the plan cites (its §1 points here). The runs are on branch `v1-final` under
`bench/loomyard-eval/ladder/results/<root>/conclusion.md`; nothing on `main` reproduces them, and
the uncommitted `raw/` data only ever existed on the machines that ran the matrices (one machine's
leftovers were deleted 2026-09-03 — the committed conclusions are the record).

| finding | run |
|---|---|
| A directory table of contents halves exploration on unfamiliar code: turns 8→4, cache_read 127k→83k, recall unchanged. The one measured win and the regression gate for the rewrite (T7). | `2026-09-02-toc`, reps 5 |
| Every LSP-shaped tool (definition, references, symbol) sat flat with or below the grep control. | `2026-08-30`, 45 runs |
| A lossy compact view cut bytes 4× and cost precision 0.96→0.82. Views must be complete by default. | `2026-09-02-compact2` |
| Symbol spans are syntactic: doc p90=12 lines, total span p90=52, max=971. A fixed grep window truncates silently. | 1741 symbols, Loomyard |
| Cold tree-sitter extraction: one package 19 ms, the whole repository 318 ms. No daemon needed. | Loomyard, 439 files |

Two harness rules carry into T2: never edit the code under test mid-matrix (binary hash per rep in
`provenance.json`), and cost numbers compare only within one results root. Everything else the old
harness enforced, `claude -p` now guarantees (plan §9a).

## 4. Next

**Wave 1 is in flight.** Both tasks spawned 2026-09-03 into `wts/glyph-package` and
`wts/ladder-harness`, both running `mill-start --orch` with the orchestrator session substituting
for discussion-review round 1 (`orch-review.md`, written; round 2+ reverts to the automated
reviewer):

- **`glyph-package` (T1):** round-1 verdict **APPROVE**, 3 NITs. The worker proceeds through
  mill-plan/mill-go on its own.
- **`ladder-harness` (T2):** round-1 verdict **REQUEST_CHANGES**, 2 BLOCKING: (1) its
  blinding-gate check treats any control transcript containing the quarry repo root as fatal,
  while its own no-tmp-paths decision puts every task worktree *under* that root — every control
  rep would trip it; (2) it read plan §9a's "1 000–1 500 lines" as including tests, contradicting
  §2's non-test parallel. The worker picks the review up and fixes. Its worktree has the
  gitignored `.scratch/ladder.env` with `LADDER_LOOMYARD_REPO=<path to a Loomyard checkout>` —
  recreate that file when resuming on another machine.

After wave 1: T3 engine core, T4 resolve + expand, T5 facade + CLI, T6 thin MCP, T7 the ladder
run, in that order (plan §12). For mechanical tasks (T5, T6) use `mill-quick`; the review rounds
were pure overhead on T0.

**The wiki holds only the two active wave-1 tasks.** Groomed 2026-09-03: every `[done]` entry
removed; `ladder-model-tier-comparison` dropped as obsolete (bound to V1's seven-tool MCP — if
wanted again it is a ladder cell against the new surface). **Wave 2+ (T3–T7) is not in the wiki
yet** — add those tasks from plan §12 as their wave approaches.

Millhouse notes: the wiki is a daemon-backed store rendering `Home.md` and the proposal files.
Edit tasks only through the `millpy-*` wrappers or `wiki._client`; a hook also blocks the literal
string `.wiki` in shell commands — resolve the path via `_paths.resolve_wiki_path`. `mill-merge`
strips `_mill/` before squashing; a manual squash must `git rm -r _mill` first.

## 5. Open decisions (plan §11)

- The phase-2 type checker: gopls or `go/packages` in-process. Decided when `impact` is built.
- Whether the new harness commits `results/**/raw/`. T2 decides with its first root.
- A C# parameter-list cap, only after a real C# repository is measured — and only when C# is wanted.

## Suggested skills

- `mill:conversation` and `mill:workflow` — load at startup (response rules, skill table).
- `mill:mill-status` — the per-task state table; run it first to see where wave 1 stands.
- `mill:orch-review` — if wave-1 round-1 reviews are still pending when you arrive (they were
  written this session; only needed again for newly spawned tasks).
- `mill:mill-resume` — to recreate an active task's worktree on a new machine.
- `mill:mill-merge` / `mill:mill-cleanup` — when a wave-1 task reaches `[ready-to-merge]`.
- `golang:golang-build` — build/test commands after any Go change.
