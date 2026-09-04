# 2026-09-04 breadth matrix (M1) — conclusion

`ladder-toc.yaml` via `go run ./bench/loomyard-eval/ladder/cmd/ladder run`, reps 5, host `Neve`, quarry
commit `f453b123876063c38e2216f556d3797cd1df1642`, server built from `./cmd/quarry-mcp`. One invocation
(`provenance.json.invocations` has exactly one entry, written 2026-09-04T18:03:56Z). 30 measured
repetitions across three shapes — `b0-none`/`b8-toc-dir` (task 04, impact analysis where the task text
names the file), `c0-none`/`c1-toc-dir` (task 02, three-package exploration), `d0-none`/`d1-toc-dir`
(task 06, whole-repo no-scope-hint exploration) — all ingested and scored (`unscored_count: 0` in every
cell; `recall_n` and `precision_n` are 5 in every cell, per `table.txt`). Ladder a (task 01) was not
re-run in this root; it was measured by T7 in `results/2026-09-04-toc` on this host and harness, and
re-measuring it here would have spent ten real calls for no new information.

## Numbers (median [min–max], this results root only)

Every figure below is quoted verbatim from `summary.json`'s per-cell `metrics` block or `table.txt`'s
`recall_n`/`precision_n` columns; none is re-derived. Column names are `table.txt`'s; the summary JSON
spells four of them differently: `cache_read` is `cache_read_input_tokens`, `cache_creation` is
`cache_creation_input_tokens`, `prefixed_tool_uses` is `quarry_tool_uses`, and `grep_fallback` is
`grep_fallback_total`.

### Ladder b — task 04, impact analysis (task text names the file)

| metric | b0-none (control) | b8-toc-dir |
|---|---|---|
| turns | 8 [6–8] | 7 [6–11] |
| duration_ms | 21761 [16548–29002] | 24340 [17088–30679] |
| cost_usd | 0.1553 [0.1249–0.2062] | 0.1327 [0.1184–0.2067] |
| cache_read | 120556 [87120–157801] | 119833 [91031–152082] |
| cache_creation | 13501 [12147–19415] | 11651 [10496–19065] |
| output_tokens | 18 [9–34] | 17 [15–34] |
| input_tokens_total | 133241 [99275–177230] | 131494 [101535–171159] |
| tool_uses | 7 [5–7] | 6 [5–10] |
| prefixed_tool_uses | 0 [0–0] | 0 [0–1] |
| grep_fallback | 4 [3–7] | 4 [3–5] |
| tool_result_bytes | 24094 [19772–30775] | 23368 [19952–36823] |
| read_bytes | 0 [0–16311] | 13907 [11278–24922] |
| recall (n) | 1 [1–1] (n=5) | 1 [1–1] (n=5) |
| precision (n) | 1 [1–1] (n=5) | 1 [1–1] (n=5) |

### Ladder c — task 02, three-package exploration (scope-axis midpoint)

| metric | c0-none (control) | c1-toc-dir |
|---|---|---|
| turns | 15 [13–19] | 12 [10–14] |
| duration_ms | 48521 [41756–50445] | 55784 [38786–74916] |
| cost_usd | 0.2786 [0.2279–0.3426] | 0.3319 [0.3168–0.3769] |
| cache_read | 183475 [116344–222362] | 231597 [142966–285120] |
| cache_creation | 27822 [18109–33734] | 30180 [29932–33186] |
| output_tokens | 34 [28–45] | 30 [16–62] |
| input_tokens_total | 201598 [145473–256110] | 261620 [175496–315068] |
| tool_uses | 14 [12–18] | 11 [9–13] |
| prefixed_tool_uses | 0 [0–0] | 3 [3–4] |
| grep_fallback | 3 [2–7] | 3 [3–6] |
| tool_result_bytes | 49997 [40039–77140] | 68933 [58366–81166] |
| read_bytes | 44629 [27580–67755] | 14800 [2516–38926] |
| recall (n) | 0.52 [0.45–0.62] (n=5) | 0.62 [0.42–0.66] (n=5) |
| precision (n) | 0.95 [0.93–1] (n=5) | 0.95 [0.82–1] (n=5) |

### Ladder d — task 06, whole-repo exploration with no scope hint (scope-axis far end)

| metric | d0-none (control) | d1-toc-dir |
|---|---|---|
| turns | 12 [11–13] | 10 [8–15] |
| duration_ms | 44150 [35454–52193] | 39595 [29971–47391] |
| cost_usd | 0.3361 [0.3005–0.4105] | 0.3026 [0.2089–0.3617] |
| cache_read | 216642 [153034–262051] | 137544 [50213–251436] |
| cache_creation | 36906 [27128–47877] | 33577 [22509–40376] |
| output_tokens | 13 [6–60] | 8 [5–18] |
| input_tokens_total | 264531 [189227–297419] | 165337 [90595–287293] |
| tool_uses | 11 [10–12] | 9 [7–14] |
| prefixed_tool_uses | 0 [0–0] | 1 [1–1] |
| grep_fallback | 3 [2–4] | 2 [1–5] |
| tool_result_bytes | 81031 [59649–90650] | 78812 [51495–79071] |
| read_bytes | 61181 [49613–71316] | 67865 [39561–71316] |
| recall (n) | 0.8 [0.71–0.82] (n=5) | 0.75 [0.62–0.82] (n=5) |
| precision (n) | 0.85 [0.8–0.93] (n=5) | 0.86 [0.8–0.93] (n=5) |

## Per-shape verdict

The separation rule (overview's `## Shared Decisions`, `separation-decision-rule`): a shape separates
only when (1) at least one cost metric's comparison entry is `separated: true`, (2) that metric's
median moves in the direction the toc hypothesis predicts (rung cheaper than control), and (3) neither
recall nor precision degrades. Anything short of all three is no separation.

**Ladder b — no separation.** `summary.json`'s comparisons mark `separated: false` for every one of
`b8-toc-dir`'s cost metrics (turns, duration_ms, cost_usd, cache_read, cache_creation, output_tokens,
input_tokens_total, tool_uses, grep_fallback, tool_result_bytes, read_bytes) — including
`quarry_tool_uses` itself (`median 0=0`, `cell-range [0, 1]` vs `control-range [0, 0]`), so requirement
(1) fails outright: no cost metric separates at all, let alone in the predicted direction. This is the
negative control's expected result — the task text already names the file, so a directory survey has
nothing to add — and it landed as expected: turns, cost_usd and several other medians run slightly
*lower* for `b8-toc-dir` than the control (e.g. cost_usd 0.1327 vs 0.1553), but every range overlaps, so
none of it separates. Recall and precision are identical between the two cells (1/1 both), so neither
degrades either way. `quarry_tool_uses`'s own range, `[0, 1]`, means the tool was called in at most one
of five repetitions — most of `b8-toc-dir`'s reps never touched the granted tool at all, consistent with
a task that already tells the agent where to look.

**Ladder c — no separation.** The one comparison marked `separated: true` for `c1-toc-dir` is
`quarry_tool_uses` (`median 3=0`, `cell-range [3, 4]` vs `control-range [0, 0]`) — a usage count, not a
cost metric, so it does not satisfy requirement (1) on its own. Every genuine cost metric is
`separated: false`, and the medians mostly move the *wrong* way for the hypothesis: `c1-toc-dir` costs
more, not less, than its control on turns is the one exception (12 vs 15, lower) but duration_ms (55784
vs 48521), cost_usd (0.3319 vs 0.2786), cache_read (231597 vs 183475), cache_creation (30180 vs 27822),
input_tokens_total (261620 vs 201598) and tool_result_bytes (68933 vs 49997) all run higher for the
toc-granted cell. `read_bytes` runs much lower (14800 vs 44629) — the agent read less of the actual
source once it had a directory map, which is the expected mechanism, but it did not translate into a
net cost win on this task. Recall moved up (0.62 vs 0.52) and precision held (0.95 both), so correctness
did not degrade — but with no cost metric separating, this shape does not separate either.

**Ladder d — no separation.** `d1-toc-dir`'s `quarry_tool_uses` is the one `separated: true` entry here
too (`median 1=0`, `cell-range [1, 1]` vs `control-range [0, 0]`), again a usage count rather than a
cost metric. Every cost metric is `separated: false`. Medians point the predicted direction more often
here than in ladder c — turns (10 vs 12), duration_ms (39595 vs 44150), cost_usd (0.3026 vs 0.3361),
cache_read (137544 vs 216642), cache_creation (33577 vs 36906), output_tokens (8 vs 13),
input_tokens_total (165337 vs 264531) and tool_uses (9 vs 11) all run lower for `d1-toc-dir` — but every
one of those ranges overlaps its control's range, so requirement (1) still fails for all of them.
Recall moved down slightly (0.75 vs 0.8) and precision moved up slightly (0.86 vs 0.85); both ranges
overlap and neither is marked `separated`, so this reads as noise, not degradation, but it is the one
shape where a real cost effect at a larger n is most plausible from the medians alone.

**Aggregate: toc does not separate anywhere in this root.** All three shapes fail requirement (1) —
no shape has even one cost metric marked `separated: true` in the direction the hypothesis predicts.
Ladder b behaved as its role requires (a flat negative control). Ladder c's medians point away from the
hypothesis on most cost metrics. Ladder d's medians point toward it on every cost metric without any
of them separating at n=5. Read together with `results/2026-09-04-toc`'s finding that the original
ladder-a win itself did not reproduce at n=5, this root gives no shape in the 2026-09-04 breadth
matrix a measured cost win for directory-level `toc`.

## Coverage: gates, invalidations, drift, provenance

- **Gate 1 (`granted_tool_used`) never fired for any rung cell.** `summary.json` carries no `gate1`
  field for `b8-toc-dir`, `c1-toc-dir` or `d1-toc-dir` (the field is `omitempty` and populated only
  when the max `quarry_tool_uses` across a cell's repetitions is zero). Each rung cell's own
  `quarry_tool_uses` range confirms this directly: `b8-toc-dir` `[0, 1]`, `c1-toc-dir` `[3, 4]`,
  `d1-toc-dir` `[1, 1]` — every rung cell had at least one repetition that actually called the granted
  `toc` tool, so none of the three measured only the tool's prompt cost. `b8-toc-dir` is the one cell
  where most repetitions never touched the tool (median 0), which is the expected behaviour for its
  negative-control role, not a gate-1 finding — gate 1 checks the *maximum* across repetitions, not the
  median, and `b8-toc-dir`'s maximum is 1.
- **Invalidation record.** Four attempts across four repetitions were invalidated and retried
  automatically within this same invocation, all with `cause: runner_error` ("the measured claude
  process failed") and no other cause value appearing anywhere in this root:
  - `b0-none` repetition 3, attempt 1 (`raw/b0-none/3.invalid-1/invalid_reason.txt`)
  - `b8-toc-dir` repetition 1, attempt 1 (`raw/b8-toc-dir/1.invalid-1/invalid_reason.txt`)
  - `d0-none` repetition 2, attempt 1 (`raw/d0-none/2.invalid-1/invalid_reason.txt`)
  - `d0-none` repetition 4, attempt 1 (`raw/d0-none/4.invalid-1/invalid_reason.txt`)

  Each of the four repetitions completed on its second attempt (well inside `MaxAttempts`, 3), and no
  repetition in this root was attempt-exhausted. No other cell (`c0-none`, `c1-toc-dir`, `d1-toc-dir`)
  carries any `.invalid-*` directory — every one of their repetitions completed on the first attempt.
  Per the M2-invalidation-record decision, `detail` values are never quoted verbatim beyond the fixed
  constructed string the harness itself writes (`"the measured claude process failed"`), which carries
  no machine path.
- **`quarry_dirty: true`, one dirty file, the tolerated carve-out.** `provenance.json`'s single
  invocation entry records `quarry_dirty: true` with `quarry_dirty_files: ["_mill/briefs/implement-matrix-and-conclusion-r1.md"]`.
  This is the plan's own named carve-out for exactly this file: the mill-go orchestrator writes the
  currently-executing batch's brief before any card in the batch can commit it, no card here has the
  standing or a `Commit:` line to commit it, and the file carries no machine path. It is not the
  resume-tolerance carve-out (this is not a resumed invocation and the file is outside the results
  root) — it is the separate, narrower carve-out card 14 names for this exact path.
- **One invocation.** `provenance.json.invocations` has exactly one entry (written
  2026-09-04T18:03:56Z, `reps_effective: 5`, `quarry_commit: f453b123876063c38e2216f556d3797cd1df1642`,
  `loomyard_commit: 72c23d9eecc1fa55add567622093a8bbbfba8c1d`). No resume was needed: every selected
  cell reached `5/5` complete, non-blinding-failed repetitions on this one invocation, so card 15's
  remedies 1–4 were never exercised beyond the automatic same-invocation attempt retries recorded
  above.
- **No permitted repetition deletion.** Card 15's remedy 3 (deleting one `raw/` repetition directory to
  force a re-score) was never used in this root — `unscored_count: 0` in every cell on the first
  invocation, so the condition that remedy exists for never arose.
- **Session-fingerprint drift.** `table.txt` records the expected drift between every rung repetition
  and `b0-none/1`: each of `b8-toc-dir`'s, `c1-toc-dir`'s and `d1-toc-dir`'s five repetitions carries
  the extra tool `mcp__quarry__toc`, `mcp_servers: [quarry]`, and `mcp_server_statuses:
  {quarry: connected}`, none of which the control repetition carries. This is the manipulation under
  test, not a defect.
- **The M2 invalidation-reason-file change (batch 1) is not a measured metric.** It closes the loop on
  why M2 was folded into this task: the four `runner_error` causes named above are legible here in a
  way T7's root could not make them (T7's own coverage section could only say three retries had "no
  diagnosable cause on disk"). That diagnostic value is reported here as context, not as a benchmark
  result, and it enters no comparison above.

No cell in this root reports `unscored_count` non-zero and no cell falls short of `5/5`, so no shape is
reported as incomplete or unmeasured — every comparison above is a full-`n=5` result on both sides.

## What this settles

Named in the roadmap's own terms, for the operator's own T8-unpark decision (not decided here): this
root is **not** a measured win that re-establishes directory-level `toc`'s value to an agent. All three
shapes in the 2026-09-04 breadth matrix — the negative control, the scope-axis midpoint, and the
scope-axis far end — reached `5/5` clean repetitions with gate 1 confirming the tool was actually used
where it was granted, and none of them shows a cost metric separating in the predicted direction. Read
alongside `results/2026-09-04-toc`'s finding that the original ladder-a win itself did not reproduce at
a clean n=5, the breadth question this task set out to answer — does toc pay off anywhere across scope,
even where the original win did not hold up — comes back negative in this root, at this n. Ladder d's
medians (the shape furthest from the original win's two-package scope) point toward the hypothesis on
every cost metric without separating; a larger n is the only lever this root's own design offers for
distinguishing that pattern from noise, and taking it is the operator's call, not this conclusion's.
