> Next agent: call the Skill tool with `mill:conversation` before reading the rest of this document.

# HANDOFF — the quarry rewrite, state as of 2026-09-04 (morning)

A fresh session on any machine acts on this file, `docs/rewrite-plan.md` and `docs/glyph.md`.
Nothing else is needed. The plan (§12) is the task table; the glyph document is the identifier
contract. Neither is summarised here — read them.

## 1. Where things stand

Quarry is being rewritten around one identifier, the glyph, and three queries: `toc`, `resolve`,
`expand`. Tree-sitter only, Go only. **Waves 0–2 are merged on `main`:** T0 (the V1 deletion),
T1 (`glyph/`), T2 (the ladder harness under `bench/loomyard-eval/ladder/`), T3 (the engine).
Each landed as one squash commit titled by its task; each task branch's full history, `_mill/`
artifacts included, is preserved under its `archive/<slug>` tag.

The engine now lives at **`internal/engine`** (renamed from V1's `internal/quarryengine` by T3's
D1 — see §2), one package, files per concern, with `internal/engine/treesitter` as the grammar
seam and `internal/cgoguard` guarding `CGO_ENABLED=0` builds.

`CLAUDE.md` is one line: Go repository, no Python. No tracked file may carry a machine path.

**Uncommitted on `main` right now:** this file only. Commit before switching machines.

## 2. The decisions the plan rests on

- **Go only.** Other languages are added one at a time, when wanted, with extractors written
  against the glyph contract. Nothing in the tree describes a language it does not support.
- **The listing verb is `toc`.** `map` was considered and rejected: a keyword in Go, a different
  operation everywhere else.
- **The type verb is `expand`** (renamed from `members`, 2026-09-03). "Member" stays the grammar's
  word for what follows `#`. Type glyphs only; `toc` never takes a glyph (rationale in plan §5).
- **T5 is split; waves 3 and 4 are parallel** (2026-09-03). The MCP exposes only `toc`, so the
  critical path to the measurement is T0 → T1 → T3 → T5a → T6 → T7; T4 and T5b sit off it
  (plan §12).
- **The engine package is `engine`, not `quarryengine`** (T3's D1, challenged and upheld
  2026-09-04). `quarryengine` stutters against the module path; `internal/` is compiler-enforced
  private, so the short name is visible only inside quarry, where there is exactly one engine.
  The "unambiguous names" rule targets no-cohesion grab-bags (`util`), not module-scoped names.
  If a second engine ever arrives, the *new* thing gets the qualifier — a rename then is one
  in-repo refactor, since nothing outside the module can ever import it.
- **Pipelined tasking works and is the pattern now:** a dependent task's *discussion* runs against
  its predecessor's decided artifacts (via the `.portals/<slug>/` junction) while the predecessor
  implements; the code dependency binds only implementation. The task body must (a) mark which
  facts come from artifacts rather than code on `main`, (b) hold before implementation until the
  predecessor merges, then `mill-merge-in` and plan against real code. T4 ran this way against T3
  and its round-1 review verified every provenance mark. Best case: the predecessor merges before
  `mill-plan` starts, and the plan is simply written against reality — no revision round.
- **The facade is the primary surface**; CLI mirrors it; **MCP is thin**: one `toc` tool, more
  only when a ladder cell measures more.
- **The plan validates before dispatch**: `glyph.Parse`, then `resolve`, then plan-internal
  checks, before any agent runs (plan §8.1).
- **Extraction is complete; views filter.** No default view drops anything.

## 3. What was measured, and still holds

The numbers the plan cites (its §1 points here). The runs are on branch `v1-final` under
`bench/loomyard-eval/ladder/results/<root>/conclusion.md`; nothing on `main` reproduces them
(the T7 rerun will). The committed conclusions are the record; the raw run data was never
committed anywhere.

| finding | run |
|---|---|
| A directory table of contents halves exploration on unfamiliar code: turns 8→4, cache_read 127k→83k, recall unchanged. The one measured win and the regression gate for the rewrite (T7). | `2026-09-02-toc`, reps 5 |
| Every LSP-shaped tool (definition, references, symbol) sat flat with or below the grep control. | `2026-08-30`, 45 runs |
| A lossy compact view cut bytes 4× and cost precision 0.96→0.82. Views must be complete by default. | `2026-09-02-compact2` |
| Symbol spans are syntactic: doc p90=12 lines, total span p90=52, max=971. A fixed grep window truncates silently. | 1741 symbols, Loomyard |
| Cold tree-sitter extraction: one package 19 ms, the whole repository 318 ms. No daemon needed. | Loomyard, 439 files |

Two harness rules carry into every matrix: never edit the code under test mid-matrix (binary hash
per rep in `provenance.json`), and cost numbers compare only within one results root.

## 4. Next

**Wave 3 is half in flight.** `resolve-expand` (T4) is `[active]`, phase `discussed`: its
discussion ran pipelined against T3's artifacts, round-1 orch-review returned REQUEST_CHANGES
(two undefined dispositions: `expand <unit>#init` with several `init`s reaches no defined answer;
engine I/O errors mid-`Resolve` have no decided home), the worker absorbed the review, and
`mill-start` has finished. **Its next steps, in the worker session:** `/mill-merge-in` (T3 is now
on `main` — this is the hold-point release), verify the discussion's provenance marks against the
real `internal/engine/` code (especially the head-span reading its discussion flagged for
post-merge verification), then `/mill-plan` directly against the code. Its implementation needs
a Loomyard checkout path from the environment (`.scratch/ladder.env` convention,
`LADDER_LOOMYARD_REPO=<path>` — gitignored, recreate per machine).

**T5a (facade + CLI, toc) is spawned** (`facade-cli-toc`, phase `discussing`) — run
`/mill-start --orch` in its worktree and orch-review round 1 here; it is too big for `mill-quick`
(see Suggested skills). T6 follows T5a and is the real `mill-quick` candidate; T7 needs T2
(merged) and T6.

**Wiki state:** `resolve-expand` active; `engine-core`, `glyph-package`, `ladder-harness` are
`[done]` (grooming policy: `[done]` entries get removed — do so at the next groom). Wave 3+
tasks (T5a, T5b, T6, T7, T8) are not in the wiki yet — add each from plan §12 as its wave
approaches. The `engine-core` worktree still exists and awaits `mill-cleanup --apply`.

**Millhouse notes.** The wiki is daemon-backed: edit tasks only through `wiki._client` /
`millpy-*` wrappers; a hook blocks the literal string `.wiki` in shell commands — resolve the
path via `_paths.resolve_wiki_path(_paths.resolve_git_root())`. The orch-review pattern: this
session owns the `Monitor` wait for `discussion.md`, a fork writes `orch-review.md` only after
the file exists. The repo has `require_pr_to_base: true`: mill-finalize opens a PR, the operator
reviews and **closes it without merging — that closed state is the approval signal**, and
mill-merge then squashes locally. The worker may run mill-merge itself; before merging as
orchestrator, check the parent's `.scratch/merge.lock` and whether the squash already landed on
`main` — do not double-merge.

## 5. Open decisions (plan §11)

- The phase-2 type checker: gopls or `go/packages` in-process. Decided when `impact` is built.
- Whether the harness commits `results/**/raw/`. Decided with T7's first results root.
- A C# parameter-list cap, only after a real C# repository is measured — and only when C# is wanted.

## Suggested skills

- `mill:conversation` and `mill:workflow` — load at startup (response rules, skill table).
- `mill:mill-status` — the per-task state table; run it first.
- `mill:orch-review` — for each newly spawned task's round-1 discussion review (T5a next).
- `mill:mill-quick` — for T6 only (thin wrapper, §9a's harness probe as an independent gate). T5a
  is too big for it (new facade + envelope + CLI, and its `after/` done-when is generated by its own
  code — review is the only independent §4 check): full pipeline with orch-review. T5b: decide when
  T5a's surfaces exist.
- `mill:mill-cleanup` — the `engine-core` worktree teardown is pending.
- `mill:mill-resume` — to recreate `resolve-expand`'s worktree on a new machine.
- `golang:golang-build` — build/test commands after any Go change.
