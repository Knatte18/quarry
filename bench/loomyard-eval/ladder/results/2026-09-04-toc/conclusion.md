# 2026-09-04 toc ladder rerun (T7) — conclusion

`ladder-toc.yaml` via `go run ./bench/loomyard-eval/ladder/cmd/ladder run`, reps 5, host `Neve`, server
built from `8804728f3c5a658c1687588cbb216b2c0070765e` (this task's own `mill-go: start batch
matrix-run` commit, i.e. the harness fix below plus everything merged up to that point). 10 runs (2
cells × 5 reps), all ingested and scored (`unscored_count: 0` in both cells; `recall_n` and
`precision_n` are 5 for both, per `table.txt`).

Only ladder a ran — `--cells a0-none,a2-toc-dir`, per `ladder-toc.yaml`'s own header comment ("T7
runs `--cells a0-none,a2-toc-dir` at this file's own `reps: 5`"). Ladder b (`b0-none`, `b8-toc-dir`) is
declared in the same file but was never selected; `provenance.json`'s `selected_cells` names only the
two ladder-a ids, and nothing below reports on ladder b.

## Numbers (median [min–max], this results root only)

| metric | a0-none (control) | a2-toc-dir |
|---|---|---|
| turns | 12 [11–15] | 12 [12–17] |
| duration_ms | 51903 [49024–66790] | 62668 [48202–69588] |
| cost_usd | 0.316 [0.272–0.361] | 0.374 [0.268–0.427] |
| cache_read | 207207 [194250–273760] | 227014 [168364–244386] |
| cache_creation | 29498 [23536–32709] | 39474 [17412–44429] |
| output_tokens | 35 [19–57] | 17 [10–21] |
| input_tokens_total | 236719 [217800–304673] | 256472 [207848–288833] |
| tool_uses | 11 [10–14] | 11 [11–16] |
| prefixed_tool_uses | 0 [0–0] | 2 [2–2] |
| grep_fallback | 3 [3–6] | 3 [1–6] |
| tool_result_bytes | 63098 [51976–69127] | 87883 [53115–95120] |
| read_bytes | 38901 [30793–55554] | 36999 [32553–38991] |
| recall (n) | 0.40 [0.38–0.46] (n=5) | 0.43 [0.34–0.50] (n=5) |
| precision (n) | 0.88 [0.76–0.93] (n=5) | 0.93 [0.82–0.94] (n=5) |

Column names are the rendered table's (`table.txt`); the summary JSON keys `cache_read_input_tokens`,
`cache_creation_input_tokens`, `quarry_tool_uses` and `grep_fallback_total` are the same four figures
under their JSON spelling (see plan's `metric-key-spellings` decision). Every number above is quoted
from `summary.json`'s per-cell `metrics` block or `table.txt`'s `recall_n`/`precision_n` columns — none
is re-derived.

Recall and precision are reported separately from every cost metric, and each with its own sample
size, because a cell can report cost at n=5 while correctness sits at a smaller n. That did not happen
here: both `recall_n` and `precision_n` are 5 in both cells, so recall/precision below are read at the
same n as the cost figures.

## Ladder a — the headline claim does **not** reproduce here; if anything, the direction reverses

`summary.json`'s comparison entries mark `separated: false` for **every** metric except
`quarry_tool_uses`, including the two the August finding was built on:

- **turns:** `median 12=12`, `cell-range [12, 17]` vs `control-range [11, 15]` — `separated: false`.
- **cache_read_input_tokens:** `median 227014=207207`, `cell-range [168364, 244386]` vs
  `control-range [194250, 273760]` — `separated: false`.

`separated` is a strict no-overlap test on the min–max ranges, and at n=5 a real effect can be present
without it firing — so the medians matter too, not just the boolean. Here the medians point the wrong
way for the August effect to be reappearing at a smaller magnitude: `a2-toc-dir`'s median turns equal
the control's (12=12, not lower), and every other cost metric's median is *higher* than the control's —
duration (62668 vs 51903), cost_usd (0.374 vs 0.316), cache_read (227014 vs 207207), cache_creation
(39474 vs 29498), input_tokens_total (256472 vs 236719), and tool_result_bytes (87883 vs 63098).
Only `output_tokens` runs lower (17 vs 35), and `read_bytes` is close to flat (36999 vs 38901).

**Gate 1 (`granted_tool_used`) never fired for `a2-toc-dir`:** `prefixed_tool_uses` is `2` in every one
of its 5 repetitions (`cell-range [2, 2]`), so the cell measured the granted `toc` tool being used, not
merely its prompt cost. The tool was called and still did not produce a turns/cache-read separation on
this rerun.

Recall and precision both sit slightly higher in `a2-toc-dir` than in the control (0.43 vs 0.40 recall,
0.93 vs 0.88 precision), but both ranges overlap heavily (`a2-toc-dir` recall range [0.34, 0.50] vs
control [0.38, 0.46]) and neither comparison is marked `separated`. Correctness did not move either
direction in a way this n=5 root can distinguish from noise.

## What this settles

1. **The August a2-toc-dir cost win does not reproduce on this host, this harness build, and this
   n=5.** The prior finding (see the prior-record section below) was turns roughly halved and
   cache_read down a third at n≤4. Here, at a clean n=5 for both cells, no cost metric separates and
   every cost metric except output_tokens trends the other way (equal-or-worse for `a2-toc-dir`, not
   better). This does not mean the August effect was wrong on its own root; it means it is not the
   robust, host- and harness-independent effect the rewrite needed to lean on. Whatever changed
   between the two roots — host, harness rewrite, CLI version, model version, cache behaviour, or
   genuine noise at small n — this rerun cannot separate those causes from each other; it can only
   report that the comparison, run again cleanly, came back flat-to-reversed.
2. **The tool was used, not merely granted.** Gate 1 raised no finding; every `a2-toc-dir` repetition
   called `toc` exactly twice. The flat cost result is not an artifact of an unused tool.
3. **Correctness is unaffected either way**, consistent with the August finding on that one point:
   recall and precision ranges overlap between the two cells at n=5.
4. **This root does not answer whether toc pays off under a different scope or task.** Ladder b (the
   negative control, task 04) was not run in this matrix; T7's scope was reproducing the ladder-a
   result specifically.

## Prior-record section — `v1-final`, `results/2026-09-02-toc`, reps 5, host OSL-1033 (WSL2)

Cited by id only, never merged into the table above (cost numbers compare only within one results
root; different host, harness, CLI version and cache behaviour make a numeric merge dishonest). Per
`git show origin/v1-final:bench/loomyard-eval/ladder/results/2026-09-02-toc/conclusion.md`:

- `a2-toc-dir` there ran at **n=4** (one repetition, rep 3, was excluded — an accidental compact-form
  response from a mid-matrix harness edit, not a blinding or gate failure) against `a0-none` at n=5.
- Prior figures (median [min–max]): turns 4 [4–4] vs control 8 [7–11]; cache_read 83k [82–106k] vs
  control 127k [116–180k]; recall 0.42 [0.38–0.47] vs control 0.43 [0.38–0.54]; precision 0.94 both.
- The prior conclusion's own claim: "half the turns... cache_read down a third... recall and precision
  unchanged." Correctness metrics (recall, precision) may be compared by id across roots per the
  plan's stated exception; this root's recall/precision (0.43/0.93 for `a2-toc-dir`, n=5) sit close to
  the prior root's (0.42/0.94, n=4) — correctness is consistent across both roots. The cost comparison
  is the one that does not carry across: this root's `a2-toc-dir` median turns (12) and cache_read
  (227014) are far above the prior root's (4 turns, 83k cache_read) in absolute terms, but that
  absolute gap is expected and uninformative — task 01's exploration is being measured on a different
  host/harness/CLI-version pairing throughout this whole root, including in the control, whose own
  turns (12) and cache_read (207207) are likewise far above the prior root's control (8 turns, 127k).
  What this root can say is qualitative only, and qualitatively the two roots disagree: the prior root's
  `a2-toc-dir` separated sharply below its control on both turns and cache_read; this root's does not
  separate from its control on either metric at all.

## Coverage: gates, invalidations, drift, provenance

- **No cell finished short of 5 repetitions.** Both `a0-none` and `a2-toc-dir` reached `5/5` complete,
  non-blinding-failed repetitions (`run.json` `state: "complete"`, `blinding_failed: false` in every
  one of the 10 kept repetitions).
- **Gate 2 checks (a) and (b) (`control_blinding_mcp_prefix`, `control_blinding_repo_root`) never
  invalidated a repetition.** No `run.json` in either cell carries `blinding_failed: true`, and no
  `observations` entry exists for any repetition (every `run.json`'s `observations` field is `null`) —
  so check (c), the always-non-fatal `target_origin_quarry_mention` bare-token observation, never fired
  either. Per the plan's own caveat, that silence is expected and is not evidence of anything on its
  own; it does not, for instance, prove the control's blinding held up to more scrutiny than the checks
  themselves perform.
- **Check (d), `CheckRenderedControlPrompt`, never failed.** It is deterministic and pre-dispatch, so a
  failure there would have stopped the run outright before any API spend; the matrix completed
  normally, so it did not fire.
- **No `ScanMemoryPaths` abort.** The run log records no abort and no `blinding_failed` repetition
  carrying a memory-path finding; the memory-path scan ran clean.
- **`a0-none` reps 4 and 5 each needed attempt retries, cause not recoverable from the surviving
  artifacts.** `raw/a0-none/` carries `4.invalid-1`, `5.invalid-1` and `5.invalid-2` alongside the
  completed `4` and `5`. `a0-none` is the control cell (empty `allowed` list), so
  `CheckServerConnected` never runs against it (`run.go`'s attempt loop gates that check on
  `!cfg.IsControl()`) — meaning these are **not** server-not-connected invalidations, and no
  `server_connect_failure.txt` was written into any of the three invalid directories (confirmed:
  each contains only `transcript.jsonl`). They are also not gate-2 (a)/(b)/(c) findings: those are
  recorded in a completed repetition's own `run.json` with `blinding_failed: true`, which none of the
  discarded attempts have (they were never accepted into a `run.json` at all — `InvalidateRep` renamed
  the attempt directory away before a state file could be written). Reading the three invalid
  transcripts by hand: each ends in a complete, non-error assistant turn (`is_error: false`,
  `terminal_reason: "completed"`) with a single well-formed fenced JSON block. That rules out an
  unparseable-answer formatting miss as the likely cause and points at `invokeMeasuredProcess`'s own
  error return — i.e. the `claude` invocation itself returning a non-zero exit or a runner-level error
  despite a complete-looking transcript — but the harness's on-disk contract does not preserve that
  specific cause anywhere (`ServerConnectFailureFile` is written only ahead of the
  server-not-connected path), so this is reported as an open, unresolved retry cause rather than a
  named finding. Neither repetition was attempt-exhausted: `4` succeeded on its 2nd attempt, `5` on its
  3rd (`MaxAttempts` is 3, so `5` used its full ceiling but did not exceed it). `a2-toc-dir` has no
  `.invalid-*` directories at all — every one of its 5 repetitions completed on the first attempt.
- **`quarry_commit` equality across every entry of `provenance.json`'s `invocations` list.** There is
  exactly one invocation entry, `quarry_commit: 8804728f3c5a658c1687588cbb216b2c0070765e` — trivially
  consistent with itself, and matching the top-level `provenance.json.quarry_commit` field. The
  out-of-band `sha256sum` reading taken after that invocation, from `.scratch/ladder-toc-run.log`'s
  implementer summary: `f76e76f619cc76444cc57733389fc1c99fcfadd7875b4271b5da7406228917bd`, which
  matches every one of `provenance.json`'s five `server_hashes["a2-toc-dir/N"]` entries. These are the
  values that were checked, not an absence of warning.
- **`quarry_dirty` and `quarry_dirty_files`, the one invocation entry.** `quarry_dirty: true`,
  `quarry_dirty_files: ["_mill/briefs/implement-matrix-run-r1.md"]`. This is **not** the plan's stated
  resume carve-out — that carve-out covers a *resumed* invocation seeing its own results root's
  untracked machine artifacts, and every listed path being inside the results root is what makes it
  benign. This matrix took exactly one invocation (no resume occurred), and the one dirty file named is
  outside the results root — it is this task's own in-flight mill orchestrator brief for the
  matrix-run batch, `_mill/briefs/implement-matrix-run-r1.md`. It falls under the plan's *second*
  `clean-tree-and-no-edits-mid-matrix` carve-out, added at holistic review round 1:
  `_mill/briefs/<currently-executing-batch>*.md` is written by the mill-go orchestrator's prepare
  stage before the implementer session (and therefore card 7) ever starts, no card in this plan has
  the standing or a `Commit:` line to commit it, and the orchestrator's own next commit point after
  prepare fires only after the whole batch has finished — so requiring it committed before the
  invocation is unsatisfiable by any card here. The file carries no machine path and was later
  committed by the pipeline's own per-batch commits. This is not a deviation; it is the narrow,
  named exception the plan states for exactly this file, and no other dirty entry appeared.
- **Session-fingerprint drift.** `table.txt` records session-fingerprint drift between every
  `a2-toc-dir` repetition and `a0-none/1`: `a2-toc-dir` sessions carry the extra tool
  `mcp__quarry__toc`, list `mcp_servers: [quarry]`, and show `mcp_server_statuses: {quarry: connected}`,
  none of which `a0-none` carries. This is the intended manipulation under test (the granted tool and
  its connected server), not a defect — it is exactly the difference the ladder exists to measure, and
  is named here rather than left as an unexplained drift line.
- **Invocation count and attempt exhaustion.** The matrix ran to completion in a single invocation
  (`provenance.json.invocations` has one entry; `.scratch/ladder-toc-run.log`'s own summary: "all
  selected cells reached 5/5 complete repetitions on the first invocation; no re-invocation was
  required"). No repetition was attempt-exhausted (see the `a0-none` retry note above).
- **Card 8's hand verification: agreement, no finding.** `a2-toc-dir/1`'s hand-counted turns (17),
  `toc` calls (2), and cache-read tokens (244386) matched `usage.json`'s own recorded figures for that
  repetition exactly, and all three sat inside `a2-toc-dir`'s cell range in `summary.json` (two of them
  at the range's own maximum). Full working in `.scratch/hand-verify.md`. Nothing here invalidates the
  numbers this conclusion quotes.
- **`output_tokens` is usable on this host.** Both cells report nonzero, varying `output_tokens`
  figures (a0-none median 35 [19–57]; a2-toc-dir median 17 [10–21]) — unlike the V1 host, where this
  figure was unusable and the prior conclusion omits it from its table.

## The pre-matrix harness fix

This task did not only measure: before the matrix ran, batch `harness-mcp-init-fix` fixed a defect
that made the harness unable to parse the transcript of any cell that actually connected an MCP server
(`SessionInit.MCPServers` was typed as a bare name list while Claude Code 2.1.236 emits objects
carrying a `status` field). Commits `0364aff` (`fix(ladder): type session-init mcp_servers as objects
and record server status`), `881ca8a` (a pinned regression test from a captured real transcript), and
`cbcc7ab` (`feat(ladder): invalidate a granted repetition whose mcp server did not connect`) landed at
16:21–16:28 (local), before `provenance.json`'s single invocation entry (written 14:32:04Z / 16:32
local). The matrix ran entirely against the post-fix harness — no repetition in this root was measured
by the pre-fix parser. The matrix did not restart: `provenance.json` and the run log show exactly one
invocation, and there is no `ABANDONED.md` under this results root or any `-r2`/`-r3` sibling. A reader
comparing this root against a later one should know the harness that produced it already carries this
fix, unlike the harness state at the start of this task.

## Section 11 — the raw tree

`results/**/raw/` stays untracked in this root, confirming `bench/loomyard-eval/ladder/.gitignore`'s
existing `results/*/raw/` entry as the settled answer to the rewrite plan's §11 open decision. Two
reasons: `raw/memory-paths.json` holds resolved auto-memory directory paths, and this repository's rule
is that no tracked file carries a machine path — `provenance.json`'s `memory_path_hashes` stores sha256
hashes of those paths for exactly that reason. Second, the raw tree here is ten transcripts of a
60-turn ceiling plus their answers, scores and usage files — large, per-host, and fully summarised by
the five files this root commits: `conclusion.md` (this file), `summary.json`, `provenance.json`,
`table.txt`, and `probe.md`. The decision is settled in this results root, `2026-09-04-toc`.

## What went wrong, and what to do next

- **The `a0-none` control-cell attempt retries (see "Coverage" above) have no diagnosable cause on
  disk.** The harness records a reason file only ahead of the server-not-connected path
  (`ServerConnectFailureFile`), which by design never runs for a control cell. A future harness change
  that also persists the runner-level error (or the process exit code) into the discarded attempt
  directory before `InvalidateRep` renames it away would close this gap — right now, three attempts
  across two repetitions were silently retried and succeeded, and nothing on disk says why the first
  attempts were rejected. This is a harness observability gap, not a defect in the code under test, and
  fixing it is a separate task.
- **The pre-matrix clean-tree check reported `quarry_dirty: true` against the `_mill/` brief file**
  named above under `quarry_dirty_files`. This is resolved, not open: the plan's
  `clean-tree-and-no-edits-mid-matrix` decision now names a second, narrow carve-out (added at
  holistic review round 1) for exactly `_mill/briefs/<currently-executing-batch>*.md`, on the grounds
  that no card in this plan has the standing or a `Commit:` line to commit that file before the
  invocation it precedes. It is named here only so a reader of this results root sees which file
  tripped the flag and why the flag is benign, not as an unresolved gap.
- **The headline separation this task set out to reproduce did not reproduce, cleanly, at n=5.** This
  is the finding, not a fault: it means the rewrite's `HANDOFF.md` and `docs/rewrite-plan.md` records
  should stop citing the August `a2-toc-dir` cost win as an established, cross-host result and instead
  cite this root's flat outcome alongside it (see the batch's other two cards for where that
  propagates). Determining *why* the two roots disagree — host, harness rewrite, CLI version, model
  version, or noise at small n — is a separate investigation, not something this conclusion can settle
  from the data it has.
