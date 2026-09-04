> Next agent: call the Skill tool with `mill:conversation` before reading the rest of this document.

# HANDOFF — the quarry rewrite, state as of 2026-09-04 (evening)

A fresh session on any machine acts on this file, `docs/rewrite-plan.md` and `docs/glyph.md`.
Nothing else is needed. The plan (§12) is the task table; the glyph document is the identifier
contract. Neither is summarised here — read them.

## 1. Where things stand

`expand`. Tree-sitter only, Go only. **Waves 0–5 are merged on `main`:** T0 (the V1 deletion),
T1 (`glyph/`), T2 (the ladder harness), T3 (the engine), T4 (`resolve` + `expand` in the engine),
T5a (facade + CLI, `toc`), T6 (`cmd/quarry-mcp`, one `toc` tool — merged `e8a4b2d`), T5b (facade +
CLI, `resolve` + `expand` — merged last, `8326587`, and reconciled the collision with T6's
`internal/repopath` extraction), T7 (the regression-gate rerun of the toc-dir finding against the
merged rewrite; see §3). Each landed as one squash commit titled by its task; each task branch's
full history, `_mill/` artifacts included, is under its `archive/<slug>` tag.

The engine is `internal/engine` (grammar seam `internal/engine/treesitter`, `internal/cgoguard`
for `CGO_ENABLED=0`). The public surface is the `quarry/` facade, mirrored by `cmd/quarry`
(verbs `toc`, `resolve`, `expand`) and by `cmd/quarry-mcp` (MCP stdio server, name `quarry`,
one tool `toc`, byte-identical to the facade's JSON). Root/target resolution is shared in
`internal/repopath`. `docs/research/output-formats/after/` holds the golden evidence for all
three verbs; the golden tests pin the Loomyard checkout to `72c23d9` (hard-fail on wrong pin,
skip when `LADDER_LOOMYARD_REPO` is unset — set it in `.scratch/ladder.env`, gitignored, per
machine).

The full `pipeline.done_gate` (`go test ./... && golangci-lint run`) is green on `main`, verified
after the last merge. Check `git status --porcelain` on `main` before switching machines — this
line is retired rather than restated, since a fixed "uncommitted right now" claim in a handoff
document goes stale the moment anything else lands.

## 2. The decisions the plan rests on

- **Go only.** Other languages are added one at a time, when wanted, with extractors written
  against the glyph contract. Nothing in the tree describes a language it does not support.
- **The listing verb is `toc`; the type verb is `expand`.** `map` rejected (Go keyword); "member"
  stays the grammar's word for what follows `#`. `toc` never takes a glyph (plan §5).
- **One target per CLI call, for every verb** (`toc`: T5a, plan §5 amended `55161a6`;
  `resolve`/`expand`: T5b's D1, plan §5 amended `4df7621`): one invocation, one answer, one exit
  code. The facade keeps the engine's multi-target `Resolve(targets []string)` — the batching
  the validator (§8.1) relies on lives there. The CLI is the mirror, not the primary.
- **Negative answers render the payload, not the error envelope** (T5b's D2): `not_found`,
  `ambiguous` and pre-resolution errors keep `unit`, `candidates` and `reason` in the §4 payload
  with exit 1; the error envelope is only for usage/internal errors with no payload at all.
- **The engine package is `engine`, not `quarryengine`** (T3's D1, challenged and upheld).
- **Pipelined tasking works** (T4 ran its discussion against T3's artifacts via `.portals/`).
  Requirements: provenance marks, hold before implementation, `mill-merge-in` then plan against
  real code.
- **Capped review rounds + an orchestrator review of the finalize PR is the speed pattern**
  (proven on T5a, T5b, T6). A gitignored `.millhouse/config.local.yaml` in the *worker's*
  worktree caps `roles.discussion-review.holistic.rounds` and `roles.plan-review.holistic.rounds`
  — standard cap 5, 3 for genuinely small tasks (T5a and T6 ran at 3, T5b at 5, T7 has 3 —
  execution task, small diff). The
  compensating independent gate is the orchestrator PR review before the operator closes the PR.
  All three wave-4 PR reviews found zero blocking defects.
- **`mill-quick` is for genuinely small tasks only.** (T6 was slated for it but ran the full
  `--orch` pipeline in the end; the flow has still not been exercised.)
- **The facade is the primary surface**; CLI mirrors it; **MCP is thin**: one `toc` tool, more
  only when a ladder cell measures more. MCP spells whole-tree depth `-1` where the CLI says
  `--depth all` (JSON schema, documented).
- **Extraction is complete; views filter.** No default view drops anything. (The Codebase-Memory
  paper, arXiv 2603.27277, was read and deliberately not adopted: its headline result trades
  quality 0.92→0.83 for 10× fewer tokens — the exact trade quarry rejects.)

## 3. What was measured, and still holds

The numbers the plan cites (its §1 points here). The runs are on branch `v1-final` under
`bench/loomyard-eval/ladder/results/<root>/conclusion.md`; T7 reran the toc-dir finding against
the merged rewrite (`main`'s `cmd/quarry-mcp`) and it did not reproduce cleanly — see the new row
below and `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md` for the full account.
The committed conclusions are the record; the raw run data was never committed anywhere.

| finding | run |
|---|---|
| A directory table of contents halves exploration on unfamiliar code: turns 8→4, cache_read 127k→83k, recall unchanged. The one measured win and the regression gate for the rewrite (T7). | `2026-09-02-toc`, reps 5 |
| T7's rerun of the same comparison, against the merged rewrite: no cost metric separated from the control at n=5 (turns and cache_read both `separated: false`), and most cost medians ran equal-or-higher for the toc-dir cell rather than lower. Recall/precision stayed consistent with the prior root. The regression gate did not reproduce the prior win on this host/harness/CLI-version pairing. | `2026-09-04-toc`, reps 5 |
| Every LSP-shaped tool (definition, references, symbol) sat flat with or below the grep control. | `2026-08-30`, 45 runs |
| A lossy compact view cut bytes 4× and cost precision 0.96→0.82. Views must be complete by default. | `2026-09-02-compact2` |
| Symbol spans are syntactic: doc p90=12 lines, total span p90=52, max=971. A fixed grep window truncates silently. | 1741 symbols, Loomyard |
| Cold tree-sitter extraction: one package 19 ms, the whole repository 318 ms. No daemon needed. | Loomyard, 439 files |

Two harness rules carry into every matrix: never edit the code under test mid-matrix (binary
hash per rep in `provenance.json`), and cost numbers compare only within one results root.

## 4. Next

**The critical path is finished.** T0 → T1 → T3 → T5a → T6 → T7 has run end to end; T7 measured
the regression gate against the merged rewrite and wrote its conclusion (see §3). What remains is
wave 6's type checker (T8: gopls vs `go/packages`, `impact`, `assert-no-callers`, `verified` per
entry, the DAG tightening of plan §8.2 — plan §12), plus the cleanup and grooming items below.

**The §9a live probe is DONE** (2026-09-04, this machine, from the Loomyard checkout): connect +
`toc` call returned the §4 envelope, and the allowlist denial landed in `permission_denials` —
both run harness-faithfully. Tell the T7 worker this when it asks (its task body instructs it to
stop and ask if the probe has not been reported done). Caveat discovered on the way: the
operator's global `defaultMode: "auto"` auto-approves read-only MCP calls, so any manual probe
MUST pass `--setting-sources ""` (as the harness itself does via `run.go`) or the denial half
never fires.

**After T7:** T8 (the type checker, `impact`) depends on T7 and is the whole next wave.
Whether `results/**/raw/` is committed is decided by T7 itself (plan §11).

**Cleanup/grooming:** all worker worktrees are torn down — only the hub remains. Wiki grooming:
seven `[done]` entries can go at the next groom.

**Small open items:**
- The flag parser silently accepts `--flag=value` on valueless flags (PR #19, NIT, left as-is).
- The doubled `engine: resolve target ...: engine: ...` error wording — engine follow-up on
  main, recorded by T5b's discussion (D10b).
- `internal/cli/doc.go` documents a known contract gap (a path target whose repo-relative form
  gains `#` from a directory name is re-classified by the engine) — deferred to the next
  engine-signature task.
- Discussed, not yet a task: move the `after/` goldens to `internal/cli/testdata/` (they are
  living test fixtures; `docs/research/output-formats/` stays a frozen research record — the
  before/after naming ages badly). Operator liked the idea; a good `mill-quick` candidate.
- In Millhouse itself (not this repo): review-prompt bulking repeats each file's absolute path
  three times; fix is `base_dir` + relative paths in `_review_common.py` — the operator knows.

**Millhouse notes.** The wiki is daemon-backed: edit tasks only through `wiki._client` /
`millpy-*` wrappers; a hook blocks the literal string `.wiki` in shell commands — resolve via
`_paths.resolve_wiki_path(_paths.resolve_git_root())`. **A task upserted with `status: "open"`
is not spawnable** — upsert without a status, or clear with `_client.set_phase(wiki, slug,
None)`. The orch-review pattern: this session owns the `Monitor` wait for `discussion.md`; a
fork writes `orch-review.md` only after the file exists. The PR-review pattern: a fork reviews
the finalize PR's diff against plan §4 and reports before the operator closes it;
`require_pr_to_base: true` means the operator **closes the PR without merging — the closed
state is the approval** — and the worker runs mill-merge itself. When watching a merge, watch
for *completion*, not the squash subject alone: squash on `origin/main` AND the `archive/<slug>`
tag AND the wiki flipped `[done]` (teardown lags the squash), and anchor any subject match on
the full task title. When two finalize PRs race on the same files, close one, let its worker
merge fully, then close the other — its `mill-merge-in` does the reconciliation (T5b did this
cleanly over T6's `repopath` move). Shell discipline: a `cd` inside any compound command
poisons the session cwd for every later command — one such slip mid-session sent git commands
at the wrong repository until caught.

## 5. Open decisions (plan §11)

- The phase-2 type checker: gopls or `go/packages` in-process. Decided when `impact` is built
  (T8). If `impact` is ever pursued, the gate is a *task-shaped* ladder cell — an edit task where
  the agent must find break sites — since reference-shaped tools measured flat in §3.
- A C# parameter-list cap, only after a real C# repository is measured — and only when C# is wanted.

The raw-tree decision is settled — see
`bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md` and `docs/rewrite-plan.md`
§11's updated bullet.

## Suggested skills

- `mill:conversation` and `mill:workflow` — load at startup (response rules, skill table).
- `mill:mill-status` — the per-task state table; run it first.
- `mill:orch-review` — re-arm the `ladder-toc-rerun` discussion wait if this session is gone.
- `mill:mill-quick` — for the `after/`-goldens move, if the operator turns it into a task.
- `golang:golang-build` — build/test commands after any Go change.
