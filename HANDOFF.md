# Handoff: task 05 (mergeresolve.Resolve) ladder — ready to run, blocked on environment

Written 2026-09-01. For whoever picks this up next, most likely a fresh agent working inside WSL2.
Delete this file once the run described below is complete and its results are committed — it is a
handoff note, not permanent documentation.

## What this is

A follow-up to the 45-run quarry-mcp capability-ladder benchmark that finished on 2026-08-30
(`bench/loomyard-eval/ladder/results/2026-08-30/summary.json`). That matrix's Ladder B (impact
analysis, task 04) turned out to be a wash: the no-tool control already scored perfect 1.00
recall/precision by grep alone, so every quarry-tool config there was measurably *slower* for zero
correctness gain (see `docs/when-to-use-quarry.md` for the distilled guidance, and
`git show archive/quarry-mcp-ladder-findings-review~1:_mill/analysis.md` — commit `f5d36a6` — for the
full prior review, including the specific bugs that were fixed afterward on main: unscoped
`workspace_symbol` saturating gopls's 100-hit cap, commit `1592f4e`; an `omitempty` wire bug dropping
empty result keys, commit `ee84d8d`).

The open question this session was working: does quarry actually separate from grep on a task with a
**genuine** symbol-disambiguation trap, since neither task 01/02 (exploration) nor task 04 tested one
that a careful grep-reading agent couldn't already solve by hand? A forked search agent surveyed
Loomyard for a real (not fabricated) case and found one: `mergeresolve.Resolver.Resolve` vs.
`modelspec.Registry.Resolve`, both spelled `.Resolve(` and — unlike task 04's decoy — sitting in the
**same package that defines the target method itself**, three lines below a doc comment that narrates
the real method's own resolution flow. It was verified directly (not just by the search agent) with an
actual compiler experiment: the signature was patched in a disposable worktree, `go build ./...` was
run across the whole module, and the resulting "not enough arguments" errors are the fasit's evidence.
Full reasoning is in the task file itself — read that before touching anything else.

## What's already built (committed in this commit)

- `bench/loomyard-eval/tasks/05-mergeresolve-resolve-impact.md` — the task text, setup, scope,
  schema, and scoring notes. Same shape as task 04 (impact-analysis schema).
- `bench/loomyard-eval/results/2026-09-01/05-mergeresolve-resolve-impact/c.json` — the fasit,
  compiler-verified (see the file's own `_meta.method_note`).
- `bench/loomyard-eval/ladder/ladder-task05.yaml` — a companion matrix (same pattern as the existing
  `ladder-followup.yaml`), **not** a change to the main `ladder.yaml`. Four configs — `c0-none`,
  `c1-impact`, `c2-assert-no-callers`, `c3-references` — reps: 3, matching the main matrix's own
  standard. Deliberately narrower than the full 8-rung Ladder B shape; see the file's own header
  comment for why each included/excluded tool was chosen.

**Read `ladder-task05.yaml` fully before running anything** — its paths (`session_dir_template`,
`source_repo`, the task's `worktree`) were written for this machine's Windows/git-bash checkout
(`/c/Code/...`) and **must be repointed** to wherever `loomyard` and `quarry` actually live inside
WSL2. Do not assume the WSL2 filesystem sees the same paths.

## The blocker that sent this to WSL2

`bench/loomyard-eval/ladder/internal/ladder/daemon.go:175` calls `syscall.Kill`, which does not exist
under `GOOS=windows` — `go build ./bench/loomyard-eval/ladder/...` fails outright on native Windows.
This is not a bug introduced by this session; it's why the ladder README already said "not expected to
run on Windows," just never actually hit until now. **Not fixed here, deliberately** — the operator
explicitly did not want an unrequested code change to the benchmark harness as a side effect of running
one task. If Windows support is ever wanted for real, the fix is small and has a template already in
this repo: mirror `internal/proc/proc_linux.go`/`proc_windows.go`'s existing split into a
`pidalive_linux.go`/`pidalive_windows.go` pair for `daemon.go`'s `pidAlive`. Not needed for this
handoff — WSL2 sidesteps it entirely since `syscall.Kill` is native there.

**Also unconfirmed, check again in WSL2 before relying on it:** on this Windows machine,
`go build -o /tmp/quarry-mcp ./cmd/quarry-mcp` failed with `reading
github.com/tree-sitter/go-tree-sitter/go.mod at revision v0.25.0: unknown revision v0.25.0`. This may
have been a transient network/module-cache issue specific to this machine, or a real problem — it was
never isolated. Verify `go build ./cmd/quarry-mcp` and `go build ./cmd/quarry` succeed cleanly in WSL2
before assuming the harness's own MCP server binary is fine; if the same error reproduces there, that's
a separate, real blocker to chase (check `GOFLAGS`, `GOPROXY`, and whether that exact tag exists
upstream) before spending on any dispatched run.

## What "done" looks like from here

1. In WSL2: confirm `go`, `git`, `tmux`, and the `claude` CLI are all on `PATH`, and that `loomyard` and
   `quarry` are checked out there (clone fresh if not — **never nest a checkout inside another
   checkout**, per this repo's own `CLAUDE.md`; use a sibling layout, e.g. `~/Code/quarry/wts/quarry`
   and `~/Code/loomyard/wts/loomyard`, matching the mill container's own `wts/<repo>/` convention).
2. `go build ./bench/loomyard-eval/ladder/...`, `go build ./cmd/quarry-mcp`, `go build ./cmd/quarry` —
   all three must succeed before anything is dispatched.
3. Fix `ladder-task05.yaml`'s three path fields (`session_dir_template`, `source_repo`,
   `05-mergeresolve-resolve-impact.worktree`) to the WSL2 paths from step 1. Also fix the literal
   `git -C /c/Code/loomyard/wts/loomyard worktree add ...` command in
   `bench/loomyard-eval/tasks/05-mergeresolve-resolve-impact.md`'s own Setup section to match.
4. `run_model`, `run_effort`, `max_turns`, and `scorer` are already pinned in `ladder-task05.yaml`
   (`RequirePins` will refuse to run otherwise) — no operator action needed there unless you want
   different values.
5. Follow the documented protocol in `bench/loomyard-eval/ladder/README.md` ("How to run") and the
   tracked `.claude/skills/ladder-run/SKILL.md`, pointed at this companion file:
   ```
   ladderbench prepare-session --ladder bench/loomyard-eval/ladder/ladder-task05.yaml \
     --config-id <config-id> --rep <n> \
     --results-root bench/loomyard-eval/ladder/results/2026-09-01-task05
   ```
   12 run sessions total (4 configs × 3 reps), then one shared scoring session, then
   `ladderbench summarize --results-root bench/loomyard-eval/ladder/results/2026-09-01-task05`.
   `tools/runmatrix` (already generalized to take an arbitrary `--ladder` file per commit `11697df`)
   should drive the 12 non-cold sessions automatically — there is no cold cell in this companion
   matrix, so `tools/runmatrix`'s own cold/scoring extension (commit `21fb2cf`) is not needed here,
   just its main loop.
6. Separately, and not blocking on the above: `bench/loomyard-eval/ladder/ladder-followup.yaml` is a
   **different**, already-staged, already-committed companion matrix from before this session that has
   also never been run — it re-tests `b1-symbol`/`b2-definition`/`b4-lsp-trio` against the two bug
   fixes (`1592f4e`, `ee84d8d`) that were never verified against a real re-run. Cheap (reps: 2, no new
   design work) and worth doing in the same WSL2 session if convenient, but it's an independent task
   from task 05 — don't conflate their results roots.
7. Once task 05's `summary.json` exists, update `docs/when-to-use-quarry.md` with whatever it shows —
   including a negative result. The current guidance's "What this doesn't tell you yet" section
   explicitly names this open question; closing it either way is the point of this whole exercise.

## Do not

- Don't point this task's `source_repo`/`worktree` back at the Windows checkout from inside WSL2 (via
  `/mnt/c/...`) as a shortcut to avoid cloning fresh — this repo's own `CLAUDE.md` rule against nested
  checkouts is about not nesting one checkout inside another, but running the actual benchmark this way
  would also cross the WSL2/Windows filesystem boundary for every file the dispatched agents touch,
  which is slow and has caused real correctness surprises in other projects (case sensitivity, file
  locking). Clone fresh inside WSL2's own filesystem.
- Don't fold task 05 into the main `ladder.yaml` matrix. It's deliberately a separate companion file,
  same as `ladder-followup.yaml` — keeps its results root independent and avoids disturbing the
  already-scored 2026-08-30 matrix.
