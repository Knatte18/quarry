> Next agent: call the Skill tool with `mill:conversation` before reading the rest of this document.

# HANDOFF — quarry at rest, the action is in Loomyard (2026-09-06, late)

A fresh session acts on this file plus `docs/roadmap.md` (main `5ff42d7`), `docs/glyph.md`,
Loomyard issues #226–#228, and the results root
`bench/loomyard-eval/ladder/results/2026-09-06-kickstart/` (read `conclusion.md` there —
the numbers and their caveats are not repeated here).

## 1. Where things stand — quarry

- `main` tip `5ff42d7`. **Tag `v0.1.0` cut and pushed at `b82e383`** — quarry's first
  semver tag, pinning the glyph contract API: `Parse`, `Self`, `String`, `IsSelf`, and the
  new `UnitPath` (glyph→disk-path, merged today via the `glyph-unitpath` mill-quick task).
- **The board is empty.** All six tasks done, merged, worktrees torn down, and wiki-groomed
  (six `[done]` entries dropped; history in `archive/<slug>` tags). Worktrees remaining:
  hub + `wts/v1-final` (read-only V1 reference; never remove).
- **M4 closed positive.** e1-pack separated from the plain-names control on both
  predeclared primary metrics (turns U=0, cost U=19, rule applied verbatim). Push-mode
  pre-resolution pays; the mid-session-tool direction remains closed negative (ladder
  a–d). Known confound recorded in the conclusion: a plain file list buys most of the turn
  collapse, spans+signatures buy the cost — decomposition deferred to M4b.
- **Roadmap** (restructured today, three commits `607bb0f`/`d2e492d`/`5ff42d7`):
  1 = Loomyard adoption (the kick-start pack is **deliberately out** of this round per the
  task's own Q1 decision — the matrix only removed the *measurement* gate); 2 = **M4b
  parked as an idea, not taken now** — full design sketch is in the roadmap text
  (re-implement a real merged Loomyard commit; test-overlay scoring; pack via
  diff-to-symbols; e0/e1/e2 arms; hard constraint: dedicated Loomyard clone, never the
  operator's live checkout); 3 = T8 parked; 4 = more languages.

## 2. Where things stand — Loomyard (other repo, operator-driven)

- The `quarry-glyph-plan-alphabet` task (#226 adoption) hit its predicted BLOCKING —
  no semver tag on quarry — and halted cleanly. Resolved today by plan B: `glyph-unitpath`
  merged first, then the one `v0.1.0` tag covering both. The operator resumed the worker
  (`/mill-go` re-fires the quarry-dependency batch); last observed phase in its worktree
  was `approved-quarry-cli`. Because the pin includes `UnitPath`, Q21's go.mod-bump
  precondition for the late disk-shaped-check cards is already satisfied.
- **Issue #227** (enhancement): implementer kick-start pack — pre-resolve a plan card's
  `Uses:` glyphs into the prompt. **Issue #228** (enhancement): reviewer kick-start for the
  plan- and code-review gates; discussion-reviewer explicitly excluded. A follow-up comment
  on #228 corrects the architecture: the pipeline's reviewer is the **Burler** (one agent
  that explores, reviews via forks, and fixes — no diff anywhere), so the pack injects once
  at the Burler's start and every fork inherits it as cache-read. Both issues are
  sequenced **after** the adoption lands; both carry the M4 evidence and its caveats.
- Per the operator's own rule (recorded in #227/#228 and the roadmap): building #227/#228
  should ideally wait for M4b's edit-task measurement — the M4 result is an explore-task
  result and the transfer to edit work is unmeasured.

## 3. The operator's next chapter (context, not tasks)

After ~4 months building Loomyard, the operator intends to **deploy it and let it build
something real for the first time** — thorough testing of the adoption first. The
self-hosting plan discussed and settled in principle:

- Loomyard develops Loomyard, but **the running Loomyard is never the Loomyard being
  edited**: a pinned, deployed release operates on a checkout, never on its own
  installation.
- Sandbox: a **separate disk clone** (working name "Lyx-klonet", e.g.
  `/home/knatte/Code/lyx/`) of the *same* GitHub repo — not a separate fork/repo —
  working only on its own branches (`lyx/*`). Success flows back as PRs to main; failure
  is deleted or re-cloned. Main stays Millhouse-governed in the existing
  `/home/knatte/Code/loomyard/` structure, which is in use and must never be touched.
- **Millhouse is the bootstrap fallback** (the assembler at the bottom of the ladder):
  when the deployed Loomyard itself breaks, a normal mill task on main fixes it.
- Advice already given for the first real build target: small but real, with a fasit the
  operator can judge — first runs find pipeline bugs, and those must be cheap to tell
  apart from task failures.

## 4. Operator queue

1. Watch the Loomyard adoption to completion; thorough testing of the glyph chain
   end-to-end (LLM plan → batched Resolve → glyph-maker canonicalization → DAG on handles
   → binding at done via diff-to-symbols; the hard-finding classifier; drift tiers).
2. Deploy Loomyard (tagged release, own install area) and run the first real build.
3. #227/#228 when their time comes — ideally after M4b, per §2.
4. M4b: parked; write the task from the roadmap sketch when wanted.
5. Small: OSL-1033 ladder-a rerun if that host returns; Millhouse #994/#995.

## 5. Standing rules (unchanged, load-bearing)

Wiki only via `wiki._client`/millpy wrappers (never files, never literal `.wiki` in Bash);
commits only on the operator's explicit word (standard trailers); never `cd` in compound
Bash (`git -C`); never `sed`; no Python in quarry; mill-quick tasks run by their own agent
in the spawned worktree; LYX never converts glyph↔path — exclusively quarry's glyph module
(`Self`, `UnitPath`, `Parse`); round caps via `roles.*.holistic.rounds`; `.scratch/` never
committed; HANDOFF.md regenerated only on `/mill:handoff` invocation; Fable pairs with
High effort, never Max; operator calibration: Opus 5 Medium ≈ Opus 4.8 High.

## Suggested skills

- `mill:conversation` and `mill:workflow` — load at startup.
- `mill:mill-status` — board view (quarry hub; the Loomyard task needs a manual look in
  its own repo).
- `mill:orch-review` — if another discussion.md review is wanted (works cross-repo with
  absolute paths + a fork).
- `golang:golang-build` — build/test commands after any Go change.
