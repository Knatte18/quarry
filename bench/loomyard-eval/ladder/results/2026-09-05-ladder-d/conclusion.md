# 2026-09-05 ladder-d decisive rerun (M3) — conclusion

`ladder-toc.yaml` via `go run ./bench/loomyard-eval/ladder/cmd/ladder run`, `-cells d0-none,d1-toc-dir
-reps 15`, host `Neve`, quarry commit `0ae4daa62af167be55142396417eed6dfa86d9dd`, server built from
`./cmd/quarry-mcp`. One invocation (`provenance.json.invocations` has exactly one entry, written
2026-09-05T09:56:57Z, `reps_effective: 15`, `quarry_dirty: false`). 30 measured repetitions in one
shape — task 06, whole-repo cold-start orientation with no scope hint — all ingested and scored
(`unscored_count: 0` in both cells; `recall_n` and `precision_n` are 15 in both, per `table.txt`).

This root stands alone. The five ladder-d repetitions in `results/2026-09-04-breadth` are **not** pooled
in and are not compared against by cost: different results root, different server binary. They appear
below only as the prior whose pattern this run was designed to test.

## Numbers (median [min–max], this results root only)

Every figure is quoted verbatim from `summary.json`'s per-cell `metrics` block or `table.txt`'s
`recall_n`/`precision_n` columns; none is re-derived. Column names are `table.txt`'s; the summary JSON
spells four of them differently: `cache_read` is `cache_read_input_tokens`, `cache_creation` is
`cache_creation_input_tokens`, `prefixed_tool_uses` is `quarry_tool_uses`, and `grep_fallback` is
`grep_fallback_total`.

| metric | d0-none (control) | d1-toc-dir |
|---|---|---|
| turns | 11 [9–24] | 11 [8–19] |
| duration_ms | 42025 [30613–62008] | 41557 [31305–67732] |
| cost_usd | 0.3006 [0.2452–0.3967] | 0.3024 [0.2345–0.359] |
| cache_read | 152169 [105333–389015] | 126468 [81000–302281] |
| cache_creation | 32005 [24875–45767] | 32170 [23848–42293] |
| output_tokens | 9 [5–57] | 8 [6–53] |
| input_tokens_total | 181838 [136742–421153] | 162904 [121930–334469] |
| tool_uses | 10 [8–23] | 10 [7–18] |
| prefixed_tool_uses | 0 [0–0] | 1 [1–2] |
| grep_fallback | 3 [2–13] | 2 [1–6] |
| tool_result_bytes | 70839 [56516–91898] | 67920 [53638–88011] |
| read_bytes | 46705 [46705–71316] | 54037 [30467–72762] |
| recall (n) | 0.76 [0.65–0.88] (n=15) | 0.78 [0.62–0.92] (n=15) |
| precision (n) | 0.92 [0.85–1] (n=15) | 0.85 [0.75–0.94] (n=15) |

## Rule part 1 — the M1 three-part separation rule, as reported context

The M1 rule (overview's `## Shared Decisions`, `separation-decision-rule`) requires (1) at least one cost
metric marked `separated: true`, (2) that metric's median moving in the predicted direction, and (3)
neither recall nor precision degrading. It is reported here, not used as the decider, because it grows
*more* conservative as n rises: its ranges widen with every added repetition.

It fails at requirement (1). `summary.json` marks `separated: false` for every cost metric — turns,
duration_ms, cost_usd, cache_read, cache_creation, output_tokens, input_tokens_total, tool_uses,
grep_fallback, tool_result_bytes, read_bytes. The one `separated: true` entry is `quarry_tool_uses`
(`median 1=0`, `cell-range [1, 2]` vs `control-range [0, 0]`), a usage count rather than a cost metric.
No shape-level separation, exactly as at n=5.

## Rule part 2 — the predeclared decider: one-sided Mann–Whitney U

Predeclared before the run: one-sided Mann–Whitney U, `d1-toc-dir` vs `d0-none`, on **turns** and on
**cost_usd**, alternative = rung cheaper than control, α = 0.05 per metric, computed by hand from the 15
per-repetition values per cell. n₁ = n₂ = 15, so the exact-table critical value is **U ≤ 72** (one-tailed,
α = 0.05); the normal approximation's critical value is z ≤ −1.645. Both are shown.

### The per-repetition values (from `raw/<cell>/<rep>/usage.json`)

| rep | d0-none turns | d1-toc-dir turns | d0-none cost_usd | d1-toc-dir cost_usd |
|---|---|---|---|---|
| 1 | 12 | 10 | 0.3688542 | 0.3306020 |
| 2 | 11 | 13 | 0.2943676 | 0.3252651 |
| 3 | 12 | 12 | 0.3227299 | 0.3326625 |
| 4 | 24 | 19 | 0.3966825 | 0.3589683 |
| 5 | 12 | 11 | 0.3678484 | 0.2883670 |
| 6 | 10 | 10 | 0.2451802 | 0.2642673 |
| 7 | 11 | 10 | 0.3005599 | 0.2555490 |
| 8 | 9 | 8 | 0.2753156 | 0.2426129 |
| 9 | 11 | 12 | 0.2632076 | 0.3046810 |
| 10 | 10 | 11 | 0.2686965 | 0.3024363 |
| 11 | 18 | 9 | 0.3139248 | 0.2345152 |
| 12 | 12 | 13 | 0.3056484 | 0.3174664 |
| 13 | 9 | 9 | 0.2818375 | 0.2372230 |
| 14 | 11 | 11 | 0.3319113 | 0.3203767 |
| 15 | 10 | 11 | 0.2839557 | 0.2518417 |

### Metric 1 — turns

Combined N = 30, ranks 1…30, ties averaged. Value counts across both cells: 8×1, 9×3, 10×7, 11×8, 12×6,
13×2, 18×1, 19×1, 24×1 (sums to 30). Midranks: 8 → 1; 9 → 3; 10 → 8; 11 → 15.5; 12 → 22.5; 13 → 26.5;
18 → 28; 19 → 29; 24 → 30.

`d1-toc-dir` holds one 8, one 9, four 10s, four 11s, two 12s, two 13s, one 19:

    R₁ = 1 + 3 + 4(8) + 4(15.5) + 2(22.5) + 2(26.5) + 29
       = 1 + 3 + 32 + 62 + 45 + 53 + 29 = 225

Total rank sum is N(N+1)/2 = 30(31)/2 = 465, so R₀ = 465 − 225 = 240 (checked directly against
`d0-none`'s own values: 2(3) + 3(8) + 4(15.5) + 4(22.5) + 28 + 30 = 6 + 24 + 62 + 90 + 28 + 30 = 240 ✓).

    U₁ = R₁ − n₁(n₁+1)/2 = 225 − 120 = 105
    U₀ = 240 − 120 = 120           (U₁ + U₀ = 225 = n₁n₂ ✓)

**U₁ = 105 > 72 → fail to reject at α = 0.05.**

Normal approximation with the tie correction, as a cross-check. μ_U = n₁n₂/2 = 112.5. Σ(t³ − t) over the
tie groups = 24 + 336 + 504 + 210 + 6 = 1080, so

    σ² = [n₁n₂ / (N(N−1))] · [ (N³ − N)/12 − Σ(t³ − t)/12 ]
       = (225 / 870) · (2247.5 − 90) = (225 / 870)(2157.5) = 557.974
    σ  = 23.6215
    z  = (105 − 112.5) / 23.6215 = −7.5 / 23.6215 = −0.318

One-sided p ≈ 0.375. Not significant.

### Metric 2 — cost_usd

No ties (fifteen distinct values per cell, thirty distinct overall). Ascending order over all 30 values,
`d1-toc-dir` taking ranks 1, 2, 3, 5, 6, 8, 13, 16, 17, 20, 21, 23, 24, 26, 27:

    R₁ = 1+2+3+5+6+8+13+16+17+20+21+23+24+26+27 = 212
    R₀ = 465 − 212 = 253   (checked directly: 4+7+9+10+11+12+14+15+18+19+22+25+28+29+30 = 253 ✓)

    U₁ = 212 − 120 = 92
    U₀ = 253 − 120 = 133           (U₁ + U₀ = 225 ✓)

**U₁ = 92 > 72 → fail to reject at α = 0.05.**

Normal approximation, no tie correction needed:

    σ = √[ n₁n₂(N+1)/12 ] = √(225 · 31 / 12) = √581.25 = 24.109
    z = (92 − 112.5) / 24.109 = −20.5 / 24.109 = −0.850

One-sided p ≈ 0.198. Not significant.

### Correctness, for completeness

Requirement 3 of the win condition is moot once both metrics fail, but the numbers are recorded: recall's
median rose (0.78 vs 0.76) and precision's median fell (0.85 vs 0.92). Both comparisons are marked
`separated: false` and both ranges overlap their control's (recall [0.62–0.92] vs [0.65–0.88]; precision
[0.75–0.94] vs [0.85–1]), so neither counts as a degradation under the rule's own wording, and neither
would have rescued a cost result that did not exist.

## Verdict: no win

Applying the predeclared rule verbatim. Rule 3's win condition — at least one of turns or cost_usd
rejecting at α = 0.05 in the predicted direction — is not met: turns gives U = 105 (p ≈ 0.375) and
cost_usd gives U = 92 (p ≈ 0.198), both far above the U ≤ 72 threshold and both far short of α = 0.05.
Rule 4 therefore applies: **no win.**

The n=5 pattern this run existed to test did not survive n=15, and it did not merely fail to reach
significance — it partly reversed. At n=5 every ladder-d cost median pointed the hypothesis's way. At
n=15 turns is a dead tie (11 = 11) and cost_usd's median moves *against* the hypothesis (0.3024 for the
toc cell vs 0.3006 for the control). The two metrics where a gap survives — cache_read (126468 vs
152169) and input_tokens_total (162904 vs 181838) — are not the predeclared metrics, are not marked
`separated`, and carry ranges wide enough (`d0-none` cache_read reaches 389015) that reading them as an
effect would be exactly the post-hoc metric-shopping the predeclared rule exists to prevent. `read_bytes`
runs *higher* for the toc cell here (54037 vs 46705), reversing even the one mechanism-level signal that
looked consistent in M1's ladders c and d.

Per rule 5, n = 15 was fixed before the first repetition and stayed fixed: no optional stopping, no
extension after seeing the numbers, no added cells. The three automatic within-invocation retries below
replaced invalid attempts; they did not grow n.

## Coverage: gates, invalidations, drift, provenance

- **Gate 1 (`granted_tool_used`) did not fire.** `summary.json` carries no `gate1` field for
  `d1-toc-dir` (the field is `omitempty` and populated only when the max `quarry_tool_uses` across a
  cell's repetitions is zero). The cell's `quarry_tool_uses` range is `[1, 2]` — *every* one of the 15
  repetitions called the granted `toc` tool at least once, the strongest form this gate can take. This
  cell measured toc, not toc's prompt cost.
- **Invalidation record.** Three attempts across three `d0-none` repetitions were invalidated and retried
  automatically within this same invocation, all with `cause: runner_error` ("the measured claude process
  failed") and no other cause value anywhere in this root:
  - `d0-none` repetition 3, attempt 1 (`raw/d0-none/3.invalid-1/invalid_reason.txt`)
  - `d0-none` repetition 11, attempt 1 (`raw/d0-none/11.invalid-1/invalid_reason.txt`)
  - `d0-none` repetition 12, attempt 1 (`raw/d0-none/12.invalid-1/invalid_reason.txt`)

  Each completed on its second attempt, well inside `MaxAttempts` (3); no repetition was
  attempt-exhausted. `d1-toc-dir` carries no `.invalid-*` directory at all — all 15 of its repetitions
  completed on the first attempt. `summary.json` records `"invalid": null` and the run's own report line
  reads `invalid: none`, since these were replaced within the invocation rather than left standing.
  Per the M2-invalidation-record decision, `detail` values are never quoted beyond the fixed constructed
  string the harness itself writes, which carries no machine path.
- **`quarry_dirty: false`.** Unlike M1's root, this invocation ran against a clean tree —
  `quarry_dirty_files` is `null`, and no carve-out is invoked. `quarry_commit`
  `0ae4daa62af167be55142396417eed6dfa86d9dd` is the task branch's head, which differs from `main` by a
  single `_mill/status.md` bookkeeping commit and by no source file, so the "from clean `main`"
  requirement holds in substance as well as in form.
- **One invocation.** `provenance.json.invocations` has exactly one entry. Both cells reached 15/15
  complete, non-blinding-failed repetitions on it; no resume was needed and none was performed.
- **Server binary identical across the cell.** `server_hashes` records the same
  `6917c1b1b8fe950ad6bcc1f41c6bf84ee4254907a5778e207f2f931193f947a4` for all 15 `d1-toc-dir`
  repetitions — one binary, no mid-matrix rebuild.
- **Session-fingerprint drift.** `table.txt` records the expected drift between each of `d1-toc-dir`'s
  15 repetitions and `d0-none/1`: the extra tool `mcp__quarry__toc`, `mcp_servers: [quarry]`, and
  `mcp_server_statuses: {quarry: connected}`. This is the manipulation under test, not a defect. No other
  fingerprint field differs — `claude_code_version` 2.1.236, `model` claude-sonnet-5, `permission_mode`
  default, `skill_count` 16 and `slash_command_count` 48 are identical across all 30 repetitions.
- **No permitted repetition deletion.** No `raw/` repetition directory was deleted to force a re-score;
  `unscored_count` was 0 in both cells on the first invocation.

Neither cell reports `unscored_count` non-zero and neither falls short of 15/15, so nothing here is
reported as incomplete or unmeasured — both sides of every comparison above are a full n=15 result.

## What this settles

Ladder d was the last shape with a live claim. It was the one place in M1's breadth matrix whose medians
all pointed the hypothesis's way, and the only pattern at n=5 that a larger n could distinguish from
noise. At the n this task fixed in advance, with the test it declared in advance, it does not separate —
and its central tendency on the two predeclared metrics is now flat-to-slightly-adverse rather than
merely non-significant.

Read together with `results/2026-09-04-toc` (the original ladder-a win did not reproduce at n=5) and
`results/2026-09-04-breadth` (no separation in any of three shapes at n=5), the measurement programme has
now tested directory-level `toc` at the scope axis's two-package origin, its three-package midpoint, its
whole-repo far end, and a negative control — and, at the far end, at three times the repetitions. None of
them produced a cost win.

In the roadmap's own terms, for T8's unpark condition (a): the surface does not pay at any measured shape
or n. Condition (a) is closed. Condition (b) — an explicit re-justification by *function* — is the one
that remains.
