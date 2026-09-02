# HANDOFF — state as of 2026-09-02 evening, and what to do next

Self-contained: a fresh session on any machine should be able to act on this file alone. The research
report it descends from is `docs/research/quarry-improvement-research.md`; the benchmark suite is
`bench/loomyard-eval/ladder/` (its README is the reference for design, metrics, scoring, and the
enforcement gates). Every results root named below has a `conclusion.md` that is the record of that
run; read those before re-deriving anything from `summary.json`.

## 0. First thing: commit the working tree

Everything from 2026-09-02 is uncommitted (46 files). It is one coherent day of work and all tests
pass (`go test ./...` at the repo root). Commit it before running anything, because the matrix
builds the server under test from the working tree and a dirty tree is the single biggest source of
invalid runs (see §2, rule 1). Suggested split, or one commit if you prefer:

1. Engine + CLI + MCP: compact toc form — `internal/quarryengine/toc/compact.go` (+test),
   `quarry/facade.go` (`CompactTOCFile/Dir`), `internal/cli/toc.go` (`--compact`),
   `internal/mcpserver/{mcpserver,tools_toc}.go` (`compact` input, `--toc-format`),
   `cmd/quarry-mcp/main.go`.
2. Harness: `run.sh` + `run-*.sh` one-command entry points, `tools/runmatrix --all` with
   `provenance.json` and per-rep `server_hashes`, worktree `prune` before `add`, `GateRunPrompt`,
   annex support (`internal/ladder/annex.go`, `annex:` config field, `annexes:` task recipes),
   `toc_format:` config field, `SKILL.md` outcome-marker rule, README updates, `.gitignore`
   `/ladderbench`.
3. Ladders and results: `ladder-toc.yaml`, `ladder-compact.yaml`, `ladder-annex.yaml`,
   `results/2026-09-01-followup/conclusion.md`, `results/2026-09-02-followup/`,
   `results/2026-09-02-toc/`.

`results/**/raw/` is gitignored; the transcripts stay on the machine that ran them. The 2026-09-01
followup conclusion argues for un-ignoring it (1.8 MB per 12 runs); not done yet.

## 1. What has been established (read the conclusions for detail)

| results root | what | verdict |
|---|---|---|
| `2026-08-30` | 45-run main matrix, tasks 01 and 04 | only toc_dir separated from control; every LSP rung flat or worse |
| `2026-09-01-task05` | task 05, c0–c3 | no separation; `output_tokens` broken on WSL2 host |
| `2026-09-01-followup` | fix verification, invalid | b1 never used its tool; one b4 row was an August transcript |
| `2026-09-02-followup` | fix verification, valid | `within` scoping works; description steering does not; b4 lost recall once from tool-surface friction |
| `2026-09-02-toc` | toc-only ladder, reps 5 | **toc_dir reproduces: turns 8→4, cache_read 127k→83k, recall unchanged.** toc_file alone never helps. Named-file task: no toc gain. Three reps excluded (harness incidents, §2) |
| `2026-09-02-compact` | compact ladder, two aborted starts | no data; `ABANDONED.md` says why the root and its stale lock are left in place |
| `2026-09-02-compact2` | compact ladder, reps 2 (**a probe, not a dataset**) | compact delivers the bytes (49k→13k, cache_creation 57k→29k) but a2's turn win does not reproduce and precision drops 0.96→0.82. **Default flip blocked**; needs reps 5 to settle (§3) |

The one-line summary of the whole programme so far: on Sonnet-class agents, quarry's LSP tools do
not beat grep on correctness or cost; a directory table of contents halves exploration cost on
unfamiliar code without changing the answer. Everything below is about making that one win cheaper
and deciding where it ships.

## 2. Rules learned the hard way (each one cost a run)

1. **Do not edit quarry source while a matrix is running.** `prepare-session` rebuilds `quarry-mcp`
   from the working tree before *every repetition*. On 2026-09-02 the compact-toc code landed mid-run;
   every later session got a server advertising `compact: true`, and two agents used it. Now recorded:
   `provenance.json` carries `server_hashes` per rep, the final table prints `!! the quarry-mcp binary
   changed during this root` when more than one build was used, and `run.sh` warns when the tree is
   dirty. The rule still has to be obeyed by the operator; the harness only catches the violation.
2. **The run agent's prompt is now gated.** An orchestrator once dispatched the subagent with the
   dispatch description as its whole prompt (a1-toc-file rep 4). `GateRunPrompt` fails ingest unless
   the first user message opens with `PARALLEL_OPENING` and contains the task text. A rep that fails
   it is re-attempted by the loop; it is never scored.
3. **The outcome marker is written with Bash, never the Write tool.** `Write` is not on the session's
   allow list; a Write call stops on a permission prompt nobody is watching and the matrix waits
   behind it. `SKILL.md` says so now.
4. **`/tmp` worktrees vanish on reboot but stay registered.** `BuildWorktree` prunes before adding.
   Nothing to do; noted so nobody "fixes" it back.
5. **`output_tokens` is unusable on the WSL2 host (OSL-1033).** Claude Code 2.1.258 transcripts carry
   only streaming snapshots. Use `cache_read_input_tokens`, `cache_creation_input_tokens`, and turns.
6. **Cost numbers are per results root.** Never compare `duration_ms` or tokens across roots or
   hosts; correctness (recall/precision) may be compared by config id across roots.
7. **One tmux session at a time.** `run.sh` refuses to start while a `ladder-run` tmux session
   exists. If a previous run was interrupted: `tmux kill-session -t ladder-run`, then re-run; the
   driver resumes from what the results root already records.

## 3. NEXT: re-run the compact ladder at reps 5

**Run once already, at reps 2, as a cheap probe: `results/2026-09-02-compact2/` — read its
`conclusion.md`.** The short version: the compact form does cut the bytes exactly as designed (49 KB →
13 KB, `cache_creation` 57k → 29k), but in that root a2's turn halving does not reproduce under it
(5.5 [4–7], no longer separated from the control) and precision falls from 0.96 to 0.82 — below the
control's 0.935. Read volume went *down*, not up, so the reading is that the agent answered from a
thinner map rather than compensating for it. **The default flip below is blocked.** Two repetitions per
cell cannot carry that conclusion either way, so the next action is this same ladder at `reps: 5` into a
fresh root, no other changes: set `reps: 5` in `ladder-compact.yaml`, bump the `session_dir_template`
prefix (see the comment there), and run. If precision holds near 0.82 the form is rejected on evidence;
if it converges on 0.96 the probe was noise and the flip proceeds. The rest of this section is the
still-current operating manual for that run.

**Question.** The toc_dir win is paid as a 49 KB JSON tool result per run. The compact form
(`quarry toc dir --compact`, MCP `compact: true`, forced per session by `--toc-format`) renders the
same map — same paths, same line numbers — as plain text at a third of the bytes (10.9 KB → 3.1 KB
for a 25-file package; ~12 KB for the task-01 scope). Does the agent do the same thing with it?

**Ladder.** `bench/loomyard-eval/ladder/ladder-compact.yaml`: task 01, reps 5, three cells —
`a0-none`, `a2-toc-dir` (JSON forced), `a9-toc-dir-compact` (compact forced). Tool delivery only;
annex delivery is deferred (§5). The ids match the toc ladder's so the two roots line up.

**Prerequisites on the machine** (all checked by `run.sh` before anything starts):
- `go`, a C toolchain (`cc`, for tree-sitter under CGO), `claude` (Claude Code CLI, logged in),
  `tmux`, `git`; `gopls` is not needed for this ladder (toc is tree-sitter only).
- A Loomyard checkout containing commit `975578cda8d6f3a81580bd4e73725e060211b766`. Point the
  harness at it with **one** gitignored file, `<repo-root>/.scratch/ladder.env`, containing
  `LADDER_LOOMYARD_REPO=/abs/path/to/loomyard` (or export the variable). No tracked file may carry a
  machine path.
- A clean `git status` (rule 1).

**Run** (from anywhere; it resolves the repo root itself):

```
bench/loomyard-eval/ladder/run-compact.sh
```

It does everything: preflight, builds `ladderbench` and `quarry-mcp`, creates or restores the task
worktree at `/tmp/loomyard-eval-01`, then for each cell and rep: `prepare-session` → `warm` → a live
`claude` session in tmux session `ladder-run` running `/ladder-run` → `ingest`. Then the scoring
session, `summarize`, `provenance.json`, and a per-cell table. Watch with
`tmux attach -t ladder-run`. Expect ~1 minute per run on the WSL2 host plus a scoring session: about
25 minutes for 15 runs. It is resumable: re-run the same command and it skips what the root already
has. Results land in `bench/loomyard-eval/ladder/results/<today>-compact/`.

**Before reading any number**, check the root:
- `provenance.json`: `quarry_dirty` false (or only the results files), one distinct value in
  `server_hashes`. **Do not check `server_vcs_modified`** — it is structurally `"true"` on every run,
  because `runmatrix` creates the untracked results root before the binary is built; see the harness
  note in `results/2026-09-02-compact2/conclusion.md`. `server_hashes` is the real evidence that the
  binary did not change mid-run.
- The printed table has no `!!` lines (an unused tool, or a changed server binary).
- `raw/*/*/ingest.json` has no fatal findings; a rep that failed `run_prompt` was re-attempted.
- Spot-check one `a9` transcript: the `toc_dir` result starts with `{"results":[{"compact":"# ` and is
  ~12 KB; one `a2` transcript: the result carries `"files":[` and is ~49 KB.

**Then write `results/<today>-compact/conclusion.md`** in the shape of
`results/2026-09-02-toc/conclusion.md`: provenance line, exclusions if any, the per-cell table
(median [min–max] of turns, greps, cache_read, cache_creation, duration, recall, precision, plus Read
chars and tool chars from the transcripts), and the reading. The reading is decided by a9 vs a2:
- cache_creation and cache_read down, turns still 4, recall still ~0.42 → the form is free; adopt it
  (next bullet).
- turns or Read volume up → the one-sentence-per-file cap (120 chars) cost the agent information it
  used; try `leadMaxRunes` higher or `--doc-sentences`-style knobs before concluding.
- recall down → same; do not adopt until understood.

**If compact holds at reps 5 — it did not at reps 2, see the top of this section — flip the defaults**
(about an hour, engine side, no benchmark needed):
- MCP server: default `compact` for both toc tools, JSON on `compact: false`. The only reader of MCP
  output is an LLM. Update the two tool descriptions to describe the compact lines.
- CLI: keep `toc dir|file` JSON as the scripting contract; add a `quarry map <path…>` verb that
  takes files and directories alike and prints the compact form. Then point `docs/when-to-use-quarry.md`
  at `map` as the recommended first call on unfamiliar code.
- Optionally re-run `run-toc.sh` afterwards against a fresh root if a clean n=5 JSON-vs-compact baseline
  under the new defaults is wanted; not required.

## 4. Open follow-ups from the toc run (cheap, do when convenient)

- `results/2026-09-02-toc`: reps a1-toc-file 4 and 5 and a2-toc-dir 3 are excluded in the conclusion
  but still in `summary.json`. To get a clean n=5: `ladderbench invalidate --config-id <id> --rep <n>
  --ladder bench/loomyard-eval/ladder/ladder-toc.yaml --results-root <root>` for each, then
  `run-toc.sh` with that root as its optional second argument re-runs only those three and
  re-summarizes. The conclusions do not depend on it.
- Un-ignore `results/**/raw/` so transcripts travel with the repo (see the 2026-09-01-followup
  conclusion for why).
- The orchestrator session (the `/ladder-run` skill runner) spends 10–20 tool calls reading harness
  source before dispatching, every rep. It does not touch the measured subagent, only wall-clock;
  tightening `SKILL.md`'s opening ("do not read harness source; the commands below are complete")
  would cut a minute per rep.

## 5. Deferred, with the reason

- **Annex ladder (`ladder-annex.yaml`, `run-annex.sh`) — built, tested, dry-run, not to be run now.**
  Injecting pre-computed quarry output into the prompt presumes (1) the caller already knows exactly
  what the agent is going to do, and (2) a non-interactive dispatch. That is the Loomyard shape, not
  the current one; run it when the Loomyard integration (§7) is the next thing being built. The
  harness support stays: `annex:` on a config, `annexes:` recipes on a task (`toc-dir`, `toc-full`,
  `toc-file`, `impact`, `plan-pack`, `compact: true`, `drop_callers: N`), generated by
  `prepare-session` from the quarry CLI, injected by `next-run` as one neutral paragraph, copied at
  ingest as `annex.txt` + `annex.meta.json`. The degraded-annex cell (one caller dropped) is the
  measurement behind the "never inject unverified" rule in §6 item 4.
- **Weak-model arm (Haiku)** — dropped. Operator does not use Haiku (too weak for reasoning and for
  fixing failing tests), so the deployment question it answers does not arise.
- **Tasks 06–08** (interface-method impact at scale, whole-repo dispersion, structurally invisible
  references) — designs in the research report §4; build only if an LSP-shaped question comes back.
  The toc results do not motivate them.
- **toc_file as a rung** — retired; it never separated from control in either ladder.
- **Review-time injection** — negative prior stands (`bench/loomyard-eval/scripts/gen_compact_toc.py`
  experiment; task-03 scorecard); not re-tested.

## 6. Quarry engine/MCP changes, ranked (independent of any benchmark round)

1. `rename_impact` tool wrapping LSP `textDocument/rename` — the fasit procedure as a product.
2. Expose `implementations` (`textDocument/implementation` is already wired internally).
3. `within` scoping default-on (workspace-wide becomes the explicit opt-out). Verified working
   2026-09-02; both agents had to know to pass it.
4. Per-entry `verified: true|false|"partial"` on refs/impact — prerequisite for any mechanical gate.
5. 1-based positions at the MCP boundary, plus a `file + line + symbol` call-site form and lenient
   input (position and symbol together; `Type.Method` qualifiers stripped). Every b2/b4 failure in
   the followup was an addressing failure.
6. Compact output — **done for toc**; pending for impact/refs (`file:line: enclosing signature`
   lines) once toc's compact form is adopted (§3).
7. Warm-daemon support beyond Go, or document the other languages as cold-spawn-only.

## 7. Loomyard mechanical integration (later)

The direction the evidence supports: quarry called from Loomyard's own Go code via the `quarry/`
facade, never by the LLM — a toc pack for unfamiliar-code exploration (the measured win), a plan-time
impact/plan pack for "use X and Y to edit Z" implementers (the annex ladder's b11 cell measures it),
and deterministic gates on deletes/moves/renames (`assert-no-callers`, before/after `impact` sets).
Blocked on §6 item 4: a gate that consumes unverified results marked `resolution: complete` recreates
the 31-false-positive incident (`docs/research/scout-agent-usage-findings.md`). Measured by the annex
ladder when the time comes.

## Dependency order

```
3 probe at reps 2 (done, blocked the flip) ──► 3 re-run at reps 5 ──► 3's default flip ──► 6.6 (compact impact/refs)
4 (toc-run tidy-ups)  — any time
5 (annex run)         — only when 7 is being built
6.1–6.5               — independent of runs; 6.4 gates 7
```
