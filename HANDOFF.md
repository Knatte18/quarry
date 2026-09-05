> Next agent: call the Skill tool with `mill:conversation` before reading the rest of this document.

# HANDOFF — quarry, state as of 2026-09-05

A fresh session on any machine acts on this file, `docs/rewrite-plan.md` (the design and its
decisions, distilled), `docs/glyph.md` (the identifier contract) and `docs/roadmap.md` (what is
ahead — the forward-looking record lives there now, not here). None of them is summarised here —
read them.

## 1. Where things stand

**The build is done and the measurement programme is done.** On `main`: T0–T7 (the whole
rewrite — glyph, engine, facade, CLI `toc`/`resolve`/`expand`, thin MCP, the ladder harness),
M1 (the breadth matrix) and M3 (the decisive ladder-d rerun at n=15, squash `03566aa`, run as
`mill-quick` — the flow's first real exercise, and it worked). Each task is one squash commit
titled by its task; full branch history including `_mill/` artifacts is under `archive/<slug>`
(22 tags on origin). The `pipeline.done_gate` (`go test ./... && golangci-lint run`) was green
after the last merge.

The surface: `internal/engine` (+ `treesitter` seam, `cgoguard`), the `quarry/` facade
(primary), `cmd/quarry` (CLI mirror), `cmd/quarry-mcp` (thin, one `toc` tool),
`internal/repopath`. Goldens in `docs/research/output-formats/after/`, pinned to Loomyard
`72c23d9` via `LADDER_LOOMYARD_REPO` in `.scratch/ladder.env` (gitignored, per machine).

Worktrees: the hub, `wts/v1-final` (deliberate read-only reference checkout of the V1 branch —
`mill-cleanup` reports it as an orphan; ignore that line, never remove it), and
`wts/glyph-self-form` (C1, spawned and ready, **not yet started** — see §3).

## 2. The measurement verdict, and the pivot it forced

The measurement programme is complete and its answer is negative, in three committed
conclusions (each root stands alone; read them, they are the record):

| root | result |
|---|---|
| `2026-09-04-toc` (T7) | the August ladder-a win (turns 8→4) did not reproduce at clean n=5 — flat-to-reversed |
| `2026-09-04-breadth` (M1) | three shapes (negative control, three-package, whole-repo) — no separation anywhere at n=5; ladder d's medians alone pointed the hypothesis's way |
| `2026-09-05-ladder-d` (M3) | ladder d at predeclared n=15 with a predeclared one-sided Mann–Whitney U: turns U=105 (p≈0.375), cost_usd U=92 (p≈0.198), critical U≤72 — **no win**, medians now flat-to-slightly-adverse |

So: **directory-level `toc` does not pay as a mid-session agent tool, at any measured shape or
n, in this model generation.** `docs/roadmap.md` records T8's unpark condition (a) as closed on
this basis. The mechanism itself is real (ladder c: read_bytes 14 800 vs 44 629, recall up) —
the agent reads less and hits better with a map; the map's retrieval cost just eats the win.
The control arm also moved: the same exploration cost 8 turns in August and 11–12 now.

**The pivot:** quarry's remaining value is the *function* track, which the ladders never
measured and never claimed to. The glyph as Loomyard's plan alphabet; the facade's batched
`Resolve(targets []string)` as the plan validator (typos/ambiguity rejected before an agent is
spawned) and as the dispatch-time "kick-start" pack (glyph → file + span + signature, injected
into the implementer's prompt: zero agent turns spent looking). Backup mode is designed in:
plain file paths in `Uses`/`Creates` validate and pack the same way — never worse than
Millhouse's current file-list plans. T8 unparks via condition (b) — the validator's Delete gate
needs `assert-no-callers` — the day the Loomyard adoption wants it. That adoption is Loomyard-
repository work (roadmap, External), unblocked since T5b.

M3's predeclared-rule pattern is worth keeping for any future measurement: decision rule, α,
n and cell list locked in the task body before rep 1; n never grows after looking; the
conclusion shows its arithmetic. It is what makes a negative answer trustworthy.

## 3. The one active task: C1 (`glyph-self-form`)

Spawned at `wts/glyph-self-form` (caps 3/3, `ladder.env` written), **worker not yet started**
— the operator starts it with `/mill-start --orch` in that worktree, then this session (or a
fresh one) arms the wait with `/mill:orch-review glyph-self-form`. The task body carries four
contract decisions settled with the operator 2026-09-05, before Loomyard consumes the envelope
(breaking changes are free now, zero external consumers):

1. **Trailing `#` = the thing itself**: `internal/reedengine/render#` is the package,
   `.../focus.go#` is the file; strip the `#` and you have the plain path.
2. **`resolve` takes glyphs only** — a bare path is rejected with an actionable error
   (`toc` takes paths; `resolve` takes names; `expand` takes type glyphs).
3. **Separator-bearing path segments rejected loudly** — closes the silent-reclassification
   gap `internal/cli/doc.go` documents (leading-`#` was considered and rejected: it is a
   comment/heading in shell, YAML and Markdown).
4. **Rename resolve's path-answer field `dir` → `listing`** (the outer block; the inner `dir`
   path field keeps its name).

Deliberately NOT in C1: `toc` gains no `id` on file/dir entries — self-glyph composition is
trivial concatenation done by tooling, and per-file full-path ids would reinstate the repeated-
prefix clutter V1 was purged of. The finalize PR gets the orchestrator review before the
operator closes it.

## 4. Small open items

- Ladder harness `mkdir -p` on the results root (in roadmap, small items — a fresh `-results`
  path fails on rep 0 today).
- Move the `after/` goldens to `internal/cli/testdata/` (roadmap, small items; mill-quick).
- Wiki grooming: eleven `[done]` entries can go at the next groom.
- The flag parser silently accepts `--flag=value` on valueless flags (PR #19 NIT, left as-is).
- The doubled `engine: ... engine: ...` error wording (T5b's D10b).
- In Millhouse itself: review-prompt bulking repeats absolute paths (fix in
  `_review_common.py`); the operator knows. Also filed by the M1 worker: #994 (mill-plan
  blocked-resume `--max-rounds` threading), #995 (TaskOutput liveness probe dumps full JSONL).

## 5. Millhouse notes (patterns that keep working)

Wiki only via `wiki._client`/`millpy-*` (a hook blocks the literal `.wiki` in shell). A task
upserted **without** `status` is spawnable; with `status: "open"` it is not. Worker setup per
task: spawn, write `.scratch/ladder.env`, append round caps to the worktree's
`.millhouse/config.local.yaml` (standard 5, 3 for well-scoped tasks; the config warning about
nested-hub pointer stubs is noise — the values load). The orch-review pattern: this session
owns each `Monitor` wait; a fork writes `orch-review.md` only after `discussion.md` exists.
The PR pattern: orchestrator fork (or a direct read for small diffs) reviews the finalize PR;
the operator **closes without merging — the closed state is the approval**; the worker runs
mill-merge itself. Watch merges to *completion*: squash on `origin/main` AND `archive/<slug>`
tag AND wiki `[done]`. Commit the hub's HANDOFF before a worker's mill-merge runs, or the
squash conflicts in the hub (it did once; the worker's rollback + `mill-merge-in` + retry
recovered it cleanly). Shell discipline: never `cd` in a compound command — it poisons the
session cwd (use `git -C`, subshells for one-off cd).

## Suggested skills

- `mill:conversation` and `mill:workflow` — load at startup.
- `mill:mill-status` — the per-task state table; run it first.
- `mill:orch-review` — arm the `glyph-self-form` discussion wait once the operator starts the
  worker.
- `mill:mill-quick` — proven by M3; right size for the goldens move and the mkdir fix.
- `golang:golang-build` — build/test commands after any Go change.
